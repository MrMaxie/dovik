//go:build !dovik_dev_harness

package main

func applyDaemonOverrides(configuration daemonConfiguration) (daemonConfiguration, error) {
	return configuration, nil
}
