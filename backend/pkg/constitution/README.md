# Constitution Package

This package implements the AI Programming Constitution system for the GO + Vue backend management framework.

## Overview

The Constitution package provides core types and interfaces for:

- **Task Tracing**: Recording all AI operations and decisions
- **Code Validation**: Validating generated code against project standards
- **Anti-Hallucination**: Verifying existence of referenced code elements
- **Rollback Management**: Rolling back invalid code changes
- **Documentation Sync**: Keeping documentation in sync with code

## Core Types

### TaskTrace
Complete record of a task execution including decisions, code changes, validations, and references.

### Decision
Records a decision made during task execution with rationale and constitution reference.

### CodeChange
Represents a code modification (create, modify, or delete operation).

### ValidationResult
Result of code validation including errors and warnings.

### RollbackInfo
Information about a rollback operation.

## Interfaces

### TaskTraceManager
Manages task trace records throughout the task lifecycle.

**Implementation**: `taskTraceManager` (see `trace.go`)

**Key Methods**:
- `CreateTask`: Creates a new task record with UUID
- `RecordDecision`: Records decisions with rationale and constitution references
- `RecordCodeChange`: Records file modifications
- `RecordValidation`: Records validation results
- `AddReference`: Adds references to documentation or examples
- `CompleteTask`: Marks task as completed
- `FailTask`: Marks task as failed with reason
- `RecordRollback`: Records rollback operations
- `GetTask`: Retrieves task by ID
- `ListTasks`: Lists tasks with filtering support

### CodeValidator
Validates code using appropriate tools (gofmt, eslint, protoc, etc.).

**Implementation**: `codeValidator` (see `validator.go`)

**Key Methods**:
- `ValidateGoCode`: Validates Go code using gofmt and golangci-lint
- `ValidateVueCode`: Validates Vue code using eslint and vue-tsc
- `ValidateProtobuf`: Validates Protobuf schemas using protoc
- `ValidateEntSchema`: Validates Ent schemas by running ent generate
- `ValidateImports`: Checks that all imports exist
- `RunTests`: Runs tests matching a pattern

**Usage Example**:
```go
// Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create validator
validator, err := constitution.NewCodeValidator(cfg)
if err != nil {
    log.Fatal(err)
}

// Validate Go code
result, err := validator.ValidateGoCode("backend/pkg/constitution/validator.go")
if err != nil {
    log.Fatal(err)
}

if !result.Passed {
    fmt.Printf("Validation failed with %d errors:\n", len(result.Errors))
    for _, err := range result.Errors {
        fmt.Printf("  %s:%d:%d: %s\n", err.File, err.Line, err.Column, err.Message)
    }
}

// Validate Vue code
vueResult, err := validator.ValidateVueCode("frontend/apps/admin/src/views/user/index.vue")
if err != nil {
    log.Fatal(err)
}

// Run tests
testResult, err := validator.RunTests("./backend/pkg/constitution/...")
if err != nil {
    log.Fatal(err)
}
```

### AntiHallucinationVerifier
Verifies that referenced APIs, functions, modules, and config keys exist.

**Implementation**: `antiHallucinationVerifier` (see `verifier.go`)

**Key Methods**:
- `VerifyAPIExists`: Verifies that an API exists in Protobuf definitions
- `VerifyFunctionExists`: Verifies that a function exists in the codebase
- `VerifyModuleExists`: Verifies that a module exists in dependencies
- `VerifyConfigKeyExists`: Verifies that a configuration key exists
- `GetAPIReference`: Retrieves API reference information
- `GetFunctionSignature`: Retrieves function signature information

**Usage Example**:
```go
// Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create verifier
verifier, err := constitution.NewAntiHallucinationVerifier(cfg)
if err != nil {
    log.Fatal(err)
}

// Verify API exists
exists, err := verifier.VerifyAPIExists("UserService", "CreateUser")
if err != nil {
    log.Fatal(err)
}

if !exists {
    fmt.Println("Warning: UserService.CreateUser does not exist!")
}

// Get API reference
ref, err := verifier.GetAPIReference("UserService", "CreateUser")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Service: %s\n", ref.ServiceName)
fmt.Printf("Method: %s\n", ref.MethodName)
fmt.Printf("Request: %s\n", ref.RequestType)
fmt.Printf("Response: %s\n", ref.ResponseType)
fmt.Printf("Documentation: %s\n", ref.Documentation)

// Verify function exists
exists, err = verifier.VerifyFunctionExists("pkg/utils", "FormatString")
if err != nil {
    log.Fatal(err)
}

// Get function signature
sig, err := verifier.GetFunctionSignature("pkg/utils", "FormatString")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Function: %s\n", sig.FunctionName)
fmt.Printf("Package: %s\n", sig.PackagePath)
fmt.Printf("Parameters: %v\n", sig.Parameters)
fmt.Printf("Returns: %v\n", sig.ReturnTypes)

// Verify module exists
exists, err = verifier.VerifyModuleExists("github.com/go-kratos/kratos/v2", "go")
if err != nil {
    log.Fatal(err)
}

// Verify config key exists
exists, err = verifier.VerifyConfigKeyExists("server.http.addr")
if err != nil {
    log.Fatal(err)
}
```

