//go:build dovik_dev_harness

package main

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	harnessControlEndpointEnvironment = "DOVIK_TUI_HARNESS_CONTROL_ENDPOINT"
	harnessRegistryPathEnvironment    = "DOVIK_TUI_HARNESS_REGISTRY_PATH"
)

func applyDaemonOverrides(configuration daemonConfiguration) (daemonConfiguration, error) {
	endpoint := os.Getenv(harnessControlEndpointEnvironment)
	registryPath := os.Getenv(harnessRegistryPathEnvironment)
	if endpoint == "" && registryPath == "" {
		return configuration, nil
	}
	if endpoint == "" || registryPath == "" {
		return daemonConfiguration{}, errors.New("incomplete TUI harness daemon configuration")
	}
	if !filepath.IsAbs(registryPath) {
		return daemonConfiguration{}, errors.New("TUI harness registry path must be absolute")
	}
	return daemonConfiguration{endpoint: endpoint, registryPath: registryPath}, nil
}
