// Package task provides background task management for Poyo.
package task

import (
	"context"
	"sync"
	"time"
)

// TaskState represents the state of a task
type TaskState string

const (
	StatePending   TaskState = "pending"
	StateRunning   TaskState = "running"
	StateCompleted TaskState = "completed"
	StateFailed    TaskState = "failed"
	StateStopped   TaskState = "stopped"
)

// TaskType represents the type of a background task
type TaskType string

const (
	TypeBash   TaskType = "bash"
	TypeAgent  TaskType = "agent"
	TypeHook   TaskType = "hook"
	TypeCustom TaskType = "custom"
)

// Task represents a background task
type Task struct {
	// ID is the unique task identifier
	ID string `json:"id"`

	// Type is the task type
	Type TaskType `json:"type"`

	// Name is a human-readable name
	Name string `json:"name"`

	// Description is a short description
	Description string `json:"description,omitempty"`

	// State is the current state
	State TaskState `json:"state"`

	// Command is the command being executed (for bash tasks)
	Command string `json:"command,omitempty"`

	// WorkingDir is the working directory
	WorkingDir string `json:"workingDir,omitempty"`

	// Output is the accumulated output
	Output string `json:"output,omitempty"`

	// Error is the error message if failed
	Error string `json:"error,omitempty"`

	// ExitCode is the exit code (for bash tasks)
	ExitCode int `json:"exitCode,omitempty"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"createdAt"`

	// StartedAt is when execution started
	StartedAt time.Time `json:"startedAt,omitempty"`

	// CompletedAt is when execution finished
	CompletedAt time.Time `json:"completedAt,omitempty"`

	// Metadata holds additional task data
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// cancel is the cancellation function
	cancel context.CancelFunc `json:"-"`

	// outputChan is the output streaming channel
	outputChan chan string `json:"-"`

	// doneChan signals task completion
	doneChan chan struct{} `json:"-"`
}

// Duration returns the task duration
func (t *Task) Duration() time.Duration {
	if t.StartedAt.IsZero() {
		return 0
	}
	if t.CompletedAt.IsZero() {
		return time.Since(t.StartedAt)
	}
	return t.CompletedAt.Sub(t.StartedAt)
}

// IsComplete returns true if the task is in a terminal state
func (t *Task) IsComplete() bool {
	return t.State == StateCompleted || t.State == StateFailed || t.State == StateStopped
}

// OutputChannel returns the output streaming channel
func (t *Task) OutputChannel() <-chan string {
	return t.outputChan
}

// DoneChannel returns the completion signal channel
func (t *Task) DoneChannel() <-chan struct{} {
	return t.doneChan
}

// TaskOutput represents a chunk of task output
type TaskOutput struct {
	TaskID    string    `json:"taskId"`
	Output    string    `json:"output"`
	IsError   bool      `json:"isError,omitempty"`
	IsDone    bool      `json:"isDone,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskEvent represents a task state change event
type TaskEvent struct {
	TaskID    string    `json:"taskId"`
	Type      string    `json:"type"` // "created", "started", "output", "completed", "failed", "stopped"
	State     TaskState `json:"state"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskEventHandler handles task events
type TaskEventHandler func(event TaskEvent)

// TaskOptions contains options for creating a task
type TaskOptions struct {
	Type        TaskType
	Name        string
	Description string
	Command     string
	WorkingDir  string
	Timeout     time.Duration
	Metadata    map[string]interface{}
}

// Executor is the interface for task execution
type Executor interface {
	Execute(ctx context.Context, task *Task) error
}
