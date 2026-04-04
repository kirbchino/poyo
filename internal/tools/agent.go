// Package tools implements the Agent tool for sub-agent execution
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kirbchino/poyo/internal/prompt"
)

// AgentExecutor defines the interface for executing sub-agents
type AgentExecutor interface {
	// ExecuteAgent executes a sub-agent with the given prompt
	ExecuteAgent(ctx context.Context, agentType string, agentPrompt string, maxTurns int) (string, error)
}

// AgentTool implements the Agent tool for launching sub-agents
type AgentTool struct {
	BaseTool
	executor      AgentExecutor
	taskOutput    *TaskOutputTool
	backgroundMu  sync.Mutex
	backgroundMap map[string]*BackgroundTask
}

// NewAgentTool creates a new Agent tool
func NewAgentTool() *AgentTool {
	return &AgentTool{
		BaseTool: BaseTool{
			name:              "Agent",
			description:       prompt.GetToolDescription("Agent"),
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		backgroundMap: make(map[string]*BackgroundTask),
	}
}

// SetExecutor sets the agent executor
func (t *AgentTool) SetExecutor(executor AgentExecutor) {
	t.executor = executor
}

// SetTaskOutput sets the task output tool for background tasks
func (t *AgentTool) SetTaskOutput(taskOutput *TaskOutputTool) {
	t.taskOutput = taskOutput
}

// AgentInput represents input for the Agent tool
type AgentInput struct {
	Prompt          string `json:"prompt"`
	Description     string `json:"description"`
	SubAgentType    string `json:"subagent_type,omitempty"`
	Model           string `json:"model,omitempty"`
	MaxTurns        int    `json:"max_turns,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Isolation       string `json:"isolation,omitempty"` // "worktree" for isolated execution
}

// AgentOutput represents output from the Agent tool
type AgentOutput struct {
	Result       string        `json:"result"`
	AgentID      string        `json:"agent_id"`
	Turns        int           `json:"turns"`
	Duration     time.Duration `json:"duration"`
	AgentType    string        `json:"agent_type"`
	IsBackground bool          `json:"is_background,omitempty"`
}

// ProgressEvent represents a progress update during tool execution
type ProgressEvent struct {
	Type     string `json:"type"`     // "start", "progress", "complete", "error"
	Message  string `json:"message"`  // Human-readable progress message
	Progress int    `json:"progress"` // Progress percentage (0-100)
	Error    string `json:"error,omitempty"`
}

// Call executes the Agent tool
func (t *AgentTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for Agent tool")
	}

	agentPrompt, _ := input["prompt"].(string)
	if agentPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	description, _ := input["description"].(string)
	agentType, _ := input["subagent_type"].(string)
	if agentType == "" {
		agentType = "general-purpose"
	}

	// model can be specified but is currently passed through the executor context
	// Reserved for future use when model selection is implemented
	_, _ = input["model"].(string)
	maxTurns, _ := input["max_turns"].(int)
	if maxTurns == 0 {
		maxTurns = 10
	}
	if maxTurns > 50 {
		maxTurns = 50
	}

	runInBackground, _ := input["run_in_background"].(bool)
	isolation, _ := input["isolation"].(string)

	agentID := generateAgentID()
	startTime := time.Now()

	// Get appropriate system prompt for agent type
	systemPrompt := t.getSystemPromptForType(agentType)

	// Check if run in background
	if runInBackground {
		return t.executeInBackground(ctx, agentID, agentType, agentPrompt, description, maxTurns, startTime)
	}

	// Execute synchronously
	var result string
	var err error

	if t.executor != nil {
		// Use the real executor
		result, err = t.executor.ExecuteAgent(ctx, agentType, systemPrompt+"\n\nTask: "+agentPrompt, maxTurns)
	} else {
		// Fallback to simulated execution
		result = t.simulateExecution(ctx, agentType, agentPrompt, maxTurns)
	}

	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	output := &AgentOutput{
		Result:    result,
		AgentID:   agentID,
		Turns:     maxTurns, // Actual turns would be tracked
		Duration:  time.Since(startTime),
		AgentType: agentType,
	}

	// Add isolation info if specified
	if isolation == "worktree" {
		output.Result = fmt.Sprintf("[Isolated in worktree]\n%s", output.Result)
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// executeInBackground runs the agent in background mode
func (t *AgentTool) executeInBackground(ctx context.Context, agentID, agentType, agentPrompt, description string, maxTurns int, startTime time.Time) (*ToolResult, error) {
	// Create background task
	task := &BackgroundTask{
		ID:        agentID,
		Status:    "running",
		StartTime: startTime,
	}

	if t.taskOutput != nil {
		t.taskOutput.RegisterTask(agentID)
	}

	// Store in local map
	t.backgroundMu.Lock()
	t.backgroundMap[agentID] = task
	t.backgroundMu.Unlock()

	// Start execution in goroutine
	go func() {
		var result string
		if t.executor != nil {
			result, _ = t.executor.ExecuteAgent(context.Background(), agentType, agentPrompt, maxTurns)
		} else {
			result = fmt.Sprintf("Background agent completed: %s", truncate(agentPrompt, 100))
		}

		if t.taskOutput != nil {
			t.taskOutput.CompleteTask(agentID, result)
		}

		t.backgroundMu.Lock()
		if task, exists := t.backgroundMap[agentID]; exists {
			task.Status = "completed"
			task.Output = result
			now := time.Now()
			task.EndTime = &now
		}
		t.backgroundMu.Unlock()
	}()

	return &ToolResult{
		Data: map[string]interface{}{
			"task_id":     agentID,
			"status":      "running",
			"message":     "🌀 Poyo 的分身已开始后台执行！",
			"description": description,
			"agent_type":  agentType,
		},
	}, nil
}

// getSystemPromptForType returns the appropriate system prompt for an agent type
func (t *AgentTool) getSystemPromptForType(agentType string) string {
	switch agentType {
	case "explore":
		return prompt.GetSystemPrompt("explore")
	case "plan":
		return prompt.GetSystemPrompt("plan_agent")
	default:
		return prompt.GetSystemPrompt("agent")
	}
}

// simulateExecution simulates agent execution when no executor is set
func (t *AgentTool) simulateExecution(ctx context.Context, agentType, agentPrompt string, maxTurns int) string {
	return fmt.Sprintf("🌀 Poyo 分身 (%s) 完成了任务！\n任务: %s\n\n[分身能力需要配置 AgentExecutor 才能真正执行]",
		agentType, truncate(agentPrompt, 200))
}

// Stream executes the Agent tool with streaming
func (t *AgentTool) Stream(ctx context.Context, input interface{}, toolCtx *ToolUseContext) (<-chan ProgressEvent, <-chan *ToolResult) {
	progressCh := make(chan ProgressEvent, 10)
	resultCh := make(chan *ToolResult, 1)

	go func() {
		defer close(progressCh)
		defer close(resultCh)

		// Send progress updates
		progressCh <- ProgressEvent{
			Type:    "start",
			Message: prompt.PoyoThinking(),
		}

		// Type assert input
		inputMap, ok := input.(map[string]interface{})
		if !ok {
			progressCh <- ProgressEvent{
				Type:  "error",
				Error: "invalid input type",
			}
			return
		}

		// Execute
		result, err := t.Call(ctx, inputMap, toolCtx, nil, nil)
		if err != nil {
			progressCh <- ProgressEvent{
				Type:  "error",
				Error: err.Error(),
			}
			return
		}

		progressCh <- ProgressEvent{
			Type:    "complete",
			Message: prompt.PoyoSuccess(),
		}

		resultCh <- result
	}()

	return progressCh, resultCh
}

// InputSchema returns the input schema for the Agent tool
func (t *AgentTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"prompt": {
				"type":        "string",
				"description": "The task for the ninja sub-agent to handle (忍者分身的任务)",
			},
			"description": {
				"type":        "string",
				"description": "Brief description of what this sub-agent does (3-5 words recommended)",
			},
			"subagent_type": {
				"type":        "string",
				"description": "The type of sub-agent (分身类型)",
				"enum":        []string{"general-purpose", "explore", "plan"},
			},
			"model": {
				"type":        "string",
				"description": "The model to use for the sub-agent (default: same as main loop)",
			},
			"max_turns": {
				"type":        "integer",
				"description": "Maximum number of turns for the sub-agent",
				"minimum":     1,
				"maximum":     50,
			},
			"run_in_background": {
				"type":        "boolean",
				"description": "Set to true to run the agent in the background (后台执行)",
			},
			"isolation": {
				"type":        "string",
				"description": "Isolation mode for the agent",
				"enum":        []string{"none", "worktree"},
			},
		},
		Required: []string{"prompt"},
	}
}

// generateAgentID generates a unique agent ID
func generateAgentID() string {
	return fmt.Sprintf("agent_%d", time.Now().UnixNano())
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetBackgroundTask returns a background task by ID
func (t *AgentTool) GetBackgroundTask(id string) *BackgroundTask {
	t.backgroundMu.Lock()
	defer t.backgroundMu.Unlock()
	return t.backgroundMap[id]
}

// ListBackgroundTasks returns all background tasks
func (t *AgentTool) ListBackgroundTasks() []*BackgroundTask {
	t.backgroundMu.Lock()
	defer t.backgroundMu.Unlock()

	result := make([]*BackgroundTask, 0, len(t.backgroundMap))
	for _, task := range t.backgroundMap {
		result = append(result, task)
	}
	return result
}
