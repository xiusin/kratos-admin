package constitution_test

import (
	"context"
	"fmt"
	"log"

	"go-wind-admin/pkg/constitution"
)

// ExampleWorkflow demonstrates a complete workflow with violation detection
func ExampleWorkflow() {
	// 1. Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Create components
	traceManager, err := constitution.NewTaskTraceManagerFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create trace manager: %v", err)
	}

	validator, err := constitution.NewCodeValidatorFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	rollbackManager, err := constitution.NewRollbackManagerFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create rollback manager: %v", err)
	}

	violationDetector, err := constitution.NewViolationDetectorFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create violation detector: %v", err)
	}

	// 3. Create task
	taskID, err := traceManager.CreateTask(
		"Implement user authentication",
		"Add JWT authentication to user service",
	)
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}

	fmt.Printf("Task created: %s\n", taskID)

	// 4. Record decision
	err = traceManager.RecordDecision(taskID, constitution.Decision{
		DecisionType:          constitution.DecisionTypeImplementation,
		Description:           "Use JWT for authentication",
		Rationale:             "JWT is stateless and scalable",
		ConstitutionReference: "Section 12.3: Security Rules",
	})
	if err != nil {
		log.Fatalf("Failed to record decision: %v", err)
	}

	// 5. Simulate code generation
	files := []string{
		"backend/app/admin/service/internal/service/auth.go",
		"backend/pkg/middleware/jwt.go",
	}

	// 6. Create backup before changes
	backupID, err := rollbackManager.CreateBackup(taskID, files)
	if err != nil {
		log.Fatalf("Failed to create backup: %v", err)
	}

	fmt.Printf("Backup created: %s\n", backupID)

	// 7. Record code changes
	for _, file := range files {
		err = traceManager.RecordCodeChange(taskID, constitution.CodeChange{
			FilePath:     file,
			Operation:    constitution.OperationCreate,
			LinesAdded:   150,
			LinesRemoved: 0,
			Summary:      "Implemented JWT authentication",
		})
		if err != nil {
			log.Fatalf("Failed to record code change: %v", err)
		}
	}

	// 8. Detect violations
	report, err := violationDetector.DetectAllViolations(context.Background(), files)
	if err != nil {
		log.Fatalf("Failed to detect violations: %v", err)
	}

	fmt.Printf("\nViolation Detection:\n")
	fmt.Printf("- Total violations: %d\n", len(report.Violations))
	fmt.Printf("- Critical: %d\n", report.CriticalCount)
	fmt.Printf("- High: %d\n", report.HighCount)

	// 9. Check if rollback is needed
	if report.ShouldRollback {
		fmt.Println("\n⚠️  Critical violations detected! Triggering rollback...")

		// Rollback changes
		err = rollbackManager.Rollback(taskID, "Critical constitution violations detected")
		if err != nil {
			log.Fatalf("Failed to rollback: %v", err)
		}

		// Record rollback in trace
		err = traceManager.RecordRollback(taskID, constitution.RollbackInfo{
			Triggered:     true,
			Reason:        "Critical constitution violations detected",
			RestoredFiles: files,
		})
		if err != nil {
			log.Fatalf("Failed to record rollback: %v", err)
		}

		// Fail task
		err = traceManager.FailTask(taskID, "Rolled back due to constitution violations")
		if err != nil {
			log.Fatalf("Failed to fail task: %v", err)
		}

		fmt.Println("✅ Rollback completed")
		return
	}

	// 10. Validate code
	for _, file := range files {
		result, err := validator.ValidateGoCode(file)
		if err != nil {
			log.Printf("Validation error for %s: %v", file, err)
			continue
		}

		err = traceManager.RecordValidation(taskID, constitution.Validation{
			Validator: result.Validator,
			Status:    constitution.ValidationStatusPassed,
			Output:    result.Output,
		})
		if err != nil {
			log.Fatalf("Failed to record validation: %v", err)
		}
	}

	// 11. Complete task
	err = traceManager.CompleteTask(taskID)
	if err != nil {
		log.Fatalf("Failed to complete task: %v", err)
	}

	fmt.Println("\n✅ Task completed successfully")
}

// ExampleRollbackTrigger demonstrates automatic rollback on violations
func ExampleRollbackTrigger() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create components
	traceManager, _ := constitution.NewTaskTraceManagerFromConfig(cfg)
	validator, _ := constitution.NewCodeValidatorFromConfig(cfg)
	rollbackManager, _ := constitution.NewRollbackManagerFromConfig(cfg)
	violationDetector, _ := constitution.NewViolationDetectorFromConfig(cfg)

	// Create rollback trigger
	trigger := constitution.NewRollbackTrigger(rollbackManager, traceManager, validator)

	// Create task
	taskID, _ := traceManager.CreateTask("Test task", "Testing rollback trigger")

	// Files with violations
	files := []string{
		"backend/app/admin/service/internal/service/user.go",
	}

	// Create backup
	rollbackManager.CreateBackup(taskID, files)

	// Detect violations
	report, _ := violationDetector.DetectAllViolations(context.Background(), files)

	// Check if rollback should be triggered
	if report.ShouldRollback {
		fmt.Println("Triggering automatic rollback...")

		// Trigger rollback
		err := trigger.TriggerOnViolation(taskID, report.Violations)
		if err != nil {
			log.Fatalf("Failed to trigger rollback: %v", err)
		}

		fmt.Println("✅ Automatic rollback completed")
	}
}

// ExampleViolationReporting demonstrates violation reporting
func ExampleViolationReporting() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := constitution.NewRuleEngine(cfg)

	// Example violations
	violations := []constitution.Violation{
		{
			Type:                  constitution.ViolationTypeSecurity,
			Severity:              constitution.SeverityCritical,
			FilePath:              "backend/app/admin/service/internal/service/auth.go",
			LineNumber:            45,
			Description:           "Hardcoded password detected",
			Rule:                  "sec-001",
			ConstitutionReference: "Section 5.4: Security Violations",
			Suggestion:            "Use environment variables for sensitive data",
		},
		{
			Type:                  constitution.ViolationTypeArchitecture,
			Severity:              constitution.SeverityHigh,
			FilePath:              "backend/pkg/utils/helper.go",
			LineNumber:            12,
			Description:           "pkg/ layer importing app/ layer",
			Rule:                  "arch-001",
			ConstitutionReference: "Section 3.1: Three-Layer Architecture",
			Suggestion:            "Move shared code to pkg/ or use dependency inversion",
		},
		{
			Type:                  constitution.ViolationTypeSecurity,
			Severity:              constitution.SeverityHigh,
			FilePath:              "backend/app/admin/service/internal/service/user.go",
			LineNumber:            78,
			Description:           "Sensitive data in logs",
			Rule:                  "sec-005",
			ConstitutionReference: "Section 12.5: Security and Performance",
			Suggestion:            "Avoid logging passwords, tokens, or keys",
		},
	}

	// Generate formatted report
	report := engine.GenerateViolationReport(violations)
	fmt.Println(report)

	// Generate fix suggestions
	suggestions := engine.GenerateFixSuggestions(violations)
	fmt.Println("\n📋 Fix Suggestions:")
	for i, s := range suggestions {
		fmt.Printf("%d. %s\n", i+1, s)
	}

	// Evaluate severity
	severity := engine.EvaluateSeverity(violations)
	fmt.Printf("\n⚠️  Overall Severity: %s\n", severity)

	// Check if rollback is needed
	shouldRollback := engine.ShouldRollback(violations)
	fmt.Printf("🔄 Should Rollback: %v\n", shouldRollback)
}
