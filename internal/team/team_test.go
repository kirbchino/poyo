package team

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAgentRole(t *testing.T) {
	roles := []AgentRole{
		RoleCoordinator,
		RoleWorker,
		RoleReviewer,
		RoleValidator,
		RoleResearcher,
		RoleExecutor,
	}

	for _, role := range roles {
		if role == "" {
			t.Error("AgentRole should not be empty")
		}
	}
}

func TestTeamState(t *testing.T) {
	states := []TeamState{
		TeamStateCreated,
		TeamStateActive,
		TeamStatePaused,
		TeamStateCompleted,
		TeamStateFailed,
	}

	for _, state := range states {
		if state == "" {
			t.Error("TeamState should not be empty")
		}
	}
}

func TestTaskPriority(t *testing.T) {
	priorities := []TaskPriority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityCritical,
	}

	for i, p := range priorities {
		if int(p) != i+1 {
			t.Errorf("Priority value = %d, want %d", p, i+1)
		}
	}
}

func TestTaskStatus(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusAssigned,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("TaskStatus should not be empty")
		}
	}
}

func TestNewTeamManager(t *testing.T) {
	m := NewTeamManager()

	if m == nil {
		t.Fatal("NewTeamManager() returned nil")
	}

	if m.teams == nil {
		t.Error("teams should be initialized")
	}

	if m.agents == nil {
		t.Error("agents should be initialized")
	}
}

func TestCreateTeam(t *testing.T) {
	m := NewTeamManager()

	team := m.CreateTeam("Test Team", "Test description")

	if team == nil {
		t.Fatal("CreateTeam() returned nil")
	}

	if team.ID == "" {
		t.Error("Team should have ID")
	}

	if team.Name != "Test Team" {
		t.Errorf("Name = %q, want 'Test Team'", team.Name)
	}

	if team.State != TeamStateCreated {
		t.Errorf("State = %v, want %v", team.State, TeamStateCreated)
	}

	if team.Agents == nil {
		t.Error("Agents should be initialized")
	}

	if team.Tasks == nil {
		t.Error("Tasks should be initialized")
	}
}

func TestGetTeam(t *testing.T) {
	m := NewTeamManager()
	created := m.CreateTeam("Test Team", "")

	team, err := m.GetTeam(created.ID)
	if err != nil {
		t.Fatalf("GetTeam() error: %v", err)
	}

	if team.ID != created.ID {
		t.Error("GetTeam() returned wrong team")
	}

	_, err = m.GetTeam("nonexistent")
	if err == nil {
		t.Error("GetTeam() should return error for nonexistent team")
	}
}

func TestDeleteTeam(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	err := m.DeleteTeam(team.ID)
	if err != nil {
		t.Fatalf("DeleteTeam() error: %v", err)
	}

	_, err = m.GetTeam(team.ID)
	if err == nil {
		t.Error("Team should be deleted")
	}

	err = m.DeleteTeam("nonexistent")
	if err == nil {
		t.Error("DeleteTeam() should return error for nonexistent team")
	}
}

func TestListTeams(t *testing.T) {
	m := NewTeamManager()

	m.CreateTeam("Team 1", "")
	m.CreateTeam("Team 2", "")
	m.CreateTeam("Team 3", "")

	teams := m.ListTeams()
	if len(teams) != 3 {
		t.Errorf("ListTeams() count = %d, want 3", len(teams))
	}
}

func TestAddAgent(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{
		Name:         "Agent 1",
		Role:         RoleWorker,
		Capabilities: []string{"code", "test"},
	}

	err := m.AddAgent(team.ID, agent)
	if err != nil {
		t.Fatalf("AddAgent() error: %v", err)
	}

	if agent.ID == "" {
		t.Error("Agent should have ID")
	}

	if _, ok := team.Agents[agent.ID]; !ok {
		t.Error("Agent should be in team")
	}

	// Add to nonexistent team
	err = m.AddAgent("nonexistent", agent)
	if err == nil {
		t.Error("AddAgent() should return error for nonexistent team")
	}
}

func TestRemoveAgent(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{Name: "Agent 1", Role: RoleWorker}
	m.AddAgent(team.ID, agent)

	err := m.RemoveAgent(team.ID, agent.ID)
	if err != nil {
		t.Fatalf("RemoveAgent() error: %v", err)
	}

	if _, ok := team.Agents[agent.ID]; ok {
		t.Error("Agent should be removed from team")
	}

	// Remove nonexistent agent
	err = m.RemoveAgent(team.ID, "nonexistent")
	if err == nil {
		t.Error("RemoveAgent() should return error for nonexistent agent")
	}
}

