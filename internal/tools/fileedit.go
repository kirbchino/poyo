// Package tools contains individual tool implementations.
package tools

import (
	"github.com/kirbchino/poyo/internal/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileEditTool implements the Edit tool for editing file contents.
type FileEditTool struct {
	BaseTool
}

// FileEditInput represents the input for the Edit tool.
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// FileEditOutput represents the output of the Edit tool.
type FileEditOutput struct {
	FilePath      string `json:"file_path"`
	Replacements  int    `json:"replacements"`
	OriginalLines int    `json:"originalLines"`
	NewLines      int    `json:"newLines"`
	Content       string `json:"content,omitempty"`
}

// NewFileEditTool creates a new FileEdit tool.
func NewFileEditTool() *FileEditTool {
	return &FileEditTool{
		BaseTool: BaseTool{
			name:        "Edit",
			aliases:     []string{"edit", "replace", "sed"},
			description: "Performs string replacements in a file",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The absolute path to the file to edit",
					},
					"old_string": {
						"type":        "string",
						"description": "The text to search for",
					},
					"new_string": {
						"type":        "string",
						"description": "The text to replace with",
					},
					"replace_all": {
						"type":        "boolean",
						"description": "Replace all occurrences (default: false)",
					},
				},
				Required: []string{"file_path", "old_string", "new_string"},
			},
			isEnabled:     true,
			isReadOnly:    false,
			isDestructive: true,
			maxResultSize: 10000,
		},
	}
}

// Call edits a file by replacing text.
func (t *FileEditTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	filePath, ok := input["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path is required and must be a string")
	}

	oldString, ok := input["old_string"].(string)
	if !ok {
		return nil, fmt.Errorf("old_string is required and must be a string")
	}

	newString, ok := input["new_string"].(string)
	if !ok {
		return nil, fmt.Errorf("new_string is required and must be a string")
	}

	replaceAll := false
	if replaceAllVal, ok := input["replace_all"].(bool); ok {
		replaceAll = replaceAllVal
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
				Data: &FileEditOutput{
					FilePath: absPath,
				},
			}, nil
		}
	}

	// Edit the file
	output, err := t.editFile(absPath, oldString, newString, replaceAll)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// editFile performs the string replacement in a file.
func (t *FileEditTool) editFile(filePath, oldString, newString string, replaceAll bool) (*FileEditOutput, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	originalLines := len(strings.Split(originalContent, "\n"))

	// Count occurrences
	occurrences := strings.Count(originalContent, oldString)
	if occurrences == 0 {
		return nil, fmt.Errorf("old_string not found in file: %s", filePath)
	}

	// Perform replacement
	var newContent string
	var replacements int

	if replaceAll {
		newContent = strings.ReplaceAll(originalContent, oldString, newString)
		replacements = occurrences
	} else {
		// Replace only the first occurrence
		idx := strings.Index(originalContent, oldString)
		if idx == -1 {
			return nil, fmt.Errorf("old_string not found in file: %s", filePath)
		}
		newContent = originalContent[:idx] + newString + originalContent[idx+len(oldString):]
		replacements = 1

		// Warn if there are more occurrences
		if occurrences > 1 {
			// Could add a note about multiple occurrences
		}
	}

	newLines := len(strings.Split(newContent, "\n"))

	// Write the file back
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &FileEditOutput{
		FilePath:      filePath,
		Replacements:  replacements,
		OriginalLines: originalLines,
		NewLines:      newLines,
	}, nil
}

// CheckPermissions checks if the file can be edited.
func (t *FileEditTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	filePath, ok := input["file_path"].(string)
	if !ok {
		return &types.PermissionResult{
			Behavior: "deny",
			Message:  "file_path is required",
		}, nil
	}

	// Check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.PermissionResult{
				Behavior: "deny",
				Message:  fmt.Sprintf("File not found: %s", filePath),
			}, nil
		}
	}

	// In default mode, ask for confirmation for edits
	if toolCtx.PermissionContext != nil && toolCtx.PermissionContext.Mode == "default" {
		oldString, _ := input["old_string"].(string)
		newString, _ := input["new_string"].(string)

		// Truncate long strings for display
		if len(oldString) > 100 {
			oldString = oldString[:97] + "..."
		}
		if len(newString) > 100 {
			newString = newString[:97] + "..."
		}

		return &types.PermissionResult{
			Behavior: "ask",
			Message:  fmt.Sprintf("Edit %s:\n\nReplace:\n  %s\n\nWith:\n  %s", filepath.Base(filePath), oldString, newString),
		}, nil
	}

	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *FileEditTool) UserFacingName(input map[string]interface{}) string {
	if path, ok := input["file_path"].(string); ok {
		return fmt.Sprintf("Edit %s", filepath.Base(path))
	}
	return "Edit file"
}

// GetDiff returns a diff-style representation of the change.
func (t *FileEditTool) GetDiff(filePath, oldString, newString string) string {
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- %s\n", filePath))
	diff.WriteString(fmt.Sprintf("+++ %s\n", filePath))

	// Show removed lines
	for _, line := range strings.Split(oldString, "\n") {
		diff.WriteString(fmt.Sprintf("-%s\n", line))
	}

	// Show added lines
	for _, line := range strings.Split(newString, "\n") {
		diff.WriteString(fmt.Sprintf("+%s\n", line))
	}

	return diff.String()
}

// EditInputWithRegex represents an edit with regex support.
type EditInputWithRegex struct {
	FilePath    string `json:"file_path"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
}

// EditWithRegex performs a regex-based replacement.
func (t *FileEditTool) EditWithRegex(filePath, pattern, replacement string) (*FileEditOutput, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	originalLines := len(strings.Split(originalContent, "\n"))

	// Compile the regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Perform replacement
	newContent := re.ReplaceAllString(originalContent, replacement)
	replacements := len(re.FindAllStringIndex(originalContent, -1))

	newLines := len(strings.Split(newContent, "\n"))

	// Write the file back
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &FileEditOutput{
		FilePath:      filePath,
		Replacements:  replacements,
		OriginalLines: originalLines,
		NewLines:      newLines,
	}, nil
}
