package common

// Returns a slice with the element at the specified index removed
func SliceRemove[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

// Pops the element at the top of the slice and returns the modified slice and the element
func SlicePop[T any](slice []T) ([]T, T) {
	s := len(slice) - 1
	value := slice[s]
	newSlice := slice[:s]

	return newSlice, value
}
