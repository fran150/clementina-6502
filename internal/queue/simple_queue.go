// Package queue provides queue data structure implementations
package queue

import "sync"

// SimpleQueue represents a thread-safe generic queue data structure.
// It provides basic queue operations with mutex-based synchronization.
type SimpleQueue[T any] struct {
	mu     *sync.Mutex
	values []T
}

// NewQueue creates and returns a new empty SimpleQueue.
// The queue is initialized with an empty slice and a mutex for thread safety.
func NewQueue[T any]() *SimpleQueue[T] {
	return &SimpleQueue[T]{
		mu:     &sync.Mutex{},
		values: make([]T, 0),
	}
}

// Size returns the current number of elements in the queue
func (queue *SimpleQueue[T]) Size() int {
	return len(queue.values)
}

// Queue adds a new element to the end of the queue
func (queue *SimpleQueue[T]) Queue(value T) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.values = append(queue.values, value)
}

// DeQueue removes and returns the first element from the queue.
// Note: This method will panic if the queue is empty
func (queue *SimpleQueue[T]) DeQueue() T {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	value := queue.values[0]
	queue.values = queue.values[1:]
	return value
}

// IsEmpty returns true if the queue contains no elements
func (queue *SimpleQueue[T]) IsEmpty() bool {
	return len(queue.values) == 0
}

// GetValues returns a slice containing all elements currently in the queue
func (queue *SimpleQueue[T]) GetValues() []T {
	return queue.values
}
