// Package tui implements the terminal user interface using Bubble Tea
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main TUI model
type Model struct {
	// State
	state ModelState

	// Components
	input    InputModel
	messages MessageListModel
	tools    ToolPanelModel
	status   StatusBarModel

	// Layout
	width  int
	height int

	// Theme
	theme *Theme

	// Messages and content
	prompt      string
	conversation []MessageItem

	// Tool execution
	activeTools []ToolExecution

	// Errors
	lastError error

	// Message handler
	onMessage func(content string) (string, error)
}

// ModelState represents the current state of the TUI
type ModelState int

const (
	StateIdle ModelState = iota
	StateProcessing
	StateWaitingInput
	StateToolExecution
	StateError
)

// MessageItem represents a single message in the conversation
type MessageItem struct {
	Role      string    // "user", "assistant", "system", "tool"
	Content   string    // Text content
	Timestamp time.Time // When the message was created
	ToolName  string    // For tool messages
	ToolID    string    // Tool use ID
	IsError   bool      // Whether this is an error message
}

// ToolExecution represents an active tool execution
type ToolExecution struct {
	ID        string
	Name      string
	Status    string // "running", "completed", "error"
	StartTime time.Time
	Output    string
}

// NewModel creates a new TUI model
func NewModel() Model {
	theme := DefaultTheme()

	return Model{
		state:       StateIdle,
		input:       NewInputModel(theme),
		messages:    NewMessageListModel(theme),
		tools:       NewToolPanelModel(theme),
		status:      NewStatusBarModel(theme),
		theme:       theme,
		conversation: make([]MessageItem, 0),
		activeTools: make([]ToolExecution, 0),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		m.messages.Init(),
		m.tools.Init(),
		m.status.Init(),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fmt.Fprintf(os.Stderr, "[DEBUG] WindowSizeMsg: %dx%d\n", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		m = m.updateLayout()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlL:
			// Clear screen
			return m, tea.ClearScreen
		}

	case ErrorMessage:
		m.lastError = msg.Err
		m.state = StateError
		m.status.SetError(msg.Err.Error())

	case ProcessingStartMsg:
		m.state = StateProcessing
		m.status.SetProcessing(true)

	case ProcessingEndMsg:
		m.state = StateIdle
		m.status.SetProcessing(false)

	case UserMessageMsg:
		// Debug: print received message
		fmt.Fprintf(os.Stderr, "[DEBUG] UserMessageMsg received: '%s'\n", msg.Content)

		m.conversation = append(m.conversation, MessageItem{
			Role:      "user",
			Content:   msg.Content,
			Timestamp: time.Now(),
		})
		m.messages.AddMessage(msg.Content, "user")

		// Debug: check message count
		fmt.Fprintf(os.Stderr, "[DEBUG] Message count: %d\n", m.messages.GetMessageCount())

		// Ensure layout is updated (critical if no WindowSizeMsg received yet)
		if m.width > 0 && m.height > 0 {
			m = m.updateLayout()
		}

		// Call message handler if set
		if m.onMessage != nil {
			m.state = StateProcessing
			m.status.SetProcessing(true)

			// Call handler synchronously (for now)
			response, err := m.onMessage(msg.Content)
			if err != nil {
				m.lastError = err
				m.state = StateError
				m.status.SetError(err.Error())
			} else {
				// Add assistant response
				m.conversation = append(m.conversation, MessageItem{
					Role:      "assistant",
					Content:   response,
					Timestamp: time.Now(),
				})
				m.messages.AddMessage(response, "assistant")
				m.state = StateIdle
				m.status.SetProcessing(false)

				// Update layout after adding assistant response
				if m.width > 0 && m.height > 0 {
					m = m.updateLayout()
				}
			}
		}

	case AssistantMessageMsg:
		m.conversation = append(m.conversation, MessageItem{
			Role:      "assistant",
			Content:   msg.Content,
			Timestamp: time.Now(),
		})
		m.messages.AddMessage(msg.Content, "assistant")

	case ToolStartMsg:
		exec := ToolExecution{
			ID:        msg.ToolID,
			Name:      msg.ToolName,
			Status:    "running",
			StartTime: time.Now(),
		}
		m.activeTools = append(m.activeTools, exec)
		m.tools.AddExecution(exec)
		m.state = StateToolExecution

	case ToolProgressMsg:
		for i := range m.activeTools {
			if m.activeTools[i].ID == msg.ToolID {
				m.activeTools[i].Output = msg.Output
			}
		}
		m.tools.UpdateExecution(msg.ToolID, msg.Output, "")

	case ToolEndMsg:
		for i := range m.activeTools {
			if m.activeTools[i].ID == msg.ToolID {
				m.activeTools[i].Status = "completed"
				m.activeTools[i].Output = msg.Output
			}
		}
		m.tools.CompleteExecution(msg.ToolID, msg.Output)
		m.state = StateIdle
	}

	// Update child components
	var cmd tea.Cmd

	// Note: We need to be careful about the order here.
	// m.messages was already modified above (AddMessage), so we save those changes
	// before calling Update() on child components.
	savedMessages := m.messages

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// Restore saved messages before calling Update
	m.messages = savedMessages
	m.messages, cmd = m.messages.Update(msg)
	cmds = append(cmds, cmd)

	m.tools, cmd = m.tools.Update(msg)
	cmds = append(cmds, cmd)

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	fmt.Fprintf(os.Stderr, "[DEBUG] Model.View() width=%d height=%d\n", m.width, m.height)

	if m.width == 0 {
		return "Initializing..."
	}

	// Calculate component heights
	statusHeight := 1
	inputHeight := 3
	toolsWidth := m.width / 4
	messagesWidth := m.width - toolsWidth - 2

	// Render components
	statusView := m.status.View()
	messagesView := m.messages.View()
	toolsView := m.tools.View()
	inputView := m.input.View()

	// Layout: messages and tools side by side
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().
			Width(messagesWidth).
			Height(m.height-statusHeight-inputHeight-2).
			Render(messagesView),
		lipgloss.NewStyle().
			Width(toolsWidth).
			Height(m.height-statusHeight-inputHeight-2).
			Render(toolsView),
	)

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Width(m.width).Render(statusView),
		lipgloss.NewStyle().Width(m.width).Render(mainContent),
		lipgloss.NewStyle().Width(m.width).Render(inputView),
	)
}

