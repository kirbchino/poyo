// Package tui implements status bar component
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel handles the status bar display
type StatusBarModel struct {
	// State
	processing   bool
	message      string
	error        string
	model        string
	tokens       TokenInfo
	permission   string

	// Dimensions
	width int

	// Styling
	theme  *Theme
	styles StatusStyles
}

// TokenInfo holds token usage information
type TokenInfo struct {
	Input  int
	Output int
	Total  int
}

// StatusStyles holds styles for status bar
type StatusStyles struct {
	Normal     lipgloss.Style
	Processing lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Model      lipgloss.Style
	Tokens     lipgloss.Style
	Separator  string
}

// NewStatusBarModel creates a new status bar model
func NewStatusBarModel(theme *Theme) StatusBarModel {
	return StatusBarModel{
		processing: false,
		model:      "claude-sonnet-4-6",
		permission: "default",
		theme:      theme,
		styles: StatusStyles{
			Normal: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Text)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Processing: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Warning)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Error: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Error)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Success: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Success)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Model: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Primary)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Tokens: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Muted)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Padding(0, 1),
			Separator: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Muted)).
				Background(lipgloss.Color(theme.Colors.Background)).
				Render("│"),
		},
	}
}

// Init initializes the status bar model
func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the status bar model
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ProcessingStartMsg:
		m.processing = true
		m.message = "Processing..."
		m.error = ""

	case ProcessingEndMsg:
		m.processing = false
		m.message = ""

	case ErrorMessage:
		m.error = msg.Err.Error()
		m.processing = false

	case StreamChunkMsg:
		if !msg.Done {
			m.processing = true
			m.message = "Receiving response..."
		}
	}
	return m, nil
}

// View renders the status bar
func (m StatusBarModel) View() string {
	var sections []string

	// Left section: Status/Message
	if m.error != "" {
		sections = append(sections, m.styles.Error.Render("✗ "+truncate(m.error, 30)))
	} else if m.processing {
		sections = append(sections, m.styles.Processing.Render("⏳ "+m.message))
	} else {
		sections = append(sections, m.styles.Success.Render("✓ Ready"))
	}

	// Add separator
	sections = append(sections, m.styles.Separator)

	// Middle section: Model
	modelDisplay := m.model
	if modelDisplay == "" {
		modelDisplay = "claude-sonnet-4-6"
	}
	// Shorten model name for display
	if len(modelDisplay) > 15 {
		modelDisplay = modelDisplay[:12] + "..."
	}
	sections = append(sections, m.styles.Model.Render(modelDisplay))

	// Add separator
	sections = append(sections, m.styles.Separator)

	// Permission mode
	permDisplay := m.permission
	if permDisplay == "" {
		permDisplay = "default"
	}
	sections = append(sections, m.styles.Tokens.Render(permDisplay))

	// Add separator
	sections = append(sections, m.styles.Separator)

	// Right section: Tokens
	if m.tokens.Total > 0 {
		tokenStr := fmt.Sprintf("Tokens: %d in / %d out", m.tokens.Input, m.tokens.Output)
		sections = append(sections, m.styles.Tokens.Render(tokenStr))
	} else {
		sections = append(sections, m.styles.Tokens.Render("Tokens: -"))
	}

	// Calculate total width and join
	content := strings.Join(sections, " ")

	// Pad to fill width
	padding := m.width - lipgloss.Width(content)
	if padding > 0 {
		content += strings.Repeat(" ", padding)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Render(content)
}

// SetProcessing sets the processing state
func (m *StatusBarModel) SetProcessing(processing bool) {
	m.processing = processing
	if processing {
		m.message = "Processing..."
	} else {
		m.message = ""
	}
}

// SetMessage sets the status message
func (m *StatusBarModel) SetMessage(message string) {
	m.message = message
}

// SetError sets the error message
func (m *StatusBarModel) SetError(err string) {
	m.error = err
	m.processing = false
}

// ClearError clears the error
func (m *StatusBarModel) ClearError() {
	m.error = ""
}

// SetModel sets the current model
func (m *StatusBarModel) SetModel(model string) {
	m.model = model
}

// SetTokens sets the token usage
func (m *StatusBarModel) SetTokens(input, output int) {
	m.tokens.Input = input
	m.tokens.Output = output
	m.tokens.Total = input + output
}

// SetPermission sets the permission mode
func (m *StatusBarModel) SetPermission(permission string) {
	m.permission = permission
}

// SetWidth sets the width
func (m *StatusBarModel) SetWidth(width int) {
	m.width = width
}

// Spinner frames for animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// GetSpinnerFrame returns the current spinner frame
func GetSpinnerFrame(frame int) string {
	return spinnerFrames[frame%len(spinnerFrames)]
}
