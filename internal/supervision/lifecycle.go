package supervision

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

const (
	TerminationReasonStartFailed       = "start-failed"
	TerminationReasonNaturalExit       = "natural-exit"
	TerminationReasonRequestedStop     = "requested-stop"
	TerminationReasonObservationFailed = "observation-failed"
)

type registryPersister interface {
	Save(*Registry) error
}

type processKey struct {
	projectID ProjectID
	processID ProcessID
}

type managedRuntime struct {
	instanceID RuntimeInstanceID
	process    *ownedProcess
	output     *OutputBuffer
	done       chan struct{}
	finishErr  error
}

// LifecycleManager owns live operating-system processes for one daemon.
type LifecycleManager struct {
	mu               sync.Mutex
	registry         *Registry
	persister        registryPersister
	active           map[processKey]*managedRuntime
	outputs          map[processKey]*managedRuntime
	outputByteLimit  int
	outputEventLimit int
	now              func() time.Time
	newInstanceID    func() (RuntimeInstanceID, error)
}

// NewLifecycleManager creates a daemon lifecycle service.
func NewLifecycleManager(registry *Registry, persister registryPersister, outputByteLimit int, outputEventLimit int) (*LifecycleManager, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	if persister == nil {
		return nil, fmt.Errorf("registry persister is nil")
	}
	if outputByteLimit <= 0 || outputEventLimit <= 0 {
		return nil, fmt.Errorf("output limits must be positive")
	}
	return &LifecycleManager{
		registry:         registry,
		persister:        persister,
		active:           make(map[processKey]*managedRuntime),
		outputs:          make(map[processKey]*managedRuntime),
		outputByteLimit:  outputByteLimit,
		outputEventLimit: outputEventLimit,
		now:              func() time.Time { return time.Now().UTC() },
		newInstanceID:    randomRuntimeInstanceID,
	}, nil
}

// Start creates one runtime or returns the existing active runtime.
func (manager *LifecycleManager) Start(projectID ProjectID, processID ProcessID) (ProcessRuntime, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	key := processKey{projectID: projectID, processID: processID}
	project, process, err := manager.findProcess(key)
	if err != nil {
		return ProcessRuntime{}, err
	}
	if process.Runtime != nil && process.Runtime.State.IsActive() {
		return cloneRuntime(*process.Runtime), nil
	}

	instanceID, err := manager.newInstanceID()
	if err != nil {
		return ProcessRuntime{}, fmt.Errorf("create runtime instance ID: %w", err)
	}
	runtimeState := NewProcessRuntime(projectID, processID, instanceID)
	previousRuntime := process.Runtime
	process.Runtime = &runtimeState
	if err := manager.persister.Save(manager.registry); err != nil {
		process.Runtime = previousRuntime
		return ProcessRuntime{}, fmt.Errorf("persist starting runtime: %w", err)
	}

	output, err := NewOutputBuffer(manager.outputByteLimit, manager.outputEventLimit)
	if err != nil {
		return ProcessRuntime{}, err
	}
	stdout, _ := output.Writer(OutputStreamStdout)
	stderr, _ := output.Writer(OutputStreamStderr)
	definition := process.Definition
	if definition.WorkingDirectory == "" {
		definition.WorkingDirectory = project.Definition.RootDirectory
	} else if !filepath.IsAbs(definition.WorkingDirectory) {
		definition.WorkingDirectory = filepath.Join(project.Definition.RootDirectory, definition.WorkingDirectory)
	}
	command := newCommand(definition)
	command.Stdout = stdout
	command.Stderr = stderr
	owned, err := startOwnedProcess(command)
	if err != nil {
		persistErr := manager.failStart(process)
		return cloneRuntime(*process.Runtime), errors.Join(fmt.Errorf("start process: %w", err), persistErr)
	}

	pid := owned.PID()
	startedAt := manager.now()
	process.Runtime.PID = &pid
	process.Runtime.StartedAt = &startedAt
	if err := process.Runtime.Transition(ProcessStateRunning); err != nil {
		_ = owned.platform.Kill()
		_ = owned.Wait()
		manager.failStart(process)
		return cloneRuntime(*process.Runtime), err
	}
	if err := manager.persister.Save(manager.registry); err != nil {
		_ = owned.platform.Kill()
		_ = owned.Wait()
		manager.failRunningProcess(process)
		return cloneRuntime(*process.Runtime), fmt.Errorf("persist running runtime: %w", err)
	}

	managed := &managedRuntime{
		instanceID: instanceID,
		process:    owned,
		output:     output,
		done:       make(chan struct{}),
	}
	manager.active[key] = managed
	manager.outputs[key] = managed
	go manager.observeExit(key, managed)
	return cloneRuntime(*process.Runtime), nil
}