// updateLayout recalculates layout based on window size
func (m Model) updateLayout() Model {
	fmt.Fprintf(os.Stderr, "[DEBUG] updateLayout() model.width=%d model.height=%d\n", m.width, m.height)
	// Update child component sizes
	m.messages.SetSize(m.width*3/4, m.height-5)
	m.tools.SetSize(m.width/4, m.height-5)
	m.input.SetWidth(m.width)
	return m
}

// AddUserMessage adds a user message to the conversation
func (m *Model) AddUserMessage(content string) {
	m.conversation = append(m.conversation, MessageItem{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	})
	m.messages.AddMessage(content, "user")
}

// AddAssistantMessage adds an assistant message to the conversation
func (m *Model) AddAssistantMessage(content string) {
	m.conversation = append(m.conversation, MessageItem{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	})
	m.messages.AddMessage(content, "assistant")
}

// AddToolMessage adds a tool result message
func (m *Model) AddToolMessage(toolName, toolID, content string, isError bool) {
	m.conversation = append(m.conversation, MessageItem{
		Role:     "tool",
		Content:  content,
		ToolName: toolName,
		ToolID:   toolID,
		IsError:  isError,
	})
}

// SetProcessing sets the processing state
func (m *Model) SetProcessing(processing bool) {
	if processing {
		m.state = StateProcessing
		m.status.SetProcessing(true)
	} else {
		m.state = StateIdle
		m.status.SetProcessing(false)
	}
}

// SetError sets an error state
func (m *Model) SetError(err error) {
	m.lastError = err
	m.state = StateError
	m.status.SetError(err.Error())
}

// ClearError clears the error state
func (m *Model) ClearError() {
	m.lastError = nil
	m.state = StateIdle
	m.status.ClearError()
}

// GetConversation returns the conversation history
func (m *Model) GetConversation() []MessageItem {
	return m.conversation
}

// Theme styles
type Theme struct {
	Colors Colors
	Styles Styles
}

// Colors defines the color palette
type Colors struct {
	Primary    string
	Secondary  string
	Accent     string
	Background string
	Text       string
	Muted      string
	Error      string
	Success    string
	Warning    string
	UserMsg    string
	Assistant  string
	Tool       string
}

// Styles defines the styles for components
type Styles struct {
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	ToolMessage      lipgloss.Style
	Error            lipgloss.Style
	StatusBar        lipgloss.Style
	Input            lipgloss.Style
	Border           lipgloss.Style
	Title            lipgloss.Style
}

// DefaultTheme returns the default theme
func DefaultTheme() *Theme {
	colors := Colors{
		Primary:    "#7C3AED",
		Secondary:  "#4F46E5",
		Accent:     "#06B6D4",
		Background: "#1E1E2E",
		Text:       "#CDD6F4",
		Muted:      "#6C7086",
		Error:      "#F38BA8",
		Success:    "#A6E3A1",
		Warning:    "#FAB387",
		UserMsg:    "#89B4FA",
		Assistant:  "#CBA6F7",
		Tool:       "#94E2D5",
	}

	return &Theme{
		Colors: colors,
		Styles: Styles{
			UserMessage: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.UserMsg)).
				Padding(0, 1),
			AssistantMessage: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Assistant)).
				Padding(0, 1),
			ToolMessage: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Tool)).
				Padding(0, 1),
			Error: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Error)).
				Bold(true),
			StatusBar: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Text)).
				Background(lipgloss.Color(colors.Background)),
			Input: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Text)).
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(colors.Muted)),
			Border: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(colors.Primary)),
			Title: lipgloss.NewStyle().
				Foreground(lipgloss.Color(colors.Primary)).
				Bold(true),
		},
	}
}

// Message types for Bubble Tea
type (
	// ErrorMessage represents an error
	ErrorMessage struct {
		Err error
	}

	// ProcessingStartMsg signals processing started
	ProcessingStartMsg struct{}

	// ProcessingEndMsg signals processing ended
	ProcessingEndMsg struct{}

	// UserMessageMsg represents a user message
	UserMessageMsg struct {
		Content string
	}

	// AssistantMessageMsg represents an assistant message
	AssistantMessageMsg struct {
		Content string
	}

	// ToolStartMsg signals a tool started
	ToolStartMsg struct {
		ToolID   string
		ToolName string
	}

	// ToolProgressMsg signals tool progress
	ToolProgressMsg struct {
		ToolID string
		Output string
	}

	// ToolEndMsg signals a tool finished
	ToolEndMsg struct {
		ToolID string
		Output string
	}

	// StreamChunkMsg represents a streaming chunk
	StreamChunkMsg struct {
		Content string
		Done    bool
	}
)

// Helper function to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// Helper to wrap text at a given width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		for len(line) > width {
			// Find a good break point
			breakPoint := width
			for i := width - 1; i > 0; i-- {
				if line[i] == ' ' {
					breakPoint = i
					break
				}
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimPrefix(line[breakPoint:], " ")
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}
