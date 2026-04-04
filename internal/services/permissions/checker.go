// Package permissions provides permission checking functionality
package permissions

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
)

// PermissionMode represents the permission mode
type PermissionMode string

const (
	// PermissionModeDefault requires confirmation for all sensitive operations
	PermissionModeDefault PermissionMode = "default"
	// PermissionModePlan only allows read operations
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeAcceptEdits trusts file edits within project
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModeAuto uses AI classifier for decisions
	PermissionModeAuto PermissionMode = "auto"
	// PermissionModeBypassPermissions allows all operations
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	// PermissionModeDontAsk denies all operations
	PermissionModeDontAsk PermissionMode = "dontAsk"
)

// Decision represents a permission decision
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
	DecisionAsk   Decision = "ask"
)

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Decision Decision
	Reason   string
	Source   PermissionSource
}

// PermissionSource indicates where the decision came from
type PermissionSource struct {
	Type   string
	RuleID string
}

// PermissionContext contains context for permission decisions
type PermissionContext struct {
	Mode          PermissionMode
	WorkingDir    string
	ProjectRoot   string
	SessionID     string
	UserRules     []PermissionRule
	SystemRules   []PermissionRule
	TrustedPaths  []string
	DeniedPaths   []string
}

// PermissionRule represents a permission rule
type PermissionRule struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Priority    int
	Condition   RuleCondition
	Action      RuleAction
}

// RuleCondition defines when a rule applies
type RuleCondition struct {
	Tool       string
	Parameters map[string]Matcher
	Path       *PathCondition
	Command    *CommandCondition
}

// RuleAction defines what action to take
type RuleAction struct {
	Decision Decision
	Reason   string
}

// Matcher represents a parameter matcher
type Matcher interface {
	Match(value interface{}) bool
}

// PathCondition defines path matching rules
type PathCondition struct {
	Patterns       []string
	InsideWorkspace bool
	ExcludePatterns []string
}

// CommandCondition defines command matching rules
type CommandCondition struct {
	AllowedCommands  []string
	BlockedCommands  []string
	AllowSudo       bool
	AllowPipe       bool
	AllowRedirect   bool
}

// PermissionRequest represents a permission request
type PermissionRequest struct {
	Tool   string
	Input  map[string]interface{}
}

