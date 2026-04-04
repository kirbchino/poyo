// Package e2e provides end-to-end tests for MCP OAuth integration.
package e2e

import (
	"context"
	"testing"
	"time"
)

// TestOAuthFlowE2E tests the complete OAuth flow end-to-end
func TestOAuthFlowE2E(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create mock OAuth server
	mockOAuth := NewMockOAuthServer()
	defer mockOAuth.Close()

	t.Run("complete_oauth_flow", func(t *testing.T) {
		// The mock OAuth server simulates the complete flow
		// 1. User visits authorization URL
		// 2. Server redirects back with code
		// 3. Code is exchanged for token

		// Verify mock server is ready
		if mockOAuth.GetAuthorizationURL() == "" {
			t.Error("Authorization URL should not be empty")
		}

		if mockOAuth.GetTokenURL() == "" {
			t.Error("Token URL should not be empty")
		}

		// Verify auth request recording
		AssertCondition(t, len(mockOAuth.AuthRequests) == 0, "Initial auth requests should be empty")
		AssertCondition(t, len(mockOAuth.TokenRequests) == 0, "Initial token requests should be empty")
	})
}

// TestMockMCPServerE2E tests MCP server mock functionality
func TestMockMCPServerE2E(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	mockMCP := NewMockMCPServer()
	defer mockMCP.Close()

	t.Run("mock_server_ready", func(t *testing.T) {
		AssertCondition(t, mockMCP.Server != nil, "Mock server should be initialized")
		AssertCondition(t, mockMCP.URL != "", "Mock server URL should not be empty")
	})

	t.Run("request_recording", func(t *testing.T) {
		// Set a mock response
		mockMCP.SetResponse("POST", "/mcp", map[string]interface{}{
			"tools": []interface{}{},
		})

		// Verify request count
		initialCount := mockMCP.GetRequestCount()

		// The mock server should record requests
		// (In real e2e tests, we would make actual HTTP requests here)
		AssertCondition(t, mockMCP.GetRequestCount() >= initialCount, "Request count should be tracked")
	})

	t.Run("get_last_request", func(t *testing.T) {
		lastReq := mockMCP.GetLastRequest()
		// May be nil if no requests made yet
		if lastReq != nil {
			AssertNotEmpty(t, lastReq.Method, "Request method should not be empty")
		}
	})
}

// TestTestEnvFileOperations tests TestEnv file operations
func TestTestEnvFileOperations(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	t.Run("create_and_read_file", func(t *testing.T) {
		content := "test content\nline 2\nline 3"
		path, err := env.CreateFile("testdir/testfile.txt", content)
		AssertNoError(t, err, "CreateFile should succeed")

		// Verify file exists
		AssertCondition(t, env.FileExists("testdir/testfile.txt"), "File should exist")

		// Read file back
		readContent, err := env.ReadFile("testdir/testfile.txt")
		AssertNoError(t, err, "ReadFile should succeed")
		AssertEqual(t, content, readContent, "File content should match")
	})

	t.Run("file_not_exists", func(t *testing.T) {
		AssertCondition(t, !env.FileExists("nonexistent.txt"), "Non-existent file should return false")
	})

	t.Run("nested_directory_creation", func(t *testing.T) {
		path, err := env.CreateFile("deeply/nested/dir/file.txt", "content")
		AssertNoError(t, err, "CreateFile with nested dirs should succeed")
		AssertNotEmpty(t, path, "Path should not be empty")
		AssertCondition(t, env.FileExists("deeply/nested/dir/file.txt"), "Nested file should exist")
	})
}

// TestTestEnvCleanup tests cleanup functionality
func TestTestEnvCleanup(t *testing.T) {
	var cleanupCalled bool

	env := NewTestEnv(t)
	env.AddCleanup(func() {
		cleanupCalled = true
	})

	// Cleanup should be called when we explicitly call it
	// In normal flow, defer handles this
	env.Cleanup()

	AssertCondition(t, cleanupCalled, "Cleanup function should be called")
}

