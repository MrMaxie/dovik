package supervision

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// OutputStream identifies the captured process stream.
type OutputStream string

const (
	OutputStreamStdout OutputStream = "stdout"
	OutputStreamStderr OutputStream = "stderr"
)

// OutputEvent is one ordered captured write from stdout or stderr.
type OutputEvent struct {
	Sequence   uint64
	CapturedAt time.Time
	Stream     OutputStream
	Data       []byte
}

// OutputTail contains a bounded ordered view of retained output.
type OutputTail struct {
	Events    []OutputEvent
	Truncated bool
}

// OutputBuffer stores ordered stdout and stderr events within byte and event
// limits. It is safe for concurrent writers and readers.
type OutputBuffer struct {
	mu           sync.RWMutex
	maxBytes     int
	maxEvents    int
	bytes        int
	nextSequence uint64
	events       []OutputEvent
	truncated    bool
	now          func() time.Time
}

// NewOutputBuffer creates a bounded output buffer.
func NewOutputBuffer(maxBytes int, maxEvents int) (*OutputBuffer, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("output byte limit must be positive")
	}
	if maxEvents <= 0 {
		return nil, fmt.Errorf("output event limit must be positive")
	}
	return &OutputBuffer{
		maxBytes:     maxBytes,
		maxEvents:    maxEvents,
		nextSequence: 1,
		now:          time.Now,
	}, nil
}

// Writer returns a process stream writer backed by the buffer.
func (buffer *OutputBuffer) Writer(stream OutputStream) (io.Writer, error) {
	if stream != OutputStreamStdout && stream != OutputStreamStderr {
		return nil, fmt.Errorf("invalid output stream %q", stream)
	}
	return outputWriter{buffer: buffer, stream: stream}, nil
}

// Capture appends bytes from one stream. Writes larger than the byte limit are
// split into complete bounded events so the newest bytes remain available.
func (buffer *OutputBuffer) Capture(stream OutputStream, data []byte, capturedAt time.Time) error {
	if stream != OutputStreamStdout && stream != OutputStreamStderr {
		return fmt.Errorf("invalid output stream %q", stream)
	}
	if len(data) == 0 {
		return nil
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	for len(data) > 0 {
		chunkLength := min(len(data), buffer.maxBytes)
		chunk := append([]byte(nil), data[:chunkLength]...)
		data = data[chunkLength:]
		buffer.events = append(buffer.events, OutputEvent{
			Sequence:   buffer.nextSequence,
			CapturedAt: capturedAt,
			Stream:     stream,
			Data:       chunk,
		})
		buffer.nextSequence++
		buffer.bytes += len(chunk)
		buffer.enforceLimits()
	}
	return nil
}

// Tail returns retained events after the supplied sequence. A zero sequence
// requests the newest events, matching logs --tail behavior.
func (buffer *OutputBuffer) Tail(afterSequence uint64, limit int) (OutputTail, error) {
	if limit <= 0 {
		return OutputTail{}, fmt.Errorf("output tail limit must be positive")
	}

	buffer.mu.RLock()
	defer buffer.mu.RUnlock()
	start := 0
	if afterSequence == 0 {
		start = max(0, len(buffer.events)-limit)
	} else {
		for start < len(buffer.events) && buffer.events[start].Sequence <= afterSequence {
			start++
		}
	}
	end := min(len(buffer.events), start+limit)
	events := make([]OutputEvent, 0, end-start)
	for _, event := range buffer.events[start:end] {
		event.Data = append([]byte(nil), event.Data...)
		events = append(events, event)
	}
	return OutputTail{Events: events, Truncated: buffer.truncated}, nil
}

func (buffer *OutputBuffer) enforceLimits() {
	for len(buffer.events) > buffer.maxEvents || buffer.bytes > buffer.maxBytes {
		buffer.bytes -= len(buffer.events[0].Data)
		buffer.events = buffer.events[1:]
		buffer.truncated = true
	}
}

type outputWriter struct {
	buffer *OutputBuffer
	stream OutputStream
}

func (writer outputWriter) Write(data []byte) (int, error) {
	if err := writer.buffer.Capture(writer.stream, data, writer.buffer.now()); err != nil {
		return 0, err
	}
	return len(data), nil
}
