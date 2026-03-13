package constitution

import (
	"fmt"
	"path/filepath"
)

// ProviderSet contains all constitution component constructors
// This can be used with dependency injection frameworks like Wire

// NewTaskTraceManagerFromConfig creates a TaskTraceManager from configuration
func NewTaskTraceManagerFromConfig(cfg *Config) (TaskTraceManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	traceDir := cfg.Trace.Directory
	if traceDir == "" {
		traceDir = ".ai/traces"
	}

	// Make path absolute if it's relative
	if !filepath.IsAbs(traceDir) {
		// In production, you might want to resolve this relative to project root
		traceDir = filepath.Clean(traceDir)
	}

	return NewTaskTraceManager(traceDir)
}

// NewCodeValidatorFromConfig creates a CodeValidator from configuration
func NewCodeValidatorFromConfig(cfg *Config) (CodeValidator, error) {
	return NewCodeValidator(cfg)
}

// NewAntiHallucinationVerifierFromConfig creates an AntiHallucinationVerifier from configuration
func NewAntiHallucinationVerifierFromConfig(cfg *Config) (AntiHallucinationVerifier, error) {
	return NewAntiHallucinationVerifier(cfg)
}

// NewIndexDatabaseFromConfig creates an IndexDatabase from configuration
func NewIndexDatabaseFromConfig(cfg *Config) (*IndexDatabase, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	indexFile := filepath.Join(cfg.ProjectRoot, ".ai", "index.json")
	return NewIndexDatabase(indexFile)
}

// NewIndexUpdateTriggerFromConfig creates an IndexUpdateTrigger from configuration
func NewIndexUpdateTriggerFromConfig(cfg *Config, verifier AntiHallucinationVerifier) *IndexUpdateTrigger {
	v := verifier.(*antiHallucinationVerifier)
	return NewIndexUpdateTrigger(cfg, v)
}

// NewRollbackManagerFromConfig creates a RollbackManager from configuration
func NewRollbackManagerFromConfig(cfg *Config) (RollbackManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	backupDir := cfg.Rollback.BackupDirectory
	if backupDir == "" {
		backupDir = ".ai/backups"
	}

	traceDir := cfg.Trace.Directory
	if traceDir == "" {
		traceDir = ".ai/traces"
	}

	// Make paths absolute if they're relative
	if !filepath.IsAbs(backupDir) {
		backupDir = filepath.Clean(backupDir)
	}
	if !filepath.IsAbs(traceDir) {
		traceDir = filepath.Clean(traceDir)
	}

	return NewRollbackManager(backupDir, traceDir)
}

// NewRollbackTriggerFromConfig creates a RollbackTrigger from configuration
func NewRollbackTriggerFromConfig(
	rollbackManager RollbackManager,
	traceManager TaskTraceManager,
	validator CodeValidator,
) *RollbackTrigger {
	return NewRollbackTrigger(rollbackManager, traceManager, validator)
}

// NewDocumentationSyncerFromConfig creates a DocumentationSyncer from configuration
func NewDocumentationSyncerFromConfig(cfg *Config) (DocumentationSyncer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	return NewDocumentationSyncer(cfg), nil
}
