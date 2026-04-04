package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ConfigManager manages hook configurations from multiple sources
type ConfigManager struct {
	mu          sync.RWMutex
	settings    HooksSettings
	snapshot    *ConfigSnapshot
	sources     map[HookSource]bool
	priorities  map[HookSource]int // Lower number = higher priority
}

// ConfigSnapshot captures hook state at session start
type ConfigSnapshot struct {
	Settings     HooksSettings `json:"settings"`
	ManagedOnly  bool          `json:"managedOnly"`
	DisableAll   bool          `json:"disableAll"`
	TrustLevel   string        `json:"trustLevel"`
}

// NewConfigManager creates a new hook configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		settings: make(HooksSettings),
		sources:  make(map[HookSource]bool),
		priorities: map[HookSource]int{
			SourcePolicySettings:  1, // Highest priority
			SourceBuiltinHook:     2,
			SourcePluginHook:      3,
			SourceUserSettings:    4,
			SourceProjectSettings: 5,
			SourceLocalSettings:   6,
			SourceSessionHook:     7, // Lowest priority
		},
	}
}

// LoadFromSettings loads hooks from a settings map
func (m *ConfigManager) LoadFromSettings(source HookSource, settings map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hooksSettings, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil // No hooks configured
	}

	for eventStr, matchers := range hooksSettings {
		event := HookEvent(eventStr)
		matcherConfigs, ok := matchers.([]interface{})
		if !ok {
			continue
		}

		for _, mc := range matcherConfigs {
			matcherConfig, ok := mc.(map[string]interface{})
			if !ok {
				continue
			}

			hookMatcher := HookMatcherConfig{}
			if matcher, ok := matcherConfig["matcher"].(string); ok {
				hookMatcher.Matcher = matcher
			}

			hooksRaw, ok := matcherConfig["hooks"].([]interface{})
			if !ok {
				continue
			}

			for _, h := range hooksRaw {
				hookMap, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				hook, err := parseHook(hookMap, source)
				if err != nil {
					continue
				}
				if hook != nil {
					hookMatcher.Hooks = append(hookMatcher.Hooks, hook)
				}
			}

			if len(hookMatcher.Hooks) > 0 {
				m.settings[event] = append(m.settings[event], hookMatcher)
			}
		}
	}

	m.sources[source] = true
	return nil
}

// parseHook parses a hook from a map
func parseHook(m map[string]interface{}, source HookSource) (Hook, error) {
	hookType, ok := m["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing hook type")
	}

	// Parse common fields
	base := BaseHook{
		Source: source,
	}
	if id, ok := m["id"].(string); ok {
		base.ID = id
	}
	if cond, ok := m["if"].(string); ok {
		base.If = cond
	}
	if timeout, ok := m["timeout"].(float64); ok {
		base.Timeout = int(timeout)
	}
	if msg, ok := m["statusMessage"].(string); ok {
		base.StatusMessage = msg
	}
	if once, ok := m["once"].(bool); ok {
		base.Once = once
	}
	if async, ok := m["async"].(bool); ok {
		base.Async = async
	}
	if asyncRewake, ok := m["asyncRewake"].(bool); ok {
		base.AsyncRewake = asyncRewake
	}

	switch HookType(hookType) {
	case HookTypeCommand:
		hook := CommandHook{BaseHook: base}
		if cmd, ok := m["command"].(string); ok {
			hook.Command = cmd
		}
		if shell, ok := m["shell"].(string); ok {
			hook.Shell = shell
		}
		return hook, nil

	case HookTypePrompt:
		hook := PromptHook{BaseHook: base}
		if prompt, ok := m["prompt"].(string); ok {
			hook.Prompt = prompt
		}
		if model, ok := m["model"].(string); ok {
			hook.Model = model
		}
		return hook, nil

	case HookTypeAgent:
		hook := AgentHook{BaseHook: base}
		if prompt, ok := m["prompt"].(string); ok {
			hook.Prompt = prompt
		}
		if model, ok := m["model"].(string); ok {
			hook.Model = model
		}
		return hook, nil

	case HookTypeHTTP:
		hook := HTTPHook{BaseHook: base}
		if url, ok := m["url"].(string); ok {
			hook.URL = url
		}
		if headers, ok := m["headers"].(map[string]interface{}); ok {
			hook.Headers = make(map[string]string)
			for k, v := range headers {
				if vs, ok := v.(string); ok {
					hook.Headers[k] = vs
				}
			}
		}
		if allowed, ok := m["allowedEnvVars"].([]interface{}); ok {
			for _, a := range allowed {
				if as, ok := a.(string); ok {
					hook.AllowedEnvVars = append(hook.AllowedEnvVars, as)
				}
			}
		}
		return hook, nil

	default:
		return nil, fmt.Errorf("unknown hook type: %s", hookType)
	}
}

// LoadFromFile loads hooks from a JSON file
func (m *ConfigManager) LoadFromFile(source HookSource, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, not an error
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return m.LoadFromSettings(source, settings)
}

// LoadUserSettings loads hooks from user-level settings
func (m *ConfigManager) LoadUserSettings() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return m.LoadFromFile(SourceUserSettings, filepath.Join(homeDir, ".claude", "settings.json"))
}

// LoadProjectSettings loads hooks from project-level settings
func (m *ConfigManager) LoadProjectSettings(projectDir string) error {
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := m.LoadFromFile(SourceProjectSettings, settingsPath); err != nil {
		return err
	}

	localPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	return m.LoadFromFile(SourceLocalSettings, localPath)
}

