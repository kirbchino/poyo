// Package e2e_test provides end-to-end testing for comparing CC and Poyo
package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/kirbchino/poyo/test/e2e"
)

// TestE2EComparison runs the full E2E comparison
func TestE2EComparison(t *testing.T) {
	// Create simulated clients for testing
	// In real scenario, these would connect to actual CC and Poyo backends
	ccClient := e2e.NewSimulatedClient("CC", 100*time.Millisecond, 0.0)
	poyoClient := e2e.NewSimulatedClient("Poyo", 80*time.Millisecond, 0.0)

	// Create test runner with the specified model
	runner := e2e.NewTestRunner(ccClient, poyoClient, "claude-sonnet-4-6")

	// Run all tests
	ctx := context.Background()
	report, err := runner.RunAll(ctx)
	if err != nil {
		t.Fatalf("Failed to run tests: %v", err)
	}

	// Verify we have results
	if len(report.CCResults) != report.TotalCases {
		t.Errorf("Expected %d CC results, got %d", report.TotalCases, len(report.CCResults))
	}
	if len(report.PoyoResults) != report.TotalCases {
		t.Errorf("Expected %d Poyo results, got %d", report.TotalCases, len(report.PoyoResults))
	}

	// Print summary
	t.Logf("\n=== E2E Comparison Summary ===")
	t.Logf("Total Cases: %d", report.TotalCases)
	t.Logf("CC Success Rate: %.1f%%", report.Summary.CCSuccessRate*100)
	t.Logf("Poyo Success Rate: %.1f%%", report.Summary.PoyoSuccessRate*100)
	t.Logf("CC Avg Score: %.2f/5", report.Summary.CCAvgScore)
	t.Logf("Poyo Avg Score: %.2f/5", report.Summary.PoyoAvgScore)
	t.Logf("Winner: %s", report.Summary.Winner)

	// Generate and save markdown report
	mdReport := e2e.GenerateMarkdownReport(report)
	t.Logf("\n%s", mdReport)
}

// TestIndividualTestCases runs individual test cases for detailed verification
func TestIndividualTestCases(t *testing.T) {
	testCases := e2e.DefaultTestCases()

	// Test a few key cases
	keyCases := []string{"BC-001", "CG-001", "TU-001", "RS-001", "PS-001"}

	ccClient := e2e.NewSimulatedClient("CC", 50*time.Millisecond, 0.0)
	poyoClient := e2e.NewSimulatedClient("Poyo", 40*time.Millisecond, 0.0)

	for _, tc := range testCases {
		found := false
		for _, key := range keyCases {
			if tc.ID == key {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		t.Run(tc.ID+"_"+tc.Name, func(t *testing.T) {
			ctx := context.Background()

			// Test with CC
			ccResp, err := ccClient.Execute(ctx, tc.Prompt, "claude-sonnet-4-6")
			if err != nil {
				t.Errorf("CC execution failed: %v", err)
			}
			if len(ccResp.Content) < 10 {
				t.Errorf("CC response too short: %s", ccResp.Content)
			}

			// Test with Poyo
			poyoResp, err := poyoClient.Execute(ctx, tc.Prompt, "claude-sonnet-4-6")
			if err != nil {
				t.Errorf("Poyo execution failed: %v", err)
			}
			if len(poyoResp.Content) < 10 {
				t.Errorf("Poyo response too short: %s", poyoResp.Content)
			}

			t.Logf("CC Response: %s...", truncate(ccResp.Content, 100))
			t.Logf("Poyo Response: %s...", truncate(poyoResp.Content, 100))
		})
	}
}

// TestCaseCategories tests each category independently
func TestCaseCategories(t *testing.T) {
	categories := map[string]int{
		"basic_conversation": 3,
		"code_generation":    3,
		"tool_usage":         3,
		"reasoning":          2,
		"code_analysis":      2,
		"plugin_system":      2,
		"error_handling":     2,
		"complex_tasks":      2,
		"context_awareness":  1,
	}

	testCases := e2e.DefaultTestCases()
	categoryCount := make(map[string]int)

	for _, tc := range testCases {
		categoryCount[tc.Category]++
	}

	for cat, expected := range categories {
		if categoryCount[cat] != expected {
			t.Errorf("Category %s: expected %d cases, got %d", cat, expected, categoryCount[cat])
		}
	}

	// Verify total
	totalExpected := 0
	for _, count := range categories {
		totalExpected += count
	}
	if len(testCases) != totalExpected {
		t.Errorf("Expected %d total cases, got %d", totalExpected, len(testCases))
	}
}

// TestComparisonMetrics tests the comparison metrics calculation
func TestComparisonMetrics(t *testing.T) {
	ccClient := e2e.NewSimulatedClient("CC", 100*time.Millisecond, 0.0)
	poyoClient := e2e.NewSimulatedClient("Poyo", 80*time.Millisecond, 0.0)

	runner := e2e.NewTestRunner(ccClient, poyoClient, "claude-sonnet-4-6")
	ctx := context.Background()

	report, err := runner.RunAll(ctx)
	if err != nil {
		t.Fatalf("Failed to run tests: %v", err)
	}

	// Check metrics are calculated
	if report.Summary.CCSuccessRate < 0 || report.Summary.CCSuccessRate > 1 {
		t.Errorf("Invalid CC success rate: %f", report.Summary.CCSuccessRate)
	}
	if report.Summary.PoyoSuccessRate < 0 || report.Summary.PoyoSuccessRate > 1 {
		t.Errorf("Invalid Poyo success rate: %f", report.Summary.PoyoSuccessRate)
	}
	if report.Summary.CCAvgDuration <= 0 {
		t.Errorf("Invalid CC avg duration: %f", report.Summary.CCAvgDuration)
	}
	if report.Summary.PoyoAvgDuration <= 0 {
		t.Errorf("Invalid Poyo avg duration: %f", report.Summary.PoyoAvgDuration)
	}

	// Check winner determination
	validWinners := map[string]bool{"cc": true, "poyo": true, "tie": true}
	if !validWinners[report.Summary.Winner] {
		t.Errorf("Invalid winner: %s", report.Summary.Winner)
	}

	// Check category scores
	if len(report.CategoryScores) == 0 {
		t.Error("No category scores calculated")
	}
}

// Helper function
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