func TestGetAgent(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{Name: "Agent 1", Role: RoleWorker}
	m.AddAgent(team.ID, agent)

	got, err := m.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("GetAgent() error: %v", err)
	}

	if got.ID != agent.ID {
		t.Error("GetAgent() returned wrong agent")
	}

	_, err = m.GetAgent("nonexistent")
	if err == nil {
		t.Error("GetAgent() should return error for nonexistent agent")
	}
}

func TestCreateTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	task := &Task{
		Title:       "Task 1",
		Description: "Test task",
		Priority:    PriorityHigh,
	}

	err := m.CreateTask(team.ID, task)
	if err != nil {
		t.Fatalf("CreateTask() error: %v", err)
	}

	if task.ID == "" {
		t.Error("Task should have ID")
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusPending)
	}

	if _, ok := team.Tasks[task.ID]; !ok {
		t.Error("Task should be in team")
	}
}

func TestAssignTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{Name: "Agent 1", Role: RoleWorker}
	m.AddAgent(team.ID, agent)

	task := &Task{Title: "Task 1"}
	m.CreateTask(team.ID, task)

	err := m.AssignTask(team.ID, task.ID, agent.ID)
	if err != nil {
		t.Fatalf("AssignTask() error: %v", err)
	}

	if task.AssignedTo != agent.ID {
		t.Error("Task should be assigned to agent")
	}

	if task.Status != TaskStatusAssigned {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusAssigned)
	}

	if agent.CurrentTask != task.ID {
		t.Error("Agent's current task should be set")
	}
}

func TestStartTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	task := &Task{Title: "Task 1"}
	m.CreateTask(team.ID, task)

	err := m.StartTask(team.ID, task.ID)
	if err != nil {
		t.Fatalf("StartTask() error: %v", err)
	}

	if task.Status != TaskStatusRunning {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusRunning)
	}

	if task.StartedAt == nil {
		t.Error("StartedAt should be set")
	}
}

func TestCompleteTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{Name: "Agent 1", Role: RoleWorker}
	m.AddAgent(team.ID, agent)

	task := &Task{Title: "Task 1"}
	m.CreateTask(team.ID, task)
	m.AssignTask(team.ID, task.ID, agent.ID)
	m.StartTask(team.ID, task.ID)

	err := m.CompleteTask(team.ID, task.ID, "Task completed successfully")
	if err != nil {
		t.Fatalf("CompleteTask() error: %v", err)
	}

	if task.Status != TaskStatusCompleted {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusCompleted)
	}

	if task.Result != "Task completed successfully" {
		t.Error("Result should be set")
	}

	if task.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	if agent.CurrentTask != "" {
		t.Error("Agent's current task should be cleared")
	}
}

func TestFailTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	agent := &Agent{Name: "Agent 1", Role: RoleWorker}
	m.AddAgent(team.ID, agent)

	task := &Task{Title: "Task 1"}
	m.CreateTask(team.ID, task)
	m.AssignTask(team.ID, task.ID, agent.ID)
	m.StartTask(team.ID, task.ID)

	err := m.FailTask(team.ID, task.ID, "Something went wrong")
	if err != nil {
		t.Fatalf("FailTask() error: %v", err)
	}

	if task.Status != TaskStatusFailed {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusFailed)
	}

	if task.Error != "Something went wrong" {
		t.Error("Error should be set")
	}
}

func TestGetTasks(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	m.CreateTask(team.ID, &Task{Title: "Task 1"})
	m.CreateTask(team.ID, &Task{Title: "Task 2"})
	m.CreateTask(team.ID, &Task{Title: "Task 3"})

	tasks, err := m.GetTasks(team.ID)
	if err != nil {
		t.Fatalf("GetTasks() error: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Tasks count = %d, want 3", len(tasks))
	}
}

func TestGetTask(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	created := &Task{Title: "Task 1"}
	m.CreateTask(team.ID, created)

	task, err := m.GetTask(team.ID, created.ID)
	if err != nil {
		t.Fatalf("GetTask() error: %v", err)
	}

	if task.ID != created.ID {
		t.Error("GetTask() returned wrong task")
	}
}

