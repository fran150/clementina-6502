package interfaces

type EmulationConsole interface {
	Ticker
	Renderer
	Run() error
	Stop()

	SetEmulator(emulator Emulator)
}
