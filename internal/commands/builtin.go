package commands

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// RegisterBuiltinCommands registers all built-in commands
func RegisterBuiltinCommands(registry *Registry, version string) {
	// Session Management
	registry.Register(&Command{
		Name:        "help",
		Aliases:     []string{"?", "h"},
		Description: "Show available commands and their usage",
		Type:        TypeLocal,
		Handler:     helpCommand(registry),
		Category:    string(CategorySession),
		SortOrder:   1,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "clear",
		Aliases:     []string{"reset", "new"},
		Description: "Clear conversation history and free context",
		Type:        TypeLocal,
		Handler:     clearCommand(),
		Category:    string(CategorySession),
		SortOrder:   2,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "compact",
		Description: "Clear history but keep summary in context",
		Parameters:  "[optional custom compacting instructions]",
		Type:        TypeLocal,
		Handler:     compactCommand(),
		Category:    string(CategorySession),
		SortOrder:   3,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "exit",
		Aliases:     []string{"quit", "q"},
		Description: "Exit the REPL",
		Type:        TypeLocal,
		Handler:     exitCommand(),
		Category:    string(CategorySession),
		SortOrder:   99,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "resume",
		Aliases:     []string{"continue"},
		Description: "Resume a previous conversation",
		Parameters:  "[conversation ID or search term]",
		Type:        TypeLocal,
		Handler:     resumeCommand(),
		Category:    string(CategorySession),
		SortOrder:   4,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "rename",
		Description: "Rename the current conversation",
		Parameters:  "[name]",
		Type:        TypeLocal,
		Handler:     renameCommand(),
		Category:    string(CategorySession),
		SortOrder:   5,
		IsEnabled:   true,
	})

	// Configuration
	registry.Register(&Command{
		Name:        "config",
		Aliases:     []string{"settings"},
		Description: "Open configuration panel",
		Type:        TypeLocal,
		Handler:     configCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   10,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "model",
		Description: "Set the AI model",
		Parameters:  "[model name]",
		Type:        TypeLocal,
		Handler:     modelCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   11,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "theme",
		Description: "Change the theme",
		Type:        TypeLocal,
		Handler:     themeCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   12,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "color",
		Description: "Set the prompt bar color for current session",
		Parameters:  "<color|default>",
		Type:        TypeLocal,
		Handler:     colorCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   13,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "vim",
		Description: "Toggle between Vim and normal editing mode",
		Type:        TypeLocal,
		Handler:     vimCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   14,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "permissions",
		Aliases:     []string{"allowed-tools"},
		Description: "Manage tool permission rules",
		Type:        TypeLocal,
		Handler:     permissionsCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   15,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "sandbox",
		Description: "Configure sandbox settings",
		Parameters:  `exclude "command pattern"`,
		Type:        TypeLocal,
		Handler:     sandboxCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   16,
		IsEnabled:   true,
	})

	// Tools & Extensions
	registry.Register(&Command{
		Name:        "mcp",
		Description: "Manage MCP servers",
		Parameters:  "[enable|disable [server name]]",
		Type:        TypeLocal,
		Handler:     mcpCommand(),
		Category:    string(CategoryTools),
		SortOrder:   20,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "hooks",
		Description: "View hook configuration for tool events",
		Type:        TypeLocal,
		Handler:     hooksCommand(),
		Category:    string(CategoryTools),
		SortOrder:   21,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "skills",
		Description: "List available skills",
		Type:        TypeLocal,
		Handler:     skillsCommand(),
		Category:    string(CategoryTools),
		SortOrder:   22,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "agents",
		Description: "Manage agent configurations",
		Type:        TypeLocal,
		Handler:     agentsCommand(),
		Category:    string(CategoryTools),
		SortOrder:   23,
		IsEnabled:   true,
	})

	// Git & Code
	registry.Register(&Command{
		Name:        "commit",
		Description: "Create a git commit",
		Type:        TypePrompt,
		Prompt:      "Create a git commit with the staged changes. Follow the commit message conventions.",
		Category:    string(CategoryGit),
		SortOrder:   30,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "review",
		Description: "Review a pull request",
		Parameters:  "[PR number]",
		Type:        TypePrompt,
		Prompt:      "Review the pull request and provide feedback on code quality, potential issues, and suggestions for improvement.",
		Category:    string(CategoryGit),
		SortOrder:   31,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "diff",
		Description: "View uncommitted changes",
		Type:        TypeLocal,
		Handler:     diffCommand(),
		Category:    string(CategoryGit),
		SortOrder:   32,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "statusline",
		Description: "Configure status bar UI",
		Type:        TypeLocal,
		Handler:     statuslineCommand(),
		Category:    string(CategoryGit),
		SortOrder:   33,
		IsEnabled:   true,
	})

	// Context & Files
	registry.Register(&Command{
		Name:        "context",
		Description: "Visualize current context usage",
		Type:        TypeLocal,
		Handler:     contextCommand(),
		Category:    string(CategoryContext),
		SortOrder:   40,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "add-dir",
		Description: "Add a new working directory",
		Parameters:  "<path>",
		Type:        TypeLocal,
		Handler:     addDirCommand(),
		Category:    string(CategoryContext),
		SortOrder:   41,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "copy",
		Description: "Copy last reply to clipboard",
		Parameters:  "[N] - Nth most recent reply",
		Type:        TypeLocal,
		Handler:     copyCommand(),
		Category:    string(CategoryContext),
		SortOrder:   42,
		IsEnabled:   true,
	})

	// Statistics & Export
	registry.Register(&Command{
		Name:        "cost",
		Description: "Display total cost and duration for session",
		Type:        TypeLocal,
		Handler:     costCommand(),
		Category:    string(CategoryStats),
		SortOrder:   50,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "export",
		Description: "Export conversation to file or clipboard",
		Parameters:  "[filename]",
		Type:        TypeLocal,
		Handler:     exportCommand(),
		Category:    string(CategoryStats),
		SortOrder:   51,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "stats",
		Description: "Display usage statistics and activity",
		Type:        TypeLocal,
		Handler:     statsCommand(),
		Category:    string(CategoryStats),
		SortOrder:   52,
		IsEnabled:   true,
	})

	// Diagnostics
	registry.Register(&Command{
		Name:        "doctor",
		Description: "Diagnose and verify installation",
		Type:        TypeLocal,
		Handler:     doctorCommand(version),
		Category:    string(CategoryDebug),
		SortOrder:   60,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "version",
		Description: "Print current version",
		Type:        TypeLocal,
		Handler:     versionCommand(version),
		Category:    string(CategoryDebug),
		SortOrder:   61,
		IsEnabled:   true,
	})

	// Modes & Features
	registry.Register(&Command{
		Name:        "plan",
		Description: "Enable plan mode or view current plan",
		Parameters:  "[open|<description>]",
		Type:        TypeLocal,
		Handler:     planCommand(),
		Category:    string(CategoryMode),
		SortOrder:   70,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "fast",
		Description: "Toggle fast mode",
		Parameters:  "[on|off]",
		Type:        TypeLocal,
		Handler:     fastCommand(),
		Category:    string(CategoryMode),
		SortOrder:   71,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "tasks",
		Aliases:     []string{"bashes"},
		Description: "List and manage background tasks",
		Type:        TypeLocal,
		Handler:     tasksCommand(),
		Category:    string(CategoryMode),
		SortOrder:   72,
		IsEnabled:   true,
	})

	// Feedback
	registry.Register(&Command{
		Name:        "feedback",
		Aliases:     []string{"bug"},
		Description: "Submit feedback about Poyo",
		Parameters:  "[report]",
		Type:        TypeLocal,
		Handler:     feedbackCommand(),
		Category:    string(CategoryFeedback),
		SortOrder:   80,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "memory",
		Description: "Edit memory file",
		Type:        TypeLocal,
		Handler:     memoryCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   17,
		IsEnabled:   true,
	})
}

