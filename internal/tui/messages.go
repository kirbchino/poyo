// Package tui implements message list component
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessageListModel handles the message display
type MessageListModel struct {
	// Content
	messages []MessageDisplay
	selected int
	offset   int

	// Dimensions
	width  int
	height int

	// Styling
	theme  *Theme
	styles MessageStyles
}

// MessageDisplay represents a message for display
type MessageDisplay struct {
	Role      string
	Content   string
	Timestamp time.Time
	ToolName  string
	IsError   bool
	IsFolded  bool
}

// MessageStyles holds styles for different message types
type MessageStyles struct {
	User      lipgloss.Style
	Assistant lipgloss.Style
	System    lipgloss.Style
	Tool      lipgloss.Style
	Error     lipgloss.Style
	Timestamp lipgloss.Style
	Border    lipgloss.Style
}

// NewMessageListModel creates a new message list model
func NewMessageListModel(theme *Theme) MessageListModel {
	return MessageListModel{
		messages: make([]MessageDisplay, 0),
		selected: -1,
		offset:   0,
		theme:    theme,
		styles: MessageStyles{
			User: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.UserMsg)).
				Padding(0, 1).
				Bold(true),
			Assistant: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Assistant)).
				Padding(0, 1),
			System: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Muted)).
				Padding(0, 1).
				Italic(true),
			Tool: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Tool)).
				Padding(0, 1),
			Error: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Error)).
				Padding(0, 1).
				Bold(true),
			Timestamp: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Muted)),
			Border: lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(lipgloss.Color(theme.Colors.Primary)).
				PaddingLeft(1),
		},
	}
}

// Init initializes the message list model
func (m MessageListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the message list model
func (m MessageListModel) Update(msg tea.Msg) (MessageListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
				m.ensureVisible()
			} else if m.selected < 0 && len(m.messages) > 0 {
				m.selected = len(m.messages) - 1
				m.ensureVisible()
			}

		case tea.KeyDown:
			if m.selected >= 0 && m.selected < len(m.messages)-1 {
				m.selected++
				m.ensureVisible()
			}

		case tea.KeyPgUp:
			m.offset -= m.height - 2
			if m.offset < 0 {
				m.offset = 0
			}

		case tea.KeyPgDown:
			maxOffset := m.calculateMaxOffset()
			m.offset += m.height - 2
			if m.offset > maxOffset {
				m.offset = maxOffset
			}

		case tea.KeyTab:
			// Toggle fold/unfold
			if m.selected >= 0 && m.selected < len(m.messages) {
				m.messages[m.selected].IsFolded = !m.messages[m.selected].IsFolded
			}
		}

	case tea.MouseMsg:
		// Handle mouse scroll (if supported)
	}
	return m, nil
}

// View renders the message list
func (m MessageListModel) View() string {
	// Debug: print dimensions
	fmt.Fprintf(os.Stderr, "[DEBUG] MessageListModel.View() width=%d height=%d msg_count=%d\n", m.width, m.height, len(m.messages))

	if m.width <= 0 || m.height <= 0 {
		return "Waiting for terminal size..."
	}

	if len(m.messages) == 0 {
		return m.renderEmpty()
	}

	var lines []string
	currentHeight := 0

	for i := m.offset; i < len(m.messages) && currentHeight < m.height-2; i++ {
		msg := m.messages[i]
		// Debug: print message being rendered
		fmt.Fprintf(os.Stderr, "[DEBUG] Rendering message %d: role=%s content_len=%d\n", i, msg.Role, len(msg.Content))
		msgLines := m.renderMessage(msg, i == m.selected)
		lines = append(lines, msgLines...)
		currentHeight += len(msgLines)
	}

	// Add scroll indicators
	if m.offset > 0 {
		lines = append([]string{m.styles.Timestamp.Render("↑ more messages above")}, lines...)
	}

	if m.offset < m.calculateMaxOffset() {
		lines = append(lines, m.styles.Timestamp.Render("↓ more messages below"))
	}

	// Pad to height
	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Wrap in border
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderMessage renders a single message
func (m MessageListModel) renderMessage(msg MessageDisplay, selected bool) []string {
	fmt.Fprintf(os.Stderr, "[DEBUG] renderMessage: role=%s content='%s' width=%d\n", msg.Role, truncate(msg.Content, 50), m.width)
	var lines []string

	// Timestamp and role header
	timestamp := msg.Timestamp.Format("15:04:05")
	header := fmt.Sprintf("[%s] %s", timestamp, m.formatRole(msg.Role))
	if msg.ToolName != "" {
		header += fmt.Sprintf(" (%s)", msg.ToolName)
	}
	if msg.IsError {
		header = m.styles.Error.Render(header + " ERROR")
	} else {
		header = m.styles.Timestamp.Render(header)
	}
	lines = append(lines, header)

	// Content
	content := msg.Content
	if msg.IsFolded && len(content) > 100 {
		content = content[:100] + "..."
	} else {
		content = wrapText(content, m.width-4)
	}

	// Style based on role
	var styledContent string
	switch msg.Role {
	case "user":
		styledContent = m.styles.User.Render(content)
	case "assistant":
		styledContent = m.styles.Assistant.Render(content)
	case "system":
		styledContent = m.styles.System.Render(content)
	case "tool":
		styledContent = m.styles.Tool.Render(content)
	default:
		styledContent = content
	}

	// Add selection indicator
	if selected {
		styledContent = "▶ " + styledContent
	} else {
		styledContent = "  " + styledContent
	}

	contentLines := strings.Split(styledContent, "\n")
	lines = append(lines, contentLines...)

	// Add separator
	lines = append(lines, m.styles.Timestamp.Render(strings.Repeat("─", m.width-4)))

	return lines
}

// formatRole formats the role for display
func (m MessageListModel) formatRole(role string) string {
	switch role {
	case "user":
		return "👤 You"
	case "assistant":
		return "💚 Poyo"
	case "system":
		return "⚙️  System"
	case "tool":
		return "🔧 Tool"
	default:
		return role
	}
}

// renderEmpty renders empty state
func (m MessageListModel) renderEmpty() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Muted)).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return emptyStyle.Render(
		"No messages yet.\nStart a conversation by typing a message below.",
	)
}

