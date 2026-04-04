package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// OutputTool provides the TaskOutput tool for retrieving background task output
type OutputTool struct {
	manager *Manager
}

// NewOutputTool creates a new task output tool
func NewOutputTool(manager *Manager) *OutputTool {
	return &OutputTool{manager: manager}
}

// Name returns the tool name
func (t *OutputTool) Name() string {
	return "TaskOutput"
}

// Description returns the tool description
func (t *OutputTool) Description() string {
	return `Retrieve output from a running or completed background task.

This tool allows you to check the status and output of background tasks
that were started with run_in_background=true in the Bash tool.

Usage:
- Provide a task_id to get the current output and status
- Set block=true to wait for task completion
- Use timeout to limit how long to wait`
}

// InputSchema returns the JSON schema for the tool input
func (t *OutputTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the task to retrieve output from",
			},
			"block": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to wait for the task to complete (default false)",
				"default":     false,
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Maximum time to wait in milliseconds (default 30000)",
				"default":     30000,
			},
		},
		"required": []string{"task_id"},
	}
}

// Execute executes the tool
func (t *OutputTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		TaskID  string `json:"task_id"`
		Block   bool   `json:"block"`
		Timeout int    `json:"timeout"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	timeout := time.Duration(params.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Get the task
	task, ok := t.manager.Get(params.TaskID)
	if !ok {
		return nil, fmt.Errorf("task %s not found", params.TaskID)
	}

	// If blocking, wait for completion
	if params.Block && !task.IsComplete() {
		var err error
		task, err = t.manager.Wait(params.TaskID, timeout)
		if err != nil {
			return nil, err
		}
	}

	// Build result
	result := &TaskOutputResult{
		TaskID:    task.ID,
		State:     task.State,
		Output:    task.Output,
		Error:     task.Error,
		ExitCode:  task.ExitCode,
		Duration:  task.Duration().String(),
		IsComplete: task.IsComplete(),
	}

	return result, nil
}

// TaskOutputResult represents the result of the TaskOutput tool
type TaskOutputResult struct {
	TaskID     string    `json:"task_id"`
	State      TaskState `json:"state"`
	Output     string    `json:"output"`
	Error      string    `json:"error,omitempty"`
	ExitCode   int       `json:"exitCode,omitempty"`
	Duration   string    `json:"duration"`
	IsComplete bool      `json:"isComplete"`
}

// ListTool provides a tool to list all tasks
type ListTool struct {
	manager *Manager
}

// NewListTool creates a new task list tool
func NewListTool(manager *Manager) *ListTool {
	return &ListTool{manager: manager}
}

// Name returns the tool name
func (t *ListTool) Name() string {
	return "TaskList"
}

// Description returns the tool description
func (t *ListTool) Description() string {
	return `List all background tasks.

Returns a list of all tasks with their current state, output preview, and metadata.`
}

// InputSchema returns the JSON schema for the tool input
func (t *ListTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"state": map[string]interface{}{
				"type":        "string",
				"description": "Filter by task state (pending, running, completed, failed, stopped)",
				"enum":        []string{"pending", "running", "completed", "failed", "stopped"},
			},
		},
	}
}

// Execute executes the tool
func (t *ListTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		State TaskState `json:"state"`
	}

	// Parse input (optional)
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			// Ignore parse errors for optional parameters
		}
	}

	var tasks []*Task
	if params.State != "" {
		tasks = t.manager.ListByState(params.State)
	} else {
		tasks = t.manager.List()
	}

	// Build result
	result := make([]*TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		summary := &TaskSummary{
			ID:          task.ID,
			Type:        task.Type,
			Name:        task.Name,
			State:       task.State,
			Duration:    task.Duration().String(),
			OutputPreview: truncateString(task.Output, 100),
		}
		if !task.StartedAt.IsZero() {
			summary.StartedAt = task.StartedAt.Format(time.RFC3339)
		}
		if !task.CompletedAt.IsZero() {
			summary.CompletedAt = task.CompletedAt.Format(time.RFC3339)
		}
		result = append(result, summary)
	}

	return result, nil
}

// TaskSummary represents a summary of a task
type TaskSummary struct {
	ID            string    `json:"id"`
	Type          TaskType  `json:"type"`
	Name          string    `json:"name"`
	State         TaskState `json:"state"`
	Duration      string    `json:"duration"`
	OutputPreview string    `json:"outputPreview,omitempty"`
	StartedAt     string    `json:"startedAt,omitempty"`
	CompletedAt   string    `json:"completedAt,omitempty"`
}

// StopTool provides a tool to stop a running task
type StopTool struct {
	manager *Manager
}

// NewStopTool creates a new task stop tool
func NewStopTool(manager *Manager) *StopTool {
	return &StopTool{manager: manager}
}

// Name returns the tool name
func (t *StopTool) Name() string {
	return "TaskStop"
}

// Description returns the tool description
func (t *StopTool) Description() string {
	return `Stop a running background task.

Sends a cancellation signal to a running task and waits for it to terminate.`
}

// InputSchema returns the JSON schema for the tool input
func (t *StopTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the task to stop",
			},
		},
		"required": []string{"task_id"},
	}
}

// Execute executes the tool
func (t *StopTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := t.manager.Stop(params.TaskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Task %s stopped", params.TaskID),
	}, nil
}

// truncateString truncates a string
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
