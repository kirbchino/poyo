// Package lsp provides Language Server Protocol support for Poyo.
package lsp

import (
	"context"
	"time"
)

// LSPClient interface for LSP operations
type LSPClient interface {
	Initialize(ctx context.Context, rootPath string) error
	Shutdown(ctx context.Context) error

	// Text synchronization
	DidOpen(ctx context.Context, uri string, languageID string, text string) error
	DidChange(ctx context.Context, uri string, changes []TextDocumentContentChangeEvent) error
	DidClose(ctx context.Context, uri string) error

	// Language features
	GotoDefinition(ctx context.Context, uri string, position Position) ([]Location, error)
	FindReferences(ctx context.Context, uri string, position Position, includeDeclaration bool) ([]Location, error)
	Hover(ctx context.Context, uri string, position Position) (*Hover, error)
	Rename(ctx context.Context, uri string, position Position, newName string) (*WorkspaceEdit, error)
	DocumentSymbol(ctx context.Context, uri string) ([]SymbolInformation, error)
	WorkspaceSymbol(ctx context.Context, query string) ([]SymbolInformation, error)
	Completion(ctx context.Context, uri string, position Position) (*CompletionList, error)
	SignatureHelp(ctx context.Context, uri string, position Position) (*SignatureHelp, error)

	// Diagnostics
	GetDiagnostics(ctx context.Context, uri string) ([]Diagnostic, error)
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`      // 0-based line number
	Character int `json:"character"` // 0-based character offset
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentItem represents an item of text document content
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentContentChangeEvent represents an event describing a change to a text document
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// Hover represents the result of a hover request
type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range,omitempty"`
}

// MarkedString represents a marked string
type MarkedString struct {
	Language string `json:"language,omitempty"`
	Value    string `json:"value"`
}

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Deprecated    bool       `json:"deprecated,omitempty"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// SymbolKind represents a symbol kind
type SymbolKind int

const (
	SymbolFile          SymbolKind = 1
	SymbolModule        SymbolKind = 2
	SymbolNamespace     SymbolKind = 3
	SymbolPackage       SymbolKind = 4
	SymbolClass         SymbolKind = 5
	SymbolMethod        SymbolKind = 6
	SymbolProperty      SymbolKind = 7
	SymbolField         SymbolKind = 8
	SymbolConstructor   SymbolKind = 9
	SymbolEnum          SymbolKind = 10
	SymbolInterface     SymbolKind = 11
	SymbolFunction      SymbolKind = 12
	SymbolVariable      SymbolKind = 13
	SymbolConstant      SymbolKind = 14
	SymbolString        SymbolKind = 15
	SymbolNumber        SymbolKind = 16
	SymbolBoolean       SymbolKind = 17
	SymbolArray         SymbolKind = 18
	SymbolObject        SymbolKind = 19
	SymbolKey           SymbolKind = 20
	SymbolNull          SymbolKind = 21
	SymbolEnumMember    SymbolKind = 22
	SymbolStruct        SymbolKind = 23
	SymbolEvent         SymbolKind = 24
	SymbolOperator      SymbolKind = 25
	SymbolTypeParameter SymbolKind = 26
)

// Diagnostic represents a diagnostic item
type Diagnostic struct {
	Range           Range            `json:"range"`
	Severity        DiagnosticSeverity `json:"severity,omitempty"`
	Code            interface{}      `json:"code,omitempty"`
	Source          string           `json:"source,omitempty"`
	Message         string           `json:"message"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// DiagnosticRelatedInformation represents related information for a diagnostic
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// WorkspaceEdit represents changes to many resources
type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []TextDocumentEdit    `json:"documentChanges,omitempty"`
}

