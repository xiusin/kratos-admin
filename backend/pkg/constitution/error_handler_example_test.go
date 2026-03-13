package constitution_test

import (
	"context"
	"fmt"
	"log"

	"backend/pkg/constitution"
)

// ExampleErrorHandler_HandleValidationError demonstrates handling validation errors
func ExampleErrorHandler_HandleValidationError() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)

	// Simulate a validation error
	validationResult := &constitution.ValidationResult{
		Passed:    false,
		Validator: "golangci-lint",
		Errors: []constitution.ValidationError{
			{
				File:     "service/user.go",
				Line:     45,
				Column:   10,
				Message:  "undefined: UserRepo",
				Severity: "error",
			},
		},
	}

	// Handle the validation error
	ctx := context.Background()
	constErr, err := errorHandler.HandleValidationError(ctx, fmt.Errorf("validation failed"), validationResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Error Category: %s\n", constErr.Category)
	fmt.Printf("Error Severity: %s\n", constErr.Severity)
	fmt.Printf("Should Rollback: %v\n", errorHandler.ShouldRollback(constErr))

	// Output:
	// Error Category: validation
	// Error Severity: error
	// Should Rollback: false
}

// ExampleErrorHandler_HandleViolationError demonstrates handling constitution violations
func ExampleErrorHandler_HandleViolationError() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)

	// Simulate a violation
	violation := constitution.Violation{
		RuleID:        "arch-001",
		Type:          constitution.ViolationTypeArchitecture,
		Severity:      constitution.SeverityCritical,
		Message:       "Cross-layer direct call detected",
		Description:   "API layer directly calling pkg layer without going through app layer",
		File:          "api/handler/user.go",
		Line:          23,
		RuleReference: "Section 3.1: Three-Layer Architecture",
		FixSuggestions: []string{
			"Move the logic to app/service layer",
			"Call app layer from API layer",
		},
	}

	// Handle the violation error
	ctx := context.Background()
	constErr, err := errorHandler.HandleViolationError(ctx, violation)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Error Category: %s\n", constErr.Category)
	fmt.Printf("Error Severity: %s\n", constErr.Severity)
	fmt.Printf("Should Rollback: %v\n", errorHandler.ShouldRollback(constErr))

	// Output:
	// Error Category: violation
	// Error Severity: critical
	// Should Rollback: true
}

// ExampleErrorHandler_HandleHallucinationError demonstrates handling hallucination errors
func ExampleErrorHandler_HandleHallucinationError() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)

	// Handle hallucination error
	ctx := context.Background()
	constErr, err := errorHandler.HandleHallucinationError(ctx, "GetUserByEmail", "function")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Error Category: %s\n", constErr.Category)
	fmt.Printf("Error Severity: %s\n", constErr.Severity)
	fmt.Printf("Should Rollback: %v\n", errorHandler.ShouldRollback(constErr))
	fmt.Printf("Fix Suggestions: %d\n", len(constErr.FixSuggestions))

	// Output:
	// Error Category: hallucination
	// Error Severity: critical
	// Should Rollback: true
	// Fix Suggestions: 3
}

// ExampleErrorRecovery_DetermineStrategy demonstrates determining recovery strategy
func ExampleErrorRecovery_DetermineStrategy() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	rollbackManager, _ := constitution.NewRollbackManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)
	errorRecovery := constitution.NewErrorRecovery(config, rollbackManager, traceManager, errorHandler)

	// Create a critical error
	constErr := &constitution.ConstitutionError{
		Category: constitution.ErrorCategoryViolation,
		Severity: constitution.ErrorSeverityCritical,
		Message:  "Architecture violation detected",
	}

	// Determine recovery strategy
	ctx := context.Background()
	action, err := errorRecovery.DetermineStrategy(ctx, constErr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Strategy: %s\n", action.Strategy)
	fmt.Printf("Auto Execute: %v\n", action.AutoExecute)
	fmt.Printf("Steps: %d\n", len(action.Steps))

	// Output:
	// Strategy: rollback
	// Auto Execute: true
	// Steps: 4
}