// ensureVisible ensures the selected message is visible
func (m *MessageListModel) ensureVisible() {
	if m.selected < m.offset {
		m.offset = m.selected
	} else {
		// Calculate height of selected message
		selectedHeight := m.getMessageHeight(m.selected)
		visibleHeight := 0
		for i := m.offset; i < m.selected; i++ {
			visibleHeight += m.getMessageHeight(i)
		}

		for visibleHeight+selectedHeight > m.height-2 && m.offset < m.selected {
			visibleHeight -= m.getMessageHeight(m.offset)
			m.offset++
		}
	}
}

// getMessageHeight calculates the display height of a message
func (m MessageListModel) getMessageHeight(index int) int {
	if index < 0 || index >= len(m.messages) {
		return 0
	}

	msg := m.messages[index]
	height := 2 // Header + separator

	content := msg.Content
	if msg.IsFolded && len(content) > 100 {
		content = content[:100] + "..."
	} else {
		content = wrapText(content, m.width-4)
	}

	height += len(strings.Split(content, "\n"))
	return height
}

// calculateMaxOffset calculates the maximum scroll offset
func (m MessageListModel) calculateMaxOffset() int {
	if len(m.messages) == 0 {
		return 0
	}

	// If height not set yet, return 0 (show from beginning)
	if m.height <= 2 {
		return 0
	}

	totalHeight := 0
	for i := range m.messages {
		totalHeight += m.getMessageHeight(i)
	}

	maxOffset := len(m.messages) - 1
	for totalHeight > m.height-2 && maxOffset > 0 {
		totalHeight -= m.getMessageHeight(maxOffset)
		maxOffset--
	}

	return maxOffset
}

// AddMessage adds a message to the list
func (m *MessageListModel) AddMessage(content, role string) {
	// Skip empty or whitespace-only messages
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] AddMessage: skipping empty message (role=%s)\n", role)
		return
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] AddMessage: role=%s content_len=%d\n", role, len(content))
	m.messages = append(m.messages, MessageDisplay{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Auto-scroll to bottom
	oldOffset := m.offset
	m.offset = m.calculateMaxOffset()
	fmt.Fprintf(os.Stderr, "[DEBUG] AddMessage: offset changed from %d to %d, total messages=%d\n", oldOffset, m.offset, len(m.messages))
}

// AddToolMessage adds a tool message
func (m *MessageListModel) AddToolMessage(toolName, content string, isError bool) {
	m.messages = append(m.messages, MessageDisplay{
		Role:      "tool",
		Content:   content,
		Timestamp: time.Now(),
		ToolName:  toolName,
		IsError:   isError,
	})

	// Auto-scroll to bottom
	m.offset = m.calculateMaxOffset()
}

// SetSize sets the dimensions
func (m *MessageListModel) SetSize(width, height int) {
	fmt.Fprintf(os.Stderr, "[DEBUG] MessageListModel.SetSize(%d, %d)\n", width, height)
	m.width = width
	m.height = height
}

// Clear clears all messages
func (m *MessageListModel) Clear() {
	m.messages = make([]MessageDisplay, 0)
	m.selected = -1
	m.offset = 0
}

// GetMessageCount returns the number of messages
func (m *MessageListModel) GetMessageCount() int {
	return len(m.messages)
}

// GetSelectedMessage returns the currently selected message
func (m *MessageListModel) GetSelectedMessage() *MessageDisplay {
	if m.selected >= 0 && m.selected < len(m.messages) {
		return &m.messages[m.selected]
	}
	return nil
}
