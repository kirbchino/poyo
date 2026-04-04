package permission

import (
	"testing"
)

func TestMode(t *testing.T) {
	modes := []Mode{ModeAsk, ModeAccept, ModeAuto, ModeDeny}
	for _, m := range modes {
		if string(m) == "" {
			t.Errorf("Mode should have a non-empty string representation")
		}
	}
}

func TestSourcePriority(t *testing.T) {
	tests := []struct {
		source         Source
		expectedPriority Priority
	}{
		{SourceSession, PrioritySession},
		{SourcePolicy, PriorityPolicy},
		{SourceLocal, PriorityLocal},
		{SourceProject, PriorityProject},
		{SourceUser, PriorityUser},
		{SourceAuto, PriorityAuto},
	}

	for _, tt := range tests {
		priority := tt.source.GetPriority()
		if priority != tt.expectedPriority {
			t.Errorf("Source(%q).GetPriority() = %d, want %d", tt.source, priority, tt.expectedPriority)
		}
	}
}

func TestRuleGetTools(t *testing.T) {
	tests := []struct {
		rule    Rule
		expect  []string
	}{
		{Rule{Tool: "Bash"}, []string{"Bash"}},
		{Rule{Tool: []string{"Read", "Write"}}, []string{"Read", "Write"}},
		{Rule{Tool: "*"}, []string{"*"}},
		{Rule{Tool: nil}, nil},
	}

	for _, tt := range tests {
		tools := tt.rule.GetTools()
		if len(tools) != len(tt.expect) {
			t.Errorf("GetTools() = %v, want %v", tools, tt.expect)
			continue
		}
		for i, tool := range tools {
			if tool != tt.expect[i] {
				t.Errorf("GetTools()[%d] = %q, want %q", i, tool, tt.expect[i])
			}
		}
	}
}

func TestRuleMatchesTool(t *testing.T) {
	tests := []struct {
		rule     Rule
		toolName string
		expect   bool
	}{
		{Rule{Tool: "Bash"}, "Bash", true},
		{Rule{Tool: "Bash"}, "Read", false},
		{Rule{Tool: "*"}, "Anything", true},
		{Rule{Tool: []string{"Read", "Write"}}, "Read", true},
		{Rule{Tool: []string{"Read", "Write"}}, "Bash", false},
		{Rule{Tool: nil}, "Any", true},
	}

	for _, tt := range tests {
		result := tt.rule.MatchesTool(tt.toolName)
		if result != tt.expect {
			t.Errorf("MatchesTool(%q) = %v, want %v", tt.toolName, result, tt.expect)
		}
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		expect  bool
	}{
		{"*", "anything", true},
		{"Bash", "Bash", true},
		{"Bash*", "BashTool", true},
		{"Bash*", "Bash", true},
		{"*Tool", "BashTool", true},
		{"mcp__*", "mcp__server__tool", true},
		{"Bash", "Read", false},
	}

	for _, tt := range tests {
		result := matchesPattern(tt.pattern, tt.name)
		if result != tt.expect {
			t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.name, result, tt.expect)
		}
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.ruleSets == nil {
		t.Error("ruleSets should be initialized")
	}
}

func TestManagerAddRule(t *testing.T) {
	m := NewManager()

	rule := &Rule{
		Mode:   ModeAccept,
		Tool:   "Read",
		Reason: "Allow reading files",
	}

	m.AddRule(SourceUser, rule)

	rules := m.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].Source != SourceUser {
		t.Errorf("Rule source = %v, want %v", rules[0].Source, SourceUser)
	}

	if rules[0].ID == "" {
		t.Error("Rule ID should be set")
	}
}

func TestManagerRemoveRule(t *testing.T) {
	m := NewManager()

	rule := &Rule{
		Mode: ModeAccept,
		Tool: "Read",
	}
	m.AddRule(SourceUser, rule)

	removed := m.RemoveRule(rule.ID)
	if !removed {
		t.Error("RemoveRule should return true")
	}

	rules := m.GetRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after removal, got %d", len(rules))
	}
}

func TestManagerGetRulesForTool(t *testing.T) {
	m := NewManager()

	m.AddRule(SourceUser, &Rule{Mode: ModeAccept, Tool: "Read"})
	m.AddRule(SourceUser, &Rule{Mode: ModeDeny, Tool: "Bash"})
	m.AddRule(SourceUser, &Rule{Mode: ModeAccept, Tool: "*"})

	rules := m.GetRulesForTool("Read")
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules for Read, got %d", len(rules))
	}
}