// ExampleErrorRecovery_ExecuteRetry demonstrates retry with exponential backoff
func ExampleErrorRecovery_ExecuteRetry() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	rollbackManager, _ := constitution.NewRollbackManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)
	errorRecovery := constitution.NewErrorRecovery(config, rollbackManager, traceManager, errorHandler)

	// Simulate an operation that fails twice then succeeds
	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	}

	// Execute with retry
	ctx := context.Background()
	result, err := errorRecovery.ExecuteRetry(ctx, operation, 5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Attempts: %d\n", result.AttemptsCount)

	// Output:
	// Success: true
	// Attempts: 3
}

// ExampleErrorReporter_GenerateReport demonstrates generating error reports
func ExampleErrorReporter_GenerateReport() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)
	errorReporter := constitution.NewErrorReporter(config, ruleEngine, errorHandler)

	// Create an error
	constErr := &constitution.ConstitutionError{
		Category:              constitution.ErrorCategoryValidation,
		Severity:              constitution.ErrorSeverityError,
		Message:               "Validation failed",
		Details:               "Go code formatting issues detected",
		File:                  "service/user.go",
		Line:                  45,
		ConstitutionReference: "Section 7: Validation Requirements",
		FixSuggestions: []string{
			"Run gofmt -w service/user.go",
			"Review and fix formatting issues",
		},
	}

	// Generate report
	report, err := errorReporter.GenerateReport(constErr, "task-001")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Report ID: %s\n", report.ReportID[:10])
	fmt.Printf("Task ID: %s\n", report.TaskID)
	fmt.Printf("Error Category: %s\n", report.Error.Category)
	fmt.Printf("Fix Suggestions: %d\n", len(report.FixSuggestions))

	// Output:
	// Report ID: report-202
	// Task ID: task-001
	// Error Category: validation
	// Fix Suggestions: 3
}

// ExampleErrorReporter_GenerateTextReport demonstrates generating text reports
func ExampleErrorReporter_GenerateTextReport() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)
	errorReporter := constitution.NewErrorReporter(config, ruleEngine, errorHandler)

	// Create an error
	constErr := &constitution.ConstitutionError{
		Category: constitution.ErrorCategoryViolation,
		Severity: constitution.ErrorSeverityCritical,
		Message:  "Architecture violation",
		Details:  "Cross-layer direct call detected",
		File:     "api/handler/user.go",
		Line:     23,
	}

	// Generate report
	report, err := errorReporter.GenerateReport(constErr, "task-001")
	if err != nil {
		log.Fatal(err)
	}

	// Generate text report
	textReport, err := errorReporter.GenerateTextReport(report)
	if err != nil {
		log.Fatal(err)
	}

	// Print first few lines
	lines := 0
	for _, char := range textReport {
		if char == '\n' {
			lines++
			if lines >= 5 {
				break
			}
		}
		fmt.Print(string(char))
	}

	// Output:
	// ===============================================================================
	// ERROR REPORT: report-
	// ===============================================================================
	//
	// Timestamp:
}

// ExampleErrorReporter_GenerateMarkdownReport demonstrates generating Markdown reports
func ExampleErrorReporter_GenerateMarkdownReport() {
	// Load configuration
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := loader.Load(); err != nil {
		log.Fatal(err)
	}

	config := loader.Get()

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(config)
	ruleEngine := constitution.NewRuleEngineFromConfig(config)
	errorHandler := constitution.NewErrorHandler(config, ruleEngine, traceManager)
	errorReporter := constitution.NewErrorReporter(config, ruleEngine, errorHandler)

	// Create an error
	constErr := &constitution.ConstitutionError{
		Category: constitution.ErrorCategoryHallucination,
		Severity: constitution.ErrorSeverityCritical,
		Message:  "Referenced function does not exist",
		Details:  "Function GetUserByEmail not found in codebase",
	}

	// Generate report
	report, err := errorReporter.GenerateReport(constErr, "task-001")
	if err != nil {
		log.Fatal(err)
	}

	// Generate Markdown report
	mdReport, err := errorReporter.GenerateMarkdownReport(report)
	if err != nil {
		log.Fatal(err)
	}

	// Print first line
	lines := 0
	for _, char := range mdReport {
		if char == '\n' {
			lines++
			if lines >= 1 {
				break
			}
		}
		fmt.Print(string(char))
	}

	// Output:
	// # Error Report: report-
}
