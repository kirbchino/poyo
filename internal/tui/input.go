// Package tui implements input component
package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel handles user input
type InputModel struct {
	// Content
	value    string
	cursor   int
	viewport int

	// State
	focused  bool
	editing  bool
	history  []string
	histIdx  int

	// Dimensions
	width    int
	height   int
	maxChars int

	// Styling
	theme    *Theme
	style    lipgloss.Style
	prompt   string
}

// NewInputModel creates a new input model
func NewInputModel(theme *Theme) InputModel {
	return InputModel{
		theme:    theme,
		focused:  true,
		editing:  false,
		history:  make([]string, 0),
		histIdx:  -1,
		width:    80,
		height:   3,
		maxChars: 4000,
		prompt:   "> ",
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Text)).
			Background(lipgloss.Color(theme.Colors.Background)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)).
			Padding(0, 1),
	}
}

// Init initializes the input model
func (m InputModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the input model
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Check for non-empty content (trim whitespace)
			if strings.TrimSpace(m.value) != "" {
				// Save the value before clearing
				content := m.value

				// Debug: print content being sent
				fmt.Fprintf(os.Stderr, "[DEBUG] Input sending: '%s'\n", content)

				// Save to history
				m.history = append(m.history, content)
				m.histIdx = len(m.history)

				// Return submit command with saved content
				cmd = func() tea.Msg {
					return UserMessageMsg{Content: content}
				}
				m.value = ""
				m.cursor = 0
				m.viewport = 0
			}

		case tea.KeyBackspace:
			if m.cursor > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
				if m.viewport > 0 {
					m.viewport--
				}
			}

		case tea.KeyDelete:
			if m.cursor < len(m.value) {
				m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
			}

		case tea.KeyLeft:
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.viewport {
					m.viewport = m.cursor
				}
			}

		case tea.KeyRight:
			if m.cursor < len(m.value) {
				m.cursor++
				if m.cursor > m.viewport+m.width-5 {
					m.viewport = m.cursor - m.width + 5
				}
			}

		case tea.KeyUp:
			// Navigate history
			if m.histIdx > 0 {
				m.histIdx--
				m.value = m.history[m.histIdx]
				m.cursor = len(m.value)
				m.viewport = 0
				if m.cursor > m.width-5 {
					m.viewport = m.cursor - m.width + 5
				}
			}

		case tea.KeyDown:
			// Navigate history
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.value = m.history[m.histIdx]
				m.cursor = len(m.value)
				m.viewport = 0
				if m.cursor > m.width-5 {
					m.viewport = m.cursor - m.width + 5
				}
			} else if m.histIdx == len(m.history)-1 {
				m.histIdx = len(m.history)
				m.value = ""
				m.cursor = 0
				m.viewport = 0
			}

		case tea.KeyCtrlU:
			// Clear line
			m.value = ""
			m.cursor = 0
			m.viewport = 0

		case tea.KeyCtrlW:
			// Delete word
			if m.cursor > 0 {
				// Find start of previous word
				start := m.cursor
				for start > 0 && m.value[start-1] == ' ' {
					start--
				}
				for start > 0 && m.value[start-1] != ' ' {
					start--
				}
				m.value = m.value[:start] + m.value[m.cursor:]
				m.cursor = start
				if m.viewport > m.cursor {
					m.viewport = m.cursor
				}
			}

		case tea.KeyHome:
			m.cursor = 0
			m.viewport = 0

		case tea.KeyEnd:
			m.cursor = len(m.value)
			if m.cursor > m.width-5 {
				m.viewport = m.cursor - m.width + 5
			}

		default:
			// Regular character input
			if msg.Type == tea.KeyRunes {
				char := string(msg.Runes)
				if len(m.value)+len(char) <= m.maxChars {
					m.value = m.value[:m.cursor] + char + m.value[m.cursor:]
					m.cursor += len(char)
					if m.cursor > m.viewport+m.width-5 {
						m.viewport = m.cursor - m.width + 5
					}
				}
			}
		}
	}

	return m, cmd
}

// View renders the input model
func (m InputModel) View() string {
	// Calculate visible portion
	visibleWidth := m.width - 6 // Account for prompt and borders
	if visibleWidth < 10 {
		visibleWidth = 10
	}

	visibleText := m.value
	if len(m.value) > visibleWidth {
		start := m.viewport
		if start+visibleWidth > len(m.value) {
			start = len(m.value) - visibleWidth
		}
		visibleText = m.value[start : start+visibleWidth]
	}

	// Calculate cursor position
	cursorPos := m.cursor - m.viewport
	if cursorPos < 0 {
		cursorPos = 0
	}
	if cursorPos > visibleWidth {
		cursorPos = visibleWidth
	}

	// Build the display line
	displayLine := m.prompt + visibleText

	// Format with cursor
	var styledLine strings.Builder
	for i, ch := range displayLine {
		if i == len(m.prompt)+cursorPos && m.focused {
			styledLine.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.Colors.Primary)).
				Render(string(ch)))
		} else {
			styledLine.WriteString(string(ch))
		}
	}

	// Add cursor if at end
	if m.focused && cursorPos >= len(visibleText) {
		styledLine.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color(m.theme.Colors.Primary)).
			Render(" "))
	}

	// Build complete view
	content := lipgloss.NewStyle().
		Width(m.width - 2).
		Render(styledLine.String())

	// Add help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Muted)).
		Render("Enter: submit | ↑↓: history | Ctrl+U: clear | Ctrl+W: del word")

	return m.style.Width(m.width).Render(
		content + "\n" + helpStyle,
	)
}

// SetValue sets the input value
func (m *InputModel) SetValue(value string) {
	m.value = value
	m.cursor = len(value)
	if m.cursor > m.width-5 {
		m.viewport = m.cursor - m.width + 5
	}
}

// GetValue returns the current input value
func (m *InputModel) GetValue() string {
	return m.value
}

// SetWidth sets the width
func (m *InputModel) SetWidth(width int) {
	m.width = width
}

// SetFocused sets focus state
func (m *InputModel) SetFocused(focused bool) {
	m.focused = focused
}

// Clear clears the input
func (m *InputModel) Clear() {
	m.value = ""
	m.cursor = 0
	m.viewport = 0
}

// GetHistory returns the command history
func (m *InputModel) GetHistory() []string {
	return m.history
}

// SetHistory sets the command history
func (m *InputModel) SetHistory(history []string) {
	m.history = history
	m.histIdx = len(history)
}
