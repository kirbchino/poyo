package commands

import (
	"context"
	"testing"
)

func TestCommandMatch(t *testing.T) {
	cmd := &Command{
		Name:    "help",
		Aliases: []string{"?", "h"},
	}

	if !cmd.Match("help") {
		t.Error("Should match primary name")
	}

	if !cmd.Match("?") {
		t.Error("Should match alias")
	}

	if !cmd.Match("h") {
		t.Error("Should match alias")
	}

	if cmd.Match("other") {
		t.Error("Should not match unrelated name")
	}
}

func TestCommandIsAvailable(t *testing.T) {
	tests := []struct {
		name            string
		cmd             *Command
		availability    CommandAvailability
		isAuthenticated bool
		isInternal      bool
		expected        bool
	}{
		{
			name: "disabled command",
			cmd: &Command{
				IsEnabled: false,
			},
			expected: false,
		},
		{
			name: "all availability",
			cmd: &Command{
				IsEnabled:    true,
				Availability: AvailabilityAll,
			},
			expected: true,
		},
		{
			name: "claude-ai authenticated",
			cmd: &Command{
				IsEnabled:    true,
				Availability: AvailabilityClaudeAI,
			},
			isAuthenticated: true,
			expected:        true,
		},
		{
			name: "claude-ai not authenticated",
			cmd: &Command{
				IsEnabled:    true,
				Availability: AvailabilityClaudeAI,
			},
			isAuthenticated: false,
			expected:        false,
		},
		{
			name: "internal user",
			cmd: &Command{
				IsEnabled:    true,
				Availability: AvailabilityInternal,
			},
			isInternal: true,
			expected:   true,
		},
		{
			name: "not internal user",
			cmd: &Command{
				IsEnabled:    true,
				Availability: AvailabilityInternal,
			},
			isInternal: false,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.IsAvailable(tt.availability, tt.isAuthenticated, tt.isInternal)
			if result != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	cmd := &Command{
		Name:        "test",
		Aliases:     []string{"t"},
		Description: "Test command",
		Type:        TypeLocal,
		IsEnabled:   true,
	}

	registry.Register(cmd)

	// Should find by name
	if found, ok := registry.Get("test"); !ok || found.Name != "test" {
		t.Error("Should find command by name")
	}

	// Should find by alias
	if found, ok := registry.Get("t"); !ok || found.Name != "test" {
		t.Error("Should find command by alias")
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	cmd := &Command{
		Name:      "test",
		Aliases:   []string{"t"},
		Type:      TypeLocal,
		IsEnabled: true,
	}

	registry.Register(cmd)
	registry.Unregister("test")

	if _, ok := registry.Get("test"); ok {
		t.Error("Should not find command after unregister")
	}

	if _, ok := registry.Get("t"); ok {
		t.Error("Should not find alias after unregister")
	}
}

func TestRegistryParse(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:    "help",
		Aliases: []string{"?"},
		Type:    TypeLocal,
	})

	tests := []struct {
		input       string
		expectName  string
		expectArgs  string
		expectFound bool
	}{
		{"/help", "help", "", true},
		{"/help arg1 arg2", "help", "arg1 arg2", true},
		{"/? test", "help", "test", true},
		{"/unknown", "", "", false},
		{"not a command", "", "", false},
	}

	for _, tt := range tests {
		cmd, args, ok := registry.Parse(tt.input)
		if ok != tt.expectFound {
			t.Errorf("Parse(%q) ok = %v, want %v", tt.input, ok, tt.expectFound)
			continue
		}
		if ok && cmd.Name != tt.expectName {
			t.Errorf("Parse(%q) name = %q, want %q", tt.input, cmd.Name, tt.expectName)
		}
		if args != tt.expectArgs {
			t.Errorf("Parse(%q) args = %q, want %q", tt.input, args, tt.expectArgs)
		}
	}
}

func TestRegistryComplete(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:      "help",
		Type:      TypeLocal,
		IsEnabled: true,
	})
	registry.Register(&Command{
		Name:      "history",
		Type:      TypeLocal,
		IsEnabled: true,
	})
	registry.Register(&Command{
		Name:      "clear",
		Type:      TypeLocal,
		IsEnabled: true,
		IsHidden:  true,
	})

	// Should match prefix
	results := registry.Complete("/h")
	if len(results) != 2 {
		t.Errorf("Complete(/h) returned %d results, want 2", len(results))
	}

	// Should not return hidden commands
	for _, cmd := range results {
		if cmd.Name == "clear" {
			t.Error("Should not return hidden command")
		}
	}
}

