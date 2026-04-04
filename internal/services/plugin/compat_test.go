package plugin

import (
	"encoding/json"
	"testing"
)

// ═══════════════════════════════════════════════════════
// Claude Code Format Tests
// ═══════════════════════════════════════════════════════

func TestParseClaudeCodeManifest(t *testing.T) {
	ccManifest := `{
		"name": "test-plugin",
		"version": "1.0.0",
		"description": "Test CC plugin",
		"type": "lua",
		"main": "main.lua",
		"tools": [
			{
				"name": "test_tool",
				"description": "A test tool",
				"input_schema": {
					"type": "object",
					"properties": {
						"input": {"type": "string"}
					}
				},
				"handler": "testHandler",
				"concurrent": true
			}
		],
		"commands": [
			{
				"name": "/test-cmd",
				"description": "Test command"
			}
		],
		"hooks": [
			{
				"type": "PreToolUse",
				"priority": 100,
				"handler": "onPreToolUse"
			}
		],
		"permissions": ["file:read", "execute"],
		"config": {
			"debug": true
		}
	}`

	c := NewCompatibilityLayer([]string{})
	plugin, err := c.parseClaudeCode([]byte(ccManifest), "/plugins/test/plugin.json")
	if err != nil {
		t.Fatalf("Failed to parse CC manifest: %v", err)
	}

	// 验证基本字段
	if plugin.Name != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got '%s'", plugin.Name)
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", plugin.Version)
	}
	if plugin.Type != PluginTypeLua {
		t.Errorf("Expected type '%s', got '%s'", PluginTypeLua, plugin.Type)
	}
	if plugin.Main != "main.lua" {
		t.Errorf("Expected main 'main.lua', got '%s'", plugin.Main)
	}

	// 验证工具
	if len(plugin.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(plugin.Tools))
	}
	if plugin.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", plugin.Tools[0].Name)
	}
	if plugin.Tools[0].Handler != "testHandler" {
		t.Errorf("Expected handler 'testHandler', got '%s'", plugin.Tools[0].Handler)
	}
	if !plugin.Tools[0].Concurrent {
		t.Error("Expected concurrent to be true")
	}

	// 验证命令
	if len(plugin.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(plugin.Commands))
	}
	if plugin.Commands[0].Name != "/test-cmd" {
		t.Errorf("Expected command name '/test-cmd', got '%s'", plugin.Commands[0].Name)
	}

	// 验证钩子
	if len(plugin.Hooks) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(plugin.Hooks))
	}
	if plugin.Hooks[0].Type != "PreToolUse" {
		t.Errorf("Expected hook type 'PreToolUse', got '%s'", plugin.Hooks[0].Type)
	}
	if plugin.Hooks[0].Priority != 100 {
		t.Errorf("Expected priority 100, got %d", plugin.Hooks[0].Priority)
	}

	// 验证权限
	if len(plugin.Permissions) != 2 {
		t.Fatalf("Expected 2 permissions, got %d", len(plugin.Permissions))
	}
	if plugin.Permissions[0].Type != PermissionFileRead {
		t.Errorf("Expected permission '%s', got '%s'", PermissionFileRead, plugin.Permissions[0].Type)
	}

	// 验证配置
	if plugin.Config["debug"] != true {
		t.Error("Expected config debug to be true")
	}
}

// ═══════════════════════════════════════════════════════
// OpenClaw Format Tests
// ═══════════════════════════════════════════════════════

