package hooks

import (
	"context"
	"testing"
	"time"
)

func TestHookEventTypes(t *testing.T) {
	events := []HookEvent{
		EventPreToolUse,
		EventPostToolUse,
		EventSessionStart,
		EventSessionEnd,
		EventStop,
		EventPreCompact,
		EventPostCompact,
	}

	for _, event := range events {
		if string(event) == "" {
			t.Errorf("Hook event should have a non-empty string representation")
		}
	}
}

func TestHookTypes(t *testing.T) {
	types := []HookType{
		HookTypeCommand,
		HookTypePrompt,
		HookTypeAgent,
		HookTypeHTTP,
		HookTypeCallback,
	}

	for _, ht := range types {
		if string(ht) == "" {
			t.Errorf("Hook type should have a non-empty string representation")
		}
	}
}

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager() returned nil")
	}

	if cm.settings == nil {
		t.Error("settings map should be initialized")
	}

	if cm.priorities == nil {
		t.Error("priorities map should be initialized")
	}
}

func TestLoadFromSettings(t *testing.T) {
	cm := NewConfigManager()

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "echo 'test'",
						},
					},
				},
			},
		},
	}

	err := cm.LoadFromSettings(SourceUserSettings, settings)
	if err != nil {
		t.Fatalf("LoadFromSettings() error: %v", err)
	}

	hooks := cm.GetHooks(EventPreToolUse)
	if len(hooks) == 0 {
		t.Error("Expected PreToolUse hooks to be loaded")
	}

	if len(hooks[0].Hooks) == 0 {
		t.Error("Expected at least one hook in matcher")
	}
}

func TestAddSessionHook(t *testing.T) {
	cm := NewConfigManager()

	hook := CommandHook{
		BaseHook: BaseHook{
			Type:   HookTypeCommand,
			ID:     "test-hook-1",
			Source: SourceSessionHook,
		},
		Command: "echo 'session hook'",
	}

	cm.AddSessionHook(EventPreToolUse, hook)

	hooks := cm.GetHooks(EventPreToolUse)
	if len(hooks) == 0 {
		t.Error("Expected session hook to be added")
	}
}

func TestRemoveSessionHook(t *testing.T) {
	cm := NewConfigManager()

	hook := CommandHook{
		BaseHook: BaseHook{
			Type:   HookTypeCommand,
			ID:     "test-hook-2",
			Source: SourceSessionHook,
		},
		Command: "echo 'test'",
	}

	cm.AddSessionHook(EventPreToolUse, hook)

	removed := cm.RemoveSessionHook(EventPreToolUse, "test-hook-2")
	if !removed {
		t.Error("Expected hook to be removed")
	}

	hooks := cm.GetHooks(EventPreToolUse)
	for _, m := range hooks {
		for _, h := range m.Hooks {
			if h.GetID() == "test-hook-2" {
				t.Error("Hook should have been removed")
			}
		}
	}
}

func TestAddCallbackHook(t *testing.T) {
	cm := NewConfigManager()

	called := false
	callback := func(ctx context.Context, input *HookInput) (*HookOutput, error) {
		called = true
		return &HookOutput{Continue: true}, nil
	}

	id := cm.AddCallbackHook(EventPreToolUse, "Bash", callback, 30)
	if id == "" {
		t.Error("Expected non-empty hook ID")
	}
}

func TestCaptureSnapshot(t *testing.T) {
	cm := NewConfigManager()

	// Add some hooks
	hook := CommandHook{
		BaseHook: BaseHook{
			Type:   HookTypeCommand,
			ID:     "snapshot-test",
			Source: SourceUserSettings,
		},
		Command: "echo 'snapshot'",
	}
	cm.AddSessionHook(EventPreToolUse, hook)

	snapshot := cm.CaptureSnapshot("trusted")
	if snapshot == nil {
		t.Fatal("CaptureSnapshot() returned nil")
	}

	if snapshot.TrustLevel != "trusted" {
		t.Errorf("Expected trust level 'trusted', got '%s'", snapshot.TrustLevel)
	}

	if snapshot.Settings == nil {
		t.Error("Snapshot settings should not be nil")
	}
}