### Index Database

The `IndexDatabase` provides persistent storage for code element indexes:

```go
// Create index database
db, err := constitution.NewIndexDatabase(".ai/index.json")
if err != nil {
    log.Fatal(err)
}

// Add API reference
db.AddAPIReference("UserService.CreateUser", &constitution.APIReference{
    ServiceName:  "UserService",
    MethodName:   "CreateUser",
    RequestType:  "CreateUserRequest",
    ResponseType: "User",
})

// Add function signature
db.AddFunctionSignature("pkg/utils.FormatString", &constitution.FunctionSignature{
    PackagePath:  "pkg/utils",
    FunctionName: "FormatString",
    Parameters:   []string{"s string"},
    ReturnTypes:  []string{"string"},
})

// Add module
db.AddModule("go:github.com/go-kratos/kratos/v2")

// Add config key
db.AddConfigKey("server.http.addr")

// Save database
if err := db.Save(); err != nil {
    log.Fatal(err)
}

// Get statistics
stats := db.Stats()
fmt.Printf("API count: %v\n", stats["api_count"])
fmt.Printf("Function count: %v\n", stats["func_count"])
fmt.Printf("Module count: %v\n", stats["module_count"])
fmt.Printf("Config count: %v\n", stats["config_count"])
```

### Index Update Trigger

The `IndexUpdateTrigger` automatically updates indexes when files change:

```go
// Create verifier
verifier, err := constitution.NewAntiHallucinationVerifier(cfg)
if err != nil {
    log.Fatal(err)
}

// Create index update trigger
trigger := constitution.NewIndexUpdateTrigger(cfg, verifier.(*constitution.AntiHallucinationVerifier))

// Start watching for file changes (checks every 5 seconds)
trigger.Start(5 * time.Second)
defer trigger.Stop()

// Manually trigger update
if err := trigger.TriggerManualUpdate(); err != nil {
    log.Fatal(err)
}

// Trigger update on specific file changes
if err := trigger.OnProtoFileChange("backend/api/protos/user/service/v1/user.proto"); err != nil {
    log.Fatal(err)
}

if err := trigger.OnGoFileChange("backend/pkg/utils/string.go"); err != nil {
    log.Fatal(err)
}

if err := trigger.OnConfigFileChange("backend/app/admin/service/configs/config.yaml"); err != nil {
    log.Fatal(err)
}

if err := trigger.OnDependencyChange("backend/go.mod"); err != nil {
    log.Fatal(err)
}
```

### Anti-Hallucination Workflow

```go
// Create verifier
verifier, err := constitution.NewAntiHallucinationVerifier(cfg)
if err != nil {
    log.Fatal(err)
}

// Before generating code that calls an API
serviceName := "UserService"
methodName := "CreateUser"

exists, err := verifier.VerifyAPIExists(serviceName, methodName)
if err != nil {
    log.Fatal(err)
}

if !exists {
    // Ask developer for confirmation
    fmt.Printf("Warning: API %s.%s does not exist. Do you want to create it? (y/n): ", serviceName, methodName)
    var response string
    fmt.Scanln(&response)
    
    if response != "y" {
        log.Fatal("Operation cancelled")
    }
}

// Before importing a package
packagePath := "github.com/example/newlib"
exists, err = verifier.VerifyModuleExists(packagePath, "go")
if err != nil {
    log.Fatal(err)
}

if !exists {
    fmt.Printf("Warning: Module %s is not in go.mod. Please add it first.\n", packagePath)
    log.Fatal("Missing dependency")
}

// Before using a config key
configKey := "server.http.max_connections"
exists, err = verifier.VerifyConfigKeyExists(configKey)
if err != nil {
    log.Fatal(err)
}

if !exists {
    fmt.Printf("Warning: Config key %s does not exist. Using default value.\n", configKey)
}
```

### RollbackManager
Creates backups and rolls back changes when violations are detected.

**Implementation**: `rollbackManager` (see `rollback.go`)

**Key Methods**:
- `CreateBackup`: Creates a backup of files before modification with SHA256 hash
- `Rollback`: Rolls back changes for a task, restoring files from backup
- `GetBackup`: Retrieves backup information by backup ID
- `ListBackups`: Lists all backups for a task
- `CleanupOldBackups`: Removes backups older than retention period

