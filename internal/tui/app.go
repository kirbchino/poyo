// Package tui implements the application runner
package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbletea"
)

// App represents the TUI application
type App struct {
	model    Model
	program  *tea.Program
	opts     AppOptions
	ctx      context.Context
	cancel   context.CancelFunc
}

// AppOptions contains application options
type AppOptions struct {
	// Model configuration
	Model string

	// Permission mode
	PermissionMode string

	// Working directory
	WorkingDir string

	// Debug mode
	Debug bool

	// Initial prompt
	InitialPrompt string

	// Alt screen (full screen mode)
	AltScreen bool

	// Mouse support
	Mouse bool

	// Message handler - called when user sends a message
	OnMessage func(content string) (string, error)
}

// DefaultAppOptions returns default app options
func DefaultAppOptions() AppOptions {
	return AppOptions{
		Model:          "claude-sonnet-4-6",
		PermissionMode: "default",
		WorkingDir:     ".",
		Debug:          false,
		AltScreen:      true,
		Mouse:          true,
	}
}

// NewApp creates a new TUI application
func NewApp(opts AppOptions) *App {
	ctx, cancel := context.WithCancel(context.Background())

	model := NewModel()
	model.status.SetModel(opts.Model)
	model.status.SetPermission(opts.PermissionMode)
	model.onMessage = opts.OnMessage

	return &App{
		model:  model,
		opts:   opts,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run starts the TUI application
func (a *App) Run() error {
	// Create tea program options
	var opts []tea.ProgramOption
	if a.opts.AltScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	if a.opts.Mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	// Create program
	a.program = tea.NewProgram(a.model, opts...)

	// Handle initial prompt if provided
	if a.opts.InitialPrompt != "" {
		go func() {
			// Wait for the program to be ready
			// Then send the initial message
			a.program.Send(UserMessageMsg{Content: a.opts.InitialPrompt})
		}()
	}

	// Run the program
	_, err := a.program.Run()
	return err
}

// RunWithContext starts the TUI application with context
func (a *App) RunWithContext(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.ctx = ctx

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	return a.Run()
}

// SendUserMessage sends a user message to the app
func (a *App) SendUserMessage(content string) {
	if a.program != nil {
		a.program.Send(UserMessageMsg{Content: content})
	}
}

// SendAssistantMessage sends an assistant message to the app
func (a *App) SendAssistantMessage(content string) {
	if a.program != nil {
		a.program.Send(AssistantMessageMsg{Content: content})
	}
}

// SendError sends an error to the app
func (a *App) SendError(err error) {
	if a.program != nil {
		a.program.Send(ErrorMessage{Err: err})
	}
}

// SendToolStart signals a tool execution start
func (a *App) SendToolStart(toolID, toolName string) {
	if a.program != nil {
		a.program.Send(ToolStartMsg{
			ToolID:   toolID,
			ToolName: toolName,
		})
	}
}

// SendToolProgress signals tool execution progress
func (a *App) SendToolProgress(toolID, output string) {
	if a.program != nil {
		a.program.Send(ToolProgressMsg{
			ToolID: toolID,
			Output: output,
		})
	}
}

// SendToolEnd signals a tool execution end
func (a *App) SendToolEnd(toolID, output string) {
	if a.program != nil {
		a.program.Send(ToolEndMsg{
			ToolID: toolID,
			Output: output,
		})
	}
}

// UpdateUsage updates the token usage display
func (a *App) UpdateUsage(inputTokens, outputTokens int) {
	if a.program != nil {
		a.program.Send(UsageUpdateMsg{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		})
	}
}

// SetProcessing sets the processing state
func (a *App) SetProcessing(processing bool) {
	if a.program != nil {
		if processing {
			a.program.Send(ProcessingStartMsg{})
		} else {
			a.program.Send(ProcessingEndMsg{})
		}
	}
}

// Quit quits the application
func (a *App) Quit() {
	if a.program != nil {
		a.program.Send(tea.Quit())
	}
}

// GetModel returns the underlying model
func (a *App) GetModel() *Model {
	return &a.model
}

// RunSimple runs a simple TUI (for basic use cases)
func RunSimple() error {
	app := NewApp(DefaultAppOptions())
	return app.Run()
}

// RunWithPrompt runs the TUI with an initial prompt
func RunWithPrompt(prompt string) error {
	opts := DefaultAppOptions()
	opts.InitialPrompt = prompt
	app := NewApp(opts)
	return app.Run()
}

// TerminalSize returns the terminal dimensions
func TerminalSize() (width, height int, err error) {
	width = 80
	height = 24

	// Try to get actual terminal size
	if os.Stdout != nil {
		if w, h, err := getTerminalSize(); err == nil {
			width = w
			height = h
		}
	}

	return width, height, nil
}

// getTerminalSize gets the terminal size using stty
func getTerminalSize() (width, height int, err error) {
	// This would use syscall or stty to get the actual size
	// For now, return defaults
	return 80, 24, nil
}

// AppError represents an application error
type AppError struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Common app errors
var (
	ErrAppNotInitialized = &AppError{Code: "APP_NOT_INITIALIZED", Message: "Application not initialized"}
	ErrAppAlreadyRunning = &AppError{Code: "APP_ALREADY_RUNNING", Message: "Application already running"}
	ErrTerminalNotAvailable = &AppError{Code: "TERMINAL_NOT_AVAILABLE", Message: "Terminal not available"}
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppErrorCode returns the error code from an AppError
func GetAppErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "UNKNOWN"
}