// Stop idempotently stops an active runtime and waits for its terminal state.
func (manager *LifecycleManager) Stop(ctx context.Context, projectID ProjectID, processID ProcessID) (ProcessRuntime, error) {
	key := processKey{projectID: projectID, processID: processID}
	manager.mu.Lock()
	_, process, err := manager.findProcess(key)
	if err != nil {
		manager.mu.Unlock()
		return ProcessRuntime{}, err
	}
	if process.Runtime == nil || !process.Runtime.State.IsActive() {
		result := ProcessRuntime{ProjectID: projectID, ProcessID: processID, State: ProcessStateStopped}
		if process.Runtime != nil {
			result = cloneRuntime(*process.Runtime)
		}
		manager.mu.Unlock()
		return result, nil
	}
	managed := manager.active[key]
	if managed == nil || managed.instanceID != process.Runtime.InstanceID {
		manager.mu.Unlock()
		return ProcessRuntime{}, fmt.Errorf("active runtime %q is not owned by this daemon", process.Runtime.InstanceID)
	}
	requestStop := process.Runtime.State != ProcessStateStopping
	if requestStop {
		previousState := process.Runtime.State
		if err := process.Runtime.Transition(ProcessStateStopping); err != nil {
			manager.mu.Unlock()
			return ProcessRuntime{}, err
		}
		if err := manager.persister.Save(manager.registry); err != nil {
			process.Runtime.State = previousState
			manager.mu.Unlock()
			return ProcessRuntime{}, fmt.Errorf("persist stopping runtime: %w", err)
		}
	}
	manager.mu.Unlock()

	if requestStop {
		if err := managed.process.platform.RequestStop(); err != nil {
			return manager.statusWithError(key, fmt.Errorf("request process stop: %w", err))
		}
	}
	select {
	case <-managed.done:
		return manager.statusWithError(key, managed.finishErr)
	case <-ctx.Done():
		_ = managed.process.platform.Kill()
		return manager.statusWithError(key, ctx.Err())
	}
}

// Restart waits for complete termination before creating a new runtime.
func (manager *LifecycleManager) Restart(ctx context.Context, projectID ProjectID, processID ProcessID) (ProcessRuntime, error) {
	if _, err := manager.Stop(ctx, projectID, processID); err != nil {
		return ProcessRuntime{}, err
	}
	return manager.Start(projectID, processID)
}

// Shutdown stops every runtime currently owned by the daemon.
func (manager *LifecycleManager) Shutdown(ctx context.Context) error {
	manager.mu.Lock()
	keys := make([]processKey, 0, len(manager.active))
	for key := range manager.active {
		keys = append(keys, key)
	}
	manager.mu.Unlock()

	var shutdownError error
	for _, key := range keys {
		if _, err := manager.Stop(ctx, key.projectID, key.processID); err != nil {
			shutdownError = errors.Join(shutdownError, err)
		}
	}
	return shutdownError
}

// Status returns the current or most recent runtime.
func (manager *LifecycleManager) Status(projectID ProjectID, processID ProcessID) (ProcessRuntime, bool, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	_, process, err := manager.findProcess(processKey{projectID: projectID, processID: processID})
	if err != nil {
		return ProcessRuntime{}, false, err
	}
	if process.Runtime == nil {
		return ProcessRuntime{}, false, nil
	}
	return cloneRuntime(*process.Runtime), true, nil
}