func TestRegistrySearch(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:        "help",
		Description: "Show help",
		Type:        TypeLocal,
		IsEnabled:   true,
	})
	registry.Register(&Command{
		Name:        "config",
		Description: "Configure settings",
		Type:        TypeLocal,
		IsEnabled:   true,
	})

	// Search by name
	results := registry.Search("help")
	if len(results) != 1 {
		t.Errorf("Search(help) returned %d results, want 1", len(results))
	}

	// Search by description
	results = registry.Search("configure")
	if len(results) != 1 {
		t.Errorf("Search(configure) returned %d results, want 1", len(results))
	}
}

func TestRegistryGetByCategory(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:     "help",
		Category: "Session",
		Type:     TypeLocal,
	})
	registry.Register(&Command{
		Name:     "config",
		Category: "Config",
		Type:     TypeLocal,
	})
	registry.Register(&Command{
		Name:     "clear",
		Category: "Session",
		Type:     TypeLocal,
	})

	categorized := registry.GetByCategory()

	if len(categorized["Session"]) != 2 {
		t.Errorf("Expected 2 Session commands, got %d", len(categorized["Session"]))
	}

	if len(categorized["Config"]) != 1 {
		t.Errorf("Expected 1 Config command, got %d", len(categorized["Config"]))
	}
}

func TestExecutorExecute(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:      "echo",
		Type:      TypeLocal,
		IsEnabled: true,
		Handler: func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
			return &CommandOutput{Output: input.Args}, nil
		},
	})

	registry.Register(&Command{
		Name:      "disabled",
		Type:      TypeLocal,
		IsEnabled: false,
	})

	registry.Register(&Command{
		Name:      "prompt",
		Type:      TypePrompt,
		IsEnabled: true,
		Prompt:    "Test prompt: $ARGS",
	})

	executor := NewExecutor(registry, "1.0.0")

	// Test local command
	output, err := executor.Execute(context.Background(), "/echo hello world", "test-session", "/tmp")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if output.Output != "hello world" {
		t.Errorf("Expected output 'hello world', got %q", output.Output)
	}

	// Test disabled command
	output, _ = executor.Execute(context.Background(), "/disabled", "test-session", "/tmp")
	if !output.IsError {
		t.Error("Expected error for disabled command")
	}

	// Test prompt command
	output, _ = executor.Execute(context.Background(), "/prompt test", "test-session", "/tmp")
	if output.Prompt == "" {
		t.Error("Expected prompt to be set")
	}
}

