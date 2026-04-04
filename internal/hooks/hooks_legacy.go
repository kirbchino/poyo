// Package hooks provides lifecycle hooks for extending Poyo functionality
//
// DEPRECATED: This file contains the legacy hook implementation.
// Use the new types and functions in types.go, config.go, executor.go instead.
package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LegacyHookType represents the type of hook (DEPRECATED: use HookEvent in types.go)
type LegacyHookType string

const (
	// LegacyHookTypePreToolUse is called before a tool is used
	LegacyHookTypePreToolUse LegacyHookType = "PreToolUse"
	// LegacyHookTypePostToolUse is called after a tool is used
	LegacyHookTypePostToolUse LegacyHookType = "PostToolUse"
	// LegacyHookTypePreQuery is called before a query is executed
	LegacyHookTypePreQuery LegacyHookType = "PreQuery"
	// LegacyHookTypePostQuery is called after a query is executed
	LegacyHookTypePostQuery LegacyHookType = "PostQuery"
	// LegacyHookTypeNotification is called for notifications
	LegacyHookTypeNotification LegacyHookType = "Notification"
	// LegacyHookTypeStop is called when the session stops
	LegacyHookTypeStop LegacyHookType = "Stop"
	// LegacyHookTypeSessionStart is called when a session starts
	LegacyHookTypeSessionStart LegacyHookType = "SessionStart"
	// LegacyHookTypeSessionEnd is called when a session ends
	LegacyHookTypeSessionEnd LegacyHookType = "SessionEnd"
)

// LegacyHook represents a legacy hook configuration (DEPRECATED: use types in types.go)
type LegacyHook struct {
	ID          string                 `json:"id"`
	Type        LegacyHookType         `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Command     string                 `json:"command"`
	Timeout     int                    `json:"timeout"` // seconds
	Environment map[string]string      `json:"environment,omitempty"`
	Conditions  []LegacyHookCondition  `json:"conditions,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// LegacyHookCondition represents a condition for hook execution
type LegacyHookCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, contains, matches
	Value    interface{} `json:"value"`
}

// LegacyHookContext contains context for hook execution
type LegacyHookContext struct {
	Type        LegacyHookType
	SessionID   string
	ToolName    string
	ToolInput   map[string]interface{}
	ToolOutput  interface{}
	Query       string
	QueryResult interface{}
	Notification string
	Metadata    map[string]interface{}
}

// LegacyHookResult contains the result of hook execution
type LegacyHookResult struct {
	HookID      string
	Success     bool
	Output      string
	Error       string
	Blocked     bool   // Whether to block the operation
	BlockReason string // Reason for blocking
	Duration    time.Duration
}

// LegacyHookRegistry manages all hooks (DEPRECATED)
type LegacyHookRegistry struct {
	mu    sync.RWMutex
	hooks map[LegacyHookType][]LegacyHook
}

// NewLegacyHookRegistry creates a new hook registry (DEPRECATED)
func NewLegacyHookRegistry() *LegacyHookRegistry {
	return &LegacyHookRegistry{
		hooks: make(map[LegacyHookType][]LegacyHook),
	}
}

// Register registers a hook
func (r *LegacyHookRegistry) Register(hook LegacyHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[hook.Type] = append(r.hooks[hook.Type], hook)
}

