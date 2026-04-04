// Package tui implements streaming display handler
package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
)

// StreamHandler handles streaming responses and TUI updates
type StreamHandler struct {
	mu          sync.Mutex
	program     *tea.Program
	currentText strings.Builder
	toolName    string
	toolID      string
	isToolUse   bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(program *tea.Program) *StreamHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamHandler{
		program: program,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// HandleStream processes an SSE stream and sends updates to the TUI
func (h *StreamHandler) HandleStream(reader io.ReadCloser) error {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	var eventType string
	var eventData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for context cancellation
		select {
		case <-h.ctx.Done():
			return h.ctx.Err()
		default:
		}

		// Parse SSE format
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			eventData.Reset()
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			eventData.WriteString(data)

			// Process complete event
			if err := h.processEvent(eventType, eventData.String()); err != nil {
				return err
			}
			eventData.Reset()
		} else if line == "" {
			// Empty line marks end of event
			if eventData.Len() > 0 {
				if err := h.processEvent(eventType, eventData.String()); err != nil {
					return err
				}
				eventData.Reset()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

// processEvent processes a single SSE event
func (h *StreamHandler) processEvent(eventType, data string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch eventType {
	case "content_block_start":
		return h.handleContentBlockStart(data)

	case "content_block_delta":
		return h.handleContentBlockDelta(data)

	case "content_block_stop":
		return h.handleContentBlockStop()

	case "message_start":
		h.currentText.Reset()
		h.program.Send(ProcessingStartMsg{})

	case "message_delta":
		// Handle message-level deltas (stop_reason, usage, etc.)
		h.handleMessageDelta(data)

	case "message_stop":
		h.program.Send(ProcessingEndMsg{})

	case "error":
		return fmt.Errorf("stream error: %s", data)
	}

	return nil
}

// handleContentBlockStart handles content_block_start events
func (h *StreamHandler) handleContentBlockStart(data string) error {
	// Parse the JSON to determine content block type
	if strings.Contains(data, `"type":"tool_use"`) {
		h.isToolUse = true
		// Extract tool name and ID from data
		// Simple parsing - in production use proper JSON parsing
		if idx := strings.Index(data, `"name":"`); idx != -1 {
			start := idx + 8
			end := strings.Index(data[start:], `"`)
			if end != -1 {
				h.toolName = data[start : start+end]
			}
		}
		if idx := strings.Index(data, `"id":"`); idx != -1 {
			start := idx + 6
			end := strings.Index(data[start:], `"`)
			if end != -1 {
				h.toolID = data[start : start+end]
			}
		}
		h.program.Send(ToolStartMsg{
			ToolID:   h.toolID,
			ToolName: h.toolName,
		})
	} else {
		h.isToolUse = false
	}
	h.currentText.Reset()
	return nil
}

// handleContentBlockDelta handles content_block_delta events
func (h *StreamHandler) handleContentBlockDelta(data string) error {
	// Extract text from delta
	// Simple parsing - look for "text" field
	if idx := strings.Index(data, `"text":"`); idx != -1 {
		start := idx + 8
		// Find end of text (handle escaped quotes)
		text := extractJSONString(data[start:])
		if text != "" {
			h.currentText.WriteString(text)
			h.program.Send(StreamChunkMsg{
				Content: h.currentText.String(),
				Done:    false,
			})
		}
	}

	// Handle tool input deltas
	if h.isToolUse && strings.Contains(data, `"type":"input_json_delta"`) {
		if idx := strings.Index(data, `"partial_json":"`); idx != -1 {
			start := idx + 16
			partialJSON := extractJSONString(data[start:])
			h.program.Send(ToolProgressMsg{
				ToolID: h.toolID,
				Output: partialJSON,
			})
		}
	}

	return nil
}

// handleContentBlockStop handles content_block_stop events
func (h *StreamHandler) handleContentBlockStop() error {
	if h.isToolUse {
		h.program.Send(ToolEndMsg{
			ToolID: h.toolID,
			Output: h.currentText.String(),
		})
	} else {
		h.program.Send(AssistantMessageMsg{
			Content: h.currentText.String(),
		})
	}
	h.isToolUse = false
	h.currentText.Reset()
	return nil
}

// handleMessageDelta handles message_delta events
func (h *StreamHandler) handleMessageDelta(data string) {
	// Extract usage information if present
	if strings.Contains(data, `"usage"`) {
		// Parse usage stats
		// Simple extraction - in production use proper JSON parsing
		var inputTokens, outputTokens int
		if idx := strings.Index(data, `"input_tokens":`); idx != -1 {
			fmt.Sscanf(data[idx:], `"input_tokens":%d`, &inputTokens)
		}
		if idx := strings.Index(data, `"output_tokens":`); idx != -1 {
			fmt.Sscanf(data[idx:], `"output_tokens":%d`, &outputTokens)
		}
		h.program.Send(UsageUpdateMsg{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		})
	}
}

// Cancel cancels the stream handling
func (h *StreamHandler) Cancel() {
	h.cancel()
}

// extractJSONString extracts a JSON string value, handling escape sequences
func extractJSONString(s string) string {
	if len(s) == 0 || s[0] != '"' {
		return ""
	}

	var result strings.Builder
	escaped := false

	for i := 1; i < len(s); i++ {
		if escaped {
			switch s[i] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '"':
				result.WriteByte('"')
			case '\\':
				result.WriteByte('\\')
			default:
				result.WriteByte(s[i])
			}
			escaped = false
		} else if s[i] == '\\' {
			escaped = true
		} else if s[i] == '"' {
			break
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// UsageUpdateMsg represents a token usage update
type UsageUpdateMsg struct {
	InputTokens  int
	OutputTokens int
}

// StreamingProgressModel is a helper model for showing streaming progress
type StreamingProgressModel struct {
	text      strings.Builder
	startTime time.Time
	speed     int // chars per second
	done      bool
}

// NewStreamingProgressModel creates a new streaming progress model
func NewStreamingProgressModel() *StreamingProgressModel {
	return &StreamingProgressModel{
		startTime: time.Now(),
	}
}

// Append adds text to the streaming progress
func (m *StreamingProgressModel) Append(text string) {
	m.text.WriteString(text)

	// Calculate speed
	elapsed := time.Since(m.startTime).Seconds()
	if elapsed > 0 {
		m.speed = int(float64(m.text.Len()) / elapsed)
	}
}

// Complete marks the streaming as complete
func (m *StreamingProgressModel) Complete() {
	m.done = true
}

// GetText returns the current text
func (m *StreamingProgressModel) GetText() string {
	return m.text.String()
}

// GetSpeed returns the streaming speed in chars per second
func (m *StreamingProgressModel) GetSpeed() int {
	return m.speed
}

// GetElapsed returns the elapsed time
func (m *StreamingProgressModel) GetElapsed() time.Duration {
	return time.Since(m.startTime)
}

// StreamingTextRenderer renders streaming text with typewriter effect
type StreamingTextRenderer struct {
	fullText   string
	rendered   int
	speed      int // characters per update
	interval   time.Duration
}

// NewStreamingTextRenderer creates a new streaming text renderer
func NewStreamingTextRenderer(text string) *StreamingTextRenderer {
	return &StreamingTextRenderer{
		fullText: text,
		speed:    5,
		interval: 50 * time.Millisecond,
	}
}

// Next returns the next chunk of text to render
func (r *StreamingTextRenderer) Next() string {
	if r.rendered >= len(r.fullText) {
		return ""
	}

	end := r.rendered + r.speed
	if end > len(r.fullText) {
		end = len(r.fullText)
	}

	chunk := r.fullText[r.rendered:end]
	r.rendered = end
	return chunk
}

// Done returns true if all text has been rendered
func (r *StreamingTextRenderer) Done() bool {
	return r.rendered >= len(r.fullText)
}

// SetSpeed sets the rendering speed
func (r *StreamingTextRenderer) SetSpeed(charsPerUpdate int) {
	r.speed = charsPerUpdate
}

// SetInterval sets the update interval
func (r *StreamingTextRenderer) SetInterval(interval time.Duration) {
	r.interval = interval
}
