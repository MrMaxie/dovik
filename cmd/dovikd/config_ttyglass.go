//go:build dovik_ttyglass

package main

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	ttyglassControlEndpointEnvironment = "DOVIK_TTYGLASS_CONTROL_ENDPOINT"
	ttyglassRegistryPathEnvironment    = "DOVIK_TTYGLASS_REGISTRY_PATH"
)

func applyDaemonOverrides(configuration daemonConfiguration) (daemonConfiguration, error) {
	endpoint := os.Getenv(ttyglassControlEndpointEnvironment)
	registryPath := os.Getenv(ttyglassRegistryPathEnvironment)
	if endpoint == "" && registryPath == "" {
		return configuration, nil
	}
	if endpoint == "" || registryPath == "" {
		return daemonConfiguration{}, errors.New("incomplete ttyglass fixture configuration")
	}
	if !filepath.IsAbs(registryPath) {
		return daemonConfiguration{}, errors.New("ttyglass fixture registry path must be absolute")
	}
	return daemonConfiguration{endpoint: endpoint, registryPath: registryPath}, nil
}
