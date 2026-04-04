// Package team provides multi-agent team collaboration capabilities.
package team

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AgentRole represents the role of an agent in a team
type AgentRole string

const (
	RoleCoordinator AgentRole = "coordinator" // 协调者
	RoleWorker      AgentRole = "worker"      // 工作者
	RoleReviewer    AgentRole = "reviewer"    // 审核者
	RoleValidator   AgentRole = "validator"   // 验证者
	RoleResearcher  AgentRole = "researcher"  // 研究者
	RoleExecutor    AgentRole = "executor"    // 执行者
)

// TeamState represents the state of a team
type TeamState string

const (
	TeamStateCreated   TeamState = "created"
	TeamStateActive    TeamState = "active"
	TeamStatePaused    TeamState = "paused"
	TeamStateCompleted TeamState = "completed"
	TeamStateFailed    TeamState = "failed"
)

// TaskPriority represents task priority
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityMedium TaskPriority = 2
	PriorityHigh   TaskPriority = 3
	PriorityCritical TaskPriority = 4
)

// TaskStatus represents task status
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusAssigned  TaskStatus = "assigned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Message represents a message between agents
type Message struct {
	ID          string                 `json:"id"`
	FromAgent   string                 `json:"from_agent"`
	ToAgent     string                 `json:"to_agent,omitempty"` // Empty = broadcast
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Read        bool                   `json:"read"`
}

// Agent represents a team member agent
type Agent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Role         AgentRole              `json:"role"`
	Capabilities []string               `json:"capabilities"`
	Status       string                 `json:"status"`
	CurrentTask  string                 `json:"current_task,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActive   time.Time              `json:"last_active"`
}

// Task represents a task in the team
type Task struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    TaskPriority           `json:"priority"`
	Status      TaskStatus             `json:"status"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Dependencies []string              `json:"dependencies,omitempty"`
	Subtasks    []string               `json:"subtasks,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// Team represents a team of agents
type Team struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	State       TeamState         `json:"state"`
	Agents      map[string]*Agent `json:"agents"`
	Tasks       map[string]*Task  `json:"tasks"`
	Messages    []Message         `json:"messages"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TeamManager manages teams
type TeamManager struct {
	teams   map[string]*Team
	agents  map[string]*Agent
	mu      sync.RWMutex
}

// NewTeamManager creates a new team manager
func NewTeamManager() *TeamManager {
	return &TeamManager{
		teams:  make(map[string]*Team),
		agents: make(map[string]*Agent),
	}
}

