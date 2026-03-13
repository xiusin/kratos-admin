package constitution

import (
	"context"
	"fmt"
	"time"
)

// RecoveryStrategy represents a strategy for recovering from errors
type RecoveryStrategy string

const (
	RecoveryStrategyRetry    RecoveryStrategy = "retry"
	RecoveryStrategyRollback RecoveryStrategy = "rollback"
	RecoveryStrategyDegrade  RecoveryStrategy = "degrade"
	RecoveryStrategyManual   RecoveryStrategy = "manual"
	RecoveryStrategySkip     RecoveryStrategy = "skip"
)

// RecoveryAction represents an action to take for error recovery
type RecoveryAction struct {
	Strategy      RecoveryStrategy `json:"strategy"`
	Description   string           `json:"description"`
	AutoExecute   bool             `json:"auto_execute"`
	RequiresInput bool             `json:"requires_input"`
	Steps         []string         `json:"steps"`
}

// RecoveryResult represents the result of a recovery attempt
type RecoveryResult struct {
	Success       bool             `json:"success"`
	Strategy      RecoveryStrategy `json:"strategy"`
	AttemptsCount int              `json:"attempts_count"`
	Duration      time.Duration    `json:"duration"`
	Message       string           `json:"message"`
	Error         error            `json:"error,omitempty"`
}

// ErrorRecovery handles error recovery strategies
type ErrorRecovery interface {
	// DetermineStrategy determines the best recovery strategy for an error
	DetermineStrategy(ctx context.Context, err *ConstitutionError) (*RecoveryAction, error)

	// ExecuteRetry executes retry strategy with exponential backoff
	ExecuteRetry(ctx context.Context, operation func() error, maxAttempts int) (*RecoveryResult, error)

	// ExecuteRollback executes rollback strategy
	ExecuteRollback(ctx context.Context, taskID string, reason string) (*RecoveryResult, error)

	// ExecuteDegrade executes graceful degradation strategy
	ExecuteDegrade(ctx context.Context, err *ConstitutionError) (*RecoveryResult, error)

	// RequestManualIntervention requests manual intervention from developer
	RequestManualIntervention(ctx context.Context, err *ConstitutionError) (*RecoveryResult, error)

	// GetRetryDelays returns retry delays based on configuration
	GetRetryDelays(maxAttempts int) []time.Duration
}

// errorRecovery implements ErrorRecovery
type errorRecovery struct {
	config          *Config
	rollbackManager RollbackManager
	traceManager    TaskTraceManager
	errorHandler    ErrorHandler
}

// NewErrorRecovery creates a new error recovery handler
func NewErrorRecovery(
	config *Config,
	rollbackManager RollbackManager,
	traceManager TaskTraceManager,
	errorHandler ErrorHandler,
) ErrorRecovery {
	return &errorRecovery{
		config:          config,
		rollbackManager: rollbackManager,
		traceManager:    traceManager,
		errorHandler:    errorHandler,
	}
}

// DetermineStrategy determines the best recovery strategy for an error
func (r *errorRecovery) DetermineStrategy(ctx context.Context, err *ConstitutionError) (*RecoveryAction, error) {
	action := &RecoveryAction{
		Steps: []string{},
	}

	// Critical errors require rollback
	if err.Severity == ErrorSeverityCritical {
		action.Strategy = RecoveryStrategyRollback
		action.Description = "Critical error detected, rolling back changes"
		action.AutoExecute = true
		action.Steps = []string{
			"Stop all operations",
			"Restore files from backup",
			"Verify rollback success",
			"Report error to developer",
		}
		return action, nil
	}

	// Violations require rollback
	if err.Category == ErrorCategoryViolation {
		action.Strategy = RecoveryStrategyRollback
		action.Description = "Constitution violation detected, rolling back changes"
		action.AutoExecute = true
		action.Steps = []string{
			"Identify violated rules",
			"Restore files from backup",
			"Record violation in trace",
			"Provide fix suggestions",
		}
		return action, nil
	}

	// Hallucinations require manual intervention
	if err.Category == ErrorCategoryHallucination {
		action.Strategy = RecoveryStrategyManual
		action.Description = "Hallucination detected, manual intervention required"
		action.AutoExecute = false
		action.RequiresInput = true
		action.Steps = []string{
			"Verify the referenced element exists",
			"If not, create the element or use an alternative",
			"Update the code to reference the correct element",
		}
		return action, nil
	}

	// System errors may be retryable
	if err.Category == ErrorCategorySystem {
		if r.errorHandler.ShouldRetry(err) {
			action.Strategy = RecoveryStrategyRetry
			action.Description = "Transient system error, retrying operation"
			action.AutoExecute = true
			action.Steps = []string{
				"Wait for backoff period",
				"Retry the operation",
				"If retry fails, escalate to manual intervention",
			}
			return action, nil
		}

		// Non-retryable system errors require manual intervention
		action.Strategy = RecoveryStrategyManual
		action.Description = "System error requires manual intervention"
		action.AutoExecute = false
		action.RequiresInput = true
		action.Steps = []string{
			"Investigate the system error",
			"Fix the underlying issue",
			"Retry the operation manually",
		}
		return action, nil
	}

	// Validation errors with warnings can degrade gracefully
	if err.Category == ErrorCategoryValidation && err.Severity == ErrorSeverityWarning {
		action.Strategy = RecoveryStrategyDegrade
		action.Description = "Validation warnings detected, continuing with warnings"
		action.AutoExecute = true
		action.Steps = []string{
			"Record warnings in trace",
			"Continue operation",
			"Notify developer of warnings",
		}
		return action, nil
	}

	// Validation errors require rollback
	if err.Category == ErrorCategoryValidation {
		action.Strategy = RecoveryStrategyRollback
		action.Description = "Validation errors detected, rolling back changes"
		action.AutoExecute = true
		action.Steps = []string{
			"Restore files from backup",
			"Record validation errors",
			"Provide fix suggestions",
		}
		return action, nil
	}

	// Default to manual intervention
	action.Strategy = RecoveryStrategyManual
	action.Description = "Unknown error type, manual intervention required"
	action.AutoExecute = false
	action.RequiresInput = true
	action.Steps = []string{
		"Analyze the error",
		"Determine appropriate action",
		"Execute recovery manually",
	}

	return action, nil
}

