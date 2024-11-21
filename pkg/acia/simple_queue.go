package acia

import "sync"

type simpleQueue struct {
	mu     *sync.Mutex
	values []byte
}

func createQueue() *simpleQueue {
	return &simpleQueue{
		mu:     &sync.Mutex{},
		values: make([]byte, 0),
	}
}

func (queue *simpleQueue) size() int {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	return len(queue.values)
}

func (queue *simpleQueue) queue(value byte) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.values = append(queue.values, value)
}

func (queue *simpleQueue) dequeue() byte {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	value := queue.values[0]
	queue.values = queue.values[1:]
	return value
}

func (queue *simpleQueue) isEmpty() bool {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	return len(queue.values) == 0
}
