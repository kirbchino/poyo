package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Manager manages MCP server connections and tools
type Manager struct {
	mu          sync.RWMutex
	connections map[string]*ServerConnection
	tools       map[string]*Tool
	resources   map[string]*Resource
	prompts     map[string]*Prompt
	configs     map[string]*ServerConfig
	options     MCPOptions
	handlers    []ConnectionEventHandler
}

// NewManager creates a new MCP manager
func NewManager(options MCPOptions) *Manager {
	return &Manager{
		connections: make(map[string]*ServerConnection),
		tools:       make(map[string]*Tool),
		resources:   make(map[string]*Resource),
		prompts:     make(map[string]*Prompt),
		configs:     make(map[string]*ServerConfig),
		options:     options,
	}
}

// AddConfig adds a server configuration
func (m *Manager) AddConfig(name string, config *ServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Name = name
	m.configs[name] = config
}

// RemoveConfig removes a server configuration
func (m *Manager) RemoveConfig(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.configs, name)
}

// ConnectAll connects to all configured servers
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	configs := make(map[string]*ServerConfig)
	for k, v := range m.configs {
		configs[k] = v
	}
	m.mu.RUnlock()

	var errors []error
	for name, config := range configs {
		if config.Disabled {
			continue
		}

		if err := m.Connect(ctx, name); err != nil {
			errors = append(errors, fmt.Errorf("failed to connect to %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("connection errors: %v", errors)
	}

	return nil
}

// Connect connects to a specific server
func (m *Manager) Connect(ctx context.Context, name string) error {
	m.mu.RLock()
	config, ok := m.configs[name]
	if !ok {
		m.mu.RUnlock()
		return fmt.Errorf("server %s not configured", name)
	}
	m.mu.RUnlock()

	// Create transport
	transport, err := m.createTransport(config)
	if err != nil {
		m.setConnectionState(name, StateFailed, err.Error())
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// Create client
	client := NewClient(m.options)

	// Set pending state
	m.setConnectionState(name, StatePending, "")

	// Connect
	connectCtx, cancel := context.WithTimeout(ctx, m.options.ConnectionTimeout)
	defer cancel()

	if err := client.Connect(connectCtx, transport); err != nil {
		m.setConnectionState(name, StateFailed, err.Error())
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Get capabilities
	capabilities := client.GetServerCapabilities()
	serverInfo := client.GetServerVersion()

	// Store connection
	m.mu.Lock()
	m.connections[name] = &ServerConnection{
		Name:         name,
		Type:         config.Type,
		State:        StateConnected,
		Client:       client,
		Capabilities: capabilities,
		ServerInfo:   serverInfo,
		Config:       config,
		ConnectedAt:  time.Now(),
	}
	m.mu.Unlock()

	// Emit event
	m.emitEvent(ConnectionEvent{
		ServerName: name,
		State:      StateConnected,
		Timestamp:  time.Now(),
	})

	// Fetch tools, resources, prompts
	go m.fetchServerCapabilities(context.Background(), name, client)

	return nil
}

// createTransport creates a transport based on config
func (m *Manager) createTransport(config *ServerConfig) (Transport, error) {
	switch config.Type {
	case TransportStdio:
		return NewStdioTransport(config.Command, config.Args, config.Env), nil

	case TransportHTTP, TransportSSE:
		return NewHTTPTransport(config.URL, config.Headers), nil

	case TransportWS:
		return NewWebSocketTransport(config.URL, config.Headers), nil

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.Type)
	}
}

// setConnectionState sets the connection state
func (m *Manager) setConnectionState(name string, state ConnectionState, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[name]
	if !ok {
		conn = &ServerConnection{
			Name:  name,
			State: state,
			Error: errMsg,
		}
		m.connections[name] = conn
	} else {
		conn.State = state
		conn.Error = errMsg
	}

	m.emitEvent(ConnectionEvent{
		ServerName: name,
		State:      state,
		Error:      errMsg,
		Timestamp:  time.Now(),
	})
}

// fetchServerCapabilities fetches tools, resources, and prompts from a server
func (m *Manager) fetchServerCapabilities(ctx context.Context, name string, client *MCPClient) {
	// Fetch tools
	if client.GetServerCapabilities().Tools != nil {
		tools, err := client.ListTools(ctx)
		if err == nil {
			m.mu.Lock()
			prefix := GetMCPPrefix(name)
			for i := range tools {
				tools[i].Name = prefix + tools[i].Name
				tools[i].IsMCP = true
				tools[i].ServerName = name
				m.tools[tools[i].Name] = &tools[i]
			}
			m.mu.Unlock()
		}
	}

	// Fetch resources
	if client.GetServerCapabilities().Resources != nil {
		resources, err := client.ListResources(ctx)
		if err == nil {
			m.mu.Lock()
			for i := range resources {
				resources[i].ServerName = name
				m.resources[resources[i].URI] = &resources[i]
			}
			m.mu.Unlock()
		}
	}

	// Fetch prompts
	if client.GetServerCapabilities().Prompts != nil {
		prompts, err := client.ListPrompts(ctx)
		if err == nil {
			m.mu.Lock()
			for i := range prompts {
				fullName := GetMCPPrefix(name) + prompts[i].Name
				prompts[i].ServerName = name
				m.prompts[fullName] = &prompts[i]
			}
			m.mu.Unlock()
		}
	}
}

// Disconnect disconnects from a server
func (m *Manager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[name]
	if !ok {
		return nil
	}

	if conn.Client != nil {
		conn.Client.Close()
	}

	conn.State = StateDisabled
	delete(m.connections, name)

	// Remove tools, resources, prompts
	prefix := GetMCPPrefix(name)
	for toolName := range m.tools {
		if strings.HasPrefix(toolName, prefix) {
			delete(m.tools, toolName)
		}
	}
	for uri, res := range m.resources {
		if res.ServerName == name {
			delete(m.resources, uri)
		}
	}
	for promptName := range m.prompts {
		if strings.HasPrefix(promptName, prefix) {
			delete(m.prompts, promptName)
		}
	}

	m.emitEvent(ConnectionEvent{
		ServerName: name,
		State:      StateDisabled,
		Timestamp:  time.Now(),
	})

	return nil
}

// GetTools returns all available tools
func (m *Manager) GetTools() []*Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]*Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetTool returns a specific tool
func (m *Manager) GetTool(name string) (*Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, ok := m.tools[name]
	return tool, ok
}

// CallTool calls a tool
func (m *Manager) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	m.mu.RLock()
	tool, ok := m.tools[name]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("tool %s not found", name)
	}

	conn, ok := m.connections[tool.ServerName]
	if !ok || conn.State != StateConnected {
		m.mu.RUnlock()
		return nil, fmt.Errorf("server %s not connected", tool.ServerName)
	}
	client := conn.Client
	m.mu.RUnlock()

	// Remove prefix from tool name
	actualName := strings.TrimPrefix(name, GetMCPPrefix(tool.ServerName))

	return client.CallTool(ctx, actualName, args)
}

