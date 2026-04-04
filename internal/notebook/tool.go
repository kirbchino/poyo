package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// NotebookTool provides the NotebookEdit tool interface
type NotebookTool struct {
	name        string
	description string
}

// NewNotebookTool creates a new NotebookEdit tool
func NewNotebookTool() *NotebookTool {
	return &NotebookTool{
		name:        "NotebookEdit",
		description: "Edit Jupyter Notebook (.ipynb) files. Supports inserting, modifying, deleting, splitting, and merging cells.",
	}
}

// Name returns the tool name
func (t *NotebookTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *NotebookTool) Description() string {
	return t.description
}

// InputSchema returns the JSON schema for the tool input
func (t *NotebookTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"notebook_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the notebook file",
			},
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"read", "insert", "replace", "delete", "move", "split", "merge", "change_type", "clear_output", "clear_all_outputs"},
				"description": "The edit operation to perform",
			},
			"cell_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the cell to modify (optional)",
			},
			"cell_index": map[string]interface{}{
				"type":        "integer",
				"description": "Index of the cell (0-based)",
			},
			"cell_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"code", "markdown", "raw"},
				"description": "Type of cell to insert or change to",
			},
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Cell source content",
			},
			"position": map[string]interface{}{
				"type":        "integer",
				"description": "Position for insert operation",
			},
			"from_index": map[string]interface{}{
				"type":        "integer",
				"description": "Source index for move operation",
			},
			"to_index": map[string]interface{}{
				"type":        "integer",
				"description": "Target index for move operation",
			},
			"split_position": map[string]interface{}{
				"type":        "integer",
				"description": "Character position to split at",
			},
			"new_cell_type": map[string]interface{}{
				"type":        "string",
				"description": "New cell type for change_type operation",
			},
			"kernel": map[string]interface{}{
				"type":        "string",
				"description": "Kernel name for new notebook",
			},
			"create": map[string]interface{}{
				"type":        "boolean",
				"description": "Create new notebook if it doesn't exist",
			},
		},
		"required": []string{"notebook_path", "operation"},
	}
}

// ToolInput represents the parsed tool input
type ToolInput struct {
	NotebookPath  string     `json:"notebook_path"`
	Operation     string     `json:"operation"`
	CellID        string     `json:"cell_id,omitempty"`
	CellIndex     int        `json:"cell_index,omitempty"`
	CellType      CellType   `json:"cell_type,omitempty"`
	Source        string     `json:"source,omitempty"`
	Position      int        `json:"position,omitempty"`
	FromIndex     int        `json:"from_index,omitempty"`
	ToIndex       int        `json:"to_index,omitempty"`
	SplitPosition int        `json:"split_position,omitempty"`
	NewCellType   CellType   `json:"new_cell_type,omitempty"`
	Kernel        string     `json:"kernel,omitempty"`
	Create        bool       `json:"create,omitempty"`
}

// Execute executes the tool with the given input
func (t *NotebookTool) Execute(ctx context.Context, input []byte) (interface{}, error) {
	var req ToolInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate input
	if req.NotebookPath == "" {
		return nil, fmt.Errorf("notebook_path is required")
	}

	if !strings.HasSuffix(req.NotebookPath, ".ipynb") {
		return nil, fmt.Errorf("notebook path must end with .ipynb")
	}

	// Load or create notebook
	nb, err := t.loadOrCreateNotebook(&req)
	if err != nil {
		return nil, err
	}

	// Execute operation
	result, err := t.executeOperation(nb, &req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// loadOrCreateNotebook loads an existing notebook or creates a new one
func (t *NotebookTool) loadOrCreateNotebook(req *ToolInput) (*Notebook, error) {
	// Try to load existing notebook
	// In real implementation, would use file system
	if req.Create {
		kernel := req.Kernel
		if kernel == "" {
			kernel = "python3"
		}
		return NewNotebook(kernel), nil
	}

	return nil, fmt.Errorf("notebook not found: %s", req.NotebookPath)
}

// executeOperation executes the requested operation
func (t *NotebookTool) executeOperation(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	switch req.Operation {
	case "read":
		return t.readNotebook(nb)

	case "insert":
		return t.insertCell(nb, req)

	case "replace":
		return t.replaceCell(nb, req)

	case "delete":
		return t.deleteCell(nb, req)

	case "move":
		return t.moveCell(nb, req)

	case "split":
		return t.splitCell(nb, req)

	case "merge":
		return t.mergeCells(nb, req)

	case "change_type":
		return t.changeCellType(nb, req)

	case "clear_output":
		return t.clearOutput(nb, req)

	case "clear_all_outputs":
		return t.clearAllOutputs(nb)

	default:
		return nil, fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

// ToolResult represents the result of a tool operation
type ToolResult struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Notebook  *Notebook   `json:"notebook,omitempty"`
	Cell      *Cell       `json:"cell,omitempty"`
	Cells     []Cell      `json:"cells,omitempty"`
	CellCount int         `json:"cell_count,omitempty"`
}

// readNotebook reads and returns the notebook structure
func (t *NotebookTool) readNotebook(nb *Notebook) (*ToolResult, error) {
	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Notebook has %d cells", nb.CellCount()),
		Notebook:  nb,
		Cells:     nb.Cells,
		CellCount: nb.CellCount(),
	}, nil
}

// insertCell inserts a new cell
func (t *NotebookTool) insertCell(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	cellType := req.CellType
	if cellType == "" {
		cellType = CellTypeCode
	}

	source := req.Source
	if source == "" && cellType == CellTypeMarkdown {
		source = "# New Cell\n\nAdd your content here."
	}

	position := req.Position
	if position < 0 {
		position = -1 // Append at end
	}

	cell := nb.AddCell(cellType, source, position)

	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Inserted %s cell at position %d", cellType, position),
		Cell:      cell,
		CellCount: nb.CellCount(),
	}, nil
}

// replaceCell replaces a cell's content
func (t *NotebookTool) replaceCell(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	var cell *Cell
	var err error

	if req.CellID != "" {
		cell, _, err = nb.GetCell(req.CellID)
	} else {
		cell, _, err = nb.GetCell(req.CellIndex)
	}

	if err != nil {
		return nil, err
	}

	if req.Source != "" {
		cell.SetSource(req.Source)
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("Updated cell %s", cell.ID),
		Cell:    cell,
	}, nil
}

// deleteCell deletes a cell
func (t *NotebookTool) deleteCell(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	var err error
	cellID := req.CellID

	if cellID == "" {
		cell, _, err := nb.GetCell(req.CellIndex)
		if err != nil {
			return nil, err
		}
		cellID = cell.ID
	}

	err = nb.DeleteCell(cellID)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Deleted cell %s", cellID),
		CellCount: nb.CellCount(),
	}, nil
}

