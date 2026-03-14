package constitution

import "time"

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusRolledBack TaskStatus = "rolled_back"
)

// DecisionType represents the type of decision made
type DecisionType string

const (
	DecisionTypeArchitecture   DecisionType = "architecture"
	DecisionTypeImplementation DecisionType = "implementation"
	DecisionTypeValidation     DecisionType = "validation"
	DecisionTypeDocumentation  DecisionType = "documentation"
)

// OperationType represents the type of code operation
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationModify OperationType = "modify"
	OperationDelete OperationType = "delete"
)

// ValidationStatus represents the status of validation
type ValidationStatus string

const (
	ValidationStatusPassed ValidationStatus = "passed"
	ValidationStatusFailed ValidationStatus = "failed"
)

// ReferenceType represents the type of reference
type ReferenceType string

const (
	ReferenceTypeDocumentation ReferenceType = "documentation"
	ReferenceTypeExample       ReferenceType = "example"
	ReferenceTypeAPIReference  ReferenceType = "api_reference"
)

// TaskTrace represents a complete task execution record
type TaskTrace struct {
	TaskID           string        `json:"task_id"`
	TimestampStart   time.Time     `json:"timestamp_start"`
	TimestampEnd     time.Time     `json:"timestamp_end"`
	Status           TaskStatus    `json:"status"`
	TaskDescription  string        `json:"task_description"`
	DeveloperRequest string        `json:"developer_request"`
	Decisions        []Decision    `json:"decisions"`
	CodeChanges      []CodeChange  `json:"code_changes"`
	Validations      []Validation  `json:"validations"`
	References       []Reference   `json:"references"`
	Rollback         *RollbackInfo `json:"rollback,omitempty"`
}

// Decision represents a decision made during task execution
type Decision struct {
	Timestamp             time.Time    `json:"timestamp"`
	DecisionType          DecisionType `json:"decision_type"`
	Description           string       `json:"description"`
	Rationale             string       `json:"rationale"`
	ConstitutionReference string       `json:"constitution_reference"`
}

// CodeChange represents a code modification
type CodeChange struct {
	FilePath     string        `json:"file_path"`
	Operation    OperationType `json:"operation"`
	LinesAdded   int           `json:"lines_added"`
	LinesRemoved int           `json:"lines_removed"`
	Summary      string        `json:"summary"`
	DiffContent  string        `json:"diff_content,omitempty"`
}

// Validation represents a validation result
type Validation struct {
	Validator string           `json:"validator"`
	Timestamp time.Time        `json:"timestamp"`
	Status    ValidationStatus `json:"status"`
	Output    string           `json:"output"`
	Errors    []string         `json:"errors,omitempty"`
}

// Reference represents a reference to documentation or examples
type Reference struct {
	Type        ReferenceType `json:"type"`
	Source      string        `json:"source"`
	Description string        `json:"description"`
}

// RollbackInfo represents rollback information
type RollbackInfo struct {
	Triggered     bool      `json:"triggered"`
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
	RestoredFiles []string  `json:"restored_files"`
}

// ValidationResult represents the result of code validation
type ValidationResult struct {
	Passed    bool                `json:"passed"`
	Validator string              `json:"validator"`
	Timestamp time.Time           `json:"timestamp"`
	Output    string              `json:"output"`
	Errors    []ValidationError   `json:"errors,omitempty"`
	Warnings  []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// Backup represents backup information
type Backup struct {
	BackupID  string       `json:"backup_id"`
	TaskID    string       `json:"task_id"`
	Timestamp time.Time    `json:"timestamp"`
	Files     []BackupFile `json:"files"`
}

// BackupFile represents a backed up file
type BackupFile struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	Hash         string `json:"hash"`
}

// APIReference represents API reference information
type APIReference struct {
	ServiceName   string `json:"service_name"`
	MethodName    string `json:"method_name"`
	FilePath      string `json:"file_path"`
	LineNumber    int    `json:"line_number"`
	RequestType   string `json:"request_type"`
	ResponseType  string `json:"response_type"`
	Documentation string `json:"documentation"`
}

// FunctionSignature represents a function signature
type FunctionSignature struct {
	PackagePath   string   `json:"package_path"`
	FunctionName  string   `json:"function_name"`
	FilePath      string   `json:"file_path"`
	LineNumber    int      `json:"line_number"`
	Parameters    []string `json:"parameters"`
	ReturnTypes   []string `json:"return_types"`
	Documentation string   `json:"documentation"`
}

// DocumentationReport is defined in doc_syncer.go

// TaskFilter represents filters for querying tasks
type TaskFilter struct {
	Status    *TaskStatus `json:"status,omitempty"`
	StartDate *time.Time  `json:"start_date,omitempty"`
	EndDate   *time.Time  `json:"end_date,omitempty"`
	Limit     int         `json:"limit,omitempty"`
	Offset    int         `json:"offset,omitempty"`
}
