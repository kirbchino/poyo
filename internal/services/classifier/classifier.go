// Package classifier provides AI-based classification for permissions
package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// OperationType represents the type of operation being classified
type OperationType string

const (
	OperationTypeFileRead    OperationType = "file_read"
	OperationTypeFileWrite   OperationType = "file_write"
	OperationTypeFileDelete  OperationType = "file_delete"
	OperationTypeCommand     OperationType = "command"
	OperationTypeNetwork     OperationType = "network"
	OperationTypeCode        OperationType = "code"
	OperationTypeConfig      OperationType = "config"
)

// RiskLevel represents the risk level of an operation
type RiskLevel string

const (
	RiskLevelSafe     RiskLevel = "safe"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// ClassificationResult contains the result of classification
type ClassificationResult struct {
	OperationType  OperationType `json:"operation_type"`
	RiskLevel      RiskLevel     `json:"risk_level"`
	ShouldAllow    bool          `json:"should_allow"`
	ShouldAsk      bool          `json:"should_ask"`
	Confidence     float64       `json:"confidence"`
	Reasoning      string        `json:"reasoning"`
	Flags          []string      `json:"flags,omitempty"`
}

// Classifier classifies operations for permission decisions
type Classifier struct {
	apiClient APIClient
	config    ClassifierConfig
}

// ClassifierConfig contains configuration for the classifier
type ClassifierConfig struct {
	Enabled         bool
	Model           string
	MaxTokens       int
	Temperature     float64
	CacheEnabled    bool
	ConfidenceThreshold float64
}

// DefaultClassifierConfig returns default classifier configuration
func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		Enabled:             true,
		Model:               "claude-haiku-4-5",
		MaxTokens:           500,
		Temperature:         0.1,
		CacheEnabled:        true,
		ConfidenceThreshold: 0.8,
	}
}

// APIClient interface for making API calls
type APIClient interface {
	CreateChatCompletion(ctx context.Context, messages []Message) (string, error)
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewClassifier creates a new classifier
func NewClassifier(apiClient APIClient, config ClassifierConfig) *Classifier {
	return &Classifier{
		apiClient: apiClient,
		config:    config,
	}
}

// ClassifyToolUse classifies a tool use operation
func (c *Classifier) ClassifyToolUse(ctx context.Context, toolName string, input map[string]interface{}) (*ClassificationResult, error) {
	// Quick rule-based classification for common cases
	if result := c.quickClassify(toolName, input); result != nil {
		return result, nil
	}

	// Use AI for complex cases
	if c.config.Enabled && c.apiClient != nil {
		return c.aiClassify(ctx, toolName, input)
	}

	// Default: ask for permission
	return &ClassificationResult{
		OperationType: c.detectOperationType(toolName),
		RiskLevel:     RiskLevelMedium,
		ShouldAllow:   false,
		ShouldAsk:     true,
		Confidence:    0.5,
		Reasoning:     "Unable to classify - requiring user confirmation",
	}, nil
}

// quickClassify performs rule-based quick classification
func (c *Classifier) quickClassify(toolName string, input map[string]interface{}) *ClassificationResult {
	switch toolName {
	case "Read", "Glob", "Grep":
		return &ClassificationResult{
			OperationType: OperationTypeFileRead,
			RiskLevel:     RiskLevelLow,
			ShouldAllow:   true,
			ShouldAsk:     false,
			Confidence:    0.95,
			Reasoning:     "Read-only file operation",
		}

	case "Write", "Edit":
		path, _ := input["file_path"].(string)
		if c.isSensitivePath(path) {
			return &ClassificationResult{
				OperationType: OperationTypeFileWrite,
				RiskLevel:     RiskLevelHigh,
				ShouldAllow:   false,
				ShouldAsk:     true,
				Confidence:    0.95,
				Reasoning:     "Writing to sensitive file",
				Flags:         []string{"sensitive_path"},
			}
		}
		return &ClassificationResult{
			OperationType: OperationTypeFileWrite,
			RiskLevel:     RiskLevelMedium,
			ShouldAllow:   false,
			ShouldAsk:     true,
			Confidence:    0.8,
			Reasoning:     "File write operation requires confirmation",
		}

	case "Bash":
		cmd, _ := input["command"].(string)
		return c.classifyCommand(cmd)

	case "WebFetch", "WebSearch":
		return &ClassificationResult{
			OperationType: OperationTypeNetwork,
			RiskLevel:     RiskLevelLow,
			ShouldAllow:   true,
			ShouldAsk:     false,
			Confidence:    0.9,
			Reasoning:     "Network fetch operation",
		}

	case "TodoWrite":
		return &ClassificationResult{
			OperationType: OperationTypeCode,
			RiskLevel:     RiskLevelSafe,
			ShouldAllow:   true,
			ShouldAsk:     false,
			Confidence:    0.99,
			Reasoning:     "Task management operation",
		}
	}

	return nil
}

// classifyCommand classifies a shell command
func (c *Classifier) classifyCommand(cmd string) *ClassificationResult {
	cmdLower := strings.ToLower(cmd)

	// Critical patterns
	criticalPatterns := []string{
		"rm -rf", "rm -r", "rm -f",
		"mkfs", "dd if=",
		"chmod 777", "chmod -R 777",
		"curl | bash", "wget | bash",
		"> /dev/", "dd of=/dev/",
	}
	for _, pattern := range criticalPatterns {
		if strings.Contains(cmdLower, pattern) {
			return &ClassificationResult{
				OperationType: OperationTypeCommand,
				RiskLevel:     RiskLevelCritical,
				ShouldAllow:   false,
				ShouldAsk:     true,
				Confidence:    0.99,
				Reasoning:     "Potentially destructive command detected",
				Flags:         []string{"destructive_command"},
			}
		}
	}

	// High risk patterns
	highRiskPatterns := []string{
		"sudo", "su ", "chmod", "chown",
		"git push --force", "git reset --hard",
		"npm publish", "pip upload",
	}
	for _, pattern := range highRiskPatterns {
		if strings.Contains(cmdLower, pattern) {
			return &ClassificationResult{
				OperationType: OperationTypeCommand,
				RiskLevel:     RiskLevelHigh,
				ShouldAllow:   false,
				ShouldAsk:     true,
				Confidence:    0.95,
				Reasoning:     "High-risk command detected",
				Flags:         []string{"elevated_privileges"},
			}
		}
	}

	// Safe read-only commands
	safePatterns := []string{
		"ls", "cat ", "head ", "tail ",
		"grep ", "find ", "git status",
		"git log", "git diff", "git branch",
		"which ", "echo ", "pwd",
	}
	for _, pattern := range safePatterns {
		if strings.HasPrefix(cmdLower, pattern) {
			return &ClassificationResult{
				OperationType: OperationTypeCommand,
				RiskLevel:     RiskLevelLow,
				ShouldAllow:   true,
				ShouldAsk:     false,
				Confidence:    0.9,
				Reasoning:     "Read-only command",
			}
		}
	}

	// Default for commands
	return &ClassificationResult{
		OperationType: OperationTypeCommand,
		RiskLevel:     RiskLevelMedium,
		ShouldAllow:   false,
		ShouldAsk:     true,
		Confidence:    0.7,
		Reasoning:     "Command requires review",
	}
}

// isSensitivePath checks if a path is sensitive
func (c *Classifier) isSensitivePath(path string) bool {
	sensitivePatterns := []string{
		".env", ".env.", "credentials", "secrets",
		".pem", ".key", ".ssh",
		"id_rsa", "id_ed25519",
		".gitconfig", ".npmrc",
		"config.local", "settings.local",
	}
	pathLower := strings.ToLower(path)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}
	return false
}

