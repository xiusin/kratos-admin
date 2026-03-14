package constitution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTaskTraceManager(t *testing.T) {
	tmpDir := t.TempDir()
	traceDir := filepath.Join(tmpDir, "traces")

	manager, err := NewTaskTraceManager(traceDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	// Verify directory was created
	if _, err := os.Stat(traceDir); os.IsNotExist(err) {
		t.Errorf("Trace directory was not created")
	}
}

func TestCreateTask(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	description := "Implement user CRUD operations"
	developerRequest := "Create user management API"

	taskID, err := manager.CreateTask(description, developerRequest)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}

	// Verify task was saved
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if trace.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, trace.TaskID)
	}
	if trace.TaskDescription != description {
		t.Errorf("Expected description %s, got %s", description, trace.TaskDescription)
	}
	if trace.DeveloperRequest != developerRequest {
		t.Errorf("Expected developer request %s, got %s", developerRequest, trace.DeveloperRequest)
	}
	if trace.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s", TaskStatusPending, trace.Status)
	}
}

func TestRecordDecision(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	decision := Decision{
		DecisionType:          DecisionTypeArchitecture,
		Description:           "Use three-layer architecture",
		Rationale:             "Follows project standards",
		ConstitutionReference: "Section 4.1",
	}

	err = manager.RecordDecision(taskID, decision)
	if err != nil {
		t.Fatalf("RecordDecision failed: %v", err)
	}

	// Verify decision was recorded
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(trace.Decisions) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(trace.Decisions))
	}

	recorded := trace.Decisions[0]
	if recorded.DecisionType != decision.DecisionType {
		t.Errorf("Expected decision type %s, got %s", decision.DecisionType, recorded.DecisionType)
	}
	if recorded.Description != decision.Description {
		t.Errorf("Expected description %s, got %s", decision.Description, recorded.Description)
	}
	if recorded.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestRecordCodeChange(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	change := CodeChange{
		FilePath:     "backend/app/admin/service/internal/service/user.go",
		Operation:    OperationCreate,
		LinesAdded:   150,
		LinesRemoved: 0,
		Summary:      "Created UserService implementation",
	}

	err = manager.RecordCodeChange(taskID, change)
	if err != nil {
		t.Fatalf("RecordCodeChange failed: %v", err)
	}

	// Verify code change was recorded
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(trace.CodeChanges) != 1 {
		t.Fatalf("Expected 1 code change, got %d", len(trace.CodeChanges))
	}

	recorded := trace.CodeChanges[0]
	if recorded.FilePath != change.FilePath {
		t.Errorf("Expected file path %s, got %s", change.FilePath, recorded.FilePath)
	}
	if recorded.Operation != change.Operation {
		t.Errorf("Expected operation %s, got %s", change.Operation, recorded.Operation)
	}
}

