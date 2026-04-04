// Package tokenizer provides token counting utilities
package tokenizer

import (
	"unicode"
	"unicode/utf8"
)

// Tokenizer provides token counting functionality
type Tokenizer struct {
	// Tokenizer for Claude models uses approximately 4 characters per token
	// This is a simplified approximation
	charsPerToken float64
}

// NewTokenizer creates a new tokenizer
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		charsPerToken: 4.0, // Approximate for Claude models
	}
}

// Count counts the number of tokens in a string
func (t *Tokenizer) Count(text string) int {
	if len(text) == 0 {
		return 0
	}

	// Simple approximation: count words and adjust
	// Claude models use a BPE tokenizer similar to GPT
	// We use a heuristic approach here

	charCount := utf8.RuneCountInString(text)
	wordCount := countWords(text)

	// Estimate tokens as average of character-based and word-based estimates
	charTokens := float64(charCount) / t.charsPerToken
	wordTokens := float64(wordCount) * 1.3 // Words typically become 1.3 tokens

	estimate := (charTokens + wordTokens) / 2

	return int(estimate)
}

// CountMessage counts tokens in a message
func (t *Tokenizer) CountMessage(role string, content string) int {
	// Each message has overhead for role and formatting
	overhead := 4 // tokens for message structure
	return t.Count(role) + t.Count(content) + overhead
}

// CountMessages counts tokens in multiple messages
func (t *Tokenizer) CountMessages(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += t.CountMessage(msg.Role, msg.Content)
	}
	return total
}

// Message represents a message for token counting
type Message struct {
	Role    string
	Content string
}

// countWords counts words in a string
func countWords(s string) int {
	count := 0
	inWord := false

	for _, r := range s {
		if isWordChar(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	return count
}

// isWordChar checks if a rune is a word character
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// TokenBudget represents a token budget
type TokenBudget struct {
	MaxTokens    int
	UsedTokens   int
	Reserved     int
	InputTokens  int
	OutputTokens int
}

// NewTokenBudget creates a new token budget
func NewTokenBudget(maxTokens int) *TokenBudget {
	return &TokenBudget{
		MaxTokens: maxTokens,
	}
}

// Available returns the number of available tokens
func (b *TokenBudget) Available() int {
	return b.MaxTokens - b.UsedTokens - b.Reserved
}

// Use uses tokens from the budget
func (b *TokenBudget) Use(tokens int) bool {
	if b.Available() < tokens {
		return false
	}
	b.UsedTokens += tokens
	return true
}

// Reserve reserves tokens from the budget
func (b *TokenBudget) Reserve(tokens int) bool {
	if b.Available() < tokens {
		return false
	}
	b.Reserved += tokens
	return true
}

// Release releases reserved tokens
func (b *TokenBudget) Release(tokens int) {
	b.Reserved -= tokens
	if b.Reserved < 0 {
		b.Reserved = 0
	}
}

// Reset resets the budget
func (b *TokenBudget) Reset() {
	b.UsedTokens = 0
	b.Reserved = 0
	b.InputTokens = 0
	b.OutputTokens = 0
}

// UpdateUsage updates token usage from API response
func (b *TokenBudget) UpdateUsage(input, output int) {
	b.InputTokens = input
	b.OutputTokens = output
	b.UsedTokens = input + output
}

// ContextLimit returns the context limit for a model
func ContextLimit(model string) int {
	limits := map[string]int{
		"claude-opus-4-6":   200000,
		"claude-sonnet-4-6": 200000,
		"claude-haiku-4-5":  200000,
		"claude-3-5-sonnet": 200000,
		"claude-3-opus":     200000,
		"claude-3-sonnet":   200000,
		"claude-3-haiku":    200000,
	}

	if limit, ok := limits[model]; ok {
		return limit
	}
	return 200000 // Default
}

// ShouldCompact determines if context should be compacted
func ShouldCompact(currentTokens, maxTokens int, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.8 // Default 80%
	}
	utilization := float64(currentTokens) / float64(maxTokens)
	return utilization >= threshold
}

// CompactionTarget returns the target token count after compaction
func CompactionTarget(maxTokens int, targetUtilization float64) int {
	if targetUtilization <= 0 {
		targetUtilization = 0.6 // Default 60%
	}
	return int(float64(maxTokens) * targetUtilization)
}
