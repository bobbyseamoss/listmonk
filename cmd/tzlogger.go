package main

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// tzWriter wraps multiple writers and prefixes each log line with an EST/EDT timestamp.
type tzWriter struct {
	writers []io.Writer
	loc     *time.Location
	mu      sync.Mutex
}

// newTZWriter creates a new timezone-aware multi-writer that prefixes logs with EST/EDT timestamps.
func newTZWriter(writers ...io.Writer) *tzWriter {
	// Load EST/EDT timezone (America/New_York automatically handles DST)
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback to UTC if EST can't be loaded
		loc = time.UTC
	}

	return &tzWriter{
		writers: writers,
		loc:     loc,
	}
}

// Write implements io.Writer and writes to all configured writers.
// Note: Go's log package already adds timestamps, so we don't add another one.
func (w *tzWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Write to all writers without adding timestamp prefix
	// (Go's logger already includes timestamp in the format we configured)
	for _, writer := range w.writers {
		if _, err := writer.Write(p); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Writef is a helper method for formatted writing (useful for direct log calls).
func (w *tzWriter) Writef(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Write([]byte(msg))
}
