package constitution

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Severity represents the severity level of a violation
type Severity string

const (
	SeverityCritical Severity = "critical" // Absolutely forbidden
	SeverityHigh     Severity = "high"     // Requires explicit approval
	SeverityMedium   Severity = "medium"   // Requires confirmation
	SeverityLow      Severity = "low"      // Warning only
)

// ViolationType represents the type of constitution violation
type ViolationType string

const (
	ViolationTypeArchitecture  ViolationType = "architecture"
	ViolationTypeSecurity      ViolationType = "security"
	ViolationTypeDependency    ViolationType = "dependency"
	ViolationTypeSchema        ViolationType = "schema"
	ViolationTypeAPI           ViolationType = "api"
	ViolationTypeConfiguration ViolationType = "configuration"
)

// Violation represents a constitution violation
type Violation struct {
	Type                  ViolationType `json:"type"`
	Severity              Severity      `json:"severity"`
	FilePath              string        `json:"file_path"`
	LineNumber            int           `json:"line_number"`
	Description           string        `json:"description"`
	Rule                  string        `json:"rule"`
	ConstitutionReference string        `json:"constitution_reference"`
	Suggestion            string        `json:"suggestion"`
}

// ViolationReport represents a report of all violations found
type ViolationReport struct {
	Violations     []Violation `json:"violations"`
	CriticalCount  int         `json:"critical_count"`
	HighCount      int         `json:"high_count"`
	MediumCount    int         `json:"medium_count"`
	LowCount       int         `json:"low_count"`
	ShouldRollback bool        `json:"should_rollback"`
}

// ViolationDetector detects constitution violations
type ViolationDetector interface {
	// DetectArchitectureViolations detects architecture-related violations
	DetectArchitectureViolations(ctx context.Context, files []string) ([]Violation, error)

	// DetectSecurityViolations detects security-related violations
	DetectSecurityViolations(ctx context.Context, files []string) ([]Violation, error)

	// DetectDependencyViolations detects dependency management violations
	DetectDependencyViolations(ctx context.Context, files []string) ([]Violation, error)

	// DetectSchemaViolations detects schema modification violations
	DetectSchemaViolations(ctx context.Context, files []string) ([]Violation, error)

	// DetectAllViolations detects all types of violations
	DetectAllViolations(ctx context.Context, files []string) (*ViolationReport, error)
}

// violationDetector implements ViolationDetector
type violationDetector struct {
	config      *Config
	projectRoot string
	ruleEngine  *RuleEngine
}

// NewViolationDetector creates a new violation detector
func NewViolationDetector(config *Config, projectRoot string) ViolationDetector {
	return &violationDetector{
		config:      config,
		projectRoot: projectRoot,
		ruleEngine:  NewRuleEngine(config),
	}
}

// DetectArchitectureViolations detects architecture-related violations
func (d *violationDetector) DetectArchitectureViolations(ctx context.Context, files []string) ([]Violation, error) {
	var violations []Violation

	for _, file := range files {
		// Check for cross-layer violations
		if v := d.detectCrossLayerCalls(file); v != nil {
			violations = append(violations, *v)
		}

		// Check for directory structure violations
		if v := d.detectDirectoryStructureViolations(file); v != nil {
			violations = append(violations, *v)
		}

		// Check for module dependency violations
		if v := d.detectModuleDependencyViolations(file); v != nil {
			violations = append(violations, *v)
		}
	}

	return violations, nil
}

// DetectSecurityViolations detects security-related violations
func (d *violationDetector) DetectSecurityViolations(ctx context.Context, files []string) ([]Violation, error) {
	var violations []Violation

	for _, file := range files {
		// Check for hardcoded secrets
		if vs := d.detectHardcodedSecrets(file); len(vs) > 0 {
			violations = append(violations, vs...)
		}

		// Check for authentication bypass
		if v := d.detectAuthenticationBypass(file); v != nil {
			violations = append(violations, *v)
		}

		// Check for sensitive data in logs
		if vs := d.detectSensitiveDataInLogs(file); len(vs) > 0 {
			violations = append(violations, vs...)
		}

		// Check for SQL injection risks
		if vs := d.detectSQLInjectionRisks(file); len(vs) > 0 {
			violations = append(violations, vs...)
		}
	}

	return violations, nil
}

