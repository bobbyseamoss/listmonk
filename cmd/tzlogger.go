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

// Write implements io.Writer and prefixes the log message with an EST/EDT timestamp.
func (w *tzWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get current time in EST/EDT
	now := time.Now().In(w.loc)

	// Format: 2025/11/01 10:23:45.123456 EST
	timestamp := now.Format("2006/01/02 15:04:05.000000 MST")

	// Prefix the message with timestamp
	prefixed := []byte(timestamp + " ")

	// Write to all writers
	for _, writer := range w.writers {
		// Write timestamp prefix
		if _, err := writer.Write(prefixed); err != nil {
			return 0, err
		}
		// Write original message
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
