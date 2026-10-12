package mia

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

// TestParseMiaWifiConfig mirrors the firmware parser's accepted and rejected
// files, including the line number reported for each rejection.
func TestParseMiaWifiConfig(t *testing.T) {
	password63 := strings.Repeat("p", 63)

	valid := []struct {
		name     string
		text     string
		mode     miaConsoleWifiMode
		ssid     string
		password string
	}{
		{"basic sta", "ssid=Home\npassword=secret123\n", miaConsoleWifiSTA, "Home", "secret123"},
		{"crlf, spaces, comments, case", "# my net\r\n\r\n  SSID = My Home Net \r\nPassword=  pass word 1 \r\nMode = STA\r\n", miaConsoleWifiSTA, "My Home Net", "pass word 1"},
		{"ap", "mode=ap\nssid=Clementina\npassword=12345678", miaConsoleWifiAP, "Clementina", "12345678"},
		{"open network", "ssid=Cafe\npassword=\n", miaConsoleWifiSTA, "Cafe", ""},
		{"no password key", "ssid=Cafe", miaConsoleWifiSTA, "Cafe", ""},
		{"off without ssid", "mode=off\n", miaConsoleWifiOff, "", ""},
		{"value with =", "ssid=a=b\npassword=x=y=z123\n", miaConsoleWifiSTA, "a=b", "x=y=z123"},
		{"32-char ssid", "ssid=" + strings.Repeat("s", 32), miaConsoleWifiSTA, strings.Repeat("s", 32), ""},
		{"63-char password", "ssid=x\npassword=" + password63, miaConsoleWifiSTA, "x", password63},
		{"last value wins", "ssid=one\nssid=two\n", miaConsoleWifiSTA, "two", ""},
		{"utf-8 bom", "\xEF\xBB\xBFssid=Home\r\npassword=secret123\r\n", miaConsoleWifiSTA, "Home", "secret123"},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			config, _, reason, ok := parseMiaWifiConfig([]byte(tc.text))
			require.True(t, ok, reason)
			assert.Equal(t, tc.mode, config.mode)
			assert.Equal(t, tc.ssid, config.ssid)
			assert.Equal(t, tc.password, config.password)
		})
	}

	invalid := []struct {
		name   string
		text   string
		line   int
		reason string
	}{
		{"empty file", "", 0, "no settings"},
		{"only comments", "# nothing\n\n", 0, "no settings"},
		{"sta without ssid", "mode=sta\npassword=12345678\n", 0, "ssid is missing"},
		{"missing =", "ssid Home\n", 1, "expected key = value"},
		{"unknown key", "ssid=Home\nchannel=6\n", 2, "unknown key (use mode, ssid or password)"},
		{"bad mode", "mode=client\nssid=Home\n", 1, "mode must be sta, ap or off"},
		{"empty ssid", "ssid=\n", 1, "ssid must be 1-32 characters"},
		{"33-char ssid", "ssid=" + strings.Repeat("s", 33), 1, "ssid must be 1-32 characters"},
		{"short password", "ssid=x\n\npassword=1234567\n", 3, "password must be empty or 8-63 characters"},
		{"64-char password", "ssid=x\npassword=" + password63 + "p", 2, "password must be empty or 8-63 characters"},
		{"non-ascii key folding", "ſsid=Home\n", 1, "unknown key (use mode, ssid or password)"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, line, reason, ok := parseMiaWifiConfig([]byte(tc.text))
			require.False(t, ok)
			assert.Equal(t, tc.line, line)
			assert.Equal(t, tc.reason, reason)
		})
	}
}

func writeWifiConfig(t *testing.T, name, text string) string {
	t.Helper()

	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, name), []byte(text), 0o644))
	return folder
}

// TestEmulatedMiaSDPresentAtBoot verifies that attaching a folder repeats the
// firmware's boot read: the card is initialized, present, and left unmounted.
func TestEmulatedMiaSDPresentAtBoot(t *testing.T) {
	chip := newSDTestCircuit(t, t.TempDir()).chip

	assert.True(t, chip.sd.initialized)
	assert.False(t, chip.sd.mounted)
	assert.NotZero(t, chip.status()&miaStatusSDPresent)
	assert.Zero(t, chip.status()&miaStatusFSMounted)
	status := sdControlByte(chip, miaSDControlStatus)
	assert.NotZero(t, status&miaSDStatusPresent)
	assert.NotZero(t, status&miaSDStatusInitialized)
	assert.Zero(t, status&miaSDStatusMounted)

	chip.sdResetRuntimeState()
	assert.NotZero(t, chip.status()&miaStatusSDPresent, "a 6502 reset keeps the card initialized")
}