func TestRecordValidation(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	validation := Validation{
		Validator: "gofmt",
		Status:    ValidationStatusPassed,
		Output:    "All files formatted correctly",
	}

	err = manager.RecordValidation(taskID, validation)
	if err != nil {
		t.Fatalf("RecordValidation failed: %v", err)
	}

	// Verify validation was recorded
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(trace.Validations) != 1 {
		t.Fatalf("Expected 1 validation, got %d", len(trace.Validations))
	}

	recorded := trace.Validations[0]
	if recorded.Validator != validation.Validator {
		t.Errorf("Expected validator %s, got %s", validation.Validator, recorded.Validator)
	}
	if recorded.Status != validation.Status {
		t.Errorf("Expected status %s, got %s", validation.Status, recorded.Status)
	}
	if recorded.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestAddReference(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	reference := Reference{
		Type:        ReferenceTypeDocumentation,
		Source:      "backend/app/admin/service/README.md",
		Description: "Service implementation pattern",
	}

	err = manager.AddReference(taskID, reference)
	if err != nil {
		t.Fatalf("AddReference failed: %v", err)
	}

	// Verify reference was added
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(trace.References) != 1 {
		t.Fatalf("Expected 1 reference, got %d", len(trace.References))
	}

	recorded := trace.References[0]
	if recorded.Type != reference.Type {
		t.Errorf("Expected type %s, got %s", reference.Type, recorded.Type)
	}
	if recorded.Source != reference.Source {
		t.Errorf("Expected source %s, got %s", reference.Source, recorded.Source)
	}
}

func TestCompleteTask(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	err = manager.CompleteTask(taskID)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	// Verify task was completed
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if trace.Status != TaskStatusCompleted {
		t.Errorf("Expected status %s, got %s", TaskStatusCompleted, trace.Status)
	}
	if trace.TimestampEnd.IsZero() {
		t.Error("Expected end timestamp to be set")
	}
}

func TestFailTask(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	reason := "Validation failed: syntax error"
	err = manager.FailTask(taskID, reason)
	if err != nil {
		t.Fatalf("FailTask failed: %v", err)
	}

	// Verify task was marked as failed
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if trace.Status != TaskStatusFailed {
		t.Errorf("Expected status %s, got %s", TaskStatusFailed, trace.Status)
	}
	if trace.TimestampEnd.IsZero() {
		t.Error("Expected end timestamp to be set")
	}

	// Verify failure reason was recorded as a decision
	if len(trace.Decisions) == 0 {
		t.Fatal("Expected failure reason to be recorded as a decision")
	}
	lastDecision := trace.Decisions[len(trace.Decisions)-1]
	if lastDecision.Rationale != reason {
		t.Errorf("Expected rationale %s, got %s", reason, lastDecision.Rationale)
	}
}

func TestRecordRollback(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	taskID, err := manager.CreateTask("Test task", "Test request")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	rollback := RollbackInfo{
		Triggered:     true,
		Reason:        "Architecture violation detected",
		RestoredFiles: []string{"file1.go", "file2.go"},
	}

	err = manager.RecordRollback(taskID, rollback)
	if err != nil {
		t.Fatalf("RecordRollback failed: %v", err)
	}

	// Verify rollback was recorded
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if trace.Rollback == nil {
		t.Fatal("Expected rollback to be recorded")
	}
	if !trace.Rollback.Triggered {
		t.Error("Expected rollback to be triggered")
	}
	if trace.Rollback.Reason != rollback.Reason {
		t.Errorf("Expected reason %s, got %s", rollback.Reason, trace.Rollback.Reason)
	}
	if trace.Status != TaskStatusRolledBack {
		t.Errorf("Expected status %s, got %s", TaskStatusRolledBack, trace.Status)
	}
	if trace.Rollback.Timestamp.IsZero() {
		t.Error("Expected rollback timestamp to be set")
	}
}

func TestGetTaskNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	_, err = manager.GetTask("non-existent-task-id")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestListTasks(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	// Create multiple tasks
	task1ID, _ := manager.CreateTask("Task 1", "Request 1")
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	task2ID, _ := manager.CreateTask("Task 2", "Request 2")
	time.Sleep(10 * time.Millisecond)
	task3ID, _ := manager.CreateTask("Task 3", "Request 3")

	// Complete one task
	manager.CompleteTask(task1ID)

	// List all tasks
	traces, err := manager.ListTasks(TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(traces) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(traces))
	}

	// Verify tasks are sorted by timestamp (newest first)
	if traces[0].TaskID != task3ID {
		t.Errorf("Expected first task to be %s, got %s", task3ID, traces[0].TaskID)
	}

	// Filter by status
	completedStatus := TaskStatusCompleted
	completedTraces, err := manager.ListTasks(TaskFilter{Status: &completedStatus})
	if err != nil {
		t.Fatalf("ListTasks with filter failed: %v", err)
	}

	if len(completedTraces) != 1 {
		t.Fatalf("Expected 1 completed task, got %d", len(completedTraces))
	}
	if completedTraces[0].TaskID != task1ID {
		t.Errorf("Expected completed task to be %s, got %s", task1ID, completedTraces[0].TaskID)
	}

	// Test limit
	limitedTraces, err := manager.ListTasks(TaskFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListTasks with limit failed: %v", err)
	}

	if len(limitedTraces) != 2 {
		t.Fatalf("Expected 2 tasks with limit, got %d", len(limitedTraces))
	}

	// Test offset
	offsetTraces, err := manager.ListTasks(TaskFilter{Offset: 1})
	if err != nil {
		t.Fatalf("ListTasks with offset failed: %v", err)
	}

	if len(offsetTraces) != 2 {
		t.Fatalf("Expected 2 tasks with offset, got %d", len(offsetTraces))
	}
	if offsetTraces[0].TaskID != task2ID {
		t.Errorf("Expected first task with offset to be %s, got %s", task2ID, offsetTraces[0].TaskID)
	}
}

func TestListTasksWithDateFilter(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	// Create tasks
	manager.CreateTask("Task 1", "Request 1")
	time.Sleep(10 * time.Millisecond)

	midTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	manager.CreateTask("Task 2", "Request 2")

	// Filter by start date
	traces, err := manager.ListTasks(TaskFilter{StartDate: &midTime})
	if err != nil {
		t.Fatalf("ListTasks with date filter failed: %v", err)
	}

	if len(traces) != 1 {
		t.Fatalf("Expected 1 task after midTime, got %d", len(traces))
	}
}

func TestMultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewTaskTraceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewTaskTraceManager failed: %v", err)
	}

	// Create a task and perform multiple operations
	taskID, err := manager.CreateTask("Complex task", "Implement user management")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Record decision
	manager.RecordDecision(taskID, Decision{
		DecisionType: DecisionTypeArchitecture,
		Description:  "Use Repository pattern",
		Rationale:    "Follows existing patterns",
	})

	// Record code changes
	manager.RecordCodeChange(taskID, CodeChange{
		FilePath:   "service/user.go",
		Operation:  OperationCreate,
		LinesAdded: 100,
		Summary:    "Created UserService",
	})

	manager.RecordCodeChange(taskID, CodeChange{
		FilePath:   "data/user.go",
		Operation:  OperationCreate,
		LinesAdded: 150,
		Summary:    "Created UserRepository",
	})

	// Record validations
	manager.RecordValidation(taskID, Validation{
		Validator: "gofmt",
		Status:    ValidationStatusPassed,
		Output:    "OK",
	})

	manager.RecordValidation(taskID, Validation{
		Validator: "golangci-lint",
		Status:    ValidationStatusPassed,
		Output:    "No issues found",
	})

	// Add references
	manager.AddReference(taskID, Reference{
		Type:        ReferenceTypeExample,
		Source:      "service/role.go",
		Description: "Reference implementation",
	})

	// Complete task
	manager.CompleteTask(taskID)

	// Verify all operations were recorded
	trace, err := manager.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(trace.Decisions) != 1 {
		t.Errorf("Expected 1 decision, got %d", len(trace.Decisions))
	}
	if len(trace.CodeChanges) != 2 {
		t.Errorf("Expected 2 code changes, got %d", len(trace.CodeChanges))
	}
	if len(trace.Validations) != 2 {
		t.Errorf("Expected 2 validations, got %d", len(trace.Validations))
	}
	if len(trace.References) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(trace.References))
	}
	if trace.Status != TaskStatusCompleted {
		t.Errorf("Expected status %s, got %s", TaskStatusCompleted, trace.Status)
	}
}
