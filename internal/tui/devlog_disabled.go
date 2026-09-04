//go:build !dovik_dev_harness

package tui

func devLog(string, ...any) {}

func closeDevLog() {}
