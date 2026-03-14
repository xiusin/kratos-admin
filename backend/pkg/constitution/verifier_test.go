package constitution

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAntiHallucinationVerifier(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	config := &Config{
		ProjectRoot: tmpDir,
	}

	// Create backend directory structure
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	if verifier == nil {
		t.Fatal("Verifier is nil")
	}
}

func TestVerifyAPIExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create proto file
	protoDir := filepath.Join(tmpDir, "backend", "api", "protos", "user", "service", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatalf("Failed to create proto directory: %v", err)
	}

	protoContent := `syntax = "proto3";

package user.service.v1;

// UserService manages user operations
service UserService {
  // CreateUser creates a new user
  rpc CreateUser(CreateUserRequest) returns (User) {}
  
  // GetUser retrieves a user by ID
  rpc GetUser(GetUserRequest) returns (User) {}
}

message CreateUserRequest {
  string username = 1;
  string email = 2;
}

message GetUserRequest {
  int64 id = 1;
}

message User {
  int64 id = 1;
  string username = 2;
  string email = 3;
}
`

	protoFile := filepath.Join(protoDir, "user.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	tests := []struct {
		name        string
		serviceName string
		methodName  string
		wantExists  bool
	}{
		{
			name:        "existing API",
			serviceName: "UserService",
			methodName:  "CreateUser",
			wantExists:  true,
		},
		{
			name:        "another existing API",
			serviceName: "UserService",
			methodName:  "GetUser",
			wantExists:  true,
		},
		{
			name:        "non-existing API",
			serviceName: "UserService",
			methodName:  "DeleteUser",
			wantExists:  false,
		},
		{
			name:        "non-existing service",
			serviceName: "OrderService",
			methodName:  "CreateOrder",
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := verifier.VerifyAPIExists(tt.serviceName, tt.methodName)
			if err != nil {
				t.Fatalf("VerifyAPIExists failed: %v", err)
			}

			if exists != tt.wantExists {
				t.Errorf("VerifyAPIExists(%s, %s) = %v, want %v",
					tt.serviceName, tt.methodName, exists, tt.wantExists)
			}
		})
	}
}

func TestGetAPIReference(t *testing.T) {
	tmpDir := t.TempDir()

	// Create proto file
	protoDir := filepath.Join(tmpDir, "backend", "api", "protos", "user", "service", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatalf("Failed to create proto directory: %v", err)
	}

	protoContent := `syntax = "proto3";

package user.service.v1;

service UserService {
  // CreateUser creates a new user in the system
  rpc CreateUser(CreateUserRequest) returns (User) {}
}

message CreateUserRequest {
  string username = 1;
}

message User {
  int64 id = 1;
}
`

	protoFile := filepath.Join(protoDir, "user.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	ref, err := verifier.GetAPIReference("UserService", "CreateUser")
	if err != nil {
		t.Fatalf("GetAPIReference failed: %v", err)
	}

	if ref.ServiceName != "UserService" {
		t.Errorf("ServiceName = %s, want UserService", ref.ServiceName)
	}

	if ref.MethodName != "CreateUser" {
		t.Errorf("MethodName = %s, want CreateUser", ref.MethodName)
	}

	if ref.RequestType != "CreateUserRequest" {
		t.Errorf("RequestType = %s, want CreateUserRequest", ref.RequestType)
	}

	if ref.ResponseType != "User" {
		t.Errorf("ResponseType = %s, want User", ref.ResponseType)
	}

	if ref.Documentation == "" {
		t.Error("Documentation is empty")
	}
}

func TestVerifyFunctionExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Go source file
	pkgDir := filepath.Join(tmpDir, "backend", "pkg", "utils")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	goContent := `package utils

// FormatString formats a string
func FormatString(s string) string {
	return s
}

// ParseInt parses an integer
func ParseInt(s string) (int, error) {
	return 0, nil
}

// unexported function
func helper() {
}
`

	goFile := filepath.Join(pkgDir, "string.go")
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("Failed to write Go file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	tests := []struct {
		name         string
		packagePath  string
		functionName string
		wantExists   bool
	}{
		{
			name:         "existing function",
			packagePath:  "pkg/utils",
			functionName: "FormatString",
			wantExists:   true,
		},
		{
			name:         "another existing function",
			packagePath:  "pkg/utils",
			functionName: "ParseInt",
			wantExists:   true,
		},
		{
			name:         "non-existing function",
			packagePath:  "pkg/utils",
			functionName: "NonExisting",
			wantExists:   false,
		},
		{
			name:         "unexported function",
			packagePath:  "pkg/utils",
			functionName: "helper",
			wantExists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := verifier.VerifyFunctionExists(tt.packagePath, tt.functionName)
			if err != nil {
				t.Fatalf("VerifyFunctionExists failed: %v", err)
			}

			if exists != tt.wantExists {
				t.Errorf("VerifyFunctionExists(%s, %s) = %v, want %v",
					tt.packagePath, tt.functionName, exists, tt.wantExists)
			}
		})
	}
}

func TestVerifyModuleExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod file
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}

	goModContent := `module github.com/example/project

go 1.21

require (
	github.com/go-kratos/kratos/v2 v2.7.0
	google.golang.org/grpc v1.58.0
	gorm.io/gorm v1.25.0
)
`

	goModFile := filepath.Join(backendDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod file: %v", err)
	}

	// Create package.json file
	frontendDir := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		t.Fatalf("Failed to create frontend directory: %v", err)
	}

	packageJSONContent := `{
  "name": "frontend",
  "dependencies": {
    "vue": "^3.3.0",
    "pinia": "^2.1.0"
  },
  "devDependencies": {
    "vite": "^4.4.0"
  }
}
`

	packageJSONFile := filepath.Join(frontendDir, "package.json")
	if err := os.WriteFile(packageJSONFile, []byte(packageJSONContent), 0644); err != nil {
		t.Fatalf("Failed to write package.json file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	tests := []struct {
		name       string
		modulePath string
		language   string
		wantExists bool
	}{
		{
			name:       "existing Go module",
			modulePath: "github.com/go-kratos/kratos/v2",
			language:   "go",
			wantExists: true,
		},
		{
			name:       "another existing Go module",
			modulePath: "google.golang.org/grpc",
			language:   "go",
			wantExists: true,
		},
		{
			name:       "non-existing Go module",
			modulePath: "github.com/non/existing",
			language:   "go",
			wantExists: false,
		},
		{
			name:       "existing NPM module",
			modulePath: "vue",
			language:   "npm",
			wantExists: true,
		},
		{
			name:       "existing NPM dev module",
			modulePath: "vite",
			language:   "npm",
			wantExists: true,
		},
		{
			name:       "non-existing NPM module",
			modulePath: "non-existing",
			language:   "npm",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := verifier.VerifyModuleExists(tt.modulePath, tt.language)
			if err != nil {
				t.Fatalf("VerifyModuleExists failed: %v", err)
			}

			if exists != tt.wantExists {
				t.Errorf("VerifyModuleExists(%s, %s) = %v, want %v",
					tt.modulePath, tt.language, exists, tt.wantExists)
			}
		})
	}
}

func TestVerifyConfigKeyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	configDir := filepath.Join(tmpDir, "backend", "app", "admin", "service", "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s

database:
  driver: mysql
  source: root:root@tcp(127.0.0.1:3306)/test

redis:
  addr: 127.0.0.1:6379
  read_timeout: 0.2s
  write_timeout: 0.2s
`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	tests := []struct {
		name       string
		configKey  string
		wantExists bool
	}{
		{
			name:       "top-level key",
			configKey:  "server",
			wantExists: true,
		},
		{
			name:       "nested key",
			configKey:  "server.http",
			wantExists: true,
		},
		{
			name:       "deep nested key",
			configKey:  "server.http.addr",
			wantExists: true,
		},
		{
			name:       "another nested key",
			configKey:  "database.driver",
			wantExists: true,
		},
		{
			name:       "non-existing key",
			configKey:  "server.websocket",
			wantExists: false,
		},
		{
			name:       "non-existing nested key",
			configKey:  "server.http.max_connections",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := verifier.VerifyConfigKeyExists(tt.configKey)
			if err != nil {
				t.Fatalf("VerifyConfigKeyExists failed: %v", err)
			}

			if exists != tt.wantExists {
				t.Errorf("VerifyConfigKeyExists(%s) = %v, want %v",
					tt.configKey, exists, tt.wantExists)
			}
		})
	}
}

func TestGetFunctionSignature(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Go source file
	pkgDir := filepath.Join(tmpDir, "backend", "pkg", "utils")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	goContent := `package utils

// FormatString formats a string with the given prefix
func FormatString(prefix string, s string) string {
	return prefix + s
}

// ParseInt parses an integer from string
func ParseInt(s string) (int, error) {
	return 0, nil
}
`

	goFile := filepath.Join(pkgDir, "string.go")
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("Failed to write Go file: %v", err)
	}

	config := &Config{
		ProjectRoot: tmpDir,
	}

	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	sig, err := verifier.GetFunctionSignature("pkg/utils", "FormatString")
	if err != nil {
		t.Fatalf("GetFunctionSignature failed: %v", err)
	}

	if sig.FunctionName != "FormatString" {
		t.Errorf("FunctionName = %s, want FormatString", sig.FunctionName)
	}

	if len(sig.Parameters) != 2 {
		t.Errorf("Parameters length = %d, want 2", len(sig.Parameters))
	}

	if len(sig.ReturnTypes) != 1 {
		t.Errorf("ReturnTypes length = %d, want 1", len(sig.ReturnTypes))
	}

	if sig.Documentation == "" {
		t.Error("Documentation is empty")
	}
}
