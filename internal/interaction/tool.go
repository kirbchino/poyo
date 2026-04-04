package interaction

import (
	"context"
	"encoding/json"
	"fmt"
)

// AskUserQuestionTool provides the tool interface for asking questions
type AskUserQuestionTool struct {
	manager *Manager
}

// NewAskUserQuestionTool creates a new ask user question tool
func NewAskUserQuestionTool(manager *Manager) *AskUserQuestionTool {
	return &AskUserQuestionTool{manager: manager}
}

// Name returns the tool name
func (t *AskUserQuestionTool) Name() string {
	return "AskUserQuestion"
}

// Description returns the tool description
func (t *AskUserQuestionTool) Description() string {
	return `Use this tool when you need to ask the user questions during execution.

This tool allows you to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices
4. Offer choices to the user about what direction to take.

When using this tool, you can ask 1-4 questions at a time. Each question
can be one of the following types:
- single: A single-select dropdown (2-4 options)
- multiple: A multi-select list (2-4 options)
- text: Free-form text input
- confirm: A yes/no confirmation

If you recommend a specific option, make that the first option in the list
and add "(Recommended)" at the end of the label.

IMPORTANT: Plan mode note: In plan mode, use this tool to clarify requirements
or choose between approaches BEFORE finalizing your plan. Do NOT use this tool
to ask "Is my plan ready?" or similar. Use ExitPlanMode instead.`
}

// InputSchema returns the JSON schema for the tool input
func (t *AskUserQuestionTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"questions": map[string]interface{}{
				"type":        "array",
				"description": "Questions to ask the user (1-4)",
				"minItems":    1,
				"maxItems":    4,
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"header": map[string]interface{}{
							"type":        "string",
							"description": "Very short label displayed as a chip/tag (max 12 chars). Examples: \"Auth method\", \"Library\", \"Approach\".",
							"maxLength":   12,
						},
						"question": map[string]interface{}{
							"type":        "string",
							"description": "The complete question to ask the user. Should be clear, specific, and end with a question mark.",
							"minLength":   1,
						},
						"multiSelect": map[string]interface{}{
							"type":        "boolean",
							"description": "Set to true to allow the user to select multiple options instead of just one.",
							"default":     false,
						},
						"options": map[string]interface{}{
							"type":        "array",
							"description": "The available choices for this question. Must have 2-4 options. Each option should be a distinct, mutually exclusive choice (unless multiSelect is enabled).",
							"minItems":    2,
							"maxItems":    4,
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"label": map[string]interface{}{
										"type":        "string",
										"description": "The display text for this option that the user will see and select. Should be concise (1-5 words) and clearly describe the choice.",
									},
									"description": map[string]interface{}{
										"type":        "string",
										"description": "Explanation of what this option means or what will happen if chosen. Useful for providing context about trade-offs or implications.",
									},
								},
								"required": []string{"label"},
							},
						},
					},
					"required": []string{"question", "header", "options", "multiSelect"},
				},
			},
		},
		"required": []string{"questions"},
	}
}

// Execute executes the tool
func (t *AskUserQuestionTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Questions []struct {
			Header      string `json:"header"`
			Question    string `json:"question"`
			MultiSelect bool   `json:"multiSelect"`
			Options     []struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"options"`
		} `json:"questions"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if len(params.Questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}

	if len(params.Questions) > 4 {
		return nil, fmt.Errorf("maximum 4 questions allowed")
	}

	// Convert to internal questions
	questions := make([]Question, len(params.Questions))
	for i, q := range params.Questions {
		question := Question{
			ID:          fmt.Sprintf("q_%d", i),
			Header:      q.Header,
			Question:    q.Question,
			MultiSelect: q.MultiSelect,
		}

		if q.MultiSelect {
			question.Type = TypeMultipleChoice
		} else {
			question.Type = TypeSingleChoice
		}

		// Convert options
		for _, opt := range q.Options {
			question.Options = append(question.Options, Option{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}

		// Validate
		if err := question.Validate(); err != nil {
			return nil, fmt.Errorf("invalid question %d: %w", i+1, err)
		}

		questions[i] = question
	}

	// Ask questions
	result, err := t.manager.Ask(ctx, questions)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &AskUserQuestionResult{
		InteractionID: result.InteractionID,
		State:         string(result.State),
		Answers:       make(map[string]interface{}),
	}

	for id, answer := range result.Answers {
		switch answer.Type {
		case TypeSingleChoice:
			response.Answers[id] = answer.Value
		case TypeMultipleChoice:
			response.Answers[id] = answer.Values
		case TypeTextInput:
			response.Answers[id] = answer.Text
		case TypeConfirmation:
			response.Answers[id] = answer.Confirmed
		}
	}

	return response, nil
}

// AskUserQuestionResult represents the result of asking questions
type AskUserQuestionResult struct {
	InteractionID string                 `json:"interactionId"`
	State         string                 `json:"state"`
	Answers       map[string]interface{} `json:"answers"`
	Error         string                 `json:"error,omitempty"`
}

// QuickAsk provides a simplified interface for asking a single question
func (t *AskUserQuestionTool) QuickAsk(ctx context.Context, header, question string, options []Option) (string, error) {
	questions := []Question{
		{
			ID:       "q_0",
			Header:   header,
			Question: question,
			Type:     TypeSingleChoice,
			Options:  options,
		},
	}

	result, err := t.manager.Ask(ctx, questions)
	if err != nil {
		return "", err
	}

	if result.State != StateAnswered {
		return "", fmt.Errorf("interaction not answered: %s", result.State)
	}

	answer, ok := result.Answers["q_0"]
	if !ok {
		return "", fmt.Errorf("no answer received")
	}

	return answer.Value, nil
}

// QuickConfirm provides a simplified interface for yes/no questions
func (t *AskUserQuestionTool) QuickConfirm(ctx context.Context, question string) (bool, error) {
	questions := []Question{
		{
			ID:       "q_0",
			Header:   "Confirm",
			Question: question,
			Type:     TypeSingleChoice,
			Options: []Option{
				{Label: "Yes", Value: "yes"},
				{Label: "No", Value: "no"},
			},
		},
	}

	result, err := t.manager.Ask(ctx, questions)
	if err != nil {
		return false, err
	}

	if result.State != StateAnswered {
		return false, fmt.Errorf("interaction not answered: %s", result.State)
	}

	answer, ok := result.Answers["q_0"]
	if !ok {
		return false, fmt.Errorf("no answer received")
	}

	return answer.Value == "yes", nil
}

// QuickText provides a simplified interface for text input
func (t *AskUserQuestionTool) QuickText(ctx context.Context, header, question string) (string, error) {
	questions := []Question{
		{
			ID:         "q_0",
			Header:     header,
			Question:   question,
			Type:       TypeTextInput,
			Required:   true,
		},
	}

	result, err := t.manager.Ask(ctx, questions)
	if err != nil {
		return "", err
	}

	if result.State != StateAnswered {
		return "", fmt.Errorf("interaction not answered: %s", result.State)
	}

	answer, ok := result.Answers["q_0"]
	if !ok {
		return "", fmt.Errorf("no answer received")
	}

	return answer.Text, nil
}
