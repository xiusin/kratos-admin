package constitution

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// codeValidator implements the CodeValidator interface
type codeValidator struct {
	config *Config
}

// NewCodeValidator creates a new CodeValidator instance
func NewCodeValidator(config *Config) (CodeValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	return &codeValidator{
		config: config,
	}, nil
}

// ValidateGoCode validates Go code using gofmt and golangci-lint
func (v *codeValidator) ValidateGoCode(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "go",
		Timestamp: time.Now(),
		Passed:    true,
	}

	// Run gofmt
	gofmtResult, err := v.runGofmt(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run gofmt: %w", err)
	}

	if !gofmtResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, gofmtResult.Errors...)
		result.Output += "gofmt:\n" + gofmtResult.Output + "\n\n"
	}

	// Run golangci-lint
	lintResult, err := v.runGolangciLint(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run golangci-lint: %w", err)
	}

	if !lintResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, lintResult.Errors...)
		result.Warnings = append(result.Warnings, lintResult.Warnings...)
		result.Output += "golangci-lint:\n" + lintResult.Output + "\n\n"
	}

	return result, nil
}

// ValidateVueCode validates Vue code using eslint and vue-tsc
func (v *codeValidator) ValidateVueCode(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "vue",
		Timestamp: time.Now(),
		Passed:    true,
	}

	// Run eslint
	eslintResult, err := v.runEslint(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run eslint: %w", err)
	}

	if !eslintResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, eslintResult.Errors...)
		result.Warnings = append(result.Warnings, eslintResult.Warnings...)
		result.Output += "eslint:\n" + eslintResult.Output + "\n\n"
	}

	// Run vue-tsc for type checking
	vueTscResult, err := v.runVueTsc(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run vue-tsc: %w", err)
	}

	if !vueTscResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, vueTscResult.Errors...)
		result.Output += "vue-tsc:\n" + vueTscResult.Output + "\n\n"
	}

	return result, nil
}

// ValidateProtobuf validates Protobuf schema using protoc
func (v *codeValidator) ValidateProtobuf(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "protobuf",
		Timestamp: time.Now(),
		Passed:    true,
	}

	protocResult, err := v.runProtoc(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run protoc: %w", err)
	}

	if !protocResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, protocResult.Errors...)
		result.Output = protocResult.Output
	}

	return result, nil
}

// ValidateEntSchema validates Ent schema by running ent generate
func (v *codeValidator) ValidateEntSchema(schemaDir string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "ent",
		Timestamp: time.Now(),
		Passed:    true,
	}

	entResult, err := v.runEntGenerate(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to run ent generate: %w", err)
	}

	if !entResult.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, entResult.Errors...)
		result.Output = entResult.Output
	}

	return result, nil
}

// ValidateImports validates that imports exist
func (v *codeValidator) ValidateImports(filePath string, language string) (*ValidationResult, error) {
	switch language {
	case "go":
		return v.validateGoImports(filePath)
	case "vue", "typescript", "javascript":
		return v.validateJSImports(filePath)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// RunTests runs tests matching the pattern
func (v *codeValidator) RunTests(testPattern string) (*ValidationResult, error) {
	// Determine if it's Go or JS tests based on pattern
	if strings.Contains(testPattern, ".go") || strings.Contains(testPattern, "/backend/") {
		return v.runGoTests(testPattern)
	}

	return v.runJSTests(testPattern)
}

// runGofmt runs gofmt on a Go file
func (v *codeValidator) runGofmt(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "gofmt",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Go.Formatter.Command
	args := append(v.config.Tools.Go.Formatter.Args, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Go.Formatter.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			File:     filePath,
			Message:  fmt.Sprintf("gofmt failed: %v", err),
			Severity: "error",
		})
	}

	// If gofmt produces output, it means the file is not formatted
	if len(output) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			File:     filePath,
			Message:  "file is not formatted according to gofmt",
			Severity: "error",
		})
	}

	return result, nil
}

// runGolangciLint runs golangci-lint on a Go file
func (v *codeValidator) runGolangciLint(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "golangci-lint",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Go.Linter.Command
	args := append(v.config.Tools.Go.Linter.Args, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Go.Linter.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		// Parse golangci-lint output
		errors := v.parseGolangciLintOutput(string(output))
		result.Errors = errors
	}

	return result, nil
}

// parseGolangciLintOutput parses golangci-lint output to extract errors
func (v *codeValidator) parseGolangciLintOutput(output string) []ValidationError {
	var errors []ValidationError

	// golangci-lint format: file:line:column: message (linter)
	re := regexp.MustCompile(`([^:]+):(\d+):(\d+):\s+(.+)`)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])

			errors = append(errors, ValidationError{
				File:     matches[1],
				Line:     lineNum,
				Column:   colNum,
				Message:  matches[4],
				Severity: "error",
			})
		}
	}

	return errors
}

