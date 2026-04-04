// Package tokenizer provides token counting functionality for LLM inputs
package tokenizer

import (
	"unicode"
	"unicode/utf8"
)

// Tokenizer provides token counting for text
type Tokenizer struct {
	// Model type determines tokenization algorithm
	model string
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	CacheReadTokens       int `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens   int `json:"cache_creation_tokens,omitempty"`
	TotalTokens           int `json:"total_tokens"`
}

// ModelUsage tracks usage by model
type ModelUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_input_tokens,omitempty"`
	WebSearchRequests   int `json:"web_search_requests,omitempty"`
	CostUSD             float64 `json:"cost_usd"`
	ContextWindow       int `json:"context_window,omitempty"`
	MaxOutputTokens     int `json:"max_output_tokens,omitempty"`
}

// NewTokenizer creates a new tokenizer for the specified model
func NewTokenizer(model string) *Tokenizer {
	return &Tokenizer{model: model}
}

// CountTokens estimates the number of tokens in a text
// This uses a simple heuristic that approximates GPT/Claude tokenization:
// - Average English text: ~4 characters per token
// - Average Chinese text: ~2 characters per token
// - Code: ~3-4 characters per token
// For accurate counting, use tiktoken or similar library
func (t *Tokenizer) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count characters and estimate tokens
	totalChars := utf8.RuneCountInString(text)

	// Count Chinese characters (they typically use more tokens)
	chineseChars := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseChars++
		}
	}

	nonChineseChars := totalChars - chineseChars

	// Estimate tokens:
	// Chinese: ~1.5 characters per token
	// Non-Chinese: ~4 characters per token
	// Add overhead for whitespace and special characters
	chineseTokens := (chineseChars * 2) / 3 // ~1.5 chars per token
	nonChineseTokens := nonChineseChars / 4

	// Add small overhead for message structure
	overhead := 0
	if totalChars > 0 {
		overhead = 3 // Base overhead for message formatting
	}

	return chineseTokens + nonChineseTokens + overhead
}

// CountMessagesTokens counts tokens in a list of messages
func (t *Tokenizer) CountMessagesTokens(messages []Message) int {
	total := 0

	// Add base tokens for conversation structure
	total += 3 // Every reply is primed with <|start|>assistant

	for _, msg := range messages {
		// Add tokens for role
		total += 4 // <|start|>{role}\n

		// Add tokens for content
		total += t.CountTokens(msg.Content)

		// Add tokens for tool calls if present
		for _, tc := range msg.ToolCalls {
			total += 10 // Tool call overhead
			total += t.CountTokens(tc.Name)
			for k, v := range tc.Input {
				total += t.CountTokens(k)
				total += t.CountTokens(fmtValue(v))
			}
		}
	}

	return total
}

// Message represents a chat message
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// fmtValue formats a value for token counting
func fmtValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64:
		return ""
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []interface{}:
		result := ""
		for _, item := range val {
			result += fmtValue(item)
		}
		return result
	case map[string]interface{}:
		result := ""
		for k, v := range val {
			result += k + fmtValue(v)
		}
		return result
	default:
		return ""
	}
}

// CountSystemPromptTokens counts tokens in a system prompt
func (t *Tokenizer) CountSystemPromptTokens(prompt string) int {
	// System prompts have some overhead
	return t.CountTokens(prompt) + 4
}

