package notebook

import (
	"encoding/json"
	"testing"
)

func TestCellType(t *testing.T) {
	types := []CellType{
		CellTypeCode,
		CellTypeMarkdown,
		CellTypeRaw,
	}

	for _, ct := range types {
		if ct == "" {
			t.Error("Cell type should not be empty")
		}
	}
}

func TestOutputType(t *testing.T) {
	types := []OutputType{
		OutputTypeStream,
		OutputTypeDisplayData,
		OutputTypeExecuteResult,
		OutputTypeError,
	}

	for _, ot := range types {
		if ot == "" {
			t.Error("Output type should not be empty")
		}
	}
}

func TestNewNotebook(t *testing.T) {
	nb := NewNotebook("python3")

	if nb == nil {
		t.Fatal("NewNotebook() returned nil")
	}

	if nb.Nbformat != 4 {
		t.Errorf("Nbformat = %d, want 4", nb.Nbformat)
	}

	if nb.Metadata.Kernelspec == nil {
		t.Error("Kernelspec should be set")
	}

	if nb.Metadata.Kernelspec["name"] != "python3" {
		t.Error("Kernelspec name should be python3")
	}

	if nb.CellCount() != 0 {
		t.Errorf("CellCount() = %d, want 0", nb.CellCount())
	}
}

func TestNotebookAddCell(t *testing.T) {
	nb := NewNotebook("python3")

	// Add code cell at end
	cell1 := nb.AddCell(CellTypeCode, "print('hello')", -1)
	if cell1 == nil {
		t.Fatal("AddCell() returned nil")
	}

	if cell1.CellType != CellTypeCode {
		t.Errorf("CellType = %v, want %v", cell1.CellType, CellTypeCode)
	}

	if cell1.ID == "" {
		t.Error("Cell ID should not be empty")
	}

	if nb.CellCount() != 1 {
		t.Errorf("CellCount() = %d, want 1", nb.CellCount())
	}

	// Add markdown cell at beginning
	cell2 := nb.AddCell(CellTypeMarkdown, "# Title", 0)
	if nb.CellCount() != 2 {
		t.Errorf("CellCount() = %d, want 2", nb.CellCount())
	}

	// First cell should be markdown
	if nb.Cells[0].CellType != CellTypeMarkdown {
		t.Error("First cell should be markdown")
	}

	// Add cell in the middle
	cell3 := nb.AddCell(CellTypeCode, "x = 1", 1)
	if nb.CellCount() != 3 {
		t.Errorf("CellCount() = %d, want 3", nb.CellCount())
	}

	// Order should be: markdown, code (cell3), code (cell1)
	if nb.Cells[0].ID != cell2.ID {
		t.Error("First cell should be cell2")
	}
	if nb.Cells[1].ID != cell3.ID {
		t.Error("Second cell should be cell3")
	}
	if nb.Cells[2].ID != cell1.ID {
		t.Error("Third cell should be cell1")
	}
}

func TestNotebookGetCell(t *testing.T) {
	nb := NewNotebook("python3")
	cell := nb.AddCell(CellTypeCode, "print('test')", -1)

	// Get by ID
	got, idx, err := nb.GetCell(cell.ID)
	if err != nil {
		t.Fatalf("GetCell(ID) error: %v", err)
	}

	if got.ID != cell.ID {
		t.Errorf("GetCell(ID) returned wrong cell")
	}

	if idx != 0 {
		t.Errorf("Index = %d, want 0", idx)
	}

	// Get by index
	got2, idx2, err := nb.GetCell(0)
	if err != nil {
		t.Fatalf("GetCell(0) error: %v", err)
	}

	if got2.ID != cell.ID {
		t.Errorf("GetCell(0) returned wrong cell")
	}

	if idx2 != 0 {
		t.Errorf("Index = %d, want 0", idx2)
	}

	// Get non-existent
	_, _, err = nb.GetCell("nonexistent")
	if err == nil {
		t.Error("GetCell(nonexistent) should return error")
	}

	_, _, err = nb.GetCell(999)
	if err == nil {
		t.Error("GetCell(999) should return error")
	}
}

func TestNotebookDeleteCell(t *testing.T) {
	nb := NewNotebook("python3")
	cell1 := nb.AddCell(CellTypeCode, "print('1')", -1)
	cell2 := nb.AddCell(CellTypeCode, "print('2')", -1)

	if nb.CellCount() != 2 {
		t.Fatalf("CellCount() = %d, want 2", nb.CellCount())
	}

	err := nb.DeleteCell(cell1.ID)
	if err != nil {
		t.Fatalf("DeleteCell() error: %v", err)
	}

	if nb.CellCount() != 1 {
		t.Errorf("CellCount() = %d, want 1", nb.CellCount())
	}

	if nb.Cells[0].ID != cell2.ID {
		t.Error("Remaining cell should be cell2")
	}

	// Delete non-existent
	err = nb.DeleteCell("nonexistent")
	if err == nil {
		t.Error("DeleteCell(nonexistent) should return error")
	}
}

