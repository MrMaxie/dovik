//go:build !windows

package daemonlaunch

import "errors"

func startDetached(string, []string, []string) (int, error) {
	return 0, errors.New("daemon launch is supported only on native Windows")
}
