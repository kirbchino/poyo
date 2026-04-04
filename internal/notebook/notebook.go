// Package notebook provides Jupyter Notebook editing capabilities.
package notebook

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// NotebookFormat represents the Jupyter Notebook format version
type NotebookFormat struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
}

// NotebookMetadata represents the notebook-level metadata
type NotebookMetadata struct {
	Kernelspec     map[string]interface{} `json:"kernelspec,omitempty"`
	LanguageInfo   map[string]interface{} `json:"language_info,omitempty"`
	OrigNbformat   int                    `json:"orig_nbformat,omitempty"`
	Title          string                 `json:"title,omitempty"`
	Authors        []map[string]string    `json:"authors,omitempty"`
	Created        time.Time              `json:"created,omitempty"`
	LastModified   time.Time              `json:"last_modified,omitempty"`
	CustomMetadata map[string]interface{} `json:"-"` // Additional custom metadata
}

// CellType represents the type of a notebook cell
type CellType string

const (
	CellTypeCode     CellType = "code"
	CellTypeMarkdown CellType = "markdown"
	CellTypeRaw      CellType = "raw"
)

// OutputType represents the type of cell output
type OutputType string

const (
	OutputTypeStream       OutputType = "stream"
	OutputTypeDisplayData  OutputType = "display_data"
	OutputTypeExecuteResult OutputType = "execute_result"
	OutputTypeError        OutputType = "error"
)

// CellOutput represents a cell execution output
type CellOutput struct {
	OutputType   OutputType              `json:"output_type"`
	Name         string                  `json:"name,omitempty"`         // For stream output (stdout/stderr)
	Text         interface{}             `json:"text,omitempty"`         // String or []string
	Data         map[string]interface{}  `json:"data,omitempty"`         // MIME bundle
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
	Ename        string                  `json:"ename,omitempty"`        // Error name
	Evalue       string                  `json:"evalue,omitempty"`       // Error value
	Traceback    []string                `json:"traceback,omitempty"`
	ExecutionCount int                    `json:"execution_count,omitempty"`
}

// CellMetadata represents cell-level metadata
type CellMetadata struct {
	ID             string                 `json:"id,omitempty"`
	ExecutionCount int                    `json:"execution_count,omitempty"`
	Collapsed      bool                   `json:"collapsed,omitempty"`
	Scrolled       interface{}            `json:"scrolled,omitempty"`
	Format         string                 `json:"format,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Jupyter        map[string]interface{} `json:"jupyter,omitempty"`
	Custom         map[string]interface{} `json:"-"`
}

// Cell represents a single notebook cell
type Cell struct {
	ID         string                 `json:"id,omitempty"`
	CellType   CellType               `json:"cell_type"`
	Source     interface{}            `json:"source"` // String or []string
	Metadata   CellMetadata           `json:"metadata"`
	Outputs    []CellOutput           `json:"outputs,omitempty"`
	ExecutionCount int                 `json:"execution_count,omitempty"`
}

// Notebook represents a complete Jupyter Notebook
type Notebook struct {
	Nbformat  int              `json:"nbformat"`
	NbformatMinor int          `json:"nbformat_minor"`
	Metadata  NotebookMetadata `json:"metadata"`
	Cells     []Cell           `json:"cells"`
}

// NotebookEditOperation represents an edit operation type
type EditOperation string

const (
	EditOpInsert     EditOperation = "insert"
	EditOpReplace    EditOperation = "replace"
	EditOpDelete     EditOperation = "delete"
	EditOpMove       EditOperation = "move"
	EditOpClear      EditOperation = "clear"
	EditOpSplit      EditOperation = "split"
	EditOpMerge      EditOperation = "merge"
	EditOpChangeType EditOperation = "change_type"
)

// EditRequest represents a notebook edit request
type EditRequest struct {
	Operation   EditOperation `json:"operation"`
	CellID      string        `json:"cell_id,omitempty"`
	CellIndex   int           `json:"cell_index,omitempty"`
	CellType    CellType      `json:"cell_type,omitempty"`
	Source      interface{}   `json:"source,omitempty"`
	Position    int           `json:"position,omitempty"`
	FromIndex   int           `json:"from_index,omitempty"`
	ToIndex     int           `json:"to_index,omitempty"`
	Metadata    interface{}   `json:"metadata,omitempty"`
}

// EditResult represents the result of an edit operation
type EditResult struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message,omitempty"`
	AffectedIDs []string `json:"affected_ids,omitempty"`
	NewIndex    int      `json:"new_index,omitempty"`
}

