package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPKCEVerifier(t *testing.T) {
	pkce, err := GeneratePKCEVerifier()
	if err != nil {
		t.Fatalf("GeneratePKCEVerifier() error: %v", err)
	}

	if pkce.CodeVerifier == "" {
		t.Error("CodeVerifier should not be empty")
	}

	if pkce.CodeChallenge == "" {
		t.Error("CodeChallenge should not be empty")
	}

	if pkce.Method != "S256" {
		t.Errorf("Method = %q, want 'S256'", pkce.Method)
	}

	// Code verifier should be at least 43 characters
	if len(pkce.CodeVerifier) < 43 {
		t.Errorf("CodeVerifier length = %d, want >= 43", len(pkce.CodeVerifier))
	}

	// Code challenge should be different from verifier (hashed)
	if pkce.CodeChallenge == pkce.CodeVerifier {
		t.Error("CodeChallenge should be hashed, not equal to CodeVerifier")
	}
}

func TestBase64URLEncode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte("hello"), "aGVsbG8"},
		{[]byte{0, 1, 2, 3}, "AAECAw"},
		{[]byte{}, ""},
	}

	for _, tt := range tests {
		result := base64URLEncode(tt.input)
		if result != tt.expected {
			t.Errorf("base64URLEncode(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		token    *Token
		expected bool
	}{
		{
			token:    &Token{AccessToken: "test"},
			expected: false, // No expiration set
		},
		{
			token:    &Token{AccessToken: "test", ExpiresAt: time.Now().Add(1 * time.Hour)},
			expected: false,
		},
		{
			token:    &Token{AccessToken: "test", ExpiresAt: time.Now().Add(-1 * time.Hour)},
			expected: true,
		},
		{
			token:    &Token{AccessToken: "test", ExpiresAt: time.Now().Add(10 * time.Second)},
			expected: false, // Within buffer
		},
	}

	for i, tt := range tests {
		result := tt.token.IsExpired()
		if result != tt.expected {
			t.Errorf("Test %d: IsExpired() = %v, want %v", i, result, tt.expected)
		}
	}
}

func TestTokenIsValid(t *testing.T) {
	tests := []struct {
		token    *Token
		expected bool
	}{
		{&Token{}, false},                               // No access token
		{&Token{AccessToken: "test"}, true},             // Valid, no expiration
		{&Token{AccessToken: "", ExpiresAt: time.Now().Add(1 * time.Hour)}, false}, // No access token
		{&Token{AccessToken: "test", ExpiresAt: time.Now().Add(-1 * time.Hour)}, false}, // Expired
	}

	for i, tt := range tests {
		result := tt.token.IsValid()
		if result != tt.expected {
			t.Errorf("Test %d: IsValid() = %v, want %v", i, result, tt.expected)
		}
	}
}

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()

	token := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    3600,
	}

	// Test Save
	err := store.Save("test-server", token)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Test Load
	loaded, err := store.Load("test-server")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}

	// Test Load non-existent
	_, err = store.Load("non-existent")
	if err != nil {
		t.Errorf("Load() for non-existent should not error, got: %v", err)
	}

	// Test Delete
	err = store.Delete("test-server")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err = store.Load("test-server")
	if err != nil {
		t.Errorf("Load() after delete should not error, got: %v", err)
	}
}

func TestOAuthFlowBuildAuthorizationURL(t *testing.T) {
	config := &OAuthConfig{
		ClientID:              "test-client-id",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		RedirectURI:           "http://localhost:8080/callback",
		Scopes:                []string{"read", "write"},
		UsePKCE:               true,
	}

	flow := NewOAuthFlow(config, NewMemoryTokenStore())

	pkce, _ := GeneratePKCEVerifier()
	url, err := flow.buildAuthorizationURL(pkce, "test-state")
	if err != nil {
		t.Fatalf("buildAuthorizationURL() error: %v", err)
	}

	// Check URL contains expected parameters
	if !strings.Contains(url, "response_type=code") {
		t.Error("URL should contain response_type=code")
	}
	if !strings.Contains(url, "client_id=test-client-id") {
		t.Error("URL should contain client_id")
	}
	if !strings.Contains(url, "scope=read+write") || !strings.Contains(url, "scope=read%20write") {
		t.Error("URL should contain scopes")
	}
	if !strings.Contains(url, "state=test-state") {
		t.Error("URL should contain state")
	}
	if !strings.Contains(url, "code_challenge=") {
		t.Error("URL should contain code_challenge")
	}
	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Error("URL should contain code_challenge_method=S256")
	}
}