// EstimateMaxTokens estimates max output tokens based on context window and input
func (t *Tokenizer) EstimateMaxTokens(contextWindow, inputTokens int) int {
	// Reserve space for input tokens and some buffer
	reserved := inputTokens + 500
	available := contextWindow - reserved

	// Model-specific max output limits
	maxOutput := 4096 // Default max output

	// Check for known model limits
	if contains(t.model, "opus") || contains(t.model, "sonnet-4") {
		maxOutput = 16384
	} else if contains(t.model, "sonnet") {
		maxOutput = 8192
	} else if contains(t.model, "haiku") {
		maxOutput = 4096
	}

	if available < maxOutput {
		return available
	}
	return maxOutput
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetContextWindow returns the context window for a model
func GetContextWindow(model string) int {
	// Claude models
	if contains(model, "claude-opus-4") || contains(model, "claude-sonnet-4") {
		return 200000
	}
	if contains(model, "claude-3-5") || contains(model, "claude-3.5") {
		return 200000
	}
	if contains(model, "claude-3") {
		return 200000
	}

	// GPT-4 models
	if contains(model, "gpt-4-turbo") || contains(model, "gpt-4o") {
		return 128000
	}
	if contains(model, "gpt-4-32k") {
		return 32768
	}
	if contains(model, "gpt-4") {
		return 8192
	}

	// Default
	return 8192
}

// CostCalculator calculates API costs
type CostCalculator struct {
	prices map[string]ModelPrice
}

// ModelPrice represents pricing for a model
type ModelPrice struct {
	InputPrice         float64 // per 1M tokens
	OutputPrice        float64 // per 1M tokens
	CacheReadPrice     float64 // per 1M tokens
	CacheCreationPrice float64 // per 1M tokens
}

// NewCostCalculator creates a cost calculator with default prices
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{
		prices: map[string]ModelPrice{
			"claude-opus-4":     {15.0, 75.0, 1.5, 18.75},
			"claude-sonnet-4":   {3.0, 15.0, 0.3, 3.75},
			"claude-sonnet-3.5": {3.0, 15.0, 0.3, 3.75},
			"claude-haiku-3.5":  {0.8, 4.0, 0.08, 1.0},
			"gpt-4o":            {2.5, 10.0, 0, 0},
			"gpt-4-turbo":       {10.0, 30.0, 0, 0},
		},
	}
}

// CalculateCost calculates the cost for a usage
func (c *CostCalculator) CalculateCost(model string, usage TokenUsage) float64 {
	price, ok := c.prices[model]
	if !ok {
		// Default pricing for unknown models
		price = ModelPrice{3.0, 15.0, 0.3, 3.75}
	}

	cost := 0.0
	cost += float64(usage.InputTokens) * price.InputPrice / 1_000_000
	cost += float64(usage.OutputTokens) * price.OutputPrice / 1_000_000
	cost += float64(usage.CacheReadTokens) * price.CacheReadPrice / 1_000_000
	cost += float64(usage.CacheCreationTokens) * price.CacheCreationPrice / 1_000_000

	return cost
}

// TokenCounter tracks cumulative token usage
type TokenCounter struct {
	totalInput  int64
	totalOutput int64
	modelUsage  map[string]*ModelUsage
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		modelUsage: make(map[string]*ModelUsage),
	}
}

// Add adds token usage for a model
func (tc *TokenCounter) Add(model string, usage TokenUsage) {
	tc.totalInput += int64(usage.InputTokens)
	tc.totalOutput += int64(usage.OutputTokens)

	if _, ok := tc.modelUsage[model]; !ok {
		tc.modelUsage[model] = &ModelUsage{
			ContextWindow:   GetContextWindow(model),
			MaxOutputTokens: 4096,
		}
	}

	mu := tc.modelUsage[model]
	mu.InputTokens += usage.InputTokens
	mu.OutputTokens += usage.OutputTokens
	mu.CacheReadTokens += usage.CacheReadTokens
	mu.CacheCreationTokens += usage.CacheCreationTokens
}

// GetTotals returns total token usage
func (tc *TokenCounter) GetTotals() (input, output int64) {
	return tc.totalInput, tc.totalOutput
}

// GetModelUsage returns usage for a specific model
func (tc *TokenCounter) GetModelUsage(model string) *ModelUsage {
	return tc.modelUsage[model]
}

// GetAllModelUsage returns usage for all models
func (tc *TokenCounter) GetAllModelUsage() map[string]*ModelUsage {
	return tc.modelUsage
}

// Reset resets the counter
func (tc *TokenCounter) Reset() {
	tc.totalInput = 0
	tc.totalOutput = 0
	tc.modelUsage = make(map[string]*ModelUsage)
}