**Usage Example**:
```go
// Create rollback manager
backupDir := ".ai/backups"
traceDir := ".ai/traces"
rollbackManager, err := constitution.NewRollbackManager(backupDir, traceDir)
if err != nil {
    log.Fatal(err)
}

// Create backup before modifying files
taskID := "task-001"
files := []string{
    "backend/app/admin/service/internal/service/user.go",
    "backend/app/admin/service/internal/data/user.go",
}

backupID, err := rollbackManager.CreateBackup(taskID, files)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created backup: %s\n", backupID)

// ... make changes to files ...

// If something goes wrong, rollback
if err := rollbackManager.Rollback(taskID, "Validation failed"); err != nil {
    log.Fatal(err)
}
fmt.Println("Changes rolled back successfully")

// List all backups for a task
backups, err := rollbackManager.ListBackups(taskID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d backups\n", len(backups))

// Cleanup old backups (older than 90 days)
if err := rollbackManager.CleanupOldBackups(90); err != nil {
    log.Fatal(err)
}
```

### RollbackTrigger
Automatically triggers rollback based on validation failures and constitution violations.

**Implementation**: `RollbackTrigger` (see `rollback_trigger.go`)

**Key Methods**:
- `CheckValidationResult`: Checks if validation result should trigger rollback
- `CheckConstitutionViolation`: Checks if code changes violate constitution rules
- `TriggerRollback`: Triggers a rollback for a task
- `ManualRollback`: Triggers a manual rollback
- `AutoRollbackOnValidation`: Automatically checks and triggers rollback on validation failure
- `AutoRollbackOnViolation`: Automatically checks and triggers rollback on constitution violation

**Trigger Conditions**:
- `validation_failure`: Critical validation errors (syntax, lint, type errors)
- `constitution_violation`: Forbidden file modifications, architecture violations
- `security_violation`: Hardcoded secrets, passwords, API keys
- `architecture_violation`: Layer dependency violations
- `manual`: User-requested rollback

**Usage Example**:
```go
// Create components
rollbackManager, _ := constitution.NewRollbackManager(".ai/backups", ".ai/traces")
traceManager, _ := constitution.NewTaskTraceManager(".ai/traces")
validator, _ := constitution.NewCodeValidator(cfg)

// Create rollback trigger
trigger := constitution.NewRollbackTrigger(rollbackManager, traceManager, validator)

// Create task and backup
taskID, _ := traceManager.CreateTask("Implement feature", "Add user export")
files := []string{"backend/app/admin/service/internal/service/user.go"}
rollbackManager.CreateBackup(taskID, files)

// Validate code
result, err := validator.ValidateGoCode("backend/app/admin/service/internal/service/user.go")
if err != nil {
    log.Fatal(err)
}

// Auto rollback on validation failure
if err := trigger.AutoRollbackOnValidation(taskID, result); err != nil {
    log.Printf("Rollback triggered: %v", err)
}

// Check for constitution violations
changes := []constitution.CodeChange{
    {
        FilePath:  "Dockerfile",
        Operation: constitution.OperationModify,
        Summary:   "Modified Dockerfile",
    },
}

// Auto rollback on violation
if err := trigger.AutoRollbackOnViolation(taskID, changes); err != nil {
    log.Printf("Rollback triggered: %v", err)
}

// Manual rollback
if err := trigger.ManualRollback(taskID, "User requested rollback"); err != nil {
    log.Fatal(err)
}
```

**Forbidden File Modifications**:
- `Dockerfile`, `docker-compose.yaml`: Container configuration
- `.golangci.yml`, `.eslintrc`: Linter configuration
- `buf.yaml`, `buf.gen.yaml`: Protobuf configuration
- `configs/*-prod.yaml`: Production configuration
- Migration files (deletion forbidden)
- Protobuf files (deletion forbidden)

**Security Violation Detection**:
- Hardcoded passwords: `password =`
- Hardcoded secrets: `secret =`
- Hardcoded API keys: `api_key =`
- Hardcoded tokens: `token =`
- Private keys: `BEGIN PRIVATE KEY`, `BEGIN RSA PRIVATE KEY`

### DocumentationSyncer
Synchronizes documentation with code changes, automatically generating and updating documentation from source code.

**Implementation**: `documentationSyncer` (see `doc_syncer_impl.go`)

**Key Methods**:
- `SyncAPIDocumentation`: Generates API documentation from Protobuf files
- `SyncComponentDocumentation`: Generates component documentation from Vue files
- `SyncFeatureDocumentation`: Generates feature documentation from code changes
- `GenerateAPIReference`: Generates complete API reference (concurrent processing)
- `ValidateDocumentation`: Validates documentation completeness and coverage
- `DetectChanges`: Detects documentation changes using diff algorithm
- `BuildSearchIndex`: Builds full-text search index for documentation
- `SearchDocumentation`: Searches documentation with keyword matching
- `GetDocumentationVersion`: Retrieves specific documentation version
- `ListDocumentationVersions`: Lists all versions of a document

