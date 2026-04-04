package task

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager manages background tasks
type Manager struct {
	mu       sync.RWMutex
	tasks    map[string]*Task
	handlers []TaskEventHandler
	notify   chan TaskEvent
}

// NewManager creates a new task manager
func NewManager() *Manager {
	m := &Manager{
		tasks:  make(map[string]*Task),
		notify: make(chan TaskEvent, 100),
	}

	// Start event dispatcher
	go m.dispatchEvents()

	return m
}

// dispatchEvents dispatches events to all handlers
func (m *Manager) dispatchEvents() {
	for event := range m.notify {
		m.mu.RLock()
		handlers := make([]TaskEventHandler, len(m.handlers))
		copy(handlers, m.handlers)
		m.mu.RUnlock()

		for _, h := range handlers {
			h(event)
		}
	}
}

// RegisterHandler registers a task event handler
func (m *Manager) RegisterHandler(handler TaskEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Create creates a new task
func (m *Manager) Create(opts TaskOptions) *Task {
	id := generateTaskID()

	task := &Task{
		ID:          id,
		Type:        opts.Type,
		Name:        opts.Name,
		Description: opts.Description,
		Command:     opts.Command,
		WorkingDir:  opts.WorkingDir,
		State:       StatePending,
		CreatedAt:   time.Now(),
		Metadata:    opts.Metadata,
		outputChan:  make(chan string, 100),
		doneChan:    make(chan struct{}),
	}

	m.mu.Lock()
	m.tasks[id] = task
	m.mu.Unlock()

	m.notify <- TaskEvent{
		TaskID:    id,
		Type:      "created",
		State:     StatePending,
		Timestamp: time.Now(),
	}

	return task
}

// Start starts a task with the given executor
func (m *Manager) Start(ctx context.Context, taskID string, executor Executor) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.State != StatePending {
		m.mu.Unlock()
		return fmt.Errorf("task %s is not in pending state", taskID)
	}

	task.State = StateRunning
	task.StartedAt = time.Now()
	m.mu.Unlock()

	m.notify <- TaskEvent{
		TaskID:    taskID,
		Type:      "started",
		State:     StateRunning,
		Timestamp: time.Now(),
	}

	// Create cancellable context
	execCtx, cancel := context.WithCancel(ctx)
	task.cancel = cancel

	// Execute in background
	go func() {
		defer close(task.doneChan)
		defer close(task.outputChan)

		err := executor.Execute(execCtx, task)

		m.mu.Lock()
		if err != nil {
			task.State = StateFailed
			task.Error = err.Error()
		} else {
			task.State = StateCompleted
		}
		task.CompletedAt = time.Now()
		m.mu.Unlock()

		eventType := "completed"
		if err != nil {
			eventType = "failed"
		}

		m.notify <- TaskEvent{
			TaskID:    taskID,
			Type:      eventType,
			State:     task.State,
			Error:     task.Error,
			Timestamp: time.Now(),
		}
	}()

	return nil
}

// Stop stops a running task
func (m *Manager) Stop(taskID string) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.State != StateRunning {
		m.mu.Unlock()
		return fmt.Errorf("task %s is not running", taskID)
	}

	if task.cancel != nil {
		task.cancel()
	}

	task.State = StateStopped
	task.CompletedAt = time.Now()
	m.mu.Unlock()

	m.notify <- TaskEvent{
		TaskID:    taskID,
		Type:      "stopped",
		State:     StateStopped,
		Timestamp: time.Now(),
	}

	return nil
}

// Get retrieves a task by ID
func (m *Manager) Get(taskID string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[taskID]
	return task, ok
}

// GetOutput retrieves output for a task
func (m *Manager) GetOutput(taskID string) (string, error) {
	m.mu.RLock()
	task, ok := m.tasks[taskID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("task %s not found", taskID)
	}

	return task.Output, nil
}

// AppendOutput appends output to a task
func (m *Manager) AppendOutput(taskID string, output string) {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	task.Output += output
	m.mu.Unlock()

	// Send to output channel
	select {
	case task.outputChan <- output:
	default:
		// Channel full, skip
	}

	m.notify <- TaskEvent{
		TaskID:    taskID,
		Type:      "output",
		Output:    output,
		Timestamp: time.Now(),
	}
}

// List returns all tasks
func (m *Manager) List() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// ListByState returns tasks filtered by state
func (m *Manager) ListByState(state TaskState) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*Task
	for _, task := range m.tasks {
		if task.State == state {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// ListRunning returns all running tasks
func (m *Manager) ListRunning() []*Task {
	return m.ListByState(StateRunning)
}

// Remove removes a task from the manager
func (m *Manager) Remove(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.State == StateRunning {
		return fmt.Errorf("cannot remove running task %s", taskID)
	}

	delete(m.tasks, taskID)
	return nil
}

// Clear removes all completed tasks
func (m *Manager) Clear() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, task := range m.tasks {
		if task.IsComplete() {
			delete(m.tasks, id)
			count++
		}
	}

	return count
}

// Wait waits for a task to complete
func (m *Manager) Wait(taskID string, timeout time.Duration) (*Task, error) {
	m.mu.RLock()
	task, ok := m.tasks[taskID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	if task.IsComplete() {
		return task, nil
	}

	select {
	case <-task.doneChan:
		m.mu.RLock()
		defer m.mu.RUnlock()
		return m.tasks[taskID], nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for task %s", taskID)
	}
}

// Watch returns a channel that receives task output
func (m *Manager) Watch(taskID string) (<-chan TaskOutput, error) {
	m.mu.RLock()
	task, ok := m.tasks[taskID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	ch := make(chan TaskOutput, 100)

	go func() {
		defer close(ch)

		for {
			select {
			case output, ok := <-task.outputChan:
				if !ok {
					// Task completed
					ch <- TaskOutput{
						TaskID:    taskID,
						IsDone:    true,
						Timestamp: time.Now(),
					}
					return
				}
				ch <- TaskOutput{
					TaskID:    taskID,
					Output:    output,
					Timestamp: time.Now(),
				}
			}
		}
	}()

	return ch, nil
}

// Count returns the total number of tasks
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tasks)
}

// CountByState returns the number of tasks in a specific state
func (m *Manager) CountByState(state TaskState) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, task := range m.tasks {
		if task.State == state {
			count++
		}
	}
	return count
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
