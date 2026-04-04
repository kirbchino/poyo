package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MCPClient implements the Client interface
type MCPClient struct {
	mu          sync.RWMutex
	transport   Transport
	capabilities *ServerCapabilities
	serverInfo  *ServerInfo
	requestID   int64
	pending     map[interface{}]chan *JSONRPCMessage
	closed      bool
	options     MCPOptions
}

// NewClient creates a new MCP client
func NewClient(options MCPOptions) *MCPClient {
	return &MCPClient{
		pending: make(map[interface{}]chan *JSONRPCMessage),
		options: options,
	}
}

// Connect connects to a server using the provided transport
func (c *MCPClient) Connect(ctx context.Context, transport Transport) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport != nil {
		return fmt.Errorf("already connected")
	}

	// Set up message handler
	transport.OnMessage(func(data []byte) {
		c.handleMessage(data)
	})

	transport.OnError(func(err error) {
		c.handleError(err)
	})

	// Start transport
	if err := transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	c.transport = transport

	// Initialize connection
	if err := c.initialize(ctx); err != nil {
		c.transport = nil
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// initialize performs the MCP initialization handshake
func (c *MCPClient) initialize(ctx context.Context) error {
	// Send initialize request
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":       map[string]interface{}{},
			"resources":   map[string]interface{}{},
			"prompts":     map[string]interface{}{},
			"elicitation": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "poyo",
			"version": "1.0.0",
		},
	}

	result, err := c.request(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	// Parse result
	var initResult struct {
		ProtocolVersion string             `json:"protocolVersion"`
		Capabilities    *ServerCapabilities `json:"capabilities"`
		ServerInfo      *ServerInfo         `json:"serverInfo"`
	}

	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.capabilities = initResult.Capabilities
	c.serverInfo = initResult.ServerInfo

	// Send initialized notification
	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// handleMessage handles an incoming message
func (c *MCPClient) handleMessage(data []byte) {
	msg, err := ParseJSONRPCMessage(data)
	if err != nil {
		c.handleError(fmt.Errorf("failed to parse message: %w", err))
		return
	}

	// Handle response to a request
	if msg.ID != nil && (msg.Result != nil || msg.Error != nil) {
		c.mu.RLock()
		ch, ok := c.pending[msg.ID]
		c.mu.RUnlock()

		if ok {
			select {
			case ch <- msg:
			default:
				// Channel full or closed
			}
		}
		return
	}

	// Handle notification
	if msg.Method != "" {
		// TODO: Handle notifications
		return
	}

	c.handleError(fmt.Errorf("unexpected message type"))
}

// handleError handles an error
func (c *MCPClient) handleError(err error) {
	// TODO: Log error or notify handler
	fmt.Printf("[MCP Error] %v\n", err)
}

// request sends a request and waits for a response
func (c *MCPClient) request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		msg.Params = paramsJSON
	}

	// Register pending response
	ch := make(chan *JSONRPCMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	// Send request
	data, err := EncodeJSONRPCMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	if err := c.transport.Send(data); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Wait for response
	select {
	case response := <-ch:
		if response.Error != nil {
			return nil, fmt.Errorf("request error: %s (code %d)", response.Error.Message, response.Error.Code)
		}
		return response.Result, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	case <-time.After(c.options.RequestTimeout):
		return nil, fmt.Errorf("request timeout")
	}
}

// notify sends a notification (no response expected)
func (c *MCPClient) notify(ctx context.Context, method string, params interface{}) error {
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
	}

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		msg.Params = paramsJSON
	}

	data, err := EncodeJSONRPCMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return c.transport.Send(data)
}

// Close closes the connection
func (c *MCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close pending channels
	for _, ch := range c.pending {
		close(ch)
	}
	c.pending = make(map[interface{}]chan *JSONRPCMessage)

	if c.transport != nil {
		return c.transport.Close()
	}

	return nil
}

// GetServerCapabilities returns the server capabilities
func (c *MCPClient) GetServerCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// GetServerVersion returns the server version
func (c *MCPClient) GetServerVersion() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// ListTools lists available tools
func (c *MCPClient) ListTools(ctx context.Context) ([]Tool, error) {
	result, err := c.request(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list request failed: %w", err)
	}

	var listResult struct {
		Tools []Tool `json:"tools"`
	}

	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/list result: %w", err)
	}

	return listResult.Tools, nil
}

// CallTool calls a tool
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.request(ctx, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("tools/call request failed: %w", err)
	}

	var callResult ToolCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call result: %w", err)
	}

	return &callResult, nil
}

// ListResources lists available resources
func (c *MCPClient) ListResources(ctx context.Context) ([]Resource, error) {
	result, err := c.request(ctx, "resources/list", nil)
	if err != nil {
		return nil, fmt.Errorf("resources/list request failed: %w", err)
	}

	var listResult struct {
		Resources []Resource `json:"resources"`
	}

	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse resources/list result: %w", err)
	}

	return listResult.Resources, nil
}

// ReadResource reads a resource
func (c *MCPClient) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	params := map[string]interface{}{
		"uri": uri,
	}

	result, err := c.request(ctx, "resources/read", params)
	if err != nil {
		return nil, fmt.Errorf("resources/read request failed: %w", err)
	}

	var readResult struct {
		Contents []ResourceContent `json:"contents"`
	}

	if err := json.Unmarshal(result, &readResult); err != nil {
		return nil, fmt.Errorf("failed to parse resources/read result: %w", err)
	}

	if len(readResult.Contents) == 0 {
		return nil, fmt.Errorf("no content returned")
	}

	return &readResult.Contents[0], nil
}

// ListPrompts lists available prompts
func (c *MCPClient) ListPrompts(ctx context.Context) ([]Prompt, error) {
	result, err := c.request(ctx, "prompts/list", nil)
	if err != nil {
		return nil, fmt.Errorf("prompts/list request failed: %w", err)
	}

	var listResult struct {
		Prompts []Prompt `json:"prompts"`
	}

	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse prompts/list result: %w", err)
	}

	return listResult.Prompts, nil
}

// GetPrompt gets a prompt
func (c *MCPClient) GetPrompt(ctx context.Context, name string, args map[string]string) ([]PromptMessage, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.request(ctx, "prompts/get", params)
	if err != nil {
		return nil, fmt.Errorf("prompts/get request failed: %w", err)
	}

	var getResult struct {
		Messages []PromptMessage `json:"messages"`
	}

	if err := json.Unmarshal(result, &getResult); err != nil {
		return nil, fmt.Errorf("failed to parse prompts/get result: %w", err)
	}

	return getResult.Messages, nil
}