// CreateTeam creates a new team
func (m *TeamManager) CreateTeam(name, description string) *Team {
	m.mu.Lock()
	defer m.mu.Unlock()

	team := &Team{
		ID:          generateTeamID(),
		Name:        name,
		Description: description,
		State:       TeamStateCreated,
		Agents:      make(map[string]*Agent),
		Tasks:       make(map[string]*Task),
		Messages:    []Message{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	m.teams[team.ID] = team
	return team
}

// GetTeam gets a team by ID
func (m *TeamManager) GetTeam(teamID string) (*Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, ok := m.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}
	return team, nil
}

// DeleteTeam deletes a team
func (m *TeamManager) DeleteTeam(teamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.teams[teamID]; !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	delete(m.teams, teamID)
	return nil
}

// ListTeams lists all teams
func (m *TeamManager) ListTeams() []*Team {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teams := make([]*Team, 0, len(m.teams))
	for _, team := range m.teams {
		teams = append(teams, team)
	}
	return teams
}

// AddAgent adds an agent to a team
func (m *TeamManager) AddAgent(teamID string, agent *Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	if agent.ID == "" {
		agent.ID = generateAgentID()
	}
	agent.CreatedAt = time.Now()
	agent.LastActive = time.Now()

	team.Agents[agent.ID] = agent
	m.agents[agent.ID] = agent
	team.UpdatedAt = time.Now()

	return nil
}

// RemoveAgent removes an agent from a team
func (m *TeamManager) RemoveAgent(teamID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	if _, ok := team.Agents[agentID]; !ok {
		return fmt.Errorf("agent not in team: %s", agentID)
	}

	delete(team.Agents, agentID)
	delete(m.agents, agentID)
	team.UpdatedAt = time.Now()

	return nil
}

// GetAgent gets an agent by ID
func (m *TeamManager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	return agent, nil
}

// CreateTask creates a new task
func (m *TeamManager) CreateTask(teamID string, task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	if task.ID == "" {
		task.ID = generateTaskID()
	}
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	team.Tasks[task.ID] = task
	team.UpdatedAt = time.Now()

	return nil
}

// AssignTask assigns a task to an agent
func (m *TeamManager) AssignTask(teamID, taskID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	task, ok := team.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	agent, ok := team.Agents[agentID]
	if !ok {
		return fmt.Errorf("agent not in team: %s", agentID)
	}

	task.AssignedTo = agentID
	task.Status = TaskStatusAssigned
	agent.CurrentTask = taskID
	team.UpdatedAt = time.Now()

	return nil
}

// StartTask starts a task
func (m *TeamManager) StartTask(teamID, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	task, ok := team.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	team.UpdatedAt = time.Now()

	return nil
}

// CompleteTask completes a task
func (m *TeamManager) CompleteTask(teamID, taskID, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	task, ok := team.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = TaskStatusCompleted
	task.Result = result
	now := time.Now()
	task.CompletedAt = &now

	// Clear agent's current task
	if task.AssignedTo != "" {
		if agent, ok := team.Agents[task.AssignedTo]; ok {
			agent.CurrentTask = ""
		}
	}

	team.UpdatedAt = time.Now()

	return nil
}

// FailTask marks a task as failed
func (m *TeamManager) FailTask(teamID, taskID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	task, ok := team.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = TaskStatusFailed
	task.Error = errMsg
	now := time.Now()
	task.CompletedAt = &now

	// Clear agent's current task
	if task.AssignedTo != "" {
		if agent, ok := team.Agents[task.AssignedTo]; ok {
			agent.CurrentTask = ""
		}
	}

	team.UpdatedAt = time.Now()

	return nil
}

// GetTasks gets all tasks for a team
func (m *TeamManager) GetTasks(teamID string) ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, ok := m.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	tasks := make([]*Task, 0, len(team.Tasks))
	for _, task := range team.Tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetTask gets a specific task
func (m *TeamManager) GetTask(teamID, taskID string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, ok := m.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	task, ok := team.Tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// SendMessage sends a message between agents
func (m *TeamManager) SendMessage(teamID string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	if msg.ID == "" {
		msg.ID = generateMessageID()
	}
	msg.Timestamp = time.Now()
	msg.Read = false

	team.Messages = append(team.Messages, msg)
	team.UpdatedAt = time.Now()

	return nil
}

// Broadcast sends a message to all agents in a team
func (m *TeamManager) Broadcast(teamID, fromAgent, msgType, content string) error {
	return m.SendMessage(teamID, Message{
		FromAgent: fromAgent,
		Type:      msgType,
		Content:   content,
	})
}

// GetMessages gets messages for a team
func (m *TeamManager) GetMessages(teamID string, agentID string, unreadOnly bool) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, ok := m.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	var messages []Message
	for _, msg := range team.Messages {
		// Filter by recipient
		if agentID != "" && msg.ToAgent != "" && msg.ToAgent != agentID {
			continue
		}

		// Filter by read status
		if unreadOnly && msg.Read {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// MarkMessageRead marks a message as read
func (m *TeamManager) MarkMessageRead(teamID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	for i := range team.Messages {
		if team.Messages[i].ID == messageID {
			team.Messages[i].Read = true
			return nil
		}
	}

	return fmt.Errorf("message not found: %s", messageID)
}

// SetTeamState sets the team state
func (m *TeamManager) SetTeamState(teamID string, state TeamState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	team.State = state
	team.UpdatedAt = time.Now()

	return nil
}

// GetTeamStats gets statistics for a team
func (m *TeamManager) GetTeamStats(teamID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, ok := m.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	stats := map[string]interface{}{
		"team_id":        team.ID,
		"team_name":      team.Name,
		"state":          team.State,
		"agent_count":    len(team.Agents),
		"task_count":     len(team.Tasks),
		"message_count":  len(team.Messages),
	}

	// Task statistics
	taskStats := map[string]int{
		"pending":   0,
		"assigned":  0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	for _, task := range team.Tasks {
		taskStats[string(task.Status)]++
	}
	stats["task_stats"] = taskStats

	// Agent statistics
	activeAgents := 0
	for _, agent := range team.Agents {
		if agent.CurrentTask != "" {
			activeAgents++
		}
	}
	stats["active_agents"] = activeAgents

	return stats, nil
}

// Coordinate distributes tasks to agents based on role and capability
func (m *TeamManager) Coordinate(teamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, ok := m.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}

	// Find pending tasks
	var pendingTasks []*Task
	for _, task := range team.Tasks {
		if task.Status == TaskStatusPending {
			pendingTasks = append(pendingTasks, task)
		}
	}

	// Find available agents
	var availableAgents []*Agent
	for _, agent := range team.Agents {
		if agent.CurrentTask == "" {
			availableAgents = append(availableAgents, agent)
		}
	}

	// Assign tasks to available agents
	for i, task := range pendingTasks {
		if i >= len(availableAgents) {
			break
		}

		agent := availableAgents[i]
		task.AssignedTo = agent.ID
		task.Status = TaskStatusAssigned
		agent.CurrentTask = task.ID
	}

	team.UpdatedAt = time.Now()
	return nil
}

// Helper functions

func generateTeamID() string {
	return fmt.Sprintf("team-%d", time.Now().UnixNano())
}

func generateAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

// TeamTool provides a tool interface for team operations
type TeamTool struct {
	manager *TeamManager
}

// NewTeamTool creates a new team tool
func NewTeamTool(manager *TeamManager) *TeamTool {
	return &TeamTool{manager: manager}
}

// Name returns the tool name
func (t *TeamTool) Name() string {
	return "Team"
}

// Description returns the tool description
func (t *TeamTool) Description() string {
	return "Manage multi-agent teams"
}

// InputSchema returns the input schema
func (t *TeamTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"create_team", "delete_team", "add_agent", "remove_agent", "create_task", "assign_task", "send_message", "get_stats"},
				"description": "Action to perform",
			},
			"team_id": map[string]interface{}{
				"type":        "string",
				"description": "Team ID",
			},
			"team_name": map[string]interface{}{
				"type":        "string",
				"description": "Team name",
			},
			"agent": map[string]interface{}{
				"type":        "object",
				"description": "Agent configuration",
			},
			"task": map[string]interface{}{
				"type":        "object",
				"description": "Task configuration",
			},
			"message": map[string]interface{}{
				"type":        "object",
				"description": "Message to send",
			},
		},
		"required": []string{"action"},
	}
}

