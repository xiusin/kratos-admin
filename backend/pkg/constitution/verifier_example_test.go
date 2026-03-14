package constitution_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go-wind-admin/pkg/constitution"
)

// ExampleAntiHallucinationVerifier demonstrates how to use the anti-hallucination verifier
func ExampleAntiHallucinationVerifier() {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "constitution-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample proto file
	protoDir := filepath.Join(tmpDir, "backend", "api", "protos", "user", "service", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		log.Fatal(err)
	}

	protoContent := `syntax = "proto3";

package user.service.v1;

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User) {}
  rpc GetUser(GetUserRequest) returns (User) {}
}

message CreateUserRequest {
  string username = 1;
}

message GetUserRequest {
  int64 id = 1;
}

message User {
  int64 id = 1;
  string username = 2;
}
`

	protoFile := filepath.Join(protoDir, "user.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		log.Fatal(err)
	}

	// Create configuration
	config := &constitution.Config{
		ProjectRoot: tmpDir,
	}

	// Create verifier
	verifier, err := constitution.NewAntiHallucinationVerifier(config)
	if err != nil {
		log.Fatal(err)
	}

	// Verify API exists
	exists, err := verifier.VerifyAPIExists("UserService", "CreateUser")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("UserService.CreateUser exists: %v\n", exists)

	// Get API reference
	ref, err := verifier.GetAPIReference("UserService", "CreateUser")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Service: %s\n", ref.ServiceName)
	fmt.Printf("Method: %s\n", ref.MethodName)
	fmt.Printf("Request: %s\n", ref.RequestType)
	fmt.Printf("Response: %s\n", ref.ResponseType)

	// Output:
	// UserService.CreateUser exists: true
	// Service: UserService
	// Method: CreateUser
	// Request: CreateUserRequest
	// Response: User
}

// ExampleAntiHallucinationVerifier_verifyFunction demonstrates function verification
func ExampleAntiHallucinationVerifier_verifyFunction() {
	tmpDir, err := os.MkdirTemp("", "constitution-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample Go file
	pkgDir := filepath.Join(tmpDir, "backend", "pkg", "utils")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		log.Fatal(err)
	}

	goContent := `package utils

// FormatString formats a string
func FormatString(s string) string {
	return s
}
`

	goFile := filepath.Join(pkgDir, "string.go")
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		log.Fatal(err)
	}

	config := &constitution.Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := constitution.NewAntiHallucinationVerifier(config)
	if err != nil {
		log.Fatal(err)
	}

	// Verify function exists
	exists, err := verifier.VerifyFunctionExists("pkg/utils", "FormatString")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("FormatString exists: %v\n", exists)

	// Get function signature
	sig, err := verifier.GetFunctionSignature("pkg/utils", "FormatString")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Function: %s\n", sig.FunctionName)
	fmt.Printf("Package: %s\n", sig.PackagePath)

	// Output:
	// FormatString exists: true
	// Function: FormatString
	// Package: pkg/utils
}

// ExampleAntiHallucinationVerifier_verifyModule demonstrates module verification
func ExampleAntiHallucinationVerifier_verifyModule() {
	tmpDir, err := os.MkdirTemp("", "constitution-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		log.Fatal(err)
	}

	goModContent := `module github.com/example/project

go 1.21

require (
	github.com/go-kratos/kratos/v2 v2.7.0
)
`

	goModFile := filepath.Join(backendDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		log.Fatal(err)
	}

	config := &constitution.Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := constitution.NewAntiHallucinationVerifier(config)
	if err != nil {
		log.Fatal(err)
	}

	// Verify module exists
	exists, err := verifier.VerifyModuleExists("github.com/go-kratos/kratos/v2", "go")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Module exists: %v\n", exists)

	// Output:
	// Module exists: true
}

// ExampleIndexDatabase demonstrates how to use the index database
func ExampleIndexDatabase() {
	tmpDir, err := os.MkdirTemp("", "constitution-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	indexFile := filepath.Join(tmpDir, "index.json")

	// Create index database
	db, err := constitution.NewIndexDatabase(indexFile)
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
	})

	// Save database
	if err := db.Save(); err != nil {
		log.Fatal(err)
	}

	// Get statistics
	stats := db.Stats()
	fmt.Printf("API count: %v\n", stats["api_count"])
	fmt.Printf("Function count: %v\n", stats["func_count"])

	// Output:
	// API count: 1
	// Function count: 1
}

// ExampleIndexUpdateTrigger demonstrates how to use the index update trigger
func ExampleIndexUpdateTrigger() {
	tmpDir, err := os.MkdirTemp("", "constitution-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend directory
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		log.Fatal(err)
	}

	config := &constitution.Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := constitution.NewAntiHallucinationVerifier(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create index update trigger
	trigger := constitution.NewIndexUpdateTrigger(config, verifier.(*constitution.AntiHallucinationVerifier))

	// Manually trigger update
	if err := trigger.TriggerManualUpdate(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Index updated successfully")

	// Output:
	// Index updated successfully
}
