package compactor

import (
	"context"
	"testing"
	"time"
)

func TestCompressionStrategy(t *testing.T) {
	strategies := []CompressionStrategy{
		StrategySummarize,
		StrategyTruncate,
		StrategySemantic,
		StrategyHierarchical,
	}

	for _, s := range strategies {
		if s == "" {
			t.Error("Strategy should not be empty")
		}
	}
}

func TestMessageType(t *testing.T) {
	types := []MessageType{
		MessageTypeUser,
		MessageTypeAssistant,
		MessageTypeTool,
		MessageTypeSystem,
		MessageTypeSummary,
	}

	for _, mt := range types {
		if mt == "" {
			t.Error("MessageType should not be empty")
		}
	}
}

func TestDefaultCompressionConfig(t *testing.T) {
	config := DefaultCompressionConfig()

	if config == nil {
		t.Fatal("DefaultCompressionConfig() returned nil")
	}

	if config.Strategy != StrategySummarize {
		t.Errorf("Strategy = %v, want %v", config.Strategy, StrategySummarize)
	}

	if config.MaxTokens != 100000 {
		t.Errorf("MaxTokens = %d, want 100000", config.MaxTokens)
	}

	if config.TargetRatio != 0.3 {
		t.Errorf("TargetRatio = %v, want 0.3", config.TargetRatio)
	}
}

func TestNewSession(t *testing.T) {
	session := NewSession()

	if session == nil {
		t.Fatal("NewSession() returned nil")
	}

	if session.ID == "" {
		t.Error("Session should have ID")
	}

	if session.Messages == nil {
		t.Error("Messages should be initialized")
	}

	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestNewCompactor(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)

	if compactor == nil {
		t.Fatal("NewCompactor() returned nil")
	}

	if compactor.config == nil {
		t.Error("Config should be initialized with defaults")
	}
}

func TestCompactorAddMessage(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)
	session := NewSession()

	msg := Message{
		Type:    MessageTypeUser,
		Content: "Hello, world!",
	}

	compactor.AddMessage(session, msg)

	if len(session.Messages) != 1 {
		t.Errorf("Message count = %d, want 1", len(session.Messages))
	}

	if session.Messages[0].ID == "" {
		t.Error("Message should have ID")
	}

	if session.Messages[0].Timestamp.IsZero() {
		t.Error("Message should have timestamp")
	}
}

func TestCompactorAddMessageWithTokenizer(t *testing.T) {
	tokenizer := NewSimpleTokenizer()
	compactor := NewCompactor(nil, tokenizer, nil)
	session := NewSession()

	msg := Message{
		Type:    MessageTypeUser,
		Content: "Hello, world!",
	}

	compactor.AddMessage(session, msg)

	if session.Messages[0].TokenCount == 0 {
		t.Error("Message should have token count")
	}

	if session.TokenCount == 0 {
		t.Error("Session token count should be updated")
	}
}

func TestCompactorShouldCompact(t *testing.T) {
	config := &CompressionConfig{
		MaxTokens:            1000,
		MinMessagesToCompact: 3,
	}

	compactor := NewCompactor(config, nil, nil)
	session := NewSession()

	// Add few messages
	for i := 0; i < 2; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "test",
		})
	}

	if compactor.ShouldCompact(session) {
		t.Error("Should not compact with few messages")
	}

	// Add more messages
	for i := 0; i < 10; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "test content that adds tokens",
		})
	}

	// Should compact now
	if !compactor.ShouldCompact(session) {
		t.Error("Should compact with many messages")
	}
}

