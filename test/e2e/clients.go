// Package e2e provides end-to-end testing implementation
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kirbchino/poyo/internal/services/api"
)

// MockReferenceClient mocks a reference implementation client
type MockReferenceClient struct {
	client  *api.Client
	baseURL string
}

// NewMockReferenceClient creates a mock reference client
func NewMockReferenceClient(apiKey, baseURL string) *MockReferenceClient {
	return &MockReferenceClient{
		client: api.NewClient(api.ClientConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}),
		baseURL: baseURL,
	}
}

// Execute executes a prompt
func (c *MockCCClient) Execute(ctx context.Context, prompt, model string) (*APIResponse, error) {
	start := time.Now()

	params := &api.MessageParams{
		Model: model,
		Messages: []api.Message{
			{
				Role: "user",
				Content: []api.ContentBlock{
					{Type: "text", Text: prompt},
				},
			},
		},
		MaxTokens: 4096,
	}

	resp, err := c.client.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Extract text from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &APIResponse{
		Content:      content,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		Duration:     time.Since(start),
		Raw:          resp,
	}, nil
}

// PoyoClient wraps poyo's API client
type PoyoClient struct {
	client  *api.Client
	baseURL string
}

// NewPoyoClient creates a Poyo client
func NewPoyoClient(apiKey, baseURL string) *PoyoClient {
	return &PoyoClient{
		client: api.NewClient(api.ClientConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}),
		baseURL: baseURL,
	}
}

// Execute executes a prompt
func (c *PoyoClient) Execute(ctx context.Context, prompt, model string) (*APIResponse, error) {
	start := time.Now()

	params := &api.MessageParams{
		Model: model,
		Messages: []api.Message{
			{
				Role: "user",
				Content: []api.ContentBlock{
					{Type: "text", Text: prompt},
				},
			},
		},
		MaxTokens: 4096,
	}

	resp, err := c.client.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Extract text from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &APIResponse{
		Content:      content,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		Duration:     time.Since(start),
		Raw:          resp,
	}, nil
}

// SimulatedClient simulates responses for testing without actual API calls
type SimulatedClient struct {
	name      string
	delay     time.Duration
	errorRate float64
}

// NewSimulatedClient creates a simulated client
func NewSimulatedClient(name string, delay time.Duration, errorRate float64) *SimulatedClient {
	return &SimulatedClient{
		name:      name,
		delay:     delay,
		errorRate: errorRate,
	}
}

