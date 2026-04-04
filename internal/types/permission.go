// Package types contains core type definitions for the Poyo implementation.
package types

// PermissionMode represents the current permission mode.
type PermissionMode string

const (
	// PermissionModeDefault prompts for permission on sensitive operations.
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits auto-accepts file edits.
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModeBypassPermissions auto-accepts all operations (dangerous).
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	// PermissionModeDontAsk remembers decisions and doesn't ask again.
	PermissionModeDontAsk PermissionMode = "dontAsk"
	// PermissionModePlan enters planning mode.
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeAuto uses classifier for auto-approval.
	PermissionModeAuto PermissionMode = "auto"
)

// PermissionBehavior represents the decision for a permission request.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionRuleSource represents where a permission rule originated.
type PermissionRuleSource string

const (
	PermissionSourceUserSettings    PermissionRuleSource = "userSettings"
	PermissionSourceProjectSettings PermissionRuleSource = "projectSettings"
	PermissionSourceLocalSettings   PermissionRuleSource = "localSettings"
	PermissionSourceFlagSettings    PermissionRuleSource = "flagSettings"
	PermissionSourcePolicySettings  PermissionRuleSource = "policySettings"
	PermissionSourceCLIArg          PermissionRuleSource = "cliArg"
	PermissionSourceCommand         PermissionRuleSource = "command"
	PermissionSourceSession         PermissionRuleSource = "session"
)

// PermissionRuleValue specifies which tool and optional content.
type PermissionRuleValue struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
}

// PermissionRule represents a permission rule with source and behavior.
type PermissionRule struct {
	Source        PermissionRuleSource `json:"source"`
	RuleBehavior  PermissionBehavior   `json:"ruleBehavior"`
	RuleValue     PermissionRuleValue  `json:"ruleValue"`
}

// PermissionUpdateDestination represents where a permission update should be persisted.
type PermissionUpdateDestination string

const (
	PermissionDestUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionDestProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionDestLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionDestSession         PermissionUpdateDestination = "session"
	PermissionDestCLIArg          PermissionUpdateDestination = "cliArg"
)

// PermissionUpdate represents an update operation for permission configuration.
type PermissionUpdate struct {
	Type        PermissionUpdateDestination `json:"type"`
	Destination PermissionUpdateDestination `json:"destination"`
	Rules       []PermissionRuleValue       `json:"rules,omitempty"`
	Behavior    PermissionBehavior          `json:"behavior,omitempty"`
	Mode        PermissionMode              `json:"mode,omitempty"`
	Directories []string                    `json:"directories,omitempty"`
}

// AdditionalWorkingDirectory represents an additional directory in permission scope.
type AdditionalWorkingDirectory struct {
	Path   string               `json:"path"`
	Source PermissionRuleSource `json:"source"`
}

// PermissionDecisionReason explains why a permission decision was made.
type PermissionDecisionReason struct {
	Type       string           `json:"type"`
	Rule       *PermissionRule  `json:"rule,omitempty"`
	Mode       PermissionMode   `json:"mode,omitempty"`
	Reason     string           `json:"reason,omitempty"`
	HookName   string           `json:"hookName,omitempty"`
	HookSource string           `json:"hookSource,omitempty"`
}

// PermissionAllowDecision represents a granted permission.
type PermissionAllowDecision struct {
	Behavior       PermissionBehavior      `json:"behavior"`
	UpdatedInput   map[string]interface{}  `json:"updatedInput,omitempty"`
	UserModified   bool                    `json:"userModified,omitempty"`
	DecisionReason *PermissionDecisionReason `json:"decisionReason,omitempty"`
	ToolUseID      string                  `json:"toolUseId,omitempty"`
	AcceptFeedback string                  `json:"acceptFeedback,omitempty"`
}

// PermissionAskDecision represents a request to prompt the user.
type PermissionAskDecision struct {
	Behavior       PermissionBehavior      `json:"behavior"`
	Message        string                  `json:"message"`
	UpdatedInput   map[string]interface{}  `json:"updatedInput,omitempty"`
	DecisionReason *PermissionDecisionReason `json:"decisionReason,omitempty"`
	Suggestions    []PermissionUpdate      `json:"suggestions,omitempty"`
	BlockedPath    string                  `json:"blockedPath,omitempty"`
}

// PermissionDenyDecision represents a denied permission.
type PermissionDenyDecision struct {
	Behavior       PermissionBehavior      `json:"behavior"`
	Message        string                  `json:"message"`
	DecisionReason *PermissionDecisionReason `json:"decisionReason"`
	ToolUseID      string                  `json:"toolUseId,omitempty"`
}

