// Package query contains the query engine and related functionality.
package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kirbchino/poyo/internal/services/api"
	"github.com/kirbchino/poyo/internal/tools"
	"github.com/kirbchino/poyo/internal/types"
)

// EngineConfig contains configuration for the QueryEngine.
type EngineConfig struct {
	WorkingDir     string
	Tools          []tools.Tool
	PermissionMode types.PermissionMode
	MaxTurns       int
	Model          string
	// API client configuration
	APIClient *api.Client
	// API type: "anthropic" or "openai"
	APIType string
	// Custom headers for API requests
	CustomHeaders map[string]string
	// Base URL for API
	BaseURL string
	// API key
	APIKey string
}

// QueryEngine manages the query lifecycle and session state.
type QueryEngine struct {
	config       EngineConfig
	apiClient    *api.Client
	messages     []types.Message
	abortCtx     context.Context
	abortCancel  context.CancelFunc
	usage        *types.Usage
	fileCache    *tools.FileStateCache
	permContext  *types.ToolPermissionContext
}

// NewQueryEngine creates a new QueryEngine.
func NewQueryEngine(config EngineConfig) *QueryEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// Create API client if not provided
	apiClient := config.APIClient
	if apiClient == nil {
		apiConfig := api.ClientConfig{
			BaseURL:      config.BaseURL,
			APIKey:       config.APIKey,
			DefaultModel: config.Model,
			APIType:      config.APIType,
			CustomHeaders: config.CustomHeaders,
		}
		apiClient = api.NewClient(apiConfig)
	}

	return &QueryEngine{
		config:      config,
		apiClient:   apiClient,
		messages:    []types.Message{},
		abortCtx:    ctx,
		abortCancel: cancel,
		fileCache:   tools.NewFileStateCache(),
		permContext: types.NewEmptyToolPermissionContext(),
	}
}

// QueryParams contains parameters for a query.
type QueryParams struct {
	Messages      []interface{}
	SystemPrompt  string
	UserContext   map[string]string
	SystemContext map[string]string
	MaxTurns      int
}

// QueryResult contains the result of a query.
type QueryResult struct {
	Messages   []interface{}
	Usage      *types.Usage
	Terminal   bool
	StopReason string
}

// Execute executes a query and returns the result.
func (e *QueryEngine) Execute(ctx context.Context, params QueryParams) (*QueryResult, error) {
	// Initialize state
	maxTurns := params.MaxTurns
	if maxTurns == 0 {
		maxTurns = e.config.MaxTurns
	}

	// Set default system prompt if not provided
	systemPrompt := params.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `你是 Poyo（波波），一个友好、智能的 AI 助手。你的名字来源于"Portal Of Your Orchestrated Omnibus-agents"的缩写。

你的特点：
- 热情友好，乐于助人
- 回答简洁明了，直击要点
- 擅长编程、分析和创意任务
- 偶尔会用可爱的emoji表达情感 💚

当被问及身份时，你是 Poyo，一个独立的 AI 助手。`
	}

	state := &queryState{
		messages:     params.Messages,
		systemPrompt: systemPrompt,
		turnCount:    0,
		maxTurns:     maxTurns,
	}

	// Main query loop
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return &QueryResult{
				Messages:   state.messages,
				Usage:      state.usage,
				Terminal:   true,
				StopReason: "cancelled",
			}, nil
		default:
		}

		// Check turn limit
		if state.maxTurns > 0 && state.turnCount >= state.maxTurns {
			return &QueryResult{
				Messages:   state.messages,
				Usage:      state.usage,
				Terminal:   true,
				StopReason: "max_turns",
			}, nil
		}

		// Process the next turn
		terminal, err := e.processTurn(ctx, state, params)
		if err != nil {
			return nil, fmt.Errorf("turn %d failed: %w", state.turnCount, err)
		}

		if terminal {
			return &QueryResult{
				Messages:   state.messages,
				Usage:      state.usage,
				Terminal:   true,
				StopReason: state.stopReason,
			}, nil
		}

		state.turnCount++
	}
}

// queryState tracks state across query turns.
type queryState struct {
	messages     []interface{}
	systemPrompt string
	turnCount    int
	maxTurns     int
	usage        *types.Usage
	stopReason   string
}

