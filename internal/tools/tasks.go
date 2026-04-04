// Package tools implements the TodoWrite tool for task management
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kirbchino/poyo/internal/prompt"
)

// TodoWriteTool implements the TodoWrite tool for managing task lists
type TodoWriteTool struct {
	BaseTool
	todos  []TodoItem
	mu     sync.RWMutex
}

// TodoItem represents a single todo item
type TodoItem struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Status      string    `json:"status"` // pending, in_progress, completed
	ActiveForm  string    `json:"activeForm,omitempty"`
	Priority    int       `json:"priority,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// NewTodoWriteTool creates a new TodoWrite tool
func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{
		BaseTool: BaseTool{
			name:              "TodoWrite",
			description:       prompt.GetToolDescription("TodoWrite"),
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		todos: make([]TodoItem, 0),
	}
}

// TodoWriteInput represents input for the TodoWrite tool
type TodoWriteInput struct {
	Todos []TodoItemInput `json:"todos"`
}

// TodoItemInput represents input for a single todo item
type TodoItemInput struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm,omitempty"`
}

// Call executes the TodoWrite tool
func (t *TodoWriteTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for TodoWrite tool")
	}

	todosRaw, ok := input["todos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("todos array is required")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// Process each todo item
	for i, todoRaw := range todosRaw {
		todoMap, ok := todoRaw.(map[string]interface{})
		if !ok {
			continue
		}

		content, _ := todoMap["content"].(string)
		status, _ := todoMap["status"].(string)
		activeForm, _ := todoMap["activeForm"].(string)

		// Find existing todo or create new one
		if i < len(t.todos) {
			// Update existing
			t.todos[i].Content = content
			t.todos[i].Status = status
			t.todos[i].ActiveForm = activeForm
			t.todos[i].UpdatedAt = now

			if status == "completed" && t.todos[i].CompletedAt == nil {
				t.todos[i].CompletedAt = &now
			}
		} else {
			// Create new
			t.todos = append(t.todos, TodoItem{
				ID:         generateTodoID(),
				Content:    content,
				Status:     status,
				ActiveForm: activeForm,
				CreatedAt:  now,
				UpdatedAt:  now,
			})
		}
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"todos":    t.todos,
			"message":  fmt.Sprintf("Updated %d todo items", len(todosRaw)),
		},
	}, nil
}

// GetTodos returns the current todo list
func (t *TodoWriteTool) GetTodos() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TodoItem, len(t.todos))
	copy(result, t.todos)
	return result
}

// AddTodo adds a new todo item
func (t *TodoWriteTool) AddTodo(content, activeForm string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.todos = append(t.todos, TodoItem{
		ID:         generateTodoID(),
		Content:    content,
		Status:     "pending",
		ActiveForm: activeForm,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
}

// UpdateTodoStatus updates the status of a todo item
func (t *TodoWriteTool) UpdateTodoStatus(id, status string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if t.todos[i].ID == id {
			t.todos[i].Status = status
			t.todos[i].UpdatedAt = time.Now()

			if status == "completed" {
				now := time.Now()
				t.todos[i].CompletedAt = &now
			}
			return nil
		}
	}

	return fmt.Errorf("todo item not found: %s", id)
}

// InputSchema returns the input schema for the TodoWrite tool
func (t *TodoWriteTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"todos": {
				"type":        "array",
				"description": "List of todo items with content and status",
			},
		},
		Required: []string{"todos"},
	}
}

// generateTodoID generates a unique todo ID
func generateTodoID() string {
	return fmt.Sprintf("todo_%d", time.Now().UnixNano())
}

// generateUUID generates a unique identifier
func generateUUID() string {
	return fmt.Sprintf("uuid_%d", time.Now().UnixNano())
}

// TaskOutputTool implements the TaskOutput tool for getting background task output
type TaskOutputTool struct {
	BaseTool
	tasks map[string]*BackgroundTask
	mu    sync.RWMutex
}

// BackgroundTask represents a background task
type BackgroundTask struct {
	ID        string
	Status    string
	Output    string
	Error     string
	StartTime time.Time
	EndTime   *time.Time
}

// NewTaskOutputTool creates a new TaskOutput tool
func NewTaskOutputTool() *TaskOutputTool {
	return &TaskOutputTool{
		BaseTool: BaseTool{
			name:              "TaskOutput",
			description:       prompt.GetToolDescription("TaskOutput"),
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		tasks: make(map[string]*BackgroundTask),
	}
}

// TaskOutputInput represents input for the TaskOutput tool
type TaskOutputInput struct {
	TaskID string `json:"task_id"`
	Block  bool   `json:"block"`
	Timeout int   `json:"timeout,omitempty"`
}

// Call executes the TaskOutput tool
func (t *TaskOutputTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for TaskOutput tool")
	}

	taskID, _ := input["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	t.mu.RLock()
	task, exists := t.tasks[taskID]
	t.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  task.Status,
			"output":  task.Output,
			"error":   task.Error,
		},
	}, nil
}

// RegisterTask registers a new background task
func (t *TaskOutputTool) RegisterTask(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.tasks[id] = &BackgroundTask{
		ID:        id,
		Status:    "running",
		StartTime: time.Now(),
	}
}

// CompleteTask marks a task as completed
func (t *TaskOutputTool) CompleteTask(id, output string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task, exists := t.tasks[id]; exists {
		task.Status = "completed"
		task.Output = output
		now := time.Now()
		task.EndTime = &now
	}
}

// FailTask marks a task as failed
func (t *TaskOutputTool) FailTask(id, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task, exists := t.tasks[id]; exists {
		task.Status = "failed"
		task.Error = errMsg
		now := time.Now()
		task.EndTime = &now
	}
}

// InputSchema returns the input schema for the TaskOutput tool
func (t *TaskOutputTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"task_id": {
				"type":        "string",
				"description": "The ID of the background task",
			},
			"block": {
				"type":        "boolean",
				"description": "Whether to wait for task completion",
			},
			"timeout": {
				"type":        "integer",
				"description": "Maximum time to wait in milliseconds",
			},
		},
		Required: []string{"task_id"},
	}
}

// TaskStopTool implements the TaskStop tool for stopping background tasks
type TaskStopTool struct {
	BaseTool
	taskOutput *TaskOutputTool
}

// NewTaskStopTool creates a new TaskStop tool
func NewTaskStopTool(taskOutput *TaskOutputTool) *TaskStopTool {
	return &TaskStopTool{
		BaseTool: BaseTool{
			name:              "TaskStop",
			description:       prompt.GetToolDescription("TaskStop"),
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		taskOutput: taskOutput,
	}
}

// Call executes the TaskStop tool
func (t *TaskStopTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for TaskStop tool")
	}

	taskID, _ := input["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	t.taskOutput.FailTask(taskID, "Task stopped by user")

	return &ToolResult{
		Data: map[string]interface{}{
			"task_id": taskID,
			"status":  "stopped",
			"message": "Task has been stopped",
		},
	}, nil
}

// InputSchema returns the input schema
func (t *TaskStopTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"task_id": {
				"type":        "string",
				"description": "The ID of the task to stop",
			},
		},
		Required: []string{"task_id"},
	}
}