func TestSendMessage(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	msg := Message{
		FromAgent: "agent-1",
		ToAgent:   "agent-2",
		Type:      "task_update",
		Content:   "Task completed",
	}

	err := m.SendMessage(team.ID, msg)
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	if msg.ID == "" {
		t.Error("Message should have ID")
	}

	if len(team.Messages) != 1 {
		t.Error("Message should be in team")
	}
}

func TestBroadcast(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	err := m.Broadcast(team.ID, "coordinator", "announcement", "Meeting at 3pm")
	if err != nil {
		t.Fatalf("Broadcast() error: %v", err)
	}

	messages, _ := m.GetMessages(team.ID, "", false)
	if len(messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(messages))
	}

	if messages[0].ToAgent != "" {
		t.Error("Broadcast message should have empty ToAgent")
	}
}

func TestGetMessages(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	m.SendMessage(team.ID, Message{FromAgent: "a1", ToAgent: "a2", Type: "msg1"})
	m.SendMessage(team.ID, Message{FromAgent: "a1", ToAgent: "a3", Type: "msg2"})
	m.SendMessage(team.ID, Message{FromAgent: "a2", ToAgent: "a2", Type: "msg3"})

	// Get all messages
	messages, err := m.GetMessages(team.ID, "", false)
	if err != nil {
		t.Fatalf("GetMessages() error: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(messages))
	}

	// Get messages for specific agent
	messages, _ = m.GetMessages(team.ID, "a2", false)
	if len(messages) != 2 {
		t.Errorf("Messages for a2 = %d, want 2", len(messages))
	}
}

func TestMarkMessageRead(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	m.SendMessage(team.ID, Message{FromAgent: "a1", Type: "test"})
	msg := team.Messages[0]

	if msg.Read {
		t.Error("Message should start unread")
	}

	err := m.MarkMessageRead(team.ID, msg.ID)
	if err != nil {
		t.Fatalf("MarkMessageRead() error: %v", err)
	}

	if !team.Messages[0].Read {
		t.Error("Message should be read")
	}
}

func TestSetTeamState(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	err := m.SetTeamState(team.ID, TeamStateActive)
	if err != nil {
		t.Fatalf("SetTeamState() error: %v", err)
	}

	if team.State != TeamStateActive {
		t.Errorf("State = %v, want %v", team.State, TeamStateActive)
	}
}

func TestGetTeamStats(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	// Add agents
	m.AddAgent(team.ID, &Agent{Name: "Agent 1", Role: RoleWorker})
	m.AddAgent(team.ID, &Agent{Name: "Agent 2", Role: RoleReviewer})

	// Add tasks
	m.CreateTask(team.ID, &Task{Title: "Task 1"})
	task2 := &Task{Title: "Task 2"}
	m.CreateTask(team.ID, task2)
	m.StartTask(team.ID, task2.ID)

	// Send messages
	m.Broadcast(team.ID, "coord", "test", "test")

	stats, err := m.GetTeamStats(team.ID)
	if err != nil {
		t.Fatalf("GetTeamStats() error: %v", err)
	}

	if stats["agent_count"].(int) != 2 {
		t.Errorf("agent_count = %v, want 2", stats["agent_count"])
	}

	if stats["task_count"].(int) != 2 {
		t.Errorf("task_count = %v, want 2", stats["task_count"])
	}

	if stats["message_count"].(int) != 1 {
		t.Errorf("message_count = %v, want 1", stats["message_count"])
	}
}

func TestCoordinate(t *testing.T) {
	m := NewTeamManager()
	team := m.CreateTeam("Test Team", "")

	// Add agents
	m.AddAgent(team.ID, &Agent{Name: "Agent 1", Role: RoleWorker})
	m.AddAgent(team.ID, &Agent{Name: "Agent 2", Role: RoleWorker})

	// Add tasks
	m.CreateTask(team.ID, &Task{Title: "Task 1"})
	m.CreateTask(team.ID, &Task{Title: "Task 2"})
	m.CreateTask(team.ID, &Task{Title: "Task 3"})

	err := m.Coordinate(team.ID)
	if err != nil {
		t.Fatalf("Coordinate() error: %v", err)
	}

	// Check that tasks are assigned
	assignedCount := 0
	for _, task := range team.Tasks {
		if task.Status == TaskStatusAssigned {
			assignedCount++
		}
	}

	if assignedCount != 2 {
		t.Errorf("Assigned tasks = %d, want 2", assignedCount)
	}
}

func TestTeamTool(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	if tool.Name() != "Team" {
		t.Errorf("Name() = %q, want 'Team'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}
}

func TestTeamToolExecuteCreateTeam(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	input := `{
		"action": "create_team",
		"team_name": "Test Team"
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	team, ok := result.(*Team)
	if !ok {
		t.Fatal("Execute() should return *Team")
	}

	if team.Name != "Test Team" {
		t.Errorf("Name = %q, want 'Test Team'", team.Name)
	}
}

func TestTeamToolExecuteAddAgent(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	// Create team first
	team := m.CreateTeam("Test Team", "")

	input := `{
		"action": "add_agent",
		"team_id": "` + team.ID + `",
		"agent": {
			"name": "Agent 1",
			"role": "worker",
			"capabilities": ["code", "test"]
		}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	agent, ok := result.(*Agent)
	if !ok {
		t.Fatal("Execute() should return *Agent")
	}

	if agent.Name != "Agent 1" {
		t.Errorf("Name = %q, want 'Agent 1'", agent.Name)
	}
}

func TestTeamToolExecuteCreateTask(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	team := m.CreateTeam("Test Team", "")

	input := `{
		"action": "create_task",
		"team_id": "` + team.ID + `",
		"task": {
			"title": "Task 1",
			"description": "Test task",
			"priority": 3
		}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	task, ok := result.(*Task)
	if !ok {
		t.Fatal("Execute() should return *Task")
	}

	if task.Title != "Task 1" {
		t.Errorf("Title = %q, want 'Task 1'", task.Title)
	}
}

func TestTeamToolExecuteGetStats(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	team := m.CreateTeam("Test Team", "")
	m.AddAgent(team.ID, &Agent{Name: "Agent 1", Role: RoleWorker})

	input := `{
		"action": "get_stats",
		"team_id": "` + team.ID + `"
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	stats, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Execute() should return map")
	}

	if stats["agent_count"].(int) != 1 {
		t.Errorf("agent_count = %v, want 1", stats["agent_count"])
	}
}

func TestTeamToolExecuteInvalidAction(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	input := `{"action": "invalid"}`

	_, err := tool.Execute(context.Background(), []byte(input))
	if err == nil {
		t.Error("Execute() should return error for invalid action")
	}
}

func TestTeamToolExecuteInvalidJSON(t *testing.T) {
	m := NewTeamManager()
	tool := NewTeamTool(m)

	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}
}

func TestAgent(t *testing.T) {
	agent := Agent{
		ID:           "agent-1",
		Name:         "Test Agent",
		Role:         RoleWorker,
		Capabilities: []string{"code", "test"},
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	if agent.ID != "agent-1" {
		t.Errorf("ID = %q", agent.ID)
	}

	if agent.Role != RoleWorker {
		t.Errorf("Role = %v, want %v", agent.Role, RoleWorker)
	}
}

func TestTask(t *testing.T) {
	now := time.Now()
	task := Task{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Test description",
		Priority:    PriorityHigh,
		Status:      TaskStatusPending,
		CreatedAt:   now,
	}

	if task.ID != "task-1" {
		t.Errorf("ID = %q", task.ID)
	}

	if task.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", task.Priority, PriorityHigh)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		ID:        "msg-1",
		FromAgent: "agent-1",
		ToAgent:   "agent-2",
		Type:      "test",
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	if msg.ID != "msg-1" {
		t.Errorf("ID = %q", msg.ID)
	}

	if msg.FromAgent != "agent-1" {
		t.Errorf("FromAgent = %q", msg.FromAgent)
	}
}

func TestTeamJSON(t *testing.T) {
	team := Team{
		ID:        "team-1",
		Name:      "Test Team",
		State:     TeamStateActive,
		Agents:    make(map[string]*Agent),
		Tasks:     make(map[string]*Task),
		Messages:  []Message{},
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(team)
	if err != nil {
		t.Fatalf("Failed to marshal team: %v", err)
	}

	var parsed Team
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal team: %v", err)
	}

	if parsed.ID != team.ID {
		t.Error("Parsed team ID mismatch")
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewTeamManager()
	done := make(chan bool, 100)

	// Concurrent team creation
	for i := 0; i < 50; i++ {
		go func(idx int) {
			m.CreateTeam("Team", "")
			done <- true
		}(i)
	}

	// Concurrent team listing
	for i := 0; i < 50; i++ {
		go func(idx int) {
			m.ListTeams()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have 50 teams
	if len(m.ListTeams()) != 50 {
		t.Errorf("Team count = %d, want 50", len(m.ListTeams()))
	}
}
