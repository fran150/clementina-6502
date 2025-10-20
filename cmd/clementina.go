package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers/beneater"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

const (
	clementinaModel string = "clementina"
	beneaterModel   string = "beneater"
)

var (
	model             string
	serialPort        string
	romFile           string
	targetMhz         float64
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
	rootCmd.Flags().StringVarP(&model, "model", "m", "clementina", "Computer model to emulate (clementina / beneater)")
	rootCmd.Flags().StringVarP(&serialPort, "port", "p", "", "Serial port to connect to (e.g., /dev/ttys004)")
	rootCmd.Flags().StringVarP(&romFile, "rom", "r", "./assets/computer/beneater/eater.bin", "ROM file to load")
	rootCmd.Flags().Float64VarP(&targetMhz, "speed", "s", 1.2, "Target emulation speed in MHz")
	rootCmd.Flags().IntVarP(&targetFps, "fps", "f", 15, "Target display refresh rate")
	rootCmd.Flags().BoolVarP(&emulateModemLines, "emulate-modem", "e", false, "Enable modem lines emulation for serial port (RTS, CTS, DTR, DSR)")
}

// ComputerRunner defines the interface for running computers in the CLI
type ComputerRunner interface {
	Run() (*common.StepContext, error)
	Stop()
}

func runEmulator(cmd *cobra.Command, args []string) {
	var emulator interfaces.Emulator

	if model == beneaterModel {
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

		config := &beneater.BenEaterComputerConfig{
			Port:              port,
			EmulateModemLines: emulateModemLines,
		}

		benEaterComputer, err := beneater.NewBenEaterComputer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
			os.Exit(1)
		}

		defer benEaterComputer.Close()

		// Try to load the ROM file
		if err := benEaterComputer.LoadRom(romFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading ROM file: %v\n", err)
			os.Exit(1)
		}

		emulator, err = beneater.NewBenEaterEmulation(benEaterComputer, targetMhz, targetFps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating emulator: %v\n", err)
			os.Exit(1)
		}
	} else {
		// config := &clementina.ClementinaComputerConfig{
		// 	DisplayFps: targetFps,
		// }

		// clementinaComputer, err := clementina.NewClementinaComputer(config)
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
		// 	os.Exit(1)
		// }

		// // Set the initial speed
		// clementinaComputer.GetSpeedController().SetTargetSpeed(targetMhz)

		// clementinaComputer.HiRamPoke(0xFFFC, 0x00, 0x00) // Set $E100 in the reset vector
		// clementinaComputer.HiRamPoke(0xFFFD, 0x00, 0xE1)

		// clementinaComputer.HiRamPoke(0xE100, 0x00, 0xA9) // LDA #$01
		// clementinaComputer.HiRamPoke(0xE101, 0x00, 0x01)
		// clementinaComputer.HiRamPoke(0xE102, 0x00, 0x1A) // INC A
		// clementinaComputer.HiRamPoke(0xE103, 0x00, 0x4C) // JMP $E102
		// clementinaComputer.HiRamPoke(0xE104, 0x00, 0x02)
		// clementinaComputer.HiRamPoke(0xE105, 0x00, 0xE1)

		// computer = clementinaComputer
	}

	t := time.Now()

	context, err := emulator.Run()
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