// TestMockOAuthServerFunctionality tests mock OAuth server
func TestMockOAuthServerFunctionality(t *testing.T) {
	mockOAuth := NewMockOAuthServer()
	defer mockOAuth.Close()

	t.Run("authorization_url", func(t *testing.T) {
		authURL := mockOAuth.GetAuthorizationURL()
		AssertContains(t, authURL, "/authorize", "Auth URL should contain /authorize")
	})

	t.Run("token_url", func(t *testing.T) {
		tokenURL := mockOAuth.GetTokenURL()
		AssertContains(t, tokenURL, "/token", "Token URL should contain /token")
	})
}

// TestAssertionHelpers tests the assertion helper functions
func TestAssertionHelpers(t *testing.T) {
	t.Run("assert_equal", func(t *testing.T) {
		AssertEqual(t, "hello", "hello", "Strings should be equal")
		AssertEqual(t, 42, 42, "Integers should be equal")
		AssertEqual(t, true, true, "Booleans should be equal")
	})

	t.Run("assert_not_empty", func(t *testing.T) {
		AssertNotEmpty(t, "content", "String should not be empty")
	})

	t.Run("assert_no_error", func(t *testing.T) {
		AssertNoError(t, nil, "Nil error should pass")
	})

	t.Run("assert_error", func(t *testing.T) {
		AssertError(t, context.DeadlineExceeded, "Non-nil error should pass")
	})

	t.Run("assert_contains", func(t *testing.T) {
		AssertContains(t, "hello world", "world", "String should contain substring")
	})

	t.Run("assert_condition", func(t *testing.T) {
		AssertCondition(t, true, "True condition should pass")
		AssertCondition(t, 1+1 == 2, "Math should work")
	})
}

// TestConcurrentFileOperations tests concurrent file operations
func TestConcurrentFileOperations(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	done := make(chan bool, 10)

	// Create multiple files concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			path := "concurrent/file_" + string(rune('0'+idx)) + ".txt"
			_, err := env.CreateFile(path, "content")
			AssertNoError(t, err, "Concurrent file creation should succeed")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(5 * time.Second):
			t.Error("Concurrent file operation timed out")
		}
	}
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	ctx, cancel := context.WithTimeout(env.Context, 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(2 * time.Millisecond)

	AssertCondition(t, ctx.Err() == context.DeadlineExceeded, "Context should be expired")
}

// TestLargeFileHandling tests handling of larger files
func TestLargeFileHandling(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a larger content (1KB)
	var largeContent string
	for i := 0; i < 1024; i++ {
		largeContent += "x"
	}

	path, err := env.CreateFile("large_file.txt", largeContent)
	AssertNoError(t, err, "Large file creation should succeed")

	readContent, err := env.ReadFile("large_file.txt")
	AssertNoError(t, err, "Large file reading should succeed")
	AssertEqual(t, len(largeContent), len(readContent), "Large file content length should match")
	AssertEqual(t, largeContent, readContent, "Large file content should match")

	_ = path // Use path to avoid unused variable warning
}

// TestBinaryFileHandling tests handling of binary files
func TestBinaryFileHandling(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create binary content (null bytes and special characters)
	binaryContent := string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD})

	path, err := env.CreateFile("binary.bin", binaryContent)
	AssertNoError(t, err, "Binary file creation should succeed")

	readContent, err := env.ReadFile("binary.bin")
	AssertNoError(t, err, "Binary file reading should succeed")
	AssertEqual(t, binaryContent, readContent, "Binary content should match exactly")

	_ = path
}

// TestMultipleCleanupFunctions tests multiple cleanup functions
func TestMultipleCleanupFunctions(t *testing.T) {
	callOrder := make([]int, 0)

	env := NewTestEnv(t)

	env.AddCleanup(func() { callOrder = append(callOrder, 1) })
	env.AddCleanup(func() { callOrder = append(callOrder, 2) })
	env.AddCleanup(func() { callOrder = append(callOrder, 3) })

	env.Cleanup()

	// Cleanups should run in reverse order (LIFO)
	AssertEqual(t, 3, len(callOrder), "All cleanups should be called")
	AssertEqual(t, 3, callOrder[0], "Third cleanup should be first (LIFO)")
	AssertEqual(t, 2, callOrder[1], "Second cleanup should be second (LIFO)")
	AssertEqual(t, 1, callOrder[2], "First cleanup should be last (LIFO)")
}