// PermissionDecision represents any permission decision.
type PermissionDecision interface {
	GetBehavior() PermissionBehavior
}

func (d *PermissionAllowDecision) GetBehavior() PermissionBehavior { return d.Behavior }
func (d *PermissionAskDecision) GetBehavior() PermissionBehavior  { return d.Behavior }
func (d *PermissionDenyDecision) GetBehavior() PermissionBehavior { return d.Behavior }

// PermissionResult represents the result of a permission check.
type PermissionResult struct {
	Behavior       PermissionBehavior      `json:"behavior"`
	Message        string                  `json:"message,omitempty"`
	UpdatedInput   map[string]interface{}  `json:"updatedInput,omitempty"`
	DecisionReason *PermissionDecisionReason `json:"decisionReason,omitempty"`
	Suggestions    []PermissionUpdate      `json:"suggestions,omitempty"`
}

// ToolPermissionRulesBySource maps permission rules by their source.
type ToolPermissionRulesBySource map[PermissionRuleSource][]string

// ToolPermissionContext provides context for permission checking in tools.
type ToolPermissionContext struct {
	Mode                          PermissionMode                 `json:"mode"`
	AdditionalWorkingDirectories  map[string]AdditionalWorkingDirectory `json:"additionalWorkingDirectories"`
	AlwaysAllowRules              ToolPermissionRulesBySource    `json:"alwaysAllowRules"`
	AlwaysDenyRules               ToolPermissionRulesBySource    `json:"alwaysDenyRules"`
	AlwaysAskRules                ToolPermissionRulesBySource    `json:"alwaysAskRules"`
	IsBypassPermissionsModeAvailable bool                          `json:"isBypassPermissionsModeAvailable"`
	StrippedDangerousRules        ToolPermissionRulesBySource    `json:"strippedDangerousRules,omitempty"`
	ShouldAvoidPermissionPrompts  bool                           `json:"shouldAvoidPermissionPrompts,omitempty"`
	AwaitAutomatedChecksBeforeDialog bool                          `json:"awaitAutomatedChecksBeforeDialog,omitempty"`
	PrePlanMode                   PermissionMode                 `json:"prePlanMode,omitempty"`
}

// NewEmptyToolPermissionContext creates a minimal permission context.
func NewEmptyToolPermissionContext() *ToolPermissionContext {
	return &ToolPermissionContext{
		Mode:                         PermissionModeDefault,
		AdditionalWorkingDirectories: make(map[string]AdditionalWorkingDirectory),
		AlwaysAllowRules:             make(ToolPermissionRulesBySource),
		AlwaysDenyRules:              make(ToolPermissionRulesBySource),
		AlwaysAskRules:               make(ToolPermissionRulesBySource),
		IsBypassPermissionsModeAvailable: false,
	}
}

// ClassifierResult represents the result of a bash command classifier.
type ClassifierResult struct {
	Matches          bool   `json:"matches"`
	MatchedDescription string `json:"matchedDescription,omitempty"`
	Confidence       string `json:"confidence"` // high, medium, low
	Reason           string `json:"reason"`
}

// YoloClassifierResult represents the result of the auto-mode classifier.
type YoloClassifierResult struct {
	Thinking         string           `json:"thinking,omitempty"`
	ShouldBlock      bool             `json:"shouldBlock"`
	Reason           string           `json:"reason"`
	Unavailable      bool             `json:"unavailable,omitempty"`
	TranscriptTooLong bool            `json:"transcriptTooLong,omitempty"`
	Model            string           `json:"model"`
	Usage            *ClassifierUsage `json:"usage,omitempty"`
	DurationMs       int64            `json:"durationMs,omitempty"`
}

// ClassifierUsage represents token usage from classifier API calls.
type ClassifierUsage struct {
	InputTokens           int64 `json:"inputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	CacheReadInputTokens  int64 `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int64 `json:"cacheCreationInputTokens"`
}

// RiskLevel represents the risk level of a permission request.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "LOW"
	RiskLevelMedium RiskLevel = "MEDIUM"
	RiskLevelHigh   RiskLevel = "HIGH"
)

// PermissionExplanation explains a permission request to the user.
type PermissionExplanation struct {
	RiskLevel   RiskLevel `json:"riskLevel"`
	Explanation string    `json:"explanation"`
	Reasoning   string    `json:"reasoning"`
	Risk        string    `json:"risk"`
}
