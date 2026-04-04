// Package types contains extended type definitions
package types

import (
	"time"
)

// Message interface methods - adding to support query engine
// These methods allow polymorphic access to message properties

// MessageRole returns the role of a message
func (m *UserMessage) MessageRole() string {
	return "user"
}

// MessageRole returns the role of a message
func (m *AssistantMessage) MessageRole() string {
	return "assistant"
}

// MessageRole returns the role of a message
func (m *SystemMessage) MessageRole() string {
	return "system"
}

// MessageContent returns content blocks
func (m *UserMessage) MessageContent() []ContentBlock {
	return m.Content
}

// MessageContent returns content blocks
func (m *AssistantMessage) MessageContent() []ContentBlock {
	return m.Content
}

// MessageContent returns content blocks for system message
func (m *SystemMessage) MessageContent() []ContentBlock {
	return []ContentBlock{{Type: "text", Text: m.Content}}
}

// ToolResultMessage represents a tool result message
type ToolResultMessage struct {
	UUID       string
	Timestamp  time.Time
	ToolUseID  string
	ToolName   string
	Content    interface{}
	IsError    bool
}

// Role returns the message role
func (m *ToolResultMessage) Role() string {
	return "user"
}

// GetContent returns the content
func (m *ToolResultMessage) GetContent() []ContentBlock {
	return []ContentBlock{
		{
			Type:      "tool_result",
			ToolUseID: m.ToolUseID,
			Content:   m.Content,
			IsError:   m.IsError,
		},
	}
}

// BaseMessage provides common message fields
type BaseMessage struct {
	UUIDValue      string
	MsgType        string
	TimestampValue time.Time
}

// ID returns the message UUID
func (m *BaseMessage) ID() string {
	return m.UUIDValue
}

// Type returns the message type
func (m *BaseMessage) Type() string {
	return m.MsgType
}

// Time returns the timestamp
func (m *BaseMessage) Time() time.Time {
	return m.TimestampValue
}

// BaseContentBlock provides base content block implementation
type BaseContentBlock struct {
	BlockType    string
	TextContent  string
	IDValue      string
	NameValue    string
	InputValue   map[string]interface{}
	ToolUseIDVal string
	ContentVal   interface{}
	IsErrorVal   bool
}

// Type returns the block type
func (b *BaseContentBlock) Type() string {
	return b.BlockType
}

// Text returns text content
func (b *BaseContentBlock) Text() string {
	return b.TextContent
}

// ID returns the ID
func (b *BaseContentBlock) ID() string {
	return b.IDValue
}

// Name returns the name
func (b *BaseContentBlock) Name() string {
	return b.NameValue
}

// Input returns input data
func (b *BaseContentBlock) Input() map[string]interface{} {
	return b.InputValue
}

// ToolUseID returns tool use ID
func (b *BaseContentBlock) ToolUseID() string {
	return b.ToolUseIDVal
}

// Content returns content
func (b *BaseContentBlock) Content() interface{} {
	return b.ContentVal
}

// IsError returns if it's an error
func (b *BaseContentBlock) IsError() bool {
	return b.IsErrorVal
}

// Message interface for polymorphic access
type MessageInterface interface {
	Role() string
	GetContent() []ContentBlock
}

// ContentBlockInterface for content block access
type ContentBlockInterface interface {
	Type() string
	Text() string
	ID() string
	Name() string
	Input() map[string]interface{}
	ToolUseID() string
	Content() interface{}
	IsError() bool
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// TokenBudgetManager manages token budgets
type TokenBudgetManager struct {
	maxTokens   int
	usedTokens  int
	inputTokens int
	outputTokens int
}

// NewTokenBudgetManager creates a new token budget manager
func NewTokenBudgetManager() *TokenBudgetManager {
	return &TokenBudgetManager{}
}

// ShouldCompact determines if compaction is needed
func (m *TokenBudgetManager) ShouldCompact(messages []Message, model string) bool {
	// Simple heuristic: compact if more than 50 messages
	return len(messages) > 50
}

// Compact performs message compaction
func (m *TokenBudgetManager) Compact(messages []Message) []Message {
	// Keep last 20 messages
	if len(messages) <= 20 {
		return messages
	}
	return messages[len(messages)-20:]
}

// ContextLimit returns the context limit for a model
func ContextLimit(model string) int {
	limits := map[string]int{
		"claude-opus-4-6":   200000,
		"claude-sonnet-4-6": 200000,
		"claude-haiku-4-5":  200000,
		"claude-3-5-sonnet": 200000,
		"claude-3-opus":     200000,
	}

	if limit, ok := limits[model]; ok {
		return limit
	}
	return 200000
}