// NewNotebook creates a new empty notebook
func NewNotebook(kernel string) *Notebook {
	now := time.Now()
	return &Notebook{
		Nbformat:      4,
		NbformatMinor: 5,
		Metadata: NotebookMetadata{
			Kernelspec: map[string]interface{}{
				"display_name": kernel,
				"language":     getKernelLanguage(kernel),
				"name":         kernel,
			},
			LanguageInfo: map[string]interface{}{
				"name":       getKernelLanguage(kernel),
				"version":    "3.9.0",
				"codemirror_mode": map[string]interface{}{
					"name":    "ipython",
					"version": 3,
				},
				"file_extension":     ".py",
				"mimetype":           "text/x-python",
				"pygments_lexer":     "ipython3",
				"nbconvert_exporter": "python",
			},
			Created:      now,
			LastModified: now,
		},
		Cells: []Cell{},
	}
}

// getKernelLanguage returns the language for a kernel name
func getKernelLanguage(kernel string) string {
	languages := map[string]string{
		"python3":    "python",
		"python2":    "python",
		"python":     "python",
		"julia":      "julia",
		"r":          "R",
		"javascript": "javascript",
		"typescript": "typescript",
		"go":         "go",
		"scala":      "scala",
		"kotlin":     "kotlin",
	}
	if lang, ok := languages[strings.ToLower(kernel)]; ok {
		return lang
	}
	return kernel
}

// ParseNotebook parses a Jupyter Notebook from JSON
func ParseNotebook(data []byte) (*Notebook, error) {
	var nb Notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, fmt.Errorf("failed to parse notebook: %w", err)
	}

	// Validate notebook format
	if nb.Nbformat < 3 || nb.Nbformat > 4 {
		return nil, fmt.Errorf("unsupported notebook format version: %d", nb.Nbformat)
	}

	// Ensure cells have IDs
	for i := range nb.Cells {
		if nb.Cells[i].ID == "" {
			nb.Cells[i].ID = generateCellID()
		}
	}

	return &nb, nil
}

// ToJSON serializes the notebook to JSON
func (nb *Notebook) ToJSON() ([]byte, error) {
	nb.Metadata.LastModified = time.Now()
	return json.MarshalIndent(nb, "", " ")
}

// AddCell adds a new cell to the notebook
func (nb *Notebook) AddCell(cellType CellType, source string, position int) *Cell {
	cell := Cell{
		ID:       generateCellID(),
		CellType: cellType,
		Source:   source,
		Metadata: CellMetadata{
			ID: generateCellID(),
		},
	}

	if cellType == CellTypeCode {
		cell.Outputs = []CellOutput{}
	}

	if position < 0 || position >= len(nb.Cells) {
		nb.Cells = append(nb.Cells, cell)
	} else {
		nb.Cells = append(nb.Cells[:position], append([]Cell{cell}, nb.Cells[position:]...)...)
	}

	return &cell
}

// GetCell retrieves a cell by ID or index
func (nb *Notebook) GetCell(identifier interface{}) (*Cell, int, error) {
	switch v := identifier.(type) {
	case string:
		for i, cell := range nb.Cells {
			if cell.ID == v {
				return &nb.Cells[i], i, nil
			}
		}
		return nil, -1, fmt.Errorf("cell with ID %s not found", v)
	case int:
		if v < 0 || v >= len(nb.Cells) {
			return nil, -1, fmt.Errorf("cell index %d out of range", v)
		}
		return &nb.Cells[v], v, nil
	default:
		return nil, -1, fmt.Errorf("invalid identifier type: %T", identifier)
	}
}