// TextDocumentEdit represents a text document edit
type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// VersionedTextDocumentIdentifier represents a versioned text document identifier
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextEdit represents a text edit
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// CompletionList represents a completion list
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionItem represents a completion item
type CompletionItem struct {
	Label            string            `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string            `json:"detail,omitempty"`
	Documentation    interface{}       `json:"documentation,omitempty"`
	InsertText       string            `json:"insertText,omitempty"`
	InsertTextFormat InsertTextFormat  `json:"insertTextFormat,omitempty"`
}

// CompletionItemKind represents a completion item kind
type CompletionItemKind int

const (
	CompletionText          CompletionItemKind = 1
	CompletionMethod        CompletionItemKind = 2
	CompletionFunction      CompletionItemKind = 3
	CompletionConstructor   CompletionItemKind = 4
	CompletionField         CompletionItemKind = 5
	CompletionVariable      CompletionItemKind = 6
	CompletionClass         CompletionItemKind = 7
	CompletionInterface     CompletionItemKind = 8
	CompletionModule        CompletionItemKind = 9
	CompletionProperty      CompletionItemKind = 10
	CompletionUnit          CompletionItemKind = 11
	CompletionValue         CompletionItemKind = 12
	CompletionEnum          CompletionItemKind = 13
	CompletionKeyword       CompletionItemKind = 14
	CompletionSnippet       CompletionItemKind = 15
	CompletionColor         CompletionItemKind = 16
	CompletionFile          CompletionItemKind = 17
	CompletionReference     CompletionItemKind = 18
	CompletionFolder        CompletionItemKind = 19
	CompletionEnumMember    CompletionItemKind = 20
	CompletionConstant      CompletionItemKind = 21
	CompletionStruct        CompletionItemKind = 22
	CompletionEvent         CompletionItemKind = 23
	CompletionOperator      CompletionItemKind = 24
	CompletionTypeParameter CompletionItemKind = 25
)

// InsertTextFormat represents the format of insertion text
type InsertTextFormat int

const (
	InsertTextPlainText InsertTextFormat = 1
	InsertTextSnippet   InsertTextFormat = 2
)

// SignatureHelp represents signature help information
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

// SignatureInformation represents signature information
type SignatureInformation struct {
	Label           string                 `json:"label"`
	Documentation   interface{}            `json:"documentation,omitempty"`
	Parameters      []ParameterInformation `json:"parameters,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

// ParameterInformation represents parameter information
type ParameterInformation struct {
	Label         string      `json:"label"`
	Documentation interface{} `json:"documentation,omitempty"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// TextDocumentClientCapabilities represents text document client capabilities
type TextDocumentClientCapabilities struct {
	Definition    *DefinitionCapabilities    `json:"definition,omitempty"`
	References    *ReferenceCapabilities    `json:"references,omitempty"`
	Hover         *HoverCapabilities         `json:"hover,omitempty"`
	Rename        *RenameCapabilities        `json:"rename,omitempty"`
	DocumentSymbol *DocumentSymbolCapabilities `json:"documentSymbol,omitempty"`
	Completion    *CompletionCapabilities    `json:"completion,omitempty"`
}

// DefinitionCapabilities represents definition capabilities
type DefinitionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// ReferenceCapabilities represents reference capabilities
type ReferenceCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// HoverCapabilities represents hover capabilities
type HoverCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

// RenameCapabilities represents rename capabilities
type RenameCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	PrepareSupport      bool `json:"prepareSupport,omitempty"`
}

// DocumentSymbolCapabilities represents document symbol capabilities
type DocumentSymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	SymbolKind          *SymbolKindCapabilities `json:"symbolKind,omitempty"`
}

// SymbolKindCapabilities represents symbol kind capabilities
type SymbolKindCapabilities struct {
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

// CompletionCapabilities represents completion capabilities
type CompletionCapabilities struct {
	DynamicRegistration      bool                           `json:"dynamicRegistration,omitempty"`
	CompletionItem           *CompletionItemCapabilities    `json:"completionItem,omitempty"`
	CompletionItemKind       *CompletionItemKindCapabilities `json:"completionItemKind,omitempty"`
	ContextSupport           bool                           `json:"contextSupport,omitempty"`
}

// CompletionItemCapabilities represents completion item capabilities
type CompletionItemCapabilities struct {
	SnippetSupport          bool `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
}

// CompletionItemKindCapabilities represents completion item kind capabilities
type CompletionItemKindCapabilities struct {
	ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
}

// WorkspaceClientCapabilities represents workspace client capabilities
type WorkspaceClientCapabilities struct {
	Symbol *WorkspaceSymbolCapabilities `json:"symbol,omitempty"`
}

// WorkspaceSymbolCapabilities represents workspace symbol capabilities
type WorkspaceSymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	SymbolKind          *SymbolKindCapabilities `json:"symbolKind,omitempty"`
}

// InitializeResult represents the result of initialization
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	DefinitionProvider     interface{}            `json:"definitionProvider,omitempty"`
	ReferencesProvider     interface{}            `json:"referencesProvider,omitempty"`
	HoverProvider          interface{}            `json:"hoverProvider,omitempty"`
	RenameProvider         interface{}            `json:"renameProvider,omitempty"`
	DocumentSymbolProvider interface{}            `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider interface{}           `json:"workspaceSymbolProvider,omitempty"`
	CompletionProvider     *CompletionOptions     `json:"completionProvider,omitempty"`
	SignatureHelpProvider  *SignatureHelpOptions  `json:"signatureHelpProvider,omitempty"`
	TextDocumentSync       interface{}            `json:"textDocumentSync,omitempty"`
}

// CompletionOptions represents completion options
type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
	ResolveProvider     bool     `json:"resolveProvider,omitempty"`
}

// SignatureHelpOptions represents signature help options
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ManagerOptions contains options for the LSP manager
type ManagerOptions struct {
	Timeout time.Duration
}

// DefaultManagerOptions returns default manager options
func DefaultManagerOptions() ManagerOptions {
	return ManagerOptions{
		Timeout: 30 * time.Second,
	}
}
