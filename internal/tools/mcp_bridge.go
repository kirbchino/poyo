// Package tools implements MCP tool bridge for dynamic tool registration
package tools

import (
	"context"
	"fmt"
	"sync"
)

// MCPToolDefinition represents a tool from an MCP server
type MCPToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPToolExecutor executes tools on MCP servers
type MCPToolExecutor interface {
	CallTool(ctx context.Context, pluginID string, toolName string, args interface{}) (interface{}, error)
}

// MCPTool wraps an MCP server tool as a Poyo tool
type MCPTool struct {
	BaseTool
	pluginID    string
	toolName    string
	executor    MCPToolExecutor
	inputSchema map[string]interface{}
}

// NewMCPTool creates a new MCP-based tool
func NewMCPTool(pluginID string, tool MCPToolDefinition, executor MCPToolExecutor) *MCPTool {
	// Convert inputSchema to ToolInputJSONSchema
	schema := ToolInputJSONSchema{
		Type: "object",
	}
	if inputSchema, ok := tool.InputSchema["type"].(string); ok {
		schema.Type = inputSchema
	}
	if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
		schema.Properties = make(map[string]map[string]interface{})
		for k, v := range props {
			if m, ok := v.(map[string]interface{}); ok {
				schema.Properties[k] = m
			}
		}
	}
	if req, ok := tool.InputSchema["required"].([]interface{}); ok {
		schema.Required = make([]string, 0, len(req))
		for _, r := range req {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return &MCPTool{
		BaseTool: BaseTool{
			name:              fmt.Sprintf("mcp_%s_%s", pluginID, tool.Name),
			aliases:           []string{tool.Name},
			description:       fmt.Sprintf("🔌 MCP: %s", tool.Description),
			inputSchema:       schema,
			isEnabled:         true,
			isConcurrencySafe: true,
		},
		pluginID:    pluginID,
		toolName:    tool.Name,
		executor:    executor,
		inputSchema: tool.InputSchema,
	}
}

// Call executes the MCP tool
func (t *MCPTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if t.executor == nil {
		return nil, fmt.Errorf("MCP executor not configured")
	}

	result, err := t.executor.CallTool(ctx, t.pluginID, t.toolName, input)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	return &ToolResult{
		Data: result,
	}, nil
}

// MCPToolRegistry manages MCP tools dynamically
type MCPToolRegistry struct {
	mu       sync.RWMutex
	tools    map[string]*MCPTool
	executor MCPToolExecutor
}

// NewMCPToolRegistry creates a new MCP tool registry
func NewMCPToolRegistry(executor MCPToolExecutor) *MCPToolRegistry {
	return &MCPToolRegistry{
		tools:    make(map[string]*MCPTool),
		executor: executor,
	}
}

// RegisterPluginTools registers all tools from an MCP plugin
func (r *MCPToolRegistry) RegisterPluginTools(pluginID string, tools []MCPToolDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tool := range tools {
		mcpTool := NewMCPTool(pluginID, tool, r.executor)
		r.tools[mcpTool.Name()] = mcpTool
		// Also register to global registry
		RegisterTool(mcpTool)
	}
}

// UnregisterPluginTools removes all tools from an MCP plugin
func (r *MCPToolRegistry) UnregisterPluginTools(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	prefix := fmt.Sprintf("mcp_%s_", pluginID)
	for name := range r.tools {
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			UnregisterTool(name)
			delete(r.tools, name)
		}
	}
}

// ListTools returns all registered MCP tools
func (r *MCPToolRegistry) ListTools() []*MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*MCPTool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// GetTool retrieves a specific MCP tool
func (r *MCPToolRegistry) GetTool(name string) *MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[name]
}

// UpdateToolDescription updates a tool description with Kirby theme
func (t *MCPTool) UpdateToolDescription(desc string) {
	t.description = fmt.Sprintf("🔌 MCP: %s", desc)
}

// PluginID returns the plugin ID this tool belongs to
func (t *MCPTool) PluginID() string {
	return t.pluginID
}

// ToolName returns the original tool name on the MCP server
func (t *MCPTool) ToolName() string {
	return t.toolName
}

// InputSchema returns the input schema for the MCP tool
func (t *MCPTool) InputSchema() ToolInputJSONSchema {
	return t.BaseTool.inputSchema
}
