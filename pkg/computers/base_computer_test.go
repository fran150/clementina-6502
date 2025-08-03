package computers

import (
	"github.com/rivo/tview"
)

func createBaseComputer() *BaseComputer {
	config := &EmulationLoopConfig{
		TargetSpeedMhz: 1.0,
		DisplayFps:     10,
	}

	loop := NewEmulationLoop(config)
	tvApp := tview.NewApplication()

	return NewBaseComputer(loop, tvApp)
}
