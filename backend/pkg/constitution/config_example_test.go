package constitution_test

import (
	"fmt"
	"log"

	"go-wind-admin/pkg/constitution"
)

// Example demonstrates basic usage of the configuration loader
func Example_basicUsage() {
	// Create a new configuration loader
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get the current configuration
	config := loader.Get()

	// Access configuration values
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Trace Directory: %s\n", config.Trace.Directory)
	fmt.Printf("Auto Rollback: %v\n", config.Rollback.AutoRollback)

	// Get a specific tool command
	goFormatter, err := config.GetToolCommand("go.formatter")
	if err != nil {
		log.Fatalf("Failed to get tool command: %v", err)
	}
	fmt.Printf("Go Formatter: %s\n", goFormatter.Command)

	// Check if a trigger should cause rollback
	if config.IsRollbackTrigger("validation_error") {
		fmt.Println("Validation errors trigger rollback")
	}

	// Calculate retry delay
	delay := config.GetRetryDelay(2)
	fmt.Printf("Retry delay for attempt 2: %v\n", delay)
}

// Example demonstrates hot reloading configuration
func Example_hotReload() {
	// Create a new configuration loader
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Start watching for configuration changes
	if err := loader.StartWatching(); err != nil {
		log.Fatalf("Failed to start watching: %v", err)
	}
	defer loader.StopWatching()

	// Register a callback for configuration changes
	loader.OnChange(func(newConfig *constitution.Config) {
		fmt.Printf("Configuration reloaded! New version: %s\n", newConfig.Version)
		fmt.Printf("New trace retention: %d days\n", newConfig.Trace.RetentionDays)
	})

	// The configuration will automatically reload when the file changes
	// Your application continues running with the updated configuration
	fmt.Println("Watching for configuration changes...")
}

// Example demonstrates accessing different configuration sections
func Example_accessingConfiguration() {
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config := loader.Get()

	// Access validation settings
	fmt.Printf("Go Max Complexity: %d\n", config.Validation.Go.MaxComplexity)
	fmt.Printf("Run Tests: %v\n", config.Validation.Go.RunTests)

	// Access trace settings
	fmt.Printf("Trace Format: %s\n", config.Trace.Format)
	fmt.Printf("Include Diff: %v\n", config.Trace.IncludeDiff)

	// Access error handling settings
	fmt.Printf("Max Retry Attempts: %d\n", config.ErrorHandling.Retry.MaxAttempts)
	fmt.Printf("Backoff Strategy: %s\n", config.ErrorHandling.Retry.BackoffStrategy)

	// Access anti-hallucination settings
	fmt.Printf("Anti-Hallucination Enabled: %v\n", config.AntiHallucination.Enabled)
	fmt.Printf("Verify API Exists: %v\n", config.AntiHallucination.VerifyAPIExists)

	// Access security settings
	fmt.Printf("Check Hardcoded Secrets: %v\n", config.Security.CheckHardcodedSecrets)
	fmt.Printf("Sensitive Patterns: %v\n", config.Security.SensitivePatterns)

	// Access architecture settings
	if len(config.Architecture.BackendLayers) > 0 {
		layer := config.Architecture.BackendLayers[0]
		fmt.Printf("First Backend Layer: %s (%s)\n", layer.Name, layer.Path)
	}
}

// Example demonstrates working with tool commands
func Example_toolCommands() {
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config := loader.Get()

	// Get Go formatter command
	goFormatter, err := config.GetToolCommand("go.formatter")
	if err != nil {
		log.Fatalf("Failed to get go.formatter: %v", err)
	}
	fmt.Printf("Command: %s\n", goFormatter.Command)
	fmt.Printf("Args: %v\n", goFormatter.Args)
	fmt.Printf("Timeout: %d seconds\n", goFormatter.Timeout)
	fmt.Printf("Working Dir: %s\n", goFormatter.WorkingDir)

	// Get Vue linter command
	vueLinter, err := config.GetToolCommand("vue.linter")
	if err != nil {
		log.Fatalf("Failed to get vue.linter: %v", err)
	}
	fmt.Printf("Vue Linter: %s\n", vueLinter.Command)

	// Get Protobuf compiler command
	protocCompiler, err := config.GetToolCommand("protobuf.compiler")
	if err != nil {
		log.Fatalf("Failed to get protobuf.compiler: %v", err)
	}
	fmt.Printf("Protobuf Compiler: %s\n", protocCompiler.Command)
}

// Example demonstrates retry delay calculation
func Example_retryDelays() {
	loader, err := constitution.NewConfigLoader(".ai/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config := loader.Get()

	// Calculate retry delays for different attempts
	fmt.Println("Retry delays with exponential backoff:")
	for attempt := 1; attempt <= 5; attempt++ {
		delay := config.GetRetryDelay(attempt)
		fmt.Printf("Attempt %d: %v\n", attempt, delay)
	}

	// The delay increases exponentially: 1s, 2s, 4s, 8s, 10s (capped at max_delay_ms)
}
