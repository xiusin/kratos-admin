package constitution

import (
	"fmt"
	"strings"
)

// RollbackTrigger handles automatic rollback triggering based on validation failures and violations
type RollbackTrigger struct {
	rollbackManager RollbackManager
	traceManager    TaskTraceManager
	validator       CodeValidator
}

// NewRollbackTrigger creates a new RollbackTrigger instance
func NewRollbackTrigger(
	rollbackManager RollbackManager,
	traceManager TaskTraceManager,
	validator CodeValidator,
) *RollbackTrigger {
	return &RollbackTrigger{
		rollbackManager: rollbackManager,
		traceManager:    traceManager,
		validator:       validator,
	}
}

// TriggerCondition represents a condition that triggers rollback
type TriggerCondition string

const (
	TriggerConditionValidationFailure   TriggerCondition = "validation_failure"
	TriggerConditionConstitutionViolation TriggerCondition = "constitution_violation"
	TriggerConditionManual              TriggerCondition = "manual"
	TriggerConditionSecurityViolation   TriggerCondition = "security_violation"
	TriggerConditionArchitectureViolation TriggerCondition = "architecture_violation"
)

// RollbackTriggerResult represents the result of a rollback trigger check
type RollbackTriggerResult struct {
	ShouldRollback bool
	Condition      TriggerCondition
	Reason         string
	Details        []string
}

// CheckValidationResult checks if validation result should trigger rollback
func (rt *RollbackTrigger) CheckValidationResult(result *ValidationResult) *RollbackTriggerResult {
	if result == nil {
		return &RollbackTriggerResult{
			ShouldRollback: false,
		}
	}

	// Check if validation passed
	if result.Passed {
		return &RollbackTriggerResult{
			ShouldRollback: false,
		}
	}

	// Check for critical errors
	criticalErrors := make([]string, 0)
	for _, err := range result.Errors {
		if err.Severity == "error" || err.Severity == "critical" {
			criticalErrors = append(criticalErrors, fmt.Sprintf("%s:%d:%d: %s", err.File, err.Line, err.Column, err.Message))
		}
	}

	if len(criticalErrors) > 0 {
		return &RollbackTriggerResult{
			ShouldRollback: true,
			Condition:      TriggerConditionValidationFailure,
			Reason:         fmt.Sprintf("Validation failed with %d critical error(s)", len(criticalErrors)),
			Details:        criticalErrors,
		}
	}

	return &RollbackTriggerResult{
		ShouldRollback: false,
	}
}

// CheckConstitutionViolation checks if code changes violate constitution rules
func (rt *RollbackTrigger) CheckConstitutionViolation(changes []CodeChange) *RollbackTriggerResult {
	violations := make([]string, 0)

	for _, change := range changes {
		// Check for forbidden file modifications
		if rt.isForbiddenFileModification(change.FilePath, change.Operation) {
			violations = append(violations, fmt.Sprintf("Forbidden modification: %s (%s)", change.FilePath, change.Operation))
		}

		// Check for architecture violations
		if rt.isArchitectureViolation(change.FilePath) {
			violations = append(violations, fmt.Sprintf("Architecture violation: %s", change.FilePath))
		}

		// Check for security violations in diff content
		if rt.hasSecurityViolation(change.DiffContent) {
			violations = append(violations, fmt.Sprintf("Security violation detected in: %s", change.FilePath))
		}
	}

	if len(violations) > 0 {
		return &RollbackTriggerResult{
			ShouldRollback: true,
			Condition:      TriggerConditionConstitutionViolation,
			Reason:         fmt.Sprintf("Constitution violation detected: %d violation(s)", len(violations)),
			Details:        violations,
		}
	}

	return &RollbackTriggerResult{
		ShouldRollback: false,
	}
}

