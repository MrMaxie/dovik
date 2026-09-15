//go:build linux

package identity

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

func peerUID(connection net.Conn) (uint32, error) {
	c, ok := connection.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("Unix connection required")
	}
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, err
	}
	var uid uint32
	var inner error
	err = raw.Control(func(fd uintptr) {
		cred, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		inner = e
		if e == nil {
			uid = cred.Uid
		}
	})
	if err != nil {
		return 0, err
	}
	return uid, inner
}
