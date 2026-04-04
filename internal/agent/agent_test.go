package agent

import (
	"context"
	"strings"
	"testing"
)

func TestAgentType(t *testing.T) {
	types := []AgentType{
		AgentTypeGeneralPurpose,
		AgentTypeExplore,
		AgentTypePlan,
		AgentTypeStatuslineSetup,
	}

	for _, at := range types {
		if string(at) == "" {
			t.Errorf("Agent type should have a non-empty string representation")
		}
	}
}

func TestAgentState(t *testing.T) {
	states := []AgentState{
		StatePending,
		StateRunning,
		StateCompleted,
		StateFailed,
		StateStopped,
	}

	for _, s := range states {
		if string(s) == "" {
			t.Errorf("Agent state should have a non-empty string representation")
		}
	}
}

func TestDefaultToolAccess(t *testing.T) {
	tests := []struct {
		agentType     AgentType
		expectReadOnly bool
		expectAllowed  []string
	}{
		{AgentTypeExplore, true, []string{"Glob", "Grep", "Read"}},
		{AgentTypePlan, true, []string{"Glob", "Grep", "Read"}},
		{AgentTypeStatuslineSetup, false, []string{"Read", "Edit", "Write"}},
		{AgentTypeGeneralPurpose, false, []string{"*"}},
	}

	for _, tt := range tests {
		access := DefaultToolAccess(tt.agentType)
		if access.ReadOnly != tt.expectReadOnly {
			t.Errorf("DefaultToolAccess(%q).ReadOnly = %v, want %v", tt.agentType, access.ReadOnly, tt.expectReadOnly)
		}

		for _, tool := range tt.expectAllowed {
			if !access.CanUseTool(tool) {
				t.Errorf("DefaultToolAccess(%q).CanUseTool(%q) = false, want true", tt.agentType, tool)
			}
		}
	}
}

func TestToolAccessCanUseTool(t *testing.T) {
	tests := []struct {
		access    *AgentToolAccess
		toolName  string
		expectCan bool
	}{
		{
			access:    &AgentToolAccess{AllowedTools: []string{"*"}},
			toolName:  "Bash",
			expectCan: true,
		},
		{
			access:    &AgentToolAccess{AllowedTools: []string{"Read", "Write"}},
			toolName:  "Read",
			expectCan: true,
		},
		{
			access:    &AgentToolAccess{AllowedTools: []string{"Read", "Write"}},
			toolName:  "Bash",
			expectCan: false,
		},
		{
			access:    &AgentToolAccess{AllowedTools: []string{"*"}, DeniedTools: []string{"Bash"}},
			toolName:  "Bash",
			expectCan: false,
		},
		{
			access:    &AgentToolAccess{AllowedTools: []string{"*"}, DeniedTools: []string{"*"}},
			toolName:  "Read",
			expectCan: false,
		},
	}

	for _, tt := range tests {
		result := tt.access.CanUseTool(tt.toolName)
		if result != tt.expectCan {
			t.Errorf("CanUseTool(%q) = %v, want %v", tt.toolName, result, tt.expectCan)
		}
	}
}

func TestAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()

	instance := &AgentInstance{
		ID:    "test-agent",
		Type:  AgentTypeGeneralPurpose,
		State: StatePending,
	}

	registry.Register(instance)

	got, ok := registry.Get("test-agent")
	if !ok {
		t.Fatal("Agent should be found in registry")
	}
	if got.ID != instance.ID {
		t.Errorf("Get() returned wrong agent")
	}

	registry.Remove("test-agent")
	_, ok = registry.Get("test-agent")
	if ok {
		t.Error("Agent should be removed from registry")
	}
}

func TestAgentRegistryList(t *testing.T) {
	registry := NewAgentRegistry()

	// Add multiple agents
	for i := 0; i < 3; i++ {
		registry.Register(&AgentInstance{
			ID:    string(rune('a' + i)),
			State: StatePending,
		})
	}

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d agents, want 3", len(list))
	}
}

func TestGenerateAgentID(t *testing.T) {
	id1 := GenerateAgentID()
	id2 := GenerateAgentID()

	if id1 == id2 {
		t.Error("GenerateAgentID should return unique IDs")
	}

	if !strings.HasPrefix(id1, "agent_") {
		t.Errorf("GenerateAgentID() = %q, should start with 'agent_'", id1)
	}
}

