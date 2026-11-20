// Command clementina-automation serves one exclusively owned headless machine.
// It runs unthrottled and has no video or input over UDP; it is meant for
// tests and checks. The real-time emulator serves the same API with
// `clementina --headless --control`.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/internal/buildinfo"
	"github.com/fran150/clementina-6502/pkg/computers/clementina"
)

func main() {
	port := flag.Uint("port", 0, "loopback TCP port; zero chooses an available port")
	sdFolder := flag.String("sd", "", "host folder mounted as the emulated MIA SD card")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String("clementina-automation"))
		return
	}
	if *port > 65535 {
		fmt.Fprintln(os.Stderr, "invalid port")
		os.Exit(1)
	}
	if *sdFolder != "" {
		info, err := os.Stat(*sdFolder)
		if err != nil || !info.IsDir() {
			fmt.Fprintln(os.Stderr, "sd path must be an existing directory")
			os.Exit(1)
		}
	}
	machine, err := clementina.NewAutomationWithSDFolder(*sdFolder)
	if err != nil {
		panic(err)
	}
	defer machine.Close()
	listener, err := clementina.ListenAutomation(*port)
	if err != nil {
		panic(err)
	}
	fmt.Println(clementina.AutomationURL(listener))
	if err := clementina.ServeAutomation(listener, machine); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
