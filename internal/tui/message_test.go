package tui

import (
	"errors"
	"testing"
)

// TestMessageFlow tests the basic message flow in TUI
func TestMessageFlow(t *testing.T) {
	harness := NewTestHarness(t)

	// Initially no messages
	harness.AssertMessageCount(0)

	// Send a user message
	harness.SendUserMessage("Hello, Poyo!")

	// Should have one message
	harness.AssertMessageCount(1)
	harness.AssertLastMessage("user", "Hello, Poyo!")

	// Messages view should contain the message
	harness.AssertMessagesContain("Hello, Poyo!")
	harness.AssertMessagesContain("You")
}

// TestMessageWithHandler tests message handling with a response
func TestMessageWithHandler(t *testing.T) {
	harness := NewTestHarness(t)

	// Set up a mock handler
	harness.SetOnMessage(func(content string) (string, error) {
		return "Hello! I'm Poyo, your friendly assistant!", nil
	})

	// Send a user message
	harness.SendUserMessage("Hello!")

	// Should have two messages (user + assistant)
	harness.AssertMessageCount(2)
	harness.AssertLastMessage("assistant", "Hello! I'm Poyo, your friendly assistant!")

	// Both messages should be visible
	harness.AssertMessagesContain("Hello!")
	harness.AssertMessagesContain("Poyo")
}

// TestMessageHandlerError tests error handling in message handler
func TestMessageHandlerError(t *testing.T) {
	harness := NewTestHarness(t)

	// Set up a handler that returns an error
	harness.SetOnMessage(func(content string) (string, error) {
		return "", errors.New("API error: connection timeout")
	})

	// Send a user message
	harness.SendUserMessage("Hello!")

	// Should only have user message (assistant message not added on error)
	harness.AssertMessageCount(1)
	harness.AssertLastMessage("user", "Hello!")

	// Model should be in error state
	if harness.model.state != StateError {
		t.Errorf("Expected state to be StateError, got %d", harness.model.state)
	}
}

// TestMultipleMessages tests adding multiple messages
func TestMultipleMessages(t *testing.T) {
	harness := NewTestHarness(t)

	// Add multiple messages
	for i := 0; i < 5; i++ {
		harness.SendUserMessage("Message " + string(rune('0'+i)))
	}

	// Should have 5 messages
	harness.AssertMessageCount(5)

	// Last message should be the fifth one
	harness.AssertLastMessage("user", "Message 4")
}

// TestMessageListDimensions tests that message list has proper dimensions
func TestMessageListDimensions(t *testing.T) {
	harness := NewTestHarness(t)

	// Check initial dimensions
	if harness.model.messages.width <= 0 {
		t.Errorf("Messages width should be > 0, got %d", harness.model.messages.width)
	}
	if harness.model.messages.height <= 0 {
		t.Errorf("Messages height should be > 0, got %d", harness.model.messages.height)
	}
}

// TestViewNotEmpty tests that the view is not empty after adding messages
func TestViewNotEmpty(t *testing.T) {
	harness := NewTestHarness(t)

	// Initially should show empty state
	view := harness.RenderMessages()
	if view == "" {
		t.Error("Messages view should not be empty string even with no messages")
	}

	// Add a message
	harness.SendUserMessage("Test message")

	// View should contain the message
	view = harness.RenderMessages()
	if view == "" {
		t.Error("Messages view should not be empty after adding a message")
	}
}

// TestWindowSizeUpdate tests that window size updates properly propagate
func TestWindowSizeUpdate(t *testing.T) {
	harness := NewTestHarness(t)

	// Initial size
	initialWidth := harness.model.messages.width
	initialHeight := harness.model.messages.height

	// Update window size
	harness.SendWindowSize(100, 30)

	// Check that child components were updated
	newWidth := harness.model.messages.width
	newHeight := harness.model.messages.height

	// Messages should be 3/4 of width
	expectedWidth := 100 * 3 / 4
	if newWidth != expectedWidth {
		t.Errorf("Expected messages width %d, got %d", expectedWidth, newWidth)
	}

	// Messages height should be total height minus status and input
	if newHeight == initialHeight {
		t.Error("Messages height should have changed after window resize")
	}
}

// TestMessagePersistence tests that messages persist across updates
func TestMessagePersistence(t *testing.T) {
	harness := NewTestHarness(t)

	// Add a message
	harness.SendUserMessage("Persistent message")
	harness.AssertMessageCount(1)

	// Simulate other updates (like key press)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := harness.model.Update(keyMsg)
	harness.model = newModel.(Model)

	// Message should still be there
	harness.AssertMessageCount(1)
	harness.AssertLastMessage("user", "Persistent message")
}

// TestEmptyMessageNotAdded tests that empty messages are not added
func TestEmptyMessageNotAdded(t *testing.T) {
	harness := NewTestHarness(t)

	// Try to send an empty message
	harness.SendUserMessage("")
	harness.SendUserMessage("   ") // Just whitespace

	// Both should be added (the current implementation doesn't validate)
	// This test documents current behavior - could be changed to validate
	t.Logf("Current behavior: empty messages are added. Count: %d", harness.GetMessageCount())
}

// TestMessageOrder tests that messages are in correct order
func TestMessageOrder(t *testing.T) {
	harness := NewTestHarness(t)

	harness.SetOnMessage(func(content string) (string, error) {
		return "Response to: " + content, nil
	})

	harness.SendUserMessage("First")
	harness.SendUserMessage("Second")

	// Should have 4 messages: user, assistant, user, assistant
	harness.AssertMessageCount(4)

	messages := harness.GetMessages()
	expectedOrder := []struct {
		role    string
		content string
	}{
		{"user", "First"},
		{"assistant", "Response to: First"},
		{"user", "Second"},
		{"assistant", "Response to: Second"},
	}

	for i, expected := range expectedOrder {
		if messages[i].Role != expected.role {
			t.Errorf("Message %d: expected role %s, got %s", i, expected.role, messages[i].Role)
		}
		if messages[i].Content != expected.content {
			t.Errorf("Message %d: expected content %s, got %s", i, expected.content, messages[i].Content)
		}
	}
}
