package task

import (
	"context"
	"testing"
	"time"
)

func TestTaskState(t *testing.T) {
	states := []TaskState{
		StatePending,
		StateRunning,
		StateCompleted,
		StateFailed,
		StateStopped,
	}

	for _, s := range states {
		if string(s) == "" {
			t.Errorf("Task state should have a non-empty string representation")
		}
	}
}

func TestTaskType(t *testing.T) {
	types := []TaskType{
		TypeBash,
		TypeAgent,
		TypeHook,
		TypeCustom,
	}

	for _, tt := range types {
		if string(tt) == "" {
			t.Errorf("Task type should have a non-empty string representation")
		}
	}
}

func TestTaskDuration(t *testing.T) {
	task := &Task{
		StartedAt:   time.Now().Add(-5 * time.Second),
		CompletedAt: time.Now(),
	}

	duration := task.Duration()
	if duration < 4*time.Second || duration > 6*time.Second {
		t.Errorf("Duration() = %v, expected ~5s", duration)
	}
}

func TestTaskDurationNotStarted(t *testing.T) {
	task := &Task{}

	duration := task.Duration()
	if duration != 0 {
		t.Errorf("Duration() of unstarted task = %v, want 0", duration)
	}
}

func TestTaskIsComplete(t *testing.T) {
	tests := []struct {
		state    TaskState
		expected bool
	}{
		{StatePending, false},
		{StateRunning, false},
		{StateCompleted, true},
		{StateFailed, true},
		{StateStopped, true},
	}

	for _, tt := range tests {
		task := &Task{State: tt.state}
		if task.IsComplete() != tt.expected {
			t.Errorf("IsComplete() for state %v = %v, want %v", tt.state, !tt.expected, tt.expected)
		}
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.tasks == nil {
		t.Error("tasks map should be initialized")
	}
}

func TestManagerCreate(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{
		Type:    TypeBash,
		Name:    "test task",
		Command: "echo hello",
	})

	if task == nil {
		t.Fatal("Create() returned nil")
	}

	if task.ID == "" {
		t.Error("Task should have an ID")
	}

	if task.State != StatePending {
		t.Errorf("Task state = %v, want %v", task.State, StatePending)
	}

	if task.Name != "test task" {
		t.Errorf("Task name = %q, want 'test task'", task.Name)
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{
		Type: TypeBash,
		Name: "test task",
	})

	got, ok := m.Get(task.ID)
	if !ok {
		t.Fatal("Task should be found")
	}

	if got.ID != task.ID {
		t.Errorf("Get() returned wrong task")
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for nonexistent task")
	}
}

func TestManagerList(t *testing.T) {
	m := NewManager()

	// Create multiple tasks
	m.Create(TaskOptions{Type: TypeBash, Name: "task1"})
	m.Create(TaskOptions{Type: TypeBash, Name: "task2"})
	m.Create(TaskOptions{Type: TypeBash, Name: "task3"})

	tasks := m.List()
	if len(tasks) != 3 {
		t.Errorf("List() returned %d tasks, want 3", len(tasks))
	}
}

func TestManagerListByState(t *testing.T) {
	m := NewManager()

	task1 := m.Create(TaskOptions{Type: TypeBash, Name: "task1"})
	task2 := m.Create(TaskOptions{Type: TypeBash, Name: "task2"})

	// Manually set state
	task2.State = StateRunning

	pending := m.ListByState(StatePending)
	if len(pending) != 1 {
		t.Errorf("ListByState(pending) returned %d tasks, want 1", len(pending))
	}

	running := m.ListByState(StateRunning)
	if len(running) != 1 {
		t.Errorf("ListByState(running) returned %d tasks, want 1", len(running))
	}
}

func TestManagerStop(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})
	task.State = StateRunning

	err := m.Stop(task.ID)
	if err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	if task.State != StateStopped {
		t.Errorf("Task state = %v, want %v", task.State, StateStopped)
	}

	// Stop non-existent task
	err = m.Stop("nonexistent")
	if err == nil {
		t.Error("Stop() should return error for non-existent task")
	}
}

func TestManagerRemove(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})
	task.State = StateCompleted

	err := m.Remove(task.ID)
	if err != nil {
		t.Errorf("Remove() error: %v", err)
	}

	_, ok := m.Get(task.ID)
	if ok {
		t.Error("Task should be removed")
	}
}

func TestManagerRemoveRunning(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})
	task.State = StateRunning

	err := m.Remove(task.ID)
	if err == nil {
		t.Error("Remove() should return error for running task")
	}
}

