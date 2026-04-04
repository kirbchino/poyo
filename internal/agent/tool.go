package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AgentTool provides the tool interface for agent execution
type AgentTool struct {
	executor   *Executor
	defaultCtx context.Context
}

// NewAgentTool creates a new AgentTool
func NewAgentTool(toolCaller ToolCaller) *AgentTool {
	return &AgentTool{
		executor: NewExecutor(toolCaller),
	}
}

// Name returns the tool name
func (t *AgentTool) Name() string {
	return "Agent"
}

// Description returns the tool description
func (t *AgentTool) Description() string {
	return `Launch a new agent that will handle complex, multi-step tasks autonomously.

The Agent tool launches specialized agents (subprocesses) that autonomously handle complex tasks. Each agent type has specific capabilities and tools available to it.

Available agent types and the tools they have access to:
- general-purpose: General-purpose agent for researching complex questions, searching for code, and executing multi-step tasks. (Tools: *)
- Explore: Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (e.g. "src/components/**/*.tsx"), search code for keywords (e.g. "API endpoints"), or answer questions about the codebase (e.g. "how do API endpoints work?"). (Tools: All tools except Agent, ExitPlanMode, Edit, Write, NotebookEdit)
- Plan: Software architect agent for designing implementation plans. Use this when you need to plan the implementation strategy for a task. Returns step-by-step plans, identifies critical files, and considers architectural trade-offs. (Tools: All tools except Agent, ExitPlanMode, Edit, Write, NotebookEdit)

When using the Agent tool, specify a subagent_type parameter to select which agent type to use. If omitted, the general-purpose agent is used.

Usage notes:
- Always include a short description (3-5 words) summarizing what the agent will do
- Launch multiple agents concurrently whenever possible, to maximize performance; do that, use a single message with multiple tool uses
- You can optionally run agents in the background using the run_in_background parameter. When you run an agent in the background, you will be automatically notified when it completes — do NOT sleep, poll, or proactively check on its progress. Continue with other work and respond to the user instead.
- **Foreground vs background**: Use foreground (default) when you need the agent's results before you can proceed — e.g., research agents whose findings inform your next steps. Use background when you have genuinely independent work to do in parallel.
- Agents can be resumed using the resume parameter by passing the agent ID from a previous invocation. When resumed, the agent continues with its full previous context preserved. When NOT resuming, each invocation starts fresh, so provide a detailed task description with all necessary context.
- Provide clear, detailed prompts so the agent can work autonomously and return exactly the information you need.
- The agent's outputs should generally be trusted
- Clearly tell the agent whether you expect it to write code or just do research (search for code, file reads, web fetches, etc.), since the agent is not aware of the user's intent

When to use this tool:
- Important: Only use this tool when the task requires planning the implementation steps of a task that requires writing code. For research tasks where you're gathering information, searching files, reading files or in general trying to understand the codebase — do NOT use this tool.`
}

// InputSchema returns the JSON schema for the tool input
func (t *AgentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"subagent_type": map[string]interface{}{
				"type":        "string",
				"description": "The type of specialized agent to use. If omitted, uses the general-purpose agent.",
				"enum":        []string{"general-purpose", "Explore", "Plan"},
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A short (3-5 word) description of the task",
				"minLength":   1,
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
			"resume": map[string]interface{}{
				"type":        "string",
				"description": "Optional agent ID to resume from. If provided, the agent will continue from the previous execution transcript.",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Optional model override for this agent. Takes precedence over the agent definition's model frontmatter. If omitted, uses the agent definition's model, or inherits from the parent.",
				"enum":        []string{"sonnet", "opus", "haiku"},
			},
			"isolation": map[string]interface{}{
				"type":        "string",
				"description": "Isolation mode. \"worktree\" creates a temporary git worktree so the agent works on an isolated copy of the repo. The worktree is automatically cleaned up if the agent makes no changes.",
				"enum":        []string{"worktree"},
			},
			"run_in_background": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to run this agent in the background. You will be notified when it completes.",
			},
		},
		"required": []string{"prompt"},
	}
}

// Execute executes the tool
func (t *AgentTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var config AgentConfig
	if err := json.Unmarshal(input, &config); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate required fields
	if config.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Execute the agent
	result, err := t.executor.Execute(ctx, &config)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAgent retrieves an agent result by ID
func (t *AgentTool) GetAgent(id string) (*AgentResult, bool) {
	instance, ok := t.executor.GetAgent(id)
	if !ok {
		return nil, false
	}
	return instance.Result, true
}

// ListAgents lists all agents
func (t *AgentTool) ListAgents() []*AgentInstance {
	return t.executor.ListAgents()
}

// StopAgent stops a running agent
func (t *AgentTool) StopAgent(id string) error {
	return t.executor.StopAgent(id)
}

// BackgroundAgentManager manages background agents
type BackgroundAgentManager struct {
	agents   map[string]*BackgroundAgent
	mu       sync.RWMutex
	notifier func(agentID string, result *AgentResult)
}

// BackgroundAgent represents a background agent
type BackgroundAgent struct {
	ID       string
	Config   *AgentConfig
	Result   *AgentResult
	Start    time.Time
	Done     chan struct{}
}

// NewBackgroundAgentManager creates a new background agent manager
func NewBackgroundAgentManager() *BackgroundAgentManager {
	return &BackgroundAgentManager{
		agents: make(map[string]*BackgroundAgent),
	}
}

// SetNotifier sets the notification callback
func (m *BackgroundAgentManager) SetNotifier(notifier func(agentID string, result *AgentResult)) {
	m.notifier = notifier
}

// Start starts a background agent
func (m *BackgroundAgentManager) Start(ctx context.Context, executor *Executor, config *AgentConfig) string {
	agentID := GenerateAgentID()

	bgAgent := &BackgroundAgent{
		ID:     agentID,
		Config: config,
		Start:  time.Now(),
		Done:   make(chan struct{}),
	}

	m.mu.Lock()
	m.agents[agentID] = bgAgent
	m.mu.Unlock()

	// Execute in background
	go func() {
		defer close(bgAgent.Done)

		result, err := executor.Execute(ctx, config)
		if err != nil {
			result = &AgentResult{
				AgentID: agentID,
				State:   StateFailed,
				Error:   err.Error(),
			}
		}
		bgAgent.Result = result

		// Notify completion
		if m.notifier != nil {
			m.notifier(agentID, result)
		}

		// Clean up after notification
		m.mu.Lock()
		delete(m.agents, agentID)
		m.mu.Unlock()
	}()

	return agentID
}

// Get retrieves a background agent
func (m *BackgroundAgentManager) Get(id string) (*BackgroundAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.agents[id]
	return agent, ok
}

// Wait waits for a background agent to complete
func (m *BackgroundAgentManager) Wait(id string, timeout time.Duration) (*AgentResult, error) {
	agent, ok := m.Get(id)
	if !ok {
		return nil, fmt.Errorf("agent %s not found", id)
	}

	select {
	case <-agent.Done:
		return agent.Result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for agent %s", id)
	}
}
