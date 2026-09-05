package cli

import (
	"errors"
	"os"
	"runtime"

	"github.com/MrMaxie/dovik/internal/daemonlaunch"
	"github.com/MrMaxie/dovik/internal/tui"
)

func productionTUIDaemonLauncher(endpoint, defaultEndpoint string) (tui.DaemonLauncher, error) {
	if runtime.GOOS != "windows" || endpoint != defaultEndpoint {
		return nil, nil
	}
	launcher, err := daemonlaunch.NewSibling()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return launcher, err
}
