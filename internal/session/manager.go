// Package session implements session persistence for conversation history
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Session represents a conversation session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	WorkingDir string   `json:"working_dir"`
	Model     string    `json:"model"`
	Config    SessionConfig `json:"config"`
}

// Message represents a message in the session
type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
	ToolCalls []ToolCall      `json:"tool_calls,omitempty"`
	ToolResult *ToolResult    `json:"tool_result,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents a tool result
type ToolResult struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error"`
}

// SessionConfig holds session configuration
type SessionConfig struct {
	PermissionMode string            `json:"permission_mode"`
	MaxTurns       int               `json:"max_turns"`
	Variables      map[string]string `json:"variables"`
}

// SessionManager manages session persistence
type SessionManager struct {
	mu          sync.RWMutex
	sessionsDir string
	current     *Session
	autoSave    bool
}

// NewSessionManager creates a new session manager
func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionsDir := filepath.Join(homeDir, ".claude", "sessions")

	// Ensure directory exists
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &SessionManager{
		sessionsDir: sessionsDir,
		autoSave:    true,
	}, nil
}

// NewSession creates a new session
func (m *SessionManager) NewSession(workingDir, model string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:         generateSessionID(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Messages:   make([]Message, 0),
		WorkingDir: workingDir,
		Model:      model,
		Config: SessionConfig{
			PermissionMode: "default",
			Variables:      make(map[string]string),
		},
	}

	m.current = session
	return session
}

// LoadSession loads a session by ID
func (m *SessionManager) LoadSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.sessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	m.current = &session
	return &session, nil
}

// SaveSession saves the current session
func (m *SessionManager) SaveSession() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return nil
	}

	m.current.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(m.current, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := m.sessionPath(m.current.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// GetCurrentSession returns the current session
func (m *SessionManager) GetCurrentSession() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// AddMessage adds a message to the current session
func (m *SessionManager) AddMessage(msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return fmt.Errorf("no active session")
	}

	m.current.Messages = append(m.current.Messages, msg)
	m.current.UpdatedAt = time.Now()

	if m.autoSave {
		go m.SaveSession()
	}

	return nil
}

// ListSessions lists all saved sessions
func (m *SessionManager) ListSessions() ([]SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		info, err := m.GetSessionInfo(id)
		if err != nil {
			continue
		}

		sessions = append(sessions, info)
	}

	// Sort by updated time (newest first)
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	return sessions, nil
}

// SessionInfo contains summary information about a session
type SessionInfo struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	Preview      string    `json:"preview"`
	Model        string    `json:"model"`
}

// GetSessionInfo gets summary info for a session
func (m *SessionManager) GetSessionInfo(id string) (SessionInfo, error) {
	path := m.sessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return SessionInfo{}, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return SessionInfo{}, err
	}

	// Get preview from first user message
	preview := "No messages"
	for _, msg := range session.Messages {
		if msg.Role == "user" {
			preview = truncate(msg.Content, 100)
			break
		}
	}

	return SessionInfo{
		ID:           session.ID,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
		MessageCount: len(session.Messages),
		Preview:      preview,
		Model:        session.Model,
	}, nil
}

// DeleteSession deletes a session
func (m *SessionManager) DeleteSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.sessionPath(id)

	if m.current != nil && m.current.ID == id {
		m.current = nil
	}

	return os.Remove(path)
}

// SetAutoSave sets the auto-save flag
func (m *SessionManager) SetAutoSave(autoSave bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoSave = autoSave
}

// Compact compacts the session by summarizing old messages
func (m *SessionManager) Compact(maxMessages int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil || len(m.current.Messages) <= maxMessages {
		return nil
	}

	// Find where to compact
	cutoffIdx := len(m.current.Messages) - maxMessages
	if cutoffIdx <= 0 {
		return nil
	}

	// Extract key information from messages to be compacted
	messagesToCompact := m.current.Messages[:cutoffIdx]
	summaryContent := m.generateSummary(messagesToCompact)

	// Build new message list
	var newMessages []Message

	// Add summary message if we have content
	if summaryContent != "" {
		summaryMsg := Message{
			ID:        generateMessageID(),
			Role:      "system",
			Content:   fmt.Sprintf("[Previous conversation summarized]\n%s", summaryContent),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"type":          "summary",
				"original_count": len(messagesToCompact),
			},
		}
		newMessages = append(newMessages, summaryMsg)
	}

	// Add recent messages
	newMessages = append(newMessages, m.current.Messages[cutoffIdx:]...)

	m.current.Messages = newMessages
	m.current.UpdatedAt = time.Now()

	// Save session inline to avoid deadlock (we already hold the lock)
	if m.autoSave {
		m.current.UpdatedAt = time.Now()
		data, err := json.MarshalIndent(m.current, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal session: %w", err)
		}
		path := m.sessionPath(m.current.ID)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("write session: %w", err)
		}
	}

	return nil
}

