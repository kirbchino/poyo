// Package streaming provides SSE parsing and event dispatching for API streaming
package streaming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// EventType represents the type of streaming event
type EventType string

const (
	EventTypeMessageStart    EventType = "message_start"
	EventTypeContentBlockStart EventType = "content_block_start"
	EventTypeContentBlockDelta EventType = "content_block_delta"
	EventTypeContentBlockStop  EventType = "content_block_stop"
	EventTypeMessageDelta      EventType = "message_delta"
	EventTypeMessageStop       EventType = "message_stop"
	EventTypePing              EventType = "ping"
	EventTypeError             EventType = "error"
)

// Event represents a streaming event
type Event struct {
	Type  EventType     `json:"type"`
	Index int           `json:"index,omitempty"`
	Delta *DeltaContent `json:"delta,omitempty"`
	// For message_start
	Message *MessageData `json:"message,omitempty"`
	// For content_block_start
	ContentBlock *ContentBlockData `json:"content_block,omitempty"`
	// For message_delta
	Usage *UsageData `json:"usage,omitempty"`
	// For error
	Error *ErrorData `json:"error,omitempty"`
}

// DeltaContent represents a content delta
type DeltaContent struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

// MessageData represents message data in events
type MessageData struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Role       string        `json:"role"`
	Content    []interface{} `json:"content"`
	Model      string        `json:"model"`
	StopReason string        `json:"stop_reason,omitempty"`
	Usage      *UsageData    `json:"usage,omitempty"`
}

// ContentBlockData represents content block data
type ContentBlockData struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	ID   string      `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

// UsageData represents token usage
type UsageData struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// ErrorData represents an error in streaming
type ErrorData struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SSEParser parses Server-Sent Events from a reader
type SSEParser struct {
	reader *bufio.Reader
	buffer strings.Builder
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(r io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(r),
	}
}

// ParseNext parses the next event from the stream
func (p *SSEParser) ParseNext() (*Event, error) {
	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read line: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for end of stream
			if data == "[DONE]" {
				return nil, io.EOF
			}

			// Parse JSON
			var event Event
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				// Try to parse as raw event
				var rawEvent map[string]interface{}
				if parseErr := json.Unmarshal([]byte(data), &rawEvent); parseErr != nil {
					continue // Skip unparseable events
				}

				// Convert raw event to Event
				if eventType, ok := rawEvent["type"].(string); ok {
					event.Type = EventType(eventType)
				}
			}

			return &event, nil
		}

		// Parse event type line
		if strings.HasPrefix(line, "event: ") {
			// Store event type for next data
			continue
		}
	}
}

// ParseAll parses all events from the stream
func (p *SSEParser) ParseAll() ([]Event, error) {
	var events []Event
	for {
		event, err := p.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return events, err
		}
		events = append(events, *event)
	}
	return events, nil
}

// StreamProcessor processes streaming events and dispatches to handlers
type StreamProcessor struct {
	mu         sync.RWMutex
	handlers   map[EventType][]EventHandler
	buffer     strings.Builder
	blocks     map[int]*ContentBlockBuilder
	finalMsg   *MessageData
}

// EventHandler is a function that handles a streaming event
type EventHandler func(event *Event) error

// ContentBlockBuilder builds content blocks from deltas
type ContentBlockBuilder struct {
	Type   string
	Text   strings.Builder
	JSON   strings.Builder
	ID     string
	Name   string
	Input  map[string]interface{}
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		handlers: make(map[EventType][]EventHandler),
		blocks:   make(map[int]*ContentBlockBuilder),
	}
}

// RegisterHandler registers a handler for an event type
func (p *StreamProcessor) RegisterHandler(eventType EventType, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventType] = append(p.handlers[eventType], handler)
}

// Process processes a single event
func (p *StreamProcessor) Process(event *Event) error {
	// Dispatch to registered handlers
	p.mu.RLock()
	handlers := p.handlers[event.Type]
	p.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			return err
		}
	}

	// Built-in processing
	switch event.Type {
	case EventTypeMessageStart:
		p.finalMsg = event.Message

	case EventTypeContentBlockStart:
		if event.ContentBlock != nil {
			p.blocks[event.Index] = &ContentBlockBuilder{
				Type: event.ContentBlock.Type,
				ID:   event.ContentBlock.ID,
				Name: event.ContentBlock.Name,
				Text: strings.Builder{},
				JSON: strings.Builder{},
			}
		}

	case EventTypeContentBlockDelta:
		if block, ok := p.blocks[event.Index]; ok && event.Delta != nil {
			switch event.Delta.Type {
			case "text_delta":
				block.Text.WriteString(event.Delta.Text)
			case "input_json_delta":
				block.JSON.WriteString(event.Delta.PartialJSON)
			}
		}

	case EventTypeContentBlockStop:
		// Block is complete

	case EventTypeMessageDelta:
		if p.finalMsg != nil && event.Usage != nil {
			p.finalMsg.Usage = event.Usage
		}
		if event.Delta != nil && event.Delta.StopReason != "" && p.finalMsg != nil {
			p.finalMsg.StopReason = event.Delta.StopReason
		}

	case EventTypeMessageStop:
		// Message is complete

	case EventTypeError:
		if event.Error != nil {
			return fmt.Errorf("stream error: %s: %s", event.Error.Type, event.Error.Message)
		}
	}

	return nil
}

// GetContent returns the accumulated content for a block
func (p *StreamProcessor) GetContent(index int) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if block, ok := p.blocks[index]; ok {
		return block.Text.String()
	}
	return ""
}

// GetAllContent returns all accumulated text content
func (p *StreamProcessor) GetAllContent() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result strings.Builder
	for i := 0; i < len(p.blocks); i++ {
		if block, ok := p.blocks[i]; ok {
			result.WriteString(block.Text.String())
		}
	}
	return result.String()
}

// GetFinalMessage returns the final message
func (p *StreamProcessor) GetFinalMessage() *MessageData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.finalMsg
}

// GetContentBlocks returns the built content blocks
func (p *StreamProcessor) GetContentBlocks() []ContentBlockData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	blocks := make([]ContentBlockData, len(p.blocks))
	for i, block := range p.blocks {
		blocks[i] = ContentBlockData{
			Type: block.Type,
			Text: block.Text.String(),
			ID:   block.ID,
			Name: block.Name,
		}
	}
	return blocks
}

// Reset resets the processor for a new stream
func (p *StreamProcessor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.buffer.Reset()
	p.blocks = make(map[int]*ContentBlockBuilder)
	p.finalMsg = nil
}

// OpenAIStreamEvent represents an OpenAI streaming event
type OpenAIStreamEvent struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
}

// OpenAIChoice represents a choice in OpenAI stream
type OpenAIChoice struct {
	Index        int          `json:"index"`
	Delta        *OpenAIDelta `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// OpenAIDelta represents a delta in OpenAI stream
type OpenAIDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// OpenAIStreamProcessor processes OpenAI-format streaming events
type OpenAIStreamProcessor struct {
	mu       sync.RWMutex
	content  strings.Builder
	role     string
	finished bool
	reason   string
}

// NewOpenAIStreamProcessor creates a new OpenAI stream processor
func NewOpenAIStreamProcessor() *OpenAIStreamProcessor {
	return &OpenAIStreamProcessor{}
}

// Process processes an OpenAI streaming event
func (p *OpenAIStreamProcessor) Process(event *OpenAIStreamEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, choice := range event.Choices {
		if choice.Delta != nil {
			if choice.Delta.Role != "" {
				p.role = choice.Delta.Role
			}
			if choice.Delta.Content != "" {
				p.content.WriteString(choice.Delta.Content)
			}
		}
		if choice.FinishReason != "" {
			p.finished = true
			p.reason = choice.FinishReason
		}
	}

	return nil
}

// GetContent returns accumulated content
func (p *OpenAIStreamProcessor) GetContent() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.content.String()
}

// GetRole returns the role
func (p *OpenAIStreamProcessor) GetRole() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.role
}

// IsFinished returns whether the stream is finished
func (p *OpenAIStreamProcessor) IsFinished() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.finished
}

// GetFinishReason returns the finish reason
func (p *OpenAIStreamProcessor) GetFinishReason() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.reason
}

// Reset resets the processor
func (p *OpenAIStreamProcessor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.content.Reset()
	p.role = ""
	p.finished = false
	p.reason = ""
}