func TestNotebookMoveCell(t *testing.T) {
	nb := NewNotebook("python3")
	cell1 := nb.AddCell(CellTypeCode, "print('1')", -1)
	cell2 := nb.AddCell(CellTypeCode, "print('2')", -1)
	cell3 := nb.AddCell(CellTypeCode, "print('3')", -1)

	// Move first to last
	err := nb.MoveCell(0, 2)
	if err != nil {
		t.Fatalf("MoveCell() error: %v", err)
	}

	// Order should be: cell2, cell3, cell1
	if nb.Cells[0].ID != cell2.ID {
		t.Error("First cell should be cell2")
	}
	if nb.Cells[2].ID != cell1.ID {
		t.Error("Last cell should be cell1")
	}

	// Test invalid indices
	err = nb.MoveCell(-1, 0)
	if err == nil {
		t.Error("MoveCell(-1, 0) should return error")
	}

	err = nb.MoveCell(0, 999)
	if err == nil {
		t.Error("MoveCell(0, 999) should return error")
	}
}

func TestNotebookSplitCell(t *testing.T) {
	nb := NewNotebook("python3")
	cell := nb.AddCell(CellTypeCode, "print('hello')\nprint('world')", -1)

	err := nb.SplitCell(0, 17) // Split after "print('hello')\n"
	if err != nil {
		t.Fatalf("SplitCell() error: %v", err)
	}

	if nb.CellCount() != 2 {
		t.Errorf("CellCount() = %d, want 2", nb.CellCount())
	}

	// First cell should have "print('hello')"
	if nb.Cells[0].SourceAsString() != "print('hello')" {
		t.Errorf("First cell source = %q, want 'print('hello')'", nb.Cells[0].SourceAsString())
	}

	// Second cell should have "print('world')"
	if nb.Cells[1].SourceAsString() != "print('world')" {
		t.Errorf("Second cell source = %q, want 'print('world')'", nb.Cells[1].SourceAsString())
	}
}

func TestNotebookMergeCells(t *testing.T) {
	nb := NewNotebook("python3")
	nb.AddCell(CellTypeCode, "x = 1", -1)
	nb.AddCell(CellTypeCode, "y = 2", -1)

	err := nb.MergeCells(0, 1)
	if err != nil {
		t.Fatalf("MergeCells() error: %v", err)
	}

	if nb.CellCount() != 1 {
		t.Errorf("CellCount() = %d, want 1", nb.CellCount())
	}

	// Merged cell should have both sources
	source := nb.Cells[0].SourceAsString()
	if source != "x = 1\n\ny = 2" {
		t.Errorf("Merged source = %q, want 'x = 1\\n\\ny = 2'", source)
	}

	// Test merging different types
	nb.AddCell(CellTypeMarkdown, "# Title", -1)
	err = nb.MergeCells(0, 1)
	if err == nil {
		t.Error("MergeCells should fail for different cell types")
	}
}

func TestNotebookChangeCellType(t *testing.T) {
	nb := NewNotebook("python3")
	cell := nb.AddCell(CellTypeCode, "print('test')", -1)

	// Add some outputs
	cell.Outputs = []CellOutput{
		{OutputType: OutputTypeStream, Name: "stdout", Text: "test\n"},
	}
	cell.ExecutionCount = 1

	// Change to markdown
	err := nb.ChangeCellType(0, CellTypeMarkdown)
	if err != nil {
		t.Fatalf("ChangeCellType() error: %v", err)
	}

	if nb.Cells[0].CellType != CellTypeMarkdown {
		t.Error("Cell type should be markdown")
	}

	// Outputs should be cleared
	if len(nb.Cells[0].Outputs) != 0 {
		t.Error("Outputs should be cleared when changing to markdown")
	}

	// Change back to code
	err = nb.ChangeCellType(0, CellTypeCode)
	if err != nil {
		t.Fatalf("ChangeCellType() error: %v", err)
	}

	// Should have empty outputs slice
	if nb.Cells[0].Outputs == nil {
		t.Error("Code cell should have outputs slice (not nil)")
	}
}

func TestNotebookClearOutput(t *testing.T) {
	nb := NewNotebook("python3")
	cell := nb.AddCell(CellTypeCode, "print('test')", -1)
	cell.Outputs = []CellOutput{
		{OutputType: OutputTypeStream, Name: "stdout", Text: "test\n"},
	}
	cell.ExecutionCount = 5

	err := nb.ClearCellOutput(0)
	if err != nil {
		t.Fatalf("ClearCellOutput() error: %v", err)
	}

	if len(cell.Outputs) != 0 {
		t.Error("Outputs should be empty")
	}

	if cell.ExecutionCount != 0 {
		t.Errorf("ExecutionCount = %d, want 0", cell.ExecutionCount)
	}

	// Clear markdown cell should fail
	nb.AddCell(CellTypeMarkdown, "# Title", -1)
	err = nb.ClearCellOutput(1)
	if err == nil {
		t.Error("ClearCellOutput should fail for markdown cell")
	}
}