// Logs returns output for the current or most recent runtime in this daemon.
func (manager *LifecycleManager) Logs(projectID ProjectID, processID ProcessID, afterSequence uint64, limit int) (OutputTail, error) {
	manager.mu.Lock()
	output := manager.outputs[processKey{projectID: projectID, processID: processID}]
	manager.mu.Unlock()
	if output == nil {
		return OutputTail{}, nil
	}
	return output.output.Tail(afterSequence, limit)
}

func (manager *LifecycleManager) observeExit(key processKey, managed *managedRuntime) {
	exit := managed.process.Wait()
	manager.mu.Lock()
	defer manager.mu.Unlock()
	defer close(managed.done)

	_, process, err := manager.findProcess(key)
	if err != nil {
		managed.finishErr = err
		return
	}
	if process.Runtime == nil || process.Runtime.InstanceID != managed.instanceID {
		managed.finishErr = fmt.Errorf("runtime ownership changed before exit")
		return
	}

	finishedAt := manager.now()
	process.Runtime.FinishedAt = &finishedAt
	if exit.Code >= 0 {
		exitCode := exit.Code
		process.Runtime.ExitCode = &exitCode
	}
	var next ProcessState
	switch {
	case exit.Err != nil:
		next = ProcessStateFailed
		process.Runtime.TerminationReason = TerminationReasonObservationFailed
	case process.Runtime.State == ProcessStateStopping:
		next = ProcessStateStopped
		process.Runtime.TerminationReason = TerminationReasonRequestedStop
	case exit.Code == 0:
		next = ProcessStateExited
		process.Runtime.TerminationReason = TerminationReasonNaturalExit
	default:
		next = ProcessStateFailed
		process.Runtime.TerminationReason = TerminationReasonNaturalExit
	}
	if err := process.Runtime.Transition(next); err != nil {
		managed.finishErr = err
	} else if err := manager.persister.Save(manager.registry); err != nil {
		managed.finishErr = fmt.Errorf("persist terminal runtime: %w", err)
	}
	delete(manager.active, key)
}

func (manager *LifecycleManager) findProcess(key processKey) (*RegisteredProject, *RegisteredProcess, error) {
	project := manager.registry.Projects[key.projectID]
	if project == nil {
		return nil, nil, fmt.Errorf("project %q not found", key.projectID)
	}
	process := project.Processes[key.processID]
	if process == nil {
		return nil, nil, fmt.Errorf("process %q not found in project %q", key.processID, key.projectID)
	}
	return project, process, nil
}

func (manager *LifecycleManager) failStart(process *RegisteredProcess) error {
	finishedAt := manager.now()
	process.Runtime.FinishedAt = &finishedAt
	process.Runtime.TerminationReason = TerminationReasonStartFailed
	if err := process.Runtime.Transition(ProcessStateFailed); err != nil {
		return err
	}
	if err := manager.persister.Save(manager.registry); err != nil {
		return fmt.Errorf("persist failed runtime: %w", err)
	}
	return nil
}

func (manager *LifecycleManager) failRunningProcess(process *RegisteredProcess) {
	finishedAt := manager.now()
	process.Runtime.FinishedAt = &finishedAt
	process.Runtime.TerminationReason = TerminationReasonObservationFailed
	_ = process.Runtime.Transition(ProcessStateFailed)
	_ = manager.persister.Save(manager.registry)
}

func (manager *LifecycleManager) statusWithError(key processKey, operationError error) (ProcessRuntime, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	_, process, lookupError := manager.findProcess(key)
	if lookupError != nil {
		return ProcessRuntime{}, errors.Join(operationError, lookupError)
	}
	if process.Runtime == nil {
		return ProcessRuntime{}, operationError
	}
	return cloneRuntime(*process.Runtime), operationError
}

func cloneRuntime(runtimeState ProcessRuntime) ProcessRuntime {
	runtimeState.PID = cloneInt(runtimeState.PID)
	runtimeState.StartedAt = cloneTime(runtimeState.StartedAt)
	runtimeState.FinishedAt = cloneTime(runtimeState.FinishedAt)
	runtimeState.ExitCode = cloneInt(runtimeState.ExitCode)
	return runtimeState
}

func randomRuntimeInstanceID() (RuntimeInstanceID, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return RuntimeInstanceID(fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	)), nil
}
