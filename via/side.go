package via

import "github.com/fran150/clementina6502/buses"

type ViaSideConfiguration struct {
	latchingEnabledMasks         viaACRLatchingMasks
	transitionConfigurationMasks [2]viaPCRTranstitionMasks
	outputConfigurationMask      viaPCROutputMasks
	handshakeMode                viaPCROutputModes
	pulseMode                    viaPCROutputModes
	fixedMode                    viaPCROutputModes
	clearC2OnRWMask              viaPCRInterruptClearMasks

	timerRunModeMask viaTimerControlMask
	timerOutputMask  viaTimerControlMask

	timerInterruptBit   viaIRQFlags
	controlLinesIRQBits [2]viaIRQFlags
}

type ViaSideRegisters struct {
	outputRegister        uint8
	inputRegister         uint8
	dataDirectionRegister uint8
	lowLatches            uint8
	highLatches           uint8
	counter               uint16
}

type ViaSide struct {
	configuration *ViaSideConfiguration
	registers     *ViaSideRegisters

	controlLines   *viaControlLines
	peripheralPort *ViaPort
	timer          *ViaTimer
}

func createViaSide(via *Via65C22S, configuration ViaSideConfiguration) ViaSide {
	side := ViaSide{
		configuration: &configuration,
		registers:     &ViaSideRegisters{},
	}

	controlLines := viaControlLines{
		side: &side,

		lines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		previousStatus: [2]bool{false, false},

		peripheralControlRegister: &via.registers.peripheralControl,
		interrupts:                &via.registers.interrupts,
		handshakeInProgress:       false,
		handshakeCycleCounter:     0,
	}

	peripheralPort := ViaPort{
		side: &side,

		auxiliaryControlRegister:  &via.registers.auxiliaryControl,
		peripheralControlRegister: &via.registers.peripheralControl,
		interrupts:                &via.registers.interrupts,

		connector: buses.CreateBusConnector[uint8](),
	}

	timer := ViaTimer{
		side: &side,

		timerEnabled: false,

		auxiliaryControlRegister: &via.registers.auxiliaryControl,
		interrupts:               &via.registers.interrupts,
	}

	side.peripheralPort = &peripheralPort
	side.controlLines = &controlLines
	side.timer = &timer

	return side
}