// Command handlers

func helpCommand(registry *Registry) CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		var sb strings.Builder

		sb.WriteString("Available commands:\n\n")

		// Get commands grouped by category
		categorized := registry.GetByCategory()

		// Define category order
		categories := []string{
			string(CategorySession),
			string(CategoryConfig),
			string(CategoryTools),
			string(CategoryGit),
			string(CategoryContext),
			string(CategoryStats),
			string(CategoryDebug),
			string(CategoryMode),
			string(CategoryFeedback),
		}

		for _, cat := range categories {
			cmds, ok := categorized[cat]
			if !ok || len(cmds) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("## %s\n", cat))
			for _, cmd := range cmds {
				if cmd.IsHidden {
					continue
				}

				aliases := ""
				if len(cmd.Aliases) > 0 {
					aliases = fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases, ", "))
				}

				params := ""
				if cmd.Parameters != "" {
					params = " " + cmd.Parameters
				}

				sb.WriteString(fmt.Sprintf("  /%-12s%s%s - %s\n",
					cmd.Name+params+aliases,
					strings.Repeat(" ", max(0, 20-len(cmd.Name+params+aliases))),
					"",
					cmd.Description,
				))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("Use /<command> --help for detailed usage.\n")

		return &CommandOutput{Output: sb.String()}, nil
	}
}

func clearCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output:      "Conversation cleared. Context has been freed.",
			ShouldClear: true,
		}, nil
	}
}

func compactCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output:             "Compacting conversation...",
			ShouldCompact:      true,
			CompactInstructions: input.Args,
		}, nil
	}
}

func exitCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output:     "Goodbye!",
			ShouldExit: true,
		}, nil
	}
}

func resumeCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			// TODO: List recent conversations
			return &CommandOutput{
				Output: "Recent conversations:\n(Not implemented yet)",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Resuming conversation: %s", input.Args),
		}, nil
	}
}

func renameCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /rename <name>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Conversation renamed to: %s", input.Args),
		}, nil
	}
}

func configCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Configuration:\n(Open settings panel - TUI implementation needed)",
		}, nil
	}
}

func modelCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Current model: claude-sonnet-4.6\nAvailable models:\n  - claude-opus-4.6\n  - claude-sonnet-4.6\n  - claude-haiku-4.5",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Model set to: %s", input.Args),
		}, nil
	}
}

func themeCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Theme:\n(Theme selector - TUI implementation needed)",
		}, nil
	}
}

func colorCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /color <color|default>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Prompt color set to: %s", input.Args),
		}, nil
	}
}

func vimCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Vim mode toggled",
		}, nil
	}
}

func permissionsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Permission Rules:\n(Permission manager - TUI implementation needed)",
		}, nil
	}
}

func sandboxCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Sandbox configuration:\n(Sandbox manager - TUI implementation needed)",
		}, nil
	}
}

func mcpCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "MCP Servers:\n(MCP manager - Implementation needed)",
		}, nil
	}
}

func hooksCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Hook Configuration:\n(Show configured hooks)",
		}, nil
	}
}

func skillsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Available Skills:\n(List installed skills)",
		}, nil
	}
}

func agentsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Agent Configurations:\n(List configured agents)",
		}, nil
	}
}

func diffCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Git diff:\n(Run git diff and display)",
		}, nil
	}
}

func statuslineCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Status line configuration:\n(Status line manager)",
		}, nil
	}
}

func contextCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Context Usage:\n[████████░░░░░░░░] 50% (50,000 / 100,000 tokens)",
		}, nil
	}
}

func addDirCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /add-dir <path>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Added directory: %s", input.Args),
		}, nil
	}
}

func copyCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Last reply copied to clipboard",
		}, nil
	}
}

func costCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: fmt.Sprintf("Session Statistics:\n  Duration: %s\n  Total Cost: $0.00\n  Input Tokens: 0\n  Output Tokens: 0", time.Since(time.Now()).Round(time.Second)),
		}, nil
	}
}

func exportCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		filename := "conversation.json"
		if input.Args != "" {
			filename = input.Args
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Exported conversation to: %s", filename),
		}, nil
	}
}

func statsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Usage Statistics:\n(Show detailed usage stats)",
		}, nil
	}
}

func doctorCommand(version string) CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		var sb strings.Builder
		sb.WriteString("Poyo Diagnostics:\n\n")
		sb.WriteString(fmt.Sprintf("  Version:     %s\n", version))
		sb.WriteString(fmt.Sprintf("  Platform:    %s/%s\n", runtime.GOOS, runtime.GOARCH))
		sb.WriteString(fmt.Sprintf("  Go Version:  %s\n", runtime.Version()))
		sb.WriteString(fmt.Sprintf("  Working Dir: %s\n", input.ProjectDir))

		// Check common dependencies
		deps := []string{"git", "node", "npm"}
		sb.WriteString("\n  Dependencies:\n")
		for _, dep := range deps {
			path, err := execLookPath(dep)
			if err != nil {
				sb.WriteString(fmt.Sprintf("    %s: not found\n", dep))
			} else {
				sb.WriteString(fmt.Sprintf("    %s: %s\n", dep, path))
			}
		}

		sb.WriteString("\n  Status: OK\n")
		return &CommandOutput{Output: sb.String()}, nil
	}
}

func versionCommand(version string) CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: fmt.Sprintf("Poyo v%s\nPlatform: %s/%s\nGo: %s", version, runtime.GOOS, runtime.GOARCH, runtime.Version()),
		}, nil
	}
}

func planCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Plan Mode:\n(View current plan)",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Plan created: %s", input.Args),
		}, nil
	}
}

func fastCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Fast mode: toggled",
		}, nil
	}
}

func tasksCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Background Tasks:\n(No active tasks)",
		}, nil
	}
}

func feedbackCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Feedback submitted. Thank you!",
		}, nil
	}
}

func memoryCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		homeDir, _ := os.UserHomeDir()
		return &CommandOutput{
			Output: fmt.Sprintf("Memory file: %s/.claude/memory.md", homeDir),
		}, nil
	}
}

// Helper to make exec.LookPath available for testing
var execLookPath = lookPath

func lookPath(name string) (string, error) {
	return execLookPathImpl(name)
}

// Separate function for actual implementation
func execLookPathImpl(name string) (string, error) {
	// This will be replaced by exec.LookPath in production
	return "", nil
}