// Unregister unregisters a hook
func (r *LegacyHookRegistry) Unregister(hookID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for hookType, hooks := range r.hooks {
		for i, h := range hooks {
			if h.ID == hookID {
				r.hooks[hookType] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
	}
}

// GetHooks gets all hooks for a type
func (r *LegacyHookRegistry) GetHooks(hookType LegacyHookType) []LegacyHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hooks[hookType]
}

// LoadFromConfig loads hooks from configuration
func (r *LegacyHookRegistry) LoadFromConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var config struct {
		Hooks []LegacyHook `json:"hooks"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	for _, hook := range config.Hooks {
		r.Register(hook)
	}

	return nil
}

// LegacyHookExecutor executes hooks (DEPRECATED)
type LegacyHookExecutor struct {
	registry   *LegacyHookRegistry
	workingDir string
	timeout    time.Duration
}

// NewLegacyHookExecutor creates a new hook executor (DEPRECATED)
func NewLegacyHookExecutor(registry *LegacyHookRegistry, workingDir string) *LegacyHookExecutor {
	return &LegacyHookExecutor{
		registry:   registry,
		workingDir: workingDir,
		timeout:    30 * time.Second,
	}
}

// Execute executes hooks for a given type
func (e *LegacyHookExecutor) Execute(ctx context.Context, hookType LegacyHookType, hookCtx *LegacyHookContext) []LegacyHookResult {
	hooks := e.registry.GetHooks(hookType)
	results := make([]LegacyHookResult, 0)

	for _, hook := range hooks {
		if !hook.Enabled {
			continue
		}

		// Check conditions
		if !e.matchesConditions(hook, hookCtx) {
			continue
		}

		result := e.executeHook(ctx, hook, hookCtx)
		results = append(results, result)
	}

	return results
}

// executeHook executes a single hook
func (e *LegacyHookExecutor) executeHook(ctx context.Context, hook LegacyHook, hookCtx *LegacyHookContext) LegacyHookResult {
	start := time.Now()
	result := LegacyHookResult{
		HookID: hook.ID,
	}

	// Set timeout
	timeout := e.timeout
	if hook.Timeout > 0 {
		timeout = time.Duration(hook.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = e.workingDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range hook.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Add hook context as environment variables
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("HOOK_TYPE=%s", hook.Type),
		fmt.Sprintf("HOOK_ID=%s", hook.ID),
		fmt.Sprintf("SESSION_ID=%s", hookCtx.SessionID),
	)
	if hookCtx.ToolName != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TOOL_NAME=%s", hookCtx.ToolName))
	}
	if hookCtx.Query != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("QUERY=%s", hookCtx.Query))
	}

	// Pass context as JSON via stdin
	contextJSON, _ := json.Marshal(hookCtx)
	cmd.Stdin = strings.NewReader(string(contextJSON))

	// Execute
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	// Parse output for block signals
	if strings.Contains(result.Output, "[BLOCK]") {
		result.Blocked = true
		// Extract block reason
		lines := strings.Split(result.Output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "[BLOCK]") {
				result.BlockReason = strings.TrimSpace(strings.TrimPrefix(line, "[BLOCK]"))
				break
			}
		}
	}

	return result
}

// matchesConditions checks if hook conditions match
func (e *LegacyHookExecutor) matchesConditions(hook LegacyHook, ctx *LegacyHookContext) bool {
	if len(hook.Conditions) == 0 {
		return true
	}

	for _, cond := range hook.Conditions {
		value := e.getFieldValue(ctx, cond.Field)
		if !e.matchCondition(value, cond) {
			return false
		}
	}

	return true
}

// getFieldValue gets a field value from context
func (e *LegacyHookExecutor) getFieldValue(ctx *LegacyHookContext, field string) interface{} {
	switch field {
	case "tool_name":
		return ctx.ToolName
	case "session_id":
		return ctx.SessionID
	case "query":
		return ctx.Query
	default:
		if ctx.Metadata != nil {
			return ctx.Metadata[field]
		}
		return nil
	}
}

// matchCondition matches a value against a condition
func (e *LegacyHookExecutor) matchCondition(value interface{}, cond LegacyHookCondition) bool {
	switch cond.Operator {
	case "eq":
		return value == cond.Value
	case "ne":
		return value != cond.Value
	case "contains":
		if str, ok := value.(string); ok {
			if pattern, ok := cond.Value.(string); ok {
				return strings.Contains(str, pattern)
			}
		}
	case "matches":
		if str, ok := value.(string); ok {
			if pattern, ok := cond.Value.(string); ok {
				matched, _ := filepath.Match(pattern, str)
				return matched
			}
		}
	}
	return false
}

// LegacyCheckBlock checks if any hook results block the operation (DEPRECATED)
func LegacyCheckBlock(results []LegacyHookResult) (bool, string) {
	for _, r := range results {
		if r.Blocked {
			return true, r.BlockReason
		}
	}
	return false, ""
}

// DefaultLegacyHookConfig returns default hook configuration (DEPRECATED)
func DefaultLegacyHookConfig() map[string]interface{} {
	return map[string]interface{}{
		"hooks": []LegacyHook{
			{
				ID:      "pre_tool_use_example",
				Type:    LegacyHookTypePreToolUse,
				Enabled: false,
				Command: "echo 'Tool ${TOOL_NAME} is about to be used'",
				Timeout: 5,
			},
			{
				ID:      "post_query_example",
				Type:    LegacyHookTypePostQuery,
				Enabled: false,
				Command: "echo 'Query completed'",
				Timeout: 5,
			},
		},
	}
}

// LegacyHookManager manages hook lifecycle (DEPRECATED)
type LegacyHookManager struct {
	registry  *LegacyHookRegistry
	executor  *LegacyHookExecutor
	configDir string
}

// NewLegacyHookManager creates a new hook manager (DEPRECATED)
func NewLegacyHookManager(configDir string) *LegacyHookManager {
	registry := NewLegacyHookRegistry()
	executor := NewLegacyHookExecutor(registry, configDir)

	return &LegacyHookManager{
		registry:  registry,
		executor:  executor,
		configDir: configDir,
	}
}

// Initialize initializes hooks from configuration
func (m *LegacyHookManager) Initialize() error {
	configPath := filepath.Join(m.configDir, "hooks.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		defaultConfig := DefaultLegacyHookConfig()
		data, _ := json.MarshalIndent(defaultConfig, "", "  ")
		os.WriteFile(configPath, data, 0644)
		return nil
	}

	return m.registry.LoadFromConfig(configPath)
}

// Execute executes hooks
func (m *LegacyHookManager) Execute(ctx context.Context, hookType LegacyHookType, hookCtx *LegacyHookContext) []LegacyHookResult {
	return m.executor.Execute(ctx, hookType, hookCtx)
}

// AddHook adds a hook
func (m *LegacyHookManager) AddHook(hook LegacyHook) {
	m.registry.Register(hook)
}

// RemoveHook removes a hook
func (m *LegacyHookManager) RemoveHook(hookID string) {
	m.registry.Unregister(hookID)
}

// ListHooks lists all hooks
func (m *LegacyHookManager) ListHooks() map[LegacyHookType][]LegacyHook {
	m.registry.mu.RLock()
	defer m.registry.mu.RUnlock()

	result := make(map[LegacyHookType][]LegacyHook)
	for k, v := range m.registry.hooks {
		result[k] = v
	}
	return result
}
