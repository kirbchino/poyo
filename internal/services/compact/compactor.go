// Package compact provides context compaction functionality
package compact

import (
	"fmt"
	"strings"
	"time"
)

// Compactor provides context compaction functionality
type Compactor struct {
	config CompactorConfig
}

// CompactorConfig contains configuration for the compactor
type CompactorConfig struct {
	TargetUtilization     float64
	MinMessagesToKeep     int
	MaxSummaryLength      int
	PreserveSystemMessages bool
	PreserveToolResults   bool
}

// DefaultCompactorConfig returns default compactor configuration
func DefaultCompactorConfig() CompactorConfig {
	return CompactorConfig{
		TargetUtilization:      0.5,
		MinMessagesToKeep:      4,
		MaxSummaryLength:       1000,
		PreserveSystemMessages: true,
		PreserveToolResults:    true,
	}
}

// NewCompactor creates a new compactor
func NewCompactor(config CompactorConfig) *Compactor {
	return &Compactor{
		config: config,
	}
}

// CompactResult contains the result of compaction
type CompactResult struct {
	Messages        []interface{}
	RemovedCount    int
	SummaryAdded    bool
	TokensBefore    int
	TokensAfter     int
	CompactionRatio float64
	Summary         string
}

// MessageData represents message data for compaction
type MessageData struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// Compact compacts messages to fit within token budget
func (c *Compactor) Compact(messages []MessageData, maxTokens int) (*CompactResult, error) {
	result := &CompactResult{
		Messages: make([]interface{}, 0),
	}

	// Count current tokens
	for _, msg := range messages {
		result.TokensBefore += len(msg.Content) / 4
	}

	// Check if compaction is needed
	if result.TokensBefore <= maxTokens {
		for _, msg := range messages {
			result.Messages = append(result.Messages, msg)
		}
		result.TokensAfter = result.TokensBefore
		return result, nil
	}

	// Build summary for older messages
	preserveCount := c.config.MinMessagesToKeep
	if preserveCount > len(messages) {
		preserveCount = len(messages)
	}

	// Create summary of removed messages
	removedMessages := messages[:len(messages)-preserveCount]
	summary := c.buildSummary(removedMessages)

	// Keep recent messages
	for _, msg := range messages[len(messages)-preserveCount:] {
		result.Messages = append(result.Messages, msg)
	}

	result.SummaryAdded = true
	result.RemovedCount = len(removedMessages)
	result.Summary = summary

	// Count new tokens
	for _, msg := range result.Messages {
		result.TokensAfter += len(fmt.Sprintf("%v", msg)) / 4
	}
	result.TokensAfter += len(summary) / 4

	result.CompactionRatio = float64(result.TokensAfter) / float64(result.TokensBefore)

	return result, nil
}

// buildSummary builds a summary of messages
func (c *Compactor) buildSummary(messages []MessageData) string {
	var summary strings.Builder

	summary.WriteString("[Context Compaction Summary]\n")
	summary.WriteString(fmt.Sprintf("Compacted %d messages:\n", len(messages)))

	userCount := 0
	assistantCount := 0
	for _, msg := range messages {
		if msg.Role == "user" {
			userCount++
		} else if msg.Role == "assistant" {
			assistantCount++
		}
	}

	summary.WriteString(fmt.Sprintf("- %d user messages\n", userCount))
	summary.WriteString(fmt.Sprintf("- %d assistant messages\n", assistantCount))

	return summary.String()
}

// ShouldCompact determines if compaction should be triggered
func (c *Compactor) ShouldCompact(messages []MessageData, maxTokens int) bool {
	currentTokens := 0
	for _, msg := range messages {
		currentTokens += len(msg.Content) / 4
	}

	utilization := float64(currentTokens) / float64(maxTokens)
	return utilization > 0.8
}

// SelectiveCompactor provides selective compaction strategies
type SelectiveCompactor struct {
	*Compactor
}

// NewSelectiveCompactor creates a new selective compactor
func NewSelectiveCompactor(config CompactorConfig) *SelectiveCompactor {
	return &SelectiveCompactor{
		Compactor: NewCompactor(config),
	}
}

// CompactByAge compacts messages older than a threshold
func (sc *SelectiveCompactor) CompactByAge(messages []MessageData, maxAge time.Duration) []MessageData {
	result := make([]MessageData, 0)
	cutoff := time.Now().Add(-maxAge)

	for _, msg := range messages {
		if msg.Timestamp.After(cutoff) {
			result = append(result, msg)
		}
	}

	return result
}

// CompactByRelevance compacts messages based on relevance scoring
func (sc *SelectiveCompactor) CompactByRelevance(messages []MessageData, query string, keepCount int) []MessageData {
	if len(messages) <= keepCount {
		return messages
	}

	// Keep most recent messages
	result := make([]MessageData, 0)
	recentCount := keepCount / 2

	// Add recent messages
	startIdx := len(messages) - recentCount
	if startIdx < 0 {
		startIdx = 0
	}
	result = append(result, messages[startIdx:]...)

	// Add messages that match query keywords
	queryLower := strings.ToLower(query)
	for i := 0; i < startIdx && len(result) < keepCount; i++ {
		if strings.Contains(strings.ToLower(messages[i].Content), queryLower) {
			result = append(result, messages[i])
		}
	}

	return result
}
