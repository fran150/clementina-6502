package computers

func createBaseComputer() *BaseComputer {
	config := &EmulationLoopConfig{
		TargetSpeedMhz: 1.0,
		DisplayFps:     10,
	}

	loop := NewEmulationLoop(config)

	return NewBaseComputer(loop)
}