// TriggerRollback triggers a rollback for a task
func (rt *RollbackTrigger) TriggerRollback(taskID string, result *RollbackTriggerResult) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}

	if result == nil || !result.ShouldRollback {
		return fmt.Errorf("rollback not required")
	}

	// Build rollback reason
	reason := fmt.Sprintf("[%s] %s", result.Condition, result.Reason)
	if len(result.Details) > 0 {
		reason += "\nDetails:\n- " + strings.Join(result.Details, "\n- ")
	}

	// Execute rollback
	if err := rt.rollbackManager.Rollback(taskID, reason); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	// Record rollback in task trace
	rollbackInfo := RollbackInfo{
		Triggered: true,
		Reason:    reason,
	}

	if err := rt.traceManager.RecordRollback(taskID, rollbackInfo); err != nil {
		// Log error but don't fail the rollback
		fmt.Printf("Warning: failed to record rollback in task trace: %v\n", err)
	}

	// Mark task as rolled back
	if err := rt.traceManager.FailTask(taskID, reason); err != nil {
		// Log error but don't fail the rollback
		fmt.Printf("Warning: failed to mark task as failed: %v\n", err)
	}

	return nil
}

// ManualRollback triggers a manual rollback
func (rt *RollbackTrigger) ManualRollback(taskID string, reason string) error {
	result := &RollbackTriggerResult{
		ShouldRollback: true,
		Condition:      TriggerConditionManual,
		Reason:         reason,
	}

	return rt.TriggerRollback(taskID, result)
}

// Helper functions

func (rt *RollbackTrigger) isForbiddenFileModification(filePath string, operation OperationType) bool {
	// Check for forbidden file patterns
	forbiddenPatterns := []string{
		"Dockerfile",
		"docker-compose",
		".golangci.yml",
		".eslintrc",
		"buf.yaml",
		"buf.gen.yaml",
		"configs/*-prod.yaml",
		"configs/*-production.yaml",
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	// Check for forbidden operations
	if operation == OperationDelete {
		// Deleting migration files is forbidden
		if strings.Contains(filePath, "/migrations/") || strings.Contains(filePath, "/ent/migrate/") {
			return true
		}

		// Deleting protobuf files is forbidden
		if strings.HasSuffix(filePath, ".proto") {
			return true
		}
	}

	return false
}

func (rt *RollbackTrigger) isArchitectureViolation(filePath string) bool {
	// Check for architecture layer violations
	// For example, pkg/ should not depend on app/
	if strings.Contains(filePath, "/pkg/") {
		// This is a simplified check - in production, you'd parse imports
		// to verify actual dependencies
		return false
	}

	return false
}

func (rt *RollbackTrigger) hasSecurityViolation(diffContent string) bool {
	if diffContent == "" {
		return false
	}

	// Check for common security violations
	securityPatterns := []string{
		"password =",
		"secret =",
		"api_key =",
		"token =",
		"private_key =",
		"BEGIN PRIVATE KEY",
		"BEGIN RSA PRIVATE KEY",
	}

	lowerDiff := strings.ToLower(diffContent)
	for _, pattern := range securityPatterns {
		if strings.Contains(lowerDiff, strings.ToLower(pattern)) {
			// Check if it's not a variable name or comment
			if !strings.Contains(lowerDiff, "// "+strings.ToLower(pattern)) &&
				!strings.Contains(lowerDiff, "# "+strings.ToLower(pattern)) {
				return true
			}
		}
	}

	return false
}

// AutoRollbackOnValidation automatically checks validation result and triggers rollback if needed
func (rt *RollbackTrigger) AutoRollbackOnValidation(taskID string, result *ValidationResult) error {
	triggerResult := rt.CheckValidationResult(result)
	if triggerResult.ShouldRollback {
		return rt.TriggerRollback(taskID, triggerResult)
	}
	return nil
}

// AutoRollbackOnViolation automatically checks code changes and triggers rollback if needed
func (rt *RollbackTrigger) AutoRollbackOnViolation(taskID string, changes []CodeChange) error {
	triggerResult := rt.CheckConstitutionViolation(changes)
	if triggerResult.ShouldRollback {
		return rt.TriggerRollback(taskID, triggerResult)
	}
	return nil
}
