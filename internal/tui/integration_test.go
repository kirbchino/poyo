package tui

import (
	"testing"
)

// IntegrationTestSimulator simulates the full TUI message flow
// without requiring a real terminal

// TestIntegration_FullMessageFlow simulates a complete user interaction
func TestIntegration_FullMessageFlow(t *testing.T) {
	t.Log("=== Integration Test: Full Message Flow ===")

	// 1. Initialize the harness (simulates TUI startup)
	harness := NewTestHarness(t)
	t.Log("Step 1: TUI initialized")

	// Verify initial state
	harness.AssertMessageCount(0)
	if harness.Render() == "" {
		t.Error("Initial view should not be empty")
	}
	t.Log("Step 2: Initial state verified - no messages, view exists")

	// 2. Simulate window size event (first event TUI receives)
	harness.SendWindowSize(120, 40)
	t.Log("Step 3: Window size set to 120x40")

	// Verify dimensions propagated
	if harness.model.messages.width != 90 { // 120 * 3/4
		t.Errorf("Messages width should be 90, got %d", harness.model.messages.width)
	}
	t.Log("Step 4: Dimensions propagated correctly")

	// 3. Set up a realistic message handler
	responseCount := 0
	harness.SetOnMessage(func(content string) (string, error) {
		responseCount++
		return simulateAIResponse(content, responseCount), nil
	})
	t.Log("Step 5: Message handler configured")

	// 4. Simulate user typing and sending a message
	userMessage := "Hello, can you help me with Go programming?"
	harness.SendUserMessage(userMessage)
	t.Logf("Step 6: User message sent: %q", userMessage)

	// Verify user message added
	harness.AssertMessageCount(2) // user + assistant response
	harness.AssertMessagesContain(userMessage)
	harness.AssertMessagesContain("Poyo") // AI should identify as Poyo
	t.Log("Step 7: User message and AI response verified")

	// 5. Send another message
	secondMessage := "What is a closure in Go?"
	harness.SendUserMessage(secondMessage)
	t.Logf("Step 8: Second user message sent: %q", secondMessage)

	// Verify conversation grows
	harness.AssertMessageCount(4) // 2 user + 2 assistant
	t.Log("Step 9: Conversation has 4 messages")

	// 6. Test message ordering
	messages := harness.GetMessages()
	for i, msg := range messages {
		t.Logf("  Message %d: [%s] %s...", i, msg.Role, truncate(msg.Content, 30))
	}

	// 7. Render and verify output
	view := harness.RenderMessages()
	if view == "" {
		t.Error("Messages view should not be empty")
	}
	if len(view) < 100 {
		t.Errorf("Messages view seems too short: %d chars", len(view))
	}
	t.Logf("Step 10: Final view rendered (%d chars)", len(view))

	t.Log("=== Integration Test PASSED ===")
}

// TestIntegration_MessagePersistenceAcrossUpdates simulates the bug scenario
func TestIntegration_MessagePersistenceAcrossUpdates(t *testing.T) {
	t.Log("=== Integration Test: Message Persistence ===")

	harness := NewTestHarness(t)
	harness.SendWindowSize(80, 24)

	// Add a message
	harness.SendUserMessage("Test message")
	harness.AssertMessageCount(1)
	initialContent := harness.GetMessages()[0].Content

	// Simulate various tea.Msg updates that happen in normal operation
	// These should NOT affect the message content

	// 1. Key press (user typing in input)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}}
	newModel, _ := harness.model.Update(keyMsg)
	harness.model = newModel.(Model)

	// Message should still exist with same content
	harness.AssertMessageCount(1)
	if harness.GetMessages()[0].Content != initialContent {
		t.Error("Message content changed after key press!")
	}
	t.Log("✅ Message persisted after key press")

	// 2. Window resize
	harness.SendWindowSize(100, 30)
	harness.AssertMessageCount(1)
	if harness.GetMessages()[0].Content != initialContent {
		t.Error("Message content changed after resize!")
	}
	t.Log("✅ Message persisted after resize")

	// 3. Multiple rapid updates
	for i := 0; i < 10; i++ {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := harness.model.Update(keyMsg)
		harness.model = newModel.(Model)
	}
	harness.AssertMessageCount(1)
	if harness.GetMessages()[0].Content != initialContent {
		t.Error("Message content changed after rapid updates!")
	}
	t.Log("✅ Message persisted after rapid updates")

	t.Log("=== Message Persistence Test PASSED ===")
}

// TestIntegration_EmptyContentDebug simulates the "empty message" bug
func TestIntegration_EmptyContentDebug(t *testing.T) {
	t.Log("=== Integration Test: Empty Content Debug ===")

	harness := NewTestHarness(t)
	harness.SendWindowSize(80, 24)

	// Track what was sent vs what was stored
	sentMessage := "This is my message"
	harness.SendUserMessage(sentMessage)

	// Get what was actually stored
	messages := harness.GetMessages()
	if len(messages) == 0 {
		t.Fatal("No messages stored!")
	}

	storedContent := messages[0].Content

	t.Logf("Sent:     %q (len=%d)", sentMessage, len(sentMessage))
	t.Logf("Stored:   %q (len=%d)", storedContent, len(storedContent))

	if storedContent != sentMessage {
		t.Errorf("MISMATCH! Sent %q but stored %q", sentMessage, storedContent)
	}

	if storedContent == "" {
		t.Error("BUG DETECTED: Stored content is empty!")
	}

	// Check if view contains the content
	view := harness.RenderMessages()
	if storedContent != "" && !containsString(view, storedContent) {
		t.Errorf("View does not contain stored message!\nView length: %d", len(view))
	}

	t.Log("=== Empty Content Debug Test PASSED ===")
}

// TestIntegration_LayoutUpdateTiming tests layout update timing
func TestIntegration_LayoutUpdateTiming(t *testing.T) {
	t.Log("=== Integration Test: Layout Update Timing ===")

	harness := NewTestHarness(t)

	// Scenario: Message arrives BEFORE window size is set
	// This can happen in some terminal setups

	// Send message without setting window size first
	// (harness initializes with default size, but let's test)
	harness.SendUserMessage("Message before resize")

	// Message should still be stored
	harness.AssertMessageCount(1)

	// Now resize
	harness.SendWindowSize(100, 30)

	// Message should still exist
	harness.AssertMessageCount(1)
	harness.AssertMessagesContain("Message before resize")

	t.Log("✅ Messages persist even when sent before resize")

	t.Log("=== Layout Update Timing Test PASSED ===")
}

// Helper functions

func simulateAIResponse(userMessage string, responseNum int) string {
	responses := []string{
		"你好！我是 Poyo（波波），一个友好、智能的 AI 助手。很高兴认识你！有什么我可以帮助你的吗？",
		"在 Go 语言中，闭包（closure）是一个函数值，它引用了其外部作用域中的变量。闭包可以访问和修改其外部定义的变量，即使外部函数已经返回。",
	}
	if responseNum <= len(responses) {
		return responses[responseNum-1]
	}
	return "I understand your question. Let me help you with that."
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
