package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// SSRFGuard checks if a URL is allowed for HTTP hooks
type SSRFGuard struct {
	allowedNetworks []string
	deniedNetworks  []string
}

// NewSSRFGuard creates a new SSRF guard with default rules
func NewSSRFGuard() *SSRFGuard {
	return &SSRFGuard{
		// Default denied networks (private, link-local, CGNAT)
		deniedNetworks: []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16", // Link-local (cloud metadata)
			"100.64.0.0/10",  // CGNAT
			"fc00::/7",       // IPv6 ULA
			"fe80::/10",      // IPv6 link-local
		},
		// Allow loopback for local development
		allowedNetworks: []string{
			"127.0.0.0/8",
			"::1/128",
		},
	}
}

// IsAllowed checks if a URL is allowed
func (g *SSRFGuard) IsAllowed(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Get host
	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("empty host in URL")
	}

	// Resolve host to IP addresses
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS resolution failed - allow but let the request fail naturally
		return nil
	}

	for _, ip := range ips {
		if !g.isIPAllowed(ip) {
			return fmt.Errorf("IP %s is in a blocked network range", ip)
		}
	}

	return nil
}

// isIPAllowed checks if an IP is allowed
func (g *SSRFGuard) isIPAllowed(ip net.IP) bool {
	// Check allowed networks first (overrides denied)
	for _, cidr := range g.allowedNetworks {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	// Check denied networks
	for _, cidr := range g.deniedNetworks {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return false
		}
	}

	return true
}

// HTTPClient is the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPHookExecutor executes HTTP hooks
type HTTPHookExecutor struct {
	client     HTTPClient
	ssrfGuard  *SSRFGuard
	envVars    map[string]string
}

// NewHTTPHookExecutor creates a new HTTP hook executor
func NewHTTPHookExecutor() *HTTPHookExecutor {
	return &HTTPHookExecutor{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		ssrfGuard: NewSSRFGuard(),
		envVars:   make(map[string]string),
	}
}

// SetClient sets a custom HTTP client
func (e *HTTPHookExecutor) SetClient(client HTTPClient) {
	e.client = client
}

// SetEnvVars sets environment variables for interpolation
func (e *HTTPHookExecutor) SetEnvVars(vars map[string]string) {
	e.envVars = vars
}

// Execute executes an HTTP hook
func (e *HTTPHookExecutor) Execute(ctx context.Context, hook HTTPHook, input *HookInput) (*HookOutput, error) {
	// 1. SSRF guard check
	if err := e.ssrfGuard.IsAllowed(hook.URL); err != nil {
		return nil, fmt.Errorf("SSRF guard blocked URL: %w", err)
	}

	// 2. Prepare headers with environment variable interpolation
	headers := make(map[string]string)
	for k, v := range hook.Headers {
		headers[k] = e.interpolateEnvVars(v, hook.AllowedEnvVars)
	}

	// 3. Prepare request body
	body := map[string]interface{}{
		"event":      input.Event,
		"toolName":   input.ToolName,
		"toolUseId":  input.ToolUseID,
		"input":      input.Input,
		"output":     input.Output,
		"error":      input.Error,
		"sessionId":  input.SessionID,
		"projectDir": input.ProjectDir,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 4. Create request
	req, err := http.NewRequestWithContext(ctx, "POST", hook.URL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 5. Set headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 6. Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	// 8. Parse response
	var hookOutput HookOutput
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&hookOutput); err != nil {
		// Try to read as plain text
		var textOutput string
		if _, scanErr := fmt.Fscanf(resp.Body, "%s", &textOutput); scanErr == nil {
			return &HookOutput{
				Continue: true,
				Reason:   textOutput,
			}, nil
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &hookOutput, nil
}

// interpolateEnvVars interpolates allowed environment variables in a string
func (e *HTTPHookExecutor) interpolateEnvVars(s string, allowedVars []string) string {
	allowedSet := make(map[string]bool)
	for _, v := range allowedVars {
		allowedSet[v] = true
	}

	// Replace ${VAR} patterns
	result := s
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		value := parts[1]

		// Check if this variable is allowed
		if len(allowedVars) > 0 && !allowedSet[name] {
			continue
		}

		// Replace ${VAR} and $VAR patterns
		result = strings.ReplaceAll(result, fmt.Sprintf("${%s}", name), value)
		result = strings.ReplaceAll(result, fmt.Sprintf("$%s", name), value)
	}

	// Also replace from provided env vars
	for name, value := range e.envVars {
		if len(allowedVars) > 0 && !allowedSet[name] {
			continue
		}
		result = strings.ReplaceAll(result, fmt.Sprintf("${%s}", name), value)
		result = strings.ReplaceAll(result, fmt.Sprintf("$%s", name), value)
	}

	return result
}