// DetectDependencyViolations detects dependency management violations
func (d *violationDetector) DetectDependencyViolations(ctx context.Context, files []string) ([]Violation, error) {
	var violations []Violation

	for _, file := range files {
		// Check go.mod modifications
		if strings.HasSuffix(file, "go.mod") {
			if vs := d.detectGoModViolations(file); len(vs) > 0 {
				violations = append(violations, vs...)
			}
		}

		// Check package.json modifications
		if strings.HasSuffix(file, "package.json") {
			if vs := d.detectPackageJSONViolations(file); len(vs) > 0 {
				violations = append(violations, vs...)
			}
		}
	}

	return violations, nil
}

// DetectSchemaViolations detects schema modification violations
func (d *violationDetector) DetectSchemaViolations(ctx context.Context, files []string) ([]Violation, error) {
	var violations []Violation

	for _, file := range files {
		// Check for migration file deletions
		if d.isMigrationFile(file) {
			if v := d.detectMigrationDeletion(file); v != nil {
				violations = append(violations, *v)
			}
		}

		// Check for Protobuf breaking changes
		if strings.HasSuffix(file, ".proto") {
			if vs := d.detectProtobufBreakingChanges(file); len(vs) > 0 {
				violations = append(violations, vs...)
			}
		}

		// Check for Ent schema breaking changes
		if d.isEntSchemaFile(file) {
			if vs := d.detectEntSchemaBreakingChanges(file); len(vs) > 0 {
				violations = append(violations, vs...)
			}
		}
	}

	return violations, nil
}

// DetectAllViolations detects all types of violations
func (d *violationDetector) DetectAllViolations(ctx context.Context, files []string) (*ViolationReport, error) {
	report := &ViolationReport{
		Violations: make([]Violation, 0),
	}

	// Detect architecture violations
	archViolations, err := d.DetectArchitectureViolations(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to detect architecture violations: %w", err)
	}
	report.Violations = append(report.Violations, archViolations...)

	// Detect security violations
	secViolations, err := d.DetectSecurityViolations(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to detect security violations: %w", err)
	}
	report.Violations = append(report.Violations, secViolations...)

	// Detect dependency violations
	depViolations, err := d.DetectDependencyViolations(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to detect dependency violations: %w", err)
	}
	report.Violations = append(report.Violations, depViolations...)

	// Detect schema violations
	schemaViolations, err := d.DetectSchemaViolations(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to detect schema violations: %w", err)
	}
	report.Violations = append(report.Violations, schemaViolations...)

	// Count violations by severity
	for _, v := range report.Violations {
		switch v.Severity {
		case SeverityCritical:
			report.CriticalCount++
		case SeverityHigh:
			report.HighCount++
		case SeverityMedium:
			report.MediumCount++
		case SeverityLow:
			report.LowCount++
		}
	}

	// Determine if rollback is needed
	report.ShouldRollback = report.CriticalCount > 0 || report.HighCount > 0

	return report, nil
}

// detectCrossLayerCalls detects cross-layer calls that violate architecture
func (d *violationDetector) detectCrossLayerCalls(filePath string) *Violation {
	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	// Check imports for cross-layer violations
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if pkg/ imports app/
		if strings.Contains(filePath, "/pkg/") && strings.Contains(importPath, "/app/") {
			return &Violation{
				Type:                  ViolationTypeArchitecture,
				Severity:              SeverityCritical,
				FilePath:              filePath,
				LineNumber:            fset.Position(imp.Pos()).Line,
				Description:           fmt.Sprintf("pkg/ layer cannot import app/ layer: %s", importPath),
				Rule:                  "architecture.three_layer.dependency_direction",
				ConstitutionReference: "Section 3.1: Three-Layer Architecture",
				Suggestion:            "Move shared code to pkg/ or use interfaces for dependency inversion",
			}
		}

		// Check if api/ imports app/ or pkg/
		if strings.Contains(filePath, "/api/") && (strings.Contains(importPath, "/app/") || strings.Contains(importPath, "/pkg/")) {
			return &Violation{
				Type:                  ViolationTypeArchitecture,
				Severity:              SeverityCritical,
				FilePath:              filePath,
				LineNumber:            fset.Position(imp.Pos()).Line,
				Description:           fmt.Sprintf("api/ layer cannot import app/ or pkg/ layers: %s", importPath),
				Rule:                  "architecture.three_layer.api_isolation",
				ConstitutionReference: "Section 3.1: Three-Layer Architecture",
				Suggestion:            "API definitions should be pure Protobuf without Go dependencies",
			}
		}
	}

	return nil
}

