// Package tools contains individual tool implementations.
package tools

import (
	"bytes"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirbchino/poyo/internal/types"
)

// FileReadTool implements the Read tool for reading file contents.
type FileReadTool struct {
	BaseTool
}

// FileReadInput represents the input for the Read tool.
type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// FileReadOutput represents the output of the Read tool.
type FileReadOutput struct {
	Content       string `json:"content"`
	FilePath      string `json:"file_path"`
	Size          int64  `json:"size"`
	LinesRead     int    `json:"linesRead"`
	TotalLines    int    `json:"totalLines"`
	IsTruncated   bool   `json:"isTruncated,omitempty"`
	Encoding      string `json:"encoding,omitempty"`
	ModifiedTime  string `json:"modifiedTime,omitempty"`
}

const (
	MaxFileSize   = 10 * 1024 * 1024 // 10MB
	MaxLinesRead  = 2000
	DefaultLimit  = 2000
)

// NewFileReadTool creates a new FileRead tool.
func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: BaseTool{
			name:        "Read",
			aliases:     []string{"read", "cat", "file"},
			description: "Reads the contents of a file",
			inputSchema: ToolInputJSONSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"file_path": {
						"type":        "string",
						"description": "The absolute path to the file to read",
					},
					"offset": {
						"type":        "integer",
						"description": "The line number to start reading from (1-indexed)",
					},
					"limit": {
						"type":        "integer",
						"description": "The number of lines to read",
					},
				},
				Required: []string{"file_path"},
			},
			isEnabled:     true,
			isReadOnly:    true,
			isDestructive: false,
			maxResultSize: 100000,
		},
	}
}

// Call reads a file and returns its contents.
func (t *FileReadTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	filePath, ok := input["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path is required and must be a string")
	}

	offset := 1
	if offsetVal, ok := input["offset"].(float64); ok {
		offset = int(offsetVal)
	}

	limit := DefaultLimit
	if limitVal, ok := input["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Validate and normalize path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path is blocked (e.g., /dev/zero, /dev/random)
	if t.isBlockedDevicePath(absPath) {
		return nil, fmt.Errorf("cannot read from blocked device path: %s", absPath)
	}

	// Check permission
	if canUseTool != nil {
		permResult, err := canUseTool(t.name, input)
		if err != nil {
			return nil, fmt.Errorf("permission check failed: %w", err)
		}
		if permResult.Behavior == "deny" {
			return &ToolResult{
				Data: &FileReadOutput{
					FilePath: absPath,
				},
			}, nil
		}
	}

	// Read the file
	output, err := t.readFile(absPath, offset, limit)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// readFile reads the content of a file.
func (t *FileReadTool) readFile(filePath string, offset, limit int) (*FileReadOutput, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file too large (max %d bytes): %s has %d bytes", MaxFileSize, filePath, info.Size())
	}

	// Detect if file is binary
	if t.isBinaryFile(file) {
		return &FileReadOutput{
			Content:    "[Binary file - cannot display]",
			FilePath:   filePath,
			Size:       info.Size(),
			Encoding:   "binary",
		}, nil
	}

	// Reset file position after binary check
	file.Seek(0, 0)

	// Read lines with line numbers
	scanner := bufio.NewScanner(file)
	var lines []string
	var totalLines int
	lineNum := 1

	for scanner.Scan() {
		totalLines++
		if lineNum >= offset && len(lines) < limit {
			lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum, scanner.Text()))
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 && totalLines > offset+len(lines)-1 {
		content += "\n"
	}

	return &FileReadOutput{
		Content:      content,
		FilePath:     filePath,
		Size:         info.Size(),
		LinesRead:    len(lines),
		TotalLines:   totalLines,
		IsTruncated:  totalLines > offset+len(lines)-1,
		ModifiedTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}

// isBinaryFile checks if a file contains binary content.
func (t *FileReadTool) isBinaryFile(file *os.File) bool {
	// Read first 512 bytes for detection
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	buf = buf[:n]

	// Check for null bytes (common in binary files)
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}

	// Check for common binary file signatures
	binarySignatures := [][]byte{
		{0x89, 0x50, 0x4E, 0x47}, // PNG
		{0xFF, 0xD8, 0xFF},       // JPEG
		{0x47, 0x49, 0x46},       // GIF
		{0x50, 0x4B, 0x03, 0x04}, // ZIP
		{0x25, 0x50, 0x44, 0x46}, // PDF
	}

	for _, sig := range binarySignatures {
		if bytes.HasPrefix(buf, sig) {
			return true
		}
	}

	return false
}

// isBlockedDevicePath checks if the path is a blocked device file.
func (t *FileReadTool) isBlockedDevicePath(path string) bool {
	blockedPaths := map[string]bool{
		"/dev/zero":     true,
		"/dev/random":   true,
		"/dev/urandom":  true,
		"/dev/full":     true,
		"/dev/stdin":    true,
		"/dev/tty":      true,
		"/dev/console":  true,
		"/dev/stdout":   true,
		"/dev/stderr":   true,
		"/dev/fd/0":     true,
		"/dev/fd/1":     true,
		"/dev/fd/2":     true,
	}
	return blockedPaths[path]
}

// CheckPermissions checks if the file can be read.
func (t *FileReadTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
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

	// Check for sensitive file patterns
	sensitivePatterns := []string{
		".env",
		".git/",
		".ssh/",
		"id_rsa",
		"id_ed25519",
		".pem",
		".key",
		"credentials",
		"secrets",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(filePath, pattern) {
			return &types.PermissionResult{
				Behavior: "ask",
				Message:  fmt.Sprintf("This file may contain sensitive information: %s. Do you want to read it?", filePath),
			}, nil
		}
	}

	return &types.PermissionResult{Behavior: "allow"}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *FileReadTool) UserFacingName(input map[string]interface{}) string {
	if path, ok := input["file_path"].(string); ok {
		return filepath.Base(path)
	}
	return "Read file"
}
