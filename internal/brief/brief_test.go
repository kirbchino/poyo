package brief

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReportType(t *testing.T) {
	types := []ReportType{
		ReportTypeSummary,
		ReportTypeDetailed,
		ReportTypeTechnical,
		ReportTypeProgress,
		ReportTypeAnalysis,
	}

	for _, rt := range types {
		if rt == "" {
			t.Error("Report type should not be empty")
		}
	}
}

func TestOutputFormat(t *testing.T) {
	formats := []OutputFormat{
		FormatMarkdown,
		FormatHTML,
		FormatText,
		FormatJSON,
	}

	for _, f := range formats {
		if f == "" {
			t.Error("Output format should not be empty")
		}
	}
}

func TestNewReportBuilder(t *testing.T) {
	builder := NewReportBuilder()
	if builder == nil {
		t.Fatal("NewReportBuilder() returned nil")
	}

	report := builder.Build()
	if report == nil {
		t.Fatal("Build() returned nil")
	}

	if report.ID == "" {
		t.Error("Report should have ID")
	}
}

func TestReportBuilder(t *testing.T) {
	report := NewReportBuilder().
		WithTitle("Test Report").
		WithType(ReportTypeSummary).
		WithFormat(FormatMarkdown).
		WithAuthor("Test Author").
		WithTags([]string{"test", "report"}).
		AddSection("Introduction", "This is the introduction.", 1).
		AddSection("Details", "These are the details.", 2).
		WithSummary("This is a summary.").
		Build()

	if report.Metadata.Title != "Test Report" {
		t.Errorf("Title = %q, want 'Test Report'", report.Metadata.Title)
	}

	if report.Type != ReportTypeSummary {
		t.Errorf("Type = %v, want ReportTypeSummary", report.Type)
	}

	if report.Format != FormatMarkdown {
		t.Errorf("Format = %v, want FormatMarkdown", report.Format)
	}

	if report.Metadata.Author != "Test Author" {
		t.Errorf("Author = %q, want 'Test Author'", report.Metadata.Author)
	}

	if len(report.Metadata.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(report.Metadata.Tags))
	}

	if len(report.Sections) != 2 {
		t.Errorf("Sections count = %d, want 2", len(report.Sections))
	}

	if report.Summary != "This is a summary." {
		t.Errorf("Summary = %q, want 'This is a summary.'", report.Summary)
	}
}

