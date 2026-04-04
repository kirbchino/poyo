package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Executor executes hooks
type Executor struct {
	mu             sync.RWMutex
	configManager  *ConfigManager
	asyncRegistry  *AsyncRegistry
	eventHandlers  []HookEventHandler
	env            map[string]string
}

// NewExecutor creates a new hook executor
func NewExecutor(configManager *ConfigManager) *Executor {
	return &Executor{
		configManager: configManager,
		asyncRegistry: NewAsyncRegistry(),
		env:          make(map[string]string),
	}
}

// SetEnvironment sets environment variables for hook execution
func (e *Executor) SetEnvironment(env map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.env = env
}

// RegisterEventHandler registers a handler for hook execution events
func (e *Executor) RegisterEventHandler(handler HookEventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventHandlers = append(e.eventHandlers, handler)
}

// emitEvent emits a hook execution event to all handlers
func (e *Executor) emitEvent(event HookExecutionEvent) {
	e.mu.RLock()
	handlers := make([]HookEventHandler, len(e.eventHandlers))
	copy(handlers, e.eventHandlers)
	e.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// ExecuteHooks executes all matching hooks for an event
func (e *Executor) ExecuteHooks(ctx context.Context, event HookEvent, input *HookInput) ([]*HookOutput, error) {
	// Check if hooks are disabled
	if e.configManager.ShouldDisableAllHooks() {
		return nil, nil
	}

	// Check managed-only mode
	if e.configManager.ShouldAllowManagedHooksOnly() {
		// Filter to only managed hooks
		return e.executeManagedHooks(ctx, event, input)
	}

	// Get matching hooks
	matchers := e.configManager.GetHooks(event)
	if len(matchers) == 0 {
		return nil, nil
	}

	var outputs []*HookOutput
	var toRemove []string // Hooks to remove after execution

	for _, matcher := range matchers {
		// Check matcher
		if !e.matcherMatches(matcher.Matcher, input) {
			continue
		}

		for _, hook := range matcher.Hooks {
			// Check if condition
			if !e.conditionMatches(hook, input) {
				continue
			}

			// Execute hook
			output, err := e.executeHook(ctx, hook, input)
			if err != nil {
				e.emitEvent(HookExecutionEvent{
					Type:      event,
					HookID:    hook.GetID(),
					HookEvent: event,
					Outcome:   "error",
				})
				continue
			}

			// Track hooks to remove
			if hook.IsOnce() {
				toRemove = append(toRemove, hook.GetID())
			}

			outputs = append(outputs, output)

			// Check for blocking
			if output != nil && !output.Continue {
				break
			}
		}
	}

	// Remove once hooks
	for _, id := range toRemove {
		e.configManager.RemoveSessionHook(event, id)
	}

	return outputs, nil
}

// executeManagedHooks executes only managed (policy) hooks
func (e *Executor) executeManagedHooks(ctx context.Context, event HookEvent, input *HookInput) ([]*HookOutput, error) {
	matchers := e.configManager.GetHooks(event)
	var outputs []*HookOutput

	for _, matcher := range matchers {
		for _, hook := range matcher.Hooks {
			var source HookSource
			switch h := hook.(type) {
			case CommandHook:
				source = h.Source
			case PromptHook:
				source = h.Source
			case AgentHook:
				source = h.Source
			case HTTPHook:
				source = h.Source
			case CallbackHook:
				source = h.Source
			}

			if source != SourcePolicySettings {
				continue
			}

			if !e.matcherMatches(matcher.Matcher, input) {
				continue
			}

			output, err := e.executeHook(ctx, hook, input)
			if err != nil {
				continue
			}
			outputs = append(outputs, output)
		}
	}

	return outputs, nil
}

// matcherMatches checks if input matches a hook matcher
func (e *Executor) matcherMatches(matcher string, input *HookInput) bool {
	if matcher == "" {
		return true
	}

	// Try to match against tool name, notification type, etc.
	if input.ToolName != "" && MatchesMatcher(matcher, input.ToolName) {
		return true
	}
	if input.NotificationType != "" && MatchesMatcher(matcher, input.NotificationType) {
		return true
	}
	if input.Source != "" && MatchesMatcher(matcher, input.Source) {
		return true
	}
	if input.AgentType != "" && MatchesMatcher(matcher, input.AgentType) {
		return true
	}
	if input.MCPServerName != "" && MatchesMatcher(matcher, input.MCPServerName) {
		return true
	}

	return false
}

// conditionMatches checks if a hook's condition matches
func (e *Executor) conditionMatches(hook Hook, input *HookInput) bool {
	var condition string
	switch h := hook.(type) {
	case CommandHook:
		condition = h.If
	case PromptHook:
		condition = h.If
	case AgentHook:
		condition = h.If
	case HTTPHook:
		condition = h.If
	case CallbackHook:
		condition = h.If
	}

	if condition == "" {
		return true
	}

	// TODO: Implement permission rule syntax parsing
	// For now, simple substring matching
	return strings.Contains(fmt.Sprintf("%v", input), condition)
}

// executeHook executes a single hook
func (e *Executor) executeHook(ctx context.Context, hook Hook, input *HookInput) (*HookOutput, error) {
	startTime := time.Now()

	e.emitEvent(HookExecutionEvent{
		Type:      EventSessionStart, // Using as "started" type
		HookID:    hook.GetID(),
		HookEvent: input.Event,
	})

	timeout := GetTimeoutDuration(hook)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var output *HookOutput
	var err error

	switch h := hook.(type) {
	case CommandHook:
		output, err = e.executeCommandHook(ctx, h, input)
	case PromptHook:
		output, err = e.executePromptHook(ctx, h, input)
	case AgentHook:
		output, err = e.executeAgentHook(ctx, h, input)
	case HTTPHook:
		output, err = e.executeHTTPHook(ctx, h, input)
	case CallbackHook:
		output, err = h.Callback(ctx, input)
	default:
		err = fmt.Errorf("unknown hook type: %T", hook)
	}

	duration := time.Since(startTime)

	if err != nil {
		e.emitEvent(HookExecutionEvent{
			Type:      EventSessionEnd,
			HookID:    hook.GetID(),
			HookEvent: input.Event,
			Outcome:   "error",
			Duration:  duration,
		})
		return nil, err
	}

	outcome := "success"
	if output != nil && !output.Continue {
		outcome = "blocked"
	}

	e.emitEvent(HookExecutionEvent{
		Type:      EventSessionEnd,
		HookID:    hook.GetID(),
		HookEvent: input.Event,
		Output:    output,
		Outcome:   outcome,
		Duration:  duration,
	})

	return output, nil
}

// executeCommandHook executes a shell command hook
func (e *Executor) executeCommandHook(ctx context.Context, hook CommandHook, input *HookInput) (*HookOutput, error) {
	// Prepare command
	cmdStr := e.expandVariables(hook.Command, input)

	// Determine shell
	shell := hook.Shell
	if shell == "" {
		shell = "bash"
	}

	// Prepare environment
	env := os.Environ()
	e.mu.RLock()
	for k, v := range e.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	e.mu.RUnlock()

	// Add hook-specific environment
	env = append(env,
		fmt.Sprintf("CLAUDE_PROJECT_DIR=%s", input.ProjectDir),
		fmt.Sprintf("CLAUDE_SESSION_ID=%s", input.SessionID),
	)

	// Execute command
	var cmd *exec.Cmd
	if shell == "powershell" {
		cmd = exec.CommandContext(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", cmdStr)
	}
	cmd.Env = env
	cmd.Dir = input.ProjectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Check for async detection
	if hook.Async {
		return e.executeAsyncCommand(ctx, hook, cmd, input)
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 2 means block
			if exitErr.ExitCode() == 2 {
				return &HookOutput{
					Continue: false,
					Reason:   stderr.String(),
				}, nil
			}
		}
		return nil, fmt.Errorf("hook command failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse output
	output := stdout.String()
	if output == "" {
		return &HookOutput{Continue: true}, nil
	}

	// Try to parse as JSON
	var hookOutput HookOutput
	if err := json.Unmarshal([]byte(output), &hookOutput); err != nil {
		// Not JSON, treat as simple output
		return &HookOutput{
			Continue: true,
			Reason:   output,
		}, nil
	}

	return &hookOutput, nil
}

// executeAsyncCommand executes a command hook asynchronously
func (e *Executor) executeAsyncCommand(ctx context.Context, hook CommandHook, cmd *exec.Cmd, input *HookInput) (*HookOutput, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start async hook: %w", err)
	}

	// Register in async registry
	processID := e.asyncRegistry.Register(AsyncHookInfo{
		HookID:     hook.GetID(),
		HookEvent:  input.Event,
		ToolName:   input.ToolName,
		Command:    hook.Command,
		StartTime:  time.Now(),
		Timeout:    GetTimeoutDuration(hook),
	})

	// Return async indicator
	return &HookOutput{
		Async:        true,
		AsyncTimeout: GetTimeoutDuration(hook),
		HookSpecific: map[string]interface{}{
			"processId": processID,
		},
	}, nil
}

// executePromptHook executes an LLM prompt hook
func (e *Executor) executePromptHook(ctx context.Context, hook PromptHook, input *HookInput) (*HookOutput, error) {
	// TODO: Implement LLM prompt execution
	// This requires integration with the model client

	// Replace $ARGUMENTS placeholder
	prompt := strings.ReplaceAll(hook.Prompt, "$ARGUMENTS", input.Arguments)

	_ = prompt // Placeholder for LLM call

	return &HookOutput{Continue: true}, nil
}

// executeAgentHook executes a multi-turn agent hook
func (e *Executor) executeAgentHook(ctx context.Context, hook AgentHook, input *HookInput) (*HookOutput, error) {
	// TODO: Implement agent hook execution
	// This requires:
	// 1. Creating a dedicated agent session
	// 2. Injecting StructuredOutputTool
	// 3. Running multi-turn conversation
	// 4. Waiting for structured output

	_ = hook
	_ = input

	return &HookOutput{Continue: true}, nil
}

// executeHTTPHook executes an HTTP POST hook
func (e *Executor) executeHTTPHook(ctx context.Context, hook HTTPHook, input *HookInput) (*HookOutput, error) {
	executor := NewHTTPHookExecutor()

	// Set environment variables
	e.mu.RLock()
	envVars := make(map[string]string)
	for k, v := range e.env {
		envVars[k] = v
	}
	e.mu.RUnlock()
	executor.SetEnvVars(envVars)

	return executor.Execute(ctx, hook, input)
}

// expandVariables expands variables in a command string
func (e *Executor) expandVariables(cmd string, input *HookInput) string {
	// Expand common variables
	cmd = strings.ReplaceAll(cmd, "${CLAUDE_PROJECT_DIR}", input.ProjectDir)
	cmd = strings.ReplaceAll(cmd, "${CLAUDE_SESSION_ID}", input.SessionID)
	cmd = strings.ReplaceAll(cmd, "$TOOL_NAME", input.ToolName)
	cmd = strings.ReplaceAll(cmd, "$TOOL_USE_ID", input.ToolUseID)

	// Expand environment variables
	e.mu.RLock()
	for k, v := range e.env {
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("${%s}", k), v)
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("$%s", k), v)
	}
	e.mu.RUnlock()

	return cmd
}

// CheckAsyncHooks checks for completed async hooks
func (e *Executor) CheckAsyncHooks() []*AsyncHookResult {
	return e.asyncRegistry.CheckCompleted()
}

// FinalizeAsyncHooks cleans up all pending async hooks
func (e *Executor) FinalizeAsyncHooks() {
	e.asyncRegistry.Finalize()
}

// AsyncHookInfo contains information about a pending async hook
type AsyncHookInfo struct {
	HookID    string
	HookEvent HookEvent
	ToolName  string
	Command   string
	StartTime time.Time
	Timeout   time.Duration
}

// AsyncHookResult contains the result of a completed async hook
type AsyncHookResult struct {
	ProcessID string
	HookID    string
	Output    *HookOutput
	Error     error
}
