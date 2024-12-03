package queue

import "sync"

type SimpleQueue struct {
	mu     *sync.Mutex
	values []byte
}

func CreateQueue() *SimpleQueue {
	return &SimpleQueue{
		mu:     &sync.Mutex{},
		values: make([]byte, 0),
	}
}

func (queue *SimpleQueue) Size() int {
	return len(queue.values)
}

func (queue *SimpleQueue) Queue(value byte) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.values = append(queue.values, value)
}

func (queue *SimpleQueue) DeQueue() byte {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	value := queue.values[0]
	queue.values = queue.values[1:]
	return value
}

func (queue *SimpleQueue) IsEmpty() bool {
	return len(queue.values) == 0
}