**Features**:
- **Change Detection**: Only regenerates documentation when source code changes
- **Version Management**: Maintains version history of all documentation
- **Search Index**: Full-text search across all documentation
- **Source Links**: Automatic links from documentation to source code
- **Concurrent Generation**: Parallel processing of multiple files
- **Template-Based**: Customizable documentation templates

**Usage Example**:
```go
// Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create documentation syncer
syncer := constitution.NewDocumentationSyncer(cfg)

// Sync API documentation from a proto file
result, err := syncer.SyncAPIDocumentation(
    context.Background(),
    "backend/api/protos/identity/service/v1/user.proto",
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Documentation: %s\n", result.FilePath)
fmt.Printf("Changed: %v\n", result.Changed)
fmt.Printf("Source: %s\n", result.SourceLink)

if result.Changed {
    fmt.Printf("Changes: %s\n", result.ChangesSummary)
}

// Sync component documentation from a Vue file
compResult, err := syncer.SyncComponentDocumentation(
    context.Background(),
    "frontend/apps/admin/src/components/UserList.vue",
)
if err != nil {
    log.Fatal(err)
}

// Generate complete API reference (processes all proto files concurrently)
if err := syncer.GenerateAPIReference(context.Background(), "docs/api"); err != nil {
    log.Fatal(err)
}

// Validate documentation completeness
report, err := syncer.ValidateDocumentation(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total APIs: %d\n", report.TotalAPIs)
fmt.Printf("Documented: %d\n", report.DocumentedAPIs)
fmt.Printf("Coverage: %.1f%%\n", report.CoveragePercent)
fmt.Printf("Missing: %d\n", len(report.MissingDocs))

// Build search index
if err := syncer.BuildSearchIndex(context.Background()); err != nil {
    log.Fatal(err)
}

// Search documentation
results, err := syncer.SearchDocumentation(context.Background(), "user authentication")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d results\n", len(results))
for _, result := range results {
    fmt.Printf("- %s (score: %.1f)\n", result.Title, result.Score)
    fmt.Printf("  %s\n", result.Snippet)
}

// Get documentation version
version, err := syncer.GetDocumentationVersion(
    context.Background(),
    "docs/api/userservice.md",
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Version: %s\n", version.Version)
fmt.Printf("Created: %s\n", version.CreatedAt.Format("2006-01-02"))

// List all versions
versions, err := syncer.ListDocumentationVersions(
    context.Background(),
    "docs/api/userservice.md",
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d versions\n", len(versions))
```

**Documentation Templates**:

The syncer uses customizable templates located in `.ai/templates/`:

- `api-doc.tmpl`: API documentation template (Protobuf → Markdown)
- `component-doc.tmpl`: Component documentation template (Vue → Markdown)
- `feature-doc.tmpl`: Feature documentation template (Changes → Markdown)

**Template Variables**:

API Template:
- `ServiceName`: Service name from proto
- `Package`: Package name
- `Description`: Service description
- `Methods`: Array of RPC methods
- `Messages`: Array of message definitions
- `SourceFile`: Source proto file path
- `SourceLink`: Link to source code

Component Template:
- `Name`: Component name
- `Description`: Component description
- `Props`: Array of component props
- `Events`: Array of component events
- `Slots`: Array of component slots
- `Examples`: Usage examples
- `SourceFile`: Source Vue file path
- `SourceLink`: Link to source code

**Change Detection**:

The syncer uses SHA256 hashing to detect changes:

```go
// Detect changes before regenerating
diff, err := syncer.DetectChanges(context.Background(), "docs/api/userservice.md")
if err != nil {
    log.Fatal(err)
}

if diff.Changed {
    fmt.Printf("Added lines: %d\n", len(diff.AddedLines))
    fmt.Printf("Removed lines: %d\n", len(diff.RemovedLines))
    fmt.Printf("Modified sections: %d\n", len(diff.ModifiedSections))
}
```

**Version Management**:

All documentation changes are versioned:

```go
// Get latest version
latest, err := syncer.GetDocumentationVersion(ctx, "docs/api/userservice.md")

// List all versions (sorted by date, newest first)
versions, err := syncer.ListDocumentationVersions(ctx, "docs/api/userservice.md")

// Each version includes:
// - Version ID (e.g., "v20260312-143000")
// - Content snapshot
// - SHA256 hash
// - Creation timestamp
// - Author
// - Commit message
```

**Search Index**:

The search index provides fast full-text search:

```go
// Build index (scans all documentation files)
if err := syncer.BuildSearchIndex(ctx); err != nil {
    log.Fatal(err)
}

// Search with multiple keywords
results, err := syncer.SearchDocumentation(ctx, "user authentication jwt")

// Results are ranked by:
// - Title matches (weight: 10x)
// - Content matches (weight: 1x)
// - Keyword matches (weight: 2x)

// Each result includes:
// - Document path
// - Title
// - Snippet (context around match)
// - Score
// - Source file link
```

**Concurrent Processing**:

The `GenerateAPIReference` method processes multiple files concurrently:

```go
// Processes all proto files in parallel
err := syncer.GenerateAPIReference(ctx, "docs/api")

// Internally uses goroutines and sync.WaitGroup
// Collects errors from all goroutines
// Returns first error encountered
```

**Integration with Task Tracing**:

```go
// Create task
taskID, _ := traceManager.CreateTask("Update API docs", "Sync user service docs")

// Sync documentation
result, err := syncer.SyncAPIDocumentation(ctx, "backend/api/protos/user.proto")
if err != nil {
    traceManager.FailTask(taskID, err.Error())
    log.Fatal(err)
}

// Record in task trace
traceManager.RecordCodeChange(taskID, constitution.CodeChange{
    FilePath:     result.FilePath,
    Operation:    constitution.OperationModify,
    LinesAdded:   len(strings.Split(result.Content, "\n")),
    Summary:      result.ChangesSummary,
})

traceManager.CompleteTask(taskID)
```

## Configuration

### ConfigLoader

The `ConfigLoader` provides configuration management with hot reloading support:

```go
// Create a new configuration loader
loader, err := constitution.NewConfigLoader(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Get the current configuration
config := loader.Get()

// Access configuration values
fmt.Printf("Trace Directory: %s\n", config.Trace.Directory)
fmt.Printf("Auto Rollback: %v\n", config.Rollback.AutoRollback)

// Get tool commands
goFormatter, err := config.GetToolCommand("go.formatter")
if err != nil {
    log.Fatal(err)
}
```

### Hot Reloading

Enable automatic configuration reloading when the config file changes:

```go
// Start watching for configuration changes
if err := loader.StartWatching(); err != nil {
    log.Fatal(err)
}
defer loader.StopWatching()

// Register a callback for configuration changes
loader.OnChange(func(newConfig *constitution.Config) {
    fmt.Printf("Configuration reloaded! Version: %s\n", newConfig.Version)
})
```

### Configuration Structure

The configuration file (`.ai/config.yaml`) includes:

- **Tools**: Command configurations for Go, Vue, Protobuf, and Ent tools
- **Validation**: Validation rules and settings for each language
- **Trace**: Task trace settings (directory, format, retention)
- **Rollback**: Rollback behavior and backup settings
- **Error Handling**: Retry configuration and error reporting
- **Documentation**: Documentation sync settings
- **Anti-Hallucination**: Code element verification settings
- **Architecture**: Layer definitions and dependency rules
- **Security**: Security check configurations
- **Performance**: Performance validation settings

### Configuration Validation

The loader automatically validates configuration on load:

- Required fields (version, trace directory, etc.)
- Valid enum values (trace format, backoff strategy)
- Positive values for timeouts and limits
- Tool command completeness

Invalid configurations will return an error during loading.

## Usage Examples

### Task Tracing

```go
// Create a task trace manager
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

// Record a decision
err = manager.RecordDecision(taskID, constitution.Decision{
    DecisionType:          constitution.DecisionTypeArchitecture,
    Description:           "Use Repository pattern for data access",
    Rationale:             "Follows existing project patterns",
    ConstitutionReference: "Section 4.1 - Three-Layer Architecture",
})

// Record code changes
err = manager.RecordCodeChange(taskID, constitution.CodeChange{
    FilePath:     "backend/app/admin/service/internal/service/user.go",
    Operation:    constitution.OperationCreate,
    LinesAdded:   150,
    Summary:      "Created UserService implementation",
})

// Record validation results
err = manager.RecordValidation(taskID, constitution.Validation{
    Validator: "gofmt",
    Status:    constitution.ValidationStatusPassed,
    Output:    "All files formatted correctly",
})

// Add reference to existing code
err = manager.AddReference(taskID, constitution.Reference{
    Type:        constitution.ReferenceTypeExample,
    Source:      "backend/app/admin/service/internal/service/role.go",
    Description: "Reference implementation pattern",
})

// Complete the task
err = manager.CompleteTask(taskID)
```

### Rollback Scenario

