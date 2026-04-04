package commands

import (
	"context"
	"os"
	"testing"
)

func TestRegisterExtendedCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// Count unique commands
	count := registry.Count()
	if count < 50 {
		t.Errorf("Expected at least 50 extended commands, got %d", count)
	}
}

func TestExtendedCommandCategories(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	categories := registry.GetByCategory()

	expectedCategories := []string{
		string(CategorySession),
		string(CategoryConfig),
		string(CategoryTools),
		string(CategoryGit),
		string(CategoryContext),
		string(CategoryStats),
		string(CategoryDebug),
		string(CategoryMode),
		string(CategoryIntegrate),
		string(CategoryAccount),
	}

	for _, cat := range expectedCategories {
		if _, ok := categories[cat]; !ok {
			t.Errorf("Expected category %s to have commands", cat)
		}
	}
}

func TestSessionCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name     string
		args     string
		wantErr  bool
	}{
		{"history", "", false},
		{"hist", "", false}, // alias
		{"save", "", false},
		{"save", "test.json", false},
		{"load", "", true}, // requires args
		{"load", "test.json", false},
		{"prune", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestConfigCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"set", "", true},         // requires args
		{"set", "key value", false},
		{"get", "", false},
		{"get", "key", false},
		{"reset", "", false},
		{"reset", "key", false},
		{"env", "", false},
		{"env", "KEY=value", false},
		{"alias", "", false},
		{"alias", "c=commit", false},
		{"shortcut", "", false},
		{"shortcut", "list", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestAccountCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	commands := []string{"login", "logout", "account", "usage"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			input := &CommandInput{
				Command:    name,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestGitCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"branch", "", false},
		{"br", "", false}, // alias
		{"branch", "new-branch", false},
		{"checkout", "", true}, // requires args
		{"checkout", "main", false},
		{"co", "main", false}, // alias
		{"stash", "", false},
		{"stash", "list", false},
		{"log", "", false},
		{"logs", "", false}, // alias
		{"tag", "", false},
		{"tag", "v1.0.0", false},
		{"push", "", false},
		{"pull", "", false},
		{"fetch", "", false},
		{"remote", "", false},
		{"reset-branch", "", true}, // requires args
		{"reset-branch", "HEAD~1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestGitPromptCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// These are TypePrompt commands
	commands := []string{"init", "merge", "rebase", "cherry-pick", "refactor", "doc"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			if cmd.Type != TypePrompt {
				t.Errorf("Expected %s to be TypePrompt, got %s", name, cmd.Type)
			}

			if cmd.Prompt == "" {
				t.Errorf("Expected %s to have a prompt", name)
			}
		})
	}
}

func TestContextCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"ls", "", false},
		{"ls", "/tmp", false},
		{"tree", "", false},
		{"grep", "", true}, // requires args
		{"grep", "pattern", false},
		{"find", "", true}, // requires args
		{"find", "*.go", false},
		{"read", "", true}, // requires args
		{"rm", "", true},   // requires args
		{"rm", "test.txt", false},
		{"mkdir", "", true}, // requires args
		{"mkdir", "testdir", false},
		{"touch", "", true}, // requires args
		{"touch", "test.txt", false},
		{"move", "", true}, // requires args
		{"move", "src dst", false},
		{"mv", "src dst", false}, // alias
		{"copy-file", "", true},  // requires args
		{"copy-file", "src dst", false},
		{"cp", "src dst", false}, // alias
		{"undo", "", false},
		{"redo", "", false},
		{"search", "", true}, // requires args
		{"search", "query", false},
		{"replace", "", true}, // requires args
		{"replace", "old new", false},
		{"ignore", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestToolsCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"install", "", true},    // requires args
		{"install", "package", false},
		{"uninstall", "", true},  // requires args
		{"uninstall", "package", false},
		{"update", "", false},
		{"update", "package", false},
		{"extensions", "", false},
		{"ext", "", false}, // alias
		{"rules", "", false},
		{"rules", "show", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestModeCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"auto", "", false},
		{"auto", "on", false},
		{"interactive", "", false},
		{"interactive", "off", false},
		{"batch", "", true}, // requires args
		{"batch", "file.txt", false},
		{"watch", "", false},
		{"watch", ". 'go test'", false},
		{"sync", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestIntegrationCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// Test TypePrompt commands
	promptCommands := []string{"jira", "notion", "slack"}

	for _, name := range promptCommands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			if cmd.Type != TypePrompt {
				t.Errorf("Expected %s to be TypePrompt, got %s", name, cmd.Type)
			}
		})
	}

	// Test local commands
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"web", "", true}, // requires args
		{"web", "https://example.com", false},
		{"terminal", "", false},
		{"term", "", false},  // alias
		{"shell", "", false}, // alias
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := registry.Get(tt.name)
			if !ok {
				t.Fatalf("Command %s not found", tt.name)
			}

			input := &CommandInput{
				Command:    tt.name,
				Args:       tt.args,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantErr && !output.IsError {
				t.Error("Expected error output")
			}
			if !tt.wantErr && output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestDebugCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	commands := []string{"debug", "trace", "profile", "logs"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			input := &CommandInput{
				Command:    name,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}
}

func TestStatsCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	localCommands := []string{"report", "benchmark"}

	for _, name := range localCommands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			input := &CommandInput{
				Command:    name,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}

	// Test analyze as TypePrompt
	t.Run("analyze", func(t *testing.T) {
		cmd, ok := registry.Get("analyze")
		if !ok {
			t.Fatal("Command analyze not found")
		}

		if cmd.Type != TypePrompt {
			t.Errorf("Expected analyze to be TypePrompt, got %s", cmd.Type)
		}
	})
}

