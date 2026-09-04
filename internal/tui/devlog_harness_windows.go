//go:build dovik_dev_harness && windows

package tui

import (
	"io"
	"time"

	"github.com/Microsoft/go-winio"
)

func dialDevLog(endpoint string) (io.ReadWriteCloser, error) {
	return winio.DialPipe(endpoint, ptrTo(2*time.Second))
}

func ptrTo[T any](value T) *T {
	return &value
}
