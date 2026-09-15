//go:build !dovik_ttyglass

package main

func applyDaemonOverrides(configuration daemonConfiguration) (daemonConfiguration, error) {
	return configuration, nil
}
