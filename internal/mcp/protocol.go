// Package mcp implements the Model Context Protocol for plugin communication
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// Version is the MCP protocol version
const Version = "2024-11-05"

// Message types
type (
	// Request represents an MCP request
	Request struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id,omitempty"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}

	// Response represents an MCP response
	Response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Result  interface{}     `json:"result,omitempty"`
		Error   *Error          `json:"error,omitempty"`
	}

	// Error represents an MCP error
	Error struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data,omitempty"`
	}

	// Notification represents an MCP notification
	Notification struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}
)

// ServerInfo contains server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientInfo contains client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities represents server/client capabilities
type Capabilities struct {
	Tools     *ToolsCapabilities     `json:"tools,omitempty"`
	Resources *ResourcesCapabilities `json:"resources,omitempty"`
	Prompts   *PromptsCapabilities   `json:"prompts,omitempty"`
}

// ToolsCapabilities represents tools capabilities
type ToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapabilities represents resources capabilities
type ResourcesCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapabilities represents prompts capabilities
type PromptsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeParams contains initialization parameters
type InitializeParams struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ClientInfo      ClientInfo   `json:"clientInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

// InitializeResult contains initialization result
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

// Tool represents a tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolCallParams contains tool call parameters
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolResult contains tool execution result
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block
type ContentBlock struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	Data     string      `json:"data,omitempty"`
	MimeType string      `json:"mimeType,omitempty"`
	Resource *Resource   `json:"resource,omitempty"`
}

// Resource represents a resource
type Resource struct {
	URI         string      `json:"uri"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	MimeType    string      `json:"mimeType,omitempty"`
}

// Server represents an MCP server
type Server struct {
	mu          sync.RWMutex
	name        string
	version     string
	tools       map[string]ToolHandler
	resources   map[string]ResourceHandler
	prompts     map[string]PromptHandler
	capabilities Capabilities
	initialized bool

	// IO
	reader  *bufio.Reader
	writer  io.Writer
	encoder *json.Encoder

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// ToolHandler handles tool calls
type ToolHandler func(ctx context.Context, params map[string]interface{}) (*ToolResult, error)

// ResourceHandler handles resource reads
type ResourceHandler func(ctx context.Context, uri string) ([]ContentBlock, error)

// PromptHandler handles prompt requests
type PromptHandler func(ctx context.Context, params map[string]interface{}) (string, error)

// NewServer creates a new MCP server
func NewServer(name, version string, reader io.Reader, writer io.Writer) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		name:      name,
		version:   version,
		tools:     make(map[string]ToolHandler),
		resources: make(map[string]ResourceHandler),
		prompts:   make(map[string]PromptHandler),
		capabilities: Capabilities{
			Tools: &ToolsCapabilities{},
		},
		reader:  bufio.NewReader(reader),
		writer:  writer,
		encoder: json.NewEncoder(writer),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// RegisterTool registers a tool handler
func (s *Server) RegisterTool(name, description string, schema interface{}, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tools[name] = handler
}

// RegisterResource registers a resource handler
func (s *Server) RegisterResource(uri, name, description string, handler ResourceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resources[uri] = handler
}

// RegisterPrompt registers a prompt handler
func (s *Server) RegisterPrompt(name string, handler PromptHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prompts[name] = handler
}

// Run starts the server
func (s *Server) Run() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			if err := s.handleMessage(); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}

// handleMessage handles a single message
func (s *Server) handleMessage() error {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return err
	}

	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return s.sendError(nil, -32700, "Parse error", nil)
	}

	// Handle different methods
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	case "prompts/list":
		return s.handlePromptsList(req)
	case "prompts/get":
		return s.handlePromptsGet(req)
	case "notifications/initialized":
		s.initialized = true
		return nil
	default:
		return s.sendError(req.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize handles initialize request
func (s *Server) handleInitialize(req Request) error {
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.sendError(req.ID, -32602, "Invalid params", nil)
		}
	}

	result := InitializeResult{
		ProtocolVersion: Version,
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: s.capabilities,
	}

	return s.sendResult(req.ID, result)
}

// handleToolsList handles tools/list request
func (s *Server) handleToolsList(req Request) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tools []Tool
	for name := range s.tools {
		// Get schema from registered tools (simplified)
		tools = append(tools, Tool{
			Name:        name,
			Description: "Tool: " + name,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		})
	}

	return s.sendResult(req.ID, map[string]interface{}{
		"tools": tools,
	})
}

