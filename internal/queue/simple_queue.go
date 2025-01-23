package queue

import "sync"

type SimpleQueue[T any] struct {
	mu     *sync.Mutex
	values []T
}

func CreateQueue[T any]() *SimpleQueue[T] {
	return &SimpleQueue[T]{
		mu:     &sync.Mutex{},
		values: make([]T, 0),
	}
}

func (queue *SimpleQueue[T]) Size() int {
	return len(queue.values)
}

func (queue *SimpleQueue[T]) Queue(value T) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.values = append(queue.values, value)
}

func (queue *SimpleQueue[T]) DeQueue() T {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	value := queue.values[0]
	queue.values = queue.values[1:]
	return value
}

func (queue *SimpleQueue[T]) IsEmpty() bool {
	return len(queue.values) == 0
}

func (queue *SimpleQueue[T]) GetValues() []T {
	return queue.values
}
