//go:build dovik_dev_harness && !windows

package tui

import (
	"io"
	"net"
	"time"
)

func dialDevLog(endpoint string) (io.ReadWriteCloser, error) {
	return net.DialTimeout("unix", endpoint, 2*time.Second)
}
