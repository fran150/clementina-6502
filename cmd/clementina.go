package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fran150/clementina6502/pkg/computers"
	"github.com/fran150/clementina6502/pkg/computers/beneater"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/spf13/cobra"
)

var (
	serialPort        string
	romFile           string
	targetMhz         float64
	targetFps         int
	emulateModemLines bool
)

var rootCmd = &cobra.Command{
	Use:   "clementina",
	Short: "Clementina 6502 - A 6502 computer emulator",
	Long: `Clementina 6502 is an emulator for the Ben Eater's 6502 computer.
It provides a terminal interface and can connect to real serial ports for I/O.`,
	Run: runEmulator,
}

func init() {
	rootCmd.Flags().StringVarP(&serialPort, "port", "p", "", "Serial port to connect to (e.g., /dev/ttys004)")
	rootCmd.Flags().StringVarP(&romFile, "rom", "r", "./assets/computer/beneater/eater.bin", "ROM file to load")
	rootCmd.Flags().Float64VarP(&targetMhz, "mhz", "m", 1.2, "Target emulation speed in MHz")
	rootCmd.Flags().IntVarP(&targetFps, "fps", "f", 15, "Target display refresh rate")
	rootCmd.Flags().BoolVarP(&emulateModemLines, "emulate-modem", "e", false, "Enable modem lines emulation for serial port (RTS, CTS, DTR, DSR)")
}

func runEmulator(cmd *cobra.Command, args []string) {
	// Create the computer instance
	computer, err := beneater.NewBenEaterComputer(serialPort, emulateModemLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
		os.Exit(1)
	}

	defer computer.Close()

	// Try to load the ROM file
	if err := computer.LoadRom(romFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM file: %v\n", err)
		os.Exit(1)
	}

	// Setup configuration
	config := terminal.ApplicationConfig{
		EmulationLoopConfig: computers.EmulationLoopConfig{
			TargetSpeedMhz: targetMhz,
			DisplayFps:     targetFps,
		},
	}

	// Create and run the application
	app := terminal.NewApplication(computer, &config)
	t := time.Now()

	context := app.Run()

	// Print statistics
	elapsed := time.Since(t)
	total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

	fmt.Printf("Executed %v cycles in %v seconds\n", context.Cycle, elapsed)
	fmt.Printf("Computer ran at %v MHz\n", total)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
