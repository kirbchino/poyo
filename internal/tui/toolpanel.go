// Package tui implements tool panel component
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToolPanelModel handles the tool execution display
type ToolPanelModel struct {
	// Content
	executions []ToolExecDisplay
	selected   int

	// Dimensions
	width  int
	height int

	// Styling
	theme  *Theme
	styles ToolPanelStyles
}

// ToolExecDisplay represents a tool execution for display
type ToolExecDisplay struct {
	ID        string
	Name      string
	Status    string // "running", "completed", "error"
	StartTime time.Time
	EndTime   *time.Time
	Output    string
	Duration  time.Duration
}

// ToolPanelStyles holds styles for tool panel
type ToolPanelStyles struct {
	Title     lipgloss.Style
	Running   lipgloss.Style
	Completed lipgloss.Style
	Error     lipgloss.Style
	Output    lipgloss.Style
	Border    lipgloss.Style
}

// NewToolPanelModel creates a new tool panel model
func NewToolPanelModel(theme *Theme) ToolPanelModel {
	return ToolPanelModel{
		executions: make([]ToolExecDisplay, 0),
		selected:   -1,
		theme:      theme,
		styles: ToolPanelStyles{
			Title: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Primary)).
				Bold(true).
				Padding(0, 1),
			Running: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Warning)),
			Completed: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Success)),
			Error: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Error)),
			Output: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Colors.Text)).
				Padding(0, 1),
			Border: lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(lipgloss.Color(theme.Colors.Muted)).
				PaddingLeft(1),
		},
	}
}

// Init initializes the tool panel model
func (m ToolPanelModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the tool panel model
func (m ToolPanelModel) Update(msg tea.Msg) (ToolPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
			} else if m.selected < 0 && len(m.executions) > 0 {
				m.selected = len(m.executions) - 1
			}

		case tea.KeyDown:
			if m.selected >= 0 && m.selected < len(m.executions)-1 {
				m.selected++
			}
		}
	}
	return m, nil
}

// View renders the tool panel
func (m ToolPanelModel) View() string {
	var lines []string

	// Title
	lines = append(lines, m.styles.Title.Render("🔧 Tools"))
	lines = append(lines, "")

	if len(m.executions) == 0 {
		lines = append(lines, m.styles.Output.Render("No active tools"))
	} else {
		// Show executions (newest first)
		for i := len(m.executions) - 1; i >= 0; i-- {
			exec := m.executions[i]
			lines = append(lines, m.renderExecution(exec, i == m.selected)...)
		}
	}

	// Pad to height
	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}

	// Truncate if too tall
	if len(lines) > m.height-2 {
		lines = lines[:m.height-2]
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderExecution renders a single tool execution
func (m ToolPanelModel) renderExecution(exec ToolExecDisplay, selected bool) []string {
	var lines []string

	// Status indicator
	var statusIcon string
	var statusStyle lipgloss.Style
	switch exec.Status {
	case "running":
		statusIcon = "⏳"
		statusStyle = m.styles.Running
	case "completed":
		statusIcon = "✓"
		statusStyle = m.styles.Completed
	case "error":
		statusIcon = "✗"
		statusStyle = m.styles.Error
	default:
		statusIcon = "?"
		statusStyle = m.styles.Output
	}

	// Duration
	duration := exec.Duration
	if duration == 0 && exec.Status == "running" {
		duration = time.Since(exec.StartTime)
	}
	durationStr := formatDuration(duration)

	// Header line
	header := fmt.Sprintf("%s %s (%s)", statusIcon, exec.Name, durationStr)
	if selected {
		header = "▶ " + header
	}
	lines = append(lines, statusStyle.Render(header))

	// Show output preview if selected
	if selected && exec.Output != "" {
		output := exec.Output
		if len(output) > 100 {
			output = output[:100] + "..."
		}
		outputLines := strings.Split(output, "\n")
		for _, line := range outputLines {
			if len(lines) < 5 { // Limit output lines
				lines = append(lines, m.styles.Output.Render("  "+line))
			}
		}
	}

	lines = append(lines, "")

	return lines
}

// AddExecution adds a new tool execution
func (m *ToolPanelModel) AddExecution(exec ToolExecution) {
	m.executions = append(m.executions, ToolExecDisplay{
		ID:        exec.ID,
		Name:      exec.Name,
		Status:    exec.Status,
		StartTime: exec.StartTime,
		Output:    exec.Output,
	})
}

// UpdateExecution updates an execution's output
func (m *ToolPanelModel) UpdateExecution(id, output, status string) {
	for i := range m.executions {
		if m.executions[i].ID == id {
			m.executions[i].Output = output
			if status != "" {
				m.executions[i].Status = status
			}
		}
	}
}

// CompleteExecution marks an execution as completed
func (m *ToolPanelModel) CompleteExecution(id, output string) {
	now := time.Now()
	for i := range m.executions {
		if m.executions[i].ID == id {
			m.executions[i].Status = "completed"
			m.executions[i].Output = output
			m.executions[i].EndTime = &now
			m.executions[i].Duration = now.Sub(m.executions[i].StartTime)
		}
	}
}

// FailExecution marks an execution as failed
func (m *ToolPanelModel) FailExecution(id, errMsg string) {
	now := time.Now()
	for i := range m.executions {
		if m.executions[i].ID == id {
			m.executions[i].Status = "error"
			m.executions[i].Output = errMsg
			m.executions[i].EndTime = &now
			m.executions[i].Duration = now.Sub(m.executions[i].StartTime)
		}
	}
}

// SetSize sets the dimensions
func (m *ToolPanelModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Clear clears all executions
func (m *ToolPanelModel) Clear() {
	m.executions = make([]ToolExecDisplay, 0)
	m.selected = -1
}

// GetExecutionCount returns the number of executions
func (m *ToolPanelModel) GetExecutionCount() int {
	return len(m.executions)
}

// GetRunningCount returns the number of running executions
func (m *ToolPanelModel) GetRunningCount() int {
	count := 0
	for _, exec := range m.executions {
		if exec.Status == "running" {
			count++
		}
	}
	return count
}
