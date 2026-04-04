// Package tools provides NotebookEdit tool implementation
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// NotebookEditTool edits Jupyter notebook cells
type NotebookEditTool struct {
	BaseTool
}

// NewNotebookEditTool creates a new NotebookEditTool
func NewNotebookEditTool() *NotebookEditTool {
	return &NotebookEditTool{
		BaseTool: BaseTool{
			name:        "NotebookEdit",
			description: "Edit cells in a Jupyter notebook (.ipynb file)",
			isEnabled:   true,
		},
	}
}

// NotebookEditInput represents input for notebook editing
type NotebookEditInput struct {
	NotebookPath string                 `json:"notebook_path"`
	CellNumber   int                    `json:"cell_number"`
	NewSource    string                 `json:"new_source"`
	CellType     string                 `json:"cell_type,omitempty"` // "code" or "markdown"
	EditMode     string                 `json:"edit_mode,omitempty"` // "replace", "insert", "delete"
	CellID       string                 `json:"cell_id,omitempty"`
}

// InputSchema returns the input schema
func (t *NotebookEditTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"notebook_path": {
				"type":        "string",
				"description": "The absolute path to the Jupyter notebook file",
			},
			"cell_number": {
				"type":        "integer",
				"description": "The index of the cell to edit (0-indexed)",
			},
			"new_source": {
				"type":        "string",
				"description": "The new source content for the cell",
			},
			"cell_type": {
				"type":        "string",
				"enum":        []string{"code", "markdown"},
				"description": "The type of cell (default: code)",
			},
			"edit_mode": {
				"type":        "string",
				"enum":        []string{"replace", "insert", "delete"},
				"description": "The edit mode (default: replace)",
			},
			"cell_id": {
				"type":        "string",
				"description": "Optional cell ID for identification",
			},
		},
		Required: []string{"notebook_path", "cell_number", "new_source"},
	}
}

// Call executes the tool
func (t *NotebookEditTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	// Parse input
	notebookPath, _ := input["notebook_path"].(string)
	cellNumber, _ := input["cell_number"].(float64)
	newSource, _ := input["new_source"].(string)
	cellType, _ := input["cell_type"].(string)
	editMode, _ := input["edit_mode"].(string)
	cellID, _ := input["cell_id"].(string)

	if cellType == "" {
		cellType = "code"
	}
	if editMode == "" {
		editMode = "replace"
	}

	// Read notebook
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook: %w", err)
	}

	var notebook map[string]interface{}
	if err := json.Unmarshal(data, &notebook); err != nil {
		return nil, fmt.Errorf("failed to parse notebook: %w", err)
	}

	// Get cells
	cells, ok := notebook["cells"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid notebook format: no cells array")
	}

	cellIdx := int(cellNumber)

	switch editMode {
	case "delete":
		if cellIdx < 0 || cellIdx >= len(cells) {
			return nil, fmt.Errorf("cell index %d out of range", cellIdx)
		}
		cells = append(cells[:cellIdx], cells[cellIdx+1:]...)

	case "insert":
		newCell := map[string]interface{}{
			"cell_type":  cellType,
			"source":     strings.Split(newSource, "\n"),
			"metadata":   map[string]interface{}{},
			"execution_count": nil,
			"outputs":    []interface{}{},
		}
		if cellID != "" {
			newCell["id"] = cellID
		}
		if cellIdx >= len(cells) {
			cells = append(cells, newCell)
		} else {
			cells = append(cells[:cellIdx], append([]interface{}{newCell}, cells[cellIdx:]...)...)
		}

	case "replace":
		if cellIdx < 0 || cellIdx >= len(cells) {
			return nil, fmt.Errorf("cell index %d out of range", cellIdx)
		}
		cell, ok := cells[cellIdx].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid cell at index %d", cellIdx)
		}
		cell["source"] = strings.Split(newSource, "\n")
		if cellType != "" {
			cell["cell_type"] = cellType
		}
		cells[cellIdx] = cell
	}

	notebook["cells"] = cells

	// Write notebook
	output, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notebook: %w", err)
	}

	if err := os.WriteFile(notebookPath, output, 0644); err != nil {
		return nil, fmt.Errorf("failed to write notebook: %w", err)
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"success":      true,
			"message":      fmt.Sprintf("Cell %d %sd in %s", cellIdx, editMode, notebookPath),
			"cell_count":   len(cells),
			"cell_type":    cellType,
			"edit_mode":    editMode,
		},
	}, nil
}

// NotebookReadTool reads Jupyter notebook contents
type NotebookReadTool struct {
	BaseTool
}

// NewNotebookReadTool creates a new NotebookReadTool
func NewNotebookReadTool() *NotebookReadTool {
	return &NotebookReadTool{
		BaseTool: BaseTool{
			name:        "NotebookRead",
			description: "Read cells from a Jupyter notebook (.ipynb file)",
			isEnabled:   true,
		},
	}
}

// InputSchema returns the input schema
func (t *NotebookReadTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"notebook_path": {
				"type":        "string",
				"description": "The absolute path to the Jupyter notebook file",
			},
			"pages": {
				"type":        "string",
				"description": "Optional cell range to read (e.g., '1-5')",
			},
		},
		Required: []string{"notebook_path"},
	}
}

// Call executes the tool
func (t *NotebookReadTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, progress ToolCallProgress) (*ToolResult, error) {
	notebookPath, _ := input["notebook_path"].(string)
	pages, _ := input["pages"].(string)

	// Read notebook
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook: %w", err)
	}

	var notebook map[string]interface{}
	if err := json.Unmarshal(data, &notebook); err != nil {
		return nil, fmt.Errorf("failed to parse notebook: %w", err)
	}

	cells, ok := notebook["cells"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid notebook format: no cells array")
	}

	// Parse page range if provided
	startCell, endCell := 0, len(cells)
	if pages != "" {
		parts := strings.Split(pages, "-")
		if len(parts) == 2 {
			if s, err := parseInt(parts[0]); err == nil && s >= 1 {
				startCell = s - 1
			}
			if e, err := parseInt(parts[1]); err == nil && e <= endCell {
				endCell = e
			}
		}
	}

	// Build output
	var output strings.Builder
	for i := startCell; i < endCell && i < len(cells); i++ {
		cell, ok := cells[i].(map[string]interface{})
		if !ok {
			continue
		}

		cellType, _ := cell["cell_type"].(string)
		source := extractCellSource(cell)

		output.WriteString(fmt.Sprintf("\n--- Cell %d (%s) ---\n", i+1, cellType))
		output.WriteString(source)
		output.WriteString("\n")
	}

	return &ToolResult{
		Data: map[string]interface{}{
			"content":     output.String(),
			"cell_count":  len(cells),
			"cells_read":  endCell - startCell,
			"notebook_path": notebookPath,
		},
	}, nil
}

// parseInt parses an integer from string
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &result)
	return result, err
}

// extractCellSource extracts source from a cell
func extractCellSource(cell map[string]interface{}) string {
	switch v := cell["source"].(type) {
	case string:
		return v
	case []interface{}:
		lines := make([]string, len(v))
		for i, line := range v {
			lines[i] = fmt.Sprintf("%v", line)
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}