```go
// Create task
taskID, _ := manager.CreateTask("Modify auth logic", "Update JWT validation")

// Record code change
manager.RecordCodeChange(taskID, constitution.CodeChange{
    FilePath:  "backend/pkg/middleware/auth.go",
    Operation: constitution.OperationModify,
    Summary:   "Updated JWT validation",
})

// Validation fails
manager.RecordValidation(taskID, constitution.Validation{
    Validator: "go test",
    Status:    constitution.ValidationStatusFailed,
    Errors:    []string{"TestJWTValidation failed"},
})

// Trigger rollback
manager.RecordRollback(taskID, constitution.RollbackInfo{
    Triggered:     true,
    Reason:        "Validation failed: JWT tests failing",
    RestoredFiles: []string{"backend/pkg/middleware/auth.go"},
})
```

### Listing and Filtering Tasks

```go
// List all tasks
allTasks, err := manager.ListTasks(constitution.TaskFilter{})

// List only completed tasks
completedStatus := constitution.TaskStatusCompleted
completedTasks, err := manager.ListTasks(constitution.TaskFilter{
    Status: &completedStatus,
})

// List with pagination
paginatedTasks, err := manager.ListTasks(constitution.TaskFilter{
    Limit:  10,
    Offset: 0,
})

// List tasks within date range
startDate := time.Now().AddDate(0, 0, -7) // Last 7 days
tasks, err := manager.ListTasks(constitution.TaskFilter{
    StartDate: &startDate,
})
```

### Using with Configuration

```go
// Load configuration
loader, err := constitution.NewConfigLoader(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create manager from config
manager, err := constitution.NewTaskTraceManagerFromConfig(loader.Get())
if err != nil {
    log.Fatal(err)
}
```

## Directory Structure

See the `.ai/` directory for:
- `constitution.md`: The AI programming rules and constraints
- `config.yaml`: Tool and validation configuration
- `traces/`: Task execution records (JSON files)
- `templates/`: Code generation templates

## Task Trace Format

Each task is stored as a JSON file in `.ai/traces/` with the following structure:

```json
{
  "task_id": "uuid",
  "timestamp_start": "2026-03-12T10:30:00Z",
  "timestamp_end": "2026-03-12T10:45:00Z",
  "status": "completed",
  "task_description": "Implement user CRUD operations",
  "developer_request": "Create user management API",
  "decisions": [
    {
      "timestamp": "2026-03-12T10:31:00Z",
      "decision_type": "architecture",
      "description": "Use Repository pattern",
      "rationale": "Follows existing patterns",
      "constitution_reference": "Section 4.1"
    }
  ],
  "code_changes": [
    {
      "file_path": "backend/app/admin/service/internal/service/user.go",
      "operation": "create",
      "lines_added": 150,
      "lines_removed": 0,
      "summary": "Created UserService implementation"
    }
  ],
  "validations": [
    {
      "validator": "gofmt",
      "timestamp": "2026-03-12T10:40:00Z",
      "status": "passed",
      "output": "All files formatted correctly"
    }
  ],
  "references": [
    {
      "type": "example",
      "source": "backend/app/admin/service/internal/service/role.go",
      "description": "Reference implementation"
    }
  ]
}
```


### ViolationDetector
Detects constitution violations in code including architecture, security, dependency, and schema violations.

**Implementation**: `violationDetector` (see `violation_detector.go`)

**Key Methods**:
- `DetectArchitectureViolations`: Detects cross-layer calls, directory structure violations, and module dependency violations
- `DetectSecurityViolations`: Detects hardcoded secrets, authentication bypass, sensitive data in logs, and SQL injection risks
- `DetectDependencyViolations`: Detects unapproved dependencies in go.mod and package.json
- `DetectSchemaViolations`: Detects migration deletions and Protobuf/Ent schema breaking changes
- `DetectAllViolations`: Runs all violation checks and generates a comprehensive report

**Violation Types**:
- `ViolationTypeArchitecture`: Architecture constraint violations
- `ViolationTypeSecurity`: Security-related violations
- `ViolationTypeDependency`: Dependency management violations
- `ViolationTypeSchema`: Schema modification violations
- `ViolationTypeAPI`: API breaking changes
- `ViolationTypeConfiguration`: Configuration violations

**Severity Levels**:
- `SeverityCritical`: Absolutely forbidden, triggers immediate rollback
- `SeverityHigh`: Requires explicit approval
- `SeverityMedium`: Requires confirmation
- `SeverityLow`: Warning only