func TestNotebookClearAllOutputs(t *testing.T) {
	nb := NewNotebook("python3")

	cell1 := nb.AddCell(CellTypeCode, "print('1')", -1)
	cell1.Outputs = []CellOutput{{OutputType: OutputTypeStream}}
	cell1.ExecutionCount = 1

	cell2 := nb.AddCell(CellTypeMarkdown, "# Title", -1)

	cell3 := nb.AddCell(CellTypeCode, "print('2')", -1)
	cell3.Outputs = []CellOutput{{OutputType: OutputTypeStream}}
	cell3.ExecutionCount = 2

	nb.ClearAllOutputs()

	// Code cells should have cleared outputs
	if len(nb.Cells[0].Outputs) != 0 {
		t.Error("First code cell outputs should be empty")
	}
	if len(nb.Cells[2].Outputs) != 0 {
		t.Error("Second code cell outputs should be empty")
	}
}

func TestNotebookToJSON(t *testing.T) {
	nb := NewNotebook("python3")
	nb.AddCell(CellTypeCode, "print('hello')", -1)
	nb.AddCell(CellTypeMarkdown, "# Title", -1)

	data, err := nb.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	if len(data) == 0 {
		t.Error("ToJSON() returned empty data")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("ToJSON() produced invalid JSON: %v", err)
	}
}

func TestParseNotebook(t *testing.T) {
	// Valid notebook JSON
	validJSON := `{
		"nbformat": 4,
		"nbformat_minor": 5,
		"metadata": {
			"kernelspec": {
				"name": "python3",
				"display_name": "Python 3"
			}
		},
		"cells": [
			{
				"cell_type": "code",
				"source": "print('hello')",
				"metadata": {},
				"outputs": []
			}
		]
	}`

	nb, err := ParseNotebook([]byte(validJSON))
	if err != nil {
		t.Fatalf("ParseNotebook() error: %v", err)
	}

	if nb.Nbformat != 4 {
		t.Errorf("Nbformat = %d, want 4", nb.Nbformat)
	}

	if nb.CellCount() != 1 {
		t.Errorf("CellCount() = %d, want 1", nb.CellCount())
	}

	// Invalid JSON
	_, err = ParseNotebook([]byte(`invalid`))
	if err == nil {
		t.Error("ParseNotebook() should return error for invalid JSON")
	}

	// Unsupported format
	unsupportedJSON := `{"nbformat": 2, "cells": []}`
	_, err = ParseNotebook([]byte(unsupportedJSON))
	if err == nil {
		t.Error("ParseNotebook() should return error for unsupported format")
	}
}

func TestNotebookValidate(t *testing.T) {
	nb := NewNotebook("python3")
	nb.AddCell(CellTypeCode, "print('1')", -1)
	nb.AddCell(CellTypeCode, "print('2')", -1)

	err := nb.Validate()
	if err != nil {
		t.Errorf("Validate() error: %v", err)
	}

	// Create invalid cell type
	nb.Cells[0].CellType = "invalid"
	err = nb.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid cell type")
	}
}

func TestNotebookClone(t *testing.T) {
	nb := NewNotebook("python3")
	nb.AddCell(CellTypeCode, "print('hello')", -1)

	clone, err := nb.Clone()
	if err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	if clone.CellCount() != nb.CellCount() {
		t.Error("Clone should have same cell count")
	}

	// Modify clone should not affect original
	clone.AddCell(CellTypeCode, "print('new')", -1)
	if nb.CellCount() == clone.CellCount() {
		t.Error("Modifying clone should not affect original")
	}
}

func TestCellSourceAsString(t *testing.T) {
	// String source
	cell := Cell{Source: "print('hello')"}
	if cell.SourceAsString() != "print('hello')" {
		t.Errorf("SourceAsString() = %q, want 'print('hello')'", cell.SourceAsString())
	}

	// Array source
	cell = Cell{Source: []string{"line1\n", "line2\n"}}
	expected := "line1\nline2\n"
	if cell.SourceAsString() != expected {
		t.Errorf("SourceAsString() = %q, want %q", cell.SourceAsString(), expected)
	}
}

func TestCellSetSource(t *testing.T) {
	cell := Cell{}

	// Single line
	cell.SetSource("single line")
	if cell.SourceAsString() != "single line" {
		t.Errorf("SetSource() failed for single line")
	}

	// Multiple lines
	cell.SetSource("line1\nline2\nline3")
	source := cell.SourceAsString()
	if !containsStr(source, "line1") || !containsStr(source, "line2") {
		t.Errorf("SetSource() failed for multiple lines: %q", source)
	}
}

