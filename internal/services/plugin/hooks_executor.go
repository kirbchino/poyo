// Package plugin provides hook execution for plugins
package plugin

import (
	"context"
	"sort"
	"sync"
)

// HookType represents the type of hook
type HookType string

const (
	HookPreToolUse  HookType = "PreToolUse"
	HookPostToolUse HookType = "PostToolUse"
	HookPrePrompt   HookType = "PrePrompt"
	HookPostPrompt  HookType = "PostPrompt"
	HookOnError     HookType = "OnError"
	HookOnStart     HookType = "OnStart"
	HookOnEnd       HookType = "OnEnd"
	HookOnLoad      HookType = "OnLoad"
	HookOnUnload    HookType = "OnUnload"
)

// HookResult represents the result of a hook execution
type HookResult struct {
	Blocked     bool                   `json:"blocked"`
	Reason      string                 `json:"reason,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Modified    bool                   `json:"modified,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	SkipDefault bool                   `json:"skipDefault,omitempty"`
}

// HookInput represents input to a hook
type HookInput struct {
	Tool    string                 `json:"tool,omitempty"`
	Input   map[string]interface{} `json:"input,omitempty"`
	Success bool                   `json:"success,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Prompt  string                 `json:"prompt,omitempty"`
	Result  interface{}            `json:"result,omitempty"`
}

// HookExecutor manages hook execution
type HookExecutor struct {
	mu      sync.RWMutex
	manager *PluginManager
}

// NewHookExecutor creates a new hook executor
func NewHookExecutor(manager *PluginManager) *HookExecutor {
	return &HookExecutor{
		manager: manager,
	}
}

// ExecuteHook executes hooks of a specific type with priority ordering
func (e *HookExecutor) ExecuteHook(ctx context.Context, hookType HookType, input HookInput) (*HookResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get all enabled plugins with this hook type
	var hooks []struct {
		plugin   *Plugin
		priority int
	}

	for _, plugin := range e.manager.GetEnabled() {
		for _, hookDef := range plugin.Hooks {
			if hookDef.Type == string(hookType) {
				hooks = append(hooks, struct {
					plugin   *Plugin
					priority int
				}{plugin: plugin, priority: hookDef.Priority})
			}
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].priority > hooks[j].priority
	})

	// Execute hooks in order
	var lastResult *HookResult
	for _, h := range hooks {
		result, err := e.executePluginHook(ctx, h.plugin, string(hookType), input)
		if err != nil {
			// Log error but continue
			continue
		}

		if result != nil {
			lastResult = result

			// If blocked, stop execution
			if result.Blocked {
				return result, nil
			}

			// If modified, update input for next hook
			if result.Modified && result.Data != nil {
				if input.Input == nil {
					input.Input = make(map[string]interface{})
				}
				for k, v := range result.Data {
					input.Input[k] = v
				}
			}
		}
	}

	return lastResult, nil
}

// executePluginHook executes a hook on a specific plugin
func (e *HookExecutor) executePluginHook(ctx context.Context, plugin *Plugin, hookName string, input HookInput) (*HookResult, error) {
	handler, ok := e.manager.handlers[string(plugin.Type)]
	if !ok {
		return nil, nil
	}

	// Convert input to map
	inputMap := map[string]interface{}{
		"tool":    input.Tool,
		"input":   input.Input,
		"success": input.Success,
		"error":   input.Error,
		"prompt":  input.Prompt,
		"result":  input.Result,
	}

	result, err := handler.Execute(ctx, plugin, hookName, inputMap)
	if err != nil {
		return nil, err
	}

	// Convert result to HookResult
	if result == nil {
		return &HookResult{Blocked: false}, nil
	}

	if hr, ok := result.(*HookResult); ok {
		return hr, nil
	}

	// Try to convert from map
	if m, ok := result.(map[string]interface{}); ok {
		hr := &HookResult{}
		if v, ok := m["blocked"].(bool); ok {
			hr.Blocked = v
		}
		if v, ok := m["reason"].(string); ok {
			hr.Reason = v
		}
		if v, ok := m["message"].(string); ok {
			hr.Message = v
		}
		if v, ok := m["modified"].(bool); ok {
			hr.Modified = v
		}
		if v, ok := m["data"].(map[string]interface{}); ok {
			hr.Data = v
		}
		return hr, nil
	}

	return &HookResult{Blocked: false}, nil
}

// PreToolUse executes PreToolUse hooks
func (e *HookExecutor) PreToolUse(ctx context.Context, tool string, input map[string]interface{}) (*HookResult, error) {
	return e.ExecuteHook(ctx, HookPreToolUse, HookInput{
		Tool:  tool,
		Input: input,
	})
}

// PostToolUse executes PostToolUse hooks
func (e *HookExecutor) PostToolUse(ctx context.Context, tool string, input map[string]interface{}, success bool, err string, result interface{}) (*HookResult, error) {
	return e.ExecuteHook(ctx, HookPostToolUse, HookInput{
		Tool:    tool,
		Input:   input,
		Success: success,
		Error:   err,
		Result:  result,
	})
}
