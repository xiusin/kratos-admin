package constitution

import (
	"context"
	"fmt"
	"time"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryValidation    ErrorCategory = "validation"
	ErrorCategoryViolation     ErrorCategory = "violation"
	ErrorCategoryHallucination ErrorCategory = "hallucination"
	ErrorCategorySystem        ErrorCategory = "system"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	ErrorSeverityWarning  ErrorSeverity = "warning"
	ErrorSeverityError    ErrorSeverity = "error"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// ConstitutionError represents a constitution-related error
type ConstitutionError struct {
	ID                    string                 `json:"id"`
	Category              ErrorCategory          `json:"category"`
	Severity              ErrorSeverity          `json:"severity"`
	Message               string                 `json:"message"`
	Details               string                 `json:"details"`
	File                  string                 `json:"file,omitempty"`
	Line                  int                    `json:"line,omitempty"`
	Column                int                    `json:"column,omitempty"`
	Timestamp             time.Time              `json:"timestamp"`
	ConstitutionReference string                 `json:"constitution_reference,omitempty"`
	CodeSnippet           string                 `json:"code_snippet,omitempty"`
	FixSuggestions        []string               `json:"fix_suggestions,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorHandler handles constitution errors
type ErrorHandler interface {
	// HandleValidationError handles validation errors
	HandleValidationError(ctx context.Context, err error, result *ValidationResult) (*ConstitutionError, error)

	// HandleViolationError handles constitution violation errors
	HandleViolationError(ctx context.Context, violation Violation) (*ConstitutionError, error)

	// HandleHallucinationError handles hallucination errors
	HandleHallucinationError(ctx context.Context, element string, elementType string) (*ConstitutionError, error)

	// HandleSystemError handles system errors
	HandleSystemError(ctx context.Context, err error, operation string) (*ConstitutionError, error)

	// ClassifyError classifies an error into a category
	ClassifyError(err error) ErrorCategory

	// DetermineSeverity determines the severity of an error
	DetermineSeverity(err *ConstitutionError) ErrorSeverity

	// ShouldRollback determines if an error should trigger a rollback
	ShouldRollback(err *ConstitutionError) bool

	// ShouldRetry determines if an operation should be retried
	ShouldRetry(err *ConstitutionError) bool
}

// errorHandler implements ErrorHandler
type errorHandler struct {
	config       *Config
	ruleEngine   *RuleEngine
	traceManager TaskTraceManager
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *Config, ruleEngine *RuleEngine, traceManager TaskTraceManager) ErrorHandler {
	return &errorHandler{
		config:       config,
		ruleEngine:   ruleEngine,
		traceManager: traceManager,
	}
}

// HandleValidationError handles validation errors
func (h *errorHandler) HandleValidationError(ctx context.Context, err error, result *ValidationResult) (*ConstitutionError, error) {
	constErr := &ConstitutionError{
		ID:        generateErrorID(),
		Category:  ErrorCategoryValidation,
		Message:   fmt.Sprintf("Validation failed: %v", err),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if result != nil {
		constErr.Details = result.Output
		constErr.Metadata["validator"] = result.Validator

		// Extract first error for details
		if len(result.Errors) > 0 {
			firstErr := result.Errors[0]
			constErr.File = firstErr.File
			constErr.Line = firstErr.Line
			constErr.Column = firstErr.Column
			constErr.Details = firstErr.Message
		}

		// Determine severity based on error count and type
		if len(result.Errors) > 10 {
			constErr.Severity = ErrorSeverityCritical
		} else if len(result.Errors) > 0 {
			constErr.Severity = ErrorSeverityError
		} else if len(result.Warnings) > 0 {
			constErr.Severity = ErrorSeverityWarning
		}

		// Generate fix suggestions
		constErr.FixSuggestions = h.generateValidationFixSuggestions(result)
	} else {
		constErr.Severity = ErrorSeverityError
		constErr.Details = err.Error()
	}

	constErr.ConstitutionReference = "Section 7: Validation Requirements"

	return constErr, nil
}

// HandleViolationError handles constitution violation errors
func (h *errorHandler) HandleViolationError(ctx context.Context, violation Violation) (*ConstitutionError, error) {
	constErr := &ConstitutionError{
		ID:                    generateErrorID(),
		Category:              ErrorCategoryViolation,
		Message:               violation.Description,
		Details:               violation.Description,
		File:                  violation.FilePath,
		Line:                  violation.LineNumber,
		Timestamp:             time.Now(),
		ConstitutionReference: violation.ConstitutionReference,
		CodeSnippet:           "", // Not available in Violation struct
		FixSuggestions:        []string{violation.Suggestion},
		Metadata:              make(map[string]interface{}),
	}

	// Map violation severity to error severity
	switch violation.Severity {
	case SeverityCritical:
		constErr.Severity = ErrorSeverityCritical
	case SeverityHigh:
		constErr.Severity = ErrorSeverityError
	case SeverityMedium:
		constErr.Severity = ErrorSeverityWarning
	case SeverityLow:
		constErr.Severity = ErrorSeverityWarning
	default:
		constErr.Severity = ErrorSeverityError
	}

	constErr.Metadata["violation_type"] = violation.Type
	constErr.Metadata["rule_id"] = violation.RuleID

	return constErr, nil
}

// HandleHallucinationError handles hallucination errors
func (h *errorHandler) HandleHallucinationError(ctx context.Context, element string, elementType string) (*ConstitutionError, error) {
	constErr := &ConstitutionError{
		ID:                    generateErrorID(),
		Category:              ErrorCategoryHallucination,
		Severity:              ErrorSeverityCritical,
		Message:               fmt.Sprintf("Referenced %s does not exist: %s", elementType, element),
		Details:               fmt.Sprintf("The AI attempted to reference a %s that does not exist in the codebase", elementType),
		Timestamp:             time.Now(),
		ConstitutionReference: "Section 4: Anti-Hallucination Rules",
		Metadata:              make(map[string]interface{}),
	}

	constErr.Metadata["element"] = element
	constErr.Metadata["element_type"] = elementType

	// Generate fix suggestions based on element type
	switch elementType {
	case "API":
		constErr.FixSuggestions = []string{
			"Define the API in the appropriate .proto file",
			"Verify the service and method names are correct",
			"Check if the API exists in a different service",
		}
	case "function":
		constErr.FixSuggestions = []string{
			"Implement the function in the appropriate package",
			"Verify the function name and package path are correct",
			"Check if the function exists with a different name",
		}
	case "module":
		constErr.FixSuggestions = []string{
			"Add the module to go.mod or package.json",
			"Verify the module path is correct",
			"Check if the module is available in the registry",
		}
	case "config_key":
		constErr.FixSuggestions = []string{
			"Add the configuration key to the config file",
			"Verify the key path is correct",
			"Use a default value if the key is optional",
		}
	}

	return constErr, nil
}

// HandleSystemError handles system errors
func (h *errorHandler) HandleSystemError(ctx context.Context, err error, operation string) (*ConstitutionError, error) {
	constErr := &ConstitutionError{
		ID:        generateErrorID(),
		Category:  ErrorCategorySystem,
		Message:   fmt.Sprintf("System error during %s: %v", operation, err),
		Details:   err.Error(),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	constErr.Metadata["operation"] = operation

	// Classify system error type and determine severity
	errStr := err.Error()
	switch {
	case contains(errStr, "permission denied", "access denied"):
		constErr.Severity = ErrorSeverityCritical
		constErr.Details = "File system permission error"
		constErr.FixSuggestions = []string{
			"Check file and directory permissions",
			"Ensure the process has necessary access rights",
		}
	case contains(errStr, "no space left", "disk full"):
		constErr.Severity = ErrorSeverityCritical
		constErr.Details = "Insufficient disk space"
		constErr.FixSuggestions = []string{
			"Free up disk space",
			"Clean up old backups and temporary files",
		}
	case contains(errStr, "timeout", "deadline exceeded"):
		constErr.Severity = ErrorSeverityError
		constErr.Details = "Operation timeout"
		constErr.FixSuggestions = []string{
			"Increase timeout configuration",
			"Check network connectivity",
			"Retry the operation",
		}
	case contains(errStr, "not found", "no such file"):
		constErr.Severity = ErrorSeverityError
		constErr.Details = "File or resource not found"
		constErr.FixSuggestions = []string{
			"Verify the file path is correct",
			"Ensure the file exists before accessing",
		}
	case contains(errStr, "connection refused", "network"):
		constErr.Severity = ErrorSeverityError
		constErr.Details = "Network connectivity error"
		constErr.FixSuggestions = []string{
			"Check network connectivity",
			"Verify service is running",
			"Retry the operation",
		}
	default:
		constErr.Severity = ErrorSeverityError
		constErr.FixSuggestions = []string{
			"Check system logs for more details",
			"Retry the operation",
			"Contact system administrator if issue persists",
		}
	}

	return constErr, nil
}

// ClassifyError classifies an error into a category
func (h *errorHandler) ClassifyError(err error) ErrorCategory {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for validation errors
	if contains(errStr, "validation", "lint", "format", "syntax", "type error") {
		return ErrorCategoryValidation
	}

	// Check for violation errors
	if contains(errStr, "violation", "forbidden", "prohibited", "not allowed") {
		return ErrorCategoryViolation
	}

	// Check for hallucination errors
	if contains(errStr, "not found", "does not exist", "undefined", "unresolved") {
		return ErrorCategoryHallucination
	}

	// Default to system error
	return ErrorCategorySystem
}

// DetermineSeverity determines the severity of an error
func (h *errorHandler) DetermineSeverity(err *ConstitutionError) ErrorSeverity {
	if err.Severity != "" {
		return err.Severity
	}

	// Determine severity based on category
	switch err.Category {
	case ErrorCategoryViolation:
		return ErrorSeverityCritical
	case ErrorCategoryHallucination:
		return ErrorSeverityCritical
	case ErrorCategoryValidation:
		return ErrorSeverityError
	case ErrorCategorySystem:
		return ErrorSeverityError
	default:
		return ErrorSeverityError
	}
}

// ShouldRollback determines if an error should trigger a rollback
func (h *errorHandler) ShouldRollback(err *ConstitutionError) bool {
	// Critical errors always trigger rollback
	if err.Severity == ErrorSeverityCritical {
		return true
	}

	// Violations and hallucinations trigger rollback
	if err.Category == ErrorCategoryViolation || err.Category == ErrorCategoryHallucination {
		return true
	}

	// Check config for rollback triggers
	if h.config != nil && h.config.Rollback.AutoRollback {
		// Check if error category is in rollback triggers
		for _, trigger := range h.config.Rollback.Triggers {
			if string(err.Category) == trigger {
				return true
			}
		}
	}

	return false
}

// ShouldRetry determines if an operation should be retried
func (h *errorHandler) ShouldRetry(err *ConstitutionError) bool {
	// Don't retry critical errors
	if err.Severity == ErrorSeverityCritical {
		return false
	}

	// Don't retry violations or hallucinations
	if err.Category == ErrorCategoryViolation || err.Category == ErrorCategoryHallucination {
		return false
	}

	// Retry system errors (timeout, network, etc.)
	if err.Category == ErrorCategorySystem {
		if err.Metadata != nil {
			operation, ok := err.Metadata["operation"].(string)
			if ok && contains(operation, "network", "timeout", "connection") {
				return true
			}
		}
	}

	return false
}

// generateValidationFixSuggestions generates fix suggestions for validation errors
func (h *errorHandler) generateValidationFixSuggestions(result *ValidationResult) []string {
	suggestions := []string{}

	if result.Validator == "gofmt" {
		suggestions = append(suggestions, "Run 'gofmt -w <file>' to format the code")
	}

	if result.Validator == "golangci-lint" {
		suggestions = append(suggestions, "Run 'golangci-lint run --fix' to auto-fix issues")
		suggestions = append(suggestions, "Review golangci-lint output for specific issues")
	}

	if result.Validator == "eslint" {
		suggestions = append(suggestions, "Run 'eslint --fix <file>' to auto-fix issues")
	}

	if result.Validator == "vue-tsc" {
		suggestions = append(suggestions, "Fix TypeScript type errors")
		suggestions = append(suggestions, "Add missing type definitions")
	}

	if len(result.Errors) > 0 {
		suggestions = append(suggestions, "Review error messages and fix issues one by one")
	}

	return suggestions
}

// generateErrorID generates a unique error ID
func generateErrorID() string {
	return fmt.Sprintf("err-%d", time.Now().UnixNano())
}

// contains checks if a string contains any of the substrings
func contains(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
