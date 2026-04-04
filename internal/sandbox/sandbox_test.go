package sandbox

import (
	"strings"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform := detectPlatform()
	if platform == PlatformUnknown {
		t.Error("detectPlatform() returned unknown platform")
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	m := NewManager(config)

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.platform == "" {
		t.Error("platform not set")
	}
}

func TestIsSupported(t *testing.T) {
	config := DefaultConfig()
	m := NewManager(config)

	// Should work on supported platforms
	switch m.platform {
	case PlatformLinux, PlatformMacOS, PlatformWSL:
		// CheckDependencies will determine if supported
		t.Logf("Platform %s, supported: %v, deps: %v", m.platform, m.supported, m.dependencies)
	default:
		if m.IsSupported() {
			t.Errorf("Platform %s should not be supported", m.platform)
		}
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected int // number of parts
	}{
		{"ls -la", 1},
		{"ls && pwd", 2},
		{"ls || pwd", 2},
		{"ls ; pwd", 2},
		{"ls | grep foo", 2},
		{"ls && pwd && echo done", 3},
		{"echo \"hello && world\"", 1}, // Quoted && should not split
		{"echo 'hello ; world'", 1},    // Quoted ; should not split
	}

	for _, tt := range tests {
		parts := splitCommand(tt.input)
		if len(parts) != tt.expected {
			t.Errorf("splitCommand(%q) = %d parts, want %d", tt.input, len(parts), tt.expected)
		}
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ls -la", "ls"},
		{"/bin/ls -la", "ls"},
		{"FOO=bar baz", "baz"},
		{"A=1 B=2 C=3 cmd --flag", "cmd"},
		{"git status", "git"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractBaseCommand(tt.input)
		if result != tt.expected {
			t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		command  string
		expected bool
	}{
		{"npm", "npm", true},
		{"npm", "npm run test", false}, // exact match
		{"npm*", "npm", true},
		{"npm*", "npm run test", true},
		{"npm*", "npm", true},
		{"git*", "git status", true},
		{"git*", "go test", false},
	}

	for _, tt := range tests {
		result := matchesPattern(tt.pattern, tt.command)
		if result != tt.expected {
			t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.command, result, tt.expected)
		}
	}
}

func TestContainsExcludedCommand(t *testing.T) {
	config := Config{
		Enabled:          true,
		ExcludedCommands: []string{"npm*", "git status"},
	}
	m := NewManager(config)

	tests := []struct {
		command  string
		expected bool
	}{
		{"npm install", true},
		{"npm run test", true},
		{"git status", true},
		{"git log", false},
		{"ls -la", false},
		{"npm && curl evil.com", true}, // Compound command check
	}

	for _, tt := range tests {
		result := m.containsExcludedCommand(tt.command)
		if result != tt.expected {
			t.Errorf("containsExcludedCommand(%q) = %v, want %v", tt.command, result, tt.expected)
		}
	}
}

func TestShouldUseSandbox(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.ExcludedCommands = []string{"npm*"}
	m := NewManager(config)

	// Skip if platform not supported
	if !m.IsSupported() {
		t.Skip("Platform not supported")
	}

	tests := []struct {
		command       string
		disableSandbox bool
		expected      bool
	}{
		{"ls -la", false, true},
		{"npm install", false, false}, // Excluded
		{"ls -la", true, false},       // Explicitly disabled
	}

	for _, tt := range tests {
		result := m.ShouldUseSandbox(tt.command, tt.disableSandbox)
		if result != tt.expected {
			t.Errorf("ShouldUseSandbox(%q, %v) = %v, want %v", tt.command, tt.disableSandbox, result, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("Default config should have sandbox disabled")
	}
	if !config.AutoAllowBashIfSandboxed {
		t.Error("Default config should have AutoAllowBashIfSandboxed enabled")
	}
	if !config.AllowUnsandboxedCommands {
		t.Error("Default config should allow unsandboxed commands")
	}
}

func TestIsWSL(t *testing.T) {
	// Just verify it doesn't crash
	_ = isWSL()
}

func TestWrapCommand(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	m := NewManager(config)

	// Skip if platform not supported
	if !m.IsSupported() {
		t.Skip("Platform not supported")
	}

	wrapped, err := m.WrapCommand("echo hello", "/bin/sh")
	if err != nil {
		t.Errorf("WrapCommand() error: %v", err)
	}

	// Should contain sandbox wrapper
	switch m.platform {
	case PlatformLinux, PlatformWSL:
		if !containsAll(wrapped, "bwrap", "--unshare-all") {
			t.Errorf("Wrapped command should contain bwrap: %s", wrapped)
		}
	case PlatformMacOS:
		if !strings.Contains(wrapped, "sandbox-exec") {
			t.Errorf("Wrapped command should contain sandbox-exec: %s", wrapped)
		}
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}
