// Package tools implements WebSearch with real API integration
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/kirbchino/poyo/internal/prompt"
)

// WebSearchTool implements web search functionality
type WebSearchTool struct {
	BaseTool
	httpClient *http.Client
	apiKey     string
	apiType    string // "brave", "google", "custom"
}

// NewWebSearchTool creates a new WebSearch tool
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		BaseTool: BaseTool{
			name:              "WebSearch",
			description:       prompt.GetToolDescription("WebSearch"),
			isConcurrencySafe: true,
			isEnabled:         true,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiType: "brave", // Default to Brave Search
	}
}

// SetAPIKey sets the API key for the search service
func (t *WebSearchTool) SetAPIKey(apiKey string) {
	t.apiKey = apiKey
}

// SetAPIType sets the search API type
func (t *WebSearchTool) SetAPIType(apiType string) {
	t.apiType = apiType
}

// WebSearchInput represents input for the WebSearch tool
type WebSearchInput struct {
	Query  string `json:"query"`
	Count  int    `json:"count,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Fresh  string `json:"fresh,omitempty"` // "day", "week", "month", "year"
}

// WebSearchOutput represents output from the WebSearch tool
type WebSearchOutput struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Source  string         `json:"source"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Published   string `json:"published,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
}

// Call executes the WebSearch tool
func (t *WebSearchTool) Call(ctx context.Context, input map[string]interface{}, toolCtx *ToolUseContext, _ CanUseToolFunc, _ ToolCallProgress) (*ToolResult, error) {
	if input == nil {
		return nil, fmt.Errorf("invalid input type for WebSearch tool")
	}

	query, _ := input["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	count, _ := input["count"].(int)
	if count == 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	fresh, _ := input["fresh"].(string)

	// Try to get API key from environment if not set
	if t.apiKey == "" {
		t.apiKey = os.Getenv("BRAVE_API_KEY")
		if t.apiKey == "" {
			t.apiKey = os.Getenv("SEARCH_API_KEY")
		}
	}

	var output *WebSearchOutput
	var err error

	// Use appropriate API based on type and available key
	if t.apiKey != "" {
		switch t.apiType {
		case "brave":
			output, err = t.searchBrave(ctx, query, count, fresh)
		case "google":
			output, err = t.searchGoogle(ctx, query, count)
		case "custom":
			output, err = t.searchCustom(ctx, query, count)
		default:
			output, err = t.searchBrave(ctx, query, count, fresh)
		}
	} else {
		// Fallback to simulated results when no API key
		output = t.simulatedSearch(query)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return &ToolResult{
		Data: output,
	}, nil
}

// searchBrave uses Brave Search API
func (t *WebSearchTool) searchBrave(ctx context.Context, query string, count int, fresh string) (*WebSearchOutput, error) {
	apiURL := "https://api.search.brave.com/res/v1/web/search"

	params := url.Values{}
	params.Set("q", query)
	params.Set("count", fmt.Sprintf("%d", count))
	if fresh != "" {
		params.Set("freshness", fresh)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse Brave Search response
	var braveResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&braveResp); err != nil {
		return nil, err
	}

	output := &WebSearchOutput{
		Query:  query,
		Source: "brave",
	}

	for _, r := range braveResp.Web.Results {
		output.Results = append(output.Results, SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Description,
		})
	}
	output.Total = len(output.Results)

	return output, nil
}

// searchGoogle uses Google Custom Search API
func (t *WebSearchTool) searchGoogle(ctx context.Context, query string, count int) (*WebSearchOutput, error) {
	// Google Custom Search API implementation
	// Requires GOOGLE_API_KEY and GOOGLE_CX environment variables
	cx := os.Getenv("GOOGLE_CX")
	if cx == "" {
		return nil, fmt.Errorf("GOOGLE_CX not set for Google Custom Search")
	}

	apiURL := "https://www.googleapis.com/customsearch/v1"

	params := url.Values{}
	params.Set("q", query)
	params.Set("key", t.apiKey)
	params.Set("cx", cx)
	params.Set("num", fmt.Sprintf("%d", count))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API error %d: %s", resp.StatusCode, string(body))
	}

	var googleResp struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, err
	}

	output := &WebSearchOutput{
		Query:  query,
		Source: "google",
	}

	for _, item := range googleResp.Items {
		output.Results = append(output.Results, SearchResult{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Snippet,
		})
	}
	output.Total = len(output.Results)

	return output, nil
}

// searchCustom uses Custom Search API
func (t *WebSearchTool) searchCustom(ctx context.Context, query string, count int) (*WebSearchOutput, error) {
	// Custom Search API implementation
	// Uses custom search API if CUSTOM_SEARCH_API_KEY is configured,
	// otherwise falls back to web scraping approach
	apiKey := os.Getenv("CUSTOM_SEARCH_API_KEY")

	if apiKey != "" {
		return t.searchCustomAPI(ctx, query, count, apiKey)
	}

	// Fallback: Use web scraping approach for custom search
	return t.searchCustomScrape(ctx, query, count)
}

// searchCustomAPI uses official search API
func (t *WebSearchTool) searchCustomAPI(ctx context.Context, query string, count int, apiKey string) (*WebSearchOutput, error) {
	// Custom Search API endpoint
	// Note: This is a placeholder for custom search API
	// The real endpoint would be configured via environment variable

	endpoint := os.Getenv("CUSTOM_SEARCH_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.example.com/search"
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("pn", "0")  // Page number
	params.Set("rn", fmt.Sprintf("%d", count))

	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var customResp struct {
		ErrNo  int `json:"errno"`
		Data   []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Summary string `json:"summary"`
		} `json:"data"`
		Total int `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&customResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if customResp.ErrNo != 0 {
		return nil, fmt.Errorf("API returned error code: %d", customResp.ErrNo)
	}

	output := &WebSearchOutput{
		Query:  query,
		Source: "custom",
	}

	for _, item := range customResp.Data {
		output.Results = append(output.Results, SearchResult{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Summary,
		})
	}
	output.Total = len(output.Results)

	return output, nil
}

