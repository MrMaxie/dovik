//go:build windows

package identity

import (
	"context"
	"fmt"
	"io"
)

func RunBridge(context.Context, io.Reader, io.Writer) error {
	return fmt.Errorf("the container bridge runs inside a Linux container")
}