// detectDirectoryStructureViolations detects violations of directory structure
func (d *violationDetector) detectDirectoryStructureViolations(filePath string) *Violation {
	// Check if file is in correct layer
	relPath, err := filepath.Rel(d.projectRoot, filePath)
	if err != nil {
		return nil
	}

	// Check for files outside of api/, app/, pkg/
	if !strings.HasPrefix(relPath, "api/") &&
		!strings.HasPrefix(relPath, "app/") &&
		!strings.HasPrefix(relPath, "pkg/") &&
		!strings.HasPrefix(relPath, "frontend/") {
		// Allow some root-level files
		allowedRootFiles := []string{"go.mod", "go.sum", "Makefile", "README.md", "Dockerfile"}
		baseName := filepath.Base(relPath)
		for _, allowed := range allowedRootFiles {
			if baseName == allowed {
				return nil
			}
		}

		return &Violation{
			Type:                  ViolationTypeArchitecture,
			Severity:              SeverityHigh,
			FilePath:              filePath,
			Description:           "File is outside of standard architecture layers (api/, app/, pkg/)",
			Rule:                  "architecture.directory_structure",
			ConstitutionReference: "Section 3.1: Three-Layer Architecture",
			Suggestion:            "Move file to appropriate layer directory",
		}
	}

	return nil
}

// detectModuleDependencyViolations detects module dependency violations
func (d *violationDetector) detectModuleDependencyViolations(filePath string) *Violation {
	// Check for direct module-to-module dependencies in app/ layer
	if !strings.Contains(filePath, "/app/") {
		return nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	// Extract current module name
	currentModule := d.extractModuleName(filePath)
	if currentModule == "" {
		return nil
	}

	// Check imports for direct module dependencies
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if importing another app/ module directly
		if strings.Contains(importPath, "/app/") && !strings.Contains(importPath, currentModule) {
			return &Violation{
				Type:                  ViolationTypeArchitecture,
				Severity:              SeverityHigh,
				FilePath:              filePath,
				LineNumber:            fset.Position(imp.Pos()).Line,
				Description:           fmt.Sprintf("Direct module-to-module dependency detected: %s", importPath),
				Rule:                  "architecture.module_isolation",
				ConstitutionReference: "Section 3.4: Event Bus Usage",
				Suggestion:            "Use event bus or gRPC for cross-module communication",
			}
		}
	}

	return nil
}

// detectHardcodedSecrets detects hardcoded secrets in code
func (d *violationDetector) detectHardcodedSecrets(filePath string) []Violation {
	var violations []Violation

	content, err := os.ReadFile(filePath)
	if err != nil {
		return violations
	}

	lines := strings.Split(string(content), "\n")

	// Patterns for detecting secrets
	secretPatterns := []struct {
		pattern *regexp.Regexp
		desc    string
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']([^"']{8,})["']`), "hardcoded password"},
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']([^"']{16,})["']`), "hardcoded API key"},
		{regexp.MustCompile(`(?i)(secret|token)\s*[:=]\s*["']([^"']{16,})["']`), "hardcoded secret/token"},
		{regexp.MustCompile(`(?i)(access[_-]?key|accesskey)\s*[:=]\s*["']([^"']{16,})["']`), "hardcoded access key"},
	}

	for lineNum, line := range lines {
		for _, sp := range secretPatterns {
			if sp.pattern.MatchString(line) {
				violations = append(violations, Violation{
					Type:                  ViolationTypeSecurity,
					Severity:              SeverityCritical,
					FilePath:              filePath,
					LineNumber:            lineNum + 1,
					Description:           fmt.Sprintf("Detected %s in code", sp.desc),
					Rule:                  "security.no_hardcoded_secrets",
					ConstitutionReference: "Section 5.4: Security Violations",
					Suggestion:            "Use environment variables or secure configuration management",
				})
			}
		}
	}

	return violations
}

