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

// GlobTool implements the Glob tool for finding files by pattern.
type GlobTool struct {
	BaseTool
}

// GlobInput represents the input for the Glob tool.
type GlobInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// GlobOutput represents the output of the Glob tool.
type GlobOutput struct {
	Files    []string `json:"files"`
	Pattern  string   `json:"pattern"`
	Path     string   `json:"path"`
	Count    int      `json:"count"`
	Truncated bool    `json:"truncated,omitempty"`
}

const (
	MaxGlobResults = 1000
)

// NewGlobTool creates a new Glob tool.
func NewGlobTool() *GlobTool {
	return &GlobTool{
		BaseTool: BaseTool{
			name:        "Glob",
			aliases:     []string{"glob", "find", "ls"},
			description: "Finds files matching a glob pattern",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"pattern": {
						"type":        "string",
						"description": "The glob pattern to match files (e.g., **/*.ts)",
					},
					"path": {
						"type":        "string",
						"description": "The directory to search in (default: current directory)",
					},
				},
				Required: []string{"pattern"},
			},
			isEnabled:     true,
			isReadOnly:    true,
			isDestructive: false,
			maxResultSize: 50000,
		},
	}
}

// Call finds files matching the glob pattern.
func (t *GlobTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	pattern, ok := input["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern is required and must be a string")
	}

	searchPath := "."
	if pathVal, ok := input["path"].(string); ok && pathVal != "" {
		searchPath = pathVal
	}

	// Validate and normalize path
	absPath, err := filepath.Abs(searchPath)
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
				Data: &GlobOutput{
					Pattern: pattern,
					Path:    absPath,
				},
			}, nil
		}
	}

	// Find files
	output, err := t.findFiles(absPath, pattern)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// findFiles searches for files matching the pattern.
func (t *GlobTool) findFiles(searchPath, pattern string) (*GlobOutput, error) {
	var files []string
	truncated := false

	// Build the full pattern
	fullPattern := filepath.Join(searchPath, pattern)

	// Use filepath.Glob for simple patterns
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// For recursive patterns (**), we need to walk the directory
	if strings.Contains(pattern, "**") {
		files, err = t.walkGlob(searchPath, pattern)
		if err != nil {
			return nil, err
		}
	} else {
		files = matches
	}

	// Truncate if too many results
	if len(files) > MaxGlobResults {
		files = files[:MaxGlobResults]
		truncated = true
	}

	// Make paths relative to search path
	relFiles := make([]string, len(files))
	for i, f := range files {
		relPath, err := filepath.Rel(searchPath, f)
		if err != nil {
			relPath = f
		}
		relFiles[i] = relPath
	}

	return &GlobOutput{
		Files:     relFiles,
		Pattern:   pattern,
		Path:      searchPath,
		Count:     len(relFiles),
		Truncated: truncated,
	}, nil
}

// walkGlob walks the directory tree for recursive patterns.
func (t *GlobTool) walkGlob(searchPath, pattern string) ([]string, error) {
	var files []string

	// Convert glob pattern to a regex-like check
	// Simple implementation: use filepath.Match for each file
	basePattern := filepath.Base(pattern)

	// Walk the directory
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(searchPath, path)
		if err != nil {
			return nil
		}

		// Check if it matches the pattern
		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			return nil
		}

		// Also try matching just the filename for patterns like "*.ts"
		if !matched {
			matched, _ = filepath.Match(basePattern, filepath.Base(path))
		}

		if matched {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// CheckPermissions checks if the directory can be searched.
func (t *GlobTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	// Glob is generally safe - it only reads file names
	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *GlobTool) UserFacingName(input map[string]interface{}) string {
	if pattern, ok := input["pattern"].(string); ok {
		return fmt.Sprintf("Glob: %s", pattern)
	}
	return "Glob search"
}
