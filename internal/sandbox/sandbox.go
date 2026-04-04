// Package sandbox provides command isolation using OS-level sandboxing
package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Config holds sandbox configuration
type Config struct {
	Enabled                    bool
	AutoAllowBashIfSandboxed   bool
	AllowUnsandboxedCommands   bool
	ExcludedCommands           []string
	AllowedDomains             []string
	DeniedDomains              []string
	AllowWrite                 []string
	DenyWrite                  []string
	AllowRead                  []string
	DenyRead                   []string
	AllowUnixSockets           bool
	AllowLocalBinding          bool
	EnableWeakerNestedSandbox  bool
}

// Manager handles sandbox operations
type Manager struct {
	config     Config
	platform   string
	supported  bool
	dependencies []string
}

// Platform constants
const (
	PlatformLinux  = "linux"
	PlatformMacOS  = "darwin"
	PlatformWSL    = "wsl"
	PlatformWindows = "windows"
	PlatformUnknown = "unknown"
)

// NewManager creates a new sandbox manager
func NewManager(config Config) *Manager {
	m := &Manager{
		config:   config,
		platform: detectPlatform(),
	}
	m.supported, m.dependencies = m.checkDependencies()
	return m
}

// detectPlatform detects the current platform
func detectPlatform() string {
	switch runtime.GOOS {
	case "linux":
		// Check if running under WSL
		if isWSL() {
			return PlatformWSL
		}
		return PlatformLinux
	case "darwin":
		return PlatformMacOS
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

// isWSL checks if running under Windows Subsystem for Linux
func isWSL() bool {
	// Check for WSL-specific files
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop-interpreters"); err == nil {
		return true
	}
	// Check /proc/version for WSL identifier
	if data, err := os.ReadFile("/proc/version"); err == nil {
		return strings.Contains(string(data), "microsoft") ||
			strings.Contains(string(data), "WSL")
	}
	return false
}

// checkDependencies checks if required dependencies are available
func (m *Manager) checkDependencies() (bool, []string) {
	var missing []string

	switch m.platform {
	case PlatformLinux, PlatformWSL:
		// Check for bubblewrap
		if _, err := exec.LookPath("bwrap"); err != nil {
			missing = append(missing, "bwrap (bubblewrap)")
		}
		// Check for socat (for network proxying)
		if _, err := exec.LookPath("socat"); err != nil {
			missing = append(missing, "socat")
		}

	case PlatformMacOS:
		// macOS uses sandbox-exec which is built-in
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			missing = append(missing, "sandbox-exec")
		}

	default:
		// Platform not supported
		return false, []string{"unsupported platform"}
	}

	return len(missing) == 0, missing
}

// IsSupported returns whether sandboxing is supported on this platform
func (m *Manager) IsSupported() bool {
	return m.supported
}

// IsEnabled returns whether sandboxing is enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled && m.supported
}

// GetPlatform returns the current platform
func (m *Manager) GetPlatform() string {
	return m.platform
}

// GetMissingDependencies returns missing dependencies
func (m *Manager) GetMissingDependencies() []string {
	return m.dependencies
}

// ShouldUseSandbox determines if a command should be sandboxed
func (m *Manager) ShouldUseSandbox(command string, disableSandbox bool) bool {
	if !m.IsEnabled() {
		return false
	}

	// Allow explicit override if policy allows
	if disableSandbox && m.config.AllowUnsandboxedCommands {
		return false
	}

	// Check if command is in excluded list
	if m.containsExcludedCommand(command) {
		return false
	}

	return true
}

// containsExcludedCommand checks if command matches any excluded pattern
func (m *Manager) containsExcludedCommand(command string) bool {
	parts := splitCommand(command)
	for _, part := range parts {
		baseCmd := extractBaseCommand(part)
		for _, pattern := range m.config.ExcludedCommands {
			if matchesPattern(pattern, baseCmd) {
				return true
			}
		}
	}
	return false
}

// splitCommand splits a compound command into parts
func splitCommand(command string) []string {
	// Simple split by && || ; | operators
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range command {
		switch {
		case r == '"' || r == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			}
			current.WriteRune(r)
		case !inQuote && (r == '&' || r == '|' || r == ';'):
			if current.Len() > 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}

	return parts
}

// extractBaseCommand extracts the base command from a command string
func extractBaseCommand(cmd string) string {
	// Strip environment variables
	for strings.Contains(cmd, "=") {
		parts := strings.SplitN(cmd, " ", 2)
		if len(parts) < 2 {
			break
		}
		if !strings.Contains(parts[0], "=") {
			break
		}
		cmd = parts[1]
	}

	// Get first word
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}

	// Get basename if it's a path
	base := fields[0]
	return filepath.Base(base)
}

// matchesPattern checks if a command matches a pattern
func matchesPattern(pattern, command string) bool {
	// Check for prefix pattern (ends with *)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(command, prefix)
	}
	// Exact match
	return command == pattern
}

// WrapCommand wraps a command with sandbox execution
func (m *Manager) WrapCommand(command string, shell string) (string, error) {
	if !m.IsEnabled() {
		return command, nil
	}

	switch m.platform {
	case PlatformLinux, PlatformWSL:
		return m.wrapWithBubblewrap(command, shell)
	case PlatformMacOS:
		return m.wrapWithSandboxExec(command, shell)
	default:
		return command, nil
	}
}