**Usage Example**:
```go
// Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create violation detector
detector := constitution.NewViolationDetector(cfg, ".")

// Files to check
files := []string{
    "backend/pkg/utils/helper.go",
    "backend/app/admin/service/internal/service/user.go",
}

// Detect all violations
report, err := detector.DetectAllViolations(context.Background(), files)
if err != nil {
    log.Fatal(err)
}

// Print report summary
fmt.Printf("Total violations: %d\n", len(report.Violations))
fmt.Printf("Critical: %d\n", report.CriticalCount)
fmt.Printf("High: %d\n", report.HighCount)
fmt.Printf("Medium: %d\n", report.MediumCount)
fmt.Printf("Low: %d\n", report.LowCount)
fmt.Printf("Should rollback: %v\n", report.ShouldRollback)

// Print violations
for _, v := range report.Violations {
    fmt.Printf("[%s] %s at %s:%d\n", v.Severity, v.Description, v.FilePath, v.LineNumber)
    fmt.Printf("  Rule: %s\n", v.Rule)
    fmt.Printf("  Reference: %s\n", v.ConstitutionReference)
    fmt.Printf("  Suggestion: %s\n", v.Suggestion)
}

// Detect specific violation types
archViolations, err := detector.DetectArchitectureViolations(context.Background(), files)
if err != nil {
    log.Fatal(err)
}

secViolations, err := detector.DetectSecurityViolations(context.Background(), files)
if err != nil {
    log.Fatal(err)
}
```

**Architecture Violations Detected**:
- Cross-layer imports (pkg/ importing app/, api/ importing app/pkg/)
- Files outside standard architecture layers
- Direct module-to-module dependencies in app/ layer

**Security Violations Detected**:
- Hardcoded passwords, API keys, secrets, tokens
- Authentication bypass attempts
- Sensitive data in log statements
- SQL injection risks (string concatenation in queries)

**Example Violations**:
```go
// ❌ Architecture violation: pkg/ importing app/
package utils
import "backend/app/admin/service/internal/data" // VIOLATION

// ❌ Security violation: hardcoded password
password := "admin123" // VIOLATION

// ❌ Security violation: sensitive data in logs
log.Printf("User password: %s", password) // VIOLATION

// ❌ Security violation: SQL injection risk
query := "SELECT * FROM users WHERE id = " + userID // VIOLATION

// ✅ Correct: use parameterized query or ORM
user, err := db.User.Query().Where(user.ID(userID)).Only(ctx)
```

### RuleEngine
Evaluates constitution rules against code and generates violation reports.

**Implementation**: `RuleEngine` (see `rule_engine.go`)

**Key Methods**:
- `AddRule`: Adds a custom rule to the engine
- `GetRules`: Returns all enabled rules
- `GetRulesByType`: Returns rules of a specific violation type
- `GetRulesBySeverity`: Returns rules of a specific severity level
- `EvaluateFile`: Evaluates all rules against a file's content
- `GenerateViolationReport`: Generates a formatted violation report
- `EvaluateSeverity`: Evaluates the overall severity of violations
- `ShouldRollback`: Determines if violations warrant a rollback
- `GenerateFixSuggestions`: Generates actionable fix suggestions

**Built-in Rules**:

Architecture Rules:
- `arch-001`: No pkg to app dependency
- `arch-002`: No api to app/pkg dependency
- `arch-003`: No direct module dependencies

Security Rules:
- `sec-001`: No hardcoded passwords
- `sec-002`: No hardcoded API keys
- `sec-003`: No hardcoded secrets
- `sec-004`: No authentication bypass
- `sec-005`: No sensitive data in logs
- `sec-006`: No SQL injection

Schema Rules:
- `schema-001`: No migration deletion
- `schema-002`: No Protobuf field deletion

Dependency Rules:
- `dep-001`: No unapproved dependencies

Configuration Rules:
- `config-001`: No production config modification

**Usage Example**:
```go
// Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create rule engine
engine := constitution.NewRuleEngine(cfg)

// Get all rules
allRules := engine.GetRules()
fmt.Printf("Total rules: %d\n", len(allRules))

// Get security rules
securityRules := engine.GetRulesByType(constitution.ViolationTypeSecurity)
fmt.Printf("Security rules: %d\n", len(securityRules))

// Get critical rules
criticalRules := engine.GetRulesBySeverity(constitution.SeverityCritical)
fmt.Printf("Critical rules: %d\n", len(criticalRules))

// Evaluate a file
filePath := "backend/app/admin/service/internal/service/user.go"
content, _ := os.ReadFile(filePath)

violations, err := engine.EvaluateFile(filePath, string(content))
if err != nil {
    log.Fatal(err)
}

// Generate formatted report
report := engine.GenerateViolationReport(violations)
fmt.Println(report)

// Get fix suggestions
suggestions := engine.GenerateFixSuggestions(violations)
fmt.Println("Fix Suggestions:")
for _, s := range suggestions {
    fmt.Printf("- %s\n", s)
}

// Check if rollback is needed
if engine.ShouldRollback(violations) {
    fmt.Println("⚠️  Rollback recommended!")
}

// Evaluate overall severity
severity := engine.EvaluateSeverity(violations)
fmt.Printf("Overall severity: %s\n", severity)
```

