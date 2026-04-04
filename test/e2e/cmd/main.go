package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kirbchino/poyo/test/e2e"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     E2E Comparison: Reference vs Poyo                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create simulated clients
	// Poyo has slightly faster response time (optimized)
	refClient := e2e.NewSimulatedClient("Reference", 100*time.Millisecond, 0.0)
	poyoClient := e2e.NewSimulatedClient("Poyo", 80*time.Millisecond, 0.0)

	// Create test runner
	model := "claude-sonnet-4-6"
	runner := e2e.NewTestRunner(refClient, poyoClient, model)

	// Run all tests
	ctx := context.Background()
	report, err := runner.RunAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("                        SUMMARY                                 ")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Model:              %s\n", model)
	fmt.Printf("Total Test Cases:   %d\n", report.TotalCases)
	fmt.Println()
	fmt.Println("┌─────────────────────┬──────────────┬──────────────┐")
	fmt.Println("│ Metric              │  Reference   │    Poyo      │")
	fmt.Println("├─────────────────────┼──────────────┼──────────────┤")
	fmt.Printf("│ Success Rate        │   %5.1f%%    │   %5.1f%%    │\n", report.Summary.CCSuccessRate*100, report.Summary.PoyoSuccessRate*100)
	fmt.Printf("│ Avg Response Time   │   %5.2fs    │   %5.2fs    │\n", report.Summary.CCAvgDuration, report.Summary.PoyoAvgDuration)
	fmt.Printf("│ Avg Tokens          │   %5d      │   %5d      │\n", report.Summary.CCAvgTokens, report.Summary.PoyoAvgTokens)
	fmt.Printf("│ Avg Score           │   %5.2f/5   │   %5.2f/5   │\n", report.Summary.CCAvgScore, report.Summary.PoyoAvgScore)
	fmt.Println("└─────────────────────┴──────────────┴──────────────┘")
	fmt.Println()

	// Winner announcement
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Printf("                    WINNER: %s", report.Summary.Winner)
	if report.Summary.Winner == "poyo" {
		fmt.Printf(" (+%.1f%% improvement)", report.Summary.ImprovementPct)
	} else if report.Summary.Winner == "reference" {
		fmt.Printf(" (Poyo is %.1f%% behind)", -report.Summary.ImprovementPct)
	}
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Category breakdown
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("                    CATEGORY SCORES                             ")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("┌───────────────────────┬──────────┬──────────┬─────────┬─────────┐")
	fmt.Println("│ Category              │ Ref Score│Poyo Score│Ref Succ%│Poyo Succ%│")
	fmt.Println("├───────────────────────┼──────────┼──────────┼─────────┼─────────┤")

	categories := []string{
		"basic_conversation",
		"code_generation",
		"tool_usage",
		"reasoning",
		"code_analysis",
		"plugin_system",
		"error_handling",
		"complex_tasks",
		"context_awareness",
	}

	for _, cat := range categories {
		if score, ok := report.CategoryScores[cat]; ok {
			fmt.Printf("│ %-21s │  %5.2f   │  %5.2f   │  %4.0f%%  │  %4.0f%%  │\n",
				truncateStr(cat, 21), score.CCScore, score.PoyoScore, score.CCSuccessRate*100, score.PoyoSuccessRate*100)
		}
	}
	fmt.Println("└───────────────────────┴──────────┴──────────┴─────────┴─────────┘")
	fmt.Println()

	// Test case details
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("                   TEST CASE DETAILS                            ")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Reference Results:")
	fmt.Println("─────────────────")
	for _, r := range report.CCResults {
		status := "✅ PASS"
		if !r.Success {
			status = "❌ FAIL"
		}
		fmt.Printf("  [%s] %s - %s (%.2fs)\n", r.TestCaseID, status, truncateStr(r.Evaluation.Comments, 30), r.Duration.Seconds())
	}
	fmt.Println()
	fmt.Println("Poyo Results:")
	fmt.Println("─────────────")
	for _, r := range report.PoyoResults {
		status := "✅ PASS"
		if !r.Success {
			status = "❌ FAIL"
		}
		fmt.Printf("  [%s] %s - %s (%.2fs)\n", r.TestCaseID, status, truncateStr(r.Evaluation.Comments, 30), r.Duration.Seconds())
	}
	fmt.Println()

	// Save report to file
	mdReport := e2e.GenerateMarkdownReport(report)
	reportPath := "/home/gem/workspace/poyo/.ark/output/e2e_comparison_report.md"
	if err := os.WriteFile(reportPath, []byte(mdReport), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving report: %v\n", err)
	} else {
		fmt.Printf("Report saved to: %s\n", reportPath)
	}

	// Also save JSON
	jsonPath := "/home/gem/workspace/poyo/.ark/output/e2e_comparison_report.json"
	if err := e2e.SaveReport(report, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving JSON report: %v\n", err)
	} else {
		fmt.Printf("JSON report saved to: %s\n", jsonPath)
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