// runEslint runs eslint on a Vue/JS/TS file
func (v *codeValidator) runEslint(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "eslint",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Vue.Linter.Command
	args := append(v.config.Tools.Vue.Linter.Args, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Vue.Linter.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		// Parse eslint output
		errors, warnings := v.parseEslintOutput(string(output))
		result.Errors = errors
		result.Warnings = warnings
	}

	return result, nil
}

// parseEslintOutput parses eslint output to extract errors and warnings
func (v *codeValidator) parseEslintOutput(output string) ([]ValidationError, []ValidationWarning) {
	var errors []ValidationError
	var warnings []ValidationWarning

	// eslint format: file:line:column: error/warning message
	re := regexp.MustCompile(`([^:]+):(\d+):(\d+):\s+(error|warning)\s+(.+)`)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 6 {
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])
			severity := matches[4]
			message := matches[5]

			if severity == "error" {
				errors = append(errors, ValidationError{
					File:     matches[1],
					Line:     lineNum,
					Column:   colNum,
					Message:  message,
					Severity: "error",
				})
			} else {
				warnings = append(warnings, ValidationWarning{
					File:     matches[1],
					Line:     lineNum,
					Column:   colNum,
					Message:  message,
					Severity: "warning",
				})
			}
		}
	}

	return errors, warnings
}

// runVueTsc runs vue-tsc for type checking
func (v *codeValidator) runVueTsc(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "vue-tsc",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Vue.TypeChecker.Command
	args := append(v.config.Tools.Vue.TypeChecker.Args, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Vue.TypeChecker.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		// Parse TypeScript errors
		errors := v.parseTypeScriptOutput(string(output))
		result.Errors = errors
	}

	return result, nil
}

// parseTypeScriptOutput parses TypeScript compiler output
func (v *codeValidator) parseTypeScriptOutput(output string) []ValidationError {
	var errors []ValidationError

	// TypeScript format: file(line,column): error TS####: message
	re := regexp.MustCompile(`([^(]+)\((\d+),(\d+)\):\s+error\s+TS\d+:\s+(.+)`)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])

			errors = append(errors, ValidationError{
				File:     matches[1],
				Line:     lineNum,
				Column:   colNum,
				Message:  matches[4],
				Severity: "error",
			})
		}
	}

	return errors
}

// runProtoc runs protoc to validate Protobuf files
func (v *codeValidator) runProtoc(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "protoc",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Protobuf.Compiler.Command
	args := append(v.config.Tools.Protobuf.Compiler.Args, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Protobuf.Compiler.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			File:     filePath,
			Message:  string(output),
			Severity: "error",
		})
	}

	return result, nil
}

// runEntGenerate runs ent generate to validate Ent schemas
func (v *codeValidator) runEntGenerate(schemaDir string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "ent",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Ent.Generator.Command
	args := append(v.config.Tools.Ent.Generator.Args, schemaDir)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Ent.Generator.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	output, err := execCmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			File:     schemaDir,
			Message:  string(output),
			Severity: "error",
		})
	}

	return result, nil
}

// validateGoImports validates Go imports
func (v *codeValidator) validateGoImports(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "go-imports",
		Timestamp: time.Now(),
		Passed:    true,
	}

	// Use go build to check if imports are valid
	dir := filepath.Dir(filePath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", "/dev/null", filePath)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		// Parse import errors
		if strings.Contains(string(output), "cannot find package") ||
			strings.Contains(string(output), "undefined:") {
			result.Errors = append(result.Errors, ValidationError{
				File:     filePath,
				Message:  string(output),
				Severity: "error",
			})
		}
	}

	return result, nil
}

// validateJSImports validates JavaScript/TypeScript imports
func (v *codeValidator) validateJSImports(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "js-imports",
		Timestamp: time.Now(),
		Passed:    true,
	}

	// Use TypeScript compiler to check imports
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tsc", "--noEmit", filePath)
	output, err := cmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Passed = false
		if strings.Contains(string(output), "Cannot find module") {
			result.Errors = append(result.Errors, ValidationError{
				File:     filePath,
				Message:  string(output),
				Severity: "error",
			})
		}
	}

	return result, nil
}

// runGoTests runs Go tests
func (v *codeValidator) runGoTests(testPattern string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "go-test",
		Timestamp: time.Now(),
		Passed:    true,
	}

	cmd := v.config.Tools.Go.TestRunner.Command
	args := append(v.config.Tools.Go.TestRunner.Args, testPattern)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.Tools.Go.TestRunner.Timeout)*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()

	result.Output = stdout.String() + stderr.String()

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			Message:  fmt.Sprintf("tests failed: %v", err),
			Severity: "error",
		})
	}

	return result, nil
}

// runJSTests runs JavaScript/TypeScript tests
func (v *codeValidator) runJSTests(testPattern string) (*ValidationResult, error) {
	result := &ValidationResult{
		Validator: "js-test",
		Timestamp: time.Now(),
		Passed:    true,
	}

	// Assume using npm test or similar
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npm", "test", "--", testPattern)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Output = stdout.String() + stderr.String()

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			Message:  fmt.Sprintf("tests failed: %v", err),
			Severity: "error",
		})
	}

	return result, nil
}
