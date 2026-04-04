// Package compactor provides session context compression capabilities.
package compactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// CompressionStrategy represents the compression strategy
type CompressionStrategy string

const (
	StrategySummarize   CompressionStrategy = "summarize"   // LLM摘要
	StrategyTruncate    CompressionStrategy = "truncate"    // 简单截断
	StrategySemantic    CompressionStrategy = "semantic"    // 语义压缩
	StrategyHierarchical CompressionStrategy = "hierarchical" // 层级压缩
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeUser      MessageType = "user"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeTool      MessageType = "tool"
	MessageTypeSystem    MessageType = "system"
	MessageTypeSummary   MessageType = "summary"
)

// Message represents a conversation message
type Message struct {
	ID        string                 `json:"id"`
	Type      MessageType            `json:"type"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Role      string                 `json:"role,omitempty"`
	Name      string                 `json:"name,omitempty"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	TokenCount int                   `json:"token_count,omitempty"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Arguments string                `json:"arguments"`
	Result   string                 `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// Summary represents a compressed summary of messages
type Summary struct {
	ID           string    `json:"id"`
	StartID      string    `json:"start_id"`
	EndID        string    `json:"end_id"`
	Content      string    `json:"content"`
	TokenCount   int       `json:"token_count"`
	OriginalTokens int     `json:"original_tokens"`
	CompressedAt time.Time `json:"compressed_at"`
	KeyPoints    []string  `json:"key_points,omitempty"`
	Entities     []Entity  `json:"entities,omitempty"`
}

// Entity represents an extracted entity
type Entity struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// CompressionConfig represents compression configuration
type CompressionConfig struct {
	Strategy           CompressionStrategy `json:"strategy"`
	MaxTokens          int                 `json:"max_tokens"`
	TargetRatio        float64             `json:"target_ratio"` // 目标压缩比
	PreserveRecent     int                 `json:"preserve_recent"` // 保留最近N条消息
	MinMessagesToCompact int               `json:"min_messages_to_compact"`
	MaxSummaryLength   int                 `json:"max_summary_length"`
}

// DefaultCompressionConfig returns default compression config
func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Strategy:           StrategySummarize,
		MaxTokens:          100000,
		TargetRatio:        0.3,
		PreserveRecent:     5,
		MinMessagesToCompact: 10,
		MaxSummaryLength:   4000,
	}
}

