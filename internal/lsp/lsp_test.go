package lsp

import (
	"context"
	"testing"
	"time"
)

func TestSymbolKind(t *testing.T) {
	kinds := []SymbolKind{
		SymbolFile,
		SymbolClass,
		SymbolFunction,
		SymbolVariable,
		SymbolInterface,
	}

	for _, k := range kinds {
		if symbolKindToString(k) == "" {
			t.Errorf("Symbol kind %d should have a string representation", k)
		}
	}
}

func TestDiagnosticSeverity(t *testing.T) {
	severities := []DiagnosticSeverity{
		SeverityError,
		SeverityWarning,
		SeverityInformation,
		SeverityHint,
	}

	for _, s := range severities {
		if int(s) == 0 {
			t.Errorf("Diagnostic severity should have a non-zero value")
		}
	}
}

func TestCompletionItemKind(t *testing.T) {
	kinds := []CompletionItemKind{
		CompletionText,
		CompletionMethod,
		CompletionFunction,
		CompletionClass,
		CompletionVariable,
	}

	for _, k := range kinds {
		if int(k) == 0 {
			t.Errorf("Completion item kind should have a non-zero value")
		}
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.clients == nil {
		t.Error("clients map should be initialized")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	client := &MockLSPClient{}

	m.Register("go", client)

	if len(m.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(m.clients))
	}
}

func TestManagerGetClient(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	client := &MockLSPClient{}
	m.Register("go", client)

	got, ok := m.GetClient("go")
	if !ok {
		t.Fatal("Client should be found")
	}
	if got != client {
		t.Error("Wrong client returned")
	}

	_, ok = m.GetClient("python")
	if ok {
		t.Error("Python client should not be found")
	}
}

func TestManagerUnregister(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	client := &MockLSPClient{}
	m.Register("go", client)

	m.Unregister("go")

	if len(m.clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(m.clients))
	}
}

func TestDetectLanguage(t *testing.T) {
	m := NewManager(DefaultManagerOptions())

	tests := []struct {
		filePath   string
		expectLang string
	}{
		{"/path/to/file.go", "go"},
		{"/path/to/file.py", "python"},
		{"/path/to/file.js", "javascript"},
		{"/path/to/file.ts", "typescript"},
		{"/path/to/file.rs", "rust"},
		{"/path/to/Dockerfile", "dockerfile"},
		{"/path/to/Makefile", "makefile"},
		{"/path/to/file.unknown", ".unknown"},
	}

	for _, tt := range tests {
		result := m.DetectLanguage(tt.filePath)
		if result != tt.expectLang {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filePath, result, tt.expectLang)
		}
	}
}

func TestLSPTool(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	tool := NewLSPTool(m)

	if tool.Name() != "LSP" {
		t.Errorf("Name() = %q, want 'LSP'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}
}

func TestLSPToolExecute(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	client := &MockLSPClient{}
	m.Register("go", client)

	tool := NewLSPTool(m)

	// Test goto_definition
	input := []byte(`{
		"operation": "goto_definition",
		"uri": "file:///path/to/file.go",
		"position": {"line": 10, "character": 5}
	}`)

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	lspResult, ok := result.(*LSPResult)
	if !ok {
		t.Fatal("Result should be *LSPResult")
	}

	if lspResult.Operation != OperationGotoDefinition {
		t.Errorf("Operation = %v, want %v", lspResult.Operation, OperationGotoDefinition)
	}
}

func TestLSPToolExecuteNoClient(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	tool := NewLSPTool(m)

	input := []byte(`{
		"operation": "goto_definition",
		"uri": "file:///path/to/file.unknown",
		"position": {"line": 10, "character": 5}
	}`)

	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("Execute() should return error for unknown language")
	}
}

func TestLSPToolExecuteInvalidInput(t *testing.T) {
	m := NewManager(DefaultManagerOptions())
	tool := NewLSPTool(m)

	// Invalid JSON
	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}

	// Missing position for goto_definition
	_, err = tool.Execute(context.Background(), []byte(`{
		"operation": "goto_definition",
		"uri": "file:///path/to/file.go"
	}`))
	if err == nil {
		t.Error("Execute() should return error for missing position")
	}

	// Missing newName for rename
	_, err = tool.Execute(context.Background(), []byte(`{
		"operation": "rename",
		"uri": "file:///path/to/file.go",
		"position": {"line": 10, "character": 5}
	}`))
	if err == nil {
		t.Error("Execute() should return error for missing newName")
	}
}