// Execute executes the tool
func (t *TeamTool) Execute(ctx context.Context, input []byte) (interface{}, error) {
	var req struct {
		Action    string          `json:"action"`
		TeamID    string          `json:"team_id,omitempty"`
		TeamName  string          `json:"team_name,omitempty"`
		AgentID   string          `json:"agent_id,omitempty"`
		TaskID    string          `json:"task_id,omitempty"`
		Agent     json.RawMessage `json:"agent,omitempty"`
		Task      json.RawMessage `json:"task,omitempty"`
		Message   json.RawMessage `json:"message,omitempty"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	switch req.Action {
	case "create_team":
		team := t.manager.CreateTeam(req.TeamName, "")
		return team, nil

	case "delete_team":
		err := t.manager.DeleteTeam(req.TeamID)
		return map[string]bool{"success": err == nil}, err

	case "add_agent":
		var agent Agent
		if err := json.Unmarshal(req.Agent, &agent); err != nil {
			return nil, err
		}
		err := t.manager.AddAgent(req.TeamID, &agent)
		return &agent, err

	case "remove_agent":
		err := t.manager.RemoveAgent(req.TeamID, req.AgentID)
		return map[string]bool{"success": err == nil}, err

	case "create_task":
		var task Task
		if err := json.Unmarshal(req.Task, &task); err != nil {
			return nil, err
		}
		err := t.manager.CreateTask(req.TeamID, &task)
		return &task, err

	case "assign_task":
		err := t.manager.AssignTask(req.TeamID, req.TaskID, req.AgentID)
		return map[string]bool{"success": err == nil}, err

	case "send_message":
		var msg Message
		if err := json.Unmarshal(req.Message, &msg); err != nil {
			return nil, err
		}
		err := t.manager.SendMessage(req.TeamID, msg)
		return &msg, err

	case "get_stats":
		return t.manager.GetTeamStats(req.TeamID)

	default:
		return nil, fmt.Errorf("unknown action: %s", req.Action)
	}
}
