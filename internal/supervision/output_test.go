package supervision

import (
	"bytes"
	"testing"
	"time"
)

func TestOutputBufferPreservesSequenceAndStream(t *testing.T) {
	buffer, err := NewOutputBuffer(1024, 10)
	if err != nil {
		t.Fatalf("NewOutputBuffer() error = %v", err)
	}
	capturedAt := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	if err := buffer.Capture(OutputStreamStdout, []byte("first"), capturedAt); err != nil {
		t.Fatalf("Capture(stdout) error = %v", err)
	}
	if err := buffer.Capture(OutputStreamStderr, []byte("second"), capturedAt); err != nil {
		t.Fatalf("Capture(stderr) error = %v", err)
	}

	tail, err := buffer.Tail(0, 10)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	if len(tail.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(tail.Events))
	}
	if tail.Events[0].Sequence != 1 || tail.Events[0].Stream != OutputStreamStdout || !bytes.Equal(tail.Events[0].Data, []byte("first")) {
		t.Errorf("first event = %#v", tail.Events[0])
	}
	if tail.Events[1].Sequence != 2 || tail.Events[1].Stream != OutputStreamStderr || !bytes.Equal(tail.Events[1].Data, []byte("second")) {
		t.Errorf("second event = %#v", tail.Events[1])
	}
}

func TestOutputBufferTailSupportsPollingAndCopiesData(t *testing.T) {
	buffer, err := NewOutputBuffer(1024, 10)
	if err != nil {
		t.Fatalf("NewOutputBuffer() error = %v", err)
	}
	for _, data := range []string{"one", "two", "three"} {
		if err := buffer.Capture(OutputStreamStdout, []byte(data), time.Time{}); err != nil {
			t.Fatalf("Capture() error = %v", err)
		}
	}

	tail, err := buffer.Tail(1, 1)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	if len(tail.Events) != 1 || tail.Events[0].Sequence != 2 {
		t.Fatalf("Tail(1, 1) events = %#v, want sequence 2", tail.Events)
	}
	tail.Events[0].Data[0] = 'X'
	reloaded, err := buffer.Tail(1, 1)
	if err != nil {
		t.Fatalf("second Tail() error = %v", err)
	}
	if string(reloaded.Events[0].Data) != "two" {
		t.Fatalf("buffer data changed through returned event: %q", reloaded.Events[0].Data)
	}
}

func TestOutputBufferEnforcesByteAndEventLimits(t *testing.T) {
	buffer, err := NewOutputBuffer(5, 2)
	if err != nil {
		t.Fatalf("NewOutputBuffer() error = %v", err)
	}
	if err := buffer.Capture(OutputStreamStdout, []byte("123456789"), time.Time{}); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	tail, err := buffer.Tail(0, 10)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	if !tail.Truncated {
		t.Fatal("Truncated = false, want true")
	}
	if len(tail.Events) != 1 || string(tail.Events[0].Data) != "6789" {
		t.Fatalf("events = %#v, want newest complete chunk", tail.Events)
	}
}

func TestOutputWriterUsesConfiguredStream(t *testing.T) {
	buffer, err := NewOutputBuffer(1024, 10)
	if err != nil {
		t.Fatalf("NewOutputBuffer() error = %v", err)
	}
	buffer.now = func() time.Time { return time.Time{} }
	writer, err := buffer.Writer(OutputStreamStderr)
	if err != nil {
		t.Fatalf("Writer() error = %v", err)
	}
	if _, err := writer.Write([]byte("failure")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	tail, err := buffer.Tail(0, 1)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	if len(tail.Events) != 1 || tail.Events[0].Stream != OutputStreamStderr {
		t.Fatalf("events = %#v, want one stderr event", tail.Events)
	}
}
