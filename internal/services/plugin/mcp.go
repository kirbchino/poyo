// Package plugin provides MCP (Model Context Protocol) support
package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// MCPMessageType represents the type of MCP message
type MCPMessageType string

const (
	MCPMessageTypeRequest  MCPMessageType = "request"
	MCPMessageTypeResponse MCPMessageType = "response"
	MCPMessageTypeNotification MCPMessageType = "notification"
)

// MCPRequest represents an MCP request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPNotification represents an MCP notification
type MCPNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPClient represents an MCP client connection
type MCPClient struct {
	mu       sync.RWMutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.Reader
	stderr   io.Reader
	pending  map[interface{}]chan *MCPResponse
	handlers map[string]NotificationHandler
	closed   bool
}

// NotificationHandler handles MCP notifications
type NotificationHandler func(notification *MCPNotification)

// MCPPluginHandler handles MCP-based plugins
type MCPPluginHandler struct {
	mu           sync.RWMutex
	clients      map[string]*MCPClient // Plugin ID -> Client
	workingDir   string
	capabilities map[string]MCPCapabilities // Plugin ID -> Capabilities
}

// MCPCapabilities represents the capabilities of an MCP server
type MCPCapabilities struct {
	Tools     *MCPToolsCapabilities     `json:"tools,omitempty"`
	Resources *MCPResourcesCapabilities `json:"resources,omitempty"`
	Prompts   *MCPPromptsCapabilities   `json:"prompts,omitempty"`
	Sampling  *MCPSamplingCapabilities  `json:"sampling,omitempty"`
	Logging   *MCPLoggingCapabilities   `json:"logging,omitempty"`
}

// MCPToolsCapabilities represents tools capabilities
type MCPToolsCapabilities struct {
	Supported bool `json:"supported"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPResourcesCapabilities represents resources capabilities
type MCPResourcesCapabilities struct {
	Supported bool `json:"supported"`
	Subscribe bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPPromptsCapabilities represents prompts capabilities
type MCPPromptsCapabilities struct {
	Supported bool `json:"supported"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPSamplingCapabilities represents sampling capabilities
type MCPSamplingCapabilities struct {
	Supported bool `json:"supported"`
}

// MCPLoggingCapabilities represents logging capabilities
type MCPLoggingCapabilities struct {
	Supported bool `json:"supported"`
}

// MCPResource represents an MCP resource
type MCPResource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MCPResourceContent represents resource content
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// MCPPrompt represents an MCP prompt template
type MCPPrompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []MCPPromptArgument    `json:"arguments,omitempty"`
}

