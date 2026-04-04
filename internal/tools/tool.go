// Package tools contains tool definitions and the tool execution framework.
package tools

import (
	"context"

	"github.com/kirbchino/poyo/internal/types"
)

// ToolInputJSONSchema represents a JSON Schema for tool input.
type ToolInputJSONSchema struct {
	Type       string                            `json:"type"`
	Properties map[string]map[string]interface{} `json:"properties,omitempty"`
	Required   []string                          `json:"required,omitempty"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Data           interface{}              `json:"data"`
	NewMessages    []interface{}            `json:"newMessages,omitempty"`
	ContextModifier func(*ToolUseContext) *ToolUseContext `json:"-"`
	MCPMeta        *MCPMeta                  `json:"mcpMeta,omitempty"`
}

// MCPMeta contains MCP protocol metadata.
type MCPMeta struct {
	Meta             map[string]interface{} `json:"_meta,omitempty"`
	StructuredContent map[string]interface{} `json:"structuredContent,omitempty"`
}

// ToolProgress represents progress information during tool execution.
type ToolProgress struct {
	ToolUseID string      `json:"toolUseId"`
	Data      interface{} `json:"data"`
}

// ToolCallProgress is a callback function for progress updates.
type ToolCallProgress func(progress ToolProgress)

// CanUseToolFunc is a function that checks if a tool can be used.
type CanUseToolFunc func(toolName string, input map[string]interface{}) (*types.PermissionResult, error)

// ToolUseContext provides context for tool execution.
type ToolUseContext struct {
	Options              ToolUseOptions           `json:"options"`
	AbortController      context.Context          `json:"-"`
	FileStateCache       *FileStateCache          `json:"-"`
	Messages             []interface{}            `json:"messages"`
	ToolUseID            string                   `json:"toolUseId,omitempty"`
	PermissionContext    *types.ToolPermissionContext `json:"permissionContext"`
}

// ToolUseOptions contains options for tool execution.
type ToolUseOptions struct {
	Commands              []Command           `json:"commands"`
	Debug                 bool                `json:"debug"`
	MainLoopModel         string              `json:"mainLoopModel"`
	Tools                 []Tool              `json:"tools"`
	Verbose               bool                `json:"verbose"`
	ThinkingConfig        *ThinkingConfig     `json:"thinkingConfig,omitempty"`
	MCPClients            []MCPServerConnection `json:"mcpClients"`
	MCPResources          map[string][]ServerResource `json:"mcpResources"`
	IsNonInteractiveSession bool              `json:"isNonInteractiveSession"`
	MaxBudgetUsd          *float64            `json:"maxBudgetUsd,omitempty"`
	CustomSystemPrompt    string              `json:"customSystemPrompt,omitempty"`
	AppendSystemPrompt    string              `json:"appendSystemPrompt,omitempty"`
	QuerySource           string              `json:"querySource,omitempty"`
}

// Command represents a slash command.
type Command struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Aliases     []string               `json:"aliases,omitempty"`
	Handler     CommandHandler         `json:"-"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// CommandHandler handles a slash command.
type CommandHandler func(ctx context.Context, args []string) error

// ThinkingConfig configures thinking behavior.
type ThinkingConfig struct {
	Enabled       bool   `json:"enabled"`
	MaxThinkingTokens int `json:"maxThinkingTokens,omitempty"`
	Type          string `json:"type,omitempty"` // enabled, disabled, interleaved
}

// MCPServerConnection represents a connection to an MCP server.
type MCPServerConnection struct {
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Tools  []Tool                 `json:"tools,omitempty"`
}

// ServerResource represents a resource from an MCP server.
type ServerResource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
}

// FileStateCache caches file state for efficiency.
type FileStateCache struct {
	cache map[string]*FileState
}

// FileState represents the state of a file.
type FileState struct {
	Path         string `json:"path"`
	Content      string `json:"content,omitempty"`
	ModifiedTime int64  `json:"modifiedTime"`
	Size         int64  `json:"size"`
}

// NewFileStateCache creates a new file state cache.
func NewFileStateCache() *FileStateCache {
	return &FileStateCache{
		cache: make(map[string]*FileState),
	}
}

