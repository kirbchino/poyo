package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Operation represents an LSP operation
type Operation string

const (
	OperationGotoDefinition  Operation = "goto_definition"
	OperationFindReferences  Operation = "find_references"
	OperationHover           Operation = "hover"
	OperationRename          Operation = "rename"
	OperationDocumentSymbol  Operation = "document_symbol"
	OperationWorkspaceSymbol Operation = "workspace_symbol"
	OperationCompletion      Operation = "completion"
	OperationSignatureHelp   Operation = "signature_help"
)

// Manager manages LSP clients for different languages
type Manager struct {
	mu      sync.RWMutex
	clients map[string]LSPClient // languageID -> client
	options ManagerOptions
}

// NewManager creates a new LSP manager
func NewManager(options ManagerOptions) *Manager {
	return &Manager{
		clients: make(map[string]LSPClient),
		options: options,
	}
}

// Register registers an LSP client for a language
func (m *Manager) Register(languageID string, client LSPClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[languageID] = client
}

// Unregister removes an LSP client for a language
func (m *Manager) Unregister(languageID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, languageID)
}

// GetClient returns the LSP client for a language
func (m *Manager) GetClient(languageID string) (LSPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[languageID]
	return client, ok
}

// DetectLanguage detects the language ID from a file path
func (m *Manager) DetectLanguage(filePath string) string {
	ext := getFileExtension(filePath)

	languageMap := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".jsx":        "javascriptreact",
		".ts":         "typescript",
		".tsx":        "typescriptreact",
		".rs":         "rust",
		".java":       "java",
		".kt":         "kotlin",
		".scala":      "scala",
		".c":          "c",
		".cpp":        "cpp",
		".h":          "c",
		".hpp":        "cpp",
		".cs":         "csharp",
		".rb":         "ruby",
		".php":        "php",
		".swift":      "swift",
		".m":          "objective-c",
		".mm":         "objective-cpp",
		".lua":        "lua",
		".r":          "r",
		".sh":         "bash",
		".bash":       "bash",
		".zsh":        "zsh",
		".ps1":        "powershell",
		".json":       "json",
		".yaml":       "yaml",
		".yml":        "yaml",
		".xml":        "xml",
		".html":       "html",
		".css":        "css",
		".scss":       "scss",
		".less":       "less",
		".sql":        "sql",
		".md":         "markdown",
		".dockerfile": "dockerfile",
		".vue":        "vue",
		".svelte":     "svelte",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	// Check for special files
	base := getBaseName(filePath)
	switch strings.ToLower(base) {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	}

	return ext // Return extension as fallback
}

// getFileExtension extracts the file extension
func getFileExtension(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}
	return strings.ToLower(path[idx:])
}

// getBaseName extracts the base name of a file
func getBaseName(path string) string {
	// Handle both / and \ as path separators
	idx := strings.LastIndexAny(path, "/\\")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

// LSPTool provides the tool interface for LSP operations
type LSPTool struct {
	manager *Manager
}

// NewLSPTool creates a new LSP tool
func NewLSPTool(manager *Manager) *LSPTool {
	return &LSPTool{manager: manager}
}

// Name returns the tool name
func (t *LSPTool) Name() string {
	return "LSP"
}

// Description returns the tool description
func (t *LSPTool) Description() string {
	return `Provides operations for interacting with Language Server Protocol servers.

This tool allows you to perform code navigation and analysis operations:
- goto_definition: Jump to the definition of a symbol
- find_references: Find all references to a symbol
- hover: Get hover information for a symbol
- rename: Rename a symbol across the workspace
- document_symbol: List symbols in a document
- workspace_symbol: Search for symbols across the workspace

LSP operations are highly efficient and should be used whenever you need to
understand code structure, find usages, or navigate codebases. Use this tool
preferentially over grep for these operations.

IMPORTANT: Only use this tool when the task requires planning the implementation
steps of a task that requires writing code. For research tasks where you're
gathering information, searching files, reading files or in general trying to
understand the codebase — do NOT use this tool.`
}

// InputSchema returns the JSON schema for the tool input
func (t *LSPTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "The LSP operation to perform",
				"enum": []string{
					string(OperationGotoDefinition),
					string(OperationFindReferences),
					string(OperationHover),
					string(OperationRename),
					string(OperationDocumentSymbol),
					string(OperationWorkspaceSymbol),
				},
			},
			"uri": map[string]interface{}{
				"type":        "string",
				"description": "The file URI (file://path/to/file)",
			},
			"position": map[string]interface{}{
				"type":        "object",
				"description": "The position in the document (for most operations)",
				"properties": map[string]interface{}{
					"line": map[string]interface{}{
						"type":        "number",
						"description": "0-based line number",
					},
					"character": map[string]interface{}{
						"type":        "number",
						"description": "0-based character offset",
					},
				},
			},
			"newName": map[string]interface{}{
				"type":        "string",
				"description": "The new name for rename operation",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query for workspace symbol operation",
			},
		},
		"required": []string{"operation"},
	}
}

