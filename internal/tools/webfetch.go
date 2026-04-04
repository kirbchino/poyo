// Package tools implements the WebFetch tool for fetching web content
package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebFetchTool implements the WebFetch tool for fetching web content
type WebFetchTool struct {
	BaseTool
	httpClient *http.Client
}

// NewWebFetchTool creates a new WebFetch tool
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		BaseTool: BaseTool{
			name:              "WebFetch",
			description:       "Fetches content from a URL and returns it as text. Useful for retrieving web pages, API responses, or other online content.",
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
	}
}

// WebFetchInput represents input for the WebFetch tool
type WebFetchInput struct {
	URL         string            `json:"url"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	MaxSize     int64             `json:"max_size,omitempty"`
	FollowRedirects bool           `json:"follow_redirects,omitempty"`
}

// WebFetchOutput represents output from the WebFetch tool
type WebFetchOutput struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	Status      string            `json:"status"`
	Headers     map[string]string `json:"headers"`
	Content     string            `json:"content"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Duration    time.Duration     `json:"duration"`
}

// Call executes the WebFetch tool
func (t *WebFetchTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for WebFetch tool")
	}

	urlStr, _ := input["url"].(string)
	if urlStr == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("only http and https URLs are allowed")
	}

	method, _ := input["method"].(string)
	if method == "" {
		method = "GET"
	}

	timeout, _ := input["timeout"].(int)
	if timeout == 0 {
		timeout = 30000
	}

	maxSize, _ := input["max_size"].(int64)
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}

	headers, _ := input["headers"].(map[string]interface{})
	stringHeaders := make(map[string]string)
	for k, v := range headers {
		stringHeaders[k] = fmt.Sprintf("%v", v)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range stringHeaders {
		req.Header.Set(key, value)
	}

	// Add body if present
	if body, _ := input["body"].(string); body != "" && method != "GET" {
		req.Body = io.NopCloser(strings.NewReader(body))
	}

	// Execute request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxSize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract response headers
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	output := &WebFetchOutput{
		URL:         urlStr,
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     respHeaders,
		Content:     string(bodyBytes),
		ContentType: resp.Header.Get("Content-Type"),
		Size:        int64(len(bodyBytes)),
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// InputSchema returns the input schema for the WebFetch tool
func (t *WebFetchTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"url": {
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"method": {
				"type":        "string",
				"description": "HTTP method to use",
				"enum":        []string{"GET", "POST", "PUT", "DELETE"},
			},
			"headers": {
				"type":        "object",
				"description": "HTTP headers to send with the request",
			},
			"body": {
				"type":        "string",
				"description": "Request body for POST/PUT requests",
			},
			"timeout": {
				"type":        "integer",
				"description": "Request timeout in milliseconds",
			},
			"max_size": {
				"type":        "integer",
				"description": "Maximum response size in bytes",
			},
		},
		Required: []string{"url"},
	}
}

// WebSearch functionality moved to websearch.go