func TestMatchesMatcher(t *testing.T) {
	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"", "anything", true},
		{"*", "anything", true},
		{"Bash", "Bash", true},
		{"Bash", "Read", false},
		{"Bash*", "Bash", true},
		{"Bash*", "BashTool", true},
		{"Bash*", "Read", false},
	}

	for _, tt := range tests {
		result := MatchesMatcher(tt.pattern, tt.value)
		if result != tt.expected {
			t.Errorf("MatchesMatcher(%q, %q) = %v, want %v", tt.pattern, tt.value, result, tt.expected)
		}
	}
}

func TestNewExecutor(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}

	if executor.configManager == nil {
		t.Error("Executor should have config manager")
	}

	if executor.asyncRegistry == nil {
		t.Error("Executor should have async registry")
	}
}

func TestExecuteHooksEmpty(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	ctx := context.Background()
	input := &HookInput{
		Event:     EventPreToolUse,
		ToolName:  "Bash",
		SessionID: "test-session",
	}

	outputs, err := executor.ExecuteHooks(ctx, EventPreToolUse, input)
	if err != nil {
		t.Fatalf("ExecuteHooks() error: %v", err)
	}

	if len(outputs) != 0 {
		t.Error("Expected no outputs when no hooks are registered")
	}
}

func TestExecuteCallbackHook(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	// Add a callback hook
	executed := false
	callback := func(ctx context.Context, input *HookInput) (*HookOutput, error) {
		executed = true
		return &HookOutput{
			Continue: true,
			Reason:   "callback executed",
		}, nil
	}

	cm.AddCallbackHook(EventPreToolUse, "Bash", callback, 30)

	ctx := context.Background()
	input := &HookInput{
		Event:     EventPreToolUse,
		ToolName:  "Bash",
		SessionID: "test-session",
	}

	outputs, err := executor.ExecuteHooks(ctx, EventPreToolUse, input)
	if err != nil {
		t.Fatalf("ExecuteHooks() error: %v", err)
	}

	if !executed {
		t.Error("Callback should have been executed")
	}

	if len(outputs) == 0 {
		t.Fatal("Expected at least one output")
	}

	if !outputs[0].Continue {
		t.Error("Expected Continue to be true")
	}
}

func TestExecuteHooksWithOnce(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	execCount := 0
	callback := func(ctx context.Context, input *HookInput) (*HookOutput, error) {
		execCount++
		return &HookOutput{Continue: true}, nil
	}

	// Add a once hook
	hook := CallbackHook{
		BaseHook: BaseHook{
			Type:    HookTypeCallback,
			ID:      "once-hook",
			Once:    true,
			Source:  SourceSessionHook,
		},
		Callback: callback,
	}
	cm.AddSessionHook(EventPreToolUse, hook)

	ctx := context.Background()
	input := &HookInput{
		Event:     EventPreToolUse,
		ToolName:  "Bash",
		SessionID: "test-session",
	}

	// First execution
	_, err := executor.ExecuteHooks(ctx, EventPreToolUse, input)
	if err != nil {
		t.Fatalf("First ExecuteHooks() error: %v", err)
	}

	if execCount != 1 {
		t.Errorf("Expected 1 execution, got %d", execCount)
	}

	// Second execution - hook should have been removed
	_, err = executor.ExecuteHooks(ctx, EventPreToolUse, input)
	if err != nil {
		t.Fatalf("Second ExecuteHooks() error: %v", err)
	}

	if execCount != 1 {
		t.Errorf("Expected still 1 execution (once hook removed), got %d", execCount)
	}
}

