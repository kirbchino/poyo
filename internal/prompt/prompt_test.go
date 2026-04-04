package prompt

import (
	"testing"
)

func TestGetSystemPrompt(t *testing.T) {
	tests := []struct {
		mode     string
		contains string
	}{
		{"interactive", "交互模式"},
		{"plan", "规划模式"},
		{"agent", "子代理"},
		{"explore", "探索"},
		{"plan_agent", "规划"},
		{"default", "Poyo"},
		{"unknown", "Poyo"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := GetSystemPrompt(tt.mode)
			if result == "" {
				t.Errorf("GetSystemPrompt(%s) returned empty string", tt.mode)
			}
		})
	}
}

func TestGetToolDescription(t *testing.T) {
	tools := []string{
		"Read", "Write", "Edit", "Glob", "Grep",
		"Bash", "Agent", "Skill", "EnterWorktree", "ExitWorktree",
		"MediaRead", "WebFetch", "WebSearch", "TodoWrite",
	}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			result := GetToolDescription(tool)
			if result == "" {
				t.Errorf("GetToolDescription(%s) returned empty string", tool)
			}
		})
	}
}

func TestToolDescriptionsNotEmpty(t *testing.T) {
	// Check all tool descriptions are set
	if ToolDescriptions.Read == "" {
		t.Error("ToolDescriptions.Read is empty")
	}
	if ToolDescriptions.Write == "" {
		t.Error("ToolDescriptions.Write is empty")
	}
	if ToolDescriptions.Edit == "" {
		t.Error("ToolDescriptions.Edit is empty")
	}
	if ToolDescriptions.Skill == "" {
		t.Error("ToolDescriptions.Skill is empty")
	}
	if ToolDescriptions.EnterWorktree == "" {
		t.Error("ToolDescriptions.EnterWorktree is empty")
	}
	if ToolDescriptions.ExitWorktree == "" {
		t.Error("ToolDescriptions.ExitWorktree is empty")
	}
	if ToolDescriptions.MediaRead == "" {
		t.Error("ToolDescriptions.MediaRead is empty")
	}
}

func TestPoyoIdentity(t *testing.T) {
	if Name != "Poyo" {
		t.Errorf("Expected Name to be 'Poyo', got '%s'", Name)
	}
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Identity == "" {
		t.Error("Identity should not be empty")
	}
}

func TestPersonality(t *testing.T) {
	if len(Personality.Greetings) == 0 {
		t.Error("Personality.Greetings should not be empty")
	}
	if len(Personality.Success) == 0 {
		t.Error("Personality.Success should not be empty")
	}
	if len(Personality.Thinking) == 0 {
		t.Error("Personality.Thinking should not be empty")
	}
	if len(Personality.Error) == 0 {
		t.Error("Personality.Error should not be empty")
	}
	if len(Personality.Celebration) == 0 {
		t.Error("Personality.Celebration should not be empty")
	}
}

func TestMessages(t *testing.T) {
	if Messages.PermissionAsk == "" {
		t.Error("Messages.PermissionAsk should not be empty")
	}
	if Messages.ErrorGeneric == "" {
		t.Error("Messages.ErrorGeneric should not be empty")
	}
	if Messages.ProgressStart == "" {
		t.Error("Messages.ProgressStart should not be empty")
	}
}

func TestMessageGenerators(t *testing.T) {
	// Test quick message generators
	result := MsgPermissionAsk("test action")
	if result == "" {
		t.Error("MsgPermissionAsk should return non-empty string")
	}

	result = MsgErrorGeneric("test error")
	if result == "" {
		t.Error("MsgErrorGeneric should return non-empty string")
	}

	result = MsgProgressStart()
	if result == "" {
		t.Error("MsgProgressStart should return non-empty string")
	}

	result = MsgProgressDone()
	if result == "" {
		t.Error("MsgProgressDone should return non-empty string")
	}

	result = PoyoGreeting()
	if result == "" {
		t.Error("PoyoGreeting should return non-empty string")
	}

	result = PoyoSuccess()
	if result == "" {
		t.Error("PoyoSuccess should return non-empty string")
	}

	result = PoyoThinking()
	if result == "" {
		t.Error("PoyoThinking should return non-empty string")
	}

	result = PoyoCelebration()
	if result == "" {
		t.Error("PoyoCelebration should return non-empty string")
	}

	result = PoyoDance()
	if result == "" {
		t.Error("PoyoDance should return non-empty string")
	}
}

func TestFormatMessage(t *testing.T) {
	template := "Hello {{.name}}, welcome to {{.place}}!"
	vars := map[string]string{
		"name":  "Poyo",
		"place": "Dream Land",
	}
	result := FormatMessage(template, vars)
	if result != "Hello Poyo, welcome to Dream Land!" {
		t.Errorf("FormatMessage result unexpected: %s", result)
	}
}

func TestGetHookDescription(t *testing.T) {
	hooks := []string{
		"PreToolUse", "PostToolUse", "PrePrompt", "PostPrompt",
		"OnStart", "OnEnd", "OnError",
	}

	for _, hook := range hooks {
		t.Run(hook, func(t *testing.T) {
			result := GetHookDescription(hook)
			if result == "" {
				t.Errorf("GetHookDescription(%s) returned empty string", hook)
			}
		})
	}
}

func TestGetPluginTypeDescription(t *testing.T) {
	types := []string{"lua", "mcp", "script"}

	for _, pt := range types {
		t.Run(pt, func(t *testing.T) {
			result := GetPluginTypeDescription(pt)
			if result == "" {
				t.Errorf("GetPluginTypeDescription(%s) returned empty string", pt)
			}
		})
	}
}

func TestTUI(t *testing.T) {
	if TUI.Welcome == "" {
		t.Error("TUI.Welcome should not be empty")
	}
	if TUI.HelpText == "" {
		t.Error("TUI.HelpText should not be empty")
	}
	if TUI.StatusReady == "" {
		t.Error("TUI.StatusReady should not be empty")
	}
}

func TestCLI(t *testing.T) {
	if CLI.CmdRoot == "" {
		t.Error("CLI.CmdRoot should not be empty")
	}
	if CLI.CmdAbility == "" {
		t.Error("CLI.CmdAbility should not be empty")
	}
	if CLI.CmdPlugin == "" {
		t.Error("CLI.CmdPlugin should not be empty")
	}
}

func TestGetWelcomeMessage(t *testing.T) {
	result := GetWelcomeMessage()
	if result == "" {
		t.Error("GetWelcomeMessage should return non-empty string")
	}
}

func TestGetHelpText(t *testing.T) {
	result := GetHelpText()
	if result == "" {
		t.Error("GetHelpText should return non-empty string")
	}
}

func TestGetStatusMessage(t *testing.T) {
	statuses := []string{"ready", "working", "thinking", "error", "unknown"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			result := GetStatusMessage(status)
			if result == "" {
				t.Errorf("GetStatusMessage(%s) returned empty string", status)
			}
		})
	}
}