func TestOAuthFlowExchangeCode(t *testing.T) {
	// Create mock token server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %s, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}

		// Send response
		resp := TokenResponse{
			AccessToken:  "test-access-token",
			TokenType:    "Bearer",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &OAuthConfig{
		ClientID:       "test-client-id",
		ClientSecret:   "test-secret",
		TokenEndpoint:  server.URL,
		RedirectURI:    "http://localhost:8080/callback",
	}

	flow := NewOAuthFlow(config, NewMemoryTokenStore())

	token, err := flow.exchangeCode(context.Background(), "test-code", nil)
	if err != nil {
		t.Fatalf("exchangeCode() error: %v", err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want 'test-access-token'", token.AccessToken)
	}

	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %q, want 'test-refresh-token'", token.RefreshToken)
	}

	if token.IsExpired() {
		t.Error("Token should not be expired")
	}
}

func TestOAuthFlowRefreshToken(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Verify grant_type
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %s, want refresh_token", r.Form.Get("grant_type"))
		}

		resp := TokenResponse{
			AccessToken:  "new-access-token",
			TokenType:    "Bearer",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &OAuthConfig{
		ClientID:      "test-client-id",
		ClientSecret:  "test-secret",
		TokenEndpoint: server.URL,
	}

	flow := NewOAuthFlow(config, NewMemoryTokenStore())

	token, err := flow.RefreshToken(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}

	if token.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want 'new-access-token'", token.AccessToken)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call to token endpoint, got %d", callCount)
	}
}

func TestOAuthFlowRefreshTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TokenResponse{
			Error:     "invalid_grant",
			ErrorDesc: "Refresh token has expired",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &OAuthConfig{
		ClientID:      "test-client-id",
		TokenEndpoint: server.URL,
	}

	flow := NewOAuthFlow(config, NewMemoryTokenStore())

	_, err := flow.RefreshToken(context.Background(), "invalid-refresh-token")
	if err == nil {
		t.Error("RefreshToken() should return error for invalid refresh token")
	}
}

func TestGetValidToken(t *testing.T) {
	store := NewMemoryTokenStore()

	// Store a valid token
	store.Save("test-server", &Token{
		AccessToken:  "valid-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	})

	config := &OAuthConfig{
		ClientID: "test-client-id",
	}

	flow := NewOAuthFlow(config, store)

	token, err := flow.GetValidToken(context.Background(), "test-server")
	if err != nil {
		t.Fatalf("GetValidToken() error: %v", err)
	}

	if token.AccessToken != "valid-token" {
		t.Errorf("AccessToken = %q, want 'valid-token'", token.AccessToken)
	}
}

func TestGetValidTokenExpired(t *testing.T) {
	store := NewMemoryTokenStore()

	// Store an expired token with refresh token
	store.Save("test-server", &Token{
		AccessToken:  "expired-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})

	// Create mock refresh server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TokenResponse{
			AccessToken:  "refreshed-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := &OAuthConfig{
		ClientID:      "test-client-id",
		TokenEndpoint: server.URL,
	}

	flow := NewOAuthFlow(config, store)

	token, err := flow.GetValidToken(context.Background(), "test-server")
	if err != nil {
		t.Fatalf("GetValidToken() error: %v", err)
	}

	if token.AccessToken != "refreshed-token" {
		t.Errorf("AccessToken = %q, want 'refreshed-token'", token.AccessToken)
	}
}

func TestGetValidTokenNoToken(t *testing.T) {
	store := NewMemoryTokenStore()
	config := &OAuthConfig{ClientID: "test-client-id"}
	flow := NewOAuthFlow(config, store)

	_, err := flow.GetValidToken(context.Background(), "non-existent")
	if err == nil {
		t.Error("GetValidToken() should return error for non-existent token")
	}
}

func TestGenerateState(t *testing.T) {
	state1 := generateState()
	state2 := generateState()

	if state1 == "" {
		t.Error("State should not be empty")
	}

	if state1 == state2 {
		t.Error("States should be unique")
	}

	// State should be URL-safe (no special characters that need encoding)
	for _, c := range state1 {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			t.Errorf("State contains invalid character: %c", c)
		}
	}
}

func TestOAuthFlowWithTimeout(t *testing.T) {
	config := &OAuthConfig{
		ClientID:              "test-client-id",
		AuthorizationEndpoint: "https://auth.example.com/authorize",
		TokenEndpoint:         "https://auth.example.com/token",
	}

	flow := NewOAuthFlow(config, NewMemoryTokenStore())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout
	_, err := flow.StartAuthorization(ctx)
	if err == nil {
		t.Error("StartAuthorization() should timeout with short context")
	}
}

func TestTokenResponseParsing(t *testing.T) {
	jsonStr := `{
		"access_token": "test-access-token",
		"token_type": "Bearer",
		"refresh_token": "test-refresh-token",
		"expires_in": 3600,
		"scope": "read write"
	}`

	var resp TokenResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	if resp.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want 'test-access-token'", resp.AccessToken)
	}

	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", resp.ExpiresIn)
	}
}

func TestTokenResponseError(t *testing.T) {
	jsonStr := `{
		"error": "invalid_request",
		"error_description": "Missing required parameter"
	}`

	var resp TokenResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if resp.Error != "invalid_request" {
		t.Errorf("Error = %q, want 'invalid_request'", resp.Error)
	}

	if resp.ErrorDesc != "Missing required parameter" {
		t.Errorf("ErrorDesc = %q, want 'Missing required parameter'", resp.ErrorDesc)
	}
}
