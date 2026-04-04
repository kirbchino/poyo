// Package mcp provides OAuth 2.0 authentication for MCP servers.
package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthConfig represents OAuth 2.0 configuration
type OAuthConfig struct {
	// ClientID is the OAuth client identifier
	ClientID string `json:"clientId"`

	// ClientSecret is the OAuth client secret (optional for PKCE)
	ClientSecret string `json:"clientSecret,omitempty"`

	// AuthorizationEndpoint is the authorization server URL
	AuthorizationEndpoint string `json:"authorizationEndpoint"`

	// TokenEndpoint is the token server URL
	TokenEndpoint string `json:"tokenEndpoint"`

	// RedirectURI is the redirect URI for the callback
	RedirectURI string `json:"redirectUri,omitempty"`

	// Scopes are the requested OAuth scopes
	Scopes []string `json:"scopes,omitempty"`

	// UsePKCE indicates whether to use PKCE
	UsePKCE bool `json:"usePKCE,omitempty"`

	// CallbackPort is the local port for OAuth callback
	CallbackPort int `json:"callbackPort,omitempty"`

	// State is the state parameter for CSRF protection
	State string `json:"state,omitempty"`
}

// PKCEVerifier contains PKCE code verifier and challenge
type PKCEVerifier struct {
	CodeVerifier  string `json:"codeVerifier"`
	CodeChallenge string `json:"codeChallenge"`
	Method        string `json:"method"` // S256 or plain
}

// GeneratePKCEVerifier generates a new PKCE verifier
func GeneratePKCEVerifier() (*PKCEVerifier, error) {
	// Generate random code verifier (43-128 characters)
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return nil, fmt.Errorf("failed to generate random verifier: %w", err)
	}

	codeVerifier := base64URLEncode(verifier)

	// Generate code challenge using S256
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64URLEncode(hash[:])

	return &PKCEVerifier{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256",
	}, nil
}

// base64URLEncode encodes bytes using base64 URL encoding without padding
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// Token represents an OAuth 2.0 access token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second)) // 30 second buffer
}

// IsValid checks if the token is valid
func (t *Token) IsValid() bool {
	return t.AccessToken != "" && !t.IsExpired()
}

// TokenResponse represents an OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// AuthorizationRequest represents an OAuth authorization request
type AuthorizationRequest struct {
	ResponseType string   `json:"response_type"`
	ClientID     string   `json:"client_id"`
	RedirectURI  string   `json:"redirect_uri"`
	Scope        string   `json:"scope"`
	State        string   `json:"state"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

// TokenRequest represents an OAuth token request
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuthFlow handles OAuth 2.0 authentication flow
type OAuthFlow struct {
	config     *OAuthConfig
	httpClient *http.Client
	store      TokenStore
	mu         sync.Mutex
}

// TokenStore is the interface for token storage
type TokenStore interface {
	Save(serverName string, token *Token) error
	Load(serverName string) (*Token, error)
	Delete(serverName string) error
}

// NewOAuthFlow creates a new OAuth flow
func NewOAuthFlow(config *OAuthConfig, store TokenStore) *OAuthFlow {
	return &OAuthFlow{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store: store,
	}
}

// StartAuthorization starts the OAuth authorization flow
func (f *OAuthFlow) StartAuthorization(ctx context.Context) (string, error) {
	// Generate PKCE verifier if enabled
	var pkce *PKCEVerifier
	var err error
	if f.config.UsePKCE {
		pkce, err = GeneratePKCEVerifier()
		if err != nil {
			return "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
		}
	}

	// Generate state for CSRF protection
	state := generateState()

	// Build authorization URL
	authURL, err := f.buildAuthorizationURL(pkce, state)
	if err != nil {
		return "", fmt.Errorf("failed to build authorization URL: %w", err)
	}

	// Start callback server
	callbackPort := f.config.CallbackPort
	if callbackPort == 0 {
		callbackPort = 0 // Use random port
	}

	code, err := f.waitForCallback(ctx, callbackPort, state)
	if err != nil {
		return "", fmt.Errorf("callback failed: %w", err)
	}

	// Exchange code for token
	token, err := f.exchangeCode(ctx, code, pkce)
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %w", err)
	}

	return token.AccessToken, nil
}

// buildAuthorizationURL builds the authorization URL
func (f *OAuthFlow) buildAuthorizationURL(pkce *PKCEVerifier, state string) (string, error) {
	u, err := url.Parse(f.config.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid authorization endpoint: %w", err)
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", f.config.ClientID)

	if f.config.RedirectURI != "" {
		params.Set("redirect_uri", f.config.RedirectURI)
	}

	if len(f.config.Scopes) > 0 {
		params.Set("scope", strings.Join(f.config.Scopes, " "))
	}

	params.Set("state", state)

	if pkce != nil {
		params.Set("code_challenge", pkce.CodeChallenge)
		params.Set("code_challenge_method", pkce.Method)
	}

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// waitForCallback starts a callback server and waits for the OAuth callback
func (f *OAuthFlow) waitForCallback(ctx context.Context, port int, expectedState string) (string, error) {
	// Find available port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", fmt.Errorf("failed to start callback listener: %w", err)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	// Create callback URL
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", actualPort)

	// Channel to receive the code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Parse callback parameters
		query := r.URL.Query()

		// Check for error
		if errParam := query.Get("error"); errParam != "" {
			errDesc := query.Get("error_description")
			http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errParam, errDesc), http.StatusBadRequest)
			errChan <- fmt.Errorf("OAuth error: %s - %s", errParam, errDesc)
			return
		}

		// Validate state
		state := query.Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			errChan <- fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state)
			return
		}

		// Get authorization code
		code := query.Get("code")
		if code == "" {
			http.Error(w, "No authorization code", http.StatusBadRequest)
			errChan <- fmt.Errorf("no authorization code in callback")
			return
		}

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Authentication successful!</h1><p>You can close this window.</p></body></html>`))

		codeChan <- code
	})

	server := &http.Server{
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Update redirect URI if using dynamic port
	if f.config.RedirectURI == "" || strings.Contains(f.config.RedirectURI, "localhost") {
		f.config.RedirectURI = callbackURL
	}

	// Print authorization URL for user to visit
	fmt.Printf("Please visit the following URL to authorize:\n%s\n", f.buildAuthorizationURLForDisplay(actualPort))

	// Wait for callback or context cancellation
	select {
	case code := <-codeChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return code, nil

	case err := <-errChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return "", err

	case <-ctx.Done():
		server.Shutdown(ctx)
		return "", ctx.Err()
	}
}

