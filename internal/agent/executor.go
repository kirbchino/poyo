package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Executor handles agent execution
type Executor struct {
	registry    *AgentRegistry
	toolCaller  ToolCaller
	mu          sync.RWMutex
}

// ToolCaller is the interface for calling tools
type ToolCaller interface {
	// CallTool calls a tool with the given input
	CallTool(ctx context.Context, toolName string, input interface{}) (interface{}, error)

	// ListTools returns the list of available tools
	ListTools() []string
}

// NewExecutor creates a new agent executor
func NewExecutor(toolCaller ToolCaller) *Executor {
	return &Executor{
		registry:   NewAgentRegistry(),
		toolCaller: toolCaller,
	}
}

// Execute runs an agent with the given configuration
func (e *Executor) Execute(parentCtx context.Context, config *AgentConfig) (*AgentResult, error) {
	// Generate agent ID
	agentID := GenerateAgentID()

	// Determine agent type
	agentType := config.Type
	if agentType == "" {
		agentType = AgentTypeGeneralPurpose
	}

	// Get tool access
	toolAccess := DefaultToolAccess(agentType)

	// Create context with timeout
	timeout := 10 * time.Minute
	if config.Isolation == IsolationWorktree {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	// Create agent instance
	instance := &AgentInstance{
		ID:        agentID,
		Type:      agentType,
		State:     StatePending,
		Config:    config,
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	// Register the agent
	e.registry.Register(instance)
	defer e.registry.Remove(agentID)

	// Update state to running
	instance.State = StateRunning

	// Execute the agent
	result, err := e.executeAgent(ctx, agentID, config, toolAccess)
	if err != nil {
		instance.State = StateFailed
		instance.EndTime = time.Now()
		result = &AgentResult{
			AgentID:  agentID,
			State:    StateFailed,
			Error:    err.Error(),
			Duration: instance.EndTime.Sub(instance.StartTime),
		}
		return result, err
	}

	// Update result
	instance.State = StateCompleted
	instance.EndTime = time.Now()
	instance.Result = result
	result.Duration = instance.EndTime.Sub(instance.StartTime)

	return result, nil
}

// executeAgent executes the agent logic
func (e *Executor) executeAgent(ctx context.Context, agentID string, config *AgentConfig, toolAccess *AgentToolAccess) (*AgentResult, error) {
	result := &AgentResult{
		AgentID: agentID,
		State:   StateRunning,
	}

	// Build system prompt based on agent type
	systemPrompt := e.buildSystemPrompt(config, toolAccess)

	// Build conversation messages
	messages := []Message{
		{Role: "user", Content: config.Prompt},
	}

	// Track execution
	turnCount := 0
	maxTurns := 10

	// Simulate agent execution (in real implementation, this would call the LLM)
	// For now, return a placeholder result
	for turnCount < maxTurns {
		select {
		case <-ctx.Done():
			result.State = StateStopped
			result.Error = "agent execution stopped"
			return result, ctx.Err()
		default:
		}

		// In real implementation:
		// 1. Call LLM with messages
		// 2. Parse response for tool calls
		// 3. Execute allowed tools
		// 4. Add tool results to messages
		// 5. Continue until no more tool calls or max turns

		turnCount++
	}

	result.State = StateCompleted
	result.Output = fmt.Sprintf("Agent %s completed execution", agentID)
	result.TokensUsed = 1000 // Placeholder

	return result, nil
}

// buildSystemPrompt builds the system prompt for the agent
func (e *Executor) buildSystemPrompt(config *AgentConfig, toolAccess *AgentToolAccess) string {
	var sb strings.Builder

	// Agent type description
	switch config.Type {
	case AgentTypeExplore:
		sb.WriteString("You are a specialized agent for exploring codebases. ")
		sb.WriteString("Use read-only tools to quickly search and read files. ")
		sb.WriteString("Focus on answering questions about code structure and content.\n")

	case AgentTypePlan:
		sb.WriteString("You are a planning agent. ")
		sb.WriteString("Analyze the codebase and create detailed implementation plans. ")
		sb.WriteString("Use read-only tools to understand the codebase before planning.\n")

	default:
		sb.WriteString("You are a general-purpose agent. ")
		sb.WriteString("Execute the given task using available tools.\n")
	}

	// Tool restrictions
	if toolAccess.ReadOnly {
		sb.WriteString("\nIMPORTANT: You can only use read-only tools. Do not modify any files.\n")
	}

	// Available tools
	sb.WriteString("\nAvailable tools: ")
	sb.WriteString(strings.Join(toolAccess.AllowedTools, ", "))
	sb.WriteString("\n")

	// Denied tools
	if len(toolAccess.DeniedTools) > 0 {
		sb.WriteString("Denied tools: ")
		sb.WriteString(strings.Join(toolAccess.DeniedTools, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetAgent retrieves an agent by ID
func (e *Executor) GetAgent(id string) (*AgentInstance, bool) {
	return e.registry.Get(id)
}

// ListAgents lists all agents
func (e *Executor) ListAgents() []*AgentInstance {
	return e.registry.List()
}

// StopAgent stops a running agent
func (e *Executor) StopAgent(id string) error {
	instance, ok := e.registry.Get(id)
	if !ok {
		return fmt.Errorf("agent %s not found", id)
	}

	if instance.State != StateRunning {
		return fmt.Errorf("agent %s is not running", id)
	}

	if instance.Cancel != nil {
		instance.Cancel()
	}

	instance.State = StateStopped
	instance.EndTime = time.Now()

	return nil
}

// Message represents a conversation message
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ToolCallFunction       `json:"function"`
}

// ToolCallFunction represents the function part of a tool call
type ToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}
