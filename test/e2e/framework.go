// Package e2e provides end-to-end testing framework for comparing CC and Poyo
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TestCase represents a single test case
type TestCase struct {
	ID          string                 `json:"id"`
	Category    string                 `json:"category"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Prompt      string                 `json:"prompt"`
	ExpectedKey string                 `json:"expected_key"` // Key aspect to evaluate
	Timeout     time.Duration          `json:"timeout"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TestResult represents the result of a test case execution
type TestResult struct {
	TestCaseID  string        `json:"test_case_id"`
	System      string        `json:"system"` // "cc" or "poyo"
	Success     bool          `json:"success"`
	Response    string        `json:"response"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	TokenUsage  TokenUsage    `json:"token_usage"`
	Evaluation  Evaluation    `json:"evaluation"`
	RawResponse interface{}   `json:"raw_response,omitempty"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Evaluation represents the evaluation of a response
type Evaluation struct {
	Correctness   int    `json:"correctness"`   // 1-5
	Completeness  int    `json:"completeness"`  // 1-5
	Relevance     int    `json:"relevance"`     // 1-5
	Clarity       int    `json:"clarity"`       // 1-5
	Speed         int    `json:"speed"`         // 1-5
	Comments      string `json:"comments"`
	PassesKey     bool   `json:"passes_key"` // Whether it passes the key evaluation
}

// ComparisonReport represents the comparison report between CC and Poyo
type ComparisonReport struct {
	Timestamp      string                 `json:"timestamp"`
	Model          string                 `json:"model"`
	TotalCases     int                    `json:"total_cases"`
	CCResults      []TestResult           `json:"cc_results"`
	PoyoResults    []TestResult           `json:"poyo_results"`
	Summary        ComparisonSummary      `json:"summary"`
	CategoryScores map[string]CategoryScore `json:"category_scores"`
}

// ComparisonSummary represents the summary of comparison
type ComparisonSummary struct {
	CCSuccessRate     float64 `json:"cc_success_rate"`
	PoyoSuccessRate   float64 `json:"poyo_success_rate"`
	CCAvgDuration     float64 `json:"cc_avg_duration"`
	PoyoAvgDuration   float64 `json:"poyo_avg_duration"`
	CCAvgTokens       int     `json:"cc_avg_tokens"`
	PoyoAvgTokens     int     `json:"poyo_avg_tokens"`
	CCAvgScore        float64 `json:"cc_avg_score"`
	PoyoAvgScore      float64 `json:"poyo_avg_score"`
	Winner            string  `json:"winner"` // "cc", "poyo", or "tie"
	ImprovementPct    float64 `json:"improvement_pct"` // Poyo improvement over CC
}

// CategoryScore represents scores by category
type CategoryScore struct {
	Category      string  `json:"category"`
	CCScore       float64 `json:"cc_score"`
	PoyoScore     float64 `json:"poyo_score"`
	CCSuccessRate float64 `json:"cc_success_rate"`
	PoyoSuccessRate float64 `json:"poyo_success_rate"`
}

// TestRunner runs end-to-end tests
type TestRunner struct {
	testCases []TestCase
	ccClient  APIClient
	poyoClient APIClient
	model     string
	results   struct {
		cc   []TestResult
		poyo []TestResult
		mu   sync.Mutex
	}
}

// APIClient defines the interface for API clients
type APIClient interface {
	Execute(ctx context.Context, prompt string, model string) (*APIResponse, error)
}

// APIResponse represents an API response
type APIResponse struct {
	Content     string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Raw          interface{}
}

// NewTestRunner creates a new test runner
func NewTestRunner(ccClient, poyoClient APIClient, model string) *TestRunner {
	return &TestRunner{
		testCases:  DefaultTestCases(),
		ccClient:   ccClient,
		poyoClient: poyoClient,
		model:      model,
	}
}

// DefaultTestCases returns the default test cases for comparison
func DefaultTestCases() []TestCase {
	return []TestCase{
		// ═══════════════════════════════════════════════════════
		// Category: Basic Conversation
		// ═══════════════════════════════════════════════════════
		{
			ID:          "BC-001",
			Category:    "basic_conversation",
			Name:        "Simple greeting",
			Description: "Test basic greeting response",
			Prompt:      "Hello! How are you today?",
			ExpectedKey: "Friendly greeting response",
			Timeout:     30 * time.Second,
		},
		{
			ID:          "BC-002",
			Category:    "basic_conversation",
			Name:        "Self introduction",
			Description: "Test self-introduction capability",
			Prompt:      "Please introduce yourself briefly.",
			ExpectedKey: "Clear self-introduction",
			Timeout:     30 * time.Second,
		},
		{
			ID:          "BC-003",
			Category:    "basic_conversation",
			Name:        "Follow-up question",
			Description: "Test multi-turn conversation context",
			Prompt:      "I'm learning about AI. Can you explain what a language model is? Then give me 3 examples of how it's used.",
			ExpectedKey: "Clear explanation with 3 examples",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Code Generation
		// ═══════════════════════════════════════════════════════
		{
			ID:          "CG-001",
			Category:    "code_generation",
			Name:        "Simple function",
			Description: "Test basic function generation",
			Prompt:      "Write a Python function that checks if a number is prime. Include docstring and type hints.",
			ExpectedKey: "Correct prime check function with docstring and type hints",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "CG-002",
			Category:    "code_generation",
			Name:        "Algorithm implementation",
			Description: "Test algorithm implementation",
			Prompt:      "Implement a binary search tree in Go with Insert, Search, and Delete methods.",
			ExpectedKey: "Complete BST implementation with all three methods",
			Timeout:     90 * time.Second,
		},
		{
			ID:          "CG-003",
			Category:    "code_generation",
			Name:        "Error handling",
			Description: "Test error handling code generation",
			Prompt:      "Write a Go function that reads a JSON file and handles all possible errors gracefully. Return appropriate error messages.",
			ExpectedKey: "Comprehensive error handling with descriptive messages",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Tool Usage
		// ═══════════════════════════════════════════════════════
		{
			ID:          "TU-001",
			Category:    "tool_usage",
			Name:        "File operations",
			Description: "Test file reading capability",
			Prompt:      "Read the file /etc/hostname and tell me the hostname of this machine.",
			ExpectedKey: "Correctly reads and reports hostname",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "TU-002",
			Category:    "tool_usage",
			Name:        "Command execution",
			Description: "Test shell command execution",
			Prompt:      "Run 'date' command and tell me the current date and time in a friendly format.",
			ExpectedKey: "Correctly executes and formats date output",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "TU-003",
			Category:    "tool_usage",
			Name:        "Multi-step file operations",
			Description: "Test creating, writing, and reading files",
			Prompt:      "Create a file /tmp/test_e2e.txt with the content 'Hello E2E Test!', then read it back and confirm the content matches.",
			ExpectedKey: "Creates, reads, and confirms file content",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Reasoning
		// ═══════════════════════════════════════════════════════
		{
			ID:          "RS-001",
			Category:    "reasoning",
			Name:        "Logical puzzle",
			Description: "Test logical reasoning",
			Prompt:      "Three people (Alice, Bob, Carol) are sitting in a row. Alice is not next to Carol. Bob is not next to Alice. Who is in the middle? Explain your reasoning.",
			ExpectedKey: "Correct answer (Bob) with clear reasoning",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "RS-002",
			Category:    "reasoning",
			Name:        "Math problem",
			Description: "Test mathematical reasoning",
			Prompt:      "A train leaves Station A at 9:00 AM going 60 km/h. Another train leaves Station B at 10:00 AM going 80 km/h toward Station A. If the stations are 280 km apart, at what time do they meet? Show your work.",
			ExpectedKey: "Correct time (11:30 AM) with clear calculation steps",
			Timeout:     90 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Code Analysis
		// ═══════════════════════════════════════════════════════
		{
			ID:          "CA-001",
			Category:    "code_analysis",
			Name:        "Bug detection",
			Description: "Test bug detection in code",
			Prompt:      `Find the bug in this Go code:

func calculateAverage(numbers []int) float64 {
    sum := 0
    for _, n := range numbers {
        sum += n
    }
    return float64(sum) / float64(len(numbers))
}

What happens when the slice is empty? How would you fix it?`,
			ExpectedKey: "Identifies division by zero and suggests fix",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "CA-002",
			Category:    "code_analysis",
			Name:        "Code review",
			Description: "Test code review capability",
			Prompt: `Review this code for best practices and potential issues:

func processData(data []byte) string {
    result := ""
    for i := 0; i < len(data); i++ {
        result += string(data[i])
    }
    return result
}

Provide at least 3 specific improvement suggestions.`,
			ExpectedKey: "Identifies string concatenation inefficiency, suggests strings.Builder, etc.",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Plugin System
		// ═══════════════════════════════════════════════════════
		{
			ID:          "PS-001",
			Category:    "plugin_system",
			Name:        "Lua plugin basic",
			Description: "Test Lua plugin understanding",
			Prompt:      "Explain how to create a simple Lua plugin that adds a 'greet' tool. Show the plugin manifest and Lua code.",
			ExpectedKey: "Shows correct plugin structure with manifest and Lua handler",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "PS-002",
			Category:    "plugin_system",
			Name:        "Plugin host API",
			Description: "Test plugin host API knowledge",
			Prompt:      "List at least 5 host APIs available to Lua plugins and explain what each does.",
			ExpectedKey: "Lists 5+ APIs like poyo.use, poyo.fs, poyo.log, poyo.cache, poyo.json with explanations",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Error Handling
		// ═══════════════════════════════════════════════════════
		{
			ID:          "EH-001",
			Category:    "error_handling",
			Name:        "Invalid file path",
			Description: "Test error handling for invalid file",
			Prompt:      "Read the file /nonexistent/path/to/file.txt and handle the error gracefully.",
			ExpectedKey: "Handles error gracefully, reports file not found",
			Timeout:     60 * time.Second,
		},
		{
			ID:          "EH-002",
			Category:    "error_handling",
			Name:        "Invalid command",
			Description: "Test handling of invalid commands",
			Prompt:      "Run the command 'nonexistent_command_xyz' and explain what happened.",
			ExpectedKey: "Reports command not found, explains the error",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Complex Tasks
		// ═══════════════════════════════════════════════════════
		{
			ID:          "CT-001",
			Category:    "complex_tasks",
			Name:        "Multi-file project",
			Description: "Test multi-file project creation",
			Prompt:      "Create a simple Go project structure in /tmp/myproject/ with: main.go (entry point), handler/handler.go (HTTP handler), and go.mod. The program should start an HTTP server on port 8080 that responds 'Hello World'.",
			ExpectedKey: "Creates complete project structure with working code",
			Timeout:     120 * time.Second,
		},
		{
			ID:          "CT-002",
			Category:    "complex_tasks",
			Name:        "Refactoring task",
			Description: "Test code refactoring",
			Prompt: `Given this code, refactor it to use proper Go idioms:

func checkUser(id int) bool {
    if id > 0 {
        if id < 100 {
            if id%2 == 0 {
                return true
            } else {
                return false
            }
        }
    }
    return false
}

Refactor for clarity and explain the changes.`,
			ExpectedKey: "Simplifies nested conditionals, uses early returns",
			Timeout:     60 * time.Second,
		},

		// ═══════════════════════════════════════════════════════
		// Category: Context Awareness
		// ═══════════════════════════════════════════════════════
		{
			ID:          "CX-001",
			Category:    "context_awareness",
			Name:        "Remember previous context",
			Description: "Test context retention",
			Prompt:      "I'm working on a Go project called 'poyo'. First, tell me what you know about this project from context. Then suggest 3 improvements.",
			ExpectedKey: "Demonstrates awareness of current project context",
			Timeout:     60 * time.Second,
		},
	}
}

// RunAll runs all test cases on both CC and Poyo
func (r *TestRunner) RunAll(ctx context.Context) (*ComparisonReport, error) {
	report := &ComparisonReport{
		Timestamp:      time.Now().Format(time.RFC3339),
		Model:          r.model,
		TotalCases:     len(r.testCases),
		RefResults:     make([]TestResult, 0, len(r.testCases)),
		PoyoResults:    make([]TestResult, 0, len(r.testCases)),
		CategoryScores: make(map[string]CategoryScore),
	}

	// Run tests for Reference
	fmt.Println("Running tests on Reference implementation...")
	for _, tc := range r.testCases {
		result := r.runTestCase(ctx, tc, r.refClient, "reference")
		report.RefResults = append(report.RefResults, result)
		fmt.Printf("  [%s] %s: %v (%.2fs)\n", tc.ID, tc.Name, result.Success, result.Duration.Seconds())
	}

	// Run tests for Poyo
	fmt.Println("\nRunning tests on Poyo...")
	for _, tc := range r.testCases {
		result := r.runTestCase(ctx, tc, r.poyoClient, "poyo")
		report.PoyoResults = append(report.PoyoResults, result)
		fmt.Printf("  [%s] %s: %v (%.2fs)\n", tc.ID, tc.Name, result.Success, result.Duration.Seconds())
	}

	// Calculate summary
	report.Summary = r.calculateSummary(report)
	report.CategoryScores = r.calculateCategoryScores(report)

	return report, nil
}

// runTestCase runs a single test case
func (r *TestRunner) runTestCase(ctx context.Context, tc TestCase, client APIClient, system string) TestResult {
	start := time.Now()
	result := TestResult{
		TestCaseID: tc.ID,
		System:     system,
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, tc.Timeout)
	defer cancel()

	// Execute the test
	resp, err := client.Execute(ctx, tc.Prompt, r.model)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		result.Success = false
		return result
	}

	result.Response = resp.Content
	result.Duration = time.Since(start)
	result.TokenUsage = TokenUsage{
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		TotalTokens:  resp.InputTokens + resp.OutputTokens,
	}
	result.RawResponse = resp.Raw

	// Evaluate the response
	result.Evaluation = r.evaluateResponse(tc, resp.Content)
	result.Success = result.Evaluation.PassesKey

	return result
}

// evaluateResponse evaluates a response against the test case
func (r *TestRunner) evaluateResponse(tc TestCase, response string) Evaluation {
	eval := Evaluation{
		Comments: "",
		PassesKey: false,
	}

	// Simple heuristic evaluation (in real scenario, could use another LLM)
	// Check for non-empty response
	if len(response) < 10 {
		eval.Comments = "Response too short"
		return eval
	}

	// Basic scoring based on response length and structure
	eval.Clarity = 3
	eval.Relevance = 4
	eval.Completeness = 3
	eval.Correctness = 4
	eval.Speed = 4

	// Check for key expected elements based on category
	switch tc.Category {
	case "basic_conversation":
		eval.PassesKey = len(response) >= 20
		eval.Completeness = 4
	case "code_generation":
		// Check if response contains code-like content
		eval.PassesKey = len(response) >= 50 && (contains(response, "func ") ||
			contains(response, "def ") ||
			contains(response, "class "))
		if eval.PassesKey {
			eval.Completeness = 4
		}
	case "tool_usage":
		eval.PassesKey = len(response) >= 20
		eval.Clarity = 4
	case "reasoning":
		eval.PassesKey = len(response) >= 50
		eval.Relevance = 4
	case "code_analysis":
		eval.PassesKey = len(response) >= 50
		eval.Correctness = 4
	case "plugin_system":
		eval.PassesKey = len(response) >= 50 && (contains(response, "poyo") ||
			contains(response, "plugin") ||
			contains(response, "Lua"))
	case "error_handling":
		eval.PassesKey = contains(response, "error") || contains(response, "not found") || contains(response, "fail")
	case "complex_tasks":
		eval.PassesKey = len(response) >= 100
		eval.Completeness = 4
	case "context_awareness":
		eval.PassesKey = contains(response, "poyo") || contains(response, "plugin")
	default:
		eval.PassesKey = len(response) >= 20
	}

	// Calculate average score
	avgScore := float64(eval.Correctness+eval.Completeness+eval.Relevance+eval.Clarity+eval.Speed) / 5
	eval.Comments = fmt.Sprintf("Average score: %.1f/5", avgScore)

	return eval
}

// calculateSummary calculates the comparison summary
func (r *TestRunner) calculateSummary(report *ComparisonReport) ComparisonSummary {
	var ccSuccess, poyoSuccess int
	var ccTotalDuration, poyoTotalDuration time.Duration
	var ccTotalTokens, poyoTotalTokens int
	var ccTotalScore, poyoTotalScore float64

	for _, r := range report.CCResults {
		if r.Success {
			ccSuccess++
		}
		ccTotalDuration += r.Duration
		ccTotalTokens += r.TokenUsage.TotalTokens
		ccTotalScore += float64(r.Evaluation.Correctness+r.Evaluation.Completeness+r.Evaluation.Relevance+r.Evaluation.Clarity+r.Evaluation.Speed) / 5
	}

	for _, r := range report.PoyoResults {
		if r.Success {
			poyoSuccess++
		}
		poyoTotalDuration += r.Duration
		poyoTotalTokens += r.TokenUsage.TotalTokens
		poyoTotalScore += float64(r.Evaluation.Correctness+r.Evaluation.Completeness+r.Evaluation.Relevance+r.Evaluation.Clarity+r.Evaluation.Speed) / 5
	}

	total := float64(report.TotalCases)
	ccSuccessRate := float64(ccSuccess) / total
	poyoSuccessRate := float64(poyoSuccess) / total
	ccAvgScore := ccTotalScore / total
	poyoAvgScore := poyoTotalScore / total

	summary := ComparisonSummary{
		CCSuccessRate:   ccSuccessRate,
		PoyoSuccessRate: poyoSuccessRate,
		CCAvgDuration:   ccTotalDuration.Seconds() / total,
		PoyoAvgDuration: poyoTotalDuration.Seconds() / total,
		CCAvgTokens:     ccTotalTokens / int(total),
		PoyoAvgTokens:   poyoTotalTokens / int(total),
		CCAvgScore:      ccAvgScore,
		PoyoAvgScore:    poyoAvgScore,
	}

	// Determine winner
	if poyoAvgScore > ccAvgScore {
		summary.Winner = "poyo"
		summary.ImprovementPct = ((poyoAvgScore - ccAvgScore) / ccAvgScore) * 100
	} else if ccAvgScore > poyoAvgScore {
		summary.Winner = "cc"
		summary.ImprovementPct = -((ccAvgScore - poyoAvgScore) / ccAvgScore) * 100
	} else {
		summary.Winner = "tie"
		summary.ImprovementPct = 0
	}

	return summary
}

// calculateCategoryScores calculates scores by category
func (r *TestRunner) calculateCategoryScores(report *ComparisonReport) map[string]CategoryScore {
	categoryScores := make(map[string]CategoryScore)

	// Collect scores by category
	ccCategoryData := make(map[string]struct{ success, total int; score float64 })
	poyoCategoryData := make(map[string]struct{ success, total int; score float64 })

	for _, tc := range r.testCases {
		ccCategoryData[tc.Category] = struct{ success, total int; score float64 }{
			total: ccCategoryData[tc.Category].total + 1,
		}
		poyoCategoryData[tc.Category] = struct{ success, total int; score float64 }{
			total: poyoCategoryData[tc.Category].total + 1,
		}
	}

	for _, result := range report.CCResults {
		tc := r.findTestCase(result.TestCaseID)
		if tc == nil {
			continue
		}
		data := ccCategoryData[tc.Category]
		if result.Success {
			data.success++
		}
		data.score += float64(result.Evaluation.Correctness+result.Evaluation.Completeness+result.Evaluation.Relevance+result.Evaluation.Clarity+result.Evaluation.Speed) / 5
		ccCategoryData[tc.Category] = data
	}

	for _, result := range report.PoyoResults {
		tc := r.findTestCase(result.TestCaseID)
		if tc == nil {
			continue
		}
		data := poyoCategoryData[tc.Category]
		if result.Success {
			data.success++
		}
		data.score += float64(result.Evaluation.Correctness+result.Evaluation.Completeness+result.Evaluation.Relevance+result.Evaluation.Clarity+result.Evaluation.Speed) / 5
		poyoCategoryData[tc.Category] = data
	}

	// Calculate final scores
	for category := range ccCategoryData {
		ccData := ccCategoryData[category]
		poyoData := poyoCategoryData[category]

		categoryScores[category] = CategoryScore{
			Category:        category,
			CCScore:         ccData.score / float64(ccData.total),
			PoyoScore:       poyoData.score / float64(poyoData.total),
			CCSuccessRate:   float64(ccData.success) / float64(ccData.total),
			PoyoSuccessRate: float64(poyoData.success) / float64(poyoData.total),
		}
	}

	return categoryScores
}

// findTestCase finds a test case by ID
func (r *TestRunner) findTestCase(id string) *TestCase {
	for i := range r.testCases {
		if r.testCases[i].ID == id {
			return &r.testCases[i]
		}
	}
	return nil
}

// SaveReport saves the comparison report to a file
func SaveReport(report *ComparisonReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// GenerateMarkdownReport generates a markdown report
func GenerateMarkdownReport(report *ComparisonReport) string {
	var md string

	md += "# E2E Comparison Report: Reference vs Poyo\n\n"
	md += fmt.Sprintf("**Generated:** %s\n", report.Timestamp)
	md += fmt.Sprintf("**Model:** %s\n", report.Model)
	md += fmt.Sprintf("**Total Test Cases:** %d\n\n", report.TotalCases)

	// Summary
	md += "## Summary\n\n"
	md += "| Metric | Reference | Poyo |\n"
	md += "|--------|-----------|------|\n"
	md += fmt.Sprintf("| Success Rate | %.1f%% | %.1f%% |\n", report.Summary.RefSuccessRate*100, report.Summary.PoyoSuccessRate*100)
	md += fmt.Sprintf("| Avg Duration | %.2fs | %.2fs |\n", report.Summary.RefAvgDuration, report.Summary.PoyoAvgDuration)
	md += fmt.Sprintf("| Avg Tokens | %d | %d |\n", report.Summary.RefAvgTokens, report.Summary.PoyoAvgTokens)
	md += fmt.Sprintf("| Avg Score | %.2f/5 | %.2f/5 |\n", report.Summary.RefAvgScore, report.Summary.PoyoAvgScore)
	md += "\n"
	md += fmt.Sprintf("**Winner:** %s", report.Summary.Winner)
	if report.Summary.Winner == "poyo" {
		md += fmt.Sprintf(" (+%.1f%% improvement)", report.Summary.ImprovementPct)
	} else if report.Summary.Winner == "reference" {
		md += fmt.Sprintf(" (Poyo is %.1f%% behind)", -report.Summary.ImprovementPct)
	}
	md += "\n\n"

	// Category Scores
	md += "## Scores by Category\n\n"
	md += "| Category | Ref Score | Poyo Score | Ref Success | Poyo Success |\n"
	md += "|----------|-----------|------------|-------------|--------------|\n"
	for cat, score := range report.CategoryScores {
		md += fmt.Sprintf("| %s | %.2f | %.2f | %.0f%% | %.0f%% |\n",
			cat, score.RefScore, score.PoyoScore, score.RefSuccessRate*100, score.PoyoSuccessRate*100)
	}
	md += "\n"

	// Detailed Results
	md += "## Detailed Results\n\n"

	md += "### Reference Results\n\n"
	for _, r := range report.RefResults {
		md += fmt.Sprintf("#### [%s] %s\n", r.TestCaseID, r.TestCaseID)
		md += fmt.Sprintf("- **Success:** %v\n", r.Success)
		md += fmt.Sprintf("- **Duration:** %.2fs\n", r.Duration.Seconds())
		md += fmt.Sprintf("- **Tokens:** %d in, %d out\n", r.TokenUsage.InputTokens, r.TokenUsage.OutputTokens)
		if r.Error != "" {
			md += fmt.Sprintf("- **Error:** %s\n", r.Error)
		}
		md += fmt.Sprintf("- **Score:** Correctness:%d, Completeness:%d, Relevance:%d, Clarity:%d, Speed:%d\n",
			r.Evaluation.Correctness, r.Evaluation.Completeness, r.Evaluation.Relevance, r.Evaluation.Clarity, r.Evaluation.Speed)
		md += "\n"
	}

	md += "### Poyo Results\n\n"
	for _, r := range report.PoyoResults {
		md += fmt.Sprintf("#### [%s] %s\n", r.TestCaseID, r.TestCaseID)
		md += fmt.Sprintf("- **Success:** %v\n", r.Success)
		md += fmt.Sprintf("- **Duration:** %.2fs\n", r.Duration.Seconds())
		md += fmt.Sprintf("- **Tokens:** %d in, %d out\n", r.TokenUsage.InputTokens, r.TokenUsage.OutputTokens)
		if r.Error != "" {
			md += fmt.Sprintf("- **Error:** %s\n", r.Error)
		}
		md += fmt.Sprintf("- **Score:** Correctness:%d, Completeness:%d, Relevance:%d, Clarity:%d, Speed:%d\n",
			r.Evaluation.Correctness, r.Evaluation.Completeness, r.Evaluation.Relevance, r.Evaluation.Clarity, r.Evaluation.Speed)
		md += "\n"
	}

	return md
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
