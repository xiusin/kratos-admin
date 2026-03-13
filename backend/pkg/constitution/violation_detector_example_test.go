package constitution_test

import (
	"context"
	"fmt"
	"log"

	"backend/pkg/constitution"
)

func ExampleViolationDetector_DetectArchitectureViolations() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create violation detector
	detector := constitution.NewViolationDetector(cfg, ".")

	// Files to check
	files := []string{
		"backend/pkg/utils/helper.go",
		"backend/app/admin/service/internal/service/user.go",
	}

	// Detect architecture violations
	violations, err := detector.DetectArchitectureViolations(context.Background(), files)
	if err != nil {
		log.Fatalf("Failed to detect violations: %v", err)
	}

	// Print violations
	if len(violations) == 0 {
		fmt.Println("No architecture violations detected")
	} else {
		fmt.Printf("Found %d architecture violations\n", len(violations))
		for _, v := range violations {
			fmt.Printf("- %s at %s:%d\n", v.Description, v.FilePath, v.LineNumber)
		}
	}
}

func ExampleViolationDetector_DetectSecurityViolations() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create violation detector
	detector := constitution.NewViolationDetector(cfg, ".")

	// Files to check
	files := []string{
		"backend/app/admin/service/internal/service/auth.go",
	}

	// Detect security violations
	violations, err := detector.DetectSecurityViolations(context.Background(), files)
	if err != nil {
		log.Fatalf("Failed to detect violations: %v", err)
	}

	// Print violations
	if len(violations) == 0 {
		fmt.Println("No security violations detected")
	} else {
		fmt.Printf("Found %d security violations\n", len(violations))
		for _, v := range violations {
			fmt.Printf("- [%s] %s at %s:%d\n", v.Severity, v.Description, v.FilePath, v.LineNumber)
			fmt.Printf("  Suggestion: %s\n", v.Suggestion)
		}
	}
}

func ExampleViolationDetector_DetectAllViolations() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create violation detector
	detector := constitution.NewViolationDetector(cfg, ".")

	// Files to check (all modified files in a task)
	files := []string{
		"backend/pkg/utils/helper.go",
		"backend/app/admin/service/internal/service/user.go",
		"backend/app/admin/service/internal/data/user.go",
	}

	// Detect all violations
	report, err := detector.DetectAllViolations(context.Background(), files)
	if err != nil {
		log.Fatalf("Failed to detect violations: %v", err)
	}

	// Print report
	fmt.Printf("Violation Report:\n")
	fmt.Printf("- Total: %d violations\n", len(report.Violations))
	fmt.Printf("- Critical: %d\n", report.CriticalCount)
	fmt.Printf("- High: %d\n", report.HighCount)
	fmt.Printf("- Medium: %d\n", report.MediumCount)
	fmt.Printf("- Low: %d\n", report.LowCount)
	fmt.Printf("- Should Rollback: %v\n", report.ShouldRollback)

	// Print violations by type
	archViolations := 0
	secViolations := 0
	for _, v := range report.Violations {
		switch v.Type {
		case constitution.ViolationTypeArchitecture:
			archViolations++
		case constitution.ViolationTypeSecurity:
			secViolations++
		}
	}
	fmt.Printf("- Architecture: %d\n", archViolations)
	fmt.Printf("- Security: %d\n", secViolations)
}

func ExampleRuleEngine_EvaluateFile() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := constitution.NewRuleEngine(cfg)

	// File content to evaluate
	filePath := "backend/app/admin/service/internal/service/user.go"
	content := `package service

import (
	"context"
	"backend/app/admin/service/internal/data"
)

func (s *UserService) CreateUser(ctx context.Context) error {
	password := "hardcoded123" // This is a violation
	log.Printf("User password: %s", password) // This is also a violation
	return nil
}
`

	// Evaluate file
	violations, err := engine.EvaluateFile(filePath, content)
	if err != nil {
		log.Fatalf("Failed to evaluate file: %v", err)
	}

	// Generate report
	report := engine.GenerateViolationReport(violations)
	fmt.Println(report)

	// Get fix suggestions
	suggestions := engine.GenerateFixSuggestions(violations)
	fmt.Println("Fix Suggestions:")
	for _, s := range suggestions {
		fmt.Printf("- %s\n", s)
	}
}

func ExampleRuleEngine_GetRulesByType() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := constitution.NewRuleEngine(cfg)

	// Get security rules
	securityRules := engine.GetRulesByType(constitution.ViolationTypeSecurity)
	fmt.Printf("Security Rules: %d\n", len(securityRules))
	for _, rule := range securityRules {
		fmt.Printf("- [%s] %s (%s)\n", rule.ID, rule.Name, rule.Severity)
	}

	// Get architecture rules
	archRules := engine.GetRulesByType(constitution.ViolationTypeArchitecture)
	fmt.Printf("\nArchitecture Rules: %d\n", len(archRules))
	for _, rule := range archRules {
		fmt.Printf("- [%s] %s (%s)\n", rule.ID, rule.Name, rule.Severity)
	}
}

func ExampleRuleEngine_AddRule() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := constitution.NewRuleEngine(cfg)

	// Add custom rule
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

	engine.AddRule(customRule)

	fmt.Printf("Added custom rule: %s\n", customRule.Name)
	fmt.Printf("Total rules: %d\n", len(engine.GetRules()))
}

func ExampleRuleEngine_ShouldRollback() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := constitution.NewRuleEngine(cfg)

	// Example violations
	violations := []constitution.Violation{
		{
			Type:        constitution.ViolationTypeSecurity,
			Severity:    constitution.SeverityCritical,
			Description: "Hardcoded password detected",
		},
		{
			Type:        constitution.ViolationTypeArchitecture,
			Severity:    constitution.SeverityMedium,
			Description: "Missing documentation",
		},
	}

	// Check if rollback is needed
	shouldRollback := engine.ShouldRollback(violations)
	fmt.Printf("Should rollback: %v\n", shouldRollback)

	// Evaluate overall severity
	severity := engine.EvaluateSeverity(violations)
	fmt.Printf("Overall severity: %s\n", severity)
}
