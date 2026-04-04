package hooks

import (
	"context"
)

// ToolHooksContext contains context for tool-related hooks
type ToolHooksContext struct {
	SessionID      string
	ProjectDir     string
	PermissionMode string
}

// RunPreToolUseHooks executes PreToolUse hooks for a tool call
func RunPreToolUseHooks(ctx context.Context, executor *Executor, toolCtx *ToolHooksContext, toolName string, input map[string]interface{}) (*PreToolUseResult, error) {
	hookInput := &HookInput{
		Event:      EventPreToolUse,
		ToolName:   toolName,
		Input:      input,
		SessionID:  toolCtx.SessionID,
		ProjectDir: toolCtx.ProjectDir,
	}

	outputs, err := executor.ExecuteHooks(ctx, EventPreToolUse, hookInput)
	if err != nil {
		return nil, err
	}

	result := &PreToolUseResult{
		Continue: true,
	}

	for _, output := range outputs {
		if output == nil {
			continue
		}

		if !output.Continue {
			result.Continue = false
			result.BlockReason = output.StopReason
			if output.Reason != "" {
				result.BlockReason = output.Reason
			}
			break
		}

		// Handle permission decision
		if output.Decision == DecisionBlock {
			result.PermissionDecision = DecisionBlock
			result.PermissionReason = output.Reason
		} else if output.Decision == DecisionApprove {
			result.PermissionDecision = DecisionApprove
		}

		// Handle input modification
		if output.UpdatedInput != nil {
			result.UpdatedInput = output.UpdatedInput
		}

		// Handle additional context
		if output.AdditionalContext != "" {
			result.AdditionalContext = append(result.AdditionalContext, output.AdditionalContext)
		}

		// Handle system message
		if output.SystemMessage != "" {
			result.SystemMessages = append(result.SystemMessages, output.SystemMessage)
		}
	}

	return result, nil
}

// RunPostToolUseHooks executes PostToolUse hooks after a tool call
func RunPostToolUseHooks(ctx context.Context, executor *Executor, toolCtx *ToolHooksContext, toolName string, toolUseID string, input map[string]interface{}, output interface{}, execErr error) (*PostToolUseResult, error) {
	event := EventPostToolUse
	if execErr != nil {
		event = EventPostToolUseFailure
	}

	hookInput := &HookInput{
		Event:     event,
		ToolName:  toolName,
		ToolUseID: toolUseID,
		Input:     input,
		Output:    output,
		SessionID: toolCtx.SessionID,
		ProjectDir: toolCtx.ProjectDir,
	}

	if execErr != nil {
		hookInput.Error = execErr.Error()
	}

	outputs, err := executor.ExecuteHooks(ctx, event, hookInput)
	if err != nil {
		return nil, err
	}

	result := &PostToolUseResult{
		Continue: true,
	}

	for _, out := range outputs {
		if out == nil {
			continue
		}

		if !out.Continue {
			result.Continue = false
			result.BlockReason = out.StopReason
			if out.Reason != "" {
				result.BlockReason = out.Reason
			}
			break
		}

		// Handle additional context
		if out.AdditionalContext != "" {
			result.AdditionalContext = append(result.AdditionalContext, out.AdditionalContext)
		}

		// Handle output modification
		if hookSpecific, ok := out.HookSpecific["updatedOutput"]; ok {
			result.UpdatedOutput = hookSpecific
		}
	}

	return result, nil
}

// RunSessionStartHooks executes SessionStart hooks
func RunSessionStartHooks(ctx context.Context, executor *Executor, sessionID, projectDir, source string) error {
	hookInput := &HookInput{
		Event:      EventSessionStart,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		Source:     source,
	}

	_, err := executor.ExecuteHooks(ctx, EventSessionStart, hookInput)
	return err
}

// RunSessionEndHooks executes SessionEnd hooks
func RunSessionEndHooks(ctx context.Context, executor *Executor, sessionID, projectDir, reason string) error {
	hookInput := &HookInput{
		Event:      EventSessionEnd,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		Reason:     reason,
	}

	_, err := executor.ExecuteHooks(ctx, EventSessionEnd, hookInput)
	return err
}

