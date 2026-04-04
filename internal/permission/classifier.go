package permission

import (
	"path/filepath"
	"strings"
)

// AutoClassifier implements automatic permission classification
type AutoClassifier struct {
	// SafeTools are tools that are always considered safe
	SafeTools map[string]bool

	// DangerousPatterns are patterns that indicate danger
	DangerousPatterns []string

	// ReadOnlyTools are tools that only read data
	ReadOnlyTools map[string]bool
}

// NewAutoClassifier creates a new auto classifier
func NewAutoClassifier() *AutoClassifier {
	return &AutoClassifier{
		SafeTools: map[string]bool{
			"Read":       true,
			"Glob":       true,
			"Grep":       true,
			"LSP":        true,
			"TaskOutput": true,
		},
		ReadOnlyTools: map[string]bool{
			"Read":           true,
			"Glob":           true,
			"Grep":           true,
			"LSP":            true,
			"TaskOutput":     true,
			"WebSearch":      true,
			"WebFetch":       true,
			"ListMcpResources": true,
		},
		DangerousPatterns: []string{
			"rm -rf",
			"sudo",
			"chmod 777",
			"mkfs",
			"dd if=",
			"> /dev/",
			"curl | bash",
			"wget | sh",
			"eval",
			"exec",
		},
	}
}

// Classify classifies a permission request
func (c *AutoClassifier) Classify(req *PermissionRequest, ctx *PermissionContext) (*ClassificationResult, error) {
	features := make(map[string]interface{})

	// Check if tool is safe
	if c.SafeTools[req.ToolName] {
		return &ClassificationResult{
			Mode:       ModeAccept,
			Confidence: 0.95,
			Reason:     "Tool is classified as safe (read-only operation)",
			Features:   features,
		}, nil
	}

	// Check if tool is read-only
	features["isReadOnly"] = c.ReadOnlyTools[req.ToolName]

	// Analyze tool input based on tool type
	switch req.ToolName {
	case "Bash":
		return c.classifyBash(req, ctx, features)

	case "Edit":
		return c.classifyEdit(req, ctx, features)

	case "Write":
		return c.classifyWrite(req, ctx, features)

	case "NotebookEdit":
		return c.classifyNotebookEdit(req, ctx, features)

	default:
		// Unknown tool, ask user
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.5,
			Reason:     "Unknown tool, requires user confirmation",
			Features:   features,
		}, nil
	}
}

// classifyBash classifies a Bash command
func (c *AutoClassifier) classifyBash(req *PermissionRequest, ctx *PermissionContext, features map[string]interface{}) (*ClassificationResult, error) {
	cmd, ok := req.Input["command"].(string)
	if !ok {
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.5,
			Reason:     "No command provided",
			Features:   features,
		}, nil
	}

	features["command"] = cmd

	// Check for dangerous patterns
	for _, pattern := range c.DangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return &ClassificationResult{
				Mode:       ModeDeny,
				Confidence: 0.9,
				Reason:     "Command contains dangerous pattern: " + pattern,
				Features:   features,
			}, nil
		}
	}

	// Check if read-only command
	readOnlyCommands := []string{
		"ls", "cat", "head", "tail", "less", "more",
		"find", "grep", "awk", "sed -n", "wc",
		"git status", "git log", "git diff", "git show", "git branch",
		"which", "whereis", "type",
		"echo", "printf", "pwd",
		"env", "printenv",
	}

	for _, roCmd := range readOnlyCommands {
		if strings.HasPrefix(cmd, roCmd+" ") || cmd == roCmd {
			features["isReadOnly"] = true
			return &ClassificationResult{
				Mode:       ModeAccept,
				Confidence: 0.85,
				Reason:     "Read-only command",
				Features:   features,
			}, nil
		}
	}

	// Check if in trusted directory
	if ctx != nil && ctx.IsTrusted {
		features["isTrusted"] = true

		// Allow common development commands in trusted directories
		devCommands := []string{
			"npm", "yarn", "pnpm", "bun",
			"go ", "cargo ", "pip ", "python ", "node ",
			"make", "cmake",
			"git ", "gh ",
		}

		for _, devCmd := range devCommands {
			if strings.HasPrefix(cmd, devCmd) {
				return &ClassificationResult{
					Mode:       ModeAccept,
					Confidence: 0.75,
					Reason:     "Development command in trusted directory",
					Features:   features,
				}, nil
			}
		}
	}

	// Check for package installations
	if strings.Contains(cmd, "npm install") ||
		strings.Contains(cmd, "yarn add") ||
		strings.Contains(cmd, "pip install") ||
		strings.Contains(cmd, "go get") ||
		strings.Contains(cmd, "cargo add") {
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.7,
			Reason:     "Package installation requires confirmation",
			Features:   features,
		}, nil
	}

	// Default: ask user
	return &ClassificationResult{
		Mode:       ModeAsk,
		Confidence: 0.6,
		Reason:     "Command requires user confirmation",
		Features:   features,
	}, nil
}