// detectAuthenticationBypass detects authentication bypass attempts
func (d *violationDetector) detectAuthenticationBypass(filePath string) *Violation {
	if !strings.HasSuffix(filePath, ".go") {
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	// Patterns that might indicate authentication bypass
	bypassPatterns := []string{
		"// skip auth",
		"// bypass auth",
		"// TODO: remove auth check",
		"if false { // auth disabled",
	}

	lines := strings.Split(string(content), "\n")
	for lineNum, line := range lines {
		lowerLine := strings.ToLower(line)
		for _, pattern := range bypassPatterns {
			if strings.Contains(lowerLine, strings.ToLower(pattern)) {
				return &Violation{
					Type:                  ViolationTypeSecurity,
					Severity:              SeverityCritical,
					FilePath:              filePath,
					LineNumber:            lineNum + 1,
					Description:           "Potential authentication bypass detected",
					Rule:                  "security.no_auth_bypass",
					ConstitutionReference: "Section 5.4: Security Violations",
					Suggestion:            "Remove authentication bypass code",
				}
			}
		}
	}

	return nil
}

// detectSensitiveDataInLogs detects sensitive data being logged
func (d *violationDetector) detectSensitiveDataInLogs(filePath string) []Violation {
	var violations []Violation

	if !strings.HasSuffix(filePath, ".go") {
		return violations
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return violations
	}

	lines := strings.Split(string(content), "\n")

	// Patterns for sensitive data in logs
	logPattern := regexp.MustCompile(`(?i)(log\.|logger\.|fmt\.Print).*\b(password|token|secret|key|credential)\b`)

	for lineNum, line := range lines {
		if logPattern.MatchString(line) {
			violations = append(violations, Violation{
				Type:                  ViolationTypeSecurity,
				Severity:              SeverityHigh,
				FilePath:              filePath,
				LineNumber:            lineNum + 1,
				Description:           "Potential sensitive data exposure in logs",
				Rule:                  "security.no_sensitive_logs",
				ConstitutionReference: "Section 12.5: Security and Performance",
				Suggestion:            "Avoid logging sensitive information like passwords, tokens, or keys",
			})
		}
	}

	return violations
}

// detectSQLInjectionRisks detects potential SQL injection risks
func (d *violationDetector) detectSQLInjectionRisks(filePath string) []Violation {
	var violations []Violation

	if !strings.HasSuffix(filePath, ".go") {
		return violations
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return violations
	}

	lines := strings.Split(string(content), "\n")

	// Pattern for string concatenation in SQL queries
	sqlPattern := regexp.MustCompile(`(?i)(Exec|Query|QueryRow)\s*\(\s*["'].*\+.*["']`)

	for lineNum, line := range lines {
		if sqlPattern.MatchString(line) {
			violations = append(violations, Violation{
				Type:                  ViolationTypeSecurity,
				Severity:              SeverityCritical,
				FilePath:              filePath,
				LineNumber:            lineNum + 1,
				Description:           "Potential SQL injection risk: string concatenation in query",
				Rule:                  "security.no_sql_injection",
				ConstitutionReference: "Section 12.2: Security and Performance",
				Suggestion:            "Use parameterized queries or Ent ORM query builders",
			})
		}
	}

	return violations
}

// Helper methods

func (d *violationDetector) extractModuleName(filePath string) string {
	parts := strings.Split(filePath, "/app/")
	if len(parts) < 2 {
		return ""
	}
	moduleParts := strings.Split(parts[1], "/")
	if len(moduleParts) > 0 {
		return moduleParts[0]
	}
	return ""
}

func (d *violationDetector) isMigrationFile(filePath string) bool {
	return strings.Contains(filePath, "/migrations/") ||
		strings.Contains(filePath, "/migrate/")
}

func (d *violationDetector) isEntSchemaFile(filePath string) bool {
	return strings.Contains(filePath, "/ent/schema/") &&
		strings.HasSuffix(filePath, ".go")
}

func (d *violationDetector) detectMigrationDeletion(filePath string) *Violation {
	// This would need to check git history to detect deletions
	// For now, we'll return nil as this requires git integration
	return nil
}

func (d *violationDetector) detectProtobufBreakingChanges(filePath string) []Violation {
	// This would need to compare with previous version
	// For now, we'll return empty as this requires version comparison
	return nil
}

func (d *violationDetector) detectEntSchemaBreakingChanges(filePath string) []Violation {
	// This would need to compare with previous version
	// For now, we'll return empty as this requires version comparison
	return nil
}

func (d *violationDetector) detectGoModViolations(filePath string) []Violation {
	// This would check for unapproved dependencies
	// For now, we'll return empty as this requires dependency whitelist
	return nil
}

func (d *violationDetector) detectPackageJSONViolations(filePath string) []Violation {
	// This would check for unapproved dependencies
	// For now, we'll return empty as this requires dependency whitelist
	return nil
}
