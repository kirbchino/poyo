// Package tools contains individual tool implementations.
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kirbchino/poyo/internal/prompt"
	"github.com/kirbchino/poyo/internal/sandbox"
	"github.com/kirbchino/poyo/internal/types"
)

// BashTool implements the Bash tool for executing shell commands.
type BashTool struct {
	BaseTool
	sandbox *sandbox.Manager
}

// BashInput represents the input for the Bash tool.
type BashInput struct {
	Command               string `json:"command"`
	Description           string `json:"description,omitempty"`
	Timeout               int    `json:"timeout,omitempty"`
	RunInBackground       bool   `json:"run_in_background,omitempty"`
	DangerouslyDisableSandbox bool `json:"dangerouslyDisableSandbox,omitempty"`
}

// BashOutput represents the output of the Bash tool.
type BashOutput struct {
	Stdout                  string `json:"stdout"`
	Stderr                  string `json:"stderr"`
	Interrupted             bool   `json:"interrupted"`
	IsImage                 bool   `json:"isImage,omitempty"`
	BackgroundTaskID        string `json:"backgroundTaskId,omitempty"`
	ReturnCodeInterpretation string `json:"returnCodeInterpretation,omitempty"`
	NoOutputExpected        bool   `json:"noOutputExpected,omitempty"`
	PersistedOutputPath     string `json:"persistedOutputPath,omitempty"`
	PersistedOutputSize     int64  `json:"persistedOutputSize,omitempty"`
	WasSandboxed            bool   `json:"wasSandboxed,omitempty"`
}

// NewBashTool creates a new Bash tool.
func NewBashTool() *BashTool {
	// Initialize sandbox with default config
	sandboxConfig := sandbox.DefaultConfig()
	sandboxMgr := sandbox.NewManager(sandboxConfig)

	return &BashTool{
		BaseTool: BaseTool{
			name:        "Bash",
			aliases:     []string{"bash", "shell"},
			description: prompt.GetToolDescription("Bash"),
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"command": {
						"type":        "string",
						"description": "The command to execute (Poyo 会用火焰能力执行)",
					},
					"description": {
						"type":        "string",
						"description": "Brief description of what this command does",
					},
					"timeout": {
						"type":        "integer",
						"description": "Optional timeout in milliseconds",
					},
					"run_in_background": {
						"type":        "boolean",
						"description": "Set to true to run this command in the background",
					},
					"dangerouslyDisableSandbox": {
						"type":        "boolean",
						"description": "Set to true to dangerously override sandbox mode",
					},
				},
				Required: []string{"command"},
			},
			isEnabled:         true,
			isReadOnly:        false,
			isDestructive:     true,
			isConcurrencySafe: true,
			maxResultSize:     100000, // 100KB default
		},
		sandbox: sandboxMgr,
	}
}

// Name returns the tool's name.
func (t *BashTool) Name() string {
	return t.name
}

// Call executes a bash command.
func (t *BashTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	cmd, ok := input["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command is required and must be a string")
	}

	timeout := 30000 // Default 30 seconds
	if timeoutVal, ok := input["timeout"].(float64); ok {
		timeout = int(timeoutVal)
	}

	description, _ := input["description"].(string)

	// Check permission
	if canUseTool != nil {
		permResult, err := canUseTool(t.name, input)
		if err != nil {
			return nil, fmt.Errorf("permission check failed: %w", err)
		}
		if permResult.Behavior == "deny" {
			return &ToolResult{
				Data: &BashOutput{
					Stderr:      permResult.Message,
					Interrupted: true,
				},
			}, nil
		}
	}

	// Execute the command
	output, err := t.executeCommand(ctx, cmd, timeout, description, onProgress)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// executeCommand runs a shell command.
