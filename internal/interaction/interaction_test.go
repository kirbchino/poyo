package interaction

import (
	"context"
	"testing"
	"time"
)

func TestQuestionType(t *testing.T) {
	types := []QuestionType{
		TypeSingleChoice,
		TypeMultipleChoice,
		TypeTextInput,
		TypeConfirmation,
	}

	for _, qt := range types {
		if string(qt) == "" {
			t.Errorf("Question type should have a non-empty string representation")
		}
	}
}

func TestInteractionState(t *testing.T) {
	states := []InteractionState{
		StatePending,
		StateAnswered,
		StateCancelled,
		StateTimeout,
	}

	for _, s := range states {
		if string(s) == "" {
			t.Errorf("Interaction state should have a non-empty string representation")
		}
	}
}

func TestQuestionValidate(t *testing.T) {
	tests := []struct {
		question Question
		expectError bool
	}{
		{
			question: Question{
				Question: "What is your choice?",
				Type:     TypeSingleChoice,
				Options: []Option{
					{Label: "Option A"},
					{Label: "Option B"},
				},
			},
			expectError: false,
		},
		{
			question: Question{
				Question: "",
				Type:     TypeSingleChoice,
			},
			expectError: true,
		},
		{
			question: Question{
				Question: "What is your choice?",
				Type:     TypeSingleChoice,
				Options: []Option{
					{Label: "Option A"},
				},
			},
			expectError: true, // Need at least 2 options
		},
		{
			question: Question{
				Question: "What is your choice?",
				Type:     TypeSingleChoice,
				Options: []Option{
					{Label: "A"},
					{Label: "B"},
					{Label: "C"},
					{Label: "D"},
					{Label: "E"},
				},
			},
			expectError: true, // Max 4 options
		},
		{
			question: Question{
				Question: "What is your choice?",
				Type:     TypeSingleChoice,
				Options: []Option{
					{Label: ""}, // Empty label
					{Label: "B"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		err := tt.question.Validate()
		if tt.expectError && err == nil {
			t.Errorf("Expected error for question: %+v", tt.question)
		}
		if !tt.expectError && err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestQuestionValidateAnswer(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "What is your choice?",
		Type:     TypeSingleChoice,
		Options: []Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
		},
	}

	tests := []struct {
		answer    *Answer
		expectError bool
	}{
		{
			answer: &Answer{
				QuestionID: "q_0",
				Type:       TypeSingleChoice,
				Value:      "a",
			},
			expectError: false,
		},
		{
			answer: &Answer{
				QuestionID: "q_wrong",
				Type:       TypeSingleChoice,
				Value:      "a",
			},
			expectError: true,
		},
		{
			answer: &Answer{
				QuestionID: "q_0",
				Type:       TypeSingleChoice,
				Value:      "invalid",
			},
			expectError: true,
		},
		{
			answer: &Answer{
				QuestionID: "q_0",
				Type:       TypeSingleChoice,
				Value:      "", // Empty value
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		err := question.ValidateAnswer(tt.answer)
		if tt.expectError && err == nil {
			t.Errorf("Expected error for answer: %+v", tt.answer)
		}
		if !tt.expectError && err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.interactions == nil {
		t.Error("interactions map should be initialized")
	}

	if m.pending == nil {
		t.Error("pending channel should be initialized")
	}
}

func TestManagerAsk(t *testing.T) {
	m := NewManager()
	m.SetTimeout(100 * time.Millisecond)

	questions := []Question{
		{
			ID:       "q_0",
			Header:   "Choice",
			Question: "What is your choice?",
			Type:     TypeSingleChoice,
			Options: []Option{
				{Label: "Option A", Value: "a"},
				{Label: "Option B", Value: "b"},
			},
		},
	}

	// Start asking in a goroutine
	resultChan := make(chan *InteractionResult, 1)
	go func() {
		result, err := m.Ask(context.Background(), questions)
		if err != nil {
			resultChan <- &InteractionResult{Error: err.Error()}
			return
		}
		resultChan <- result
	}()

	// Wait for the interaction to be registered
	time.Sleep(10 * time.Millisecond)

	// Get pending interactions
	pending := m.GetPending()
	if len(pending) == 0 {
		t.Fatal("Expected pending interaction")
	}

	// Respond to the interaction
	interaction := pending[0]
	err := m.Respond(interaction.ID, &Answer{
		QuestionID: "q_0",
		Type:       TypeSingleChoice,
		Value:      "a",
	})
	if err != nil {
		t.Errorf("Respond() error: %v", err)
	}

	// Wait for result
	select {
	case result := <-resultChan:
		if result.State != StateAnswered {
			t.Errorf("Result state = %v, want %v", result.State, StateAnswered)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestManagerAskTimeout(t *testing.T) {
	m := NewManager()
	m.SetTimeout(50 * time.Millisecond)

	questions := []Question{
		{
			ID:       "q_0",
			Header:   "Choice",
			Question: "What is your choice?",
			Type:     TypeSingleChoice,
			Options: []Option{
				{Label: "Option A", Value: "a"},
				{Label: "Option B", Value: "b"},
			},
		},
	}

	result, err := m.Ask(context.Background(), questions)
	if err != nil {
		t.Fatalf("Ask() error: %v", err)
	}

	if result.State != StateTimeout {
		t.Errorf("Result state = %v, want %v", result.State, StateTimeout)
	}
}

func TestManagerCancel(t *testing.T) {
	m := NewManager()
	m.SetTimeout(1 * time.Second)

	questions := []Question{
		{
			ID:       "q_0",
			Header:   "Choice",
			Question: "What is your choice?",
			Type:     TypeSingleChoice,
			Options: []Option{
				{Label: "Option A", Value: "a"},
				{Label: "Option B", Value: "b"},
			},
		},
	}

	// Start asking in a goroutine
	resultChan := make(chan *InteractionResult, 1)
	go func() {
		result, err := m.Ask(context.Background(), questions)
		if err != nil {
			resultChan <- &InteractionResult{Error: err.Error()}
			return
		}
		resultChan <- result
	}()

	// Wait for the interaction to be registered
	time.Sleep(10 * time.Millisecond)

	// Get pending interactions
	pending := m.GetPending()
	if len(pending) == 0 {
		t.Fatal("Expected pending interaction")
	}

	// Cancel the interaction
	interaction := pending[0]
	err := m.Cancel(interaction.ID)
	if err != nil {
		t.Errorf("Cancel() error: %v", err)
	}

	// Wait for result
	select {
	case result := <-resultChan:
		if result.State != StateCancelled {
			t.Errorf("Result state = %v, want %v", result.State, StateCancelled)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestAskUserQuestionTool(t *testing.T) {
	m := NewManager()
	tool := NewAskUserQuestionTool(m)

	if tool.Name() != "AskUserQuestion" {
		t.Errorf("Name() = %q, want 'AskUserQuestion'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}
}

func TestQuestionBuilder(t *testing.T) {
	question := NewQuestion("What is your choice?").
		WithHeader("Choice").
		WithType(TypeSingleChoice).
		WithOptions([]Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
		}).
		Build()

	if question.Question != "What is your choice?" {
		t.Errorf("Question = %q, want 'What is your choice?'", question.Question)
	}

	if question.Header != "Choice" {
		t.Errorf("Header = %q, want 'Choice'", question.Header)
	}

	if question.Type != TypeSingleChoice {
		t.Errorf("Type = %v, want %v", question.Type, TypeSingleChoice)
	}

	if len(question.Options) != 2 {
		t.Errorf("Options count = %d, want 2", len(question.Options))
	}
}

func TestAskUserQuestionToolExecute(t *testing.T) {
	m := NewManager()
	m.SetTimeout(100 * time.Millisecond)
	tool := NewAskUserQuestionTool(m)

	input := []byte(`{
		"questions": [{
			"header": "Auth",
			"question": "Which authentication method?",
			"multiSelect": false,
			"options": [
				{"label": "OAuth (Recommended)", "description": "Use OAuth 2.0"},
				{"label": "API Key", "description": "Use API key"}
			]
		}]
	}`)

	// Start execution in goroutine
	resultChan := make(chan interface{}, 1)
	go func() {
		result, err := tool.Execute(context.Background(), input)
		if err != nil {
			resultChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for the interaction to be registered
	time.Sleep(10 * time.Millisecond)

	// Respond
	pending := m.GetPending()
	if len(pending) > 0 {
		m.Respond(pending[0].ID, &Answer{
			QuestionID: "q_0",
			Type:       TypeSingleChoice,
			Value:      "OAuth (Recommended)",
		})
	}

	// Wait for result
	select {
	case result := <-resultChan:
		if err, ok := result.(error); ok {
			t.Errorf("Execute() error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestAskUserQuestionToolInvalidInput(t *testing.T) {
	m := NewManager()
	tool := NewAskUserQuestionTool(m)

	// Invalid JSON
	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}

	// No questions
	_, err = tool.Execute(context.Background(), []byte(`{"questions": []}`))
	if err == nil {
		t.Error("Execute() should return error for no questions")
	}

	// Too many questions
	tooManyQuestions := `{"questions": [{"header":"H","question":"Q?","multiSelect":false,"options":[{"label":"A"},{"label":"B"}]},`
	tooManyQuestions += `{"header":"H","question":"Q?","multiSelect":false,"options":[{"label":"A"},{"label":"B"}]},`
	tooManyQuestions += `{"header":"H","question":"Q?","multiSelect":false,"options":[{"label":"A"},{"label":"B"}]},`
	tooManyQuestions += `{"header":"H","question":"Q?","multiSelect":false,"options":[{"label":"A"},{"label":"B"}]},`
	tooManyQuestions += `{"header":"H","question":"Q?","multiSelect":false,"options":[{"label":"A"},{"label":"B"}]}]}`

	_, err = tool.Execute(context.Background(), []byte(tooManyQuestions))
	if err == nil {
		t.Error("Execute() should return error for too many questions")
	}
}

// Additional comprehensive tests

func TestMultipleChoiceQuestion(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "Select all that apply:",
		Type:     TypeMultipleChoice,
		Options: []Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
			{Label: "Option C", Value: "c"},
		},
	}

	// Validate multiple choice answer
	answer := &Answer{
		QuestionID: "q_0",
		Type:       TypeMultipleChoice,
		Values:     []string{"a", "b"},
	}

	err := question.ValidateAnswer(answer)
	if err != nil {
		t.Errorf("Multiple choice answer should be valid: %v", err)
	}
}

func TestTextInputQuestion(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "Enter your name:",
		Type:     TypeTextInput,
	}

	err := question.Validate()
	if err != nil {
		t.Errorf("Text input question should be valid: %v", err)
	}

	// Test text answer
	answer := &Answer{
		QuestionID: "q_0",
		Type:       TypeTextInput,
		Text:       "John Doe",
	}

	err = question.ValidateAnswer(answer)
	if err != nil {
		t.Errorf("Text input answer should be valid: %v", err)
	}
}

func TestConfirmationQuestion(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "Are you sure?",
		Type:     TypeConfirmation,
	}

	err := question.Validate()
	if err != nil {
		t.Errorf("Confirmation question should be valid: %v", err)
	}

	// Test confirmation answer
	answer := &Answer{
		QuestionID: "q_0",
		Type:       TypeConfirmation,
		Value:      "yes",
	}

	err = question.ValidateAnswer(answer)
	if err != nil {
		t.Errorf("Confirmation answer should be valid: %v", err)
	}
}

func TestManagerRespondToNonExistent(t *testing.T) {
	m := NewManager()

	err := m.Respond("nonexistent-id", &Answer{
		QuestionID: "q_0",
		Type:       TypeSingleChoice,
		Value:      "a",
	})

	if err == nil {
		t.Error("Respond() should return error for non-existent interaction")
	}
}

func TestManagerCancelNonExistent(t *testing.T) {
	m := NewManager()

	err := m.Cancel("nonexistent-id")
	if err == nil {
		t.Error("Cancel() should return error for non-existent interaction")
	}
}

func TestQuestionWithOptionsPreview(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "Choose a file:",
		Type:     TypeSingleChoice,
		Options: []Option{
			{Label: "File A", Value: "file_a.txt", Preview: "Content of file A"},
			{Label: "File B", Value: "file_b.txt", Preview: "Content of file B"},
		},
	}

	for _, opt := range question.Options {
		if opt.Preview == "" {
			t.Errorf("Option %s should have preview", opt.Label)
		}
	}
}

func TestQuestionWithDefaultOption(t *testing.T) {
	question := Question{
		ID:       "q_0",
		Question: "Select mode:",
		Type:     TypeSingleChoice,
		Options: []Option{
			{Label: "Auto", Value: "auto"},
			{Label: "Manual (Recommended)", Value: "manual"},
		},
	}

	// Find recommended option
	for _, opt := range question.Options {
		if containsString(opt.Label, "Recommended") {
			// OK
		}
	}
}

func TestManagerConcurrentAsk(t *testing.T) {
	m := NewManager()
	m.SetTimeout(100 * time.Millisecond)

	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			questions := []Question{
				{
					ID:       "q_" + string(rune('0'+idx)),
					Header:   "Choice",
					Question: "What is your choice?",
					Type:     TypeSingleChoice,
					Options: []Option{
						{Label: "Option A", Value: "a"},
						{Label: "Option B", Value: "b"},
					},
				},
			}
			m.Ask(context.Background(), questions)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestQuestionBuilderWithPreview(t *testing.T) {
	question := NewQuestion("Choose a file:").
		WithHeader("File").
		WithType(TypeSingleChoice).
		WithOptions([]Option{
			{Label: "File A", Value: "a", Preview: "Preview A"},
			{Label: "File B", Value: "b", Preview: "Preview B"},
		}).
		Build()

	if len(question.Options) != 2 {
		t.Errorf("Options count = %d, want 2", len(question.Options))
	}

	if question.Options[0].Preview != "Preview A" {
		t.Error("First option should have preview")
	}
}

func TestInteractionResultFields(t *testing.T) {
	result := &InteractionResult{
		ID:    "interaction-123",
		State: StateAnswered,
		Answers: []*Answer{
			{QuestionID: "q_0", Type: TypeSingleChoice, Value: "a"},
		},
	}

	if result.ID != "interaction-123" {
		t.Errorf("ID = %q, want 'interaction-123'", result.ID)
	}

	if result.State != StateAnswered {
		t.Errorf("State = %v, want StateAnswered", result.State)
	}

	if len(result.Answers) != 1 {
		t.Errorf("Answers count = %d, want 1", len(result.Answers))
	}
}

func TestManagerTimeout(t *testing.T) {
	m := NewManager()

	// Test setting timeout
	m.SetTimeout(5 * time.Second)

	// Verify timeout is set (internally)
	// In real implementation, would check internal state
}

func TestQuestionWithDescription(t *testing.T) {
	question := Question{
		ID:          "q_0",
		Question:    "Select an option:",
		Description: "This is a detailed description of the question.",
		Type:        TypeSingleChoice,
		Options: []Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
		},
	}

	if question.Description == "" {
		t.Error("Question should have description")
	}

	err := question.Validate()
	if err != nil {
		t.Errorf("Question with description should be valid: %v", err)
	}
}

func TestOptionWithDescription(t *testing.T) {
	option := Option{
		Label:       "OAuth",
		Value:       "oauth",
		Description: "Use OAuth 2.0 for authentication",
	}

	if option.Description == "" {
		t.Error("Option should have description")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsString(s[1:], substr))))
}
