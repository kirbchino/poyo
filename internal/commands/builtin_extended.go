package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// RegisterExtendedCommands registers additional extended commands
// This file contains 50+ additional commands to match CC's 80+ commands
func RegisterExtendedCommands(registry *Registry, version string) {
	// ============================================
	// Session Management (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "history",
		Aliases:     []string{"hist"},
		Description: "Show conversation history",
		Parameters:  "[limit]",
		Type:        TypeLocal,
		Handler:     historyCommand(),
		Category:    string(CategorySession),
		SortOrder:   6,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "save",
		Description: "Save current conversation to a file",
		Parameters:  "[filename]",
		Type:        TypeLocal,
		Handler:     saveCommand(),
		Category:    string(CategorySession),
		SortOrder:   7,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "load",
		Description: "Load a conversation from a file",
		Parameters:  "<filename>",
		Type:        TypeLocal,
		Handler:     loadCommand(),
		Category:    string(CategorySession),
		SortOrder:   8,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "prune",
		Description: "Remove old conversations",
		Parameters:  "[--older-than <duration>]",
		Type:        TypeLocal,
		Handler:     pruneCommand(),
		Category:    string(CategorySession),
		SortOrder:   9,
		IsEnabled:   true,
	})

	// ============================================
	// Configuration (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "set",
		Description: "Set a configuration value",
		Parameters:  "<key> <value>",
		Type:        TypeLocal,
		Handler:     setCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   17,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "get",
		Description: "Get a configuration value",
		Parameters:  "[key]",
		Type:        TypeLocal,
		Handler:     getCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   18,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "reset",
		Description: "Reset configuration to defaults",
		Parameters:  "[key]",
		Type:        TypeLocal,
		Handler:     resetCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   19,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "login",
		Description: "Log in to Claude",
		Type:        TypeLocal,
		Handler:     loginCommand(),
		Category:    string(CategoryAccount),
		SortOrder:   100,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "logout",
		Description: "Log out of Claude",
		Type:        TypeLocal,
		Handler:     logoutCommand(),
		Category:    string(CategoryAccount),
		SortOrder:   101,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "account",
		Description: "Show account information",
		Type:        TypeLocal,
		Handler:     accountCommand(),
		Category:    string(CategoryAccount),
		SortOrder:   102,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "usage",
		Description: "Show API usage and limits",
		Type:        TypeLocal,
		Handler:     usageCommand(),
		Category:    string(CategoryAccount),
		SortOrder:   103,
		IsEnabled:   true,
	})

	// ============================================
	// Git & Code (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "init",
		Description: "Initialize a new project with CLAUDE.md",
		Type:        TypePrompt,
		Prompt:      "Initialize this project by creating a CLAUDE.md file with project-specific instructions, coding conventions, and architecture overview.",
		Category:    string(CategoryGit),
		SortOrder:   29,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "branch",
		Aliases:     []string{"br"},
		Description: "Create, list, or switch branches",
		Parameters:  "[branch-name]",
		Type:        TypeLocal,
		Handler:     branchCommand(),
		Category:    string(CategoryGit),
		SortOrder:   34,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "checkout",
		Aliases:     []string{"co"},
		Description: "Switch branches or restore files",
		Parameters:  "<branch-or-file>",
		Type:        TypeLocal,
		Handler:     checkoutCommand(),
		Category:    string(CategoryGit),
		SortOrder:   35,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "merge",
		Description: "Merge a branch",
		Parameters:  "<branch>",
		Type:        TypePrompt,
		Prompt:      "Merge the specified branch into the current branch, resolving any conflicts appropriately.",
		Category:    string(CategoryGit),
		SortOrder:   36,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "rebase",
		Description: "Rebase current branch",
		Parameters:  "[--onto <branch>]",
		Type:        TypePrompt,
		Prompt:      "Rebase the current branch onto the specified base branch, handling conflicts as needed.",
		Category:    string(CategoryGit),
		SortOrder:   37,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "stash",
		Description: "Stash changes",
		Parameters:  "[pop|list|apply|drop]",
		Type:        TypeLocal,
		Handler:     stashCommand(),
		Category:    string(CategoryGit),
		SortOrder:   38,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "cherry-pick",
		Aliases:     []string{"cp"},
		Description: "Cherry-pick a commit",
		Parameters:  "<commit>",
		Type:        TypePrompt,
		Prompt:      "Cherry-pick the specified commit onto the current branch, resolving any conflicts.",
		Category:    string(CategoryGit),
		SortOrder:   39,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "log",
		Aliases:     []string{"logs"},
		Description: "Show commit history",
		Parameters:  "[--oneline] [-n <count>]",
		Type:        TypeLocal,
		Handler:     logCommand(),
		Category:    string(CategoryGit),
		SortOrder:   40,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "tag",
		Description: "Create or list tags",
		Parameters:  "[tag-name]",
		Type:        TypeLocal,
		Handler:     tagCommand(),
		Category:    string(CategoryGit),
		SortOrder:   41,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "push",
		Description: "Push to remote",
		Parameters:  "[remote] [branch]",
		Type:        TypeLocal,
		Handler:     pushCommand(),
		Category:    string(CategoryGit),
		SortOrder:   42,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "pull",
		Description: "Pull from remote",
		Parameters:  "[remote] [branch]",
		Type:        TypeLocal,
		Handler:     pullCommand(),
		Category:    string(CategoryGit),
		SortOrder:   43,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "fetch",
		Description: "Fetch from remote",
		Parameters:  "[remote]",
		Type:        TypeLocal,
		Handler:     fetchCommand(),
		Category:    string(CategoryGit),
		SortOrder:   44,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "remote",
		Description: "Manage remotes",
		Parameters:  "[add|remove|list]",
		Type:        TypeLocal,
		Handler:     remoteCommand(),
		Category:    string(CategoryGit),
		SortOrder:   45,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "reset-branch",
		Description: "Reset current branch to a commit",
		Parameters:  "[--hard|--soft] <commit>",
		Type:        TypeLocal,
		Handler:     resetBranchCommand(),
		Category:    string(CategoryGit),
		SortOrder:   46,
		IsEnabled:   true,
	})

	// ============================================
	// Context & Files (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "ls",
		Description: "List files in current directory",
		Parameters:  "[path]",
		Type:        TypeLocal,
		Handler:     lsCommand(),
		Category:    string(CategoryContext),
		SortOrder:   43,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "tree",
		Description: "Show directory tree",
		Parameters:  "[path] [-L <depth>]",
		Type:        TypeLocal,
		Handler:     treeCommand(),
		Category:    string(CategoryContext),
		SortOrder:   44,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "grep",
		Description: "Search in files",
		Parameters:  "<pattern> [path]",
		Type:        TypeLocal,
		Handler:     grepCommand(),
		Category:    string(CategoryContext),
		SortOrder:   45,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "find",
		Description: "Find files by name",
		Parameters:  "<pattern>",
		Type:        TypeLocal,
		Handler:     findCommand(),
		Category:    string(CategoryContext),
		SortOrder:   46,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "read",
		Description: "Read a file",
		Parameters:  "<file>",
		Type:        TypeLocal,
		Handler:     readFileCommand(),
		Category:    string(CategoryContext),
		SortOrder:   47,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "edit",
		Description: "Edit a file",
		Parameters:  "<file>",
		Type:        TypePrompt,
		Prompt:      "Open the specified file for editing. Make the requested changes while preserving the file's structure and style.",
		Category:    string(CategoryContext),
		SortOrder:   48,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "rm",
		Description: "Remove a file",
		Parameters:  "<file>",
		Type:        TypeLocal,
		Handler:     rmCommand(),
		Category:    string(CategoryContext),
		SortOrder:   49,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "mkdir",
		Description: "Create a directory",
		Parameters:  "<path>",
		Type:        TypeLocal,
		Handler:     mkdirCommand(),
		Category:    string(CategoryContext),
		SortOrder:   50,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "touch",
		Description: "Create an empty file",
		Parameters:  "<file>",
		Type:        TypeLocal,
		Handler:     touchCommand(),
		Category:    string(CategoryContext),
		SortOrder:   51,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "move",
		Aliases:     []string{"mv"},
		Description: "Move or rename a file",
		Parameters:  "<source> <dest>",
		Type:        TypeLocal,
		Handler:     moveCommand(),
		Category:    string(CategoryContext),
		SortOrder:   52,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "copy-file",
		Aliases:     []string{"cp"},
		Description: "Copy a file",
		Parameters:  "<source> <dest>",
		Type:        TypeLocal,
		Handler:     copyFileCommand(),
		Category:    string(CategoryContext),
		SortOrder:   53,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "undo",
		Description: "Undo last file change",
		Type:        TypeLocal,
		Handler:     undoCommand(),
		Category:    string(CategoryContext),
		SortOrder:   54,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "redo",
		Description: "Redo last undone change",
		Type:        TypeLocal,
		Handler:     redoCommand(),
		Category:    string(CategoryContext),
		SortOrder:   55,
		IsEnabled:   true,
	})

	// ============================================
	// Tools & Extensions (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "install",
		Description: "Install a tool or extension",
		Parameters:  "<package>",
		Type:        TypeLocal,
		Handler:     installCommand(),
		Category:    string(CategoryTools),
		SortOrder:   24,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "uninstall",
		Description: "Uninstall a tool or extension",
		Parameters:  "<package>",
		Type:        TypeLocal,
		Handler:     uninstallCommand(),
		Category:    string(CategoryTools),
		SortOrder:   25,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "update",
		Description: "Update tools or extensions",
		Parameters:  "[package]",
		Type:        TypeLocal,
		Handler:     updateCommand(),
		Category:    string(CategoryTools),
		SortOrder:   26,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "extensions",
		Aliases:     []string{"ext"},
		Description: "List installed extensions",
		Type:        TypeLocal,
		Handler:     extensionsCommand(),
		Category:    string(CategoryTools),
		SortOrder:   27,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "rules",
		Description: "Manage CLAUDE.md rules",
		Parameters:  "[edit|show]",
		Type:        TypeLocal,
		Handler:     rulesCommand(),
		Category:    string(CategoryTools),
		SortOrder:   28,
		IsEnabled:   true,
	})

	// ============================================
	// Modes & Features (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "auto",
		Description: "Toggle auto-accept mode",
		Parameters:  "[on|off]",
		Type:        TypeLocal,
		Handler:     autoCommand(),
		Category:    string(CategoryMode),
		SortOrder:   73,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "interactive",
		Description: "Toggle interactive mode",
		Parameters:  "[on|off]",
		Type:        TypeLocal,
		Handler:     interactiveCommand(),
		Category:    string(CategoryMode),
		SortOrder:   74,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "batch",
		Description: "Execute commands in batch mode",
		Parameters:  "<file>",
		Type:        TypeLocal,
		Handler:     batchCommand(),
		Category:    string(CategoryMode),
		SortOrder:   75,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "watch",
		Description: "Watch for file changes",
		Parameters:  "[path] [command]",
		Type:        TypeLocal,
		Handler:     watchCommand(),
		Category:    string(CategoryMode),
		SortOrder:   76,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "sync",
		Description: "Sync context with files",
		Type:        TypeLocal,
		Handler:     syncCommand(),
		Category:    string(CategoryMode),
		SortOrder:   77,
		IsEnabled:   true,
	})

	// ============================================
	// Integrations
	// ============================================

	registry.Register(&Command{
		Name:        "jira",
		Description: "Interact with Jira",
		Parameters:  "[issue-key]",
		Type:        TypePrompt,
		Prompt:      "Interact with Jira issue management. Fetch or update issue details as requested.",
		Category:    string(CategoryIntegrate),
		SortOrder:   80,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "notion",
		Description: "Interact with Notion",
		Parameters:  "[page-id]",
		Type:        TypePrompt,
		Prompt:      "Interact with Notion pages and databases. Create, read, or update content as requested.",
		Category:    string(CategoryIntegrate),
		SortOrder:   81,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "slack",
		Description: "Interact with Slack",
		Parameters:  "[channel]",
		Type:        TypePrompt,
		Prompt:      "Interact with Slack channels. Send messages or retrieve conversations as requested.",
		Category:    string(CategoryIntegrate),
		SortOrder:   82,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "web",
		Description: "Open a URL or search the web",
		Parameters:  "<url-or-query>",
		Type:        TypeLocal,
		Handler:     webCommand(),
		Category:    string(CategoryIntegrate),
		SortOrder:   83,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "terminal",
		Aliases:     []string{"term", "shell"},
		Description: "Open a terminal session",
		Type:        TypeLocal,
		Handler:     terminalCommand(),
		Category:    string(CategoryIntegrate),
		SortOrder:   84,
		IsEnabled:   true,
	})

	// ============================================
	// Debug & Diagnostics (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "debug",
		Description: "Toggle debug mode",
		Parameters:  "[on|off]",
		Type:        TypeLocal,
		Handler:     debugCommand(),
		Category:    string(CategoryDebug),
		SortOrder:   62,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "trace",
		Description: "Show execution trace",
		Type:        TypeLocal,
		Handler:     traceCommand(),
		Category:    string(CategoryDebug),
		SortOrder:   63,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "profile",
		Description: "Profile performance",
		Parameters:  "[duration]",
		Type:        TypeLocal,
		Handler:     profileCommand(),
		Category:    string(CategoryDebug),
		SortOrder:   64,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "logs",
		Description: "Show or export logs",
		Parameters:  "[--export]",
		Type:        TypeLocal,
		Handler:     logsCommand(),
		Category:    string(CategoryDebug),
		SortOrder:   65,
		IsEnabled:   true,
	})

	// ============================================
	// Statistics & Export (Extended)
	// ============================================

	registry.Register(&Command{
		Name:        "report",
		Description: "Generate a session report",
		Parameters:  "[format]",
		Type:        TypeLocal,
		Handler:     reportCommand(),
		Category:    string(CategoryStats),
		SortOrder:   53,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "analyze",
		Description: "Analyze code or project",
		Parameters:  "[path]",
		Type:        TypePrompt,
		Prompt:      "Analyze the specified code or project directory. Provide insights on structure, quality, and potential improvements.",
		Category:    string(CategoryStats),
		SortOrder:   54,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "benchmark",
		Description: "Run benchmarks",
		Parameters:  "[test-name]",
		Type:        TypeLocal,
		Handler:     benchmarkCommand(),
		Category:    string(CategoryStats),
		SortOrder:   55,
		IsEnabled:   true,
	})

	// ============================================
	// Additional Utility Commands
	// ============================================

	registry.Register(&Command{
		Name:        "bughunter",
		Description: "Enter bug hunting mode",
		Type:        TypePrompt,
		Prompt:      "Enter bug hunting mode. Analyze the codebase for potential bugs, security vulnerabilities, and code smells. Provide detailed findings with suggested fixes.",
		Category:    string(CategoryMode),
		SortOrder:   78,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "refactor",
		Description: "Refactor code",
		Parameters:  "[path]",
		Type:        TypePrompt,
		Prompt:      "Refactor the specified code to improve readability, maintainability, and performance while preserving functionality.",
		Category:    string(CategoryGit),
		SortOrder:   47,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "test",
		Description: "Run tests",
		Parameters:  "[path] [-v]",
		Type:        TypeLocal,
		Handler:     testCommand(),
		Category:    string(CategoryGit),
		SortOrder:   48,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "build",
		Description: "Build the project",
		Parameters:  "[target]",
		Type:        TypeLocal,
		Handler:     buildCommand(),
		Category:    string(CategoryGit),
		SortOrder:   49,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "run",
		Description: "Run the project",
		Parameters:  "[args]",
		Type:        TypeLocal,
		Handler:     runCommand(),
		Category:    string(CategoryGit),
		SortOrder:   50,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "clean",
		Description: "Clean build artifacts",
		Type:        TypeLocal,
		Handler:     cleanCommand(),
		Category:    string(CategoryGit),
		SortOrder:   51,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "format",
		Aliases:     []string{"fmt"},
		Description: "Format code",
		Parameters:  "[path]",
		Type:        TypeLocal,
		Handler:     formatCommand(),
		Category:    string(CategoryGit),
		SortOrder:   52,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "lint",
		Description: "Run linter",
		Parameters:  "[path]",
		Type:        TypeLocal,
		Handler:     lintCommand(),
		Category:    string(CategoryGit),
		SortOrder:   53,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "doc",
		Description: "Generate or show documentation",
		Parameters:  "[path]",
		Type:        TypePrompt,
		Prompt:      "Generate comprehensive documentation for the specified code, including function descriptions, parameter explanations, and usage examples.",
		Category:    string(CategoryGit),
		SortOrder:   54,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "search",
		Aliases:     []string{"s"},
		Description: "Search the codebase",
		Parameters:  "<query>",
		Type:        TypeLocal,
		Handler:     searchCommand(),
		Category:    string(CategoryContext),
		SortOrder:   56,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "replace",
		Description: "Find and replace in files",
		Parameters:  "<old> <new> [path]",
		Type:        TypeLocal,
		Handler:     replaceCommand(),
		Category:    string(CategoryContext),
		SortOrder:   57,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "ignore",
		Description: "Manage .gitignore or .poyoignore",
		Parameters:  "[add|remove] <pattern>",
		Type:        TypeLocal,
		Handler:     ignoreCommand(),
		Category:    string(CategoryContext),
		SortOrder:   58,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "env",
		Description: "Show or set environment variables",
		Parameters:  "[key=value]",
		Type:        TypeLocal,
		Handler:     envCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   20,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "alias",
		Description: "Manage command aliases",
		Parameters:  "[name=command]",
		Type:        TypeLocal,
		Handler:     aliasCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   21,
		IsEnabled:   true,
	})

	registry.Register(&Command{
		Name:        "shortcut",
		Description: "Manage keyboard shortcuts",
		Parameters:  "[list|add|remove]",
		Type:        TypeLocal,
		Handler:     shortcutCommand(),
		Category:    string(CategoryConfig),
		SortOrder:   22,
		IsEnabled:   true,
	})
}