// buildAuthorizationURLForDisplay builds the authorization URL with the correct callback port
func (f *OAuthFlow) buildAuthorizationURLForDisplay(port int) string {
	u, _ := url.Parse(f.config.AuthorizationEndpoint)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", f.config.ClientID)
	params.Set("redirect_uri", fmt.Sprintf("http://127.0.0.1:%d/callback", port))

	if len(f.config.Scopes) > 0 {
		params.Set("scope", strings.Join(f.config.Scopes, " "))
	}

	u.RawQuery = params.Encode()
	return u.String()
}

// exchangeCode exchanges the authorization code for a token
func (f *OAuthFlow) exchangeCode(ctx context.Context, code string, pkce *PKCEVerifier) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", f.config.ClientID)

	if f.config.RedirectURI != "" {
		data.Set("redirect_uri", f.config.RedirectURI)
	}

	if f.config.ClientSecret != "" {
		data.Set("client_secret", f.config.ClientSecret)
	}

	if pkce != nil {
		data.Set("code_verifier", pkce.CodeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Scope:        tokenResp.Scope,
		IDToken:      tokenResp.IDToken,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}

// RefreshToken refreshes an access token using a refresh token
func (f *OAuthFlow) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", f.config.ClientID)

	if f.config.ClientSecret != "" {
		data.Set("client_secret", f.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("refresh error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// If no new refresh token, keep the old one
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return token, nil
}

// GetValidToken returns a valid token, refreshing if necessary
func (f *OAuthFlow) GetValidToken(ctx context.Context, serverName string) (*Token, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load existing token
	token, err := f.store.Load(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	if token == nil {
		return nil, fmt.Errorf("no token found for %s", serverName)
	}

	// Check if token is valid
	if token.IsValid() {
		return token, nil
	}

	// Try to refresh
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("token expired and no refresh token available")
	}

	newToken, err := f.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Save refreshed token
	if err := f.store.Save(serverName, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return newToken, nil
}

// generateState generates a random state string
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64URLEncode(b)
}

// MemoryTokenStore is an in-memory token store
type MemoryTokenStore struct {
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewMemoryTokenStore creates a new in-memory token store
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens: make(map[string]*Token),
	}
}

// Save saves a token
func (s *MemoryTokenStore) Save(serverName string, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[serverName] = token
	return nil
}

// Load loads a token
func (s *MemoryTokenStore) Load(serverName string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[serverName], nil
}

// Delete deletes a token
func (s *MemoryTokenStore) Delete(serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, serverName)
	return nil
}