func TestCompactorCompactTruncate(t *testing.T) {
	config := &CompressionConfig{
		Strategy:           StrategyTruncate,
		MaxTokens:          1000,
		MaxSummaryLength:   1000,
		PreserveRecent:     2,
		MinMessagesToCompact: 5,
	}

	compactor := NewCompactor(config, nil, nil)
	session := NewSession()

	// Add messages
	for i := 0; i < 10; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "This is test message number " + string(rune('0'+i)),
		})
	}

	summary, err := compactor.Compact(context.Background(), session)

	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}

	if summary == nil {
		t.Fatal("Summary should not be nil")
	}

	if summary.ID == "" {
		t.Error("Summary should have ID")
	}

	if summary.Content == "" {
		t.Error("Summary should have content")
	}

	// Check that recent messages are preserved
	if len(session.Messages) != 3 { // 1 summary + 2 preserved
		t.Errorf("Message count = %d, want 3", len(session.Messages))
	}
}

func TestCompactorCompactNotEnoughMessages(t *testing.T) {
	config := &CompressionConfig{
		Strategy:           StrategyTruncate,
		PreserveRecent:     5,
		MinMessagesToCompact: 10,
	}

	compactor := NewCompactor(config, nil, nil)
	session := NewSession()

	// Add only 5 messages
	for i := 0; i < 5; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "test",
		})
	}

	_, err := compactor.Compact(context.Background(), session)

	if err == nil {
		t.Error("Compact() should return error for not enough messages")
	}
}

func TestSimpleTokenizer(t *testing.T) {
	tokenizer := NewSimpleTokenizer()

	tests := []struct {
		text     string
		minCount int
		maxCount int
	}{
		{"Hello", 1, 5},
		{"Hello, world!", 2, 10},
		{"This is a longer piece of text.", 5, 20},
	}

	for _, tt := range tests {
		count := tokenizer.CountTokens(tt.text)
		if count < tt.minCount || count > tt.maxCount {
			t.Errorf("CountTokens(%q) = %d, want between %d and %d",
				tt.text, count, tt.minCount, tt.maxCount)
		}
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		ID:        "msg-123",
		Type:      MessageTypeUser,
		Content:   "Test content",
		Timestamp: time.Now(),
		TokenCount: 10,
	}

	if msg.ID != "msg-123" {
		t.Errorf("ID = %q, want 'msg-123'", msg.ID)
	}

	if msg.Type != MessageTypeUser {
		t.Errorf("Type = %v, want %v", msg.Type, MessageTypeUser)
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := Message{
		ID:   "msg-123",
		Type: MessageTypeTool,
		ToolCalls: []ToolCall{
			{ID: "tc-1", Name: "Bash", Arguments: `{"command": "ls"}`},
			{ID: "tc-2", Name: "Read", Arguments: `{"path": "test.txt"}`},
		},
	}

	if len(msg.ToolCalls) != 2 {
		t.Errorf("ToolCalls count = %d, want 2", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].Name != "Bash" {
		t.Errorf("First tool = %q, want 'Bash'", msg.ToolCalls[0].Name)
	}
}

func TestSummary(t *testing.T) {
	summary := Summary{
		ID:             "sum-123",
		StartID:        "msg-1",
		EndID:          "msg-10",
		Content:        "This is a summary",
		TokenCount:     50,
		OriginalTokens: 500,
		CompressedAt:   time.Now(),
		KeyPoints:      []string{"Key point 1", "Key point 2"},
	}

	if summary.ID != "sum-123" {
		t.Errorf("ID = %q, want 'sum-123'", summary.ID)
	}

	if summary.OriginalTokens != 500 {
		t.Errorf("OriginalTokens = %d, want 500", summary.OriginalTokens)
	}

	if len(summary.KeyPoints) != 2 {
		t.Errorf("KeyPoints count = %d, want 2", len(summary.KeyPoints))
	}
}

func TestEntity(t *testing.T) {
	entity := Entity{
		Type:  "file",
		Name:  "main.go",
		Value: "/path/to/main.go",
	}

	if entity.Type != "file" {
		t.Errorf("Type = %q, want 'file'", entity.Type)
	}
}

func TestCompactorGetStatistics(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)
	session := NewSession()

	for i := 0; i < 5; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "test",
		})
	}

	stats := compactor.GetStatistics(session)

	if stats == nil {
		t.Fatal("GetStatistics() returned nil")
	}

	if stats["message_count"].(int) != 5 {
		t.Errorf("message_count = %v, want 5", stats["message_count"])
	}
}