// GetResources returns all available resources
func (m *Manager) GetResources() []*Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resources := make([]*Resource, 0, len(m.resources))
	for _, res := range m.resources {
		resources = append(resources, res)
	}
	return resources
}

// ReadResource reads a resource
func (m *Manager) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	m.mu.RLock()
	res, ok := m.resources[uri]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("resource %s not found", uri)
	}

	conn, ok := m.connections[res.ServerName]
	if !ok || conn.State != StateConnected {
		m.mu.RUnlock()
		return nil, fmt.Errorf("server %s not connected", res.ServerName)
	}
	client := conn.Client
	m.mu.RUnlock()

	return client.ReadResource(ctx, uri)
}

// GetPrompts returns all available prompts
func (m *Manager) GetPrompts() []*Prompt {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prompts := make([]*Prompt, 0, len(m.prompts))
	for _, prompt := range m.prompts {
		prompts = append(prompts, prompt)
	}
	return prompts
}

// GetPrompt gets a prompt
func (m *Manager) GetPrompt(ctx context.Context, name string, args map[string]string) ([]PromptMessage, error) {
	m.mu.RLock()
	prompt, ok := m.prompts[name]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("prompt %s not found", name)
	}

	conn, ok := m.connections[prompt.ServerName]
	if !ok || conn.State != StateConnected {
		m.mu.RUnlock()
		return nil, fmt.Errorf("server %s not connected", prompt.ServerName)
	}
	client := conn.Client
	m.mu.RUnlock()

	// Remove prefix from prompt name
	actualName := strings.TrimPrefix(name, GetMCPPrefix(prompt.ServerName))

	return client.GetPrompt(ctx, actualName, args)
}

// GetConnections returns all connections
func (m *Manager) GetConnections() []*ServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*ServerConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns
}

// GetConnection returns a specific connection
func (m *Manager) GetConnection(name string) (*ServerConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[name]
	return conn, ok
}

// OnEvent registers a connection event handler
func (m *Manager) OnEvent(handler ConnectionEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// emitEvent emits a connection event
func (m *Manager) emitEvent(event ConnectionEvent) {
	for _, handler := range m.handlers {
		go handler(event)
	}
}

// Close closes all connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error
	for name, conn := range m.connections {
		if conn.Client != nil {
			if err := conn.Client.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close %s: %w", name, err))
			}
		}
	}

	m.connections = make(map[string]*ServerConnection)
	m.tools = make(map[string]*Tool)
	m.resources = make(map[string]*Resource)
	m.prompts = make(map[string]*Prompt)

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %v", errors)
	}

	return nil
}

// GetMCPPrefix returns the prefix for MCP tool/resource names
func GetMCPPrefix(serverName string) string {
	return "mcp__" + NormalizeNameForMCP(serverName) + "__"
}

// NormalizeNameForMCP normalizes a name for MCP
func NormalizeNameForMCP(name string) string {
	// Replace non-alphanumeric characters with underscore
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			result = append(result, byte(c))
		} else {
			result = append(result, '_')
		}
	}

	// Compress consecutive underscores
	normalized := strings.ReplaceAll(string(result), "__", "_")

	return strings.Trim(normalized, "_")
}

// IsMCPTool checks if a tool name is from an MCP server
func IsMCPTool(name string) bool {
	return strings.HasPrefix(name, "mcp__")
}

// ParseMCPToolName parses an MCP tool name into server and tool name
func ParseMCPToolName(name string) (serverName, toolName string, ok bool) {
	if !strings.HasPrefix(name, "mcp__") {
		return "", "", false
	}

	parts := strings.SplitN(name[5:], "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