func (t *BashTool) executeCommand(ctx context.Context, command string, timeoutMs int, description string, onProgress ToolCallProgress) (*BashOutput, error) {
	// Determine the shell to use
	shell := "/bin/sh"
	if _, err := os.Stat("/bin/bash"); err == nil {
		shell = "/bin/bash"
	}

	// Check if sandbox should be used
	useSandbox := t.sandbox != nil && t.sandbox.ShouldUseSandbox(command, false)

	// Create context with timeout
	timeout := time.Duration(timeoutMs) * time.Millisecond
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare the command
	var cmdStr string
	if useSandbox {
		wrapped, err := t.sandbox.WrapCommand(command, shell)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap command with sandbox: %w", err)
		}
		cmdStr = wrapped
	} else {
		cmdStr = command
	}

	cmd := exec.CommandContext(execCtx, shell, "-c", cmdStr)

	// Set working directory if available
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start progress reporting if command is expected to take long
	startTime := time.Now()
	progressThreshold := 2 * time.Second

	// Start the command
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for the command in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Monitor for progress and interruption
	select {
	case <-execCtx.Done():
		// Context was cancelled or timed out
		if execCtx.Err() == context.DeadlineExceeded {
			return &BashOutput{
				Stdout:      stdout.String(),
				Stderr:      stderr.String(),
				Interrupted: true,
			}, nil
		}
		return nil, execCtx.Err()

	case err := <-done:
		// Command completed
		elapsed := time.Since(startTime)

		// Cleanup sandbox resources
		if t.sandbox != nil {
			t.sandbox.CleanupAfterCommand()
		}

		// Report progress if command took longer than threshold
		if elapsed > progressThreshold && onProgress != nil {
			onProgress(ToolProgress{
				ToolUseID: "",
				Data: map[string]interface{}{
					"type":        "complete",
					"duration_ms": elapsed.Milliseconds(),
				},
			})
		}

		output := &BashOutput{
			Stdout:      stdout.String(),
			Stderr:      stderr.String(),
			Interrupted: false,
			WasSandboxed: useSandbox,
		}

		// Add sandbox indicator to output
		if useSandbox {
			output.Stderr = "[Sandboxed] " + output.Stderr
		}

		// Check if command is expected to have no output
		if output.Stdout == "" && output.Stderr == "" && t.isSilentCommand(command) {
			output.NoOutputExpected = true
		}

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				output.Stderr = fmt.Sprintf("%s\nExit code: %d", output.Stderr, exitErr.ExitCode())
			}
		}

		return output, nil
	}
}

// isSearchOrReadCommand checks if a command is a search or read operation.
func (t *BashTool) isSearchOrReadCommand(command string) (isSearch, isRead, isList bool) {
	searchCommands := map[string]bool{
		"find": true, "grep": true, "rg": true, "ag": true, "ack": true,
		"locate": true, "which": true, "whereis": true,
	}

	readCommands := map[string]bool{
		"cat": true, "head": true, "tail": true, "less": true, "more": true,
		"wc": true, "stat": true, "file": true, "strings": true,
		"jq": true, "awk": true, "cut": true, "sort": true, "uniq": true, "tr": true,
	}

	listCommands := map[string]bool{
		"ls": true, "tree": true, "du": true,
	}

	// Extract base command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, false, false
	}

	baseCmd := parts[0]
	if strings.HasPrefix(baseCmd, "/") {
		// Get the last component of the path
		parts := strings.Split(baseCmd, "/")
		baseCmd = parts[len(parts)-1]
	}

	return searchCommands[baseCmd], readCommands[baseCmd], listCommands[baseCmd]
}

// isSilentCommand checks if a command typically produces no stdout on success.
func (t *BashTool) isSilentCommand(command string) bool {
	silentCommands := map[string]bool{
		"mv": true, "cp": true, "rm": true, "mkdir": true, "rmdir": true,
		"chmod": true, "chown": true, "chgrp": true, "touch": true, "ln": true,
		"cd": true, "export": true, "unset": true, "wait": true,
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	baseCmd := parts[0]
	return silentCommands[baseCmd]
}

// CheckPermissions checks if the bash command can be executed.
func (t *BashTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	command, ok := input["command"].(string)
	if !ok {
		return &types.PermissionResult{
			Behavior: "deny",
			Message:  "command is required",
		}, nil
	}

	// Check for dangerous commands
	dangerousPatterns := []string{
		"rm -rf /",
		"mkfs",
		"dd if=/dev/zero",
		":(){ :|:& };:",
		"chmod -R 777 /",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return &types.PermissionResult{
				Behavior: "ask",
				Message:  prompt.MsgBashDangerous(command),
			}, nil
		}
	}

	// Check if it's a read-only command
	isSearch, isRead, _ := t.isSearchOrReadCommand(command)
	if isSearch || isRead {
		return &types.PermissionResult{
			Behavior: "allow",
		}, nil
	}

	// For write operations, ask for permission in non-auto mode
	if toolCtx.PermissionContext != nil {
		switch toolCtx.PermissionContext.Mode {
		case "bypassPermissions":
			return &types.PermissionResult{Behavior: "allow"}, nil
		case "default":
			return &types.PermissionResult{
				Behavior: "ask",
				Message:  fmt.Sprintf("Run command: %s", command),
			}, nil
		}
	}

	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *BashTool) UserFacingName(input map[string]interface{}) string {
	if desc, ok := input["description"].(string); ok && desc != "" {
		return desc
	}
	if cmd, ok := input["command"].(string); ok {
		// Truncate long commands
		if len(cmd) > 50 {
			return cmd[:47] + "..."
		}
		return cmd
	}
	return "Bash command"
}

// SetSandbox sets the sandbox manager for the tool
func (t *BashTool) SetSandbox(s *sandbox.Manager) {
	t.sandbox = s
}

// GetSandbox returns the current sandbox manager
func (t *BashTool) GetSandbox() *sandbox.Manager {
	return t.sandbox
}
