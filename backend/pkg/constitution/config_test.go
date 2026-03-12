package constitution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigLoader(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a valid config
	validConfig := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      args: ["-l", "-w"]
      timeout: 30
      working_dir: "backend"
    linter:
      command: "golangci-lint"
      args: ["run"]
      timeout: 120
      working_dir: "backend"
    test_runner:
      command: "go"
      args: ["test", "./..."]
      timeout: 300
      working_dir: "backend"
  vue:
    linter:
      command: "eslint"
      args: ["--fix"]
      timeout: 60
      working_dir: "frontend"
    type_checker:
      command: "vue-tsc"
      args: ["--noEmit"]
      timeout: 120
      working_dir: "frontend"
    formatter:
      command: "prettier"
      args: ["--write"]
      timeout: 30
      working_dir: "frontend"
  protobuf:
    compiler:
      command: "protoc"
      args: ["--go_out=."]
      timeout: 60
      working_dir: "."
  ent:
    generator:
      command: "go"
      args: ["generate", "./ent"]
      timeout: 60
      working_dir: "backend"
validation:
  go:
    run_tests: true
    check_imports: true
    max_complexity: 15
    check_error_handling: true
    check_context_usage: true
    rules: ["no_raw_sql"]
  vue:
    check_types: true
    check_props: true
    check_emits: true
    check_component_naming: true
    rules: ["require_prop_types"]
  protobuf:
    lint_rules: ["PACKAGE_DEFINED"]
    check_compatibility: true
  ent:
    check_constraints: true
    check_indexes: true
    check_relations: true
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
  verbose: true
  include_diff: true
  include_rationale: true
  include_references: true
rollback:
  auto_rollback: true
  backup_before_change: true
  backup_directory: ".ai/backups"
  backup_retention_success: 7
  backup_retention_failure: 30
  verify_rollback: true
  triggers: ["validation_error", "constitution_violation"]
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
  reporting:
    include_stack_trace: true
    include_context: true
    include_suggestions: true
documentation:
  auto_sync: true
  api_docs_path: "docs/api"
  component_docs_path: "docs/components"
  feature_docs_path: "docs/features"
  templates_path: ".ai/templates"
  generate_examples: true
  generate_api_reference: true
anti_hallucination:
  enabled: true
  verify_api_exists: true
  verify_function_exists: true
  verify_module_exists: true
  verify_config_exists: true
  request_confirmation: true
  index_update_triggers: ["protobuf_change"]
architecture:
  backend_layers:
    - name: "api"
      path: "backend/api"
      description: "API layer"
      allowed_dependencies: []
  frontend_structure:
    - name: "apps"
      path: "frontend/apps"
      description: "Applications"
  check_dependency_direction: true
  allow_cross_layer_calls: false
security:
  check_hardcoded_secrets: true
  check_auth_bypass: true
  check_sql_injection: true
  check_sensitive_data_leak: true
  sensitive_patterns: ["password", "secret"]
performance:
  check_n_plus_one: true
  check_large_loops: true
  check_memory_leaks: true
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	require.NoError(t, err)

	// Test creating a new config loader
	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)
	require.NotNil(t, loader)

	// Verify config was loaded
	config := loader.Get()
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, ".ai/traces", config.Trace.Directory)
	assert.Equal(t, "json", config.Trace.Format)
	assert.Equal(t, 90, config.Trace.RetentionDays)
	assert.True(t, config.Rollback.AutoRollback)
	assert.Equal(t, 3, config.ErrorHandling.Retry.MaxAttempts)
}

func TestConfigLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write minimal valid config
	minimalConfig := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
  vue:
    linter:
      command: "eslint"
      timeout: 60
    type_checker:
      command: "vue-tsc"
      timeout: 120
    formatter:
      command: "prettier"
      timeout: 30
  protobuf:
    compiler:
      command: "protoc"
      timeout: 60
  ent:
    generator:
      command: "go"
      timeout: 60
validation:
  go:
    max_complexity: 15
  protobuf:
    lint_rules: []
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
performance:
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)

	config := loader.Get()
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "gofmt", config.Tools.Go.Formatter.Command)
}

