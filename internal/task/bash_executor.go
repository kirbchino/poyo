package task

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BashExecutor executes bash commands as background tasks
type BashExecutor struct {
	manager    *Manager
	sandbox    SandboxFunc
	env        map[string]string
	timeout    time.Duration
}

// SandboxFunc wraps a command with sandbox isolation
type SandboxFunc func(cmd *exec.Cmd) error

// NewBashExecutor creates a new bash executor
func NewBashExecutor(manager *Manager) *BashExecutor {
	return &BashExecutor{
		manager: manager,
		env:     make(map[string]string),
		timeout: 2 * time.Minute,
	}
}

// SetSandbox sets the sandbox function
func (e *BashExecutor) SetSandbox(sandbox SandboxFunc) {
	e.sandbox = sandbox
}

// SetEnvironment sets environment variables
func (e *BashExecutor) SetEnvironment(env map[string]string) {
	e.env = env
}

// SetTimeout sets the default timeout
func (e *BashExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// CreateTask creates a new bash task
func (e *BashExecutor) CreateTask(command string, workingDir string) *Task {
	return e.manager.Create(TaskOptions{
		Type:       TypeBash,
		Name:       truncateCommand(command, 50),
		Command:    command,
		WorkingDir: workingDir,
	})
}

// Execute runs the bash task
func (e *BashExecutor) Execute(ctx context.Context, task *Task) error {
	// Create command
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", task.Command)
	cmd.Dir = task.WorkingDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range e.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Apply sandbox if configured
	if e.sandbox != nil {
		if err := e.sandbox(cmd); err != nil {
			return fmt.Errorf("sandbox error: %w", err)
		}
	}

	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		e.streamOutput(task.ID, stdout, false)
	}()

	go func() {
		defer wg.Done()
		e.streamOutput(task.ID, stderr, true)
	}()

	// Wait for output streaming to complete
	wg.Wait()

	// Wait for command to finish
	err = cmd.Wait()

	// Get exit code
	if exitErr, ok := err.(*exec.ExitError); ok {
		task.ExitCode = exitErr.ExitCode()
	} else if err == nil {
		task.ExitCode = 0
	} else {
		task.ExitCode = -1
	}

	return err
}

// streamOutput streams output from a reader to the task
func (e *BashExecutor) streamOutput(taskID string, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if isStderr {
			line = "[stderr] " + line
		}
		e.manager.AppendOutput(taskID, line+"\n")
	}
}

// RunSync runs a command synchronously and returns the output
func (e *BashExecutor) RunSync(ctx context.Context, command string, workingDir string, timeout time.Duration) (string, int, error) {
	if timeout == 0 {
		timeout = e.timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	cmd.Dir = workingDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range e.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Apply sandbox if configured
	if e.sandbox != nil {
		if err := e.sandbox(cmd); err != nil {
			return "", -1, fmt.Errorf("sandbox error: %w", err)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]\n" + stderr.String()
	}

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	return output, exitCode, err
}

// truncateCommand truncates a command for display
func truncateCommand(cmd string, maxLen int) string {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) <= maxLen {
		return cmd
	}
	return cmd[:maxLen-3] + "..."
}