func TestCompactorExportImport(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)
	session := NewSession()

	compactor.AddMessage(session, Message{
		Type:    MessageTypeUser,
		Content: "Hello",
	})

	// Export
	data, err := compactor.Export(session)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if len(data) == 0 {
		t.Error("Export() returned empty data")
	}

	// Import
	imported, err := compactor.Import(data)
	if err != nil {
		t.Fatalf("Import() error: %v", err)
	}

	if imported.ID != session.ID {
		t.Error("Imported session ID mismatch")
	}

	if len(imported.Messages) != len(session.Messages) {
		t.Error("Imported messages count mismatch")
	}
}

func TestCompactorHierarchicalCompress(t *testing.T) {
	config := &CompressionConfig{
		Strategy:           StrategyHierarchical,
		MaxSummaryLength:   1000,
		PreserveRecent:     2,
		MinMessagesToCompact: 5,
	}

	compactor := NewCompactor(config, nil, nil)
	session := NewSession()

	// Add messages
	for i := 0; i < 10; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "Test message",
		})
	}

	summary, err := compactor.Compact(context.Background(), session)

	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}

	if summary.Content == "" {
		t.Error("Summary should have content")
	}
}

func TestCompactorSemanticCompress(t *testing.T) {
	config := &CompressionConfig{
		Strategy:           StrategySemantic,
		MaxSummaryLength:   1000,
		PreserveRecent:     2,
		MinMessagesToCompact: 5,
	}

	compactor := NewCompactor(config, nil, nil)
	session := NewSession()

	// Add messages with tool calls
	for i := 0; i < 5; i++ {
		compactor.AddMessage(session, Message{
			Type:    MessageTypeUser,
			Content: "User request",
		})
		compactor.AddMessage(session, Message{
			Type: MessageTypeTool,
			ToolCalls: []ToolCall{
				{Name: "Bash", Arguments: `{"command": "test"}`},
			},
		})
	}

	summary, err := compactor.Compact(context.Background(), session)

	if err != nil {
		t.Fatalf("Compact() error: %v", err)
	}

	// Check for key points extraction
	if len(summary.KeyPoints) == 0 {
		t.Error("Should extract key points from tool calls")
	}
}

func TestCompactorRestore(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)
	session := NewSession()

	_, err := compactor.Restore(session, "nonexistent")

	if err == nil {
		t.Error("Restore() should return error (not implemented)")
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		text     string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long text", 10, "this is a ..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateText(tt.text, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateText(%q, %d) = %q, want %q",
				tt.text, tt.maxLen, result, tt.expected)
		}
	}
}

func TestChunkMessages(t *testing.T) {
	messages := make([]Message, 12)
	for i := range messages {
		messages[i] = Message{ID: string(rune('a' + i))}
	}

	chunks := chunkMessages(messages, 5)

	if len(chunks) != 3 {
		t.Errorf("Chunk count = %d, want 3", len(chunks))
	}

	// First two chunks should have 5 messages
	if len(chunks[0]) != 5 || len(chunks[1]) != 5 {
		t.Error("First two chunks should have 5 messages")
	}

	// Last chunk should have 2 messages
	if len(chunks[2]) != 2 {
		t.Errorf("Last chunk should have 2 messages, got %d", len(chunks[2]))
	}
}

func TestCompactorConcurrency(t *testing.T) {
	compactor := NewCompactor(nil, nil, nil)
	session := NewSession()
	done := make(chan bool, 100)

	// Concurrent message additions
	for i := 0; i < 50; i++ {
		go func(idx int) {
			compactor.AddMessage(session, Message{
				Type:    MessageTypeUser,
				Content: "concurrent test",
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	if len(session.Messages) != 50 {
		t.Errorf("Message count = %d, want 50", len(session.Messages))
	}
}
