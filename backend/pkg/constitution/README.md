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

### DocumentationSyncer
Synchronizes documentation with code changes.

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