func TestReportRenderMarkdown(t *testing.T) {
	report := &Report{
		ID:     "test-1",
		Type:   ReportTypeSummary,
		Format: FormatMarkdown,
		Metadata: ReportMetadata{
			Title:     "Test Report",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Summary: "This is a summary.",
		Sections: []ReportSection{
			{ID: "s1", Title: "Section 1", Content: "Content 1", Level: 1},
			{ID: "s2", Title: "Section 2", Content: "Content 2", Level: 2},
		},
	}

	markdown := report.Render(FormatMarkdown)

	if markdown == "" {
		t.Error("Render() returned empty string")
	}

	// Check for title
	if !containsSubstring(markdown, "# Test Report") {
		t.Error("Markdown should contain title")
	}

	// Check for summary
	if !containsSubstring(markdown, "This is a summary") {
		t.Error("Markdown should contain summary")
	}

	// Check for sections
	if !containsSubstring(markdown, "Section 1") {
		t.Error("Markdown should contain Section 1")
	}
}

func TestReportRenderHTML(t *testing.T) {
	report := &Report{
		ID:     "test-1",
		Type:   ReportTypeSummary,
		Format: FormatHTML,
		Metadata: ReportMetadata{
			Title:     "Test Report",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Summary: "This is a summary.",
		Sections: []ReportSection{
			{ID: "s1", Title: "Section 1", Content: "Content 1", Level: 1},
		},
	}

	html := report.Render(FormatHTML)

	if html == "" {
		t.Error("Render() returned empty string")
	}

	// Check for HTML structure
	if !containsSubstring(html, "<!DOCTYPE html>") {
		t.Error("HTML should have doctype")
	}

	if !containsSubstring(html, "<title>Test Report</title>") {
		t.Error("HTML should contain title")
	}

	if !containsSubstring(html, "<h1>Test Report</h1>") {
		t.Error("HTML should contain h1 title")
	}
}

func TestReportRenderText(t *testing.T) {
	report := &Report{
		ID:     "test-1",
		Type:   ReportTypeSummary,
		Format: FormatText,
		Metadata: ReportMetadata{
			Title:     "Test Report",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Summary: "This is a summary.",
	}

	text := report.Render(FormatText)

	if text == "" {
		t.Error("Render() returned empty string")
	}

	if !containsSubstring(text, "Test Report") {
		t.Error("Text should contain title")
	}

	if !containsSubstring(text, "Summary:") {
		t.Error("Text should contain Summary header")
	}
}

func TestReportRenderJSON(t *testing.T) {
	report := &Report{
		ID:     "test-1",
		Type:   ReportTypeSummary,
		Format: FormatJSON,
		Metadata: ReportMetadata{
			Title:     "Test Report",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Summary: "This is a summary.",
	}

	jsonStr := report.Render(FormatJSON)

	if jsonStr == "" {
		t.Error("Render() returned empty string")
	}

	// Verify it's valid JSON
	var parsed Report
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("Render() produced invalid JSON: %v", err)
	}
}

func TestBriefTool(t *testing.T) {
	tool := NewBriefTool(nil)

	if tool.Name() != "Brief" {
		t.Errorf("Name() = %q, want 'Brief'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("InputSchema() should not be nil")
	}
}

func TestBriefToolGenerateReport(t *testing.T) {
	tool := NewBriefTool(nil)

	input := `{
		"action": "generate",
		"report": {
			"type": "summary",
			"format": "markdown",
			"title": "Test Report",
			"content": "This is a test report."
		}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	report, ok := result.(*Report)
	if !ok {
		t.Fatal("Execute() should return *Report")
	}

	if report.Metadata.Title != "Test Report" {
		t.Errorf("Title = %q, want 'Test Report'", report.Metadata.Title)
	}

	if report.Summary != "This is a test report." {
		t.Errorf("Summary = %q, want 'This is a test report.'", report.Summary)
	}
}

func TestBriefToolGenerateReportWithOutput(t *testing.T) {
	tool := NewBriefTool(nil)

	// Create temp file
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "test-report.md")
	defer os.Remove(outputPath)

	input := `{
		"action": "generate",
		"report": {
			"type": "summary",
			"format": "markdown",
			"title": "Test Report",
			"content": "This is a test report.",
			"output_path": "` + outputPath + `"
		}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	report, ok := result.(*Report)
	if !ok {
		t.Fatal("Execute() should return *Report")
	}

	if report.FilePath != outputPath {
		t.Errorf("FilePath = %q, want %q", report.FilePath, outputPath)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Report file should be created")
	}
}

func TestBriefToolInvalidAction(t *testing.T) {
	tool := NewBriefTool(nil)

	input := `{"action": "invalid"}`

	_, err := tool.Execute(context.Background(), []byte(input))
	if err == nil {
		t.Error("Execute() should return error for invalid action")
	}
}

func TestBriefToolInvalidJSON(t *testing.T) {
	tool := NewBriefTool(nil)

	_, err := tool.Execute(context.Background(), []byte(`invalid`))
	if err == nil {
		t.Error("Execute() should return error for invalid JSON")
	}
}

func TestBriefToolUpload(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("test content")
	tmpFile.Close()

	tool := NewBriefTool(nil)

	input := `{
		"action": "upload",
		"upload": {
			"file_path": "` + tmpFile.Name() + `"
		}
	}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	att, ok := result.(*Attachment)
	if !ok {
		t.Fatal("Execute() should return *Attachment")
	}

	if att.Name == "" {
		t.Error("Attachment should have name")
	}

	if att.Size == 0 {
		t.Error("Attachment should have size")
	}
}

func TestBriefToolUploadNonExistent(t *testing.T) {
	tool := NewBriefTool(nil)

	input := `{
		"action": "upload",
		"upload": {
			"file_path": "/nonexistent/file.txt"
		}
	}`

	_, err := tool.Execute(context.Background(), []byte(input))
	if err == nil {
		t.Error("Execute() should return error for non-existent file")
	}
}

func TestLocalFileUploader(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "uploader-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "source", "test.txt")
	os.MkdirAll(filepath.Dir(testFile), 0755)
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	uploader := NewLocalFileUploader(filepath.Join(tmpDir, "uploads"))

	// Upload file
	att, err := uploader.Upload(context.Background(), testFile, false)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if att.ID == "" {
		t.Error("Attachment should have ID")
	}

	if att.Name != "test.txt" {
		t.Errorf("Name = %q, want 'test.txt'", att.Name)
	}

	// Get URL
	url, err := uploader.GetURL(att.ID)
	if err != nil {
		t.Fatalf("GetURL() error: %v", err)
	}

	if url == "" {
		t.Error("GetURL() should return URL")
	}

	// Delete
	err = uploader.Delete(att.ID)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.size)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.size, result, tt.expected)
		}
	}
}

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".txt", "text/plain"},
		{".md", "text/markdown"},
		{".pdf", "application/pdf"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".json", "application/json"},
		{".unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		result := detectMimeType("file" + tt.ext)
		if result != tt.expected {
			t.Errorf("detectMimeType(%q) = %q, want %q", tt.ext, result, tt.expected)
		}
	}
}

func TestReportSection(t *testing.T) {
	section := ReportSection{
		ID:      "s1",
		Title:   "Test Section",
		Content: "Test content",
		Level:   1,
		Subsections: []ReportSection{
			{ID: "s1-1", Title: "Subsection", Level: 2},
		},
	}

	if section.ID != "s1" {
		t.Errorf("ID = %q, want 's1'", section.ID)
	}

	if len(section.Subsections) != 1 {
		t.Errorf("Subsections count = %d, want 1", len(section.Subsections))
	}
}

func TestAttachment(t *testing.T) {
	att := Attachment{
		ID:        "att-1",
		Name:      "test.txt",
		Path:      "/path/to/test.txt",
		Size:      1024,
		MimeType:  "text/plain",
		UploadedAt: time.Now(),
	}

	if att.ID != "att-1" {
		t.Errorf("ID = %q, want 'att-1'", att.ID)
	}

	if att.Size != 1024 {
		t.Errorf("Size = %d, want 1024", att.Size)
	}
}

func TestReportWithAttachments(t *testing.T) {
	report := &Report{
		ID:     "test-1",
		Type:   ReportTypeSummary,
		Format: FormatMarkdown,
		Metadata: ReportMetadata{
			Title: "Test Report",
		},
		Attachments: []Attachment{
			{ID: "att-1", Name: "file1.txt", Size: 100, URL: "http://example.com/file1"},
			{ID: "att-2", Name: "file2.pdf", Size: 1000, URL: "http://example.com/file2"},
		},
	}

	markdown := report.Render(FormatMarkdown)

	if !containsSubstring(markdown, "Attachments") {
		t.Error("Markdown should contain Attachments section")
	}

	if !containsSubstring(markdown, "file1.txt") {
		t.Error("Markdown should contain attachment name")
	}
}

func TestBriefToolListAttachments(t *testing.T) {
	tool := NewBriefTool(nil)

	input := `{"action": "list"}`

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	attachments, ok := result.([]Attachment)
	if !ok {
		t.Fatal("Execute() should return []Attachment")
	}

	// Empty list expected
	if len(attachments) != 0 {
		t.Errorf("Attachments count = %d, want 0", len(attachments))
	}
}

func TestBriefToolDeleteAttachment(t *testing.T) {
	tool := NewBriefTool(nil)

	input := `{
		"action": "delete",
		"attachment_id": "att-123"
	}`

	_, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstringImpl(s, substr)))
}

func containsSubstringImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
