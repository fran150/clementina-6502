package decoders

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type testDecoderCircuit struct {
	a [3]buses.Line
	e [3]buses.Line
	y buses.Bus[uint8]
}

func newTestDecoderCircuit() *testDecoderCircuit {
	return &testDecoderCircuit{
		a: [3]buses.Line{
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
		},
		e: [3]buses.Line{
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
		},
		y: buses.New8BitStandaloneBus(),
	}
}

func (c *testDecoderCircuit) wire(chip *Decoder74HC138) {
	for i := 0; i < 3; i++ {
		chip.APin(i).Connect(c.a[i])
		chip.EPin(i).Connect(c.e[i])
	}

	chip.YPin().Connect(c.y)
}

func createChipAndCircuit() (*Decoder74HC138, *testDecoderCircuit) {
	chip := NewDecoder74HC138()
	circuit := newTestDecoderCircuit()
	circuit.wire(chip)
	return chip, circuit
}

type decoderTestCase struct {
	aPinInput uint8
	ePinState [3]bool
	expectedY uint8
}

func (tc *decoderTestCase) test(t *testing.T, circuit *testDecoderCircuit, chip *Decoder74HC138, step *common.StepContext) {
	for i := range 3 {
		circuit.e[i].Set(tc.ePinState[i])
		circuit.a[i].Set(tc.aPinInput&(1<<i) != 0)
	}

	chip.Tick(step)

	assert.Equal(t, tc.expectedY, circuit.y.Read())
}

func TestDecoder74HC138_AllAddressCombinations(t *testing.T) {
	step := common.NewStepContext()
	chip, circuit := createChipAndCircuit()

	for i := range uint8(8) {
		tc := decoderTestCase{
			aPinInput: i,
			ePinState: [3]bool{false, false, true},
			expectedY: 1<<i ^ 0xFF,
		}

		tc.test(t, circuit, chip, &step)
	}
}

func TestDecoder74HC138_DisableReturnsAllPinsHigh(t *testing.T) {
	step := common.NewStepContext()
	chip, circuit := createChipAndCircuit()

	for i := range uint8(3) {
		tc := decoderTestCase{
			aPinInput: 0x05,
			ePinState: [3]bool{false, false, true},
			expectedY: 0xFF,
		}

		tc.ePinState[i] = !tc.ePinState[i]

		tc.test(t, circuit, chip, &step)
	}
}

func TestDecoder74HC138_InvalidPinNumberReturnsNil(t *testing.T) {
	chip := NewDecoder74HC138()
	assert.Nil(t, chip.APin(-1))
	assert.Nil(t, chip.APin(3))
	assert.Nil(t, chip.EPin(-1))
	assert.Nil(t, chip.EPin(3))
}
