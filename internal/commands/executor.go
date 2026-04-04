package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Executor handles command execution
type Executor struct {
	registry *Registry
	version  string
}

// NewExecutor creates a new command executor
func NewExecutor(registry *Registry, version string) *Executor {
	return &Executor{
		registry: registry,
		version:  version,
	}
}

// Execute runs a command and returns the result
func (e *Executor) Execute(ctx context.Context, input string, sessionID, projectDir string) (*CommandOutput, error) {
	// Parse the command
	cmd, args, ok := e.registry.Parse(input)
	if !ok {
		return &CommandOutput{
			Output:  fmt.Sprintf("Unknown command: %s", input),
			IsError: true,
		}, nil
	}

	// Check if command is enabled
	if !cmd.IsEnabled {
		return &CommandOutput{
			Output:  fmt.Sprintf("Command /%s is disabled", cmd.Name),
			IsError: true,
		}, nil
	}

	// Build command input
	cmdInput := &CommandInput{
		Command:    cmd.Name,
		Args:       args,
		SessionID:  sessionID,
		ProjectDir: projectDir,
		Environment: make(map[string]string),
	}

	// Execute based on type
	switch cmd.Type {
	case TypePrompt:
		return e.executePromptCommand(cmd, cmdInput)

	case TypeLocal, TypeLocalJSX:
		if cmd.Handler == nil {
			return &CommandOutput{
				Output:  fmt.Sprintf("Command /%s has no handler", cmd.Name),
				IsError: true,
			}, nil
		}
		return cmd.Handler(ctx, cmdInput)

	default:
		return &CommandOutput{
			Output:  fmt.Sprintf("Unknown command type: %s", cmd.Type),
			IsError: true,
		}, nil
	}
}

// executePromptCommand handles prompt-type commands
func (e *Executor) executePromptCommand(cmd *Command, input *CommandInput) (*CommandOutput, error) {
	prompt := cmd.Prompt

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "$ARGS", input.Args)
	prompt = strings.ReplaceAll(prompt, "${ARGS}", input.Args)

	// If args provided, append them to the prompt
	if input.Args != "" {
		prompt = prompt + "\n\n" + input.Args
	}

	return &CommandOutput{
		Output: fmt.Sprintf("Executing: /%s", cmd.Name),
		Prompt:  prompt,
	}, nil
}

// GetRegistry returns the command registry
func (e *Executor) GetRegistry() *Registry {
	return e.registry
}

// Complete returns autocomplete suggestions
func (e *Executor) Complete(prefix string) []string {
	cmds := e.registry.Complete(prefix)
	var result []string
	for _, cmd := range cmds {
		result = append(result, "/"+cmd.Name)
	}
	return result
}

// IsCommand checks if input looks like a command
func IsCommand(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "/")
}

// ParseCommand extracts command name and args from input
func ParseCommand(input string) (name string, args string) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", ""
	}

	parts := strings.SplitN(input[1:], " ", 2)
	name = parts[0]
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return name, args
}

// FormatHelp formats help text for a command
func FormatHelp(cmd *Command) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("/%s", cmd.Name))
	if len(cmd.Aliases) > 0 {
		sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases, ", ")))
	}
	sb.WriteString("\n")

	if cmd.Parameters != "" {
		sb.WriteString(fmt.Sprintf("Usage: /%s %s\n", cmd.Name, cmd.Parameters))
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", cmd.Description))

	if cmd.LongDescription != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", cmd.LongDescription))
	}

	if len(cmd.Examples) > 0 {
		sb.WriteString("\nExamples:\n")
		for _, ex := range cmd.Examples {
			sb.WriteString(fmt.Sprintf("  %s\n", ex))
		}
	}

	return sb.String()
}

// Initialize initializes the command system with built-in commands
func Initialize(version string) (*Executor, error) {
	registry := NewRegistry()
	RegisterBuiltinCommands(registry, version)

	// Set exec.LookPath for doctor command
	execLookPath = exec.LookPath

	return NewExecutor(registry, version), nil
}
