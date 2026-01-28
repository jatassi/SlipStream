package logger

import "sync"

// RingBuffer is a thread-safe circular buffer for storing log entries.
type RingBuffer[T any] struct {
	buffer []T
	head   int
	tail   int
	count  int
	size   int
	mu     sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buffer: make([]T, capacity),
		size:   capacity,
	}
}

// Push adds an item to the buffer, overwriting the oldest if full.
func (r *RingBuffer[T]) Push(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buffer[r.tail] = item
	r.tail = (r.tail + 1) % r.size

	if r.count < r.size {
		r.count++
	} else {
		r.head = (r.head + 1) % r.size
	}
}

// GetAll returns all items in order from oldest to newest.
func (r *RingBuffer[T]) GetAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]T, r.count)
	for i := 0; i < r.count; i++ {
		idx := (r.head + i) % r.size
		result[i] = r.buffer[idx]
	}
	return result
}

// Len returns the current number of items in the buffer.
func (r *RingBuffer[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

// Clear removes all items from the buffer.
func (r *RingBuffer[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.tail = 0
	r.count = 0
}