// MCPPromptArgument represents a prompt argument
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// MCPPromptMessage represents a message in a prompt
type MCPPromptMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// MCPSamplingMessage represents a message for sampling
type MCPSamplingMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// MCPSamplingParams represents parameters for sampling
type MCPSamplingParams struct {
	Messages         []MCPSamplingMessage `json:"messages"`
	ModelPreferences *MCPModelPreferences `json:"modelPreferences,omitempty"`
	SystemPrompt     string               `json:"systemPrompt,omitempty"`
	IncludeContext   string               `json:"includeContext,omitempty"`
	Temperature      float64              `json:"temperature,omitempty"`
	MaxTokens        int                  `json:"maxTokens"`
	StopSequences    []string             `json:"stopSequences,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// MCPModelPreferences represents model preferences
type MCPModelPreferences struct {
	Hints            []MCPModelHint `json:"hints,omitempty"`
	CostPriority     float64        `json:"costPriority,omitempty"`
	SpeedPriority    float64        `json:"speedPriority,omitempty"`
	IntelligencePriority float64     `json:"intelligencePriority,omitempty"`
}

// MCPModelHint represents a model hint
type MCPModelHint struct {
	Name string `json:"name,omitempty"`
}

// NewMCPPluginHandler creates a new MCP plugin handler
func NewMCPPluginHandler(workingDir string) *MCPPluginHandler {
	return &MCPPluginHandler{
		clients:      make(map[string]*MCPClient),
		workingDir:   workingDir,
		capabilities: make(map[string]MCPCapabilities),
	}
}

// Load loads an MCP plugin
func (h *MCPPluginHandler) Load(ctx context.Context, plugin *Plugin) error {
	if plugin.Main == "" {
		return fmt.Errorf("no main command specified for MCP plugin")
	}

	// Create command
	cmd := exec.CommandContext(ctx, plugin.Main)
	cmd.Dir = plugin.Path
	cmd.Env = append(os.Environ(), "MCP_MODE=1")

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start MCP process: %w", err)
	}

	// Create client
	client := &MCPClient{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		pending:  make(map[interface{}]chan *MCPResponse),
		handlers: make(map[string]NotificationHandler),
	}

	// Start reading responses
	go h.readResponses(client)

	// Store client
	h.mu.Lock()
	h.clients[plugin.ID] = client
	h.mu.Unlock()

	// Initialize MCP connection with full capabilities
	result, err := h.Call(ctx, plugin.ID, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"supported": true,
			},
			"resources": map[string]interface{}{
				"supported":   true,
				"subscribe":   true,
				"listChanged": true,
			},
			"prompts": map[string]interface{}{
				"supported": true,
			},
			"sampling": map[string]interface{}{
				"supported": true,
			},
			"logging": map[string]interface{}{
				"supported": true,
			},
		},
		"clientInfo": map[string]interface{}{
			"name":    "poyo",
			"version": "1.0.0",
		},
	})
	if err != nil {
		h.Unload(ctx, plugin)
		return fmt.Errorf("initialize MCP: %w", err)
	}

	// Parse server capabilities
	if resultMap, ok := result.(map[string]interface{}); ok {
		caps := MCPCapabilities{}
		if capsMap, ok := resultMap["capabilities"].(map[string]interface{}); ok {
			if tools, ok := capsMap["tools"].(map[string]interface{}); ok {
				caps.Tools = &MCPToolsCapabilities{
					Supported:   true,
					ListChanged: tools["listChanged"].(bool),
				}
			}
			if resources, ok := capsMap["resources"].(map[string]interface{}); ok {
				caps.Resources = &MCPResourcesCapabilities{
					Supported:   true,
					Subscribe:   resources["subscribe"].(bool),
					ListChanged: resources["listChanged"].(bool),
				}
			}
			if prompts, ok := capsMap["prompts"].(map[string]interface{}); ok {
				caps.Prompts = &MCPPromptsCapabilities{
					Supported:   true,
					ListChanged: prompts["listChanged"].(bool),
				}
			}
			if _, ok := capsMap["sampling"].(map[string]interface{}); ok {
				caps.Sampling = &MCPSamplingCapabilities{Supported: true}
			}
			if _, ok := capsMap["logging"].(map[string]interface{}); ok {
				caps.Logging = &MCPLoggingCapabilities{Supported: true}
			}
		}
		h.mu.Lock()
		h.capabilities[plugin.ID] = caps
		h.mu.Unlock()
	}

	// Send initialized notification
	h.sendNotification(client, &MCPNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})

	return nil
}

// Unload unloads an MCP plugin
func (h *MCPPluginHandler) Unload(ctx context.Context, plugin *Plugin) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[plugin.ID]
	if !ok {
		return nil
	}

	// Send shutdown notification
	notification := &MCPNotification{
		JSONRPC: "2.0",
		Method:  "shutdown",
	}
	h.sendNotification(client, notification)

	// Close stdin
	client.stdin.Close()

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- client.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		client.cmd.Process.Kill()
	case <-done:
	}

	delete(h.clients, plugin.ID)
	return nil
}

// Execute executes an MCP plugin method
func (h *MCPPluginHandler) Execute(ctx context.Context, plugin *Plugin, method string, input interface{}) (interface{}, error) {
	return h.Call(ctx, plugin.ID, method, input)
}

// Call calls an MCP method on a plugin
func (h *MCPPluginHandler) Call(ctx context.Context, pluginID string, method string, params interface{}) (interface{}, error) {
	h.mu.RLock()
	client, ok := h.clients[pluginID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %s not connected", pluginID)
	}

	// Create request
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	paramsJSON, _ := json.Marshal(params)

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	// Create response channel
	responseChan := make(chan *MCPResponse, 1)
	client.mu.Lock()
	client.pending[id] = responseChan
	client.mu.Unlock()

	defer func() {
		client.mu.Lock()
		delete(client.pending, id)
		client.mu.Unlock()
	}()

	// Send request
	if err := h.sendRequest(client, request); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", response.Error.Code, response.Error.Message)
		}
		return response.Result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("MCP call timeout")
	}
}

// sendRequest sends an MCP request
func (h *MCPPluginHandler) sendRequest(client *MCPClient, request *MCPRequest) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	// MCP uses newline-delimited JSON
	data = append(data, '\n')
	_, err = client.stdin.Write(data)
	return err
}

// sendNotification sends an MCP notification
func (h *MCPPluginHandler) sendNotification(client *MCPClient, notification *MCPNotification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = client.stdin.Write(data)
	return err
}

// readResponses reads responses from an MCP client
func (h *MCPPluginHandler) readResponses(client *MCPClient) {
	scanner := bufio.NewScanner(client.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Try to parse as response
		var response MCPResponse
		if err := json.Unmarshal(line, &response); err == nil && response.ID != nil {
			client.mu.Lock()
			if ch, ok := client.pending[response.ID]; ok {
				ch <- &response
			}
			client.mu.Unlock()
			continue
		}

		// Try to parse as notification
		var notification MCPNotification
		if err := json.Unmarshal(line, &notification); err == nil && notification.Method != "" {
			client.mu.RLock()
			if handler, ok := client.handlers[notification.Method]; ok {
				handler(&notification)
			}
			client.mu.RUnlock()
		}
	}
}

// OnNotification registers a notification handler
func (h *MCPPluginHandler) OnNotification(pluginID string, method string, handler NotificationHandler) error {
	h.mu.RLock()
	client, ok := h.clients[pluginID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not connected", pluginID)
	}

	client.mu.Lock()
	client.handlers[method] = handler
	client.mu.Unlock()

	return nil
}

// ListTools lists tools provided by an MCP plugin
func (h *MCPPluginHandler) ListTools(ctx context.Context, pluginID string) ([]ToolDefinition, error) {
	result, err := h.Call(ctx, pluginID, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var tools []ToolDefinition
	if resultMap, ok := result.(map[string]interface{}); ok {
		if toolsList, ok := resultMap["tools"].([]interface{}); ok {
			for _, t := range toolsList {
				if toolMap, ok := t.(map[string]interface{}); ok {
					tool := ToolDefinition{}
					if name, ok := toolMap["name"].(string); ok {
						tool.Name = name
					}
					if desc, ok := toolMap["description"].(string); ok {
						tool.Description = desc
					}
					tool.InputSchema = toolMap["inputSchema"]
					tools = append(tools, tool)
				}
			}
		}
	}

	return tools, nil
}

// CallTool calls a tool on an MCP plugin
func (h *MCPPluginHandler) CallTool(ctx context.Context, pluginID string, toolName string, args interface{}) (interface{}, error) {
	return h.Call(ctx, pluginID, "tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	})
}

// ============== Resources API ==============

// ListResources lists resources provided by an MCP plugin
func (h *MCPPluginHandler) ListResources(ctx context.Context, pluginID string) ([]MCPResource, error) {
	result, err := h.Call(ctx, pluginID, "resources/list", nil)
	if err != nil {
		return nil, err
	}

	var resources []MCPResource
	if resultMap, ok := result.(map[string]interface{}); ok {
		if resourcesList, ok := resultMap["resources"].([]interface{}); ok {
			for _, r := range resourcesList {
				if resourceMap, ok := r.(map[string]interface{}); ok {
					resource := MCPResource{
						URI:         resourceMap["uri"].(string),
						Name:        resourceMap["name"].(string),
						Description: getString(resourceMap, "description"),
						MimeType:    getString(resourceMap, "mimeType"),
					}
					if meta, ok := resourceMap["metadata"].(map[string]interface{}); ok {
						resource.Metadata = meta
					}
					resources = append(resources, resource)
				}
			}
		}
	}

	return resources, nil
}

// ReadResource reads a resource from an MCP plugin
func (h *MCPPluginHandler) ReadResource(ctx context.Context, pluginID string, uri string) ([]MCPResourceContent, error) {
	result, err := h.Call(ctx, pluginID, "resources/read", map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return nil, err
	}

	var contents []MCPResourceContent
	if resultMap, ok := result.(map[string]interface{}); ok {
		if contentList, ok := resultMap["contents"].([]interface{}); ok {
			for _, c := range contentList {
				if contentMap, ok := c.(map[string]interface{}); ok {
					content := MCPResourceContent{
						URI:      contentMap["uri"].(string),
						MimeType: getString(contentMap, "mimeType"),
						Text:     getString(contentMap, "text"),
					}
					if blob, ok := contentMap["blob"].([]byte); ok {
						content.Blob = blob
					}
					contents = append(contents, content)
				}
			}
		}
	}

	return contents, nil
}

// SubscribeResource subscribes to resource updates
func (h *MCPPluginHandler) SubscribeResource(ctx context.Context, pluginID string, uri string) error {
	_, err := h.Call(ctx, pluginID, "resources/subscribe", map[string]interface{}{
		"uri": uri,
	})
	return err
}

// UnsubscribeResource unsubscribes from resource updates
func (h *MCPPluginHandler) UnsubscribeResource(ctx context.Context, pluginID string, uri string) error {
	_, err := h.Call(ctx, pluginID, "resources/unsubscribe", map[string]interface{}{
		"uri": uri,
	})
	return err
}

// ============== Prompts API ==============

// ListPrompts lists prompts provided by an MCP plugin
func (h *MCPPluginHandler) ListPrompts(ctx context.Context, pluginID string) ([]MCPPrompt, error) {
	result, err := h.Call(ctx, pluginID, "prompts/list", nil)
	if err != nil {
		return nil, err
	}

	var prompts []MCPPrompt
	if resultMap, ok := result.(map[string]interface{}); ok {
		if promptsList, ok := resultMap["prompts"].([]interface{}); ok {
			for _, p := range promptsList {
				if promptMap, ok := p.(map[string]interface{}); ok {
					prompt := MCPPrompt{
						Name:        promptMap["name"].(string),
						Description: getString(promptMap, "description"),
					}
					if args, ok := promptMap["arguments"].([]interface{}); ok {
						for _, a := range args {
							if argMap, ok := a.(map[string]interface{}); ok {
								prompt.Arguments = append(prompt.Arguments, MCPPromptArgument{
									Name:        argMap["name"].(string),
									Description: getString(argMap, "description"),
									Required:    getBool(argMap, "required"),
								})
							}
						}
					}
					prompts = append(prompts, prompt)
				}
			}
		}
	}

	return prompts, nil
}

// GetPrompt gets a prompt from an MCP plugin
func (h *MCPPluginHandler) GetPrompt(ctx context.Context, pluginID string, name string, args map[string]string) ([]MCPPromptMessage, error) {
	result, err := h.Call(ctx, pluginID, "prompts/get", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}

	var messages []MCPPromptMessage
	if resultMap, ok := result.(map[string]interface{}); ok {
		if messagesList, ok := resultMap["messages"].([]interface{}); ok {
			for _, m := range messagesList {
				if msgMap, ok := m.(map[string]interface{}); ok {
					message := MCPPromptMessage{
						Role:    msgMap["role"].(string),
						Content: msgMap["content"],
					}
					messages = append(messages, message)
				}
			}
		}
	}

	return messages, nil
}

// ============== Sampling API ==============

// Sample requests a sample from the LLM via MCP
func (h *MCPPluginHandler) Sample(ctx context.Context, pluginID string, params MCPSamplingParams) (interface{}, error) {
	return h.Call(ctx, pluginID, "sampling/createMessage", params)
}

// ============== Logging API ==============

// SetLogLevel sets the log level for an MCP plugin
func (h *MCPPluginHandler) SetLogLevel(ctx context.Context, pluginID string, level string) error {
	_, err := h.Call(ctx, pluginID, "logging/setLevel", map[string]interface{}{
		"level": level,
	})
	return err
}

// ============== Roots API ==============

// ListRoots lists the roots for an MCP plugin
func (h *MCPPluginHandler) ListRoots(ctx context.Context, pluginID string) ([]string, error) {
	result, err := h.Call(ctx, pluginID, "roots/list", nil)
	if err != nil {
		return nil, err
	}

	var roots []string
	if resultMap, ok := result.(map[string]interface{}); ok {
		if rootsList, ok := resultMap["roots"].([]interface{}); ok {
			for _, r := range rootsList {
				if rootMap, ok := r.(map[string]interface{}); ok {
					if uri, ok := rootMap["uri"].(string); ok {
						roots = append(roots, uri)
					}
				}
			}
		}
	}

	return roots, nil
}

// ============== Capabilities API ==============

// GetCapabilities returns the capabilities of an MCP plugin
func (h *MCPPluginHandler) GetCapabilities(pluginID string) *MCPCapabilities {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if caps, ok := h.capabilities[pluginID]; ok {
		return &caps
	}
	return nil
}

// HasTools checks if an MCP plugin supports tools
func (h *MCPPluginHandler) HasTools(pluginID string) bool {
	caps := h.GetCapabilities(pluginID)
	return caps != nil && caps.Tools != nil && caps.Tools.Supported
}

// HasResources checks if an MCP plugin supports resources
func (h *MCPPluginHandler) HasResources(pluginID string) bool {
	caps := h.GetCapabilities(pluginID)
	return caps != nil && caps.Resources != nil && caps.Resources.Supported
}

// HasPrompts checks if an MCP plugin supports prompts
func (h *MCPPluginHandler) HasPrompts(pluginID string) bool {
	caps := h.GetCapabilities(pluginID)
	return caps != nil && caps.Prompts != nil && caps.Prompts.Supported
}

// HasSampling checks if an MCP plugin supports sampling
func (h *MCPPluginHandler) HasSampling(pluginID string) bool {
	caps := h.GetCapabilities(pluginID)
	return caps != nil && caps.Sampling != nil && caps.Sampling.Supported
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
