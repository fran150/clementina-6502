package mia

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// This file ports MIA's boot-time Wi-Fi settings (firmware src/mia/net/
// wifi_config.c and mia_net_wifi_init in wifi.c). At boot the firmware reads
// WIFI.CFG from the SD card root and joins that network, starts an access point,
// or stays off, falling back to the MIA_WIFI_* build settings.
//
// The emulator has no radio: video and input use the host's network, and it has
// no MIA_WIFI_* build settings. The file only drives the emulated console's Wi-Fi
// state and boot log, so a WIFI.CFG can be checked here before it goes on a card.

const (
	miaWifiConfigName        = "WIFI.CFG"
	miaWifiConfigMaxSize     = 512
	miaWifiSSIDMaxLen        = 32
	miaWifiPasswordMinLen    = 8
	miaWifiPasswordMaxLen    = 63
	miaWifiConfigSourceFile  = "WIFI.CFG"
	miaWifiConfigSourceBuild = "build"
)

// miaWifiConfig mirrors mia_wifi_config_t.
type miaWifiConfig struct {
	mode     miaConsoleWifiMode
	ssid     string
	password string
}

// parseMiaWifiConfig mirrors mia_wifi_config_parse: one "key = value" per line
// (keys mode, ssid, password), blank lines and lines starting with '#' ignored.
// On failure it returns the line at fault (0 when the problem is the file as a
// whole) and a short reason.
func parseMiaWifiConfig(text []byte) (miaWifiConfig, int, string, bool) {
	config := miaWifiConfig{mode: miaConsoleWifiSTA}
	hasMode := false
	hasSSID := false

	// Skip the UTF-8 byte order mark some editors write at the start.
	text = bytes.TrimPrefix(text, []byte("\xEF\xBB\xBF"))

	for i, raw := range strings.Split(string(text), "\n") {
		line := i + 1
		entry := strings.Trim(raw, " \t\r")
		if entry == "" || entry[0] == '#' {
			continue
		}

		key, value, found := strings.Cut(entry, "=")
		if !found {
			return miaWifiConfig{}, line, "expected key = value", false
		}
		key = strings.Trim(key, " \t\r")
		value = strings.Trim(value, " \t\r")

		switch {
		case asciiEqualFold(key, "mode"):
			switch {
			case asciiEqualFold(value, "sta"):
				config.mode = miaConsoleWifiSTA
			case asciiEqualFold(value, "ap"):
				config.mode = miaConsoleWifiAP
			case asciiEqualFold(value, "off"):
				config.mode = miaConsoleWifiOff
			default:
				return miaWifiConfig{}, line, "mode must be sta, ap or off", false
			}
			hasMode = true
		case asciiEqualFold(key, "ssid"):
			if len(value) == 0 || len(value) > miaWifiSSIDMaxLen {
				return miaWifiConfig{}, line, "ssid must be 1-32 characters", false
			}
			config.ssid = value
			hasSSID = true
		case asciiEqualFold(key, "password"):
			if (len(value) != 0 && len(value) < miaWifiPasswordMinLen) || len(value) > miaWifiPasswordMaxLen {
				return miaWifiConfig{}, line, "password must be empty or 8-63 characters", false
			}
			config.password = value
		default:
			return miaWifiConfig{}, line, "unknown key (use mode, ssid or password)", false
		}
	}

	if !hasMode && !hasSSID {
		return miaWifiConfig{}, 0, "no settings", false
	}
	if config.mode != miaConsoleWifiOff && !hasSSID {
		return miaWifiConfig{}, 0, "ssid is missing", false
	}
	return config, 0, "", true
}

// asciiEqualFold compares like the firmware's key_is: ASCII letters only, so
// Unicode case folding never matches a key the firmware would reject.
func asciiEqualFold(value, name string) bool {
	if len(value) != len(name) {
		return false
	}
	for i := 0; i < len(value); i++ {
		a, b := value[i], name[i]
		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// wifiFindConfigFile returns the host path of WIFI.CFG in the SD root. FAT
// names are case-insensitive, so any letter case matches.
func (c *emulated_mia) wifiFindConfigFile() (string, bool) {
	entries, err := os.ReadDir(c.sd.rootDir)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(entry.Name(), miaWifiConfigName) {
			return filepath.Join(c.sd.rootDir, entry.Name()), true
		}
	}
	return "", false
}

// wifiLoadSDConfig mirrors wifi_load_sd_config, writing the firmware's console
// message to log when the file cannot be used. Never logs the password.
func (c *emulated_mia) wifiLoadSDConfig(log *strings.Builder) (miaWifiConfig, bool) {
	const fallback = "Wi-Fi stays off (the emulator has no build settings)"

	if !c.sdRootIsDir() {
		fmt.Fprintf(log, "Wi-Fi: no readable SD card; %s.\n", fallback)
		return miaWifiConfig{}, false
	}
	hostPath, found := c.wifiFindConfigFile()
	if !found {
		fmt.Fprintf(log, "Wi-Fi: no WIFI.CFG on the SD card; %s.\n", fallback)
		return miaWifiConfig{}, false
	}
	info, err := os.Stat(hostPath)
	if err != nil {
		fmt.Fprintf(log, "Wi-Fi: could not read WIFI.CFG; %s.\n", fallback)
		return miaWifiConfig{}, false
	}
	if info.Size() > miaWifiConfigMaxSize {
		fmt.Fprintf(log, "Wi-Fi: WIFI.CFG is over %d bytes; %s.\n", miaWifiConfigMaxSize, fallback)
		return miaWifiConfig{}, false
	}
	text, err := os.ReadFile(hostPath)
	if err != nil {
		fmt.Fprintf(log, "Wi-Fi: could not read WIFI.CFG; %s.\n", fallback)
		return miaWifiConfig{}, false
	}

	config, line, reason, ok := parseMiaWifiConfig(text)
	if !ok && line != 0 {
		fmt.Fprintf(log, "Wi-Fi: WIFI.CFG line %d: %s; %s.\n", line, reason, fallback)
	} else if !ok {
		fmt.Fprintf(log, "Wi-Fi: WIFI.CFG: %s; %s.\n", reason, fallback)
	}
	return config, ok
}

// wifiBoot mirrors mia_net_wifi_init: WIFI.CFG from the SD folder sets the
// emulated Wi-Fi state, otherwise Wi-Fi stays off. Like the firmware it runs at
// power-on only, not on a MIA reset request; the console shows its messages
// before the ready banner.
func (c *emulated_mia) wifiBoot() {
	var log strings.Builder

	c.console.wifiMode = miaConsoleWifiOff
	c.console.wifiSSID = ""
	c.console.wifiConfigSource = miaWifiConfigSourceBuild

	if config, ok := c.wifiLoadSDConfig(&log); ok {
		c.console.wifiConfigSource = miaWifiConfigSourceFile
		log.WriteString("Wi-Fi: using WIFI.CFG from the SD card.\n")
		switch config.mode {
		case miaConsoleWifiSTA:
			log.WriteString(c.consoleWifiSetSTA(config.ssid))
		case miaConsoleWifiAP:
			log.WriteString(c.consoleWifiSetAP(config.ssid))
		default:
			log.WriteString(c.consoleWifiSetOff())
		}
	}

	c.console.bootLog = log.String()
}