// processTurn processes a single turn of the query loop.
func (e *QueryEngine) processTurn(ctx context.Context, state *queryState, params QueryParams) (bool, error) {
	// Build the API request
	req := e.buildRequest(state, params)

	// Call the API (placeholder - would integrate with actual API client)
	resp, err := e.callAPI(ctx, req)
	if err != nil {
		return false, fmt.Errorf("API call failed: %w", err)
	}

	// Update usage
	if resp.Usage != nil {
		if state.usage == nil {
			state.usage = &types.Usage{}
		}
		state.usage.InputTokens += resp.Usage.InputTokens
		state.usage.OutputTokens += resp.Usage.OutputTokens
	}

	// Add assistant message
	assistantMsg := &types.AssistantMessage{
		Message: types.Message{
			UUID:      types.UUID(generateUUID()),
			Type:      types.MessageTypeAssistant,
			Timestamp: time.Now(),
		},
		Content:    resp.Content,
		Model:      resp.Model,
		StopReason: resp.StopReason,
		Usage:      resp.Usage,
	}
	state.messages = append(state.messages, assistantMsg)

	// Check for stop conditions
	if resp.StopReason == "end_turn" || resp.StopReason == "stop_sequence" {
		state.stopReason = resp.StopReason
		return true, nil
	}

	// Process tool uses
	if len(resp.ToolUses) > 0 {
		toolResults, err := e.executeTools(ctx, resp.ToolUses, state)
		if err != nil {
			return false, fmt.Errorf("tool execution failed: %w", err)
		}

		// Add tool results as user message
		userMsg := &types.UserMessage{
			Message: types.Message{
				UUID:      types.UUID(generateUUID()),
				Type:      types.MessageTypeUser,
				Timestamp: time.Now(),
			},
			Content: toolResults,
		}
		state.messages = append(state.messages, userMsg)

		// Continue for another turn
		return false, nil
	}

	return true, nil
}

// APIRequest represents a request to the API.
type APIRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	MaxTokens   int                      `json:"max_tokens"`
	System      string                   `json:"system,omitempty"`
	Tools       []ToolDef                `json:"tools,omitempty"`
}

// ToolDef represents a tool definition for the API.
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// APIResponse represents a response from the API.
type APIResponse struct {
	Content    []types.ContentBlock `json:"content"`
	Model      string               `json:"model"`
	StopReason string               `json:"stop_reason"`
	Usage      *types.Usage         `json:"usage,omitempty"`
	ToolUses   []ToolUse            `json:"tool_uses,omitempty"`
}

// ToolUse represents a tool use from the API.
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// buildRequest builds an API request from the current state.
func (e *QueryEngine) buildRequest(state *queryState, params QueryParams) *APIRequest {
	// Convert messages to API format
	apiMessages := make([]map[string]interface{}, len(state.messages))
	for i, msg := range state.messages {
		apiMessages[i] = types.NormalizeMessage(msg)
	}

	// Build tool definitions
	toolDefs := make([]ToolDef, 0, len(e.config.Tools))
	for _, tool := range e.config.Tools {
		schema := tool.InputSchema()

		// Build input schema with proper defaults
		properties := schema.Properties
		if properties == nil {
			properties = map[string]map[string]interface{}{}
		}

		// Ensure required is an empty array, not nil
		required := schema.Required
		if required == nil {
			required = []string{}
		}

		// Convert properties to map[string]interface{}
		propsIface := make(map[string]interface{})
		for k, v := range properties {
			propsIface[k] = v
		}

		inputSchema := map[string]interface{}{
			"type":       "object",
			"properties": propsIface,
			"required":   required,
		}

		toolDefs = append(toolDefs, ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: inputSchema,
		})
	}

	req := &APIRequest{
		Model:     e.config.Model,
		Messages:  apiMessages,
		MaxTokens: 4096,
		System:    state.systemPrompt,
		Tools:     toolDefs,
	}

	return req
}

