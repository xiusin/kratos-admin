package constitution

import "context"

// TaskTraceManager manages task trace records
type TaskTraceManager interface {
	// CreateTask creates a new task record
	CreateTask(description string, developerRequest string) (taskID string, err error)

	// RecordDecision records a decision made during task execution
	RecordDecision(taskID string, decision Decision) error

	// RecordCodeChange records a code change
	RecordCodeChange(taskID string, change CodeChange) error

	// RecordValidation records a validation result
	RecordValidation(taskID string, validation Validation) error

	// AddReference adds a reference to documentation or examples
	AddReference(taskID string, ref Reference) error

	// CompleteTask marks a task as completed
	CompleteTask(taskID string) error

	// FailTask marks a task as failed
	FailTask(taskID string, reason string) error

	// RecordRollback records a rollback operation
	RecordRollback(taskID string, rollback RollbackInfo) error

	// GetTask retrieves a task record by ID
	GetTask(taskID string) (*TaskTrace, error)

	// ListTasks lists task records with optional filters
	ListTasks(filter TaskFilter) ([]*TaskTrace, error)
}

// CodeValidator validates generated code
type CodeValidator interface {
	// ValidateGoCode validates Go code
	ValidateGoCode(filePath string) (*ValidationResult, error)

	// ValidateVueCode validates Vue code
	ValidateVueCode(filePath string) (*ValidationResult, error)

	// ValidateProtobuf validates Protobuf schema
	ValidateProtobuf(filePath string) (*ValidationResult, error)

	// ValidateEntSchema validates Ent schema
	ValidateEntSchema(schemaDir string) (*ValidationResult, error)

	// ValidateImports validates that imports exist
	ValidateImports(filePath string, language string) (*ValidationResult, error)

	// RunTests runs tests matching the pattern
	RunTests(testPattern string) (*ValidationResult, error)
}

// AntiHallucinationVerifier verifies the existence of code elements
type AntiHallucinationVerifier interface {
	// VerifyAPIExists verifies that an API exists in Protobuf definitions
	VerifyAPIExists(serviceName, methodName string) (bool, error)

	// VerifyFunctionExists verifies that a function exists in the codebase
	VerifyFunctionExists(packagePath, functionName string) (bool, error)

	// VerifyModuleExists verifies that a module exists
	VerifyModuleExists(modulePath string, language string) (bool, error)

	// VerifyConfigKeyExists verifies that a configuration key exists
	VerifyConfigKeyExists(configKey string) (bool, error)

	// GetAPIReference retrieves API reference information
	GetAPIReference(serviceName, methodName string) (*APIReference, error)

	// GetFunctionSignature retrieves function signature information
	GetFunctionSignature(packagePath, functionName string) (*FunctionSignature, error)
}

// RollbackManager manages code rollback operations
type RollbackManager interface {
	// CreateBackup creates a backup of files before modification
	CreateBackup(taskID string, files []string) (backupID string, err error)

	// Rollback rolls back changes for a task
	Rollback(taskID string, reason string) error

	// GetBackup retrieves backup information
	GetBackup(backupID string) (*Backup, error)

	// ListBackups lists backups for a task
	ListBackups(taskID string) ([]*Backup, error)

	// CleanupOldBackups removes backups older than retention period
	CleanupOldBackups(retentionDays int) error
}

// DocumentationSyncer is defined in doc_syncer.go

// ViolationDetector is defined in violation_detector.go

// RuleEngine is defined in rule_engine.go as a concrete type
