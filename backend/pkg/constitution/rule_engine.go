package constitution

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule represents a constitution rule
type Rule struct {
	ID                    string        `yaml:"id"`
	Name                  string        `yaml:"name"`
	Description           string        `yaml:"description"`
	Type                  ViolationType `yaml:"type"`
	Severity              Severity      `yaml:"severity"`
	Pattern               string        `yaml:"pattern"`
	FilePattern           string        `yaml:"file_pattern"`
	ConstitutionReference string        `yaml:"constitution_reference"`
	Suggestion            string        `yaml:"suggestion"`
	Enabled               bool          `yaml:"enabled"`
}

// RuleMatch represents a rule match result
type RuleMatch struct {
	Rule       Rule
	FilePath   string
	LineNumber int
	MatchText  string
}

// RuleEngine evaluates constitution rules against code
type RuleEngine struct {
	config *Config
	rules  []Rule
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(config *Config) *RuleEngine {
	engine := &RuleEngine{
		config: config,
		rules:  make([]Rule, 0),
	}

	// Load default rules
	engine.loadDefaultRules()

	return engine
}

// loadDefaultRules loads the default set of constitution rules
func (e *RuleEngine) loadDefaultRules() {
	e.rules = []Rule{
		// Architecture rules
		{
			ID:                    "arch-001",
			Name:                  "No pkg to app dependency",
			Description:           "pkg/ layer cannot depend on app/ layer",
			Type:                  ViolationTypeArchitecture,
			Severity:              SeverityCritical,
			Pattern:               `import.*"/app/`,
			FilePattern:           "*/pkg/*",
			ConstitutionReference: "Section 3.1: Three-Layer Architecture",
			Suggestion:            "Move shared code to pkg/ or use dependency inversion",
			Enabled:               true,
		},
		{
			ID:                    "arch-002",
			Name:                  "No api to app/pkg dependency",
			Description:           "api/ layer should be pure Protobuf definitions",
			Type:                  ViolationTypeArchitecture,
			Severity:              SeverityCritical,
			Pattern:               `import.*"/(app|pkg)/`,
			FilePattern:           "*/api/*",
			ConstitutionReference: "Section 3.1: Three-Layer Architecture",
			Suggestion:            "Keep API definitions pure without Go dependencies",
			Enabled:               true,
		},
		{
			ID:                    "arch-003",
			Name:                  "No direct module dependencies",
			Description:           "Modules should communicate via event bus or gRPC",
			Type:                  ViolationTypeArchitecture,
			Severity:              SeverityHigh,
			Pattern:               `import.*"/app/[^/]+/`,
			FilePattern:           "*/app/*",
			ConstitutionReference: "Section 3.4: Event Bus Usage",
			Suggestion:            "Use event bus or gRPC for cross-module communication",
			Enabled:               true,
		},

		// Security rules
		{
			ID:                    "sec-001",
			Name:                  "No hardcoded passwords",
			Description:           "Passwords must not be hardcoded in source code",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityCritical,
			Pattern:               `(?i)(password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}["']`,
			FilePattern:           "*",
			ConstitutionReference: "Section 5.4: Security Violations",
			Suggestion:            "Use environment variables or secure configuration",
			Enabled:               true,
		},
		{
			ID:                    "sec-002",
			Name:                  "No hardcoded API keys",
			Description:           "API keys must not be hardcoded in source code",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityCritical,
			Pattern:               `(?i)(api[_-]?key|apikey)\s*[:=]\s*["'][^"']{16,}["']`,
			FilePattern:           "*",
			ConstitutionReference: "Section 5.4: Security Violations",
			Suggestion:            "Use environment variables or secure configuration",
			Enabled:               true,
		},
		{
			ID:                    "sec-003",
			Name:                  "No hardcoded secrets",
			Description:           "Secrets and tokens must not be hardcoded",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityCritical,
			Pattern:               `(?i)(secret|token)\s*[:=]\s*["'][^"']{16,}["']`,
			FilePattern:           "*",
			ConstitutionReference: "Section 5.4: Security Violations",
			Suggestion:            "Use environment variables or secure configuration",
			Enabled:               true,
		},
		{
			ID:                    "sec-004",
			Name:                  "No authentication bypass",
			Description:           "Authentication checks must not be bypassed",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityCritical,
			Pattern:               `(?i)(skip auth|bypass auth|auth disabled)`,
			FilePattern:           "*.go",
			ConstitutionReference: "Section 5.4: Security Violations",
			Suggestion:            "Remove authentication bypass code",
			Enabled:               true,
		},
		{
			ID:                    "sec-005",
			Name:                  "No sensitive data in logs",
			Description:           "Sensitive data must not be logged",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityHigh,
			Pattern:               `(?i)(log\.|logger\.|fmt\.Print).*\b(password|token|secret|key)\b`,
			FilePattern:           "*.go",
			ConstitutionReference: "Section 12.5: Security and Performance",
			Suggestion:            "Avoid logging sensitive information",
			Enabled:               true,
		},
		{
			ID:                    "sec-006",
			Name:                  "No SQL injection",
			Description:           "SQL queries must use parameterization",
			Type:                  ViolationTypeSecurity,
			Severity:              SeverityCritical,
			Pattern:               `(?i)(Exec|Query|QueryRow)\s*\(\s*["'].*\+`,
			FilePattern:           "*.go",
			ConstitutionReference: "Section 12.2: Security and Performance",
			Suggestion:            "Use parameterized queries or Ent ORM",
			Enabled:               true,
		},

		// Schema rules
		{
			ID:                    "schema-001",
			Name:                  "No migration deletion",
			Description:           "Database migration files must not be deleted",
			Type:                  ViolationTypeSchema,
			Severity:              SeverityCritical,
			Pattern:               "",
			FilePattern:           "*/migrations/*",
			ConstitutionReference: "Section 5.2: Forbidden Actions",
			Suggestion:            "Create a new migration to revert changes",
			Enabled:               true,
		},
		{
			ID:                    "schema-002",
			Name:                  "No Protobuf field deletion",
			Description:           "Protobuf fields must not be deleted (use deprecated)",
			Type:                  ViolationTypeSchema,
			Severity:              SeverityCritical,
			Pattern:               "",
			FilePattern:           "*.proto",
			ConstitutionReference: "Section 5.3: API and Data Model",
			Suggestion:            "Mark field as deprecated instead of deleting",
			Enabled:               true,
		},

		// Dependency rules
		{
			ID:                    "dep-001",
			Name:                  "No unapproved dependencies",
			Description:           "New dependencies require approval",
			Type:                  ViolationTypeDependency,
			Severity:              SeverityHigh,
			Pattern:               "",
			FilePattern:           "go.mod,package.json",
			ConstitutionReference: "Section 5.5: Dependency Management",
			Suggestion:            "Request approval for new dependencies",
			Enabled:               true,
		},

		// Configuration rules
		{
			ID:                    "config-001",
			Name:                  "No production config modification",
			Description:           "Production configuration must not be modified",
			Type:                  ViolationTypeConfiguration,
			Severity:              SeverityCritical,
			Pattern:               "",
			FilePattern:           "*-prod.yaml,*-production.yaml",
			ConstitutionReference: "Section 5.6: Configuration and Environment",
			Suggestion:            "Production config changes require manual approval",
			Enabled:               true,
		},
	}
}

// AddRule adds a custom rule to the engine
func (e *RuleEngine) AddRule(rule Rule) {
	e.rules = append(e.rules, rule)
}

// GetRules returns all rules
func (e *RuleEngine) GetRules() []Rule {
	return e.rules
}

// GetRulesByType returns rules of a specific type
func (e *RuleEngine) GetRulesByType(violationType ViolationType) []Rule {
	var filtered []Rule
	for _, rule := range e.rules {
		if rule.Enabled && rule.Type == violationType {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// GetRulesBySeverity returns rules of a specific severity
func (e *RuleEngine) GetRulesBySeverity(severity Severity) []Rule {
	var filtered []Rule
	for _, rule := range e.rules {
		if rule.Enabled && rule.Severity == severity {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// EvaluateFile evaluates all rules against a file
func (e *RuleEngine) EvaluateFile(filePath string, content string) ([]Violation, error) {
	var violations []Violation

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		// Check if file matches the rule's file pattern
		if !e.matchFilePattern(filePath, rule.FilePattern) {
			continue
		}

		// Evaluate the rule pattern
		matches, err := e.evaluatePattern(filePath, content, rule)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %s: %w", rule.ID, err)
		}

		// Convert matches to violations
		for _, match := range matches {
			violations = append(violations, Violation{
				Type:                  rule.Type,
				Severity:              rule.Severity,
				FilePath:              match.FilePath,
				LineNumber:            match.LineNumber,
				Description:           rule.Description,
				Rule:                  rule.ID,
				ConstitutionReference: rule.ConstitutionReference,
				Suggestion:            rule.Suggestion,
			})
		}
	}

	return violations, nil
}

// evaluatePattern evaluates a rule pattern against file content
func (e *RuleEngine) evaluatePattern(filePath string, content string, rule Rule) ([]RuleMatch, error) {
	var matches []RuleMatch

	if rule.Pattern == "" {
		return matches, nil
	}

	// Compile regex pattern
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Check each line
	for lineNum, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, RuleMatch{
				Rule:       rule,
				FilePath:   filePath,
				LineNumber: lineNum + 1,
				MatchText:  line,
			})
		}
	}

	return matches, nil
}

// matchFilePattern checks if a file path matches a pattern
func (e *RuleEngine) matchFilePattern(filePath string, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Handle multiple patterns separated by comma
	patterns := strings.Split(pattern, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)

		// Simple wildcard matching
		if strings.HasPrefix(p, "*/") {
			// Match any path ending with the pattern
			suffix := strings.TrimPrefix(p, "*/")
			if strings.Contains(filePath, suffix) {
				return true
			}
		} else if strings.HasPrefix(p, "*.") {
			// Match file extension
			ext := strings.TrimPrefix(p, "*")
			if strings.HasSuffix(filePath, ext) {
				return true
			}
		} else if strings.Contains(p, "*") {
			// Convert to regex for complex patterns
			regexPattern := strings.ReplaceAll(p, "*", ".*")
			if matched, _ := regexp.MatchString(regexPattern, filePath); matched {
				return true
			}
		} else {
			// Exact match or contains
			if strings.Contains(filePath, p) {
				return true
			}
		}
	}

	return false
}

// GenerateViolationReport generates a formatted violation report
func (e *RuleEngine) GenerateViolationReport(violations []Violation) string {
	if len(violations) == 0 {
		return "✅ No constitution violations detected.\n"
	}

	var report strings.Builder

	report.WriteString("❌ Constitution Violations Detected\n")
	report.WriteString("=====================================\n\n")

	// Group by severity
	critical := filterBySeverity(violations, SeverityCritical)
	high := filterBySeverity(violations, SeverityHigh)
	medium := filterBySeverity(violations, SeverityMedium)
	low := filterBySeverity(violations, SeverityLow)

	if len(critical) > 0 {
		report.WriteString(fmt.Sprintf("🔴 CRITICAL (%d)\n", len(critical)))
		report.WriteString("─────────────────\n")
		for _, v := range critical {
			e.formatViolation(&report, v)
		}
		report.WriteString("\n")
	}

	if len(high) > 0 {
		report.WriteString(fmt.Sprintf("🟠 HIGH (%d)\n", len(high)))
		report.WriteString("─────────────────\n")
		for _, v := range high {
			e.formatViolation(&report, v)
		}
		report.WriteString("\n")
	}

	if len(medium) > 0 {
		report.WriteString(fmt.Sprintf("🟡 MEDIUM (%d)\n", len(medium)))
		report.WriteString("─────────────────\n")
		for _, v := range medium {
			e.formatViolation(&report, v)
		}
		report.WriteString("\n")
	}

	if len(low) > 0 {
		report.WriteString(fmt.Sprintf("🟢 LOW (%d)\n", len(low)))
		report.WriteString("─────────────────\n")
		for _, v := range low {
			e.formatViolation(&report, v)
		}
		report.WriteString("\n")
	}

	// Summary
	report.WriteString("Summary\n")
	report.WriteString("─────────────────\n")
	report.WriteString(fmt.Sprintf("Total: %d violations\n", len(violations)))
	report.WriteString(fmt.Sprintf("Critical: %d, High: %d, Medium: %d, Low: %d\n",
		len(critical), len(high), len(medium), len(low)))

	if len(critical) > 0 || len(high) > 0 {
		report.WriteString("\n⚠️  Rollback recommended due to critical/high severity violations.\n")
	}

	return report.String()
}

// formatViolation formats a single violation for the report
func (e *RuleEngine) formatViolation(report *strings.Builder, v Violation) {
	report.WriteString(fmt.Sprintf("  [%s] %s\n", v.Rule, v.Description))
	report.WriteString(fmt.Sprintf("  File: %s:%d\n", v.FilePath, v.LineNumber))
	report.WriteString(fmt.Sprintf("  Type: %s\n", v.Type))
	if v.ConstitutionReference != "" {
		report.WriteString(fmt.Sprintf("  Reference: %s\n", v.ConstitutionReference))
	}
	if v.Suggestion != "" {
		report.WriteString(fmt.Sprintf("  💡 Suggestion: %s\n", v.Suggestion))
	}
	report.WriteString("\n")
}

// Helper function to filter violations by severity
func filterBySeverity(violations []Violation, severity Severity) []Violation {
	var filtered []Violation
	for _, v := range violations {
		if v.Severity == severity {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// EvaluateSeverity evaluates the overall severity of violations
func (e *RuleEngine) EvaluateSeverity(violations []Violation) Severity {
	hasCritical := false
	hasHigh := false
	hasMedium := false

	for _, v := range violations {
		switch v.Severity {
		case SeverityCritical:
			hasCritical = true
		case SeverityHigh:
			hasHigh = true
		case SeverityMedium:
			hasMedium = true
		}
	}

	if hasCritical {
		return SeverityCritical
	}
	if hasHigh {
		return SeverityHigh
	}
	if hasMedium {
		return SeverityMedium
	}
	return SeverityLow
}

// ShouldRollback determines if violations warrant a rollback
func (e *RuleEngine) ShouldRollback(violations []Violation) bool {
	for _, v := range violations {
		if v.Severity == SeverityCritical || v.Severity == SeverityHigh {
			return true
		}
	}
	return false
}

// GenerateFixSuggestions generates actionable fix suggestions
func (e *RuleEngine) GenerateFixSuggestions(violations []Violation) []string {
	suggestions := make([]string, 0)
	seen := make(map[string]bool)

	for _, v := range violations {
		if v.Suggestion != "" && !seen[v.Suggestion] {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", v.Rule, v.Suggestion))
			seen[v.Suggestion] = true
		}
	}

	return suggestions
}
