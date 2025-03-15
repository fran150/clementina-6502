package queue

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue[int]()
	if q == nil {
		t.Error("Expected non-nil queue")
	}
	if !q.IsEmpty() {
		t.Error("New queue should be empty")
	}
}

func TestQueue(t *testing.T) {
	q := NewQueue[string]()

	// Test queueing single item
	q.Queue("first")
	if q.Size() != 1 {
		t.Errorf("Expected size 1, got %d", q.Size())
	}
	if q.IsEmpty() {
		t.Error("Queue should not be empty")
	}

	// Test queueing multiple items
	q.Queue("second")
	q.Queue("third")
	if q.Size() != 3 {
		t.Errorf("Expected size 3, got %d", q.Size())
	}

	// Test values order
	values := q.GetValues()
	expected := []string{"first", "second", "third"}
	for i, v := range values {
		if v != expected[i] {
			t.Errorf("Expected %s at position %d, got %s", expected[i], i, v)
		}
	}
}

func TestDeQueue(t *testing.T) {
	q := NewQueue[int]()

	// Setup test data
	q.Queue(1)
	q.Queue(2)
	q.Queue(3)

	// Test dequeuing
	value := q.DeQueue()
	if value != 1 {
		t.Errorf("Expected 1, got %d", value)
	}
	if q.Size() != 2 {
		t.Errorf("Expected size 2, got %d", q.Size())
	}

	// Test remaining values
	values := q.GetValues()
	expected := []int{2, 3}
	for i, v := range values {
		if v != expected[i] {
			t.Errorf("Expected %d at position %d, got %d", expected[i], i, v)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	q := NewQueue[int]()

	if !q.IsEmpty() {
		t.Error("New queue should be empty")
	}

	q.Queue(1)
	if q.IsEmpty() {
		t.Error("Queue should not be empty after adding element")
	}

	q.DeQueue()
	if !q.IsEmpty() {
		t.Error("Queue should be empty after removing all elements")
	}
}

func TestSize(t *testing.T) {
	q := NewQueue[int]()

	if q.Size() != 0 {
		t.Errorf("Expected size 0, got %d", q.Size())
	}

	q.Queue(1)
	q.Queue(2)
	if q.Size() != 2 {
		t.Errorf("Expected size 2, got %d", q.Size())
	}

	q.DeQueue()
	if q.Size() != 1 {
		t.Errorf("Expected size 1, got %d", q.Size())
	}
}