// searchCustomScrape uses web scraping to get search results
func (t *WebSearchTool) searchCustomScrape(ctx context.Context, query string, count int) (*WebSearchOutput, error) {
	// Use search page to get results
	params := url.Values{}
	params.Set("q", query)
	params.Set("num", fmt.Sprintf("%d", count))

	reqURL := "https://www.example.com/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to mimic browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Parse HTML to extract search results
	return t.parseCustomResults(query, string(body), count)
}

// parseCustomResults parses search result HTML
func (t *WebSearchTool) parseCustomResults(query, html string, maxResults int) (*WebSearchOutput, error) {
	output := &WebSearchOutput{
		Query:  query,
		Source: "custom",
	}

	// Simple regex-based parsing for search results
	// Result containers typically have class "result"
	// This is a simplified parser - in production would use proper HTML parsing

	titlePattern := regexp.MustCompile(`<h3[^>]*class="[^"]*t[^"]*"[^>]*>.*?<a[^>]*href="([^"]*)"[^>]*>([^<]+)</a>`)
	descPattern := regexp.MustCompile(`<span[^>]*class="[^"]*content-right_[^"]*"[^>]*>([^<]+)</span>`)

	// Find all title/URL matches
	titleMatches := titlePattern.FindAllStringSubmatch(html, -1)
	descMatches := descPattern.FindAllStringSubmatch(html, -1)

	for i, match := range titleMatches {
		if i >= maxResults {
			break
		}

		if len(match) >= 3 {
			result := SearchResult{
				URL:   match[1],
				Title: stripTags(match[2]),
			}

			// Get description if available
			if i < len(descMatches) && len(descMatches[i]) >= 2 {
				result.Description = stripTags(descMatches[i][1])
			}

			output.Results = append(output.Results, result)
		}
	}

	output.Total = len(output.Results)

	// If no results found via parsing, return a helpful message
	if output.Total == 0 {
		output.Results = append(output.Results, SearchResult{
			Title:       "网络搜索",
			URL:         fmt.Sprintf("https://www.example.com/search?q=%s", url.QueryEscape(query)),
			Description: fmt.Sprintf("请访问搜索引擎查看结果: %s", query),
		})
		output.Total = 1
	}

	return output, nil
}

// stripTags removes HTML tags from a string
func stripTags(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Decode HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}

// simulatedSearch returns simulated results when no API key is available
func (t *WebSearchTool) simulatedSearch(query string) *WebSearchOutput {
	return &WebSearchOutput{
		Query:  query,
		Source: "simulated",
		Results: []SearchResult{
			{
				Title:       "Search API Not Configured",
				URL:         "https://example.com",
				Description: fmt.Sprintf("🔎 要启用真实搜索，请设置 BRAVE_API_KEY 环境变量。模拟搜索: %s", query),
			},
		},
		Total: 1,
	}
}

// InputSchema returns the input schema for the WebSearch tool
func (t *WebSearchTool) InputSchema() ToolInputJSONSchema {
	return ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]map[string]interface{}{
			"query": {
				"type":        "string",
				"description": "The search query (搜索查询)",
			},
			"count": {
				"type":        "integer",
				"description": "Number of results to return (default: 10, max: 50)",
			},
			"fresh": {
				"type":        "string",
				"description": "Filter by freshness (时效性过滤)",
				"enum":        []string{"day", "week", "month", "year"},
			},
		},
		Required: []string{"query"},
	}
}

// WebFetchEnhancedTool implements enhanced web fetching with better content extraction
type WebFetchEnhancedTool struct {
	*WebFetchTool
}

// NewWebFetchEnhancedTool creates an enhanced WebFetch tool
func NewWebFetchEnhancedTool() *WebFetchEnhancedTool {
	return &WebFetchEnhancedTool{
		WebFetchTool: NewWebFetchTool(),
	}
}

// ExtractContent extracts main content from a webpage
func (t *WebFetchEnhancedTool) ExtractContent(html string) string {
	// Simple content extraction - in real implementation would use readability algorithm
	// This is a placeholder for content extraction logic
	return html
}

// FetchAndSummarize fetches a URL and provides a summary
func (t *WebFetchEnhancedTool) FetchAndSummarize(ctx context.Context, url string) (string, error) {
	result, err := t.Call(ctx, map[string]interface{}{
		"url": url,
	}, nil, nil, nil)
	if err != nil {
		return "", err
	}

	// Return content for summarization
	if data, ok := result.Data.(*WebFetchOutput); ok {
		return data.Content, nil
	}

	return "", fmt.Errorf("unexpected result type")
}
