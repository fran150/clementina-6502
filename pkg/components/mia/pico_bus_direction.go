package mia

type picoDataBusDirection uint8

const (
	picoDataBusHighZ picoDataBusDirection = iota
	picoDataBusInput
	picoDataBusOutput
)

func picoDataBusDirectionForCycle(miaSelected bool, writeEnabled bool) picoDataBusDirection {
	if !miaSelected {
		return picoDataBusHighZ
	}

	if miaSelected && writeEnabled {
		return picoDataBusOutput
	}

	return picoDataBusInput
}