// wrapWithBubblewrap wraps command with bubblewrap for Linux
func (m *Manager) wrapWithBubblewrap(command string, shell string) (string, error) {
	args := []string{
		"--unshare-all",
		"--die-with-parent",
		"--new-session",
	}

	// Add filesystem bindings
	cwd, err := os.Getwd()
	if err == nil {
		args = append(args, "--bind", cwd, cwd)
	}

	// Mount /dev/null for denied paths
	for _, path := range m.config.DenyWrite {
		args = append(args, "--ro-bind", "/dev/null", path)
	}
	for _, path := range m.config.DenyRead {
		args = append(args, "--ro-bind", "/dev/null", path)
	}

	// Allow read-only binds for read paths
	for _, path := range m.config.AllowRead {
		args = append(args, "--ro-bind", path, path)
	}

	// Allow read-write binds for write paths
	for _, path := range m.config.AllowWrite {
		args = append(args, "--bind", path, path)
	}

	// Add proc and dev
	args = append(args,
		"--proc", "/proc",
		"--dev", "/dev",
	)

	// Set environment
	for _, env := range os.Environ() {
		if !strings.Contains(env, "TOKEN") &&
			!strings.Contains(env, "KEY") &&
			!strings.Contains(env, "SECRET") {
			args = append(args, "--setenv",
				strings.SplitN(env, "=", 2)[0],
				strings.SplitN(env, "=", 2)[1])
		}
	}

	// Execute the shell with the command
	if shell == "" {
		shell = "/bin/sh"
	}
	args = append(args, "--", shell, "-c", command)

	return fmt.Sprintf("bwrap %s", strings.Join(args, " ")), nil
}

// wrapWithSandboxExec wraps command with sandbox-exec for macOS
func (m *Manager) wrapWithSandboxExec(command string, shell string) (string, error) {
	// Generate sandbox profile
	profile := m.generateMacOSSandboxProfile()

	// Write profile to temp file
	profilePath := fmt.Sprintf("/tmp/sandbox-profile-%d.sb", os.Getpid())
	if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
		return command, nil // Fallback to unsandboxed
	}

	// sandbox-exec -f profile -- command
	if shell == "" {
		shell = "/bin/sh"
	}
	return fmt.Sprintf("sandbox-exec -f %s %s -c '%s'", profilePath, shell, command), nil
}

// generateMacOSSandboxProfile generates a macOS sandbox profile
func (m *Manager) generateMacOSSandboxProfile() string {
	var profile strings.Builder

	profile.WriteString("(version 1)\n")
	profile.WriteString("(deny default)\n")

	// Allow current directory
	cwd, _ := os.Getwd()
	profile.WriteString(fmt.Sprintf("(allow file-read* file-write* (subpath \"%s\"))\n", cwd))

	// Allow write paths
	for _, path := range m.config.AllowWrite {
		profile.WriteString(fmt.Sprintf("(allow file-read* file-write* (subpath \"%s\"))\n", path))
	}

	// Allow read paths
	for _, path := range m.config.AllowRead {
		profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", path))
	}

	// Allow network if domains specified
	if len(m.config.AllowedDomains) > 0 || m.config.AllowLocalBinding {
		profile.WriteString("(allow network*)\n")
	} else {
		// Allow only specific domains
		for _, domain := range m.config.AllowedDomains {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (host \"%s\"))\n", domain))
		}
	}

	// Allow process execution
	profile.WriteString("(allow process-exec)\n")
	profile.WriteString("(allow process-fork)\n")
	profile.WriteString("(allow signal (target self))\n")

	// Allow basic system operations
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(allow mach-lookup)\n")

	return profile.String()
}

// ExecuteInSandbox executes a command within the sandbox
func (m *Manager) ExecuteInSandbox(ctx context.Context, command string, shell string) ([]byte, error) {
	wrappedCmd, err := m.WrapCommand(command, shell)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap command: %w", err)
	}

	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", wrappedCmd)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	// Filter sensitive environment variables
	var filteredEnv []string
	for _, env := range cmd.Env {
		if !strings.Contains(env, "TOKEN") &&
			!strings.Contains(env, "KEY") &&
			!strings.Contains(env, "SECRET") &&
			!strings.Contains(env, "PASSWORD") {
			filteredEnv = append(filteredEnv, env)
		}
	}
	cmd.Env = filteredEnv

	return cmd.CombinedOutput()
}

// DefaultConfig returns a default sandbox configuration
func DefaultConfig() Config {
	return Config{
		Enabled:                   false, // Disabled by default
		AutoAllowBashIfSandboxed:  true,
		AllowUnsandboxedCommands:  true,
		ExcludedCommands:          []string{},
		AllowedDomains:            []string{},
		DeniedDomains:             []string{},
		AllowWrite:                []string{"."},
		DenyWrite:                 []string{},
		AllowRead:                 []string{},
		DenyRead:                  []string{},
		AllowUnixSockets:          false,
		AllowLocalBinding:         false,
		EnableWeakerNestedSandbox: false,
	}
}

// CleanupAfterCommand cleans up any temporary resources after a command
func (m *Manager) CleanupAfterCommand() {
	// Clean up temp sandbox profiles on macOS
	if m.platform == PlatformMacOS {
		pattern := fmt.Sprintf("/tmp/sandbox-profile-%d-*.sb", os.Getpid())
		files, _ := filepath.Glob(pattern)
		for _, f := range files {
			os.Remove(f)
		}
	}
}