// CaptureSnapshot captures the current configuration state
func (m *ConfigManager) CaptureSnapshot(trustLevel string) *ConfigSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Deep copy settings
	settingsCopy := make(HooksSettings)
	for event, matchers := range m.settings {
		settingsCopy[event] = append([]HookMatcherConfig{}, matchers...)
	}

	snapshot := &ConfigSnapshot{
		Settings:   settingsCopy,
		TrustLevel: trustLevel,
	}
	m.snapshot = snapshot
	return snapshot
}

// GetSnapshot returns the current snapshot
func (m *ConfigManager) GetSnapshot() *ConfigSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshot
}

// SetManagedOnly configures whether only managed hooks are allowed
func (m *ConfigManager) SetManagedOnly(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.snapshot != nil {
		m.snapshot.ManagedOnly = enabled
	}
}

// SetDisableAll disables all hooks
func (m *ConfigManager) SetDisableAll(disabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.snapshot != nil {
		m.snapshot.DisableAll = disabled
	}
}

// ShouldAllowManagedHooksOnly checks if only managed hooks are allowed
func (m *ConfigManager) ShouldAllowManagedHooksOnly() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.snapshot == nil {
		return false
	}
	return m.snapshot.ManagedOnly
}

// ShouldDisableAllHooks checks if all hooks should be disabled
func (m *ConfigManager) ShouldDisableAllHooks() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.snapshot == nil {
		return false
	}
	return m.snapshot.DisableAll
}

// GetHooks returns all hooks for an event, sorted by priority
func (m *ConfigManager) GetHooks(event HookEvent) []HookMatcherConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matchers := m.settings[event]
	if len(matchers) == 0 {
		return nil
	}

	// Sort hooks within each matcher by source priority
	result := make([]HookMatcherConfig, len(matchers))
	for i, matcher := range matchers {
		sortedHooks := make([]Hook, len(matcher.Hooks))
		copy(sortedHooks, matcher.Hooks)

		sort.Slice(sortedHooks, func(a, b int) bool {
			var srcA, srcB HookSource
			switch h := sortedHooks[a].(type) {
			case CommandHook:
				srcA = h.Source
			case PromptHook:
				srcA = h.Source
			case AgentHook:
				srcA = h.Source
			case HTTPHook:
				srcA = h.Source
			case CallbackHook:
				srcA = h.Source
			}
			switch h := sortedHooks[b].(type) {
			case CommandHook:
				srcB = h.Source
			case PromptHook:
				srcB = h.Source
			case AgentHook:
				srcB = h.Source
			case HTTPHook:
				srcB = h.Source
			case CallbackHook:
				srcB = h.Source
			}

			priA, okA := m.priorities[srcA]
			priB, okB := m.priorities[srcB]
			if !okA {
				priA = 999
			}
			if !okB {
				priB = 999
			}
			return priA < priB
		})

		result[i] = HookMatcherConfig{
			Matcher: matcher.Matcher,
			Hooks:   sortedHooks,
		}
	}

	return result
}

// AddSessionHook adds a temporary session hook
func (m *ConfigManager) AddSessionHook(event HookEvent, hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find existing matcher or create new one
	var targetMatcher *HookMatcherConfig
	for i := range m.settings[event] {
		if m.settings[event][i].Matcher == "" {
			targetMatcher = &m.settings[event][i]
			break
		}
	}

	if targetMatcher == nil {
		m.settings[event] = append(m.settings[event], HookMatcherConfig{
			Hooks: []Hook{hook},
		})
	} else {
		targetMatcher.Hooks = append(targetMatcher.Hooks, hook)
	}
}

// RemoveSessionHook removes a session hook by ID
func (m *ConfigManager) RemoveSessionHook(event HookEvent, hookID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.settings[event] {
		for j := len(m.settings[event][i].Hooks) - 1; j >= 0; j-- {
			if m.settings[event][i].Hooks[j].GetID() == hookID {
				m.settings[event][i].Hooks = append(
					m.settings[event][i].Hooks[:j],
					m.settings[event][i].Hooks[j+1:]...,
				)
				return true
			}
		}
	}
	return false
}

// ClearSessionHooks removes all session hooks
func (m *ConfigManager) ClearSessionHooks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for event := range m.settings {
		var filtered []HookMatcherConfig
		for _, matcher := range m.settings[event] {
			var hooks []Hook
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
				if source != SourceSessionHook {
					hooks = append(hooks, hook)
				}
			}
			if len(hooks) > 0 {
				filtered = append(filtered, HookMatcherConfig{
					Matcher: matcher.Matcher,
					Hooks:   hooks,
				})
			}
		}
		m.settings[event] = filtered
	}
}

// AddCallbackHook adds a Go callback hook
func (m *ConfigManager) AddCallbackHook(event HookEvent, matcher string, callback HookCallbackFunc, timeout int) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateHookID()
	hook := CallbackHook{
		BaseHook: BaseHook{
			Type:    HookTypeCallback,
			ID:      id,
			Timeout: timeout,
			Source:  SourceSessionHook,
		},
		Callback: callback,
	}

	// Find matching matcher or create new one
	for i := range m.settings[event] {
		if m.settings[event][i].Matcher == matcher {
			m.settings[event][i].Hooks = append(m.settings[event][i].Hooks, hook)
			return id
		}
	}

	m.settings[event] = append(m.settings[event], HookMatcherConfig{
		Matcher: matcher,
		Hooks:   []Hook{hook},
	})

	return id
}

// generateHookID generates a unique hook ID
func generateHookID() string {
	return fmt.Sprintf("hook_%d", os.Getpid())
}

// MatchesMatcher checks if a value matches a hook matcher pattern
func MatchesMatcher(pattern, value string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Support glob-style patterns
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	// Exact match
	return pattern == value
}
