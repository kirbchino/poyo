package tokenizer

import (
	"testing"
)

func TestCountTokens(t *testing.T) {
	tokenizer := NewTokenizer("claude-sonnet-4")

	tests := []struct {
		name     string
		text     string
		minToken int
		maxToken int
	}{
		{
			name:     "Empty string",
			text:     "",
			minToken: 0,
			maxToken: 0,
		},
		{
			name:     "Simple English",
			text:     "Hello, world!",
			minToken: 2,
			maxToken: 10,
		},
		{
			name:     "Chinese text",
			text:     "你好世界",
			minToken: 2,
			maxToken: 6,
		},
		{
			name:     "Mixed text",
			text:     "Hello 你好 World 世界",
			minToken: 4,
			maxToken: 10,
		},
		{
			name:     "Long text",
			text:     "This is a longer piece of text that should have more tokens.",
			minToken: 10,
			maxToken: 20,
		},
		{
			name:     "Code snippet",
			text:     "func main() { fmt.Println(\"Hello\") }",
			minToken: 5,
			maxToken: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tokenizer.CountTokens(tt.text)
			if count < tt.minToken || count > tt.maxToken {
				t.Errorf("CountTokens(%q) = %d, want between %d and %d", tt.text, count, tt.minToken, tt.maxToken)
			}
		})
	}
}

func TestCountMessagesTokens(t *testing.T) {
	tokenizer := NewTokenizer("claude-sonnet-4")

	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	count := tokenizer.CountMessagesTokens(messages)
	if count <= 0 {
		t.Errorf("CountMessagesTokens returned %d, expected positive", count)
	}

	// Should be more than sum of individual contents
	sum := tokenizer.CountTokens("Hello") + tokenizer.CountTokens("Hi there!") + tokenizer.CountTokens("How are you?")
	if count < sum {
		t.Errorf("Message token count %d should be >= content sum %d", count, sum)
	}
}

func TestGetContextWindow(t *testing.T) {
	tests := []struct {
		model         string
		minContext    int
		expectedExact int
	}{
		{"claude-opus-4", 200000, 200000},
		{"claude-sonnet-4", 200000, 200000},
		{"claude-3-5-sonnet", 200000, 200000},
		{"claude-3-haiku", 200000, 200000},
		{"gpt-4-turbo", 128000, 128000},
		{"gpt-4o", 128000, 128000},
		{"gpt-4", 8192, 8192},
		{"unknown-model", 8192, 8192},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			context := GetContextWindow(tt.model)
			if context != tt.expectedExact {
				t.Errorf("GetContextWindow(%q) = %d, want %d", tt.model, context, tt.expectedExact)
			}
		})
	}
}

func TestEstimateMaxTokens(t *testing.T) {
	tokenizer := NewTokenizer("claude-sonnet-4")

	tests := []struct {
		name          string
		contextWindow int
		inputTokens   int
		minOutput     int
	}{
		{
			name:          "Small input",
			contextWindow: 200000,
			inputTokens:   1000,
			minOutput:     8192, // Should hit model max
		},
		{
			name:          "Large input",
			contextWindow: 200000,
			inputTokens:   195000,
			minOutput:     100, // Limited by remaining context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxTokens := tokenizer.EstimateMaxTokens(tt.contextWindow, tt.inputTokens)
			if maxTokens < tt.minOutput {
				t.Errorf("EstimateMaxTokens() = %d, want >= %d", maxTokens, tt.minOutput)
			}
		})
	}
}

func TestCostCalculator(t *testing.T) {
	calc := NewCostCalculator()

	tests := []struct {
		model     string
		usage     TokenUsage
		minCost   float64
		maxCost   float64
	}{
		{
			model: "claude-sonnet-4",
			usage: TokenUsage{InputTokens: 1000, OutputTokens: 500},
			minCost: 0.003,  // 1000 * 3/1M + 500 * 15/1M
			maxCost: 0.015,
		},
		{
			model: "claude-opus-4",
			usage: TokenUsage{InputTokens: 1000, OutputTokens: 500},
			minCost: 0.015,  // 1000 * 15/1M + 500 * 75/1M
			maxCost: 0.075,
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cost := calc.CalculateCost(tt.model, tt.usage)
			if cost < tt.minCost || cost > tt.maxCost {
				t.Errorf("CalculateCost() = %f, want between %f and %f", cost, tt.minCost, tt.maxCost)
			}
		})
	}
}

func TestTokenCounter(t *testing.T) {
	counter := NewTokenCounter()

	// Add usage for model 1
	counter.Add("claude-sonnet-4", TokenUsage{
		InputTokens:  1000,
		OutputTokens: 500,
	})

	input, output := counter.GetTotals()
	if input != 1000 || output != 500 {
		t.Errorf("GetTotals() = (%d, %d), want (1000, 500)", input, output)
	}

	// Add usage for model 2
	counter.Add("claude-opus-4", TokenUsage{
		InputTokens:  2000,
		OutputTokens: 1000,
	})

	input, output = counter.GetTotals()
	if input != 3000 || output != 1500 {
		t.Errorf("GetTotals() = (%d, %d), want (3000, 1500)", input, output)
	}

	// Check model usage
	sonnetUsage := counter.GetModelUsage("claude-sonnet-4")
	if sonnetUsage == nil || sonnetUsage.InputTokens != 1000 {
		t.Errorf("GetModelUsage() returned wrong data")
	}

	// Reset
	counter.Reset()
	input, output = counter.GetTotals()
	if input != 0 || output != 0 {
		t.Errorf("After Reset(), GetTotals() = (%d, %d), want (0, 0)", input, output)
	}
}

func TestCountTokensWithSpecialChars(t *testing.T) {
	tokenizer := NewTokenizer("claude-sonnet-4")

	tests := []struct {
		name string
		text string
	}{
		{"Newlines", "Line1\nLine2\nLine3"},
		{"Tabs", "Col1\tCol2\tCol3"},
		{"Mixed whitespace", "  Multiple   spaces  "},
		{"Unicode", "🎉🚀💻"},
		{"JSON-like", `{"key": "value", "nested": {"a": 1}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tokenizer.CountTokens(tt.text)
			if count < 0 {
				t.Errorf("CountTokens returned negative: %d", count)
			}
		})
	}
}
