package slicesext

import (
	"reflect"
	"testing"
)

func TestSliceRemove(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		index    int
		expected []int
	}{
		{
			name:     "remove from middle",
			slice:    []int{1, 2, 3, 4, 5},
			index:    2,
			expected: []int{1, 2, 4, 5},
		},
		{
			name:     "remove first element",
			slice:    []int{1, 2, 3},
			index:    0,
			expected: []int{2, 3},
		},
		{
			name:     "remove last element",
			slice:    []int{1, 2, 3},
			index:    2,
			expected: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceRemove(tt.slice, tt.index)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SliceRemove() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSlicePop(t *testing.T) {
	tests := []struct {
		name          string
		slice         []int
		expectedSlice []int
		expectedValue int
	}{
		{
			name:          "pop from non-empty slice",
			slice:         []int{1, 2, 3},
			expectedSlice: []int{1, 2},
			expectedValue: 3,
		},
		{
			name:          "pop from single element slice",
			slice:         []int{1},
			expectedSlice: []int{},
			expectedValue: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newSlice, value := SlicePop(tt.slice)
			if !reflect.DeepEqual(newSlice, tt.expectedSlice) {
				t.Errorf("SlicePop() slice = %v, want %v", newSlice, tt.expectedSlice)
			}
			if value != tt.expectedValue {
				t.Errorf("SlicePop() value = %v, want %v", value, tt.expectedValue)
			}
		})
	}
}

// Test with string type to verify generic implementation
func TestSliceRemoveString(t *testing.T) {
	slice := []string{"a", "b", "c"}
	result := SliceRemove(slice, 1)
	expected := []string{"a", "c"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SliceRemove() = %v, want %v", result, expected)
	}
}

// Test with string type to verify generic implementation
func TestSlicePopString(t *testing.T) {
	slice := []string{"a", "b", "c"}
	expectedSlice := []string{"a", "b"}
	expectedValue := "c"

	newSlice, value := SlicePop(slice)

	if !reflect.DeepEqual(newSlice, expectedSlice) {
		t.Errorf("SlicePop() slice = %v, want %v", newSlice, expectedSlice)
	}
	if value != expectedValue {
		t.Errorf("SlicePop() value = %v, want %v", value, expectedValue)
	}
}