// ExecuteRetry executes retry strategy with exponential backoff
func (r *errorRecovery) ExecuteRetry(ctx context.Context, operation func() error, maxAttempts int) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Strategy:      RecoveryStrategyRetry,
		AttemptsCount: 0,
	}

	startTime := time.Now()
	delays := r.GetRetryDelays(maxAttempts)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result.AttemptsCount++

		// Execute operation
		err := operation()
		if err == nil {
			result.Success = true
			result.Duration = time.Since(startTime)
			result.Message = fmt.Sprintf("Operation succeeded after %d attempt(s)", result.AttemptsCount)
			return result, nil
		}

		result.Error = err

		// If this is the last attempt, don't wait
		if attempt == maxAttempts-1 {
			break
		}

		// Wait before retry
		delay := delays[attempt]
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			result.Message = "Retry cancelled by context"
			return result, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	result.Success = false
	result.Duration = time.Since(startTime)
	result.Message = fmt.Sprintf("Operation failed after %d attempt(s)", result.AttemptsCount)

	return result, result.Error
}

// ExecuteRollback executes rollback strategy
func (r *errorRecovery) ExecuteRollback(ctx context.Context, taskID string, reason string) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Strategy:      RecoveryStrategyRollback,
		AttemptsCount: 1,
	}

	startTime := time.Now()

	// Execute rollback
	err := r.rollbackManager.Rollback(taskID, reason)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Duration = time.Since(startTime)
		result.Message = fmt.Sprintf("Rollback failed: %v", err)
		return result, err
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	result.Message = "Rollback completed successfully"

	return result, nil
}

// ExecuteDegrade executes graceful degradation strategy
func (r *errorRecovery) ExecuteDegrade(ctx context.Context, err *ConstitutionError) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Strategy:      RecoveryStrategyDegrade,
		AttemptsCount: 1,
		Success:       true,
	}

	startTime := time.Now()

	// Log the degradation
	result.Message = fmt.Sprintf("Continuing with degraded functionality: %s", err.Message)

	// Record in trace if available
	if r.traceManager != nil && err.Metadata != nil {
		if taskID, ok := err.Metadata["task_id"].(string); ok {
			_ = r.traceManager.RecordDecision(taskID, Decision{
				Timestamp:    time.Now(),
				DecisionType: DecisionTypeValidation,
				Description:  "Graceful degradation",
				Rationale:    result.Message,
			})
		}
	}

	result.Duration = time.Since(startTime)

	return result, nil
}

// RequestManualIntervention requests manual intervention from developer
func (r *errorRecovery) RequestManualIntervention(ctx context.Context, err *ConstitutionError) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Strategy:      RecoveryStrategyManual,
		AttemptsCount: 1,
		Success:       false,
	}

	startTime := time.Now()

	// Format intervention request
	result.Message = fmt.Sprintf(
		"Manual intervention required:\n"+
			"Category: %s\n"+
			"Severity: %s\n"+
			"Message: %s\n"+
			"Details: %s\n"+
			"Fix Suggestions:\n%s",
		err.Category,
		err.Severity,
		err.Message,
		err.Details,
		formatFixSuggestions(err.FixSuggestions),
	)

	result.Duration = time.Since(startTime)

	return result, nil
}

// GetRetryDelays returns retry delays based on configuration
func (r *errorRecovery) GetRetryDelays(maxAttempts int) []time.Duration {
	if r.config != nil && len(r.config.Retry.Delays) > 0 {
		// Use configured delays
		delays := make([]time.Duration, maxAttempts)
		for i := 0; i < maxAttempts; i++ {
			if i < len(r.config.Retry.Delays) {
				delays[i] = r.config.Retry.Delays[i]
			} else {
				// Use last configured delay for remaining attempts
				delays[i] = r.config.Retry.Delays[len(r.config.Retry.Delays)-1]
			}
		}
		return delays
	}

	// Default exponential backoff: 1s, 2s, 4s, 8s, 16s
	delays := make([]time.Duration, maxAttempts)
	for i := 0; i < maxAttempts; i++ {
		delays[i] = time.Duration(1<<uint(i)) * time.Second
		// Cap at 30 seconds
		if delays[i] > 30*time.Second {
			delays[i] = 30 * time.Second
		}
	}

	return delays
}

// formatFixSuggestions formats fix suggestions for display
func formatFixSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return "  (No suggestions available)"
	}

	result := ""
	for i, suggestion := range suggestions {
		result += fmt.Sprintf("  %d. %s\n", i+1, suggestion)
	}

	return result
}
