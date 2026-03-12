package constitution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewIndexDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "index.json")
	
	db, err := NewIndexDatabase(indexFile)
	if err != nil {
		t.Fatalf("Failed to create index database: %v", err)
	}
	
	if db == nil {
		t.Fatal("Index database is nil")
	}
	
	if db.APIIndex == nil {
		t.Error("APIIndex is nil")
	}
	
	if db.FuncIndex == nil {
		t.Error("FuncIndex is nil")
	}
	
	if db.ModuleIndex == nil {
		t.Error("ModuleIndex is nil")
	}
	
	if db.ConfigIndex == nil {
		t.Error("ConfigIndex is nil")
	}
}

func TestIndexDatabaseSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "index.json")
	
	// Create and populate database
	db, err := NewIndexDatabase(indexFile)
	if err != nil {
		t.Fatalf("Failed to create index database: %v", err)
	}
	
	// Add some data
	db.AddAPIReference("UserService.CreateUser", &APIReference{
		ServiceName:  "UserService",
		MethodName:   "CreateUser",
		FilePath:     "/path/to/user.proto",
		LineNumber:   10,
		RequestType:  "CreateUserRequest",
		ResponseType: "User",
	})
	
	db.AddFunctionSignature("pkg/utils.FormatString", &FunctionSignature{
		PackagePath:  "pkg/utils",
		FunctionName: "FormatString",
		FilePath:     "/path/to/string.go",
		LineNumber:   5,
		Parameters:   []string{"s string"},
		ReturnTypes:  []string{"string"},
	})
	
	db.AddModule("go:github.com/go-kratos/kratos/v2")
	db.AddConfigKey("server.http.addr")
	
	// Save database
	if err := db.Save(); err != nil {
		t.Fatalf("Failed to save database: %v", err)
	}
	
	// Check file exists
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		t.Fatal("Index file was not created")
	}
	
	// Load database
	db2, err := NewIndexDatabase(indexFile)
	if err != nil {
		t.Fatalf("Failed to load index database: %v", err)
	}
	
	// Verify data
	ref, exists := db2.GetAPIReference("UserService.CreateUser")
	if !exists {
		t.Error("API reference not found after load")
	}
	if ref.ServiceName != "UserService" {
		t.Errorf("ServiceName = %s, want UserService", ref.ServiceName)
	}
	
	sig, exists := db2.GetFunctionSignature("pkg/utils.FormatString")
	if !exists {
		t.Error("Function signature not found after load")
	}
	if sig.FunctionName != "FormatString" {
		t.Errorf("FunctionName = %s, want FormatString", sig.FunctionName)
	}
	
	if !db2.HasModule("go:github.com/go-kratos/kratos/v2") {
		t.Error("Module not found after load")
	}
	
	if !db2.HasConfigKey("server.http.addr") {
		t.Error("Config key not found after load")
	}
}

func TestIndexDatabaseStats(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "index.json")
	
	db, err := NewIndexDatabase(indexFile)
	if err != nil {
		t.Fatalf("Failed to create index database: %v", err)
	}
	
	// Add some data
	db.AddAPIReference("Service1.Method1", &APIReference{})
	db.AddAPIReference("Service1.Method2", &APIReference{})
	db.AddFunctionSignature("pkg1.Func1", &FunctionSignature{})
	db.AddModule("go:module1")
	db.AddModule("go:module2")
	db.AddModule("npm:module3")
	db.AddConfigKey("key1")
	
	stats := db.Stats()
	
	if stats["api_count"] != 2 {
		t.Errorf("api_count = %v, want 2", stats["api_count"])
	}
	
	if stats["func_count"] != 1 {
		t.Errorf("func_count = %v, want 1", stats["func_count"])
	}
	
	if stats["module_count"] != 3 {
		t.Errorf("module_count = %v, want 3", stats["module_count"])
	}
	
	if stats["config_count"] != 1 {
		t.Errorf("config_count = %v, want 1", stats["config_count"])
	}
}

func TestIndexDatabaseClear(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "index.json")
	
	db, err := NewIndexDatabase(indexFile)
	if err != nil {
		t.Fatalf("Failed to create index database: %v", err)
	}
	
	// Add some data
	db.AddAPIReference("Service1.Method1", &APIReference{})
	db.AddFunctionSignature("pkg1.Func1", &FunctionSignature{})
	db.AddModule("go:module1")
	db.AddConfigKey("key1")
	
	// Clear database
	db.Clear()
	
	stats := db.Stats()
	
	if stats["api_count"] != 0 {
		t.Errorf("api_count after clear = %v, want 0", stats["api_count"])
	}
	
	if stats["func_count"] != 0 {
		t.Errorf("func_count after clear = %v, want 0", stats["func_count"])
	}
	
	if stats["module_count"] != 0 {
		t.Errorf("module_count after clear = %v, want 0", stats["module_count"])
	}
	
	if stats["config_count"] != 0 {
		t.Errorf("config_count after clear = %v, want 0", stats["config_count"])
	}
}

func TestIndexWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &Config{
		ProjectRoot: tmpDir,
	}
	
	// Create backend directory
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}
	
	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}
	
	v := verifier.(*antiHallucinationVerifier)
	watcher := NewIndexWatcher(config, v)
	
	// Add watch path
	watcher.AddWatchPath(backendDir)
	
	// Start watcher
	watcher.Start(100 * time.Millisecond)
	
	// Create a file
	testFile := filepath.Join(backendDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Wait for watcher to detect change
	time.Sleep(200 * time.Millisecond)
	
	// Stop watcher
	watcher.Stop()
}

func TestIndexUpdateTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &Config{
		ProjectRoot: tmpDir,
	}
	
	// Create directory structure
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}
	
	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}
	
	v := verifier.(*antiHallucinationVerifier)
	trigger := NewIndexUpdateTrigger(config, v)
	
	// Test manual update
	if err := trigger.TriggerManualUpdate(); err != nil {
		t.Errorf("TriggerManualUpdate failed: %v", err)
	}
	
	// Test proto file change
	if err := trigger.OnProtoFileChange("/path/to/user.proto"); err != nil {
		t.Errorf("OnProtoFileChange failed: %v", err)
	}
	
	// Test Go file change
	if err := trigger.OnGoFileChange("/path/to/service.go"); err != nil {
		t.Errorf("OnGoFileChange failed: %v", err)
	}
	
	// Test config file change
	if err := trigger.OnConfigFileChange("/path/to/config.yaml"); err != nil {
		t.Errorf("OnConfigFileChange failed: %v", err)
	}
	
	// Test dependency change
	if err := trigger.OnDependencyChange("/path/to/go.mod"); err != nil {
		t.Errorf("OnDependencyChange failed: %v", err)
	}
}

func TestIndexUpdateTriggerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &Config{
		ProjectRoot: tmpDir,
	}
	
	// Create directory structure
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend directory: %v", err)
	}
	
	verifier, err := NewAntiHallucinationVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}
	
	v := verifier.(*antiHallucinationVerifier)
	trigger := NewIndexUpdateTrigger(config, v)
	
	// Start trigger
	trigger.Start(100 * time.Millisecond)
	
	// Wait a bit
	time.Sleep(200 * time.Millisecond)
	
	// Stop trigger
	trigger.Stop()
}
