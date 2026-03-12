package constitution_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go-wind-admin/pkg/constitution"
)

// Example demonstrates basic rollback functionality
func Example_rollback() {
	// Create temporary directory for example
	tempDir, err := os.MkdirTemp("", "rollback-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize components
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
	if err != nil {
		log.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "example.go")
	originalContent := []byte("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}\n")
	if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
		log.Fatal(err)
	}

	// Create backup before modification
	taskID := "example-task-001"
	backupID, err := rollbackManager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created backup: %s\n", backupID)

	// Modify the file
	modifiedContent := []byte("package main\n\nfunc main() {\n\tprintln(\"Modified\")\n}\n")
	if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("File modified")

	// Rollback the changes
	if err := rollbackManager.Rollback(taskID, "Example rollback"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Rollback completed")

	// Verify file was restored
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		log.Fatal(err)
	}

	if string(restoredContent) == string(originalContent) {
		fmt.Println("File successfully restored to original content")
	}

	// Output:
	// Created backup: [backup-id]
	// File modified
	// Rollback completed
	// File successfully restored to original content
}

// Example demonstrates automatic rollback on validation failure
func Example_autoRollbackOnValidation() {
	// Create temporary directory for example
	tempDir, err := os.MkdirTemp("", "auto-rollback-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize components
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
	if err != nil {
		log.Fatal(err)
	}

	traceManager, err := constitution.NewTaskTraceManager(traceDir)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &constitution.Config{
		ProjectRoot: tempDir,
	}
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatal(err)
	}

	trigger := constitution.NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task
	taskID, err := traceManager.CreateTask("Example task", "Implement feature")
	if err != nil {
		log.Fatal(err)
	}

	// Create test file and backup
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		log.Fatal(err)
	}

	if _, err := rollbackManager.CreateBackup(taskID, []string{testFile}); err != nil {
		log.Fatal(err)
	}

	// Simulate validation failure
	validationResult := &constitution.ValidationResult{
		Passed:    false,
		Validator: "gofmt",
		Errors: []constitution.ValidationError{
			{
				File:     testFile,
				Line:     10,
				Column:   5,
				Message:  "syntax error: unexpected EOF",
				Severity: "error",
			},
		},
	}

	// Auto rollback on validation failure
	if err := trigger.AutoRollbackOnValidation(taskID, validationResult); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Auto rollback triggered on validation failure")

	// Output:
	// Auto rollback triggered on validation failure
}

// Example demonstrates rollback on constitution violation
func Example_rollbackOnViolation() {
	// Create temporary directory for example
	tempDir, err := os.MkdirTemp("", "violation-rollback-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize components
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
	if err != nil {
		log.Fatal(err)
	}

	traceManager, err := constitution.NewTaskTraceManager(traceDir)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &constitution.Config{
		ProjectRoot: tempDir,
	}
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatal(err)
	}

	trigger := constitution.NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task
	taskID, err := traceManager.CreateTask("Example task", "Modify Dockerfile")
	if err != nil {
		log.Fatal(err)
	}

	// Create backup
	dockerFile := filepath.Join(tempDir, "Dockerfile")
	if _, err := rollbackManager.CreateBackup(taskID, []string{dockerFile}); err != nil {
		log.Fatal(err)
	}

	// Simulate forbidden file modification
	changes := []constitution.CodeChange{
		{
			FilePath:  dockerFile,
			Operation: constitution.OperationModify,
			Summary:   "Modified Dockerfile without approval",
		},
	}

	// Check for violations
	triggerResult := trigger.CheckConstitutionViolation(changes)
	if triggerResult.ShouldRollback {
		fmt.Printf("Violation detected: %s\n", triggerResult.Reason)
		
		// Trigger rollback
		if err := trigger.TriggerRollback(taskID, triggerResult); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Rollback completed due to constitution violation")
	}

	// Output:
	// Violation detected: Constitution violation detected: 1 violation(s)
	// Rollback completed due to constitution violation
}

// Example demonstrates manual rollback
func Example_manualRollback() {
	// Create temporary directory for example
	tempDir, err := os.MkdirTemp("", "manual-rollback-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize components
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
	if err != nil {
		log.Fatal(err)
	}

	traceManager, err := constitution.NewTaskTraceManager(traceDir)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &constitution.Config{
		ProjectRoot: tempDir,
	}
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatal(err)
	}

	trigger := constitution.NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create a task
	taskID, err := traceManager.CreateTask("Example task", "Implement feature")
	if err != nil {
		log.Fatal(err)
	}

	// Create test file and backup
	testFile := filepath.Join(tempDir, "feature.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		log.Fatal(err)
	}

	if _, err := rollbackManager.CreateBackup(taskID, []string{testFile}); err != nil {
		log.Fatal(err)
	}

	// User decides to rollback manually
	if err := trigger.ManualRollback(taskID, "User requested rollback - wrong approach"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Manual rollback completed")

	// Output:
	// Manual rollback completed
}

// Example demonstrates cleanup of old backups
func Example_cleanupOldBackups() {
	// Create temporary directory for example
	tempDir, err := os.MkdirTemp("", "cleanup-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize rollback manager
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
	if err != nil {
		log.Fatal(err)
	}

	// Create some backups
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		if _, err := rollbackManager.CreateBackup(taskID, []string{testFile}); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Created 3 backups")

	// Cleanup backups older than 90 days
	if err := rollbackManager.CleanupOldBackups(90); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Cleanup completed (backups within retention period are kept)")

	// Output:
	// Created 3 backups
	// Cleanup completed (backups within retention period are kept)
}
