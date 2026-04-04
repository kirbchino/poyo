// Package interaction provides user interaction capabilities for Poyo.
package interaction

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuestionType represents the type of question to ask
type QuestionType string

const (
	// TypeSingleChoice presents a single-select dropdown
	TypeSingleChoice QuestionType = "single"
	// TypeMultipleChoice presents a multi-select list
	TypeMultipleChoice QuestionType = "multiple"
	// TypeTextInput presents a free-form text input
	TypeTextInput QuestionType = "text"
	// TypeConfirmation presents a yes/no confirmation
	TypeConfirmation QuestionType = "confirm"
)

// Question represents a question to ask the user
type Question struct {
	// ID is a unique identifier for the question
	ID string `json:"id"`

	// Header is a short label displayed as a chip (max 12 chars)
	Header string `json:"header"`

	// Question is the full question text
	Question string `json:"question"`

	// Type is the question type
	Type QuestionType `json:"type"`

	// Options are the available choices (for single/multiple choice)
	Options []Option `json:"options,omitempty"`

	// Default is the default value
	Default interface{} `json:"default,omitempty"`

	// Placeholder is the input placeholder text
	Placeholder string `json:"placeholder,omitempty"`

	// Required indicates if the question must be answered
	Required bool `json:"required,omitempty"`

	// MinLength is the minimum text length (for text input)
	MinLength int `json:"minLength,omitempty"`

	// MaxLength is the maximum text length (for text input)
	MaxLength int `json:"maxLength,omitempty"`

	// MultiSelect enables multi-select for choice questions
	MultiSelect bool `json:"multiSelect,omitempty"`

	// Metadata contains additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Source indicates where this question originated
	Source string `json:"source,omitempty"`
}

// Option represents a choice option
type Option struct {
	// Label is the display text
	Label string `json:"label"`

	// Description explains what this option means
	Description string `json:"description,omitempty"`

	// Value is the actual value returned
	Value string `json:"value,omitempty"`

	// Preview shows a preview when focused
	Preview string `json:"preview,omitempty"`

	// Disabled indicates if this option is not selectable
	Disabled bool `json:"disabled,omitempty"`

	// Default indicates if this option is pre-selected
	Default bool `json:"default,omitempty"`
}

// Answer represents a user's answer to a question
type Answer struct {
	// QuestionID is the ID of the question being answered
	QuestionID string `json:"questionId"`

	// Type is the question type
	Type QuestionType `json:"type"`

	// Value is the selected value (for single choice) or text input
	Value string `json:"value,omitempty"`

	// Values is the selected values (for multiple choice)
	Values []string `json:"values,omitempty"`

	// Text is the free-form text input
	Text string `json:"text,omitempty"`

	// Confirmed is for confirmation dialogs
	Confirmed bool `json:"confirmed,omitempty"`

	// IsOther indicates if the "Other" option was selected
	IsOther bool `json:"isOther,omitempty"`

	// OtherText contains custom text when "Other" is selected
	OtherText string `json:"otherText,omitempty"`

	// Timestamp is when the answer was given
	Timestamp time.Time `json:"timestamp"`
}

// Interaction represents a user interaction session
type Interaction struct {
	// ID is the unique interaction ID
	ID string `json:"id"`

	// Questions are the questions to ask
	Questions []Question `json:"questions"`

	// Answers are the received answers
	Answers map[string]*Answer `json:"answers,omitempty"`

	// State is the current state
	State InteractionState `json:"state"`

	// CreatedAt is when the interaction was created
	CreatedAt time.Time `json:"createdAt"`

	// CompletedAt is when the interaction was completed
	CompletedAt time.Time `json:"completedAt,omitempty"`

	// Context provides additional context
	Context *InteractionContext `json:"context,omitempty"`

	// respondChan is used to send answers
	respondChan chan *Answer `json:"-"`

	// doneChan signals completion
	doneChan chan struct{} `json:"-"`
}

// InteractionState represents the state of an interaction
type InteractionState string

const (
	StatePending    InteractionState = "pending"
	StateAnswered   InteractionState = "answered"
	StateCancelled  InteractionState = "cancelled"
	StateTimeout    InteractionState = "timeout"
)

// InteractionContext provides context for the interaction
type InteractionContext struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId,omitempty"`

	// ConversationID is the conversation identifier
	ConversationID string `json:"conversationId,omitempty"`

	// ToolUseID is the tool use ID that triggered this
	ToolUseID string `json:"toolUseId,omitempty"`

	// Prompt is the prompt that led to this question
	Prompt string `json:"prompt,omitempty"`

	// AllowedPrompts lists prompts that the user can approve
	AllowedPrompts []string `json:"allowedPrompts,omitempty"`
}

