package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fran150/clementina-6502/pkg/computers/clementina"
)

// runHeadless runs computer in real time with no terminal UI and no console
// output. Video and input stay on UDP for the video client. With --control the
// automation API is served on loopback and its URL is the only line printed, so
// a debugger can start the emulator and read where to connect.
func runHeadless(computer *clementina.ClementinaComputer) {
	machine := clementina.NewRealtimeAutomation(computer, targetMhz)
	defer machine.Close()

	exitf := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
		machine.Close()
		os.Exit(1)
	}
	if _, err := machine.Execute(clementina.AutomationRequest{Version: 1, Method: "reset"}); err != nil {
		exitf("reset: %v", err)
	}
	if !startPaused {
		if _, err := machine.Execute(clementina.AutomationRequest{Version: 1, Method: "run"}); err != nil {
			exitf("run: %v", err)
		}
	}

	served := make(chan error, 1)
	if control {
		listener, err := clementina.ListenAutomation(controlPort)
		if err != nil {
			exitf("control port: %v", err)
		}
		fmt.Println(clementina.AutomationURL(listener))
		go func() { served <- clementina.ServeAutomation(listener, machine) }()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case <-signals:
	case err := <-served:
		exitf("control server: %v", err)
	}
}
