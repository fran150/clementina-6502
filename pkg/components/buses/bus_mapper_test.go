package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func new8To16Mapper() (Bus[uint16], Bus[uint8]) {
	bus := New16BitStandaloneBus()

	return bus, New8BitMappedBus(bus,
		func(value uint8) uint16 {
			return uint16(value) << 8
		},
		func(value uint16) uint8 {
			return uint8(value >> 8)
		})
}

func new8To8Mapper() (Bus[uint8], Bus[uint8]) {
	bus := New8BitStandaloneBus()

	return bus, New8BitMappedBus(bus,
		func(value uint8) uint8 {
			return value << 4
		},
		func(value uint8) uint8 {
			return value >> 4
		})
}

func new16To8Mapper() (Bus[uint8], Bus[uint16]) {
	bus := New8BitStandaloneBus()

	return bus, New16BitMappedBus(bus,
		func(value uint16) uint8 {
			return uint8(value)
		},
		func(value uint8) uint16 {
			return uint16(value)
		})
}

func new16To16Mapper() (Bus[uint16], Bus[uint16]) {
	bus := New16BitStandaloneBus()

	return bus, New16BitMappedBus(bus,
		func(value uint16) uint16 {
			msb := value & 0xFF00
			lsb := value & 0x00FF

			msb = msb >> 8
			lsb = lsb << 8

			return msb | lsb
		},
		func(value uint16) uint16 {
			msb := value & 0xFF00
			lsb := value & 0x00FF

			msb = msb >> 8
			lsb = lsb << 8

			return msb | lsb
		})
}

func TestBusMapper8To16(t *testing.T) {
	t.Run("8 to 16 bit mapping", func(t *testing.T) {
		sourceBus, mappedBus := new8To16Mapper()

		// Test writing to mapped bus
		mappedBus.Write(0x12)
		value := sourceBus.Read()
		assert.Equal(t, uint16(0x1200), value, "Value should be shifted left 8 bits")

		// Test reading from mapped bus
		sourceBus.Write(0x3400)
		value8 := mappedBus.Read()
		assert.Equal(t, uint8(0x34), value8, "Should read high byte")
	})
}

func TestBusMapper8To8(t *testing.T) {
	t.Run("8 to 8 bit mapping", func(t *testing.T) {
		sourceBus, mappedBus := new8To8Mapper()

		// Test writing to mapped bus
		mappedBus.Write(uint8(0x12))
		value := sourceBus.Read()
		assert.Equal(t, uint8(0x20), value, "Value should be shifted left 4 bits")

		// Test reading from mapped bus
		sourceBus.Write(uint8(0x40))
		value8 := mappedBus.Read()
		assert.Equal(t, uint8(0x04), value8, "Value should be shifted right 4 bits")
	})
}

func TestBusMapper16To8(t *testing.T) {
	t.Run("16 to 8 bit mapping", func(t *testing.T) {
		sourceBus, mappedBus := new16To8Mapper()

		// Test writing to mapped bus
		mappedBus.Write(uint16(0x1234))
		value := sourceBus.Read()
		assert.Equal(t, uint8(0x34), value, "Should store lower byte")

		// Test reading from mapped bus
		sourceBus.Write(uint8(0x56))
		value16 := mappedBus.Read()
		assert.Equal(t, uint16(0x56), value16, "Should extend to 16 bits")
	})
}

func TestBusMapper16To16(t *testing.T) {
	t.Run("16 to 16 bit mapping", func(t *testing.T) {
		sourceBus, mappedBus := new16To16Mapper()

		// Test writing to mapped bus
		mappedBus.Write(uint16(0x1234))
		value := sourceBus.Read()
		assert.Equal(t, uint16(0x3412), value, "Bytes should be swapped")

		// Test reading from mapped bus
		sourceBus.Write(uint16(0x5678))
		value16 := mappedBus.Read()
		assert.Equal(t, uint16(0x7856), value16, "Bytes should be swapped back")
	})
}

func TestBusMapperEdgeCases(t *testing.T) {
	t.Run("8 to 16 zero value", func(t *testing.T) {
		sourceBus, mappedBus := new8To16Mapper()
		mappedBus.Write(0x00)
		assert.Equal(t, uint16(0x0000), sourceBus.Read())
	})

	t.Run("16 to 16 zero value", func(t *testing.T) {
		sourceBus, mappedBus := new16To16Mapper()
		mappedBus.Write(0x0000)
		assert.Equal(t, uint16(0x0000), sourceBus.Read())
	})

	t.Run("8 to 8 max value", func(t *testing.T) {
		sourceBus, mappedBus := new8To8Mapper()
		mappedBus.Write(0xFF)
		assert.Equal(t, uint8(0xF0), sourceBus.Read())
	})
}

func TestGetBusLine(t *testing.T) {
	t.Run("returns correct bus line for valid index", func(t *testing.T) {
		// Setup
		bus := &MappedBus[uint8, uint8]{
			busLines: make([]*BusLine[uint8], 8), // Create 8 bus lines
		}
		// Initialize a specific bus line for testing
		testLine := &BusLine[uint8]{}
		bus.busLines[3] = testLine

		// Test
		result := bus.GetBusLine(3)

		// Assert
		assert.Equal(t, testLine, result, "Should return the correct bus line")
	})

	t.Run("returns nil for out of bounds index", func(t *testing.T) {
		// Setup
		bus := &MappedBus[uint8, uint8]{
			busLines: make([]*BusLine[uint8], 8), // Create 8 bus lines
		}

		// Test cases
		testCases := []struct {
			name    string
			index   uint8
			message string
		}{
			{"at length", 8, "Should return nil at length boundary"},
			{"beyond length", 255, "Should return nil beyond length"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := bus.GetBusLine(tc.index)
				assert.Nil(t, result, tc.message)
			})
		}
	})

	t.Run("returns first and last valid bus lines", func(t *testing.T) {
		// Setup
		bus := &MappedBus[uint8, uint8]{
			busLines: make([]*BusLine[uint8], 8),
		}
		firstLine := &BusLine[uint8]{}
		lastLine := &BusLine[uint8]{}
		bus.busLines[0] = firstLine
		bus.busLines[7] = lastLine

		// Test
		firstResult := bus.GetBusLine(0)
		lastResult := bus.GetBusLine(7)

		// Assert
		assert.Equal(t, firstLine, firstResult, "Should return correct first bus line")
		assert.Equal(t, lastLine, lastResult, "Should return correct last bus line")
	})
}
