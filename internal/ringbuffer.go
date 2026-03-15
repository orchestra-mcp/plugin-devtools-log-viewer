package internal

import "sync"

// RingBuffer is a thread-safe circular buffer of string lines.
type RingBuffer struct {
	mu    sync.RWMutex
	lines []string
	pos   int // next write position (wraps around)
	count int // total lines written (may exceed cap)
	cap   int
}

// NewRingBuffer creates a ring buffer that holds the last capacity lines.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

// Write appends a line to the buffer. If the buffer is full, the oldest
// line is overwritten.
func (rb *RingBuffer) Write(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.lines[rb.pos%rb.cap] = line
	rb.pos++
	rb.count++
}

// LastN returns the last n lines in chronological order. If n exceeds the
// number of available lines, all available lines are returned.
func (rb *RingBuffer) LastN(n int) []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	available := rb.count
	if available > rb.cap {
		available = rb.cap
	}
	if n > available {
		n = available
	}
	if n <= 0 {
		return nil
	}

	result := make([]string, n)
	start := rb.pos - n
	for i := 0; i < n; i++ {
		idx := (start + i) % rb.cap
		if idx < 0 {
			idx += rb.cap
		}
		result[i] = rb.lines[idx]
	}
	return result
}

// All returns all available lines (up to capacity) in chronological order.
func (rb *RingBuffer) All() []string {
	return rb.LastN(rb.cap)
}

// TotalLines returns the total number of lines that have been written,
// including those that have been overwritten.
func (rb *RingBuffer) TotalLines() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