// detectOperationType detects the operation type from tool name
func (c *Classifier) detectOperationType(toolName string) OperationType {
	switch toolName {
	case "Read", "Glob", "Grep":
		return OperationTypeFileRead
	case "Write", "Edit":
		return OperationTypeFileWrite
	case "Bash":
		return OperationTypeCommand
	case "WebFetch", "WebSearch":
		return OperationTypeNetwork
	default:
		return OperationTypeCode
	}
}

// aiClassify uses AI to classify the operation
func (c *Classifier) aiClassify(ctx context.Context, toolName string, input map[string]interface{}) (*ClassificationResult, error) {
	prompt := c.buildClassificationPrompt(toolName, input)

	messages := []Message{
		{Role: "system", Content: c.getSystemPrompt()},
		{Role: "user", Content: prompt},
	}

	response, err := c.apiClient.CreateChatCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI classification failed: %w", err)
	}

	return c.parseClassificationResponse(response)
}

// getSystemPrompt returns the system prompt for classification
func (c *Classifier) getSystemPrompt() string {
	return `You are a security classifier. Analyze operations and return JSON with:
{
  "operation_type": "file_read|file_write|file_delete|command|network|code|config",
  "risk_level": "safe|low|medium|high|critical",
  "should_allow": true|false,
  "should_ask": true|false,
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation",
  "flags": ["optional", "risk", "flags"]
}

Be conservative. When uncertain, set should_ask=true.`
}

// buildClassificationPrompt builds the classification prompt
func (c *Classifier) buildClassificationPrompt(toolName string, input map[string]interface{}) string {
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	return fmt.Sprintf("Classify this operation:\nTool: %s\nInput:\n%s", toolName, string(inputJSON))
}

// parseClassificationResponse parses the AI response
func (c *Classifier) parseClassificationResponse(response string) (*ClassificationResult, error) {
	// Extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result ClassificationResult
	if err := json.Unmarshal([]byte(response[start:end+1]), &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

// BatchClassifierResult contains results for multiple classifications
type BatchClassifierResult struct {
	Results []*ClassificationResult
	Errors  []error
}

// ClassifyBatch classifies multiple operations
func (c *Classifier) ClassifyBatch(ctx context.Context, operations []struct {
	ToolName string
	Input    map[string]interface{}
}) *BatchClassifierResult {
	result := &BatchClassifierResult{
		Results: make([]*ClassificationResult, len(operations)),
		Errors:  make([]error, len(operations)),
	}

	for i, op := range operations {
		classification, err := c.ClassifyToolUse(ctx, op.ToolName, op.Input)
		result.Results[i] = classification
		result.Errors[i] = err
	}

	return result
}