func TestLSPResultFormatResult(t *testing.T) {
	tests := []struct {
		result   *LSPResult
		contains string
	}{
		{
			result: &LSPResult{
				Operation: OperationGotoDefinition,
				Locations: []Location{
					{URI: "file:///path/to/file.go", Range: Range{Start: Position{Line: 10, Character: 5}}},
				},
			},
			contains: "Definition locations",
		},
		{
			result: &LSPResult{
				Operation: OperationFindReferences,
				Locations: []Location{
					{URI: "file:///path/to/file.go", Range: Range{Start: Position{Line: 10, Character: 5}}},
				},
			},
			contains: "Found 1 references",
		},
		{
			result: &LSPResult{
				Operation: OperationDocumentSymbol,
				Symbols: []SymbolInformation{
					{Name: "MyFunc", Kind: SymbolFunction},
				},
			},
			contains: "Found 1 symbols",
		},
	}

	for _, tt := range tests {
		output := tt.result.FormatResult()
		if !containsString(output, tt.contains) {
			t.Errorf("FormatResult() should contain %q, got %q", tt.contains, output)
		}
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.go", ".go"},
		{"/path/to/file.test.js", ".js"},
		{"file.py", ".py"},
		{"/no/extension", ""},
	}

	for _, tt := range tests {
		result := getFileExtension(tt.path)
		if result != tt.expected {
			t.Errorf("getFileExtension(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestGetBaseName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.go", "file.go"},
		{"file.go", "file.go"},
		{"/path/to/Dockerfile", "Dockerfile"},
		{"/path\\to\\file.go", "file.go"}, // Windows path
	}

	for _, tt := range tests {
		result := getBaseName(tt.path)
		if result != tt.expected {
			t.Errorf("getBaseName(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestSymbolKindToString(t *testing.T) {
	tests := []struct {
		kind     SymbolKind
		expected string
	}{
		{SymbolClass, "Class"},
		{SymbolFunction, "Function"},
		{SymbolVariable, "Variable"},
		{SymbolInterface, "Interface"},
		{SymbolKind(999), "Unknown"},
	}

	for _, tt := range tests {
		result := symbolKindToString(tt.kind)
		if result != tt.expected {
			t.Errorf("symbolKindToString(%v) = %q, want %q", tt.kind, result, tt.expected)
		}
	}
}

func TestDefaultManagerOptions(t *testing.T) {
	opts := DefaultManagerOptions()

	if opts.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", opts.Timeout)
	}
}

// MockLSPClient is a mock implementation of LSPClient
type MockLSPClient struct{}

func (m *MockLSPClient) Initialize(ctx context.Context, rootPath string) error {
	return nil
}

func (m *MockLSPClient) Shutdown(ctx context.Context) error {
	return nil
}

func (m *MockLSPClient) DidOpen(ctx context.Context, uri string, languageID string, text string) error {
	return nil
}

func (m *MockLSPClient) DidChange(ctx context.Context, uri string, changes []TextDocumentContentChangeEvent) error {
	return nil
}

func (m *MockLSPClient) DidClose(ctx context.Context, uri string) error {
	return nil
}

func (m *MockLSPClient) GotoDefinition(ctx context.Context, uri string, position Position) ([]Location, error) {
	return []Location{
		{URI: "file:///path/to/definition.go", Range: Range{Start: Position{Line: 5, Character: 0}}},
	}, nil
}

func (m *MockLSPClient) FindReferences(ctx context.Context, uri string, position Position, includeDeclaration bool) ([]Location, error) {
	return []Location{
		{URI: uri, Range: Range{Start: position}},
	}, nil
}

func (m *MockLSPClient) Hover(ctx context.Context, uri string, position Position) (*Hover, error) {
	return &Hover{Contents: "mock hover info"}, nil
}

func (m *MockLSPClient) Rename(ctx context.Context, uri string, position Position, newName string) (*WorkspaceEdit, error) {
	return &WorkspaceEdit{
		Changes: map[string][]TextEdit{
			uri: {{Range: Range{Start: position}, NewText: newName}},
		},
	}, nil
}

func (m *MockLSPClient) DocumentSymbol(ctx context.Context, uri string) ([]SymbolInformation, error) {
	return []SymbolInformation{
		{Name: "MockFunc", Kind: SymbolFunction},
	}, nil
}

func (m *MockLSPClient) WorkspaceSymbol(ctx context.Context, query string) ([]SymbolInformation, error) {
	return []SymbolInformation{
		{Name: "MockSymbol", Kind: SymbolVariable},
	}, nil
}

func (m *MockLSPClient) Completion(ctx context.Context, uri string, position Position) (*CompletionList, error) {
	return &CompletionList{
		Items: []CompletionItem{
			{Label: "mockCompletion"},
		},
	}, nil
}

func (m *MockLSPClient) SignatureHelp(ctx context.Context, uri string, position Position) (*SignatureHelp, error) {
	return &SignatureHelp{}, nil
}

func (m *MockLSPClient) GetDiagnostics(ctx context.Context, uri string) ([]Diagnostic, error) {
	return []Diagnostic{}, nil
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsString(s[1:], substr))))
}
