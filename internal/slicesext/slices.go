// Package slicesext provides utility functions for slice manipulation in Go.
package slicesext

// SliceRemove removes and returns a new slice with the element at the specified index removed.
// It preserves the order of the remaining elements.
//
// Parameters:
//   - slice: The input slice of any type
//   - index: The index of the element to remove
//
// Returns:
//   - A new slice with the element at the specified index removed
func SliceRemove[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

// SlicePop removes and returns the last element of the slice along with the modified slice.
// This operation is similar to pop operations in stack data structures.
//
// Parameters:
//   - slice: The input slice of any type
//
// Returns:
//   - The modified slice with the last element removed
//   - The removed last element
func SlicePop[T any](slice []T) ([]T, T) {
	s := len(slice) - 1
	value := slice[s]
	newSlice := slice[:s]

	return newSlice, value
}