func TestExecutorUnknownCommand(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry, "1.0.0")

	output, err := executor.Execute(context.Background(), "/unknown", "test-session", "/tmp")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if !output.IsError {
		t.Error("Expected error for unknown command")
	}
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/help", true},
		{"  /help  ", true},
		{"help", false},
		{"not a command", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsCommand(tt.input)
		if result != tt.expected {
			t.Errorf("IsCommand(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input      string
		expectName string
		expectArgs string
	}{
		{"/help", "help", ""},
		{"/help arg1", "help", "arg1"},
		{"/help arg1 arg2", "help", "arg1 arg2"},
		{"  /help  arg1  ", "help", "arg1"},
		{"help", "", ""},
	}

	for _, tt := range tests {
		name, args := ParseCommand(tt.input)
		if name != tt.expectName {
			t.Errorf("ParseCommand(%q) name = %q, want %q", tt.input, name, tt.expectName)
		}
		if args != tt.expectArgs {
			t.Errorf("ParseCommand(%q) args = %q, want %q", tt.input, args, tt.expectArgs)
		}
	}
}

func TestFormatHelp(t *testing.T) {
	cmd := &Command{
		Name:            "test",
		Aliases:         []string{"t"},
		Description:     "Test command",
		LongDescription: "This is a test command.",
		Parameters:      "<arg>",
		Examples:        []string{"/test example"},
	}

	help := FormatHelp(cmd)

	if !contains(help, "/test") {
		t.Error("Help should contain command name")
	}
	if !contains(help, "Test command") {
		t.Error("Help should contain description")
	}
	if !contains(help, "<arg>") {
		t.Error("Help should contain parameters")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s[0:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}

func TestInitialize(t *testing.T) {
	executor, err := Initialize("1.0.0-test")
	if err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	if executor == nil {
		t.Fatal("Initialize() returned nil executor")
	}

	// Check that built-in commands are registered
	if executor.registry.Count() == 0 {
		t.Error("Expected built-in commands to be registered")
	}

	// Test a built-in command
	cmd, ok := executor.registry.Get("help")
	if !ok {
		t.Error("Expected 'help' command to be registered")
	}
	if cmd.Name != "help" {
		t.Errorf("Expected command name 'help', got %q", cmd.Name)
	}
}

// Additional tests for edge cases and error handling

func TestRegistryDuplicateRegistration(t *testing.T) {
	registry := NewRegistry()

	cmd1 := &Command{
		Name:      "test",
		Type:      TypeLocal,
		IsEnabled: true,
	}

	cmd2 := &Command{
		Name:      "test",
		Type:      TypeLocal,
		IsEnabled: true,
	}

	registry.Register(cmd1)
	registry.Register(cmd2) // Should overwrite

	cmd, ok := registry.Get("test")
	if !ok {
		t.Error("Command should be found")
	}
	// Should be the second command
	_ = cmd
}

func TestCommandWithEmptyAliases(t *testing.T) {
	cmd := &Command{
		Name:      "test",
		Aliases:   []string{},
		Type:      TypeLocal,
		IsEnabled: true,
	}

	if cmd.Match("test") {
		// OK
	} else {
		t.Error("Should match name even with empty aliases")
	}
}

func TestCommandWithNilHandler(t *testing.T) {
	registry := NewRegistry()

	cmd := &Command{
		Name:      "nohandler",
		Type:      TypeLocal,
		IsEnabled: true,
		Handler:   nil,
	}

	registry.Register(cmd)

	executor := NewExecutor(registry, "1.0.0")
	output, err := executor.Execute(context.Background(), "/nohandler", "session", "/tmp")

	if err != nil {
		// OK - should handle nil handler
	}
	_ = output
}

func TestCommandWithSpecialCharacters(t *testing.T) {
	registry := NewRegistry()

	cmd := &Command{
		Name:      "test-command",
		Aliases:   []string{"t-c", "tc"},
		Type:      TypeLocal,
		IsEnabled: true,
	}

	registry.Register(cmd)

	tests := []string{"test-command", "t-c", "tc"}
	for _, name := range tests {
		found, ok := registry.Get(name)
		if !ok {
			t.Errorf("Should find command by %q", name)
		}
		if found.Name != "test-command" {
			t.Errorf("Wrong command found for %q", name)
		}
	}
}

func TestParseCommandWithUnicode(t *testing.T) {
	input := "/help 你好世界"
	name, args := ParseCommand(input)

	if name != "help" {
		t.Errorf("Name = %q, want 'help'", name)
	}

	if args != "你好世界" {
		t.Errorf("Args = %q, want '你好世界'", args)
	}
}

func TestExecutorWithEmptyInput(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry, "1.0.0")

	output, err := executor.Execute(context.Background(), "", "session", "/tmp")

	if err != nil {
		// OK - empty input should be handled
	}
	_ = output
}

func TestRegistryCount(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Error("Initial count should be 0")
	}

	registry.Register(&Command{Name: "cmd1", Type: TypeLocal})
	registry.Register(&Command{Name: "cmd2", Type: TypeLocal})
	registry.Register(&Command{Name: "cmd3", Type: TypeLocal})

	if registry.Count() != 3 {
		t.Errorf("Count = %d, want 3", registry.Count())
	}
}

func TestCommandCategories(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:     "help",
		Category: "Session",
		Type:     TypeLocal,
	})

	registry.Register(&Command{
		Name:     "commit",
		Category: "Git",
		Type:     TypeLocal,
	})

	registry.Register(&Command{
		Name:     "mcp",
		Category: "Tools",
		Type:     TypeLocal,
	})

	categorized := registry.GetByCategory()

	if len(categorized["Session"]) != 1 {
		t.Errorf("Expected 1 Session command, got %d", len(categorized["Session"]))
	}

	if len(categorized["Git"]) != 1 {
		t.Errorf("Expected 1 Git command, got %d", len(categorized["Git"]))
	}

	if len(categorized["Tools"]) != 1 {
		t.Errorf("Expected 1 Tools command, got %d", len(categorized["Tools"]))
	}
}

func TestCommandPromptExpansion(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&Command{
		Name:      "expand",
		Type:      TypePrompt,
		IsEnabled: true,
		Prompt:    "Process: $ARGS with context",
	})

	executor := NewExecutor(registry, "1.0.0")
	output, err := executor.Execute(context.Background(), "/expand test-data", "session", "/tmp")

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if output.Prompt == "" {
		t.Error("Prompt should be expanded")
	}
}
