// Package audit provides security auditing functionality
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditLogger logs security-relevant events
type AuditLogger struct {
	mu       sync.Mutex
	logPath  string
	logFile  *os.File
	enabled  bool
	level    AuditLevel
	maxSize  int64
	retention time.Duration
}

// AuditLevel represents the audit logging level
type AuditLevel string

const (
	AuditLevelOff     AuditLevel = "off"
	AuditLevelMinimal AuditLevel = "minimal"
	AuditLevelNormal  AuditLevel = "normal"
	AuditLevelVerbose AuditLevel = "verbose"
)

// AuditEvent represents an audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      AuditEventType         `json:"type"`
	Level     AuditLevel             `json:"level"`
	SessionID string                 `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Tool      string                 `json:"tool,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	Result    string                 `json:"result"` // allowed, denied, error
	Reason    string                 `json:"reason,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
}

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventToolUse        AuditEventType = "tool_use"
	AuditEventPermission     AuditEventType = "permission"
	AuditEventAuthentication AuditEventType = "authentication"
	AuditEventAuthorization  AuditEventType = "authorization"
	AuditEventDataAccess     AuditEventType = "data_access"
	AuditEventConfigChange   AuditEventType = "config_change"
	AuditEventSession        AuditEventType = "session"
	AuditEventError          AuditEventType = "error"
	AuditEventSecurity       AuditEventType = "security"
)

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string, level AuditLevel) (*AuditLogger, error) {
	// Ensure log directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create audit log directory: %w", err)
	}

	logger := &AuditLogger{
		logPath:   logPath,
		enabled:   level != AuditLevelOff,
		level:     level,
		maxSize:   100 * 1024 * 1024, // 100MB
		retention: 30 * 24 * time.Hour, // 30 days
	}

	if logger.enabled {
		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("open audit log: %w", err)
		}
		logger.logFile = file
	}

	return logger, nil
}

// Log logs an audit event
func (al *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	if !al.enabled {
		return nil
	}

	// Check level filter
	if !al.shouldLog(event.Level) {
		return nil
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Marshal event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	// Check for log rotation
	if err := al.rotateIfNeeded(); err != nil {
		// Log rotation failed, but continue logging
		fmt.Fprintf(os.Stderr, "Audit log rotation failed: %v\n", err)
	}

	// Write event
	_, err = al.logFile.WriteString(string(data) + "\n")
	if err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}

// shouldLog checks if an event should be logged based on level
func (al *AuditLogger) shouldLog(level AuditLevel) bool {
	levels := map[AuditLevel]int{
		AuditLevelOff:     0,
		AuditLevelMinimal: 1,
		AuditLevelNormal:  2,
		AuditLevelVerbose: 3,
	}

	return levels[level] <= levels[al.level]
}

// rotateIfNeeded rotates the log file if it exceeds max size
func (al *AuditLogger) rotateIfNeeded() error {
	info, err := al.logFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() < al.maxSize {
		return nil
	}

	// Close current file
	al.logFile.Close()

	// Rename current file
	rotatedPath := fmt.Sprintf("%s.%s", al.logPath, time.Now().Format("20060102-150405"))
	if err := os.Rename(al.logPath, rotatedPath); err != nil {
		return err
	}

	// Open new file
	file, err := os.OpenFile(al.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	al.logFile = file

	// Clean up old log files
	go al.cleanupOldLogs()

	return nil
}

// cleanupOldLogs removes log files older than retention period
func (al *AuditLogger) cleanupOldLogs() {
	dir := filepath.Dir(al.logPath)
	base := filepath.Base(al.logPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-al.retention)

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), base+".") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(dir, entry.Name()))
			}
		}
	}
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.logFile != nil {
		return al.logFile.Close()
	}
	return nil
}

// SetLevel sets the audit level
func (al *AuditLogger) SetLevel(level AuditLevel) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.level = level
	al.enabled = level != AuditLevelOff
}

// SecurityAuditor performs security audits
type SecurityAuditor struct {
	logger   *AuditLogger
	checks   []SecurityCheck
	findings []SecurityFinding
	mu       sync.RWMutex
}

// SecurityCheck represents a security check
type SecurityCheck struct {
	ID          string
	Name        string
	Description string
	Severity    Severity
	CheckFunc   func(ctx context.Context) ([]SecurityFinding, error)
}

// Severity represents the severity of a finding
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// SecurityFinding represents a security finding
type SecurityFinding struct {
	ID          string                 `json:"id"`
	CheckID     string                 `json:"check_id"`
	Severity    Severity               `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Remediation string                 `json:"remediation"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(logger *AuditLogger) *SecurityAuditor {
	auditor := &SecurityAuditor{
		logger:   logger,
		findings: make([]SecurityFinding, 0),
	}

	// Register default checks
	auditor.RegisterDefaultChecks()

	return auditor
}

// RegisterCheck registers a security check
func (sa *SecurityAuditor) RegisterCheck(check SecurityCheck) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.checks = append(sa.checks, check)
}

// RegisterDefaultChecks registers default security checks
func (sa *SecurityAuditor) RegisterDefaultChecks() {
	sa.checks = []SecurityCheck{
		{
			ID:          "sensitive_files",
			Name:        "Sensitive File Access",
			Description: "Check for access to sensitive files",
			Severity:    SeverityHigh,
			CheckFunc:   sa.checkSensitiveFiles,
		},
		{
			ID:          "dangerous_commands",
			Name:        "Dangerous Commands",
			Description: "Check for execution of dangerous commands",
			Severity:    SeverityCritical,
			CheckFunc:   sa.checkDangerousCommands,
		},
		{
			ID:          "permission_bypass",
			Name:        "Permission Bypass",
			Description: "Check for permission bypass attempts",
			Severity:    SeverityCritical,
			CheckFunc:   sa.checkPermissionBypass,
		},
		{
			ID:          "excessive_permissions",
			Name:        "Excessive Permissions",
			Description: "Check for excessive permission grants",
			Severity:    SeverityMedium,
			CheckFunc:   sa.checkExcessivePermissions,
		},
	}
}

// RunAudit runs all security checks
func (sa *SecurityAuditor) RunAudit(ctx context.Context) ([]SecurityFinding, error) {
	sa.mu.RLock()
	checks := make([]SecurityCheck, len(sa.checks))
	copy(checks, sa.checks)
	sa.mu.RUnlock()

	var allFindings []SecurityFinding

	for _, check := range checks {
		findings, err := check.CheckFunc(ctx)
		if err != nil {
			// Log error but continue
			if sa.logger != nil {
				sa.logger.Log(ctx, &AuditEvent{
					Type:     AuditEventError,
					Level:    AuditLevelNormal,
					Result:   "error",
					Reason:   fmt.Sprintf("Security check %s failed: %v", check.ID, err),
				})
			}
			continue
		}

		for i := range findings {
			findings[i].CheckID = check.ID
			findings[i].Timestamp = time.Now()
			findings[i].ID = fmt.Sprintf("finding_%s_%d", check.ID, time.Now().UnixNano())
		}

		allFindings = append(allFindings, findings...)
	}

	sa.mu.Lock()
	sa.findings = append(sa.findings, allFindings...)
	sa.mu.Unlock()

	return allFindings, nil
}

// GetFindings returns all findings
func (sa *SecurityAuditor) GetFindings() []SecurityFinding {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.findings
}

// ClearFindings clears all findings
func (sa *SecurityAuditor) ClearFindings() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.findings = make([]SecurityFinding, 0)
}

// Default check implementations

func (sa *SecurityAuditor) checkSensitiveFiles(ctx context.Context) ([]SecurityFinding, error) {
	// This would check for access to sensitive files
	// Placeholder implementation
	return []SecurityFinding{}, nil
}

func (sa *SecurityAuditor) checkDangerousCommands(ctx context.Context) ([]SecurityFinding, error) {
	// This would check for dangerous command execution
	// Placeholder implementation
	return []SecurityFinding{}, nil
}

func (sa *SecurityAuditor) checkPermissionBypass(ctx context.Context) ([]SecurityFinding, error) {
	// This would check for permission bypass attempts
	// Placeholder implementation
	return []SecurityFinding{}, nil
}

func (sa *SecurityAuditor) checkExcessivePermissions(ctx context.Context) ([]SecurityFinding, error) {
	// This would check for excessive permission grants
	// Placeholder implementation
	return []SecurityFinding{}, nil
}

// AuditReport represents an audit report
type AuditReport struct {
	GeneratedAt   time.Time         `json:"generated_at"`
	Summary       AuditSummary      `json:"summary"`
	Findings      []SecurityFinding `json:"findings"`
	Recommendations []string        `json:"recommendations"`
}

// AuditSummary represents audit summary
type AuditSummary struct {
	TotalFindings int `json:"total_findings"`
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	Info          int `json:"info"`
}

// GenerateReport generates an audit report
func (sa *SecurityAuditor) GenerateReport() *AuditReport {
	findings := sa.GetFindings()

	summary := AuditSummary{
		TotalFindings: len(findings),
	}

	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityHigh:
			summary.High++
		case SeverityMedium:
			summary.Medium++
		case SeverityLow:
			summary.Low++
		case SeverityInfo:
			summary.Info++
		}
	}

	report := &AuditReport{
		GeneratedAt: time.Now(),
		Summary:     summary,
		Findings:    findings,
	}

	// Generate recommendations based on findings
	if summary.Critical > 0 {
		report.Recommendations = append(report.Recommendations,
			"Immediate attention required for critical security findings")
	}
	if summary.High > 0 {
		report.Recommendations = append(report.Recommendations,
			"High severity findings should be addressed within 24 hours")
	}
	if summary.Medium > 0 {
		report.Recommendations = append(report.Recommendations,
			"Medium severity findings should be addressed within a week")
	}

	return report
}
