package internal

import (
	"fmt"
	"sync"
	"testing"
)

func TestRingBuffer_WriteAndLastN(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 1; i <= 5; i++ {
		rb.Write(fmt.Sprintf("line %d", i))
	}

	got := rb.LastN(3)
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}
	if got[0] != "line 3" || got[1] != "line 4" || got[2] != "line 5" {
		t.Errorf("unexpected lines: %v", got)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 1; i <= 25; i++ {
		rb.Write(fmt.Sprintf("line %d", i))
	}

	// Only last 10 should be available.
	got := rb.All()
	if len(got) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(got))
	}
	if got[0] != "line 16" {
		t.Errorf("expected first line 'line 16', got %q", got[0])
	}
	if got[9] != "line 25" {
		t.Errorf("expected last line 'line 25', got %q", got[9])
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(10)
	got := rb.LastN(5)
	if got != nil {
		t.Errorf("expected nil for empty buffer, got %v", got)
	}
}

func TestRingBuffer_LastNMoreThanAvailable(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Write("a")
	rb.Write("b")

	got := rb.LastN(100)
	if len(got) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(got))
	}
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("unexpected lines: %v", got)
	}
}

func TestRingBuffer_All(t *testing.T) {
	rb := NewRingBuffer(5)
	for i := 1; i <= 5; i++ {
		rb.Write(fmt.Sprintf("line %d", i))
	}

	got := rb.All()
	if len(got) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(got))
	}
	for i, line := range got {
		want := fmt.Sprintf("line %d", i+1)
		if line != want {
			t.Errorf("line %d: expected %q, got %q", i, want, line)
		}
	}
}

func TestRingBuffer_TotalLines(t *testing.T) {
	rb := NewRingBuffer(5)
	for i := 0; i < 20; i++ {
		rb.Write("x")
	}
	if rb.TotalLines() != 20 {
		t.Errorf("expected total 20, got %d", rb.TotalLines())
	}
}

func TestRingBuffer_ConcurrentReadWrite(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Writer goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			rb.Write(fmt.Sprintf("line %d", i))
		}
	}()

	// Reader goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = rb.LastN(10)
			_ = rb.TotalLines()
		}
	}()

	wg.Wait()

	if rb.TotalLines() != 1000 {
		t.Errorf("expected 1000 total lines, got %d", rb.TotalLines())
	}
}
