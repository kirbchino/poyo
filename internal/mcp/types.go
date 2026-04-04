// Package mcp provides Model Context Protocol integration for Poyo.
package mcp

import (
	"context"
	"time"
)

// TransportType represents the type of MCP transport
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
	TransportHTTP  TransportType = "http"
	TransportWS    TransportType = "ws"
	TransportSDK   TransportType = "sdk"
)

// ConfigScope represents where a configuration is defined
type ConfigScope string

const (
	ScopeLocal     ConfigScope = "local"     // Local project-private
	ScopeUser      ConfigScope = "user"      // User global settings
	ScopeProject   ConfigScope = "project"   // Project .mcp.json
	ScopeDynamic   ConfigScope = "dynamic"   // Command-line
	ScopeEnterprise ConfigScope = "enterprise" // Enterprise managed
	ScopeManaged   ConfigScope = "managed"   // Managed configuration
)

// ConnectionState represents the state of an MCP server connection
type ConnectionState string

const (
	StatePending    ConnectionState = "pending"
	StateConnected  ConnectionState = "connected"
	StateFailed     ConnectionState = "failed"
	StateDisabled   ConnectionState = "disabled"
	StateNeedsAuth  ConnectionState = "needs-auth"
)

// ServerConfig represents an MCP server configuration
type ServerConfig struct {
	// Type is the transport type
	Type TransportType `json:"type,omitempty"`

	// Name is the server identifier
	Name string `json:"name"`

	// Scope indicates where the config is defined
	Scope ConfigScope `json:"scope,omitempty"`

	// Stdio transport fields
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// Network transport fields (SSE/HTTP/WS)
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// HeadersHelper is a script to dynamically get headers
	HeadersHelper string `json:"headersHelper,omitempty"`

	// OAuth configuration
	OAuth *OAuthConfig `json:"oauth,omitempty"`

	// Disabled flag
	Disabled bool `json:"disabled,omitempty"`

	// Timeout for operations
	Timeout time.Duration `json:"timeout,omitempty"`
}

// OAuthConfig represents OAuth 2.0 configuration
type OAuthConfig struct {
	ClientID             string `json:"clientId"`
	CallbackPort         int    `json:"callbackPort,omitempty"`
	AuthServerMetadataURL string `json:"authServerMetadataUrl,omitempty"`

	// XAA (Cross-App Access) configuration
	XAA *XAAConfig `json:"xaa,omitempty"`
}

// XAAConfig represents Cross-App Access configuration
type XAAConfig struct {
	Enabled bool   `json:"enabled"`
	IdPToken string `json:"idPToken,omitempty"`
}

// ServerConnection represents a connected MCP server
type ServerConnection struct {
	Name         string           `json:"name"`
	Type         TransportType    `json:"type"`
	State        ConnectionState  `json:"state"`
	Client       *Client          `json:"-"`
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`
	ServerInfo   *ServerInfo      `json:"serverInfo,omitempty"`
	Config       *ServerConfig    `json:"config,omitempty"`
	Error        string           `json:"error,omitempty"`
	ConnectedAt  time.Time        `json:"connectedAt,omitempty"`
}

// ServerCapabilities represents the capabilities of an MCP server
type ServerCapabilities struct {
	Tools       *ToolsCapability       `json:"tools,omitempty"`
	Resources   *ResourcesCapability   `json:"resources,omitempty"`
	Prompts     *PromptsCapability     `json:"prompts,omitempty"`
	Logging     *LoggingCapability     `json:"logging,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// ToolsCapability indicates tools support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resources support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompts support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// ServerInfo contains server version information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	IsMCP       bool                   `json:"isMcp"`
	ServerName  string                 `json:"serverName,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	ServerName  string                 `json:"serverName,omitempty"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	ServerName  string                 `json:"serverName,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ToolCallResult represents the result of a tool call
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a block of content
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// ConnectionManager manages MCP server connections
type ConnectionManager struct {
	connections map[string]*ServerConnection
	tools       map[string]*Tool
	resources   map[string]*Resource
	prompts     map[string]*Prompt
	configs     map[string]*ServerConfig
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*ServerConnection),
		tools:       make(map[string]*Tool),
		resources:   make(map[string]*Resource),
		prompts:     make(map[string]*Prompt),
		configs:     make(map[string]*ServerConfig),
	}
}

// Client interface for MCP operations
type Client interface {
	Connect(ctx context.Context, transport Transport) error
	Close() error
	GetServerCapabilities() *ServerCapabilities
	GetServerVersion() *ServerInfo
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error)
	ListResources(ctx context.Context) ([]Resource, error)
	ReadResource(ctx context.Context, uri string) (*ResourceContent, error)
	ListPrompts(ctx context.Context) ([]Prompt, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) ([]PromptMessage, error)
}

// Transport interface for MCP communication
type Transport interface {
	Start(ctx context.Context) error
	Close() error
	Send(message []byte) error
	Receive() ([]byte, error)
	OnMessage(handler func([]byte))
	OnError(handler func(error))
}

// ConnectionEvent represents a connection state change event
type ConnectionEvent struct {
	ServerName string          `json:"serverName"`
	State      ConnectionState `json:"state"`
	Error      string          `json:"error,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// ConnectionEventHandler handles connection events
type ConnectionEventHandler func(event ConnectionEvent)

// MCPOptions contains options for MCP initialization
type MCPOptions struct {
	MaxReconnectAttempts int           `json:"maxReconnectAttempts"`
	InitialBackoff       time.Duration `json:"initialBackoff"`
	MaxBackoff           time.Duration `json:"maxBackoff"`
	ConnectionTimeout    time.Duration `json:"connectionTimeout"`
	RequestTimeout       time.Duration `json:"requestTimeout"`
}

// DefaultMCPOptions returns default MCP options
func DefaultMCPOptions() MCPOptions {
	return MCPOptions{
		MaxReconnectAttempts: 5,
		InitialBackoff:       1 * time.Second,
		MaxBackoff:           30 * time.Second,
		ConnectionTimeout:    30 * time.Second,
		RequestTimeout:       60 * time.Second,
	}
}
