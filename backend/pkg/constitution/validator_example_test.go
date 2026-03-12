package constitution_test

import (
	"fmt"
	"log"

	"backend/pkg/constitution"
)

func ExampleCodeValidator_ValidateGoCode() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Validate a Go file
	result, err := validator.ValidateGoCode("backend/pkg/constitution/validator.go")
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	if result.Passed {
		fmt.Println("Validation passed")
	} else {
		fmt.Printf("Validation failed with %d errors\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  %s:%d:%d: %s\n", err.File, err.Line, err.Column, err.Message)
		}
	}
}

func ExampleCodeValidator_ValidateVueCode() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Validate a Vue file
	result, err := validator.ValidateVueCode("frontend/apps/admin/src/views/user/index.vue")
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	if result.Passed {
		fmt.Println("Validation passed")
	} else {
		fmt.Printf("Validation failed with %d errors and %d warnings\n",
			len(result.Errors), len(result.Warnings))
	}
}

func ExampleCodeValidator_ValidateProtobuf() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Validate a Protobuf file
	result, err := validator.ValidateProtobuf("backend/api/protos/admin/service/v1/user.proto")
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	if result.Passed {
		fmt.Println("Protobuf validation passed")
	} else {
		fmt.Printf("Protobuf validation failed: %s\n", result.Output)
	}
}

func ExampleCodeValidator_RunTests() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Run tests for a package
	result, err := validator.RunTests("./backend/pkg/constitution/...")
	if err != nil {
		log.Fatalf("Test execution failed: %v", err)
	}

	if result.Passed {
		fmt.Println("All tests passed")
	} else {
		fmt.Printf("Tests failed: %s\n", result.Output)
	}
}

func ExampleCodeValidator_ValidateImports() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create validator
	validator, err := constitution.NewCodeValidator(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Validate Go imports
	result, err := validator.ValidateImports("backend/pkg/constitution/validator.go", "go")
	if err != nil {
		log.Fatalf("Import validation failed: %v", err)
	}

	if result.Passed {
		fmt.Println("All imports are valid")
	} else {
		fmt.Printf("Invalid imports found: %d errors\n", len(result.Errors))
	}
}