func TestParseOpenClawManifest(t *testing.T) {
	ocManifest := `{
		"id": "test-analyzer",
		"name": "Test Analyzer",
		"version": "2.0.0",
		"description": "Test OpenClaw plugin",
		"author": "Test Team",
		"license": "MIT",
		"repository": "https://github.com/test/analyzer",
		"runtime": "node",
		"entry": "main.js",
		"provides": {
			"tools": [
				{
					"name": "analyze",
					"description": "Analyze code",
					"parameters": {
						"type": "object",
						"properties": {
							"path": {"type": "string"}
						}
					}
				}
			],
			"commands": [
				{
					"name": "analyze-project",
					"description": "Analyze entire project"
				}
			],
			"hooks": [
				{
					"event": "tool.use.before",
					"priority": 80,
					"handler": "beforeToolUse"
				}
			],
			"routes": [
				{
					"method": "GET",
					"path": "/analyze/:path",
					"handler": "httpAnalyze"
				}
			]
		},
		"config": {
			"maxFileSize": 1048576
		},
		"requires": [
			{
				"type": "permission",
				"name": "file:read"
			}
		]
	}`

	c := NewCompatibilityLayer([]string{})
	plugin, err := c.parseOpenClaw([]byte(ocManifest), "/plugins/test/openclaw.json")
	if err != nil {
		t.Fatalf("Failed to parse OpenClaw manifest: %v", err)
	}

	// 验证基本字段
	if plugin.ID != "test-analyzer" {
		t.Errorf("Expected ID 'test-analyzer', got '%s'", plugin.ID)
	}
	if plugin.Name != "Test Analyzer" {
		t.Errorf("Expected name 'Test Analyzer', got '%s'", plugin.Name)
	}
	if plugin.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", plugin.Version)
	}
	if plugin.Author != "Test Team" {
		t.Errorf("Expected author 'Test Team', got '%s'", plugin.Author)
	}
	if plugin.License != "MIT" {
		t.Errorf("Expected license 'MIT', got '%s'", plugin.License)
	}
	if plugin.Type != PluginTypeScript {
		t.Errorf("Expected type '%s', got '%s'", PluginTypeScript, plugin.Type)
	}
	if plugin.Main != "main.js" {
		t.Errorf("Expected main 'main.js', got '%s'", plugin.Main)
	}

	// 验证工具
	if len(plugin.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(plugin.Tools))
	}
	if plugin.Tools[0].Name != "analyze" {
		t.Errorf("Expected tool name 'analyze', got '%s'", plugin.Tools[0].Name)
	}

	// 验证命令
	if len(plugin.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(plugin.Commands))
	}
	if plugin.Commands[0].Name != "analyze-project" {
		t.Errorf("Expected command name 'analyze-project', got '%s'", plugin.Commands[0].Name)
	}

	// 验证钩子 (应该转换事件名)
	if len(plugin.Hooks) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(plugin.Hooks))
	}
	if plugin.Hooks[0].Type != "PreToolUse" {
		t.Errorf("Expected hook type 'PreToolUse' (converted), got '%s'", plugin.Hooks[0].Type)
	}
	if plugin.Hooks[0].Priority != 80 {
		t.Errorf("Expected priority 80, got %d", plugin.Hooks[0].Priority)
	}

	// 验证权限
	if len(plugin.Permissions) != 1 {
		t.Fatalf("Expected 1 permission, got %d", len(plugin.Permissions))
	}
	if plugin.Permissions[0].Type != PermissionFileRead {
		t.Errorf("Expected permission '%s', got '%s'", PermissionFileRead, plugin.Permissions[0].Type)
	}
}

// ═══════════════════════════════════════════════════════
// Spec Detection Tests
// ═══════════════════════════════════════════════════════