// Execute executes the tool
func (t *LSPTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Operation Operation `json:"operation"`
		URI       string    `json:"uri"`
		Position  *Position `json:"position"`
		NewName   string    `json:"newName"`
		Query     string    `json:"query"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Add timeout context
	ctx, cancel := context.WithTimeout(ctx, t.manager.options.Timeout)
	defer cancel()

	// Get the language client
	languageID := t.manager.DetectLanguage(params.URI)
	client, ok := t.manager.GetClient(languageID)
	if !ok {
		return nil, fmt.Errorf("no LSP client available for language: %s", languageID)
	}

	// Execute the operation
	switch params.Operation {
	case OperationGotoDefinition:
		return t.executeGotoDefinition(ctx, client, params)

	case OperationFindReferences:
		return t.executeFindReferences(ctx, client, params)

	case OperationHover:
		return t.executeHover(ctx, client, params)

	case OperationRename:
		return t.executeRename(ctx, client, params)

	case OperationDocumentSymbol:
		return t.executeDocumentSymbol(ctx, client, params)

	case OperationWorkspaceSymbol:
		return t.executeWorkspaceSymbol(ctx, client, params)

	default:
		return nil, fmt.Errorf("unknown operation: %s", params.Operation)
	}
}

func (t *LSPTool) executeGotoDefinition(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	if params.Position == nil {
		return nil, fmt.Errorf("position is required for goto_definition")
	}

	locations, err := client.GotoDefinition(ctx, params.URI, *params.Position)
	if err != nil {
		return nil, fmt.Errorf("goto_definition failed: %w", err)
	}

	return &LSPResult{
		Operation: OperationGotoDefinition,
		Locations: locations,
	}, nil
}

func (t *LSPTool) executeFindReferences(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	if params.Position == nil {
		return nil, fmt.Errorf("position is required for find_references")
	}

	locations, err := client.FindReferences(ctx, params.URI, *params.Position, true)
	if err != nil {
		return nil, fmt.Errorf("find_references failed: %w", err)
	}

	return &LSPResult{
		Operation: OperationFindReferences,
		Locations: locations,
	}, nil
}

func (t *LSPTool) executeHover(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	if params.Position == nil {
		return nil, fmt.Errorf("position is required for hover")
	}

	hover, err := client.Hover(ctx, params.URI, *params.Position)
	if err != nil {
		return nil, fmt.Errorf("hover failed: %w", err)
	}

	return &LSPResult{
		Operation: OperationHover,
		Hover:     hover,
	}, nil
}

func (t *LSPTool) executeRename(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	if params.Position == nil {
		return nil, fmt.Errorf("position is required for rename")
	}
	if params.NewName == "" {
		return nil, fmt.Errorf("newName is required for rename")
	}

	edit, err := client.Rename(ctx, params.URI, *params.Position, params.NewName)
	if err != nil {
		return nil, fmt.Errorf("rename failed: %w", err)
	}

	return &LSPResult{
		Operation:    OperationRename,
		WorkspaceEdit: edit,
	}, nil
}

func (t *LSPTool) executeDocumentSymbol(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	symbols, err := client.DocumentSymbol(ctx, params.URI)
	if err != nil {
		return nil, fmt.Errorf("document_symbol failed: %w", err)
	}

	return &LSPResult{
		Operation: OperationDocumentSymbol,
		Symbols:   symbols,
	}, nil
}

func (t *LSPTool) executeWorkspaceSymbol(ctx context.Context, client LSPClient, params struct {
	Operation Operation `json:"operation"`
	URI       string    `json:"uri"`
	Position  *Position `json:"position"`
	NewName   string    `json:"newName"`
	Query     string    `json:"query"`
}) (*LSPResult, error) {
	symbols, err := client.WorkspaceSymbol(ctx, params.Query)
	if err != nil {
		return nil, fmt.Errorf("workspace_symbol failed: %w", err)
	}

	return &LSPResult{
		Operation: OperationWorkspaceSymbol,
		Symbols:   symbols,
	}, nil
}

// LSPResult represents the result of an LSP operation
type LSPResult struct {
	Operation     Operation       `json:"operation"`
	Locations     []Location      `json:"locations,omitempty"`
	Hover         *Hover          `json:"hover,omitempty"`
	WorkspaceEdit *WorkspaceEdit  `json:"workspaceEdit,omitempty"`
	Symbols       []SymbolInformation `json:"symbols,omitempty"`
}

// FormatResult formats the result for display
func (r *LSPResult) FormatResult() string {
	var sb strings.Builder

	switch r.Operation {
	case OperationGotoDefinition:
		sb.WriteString("Definition locations:\n")
		for i, loc := range r.Locations {
			sb.WriteString(fmt.Sprintf("  %d. %s:%d:%d\n", i+1,
				strings.TrimPrefix(loc.URI, "file://"),
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1))
		}

	case OperationFindReferences:
		sb.WriteString(fmt.Sprintf("Found %d references:\n", len(r.Locations)))
		for i, loc := range r.Locations {
			sb.WriteString(fmt.Sprintf("  %d. %s:%d:%d\n", i+1,
				strings.TrimPrefix(loc.URI, "file://"),
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1))
		}

	case OperationHover:
		if r.Hover != nil {
			sb.WriteString("Hover information:\n")
			switch v := r.Hover.Contents.(type) {
			case string:
				sb.WriteString(v)
			case MarkedString:
				sb.WriteString(v.Value)
			case []MarkedString:
				for _, ms := range v {
					sb.WriteString(ms.Value)
					sb.WriteString("\n")
				}
			}
		}

	case OperationDocumentSymbol, OperationWorkspaceSymbol:
		sb.WriteString(fmt.Sprintf("Found %d symbols:\n", len(r.Symbols)))
		for i, sym := range r.Symbols {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s", i+1, symbolKindToString(sym.Kind), sym.Name))
			if sym.ContainerName != "" {
				sb.WriteString(fmt.Sprintf(" (in %s)", sym.ContainerName))
			}
			sb.WriteString("\n")
		}

	case OperationRename:
		if r.WorkspaceEdit != nil {
			fileCount := len(r.WorkspaceEdit.Changes)
			changeCount := 0
			for _, edits := range r.WorkspaceEdit.Changes {
				changeCount += len(edits)
			}
			sb.WriteString(fmt.Sprintf("Rename will affect %d files with %d changes\n", fileCount, changeCount))
		}
	}

	return sb.String()
}

// symbolKindToString converts a symbol kind to a string
func symbolKindToString(kind SymbolKind) string {
	switch kind {
	case SymbolFile:
		return "File"
	case SymbolModule:
		return "Module"
	case SymbolNamespace:
		return "Namespace"
	case SymbolPackage:
		return "Package"
	case SymbolClass:
		return "Class"
	case SymbolMethod:
		return "Method"
	case SymbolProperty:
		return "Property"
	case SymbolField:
		return "Field"
	case SymbolConstructor:
		return "Constructor"
	case SymbolEnum:
		return "Enum"
	case SymbolInterface:
		return "Interface"
	case SymbolFunction:
		return "Function"
	case SymbolVariable:
		return "Variable"
	case SymbolConstant:
		return "Constant"
	case SymbolString:
		return "String"
	case SymbolNumber:
		return "Number"
	case SymbolBoolean:
		return "Boolean"
	case SymbolArray:
		return "Array"
	case SymbolObject:
		return "Object"
	case SymbolKey:
		return "Key"
	case SymbolNull:
		return "Null"
	case SymbolEnumMember:
		return "EnumMember"
	case SymbolStruct:
		return "Struct"
	case SymbolEvent:
		return "Event"
	case SymbolOperator:
		return "Operator"
	case SymbolTypeParameter:
		return "TypeParameter"
	default:
		return "Unknown"
	}
}
