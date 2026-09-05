package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (model Model) beginDaemonStart() (tea.Model, tea.Cmd) {
	if model.daemonLauncher == nil || model.daemonStarting || model.loading || model.pending {
		return model, nil
	}
	model.daemonStarting = true
	model.notice = "Starting daemon..."
	model.diagnostic = ""
	model.showDetail = false
	devLog("daemon.launch.requested")
	return model, model.daemonStartCmd()
}

func (model Model) daemonStartCmd() tea.Cmd {
	return func() tea.Msg {
		pid, launchErr := model.daemonLauncher.Start()
		if launchErr == nil {
			devLog("daemon.launch.started", "pid", pid)
		}

		ctx, cancel := context.WithTimeout(model.ctx, model.daemonReadyTimeout)
		defer cancel()
		var readinessErr error
		for {
			items, err := model.loadRegistry(ctx)
			if err == nil {
				return daemonLaunchMsg{items: items}
			}
			readinessErr = err
			timer := time.NewTimer(model.daemonRetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				if launchErr != nil {
					return daemonLaunchMsg{err: errors.Join(launchErr, readinessErr)}
				}
				return daemonLaunchMsg{err: fmt.Errorf("wait for daemon readiness: %w", readinessErr)}
			case <-timer.C:
			}
		}
	}
}
