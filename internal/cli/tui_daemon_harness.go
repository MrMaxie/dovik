//go:build dovik_dev_harness

package cli

import (
	"fmt"
	"os"

	"github.com/MrMaxie/dovik/internal/daemonlaunch"
	"github.com/MrMaxie/dovik/internal/tui"
)

const (
	harnessDaemonExecutableEnvironment = "DOVIK_TUI_HARNESS_DAEMON_EXECUTABLE"
	harnessControlEndpointEnvironment  = "DOVIK_TUI_HARNESS_CONTROL_ENDPOINT"
	harnessRegistryPathEnvironment     = "DOVIK_TUI_HARNESS_REGISTRY_PATH"
)

func resolveTUIDaemonLauncher(endpoint, defaultEndpoint string) (tui.DaemonLauncher, error) {
	harnessEndpoint := os.Getenv(harnessControlEndpointEnvironment)
	executable := os.Getenv(harnessDaemonExecutableEnvironment)
	registryPath := os.Getenv(harnessRegistryPathEnvironment)
	if harnessEndpoint == "" && executable == "" && registryPath == "" {
		return productionTUIDaemonLauncher(endpoint, defaultEndpoint)
	}
	if harnessEndpoint == "" || executable == "" || registryPath == "" {
		return nil, fmt.Errorf("incomplete TUI harness daemon configuration")
	}
	if endpoint != harnessEndpoint {
		return nil, nil
	}
	return daemonlaunch.New(executable, nil, append(os.Environ(),
		harnessControlEndpointEnvironment+"="+harnessEndpoint,
		harnessRegistryPathEnvironment+"="+registryPath,
	))
}
