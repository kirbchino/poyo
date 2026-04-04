// Package tools implements the Skill tool for invoking skills
package tools

import (
	"context"
	"fmt"

	"github.com/kirbchino/poyo/internal/prompt"
)

// SkillTool implements the Skill tool for invoking skills
type SkillTool struct {
	BaseTool
	skillExecutor func(skill string, args string) (string, error)
}

// NewSkillTool creates a new Skill tool
func NewSkillTool() *SkillTool {
	return &SkillTool{
		BaseTool: BaseTool{
			name:        "Skill",
			aliases:     []string{"skill"},
			description: prompt.GetToolDescription("Skill"),
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"skill": {
						"type":        "string",
						"description": "The name of the skill to invoke (e.g., 'pdf', 'docx')",
					},
					"args": {
						"type":        "string",
						"description": "Optional arguments to pass to the skill",
					},
				},
				Required: []string{"skill"},
			},
			isEnabled:         true,
			isConcurrencySafe: true,
		},
	}
}

// SetSkillExecutor sets the skill executor function
func (t *SkillTool) SetSkillExecutor(executor func(skill string, args string) (string, error)) {
	t.skillExecutor = executor
}

// Call executes the Skill tool
func (t *SkillTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	skill, ok := input["skill"].(string)
	if !ok || skill == "" {
		return nil, fmt.Errorf("skill name is required")
	}

	args, _ := input["args"].(string)

	// If no executor is set, return a placeholder
	if t.skillExecutor == nil {
		return &ToolResult{
			Data: map[string]interface{}{
				"skill":    skill,
				"args":     args,
				"result":   fmt.Sprintf("Skill '%s' would be invoked here", skill),
				"message":  "Skill tool not fully configured - set executor with SetSkillExecutor()",
			},
		}, nil
	}

	result, err := t.skillExecutor(skill, args)
	if err != nil {
		return nil, fmt.Errorf("skill execution failed: %w", err)
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"skill":  skill,
			"args":   args,
			"result": result,
		},
	}, nil
}

// InputSchema returns the input schema for the Skill tool
func (t *SkillTool) InputSchema() ToolInputJSONSchema {
	return t.inputSchema
}