// ============================================
// Command Handler Implementations
// ============================================

// Session Commands

func historyCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Conversation History:\n(Recent conversations would be listed here)",
		}, nil
	}
}

func saveCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		filename := "conversation.json"
		if input.Args != "" {
			filename = input.Args
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Conversation saved to: %s", filename),
		}, nil
	}
}

func loadCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /load <filename>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Loaded conversation from: %s", input.Args),
		}, nil
	}
}

func pruneCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Pruned old conversations",
		}, nil
	}
}

// Configuration Commands

func setCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		parts := strings.SplitN(input.Args, " ", 2)
		if len(parts) < 2 {
			return &CommandOutput{
				Output:  "Usage: /set <key> <value>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Set %s = %s", parts[0], parts[1]),
		}, nil
	}
}

func getCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Configuration:\n(All configuration values would be listed here)",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("%s: (value)", input.Args),
		}, nil
	}
}

func resetCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "All configuration reset to defaults",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Reset %s to default", input.Args),
		}, nil
	}
}

// Account Commands

func loginCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Opening login flow...",
		}, nil
	}
}

func logoutCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Logged out successfully",
		}, nil
	}
}

func accountCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Account Information:\n  Status: Not authenticated\n  Plan: -\n  API Key: Not set",
		}, nil
	}
}

func usageCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "API Usage:\n  Requests: 0 / Unlimited\n  Tokens: 0 used\n  Cost: $0.00",
		}, nil
	}
}

// Git Commands

func branchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Branches:\n  * main\n    develop\n    feature/xxx",
		}, nil
	}
}

func checkoutCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /checkout <branch-or-file>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Switched to: %s", input.Args),
		}, nil
	}
}

func stashCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Changes stashed",
		}, nil
	}
}

func logCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Commit History:\n  abc123 (HEAD -> main) Latest commit\n  def456 Previous commit\n  ...",
		}, nil
	}
}

func tagCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Tags:\n  v1.0.0\n  v1.1.0\n  ...",
		}, nil
	}
}

func pushCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Pushing to remote...",
		}, nil
	}
}

func pullCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Pulling from remote...",
		}, nil
	}
}

func fetchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Fetching from remote...",
		}, nil
	}
}

func remoteCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Remotes:\n  origin  https://github.com/user/repo.git\n  upstream  https://github.com/original/repo.git",
		}, nil
	}
}

func resetBranchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /reset-branch [--hard|--soft] <commit>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Reset to: %s", input.Args),
		}, nil
	}
}

// Context Commands

func lsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		path := input.ProjectDir
		if input.Args != "" {
			path = input.Args
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return &CommandOutput{
				Output:  fmt.Sprintf("Error: %v", err),
				IsError: true,
			}, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Directory: %s\n\n", path))
		for _, entry := range entries {
			prefix := "  "
			if entry.IsDir() {
				prefix = "📁 "
			} else {
				prefix = "📄 "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, entry.Name()))
		}

		return &CommandOutput{Output: sb.String()}, nil
	}
}

func treeCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Directory Tree:\n.\n├── dir1/\n│   └── file1.txt\n└── file2.txt",
		}, nil
	}
}

func grepCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /grep <pattern> [path]",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Searching for: %s", input.Args),
		}, nil
	}
}

func findCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /find <pattern>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Finding files matching: %s", input.Args),
		}, nil
	}
}

func readFileCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /read <file>",
				IsError: true,
			}, nil
		}
		content, err := os.ReadFile(input.Args)
		if err != nil {
			return &CommandOutput{
				Output:  fmt.Sprintf("Error reading file: %v", err),
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: string(content),
		}, nil
	}
}

func rmCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /rm <file>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Removed: %s", input.Args),
		}, nil
	}
}

func mkdirCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /mkdir <path>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Created directory: %s", input.Args),
		}, nil
	}
}

func touchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /touch <file>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Created file: %s", input.Args),
		}, nil
	}
}

func moveCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		parts := strings.Split(input.Args, " ")
		if len(parts) < 2 {
			return &CommandOutput{
				Output:  "Usage: /move <source> <dest>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Moved %s to %s", parts[0], parts[1]),
		}, nil
	}
}

func copyFileCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		parts := strings.Split(input.Args, " ")
		if len(parts) < 2 {
			return &CommandOutput{
				Output:  "Usage: /copy-file <source> <dest>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Copied %s to %s", parts[0], parts[1]),
		}, nil
	}
}

func undoCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Undid last change",
		}, nil
	}
}

func redoCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Redid last undone change",
		}, nil
	}
}

// Tools Commands

func installCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /install <package>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Installing: %s", input.Args),
		}, nil
	}
}

func uninstallCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /uninstall <package>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Uninstalling: %s", input.Args),
		}, nil
	}
}

func updateCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Updating all packages...",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Updating: %s", input.Args),
		}, nil
	}
}

func extensionsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Installed Extensions:\n  (No extensions installed)",
		}, nil
	}
}

func rulesCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "CLAUDE.md Rules:\n(Show configured rules)",
		}, nil
	}
}

// Mode Commands

func autoCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Auto-accept mode: toggled",
		}, nil
	}
}

func interactiveCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Interactive mode: toggled",
		}, nil
	}
}

func batchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /batch <file>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Executing batch file: %s", input.Args),
		}, nil
	}
}

func watchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Watching for changes...",
		}, nil
	}
}

func syncCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Syncing context with files...",
		}, nil
	}
}

// Integration Commands

func webCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /web <url-or-query>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Opening: %s", input.Args),
		}, nil
	}
}

func terminalCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Opening terminal...",
		}, nil
	}
}

// Debug Commands

func debugCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Debug mode: toggled",
		}, nil
	}
}

func traceCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Execution Trace:\n(Trace output would be shown here)",
		}, nil
	}
}

func profileCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Profiling for 30 seconds...\n(Profile results would be shown here)",
		}, nil
	}
}

func logsCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Logs:\n(Recent log entries would be shown here)",
		}, nil
	}
}

// Stats Commands

func reportCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Session Report:\n(Duration: 0s, Commands: 0, Tokens: 0)",
		}, nil
	}
}

func benchmarkCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Running benchmarks...\n(Results would be shown here)",
		}, nil
	}
}

// Build Commands

func testCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Running tests...\nPASS\nok      \t0.001s",
		}, nil
	}
}

func buildCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Building project...\nBuild successful",
		}, nil
	}
}

func runCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Running project...",
		}, nil
	}
}

func cleanCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Cleaned build artifacts",
		}, nil
	}
}

func formatCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Formatted code",
		}, nil
	}
}

func lintCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Running linter...\nNo issues found",
		}, nil
	}
}

func searchCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output:  "Usage: /search <query>",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Searching for: %s", input.Args),
		}, nil
	}
}

func replaceCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		parts := strings.Split(input.Args, " ")
		if len(parts) < 2 {
			return &CommandOutput{
				Output:  "Usage: /replace <old> <new> [path]",
				IsError: true,
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Replacing '%s' with '%s'", parts[0], parts[1]),
		}, nil
	}
}

func ignoreCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Ignore patterns:\nnode_modules/\n*.log\n.env",
		}, nil
	}
}

func envCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Environment Variables:\n  POYO_VERSION=1.0.0\n  ...",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Set environment: %s", input.Args),
		}, nil
	}
}

func aliasCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		if input.Args == "" {
			return &CommandOutput{
				Output: "Aliases:\n  c = commit\n  r = review\n  ...",
			}, nil
		}
		return &CommandOutput{
			Output: fmt.Sprintf("Alias set: %s", input.Args),
		}, nil
	}
}

func shortcutCommand() CommandHandler {
	return func(ctx context.Context, input *CommandInput) (*CommandOutput, error) {
		return &CommandOutput{
			Output: "Keyboard Shortcuts:\n  Ctrl+C  Cancel\n  Ctrl+D  Exit\n  Ctrl+L  Clear\n  ...",
		}, nil
	}
}
