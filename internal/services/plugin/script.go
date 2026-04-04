// Package plugin provides script-based plugin support
package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ScriptRuntime represents a script runtime
type ScriptRuntime string

const (
	RuntimeShell   ScriptRuntime = "shell"
	RuntimePython  ScriptRuntime = "python"
	RuntimeNode    ScriptRuntime = "node"
	RuntimeRuby    ScriptRuntime = "ruby"
	RuntimePerl    ScriptRuntime = "perl"
	RuntimeGeneric ScriptRuntime = "generic"
)

// ScriptEnvironment represents the environment passed to scripts
type ScriptEnvironment struct {
	PluginID      string                 `json:"pluginId"`
	PluginName    string                 `json:"pluginName"`
	PluginVersion string                 `json:"pluginVersion"`
	PluginPath    string                 `json:"pluginPath"`
	WorkingDir    string                 `json:"workingDir"`
	Config        map[string]interface{} `json:"config"`
	Context       map[string]interface{} `json:"context"`
}

// ScriptInput represents input passed to scripts
type ScriptInput struct {
	Method  string                 `json:"method"`
	Args    map[string]interface{} `json:"args"`
	Env     ScriptEnvironment      `json:"env"`
	Context map[string]interface{} `json:"context"`
}

// ScriptOutput represents output from scripts
type ScriptOutput struct {
	Success bool                   `json:"success"`
	Result  interface{}            `json:"result,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Logs    []string               `json:"logs,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ScriptPluginHandler handles script-based plugins (Python, Node, Shell, etc.)
type ScriptPluginHandler struct {
	mu           sync.RWMutex
	workingDir   string
	toolExec     ToolExecutor
	env          map[string]string
	processes    map[string]*exec.Cmd // Plugin ID -> running process
}

// NewScriptPluginHandler creates a new script plugin handler
func NewScriptPluginHandler(workingDir string) *ScriptPluginHandler {
	return &ScriptPluginHandler{
		workingDir: workingDir,
		env:        make(map[string]string),
		processes:  make(map[string]*exec.Cmd),
	}
}

// SetToolExecutor sets the tool executor for script plugins
func (h *ScriptPluginHandler) SetToolExecutor(executor ToolExecutor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolExec = executor
}

// SetEnv sets an environment variable for scripts
func (h *ScriptPluginHandler) SetEnv(key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.env[key] = value
}

// Load loads a script plugin
func (h *ScriptPluginHandler) Load(ctx context.Context, plugin *Plugin) error {
	if plugin.Main == "" {
		return fmt.Errorf("no main script specified")
	}

	scriptPath := filepath.Join(plugin.Path, plugin.Main)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	// Make script executable
	os.Chmod(scriptPath, 0755)

	return nil
}

// Unload unloads a script plugin
func (h *ScriptPluginHandler) Unload(ctx context.Context, plugin *Plugin) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cmd, ok := h.processes[plugin.ID]; ok {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		delete(h.processes, plugin.ID)
	}

	return nil
}

// Execute executes a script plugin method
func (h *ScriptPluginHandler) Execute(ctx context.Context, plugin *Plugin, method string, input interface{}) (interface{}, error) {
	scriptPath := filepath.Join(plugin.Path, plugin.Main)

	// Build script input
	scriptInput := ScriptInput{
		Method: method,
		Args:   make(map[string]interface{}),
		Env: ScriptEnvironment{
			PluginID:      plugin.ID,
			PluginName:    plugin.Name,
			PluginVersion: plugin.Version,
			PluginPath:    plugin.Path,
			WorkingDir:    h.workingDir,
			Config:        plugin.Config,
		},
	}

	// Convert input to args
	if input != nil {
		if m, ok := input.(map[string]interface{}); ok {
			scriptInput.Args = m
		}
	}

	// Determine runtime
	runtime := h.detectRuntime(scriptPath)

	// Create command
	cmd := h.createCommand(ctx, runtime, scriptPath, method)
	cmd.Dir = plugin.Path

	// Set environment
	cmd.Env = h.buildEnv(plugin)

	// Pass input as JSON via stdin
	inputJSON, err := json.Marshal(scriptInput)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start script: %w", err)
	}

	// Store process
	h.mu.Lock()
	h.processes[plugin.ID] = cmd
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.processes, plugin.ID)
		h.mu.Unlock()
	}()

	// Send input
	go func() {
		defer stdin.Close()
		stdin.Write(inputJSON)
	}()

	// Read output with timeout
	outputChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		output, err := io.ReadAll(stdout)
		if err != nil {
			errChan <- err
		} else {
			outputChan <- output
		}
	}()

	// Read stderr for logging
	var stderrOutput string
	go func() {
		scanner := bufio.NewScanner(stderr)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		stderrOutput = strings.Join(lines, "\n")
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()
	case output := <-outputChan:
		// Wait for process to complete
		cmd.Wait()

		// Parse output
		var scriptOutput ScriptOutput
		if err := json.Unmarshal(output, &scriptOutput); err != nil {
			// Return as string if not JSON
			result := strings.TrimSpace(string(output))
			if result == "" && stderrOutput != "" {
				return nil, fmt.Errorf("script error: %s", stderrOutput)
			}
			return result, nil
		}

		if !scriptOutput.Success {
			return nil, fmt.Errorf("script error: %s", scriptOutput.Error)
		}

		return scriptOutput.Result, nil
	case err := <-errChan:
		cmd.Process.Kill()
		return nil, fmt.Errorf("read output: %w", err)
	case <-time.After(60 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("script execution timeout")
	}
}

