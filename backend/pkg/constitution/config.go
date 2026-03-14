package constitution

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config represents the complete constitution configuration
type Config struct {
	Version     string `yaml:"version"`
	ProjectRoot string `yaml:"project_root"` // 项目根目录

	Tools             ToolsConfig             `yaml:"tools"`
	Validation        ValidationConfig        `yaml:"validation"`
	Trace             TraceConfig             `yaml:"trace"`
	Rollback          RollbackConfig          `yaml:"rollback"`
	ErrorHandling     ErrorHandlingConfig     `yaml:"error_handling"`
	Documentation     DocumentationConfig     `yaml:"documentation"`
	AntiHallucination AntiHallucinationConfig `yaml:"anti_hallucination"`
	Architecture      ArchitectureConfig      `yaml:"architecture"`
	Security          SecurityConfig          `yaml:"security"`
	Performance       PerformanceConfig       `yaml:"performance"`
}

// ToolsConfig contains tool configurations
type ToolsConfig struct {
	Go       GoToolsConfig       `yaml:"go"`
	Vue      VueToolsConfig      `yaml:"vue"`
	Protobuf ProtobufToolsConfig `yaml:"protobuf"`
	Ent      EntToolsConfig      `yaml:"ent"`
}

// GoToolsConfig contains Go tool configurations
type GoToolsConfig struct {
	Formatter  ToolCommand `yaml:"formatter"`
	Linter     ToolCommand `yaml:"linter"`
	TestRunner ToolCommand `yaml:"test_runner"`
}

// VueToolsConfig contains Vue tool configurations
type VueToolsConfig struct {
	Linter      ToolCommand `yaml:"linter"`
	TypeChecker ToolCommand `yaml:"type_checker"`
	Formatter   ToolCommand `yaml:"formatter"`
}

// ProtobufToolsConfig contains Protobuf tool configurations
type ProtobufToolsConfig struct {
	Compiler ToolCommand `yaml:"compiler"`
}

// EntToolsConfig contains Ent tool configurations
type EntToolsConfig struct {
	Generator ToolCommand `yaml:"generator"`
}

// ToolCommand represents a tool command configuration
type ToolCommand struct {
	Command    string   `yaml:"command"`
	Args       []string `yaml:"args"`
	Timeout    int      `yaml:"timeout"`
	WorkingDir string   `yaml:"working_dir"`
}

// ValidationConfig contains validation configurations
type ValidationConfig struct {
	Go       GoValidationConfig       `yaml:"go"`
	Vue      VueValidationConfig      `yaml:"vue"`
	Protobuf ProtobufValidationConfig `yaml:"protobuf"`
	Ent      EntValidationConfig      `yaml:"ent"`
}

// GoValidationConfig contains Go validation settings
type GoValidationConfig struct {
	RunTests           bool     `yaml:"run_tests"`
	CheckImports       bool     `yaml:"check_imports"`
	MaxComplexity      int      `yaml:"max_complexity"`
	CheckErrorHandling bool     `yaml:"check_error_handling"`
	CheckContextUsage  bool     `yaml:"check_context_usage"`
	Rules              []string `yaml:"rules"`
}

// VueValidationConfig contains Vue validation settings
type VueValidationConfig struct {
	CheckTypes           bool     `yaml:"check_types"`
	CheckProps           bool     `yaml:"check_props"`
	CheckEmits           bool     `yaml:"check_emits"`
	CheckComponentNaming bool     `yaml:"check_component_naming"`
	Rules                []string `yaml:"rules"`
}

// ProtobufValidationConfig contains Protobuf validation settings
type ProtobufValidationConfig struct {
	LintRules          []string `yaml:"lint_rules"`
	CheckCompatibility bool     `yaml:"check_compatibility"`
}

// EntValidationConfig contains Ent validation settings
type EntValidationConfig struct {
	CheckConstraints bool `yaml:"check_constraints"`
	CheckIndexes     bool `yaml:"check_indexes"`
	CheckRelations   bool `yaml:"check_relations"`
}

