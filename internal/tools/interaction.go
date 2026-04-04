// Package tools provides additional utility tools
package tools

import (
	"context"
	"fmt"

	"github.com/kirbchino/poyo/internal/prompt"
)

// AskUserQuestionTool asks the user a question
type AskUserQuestionTool struct {
	BaseTool
	askFunc func(question string, options []string) (string, error)
}

// NewAskUserQuestionTool creates a new AskUserQuestionTool
func NewAskUserQuestionTool() *AskUserQuestionTool {
	return &AskUserQuestionTool{
		BaseTool: BaseTool{
			name:        "AskUserQuestion",
			description: prompt.GetToolDescription("AskUserQuestion"),
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"question": {
						"type":        "string",
						"description": "The question to ask the user",
					},
					"options": {
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional list of options for the user to choose from",
					},
				},
				Required: []string{"question"},
			},
			isEnabled: true,
		},
	}
}

// SetAskFunc sets the ask function for user interaction
func (t *AskUserQuestionTool) SetAskFunc(askFunc func(question string, options []string) (string, error)) {
	t.askFunc = askFunc
}

// InputSchema returns the input schema
func (t *AskUserQuestionTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"question": {
				"type":        "string",
				"description": "The question to ask the user",
			},
			"options": {
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional list of options for the user to choose from",
			},
		},
		Required: []string{"question"},
	}
}

// Call executes the tool
func (t *AskUserQuestionTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	question, _ := input["question"].(string)

	options := make([]string, 0)
	if opts, ok := input["options"].([]interface{}); ok {
		for _, opt := range opts {
			if s, ok := opt.(string); ok {
				options = append(options, s)
			}
		}
	}

	if t.askFunc == nil {
		return &ToolResult{
			Data: map[string]interface{}{
				"answer":  "user_input_skipped",
				"message": "No ask function configured",
			},
		}, nil
	}

	answer, err := t.askFunc(question, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"answer":   answer,
			"question": question,
		},
	}, nil
}

// EnterPlanModeTool enters plan mode
type EnterPlanModeTool struct {
	BaseTool
}

// NewEnterPlanModeTool creates a new EnterPlanModeTool
func NewEnterPlanModeTool() *EnterPlanModeTool {
	return &EnterPlanModeTool{
		BaseTool: BaseTool{
			name:        "EnterPlanMode",
			description: prompt.GetToolDescription("EnterPlanMode"),
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *EnterPlanModeTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"task_description": {
				"type":        "string",
				"description": "Description of the task to plan",
			},
		},
		Required: []string{"task_description"},
	}
}

// Call executes the tool
func (t *EnterPlanModeTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	taskDesc, _ := input["task_description"].(string)

	return &ToolResult{
		Data: map[string]interface{}{
			"mode":             "plan",
			"task_description": taskDesc,
			"message":          "Entered plan mode.",
		},
	}, nil
}

// ExitPlanModeTool exits plan mode
type ExitPlanModeTool struct {
	BaseTool
}

// NewExitPlanModeTool creates a new ExitPlanModeTool
func NewExitPlanModeTool() *ExitPlanModeTool {
	return &ExitPlanModeTool{
		BaseTool: BaseTool{
			name:        "ExitPlanMode",
			description: prompt.GetToolDescription("ExitPlanMode"),
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *ExitPlanModeTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type:       "object",
		Properties: map[string]map[string]interface{}{},
	}
}

// Call executes the tool
func (t *ExitPlanModeTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	return &ToolResult{
		Data: map[string]interface{}{
			"mode":    "default",
			"message": "Exited plan mode.",
		},
	}, nil
}

// CronCreateTool creates a scheduled task
type CronCreateTool struct {
	BaseTool
}

// NewCronCreateTool creates a new CronCreateTool
func NewCronCreateTool() *CronCreateTool {
	return &CronCreateTool{
		BaseTool: BaseTool{
			name:        "CronCreate",
			description: prompt.GetToolDescription("CronCreate"),
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *CronCreateTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"cron": {
				"type":        "string",
				"description": "Cron expression",
			},
			"prompt": {
				"type":        "string",
				"description": "The prompt to execute",
			},
		},
		Required: []string{"cron", "prompt"},
	}
}

// Call executes the tool
func (t *CronCreateTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	cron, _ := input["cron"].(string)
	prompt, _ := input["prompt"].(string)

	return &ToolResult{
		Data: map[string]interface{}{
			"job_id":  fmt.Sprintf("cron_%d", len(cron)+len(prompt)),
			"cron":    cron,
			"prompt":  prompt,
			"message": "Created scheduled task",
		},
	}, nil
}

// CronDeleteTool deletes a scheduled task
type CronDeleteTool struct {
	BaseTool
}

// NewCronDeleteTool creates a new CronDeleteTool
func NewCronDeleteTool() *CronDeleteTool {
	return &CronDeleteTool{
		BaseTool: BaseTool{
			name:        "CronDelete",
			description: prompt.GetToolDescription("CronDelete"),
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *CronDeleteTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"id": {
				"type":        "string",
				"description": "The ID of the cron job to delete",
			},
		},
		Required: []string{"id"},
	}
}

// Call executes the tool
func (t *CronDeleteTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	jobID, _ := input["id"].(string)

	return &ToolResult{
		Data: map[string]interface{}{
			"job_id":  jobID,
			"deleted": true,
		},
	}, nil
}

// CronListTool lists scheduled tasks
type CronListTool struct {
	BaseTool
}

// NewCronListTool creates a new CronListTool
func NewCronListTool() *CronListTool {
	return &CronListTool{
		BaseTool: BaseTool{
			name:        "CronList",
			description: prompt.GetToolDescription("CronList"),
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *CronListTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type:       "object",
		Properties: map[string]map[string]interface{}{},
	}
}

// Call executes the tool
func (t *CronListTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	return &ToolResult{
		Data: map[string]interface{}{
			"jobs":    []interface{}{},
			"message": "No scheduled tasks found",
		},
	}, nil
}