// Execute simulates a response
func (c *SimulatedClient) Execute(ctx context.Context, prompt, model string) (*APIResponse, error) {
	// Simulate delay
	select {
	case <-time.After(c.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Generate simulated response
	response := c.generateResponse(prompt)

	return &APIResponse{
		Content:      response,
		InputTokens:  len(prompt) / 4, // Rough estimate
		OutputTokens: len(response) / 4,
		Duration:     c.delay,
		Raw:          nil,
	}, nil
}

// generateResponse generates a simulated response based on the prompt
func (c *SimulatedClient) generateResponse(prompt string) string {
	// Analyze prompt and generate appropriate response
	promptLower := strings.ToLower(prompt)

	// Basic conversation
	if strings.Contains(promptLower, "hello") || strings.Contains(promptLower, "how are you") {
		return fmt.Sprintf("[%s] Hello! I'm doing well, thank you for asking. How can I help you today?", c.name)
	}

	// Self introduction
	if strings.Contains(promptLower, "introduce yourself") {
		return fmt.Sprintf("[%s] I'm an AI assistant, specifically designed to help with coding and software development tasks. I can help with code generation, debugging, file operations, and more.", c.name)
	}

	// Code generation - prime
	if strings.Contains(promptLower, "prime") {
		return fmt.Sprintf("[%s] Here's a Python function to check if a number is prime:\n\ndef is_prime(n: int) -> bool:\n    if n < 2:\n        return False\n    for i in range(2, int(n ** 0.5) + 1):\n        if n %% i == 0:\n            return False\n    return True\n\nThis function uses trial division up to the square root of n for efficiency.", c.name)
	}

	// Code generation - BST
	if strings.Contains(promptLower, "binary search tree") {
		return fmt.Sprintf("[%s] Here's a binary search tree implementation in Go with Insert, Search, and Delete methods:\n\ntype Node struct { Value int; Left, Right *Node }\ntype BST struct { Root *Node }\n\nfunc (t *BST) Insert(v int) { t.Root = insert(t.Root, v) }\nfunc insert(n *Node, v int) *Node {\n    if n == nil { return &Node{Value: v} }\n    if v < n.Value { n.Left = insert(n.Left, v) }\n    else { n.Right = insert(n.Right, v) }\n    return n\n}\n\nfunc (t *BST) Search(v int) bool { return search(t.Root, v) }\nfunc search(n *Node, v int) bool {\n    if n == nil { return false }\n    if v == n.Value { return true }\n    if v < n.Value { return search(n.Left, v) }\n    return search(n.Right, v)\n}", c.name)
	}

	// Code generation - error handling
	if strings.Contains(promptLower, "json file") && strings.Contains(promptLower, "error") {
		return fmt.Sprintf("[%s] Here's a Go function that reads a JSON file with proper error handling:\n\nfunc readJSONFile(path string, v interface{}) error {\n    data, err := os.ReadFile(path)\n    if err != nil {\n        return fmt.Errorf(\"failed to read file: %%w\", err)\n    }\n    if err := json.Unmarshal(data, v); err != nil {\n        return fmt.Errorf(\"failed to parse JSON: %%w\", err)\n    }\n    return nil\n}", c.name)
	}

	// File operations
	if strings.Contains(promptLower, "read") || strings.Contains(promptLower, "file") {
		if strings.Contains(promptLower, "/etc/hostname") {
			return fmt.Sprintf("[%s] I read the file /etc/hostname. The hostname is: sandbox-host-001", c.name)
		}
		if strings.Contains(promptLower, "nonexistent") || strings.Contains(promptLower, "invalid") {
			return fmt.Sprintf("[%s] Error: The file does not exist. Please check the path and try again.", c.name)
		}
		if strings.Contains(promptLower, "/tmp/test_e2e.txt") {
			return fmt.Sprintf("[%s] I created the file /tmp/test_e2e.txt with content 'Hello E2E Test!' and verified the content matches.", c.name)
		}
		return fmt.Sprintf("[%s] I'll help you with file operations. What would you like to do?", c.name)
	}

	// Command execution
	if strings.Contains(promptLower, "run") || strings.Contains(promptLower, "command") || strings.Contains(promptLower, "execute") {
		if strings.Contains(promptLower, "date") {
			return fmt.Sprintf("[%s] I executed the 'date' command. The current date and time is: Thu Apr 02 2026 15:30:00 UTC", c.name)
		}
		if strings.Contains(promptLower, "nonexistent") {
			return fmt.Sprintf("[%s] Error: Command 'nonexistent_command_xyz' not found. Please check the command name.", c.name)
		}
		return fmt.Sprintf("[%s] I'll execute that command for you...", c.name)
	}

	// Reasoning - logic puzzle
	if strings.Contains(promptLower, "alice") && strings.Contains(promptLower, "bob") && strings.Contains(promptLower, "carol") {
		return fmt.Sprintf("[%s] Let me solve this logic puzzle:\n\nGiven: Alice is NOT next to Carol, Bob is NOT next to Alice.\n\nTrying arrangements:\n- If Bob is in the middle: A B C or C B A\n  Check: Alice next to Carol? No. Bob next to Alice? Yes (in A B C)\n  \nLet me try A C B:\n- Alice NOT next to Carol? Check: A and C are adjacent - FAILS\n\nLet me try B A C:\n- Alice NOT next to Carol? A and C are adjacent - FAILS\n\nThe answer is: **Bob is in the middle** with arrangement C B A:\n- Alice (end) NOT next to Carol (end) - TRUE\n- Bob (middle) NOT next to Alice (end) - TRUE", c.name)
	}

	// Reasoning - math
	if strings.Contains(promptLower, "train") && strings.Contains(promptLower, "km") {
		return fmt.Sprintf("[%s] Let me solve this math problem:\n\nTrain A: Leaves at 9:00 AM, 60 km/h\nTrain B: Leaves at 10:00 AM, 80 km/h toward A\nDistance: 280 km\n\nBy 10:00 AM, Train A has traveled: 60 km (1 hour)\nRemaining distance: 280 - 60 = 220 km\n\nAfter 10:00 AM, both trains are moving.\nCombined speed: 60 + 80 = 140 km/h\nTime to meet after 10:00 AM: 220 / 140 = 1.57 hours ≈ 1 hour 34 minutes\n\nMeeting time: 10:00 AM + 1:34 = 11:34 AM\n\nAnswer: They meet at approximately **11:34 AM**", c.name)
	}

	// Plugin system
	if strings.Contains(promptLower, "plugin") || strings.Contains(promptLower, "lua") || strings.Contains(promptLower, "poyo") {
		return fmt.Sprintf("[%s] Here's information about the plugin system:\n\nAvailable Host APIs for Lua plugins:\n1. poyo.use(tool, input) - Call host tools\n2. poyo.fs.read/write/exists/remove/mkdir - File operations\n3. poyo.json.encode/decode/pretty - JSON handling\n4. poyo.cache.get/set/delete - Cache operations\n5. poyo.env.get/set/list - Environment variables\n6. poyo.log(level, message) - Logging\n7. poyo.plugin.id/name/version/path - Plugin metadata\n\nExample Lua plugin:\nlocal M = {}\nfunction M.init() poyo.log('info', 'Loaded!') end\nfunction M.tools.greet(input) return 'Hello, ' .. input.name end\nreturn M", c.name)
	}

	// Bug detection
	if strings.Contains(promptLower, "bug") && strings.Contains(promptLower, "calculateAverage") {
		return fmt.Sprintf("[%s] I found the bug!\n\nThe issue is **division by zero** when the slice is empty. If numbers is empty, len(numbers) returns 0, and dividing by 0 causes a panic.\n\n**Fix:**\nfunc calculateAverage(numbers []int) (float64, error) {\n    if len(numbers) == 0 {\n        return 0, errors.New(\"cannot calculate average of empty slice\")\n    }\n    sum := 0\n    for _, n := range numbers { sum += n }\n    return float64(sum) / float64(len(numbers)), nil\n}", c.name)
	}

	// Code review
	if strings.Contains(promptLower, "review") || strings.Contains(promptLower, "improvement") {
		return fmt.Sprintf("[%s] Here are my improvement suggestions:\n\n1. **Use strings.Builder** instead of string concatenation in loops. The current approach creates a new string on each iteration, which is O(n²) complexity.\n\n2. **Pre-allocate capacity** if you know the expected size.\n\n3. **Handle nil/empty input** gracefully at the start of the function.\n\n4. **Consider using bytes.Buffer** if working with byte slices directly.", c.name)
	}

	// Error handling
	if strings.Contains(promptLower, "error") || strings.Contains(promptLower, "fail") {
		return fmt.Sprintf("[%s] I've handled the error gracefully. The operation could not be completed due to an invalid input. Please check your parameters and try again.", c.name)
	}

	// Context awareness
	if strings.Contains(promptLower, "poyo") && strings.Contains(promptLower, "project") {
		return fmt.Sprintf("[%s] Based on the current context, the 'poyo' project is a Go-based CLI tool that provides an AI assistant interface. It features:\n- Multi-provider API support (Anthropic, OpenAI)\n- Plugin system (Lua, Script, MCP, WASM)\n- Tool execution capabilities\n- Hook system for extensibility\n\nSuggested improvements:\n1. Add more comprehensive error handling\n2. Implement plugin hot-reload\n3. Add configuration validation", c.name)
	}

	// Multi-file project
	if strings.Contains(promptLower, "create") && strings.Contains(promptLower, "project") {
		return fmt.Sprintf("[%s] I've created the project structure in /tmp/myproject/:\n\n- main.go: HTTP server entry point on port 8080\n- handler/handler.go: HTTP handler returning 'Hello World'\n- go.mod: Module definition\n\nThe program is ready to run with 'go run main.go'.", c.name)
	}

	// Refactoring
	if strings.Contains(promptLower, "refactor") {
		return fmt.Sprintf("[%s] Here's the refactored code:\n\nfunc checkUser(id int) bool {\n    return id > 0 && id < 100 && id%%2 == 0\n}\n\nChanges:\n1. Simplified nested conditionals into a single expression\n2. Used short-circuit evaluation for clarity\n3. Removed redundant else branches\n4. More idiomatic Go style", c.name)
	}

	// AI/ML explanation
	if strings.Contains(promptLower, "language model") || strings.Contains(promptLower, "ai") {
		return fmt.Sprintf("[%s] A language model is a statistical model that learns patterns from text data. It predicts the next word based on context.\n\nThree examples of usage:\n1. **Code completion**: Predicting the next line of code\n2. **Translation**: Converting text between languages\n3. **Chatbots**: Generating conversational responses", c.name)
	}

	// Default response
	return fmt.Sprintf("[%s] I understand your request. Let me help you with that.", c.name)
}

// JSON helpers
func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
