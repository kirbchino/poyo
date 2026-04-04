package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHarness provides a test harness for TUI components
type TestHarness struct {
	model Model
	t     *testing.T
}

// NewTestHarness creates a new test harness
func NewTestHarness(t *testing.T) *TestHarness {
	theme := DefaultTheme()
	model := NewModel(theme)
	model.width = 80
	model.height = 24

	// Initialize child components with dimensions
	model.messages.SetSize(60, 19)
	model.input.SetWidth(80)
	model.tools.SetSize(20, 19)

	return &TestHarness{
		model: model,
		t:     t,
	}
}

// SetOnMessage sets the message handler
func (h *TestHarness) SetOnMessage(handler func(content string) (string, error)) {
	h.model.onMessage = handler
}

// SendWindowSize simulates a window size event
func (h *TestHarness) SendWindowSize(width, height int) tea.Msg {
	h.model.width = width
	h.model.height = height
	h.model = h.model.updateLayout()
	return tea.WindowSizeMsg{Width: width, Height: height}
}

// SendUserMessage simulates a user message
func (h *TestHarness) SendUserMessage(content string) (Model, tea.Cmd) {
	msg := UserMessageMsg{Content: content}
	newModel, cmd := h.model.Update(msg)
	h.model = newModel.(Model)
	return h.model, cmd
}

// GetMessages returns the current messages
func (h *TestHarness) GetMessages() []MessageDisplay {
	return h.model.messages.messages
}

// GetMessageCount returns the number of messages
func (h *TestHarness) GetMessageCount() int {
	return len(h.model.messages.messages)
}

// Render renders the current model view
func (h *TestHarness) Render() string {
	return h.model.View()
}

// RenderMessages renders just the messages component
func (h *TestHarness) RenderMessages() string {
	return h.model.messages.View()
}

// AssertMessageCount asserts the number of messages
func (h *TestHarness) AssertMessageCount(expected int) {
	actual := h.GetMessageCount()
	if actual != expected {
		h.t.Helper()
		h.t.Errorf("Expected %d messages, got %d", expected, actual)
	}
}

// AssertLastMessage asserts the last message content
func (h *TestHarness) AssertLastMessage(role, content string) {
	messages := h.GetMessages()
	if len(messages) == 0 {
		h.t.Helper()
		h.t.Error("No messages to assert")
		return
	}

	last := messages[len(messages)-1]
	if last.Role != role {
		h.t.Helper()
		h.t.Errorf("Expected role '%s', got '%s'", role, last.Role)
	}
	if last.Content != content {
		h.t.Helper()
		h.t.Errorf("Expected content '%s', got '%s'", content, last.Content)
	}
}

// AssertContains asserts that the view contains a string
func (h *TestHarness) AssertContains(substr string) {
	view := h.Render()
	if !strings.Contains(view, substr) {
		h.t.Helper()
		h.t.Errorf("View does not contain '%s'\nActual view:\n%s", substr, view)
	}
}

// AssertMessagesContain asserts that the messages view contains a string
func (h *TestHarness) AssertMessagesContain(substr string) {
	view := h.RenderMessages()
	if !strings.Contains(view, substr) {
		h.t.Helper()
		h.t.Errorf("Messages view does not contain '%s'\nActual view:\n%s", substr, view)
	}
}

// Debug prints debug info about the current state
func (h *TestHarness) Debug() {
	fmt.Printf("=== TUI Debug ===\n")
	fmt.Printf("Model dimensions: %dx%d\n", h.model.width, h.model.height)
	fmt.Printf("Messages dimensions: %dx%d\n", h.model.messages.width, h.model.messages.height)
	fmt.Printf("Message count: %d\n", h.GetMessageCount())
	for i, msg := range h.GetMessages() {
		fmt.Printf("  [%d] %s: %q\n", i, msg.Role, truncate(msg.Content, 50))
	}
	fmt.Printf("=================\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