func TestAsyncRegistry(t *testing.T) {
	registry := NewAsyncRegistry()

	if registry.Count() != 0 {
		t.Error("Registry should be empty initially")
	}

	info := AsyncHookInfo{
		HookID:    "test-async",
		HookEvent: EventPreToolUse,
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	processID := registry.Register(info)
	if processID == "" {
		t.Error("Expected non-empty process ID")
	}

	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}

	// Set result
	output := &HookOutput{Continue: true}
	registry.SetResult(processID, output, nil)

	// Check completed
	results := registry.CheckCompleted()
	if len(results) != 1 {
		t.Errorf("Expected 1 completed result, got %d", len(results))
	}

	// Remove
	registry.Remove(processID)
	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after removal, got %d", registry.Count())
	}
}

func TestAsyncRegistryFinalize(t *testing.T) {
	registry := NewAsyncRegistry()

	info := AsyncHookInfo{
		HookID:    "test-finalize",
		HookEvent: EventPreToolUse,
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	registry.Register(info)
	registry.Register(AsyncHookInfo{HookID: "test-2"})

	if registry.Count() != 2 {
		t.Errorf("Expected count 2, got %d", registry.Count())
	}

	registry.Finalize()

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after finalize, got %d", registry.Count())
	}
}

func TestGetTimeoutDuration(t *testing.T) {
	tests := []struct {
		hook     Hook
		expected time.Duration
	}{
		{CommandHook{BaseHook: BaseHook{Timeout: 30}}, 30 * time.Second},
		{CommandHook{BaseHook: BaseHook{Timeout: 0}}, DefaultHookTimeout},
		{CommandHook{BaseHook: BaseHook{Timeout: -5}}, DefaultHookTimeout},
	}

	for _, tt := range tests {
		result := GetTimeoutDuration(tt.hook)
		if result != tt.expected {
			t.Errorf("GetTimeoutDuration() = %v, want %v", result, tt.expected)
		}
	}
}

func TestHookOutputJSON(t *testing.T) {
	output := &HookOutput{
		Continue:       false,
		StopReason:     "Blocked by hook",
		Decision:       DecisionBlock,
		Reason:         "Security policy",
		SystemMessage:  "Hook executed",
	}

	// Verify fields are accessible
	if output.Continue {
		t.Error("Continue should be false")
	}

	if output.Decision != DecisionBlock {
		t.Error("Decision should be block")
	}
}

func TestRunPreToolUseHooks(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	// Add a callback hook that blocks Bash
	cm.AddCallbackHook(EventPreToolUse, "Bash", func(ctx context.Context, input *HookInput) (*HookOutput, error) {
		return &HookOutput{
			Continue: false,
			Reason:   "Bash is blocked",
		}, nil
	}, 30)

	ctx := context.Background()
	toolCtx := &ToolHooksContext{
		SessionID:  "test-session",
		ProjectDir: "/tmp",
	}

	result, err := RunPreToolUseHooks(ctx, executor, toolCtx, "Bash", map[string]interface{}{"command": "ls"})
	if err != nil {
		t.Fatalf("RunPreToolUseHooks() error: %v", err)
	}

	if result.Continue {
		t.Error("Expected Continue to be false (blocked)")
	}

	if result.BlockReason != "Bash is blocked" {
		t.Errorf("Unexpected block reason: %s", result.BlockReason)
	}
}

func TestRunPostToolUseHooks(t *testing.T) {
	cm := NewConfigManager()
	executor := NewExecutor(cm)

	executed := false
	cm.AddCallbackHook(EventPostToolUse, "Read", func(ctx context.Context, input *HookInput) (*HookOutput, error) {
		executed = true
		return &HookOutput{Continue: true}, nil
	}, 30)

	ctx := context.Background()
	toolCtx := &ToolHooksContext{
		SessionID:  "test-session",
		ProjectDir: "/tmp",
	}

	result, err := RunPostToolUseHooks(ctx, executor, toolCtx, "Read", "tool-use-123", map[string]interface{}{"file": "/tmp/test"}, "output content", nil)
	if err != nil {
		t.Fatalf("RunPostToolUseHooks() error: %v", err)
	}

	if !executed {
		t.Error("PostToolUse hook should have been executed")
	}

	if !result.Continue {
		t.Error("Expected Continue to be true")
	}
}
