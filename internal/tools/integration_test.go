package tools

import (
	"context"
	"testing"
	"time"
)

// Integration tests for the tools system

func TestInitializeBuiltinTools(t *testing.T) {
	// Clear default registry first
	DefaultRegistry.Clear()

	// Initialize builtin tools
	InitializeBuiltinTools()

	// Verify all tools are registered
	expectedTools := []string{
		"Bash", "Read", "Write", "Edit", "Glob", "Grep",
		"Agent", "TodoWrite", "TaskOutput", "TaskStop",
		"WebFetch", "WebSearch", "NotebookEdit",
		"AskUserQuestion", "EnterPlanMode", "ExitPlanMode",
		"CronCreate", "CronDelete", "CronList",
		"EnterWorktree", "ExitWorktree", "Skill", "MediaRead",
	}

	for _, toolName := range expectedTools {
		tool := GetTool(toolName)
		if tool == nil {
			t.Errorf("Expected tool '%s' to be registered", toolName)
		}
	}

	// Count total tools
	tools := GetAllTools()
	if len(tools) < len(expectedTools) {
		t.Errorf("Expected at least %d tools, got %d", len(expectedTools), len(tools))
	}
}

func TestToolExecution_TodoWrite(t *testing.T) {
	tool := NewTodoWriteTool()
	ctx := context.Background()

	// Test creating todos
	result, err := tool.Call(ctx, map[string]interface{}{
		"todos": []interface{}{
			map[string]interface{}{
				"content":    "Task 1",
				"status":     "pending",
				"activeForm": "Doing Task 1",
			},
			map[string]interface{}{
				"content":    "Task 2",
				"status":     "pending",
				"activeForm": "Doing Task 2",
			},
		},
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("TodoWrite failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	todos, ok := data["todos"].([]TodoItem)
	if !ok {
		t.Fatal("Todos should be a slice")
	}

	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

func TestToolExecution_TaskOutput(t *testing.T) {
	tool := NewTaskOutputTool()
	ctx := context.Background()

	// Register a task
	taskID := "test_task_" + time.Now().Format("20060102150405")
	tool.RegisterTask(taskID)

	// Get task output
	result, err := tool.Call(ctx, map[string]interface{}{
		"task_id": taskID,
		"block":   false,
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("TaskOutput failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	status, _ := data["status"].(string)
	if status != "running" {
		t.Errorf("Expected status 'running', got '%s'", status)
	}

	// Complete the task
	tool.CompleteTask(taskID, "Task completed successfully")

	// Check status again
	result2, _ := tool.Call(ctx, map[string]interface{}{
		"task_id": taskID,
	}, nil, nil, nil)

	data2, _ := result2.Data.(map[string]interface{})
	status2, _ := data2["status"].(string)
	if status2 != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status2)
	}
}

func TestToolExecution_TaskStop(t *testing.T) {
	taskOutput := NewTaskOutputTool()
	tool := NewTaskStopTool(taskOutput)
	ctx := context.Background()

	// Register a task
	taskID := "stop_test_" + time.Now().Format("20060102150405")
	taskOutput.RegisterTask(taskID)

	// Stop the task
	result, err := tool.Call(ctx, map[string]interface{}{
		"task_id": taskID,
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("TaskStop failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	status, _ := data["status"].(string)
	if status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", status)
	}
}

func TestToolExecution_Skill(t *testing.T) {
	tool := NewSkillTool()
	ctx := context.Background()

	// Test skill call without executor (should return placeholder)
	result, err := tool.Call(ctx, map[string]interface{}{
		"skill": "test-skill",
		"args":  "test arguments",
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("Skill call failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	skill, _ := data["skill"].(string)
	if skill != "test-skill" {
		t.Errorf("Expected skill 'test-skill', got '%s'", skill)
	}
}

func TestToolExecution_EnterPlanMode(t *testing.T) {
	tool := NewEnterPlanModeTool()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"task_description": "Test planning task",
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("EnterPlanMode failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	mode, _ := data["mode"].(string)
	if mode != "plan" {
		t.Errorf("Expected mode 'plan', got '%s'", mode)
	}
}

func TestToolExecution_ExitPlanMode(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{}, nil, nil, nil)

	if err != nil {
		t.Fatalf("ExitPlanMode failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	mode, _ := data["mode"].(string)
	if mode != "default" {
		t.Errorf("Expected mode 'default', got '%s'", mode)
	}
}

func TestToolExecution_CronCreate(t *testing.T) {
	tool := NewCronCreateTool()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"cron":   "0 9 * * *",
		"prompt": "Daily reminder",
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("CronCreate failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	if data["cron"] != "0 9 * * *" {
		t.Errorf("Expected cron '0 9 * * *', got '%s'", data["cron"])
	}
}

func TestToolExecution_CronList(t *testing.T) {
	tool := NewCronListTool()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{}, nil, nil, nil)

	if err != nil {
		t.Fatalf("CronList failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	// Should have a jobs array
	_, exists := data["jobs"]
	if !exists {
		t.Error("Expected 'jobs' key in result")
	}
}

func TestToolExecution_AskUserQuestion(t *testing.T) {
	tool := NewAskUserQuestionTool()
	ctx := context.Background()

	// Test without askFunc set
	result, err := tool.Call(ctx, map[string]interface{}{
		"question": "What is your choice?",
		"options":  []interface{}{"A", "B", "C"},
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("AskUserQuestion failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	// Without askFunc, should return skipped
	answer, _ := data["answer"].(string)
	if answer != "user_input_skipped" {
		t.Logf("Answer: %s", answer)
	}
}

func TestToolExecution_MediaRead(t *testing.T) {
	tool := NewMediaReadTool()
	ctx := context.Background()

	// Test with non-existent file
	_, err := tool.Call(ctx, map[string]interface{}{
		"path": "/nonexistent/file.png",
		"type": "image",
	}, nil, nil, nil)

	// Should return error for non-existent file
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestToolExecution_Agent(t *testing.T) {
	tool := NewAgentTool()
	ctx := context.Background()

	// Test agent without executor
	result, err := tool.Call(ctx, map[string]interface{}{
		"prompt":          "Test task",
		"description":     "Test agent",
		"subagent_type":   "general-purpose",
		"max_turns":       5,
		"run_in_background": false,
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("Agent failed: %v", err)
	}

	data, ok := result.Data.(*AgentOutput)
	if !ok {
		t.Fatal("Result data should be AgentOutput")
	}

	if data.AgentID == "" {
		t.Error("AgentID should not be empty")
	}
}

func TestToolExecution_AgentBackground(t *testing.T) {
	tool := NewAgentTool()
	taskOutput := NewTaskOutputTool()
	tool.SetTaskOutput(taskOutput)
	ctx := context.Background()

	// Test background execution
	result, err := tool.Call(ctx, map[string]interface{}{
		"prompt":            "Background task",
		"description":       "Background agent test",
		"run_in_background": true,
	}, nil, nil, nil)

	if err != nil {
		t.Fatalf("Agent background failed: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data should be a map")
	}

	status, _ := data["status"].(string)
	if status != "running" {
		t.Errorf("Expected status 'running', got '%s'", status)
	}

	taskID, _ := data["task_id"].(string)
	if taskID == "" {
		t.Error("Task ID should not be empty")
	}
}

func TestMCPToolIntegration(t *testing.T) {
	// Create a mock executor
	executor := &mockMCPExecutor{}
	registry := NewMCPToolRegistry(executor)

	// Define tools
	tools := []MCPToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	// Register tools
	registry.RegisterPluginTools("fs-plugin", tools)

	// Verify registration
	list := registry.ListTools()
	if len(list) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(list))
	}

	// Verify tool retrieval
	tool := registry.GetTool("mcp_fs-plugin_read_file")
	if tool == nil {
		t.Error("Expected to get read_file tool")
	}

	// Unregister
	registry.UnregisterPluginTools("fs-plugin")
	list = registry.ListTools()
	if len(list) != 0 {
		t.Errorf("Expected 0 tools after unregister, got %d", len(list))
	}
}

// Mock MCP executor for testing
type mockMCPExecutor struct{}

func (m *mockMCPExecutor) CallTool(ctx context.Context, pluginID string, toolName string, args interface{}) (interface{}, error) {
	return map[string]interface{}{
		"result":  "mock result",
		"tool":    toolName,
		"plugin":  pluginID,
		"args":    args,
	}, nil
}