// TraceConfig contains task trace configurations
type TraceConfig struct {
	Directory         string `yaml:"directory"`
	Format            string `yaml:"format"`
	RetentionDays     int    `yaml:"retention_days"`
	Verbose           bool   `yaml:"verbose"`
	IncludeDiff       bool   `yaml:"include_diff"`
	IncludeRationale  bool   `yaml:"include_rationale"`
	IncludeReferences bool   `yaml:"include_references"`
}

// RollbackConfig contains rollback configurations
type RollbackConfig struct {
	AutoRollback           bool     `yaml:"auto_rollback"`
	BackupBeforeChange     bool     `yaml:"backup_before_change"`
	BackupDirectory        string   `yaml:"backup_directory"`
	BackupRetentionSuccess int      `yaml:"backup_retention_success"`
	BackupRetentionFailure int      `yaml:"backup_retention_failure"`
	VerifyRollback         bool     `yaml:"verify_rollback"`
	Triggers               []string `yaml:"triggers"`
}

// ErrorHandlingConfig contains error handling configurations
type ErrorHandlingConfig struct {
	Retry     RetryConfig     `yaml:"retry"`
	Reporting ReportingConfig `yaml:"reporting"`
}

// RetryConfig contains retry configurations
type RetryConfig struct {
	MaxAttempts     int             `yaml:"max_attempts"`
	BackoffStrategy string          `yaml:"backoff_strategy"`
	InitialDelayMs  int             `yaml:"initial_delay_ms"`
	MaxDelayMs      int             `yaml:"max_delay_ms"`
	Delays          []time.Duration `yaml:"delays"` // 自定义重试延迟
}

// ReportingConfig contains error reporting configurations
type ReportingConfig struct {
	IncludeStackTrace  bool `yaml:"include_stack_trace"`
	IncludeContext     bool `yaml:"include_context"`
	IncludeSuggestions bool `yaml:"include_suggestions"`
}

// DocumentationConfig contains documentation configurations
type DocumentationConfig struct {
	AutoSync             bool   `yaml:"auto_sync"`
	APIDocsPath          string `yaml:"api_docs_path"`
	ComponentDocsPath    string `yaml:"component_docs_path"`
	FeatureDocsPath      string `yaml:"feature_docs_path"`
	TemplatesPath        string `yaml:"templates_path"`
	GenerateExamples     bool   `yaml:"generate_examples"`
	GenerateAPIReference bool   `yaml:"generate_api_reference"`
}

// AntiHallucinationConfig contains anti-hallucination configurations
type AntiHallucinationConfig struct {
	Enabled              bool     `yaml:"enabled"`
	VerifyAPIExists      bool     `yaml:"verify_api_exists"`
	VerifyFunctionExists bool     `yaml:"verify_function_exists"`
	VerifyModuleExists   bool     `yaml:"verify_module_exists"`
	VerifyConfigExists   bool     `yaml:"verify_config_exists"`
	RequestConfirmation  bool     `yaml:"request_confirmation"`
	IndexUpdateTriggers  []string `yaml:"index_update_triggers"`
}

// ArchitectureConfig contains architecture configurations
type ArchitectureConfig struct {
	BackendLayers            []LayerConfig `yaml:"backend_layers"`
	FrontendStructure        []LayerConfig `yaml:"frontend_structure"`
	CheckDependencyDirection bool          `yaml:"check_dependency_direction"`
	AllowCrossLayerCalls     bool          `yaml:"allow_cross_layer_calls"`
}

// LayerConfig represents a layer configuration
type LayerConfig struct {
	Name                string   `yaml:"name"`
	Path                string   `yaml:"path"`
	Description         string   `yaml:"description"`
	AllowedDependencies []string `yaml:"allowed_dependencies"`
}

// SecurityConfig contains security configurations
type SecurityConfig struct {
	CheckHardcodedSecrets  bool     `yaml:"check_hardcoded_secrets"`
	CheckAuthBypass        bool     `yaml:"check_auth_bypass"`
	CheckSQLInjection      bool     `yaml:"check_sql_injection"`
	CheckSensitiveDataLeak bool     `yaml:"check_sensitive_data_leak"`
	SensitivePatterns      []string `yaml:"sensitive_patterns"`
}

