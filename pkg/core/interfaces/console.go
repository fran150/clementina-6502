package interfaces

// EmulationConsole represents the console interface for the emulator.
// This is not the main display but the window that allows the user to interact with the emulator.
type EmulationConsole interface {
	// Ticker is used to update the internal state of console's components after each emulation frame
	Ticker
	// Renderer is called to draw the console.
	Renderer

	// Run starts the emulation console and returns an error if startup fails.
	Run() error

	// Stop gracefully shuts down the emulation console.
	Stop()

	// SetEmulator assigns an emulator instance to this console.
	SetEmulator(emulator Emulator)
}