// generateSummary generates a summary from a list of messages
func (m *SessionManager) generateSummary(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString("Key points from earlier conversation:\n")

	// Track topics and actions
	var topics []string
	var toolsUsed []string
	var filesAccessed []string
	var decisions []string

	for _, msg := range messages {
		// Extract user questions/topics
		if msg.Role == "user" {
			content := strings.TrimSpace(msg.Content)
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			if content != "" {
				topics = append(topics, fmt.Sprintf("- User asked: %s", content))
			}
		}

		// Extract tool calls
		for _, tc := range msg.ToolCalls {
			toolsUsed = append(toolsUsed, tc.Name)

			// Track file operations
			if tc.Name == "Read" || tc.Name == "Write" || tc.Name == "Edit" {
				if path, ok := tc.Input["file_path"].(string); ok {
					filesAccessed = append(filesAccessed, path)
				}
			}
			if tc.Name == "Bash" {
				if cmd, ok := tc.Input["command"].(string); ok {
					// Extract key command info
					if strings.Contains(cmd, "git") {
						filesAccessed = append(filesAccessed, "[git operations]")
					}
				}
			}
		}

		// Extract key information from assistant responses
		if msg.Role == "assistant" {
			content := strings.TrimSpace(msg.Content)
			// Look for decision-like content
			if strings.Contains(content, "决定") ||
				strings.Contains(content, "选择") ||
				strings.Contains(content, "建议") ||
				strings.Contains(content, "recommend") ||
				strings.Contains(content, "decided") {
				// Extract first sentence as decision
				if idx := strings.Index(content, "\n"); idx > 0 {
					decisions = append(decisions, content[:min(idx, 100)])
				}
			}
		}
	}

	// Build summary sections
	if len(topics) > 0 {
		// Limit to last 5 topics
		start := 0
		if len(topics) > 5 {
			start = len(topics) - 5
		}
		summary.WriteString("\nTopics discussed:\n")
		for _, t := range topics[start:] {
			summary.WriteString(t + "\n")
		}
	}

	if len(toolsUsed) > 0 {
		// Count tool usage
		toolCounts := make(map[string]int)
		for _, t := range toolsUsed {
			toolCounts[t]++
		}
		summary.WriteString("\nTools used: ")
		var toolStrs []string
		for tool, count := range toolCounts {
			toolStrs = append(toolStrs, fmt.Sprintf("%s(%dx)", tool, count))
		}
		summary.WriteString(strings.Join(toolStrs, ", "))
		summary.WriteString("\n")
	}

	if len(filesAccessed) > 0 {
		// Deduplicate files
		seen := make(map[string]bool)
		var uniqueFiles []string
		for _, f := range filesAccessed {
			if !seen[f] {
				seen[f] = true
				uniqueFiles = append(uniqueFiles, f)
			}
		}
		summary.WriteString("\nFiles accessed: ")
		if len(uniqueFiles) > 5 {
			summary.WriteString(strings.Join(uniqueFiles[:5], ", "))
			summary.WriteString(fmt.Sprintf(" and %d more", len(uniqueFiles)-5))
		} else {
			summary.WriteString(strings.Join(uniqueFiles, ", "))
		}
		summary.WriteString("\n")
	}

	if len(decisions) > 0 {
		summary.WriteString("\nKey decisions:\n")
		for i, d := range decisions {
			if i >= 3 {
				break
			}
			summary.WriteString(fmt.Sprintf("- %s\n", d))
		}
	}

	return summary.String()
}

// CompactWithContext compacts the session using an LLM for better summarization
func (m *SessionManager) CompactWithContext(maxMessages int, summarizeFunc func(messages []Message) (string, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil || len(m.current.Messages) <= maxMessages {
		return nil
	}

	cutoffIdx := len(m.current.Messages) - maxMessages
	if cutoffIdx <= 0 {
		return nil
	}

	messagesToCompact := m.current.Messages[:cutoffIdx]

	// Use provided summarization function
	summaryContent, err := summarizeFunc(messagesToCompact)
	if err != nil {
		// Fallback to basic summarization
		summaryContent = m.generateSummary(messagesToCompact)
	}

	var newMessages []Message

	if summaryContent != "" {
		summaryMsg := Message{
			ID:        generateMessageID(),
			Role:      "system",
			Content:   fmt.Sprintf("[Previous conversation summarized]\n%s", summaryContent),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"type":          "llm_summary",
				"original_count": len(messagesToCompact),
			},
		}
		newMessages = append(newMessages, summaryMsg)
	}

	newMessages = append(newMessages, m.current.Messages[cutoffIdx:]...)
	m.current.Messages = newMessages
	m.current.UpdatedAt = time.Now()

	// Save session inline to avoid deadlock (we already hold the lock)
	if m.autoSave {
		m.current.UpdatedAt = time.Now()
		data, err := json.MarshalIndent(m.current, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal session: %w", err)
		}
		path := m.sessionPath(m.current.ID)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("write session: %w", err)
		}
	}

	return nil
}

// ExportSession exports a session to a file
func (m *SessionManager) ExportSession(id, format, outputPath string) error {
	session, err := m.LoadSession(id)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, data, 0644)

	case "markdown", "md":
		return m.exportMarkdown(session, outputPath)

	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportMarkdown exports session as markdown
func (m *SessionManager) exportMarkdown(session *Session, outputPath string) error {
	var content string
	content += fmt.Sprintf("# Session: %s\n\n", session.ID)
	content += fmt.Sprintf("Created: %s\n", session.CreatedAt.Format(time.RFC3339))
	content += fmt.Sprintf("Model: %s\n\n", session.Model)
	content += "---\n\n"

	for _, msg := range session.Messages {
		switch msg.Role {
		case "user":
			content += fmt.Sprintf("## User\n\n%s\n\n", msg.Content)
		case "assistant":
			content += fmt.Sprintf("## Assistant\n\n%s\n\n", msg.Content)
		case "system":
			content += fmt.Sprintf("## System\n\n%s\n\n", msg.Content)
		}

		if msg.ToolResult != nil {
			content += fmt.Sprintf("**Tool Result (%s):**\n```\n%v\n```\n\n",
				msg.ToolResult.ToolUseID, msg.ToolResult.Content)
		}
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// Helper functions

func (m *SessionManager) sessionPath(id string) string {
	return filepath.Join(m.sessionsDir, id+".json")
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