// PerformanceConfig contains performance configurations
type PerformanceConfig struct {
	CheckNPlusOne      bool `yaml:"check_n_plus_one"`
	CheckLargeLoops    bool `yaml:"check_large_loops"`
	CheckMemoryLeaks   bool `yaml:"check_memory_leaks"`
	MaxQueryComplexity int  `yaml:"max_query_complexity"`
}

// ConfigLoader loads and manages constitution configuration
type ConfigLoader struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	onChange   []func(*Config)
	stopChan   chan struct{}
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPath string) (*ConfigLoader, error) {
	loader := &ConfigLoader{
		configPath: configPath,
		onChange:   make([]func(*Config), 0),
		stopChan:   make(chan struct{}),
	}

	// Load initial configuration
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	return loader, nil
}

// Load reads and parses the configuration file
func (cl *ConfigLoader) Load() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Read configuration file
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Validate configuration
	if err := cl.validateConfig(&config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	cl.config = &config
	return nil
}

// validateConfig validates the configuration values
func (cl *ConfigLoader) validateConfig(config *Config) error {
	// Validate version
	if config.Version == "" {
		return fmt.Errorf("config version is required")
	}

	// Validate trace directory
	if config.Trace.Directory == "" {
		return fmt.Errorf("trace directory is required")
	}

	// Validate trace format
	if config.Trace.Format != "json" {
		return fmt.Errorf("trace format must be 'json', got: %s", config.Trace.Format)
	}

	// Validate retention days
	if config.Trace.RetentionDays < 0 {
		return fmt.Errorf("trace retention days must be non-negative, got: %d", config.Trace.RetentionDays)
	}

	// Validate rollback backup directory
	if config.Rollback.BackupBeforeChange && config.Rollback.BackupDirectory == "" {
		return fmt.Errorf("backup directory is required when backup_before_change is enabled")
	}

	// Validate retry configuration
	if config.ErrorHandling.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry max attempts must be at least 1, got: %d", config.ErrorHandling.Retry.MaxAttempts)
	}

	// Validate backoff strategy
	validStrategies := map[string]bool{"exponential": true, "linear": true, "constant": true}
	if !validStrategies[config.ErrorHandling.Retry.BackoffStrategy] {
		return fmt.Errorf("invalid backoff strategy: %s", config.ErrorHandling.Retry.BackoffStrategy)
	}

	// Validate tool commands
	if err := cl.validateToolCommand("go.formatter", config.Tools.Go.Formatter); err != nil {
		return err
	}
	if err := cl.validateToolCommand("go.linter", config.Tools.Go.Linter); err != nil {
		return err
	}
	if err := cl.validateToolCommand("go.test_runner", config.Tools.Go.TestRunner); err != nil {
		return err
	}

	// Validate complexity limits
	if config.Validation.Go.MaxComplexity < 1 {
		return fmt.Errorf("go max complexity must be at least 1, got: %d", config.Validation.Go.MaxComplexity)
	}

	if config.Performance.MaxQueryComplexity < 1 {
		return fmt.Errorf("max query complexity must be at least 1, got: %d", config.Performance.MaxQueryComplexity)
	}

	return nil
}

// validateToolCommand validates a tool command configuration
func (cl *ConfigLoader) validateToolCommand(name string, cmd ToolCommand) error {
	if cmd.Command == "" {
		return fmt.Errorf("tool command for %s is required", name)
	}
	if cmd.Timeout < 0 {
		return fmt.Errorf("tool timeout for %s must be non-negative, got: %d", name, cmd.Timeout)
	}
	return nil
}

// Get returns the current configuration (thread-safe)
func (cl *ConfigLoader) Get() *Config {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.config
}

// StartWatching starts watching the configuration file for changes
func (cl *ConfigLoader) StartWatching() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	cl.watcher = watcher

	// Watch the config file
	if err := watcher.Add(cl.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// Watch the config directory (for atomic writes)
	configDir := filepath.Dir(cl.configPath)
	if err := watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch config directory: %w", err)
	}

	// Start watching in a goroutine
	go cl.watchLoop()

	return nil
}

