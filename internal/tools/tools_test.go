package tools

import (
	"context"
	"testing"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test registering a tool
	tool := NewBashTool()
	registry.Register(tool)

	// Test getting tool by name
	retrieved := registry.Get("Bash")
	if retrieved == nil {
		t.Error("Expected to retrieve Bash tool")
	}
	if retrieved.Name() != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", retrieved.Name())
	}

	// Test getting tool by alias
	retrievedByAlias := registry.Get("bash")
	if retrievedByAlias == nil {
		t.Error("Expected to retrieve Bash tool by alias")
	}

	// Test list
	tools := registry.List()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()
	tool := NewBashTool()
	registry.Register(tool)

	registry.Unregister("Bash")

	retrieved := registry.Get("Bash")
	if retrieved != nil {
		t.Error("Expected tool to be unregistered")
	}
}

func TestRegistryClear(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewBashTool())
	registry.Register(NewFileReadTool())

	registry.Clear()

	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools after clear, got %d", len(tools))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Test default registry functions
	tool := NewBashTool()
	RegisterTool(tool)

	retrieved := GetTool("Bash")
	if retrieved == nil {
		t.Error("Expected to retrieve tool from default registry")
	}

	tools := GetAllTools()
	if len(tools) == 0 {
		t.Error("Expected at least one tool in default registry")
	}
}

func TestBaseTool(t *testing.T) {
	tool := NewBashTool()

	// Test Name
	if tool.Name() != "Bash" {
		t.Errorf("Expected name 'Bash', got '%s'", tool.Name())
	}

	// Test Aliases
	aliases := tool.Aliases()
	if len(aliases) == 0 {
		t.Error("Expected at least one alias")
	}

	// Test Description
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	// Test IsEnabled
	if !tool.IsEnabled() {
		t.Error("Tool should be enabled by default")
	}

	// Test IsConcurrencySafe
	if !tool.IsConcurrencySafe(nil) {
		t.Error("Bash should be concurrency safe")
	}

	// Test InputSchema
	schema := tool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", schema.Type)
	}
}

func TestToolMatchesName(t *testing.T) {
	tool := NewBashTool()

	if !ToolMatchesName(tool, "Bash") {
		t.Error("Expected to match 'Bash'")
	}
	if !ToolMatchesName(tool, "bash") {
		t.Error("Expected to match alias 'bash'")
	}
	if ToolMatchesName(tool, "UnknownTool") {
		t.Error("Should not match 'UnknownTool'")
	}
}

func TestFindToolByName(t *testing.T) {
	tools := []Tool{
		NewBashTool(),
		NewFileReadTool(),
	}

	found := FindToolByName(tools, "Bash")
	if found == nil {
		t.Error("Expected to find Bash tool")
	}

	found = FindToolByName(tools, "Read")
	if found == nil {
		t.Error("Expected to find Read tool")
	}

	found = FindToolByName(tools, "UnknownTool")
	if found != nil {
		t.Error("Should not find unknown tool")
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	if uuid1 == "" {
		t.Error("UUID should not be empty")
	}
	if uuid1 == uuid2 {
		t.Error("UUIDs should be unique")
	}
}

func TestGenerateTodoID(t *testing.T) {
	id1 := generateTodoID()
	id2 := generateTodoID()

	if id1 == "" {
		t.Error("Todo ID should not be empty")
	}
	if id1 == id2 {
		t.Error("Todo IDs should be unique")
	}
}

func TestGenerateAgentID(t *testing.T) {
	id1 := generateAgentID()
	id2 := generateAgentID()

	if id1 == "" {
		t.Error("Agent ID should not be empty")
	}
	if id1 == id2 {
		t.Error("Agent IDs should be unique")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"short", 10, "short"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFileStateCache(t *testing.T) {
	cache := NewFileStateCache()

	// Test Set and Get
	state := &FileState{
		Path:    "/test/file.go",
		Content: "package main",
	}
	cache.Set("/test/file.go", state)

	retrieved, ok := cache.Get("/test/file.go")
	if !ok {
		t.Error("Expected to find cached file state")
	}
	if retrieved.Path != "/test/file.go" {
		t.Errorf("Expected path '/test/file.go', got '%s'", retrieved.Path)
	}

	// Test non-existent
	_, ok = cache.Get("/nonexistent")
	if ok {
		t.Error("Should not find non-existent file state")
	}
}

func TestBackgroundTask(t *testing.T) {
	taskOutput := NewTaskOutputTool()

	// Register task
	taskID := "test_task_123"
	taskOutput.RegisterTask(taskID)

	// Check task exists
	task, exists := taskOutput.tasks[taskID]
	if !exists {
		t.Error("Task should be registered")
	}
	if task.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", task.Status)
	}

	// Complete task
	taskOutput.CompleteTask(taskID, "task completed")
	if task.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", task.Status)
	}

	// Fail another task
	taskID2 := "test_task_456"
	taskOutput.RegisterTask(taskID2)
	taskOutput.FailTask(taskID2, "task failed")
	task2 := taskOutput.tasks[taskID2]
	if task2.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", task2.Status)
	}
}