func TestCellAddOutput(t *testing.T) {
	// Code cell
	cell := Cell{CellType: CellTypeCode, Outputs: []CellOutput{}}
	output := CellOutput{OutputType: OutputTypeStream, Name: "stdout", Text: "hello\n"}

	err := cell.AddOutput(output)
	if err != nil {
		t.Fatalf("AddOutput() error: %v", err)
	}

	if len(cell.Outputs) != 1 {
		t.Error("Output should be added")
	}

	// Markdown cell
	markdownCell := Cell{CellType: CellTypeMarkdown}
	err = markdownCell.AddOutput(output)
	if err == nil {
		t.Error("AddOutput() should fail for markdown cell")
	}
}

func TestNotebookGetCellsByType(t *testing.T) {
	nb := NewNotebook("python3")
	nb.AddCell(CellTypeCode, "print('1')", -1)
	nb.AddCell(CellTypeMarkdown, "# Title", -1)
	nb.AddCell(CellTypeCode, "print('2')", -1)
	nb.AddCell(CellTypeMarkdown, "## Subtitle", -1)

	codeCells := nb.GetCellsByType(CellTypeCode)
	if len(codeCells) != 2 {
		t.Errorf("Code cells count = %d, want 2", len(codeCells))
	}

	markdownCells := nb.GetCellsByType(CellTypeMarkdown)
	if len(markdownCells) != 2 {
		t.Errorf("Markdown cells count = %d, want 2", len(markdownCells))
	}
}

func TestNotebookManager(t *testing.T) {
	m := NewNotebookManager()

	if m == nil {
		t.Fatal("NewNotebookManager() returned nil")
	}

	// Create notebook
	nb := m.Create("test.ipynb", "python3")
	if nb == nil {
		t.Fatal("Create() returned nil")
	}

	// Get notebook
	got, ok := m.Get("test.ipynb")
	if !ok {
		t.Error("Get() should find created notebook")
	}
	if got != nb {
		t.Error("Get() returned different notebook")
	}

	// List notebooks
	paths := m.List()
	if len(paths) != 1 {
		t.Errorf("List() returned %d paths, want 1", len(paths))
	}

	// Save notebook
	data, err := m.Save("test.ipynb")
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Save() returned empty data")
	}

	// Close notebook
	m.Close("test.ipynb")
	_, ok = m.Get("test.ipynb")
	if ok {
		t.Error("Get() should not find closed notebook")
	}
}

func TestNotebookTool(t *testing.T) {
	tool := NewNotebookTool()

	if tool.Name() != "NotebookEdit" {
		t.Errorf("Name() = %q, want 'NotebookEdit'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema should have properties")
	}

	if _, ok := props["operation"]; !ok {
		t.Error("InputSchema should have operation property")
	}
}

func TestNotebookToolExecute(t *testing.T) {
	tool := NewNotebookTool()

	// Test create notebook
	input := `{
		"notebook_path": "test.ipynb",
		"operation": "insert",
		"cell_type": "code",
		"source": "print('hello')",
		"create": true
	}`

	result, err := tool.Execute(nil, []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	toolResult, ok := result.(*ToolResult)
	if !ok {
		t.Fatal("Execute() should return ToolResult")
	}

	if !toolResult.Success {
		t.Error("Execute() should succeed")
	}

	// Test invalid JSON
	_, err = tool.Execute(nil, []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}

	// Test missing notebook_path
	_, err = tool.Execute(nil, []byte(`{"operation": "read"}`))
	if err == nil {
		t.Error("Execute() should return error for missing notebook_path")
	}

	// Test invalid extension
	_, err = tool.Execute(nil, []byte(`{"notebook_path": "test.txt", "operation": "read"}`))
	if err == nil {
		t.Error("Execute() should return error for invalid extension")
	}
}

func TestGetKernelLanguage(t *testing.T) {
	tests := []struct {
		kernel   string
		expected string
	}{
		{"python3", "python"},
		{"python2", "python"},
		{"julia", "julia"},
		{"r", "R"},
		{"javascript", "javascript"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := getKernelLanguage(tt.kernel)
		if result != tt.expected {
			t.Errorf("getKernelLanguage(%q) = %q, want %q", tt.kernel, result, tt.expected)
		}
	}
}

func TestIsNotebookFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.ipynb", true},
		{"notebook.ipynb", true},
		{"/path/to/notebook.ipynb", true},
		{"test.py", false},
		{"test.txt", false},
		{"ipynb", false},
	}

	for _, tt := range tests {
		result := IsNotebookFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsNotebookFile(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

// Helper function
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsStr(s[1:], substr))))
}
