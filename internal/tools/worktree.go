// Package tools implements Git Worktree tools
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnterWorktreeTool implements the EnterWorktree tool
type EnterWorktreeTool struct {
	BaseTool
	originalDir string
	worktreeDir string
}

// NewEnterWorktreeTool creates a new EnterWorktree tool
func NewEnterWorktreeTool() *EnterWorktreeTool {
	return &EnterWorktreeTool{
		BaseTool: BaseTool{
			name:        "EnterWorktree",
			description: "🌿 Create and enter a Git worktree for isolated work. This creates an isolated copy of the repository on a new branch.",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"name": {
						"type":        "string",
						"description": "Optional name for the worktree. If not provided, a random name is generated.",
					},
				},
			},
			isEnabled:         true,
			isConcurrencySafe: false,
		},
	}
}

// Call executes the EnterWorktree tool
func (t *EnterWorktreeTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	name, _ := input["name"].(string)
	if name == "" {
		name = fmt.Sprintf("worktree_%d", os.Getpid())
	}

	// Check if we're already in a worktree
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if already in a git repo
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return nil, fmt.Errorf("not in a git repository")
	}

	// Check for .git file (indicates worktree)
	if info, _ := os.Stat(gitDir); info != nil && !info.IsDir() {
		return nil, fmt.Errorf("already in a worktree - exit current worktree first")
	}

	// Create worktree directory
	worktreePath := filepath.Join(cwd, ".poyo", "worktrees", name)
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Create a new branch and worktree
	branchName := "poyo/" + name

	// Run git worktree add
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(worktreePath)
		return nil, fmt.Errorf("failed to create worktree: %w\n%s", err, string(output))
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"worktree_path": worktreePath,
			"branch":        branchName,
			"message":       fmt.Sprintf("🌿 Created worktree at %s on branch %s", worktreePath, branchName),
		},
	}, nil
}

// ExitWorktreeTool implements the ExitWorktree tool
type ExitWorktreeTool struct {
	BaseTool
}

// NewExitWorktreeTool creates a new ExitWorktree tool
func NewExitWorktreeTool() *ExitWorktreeTool {
	return &ExitWorktreeTool{
		BaseTool: BaseTool{
			name:        "ExitWorktree",
			description: "🚪 Exit a Git worktree and optionally remove it.",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"action": {
						"type":        "string",
						"description": "\"keep\" or \"remove\" the worktree",
						"enum":        []string{"keep", "remove"},
					},
					"discard_changes": {
						"type":        "boolean",
						"description": "If true, discard uncommitted changes when removing",
					},
				},
				Required: []string{"action"},
			},
			isEnabled:         true,
			isConcurrencySafe: false,
		},
	}
}

// Call executes the ExitWorktree tool
func (t *ExitWorktreeTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	action, _ := input["action"].(string)
	discardChanges, _ := input["discard_changes"].(bool)

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a worktree
	gitFile := filepath.Join(cwd, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil || !strings.HasPrefix(string(content), "gitdir:") {
		return nil, fmt.Errorf("not in a worktree")
	}

	// Find the main worktree
	cmd := exec.CommandContext(ctx, "git", "worktree", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var mainWorktree string
	var currentBranch string

	for _, line := range lines {
		if strings.Contains(line, cwd) {
			// Current worktree
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentBranch = strings.Trim(parts[2], "[]")
			}
		} else if mainWorktree == "" && strings.Contains(line, ".git") {
			// Main worktree (first one that has .git directory, not file)
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				mainWorktree = parts[0]
			}
		}
	}

	if mainWorktree == "" {
		return nil, fmt.Errorf("could not find main worktree")
	}

	if action == "remove" {
		// Check for uncommitted changes
		if !discardChanges {
			cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
			cmd.Dir = cwd
			output, err = cmd.Output()
			if err == nil && len(strings.TrimSpace(string(output))) > 0 {
				return nil, fmt.Errorf("worktree has uncommitted changes. Set discard_changes=true to force removal")
			}
		}

		// Remove worktree
		cmd = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", cwd)
		if output, err = cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to remove worktree: %w\n%s", err, string(output))
		}

		return &ToolResult{
			Data: map[string]interface{}{
				"action":       "removed",
				"branch":       currentBranch,
				"main_worktree": mainWorktree,
				"message":      fmt.Sprintf("🚪 Removed worktree and switched to %s", mainWorktree),
			},
		}, nil
	}

	// Keep worktree - just report the main worktree
	return &ToolResult{
		Data: map[string]interface{}{
			"action":        "kept",
			"branch":        currentBranch,
			"main_worktree": mainWorktree,
			"message":       fmt.Sprintf("🚪 Main worktree is at %s", mainWorktree),
		},
	}, nil
}

// InputSchema returns the input schema
func (t *EnterWorktreeTool) InputSchema() ToolInputJSONSchema {
	return t.inputSchema
}

// InputSchema returns the input schema
func (t *ExitWorktreeTool) InputSchema() ToolInputJSONSchema {
	return t.inputSchema
}
