package constitution_test

import (
	"fmt"
	"log"

	"go-wind-admin/pkg/constitution"
)

// ExampleTaskTraceManager demonstrates the basic usage of TaskTraceManager
func ExampleTaskTraceManager() {
	// Create a new task trace manager
	manager, err := constitution.NewTaskTraceManager(".ai/traces")
	if err != nil {
		log.Fatal(err)
	}

	// Create a new task
	taskID, err := manager.CreateTask(
		"Implement user CRUD operations",
		"Create user management API with full CRUD support",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created task: %s\n", taskID)

	// Record a decision
	err = manager.RecordDecision(taskID, constitution.Decision{
		DecisionType:          constitution.DecisionTypeArchitecture,
		Description:           "Use Repository pattern for data access",
		Rationale:             "Follows existing project patterns and maintains consistency",
		ConstitutionReference: "Section 4.1 - Three-Layer Architecture",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Record code changes
	err = manager.RecordCodeChange(taskID, constitution.CodeChange{
		FilePath:     "backend/app/admin/service/internal/service/user.go",
		Operation:    constitution.OperationCreate,
		LinesAdded:   150,
		LinesRemoved: 0,
		Summary:      "Created UserService implementation with CRUD methods",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Record validation results
	err = manager.RecordValidation(taskID, constitution.Validation{
		Validator: "gofmt",
		Status:    constitution.ValidationStatusPassed,
		Output:    "All files formatted correctly",
	})
	if err != nil {
		log.Fatal(err)
	}

	err = manager.RecordValidation(taskID, constitution.Validation{
		Validator: "golangci-lint",
		Status:    constitution.ValidationStatusPassed,
		Output:    "No issues found",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Add reference to existing code
	err = manager.AddReference(taskID, constitution.Reference{
		Type:        constitution.ReferenceTypeExample,
		Source:      "backend/app/admin/service/internal/service/role.go",
		Description: "Reference implementation for service layer pattern",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Complete the task
	err = manager.CompleteTask(taskID)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve and display the task
	trace, err := manager.GetTask(taskID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task Status: %s\n", trace.Status)
	fmt.Printf("Decisions: %d\n", len(trace.Decisions))
	fmt.Printf("Code Changes: %d\n", len(trace.CodeChanges))
	fmt.Printf("Validations: %d\n", len(trace.Validations))
	fmt.Printf("References: %d\n", len(trace.References))
}

// ExampleTaskTraceManager_rollback demonstrates rollback scenario
func ExampleTaskTraceManager_rollback() {
	manager, err := constitution.NewTaskTraceManager(".ai/traces")
	if err != nil {
		log.Fatal(err)
	}

	// Create a task
	taskID, err := manager.CreateTask(
		"Modify authentication logic",
		"Update JWT token validation",
	)
	if err != nil {
		log.Fatal(err)
	}

	// Record code change
	err = manager.RecordCodeChange(taskID, constitution.CodeChange{
		FilePath:     "backend/pkg/middleware/auth.go",
		Operation:    constitution.OperationModify,
		LinesAdded:   20,
		LinesRemoved: 15,
		Summary:      "Updated JWT validation logic",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Validation fails
	err = manager.RecordValidation(taskID, constitution.Validation{
		Validator: "go test",
		Status:    constitution.ValidationStatusFailed,
		Output:    "TestJWTValidation failed",
		Errors:    []string{"Expected valid token to pass validation"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Trigger rollback
	err = manager.RecordRollback(taskID, constitution.RollbackInfo{
		Triggered:     true,
		Reason:        "Validation failed: JWT tests failing",
		RestoredFiles: []string{"backend/pkg/middleware/auth.go"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve task
	trace, err := manager.GetTask(taskID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task Status: %s\n", trace.Status)
	fmt.Printf("Rollback Triggered: %v\n", trace.Rollback.Triggered)
	fmt.Printf("Rollback Reason: %s\n", trace.Rollback.Reason)
}

// ExampleTaskTraceManager_listTasks demonstrates listing and filtering tasks
func ExampleTaskTraceManager_listTasks() {
	manager, err := constitution.NewTaskTraceManager(".ai/traces")
	if err != nil {
		log.Fatal(err)
	}

	// Create multiple tasks
	task1, _ := manager.CreateTask("Task 1", "Request 1")
	manager.CompleteTask(task1)

	task2, _ := manager.CreateTask("Task 2", "Request 2")
	manager.FailTask(task2, "Validation failed")

	manager.CreateTask("Task 3", "Request 3")

	// List all tasks
	allTasks, err := manager.ListTasks(constitution.TaskFilter{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total tasks: %d\n", len(allTasks))

	// List only completed tasks
	completedStatus := constitution.TaskStatusCompleted
	completedTasks, err := manager.ListTasks(constitution.TaskFilter{
		Status: &completedStatus,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Completed tasks: %d\n", len(completedTasks))

	// List with limit
	limitedTasks, err := manager.ListTasks(constitution.TaskFilter{
		Limit: 2,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Limited tasks: %d\n", len(limitedTasks))
}
