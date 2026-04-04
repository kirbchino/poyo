// Package tools contains individual tool implementations.
package tools

import (
	"github.com/kirbchino/poyo/internal/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileWriteTool implements the Write tool for writing file contents.
type FileWriteTool struct {
	BaseTool
}

// FileWriteInput represents the input for the Write tool.
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// FileWriteOutput represents the output of the Write tool.
type FileWriteOutput struct {
	FilePath     string `json:"file_path"`
	BytesWritten int    `json:"bytesWritten"`
	Created      bool   `json:"created"`
}

// NewFileWriteTool creates a new FileWrite tool.
func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: BaseTool{
			name:        "Write",
			aliases:     []string{"write", "create", "save"},
			description: "Writes content to a file, creating it if it doesn't exist",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The absolute path to the file to write",
					},
					"content": {
						"type":        "string",
						"description": "The content to write to the file",
					},
				},
				Required: []string{"file_path", "content"},
			},
			isEnabled:     true,
			isReadOnly:    false,
			isDestructive: true,
			maxResultSize: 10000,
		},
	}
}

// Call writes content to a file.
func (t *FileWriteTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	filePath, ok := input["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path is required and must be a string")
	}

	content, ok := input["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required and must be a string")
	}

	// Validate and normalize path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check permission
	if canUseTool != nil {
		permResult, err := canUseTool(t.name, input)
		if err != nil {
			return nil, fmt.Errorf("permission check failed: %w", err)
		}
		if permResult.Behavior == "deny" {
			return &ToolResult{
				Data: &FileWriteOutput{
					FilePath: absPath,
				},
			}, nil
		}
	}

	// Write the file
	output, err := t.writeFile(absPath, content)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// writeFile writes content to a file.
func (t *FileWriteTool) writeFile(filePath, content string) (*FileWriteOutput, error) {
	// Check if file already exists
	_, err := os.Stat(filePath)
	created := os.IsNotExist(err)

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &FileWriteOutput{
		FilePath:     filePath,
		BytesWritten: len(content),
		Created:      created,
	}, nil
}

// CheckPermissions checks if the file can be written.
func (t *FileWriteTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	filePath, ok := input["file_path"].(string)
	if !ok {
		return &types.PermissionResult{
			Behavior: "deny",
			Message:  "file_path is required",
		}, nil
	}

	// Check for protected paths
	protectedPaths := []string{
		"/etc/",
		"/usr/",
		"/bin/",
		"/sbin/",
		"/lib/",
		"/System/",
		"/Windows/",
	}

	for _, protected := range protectedPaths {
		if strings.HasPrefix(filePath, protected) {
			return &types.PermissionResult{
				Behavior: "deny",
				Message:  fmt.Sprintf("Cannot write to protected path: %s", filePath),
			}, nil
		}
	}

	// Check if file already exists
	_, err := os.Stat(filePath)
	if err == nil {
		// File exists, ask for confirmation to overwrite
		return &types.PermissionResult{
			Behavior: "ask",
			Message:  fmt.Sprintf("File already exists: %s. Overwrite?", filePath),
		}, nil
	}

	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *FileWriteTool) UserFacingName(input map[string]interface{}) string {
	if path, ok := input["file_path"].(string); ok {
		return filepath.Base(path)
	}
	return "Write file"
}

// IsDestructive returns true since writing to a file is destructive.
func (t *FileWriteTool) IsDestructive(input map[string]interface{}) bool {
	// Check if file exists - overwriting is destructive
	if path, ok := input["file_path"].(string); ok {
		_, err := os.Stat(path)
		return err == nil
	}
	return false
}
