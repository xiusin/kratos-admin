// Package constitution implements the AI Programming Constitution system.
//
// The Constitution system provides a framework for constraining and auditing
// AI-assisted code generation. It ensures that AI-generated code follows
// project architecture, coding standards, and design principles while
// preventing hallucinations and maintaining complete audit trails.
//
// Core Components:
//
// Task Tracing: Records all AI operations, decisions, and code changes
// in structured JSON format for audit and review.
//
// Code Validation: Integrates with standard tools (gofmt, golangci-lint,
// eslint, vue-tsc, protoc) to validate generated code. The CodeValidator
// interface provides methods for validating Go, Vue, Protobuf, and Ent
// code, as well as checking imports and running tests.
//
// Anti-Hallucination: Verifies that referenced APIs, functions, modules,
// and configuration keys actually exist before generating code.
//
// Rollback Management: Automatically rolls back code changes that violate
// constitution rules or fail validation.
//
// Documentation Sync: Keeps API, component, and feature documentation
// synchronized with code changes.
//
// Usage Example:
//
//	// Load configuration
//	cfg, err := constitution.LoadConfig(".ai/config.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create validator
//	validator, err := constitution.NewCodeValidator(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Validate Go code
//	result, err := validator.ValidateGoCode("path/to/file.go")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if !result.Passed {
//		for _, err := range result.Errors {
//			fmt.Printf("%s:%d:%d: %s\n", err.File, err.Line, err.Column, err.Message)
//		}
//	}
//
// The constitution rules are defined in .ai/constitution.md and configuration
// is stored in .ai/config.yaml. Task traces are saved to .ai/traces/.
package constitution