// InteractionResult represents the result of an interaction
type InteractionResult struct {
	// InteractionID is the interaction ID
	InteractionID string `json:"interactionId"`

	// State is the final state
	State InteractionState `json:"state"`

	// Answers are the collected answers
	Answers map[string]*Answer `json:"answers"`

	// Error is an error message if failed
	Error string `json:"error,omitempty"`
}

// Manager manages user interactions
type Manager struct {
	interactions map[string]*Interaction
	pending      chan *Interaction
	mu           sync.RWMutex
	timeout      time.Duration
}

// NewManager creates a new interaction manager
func NewManager() *Manager {
	return &Manager{
		interactions: make(map[string]*Interaction),
		pending:      make(chan *Interaction, 10),
		timeout:      5 * time.Minute,
	}
}

// SetTimeout sets the default timeout for interactions
func (m *Manager) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

// Ask creates a new interaction and waits for answers
func (m *Manager) Ask(ctx context.Context, questions []Question) (*InteractionResult, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("no questions provided")
	}

	if len(questions) > 4 {
		return nil, fmt.Errorf("maximum 4 questions allowed")
	}

	// Create interaction
	interaction := &Interaction{
		ID:          generateInteractionID(),
		Questions:   questions,
		Answers:     make(map[string]*Answer),
		State:       StatePending,
		CreatedAt:   time.Now(),
		respondChan: make(chan *Answer, len(questions)),
		doneChan:    make(chan struct{}),
	}

	// Set question IDs if not set
	for i := range interaction.Questions {
		if interaction.Questions[i].ID == "" {
			interaction.Questions[i].ID = fmt.Sprintf("q_%d", i)
		}
	}

	// Register interaction
	m.mu.Lock()
	m.interactions[interaction.ID] = interaction
	m.mu.Unlock()

	// Send to pending queue
	select {
	case m.pending <- interaction:
	default:
		// Channel full, remove and return error
		m.mu.Lock()
		delete(m.interactions, interaction.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("interaction queue full")
	}

	// Wait for answers or timeout
	timeout := m.timeout
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	answerCount := 0
	for answerCount < len(questions) {
		select {
		case answer := <-interaction.respondChan:
			interaction.Answers[answer.QuestionID] = answer
			answerCount++

		case <-ctx.Done():
			m.mu.Lock()
			interaction.State = StateTimeout
			delete(m.interactions, interaction.ID)
			m.mu.Unlock()

			return &InteractionResult{
				InteractionID: interaction.ID,
				State:         StateTimeout,
				Answers:       interaction.Answers,
				Error:         "interaction timeout",
			}, nil

		case <-interaction.doneChan:
			// Interaction was cancelled
			m.mu.Lock()
			delete(m.interactions, interaction.ID)
			m.mu.Unlock()

			return &InteractionResult{
				InteractionID: interaction.ID,
				State:         interaction.State,
				Answers:       interaction.Answers,
			}, nil
		}
	}

	// Mark as completed
	m.mu.Lock()
	interaction.State = StateAnswered
	interaction.CompletedAt = time.Now()
	delete(m.interactions, interaction.ID)
	m.mu.Unlock()

	return &InteractionResult{
		InteractionID: interaction.ID,
		State:         StateAnswered,
		Answers:       interaction.Answers,
	}, nil
}

// Respond submits an answer for a pending interaction
func (m *Manager) Respond(interactionID string, answer *Answer) error {
	m.mu.RLock()
	interaction, ok := m.interactions[interactionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("interaction %s not found", interactionID)
	}

	answer.Timestamp = time.Now()

	select {
	case interaction.respondChan <- answer:
		return nil
	default:
		return fmt.Errorf("interaction channel full")
	}
}

// Cancel cancels a pending interaction
func (m *Manager) Cancel(interactionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	interaction, ok := m.interactions[interactionID]
	if !ok {
		return fmt.Errorf("interaction %s not found", interactionID)
	}

	interaction.State = StateCancelled
	close(interaction.doneChan)

	return nil
}

// GetPending returns all pending interactions
func (m *Manager) GetPending() []*Interaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*Interaction
	for _, interaction := range m.interactions {
		if interaction.State == StatePending {
			pending = append(pending, interaction)
		}
	}
	return pending
}