// detectRuntime detects the script runtime from file extension
func (h *ScriptPluginHandler) detectRuntime(scriptPath string) ScriptRuntime {
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".sh":
		return RuntimeShell
	case ".py":
		return RuntimePython
	case ".js", ".mjs":
		return RuntimeNode
	case ".rb":
		return RuntimeRuby
	case ".pl":
		return RuntimePerl
	default:
		return RuntimeGeneric
	}
}

// createCommand creates an exec command for the given runtime
func (h *ScriptPluginHandler) createCommand(ctx context.Context, runtime ScriptRuntime, scriptPath string, method string) *exec.Cmd {
	switch runtime {
	case RuntimeShell:
		return exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("source %s && %s", scriptPath, method))
	case RuntimePython:
		return exec.CommandContext(ctx, "python", scriptPath, method)
	case RuntimeNode:
		return exec.CommandContext(ctx, "node", scriptPath, method)
	case RuntimeRuby:
		return exec.CommandContext(ctx, "ruby", scriptPath, method)
	case RuntimePerl:
		return exec.CommandContext(ctx, "perl", scriptPath, method)
	default:
		return exec.CommandContext(ctx, scriptPath, method)
	}
}

// buildEnv builds environment variables for scripts
func (h *ScriptPluginHandler) buildEnv(plugin *Plugin) []string {
	env := os.Environ()

	// Add Kirby API environment
	env = append(env,
		"POYO_API_VERSION=1.0",
		"POYO_PLUGIN_ID="+plugin.ID,
		"POYO_PLUGIN_NAME="+plugin.Name,
		"POYO_PLUGIN_VERSION="+plugin.Version,
		"POYO_PLUGIN_PATH="+plugin.Path,
		"POYO_DREAM_LAND="+h.workingDir, // 梦之国 Dream Land
	)

	// Add custom environment variables
	h.mu.RLock()
	for k, v := range h.env {
		env = append(env, k+"="+v)
	}
	h.mu.RUnlock()

	// Add plugin config as JSON
	if len(plugin.Config) > 0 {
		configJSON, _ := json.Marshal(plugin.Config)
		env = append(env, "POYO_PLUGIN_CONFIG="+string(configJSON))
	}

	return env
}

// ExecuteHook executes a hook for a script plugin
func (h *ScriptPluginHandler) ExecuteHook(ctx context.Context, plugin *Plugin, hookType string, input map[string]interface{}) (*HookResult, error) {
	result, err := h.Execute(ctx, plugin, hookType, input)
	if err != nil {
		return nil, err
	}

	// Convert result to HookResult
	hr := &HookResult{}

	switch v := result.(type) {
	case map[string]interface{}:
		if blocked, ok := v["blocked"].(bool); ok {
			hr.Blocked = blocked
		}
		if reason, ok := v["reason"].(string); ok {
			hr.Reason = reason
		}
		if message, ok := v["message"].(string); ok {
			hr.Message = message
		}
		if modified, ok := v["modified"].(bool); ok {
			hr.Modified = modified
		}
		if data, ok := v["data"].(map[string]interface{}); ok {
			hr.Data = data
		}
	case string:
		if v == "block" || v == "blocked" {
			hr.Blocked = true
		}
	}

	return hr, nil
}

// ExecuteTool executes a tool from a script plugin
func (h *ScriptPluginHandler) ExecuteTool(ctx context.Context, plugin *Plugin, toolName string, input interface{}) (interface{}, error) {
	return h.Execute(ctx, plugin, "tool_"+toolName, input)
}

// PythonScriptHelper returns Python helper code for Kirby API
func PythonScriptHelper() string {
	return `
import json
import sys
import os

class KirbyAPI:
    @staticmethod
    def get_input():
        """Get input from stdin"""
        return json.loads(sys.stdin.read())

    @staticmethod
    def output(result, success=True, error=None):
        """Output result to stdout"""
        output = {"success": success, "result": result}
        if error:
            output["error"] = error
        print(json.dumps(output))

    @staticmethod
    def get_env():
        """Get plugin environment"""
        return {
            "plugin_id": os.environ.get("POYO_PLUGIN_ID", ""),
            "plugin_name": os.environ.get("POYO_PLUGIN_NAME", ""),
            "plugin_version": os.environ.get("POYO_PLUGIN_VERSION", ""),
            "plugin_path": os.environ.get("POYO_PLUGIN_PATH", ""),
            "dream_land": os.environ.get("POYO_DREAM_LAND", ""),
            "config": json.loads(os.environ.get("POYO_PLUGIN_CONFIG", "{}"))
        }

    @staticmethod
    def log(message, level="info"):
        """Log a message"""
        print(f"[{level.upper()}] {message}", file=sys.stderr)

    @staticmethod
    def say(message):
        """Kirby says something!"""
        print(f"💚 Kirby says: {message}", file=sys.stderr)
`
}

// NodeScriptHelper returns Node.js helper code for Kirby API
func NodeScriptHelper() string {
	return `
const Kirby = {
    getInput: () => JSON.parse(require('fs').readFileSync(0, 'utf-8')),

    output: (result, success = true, error = null) => {
        const output = { success, result };
        if (error) output.error = error;
        console.log(JSON.stringify(output));
    },

    getEnv: () => ({
        pluginId: process.env.POYO_PLUGIN_ID || '',
        pluginName: process.env.POYO_PLUGIN_NAME || '',
        pluginVersion: process.env.POYO_PLUGIN_VERSION || '',
        pluginPath: process.env.POYO_PLUGIN_PATH || '',
        dreamLand: process.env.POYO_DREAM_LAND || '',
        config: JSON.parse(process.env.POYO_PLUGIN_CONFIG || '{}')
    }),

    log: (message, level = 'info') => {
        console.error('[' + level.toUpperCase() + '] ' + message);
    },

    say: (message) => {
        console.error('💚 Kirby says: ' + message);
    }
};
`
}
