package plugin

import (
	"testing"
)

func TestPluginStruct(t *testing.T) {
	plugin := &Plugin{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Type:        PluginTypeLua,
		Path:        "/plugins/test-plugin",
		Main:        "main.lua",
		Enabled:     true,
	}

	if plugin.ID != "test-plugin" {
		t.Errorf("Expected ID 'test-plugin', got '%s'", plugin.ID)
	}
	if plugin.Name != "Test Plugin" {
		t.Errorf("Expected Name 'Test Plugin', got '%s'", plugin.Name)
	}
	if plugin.Type != PluginTypeLua {
		t.Errorf("Expected Type '%s', got '%s'", PluginTypeLua, plugin.Type)
	}
}

func TestPluginTypes(t *testing.T) {
	types := []struct {
		pluginType PluginType
		expected   string
	}{
		{PluginTypeLua, "lua"},
		{PluginTypeMCP, "mcp"},
		{PluginTypeScript, "script"},
	}

	for _, tt := range types {
		if string(tt.pluginType) != tt.expected {
			t.Errorf("Expected plugin type '%s', got '%s'", tt.expected, tt.pluginType)
		}
	}
}

func TestHookTypes(t *testing.T) {
	hooks := []struct {
		hookType HookType
		expected string
	}{
		{HookPreToolUse, "PreToolUse"},
		{HookPostToolUse, "PostToolUse"},
		{HookPrePrompt, "PrePrompt"},
		{HookPostPrompt, "PostPrompt"},
		{HookOnStart, "OnStart"},
		{HookOnEnd, "OnEnd"},
		{HookOnError, "OnError"},
	}

	for _, tt := range hooks {
		if string(tt.hookType) != tt.expected {
			t.Errorf("Expected hook type '%s', got '%s'", tt.expected, tt.hookType)
		}
	}
}

func TestMCPRequest(t *testing.T) {
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/list",
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got '%s'", req.JSONRPC)
	}
	if req.Method != "tools/list" {
		t.Errorf("Expected Method 'tools/list', got '%s'", req.Method)
	}
}

func TestMCPResponse(t *testing.T) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      "1",
		Result:  map[string]interface{}{"status": "ok"},
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got '%s'", resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Error("Error should be nil for success response")
	}
}

func TestMCPError(t *testing.T) {
	err := &MCPError{
		Code:    -32600,
		Message: "Invalid Request",
	}

	if err.Code != -32600 {
		t.Errorf("Expected Code -32600, got %d", err.Code)
	}
	if err.Message != "Invalid Request" {
		t.Errorf("Expected Message 'Invalid Request', got '%s'", err.Message)
	}
}

func TestMCPNotification(t *testing.T) {
	notif := &MCPNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	if notif.Method != "notifications/initialized" {
		t.Errorf("Expected Method 'notifications/initialized', got '%s'", notif.Method)
	}
}

func TestMCPCapabilities(t *testing.T) {
	caps := &MCPCapabilities{
		Tools: &MCPToolsCapabilities{
			Supported:   true,
			ListChanged: true,
		},
		Resources: &MCPResourcesCapabilities{
			Supported:   true,
			Subscribe:   true,
			ListChanged: false,
		},
		Prompts: &MCPPromptsCapabilities{
			Supported: true,
		},
	}

	if !caps.Tools.Supported {
		t.Error("Tools should be supported")
	}
	if !caps.Resources.Supported {
		t.Error("Resources should be supported")
	}
	if !caps.Prompts.Supported {
		t.Error("Prompts should be supported")
	}
}

func TestMCPResource(t *testing.T) {
	resource := &MCPResource{
		URI:         "file:///test/file.go",
		Name:        "test.go",
		Description: "A test file",
		MimeType:    "text/plain",
	}

	if resource.URI != "file:///test/file.go" {
		t.Errorf("Expected URI 'file:///test/file.go', got '%s'", resource.URI)
	}
	if resource.Name != "test.go" {
		t.Errorf("Expected Name 'test.go', got '%s'", resource.Name)
	}
}

func TestMCPPrompt(t *testing.T) {
	prompt := &MCPPrompt{
		Name:        "code-review",
		Description: "Review code for issues",
		Arguments: []MCPPromptArgument{
			{
				Name:        "code",
				Description: "Code to review",
				Required:    true,
			},
		},
	}

	if prompt.Name != "code-review" {
		t.Errorf("Expected Name 'code-review', got '%s'", prompt.Name)
	}
	if len(prompt.Arguments) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(prompt.Arguments))
	}
}

func TestMCPSamplingParams(t *testing.T) {
	params := &MCPSamplingParams{
		Messages: []MCPSamplingMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1024,
	}

	if len(params.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(params.Messages))
	}
	if params.MaxTokens != 1024 {
		t.Errorf("Expected MaxTokens 1024, got %d", params.MaxTokens)
	}
}

func TestToolDefinition(t *testing.T) {
	def := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}

	if def.Name != "test_tool" {
		t.Errorf("Expected Name 'test_tool', got '%s'", def.Name)
	}
}

func TestHookExecution(t *testing.T) {
	// Test hook input
	input := &HookInput{
		Tool:  "Bash",
		Input: map[string]interface{}{"command": "ls"},
	}

	if input.Tool != "Bash" {
		t.Errorf("Expected Tool 'Bash', got '%s'", input.Tool)
	}
}

func TestHookResult(t *testing.T) {
	result := &HookResult{
		Blocked:  false,
		Modified: true,
		Message:  "Modified command",
	}

	if result.Blocked {
		t.Error("Should not be blocked")
	}
	if !result.Modified {
		t.Error("Should be modified")
	}
}
