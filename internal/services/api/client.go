// Package api provides HTTP client for Claude API communication
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client is the API client for Claude API
type Client struct {
	config     ClientConfig
	httpClient *http.Client
	retry      *RetryPolicy
}

// ClientConfig holds the client configuration
type ClientConfig struct {
	APIKey           string
	BaseURL          string
	DefaultModel     string
	DefaultMaxTokens int
	Timeout          time.Duration
	// Custom headers for specific API providers
	CustomHeaders map[string]string
	// API type: "anthropic" or "openai"
	APIType string
}

// NewClient creates a new API client
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	if config.DefaultModel == "" {
		config.DefaultModel = "claude-sonnet-4-6"
	}
	if config.DefaultMaxTokens == 0 {
		config.DefaultMaxTokens = 4096
	}
	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		retry: NewRetryPolicy(DefaultRetryConfig()),
	}
}

// MessageParams contains parameters for creating a message
type MessageParams struct {
	Model     string         `json:"model"`
	Messages  []Message      `json:"messages"`
	MaxTokens int            `json:"max_tokens"`
	System    interface{}    `json:"system,omitempty"`
	Tools     []ToolSchema   `json:"tools,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
	Metadata  interface{}    `json:"metadata,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a block of content
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	// For tool_use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// For tool_result
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

// ToolSchema represents a tool definition
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// APIResponse represents the API response
type APIResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        TokenUsage     `json:"usage"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CreateMessage creates a message synchronously
func (c *Client) CreateMessage(ctx context.Context, params *MessageParams) (*APIResponse, error) {
	if params.Model == "" {
		params.Model = c.config.DefaultModel
	}
	if params.MaxTokens == 0 {
		params.MaxTokens = c.config.DefaultMaxTokens
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var response *APIResponse
	err = c.retry.Execute(ctx, func() error {
		resp, err := c.doRequest(ctx, "POST", "/v1/messages", body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return c.handleError(resp)
		}

		response = &APIResponse{}
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return response, nil
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	ContentBlock ContentBlock  `json:"content_block,omitempty"`
	Delta        *StreamDelta  `json:"delta,omitempty"`
	Message      *APIResponse  `json:"message,omitempty"`
	Usage        *TokenUsage   `json:"usage,omitempty"`
	Error        *APIError     `json:"error,omitempty"`
}

// StreamDelta represents a streaming delta
type StreamDelta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
}

// StreamResult contains the result of streaming
type StreamResult struct {
	Response *APIResponse
	Error    error
}

// StreamMessage creates a message with streaming
func (c *Client) StreamMessage(ctx context.Context, params *MessageParams) (<-chan StreamEvent, <-chan *StreamResult) {
	eventCh := make(chan StreamEvent, 100)
	resultCh := make(chan *StreamResult, 1)

	go func() {
		defer close(eventCh)
		defer close(resultCh)

		if params.Model == "" {
			params.Model = c.config.DefaultModel
		}
		if params.MaxTokens == 0 {
			params.MaxTokens = c.config.DefaultMaxTokens
		}
		params.Stream = true

		body, err := json.Marshal(params)
		if err != nil {
			eventCh <- StreamEvent{Type: "error", Error: &APIError{Message: err.Error()}}
			return
		}

		resp, err := c.doRequest(ctx, "POST", "/v1/messages", body)
		if err != nil {
			eventCh <- StreamEvent{Type: "error", Error: &APIError{Message: err.Error()}}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			apiErr := c.handleError(resp)
			eventCh <- StreamEvent{Type: "error", Error: &APIError{Message: apiErr.Error()}}
			return
		}

		// Parse SSE stream
		decoder := NewSSEDecoder(resp.Body)
		var finalResponse *APIResponse

		for {
			event, err := decoder.Decode()
			if err != nil {
				if err == io.EOF {
					break
				}
				eventCh <- StreamEvent{Type: "error", Error: &APIError{Message: err.Error()}}
				return
			}

			if event == nil {
				continue
			}

			streamEvent := c.parseSSEEvent(event)
			if streamEvent != nil {
				eventCh <- *streamEvent

				if streamEvent.Type == "message_stop" {
					break
				}
				if streamEvent.Message != nil {
					finalResponse = streamEvent.Message
				}
			}
		}

		resultCh <- &StreamResult{Response: finalResponse}
	}()

	return eventCh, resultCh
}

// doRequest performs an HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	url := c.config.BaseURL + path

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Set authentication based on API type
	if c.config.APIType == "openai" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	} else {
		req.Header.Set("x-api-key", c.config.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	// Add custom headers
	for key, value := range c.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// handleError handles API error responses
func (c *Client) handleError(resp *http.Response) *APIError {
	var apiErr APIError
	body, _ := io.ReadAll(resp.Body)

	if len(body) > 0 {
		if err := json.Unmarshal(body, &apiErr); err != nil {
			apiErr = APIError{
				Type:    "api_error",
				Message: string(body),
			}
		}
	} else {
		apiErr = APIError{
			Type:    "http_error",
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
		}
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}

// parseSSEEvent parses a SSE event
func (c *Client) parseSSEEvent(event *SSEEvent) *StreamEvent {
	if event == nil || event.Data == "" {
		return nil
	}

	if event.Data == "[DONE]" {
		return nil
	}

	var streamEvent StreamEvent
	if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
		return nil
	}

	return &streamEvent
}

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// SSEDecoder decodes server-sent events
type SSEDecoder struct {
	reader *bufio.Reader
}

// NewSSEDecoder creates a new SSE decoder
func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{
		reader: bufio.NewReader(r),
	}
}

// Decode decodes the next SSE event
func (d *SSEDecoder) Decode() (*SSEEvent, error) {
	event := &SSEEvent{}

	for {
		line, err := d.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if event.Data != "" {
				return event, nil
			}
			continue
		}

		if line[0] == ':' {
			// Comment line, skip
			continue
		}

		colon := bytes.IndexByte(line, ':')
		if colon == -1 {
			continue
		}

		field := string(line[:colon])
		value := line[colon+1:]
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch field {
		case "event":
			event.Event = string(value)
		case "data":
			if event.Data != "" {
				event.Data += "\n"
			}
			event.Data += string(value)
		case "id":
			event.ID = string(value)
		case "retry":
			// Parse retry value
		}
	}
}

// APIError represents an API error
type APIError struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// IsRateLimit returns true if the error is a rate limit error
func (e *APIError) IsRateLimit() bool {
	return e.StatusCode == 429
}

// IsServerError returns true if the error is a server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsRetryable returns true if the error is retryable
func (e *APIError) IsRetryable() bool {
	return e.IsRateLimit() || e.IsServerError()
}

// ResponseCache provides caching for API responses
type ResponseCache struct {
	cache map[string]*cachedResponse
	mu    sync.RWMutex
	ttl   time.Duration
}

type cachedResponse struct {
	response  interface{}
	timestamp time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache(ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		cache: make(map[string]*cachedResponse),
		ttl:   ttl,
	}
}

// Get retrieves a cached response
func (c *ResponseCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache[key]; ok {
		if time.Since(cached.timestamp) < c.ttl {
			return cached.response
		}
	}
	return nil
}

// Set stores a response in the cache
func (c *ResponseCache) Set(key string, response interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cachedResponse{
		response:  response,
		timestamp: time.Now(),
	}
}

// Delete removes a cached response
func (c *ResponseCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

// Clear clears all cached responses
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cachedResponse)
}

// generateCacheKey generates a cache key from request parameters
func generateCacheKey(params interface{}) string {
	bytes, _ := json.Marshal(params)
	return fmt.Sprintf("%x", bytes)
}

// OpenAI compatible types and methods

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// OpenAIToolCall represents a tool call in OpenAI format
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall represents a function call
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIRequest represents an OpenAI API request
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []OpenAITool    `json:"tools,omitempty"`
}

// OpenAITool represents a tool in OpenAI format
type OpenAITool struct {
	Type     string          `json:"type"`
	Function OpenAIFunction  `json:"function"`
}

// OpenAIFunction represents a function definition
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenAIResponse represents an OpenAI API response
type OpenAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []OpenAIChoice   `json:"choices"`
	Usage   *OpenAIUsage     `json:"usage,omitempty"`
}

// OpenAIChoice represents a choice in OpenAI response
type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      *OpenAIMessage `json:"message,omitempty"`
	Delta        *OpenAIMessage `json:"delta,omitempty"`
	FinishReason string         `json:"finish_reason"`
}

// OpenAIUsage represents token usage in OpenAI format
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CreateChatCompletion creates a chat completion using OpenAI compatible API
func (c *Client) CreateChatCompletion(ctx context.Context, req *OpenAIRequest) (*OpenAIResponse, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = c.config.DefaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var response *OpenAIResponse
	err = c.retry.Execute(ctx, func() error {
		resp, err := c.doRequest(ctx, "POST", "/chat/completions", body)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		// Read response body for debugging
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		response = &OpenAIResponse{}
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("decode response: %w, body: %s", err, string(respBody))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return response, nil
}

// CreateChatCompletionSimple is a convenience method for simple chat requests
func (c *Client) CreateChatCompletionSimple(ctx context.Context, prompt string) (string, error) {
	req := &OpenAIRequest{
		Messages: []OpenAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := c.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return "", fmt.Errorf("no response generated")
	}

	return resp.Choices[0].Message.Content, nil
}