func TestConfigLoader_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectedErr string
	}{
		{
			name: "missing version",
			config: `tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
`,
			expectedErr: "config version is required",
		},
		{
			name: "missing trace directory",
			config: `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
trace:
  format: "json"
  retention_days: 90
`,
			expectedErr: "trace directory is required",
		},
		{
			name: "invalid trace format",
			config: `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
trace:
  directory: ".ai/traces"
  format: "xml"
  retention_days: 90
`,
			expectedErr: "trace format must be 'json'",
		},
		{
			name: "negative retention days",
			config: `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: -1
`,
			expectedErr: "trace retention days must be non-negative",
		},
		{
			name: "invalid backoff strategy",
			config: `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "invalid"
    initial_delay_ms: 1000
    max_delay_ms: 10000
`,
			expectedErr: "invalid backoff strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			_, err = NewConfigLoader(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestConfigLoader_GetToolCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      args: ["-l", "-w"]
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
  vue:
    linter:
      command: "eslint"
      timeout: 60
    type_checker:
      command: "vue-tsc"
      timeout: 120
    formatter:
      command: "prettier"
      timeout: 30
  protobuf:
    compiler:
      command: "protoc"
      timeout: 60
  ent:
    generator:
      command: "go"
      timeout: 60
validation:
  go:
    max_complexity: 15
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
performance:
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)

	cfg := loader.Get()

	// Test valid tool commands
	cmd, err := cfg.GetToolCommand("go.formatter")
	require.NoError(t, err)
	assert.Equal(t, "gofmt", cmd.Command)
	assert.Equal(t, []string{"-l", "-w"}, cmd.Args)

	cmd, err = cfg.GetToolCommand("vue.linter")
	require.NoError(t, err)
	assert.Equal(t, "eslint", cmd.Command)

	// Test invalid tool command
	_, err = cfg.GetToolCommand("invalid.tool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestConfigLoader_IsRollbackTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
  triggers:
    - "validation_error"
    - "constitution_violation"
    - "test_failure"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
validation:
  go:
    max_complexity: 15
performance:
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)

	cfg := loader.Get()

	assert.True(t, cfg.IsRollbackTrigger("validation_error"))
	assert.True(t, cfg.IsRollbackTrigger("constitution_violation"))
	assert.True(t, cfg.IsRollbackTrigger("test_failure"))
	assert.False(t, cfg.IsRollbackTrigger("unknown_trigger"))
}

func TestConfigLoader_GetRetryDelay(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	tests := []struct {
		name     string
		strategy string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "exponential backoff attempt 1",
			strategy: "exponential",
			attempt:  1,
			expected: 1000 * time.Millisecond,
		},
		{
			name:     "exponential backoff attempt 2",
			strategy: "exponential",
			attempt:  2,
			expected: 2000 * time.Millisecond,
		},
		{
			name:     "exponential backoff attempt 3",
			strategy: "exponential",
			attempt:  3,
			expected: 4000 * time.Millisecond,
		},
		{
			name:     "linear backoff attempt 1",
			strategy: "linear",
			attempt:  1,
			expected: 1000 * time.Millisecond,
		},
		{
			name:     "linear backoff attempt 2",
			strategy: "linear",
			attempt:  2,
			expected: 2000 * time.Millisecond,
		},
		{
			name:     "linear backoff attempt 3",
			strategy: "linear",
			attempt:  3,
			expected: 3000 * time.Millisecond,
		},
		{
			name:     "constant backoff attempt 1",
			strategy: "constant",
			attempt:  1,
			expected: 1000 * time.Millisecond,
		},
		{
			name:     "constant backoff attempt 3",
			strategy: "constant",
			attempt:  3,
			expected: 1000 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "` + tt.strategy + `"
    initial_delay_ms: 1000
    max_delay_ms: 10000
validation:
  go:
    max_complexity: 15
performance:
  max_query_complexity: 10
`

			err := os.WriteFile(configPath, []byte(config), 0644)
			require.NoError(t, err)

			loader, err := NewConfigLoader(configPath)
			require.NoError(t, err)

			cfg := loader.Get()
			delay := cfg.GetRetryDelay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestConfigLoader_GetRetryDelay_MaxDelay(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 10
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 5000
validation:
  go:
    max_complexity: 15
performance:
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)

	cfg := loader.Get()

	// Attempt 10 would normally be 1000 * 2^9 = 512000ms, but should be capped at 5000ms
	delay := cfg.GetRetryDelay(10)
	assert.Equal(t, 5000*time.Millisecond, delay)
}

func TestConfigLoader_HotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 90
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
validation:
  go:
    max_complexity: 15
performance:
  max_query_complexity: 10
`

	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	loader, err := NewConfigLoader(configPath)
	require.NoError(t, err)

	// Start watching
	err = loader.StartWatching()
	require.NoError(t, err)
	defer loader.StopWatching()

	// Register a callback
	callbackCalled := false
	var newConfig *Config
	loader.OnChange(func(cfg *Config) {
		callbackCalled = true
		newConfig = cfg
	})

	// Verify initial config
	config := loader.Get()
	assert.Equal(t, 90, config.Trace.RetentionDays)

	// Update config file
	updatedConfig := `version: "1.0"
tools:
  go:
    formatter:
      command: "gofmt"
      timeout: 30
    linter:
      command: "golangci-lint"
      timeout: 120
    test_runner:
      command: "go"
      timeout: 300
trace:
  directory: ".ai/traces"
  format: "json"
  retention_days: 180
rollback:
  backup_before_change: true
  backup_directory: ".ai/backups"
error_handling:
  retry:
    max_attempts: 3
    backoff_strategy: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000
validation:
  go:
    max_complexity: 15
performance:
  max_query_complexity: 10
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Wait for file watcher to detect change and reload
	time.Sleep(1 * time.Second)

	// Verify config was reloaded
	config = loader.Get()
	assert.Equal(t, 180, config.Trace.RetentionDays)

	// Verify callback was called
	assert.True(t, callbackCalled)
	assert.NotNil(t, newConfig)
	assert.Equal(t, 180, newConfig.Trace.RetentionDays)
}
