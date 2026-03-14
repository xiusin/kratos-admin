package constitution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// taskTraceManager implements TaskTraceManager interface
type taskTraceManager struct {
	traceDir string
}

// NewTaskTraceManager creates a new TaskTraceManager instance
func NewTaskTraceManager(traceDir string) (TaskTraceManager, error) {
	// Ensure trace directory exists
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create trace directory: %w", err)
	}

	return &taskTraceManager{
		traceDir: traceDir,
	}, nil
}

// CreateTask creates a new task record
func (m *taskTraceManager) CreateTask(description string, developerRequest string) (string, error) {
	taskID := uuid.New().String()

	trace := &TaskTrace{
		TaskID:           taskID,
		TimestampStart:   time.Now(),
		Status:           TaskStatusPending,
		TaskDescription:  description,
		DeveloperRequest: developerRequest,
		Decisions:        []Decision{},
		CodeChanges:      []CodeChange{},
		Validations:      []Validation{},
		References:       []Reference{},
	}

	if err := m.saveTrace(trace); err != nil {
		return "", fmt.Errorf("failed to save task trace: %w", err)
	}

	return taskID, nil
}

// RecordDecision records a decision made during task execution
func (m *taskTraceManager) RecordDecision(taskID string, decision Decision) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	// Set timestamp if not provided
	if decision.Timestamp.IsZero() {
		decision.Timestamp = time.Now()
	}

	trace.Decisions = append(trace.Decisions, decision)

	return m.saveTrace(trace)
}

// RecordCodeChange records a code change
func (m *taskTraceManager) RecordCodeChange(taskID string, change CodeChange) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	trace.CodeChanges = append(trace.CodeChanges, change)

	return m.saveTrace(trace)
}

// RecordValidation records a validation result
func (m *taskTraceManager) RecordValidation(taskID string, validation Validation) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	// Set timestamp if not provided
	if validation.Timestamp.IsZero() {
		validation.Timestamp = time.Now()
	}

	trace.Validations = append(trace.Validations, validation)

	return m.saveTrace(trace)
}

// AddReference adds a reference to documentation or examples
func (m *taskTraceManager) AddReference(taskID string, ref Reference) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	trace.References = append(trace.References, ref)

	return m.saveTrace(trace)
}

// CompleteTask marks a task as completed
func (m *taskTraceManager) CompleteTask(taskID string) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	trace.Status = TaskStatusCompleted
	trace.TimestampEnd = time.Now()

	return m.saveTrace(trace)
}

// FailTask marks a task as failed
func (m *taskTraceManager) FailTask(taskID string, reason string) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	trace.Status = TaskStatusFailed
	trace.TimestampEnd = time.Now()

	// Add a decision record explaining the failure
	decision := Decision{
		Timestamp:    time.Now(),
		DecisionType: DecisionTypeValidation,
		Description:  "Task failed",
		Rationale:    reason,
	}
	trace.Decisions = append(trace.Decisions, decision)

	return m.saveTrace(trace)
}

// RecordRollback records a rollback operation
func (m *taskTraceManager) RecordRollback(taskID string, rollback RollbackInfo) error {
	trace, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	// Set timestamp if not provided
	if rollback.Timestamp.IsZero() {
		rollback.Timestamp = time.Now()
	}

	trace.Rollback = &rollback
	trace.Status = TaskStatusRolledBack
	trace.TimestampEnd = time.Now()

	return m.saveTrace(trace)
}

// GetTask retrieves a task record by ID
func (m *taskTraceManager) GetTask(taskID string) (*TaskTrace, error) {
	filePath := m.getTraceFilePath(taskID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to read task trace: %w", err)
	}

	var trace TaskTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("failed to parse task trace: %w", err)
	}

	return &trace, nil
}

// ListTasks lists task records with optional filters
func (m *taskTraceManager) ListTasks(filter TaskFilter) ([]*TaskTrace, error) {
	files, err := os.ReadDir(m.traceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace directory: %w", err)
	}

	var traces []*TaskTrace

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(m.traceDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var trace TaskTrace
		if err := json.Unmarshal(data, &trace); err != nil {
			continue // Skip files that can't be parsed
		}

		// Apply filters
		if filter.Status != nil && trace.Status != *filter.Status {
			continue
		}
		if filter.StartDate != nil && trace.TimestampStart.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && trace.TimestampStart.After(*filter.EndDate) {
			continue
		}

		traces = append(traces, &trace)
	}

	// Sort by timestamp (newest first)
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].TimestampStart.After(traces[j].TimestampStart)
	})

	// Apply limit and offset
	if filter.Offset > 0 {
		if filter.Offset >= len(traces) {
			return []*TaskTrace{}, nil
		}
		traces = traces[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(traces) {
		traces = traces[:filter.Limit]
	}

	return traces, nil
}

// saveTrace saves a task trace to disk
func (m *taskTraceManager) saveTrace(trace *TaskTrace) error {
	filePath := m.getTraceFilePath(trace.TaskID)

	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task trace: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task trace: %w", err)
	}

	return nil
}

// getTraceFilePath returns the file path for a task trace
func (m *taskTraceManager) getTraceFilePath(taskID string) string {
	// Use task ID as filename for simplicity
	// Format: {taskID}.json
	return filepath.Join(m.traceDir, fmt.Sprintf("%s.json", taskID))
}
