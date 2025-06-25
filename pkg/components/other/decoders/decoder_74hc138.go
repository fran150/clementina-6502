package decoders

import (
	"math"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// The 74HC138 is a 3-to-8 line decoder/demultiplexer.
// It takes 3 input lines (A0, A1, A2) and decodes them into one of the 8 output lines (Y0 to Y7).
// The outputs are active low, meaning that when a specific input combination is selected,
// the corresponding output line will be low (0), while all other outputs will be high (1).
// The chip also has three enable inputs (E1, E2, E3) that control whether the decoder is active or not.
// If any of the enable inputs are low, all outputs will be high (inactive).
type Decoder74HC138 struct {
	yPin *buses.BusConnector[uint8]
	aPin [3]buses.LineConnector
	ePin [3]buses.LineConnector
}

// Creates a new 74HC138
func NewDecoder74HC138() *Decoder74HC138 {
	chip := Decoder74HC138{}

	chip.yPin = buses.NewBusConnector[uint8]()

	chip.aPin[0] = buses.NewConnectorEnabledHigh()
	chip.aPin[1] = buses.NewConnectorEnabledHigh()
	chip.aPin[2] = buses.NewConnectorEnabledHigh()

	chip.ePin[0] = buses.NewConnectorEnabledLow()
	chip.ePin[1] = buses.NewConnectorEnabledLow()
	chip.ePin[2] = buses.NewConnectorEnabledHigh()

	return &chip
}

// Returns the connector for the 8 bits output. This will be the active low output Y0 to Y7.
// If input in A pins is 000, Y0 will be low, if input is 001, Y1 will be low, etc.
func (gate *Decoder74HC138) YPin() *buses.BusConnector[uint8] {
	return gate.yPin
}

// Returns the connector for the specified pin A (Address input)
// The A pins are used to select which output line will be activated.
// The A0 pin corresponds to the least significant bit (LSB) and A2 to the most significant bit (MSB).
// The combination of these three pins determines which output line will be low (active).
// For example, if A0=1, A1=0, A2=0, then Y1 will be low (active), and all other outputs will be high (inactive).
// if A0=1, A1=1, A2=0, then Y3 will be low (active), and all other outputs will be high (inactive).
func (gate *Decoder74HC138) APin(index int) buses.LineConnector {
	if index >= 0 && index < 3 {
		return gate.aPin[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin E (Enable input)
// The E pins are used to enable or disable the decoder.
// E1 and E2 must be low and E3 high for the decoder to function respond. If the decoder
// is not enabled, all outputs will be high (inactive).
func (gate *Decoder74HC138) EPin(index int) buses.LineConnector {
	if index >= 0 && index < 3 {
		return gate.ePin[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *Decoder74HC138) Tick(stepContext *common.StepContext) {
	if gate.ePin[0].Enabled() && gate.ePin[1].Enabled() && gate.ePin[2].Enabled() {
		value := uint8(0)
		pin := uint8(1)

		for i := range 3 {
			if gate.aPin[i].Enabled() {
				value += pin
			}

			pin <<= 1
		}

		value = uint8(math.Pow(2.0, float64(value)))

		gate.yPin.Write(^value)
	} else {
		gate.yPin.Write(0xFF)
	}
}