// handleToolsCall handles tools/call request
func (s *Server) handleToolsCall(req Request) error {
	var params ToolCallParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.sendError(req.ID, -32602, "Invalid params", nil)
		}
	}

	s.mu.RLock()
	handler, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return s.sendError(req.ID, -32602, fmt.Sprintf("Unknown tool: %s", params.Name), nil)
	}

	result, err := handler(s.ctx, params.Arguments)
	if err != nil {
		return s.sendError(req.ID, -32603, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// handleResourcesList handles resources/list request
func (s *Server) handleResourcesList(req Request) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var resources []Resource
	for uri := range s.resources {
		resources = append(resources, Resource{
			URI:  uri,
			Name: uri,
		})
	}

	return s.sendResult(req.ID, map[string]interface{}{
		"resources": resources,
	})
}

// handleResourcesRead handles resources/read request
func (s *Server) handleResourcesRead(req Request) error {
	var params struct {
		URI string `json:"uri"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.sendError(req.ID, -32602, "Invalid params", nil)
		}
	}

	s.mu.RLock()
	handler, ok := s.resources[params.URI]
	s.mu.RUnlock()

	if !ok {
		return s.sendError(req.ID, -32602, fmt.Sprintf("Unknown resource: %s", params.URI), nil)
	}

	contents, err := handler(s.ctx, params.URI)
	if err != nil {
		return s.sendError(req.ID, -32603, err.Error(), nil)
	}

	return s.sendResult(req.ID, map[string]interface{}{
		"contents": contents,
	})
}

// handlePromptsList handles prompts/list request
func (s *Server) handlePromptsList(req Request) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var prompts []map[string]interface{}
	for name := range s.prompts {
		prompts = append(prompts, map[string]interface{}{
			"name": name,
		})
	}

	return s.sendResult(req.ID, map[string]interface{}{
		"prompts": prompts,
	})
}

// handlePromptsGet handles prompts/get request
func (s *Server) handlePromptsGet(req Request) error {
	var params struct {
		Name string `json:"name"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.sendError(req.ID, -32602, "Invalid params", nil)
		}
	}

	s.mu.RLock()
	handler, ok := s.prompts[params.Name]
	s.mu.RUnlock()

	if !ok {
		return s.sendError(req.ID, -32602, fmt.Sprintf("Unknown prompt: %s", params.Name), nil)
	}

	content, err := handler(s.ctx, nil)
	if err != nil {
		return s.sendError(req.ID, -32603, err.Error(), nil)
	}

	return s.sendResult(req.ID, map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": content,
				},
			},
		},
	})
}

// sendResult sends a successful response
func (s *Server) sendResult(id interface{}, result interface{}) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.encoder.Encode(resp)
}

// sendError sends an error response
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	return s.encoder.Encode(resp)
}

// Shutdown shuts down the server
func (s *Server) Shutdown() {
	s.cancel()
}

// Client represents an MCP client
type Client struct {
	mu          sync.Mutex
	name        string
	version     string
	reader      *bufio.Reader
	writer      io.Writer
	encoder     *json.Encoder
	decoder     *json.Decoder
	requestID   int
	pending     map[int]chan *Response
	serverInfo  ServerInfo
	capabilities Capabilities
}

// NewClient creates a new MCP client
func NewClient(name, version string, reader io.Reader, writer io.Writer) *Client {
	return &Client{
		name:     name,
		version:  version,
		reader:   bufio.NewReader(reader),
		writer:   writer,
		encoder:  json.NewEncoder(writer),
		decoder:  json.NewDecoder(reader),
		pending:  make(map[int]chan *Response),
	}
}

// Initialize initializes the connection
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: Version,
		ClientInfo: ClientInfo{
			Name:    c.name,
			Version: c.version,
		},
		Capabilities: Capabilities{
			Tools: &ToolsCapabilities{},
		},
	}

	result, err := c.call(ctx, "initialize", params)
	if err != nil {
		return nil, err
	}

	var initResult InitializeResult
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		return nil, err
	}

	c.serverInfo = initResult.ServerInfo
	c.capabilities = initResult.Capabilities

	// Send initialized notification
	if err := c.notify("notifications/initialized", nil); err != nil {
		return nil, err
	}

	return &initResult, nil
}

// ListTools lists available tools
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	result, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var listResult struct {
		Tools []Tool `json:"tools"`
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, err
	}

	return listResult.Tools, nil
}

// CallTool calls a tool
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	params := ToolCallParams{
		Name:      name,
		Arguments: args,
	}

	result, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var toolResult ToolResult
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resultBytes, &toolResult); err != nil {
		return nil, err
	}

	return &toolResult, nil
}

// call makes a JSON-RPC call
func (c *Client) call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	c.mu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}

	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		req.Params = paramsBytes
	}

	// Send request
	if err := c.encoder.Encode(req); err != nil {
		return nil, err
	}

	// Wait for response
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			var resp Response
			if err := c.decoder.Decode(&resp); err != nil {
				return nil, err
			}

			if resp.ID == id {
				if resp.Error != nil {
					return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
				}
				return resp.Result, nil
			}
		}
	}
}

// notify sends a notification
func (c *Client) notify(method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
	}

	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return err
		}
		notif.Params = paramsBytes
	}

	return c.encoder.Encode(notif)
}

// GetServerInfo returns server information
func (c *Client) GetServerInfo() ServerInfo {
	return c.serverInfo
}

// GetCapabilities returns server capabilities
func (c *Client) GetCapabilities() Capabilities {
	return c.capabilities
}

// Transport represents a transport layer
type Transport interface {
	Read() ([]byte, error)
	Write([]byte) error
	Close() error
}

// StdioTransportImpl implements stdio transport
type StdioTransportImpl struct {
	reader io.Reader
	writer io.Writer
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(r io.Reader, w io.Writer) *StdioTransportImpl {
	return &StdioTransportImpl{
		reader: r,
		writer: w,
	}
}

// Read reads from stdin
func (t *StdioTransportImpl) Read() ([]byte, error) {
	var buf []byte
	_, err := t.reader.Read(buf)
	return buf, err
}

// Write writes to stdout
func (t *StdioTransportImpl) Write(data []byte) error {
	_, err := t.writer.Write(data)
	return err
}

// Close closes the transport
func (t *StdioTransportImpl) Close() error {
	return nil
}

// Progress represents progress information
type Progress struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"`
	Total         float64 `json:"total,omitempty"`
}

// ProgressNotification represents a progress notification
type ProgressNotification struct {
	ProgressToken string      `json:"progressToken"`
	Progress      interface{} `json:"progress"`
	Total         interface{} `json:"total,omitempty"`
}

// CreateProgressToken creates a unique progress token
func CreateProgressToken() string {
	return fmt.Sprintf("progress_%d", time.Now().UnixNano())
}
