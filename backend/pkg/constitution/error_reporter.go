package constitution

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ErrorReport represents a comprehensive error report
type ErrorReport struct {
	ReportID              string             `json:"report_id"`
	TaskID                string             `json:"task_id,omitempty"`
	Timestamp             time.Time          `json:"timestamp"`
	Error                 *ConstitutionError `json:"error"`
	RecoveryAction        *RecoveryAction    `json:"recovery_action,omitempty"`
	RecoveryResult        *RecoveryResult    `json:"recovery_result,omitempty"`
	ConstitutionReference string             `json:"constitution_reference,omitempty"`
	CodeContext           *CodeContext       `json:"code_context,omitempty"`
	RelatedFiles          []string           `json:"related_files,omitempty"`
	FixSuggestions        []FixSuggestion    `json:"fix_suggestions,omitempty"`
	DeveloperNotes        string             `json:"developer_notes,omitempty"`
}

// CodeContext represents the code context around an error
type CodeContext struct {
	File          string   `json:"file"`
	StartLine     int      `json:"start_line"`
	EndLine       int      `json:"end_line"`
	Lines         []string `json:"lines"`
	HighlightLine int      `json:"highlight_line"`
}

// FixSuggestion represents a suggested fix for an error
type FixSuggestion struct {
	Priority    int      `json:"priority"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	CodeExample string   `json:"code_example,omitempty"`
	Reference   string   `json:"reference,omitempty"`
}

// ErrorReporter generates error reports
type ErrorReporter interface {
	// GenerateReport generates a comprehensive error report
	GenerateReport(err *ConstitutionError, taskID string) (*ErrorReport, error)

	// GenerateJSONReport generates a JSON-formatted error report
	GenerateJSONReport(report *ErrorReport) (string, error)

	// GenerateTextReport generates a human-readable text error report
	GenerateTextReport(report *ErrorReport) (string, error)

	// GenerateMarkdownReport generates a Markdown-formatted error report
	GenerateMarkdownReport(report *ErrorReport) (string, error)

	// AddCodeContext adds code context to an error report
	AddCodeContext(report *ErrorReport, filePath string, line int, contextLines int) error

	// AddFixSuggestions adds fix suggestions to an error report
	AddFixSuggestions(report *ErrorReport, suggestions []FixSuggestion) error

	// SaveReport saves an error report to a file
	SaveReport(report *ErrorReport, outputPath string) error
}

// errorReporter implements ErrorReporter
type errorReporter struct {
	config       *Config
	ruleEngine   RuleEngine
	errorHandler ErrorHandler
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(config *Config, ruleEngine RuleEngine, errorHandler ErrorHandler) ErrorReporter {
	return &errorReporter{
		config:       config,
		ruleEngine:   ruleEngine,
		errorHandler: errorHandler,
	}
}

// GenerateReport generates a comprehensive error report
func (r *errorReporter) GenerateReport(err *ConstitutionError, taskID string) (*ErrorReport, error) {
	report := &ErrorReport{
		ReportID:              generateReportID(),
		TaskID:                taskID,
		Timestamp:             time.Now(),
		Error:                 err,
		ConstitutionReference: err.ConstitutionReference,
		RelatedFiles:          []string{},
		FixSuggestions:        []FixSuggestion{},
	}

	// Add file to related files if present
	if err.File != "" {
		report.RelatedFiles = append(report.RelatedFiles, err.File)
	}

	// Convert error fix suggestions to structured fix suggestions
	for i, suggestion := range err.FixSuggestions {
		report.FixSuggestions = append(report.FixSuggestions, FixSuggestion{
			Priority:    i + 1,
			Description: suggestion,
			Steps:       []string{suggestion},
		})
	}

	// Add code context if file and line are available
	if err.File != "" && err.Line > 0 {
		_ = r.AddCodeContext(report, err.File, err.Line, 3)
	}

	// Generate additional fix suggestions based on error category
	additionalSuggestions := r.generateCategorySpecificSuggestions(err)
	for _, suggestion := range additionalSuggestions {
		report.FixSuggestions = append(report.FixSuggestions, suggestion)
	}

	return report, nil
}

// GenerateJSONReport generates a JSON-formatted error report
func (r *errorReporter) GenerateJSONReport(report *ErrorReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	return string(data), nil
}

// GenerateTextReport generates a human-readable text error report
func (r *errorReporter) GenerateTextReport(report *ErrorReport) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("=" + strings.Repeat("=", 78) + "\n")
	sb.WriteString(fmt.Sprintf("ERROR REPORT: %s\n", report.ReportID))
	sb.WriteString("=" + strings.Repeat("=", 78) + "\n\n")

	// Basic information
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339)))
	if report.TaskID != "" {
		sb.WriteString(fmt.Sprintf("Task ID: %s\n", report.TaskID))
	}
	sb.WriteString("\n")

	// Error details
	sb.WriteString("ERROR DETAILS\n")
	sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
	sb.WriteString(fmt.Sprintf("Category: %s\n", report.Error.Category))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", report.Error.Severity))
	sb.WriteString(fmt.Sprintf("Message: %s\n", report.Error.Message))
	if report.Error.Details != "" {
		sb.WriteString(fmt.Sprintf("Details: %s\n", report.Error.Details))
	}
	if report.Error.File != "" {
		sb.WriteString(fmt.Sprintf("File: %s", report.Error.File))
		if report.Error.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", report.Error.Line))
			if report.Error.Column > 0 {
				sb.WriteString(fmt.Sprintf(":%d", report.Error.Column))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Constitution reference
	if report.ConstitutionReference != "" {
		sb.WriteString("CONSTITUTION REFERENCE\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", report.ConstitutionReference))
	}

	// Code context
	if report.CodeContext != nil {
		sb.WriteString("CODE CONTEXT\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		sb.WriteString(fmt.Sprintf("File: %s (lines %d-%d)\n\n",
			report.CodeContext.File,
			report.CodeContext.StartLine,
			report.CodeContext.EndLine))

		for i, line := range report.CodeContext.Lines {
			lineNum := report.CodeContext.StartLine + i
			prefix := "  "
			if lineNum == report.CodeContext.HighlightLine {
				prefix = "→ "
			}
			sb.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, lineNum, line))
		}
		sb.WriteString("\n")
	}

	// Fix suggestions
	if len(report.FixSuggestions) > 0 {
		sb.WriteString("FIX SUGGESTIONS\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		for _, suggestion := range report.FixSuggestions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", suggestion.Priority, suggestion.Description))
			if len(suggestion.Steps) > 0 {
				for _, step := range suggestion.Steps {
					sb.WriteString(fmt.Sprintf("   - %s\n", step))
				}
			}
			if suggestion.CodeExample != "" {
				sb.WriteString(fmt.Sprintf("   Example:\n%s\n", indentCode(suggestion.CodeExample, 6)))
			}
			if suggestion.Reference != "" {
				sb.WriteString(fmt.Sprintf("   Reference: %s\n", suggestion.Reference))
			}
			sb.WriteString("\n")
		}
	}

	// Recovery action
	if report.RecoveryAction != nil {
		sb.WriteString("RECOVERY ACTION\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		sb.WriteString(fmt.Sprintf("Strategy: %s\n", report.RecoveryAction.Strategy))
		sb.WriteString(fmt.Sprintf("Description: %s\n", report.RecoveryAction.Description))
		sb.WriteString(fmt.Sprintf("Auto Execute: %v\n", report.RecoveryAction.AutoExecute))
		if len(report.RecoveryAction.Steps) > 0 {
			sb.WriteString("Steps:\n")
			for i, step := range report.RecoveryAction.Steps {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
			}
		}
		sb.WriteString("\n")
	}

	// Recovery result
	if report.RecoveryResult != nil {
		sb.WriteString("RECOVERY RESULT\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		sb.WriteString(fmt.Sprintf("Success: %v\n", report.RecoveryResult.Success))
		sb.WriteString(fmt.Sprintf("Strategy: %s\n", report.RecoveryResult.Strategy))
		sb.WriteString(fmt.Sprintf("Attempts: %d\n", report.RecoveryResult.AttemptsCount))
		sb.WriteString(fmt.Sprintf("Duration: %s\n", report.RecoveryResult.Duration))
		sb.WriteString(fmt.Sprintf("Message: %s\n", report.RecoveryResult.Message))
		sb.WriteString("\n")
	}

	// Developer notes
	if report.DeveloperNotes != "" {
		sb.WriteString("DEVELOPER NOTES\n")
		sb.WriteString("-" + strings.Repeat("-", 78) + "\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", report.DeveloperNotes))
	}

	// Footer
	sb.WriteString("=" + strings.Repeat("=", 78) + "\n")

	return sb.String(), nil
}

// GenerateMarkdownReport generates a Markdown-formatted error report
func (r *errorReporter) GenerateMarkdownReport(report *ErrorReport) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Error Report: %s\n\n", report.ReportID))

	// Basic information
	sb.WriteString("## Overview\n\n")
	sb.WriteString(fmt.Sprintf("- **Timestamp**: %s\n", report.Timestamp.Format(time.RFC3339)))
	if report.TaskID != "" {
		sb.WriteString(fmt.Sprintf("- **Task ID**: %s\n", report.TaskID))
	}
	sb.WriteString(fmt.Sprintf("- **Category**: %s\n", report.Error.Category))
	sb.WriteString(fmt.Sprintf("- **Severity**: %s\n", report.Error.Severity))
	sb.WriteString("\n")

	// Error details
	sb.WriteString("## Error Details\n\n")
	sb.WriteString(fmt.Sprintf("**Message**: %s\n\n", report.Error.Message))
	if report.Error.Details != "" {
		sb.WriteString(fmt.Sprintf("**Details**: %s\n\n", report.Error.Details))
	}
	if report.Error.File != "" {
		sb.WriteString(fmt.Sprintf("**Location**: `%s", report.Error.File))
		if report.Error.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", report.Error.Line))
			if report.Error.Column > 0 {
				sb.WriteString(fmt.Sprintf(":%d", report.Error.Column))
			}
		}
		sb.WriteString("`\n\n")
	}

	// Constitution reference
	if report.ConstitutionReference != "" {
		sb.WriteString("## Constitution Reference\n\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", report.ConstitutionReference))
	}

	// Code context
	if report.CodeContext != nil {
		sb.WriteString("## Code Context\n\n")
		sb.WriteString(fmt.Sprintf("File: `%s` (lines %d-%d)\n\n",
			report.CodeContext.File,
			report.CodeContext.StartLine,
			report.CodeContext.EndLine))
		sb.WriteString("```\n")
		for i, line := range report.CodeContext.Lines {
			lineNum := report.CodeContext.StartLine + i
			prefix := "  "
			if lineNum == report.CodeContext.HighlightLine {
				prefix = "→ "
			}
			sb.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, lineNum, line))
		}
		sb.WriteString("```\n\n")
	}

	// Fix suggestions
	if len(report.FixSuggestions) > 0 {
		sb.WriteString("## Fix Suggestions\n\n")
		for _, suggestion := range report.FixSuggestions {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", suggestion.Priority, suggestion.Description))
			if len(suggestion.Steps) > 0 {
				for _, step := range suggestion.Steps {
					sb.WriteString(fmt.Sprintf("- %s\n", step))
				}
				sb.WriteString("\n")
			}
			if suggestion.CodeExample != "" {
				sb.WriteString("**Example**:\n\n```\n")
				sb.WriteString(suggestion.CodeExample)
				sb.WriteString("\n```\n\n")
			}
			if suggestion.Reference != "" {
				sb.WriteString(fmt.Sprintf("**Reference**: %s\n\n", suggestion.Reference))
			}
		}
	}

	// Recovery action
	if report.RecoveryAction != nil {
		sb.WriteString("## Recovery Action\n\n")
		sb.WriteString(fmt.Sprintf("- **Strategy**: %s\n", report.RecoveryAction.Strategy))
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", report.RecoveryAction.Description))
		sb.WriteString(fmt.Sprintf("- **Auto Execute**: %v\n", report.RecoveryAction.AutoExecute))
		if len(report.RecoveryAction.Steps) > 0 {
			sb.WriteString("\n**Steps**:\n\n")
			for i, step := range report.RecoveryAction.Steps {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
			}
		}
		sb.WriteString("\n")
	}

	// Recovery result
	if report.RecoveryResult != nil {
		sb.WriteString("## Recovery Result\n\n")
		sb.WriteString(fmt.Sprintf("- **Success**: %v\n", report.RecoveryResult.Success))
		sb.WriteString(fmt.Sprintf("- **Strategy**: %s\n", report.RecoveryResult.Strategy))
		sb.WriteString(fmt.Sprintf("- **Attempts**: %d\n", report.RecoveryResult.AttemptsCount))
		sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", report.RecoveryResult.Duration))
		sb.WriteString(fmt.Sprintf("- **Message**: %s\n", report.RecoveryResult.Message))
		sb.WriteString("\n")
	}

	// Developer notes
	if report.DeveloperNotes != "" {
		sb.WriteString("## Developer Notes\n\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", report.DeveloperNotes))
	}

	return sb.String(), nil
}