// PermissionChecker checks permissions
type PermissionChecker struct {
	mode         PermissionMode
	rules        []PermissionRule
	workingDir   string
	projectRoot  string
	trustedPaths []string
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(mode PermissionMode, workingDir string) *PermissionChecker {
	return &PermissionChecker{
		mode:        mode,
		workingDir:  workingDir,
		projectRoot: workingDir,
		rules:       DefaultRules(),
	}
}

// Check checks permission for a request
func (c *PermissionChecker) Check(ctx context.Context, req *PermissionRequest) *PermissionResult {
	// Mode-based decisions
	switch c.mode {
	case PermissionModeBypassPermissions:
		return &PermissionResult{
			Decision: DecisionAllow,
			Reason:   "Bypass permissions mode",
			Source:   PermissionSource{Type: "mode"},
		}

	case PermissionModeDontAsk:
		return &PermissionResult{
			Decision: DecisionDeny,
			Reason:   "Dont ask mode",
			Source:   PermissionSource{Type: "mode"},
		}

	case PermissionModePlan:
		return c.checkPlanMode(req)
	}

	// Check rules
	for _, rule := range c.rules {
		if !rule.Enabled {
			continue
		}

		if c.matchesRule(req, &rule) {
			return &PermissionResult{
				Decision: rule.Action.Decision,
				Reason:   rule.Action.Reason,
				Source:   PermissionSource{Type: "rule", RuleID: rule.ID},
			}
		}
	}

	// Default: ask user
	return &PermissionResult{
		Decision: DecisionAsk,
		Reason:   "No matching rule",
		Source:   PermissionSource{Type: "default"},
	}
}

// checkPlanMode checks permissions in plan mode (read-only)
func (c *PermissionChecker) checkPlanMode(req *PermissionRequest) *PermissionResult {
	readOnlyTools := map[string]bool{
		"Read":    true,
		"Glob":    true,
		"Grep":    true,
		"WebFetch": true,
	}

	if readOnlyTools[req.Tool] {
		return &PermissionResult{
			Decision: DecisionAllow,
			Reason:   "Read-only operation in plan mode",
			Source:   PermissionSource{Type: "mode"},
		}
	}

	return &PermissionResult{
		Decision: DecisionDeny,
		Reason:   "Write operations not allowed in plan mode",
		Source:   PermissionSource{Type: "mode"},
	}
}

// matchesRule checks if a request matches a rule
func (c *PermissionChecker) matchesRule(req *PermissionRequest, rule *PermissionRule) bool {
	cond := rule.Condition

	// Check tool match
	if cond.Tool != "" && cond.Tool != req.Tool {
		return false
	}

	// Check path condition
	if cond.Path != nil {
		pathInput, ok := req.Input["file_path"].(string)
		if !ok {
			pathInput, ok = req.Input["path"].(string)
		}
		if ok && !c.matchesPathCondition(pathInput, cond.Path) {
			return false
		}
	}

	// Check command condition
	if cond.Command != nil {
		cmdInput, ok := req.Input["command"].(string)
		if ok && !c.matchesCommandCondition(cmdInput, cond.Command) {
			return false
		}
	}

	return true
}

// matchesPathCondition checks if a path matches the condition
func (c *PermissionChecker) matchesPathCondition(path string, cond *PathCondition) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check workspace constraint
	if cond.InsideWorkspace {
		if !strings.HasPrefix(absPath, c.workingDir) {
			return false
		}
	}

	// Check patterns
	if len(cond.Patterns) > 0 {
		matched := false
		for _, pattern := range cond.Patterns {
			if m, _ := filepath.Match(pattern, absPath); m {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range cond.ExcludePatterns {
		if m, _ := filepath.Match(pattern, absPath); m {
			return false
		}
	}

	return true
}

// matchesCommandCondition checks if a command matches the condition
func (c *PermissionChecker) matchesCommandCondition(cmd string, cond *CommandCondition) bool {
	// Extract base command
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	baseCmd := parts[0]

	// Check allowed commands
	if len(cond.AllowedCommands) > 0 {
		allowed := false
		for _, ac := range cond.AllowedCommands {
			if ac == baseCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check blocked commands
	for _, bc := range cond.BlockedCommands {
		if bc == baseCmd {
			return false
		}
	}

	// Check sudo
	if !cond.AllowSudo && strings.Contains(cmd, "sudo") {
		return false
	}

	// Check pipe
	if !cond.AllowPipe && strings.Contains(cmd, "|") {
		return false
	}

	// Check redirect
	if !cond.AllowRedirect && (strings.Contains(cmd, ">") || strings.Contains(cmd, "<")) {
		return false
	}

	return true
}

// DefaultRules returns default permission rules
func DefaultRules() []PermissionRule {
	return []PermissionRule{
		{
			ID:          "allow_read_workspace",
			Name:        "Allow Read in Workspace",
			Description: "Allow reading files within the workspace",
			Enabled:     true,
			Priority:    100,
			Condition: RuleCondition{
				Tool: "Read",
				Path: &PathCondition{
					InsideWorkspace: true,
				},
			},
			Action: RuleAction{
				Decision: DecisionAllow,
				Reason:   "Reading files in workspace is allowed",
			},
		},
		{
			ID:          "block_sensitive_files",
			Name:        "Block Sensitive Files",
			Description: "Block access to sensitive files like .env",
			Enabled:     true,
			Priority:    200,
			Condition: RuleCondition{
				Path: &PathCondition{
					Patterns: []string{
						"*.env",
						"*.env.*",
						"*credentials*",
						"*secret*",
						"*.pem",
						"*.key",
					},
				},
			},
			Action: RuleAction{
				Decision: DecisionDeny,
				Reason:   "Access to sensitive files is blocked",
			},
		},
		{
			ID:          "block_dangerous_commands",
			Name:        "Block Dangerous Commands",
			Description: "Block dangerous shell commands",
			Enabled:     true,
			Priority:    200,
			Condition: RuleCondition{
				Command: &CommandCondition{
					BlockedCommands: []string{
						"rm", "rmdir", "mkfs", "dd", "fdisk",
						"shutdown", "reboot", "halt", "poweroff",
					},
				},
			},
			Action: RuleAction{
				Decision: DecisionDeny,
				Reason:   "Dangerous commands are blocked",
			},
		},
	}
}

// SetMode sets the permission mode
func (c *PermissionChecker) SetMode(mode PermissionMode) {
	c.mode = mode
}

// GetMode returns the current permission mode
func (c *PermissionChecker) GetMode() PermissionMode {
	return c.mode
}

// AddRule adds a permission rule
func (c *PermissionChecker) AddRule(rule PermissionRule) {
	c.rules = append(c.rules, rule)
	// Sort by priority descending
	for i := len(c.rules) - 1; i > 0; i-- {
		if c.rules[i].Priority > c.rules[i-1].Priority {
			c.rules[i], c.rules[i-1] = c.rules[i-1], c.rules[i]
		}
	}
}

// AnalyzeCommand analyzes a command for security risks
func AnalyzeCommand(cmd string) *CommandAnalysis {
	analysis := &CommandAnalysis{
		Command:     cmd,
		BaseCommand: extractBaseCommand(cmd),
		RiskLevel:   RiskLevelLow,
		Warnings:    []string{},
	}

	// Check for dangerous patterns
	dangerousPatterns := []struct {
		pattern   *regexp.Regexp
		risk      RiskLevel
		warning   string
	}{
		{regexp.MustCompile(`rm\s+-rf`), RiskLevelCritical, "Recursive force delete"},
		{regexp.MustCompile(`sudo`), RiskLevelHigh, "Elevated privileges"},
		{regexp.MustCompile(`>\s*/dev/`), RiskLevelCritical, "Writing to device files"},
		{regexp.MustCompile(`mkfs`), RiskLevelCritical, "Filesystem formatting"},
		{regexp.MustCompile(`dd\s+if=`), RiskLevelHigh, "Disk duplication"},
		{regexp.MustCompile(`chmod\s+777`), RiskLevelHigh, "Unrestricted permissions"},
		{regexp.MustCompile(`curl.*\|.*bash`), RiskLevelHigh, "Piping curl to bash"},
		{regexp.MustCompile(`wget.*\|.*bash`), RiskLevelHigh, "Piping wget to bash"},
	}

	for _, dp := range dangerousPatterns {
		if dp.pattern.MatchString(cmd) {
			analysis.Warnings = append(analysis.Warnings, dp.warning)
			if dp.risk > analysis.RiskLevel {
				analysis.RiskLevel = dp.risk
			}
		}
	}

	// Check for read-only commands
	readOnlyPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^ls(\s|$)`),
		regexp.MustCompile(`^cat(\s|$)`),
		regexp.MustCompile(`^head(\s|$)`),
		regexp.MustCompile(`^tail(\s|$)`),
		regexp.MustCompile(`^grep(\s|$)`),
		regexp.MustCompile(`^find(\s|$)`),
		regexp.MustCompile(`^git\s+status`),
		regexp.MustCompile(`^git\s+log`),
		regexp.MustCompile(`^git\s+diff`),
	}

	for _, rp := range readOnlyPatterns {
		if rp.MatchString(cmd) {
			analysis.IsReadOnly = true
			break
		}
	}

	return analysis
}

// RiskLevel represents the risk level of a command
type RiskLevel int

const (
	RiskLevelLow      RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

// CommandAnalysis contains analysis of a command
type CommandAnalysis struct {
	Command     string
	BaseCommand string
	RiskLevel   RiskLevel
	IsReadOnly  bool
	Warnings    []string
}

// extractBaseCommand extracts the base command from a command string
func extractBaseCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