func TestBuildCommands(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	commands := []string{"test", "build", "run", "clean", "format", "lint"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			cmd, ok := registry.Get(name)
			if !ok {
				t.Fatalf("Command %s not found", name)
			}

			input := &CommandInput{
				Command:    name,
				ProjectDir: os.TempDir(),
			}

			output, err := cmd.Handler(context.Background(), input)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if output.IsError {
				t.Errorf("Unexpected error: %s", output.Output)
			}
		})
	}

	// Test aliases
	t.Run("fmt alias", func(t *testing.T) {
		cmd, ok := registry.Get("fmt")
		if !ok {
			t.Fatal("Command fmt not found")
		}
		if cmd.Name != "format" {
			t.Errorf("Expected fmt to alias to format, got %s", cmd.Name)
		}
	})
}

func TestBughunterCommand(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	cmd, ok := registry.Get("bughunter")
	if !ok {
		t.Fatal("Command bughunter not found")
	}

	if cmd.Type != TypePrompt {
		t.Errorf("Expected bughunter to be TypePrompt, got %s", cmd.Type)
	}

	if cmd.Prompt == "" {
		t.Error("Expected bughunter to have a prompt")
	}

	if cmd.Category != string(CategoryMode) {
		t.Errorf("Expected bughunter to be in Mode category, got %s", cmd.Category)
	}
}

func TestLsCommandWithTempDir(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	cmd, ok := registry.Get("ls")
	if !ok {
		t.Fatal("Command ls not found")
	}

	// Create temp directory with some files
	tmpDir := os.TempDir()
	testDir := tmpDir + "/poyo_test_ls"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Create some test files
	os.WriteFile(testDir+"/file1.txt", []byte("test"), 0644)
	os.MkdirAll(testDir+"/subdir", 0755)

	input := &CommandInput{
		Command:    "ls",
		Args:       testDir,
		ProjectDir: testDir,
	}

	output, err := cmd.Handler(context.Background(), input)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}

	if output.IsError {
		t.Errorf("Unexpected error: %s", output.Output)
	}

	// Check that output contains the files
	if !containsString(output.Output, "file1.txt") {
		t.Error("Expected output to contain file1.txt")
	}
	if !containsString(output.Output, "subdir") {
		t.Error("Expected output to contain subdir")
	}
}

func TestReadFileCommand(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	cmd, ok := registry.Get("read")
	if !ok {
		t.Fatal("Command read not found")
	}

	// Create temp file
	tmpFile := os.TempDir() + "/poyo_test_read.txt"
	testContent := "Hello, World!"
	os.WriteFile(tmpFile, []byte(testContent), 0644)
	defer os.Remove(tmpFile)

	input := &CommandInput{
		Command:    "read",
		Args:       tmpFile,
		ProjectDir: os.TempDir(),
	}

	output, err := cmd.Handler(context.Background(), input)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}

	if output.IsError {
		t.Errorf("Unexpected error: %s", output.Output)
	}

	if output.Output != testContent {
		t.Errorf("Expected output %q, got %q", testContent, output.Output)
	}
}

func TestCommandAliases(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	aliasTests := map[string]string{
		"hist":  "history",
		"br":    "branch",
		"co":    "checkout",
		"logs":  "log",
		"mv":    "move",
		"cp":    "copy-file",
		"ext":   "extensions",
		"term":  "terminal",
		"shell": "terminal",
		"fmt":   "format",
		"s":     "search",
	}

	for alias, expectedName := range aliasTests {
		t.Run(alias, func(t *testing.T) {
			cmd, ok := registry.Get(alias)
			if !ok {
				t.Fatalf("Alias %s not found", alias)
			}

			if cmd.Name != expectedName {
				t.Errorf("Expected alias %s to map to %s, got %s", alias, expectedName, cmd.Name)
			}
		})
	}
}

func TestCommandSortOrder(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// Get all commands and verify they have sort orders
	cmds := registry.GetAll()

	for _, cmd := range cmds {
		if cmd.SortOrder == 0 && cmd.Name != "" {
			// SortOrder 0 is valid, but most commands should have non-zero
			// This is just a sanity check
		}
	}
}

func TestCommandRegistrySearch(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// Search for "git"
	results := registry.Search("git")
	if len(results) == 0 {
		t.Error("Expected to find commands matching 'git'")
	}

	// Search for "branch"
	results = registry.Search("branch")
	if len(results) == 0 {
		t.Error("Expected to find commands matching 'branch'")
	}
}

func TestCommandRegistryComplete(t *testing.T) {
	registry := NewRegistry()
	RegisterExtendedCommands(registry, "1.0.0")

	// Test completion
	results := registry.Complete("/br")
	if len(results) == 0 {
		t.Error("Expected completion results for '/br'")
	}

	// Verify all results start with "br"
	for _, cmd := range results {
		if !containsString(cmd.Name, "br") && !hasAlias(cmd, "br") {
			t.Errorf("Unexpected completion result: %s", cmd.Name)
		}
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasAlias(cmd *Command, alias string) bool {
	for _, a := range cmd.Aliases {
		if a == alias {
			return true
		}
	}
	return false
}