// AddCodeContext adds code context to an error report
func (r *errorReporter) AddCodeContext(report *ErrorReport, filePath string, line int, contextLines int) error {
	// This is a placeholder implementation
	// In a real implementation, this would read the file and extract the relevant lines
	report.CodeContext = &CodeContext{
		File:          filePath,
		StartLine:     max(1, line-contextLines),
		EndLine:       line + contextLines,
		Lines:         []string{}, // Would be populated from file
		HighlightLine: line,
	}

	return nil
}

// AddFixSuggestions adds fix suggestions to an error report
func (r *errorReporter) AddFixSuggestions(report *ErrorReport, suggestions []FixSuggestion) error {
	report.FixSuggestions = append(report.FixSuggestions, suggestions...)
	return nil
}

// SaveReport saves an error report to a file
func (r *errorReporter) SaveReport(report *ErrorReport, outputPath string) error {
	// Generate JSON report
	jsonReport, err := r.GenerateJSONReport(report)
	if err != nil {
		return err
	}

	// In a real implementation, this would write to the file
	_ = jsonReport
	_ = outputPath

	return nil
}

// generateCategorySpecificSuggestions generates fix suggestions based on error category
func (r *errorReporter) generateCategorySpecificSuggestions(err *ConstitutionError) []FixSuggestion {
	suggestions := []FixSuggestion{}

	switch err.Category {
	case ErrorCategoryValidation:
		suggestions = append(suggestions, FixSuggestion{
			Priority:    len(err.FixSuggestions) + 1,
			Description: "Review and fix validation errors",
			Steps: []string{
				"Run the validator locally to see detailed errors",
				"Fix errors one by one",
				"Re-run validation to confirm fixes",
			},
			Reference: "Section 7: Validation Requirements",
		})

	case ErrorCategoryViolation:
		suggestions = append(suggestions, FixSuggestion{
			Priority:    len(err.FixSuggestions) + 1,
			Description: "Review constitution rules and fix violations",
			Steps: []string{
				"Read the referenced constitution section",
				"Understand the violated rule",
				"Modify code to comply with the rule",
				"Verify no other violations exist",
			},
			Reference: err.ConstitutionReference,
		})

	case ErrorCategoryHallucination:
		suggestions = append(suggestions, FixSuggestion{
			Priority:    len(err.FixSuggestions) + 1,
			Description: "Verify and create missing code elements",
			Steps: []string{
				"Search the codebase for the referenced element",
				"If not found, create the element in the appropriate location",
				"Update references to use the correct element",
				"Rebuild indexes if necessary",
			},
			Reference: "Section 4: Anti-Hallucination Rules",
		})

	case ErrorCategorySystem:
		suggestions = append(suggestions, FixSuggestion{
			Priority:    len(err.FixSuggestions) + 1,
			Description: "Investigate and resolve system issues",
			Steps: []string{
				"Check system logs for more details",
				"Verify system resources (disk, memory, network)",
				"Ensure required tools and dependencies are installed",
				"Retry the operation after fixing system issues",
			},
		})
	}

	return suggestions
}

// generateReportID generates a unique report ID
func generateReportID() string {
	return fmt.Sprintf("report-%d", time.Now().UnixNano())
}

// indentCode indents code by the specified number of spaces
func indentCode(code string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