func TestManagerCheck(t *testing.T) {
	m := NewManager()

	// Add a rule
	m.AddRule(SourceUser, &Rule{
		Mode:   ModeAccept,
		Tool:   "Read",
		Reason: "Reading is safe",
	})

	// Check permission
	result, err := m.Check(&PermissionRequest{
		ToolName: "Read",
	}, nil)

	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if !result.Allowed {
		t.Error("Read should be allowed")
	}

	if result.Decision == nil {
		t.Fatal("Decision should not be nil")
	}

	if result.Decision.Mode != ModeAccept {
		t.Errorf("Decision mode = %v, want %v", result.Decision.Mode, ModeAccept)
	}
}

func TestManagerCheckDeny(t *testing.T) {
	m := NewManager()

	m.AddRule(SourceUser, &Rule{
		Mode:   ModeDeny,
		Tool:   "Bash",
		Reason: "Bash is dangerous",
	})

	result, err := m.Check(&PermissionRequest{
		ToolName: "Bash",
	}, nil)

	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if !result.Denied {
		t.Error("Bash should be denied")
	}
}

func TestManagerCheckPriority(t *testing.T) {
	m := NewManager()

	// Add lower priority rule first
	m.AddRule(SourceUser, &Rule{
		Mode:   ModeDeny,
		Tool:   "Read",
		Reason: "Deny at user level",
	})

	// Add higher priority rule
	m.AddRule(SourceProject, &Rule{
		Mode:   ModeAccept,
		Tool:   "Read",
		Reason: "Allow at project level",
	})

	// Project should override user
	result, err := m.Check(&PermissionRequest{
		ToolName: "Read",
	}, nil)

	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if !result.Allowed {
		t.Error("Project rule should override user rule")
	}
}

func TestManagerDetectShadowedRules(t *testing.T) {
	m := NewManager()

	// Add a general rule with high priority
	m.AddRule(SourcePolicy, &Rule{
		Mode:   ModeAccept,
		Tool:   "*",
		Reason: "Allow all",
	})

	// Add a specific rule with lower priority
	m.AddRule(SourceUser, &Rule{
		Mode:   ModeAccept,
		Tool:   "Read",
		Reason: "Allow read",
	})

	shadowed := m.DetectShadowedRules()

	if len(shadowed) == 0 {
		t.Error("Should detect shadowed rules")
	}
}

func TestAutoClassifier(t *testing.T) {
	classifier := NewAutoClassifier()

	// Test safe tool
	result, err := classifier.Classify(&PermissionRequest{
		ToolName: "Read",
	}, nil)

	if err != nil {
		t.Fatalf("Classify() error: %v", err)
	}

	if result.Mode != ModeAccept {
		t.Errorf("Read should be accepted, got %v", result.Mode)
	}
}

func TestAutoClassifierDangerousCommand(t *testing.T) {
	classifier := NewAutoClassifier()

	result, err := classifier.Classify(&PermissionRequest{
		ToolName: "Bash",
		Input: map[string]interface{}{
			"command": "rm -rf /",
		},
	}, nil)

	if err != nil {
		t.Fatalf("Classify() error: %v", err)
	}

	if result.Mode != ModeDeny {
		t.Errorf("Dangerous command should be denied, got %v", result.Mode)
	}
}

func TestAutoClassifierReadOnlyCommand(t *testing.T) {
	classifier := NewAutoClassifier()

	result, err := classifier.Classify(&PermissionRequest{
		ToolName: "Bash",
		Input: map[string]interface{}{
			"command": "ls -la",
		},
	}, nil)

	if err != nil {
		t.Fatalf("Classify() error: %v", err)
	}

	if result.Mode != ModeAccept {
		t.Errorf("Read-only command should be accepted, got %v", result.Mode)
	}
}

func TestAutoClassifierTrustedContext(t *testing.T) {
	classifier := NewAutoClassifier()

	result, err := classifier.Classify(&PermissionRequest{
		ToolName: "Bash",
		Input: map[string]interface{}{
			"command": "npm test",
		},
	}, &PermissionContext{
		IsTrusted:  true,
		ProjectDir: "/home/user/project",
	})

	if err != nil {
		t.Fatalf("Classify() error: %v", err)
	}

	if result.Mode != ModeAccept {
		t.Errorf("Dev command in trusted dir should be accepted, got %v", result.Mode)
	}
}

func TestClearSessionRules(t *testing.T) {
	m := NewManager()

	m.AddRule(SourceSession, &Rule{
		Mode:       ModeAccept,
		Tool:       "Read",
		SessionOnly: true,
	})

	m.AddRule(SourceUser, &Rule{
		Mode:   ModeAccept,
		Tool:   "Write",
	})

	m.ClearSessionRules()

	rules := m.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after clearing session rules, got %d", len(rules))
	}
}

// Additional comprehensive tests