// Session represents a conversation session
type Session struct {
	ID        string     `json:"id"`
	Messages  []Message  `json:"messages"`
	Summaries []Summary  `json:"summaries"`
	TokenCount int       `json:"token_count"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Compactor handles session compression
type Compactor struct {
	config      *CompressionConfig
	tokenizer   Tokenizer
	summarizer  Summarizer
	mu          sync.RWMutex
}

// Tokenizer interface for token counting
type Tokenizer interface {
	CountTokens(text string) int
}

// Summarizer interface for LLM summarization
type Summarizer interface {
	Summarize(ctx context.Context, messages []Message, maxTokens int) (*Summary, error)
}

// NewCompactor creates a new compactor
func NewCompactor(config *CompressionConfig, tokenizer Tokenizer, summarizer Summarizer) *Compactor {
	if config == nil {
		config = DefaultCompressionConfig()
	}
	return &Compactor{
		config:     config,
		tokenizer:  tokenizer,
		summarizer: summarizer,
	}
}

// AddMessage adds a message to the session
func (c *Compactor) AddMessage(session *Session, msg Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if msg.ID == "" {
		msg.ID = generateMessageID()
	}
	msg.Timestamp = time.Now()

	// Calculate token count
	if c.tokenizer != nil {
		msg.TokenCount = c.tokenizer.CountTokens(msg.Content)
	}

	session.Messages = append(session.Messages, msg)
	session.TokenCount += msg.TokenCount
	session.UpdatedAt = time.Now()
}

// ShouldCompact checks if the session needs compression
func (c *Compactor) ShouldCompact(session *Session) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(session.Messages) < c.config.MinMessagesToCompact {
		return false
	}

	if session.TokenCount > c.config.MaxTokens {
		return true
	}

	return false
}

// Compact performs compression on the session
func (c *Compactor) Compact(ctx context.Context, session *Session) (*Summary, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we have enough messages
	if len(session.Messages) <= c.config.PreserveRecent {
		return nil, fmt.Errorf("not enough messages to compact")
	}

	// Identify messages to compact (preserve recent)
	compactEnd := len(session.Messages) - c.config.PreserveRecent
	messagesToCompact := session.Messages[:compactEnd]

	if len(messagesToCompact) == 0 {
		return nil, fmt.Errorf("no messages to compact")
	}

	// Calculate original token count
	originalTokens := 0
	for _, msg := range messagesToCompact {
		originalTokens += msg.TokenCount
	}

	// Perform compression based on strategy
	var summary *Summary
	var err error

	switch c.config.Strategy {
	case StrategySummarize:
		summary, err = c.summarizeMessages(ctx, session, messagesToCompact)
	case StrategyTruncate:
		summary, err = c.truncateMessages(session, messagesToCompact)
	case StrategySemantic:
		summary, err = c.semanticCompress(ctx, session, messagesToCompact)
	case StrategyHierarchical:
		summary, err = c.hierarchicalCompress(ctx, session, messagesToCompact)
	default:
		summary, err = c.summarizeMessages(ctx, session, messagesToCompact)
	}

	if err != nil {
		return nil, err
	}

	// Update session
	summary.OriginalTokens = originalTokens

	// Replace compacted messages with summary
	summaryMsg := Message{
		ID:         generateMessageID(),
		Type:       MessageTypeSummary,
		Content:    summary.Content,
		Timestamp:  time.Now(),
		TokenCount: summary.TokenCount,
		Metadata: map[string]interface{}{
			"summary_id":      summary.ID,
			"original_tokens": originalTokens,
			"compacted_at":    summary.CompressedAt,
		},
	}

	// Rebuild session with summary + preserved messages
	session.Messages = append([]Message{summaryMsg}, session.Messages[compactEnd:]...)
	session.TokenCount = c.calculateTotalTokens(session.Messages)
	session.Summaries = append(session.Summaries, *summary)

	return summary, nil
}

// summarizeMessages uses LLM to summarize messages
func (c *Compactor) summarizeMessages(ctx context.Context, session *Session, messages []Message) (*Summary, error) {
	if c.summarizer == nil {
		// Fallback to simple truncation
		return c.truncateMessages(session, messages)
	}

	summary, err := c.summarizer.Summarize(ctx, messages, c.config.MaxSummaryLength)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	summary.ID = generateSummaryID()
	summary.StartID = messages[0].ID
	summary.EndID = messages[len(messages)-1].ID
	summary.CompressedAt = time.Now()

	return summary, nil
}

// truncateMessages performs simple truncation
func (c *Compactor) truncateMessages(session *Session, messages []Message) (*Summary, error) {
	var content strings.Builder
	keyPoints := []string{}

	// Build truncated summary
	for i, msg := range messages {
		if i > 0 {
			content.WriteString("\n\n")
		}

		// Add message type indicator
		content.WriteString(fmt.Sprintf("[%s]: ", msg.Type))

		// Truncate long messages
		maxLen := c.config.MaxSummaryLength / len(messages)
		if maxLen < 100 {
			maxLen = 100
		}

		text := msg.Content
		if len(text) > maxLen {
			text = text[:maxLen] + "..."
		}
		content.WriteString(text)

		// Extract key points from tool calls
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				keyPoints = append(keyPoints, fmt.Sprintf("Used tool: %s", tc.Name))
			}
		}
	}

	summary := &Summary{
		ID:            generateSummaryID(),
		StartID:       messages[0].ID,
		EndID:         messages[len(messages)-1].ID,
		Content:       content.String(),
		KeyPoints:     keyPoints,
		CompressedAt:  time.Now(),
	}

	if c.tokenizer != nil {
		summary.TokenCount = c.tokenizer.CountTokens(summary.Content)
	}

	return summary, nil
}

// semanticCompress performs semantic compression
func (c *Compactor) semanticCompress(ctx context.Context, session *Session, messages []Message) (*Summary, error) {
	// Group messages by topic/semantic similarity
	groups := c.groupMessagesByTopic(messages)

	var content strings.Builder
	keyPoints := []string{}
	entities := []Entity{}

	for i, group := range groups {
		if i > 0 {
			content.WriteString("\n\n---\n\n")
		}

		// Summarize each group
		groupSummary := c.summarizeGroup(group)
		content.WriteString(groupSummary)

		// Extract key points
		keyPoints = append(keyPoints, c.extractKeyPoints(group)...)

		// Extract entities
		entities = append(entities, c.extractEntities(group)...)
	}

	summary := &Summary{
		ID:           generateSummaryID(),
		StartID:      messages[0].ID,
		EndID:        messages[len(messages)-1].ID,
		Content:      content.String(),
		KeyPoints:    keyPoints,
		Entities:     entities,
		CompressedAt: time.Now(),
	}

	if c.tokenizer != nil {
		summary.TokenCount = c.tokenizer.CountTokens(summary.Content)
	}

	return summary, nil
}

// hierarchicalCompress performs hierarchical compression
func (c *Compactor) hierarchicalCompress(ctx context.Context, session *Session, messages []Message) (*Summary, error) {
	// Create hierarchical summary
	// Level 1: Group adjacent messages
	// Level 2: Summarize groups
	// Level 3: Combine into final summary

	chunkSize := 5
	chunks := chunkMessages(messages, chunkSize)

	var summaries []string
	keyPoints := []string{}

	for _, chunk := range chunks {
		chunkSummary := c.summarizeChunk(chunk)
		summaries = append(summaries, chunkSummary)
		keyPoints = append(keyPoints, c.extractKeyPoints(chunk)...)
	}

	// Combine chunk summaries
	finalContent := strings.Join(summaries, "\n\n")

	summary := &Summary{
		ID:           generateSummaryID(),
		StartID:      messages[0].ID,
		EndID:        messages[len(messages)-1].ID,
		Content:      finalContent,
		KeyPoints:    keyPoints,
		CompressedAt: time.Now(),
	}

	if c.tokenizer != nil {
		summary.TokenCount = c.tokenizer.CountTokens(summary.Content)
	}

	return summary, nil
}

// Helper functions

func (c *Compactor) calculateTotalTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += msg.TokenCount
	}
	return total
}

func (c *Compactor) groupMessagesByTopic(messages []Message) [][]Message {
	// Simple grouping by consecutive tool calls and user messages
	var groups [][]Message
	var currentGroup []Message

	for _, msg := range messages {
		if len(currentGroup) > 0 {
			// Start new group on topic change
			if msg.Type == MessageTypeUser && currentGroup[len(currentGroup)-1].Type != MessageTypeUser {
				groups = append(groups, currentGroup)
				currentGroup = nil
			}
		}
		currentGroup = append(currentGroup, msg)
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

func (c *Compactor) summarizeGroup(messages []Message) string {
	var parts []string

	for _, msg := range messages {
		switch msg.Type {
		case MessageTypeUser:
			parts = append(parts, "User asked: "+truncateText(msg.Content, 200))
		case MessageTypeAssistant:
			parts = append(parts, "Assistant responded: "+truncateText(msg.Content, 200))
		case MessageTypeTool:
			for _, tc := range msg.ToolCalls {
				parts = append(parts, fmt.Sprintf("Tool %s executed", tc.Name))
			}
		}
	}

	return strings.Join(parts, "\n")
}

func (c *Compactor) extractKeyPoints(messages []Message) []string {
	var points []string

	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				points = append(points, fmt.Sprintf("Used %s: %s", tc.Name, truncateText(tc.Arguments, 100)))
			}
		}

		// Check for important keywords
		content := strings.ToLower(msg.Content)
		if strings.Contains(content, "error") || strings.Contains(content, "failed") {
			points = append(points, "Error encountered: "+truncateText(msg.Content, 100))
		}
		if strings.Contains(content, "important") || strings.Contains(content, "note") {
			points = append(points, "Important: "+truncateText(msg.Content, 100))
		}
	}

	return points
}

func (c *Compactor) extractEntities(messages []Message) []Entity {
	// Simple entity extraction
	var entities []Entity

	for _, msg := range messages {
		// Extract file paths
		if strings.Contains(msg.Content, ".go") || strings.Contains(msg.Content, ".py") {
			entities = append(entities, Entity{
				Type: "file",
				Name: "referenced file",
			})
		}

		// Extract URLs
		if strings.Contains(msg.Content, "http://") || strings.Contains(msg.Content, "https://") {
			entities = append(entities, Entity{
				Type: "url",
				Name: "referenced URL",
			})
		}
	}

	return entities
}

func (c *Compactor) summarizeChunk(messages []Message) string {
	var parts []string

	for _, msg := range messages {
		switch msg.Type {
		case MessageTypeUser:
			parts = append(parts, "Q: "+truncateText(msg.Content, 100))
		case MessageTypeAssistant:
			parts = append(parts, "A: "+truncateText(msg.Content, 100))
		case MessageTypeTool:
			for _, tc := range msg.ToolCalls {
				parts = append(parts, fmt.Sprintf("[%s]", tc.Name))
			}
		}
	}

	return strings.Join(parts, " | ")
}

func chunkMessages(messages []Message, size int) [][]Message {
	var chunks [][]Message

	for i := 0; i < len(messages); i += size {
		end := i + size
		if end > len(messages) {
			end = len(messages)
		}
		chunks = append(chunks, messages[i:end])
	}

	return chunks
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

func generateSummaryID() string {
	return fmt.Sprintf("sum-%d", time.Now().UnixNano())
}

// NewSession creates a new session
func NewSession() *Session {
	return &Session{
		ID:        generateSessionID(),
		Messages:  []Message{},
		Summaries: []Summary{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func generateSessionID() string {
	return fmt.Sprintf("sess-%d", time.Now().UnixNano())
}

// SimpleTokenizer is a basic tokenizer implementation
type SimpleTokenizer struct {
	avgCharsPerToken float64
}

// NewSimpleTokenizer creates a new simple tokenizer
func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{
		avgCharsPerToken: 4.0, // Rough estimate for English text
	}
}

// CountTokens estimates token count
func (t *SimpleTokenizer) CountTokens(text string) int {
	return int(float64(len(text)) / t.avgCharsPerToken)
}

// GetStatistics returns compression statistics
func (c *Compactor) GetStatistics(session *Session) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalOriginalTokens := 0
	for _, sum := range session.Summaries {
		totalOriginalTokens += sum.OriginalTokens
	}

	return map[string]interface{}{
		"message_count":      len(session.Messages),
		"token_count":        session.TokenCount,
		"summary_count":      len(session.Summaries),
		"original_tokens":    totalOriginalTokens,
		"compression_ratio":  float64(session.TokenCount) / float64(totalOriginalTokens+session.TokenCount),
		"last_updated":       session.UpdatedAt,
	}
}

// Restore attempts to restore compressed messages (if available)
func (c *Compactor) Restore(session *Session, summaryID string) ([]Message, error) {
	// In a full implementation, this would restore from a cache or storage
	return nil, fmt.Errorf("restoration not supported in this implementation")
}

// Export exports the session in JSON format
func (c *Compactor) Export(session *Session) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return json.MarshalIndent(session, "", "  ")
}

// Import imports a session from JSON
func (c *Compactor) Import(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to import session: %w", err)
	}
	return &session, nil
}
