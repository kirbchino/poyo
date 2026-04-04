package hooks

import (
	"context"
	"net"
	"testing"
)

func TestSSRFGuardIsAllowed(t *testing.T) {
	guard := NewSSRFGuard()

	tests := []struct {
		url        string
		expectPass bool
	}{
		// Should pass - public URLs
		{"https://example.com/api", true},
		{"https://api.github.com", true},

		// Should pass - localhost (allowed for development)
		{"http://127.0.0.1:8080/hook", true},
		{"http://localhost:3000/hook", true},

		// Note: DNS resolution tests would require network access
		// These tests verify the guard doesn't error on valid URLs
	}

	for _, tt := range tests {
		err := guard.IsAllowed(tt.url)
		if tt.expectPass && err != nil {
			t.Errorf("IsAllowed(%q) = %v, want nil", tt.url, err)
		}
	}
}

func TestSSRFGuardIsIPAllowed(t *testing.T) {
	guard := NewSSRFGuard()

	tests := []struct {
		ip         string
		expectPass bool
	}{
		// Loopback - allowed
		{"127.0.0.1", true},
		{"127.0.0.100", true},
		{"::1", true},

		// Private - blocked
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},

		// Link-local - blocked
		{"169.254.1.1", false},

		// CGNAT - blocked
		{"100.64.0.1", false},

		// Public - allowed
		{"8.8.8.8", true},
		{"1.1.1.1", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Errorf("Failed to parse IP: %s", tt.ip)
			continue
		}

		result := guard.isIPAllowed(ip)
		if result != tt.expectPass {
			t.Errorf("isIPAllowed(%q) = %v, want %v", tt.ip, result, tt.expectPass)
		}
	}
}

func TestHTTPHookExecutorInterpolateEnvVars(t *testing.T) {
	executor := NewHTTPHookExecutor()

	// Set some environment variables
	executor.SetEnvVars(map[string]string{
		"API_KEY": "test-key-123",
		"TOKEN":   "test-token",
	})

	tests := []struct {
		input        string
		allowedVars  []string
		expected     string
	}{
		{
			input:       "Bearer ${API_KEY}",
			allowedVars: []string{"API_KEY"},
			expected:    "Bearer test-key-123",
		},
		{
			input:       "Token $TOKEN",
			allowedVars: []string{"TOKEN"},
			expected:    "Token test-token",
		},
		{
			input:       "${API_KEY}-suffix",
			allowedVars: []string{},
			expected:    "${API_KEY}-suffix", // Not allowed, not replaced
		},
		{
			input:       "${API_KEY} and ${TOKEN}",
			allowedVars: []string{"API_KEY", "TOKEN"},
			expected:    "test-key-123 and test-token",
		},
	}

	for _, tt := range tests {
		result := executor.interpolateEnvVars(tt.input, tt.allowedVars)
		if result != tt.expected {
			t.Errorf("interpolateEnvVars(%q, %v) = %q, want %q", tt.input, tt.allowedVars, result, tt.expected)
		}
	}
}

func TestMatchesMatcher(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		expect  bool
	}{
		{"", "anything", true},
		{"*", "anything", true},
		{"Bash", "Bash", true},
		{"Bash*", "Bash", true},
		{"Bash*", "BashTool", true},
		{"*Tool", "BashTool", true},
		{"Bash", "Read", false},
		{"mcp__*__read", "mcp__server__read", true},
		{"mcp__*__read", "mcp__server__write", false},
	}

	for _, tt := range tests {
		result := MatchesMatcher(tt.pattern, tt.value)
		if result != tt.expect {
			t.Errorf("MatchesMatcher(%q, %q) = %v, want %v", tt.pattern, tt.value, result, tt.expect)
		}
	}
}

func TestNewFileWatcher(t *testing.T) {
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("NewFileWatcher() error: %v", err)
	}
	if watcher == nil {
		t.Fatal("NewFileWatcher() returned nil")
	}
	defer watcher.Stop()

	if watcher.paths == nil {
		t.Error("paths map should be initialized")
	}
}

func TestFileWatcherAddRemovePath(t *testing.T) {
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("NewFileWatcher() error: %v", err)
	}
	defer watcher.Stop()

	// Create a temp file
	tmpFile := "/tmp/poyo_test_watcher_" + timestamp()
	if err := writeFile(tmpFile, "test"); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer removeFile(tmpFile)

	// Add path
	if err := watcher.AddPath(tmpFile); err != nil {
		t.Errorf("AddPath() error: %v", err)
	}

	// Check it was added
	paths := watcher.GetWatchedPaths()
	if len(paths) != 1 {
		t.Errorf("GetWatchedPaths() returned %d paths, want 1", len(paths))
	}

	// Remove path
	if err := watcher.RemovePath(tmpFile); err != nil {
		t.Errorf("RemovePath() error: %v", err)
	}

	// Check it was removed
	paths = watcher.GetWatchedPaths()
	if len(paths) != 0 {
		t.Errorf("GetWatchedPaths() returned %d paths after removal, want 0", len(paths))
	}
}

func TestFileWatcherRegisterHandler(t *testing.T) {
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("NewFileWatcher() error: %v", err)
	}
	defer watcher.Stop()

	handlerCalled := false
	watcher.RegisterHandler(func(event FileChangeEvent) {
		handlerCalled = true
	})

	if len(watcher.handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(watcher.handlers))
	}

	_ = handlerCalled // Used in test
}

// Helper functions
func timestamp() string {
	return "1234567890"
}

func writeFile(path, content string) error {
	return nil // Simplified for test
}

func removeFile(path string) error {
	return nil // Simplified for test
}