// UpdateCell updates a cell's source or metadata
func (nb *Notebook) UpdateCell(cellID string, source string, metadata *CellMetadata) error {
	cell, _, err := nb.GetCell(cellID)
	if err != nil {
		return err
	}

	if source != "" {
		cell.Source = source
	}

	if metadata != nil {
		cell.Metadata = *metadata
	}

	return nil
}

// DeleteCell removes a cell from the notebook
func (nb *Notebook) DeleteCell(cellID string) error {
	for i, cell := range nb.Cells {
		if cell.ID == cellID {
			nb.Cells = append(nb.Cells[:i], nb.Cells[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("cell with ID %s not found", cellID)
}

// MoveCell moves a cell from one position to another
func (nb *Notebook) MoveCell(fromIndex, toIndex int) error {
	if fromIndex < 0 || fromIndex >= len(nb.Cells) {
		return fmt.Errorf("from index %d out of range", fromIndex)
	}
	if toIndex < 0 || toIndex >= len(nb.Cells) {
		return fmt.Errorf("to index %d out of range", toIndex)
	}

	cell := nb.Cells[fromIndex]
	nb.Cells = append(nb.Cells[:fromIndex], nb.Cells[fromIndex+1:]...)
	nb.Cells = append(nb.Cells[:toIndex], append([]Cell{cell}, nb.Cells[toIndex:]...)...)

	return nil
}

// SplitCell splits a cell at a given position
func (nb *Notebook) SplitCell(cellIndex, splitPosition int) error {
	cell, _, err := nb.GetCell(cellIndex)
	if err != nil {
		return err
	}

	source := cell.SourceAsString()
	if splitPosition < 0 || splitPosition > len(source) {
		return fmt.Errorf("split position %d out of range", splitPosition)
	}

	// Create two new cells
	firstSource := source[:splitPosition]
	secondSource := source[splitPosition:]

	// Update original cell
	cell.Source = strings.TrimRight(firstSource, "\n")

	// Add new cell after
	newCell := nb.AddCell(cell.CellType, strings.TrimLeft(secondSource, "\n"), cellIndex+1)
	newCell.Metadata = cell.Metadata
	newCell.Metadata.ID = generateCellID()

	return nil
}

// MergeCells merges two adjacent cells
func (nb *Notebook) MergeCells(firstIndex, secondIndex int) error {
	if firstIndex < 0 || firstIndex >= len(nb.Cells)-1 {
		return fmt.Errorf("invalid first index %d", firstIndex)
	}
	if secondIndex != firstIndex+1 {
		return fmt.Errorf("cells must be adjacent for merging")
	}

	first := &nb.Cells[firstIndex]
	second := &nb.Cells[secondIndex]

	// Check cell types match
	if first.CellType != second.CellType {
		return fmt.Errorf("cannot merge cells of different types")
	}

	// Merge sources
	firstSource := first.SourceAsString()
	secondSource := second.SourceAsString()
	first.Source = firstSource + "\n\n" + secondSource

	// Merge outputs for code cells
	if first.CellType == CellTypeCode {
		first.Outputs = append(first.Outputs, second.Outputs...)
	}

	// Remove second cell
	nb.Cells = append(nb.Cells[:secondIndex], nb.Cells[secondIndex+1:]...)

	return nil
}

// ChangeCellType changes the type of a cell
func (nb *Notebook) ChangeCellType(cellIndex int, newType CellType) error {
	if cellIndex < 0 || cellIndex >= len(nb.Cells) {
		return fmt.Errorf("cell index %d out of range", cellIndex)
	}

	cell := &nb.Cells[cellIndex]
	oldType := cell.CellType
	cell.CellType = newType

	// Handle outputs based on type change
	if oldType == CellTypeCode && newType != CellTypeCode {
		cell.Outputs = nil
		cell.ExecutionCount = 0
	} else if oldType != CellTypeCode && newType == CellTypeCode {
		cell.Outputs = []CellOutput{}
		cell.ExecutionCount = 0
	}

	return nil
}

// ClearCellOutput clears the outputs of code cells
func (nb *Notebook) ClearCellOutput(cellIndex int) error {
	if cellIndex < 0 || cellIndex >= len(nb.Cells) {
		return fmt.Errorf("cell index %d out of range", cellIndex)
	}

	cell := &nb.Cells[cellIndex]
	if cell.CellType != CellTypeCode {
		return fmt.Errorf("can only clear outputs of code cells")
	}

	cell.Outputs = []CellOutput{}
	cell.ExecutionCount = 0
	return nil
}

// ClearAllOutputs clears outputs from all code cells
func (nb *Notebook) ClearAllOutputs() {
	for i := range nb.Cells {
		if nb.Cells[i].CellType == CellTypeCode {
			nb.Cells[i].Outputs = []CellOutput{}
			nb.Cells[i].ExecutionCount = 0
		}
	}
}

// SourceAsString returns the cell source as a string
func (c *Cell) SourceAsString() string {
	switch v := c.Source.(type) {
	case string:
		return v
	case []interface{}:
		lines := make([]string, len(v))
		for i, line := range v {
			lines[i] = fmt.Sprintf("%v", line)
		}
		return strings.Join(lines, "")
	case []string:
		return strings.Join(v, "")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// SetSource sets the cell source from a string
func (c *Cell) SetSource(source string) {
	lines := strings.Split(source, "\n")
	if len(lines) == 1 {
		c.Source = source
	} else {
		// Add newlines to all but last line
		sourceLines := make([]string, len(lines))
		for i, line := range lines {
			if i < len(lines)-1 {
				sourceLines[i] = line + "\n"
			} else {
				sourceLines[i] = line
			}
		}
		c.Source = sourceLines
	}
}

// AddOutput adds an output to a code cell
func (c *Cell) AddOutput(output CellOutput) error {
	if c.CellType != CellTypeCode {
		return fmt.Errorf("can only add outputs to code cells")
	}
	c.Outputs = append(c.Outputs, output)
	return nil
}

// CellCount returns the number of cells
func (nb *Notebook) CellCount() int {
	return len(nb.Cells)
}

// GetCellsByType returns all cells of a given type
func (nb *Notebook) GetCellsByType(cellType CellType) []Cell {
	var result []Cell
	for _, cell := range nb.Cells {
		if cell.CellType == cellType {
			result = append(result, cell)
		}
	}
	return result
}

// generateCellID generates a unique cell ID
func generateCellID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

// Validate validates the notebook structure
func (nb *Notebook) Validate() error {
	if nb.Nbformat < 3 || nb.Nbformat > 4 {
		return fmt.Errorf("invalid nbformat: %d", nb.Nbformat)
	}

	cellIDs := make(map[string]bool)
	for i, cell := range nb.Cells {
		// Check for duplicate IDs
		if cell.ID != "" {
			if cellIDs[cell.ID] {
				return fmt.Errorf("duplicate cell ID: %s", cell.ID)
			}
			cellIDs[cell.ID] = true
		}

		// Validate cell type
		switch cell.CellType {
		case CellTypeCode, CellTypeMarkdown, CellTypeRaw:
			// Valid
		default:
			return fmt.Errorf("invalid cell type at index %d: %s", i, cell.CellType)
		}
	}

	return nil
}

// Clone creates a deep copy of the notebook
func (nb *Notebook) Clone() (*Notebook, error) {
	data, err := nb.ToJSON()
	if err != nil {
		return nil, err
	}
	return ParseNotebook(data)
}