// MockToolCaller is a mock implementation of ToolCaller
type MockToolCaller struct {
	tools []string
}

func (m *MockToolCaller) CallTool(ctx context.Context, toolName string, input interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "mock"}, nil
}

func (m *MockToolCaller) ListTools() []string {
	return m.tools
}

func TestNewExecutor(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read", "Write"}}
	executor := NewExecutor(caller)

	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}

	if executor.registry == nil {
		t.Error("Executor should have a registry")
	}
}

func TestExecutorExecute(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read", "Write"}}
	executor := NewExecutor(caller)

	config := &AgentConfig{
		Type:        AgentTypeGeneralPurpose,
		Description: "Test agent",
		Prompt:      "Test prompt",
	}

	result, err := executor.Execute(context.Background(), config)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if result.AgentID == "" {
		t.Error("Result should have an AgentID")
	}

	if result.State != StateCompleted {
		t.Errorf("Result.State = %v, want %v", result.State, StateCompleted)
	}
}

func TestExecutorExecuteWithTimeout(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read"}}
	executor := NewExecutor(caller)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := &AgentConfig{
		Type:        AgentTypeExplore,
		Description: "Test agent",
		Prompt:      "Test prompt",
	}

	result, err := executor.Execute(ctx, config)
	if err == nil {
		t.Error("Execute should return error with cancelled context")
	}

	if result.State != StateFailed && result.State != StateStopped {
		t.Errorf("Result.State = %v, want Failed or Stopped", result.State)
	}
}

func TestExecutorStopAgent(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read"}}
	executor := NewExecutor(caller)

	// Stop non-existent agent
	err := executor.StopAgent("nonexistent")
	if err == nil {
		t.Error("StopAgent should return error for non-existent agent")
	}
}

func TestAgentTool(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read", "Write"}}
	tool := NewAgentTool(caller)

	if tool.Name() != "Agent" {
		t.Errorf("Name() = %q, want 'Agent'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema should have properties")
	}

	if _, ok := props["prompt"]; !ok {
		t.Error("Schema should have prompt property")
	}
}

func TestAgentToolExecute(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read"}}
	tool := NewAgentTool(caller)

	input := []byte(`{
		"type": "general-purpose",
		"description": "Test agent",
		"prompt": "Test the agent"
	}`)

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	agentResult, ok := result.(*AgentResult)
	if !ok {
		t.Fatal("Execute should return *AgentResult")
	}

	if agentResult.AgentID == "" {
		t.Error("Result should have AgentID")
	}
}

func TestAgentToolExecuteInvalidInput(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read"}}
	tool := NewAgentTool(caller)

	// Invalid JSON
	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute should return error for invalid JSON")
	}

	// Missing prompt
	_, err = tool.Execute(context.Background(), []byte(`{"type": "general-purpose"}`))
	if err == nil {
		t.Error("Execute should return error for missing prompt")
	}
}

func TestBackgroundAgentManager(t *testing.T) {
	manager := NewBackgroundAgentManager()

	if manager == nil {
		t.Fatal("NewBackgroundAgentManager() returned nil")
	}

	if manager.agents == nil {
		t.Error("Manager should have agents map initialized")
	}
}

func TestBackgroundAgentManagerSetNotifier(t *testing.T) {
	manager := NewBackgroundAgentManager()

	notified := make(chan string, 1)
	manager.SetNotifier(func(agentID string, result *AgentResult) {
		notified <- agentID
	})

	if manager.notifier == nil {
		t.Error("SetNotifier should set the notifier")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	caller := &MockToolCaller{tools: []string{"Read"}}
	executor := NewExecutor(caller)

	tests := []struct {
		agentType AgentType
		contains  string
	}{
		{AgentTypeExplore, "exploring codebases"},
		{AgentTypePlan, "planning agent"},
		{AgentTypeGeneralPurpose, "general-purpose agent"},
	}

	for _, tt := range tests {
		config := &AgentConfig{Type: tt.agentType}
		access := DefaultToolAccess(tt.agentType)
		prompt := executor.buildSystemPrompt(config, access)

		if !strings.Contains(prompt, tt.contains) {
			t.Errorf("buildSystemPrompt(%q) should contain %q", tt.agentType, tt.contains)
		}
	}
}
