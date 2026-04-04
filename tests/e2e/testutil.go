// Package e2e provides end-to-end testing infrastructure for Poyo.
package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEnv represents a test environment
type TestEnv struct {
	// RootDir is the temporary directory for the test
	RootDir string

	// HTTPServer is a mock HTTP server
	HTTPServer *httptest.Server

	// Context is the test context
	Context context.Context

	// Cancel cancels the context
	Cancel context.CancelFunc

	// TB is the testing.TB
	TB testing.TB

	// Cleanup functions
	cleanup []func()
}

// NewTestEnv creates a new test environment
func NewTestEnv(tb testing.TB) *TestEnv {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "poyo-test-*")
	if err != nil {
		tb.Fatalf("Failed to create temp dir: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	env := &TestEnv{
		RootDir:    tmpDir,
		HTTPServer: httptest.NewServer(nil),
		Context:    ctx,
		Cancel:     cancel,
		TB:         tb,
		cleanup:    make([]func(), 0),
	}

	// Add cleanup
	env.AddCleanup(func() {
		cancel()
		env.HTTPServer.Close()
		os.RemoveAll(tmpDir)
	})

	return env
}

// Cleanup runs all cleanup functions
func (e *TestEnv) Cleanup() {
	// Run cleanup in reverse order
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
}

// AddCleanup adds a cleanup function
func (e *TestEnv) AddCleanup(fn func()) {
	e.cleanup = append(e.cleanup, fn)
}

// CreateFile creates a file in the test directory
func (e *TestEnv) CreateFile(path string, content string) (string, error) {
	fullPath := filepath.Join(e.RootDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fullPath, nil
}

// ReadFile reads a file from the test directory
func (e *TestEnv) ReadFile(path string) (string, error) {
	fullPath := filepath.Join(e.RootDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// FileExists checks if a file exists
func (e *TestEnv) FileExists(path string) bool {
	fullPath := filepath.Join(e.RootDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// MockMCPServer creates a mock MCP server for testing
type MockMCPServer struct {
	*httptest.Server
	Requests []MockRequest
	Responses map[string]interface{}
}

// MockRequest represents a recorded request
type MockRequest struct {
	Method string
	Path   string
	Body   string
}

// NewMockMCPServer creates a new mock MCP server
func NewMockMCPServer() *MockMCPServer {
	s := &MockMCPServer{
		Requests:  make([]MockRequest, 0),
		Responses: make(map[string]interface{}),
	}

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record request
		body, _ := io.ReadAll(r.Body)
		s.Requests = append(s.Requests, MockRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
		})

		// Check for mock response
		key := r.Method + ":" + r.URL.Path
		if resp, ok := s.Responses[key]; ok {
			switch v := resp.(type) {
			case string:
				w.Write([]byte(v))
			case []byte:
				w.Write(v)
			default:
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%v}`, resp)
			}
			return
		}

		// Default response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))

	return s
}

// SetResponse sets a mock response
func (s *MockMCPServer) SetResponse(method, path string, response interface{}) {
	key := method + ":" + path
	s.Responses[key] = response
}

// GetRequestCount returns the number of requests received
func (s *MockMCPServer) GetRequestCount() int {
	return len(s.Requests)
}

// GetLastRequest returns the last request
func (s *MockMCPServer) GetLastRequest() *MockRequest {
	if len(s.Requests) == 0 {
		return nil
	}
	return &s.Requests[len(s.Requests)-1]
}

// MockOAuthServer creates a mock OAuth server for testing
type MockOAuthServer struct {
	*httptest.Server
	AuthRequests  []AuthRequest
	TokenRequests []TokenRequest
}

// AuthRequest represents a recorded authorization request
type AuthRequest struct {
	ClientID     string
	RedirectURI  string
	Scope        string
	State        string
	CodeChallenge string
}

// TokenRequest represents a recorded token request
type TokenRequest struct {
	GrantType    string
	Code         string
	ClientID     string
	CodeVerifier string
}

// NewMockOAuthServer creates a new mock OAuth server
func NewMockOAuthServer() *MockOAuthServer {
	s := &MockOAuthServer{
		AuthRequests:  make([]AuthRequest, 0),
		TokenRequests: make([]TokenRequest, 0),
	}

	mux := http.NewServeMux()

	// Authorization endpoint
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		s.AuthRequests = append(s.AuthRequests, AuthRequest{
			ClientID:      r.URL.Query().Get("client_id"),
			RedirectURI:   r.URL.Query().Get("redirect_uri"),
			Scope:         r.URL.Query().Get("scope"),
			State:         r.URL.Query().Get("state"),
			CodeChallenge: r.URL.Query().Get("code_challenge"),
		})

		// Redirect back with a code
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		redirectURL := fmt.Sprintf("%s?code=test-code&state=%s", redirectURI, state)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		s.TokenRequests = append(s.TokenRequests, TokenRequest{
			GrantType:    r.Form.Get("grant_type"),
			Code:         r.Form.Get("code"),
			ClientID:     r.Form.Get("client_id"),
			CodeVerifier: r.Form.Get("code_verifier"),
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"refresh_token": "test-refresh-token",
			"expires_in": 3600
		}`))
	})

	s.Server = httptest.NewServer(mux)
	return s
}

// GetAuthorizationURL returns the authorization URL
func (s *MockOAuthServer) GetAuthorizationURL() string {
	return s.URL + "/authorize"
}

// GetTokenURL returns the token URL
func (s *MockOAuthServer) GetTokenURL() string {
	return s.URL + "/token"
}

// AssertCondition asserts a condition is true
func AssertCondition(tb testing.TB, condition bool, message string) {
	tb.Helper()
	if !condition {
		tb.Errorf("Assertion failed: %s", message)
	}
}

// AssertEqual asserts two values are equal
func AssertEqual[T comparable](tb testing.TB, expected, actual T, message string) {
	tb.Helper()
	if expected != actual {
		tb.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertNotEmpty asserts a string is not empty
func AssertNotEmpty(tb testing.TB, value string, message string) {
	tb.Helper()
	if value == "" {
		tb.Errorf("%s: expected non-empty string", message)
	}
}

// AssertNoError asserts no error occurred
func AssertNoError(tb testing.TB, err error, message string) {
	tb.Helper()
	if err != nil {
		tb.Errorf("%s: unexpected error: %v", message, err)
	}
}

// AssertError asserts an error occurred
func AssertError(tb testing.TB, err error, message string) {
	tb.Helper()
	if err == nil {
		tb.Errorf("%s: expected error but got nil", message)
	}
}

// AssertContains asserts a string contains a substring
func AssertContains(tb testing.TB, str, substr string, message string) {
	tb.Helper()
	if !contains(str, substr) {
		tb.Errorf("%s: string %q does not contain %q", message, str, substr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