// Get retrieves a file state from the cache.
func (c *FileStateCache) Get(path string) (*FileState, bool) {
	state, ok := c.cache[path]
	return state, ok
}

// Set stores a file state in the cache.
func (c *FileStateCache) Set(path string, state *FileState) {
	c.cache[path] = state
}

// Tool is the interface for all tools.
type Tool interface {
	// Name returns the tool's name.
	Name() string

	// Aliases returns alternate names for the tool.
	Aliases() []string

	// Description returns a description of what the tool does.
	Description() string

	// InputSchema returns the JSON schema for the tool's input.
	InputSchema() ToolInputJSONSchema

	// Call executes the tool with the given input.
	Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, canUseTool CanUseToolFunc, onProgress ToolCallProgress) (*ToolResult, error)

	// IsEnabled returns whether the tool is currently enabled.
	IsEnabled() bool

	// IsConcurrencySafe returns whether the tool can be safely called concurrently.
	IsConcurrencySafe(input map[string]interface{}) bool

	// IsReadOnly returns whether the tool only reads without modifying.
	IsReadOnly(input map[string]interface{}) bool

	// IsDestructive returns whether the tool performs irreversible operations.
	IsDestructive(input map[string]interface{}) bool

	// CheckPermissions checks if the tool can be used with the given input.
	CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error)

	// UserFacingName returns a human-readable name for the tool.
	UserFacingName(input map[string]interface{}) string

	// MaxResultSizeChars returns the maximum size in characters for tool results.
	MaxResultSizeChars() int
}

// BaseTool provides default implementations for common tool methods.
type BaseTool struct {
	name              string
	aliases           []string
	description       string
	inputSchema       ToolInputJSONSchema
	isEnabled         bool
	isReadOnly        bool
	isDestructive     bool
	isConcurrencySafe bool
	maxResultSize     int
}

// Name returns the tool's name.
func (t *BaseTool) Name() string {
	return t.name
}

// Aliases returns alternate names for the tool.
func (t *BaseTool) Aliases() []string {
	return t.aliases
}

// Description returns a description of what the tool does.
func (t *BaseTool) Description() string {
	return t.description
}

// InputSchema returns the JSON schema for the tool's input.
func (t *BaseTool) InputSchema() ToolInputJSONSchema {
	return t.inputSchema
}

// IsEnabled returns whether the tool is currently enabled.
func (t *BaseTool) IsEnabled() bool {
	return t.isEnabled
}

// IsConcurrencySafe returns whether the tool can be safely called concurrently.
func (t *BaseTool) IsConcurrencySafe(input map[string]interface{}) bool {
	return t.isConcurrencySafe
}

// IsReadOnly returns whether the tool only reads without modifying.
func (t *BaseTool) IsReadOnly(input map[string]interface{}) bool {
	return t.isReadOnly
}

// IsDestructive returns whether the tool performs irreversible operations.
func (t *BaseTool) IsDestructive(input map[string]interface{}) bool {
	return t.isDestructive
}

// CheckPermissions checks if the tool can be used with the given input.
func (t *BaseTool) CheckPermissions(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext) (*types.PermissionResult, error) {
	return &types.PermissionResult{
		Behavior: types.PermissionBehaviorAllow,
	}, nil
}

// UserFacingName returns a human-readable name for the tool.
func (t *BaseTool) UserFacingName(input map[string]interface{}) string {
	return t.name
}

// MaxResultSizeChars returns the maximum size in characters for tool results.
func (t *BaseTool) MaxResultSizeChars() int {
	if t.maxResultSize == 0 {
		return 50000 // Default max size
	}
	return t.maxResultSize
}

// ToolMatchesName checks if a tool matches the given name (primary name or alias).
func ToolMatchesName(tool Tool, name string) bool {
	if tool.Name() == name {
		return true
	}
	for _, alias := range tool.Aliases() {
		if alias == name {
			return true
		}
	}
	return false
}

// FindToolByName finds a tool by name or alias from a list of tools.
func FindToolByName(tools []Tool, name string) Tool {
	for _, tool := range tools {
		if ToolMatchesName(tool, name) {
			return tool
		}
	}
	return nil
}