// callAPI makes a call to the API.
func (e *QueryEngine) callAPI(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	// Convert to OpenAI format messages
	openAIMessages := make([]api.OpenAIMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role, _ := msg["role"].(string)
		content := extractContentString(msg["content"])
		openAIMessages = append(openAIMessages, api.OpenAIMessage{
			Role:    role,
			Content: content,
		})
	}

	// Add system prompt as first message if present
	if req.System != "" {
		systemMsg := api.OpenAIMessage{
			Role:    "system",
			Content: req.System,
		}
		openAIMessages = append([]api.OpenAIMessage{systemMsg}, openAIMessages...)
	}

	// Convert tools to OpenAI format
	openAITools := make([]api.OpenAITool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		openAITools = append(openAITools, api.OpenAITool{
			Type: "function",
			Function: api.OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Create OpenAI request
	openAIReq := &api.OpenAIRequest{
		Model:    req.Model,
		Messages: openAIMessages,
		MaxTokens: req.MaxTokens,
		Tools:    openAITools,
	}

	// Call the API
	resp, err := e.apiClient.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	// Convert response to internal format
	return e.convertOpenAIResponse(resp, req.Model), nil
}

// extractContentString extracts a string from content which may be a string or array.
func extractContentString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// Handle array of content blocks
		var result string
		for _, block := range v {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if text, ok := blockMap["text"].(string); ok {
					result += text
				}
			}
		}
		return result
	default:
		return fmt.Sprintf("%v", content)
	}
}

// convertOpenAIResponse converts OpenAI response to internal APIResponse format.
func (e *QueryEngine) convertOpenAIResponse(resp *api.OpenAIResponse, model string) *APIResponse {
	result := &APIResponse{
		Model:      model,
		StopReason: "end_turn",
		Content:    []types.ContentBlock{},
	}

	// Extract usage
	if resp.Usage != nil {
		result.Usage = &types.Usage{
			InputTokens:  int64(resp.Usage.PromptTokens),
			OutputTokens: int64(resp.Usage.CompletionTokens),
		}
	}

	// Extract content from choices
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		choice := resp.Choices[0]

		// Add text content if present
		if choice.Message.Content != "" {
			result.Content = append(result.Content, types.ContentBlock{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// Extract tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			for _, tc := range choice.Message.ToolCalls {
				// Parse arguments from JSON string
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					input = map[string]interface{}{"raw": tc.Function.Arguments}
				}

				result.ToolUses = append(result.ToolUses, ToolUse{
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			result.StopReason = "tool_use"
		}

		// Map finish reason to stop reason
		switch choice.FinishReason {
		case "stop":
			result.StopReason = "end_turn"
		case "length":
			result.StopReason = "max_tokens"
		case "tool_calls":
			result.StopReason = "tool_use"
		}
	}

	return result
}

// executeTools executes a list of tool uses.
func (e *QueryEngine) executeTools(ctx context.Context, toolUses []ToolUse, state *queryState) ([]types.ContentBlock, error) {
	var results []types.ContentBlock

	for _, toolUse := range toolUses {
		// Find the tool
		tool := tools.FindToolByName(e.config.Tools, toolUse.Name)
		if tool == nil {
			results = append(results, types.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Unknown tool: %s", toolUse.Name),
				IsError:   true,
			})
			continue
		}

		// Create tool context
		toolCtx := &tools.ToolUseContext{
			Options: tools.ToolUseOptions{
				Tools:         e.config.Tools,
				MainLoopModel: e.config.Model,
			},
			PermissionContext: e.permContext,
			Messages:          state.messages,
		}

		// Execute the tool
		result, err := tool.Call(ctx, toolUse.Input, toolCtx, nil, nil)
		if err != nil {
			results = append(results, types.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   fmt.Sprintf("Error: %v", err),
				IsError:   true,
			})
			continue
		}

		// Convert result.Data to string
		var contentStr string
		switch v := result.Data.(type) {
		case string:
			contentStr = v
		case fmt.Stringer:
			contentStr = v.String()
		default:
			// Try to marshal as JSON
			bytes, err := json.Marshal(result.Data)
			if err != nil {
				contentStr = fmt.Sprintf("%v", result.Data)
			} else {
				contentStr = string(bytes)
			}
		}

		// Add result
		results = append(results, types.ContentBlock{
			Type:      "tool_result",
			ToolUseID: toolUse.ID,
			Content:   contentStr,
		})
	}

	return results, nil
}

// generateUUID generates a new UUID.
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// AddMessage adds a message to the engine's message history.
func (e *QueryEngine) AddMessage(msg types.Message) {
	e.messages = append(e.messages, msg)
}

// GetMessages returns the current message history.
func (e *QueryEngine) GetMessages() []types.Message {
	return e.messages
}

// Stop stops the query engine.
func (e *QueryEngine) Stop() {
	e.abortCancel()
}

// SetPermissionMode sets the permission mode.
func (e *QueryEngine) SetPermissionMode(mode types.PermissionMode) {
	e.permContext.Mode = mode
}