**Adding Custom Rules**:
```go
// Create a custom rule
customRule := constitution.Rule{
    ID:                    "custom-001",
    Name:                  "No TODO comments in production",
    Description:           "TODO comments should be resolved before production",
    Type:                  constitution.ViolationTypeConfiguration,
    Severity:              constitution.SeverityMedium,
    Pattern:               `(?i)//\s*TODO`,
    FilePattern:           "*.go",
    ConstitutionReference: "Custom Rule",
    Suggestion:            "Resolve TODO or create a ticket",
    Enabled:               true,
}

// Add to engine
engine.AddRule(customRule)

// Now the rule will be evaluated on all files
```

**Violation Report Format**:
```
❌ Constitution Violations Detected
=====================================

🔴 CRITICAL (2)
─────────────────
  [sec-001] Hardcoded password detected
  File: backend/app/admin/service/internal/service/auth.go:45
  Type: security
  Reference: Section 5.4: Security Violations
  💡 Suggestion: Use environment variables for sensitive data

  [sec-006] Potential SQL injection risk: string concatenation in query
  File: backend/app/admin/service/internal/data/user.go:78
  Type: security
  Reference: Section 12.2: Security and Performance
  💡 Suggestion: Use parameterized queries or Ent ORM

🟠 HIGH (1)
─────────────────
  [arch-001] pkg/ layer cannot depend on app/ layer
  File: backend/pkg/utils/helper.go:12
  Type: architecture
  Reference: Section 3.1: Three-Layer Architecture
  💡 Suggestion: Move shared code to pkg/ or use dependency inversion

Summary
─────────────────
Total: 3 violations
Critical: 2, High: 1, Medium: 0, Low: 0

⚠️  Rollback recommended due to critical/high severity violations.
```

### Complete Workflow with Violation Detection

```go
// 1. Load configuration
cfg, err := constitution.LoadConfig(".ai/config.yaml")
if err != nil {
    log.Fatal(err)
}

// 2. Create all components
traceManager, _ := constitution.NewTaskTraceManagerFromConfig(cfg)
validator, _ := constitution.NewCodeValidatorFromConfig(cfg)
rollbackManager, _ := constitution.NewRollbackManagerFromConfig(cfg)
violationDetector, _ := constitution.NewViolationDetectorFromConfig(cfg)
ruleEngine := constitution.NewRuleEngineFromConfig(cfg)

// 3. Create task
taskID, _ := traceManager.CreateTask(
    "Implement user authentication",
    "Add JWT authentication to user service",
)

// 4. Record decision
traceManager.RecordDecision(taskID, constitution.Decision{
    DecisionType:          constitution.DecisionTypeImplementation,
    Description:           "Use JWT for authentication",
    Rationale:             "JWT is stateless and scalable",
    ConstitutionReference: "Section 12.3: Security Rules",
})

// 5. Create backup before changes
files := []string{
    "backend/app/admin/service/internal/service/auth.go",
    "backend/pkg/middleware/jwt.go",
}
backupID, _ := rollbackManager.CreateBackup(taskID, files)

// 6. Record code changes
for _, file := range files {
    traceManager.RecordCodeChange(taskID, constitution.CodeChange{
        FilePath:     file,
        Operation:    constitution.OperationCreate,
        LinesAdded:   150,
        Summary:      "Implemented JWT authentication",
    })
}

// 7. Detect violations
report, err := violationDetector.DetectAllViolations(context.Background(), files)
if err != nil {
    log.Fatal(err)
}

// 8. Check if rollback is needed
if report.ShouldRollback {
    fmt.Println("⚠️  Critical violations detected! Triggering rollback...")
    
    // Rollback changes
    rollbackManager.Rollback(taskID, "Critical constitution violations detected")
    
    // Record rollback
    traceManager.RecordRollback(taskID, constitution.RollbackInfo{
        Triggered:     true,
        Reason:        "Critical constitution violations detected",
        RestoredFiles: files,
    })
    
    // Fail task
    traceManager.FailTask(taskID, "Rolled back due to constitution violations")
    
    // Print violation report
    fmt.Println(ruleEngine.GenerateViolationReport(report.Violations))
    
    return
}

// 9. Validate code
for _, file := range files {
    result, _ := validator.ValidateGoCode(file)
    traceManager.RecordValidation(taskID, constitution.Validation{
        Validator: result.Validator,
        Status:    constitution.ValidationStatusPassed,
        Output:    result.Output,
    })
}

// 10. Complete task
traceManager.CompleteTask(taskID)
fmt.Println("✅ Task completed successfully")
```

## Testing

Run tests for the constitution package:

```bash
# Run all tests
go test ./pkg/constitution/...

# Run with coverage
go test -cover ./pkg/constitution/...

# Run specific test
go test -run TestTaskTraceManager ./pkg/constitution/

# Run with verbose output
go test -v ./pkg/constitution/...
```

## License

This package is part of the GO + Vue backend management framework project.