// Get returns a specific interaction
func (m *Manager) Get(interactionID string) (*Interaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	interaction, ok := m.interactions[interactionID]
	return interaction, ok
}

// PendingChannel returns the channel for new pending interactions
func (m *Manager) PendingChannel() <-chan *Interaction {
	return m.pending
}

// generateInteractionID generates a unique interaction ID
func generateInteractionID() string {
	return fmt.Sprintf("int_%d", time.Now().UnixNano())
}

// QuestionBuilder helps build questions fluently
type QuestionBuilder struct {
	question Question
}

// NewQuestion creates a new question builder
func NewQuestion(questionText string) *QuestionBuilder {
	return &QuestionBuilder{
		question: Question{
			Question: questionText,
		},
	}
}

// WithHeader sets the header
func (b *QuestionBuilder) WithHeader(header string) *QuestionBuilder {
	b.question.Header = header
	return b
}

// WithType sets the question type
func (b *QuestionBuilder) WithType(t QuestionType) *QuestionBuilder {
	b.question.Type = t
	return b
}

// WithOptions sets the options for choice questions
func (b *QuestionBuilder) WithOptions(options []Option) *QuestionBuilder {
	b.question.Options = options
	return b
}

// WithDefault sets the default value
func (b *QuestionBuilder) WithDefault(value interface{}) *QuestionBuilder {
	b.question.Default = value
	return b
}

// WithPlaceholder sets the placeholder text
func (b *QuestionBuilder) WithPlaceholder(placeholder string) *QuestionBuilder {
	b.question.Placeholder = placeholder
	return b
}

// WithRequired sets whether the question is required
func (b *QuestionBuilder) WithRequired(required bool) *QuestionBuilder {
	b.question.Required = required
	return b
}

// WithMultiSelect enables multi-select for choice questions
func (b *QuestionBuilder) WithMultiSelect(multiSelect bool) *QuestionBuilder {
	b.question.MultiSelect = multiSelect
	return b
}

// Build returns the built question
func (b *QuestionBuilder) Build() Question {
	if b.question.ID == "" {
		b.question.ID = fmt.Sprintf("q_%d", time.Now().UnixNano())
	}
	return b.question
}

// Validate validates a question
func (q *Question) Validate() error {
	if q.Question == "" {
		return fmt.Errorf("question text is required")
	}

	if len(q.Question) > 500 {
		return fmt.Errorf("question text too long (max 500 chars)")
	}

	if len(q.Header) > 12 {
		return fmt.Errorf("header too long (max 12 chars)")
	}

	switch q.Type {
	case TypeSingleChoice, TypeMultipleChoice:
		if len(q.Options) < 2 {
			return fmt.Errorf("choice questions require at least 2 options")
		}
		if len(q.Options) > 4 {
			return fmt.Errorf("maximum 4 options allowed")
		}
		for _, opt := range q.Options {
			if opt.Label == "" {
				return fmt.Errorf("option label is required")
			}
		}
	}

	return nil
}

// ValidateAnswer validates an answer against a question
func (q *Question) ValidateAnswer(answer *Answer) error {
	if answer.QuestionID != q.ID {
		return fmt.Errorf("answer question ID mismatch")
	}

	switch q.Type {
	case TypeSingleChoice:
		if answer.Value == "" {
			return fmt.Errorf("value is required for single choice")
		}
		// Check if value is valid
		valid := false
		for _, opt := range q.Options {
			if opt.Value == answer.Value || opt.Label == answer.Value {
				valid = true
				break
			}
		}
		if !valid && !answer.IsOther {
			return fmt.Errorf("invalid choice: %s", answer.Value)
		}

	case TypeMultipleChoice:
		if len(answer.Values) == 0 {
			return fmt.Errorf("at least one value is required for multiple choice")
		}
		// Validate each value
		for _, v := range answer.Values {
			valid := false
			for _, opt := range q.Options {
				if opt.Value == v || opt.Label == v {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid choice: %s", v)
			}
		}

	case TypeTextInput:
		if q.Required && answer.Text == "" {
			return fmt.Errorf("text input is required")
		}
		if q.MinLength > 0 && len(answer.Text) < q.MinLength {
			return fmt.Errorf("text must be at least %d characters", q.MinLength)
		}
		if q.MaxLength > 0 && len(answer.Text) > q.MaxLength {
			return fmt.Errorf("text must be at most %d characters", q.MaxLength)
		}

	case TypeConfirmation:
		// Confirmation always valid
	}

	return nil
}