// classifyEdit classifies an Edit operation
func (c *AutoClassifier) classifyEdit(req *PermissionRequest, ctx *PermissionContext, features map[string]interface{}) (*ClassificationResult, error) {
	filePath, ok := req.Input["file_path"].(string)
	if !ok {
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.5,
			Reason:     "No file path provided",
			Features:   features,
		}, nil
	}

	features["filePath"] = filePath

	// Check if editing sensitive files
	sensitiveFiles := []string{
		".env", ".env.local", ".env.production",
		"credentials", "secrets", "password",
		"id_rsa", "id_ed25519", ".pem", ".key",
		".gitignore", ".dockerignore",
	}

	for _, sf := range sensitiveFiles {
		if strings.Contains(filePath, sf) {
			return &ClassificationResult{
				Mode:       ModeAsk,
				Confidence: 0.9,
				Reason:     "Editing potentially sensitive file",
				Features:   features,
			}, nil
		}
	}

	// Check if in trusted directory
	if ctx != nil && ctx.IsTrusted {
		// Check if file is within project directory
		if ctx.ProjectDir != "" {
			absPath, err := filepath.Abs(filePath)
			if err == nil && strings.HasPrefix(absPath, ctx.ProjectDir) {
				return &ClassificationResult{
					Mode:       ModeAccept,
					Confidence: 0.8,
					Reason:     "Editing file in trusted project directory",
					Features:   features,
				}, nil
			}
		}
	}

	// Default: ask user
	return &ClassificationResult{
		Mode:       ModeAsk,
		Confidence: 0.6,
		Reason:     "File edit requires confirmation",
		Features:   features,
	}, nil
}

// classifyWrite classifies a Write operation
func (c *AutoClassifier) classifyWrite(req *PermissionRequest, ctx *PermissionContext, features map[string]interface{}) (*ClassificationResult, error) {
	filePath, ok := req.Input["file_path"].(string)
	if !ok {
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.5,
			Reason:     "No file path provided",
			Features:   features,
		}, nil
	}

	features["filePath"] = filePath

	// Check if writing to sensitive locations
	sensitivePaths := []string{
		"/etc/", "/usr/", "/bin/", "/sbin/",
		"~/.ssh/", "~/.gnupg/",
		"/root/",
	}

	for _, sp := range sensitivePaths {
		if strings.HasPrefix(filePath, sp) || strings.Contains(filePath, sp) {
			return &ClassificationResult{
				Mode:       ModeDeny,
				Confidence: 0.95,
				Reason:     "Writing to sensitive system location",
				Features:   features,
			}, nil
		}
	}

	// Check if in trusted directory
	if ctx != nil && ctx.IsTrusted {
		if ctx.ProjectDir != "" {
			absPath, err := filepath.Abs(filePath)
			if err == nil && strings.HasPrefix(absPath, ctx.ProjectDir) {
				return &ClassificationResult{
					Mode:       ModeAccept,
					Confidence: 0.75,
					Reason:     "Writing to trusted project directory",
					Features:   features,
				}, nil
			}
		}
	}

	// Default: ask user
	return &ClassificationResult{
		Mode:       ModeAsk,
		Confidence: 0.6,
		Reason:     "File write requires confirmation",
		Features:   features,
	}, nil
}

// classifyNotebookEdit classifies a NotebookEdit operation
func (c *AutoClassifier) classifyNotebookEdit(req *PermissionRequest, ctx *PermissionContext, features map[string]interface{}) (*ClassificationResult, error) {
	notebookPath, ok := req.Input["notebook_path"].(string)
	if !ok {
		return &ClassificationResult{
			Mode:       ModeAsk,
			Confidence: 0.5,
			Reason:     "No notebook path provided",
			Features:   features,
		}, nil
	}

	features["notebookPath"] = notebookPath

	// Check if in trusted directory
	if ctx != nil && ctx.IsTrusted {
		if ctx.ProjectDir != "" {
			absPath, err := filepath.Abs(notebookPath)
			if err == nil && strings.HasPrefix(absPath, ctx.ProjectDir) {
				return &ClassificationResult{
					Mode:       ModeAccept,
					Confidence: 0.8,
					Reason:     "Editing notebook in trusted project directory",
					Features:   features,
				}, nil
			}
		}
	}

	// Default: ask user
	return &ClassificationResult{
		Mode:       ModeAsk,
		Confidence: 0.6,
		Reason:     "Notebook edit requires confirmation",
		Features:   features,
	}, nil
}
