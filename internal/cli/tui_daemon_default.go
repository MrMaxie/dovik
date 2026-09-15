package cli

import "github.com/MrMaxie/dovik/internal/tui"

func resolveTUIDaemonLauncher(endpoint, defaultEndpoint string) (tui.DaemonLauncher, error) {
	return productionTUIDaemonLauncher(endpoint, defaultEndpoint)
}