// watchLoop monitors file system events and reloads configuration
func (cl *ConfigLoader) watchLoop() {
	// Debounce timer to avoid multiple reloads for rapid changes
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-cl.watcher.Events:
			if !ok {
				return
			}

			// Check if the event is for our config file
			if event.Name != cl.configPath {
				continue
			}

			// Only reload on write or create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDuration, func() {
					if err := cl.reload(); err != nil {
						// Log error but don't stop watching
						fmt.Fprintf(os.Stderr, "Failed to reload config: %v\n", err)
					}
				})
			}

		case err, ok := <-cl.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Config watcher error: %v\n", err)

		case <-cl.stopChan:
			return
		}
	}
}

// reload reloads the configuration and notifies listeners
func (cl *ConfigLoader) reload() error {
	// Load new configuration
	if err := cl.Load(); err != nil {
		return err
	}

	// Notify all listeners
	cl.mu.RLock()
	config := cl.config
	listeners := cl.onChange
	cl.mu.RUnlock()

	for _, listener := range listeners {
		listener(config)
	}

	return nil
}

// OnChange registers a callback to be called when configuration changes
func (cl *ConfigLoader) OnChange(callback func(*Config)) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.onChange = append(cl.onChange, callback)
}

// StopWatching stops watching the configuration file
func (cl *ConfigLoader) StopWatching() error {
	close(cl.stopChan)

	if cl.watcher != nil {
		return cl.watcher.Close()
	}

	return nil
}

// GetToolCommand returns a tool command configuration
func (c *Config) GetToolCommand(tool string) (*ToolCommand, error) {
	switch tool {
	case "go.formatter":
		return &c.Tools.Go.Formatter, nil
	case "go.linter":
		return &c.Tools.Go.Linter, nil
	case "go.test_runner":
		return &c.Tools.Go.TestRunner, nil
	case "vue.linter":
		return &c.Tools.Vue.Linter, nil
	case "vue.type_checker":
		return &c.Tools.Vue.TypeChecker, nil
	case "vue.formatter":
		return &c.Tools.Vue.Formatter, nil
	case "protobuf.compiler":
		return &c.Tools.Protobuf.Compiler, nil
	case "ent.generator":
		return &c.Tools.Ent.Generator, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}

// GetTraceDirectory returns the absolute path to the trace directory
func (c *Config) GetTraceDirectory() string {
	if filepath.IsAbs(c.Trace.Directory) {
		return c.Trace.Directory
	}
	// Relative to project root (.ai/traces)
	return c.Trace.Directory
}

// GetBackupDirectory returns the absolute path to the backup directory
func (c *Config) GetBackupDirectory() string {
	if filepath.IsAbs(c.Rollback.BackupDirectory) {
		return c.Rollback.BackupDirectory
	}
	// Relative to project root (.ai/backups)
	return c.Rollback.BackupDirectory
}

// IsRollbackTrigger checks if a trigger should cause a rollback
func (c *Config) IsRollbackTrigger(trigger string) bool {
	for _, t := range c.Rollback.Triggers {
		if t == trigger {
			return true
		}
	}
	return false
}

// GetRetryDelay calculates the retry delay based on attempt number
func (c *Config) GetRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	initialDelay := time.Duration(c.ErrorHandling.Retry.InitialDelayMs) * time.Millisecond
	maxDelay := time.Duration(c.ErrorHandling.Retry.MaxDelayMs) * time.Millisecond

	var delay time.Duration
	switch c.ErrorHandling.Retry.BackoffStrategy {
	case "exponential":
		delay = initialDelay * time.Duration(1<<uint(attempt-1))
	case "linear":
		delay = initialDelay * time.Duration(attempt)
	case "constant":
		delay = initialDelay
	default:
		delay = initialDelay
	}

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// LoadConfig is a convenience function to load configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	loader, err := NewConfigLoader(configPath)
	if err != nil {
		return nil, err
	}
	return loader.Get(), nil
}
