// Package tools contains individual tool implementations.
package tools

import (
	"github.com/kirbchino/poyo/internal/types"
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool implements the Grep tool for searching file contents.
type GrepTool struct {
	BaseTool
}

// GrepInput represents the input for the Grep tool.
type GrepInput struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path,omitempty"`
	OutputMode  string `json:"output_mode,omitempty"` // files_with_matches, content, content_with_line_numbers
	IgnoreCase  bool   `json:"ignore_case,omitempty"`
	Include     string `json:"include,omitempty"`     // file pattern to include
	Exclude     string `json:"exclude,omitempty"`     // file pattern to exclude
	HeadLimit   int    `json:"head_limit,omitempty"`  // limit number of results
}

// GrepOutput represents the output of the Grep tool.
type GrepOutput struct {
	Matches    []GrepMatch `json:"matches,omitempty"`
	Files      []string    `json:"files,omitempty"`
	Pattern    string      `json:"pattern"`
	Path       string      `json:"path"`
	Count      int         `json:"count"`
	Truncated  bool        `json:"truncated,omitempty"`
	OutputMode string      `json:"outputMode"`
}

// GrepMatch represents a single match.
type GrepMatch struct {
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Content  string `json:"content"`
}

const (
	MaxGrepMatches = 100
)

// NewGrepTool creates a new Grep tool.
func NewGrepTool() *GrepTool {
	return &GrepTool{
		BaseTool: BaseTool{
			name:        "Grep",
			aliases:     []string{"grep", "search", "rg"},
			description: "Searches for patterns in file contents",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"pattern": {
						"type":        "string",
						"description": "The pattern to search for (regex supported)",
					},
					"path": {
						"type":        "string",
						"description": "The directory to search in (default: current directory)",
					},
					"output_mode": {
						"type":        "string",
						"description": "Output mode: files_with_matches, content, content_with_line_numbers",
					},
					"ignore_case": {
						"type":        "boolean",
						"description": "Case insensitive search",
					},
					"include": {
						"type":        "string",
						"description": "File pattern to include (e.g., *.ts)",
					},
					"exclude": {
						"type":        "string",
						"description": "File pattern to exclude",
					},
					"head_limit": {
						"type":        "integer",
						"description": "Maximum number of results",
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

// Call searches for the pattern in files.
func (t *GrepTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	pattern, ok := input["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern is required and must be a string")
	}

	searchPath := "."
	if pathVal, ok := input["path"].(string); ok && pathVal != "" {
		searchPath = pathVal
	}

	outputMode := "files_with_matches"
	if modeVal, ok := input["output_mode"].(string); ok {
		outputMode = modeVal
	}

	ignoreCase := false
	if ignoreVal, ok := input["ignore_case"].(bool); ok {
		ignoreCase = ignoreVal
	}

	headLimit := MaxGrepMatches
	if limitVal, ok := input["head_limit"].(float64); ok {
		headLimit = int(limitVal)
	}

	includePattern, _ := input["include"].(string)
	excludePattern, _ := input["exclude"].(string)

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
				Data: &GrepOutput{
					Pattern: pattern,
					Path:    absPath,
				},
			}, nil
		}
	}

	// Search files
	output, err := t.searchFiles(absPath, pattern, outputMode, ignoreCase, includePattern, excludePattern, headLimit)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// searchFiles searches for the pattern in all files.
func (t *GrepTool) searchFiles(searchPath, pattern, outputMode string, ignoreCase bool, includePattern, excludePattern string, headLimit int) (*GrepOutput, error) {
	// Compile the regex
	flags := ""
	if ignoreCase {
		flags = "(?i)"
	}
	regex, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	output := &GrepOutput{
		Pattern:    pattern,
		Path:       searchPath,
		OutputMode: outputMode,
		Files:      []string{},
		Matches:    []GrepMatch{},
	}

	fileSet := make(map[string]bool)

	// Walk the directory
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip common non-code directories
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check include/exclude patterns
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		if excludePattern != "" {
			matched, err := filepath.Match(excludePattern, info.Name())
			if err == nil && matched {
				return nil
			}
		}

		// Skip binary files
		if t.isBinaryFile(path) {
			return nil
		}

		// Search the file
		matches, err := t.searchFile(path, regex, headLimit-len(output.Matches))
		if err != nil {
			return nil
		}

		for _, m := range matches {
			fileSet[m.FilePath] = true

			if outputMode == "content" || outputMode == "content_with_line_numbers" {
				output.Matches = append(output.Matches, m)
			}

			if len(output.Matches) >= headLimit {
				output.Truncated = true
				return fmt.Errorf("limit reached") // Stop walking
			}
		}

		return nil
	})

	// Build file list
	for f := range fileSet {
		output.Files = append(output.Files, f)
	}

	output.Count = len(output.Matches)
	if outputMode == "files_with_matches" {
		output.Count = len(output.Files)
	}

	return output, nil
}

// searchFile searches for matches in a single file.
func (t *GrepTool) searchFile(filePath string, regex *regexp.Regexp, limit int) ([]GrepMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()

		if regex.MatchString(line) {
			matches = append(matches, GrepMatch{
				FilePath: filePath,
				Line:     lineNum,
				Content:  line,
			})

			if len(matches) >= limit {
				break
			}
		}

		lineNum++
	}

	return matches, scanner.Err()
}

// isBinaryFile checks if a file is binary.
func (t *GrepTool) isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}

	return false
}

// CheckPermissions checks if the search can be performed.
func (t *GrepTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	// Grep is generally safe - it only reads file contents
	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *GrepTool) UserFacingName(input map[string]interface{}) string {
	if pattern, ok := input["pattern"].(string); ok {
		return fmt.Sprintf("Grep: %s", pattern)
	}
	return "Grep search"
}

// FormatOutput formats the grep output for display.
func (t *GrepTool) FormatOutput(output *GrepOutput) string {
	var sb strings.Builder

	switch output.OutputMode {
	case "files_with_matches":
		for _, f := range output.Files {
			relPath, _ := filepath.Rel(output.Path, f)
			sb.WriteString(relPath + "\n")
		}

	case "content":
		for _, m := range output.Matches {
			relPath, _ := filepath.Rel(output.Path, m.FilePath)
			sb.WriteString(fmt.Sprintf("%s:%s\n", relPath, m.Content))
		}

	case "content_with_line_numbers":
		for _, m := range output.Matches {
			relPath, _ := filepath.Rel(output.Path, m.FilePath)
			sb.WriteString(fmt.Sprintf("%s:%d:%s\n", relPath, m.Line, m.Content))
		}
	}

	return sb.String()
}