func TestTodoWriteTool(t *testing.T) {
	tool := NewTodoWriteTool()

	if tool.Name() != "TodoWrite" {
		t.Errorf("Expected name 'TodoWrite', got '%s'", tool.Name())
	}

	// Test adding todos
	tool.AddTodo("Test task", "Testing task")

	todos := tool.GetTodos()
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}

	// Test updating status
	if len(todos) > 0 {
		err := tool.UpdateTodoStatus(todos[0].ID, "completed")
		if err != nil {
			t.Errorf("Failed to update todo status: %v", err)
		}

		updated := tool.GetTodos()
		if updated[0].Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", updated[0].Status)
		}
	}
}

func TestSkillTool(t *testing.T) {
	tool := NewSkillTool()

	if tool.Name() != "Skill" {
		t.Errorf("Expected name 'Skill', got '%s'", tool.Name())
	}

	// Test call without executor
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"skill": "test-skill",
		"args":  "test args",
	}, nil, nil, nil)

	if err != nil {
		t.Errorf("Call should not error: %v", err)
	}
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestImageReader(t *testing.T) {
	reader := NewImageReader()

	// Test CanRead
	if !reader.CanRead("test.png") {
		t.Error("Should read PNG files")
	}
	if !reader.CanRead("test.jpg") {
		t.Error("Should read JPG files")
	}
	if !reader.CanRead("test.jpeg") {
		t.Error("Should read JPEG files")
	}
	if !reader.CanRead("test.gif") {
		t.Error("Should read GIF files")
	}
	if !reader.CanRead("test.webp") {
		t.Error("Should read WebP files")
	}
	if reader.CanRead("test.txt") {
		t.Error("Should not read TXT files")
	}
}

func TestPDFReader(t *testing.T) {
	reader := NewPDFReader()

	// Test CanRead
	if !reader.CanRead("test.pdf") {
		t.Error("Should read PDF files")
	}
	if reader.CanRead("test.txt") {
		t.Error("Should not read TXT files")
	}
}

func TestMediaReadTool(t *testing.T) {
	tool := NewMediaReadTool()

	if tool.Name() != "MediaRead" {
		t.Errorf("Expected name 'MediaRead', got '%s'", tool.Name())
	}

	// Test schema
	schema := tool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", schema.Type)
	}
}

func TestMCPTool(t *testing.T) {
	toolDef := MCPToolDefinition{
		Name:        "test_tool",
		Description: "A test MCP tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	tool := NewMCPTool("test-plugin", toolDef, nil)

	if tool.Name() != "mcp_test-plugin_test_tool" {
		t.Errorf("Expected name 'mcp_test-plugin_test_tool', got '%s'", tool.Name())
	}
	if tool.ToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.ToolName())
	}
	if tool.PluginID() != "test-plugin" {
		t.Errorf("Expected plugin ID 'test-plugin', got '%s'", tool.PluginID())
	}
}

func TestMCPToolRegistry(t *testing.T) {
	registry := NewMCPToolRegistry(nil)

	tools := []MCPToolDefinition{
		{
			Name:        "tool1",
			Description: "First tool",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	registry.RegisterPluginTools("test-plugin", tools)

	list := registry.ListTools()
	if len(list) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(list))
	}

	registry.UnregisterPluginTools("test-plugin")
	list = registry.ListTools()
	if len(list) != 0 {
		t.Errorf("Expected 0 tools after unregister, got %d", len(list))
	}
}

func TestEnterWorktreeTool(t *testing.T) {
	tool := NewEnterWorktreeTool()

	if tool.Name() != "EnterWorktree" {
		t.Errorf("Expected name 'EnterWorktree', got '%s'", tool.Name())
	}
}

func TestExitWorktreeTool(t *testing.T) {
	tool := NewExitWorktreeTool()

	if tool.Name() != "ExitWorktree" {
		t.Errorf("Expected name 'ExitWorktree', got '%s'", tool.Name())
	}
}

func TestProgressEvent(t *testing.T) {
	event := ProgressEvent{
		Type:     "start",
		Message:  "Starting task",
		Progress: 0,
	}

	if event.Type != "start" {
		t.Errorf("Expected type 'start', got '%s'", event.Type)
	}
	if event.Message != "Starting task" {
		t.Errorf("Expected message 'Starting task', got '%s'", event.Message)
	}
}
