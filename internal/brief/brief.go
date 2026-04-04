// Package brief provides report generation and file attachment capabilities.
package brief

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportType represents the type of report to generate
type ReportType string

const (
	ReportTypeSummary   ReportType = "summary"
	ReportTypeDetailed  ReportType = "detailed"
	ReportTypeTechnical ReportType = "technical"
	ReportTypeProgress  ReportType = "progress"
	ReportTypeAnalysis  ReportType = "analysis"
)

// OutputFormat represents the output format for reports
type OutputFormat string

const (
	FormatMarkdown OutputFormat = "markdown"
	FormatHTML     OutputFormat = "html"
	FormatText     OutputFormat = "text"
	FormatJSON     OutputFormat = "json"
)

// ReportSection represents a section in the report
type ReportSection struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Level       int         `json:"level"` // 1 = h1, 2 = h2, etc.
	Subsections []ReportSection `json:"subsections,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReportMetadata contains report metadata
type ReportMetadata struct {
	Title       string    `json:"title"`
	Author      string    `json:"author,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     string    `json:"version,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Project     string    `json:"project,omitempty"`
	Environment string    `json:"environment,omitempty"`
}

// Report represents a generated report
type Report struct {
	ID          string          `json:"id"`
	Type        ReportType      `json:"type"`
	Format      OutputFormat    `json:"format"`
	Metadata    ReportMetadata  `json:"metadata"`
	Sections    []ReportSection `json:"sections"`
	Summary     string          `json:"summary,omitempty"`
	Attachments []Attachment    `json:"attachments,omitempty"`
	FilePath    string          `json:"file_path,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Size        int64             `json:"size"`
	MimeType    string            `json:"mime_type"`
	Checksum    string            `json:"checksum,omitempty"`
	UploadedAt  time.Time         `json:"uploaded_at"`
	URL         string            `json:"url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BriefRequest represents a request to generate a report
type BriefRequest struct {
	Type        ReportType   `json:"type"`
	Format      OutputFormat `json:"format"`
	Title       string       `json:"title"`
	Content     string       `json:"content,omitempty"`
	Sections    []ReportSection `json:"sections,omitempty"`
	Template    string       `json:"template,omitempty"`
	OutputPath  string       `json:"output_path,omitempty"`
	Attachments []string     `json:"attachments,omitempty"`
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	FilePath    string `json:"file_path"`
	Destination string `json:"destination,omitempty"`
	Public      bool   `json:"public,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BriefTool provides report generation and file upload capabilities
type BriefTool struct {
	name        string
	description string
	uploader    FileUploader
}

// FileUploader defines the interface for file upload
type FileUploader interface {
	Upload(ctx context.Context, filePath string, public bool) (*Attachment, error)
	GetURL(attachmentID string) (string, error)
	Delete(attachmentID string) error
}

// NewBriefTool creates a new BriefTool
func NewBriefTool(uploader FileUploader) *BriefTool {
	return &BriefTool{
		name:        "Brief",
		description: "Generate reports and upload file attachments",
		uploader:    uploader,
	}
}

// Name returns the tool name
func (t *BriefTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BriefTool) Description() string {
	return t.description
}

// InputSchema returns the JSON schema for tool input
func (t *BriefTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"generate", "upload", "list", "delete"},
				"description": "Action to perform",
			},
			"report": map[string]interface{}{
				"type":        "object",
				"description": "Report configuration",
				"properties": map[string]interface{}{
					"type":        map[string]interface{}{"type": "string", "enum": []string{"summary", "detailed", "technical", "progress", "analysis"}},
					"format":      map[string]interface{}{"type": "string", "enum": []string{"markdown", "html", "text", "json"}},
					"title":       map[string]interface{}{"type": "string"},
					"content":     map[string]interface{}{"type": "string"},
					"output_path": map[string]interface{}{"type": "string"},
				},
			},
			"upload": map[string]interface{}{
				"type":        "object",
				"description": "Upload configuration",
				"properties": map[string]interface{}{
					"file_path":   map[string]interface{}{"type": "string"},
					"destination": map[string]interface{}{"type": "string"},
					"public":      map[string]interface{}{"type": "boolean"},
				},
			},
			"attachment_id": map[string]interface{}{
				"type":        "string",
				"description": "Attachment ID for delete/get operations",
			},
		},
		"required": []string{"action"},
	}
}

// ToolInput represents the parsed tool input
type ToolInput struct {
	Action       string        `json:"action"`
	Report       *BriefRequest `json:"report,omitempty"`
	Upload       *UploadRequest `json:"upload,omitempty"`
	AttachmentID string        `json:"attachment_id,omitempty"`
}

// Execute executes the tool
func (t *BriefTool) Execute(ctx context.Context, input []byte) (interface{}, error) {
	var req ToolInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	switch req.Action {
	case "generate":
		return t.generateReport(ctx, req.Report)
	case "upload":
		return t.uploadFile(ctx, req.Upload)
	case "list":
		return t.listAttachments()
	case "delete":
		return t.deleteAttachment(req.AttachmentID)
	default:
		return nil, fmt.Errorf("unknown action: %s", req.Action)
	}
}

// generateReport generates a report
func (t *BriefTool) generateReport(ctx context.Context, req *BriefRequest) (*Report, error) {
	if req == nil {
		return nil, fmt.Errorf("report configuration is required")
	}

	// Set defaults
	if req.Type == "" {
		req.Type = ReportTypeSummary
	}
	if req.Format == "" {
		req.Format = FormatMarkdown
	}

	// Create report
	report := &Report{
		ID:       generateReportID(),
		Type:     req.Type,
		Format:   req.Format,
		Metadata: ReportMetadata{
			Title:     req.Title,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Sections: req.Sections,
		Summary:  req.Content,
	}

	// Generate content based on format
	var content string
	switch req.Format {
	case FormatMarkdown:
		content = renderMarkdown(report)
	case FormatHTML:
		content = renderHTML(report)
	case FormatText:
		content = renderText(report)
	case FormatJSON:
		content = renderJSON(report)
	}

	// Save to file if path specified
	if req.OutputPath != "" {
		if err := ioutil.WriteFile(req.OutputPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write report: %w", err)
		}
		report.FilePath = req.OutputPath
	}

	return report, nil
}

// uploadFile uploads a file
func (t *BriefTool) uploadFile(ctx context.Context, req *UploadRequest) (*Attachment, error) {
	if req == nil {
		return nil, fmt.Errorf("upload configuration is required")
	}

	if req.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Check if file exists
	if _, err := os.Stat(req.FilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", req.FilePath)
	}

	// Use uploader if available
	if t.uploader != nil {
		return t.uploader.Upload(ctx, req.FilePath, req.Public)
	}

	// Default: create local attachment record
	return createLocalAttachment(req.FilePath)
}

// listAttachments lists all attachments
func (t *BriefTool) listAttachments() ([]Attachment, error) {
	// In real implementation, would query from storage
	return []Attachment{}, nil
}

// deleteAttachment deletes an attachment
func (t *BriefTool) deleteAttachment(attachmentID string) error {
	if attachmentID == "" {
		return fmt.Errorf("attachment_id is required")
	}

	if t.uploader != nil {
		return t.uploader.Delete(attachmentID)
	}

	return nil
}

// ReportBuilder provides a fluent API for building reports
type ReportBuilder struct {
	report *Report
}

// NewReportBuilder creates a new ReportBuilder
func NewReportBuilder() *ReportBuilder {
	return &ReportBuilder{
		report: &Report{
			ID:       generateReportID(),
			Sections: []ReportSection{},
			Metadata: ReportMetadata{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
}

// WithTitle sets the report title
func (b *ReportBuilder) WithTitle(title string) *ReportBuilder {
	b.report.Metadata.Title = title
	return b
}

// WithType sets the report type
func (b *ReportBuilder) WithType(reportType ReportType) *ReportBuilder {
	b.report.Type = reportType
	return b
}

// WithFormat sets the output format
func (b *ReportBuilder) WithFormat(format OutputFormat) *ReportBuilder {
	b.report.Format = format
	return b
}

// WithAuthor sets the author
func (b *ReportBuilder) WithAuthor(author string) *ReportBuilder {
	b.report.Metadata.Author = author
	return b
}

// WithTags sets the tags
func (b *ReportBuilder) WithTags(tags []string) *ReportBuilder {
	b.report.Metadata.Tags = tags
	return b
}

// AddSection adds a section to the report
func (b *ReportBuilder) AddSection(title, content string, level int) *ReportBuilder {
	b.report.Sections = append(b.report.Sections, ReportSection{
		ID:      generateSectionID(),
		Title:   title,
		Content: content,
		Level:   level,
	})
	return b
}

// WithSummary sets the report summary
func (b *ReportBuilder) WithSummary(summary string) *ReportBuilder {
	b.report.Summary = summary
	return b
}

// Build returns the built report
func (b *ReportBuilder) Build() *Report {
	b.report.Metadata.UpdatedAt = time.Now()
	return b.report
}

// Render renders the report in the specified format
func (r *Report) Render(format OutputFormat) string {
	switch format {
	case FormatMarkdown:
		return renderMarkdown(r)
	case FormatHTML:
		return renderHTML(r)
	case FormatText:
		return renderText(r)
	case FormatJSON:
		return renderJSON(r)
	default:
		return renderMarkdown(r)
	}
}

// renderMarkdown renders the report as Markdown
func renderMarkdown(r *Report) string {
	var sb strings.Builder

	// Title
	if r.Metadata.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", r.Metadata.Title))
	}

	// Metadata
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", r.Metadata.CreatedAt.Format(time.RFC3339)))

	// Summary
	if r.Summary != "" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString(r.Summary)
		sb.WriteString("\n\n")
	}

	// Sections
	for _, section := range r.Sections {
		sb.WriteString(renderSectionMarkdown(section))
	}

	// Attachments
	if len(r.Attachments) > 0 {
		sb.WriteString("## Attachments\n\n")
		for _, att := range r.Attachments {
			sb.WriteString(fmt.Sprintf("- [%s](%s) (%s)\n", att.Name, att.URL, formatSize(att.Size)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderSectionMarkdown renders a section as Markdown
func renderSectionMarkdown(section ReportSection) string {
	var sb strings.Builder

	prefix := strings.Repeat("#", section.Level+1)
	sb.WriteString(fmt.Sprintf("%s %s\n\n", prefix, section.Title))

	if section.Content != "" {
		sb.WriteString(section.Content)
		sb.WriteString("\n\n")
	}

	for _, sub := range section.Subsections {
		sb.WriteString(renderSectionMarkdown(sub))
	}

	return sb.String()
}

// renderHTML renders the report as HTML
func renderHTML(r *Report) string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", r.Metadata.Title))
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; margin: 40px; }\n")
	sb.WriteString("h1 { color: #333; }\n")
	sb.WriteString("h2 { color: #666; margin-top: 20px; }\n")
	sb.WriteString("pre { background: #f5f5f5; padding: 10px; overflow-x: auto; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n")

	if r.Metadata.Title != "" {
		sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", r.Metadata.Title))
	}

	sb.WriteString(fmt.Sprintf("<p><em>Generated: %s</em></p>\n", r.Metadata.CreatedAt.Format(time.RFC3339)))

	if r.Summary != "" {
		sb.WriteString("<h2>Summary</h2>\n")
		sb.WriteString(fmt.Sprintf("<p>%s</p>\n", r.Summary))
	}

	for _, section := range r.Sections {
		sb.WriteString(renderSectionHTML(section))
	}

	sb.WriteString("</body>\n</html>")
	return sb.String()
}

// renderSectionHTML renders a section as HTML
func renderSectionHTML(section ReportSection) string {
	var sb strings.Builder

	tag := fmt.Sprintf("h%d", section.Level+1)
	sb.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, section.Title, tag))

	if section.Content != "" {
		sb.WriteString(fmt.Sprintf("<div>%s</div>\n", section.Content))
	}

	for _, sub := range section.Subsections {
		sb.WriteString(renderSectionHTML(sub))
	}

	return sb.String()
}

// renderText renders the report as plain text
func renderText(r *Report) string {
	var sb strings.Builder

	if r.Metadata.Title != "" {
		sb.WriteString(r.Metadata.Title)
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("=", len(r.Metadata.Title)))
		sb.WriteString("\n\n")
	}

	if r.Summary != "" {
		sb.WriteString("Summary:\n")
		sb.WriteString(r.Summary)
		sb.WriteString("\n\n")
	}

	for _, section := range r.Sections {
		sb.WriteString(renderSectionText(section))
	}

	return sb.String()
}

// renderSectionText renders a section as plain text
func renderSectionText(section ReportSection) string {
	var sb strings.Builder

	sb.WriteString(section.Title)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(section.Title)))
	sb.WriteString("\n\n")

	if section.Content != "" {
		sb.WriteString(section.Content)
		sb.WriteString("\n\n")
	}

	for _, sub := range section.Subsections {
		sb.WriteString(renderSectionText(sub))
	}

	return sb.String()
}

// renderJSON renders the report as JSON
func renderJSON(r *Report) string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// Helper functions

func generateReportID() string {
	return fmt.Sprintf("report-%d", time.Now().UnixNano())
}

func generateSectionID() string {
	return fmt.Sprintf("section-%d", time.Now().UnixNano())
}

func generateAttachmentID() string {
	return fmt.Sprintf("att-%d", time.Now().UnixNano())
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func createLocalAttachment(filePath string) (*Attachment, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &Attachment{
		ID:         generateAttachmentID(),
		Name:       filepath.Base(filePath),
		Path:       filePath,
		Size:       info.Size(),
		MimeType:   detectMimeType(filePath),
		UploadedAt: time.Now(),
	}, nil
}

func detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".zip":  "application/zip",
		".json": "application/json",
		".csv":  "text/csv",
	}
	if mt, ok := mimeTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

// LocalFileUploader is a simple file uploader that stores files locally
type LocalFileUploader struct {
	baseDir string
}

// NewLocalFileUploader creates a new LocalFileUploader
func NewLocalFileUploader(baseDir string) *LocalFileUploader {
	return &LocalFileUploader{baseDir: baseDir}
}

// Upload uploads a file to local storage
func (u *LocalFileUploader) Upload(ctx context.Context, filePath string, public bool) (*Attachment, error) {
	// Open source file
	src, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Get file info
	info, err := src.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Create destination path
	attID := generateAttachmentID()
	destDir := filepath.Join(u.baseDir, attID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(filePath))

	// Copy file
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	return &Attachment{
		ID:         attID,
		Name:       filepath.Base(filePath),
		Path:       destPath,
		Size:       info.Size(),
		MimeType:   detectMimeType(filePath),
		UploadedAt: time.Now(),
	}, nil
}

// GetURL returns the URL for an attachment
func (u *LocalFileUploader) GetURL(attachmentID string) (string, error) {
	return fmt.Sprintf("file://%s/%s", u.baseDir, attachmentID), nil
}

// Delete deletes an attachment
func (u *LocalFileUploader) Delete(attachmentID string) error {
	dir := filepath.Join(u.baseDir, attachmentID)
	return os.RemoveAll(dir)
}