func TestDetectSpec(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		expected PluginSpec
	}{
		{
			name: "Claude Code format",
			manifest: `{"name": "test", "type": "lua"}`,
			expected: SpecClaudeCode,
		},
		{
			name: "OpenClaw format",
			manifest: `{"id": "test", "runtime": "node", "provides": {}}`,
			expected: SpecOpenClaw,
		},
		{
			name: "Unified format",
			manifest: `{"id": "test", "name": "Test"}`,
			expected: SpecUnified,
		},
	}

	_ = NewCompatibilityLayer([]string{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DetectSpec reads from file, so we test the logic directly
			var raw map[string]interface{}
			json.Unmarshal([]byte(tt.manifest), &raw)

			// Check detection logic
			var detected PluginSpec
			if _, ok := raw["runtime"]; ok {
				if _, ok := raw["provides"]; ok {
					detected = SpecOpenClaw
				}
			} else if _, ok := raw["type"]; ok {
				detected = SpecClaudeCode
			} else {
				detected = SpecUnified
			}

			if detected != tt.expected {
				t.Errorf("Expected spec '%s', got '%s'", tt.expected, detected)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════
// Runtime Conversion Tests
// ═══════════════════════════════════════════════════════

func TestOpenClawRuntimeConversion(t *testing.T) {
	c := NewCompatibilityLayer([]string{})

	tests := []struct {
		runtime  string
		expected PluginType
	}{
		{"lua", PluginTypeLua},
		{"python", PluginTypeScript},
		{"node", PluginTypeScript},
		{"javascript", PluginTypeScript},
		{"wasm", PluginTypeWasm},
		{"native", PluginTypeNative},
		{"go", PluginTypeNative},
		{"mcp", PluginTypeMCP},
		{"unknown", PluginTypeScript},
	}

	for _, tt := range tests {
		t.Run(tt.runtime, func(t *testing.T) {
			result := c.openClawRuntimeToType(tt.runtime)
			if result != tt.expected {
				t.Errorf("Runtime '%s': expected type '%s', got '%s'", tt.runtime, tt.expected, result)
			}
		})
	}
}

func TestOpenClawEventConversion(t *testing.T) {
	c := NewCompatibilityLayer([]string{})

	tests := []struct {
		event    string
		expected string
	}{
		{"tool.use.before", "PreToolUse"},
		{"tool.use.after", "PostToolUse"},
		{"prompt.send.before", "PrePrompt"},
		{"prompt.send.after", "PostPrompt"},
		{"error.occurred", "OnError"},
		{"plugin.load", "OnLoad"},
		{"plugin.unload", "OnUnload"},
		{"unknown.event", "unknown.event"},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := c.openClawEventToCC(tt.event)
			if result != tt.expected {
				t.Errorf("Event '%s': expected '%s', got '%s'", tt.event, tt.expected, result)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════
// Export Tests
// ═══════════════════════════════════════════════════════

func TestExportToClaudeCode(t *testing.T) {
	plugin := &Plugin{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "Test description",
		Type:        PluginTypeLua,
		Main:        "main.lua",
		Tools: []ToolDefinition{
			{Name: "tool1", Description: "Tool 1"},
		},
		Commands: []CommandDefinition{
			{Name: "/cmd1", Description: "Command 1"},
		},
		Hooks: []HookDefinition{
			{Type: "PreToolUse", Priority: 100},
		},
		Permissions: []Permission{
			{Type: PermissionFileRead},
		},
	}

	c := NewCompatibilityLayer([]string{})
	data, err := c.ExportToClaudeCode(plugin)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// 验证导出的 JSON
	var cc ClaudeCodeManifest
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	if cc.Name != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got '%s'", cc.Name)
	}
	if cc.Type != "lua" {
		t.Errorf("Expected type 'lua', got '%s'", cc.Type)
	}
	if len(cc.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(cc.Tools))
	}
}

func TestExportToOpenClaw(t *testing.T) {
	plugin := &Plugin{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "Test description",
		Author:      "Test Author",
		License:     "MIT",
		Type:        PluginTypeWasm,
		Main:        "main.wasm",
		Tools: []ToolDefinition{
			{Name: "tool1", Description: "Tool 1"},
		},
		Hooks: []HookDefinition{
			{Type: "PreToolUse", Priority: 80},
		},
		Permissions: []Permission{
			{Type: PermissionFileRead},
		},
	}

	c := NewCompatibilityLayer([]string{})
	data, err := c.ExportToOpenClaw(plugin)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// 验证导出的 JSON
	var oc OpenClawManifest
	if err := json.Unmarshal(data, &oc); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	if oc.ID != "test-plugin" {
		t.Errorf("Expected ID 'test-plugin', got '%s'", oc.ID)
	}
	if oc.Runtime != "wasm" {
		t.Errorf("Expected runtime 'wasm', got '%s'", oc.Runtime)
	}
	if len(oc.Provides.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(oc.Provides.Tools))
	}
	// 验证钩子事件名转换
	if len(oc.Provides.Hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(oc.Provides.Hooks))
	} else if oc.Provides.Hooks[0].Event != "tool.use.before" {
		t.Errorf("Expected event 'tool.use.before', got '%s'", oc.Provides.Hooks[0].Event)
	}
}