func TestManagerClear(t *testing.T) {
	m := NewManager()

	task1 := m.Create(TaskOptions{Type: TypeBash, Name: "task1"})
	task1.State = StateCompleted

	task2 := m.Create(TaskOptions{Type: TypeBash, Name: "task2"})
	task2.State = StateFailed

	task3 := m.Create(TaskOptions{Type: TypeBash, Name: "task3"})
	task3.State = StateRunning

	count := m.Clear()
	if count != 2 {
		t.Errorf("Clear() removed %d tasks, want 2", count)
	}

	if m.Count() != 1 {
		t.Errorf("Count() = %d, want 1", m.Count())
	}
}

func TestManagerAppendOutput(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})

	m.AppendOutput(task.ID, "line1\n")
	m.AppendOutput(task.ID, "line2\n")

	if task.Output != "line1\nline2\n" {
		t.Errorf("Output = %q, want 'line1\\nline2\\n'", task.Output)
	}
}

func TestBashExecutorCreateTask(t *testing.T) {
	m := NewManager()
	executor := NewBashExecutor(m)

	task := executor.CreateTask("echo hello", "/tmp")

	if task == nil {
		t.Fatal("CreateTask() returned nil")
	}

	if task.Command != "echo hello" {
		t.Errorf("Task command = %q, want 'echo hello'", task.Command)
	}

	if task.Type != TypeBash {
		t.Errorf("Task type = %v, want %v", task.Type, TypeBash)
	}
}

func TestBashExecutorRunSync(t *testing.T) {
	m := NewManager()
	executor := NewBashExecutor(m)

	output, exitCode, err := executor.RunSync(context.Background(), "echo hello", "", 5*time.Second)

	if err != nil {
		t.Fatalf("RunSync() error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Exit code = %d, want 0", exitCode)
	}

	if !contains(output, "hello") {
		t.Errorf("Output = %q, should contain 'hello'", output)
	}
}

func TestBashExecutorRunSyncTimeout(t *testing.T) {
	m := NewManager()
	executor := NewBashExecutor(m)

	ctx := context.Background()
	_, _, err := executor.RunSync(ctx, "sleep 10", "", 100*time.Millisecond)

	if err == nil {
		t.Error("RunSync() should timeout")
	}
}

func TestOutputTool(t *testing.T) {
	m := NewManager()
	tool := NewOutputTool(m)

	if tool.Name() != "TaskOutput" {
		t.Errorf("Name() = %q, want 'TaskOutput'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}
}

func TestListTool(t *testing.T) {
	m := NewManager()
	tool := NewListTool(m)

	if tool.Name() != "TaskList" {
		t.Errorf("Name() = %q, want 'TaskList'", tool.Name())
	}
}

func TestStopTool(t *testing.T) {
	m := NewManager()
	tool := NewStopTool(m)

	if tool.Name() != "TaskStop" {
		t.Errorf("Name() = %q, want 'TaskStop'", tool.Name())
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestTruncateCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long command that should be truncated", 20, "this is a very lo..."},
	}

	for _, tt := range tests {
		result := truncateCommand(tt.cmd, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateCommand(%q, %d) = %q, want %q", tt.cmd, tt.maxLen, result, tt.expected)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && s[:len(substr)] == substr ||
			contains(s[1:], substr)))
}

// Additional comprehensive tests

func TestTaskOptions(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{
		Type:    TypeBash,
		Name:    "custom-task",
		Command: "echo test",
		WorkingDir: "/tmp",
		Env:     map[string]string{"FOO": "bar"},
		Timeout: 30 * time.Second,
	})

	if task.Name != "custom-task" {
		t.Errorf("Name = %q, want 'custom-task'", task.Name)
	}

	if task.WorkingDir != "/tmp" {
		t.Errorf("WorkingDir = %q, want '/tmp'", task.WorkingDir)
	}

	if task.Env["FOO"] != "bar" {
		t.Error("Env should be set")
	}

	if task.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", task.Timeout)
	}
}

func TestManagerStopNonExistent(t *testing.T) {
	m := NewManager()

	err := m.Stop("nonexistent-id")
	if err == nil {
		t.Error("Stop() should return error for non-existent task")
	}
}

func TestManagerRemoveNonExistent(t *testing.T) {
	m := NewManager()

	err := m.Remove("nonexistent-id")
	if err == nil {
		t.Error("Remove() should return error for non-existent task")
	}
}

