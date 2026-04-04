// Package types contains core type definitions for the Poyo implementation.
package types

import (
	"time"
)

// UUID represents a unique identifier for messages and sessions.
type UUID string

// MessageOrigin represents where a message came from.
type MessageOrigin string

const (
	OriginHuman    MessageOrigin = "human"
	OriginTeammate MessageOrigin = "teammate"
	OriginSystem   MessageOrigin = "system"
	OriginTick     MessageOrigin = "tick"
	OriginTask     MessageOrigin = "task"
)

// MessageType discriminates between different message types.
type MessageType string

const (
	MessageTypeUser      MessageType = "user"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeSystem    MessageType = "system"
	MessageTypeProgress  MessageType = "progress"
	MessageTypeAttachment MessageType = "attachment"
	MessageTypeTombstone MessageType = "tombstone"
	MessageTypeToolUseSummary MessageType = "tool-use-summary"
	MessageTypeStreamEvent MessageType = "stream-event"
	MessageTypeRequestStart MessageType = "request-start"
)

// ContentBlock represents a block of content in a message.
// This mirrors Anthropic's content block structure.
type ContentBlock struct {
	Type string `json:"type"`

	// For text blocks
	Text string `json:"text,omitempty"`

	// For tool_use blocks
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Input    map[string]interface{} `json:"input,omitempty"`

	// For tool_result blocks
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`

	// For image blocks
	Source   *ImageSource `json:"source,omitempty"`

	// For thinking blocks
	Thinking string `json:"thinking,omitempty"`
}

// ImageSource represents the source of an image.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
	URL       string `json:"url,omitempty"`
}

// Message is the base message type.
type Message struct {
	UUID      UUID         `json:"uuid"`
	Type      MessageType  `json:"type"`
	Timestamp time.Time    `json:"timestamp"`
	Origin    MessageOrigin `json:"origin,omitempty"`
}

// UserMessage represents a message from the user.
type UserMessage struct {
	Message
	Content       []ContentBlock `json:"content"`
	IsMeta        bool           `json:"isMeta,omitempty"`
	SourceToolAssistantUUID *UUID `json:"sourceToolAssistantUUID,omitempty"`
	ToolUseResult string         `json:"toolUseResult,omitempty"`
}

// AssistantMessage represents a message from the assistant.
type AssistantMessage struct {
	Message
	Content         []ContentBlock `json:"content"`
	APIError        string         `json:"apiError,omitempty"`
	Usage           *Usage         `json:"usage,omitempty"`
	Model           string         `json:"model,omitempty"`
	StopReason      string         `json:"stopReason,omitempty"`
}

// SystemMessage represents a system-level message.
type SystemMessage struct {
	Message
	Content     string `json:"content"`
	Level       string `json:"level,omitempty"` // info, warning, error
	SubType     string `json:"subType,omitempty"`
}

// ProgressMessage represents progress updates during tool execution.
type ProgressMessage struct {
	Message
	ToolUseID string      `json:"toolUseId"`
	Data      interface{} `json:"data"`
}

// AttachmentMessage represents an attachment in a message.
type AttachmentMessage struct {
	Message
	Attachments []Attachment `json:"attachments"`
}

// Attachment represents a file attachment.
type Attachment struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	MediaType   string `json:"mediaType,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Dimensions  *ImageDimensions `json:"dimensions,omitempty"`
}

// ImageDimensions represents the dimensions of an image.
type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TombstoneMessage represents a deleted or replaced message.
type TombstoneMessage struct {
	Message
	Reason string `json:"reason"`
}

// ToolUseSummaryMessage represents a summary of tool usage.
type ToolUseSummaryMessage struct {
	Message
	ToolName    string `json:"toolName"`
	Summary     string `json:"summary"`
	Success     bool   `json:"success"`
	DurationMs  int64  `json:"durationMs,omitempty"`
}

// StreamEvent represents a streaming event from the API.
type StreamEvent struct {
	Type      string      `json:"type"`
	Index     int         `json:"index,omitempty"`
	Delta     string      `json:"delta,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Message   *AssistantMessage `json:"message,omitempty"`
}

// RequestStartEvent represents the start of an API request.
type RequestStartEvent struct {
	RequestID string    `json:"requestId"`
	Model     string    `json:"model"`
	Timestamp time.Time `json:"timestamp"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens       int64 `json:"input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
}

// MessageLookups provides lookup maps for messages.
type MessageLookups struct {
	ByUUID map[UUID]Message
	ByType map[MessageType][]Message
}

// NormalizeMessage normalizes a message for API transmission.
func NormalizeMessage(msg interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	switch m := msg.(type) {
	case *UserMessage:
		result["uuid"] = m.UUID
		result["type"] = m.Type
		result["timestamp"] = m.Timestamp
		result["content"] = m.Content
		result["isMeta"] = m.IsMeta
		result["role"] = "user"
	case *AssistantMessage:
		result["uuid"] = m.UUID
		result["type"] = m.Type
		result["timestamp"] = m.Timestamp
		result["content"] = m.Content
		result["model"] = m.Model
		result["stop_reason"] = m.StopReason
		result["role"] = "assistant"
		if m.Usage != nil {
			result["usage"] = m.Usage
		}
	case *SystemMessage:
		result["uuid"] = m.UUID
		result["type"] = m.Type
		result["timestamp"] = m.Timestamp
		result["content"] = m.Content
		result["level"] = m.Level
		result["role"] = "system"
	case map[string]string:
		// Handle simple map[string]string (e.g., {"role": "user", "content": "hello"})
		for k, v := range m {
			result[k] = v
		}
	case map[string]interface{}:
		// Handle map[string]interface{}
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}