// TestEmulatedMiaWifiBootFromConfigFile verifies a valid WIFI.CFG, in any
// letter case, sets the emulated Wi-Fi state without logging the password.
func TestEmulatedMiaWifiBootFromConfigFile(t *testing.T) {
	folder := writeWifiConfig(t, "wifi.cfg", "ssid = My Net\npassword = secret123\n")
	chip := newSDTestCircuit(t, folder).chip

	assert.Equal(t, miaConsoleWifiSTA, chip.console.wifiMode)
	assert.Equal(t, "My Net", chip.console.wifiSSID)
	assert.Equal(t, miaWifiConfigSourceFile, chip.console.wifiConfigSource)
	assert.Equal(t, "Wi-Fi: using WIFI.CFG from the SD card.\nWi-Fi: connecting to 'My Net'...\nWi-Fi: connected\n", chip.console.bootLog)
	assert.Contains(t, chip.consoleWifiDetail(), "  config:     WIFI.CFG\n")
	assert.NotContains(t, chip.consoleWifiDetail(), "secret123")
}

// TestEmulatedMiaWifiBootModes verifies the AP and off modes from WIFI.CFG.
func TestEmulatedMiaWifiBootModes(t *testing.T) {
	chip := newSDTestCircuit(t, writeWifiConfig(t, "WIFI.CFG", "mode=ap\nssid=Clementina\n")).chip
	assert.Equal(t, miaConsoleWifiAP, chip.console.wifiMode)
	assert.Equal(t, "Clementina", chip.console.wifiSSID)
	assert.Contains(t, chip.console.bootLog, "Wi-Fi: AP 'Clementina' active at 192.168.4.1\n")

	chip = newSDTestCircuit(t, writeWifiConfig(t, "WIFI.CFG", "mode=off\n")).chip
	assert.Equal(t, miaConsoleWifiOff, chip.console.wifiMode)
	assert.Equal(t, miaWifiConfigSourceFile, chip.console.wifiConfigSource)
	assert.Equal(t, "Wi-Fi: using WIFI.CFG from the SD card.\nWi-Fi: off\n", chip.console.bootLog)
}

// TestEmulatedMiaWifiBootFallbacks verifies each reason WIFI.CFG is not used
// leaves Wi-Fi off and logs why, never the password.
func TestEmulatedMiaWifiBootFallbacks(t *testing.T) {
	cases := []struct {
		name   string
		folder func(t *testing.T) string
		log    string
	}{
		{"no card", func(t *testing.T) string { return "" }, "Wi-Fi: no readable SD card; Wi-Fi stays off (the emulator has no build settings).\n"},
		{"no file", func(t *testing.T) string { return t.TempDir() }, "Wi-Fi: no WIFI.CFG on the SD card; Wi-Fi stays off (the emulator has no build settings).\n"},
		{"bad line", func(t *testing.T) string {
			return writeWifiConfig(t, "WIFI.CFG", "ssid=Home\npassword=secret123\nchannel=6\n")
		}, "Wi-Fi: WIFI.CFG line 3: unknown key (use mode, ssid or password); Wi-Fi stays off (the emulator has no build settings).\n"},
		{"whole file", func(t *testing.T) string {
			return writeWifiConfig(t, "WIFI.CFG", "mode=ap\npassword=secret123\n")
		}, "Wi-Fi: WIFI.CFG: ssid is missing; Wi-Fi stays off (the emulator has no build settings).\n"},
		{"too large", func(t *testing.T) string {
			return writeWifiConfig(t, "WIFI.CFG", "ssid=Home\n"+strings.Repeat("#", miaWifiConfigMaxSize))
		}, "Wi-Fi: WIFI.CFG is over 512 bytes; Wi-Fi stays off (the emulator has no build settings).\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chip := newSDTestCircuit(t, tc.folder(t)).chip

			assert.Equal(t, miaConsoleWifiOff, chip.console.wifiMode)
			assert.Equal(t, miaWifiConfigSourceBuild, chip.console.wifiConfigSource)
			assert.Equal(t, tc.log, chip.console.bootLog)
			assert.NotContains(t, chip.console.bootLog, "secret123")
		})
	}
}

// TestEmulatedMiaConsoleShowsWifiBootLog verifies the console prints the boot
// messages before the ready banner, as the firmware's USB console does.
func TestEmulatedMiaConsoleShowsWifiBootLog(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	chip.SetSDFolder(writeWifiConfig(t, "WIFI.CFG", "ssid=Home\n"))
	mock := testutils.NewPortMock(&serial.Mode{})
	require.NoError(t, chip.ConnectToPort(mock))
	t.Cleanup(func() {
		chip.Close()
		_ = mock.Close()
	})

	waitForMiaConsoleOutput(t, mock, "Wi-Fi: using WIFI.CFG from the SD card.\nWi-Fi: connecting to 'Home'...\nWi-Fi: connected\nMIA ready.")
	sendMiaConsoleInput(mock, "status wifi\n")
	waitForMiaConsoleOutput(t, mock, "  config:     WIFI.CFG\n")
}