func TestManagerCheckWithInput(t *testing.T) {
	m := NewManager()

	m.AddRule(SourceUser, &Rule{
		Mode:          ModeDeny,
		Tool:          "Bash",
		InputPattern:  "rm -rf *",
		Reason:        "Deny dangerous rm commands",
	})

	// Test with matching input
	result, err := m.Check(&PermissionRequest{
		ToolName: "Bash",
		Input: map[string]interface{}{
			"command": "rm -rf /home/user/project",
		},
	}, nil)

	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if !result.Denied {
		t.Error("Dangerous command should be denied")
	}
}

func TestAutoClassifierEdgeCases(t *testing.T) {
	classifier := NewAutoClassifier()

	tests := []struct {
		name     string
		tool     string
		input    map[string]interface{}
		expected Mode
	}{
		{
			name:     "empty input",
			tool:     "Bash",
			input:    map[string]interface{}{},
			expected: ModeAsk,
		},
		{
			name:     "nil input",
			tool:     "Bash",
			input:    nil,
			expected: ModeAsk,
		},
		{
			name: "suspicious curl",
			tool: "Bash",
			input: map[string]interface{}{
				"command": "curl http://internal-server/admin",
			},
			expected: ModeAsk,
		},
		{
			name: "safe grep",
			tool: "Bash",
			input: map[string]interface{}{
				"command": "grep -r 'pattern' .",
			},
			expected: ModeAccept,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(&PermissionRequest{
				ToolName: tt.tool,
				Input:    tt.input,
			}, nil)

			if err != nil {
				t.Fatalf("Classify() error: %v", err)
			}

			if result.Mode != tt.expected {
				t.Errorf("Mode = %v, want %v", result.Mode, tt.expected)
			}
		})
	}
}

func TestManagerMultipleRulesForSameTool(t *testing.T) {
	m := NewManager()

	// Add multiple rules for the same tool from different sources
	m.AddRule(SourceAuto, &Rule{Mode: ModeAsk, Tool: "Bash"})
	m.AddRule(SourceUser, &Rule{Mode: ModeAccept, Tool: "Bash"})
	m.AddRule(SourceProject, &Rule{Mode: ModeDeny, Tool: "Bash"})
	m.AddRule(SourceSession, &Rule{Mode: ModeAccept, Tool: "Bash"})

	rules := m.GetRulesForTool("Bash")

	if len(rules) < 4 {
		t.Errorf("Expected at least 4 rules, got %d", len(rules))
	}

	// Test that highest priority wins
	result, _ := m.Check(&PermissionRequest{
		ToolName: "Bash",
	}, nil)

	// Session has highest priority and says Accept
	if !result.Allowed {
		t.Error("Session rule should win")
	}
}

func TestRuleWithMultipleTools(t *testing.T) {
	rule := &Rule{
		Mode: ModeAccept,
		Tool: []string{"Read", "Write", "Glob"},
	}

	if !rule.MatchesTool("Read") {
		t.Error("Should match Read")
	}

	if !rule.MatchesTool("Write") {
		t.Error("Should match Write")
	}

	if !rule.MatchesTool("Glob") {
		t.Error("Should match Glob")
	}

	if rule.MatchesTool("Bash") {
		t.Error("Should not match Bash")
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManager()
	done := make(chan bool, 10)

	// Concurrent rule additions
	for i := 0; i < 10; i++ {
		go func(idx int) {
			m.AddRule(SourceUser, &Rule{
				Mode: ModeAccept,
				Tool: "Tool" + string(rune('0'+idx)),
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent checks
	for i := 0; i < 10; i++ {
		go func(idx int) {
			m.Check(&PermissionRequest{
				ToolName: "Tool" + string(rune('0'+idx)),
			}, nil)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPermissionRequestValidation(t *testing.T) {
	m := NewManager()

	// Test with empty tool name
	result, err := m.Check(&PermissionRequest{
		ToolName: "",
	}, nil)

	if err == nil {
		t.Error("Should return error for empty tool name")
	}
	_ = result
}

func TestSourceString(t *testing.T) {
	sources := map[Source]string{
		SourceSession: "session",
		SourcePolicy:  "policy",
		SourceLocal:   "local",
		SourceProject: "project",
		SourceUser:    "user",
		SourceAuto:    "auto",
	}

	for source, expected := range sources {
		if string(source) != expected {
			t.Errorf("Source %v string = %q, want %q", source, string(source), expected)
		}
	}
}

func TestModeString(t *testing.T) {
	modes := map[Mode]string{
		ModeAsk:    "ask",
		ModeAccept: "accept",
		ModeDeny:   "deny",
		ModeAuto:   "auto",
	}

	for mode, expected := range modes {
		if string(mode) != expected {
			t.Errorf("Mode %v string = %q, want %q", mode, string(mode), expected)
		}
	}
}
