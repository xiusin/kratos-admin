package constitution

import (
	"testing"
	"time"
)

func TestNewRollbackTrigger(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)
	if trigger == nil {
		t.Fatal("Expected non-nil trigger")
	}
}

func TestCheckValidationResultPassed(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	result := &ValidationResult{
		Passed:    true,
		Validator: "test",
		Timestamp: time.Now(),
	}

	triggerResult := trigger.CheckValidationResult(result)
	if triggerResult.ShouldRollback {
		t.Error("Expected no rollback for passed validation")
	}
}

func TestCheckValidationResultFailed(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	result := &ValidationResult{
		Passed:    false,
		Validator: "test",
		Timestamp: time.Now(),
		Errors: []ValidationError{
			{
				File:     "test.go",
				Line:     10,
				Column:   5,
				Message:  "syntax error",
				Severity: "error",
			},
		},
	}

	triggerResult := trigger.CheckValidationResult(result)
	if !triggerResult.ShouldRollback {
		t.Error("Expected rollback for failed validation")
	}

	if triggerResult.Condition != TriggerConditionValidationFailure {
		t.Errorf("Expected condition %s, got %s", TriggerConditionValidationFailure, triggerResult.Condition)
	}

	if len(triggerResult.Details) == 0 {
		t.Error("Expected error details")
	}
}

func TestCheckValidationResultWarningsOnly(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	result := &ValidationResult{
		Passed:    false,
		Validator: "test",
		Timestamp: time.Now(),
		Errors: []ValidationError{
			{
				File:     "test.go",
				Line:     10,
				Column:   5,
				Message:  "unused variable",
				Severity: "warning",
			},
		},
	}

	triggerResult := trigger.CheckValidationResult(result)
	if triggerResult.ShouldRollback {
		t.Error("Expected no rollback for warnings only")
	}
}

func TestCheckConstitutionViolationForbiddenFile(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	changes := []CodeChange{
		{
			FilePath:  "Dockerfile",
			Operation: OperationModify,
			Summary:   "Modified Dockerfile",
		},
	}

	triggerResult := trigger.CheckConstitutionViolation(changes)
	if !triggerResult.ShouldRollback {
		t.Error("Expected rollback for forbidden file modification")
	}

	if triggerResult.Condition != TriggerConditionConstitutionViolation {
		t.Errorf("Expected condition %s, got %s", TriggerConditionConstitutionViolation, triggerResult.Condition)
	}
}

func TestCheckConstitutionViolationDeleteMigration(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	changes := []CodeChange{
		{
			FilePath:  "backend/app/admin/service/internal/data/ent/migrate/20240101_create_users.sql",
			Operation: OperationDelete,
			Summary:   "Deleted migration file",
		},
	}

	triggerResult := trigger.CheckConstitutionViolation(changes)
	if !triggerResult.ShouldRollback {
		t.Error("Expected rollback for deleting migration file")
	}
}

func TestCheckConstitutionViolationDeleteProto(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	changes := []CodeChange{
		{
			FilePath:  "backend/api/protos/user/v1/user.proto",
			Operation: OperationDelete,
			Summary:   "Deleted proto file",
		},
	}

	triggerResult := trigger.CheckConstitutionViolation(changes)
	if !triggerResult.ShouldRollback {
		t.Error("Expected rollback for deleting proto file")
	}
}

func TestCheckConstitutionViolationSecurityIssue(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	changes := []CodeChange{
		{
			FilePath:    "backend/config.go",
			Operation:   OperationModify,
			Summary:     "Added hardcoded password",
			DiffContent: "+const password = \"mysecretpassword123\"",
		},
	}

	triggerResult := trigger.CheckConstitutionViolation(changes)
	if !triggerResult.ShouldRollback {
		t.Error("Expected rollback for security violation")
	}
}

func TestCheckConstitutionViolationNoViolation(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	changes := []CodeChange{
		{
			FilePath:  "backend/app/admin/service/internal/service/user.go",
			Operation: OperationCreate,
			Summary:   "Created user service",
		},
	}

	triggerResult := trigger.CheckConstitutionViolation(changes)
	if triggerResult.ShouldRollback {
		t.Error("Expected no rollback for valid changes")
	}
}

func TestManualRollback(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task and backup
	taskID, _ := traceManager.CreateTask("test task", "test request")
	testFile := tempDir + "/test.txt"
	// Create test file
	// Note: In real scenario, file should exist before backup
	
	// Create backup (will handle non-existent file)
	_, err := rollbackManager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Trigger manual rollback
	err = trigger.ManualRollback(taskID, "User requested rollback")
	if err != nil {
		t.Errorf("Manual rollback failed: %v", err)
	}

	// Verify task was marked as failed
	task, err := traceManager.GetTask(taskID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if task.Status != TaskStatusRolledBack && task.Status != TaskStatusFailed {
		t.Errorf("Expected task status to be rolled_back or failed, got %s", task.Status)
	}
}

func TestAutoRollbackOnValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task and backup
	taskID, _ := traceManager.CreateTask("test task", "test request")
	testFile := tempDir + "/test.txt"
	_, err := rollbackManager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Create failed validation result
	result := &ValidationResult{
		Passed:    false,
		Validator: "test",
		Timestamp: time.Now(),
		Errors: []ValidationError{
			{
				File:     "test.go",
				Line:     10,
				Column:   5,
				Message:  "syntax error",
				Severity: "error",
			},
		},
	}

	// Auto rollback
	err = trigger.AutoRollbackOnValidation(taskID, result)
	if err != nil {
		t.Errorf("Auto rollback failed: %v", err)
	}
}

func TestAutoRollbackOnViolation(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task and backup
	taskID, _ := traceManager.CreateTask("test task", "test request")
	testFile := tempDir + "/test.txt"
	_, err := rollbackManager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Create code changes with violation
	changes := []CodeChange{
		{
			FilePath:  "Dockerfile",
			Operation: OperationModify,
			Summary:   "Modified Dockerfile",
		},
	}

	// Auto rollback
	err = trigger.AutoRollbackOnViolation(taskID, changes)
	if err != nil {
		t.Errorf("Auto rollback failed: %v", err)
	}
}

func TestHasSecurityViolationComment(t *testing.T) {
	tempDir := t.TempDir()
	
	rollbackManager, _ := NewRollbackManager(tempDir+"/backups", tempDir+"/traces")
	traceManager, _ := NewTaskTraceManager(tempDir + "/traces")
	validator, _ := NewCodeValidator(&Config{})

	trigger := NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Password in comment should not trigger violation
	diffContent := "// password = \"example\""
	if trigger.hasSecurityViolation(diffContent) {
		t.Error("Expected no violation for password in comment")
	}

	// Actual password assignment should trigger violation
	diffContent = "password = \"mysecret\""
	if !trigger.hasSecurityViolation(diffContent) {
		t.Error("Expected violation for actual password assignment")
	}
}