// moveCell moves a cell to a new position
func (t *NotebookTool) moveCell(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	err := nb.MoveCell(req.FromIndex, req.ToIndex)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Moved cell from %d to %d", req.FromIndex, req.ToIndex),
		CellCount: nb.CellCount(),
	}, nil
}

// splitCell splits a cell at a position
func (t *NotebookTool) splitCell(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	err := nb.SplitCell(req.CellIndex, req.SplitPosition)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Split cell at position %d", req.SplitPosition),
		CellCount: nb.CellCount(),
	}, nil
}

// mergeCells merges two adjacent cells
func (t *NotebookTool) mergeCells(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	err := nb.MergeCells(req.FromIndex, req.ToIndex)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Success:   true,
		Message:   fmt.Sprintf("Merged cells at %d and %d", req.FromIndex, req.ToIndex),
		CellCount: nb.CellCount(),
	}, nil
}

// changeCellType changes a cell's type
func (t *NotebookTool) changeCellType(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	newType := req.NewCellType
	if newType == "" {
		newType = req.CellType
	}

	err := nb.ChangeCellType(req.CellIndex, newType)
	if err != nil {
		return nil, err
	}

	cell, _, _ := nb.GetCell(req.CellIndex)

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("Changed cell %d to %s", req.CellIndex, newType),
		Cell:    cell,
	}, nil
}

// clearOutput clears a cell's output
func (t *NotebookTool) clearOutput(nb *Notebook, req *ToolInput) (*ToolResult, error) {
	err := nb.ClearCellOutput(req.CellIndex)
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Success: true,
		Message: fmt.Sprintf("Cleared output of cell %d", req.CellIndex),
	}, nil
}

// clearAllOutputs clears all cell outputs
func (t *NotebookTool) clearAllOutputs(nb *Notebook) (*ToolResult, error) {
	nb.ClearAllOutputs()

	return &ToolResult{
		Success:   true,
		Message:   "Cleared all cell outputs",
		CellCount: nb.CellCount(),
	}, nil
}

// NotebookManager manages multiple notebooks
type NotebookManager struct {
	notebooks map[string]*Notebook
}

// NewNotebookManager creates a new notebook manager
func NewNotebookManager() *NotebookManager {
	return &NotebookManager{
		notebooks: make(map[string]*Notebook),
	}
}

// Open opens a notebook file
func (m *NotebookManager) Open(path string, data []byte) (*Notebook, error) {
	nb, err := ParseNotebook(data)
	if err != nil {
		return nil, err
	}

	m.notebooks[path] = nb
	return nb, nil
}

// Get retrieves a loaded notebook
func (m *NotebookManager) Get(path string) (*Notebook, bool) {
	nb, ok := m.notebooks[path]
	return nb, ok
}

// Create creates a new notebook
func (m *NotebookManager) Create(path string, kernel string) *Notebook {
	nb := NewNotebook(kernel)
	m.notebooks[path] = nb
	return nb
}

// Save saves a notebook
func (m *NotebookManager) Save(path string) ([]byte, error) {
	nb, ok := m.notebooks[path]
	if !ok {
		return nil, fmt.Errorf("notebook not found: %s", path)
	}

	return nb.ToJSON()
}

// Close closes a notebook
func (m *NotebookManager) Close(path string) {
	delete(m.notebooks, path)
}

// List returns all open notebook paths
func (m *NotebookManager) List() []string {
	paths := make([]string, 0, len(m.notebooks))
	for path := range m.notebooks {
		paths = append(paths, path)
	}
	return paths
}

// GetNotebookExtension returns the file extension for notebooks
func GetNotebookExtension() string {
	return ".ipynb"
}

// IsNotebookFile checks if a path is a notebook file
func IsNotebookFile(path string) bool {
	return filepath.Ext(path) == ".ipynb"
}