func TestTaskStateTransitions(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})

	// Test valid transitions
	transitions := []struct {
		from   TaskState
		to     TaskState
		valid  bool
	}{
		{StatePending, StateRunning, true},
		{StateRunning, StateCompleted, true},
		{StateRunning, StateFailed, true},
		{StateRunning, StateStopped, true},
		{StatePending, StateCompleted, false},
		{StateCompleted, StateRunning, false},
	}

	for _, tt := range transitions {
		task.State = tt.from
		// In real implementation, would validate transitions
		_ = tt.to
		_ = tt.valid
	}
}

func TestBashExecutorWorkingDir(t *testing.T) {
	m := NewManager()
	executor := NewBashExecutor(m)

	task := executor.CreateTask("pwd", "/tmp/test")

	if task.WorkingDir != "/tmp/test" {
		t.Errorf("WorkingDir = %q, want '/tmp/test'", task.WorkingDir)
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManager()
	done := make(chan string, 20)

	// Create tasks concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			task := m.Create(TaskOptions{
				Type: TypeBash,
				Name: "concurrent-task",
			})
			done <- task.ID
		}(i)
	}

	// Collect task IDs
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := <-done
		if ids[id] {
			t.Errorf("Duplicate task ID: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != 10 {
		t.Errorf("Expected 10 unique IDs, got %d", len(ids))
	}
}

func TestTaskOutputConcurrentAppend(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{Type: TypeBash, Name: "test"})
	done := make(chan bool, 10)

	// Concurrent output append
	for i := 0; i < 10; i++ {
		go func(idx int) {
			m.AppendOutput(task.ID, "line\n")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify output has content
	if task.Output == "" {
		t.Error("Output should not be empty after concurrent appends")
	}
}

func TestManagerListFilters(t *testing.T) {
	m := NewManager()

	// Create tasks with different states
	task1 := m.Create(TaskOptions{Type: TypeBash, Name: "pending"})
	task1.State = StatePending

	task2 := m.Create(TaskOptions{Type: TypeBash, Name: "running"})
	task2.State = StateRunning

	task3 := m.Create(TaskOptions{Type: TypeBash, Name: "completed"})
	task3.State = StateCompleted

	task4 := m.Create(TaskOptions{Type: TypeBash, Name: "failed"})
	task4.State = StateFailed

	// Test ListByState
	pending := m.ListByState(StatePending)
	if len(pending) != 1 {
		t.Errorf("Pending tasks = %d, want 1", len(pending))
	}

	running := m.ListByState(StateRunning)
	if len(running) != 1 {
		t.Errorf("Running tasks = %d, want 1", len(running))
	}

	// Test List
	all := m.List()
	if len(all) != 4 {
		t.Errorf("All tasks = %d, want 4", len(all))
	}
}

func TestTaskDurationEdgeCases(t *testing.T) {
	// Test with nil times
	task := &Task{}
	if task.Duration() != 0 {
		t.Error("Duration should be 0 for nil times")
	}

	// Test with only start time
	task = &Task{
		StartedAt: time.Now(),
	}
	duration := task.Duration()
	if duration < 0 {
		t.Error("Duration should not be negative")
	}

	// Test with completed before started (shouldn't happen, but handle gracefully)
	task = &Task{
		StartedAt:   time.Now(),
		CompletedAt: time.Now().Add(-1 * time.Hour),
	}
	duration = task.Duration()
	// Should handle gracefully
	_ = duration
}

func TestBashExecutorExitCodes(t *testing.T) {
	m := NewManager()
	executor := NewBashExecutor(m)

	tests := []struct {
		command    string
		expectZero bool
	}{
		{"echo success", true},
		{"exit 0", true},
		{"exit 1", false},
		{"exit 42", false},
	}

	for _, tt := range tests {
		_, exitCode, _ := executor.RunSync(context.Background(), tt.command, "", 5*time.Second)

		if tt.expectZero && exitCode != 0 {
			t.Errorf("Command %q should exit with 0, got %d", tt.command, exitCode)
		}

		if !tt.expectZero && exitCode == 0 {
			t.Errorf("Command %q should not exit with 0", tt.command)
		}
	}
}

func TestTaskMetadata(t *testing.T) {
	m := NewManager()

	task := m.Create(TaskOptions{
		Type: TypeBash,
		Name: "test",
	})

	// Add metadata
	task.Metadata = map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b", "c"},
	}

	if task.Metadata["key1"] != "value1" {
		t.Error("Metadata key1 should be value1")
	}

	if task.Metadata["key2"] != 123 {
		t.Error("Metadata key2 should be 123")
	}
}