// RunStopHooks executes Stop hooks before model response ends
func RunStopHooks(ctx context.Context, executor *Executor, sessionID, projectDir string) error {
	hookInput := &HookInput{
		Event:      EventStop,
		SessionID:  sessionID,
		ProjectDir: projectDir,
	}

	_, err := executor.ExecuteHooks(ctx, EventStop, hookInput)
	return err
}

// RunPreCompactHooks executes PreCompact hooks before conversation compression
func RunPreCompactHooks(ctx context.Context, executor *Executor, sessionID, projectDir, trigger string) error {
	hookInput := &HookInput{
		Event:      EventPreCompact,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		Trigger:    trigger,
	}

	_, err := executor.ExecuteHooks(ctx, EventPreCompact, hookInput)
	return err
}

// RunPostCompactHooks executes PostCompact hooks after conversation compression
func RunPostCompactHooks(ctx context.Context, executor *Executor, sessionID, projectDir, trigger string) error {
	hookInput := &HookInput{
		Event:      EventPostCompact,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		Trigger:    trigger,
	}

	_, err := executor.ExecuteHooks(ctx, EventPostCompact, hookInput)
	return err
}

// RunSubagentStartHooks executes SubagentStart hooks
func RunSubagentStartHooks(ctx context.Context, executor *Executor, sessionID, projectDir, agentType string) error {
	hookInput := &HookInput{
		Event:      EventSubagentStart,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		AgentType:  agentType,
	}

	_, err := executor.ExecuteHooks(ctx, EventSubagentStart, hookInput)
	return err
}

// RunSubagentStopHooks executes SubagentStop hooks
func RunSubagentStopHooks(ctx context.Context, executor *Executor, sessionID, projectDir, agentType string) error {
	hookInput := &HookInput{
		Event:      EventSubagentStop,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		AgentType:  agentType,
	}

	_, err := executor.ExecuteHooks(ctx, EventSubagentStop, hookInput)
	return err
}

// RunPermissionRequestHooks executes PermissionRequest hooks
func RunPermissionRequestHooks(ctx context.Context, executor *Executor, sessionID, projectDir, toolName string) error {
	hookInput := &HookInput{
		Event:      EventPermissionRequest,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		ToolName:   toolName,
	}

	_, err := executor.ExecuteHooks(ctx, EventPermissionRequest, hookInput)
	return err
}

// RunUserPromptSubmitHooks executes UserPromptSubmit hooks
func RunUserPromptSubmitHooks(ctx context.Context, executor *Executor, sessionID, projectDir string) error {
	hookInput := &HookInput{
		Event:      EventUserPromptSubmit,
		SessionID:  sessionID,
		ProjectDir: projectDir,
	}

	_, err := executor.ExecuteHooks(ctx, EventUserPromptSubmit, hookInput)
	return err
}

// RunCwdChangedHooks executes CwdChanged hooks
func RunCwdChangedHooks(ctx context.Context, executor *Executor, sessionID, oldCwd, newCwd string) error {
	hookInput := &HookInput{
		Event:      EventCwdChanged,
		SessionID:  sessionID,
		ProjectDir: newCwd,
		OldCwd:     oldCwd,
		NewCwd:     newCwd,
	}

	_, err := executor.ExecuteHooks(ctx, EventCwdChanged, hookInput)
	return err
}

// PreToolUseResult contains the result of PreToolUse hooks
type PreToolUseResult struct {
	Continue           bool
	BlockReason        string
	PermissionDecision Decision
	PermissionReason   string
	UpdatedInput       map[string]interface{}
	AdditionalContext  []string
	SystemMessages     []string
}

// PostToolUseResult contains the result of PostToolUse hooks
type PostToolUseResult struct {
	Continue          bool
	BlockReason       string
	AdditionalContext []string
	UpdatedOutput     interface{}
}
