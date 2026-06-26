package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/mia"
	"github.com/fran150/clementina-6502/pkg/computers/beneater"
	"github.com/fran150/clementina-6502/pkg/computers/clementina"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

const (
	beneaterModel       string = "beneater"
	clementinaGPIOModel string = "clementina-gpio"
)

var (
	model             string
	serialPort        string
	gpioChipName      string
	romFile           string
	videoUDPAddress   string
	inputUDPAddress   string
	sdFolder          string
	charset           string
	palette           string
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
	rootCmd.Flags().StringVarP(&model, "model", "m", "clementina", "Computer model to emulate (clementina / beneater / clementina-gpio)")
	rootCmd.Flags().StringVarP(&serialPort, "port", "p", "", "Serial port to connect to (e.g., /dev/ttys004)")
	rootCmd.Flags().StringVar(&gpioChipName, "gpio-chip", "gpiochip4", "GPIO chip to use for clementina-gpio")
	rootCmd.Flags().StringVar(&videoUDPAddress, "video-udp", mia.DefaultVideoUDPAddress, "UDP address for emulated Clementina MIA video; empty disables video UDP")
	rootCmd.Flags().StringVar(&inputUDPAddress, "input-udp", mia.DefaultInputUDPAddress, "UDP address for emulated Clementina MIA input; empty disables input UDP")
	rootCmd.Flags().StringVar(&sdFolder, "sd", "", "Host folder used as the emulated Clementina MIA SD card; empty leaves the slot empty")
	rootCmd.Flags().StringVar(&charset, "charset", "clascii", "Character set MIA loads into CHR bank 0 (name under assets/computer/mia/charsets)")
	rootCmd.Flags().StringVar(&palette, "palette", "clementina-text", "Palette MIA loads into video palette RAM (name under assets/computer/mia/palettes)")
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
	var emulator core.BaseEmulator

	switch model {
	case beneaterModel:
		var port serial.Port

		if serialPort != "" {
			var err error

			port, err = serial.Open(serialPort, &serial.Mode{
				BaudRate: 230400,
				DataBits: 8,
				Parity:   serial.NoParity,
				StopBits: serial.OneStopBit,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating port: %v\n", err)
				os.Exit(1)
			}
		}

		benEaterComputer, err := beneater.NewBenEaterComputer(&beneater.BenEaterComputerConfig{
			Port:              port,
			EmulateModemLines: emulateModemLines,
		})
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

		emulator, err = beneater.NewBenEaterEmulator(benEaterComputer, targetMhz, targetFps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating emulator: %v\n", err)
			os.Exit(1)
		}
	case clementinaGPIOModel:
		clementinaComputer, err := clementina.NewClementinaGPIOComputer(gpioChipName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
			os.Exit(1)
		}

		emulator, err = clementina.NewClemetinaGPIOEmulator(clementinaComputer, targetFps, gpioChipName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating GPIO emulator: %v\n", err)
			os.Exit(1)
		}
	default:
		clementinaComputer, err := clementina.NewClementinaComputerWithUDP(videoUDPAddress, inputUDPAddress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating computer: %v\n", err)
			os.Exit(1)
		}
		defer clementinaComputer.Close()

		clementinaComputer.SetMiaCharset(charset)
		clementinaComputer.SetMiaPalette(palette)

		if sdFolder != "" {
			info, err := os.Stat(sdFolder)
			if err != nil || !info.IsDir() {
				fmt.Fprintf(os.Stderr, "Error: --sd folder %q is not an accessible directory\n", sdFolder)
				os.Exit(1)
			}

			clementinaComputer.SetMiaSDFolder(sdFolder)
		}

		if serialPort != "" {
			port, err := serial.Open(serialPort, &serial.Mode{
				BaudRate: 115200,
				DataBits: 8,
				Parity:   serial.NoParity,
				StopBits: serial.OneStopBit,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating MIA console port: %v\n", err)
				os.Exit(1)
			}

			if err := clementinaComputer.ConnectMiaConsole(port); err != nil {
				fmt.Fprintf(os.Stderr, "Error connecting MIA console port: %v\n", err)
				os.Exit(1)
			}
		}

		emulator, err = clementina.NewClemetinaEmulator(clementinaComputer, targetMhz, targetFps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating emulator: %v\n", err)
			os.Exit(1)
		}
	}

	t := time.Now()

	context, err := emulator.Start()
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
