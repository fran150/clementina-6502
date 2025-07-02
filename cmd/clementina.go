package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/computers/beneater"
	"github.com/fran150/clementina-6502/pkg/computers/clementina"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

type computerModel string

const (
	clementinaModel computerModel = "clementina"
	beneaterModel   computerModel = "beneater"
)

var (
	model             computerModel
	serialPort        string
	romFile           string
	delay             int64
	targetFps         int
	emulateModemLines bool
)

var rootCmd = &cobra.Command{
	Use:   "clementina",
	Short: "Clementina 6502 - A 6502 computer emulator",
	Long:  `Clementina 6502 is an emulator for the Clementina 6502 or Ben Eater's 6502 computer.`,
	Run:   runEmulator,
}

func init() {
	var strModel string
	rootCmd.Flags().StringVarP(&strModel, "model", "m", "clementina", "Computer model to emulate (clementina / beneater)")
	rootCmd.Flags().StringVarP(&serialPort, "port", "p", "", "Serial port to connect to (e.g., /dev/ttys004)")
	rootCmd.Flags().StringVarP(&romFile, "rom", "r", "./assets/computer/beneater/eater.bin", "ROM file to load")
	rootCmd.Flags().Int64VarP(&delay, "skip-cycles", "s", 0, "Number of CPU cycles to skip on every loop")
	rootCmd.Flags().IntVarP(&targetFps, "fps", "f", 15, "Target display refresh rate")
	rootCmd.Flags().BoolVarP(&emulateModemLines, "emulate-modem", "e", false, "Enable modem lines emulation for serial port (RTS, CTS, DTR, DSR)")

	model = computerModel(strModel)
}

func runEmulator(cmd *cobra.Command, args []string) {
	var port serial.Port

	if serialPort != "" {
		var err error

		port, err = serial.Open(serialPort, &serial.Mode{
			BaudRate: 19200,
			DataBits: 8,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating port: %v\n", err)
			os.Exit(1)
		}
	}

	// Create the computer instance
	var computer terminal.Computer
	if model == beneaterModel {
		beneater, err := beneater.NewBenEaterComputer(port, emulateModemLines)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
			os.Exit(1)
		}

		defer beneater.Close()

		// Try to load the ROM file
		if err := beneater.LoadRom(romFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading ROM file: %v\n", err)
			os.Exit(1)
		}

		computer = beneater
	} else {
		clementina, err := clementina.NewClementinaComputer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
			os.Exit(1)
		}

		clementina.HiRamPoke(0xFFFC, 0x00, 0x00) // Set $E100 in the reset vector
		clementina.HiRamPoke(0xFFFD, 0x00, 0xE1)

		clementina.HiRamPoke(0xE100, 0x00, 0xA9) // LDA #$01
		clementina.HiRamPoke(0xE101, 0x00, 0x01)
		clementina.HiRamPoke(0xE102, 0x00, 0x1A) // INC A
		clementina.HiRamPoke(0xE103, 0x00, 0x4C) // JMP $E102
		clementina.HiRamPoke(0xE104, 0x00, 0x02)
		clementina.HiRamPoke(0xE105, 0x00, 0xE1)

		computer = clementina
	}

	// Setup configuration
	config := terminal.ApplicationConfig{
		EmulationLoopConfig: computers.EmulationLoopConfig{
			SkipCycles: delay,
			DisplayFps: targetFps,
		},
	}

	// Create and run the application
	app := terminal.NewApplication(computer, &config)
	t := time.Now()

	context, err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}

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
