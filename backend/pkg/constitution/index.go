package constitution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexDatabase represents a persistent index database
type IndexDatabase struct {
	// Index data
	APIIndex    map[string]*APIReference      `json:"api_index"`
	FuncIndex   map[string]*FunctionSignature `json:"func_index"`
	ModuleIndex map[string]bool               `json:"module_index"`
	ConfigIndex map[string]bool               `json:"config_index"`

	// Metadata
	LastUpdated time.Time `json:"last_updated"`
	Version     string    `json:"version"`

	// File path for persistence
	filePath string
	mu       sync.RWMutex
}

// NewIndexDatabase creates a new index database
func NewIndexDatabase(filePath string) (*IndexDatabase, error) {
	db := &IndexDatabase{
		APIIndex:    make(map[string]*APIReference),
		FuncIndex:   make(map[string]*FunctionSignature),
		ModuleIndex: make(map[string]bool),
		ConfigIndex: make(map[string]bool),
		Version:     "1.0.0",
		filePath:    filePath,
	}

	// Try to load existing index
	if err := db.Load(); err != nil {
		// If file doesn't exist, that's okay
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load index: %w", err)
		}
	}

	return db, nil
}

// Load loads the index from disk
func (db *IndexDatabase) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := os.ReadFile(db.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, db)
}

// Save saves the index to disk
func (db *IndexDatabase) Save() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.LastUpdated = time.Now()

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(db.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tmpFile := db.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	// Rename to final file (atomic operation)
	if err := os.Rename(tmpFile, db.filePath); err != nil {
		return fmt.Errorf("failed to rename index file: %w", err)
	}

	return nil
}

// AddAPIReference adds an API reference to the index
func (db *IndexDatabase) AddAPIReference(key string, ref *APIReference) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.APIIndex[key] = ref
}

// AddFunctionSignature adds a function signature to the index
func (db *IndexDatabase) AddFunctionSignature(key string, sig *FunctionSignature) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.FuncIndex[key] = sig
}

// AddModule adds a module to the index
func (db *IndexDatabase) AddModule(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ModuleIndex[key] = true
}

// AddConfigKey adds a config key to the index
func (db *IndexDatabase) AddConfigKey(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ConfigIndex[key] = true
}

// GetAPIReference retrieves an API reference from the index
func (db *IndexDatabase) GetAPIReference(key string) (*APIReference, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	ref, exists := db.APIIndex[key]
	return ref, exists
}

// GetFunctionSignature retrieves a function signature from the index
func (db *IndexDatabase) GetFunctionSignature(key string) (*FunctionSignature, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sig, exists := db.FuncIndex[key]
	return sig, exists
}

// HasModule checks if a module exists in the index
func (db *IndexDatabase) HasModule(key string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.ModuleIndex[key]
}

// HasConfigKey checks if a config key exists in the index
func (db *IndexDatabase) HasConfigKey(key string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.ConfigIndex[key]
}

// Clear clears all indexes
func (db *IndexDatabase) Clear() {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.APIIndex = make(map[string]*APIReference)
	db.FuncIndex = make(map[string]*FunctionSignature)
	db.ModuleIndex = make(map[string]bool)
	db.ConfigIndex = make(map[string]bool)
}

// Stats returns statistics about the index
func (db *IndexDatabase) Stats() map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return map[string]interface{}{
		"api_count":    len(db.APIIndex),
		"func_count":   len(db.FuncIndex),
		"module_count": len(db.ModuleIndex),
		"config_count": len(db.ConfigIndex),
		"last_updated": db.LastUpdated,
		"version":      db.Version,
	}
}

// IndexWatcher watches for file changes and triggers index updates
type IndexWatcher struct {
	config   *Config
	verifier *antiHallucinationVerifier

	// Watch paths
	watchPaths []string

	// Last modification times
	lastModTimes map[string]time.Time

	// Control channels
	stopCh chan struct{}
	doneCh chan struct{}

	mu sync.RWMutex
}

// NewIndexWatcher creates a new index watcher
func NewIndexWatcher(config *Config, verifier *antiHallucinationVerifier) *IndexWatcher {
	return &IndexWatcher{
		config:       config,
		verifier:     verifier,
		watchPaths:   []string{},
		lastModTimes: make(map[string]time.Time),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

// AddWatchPath adds a path to watch for changes
func (w *IndexWatcher) AddWatchPath(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.watchPaths = append(w.watchPaths, path)
}

// Start starts watching for file changes
func (w *IndexWatcher) Start(interval time.Duration) {
	go w.watch(interval)
}

// Stop stops watching for file changes
func (w *IndexWatcher) Stop() {
	close(w.stopCh)
	<-w.doneCh
}

// watch periodically checks for file changes
func (w *IndexWatcher) watch(interval time.Duration) {
	defer close(w.doneCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if w.checkForChanges() {
				// Rebuild indexes if changes detected
				if err := w.verifier.RebuildIndexes(); err != nil {
					// Log error but continue watching
					fmt.Printf("Failed to rebuild indexes: %v\n", err)
				}
			}
		case <-w.stopCh:
			return
		}
	}
}

// checkForChanges checks if any watched files have changed
func (w *IndexWatcher) checkForChanges() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false

	for _, path := range w.watchPaths {
		// Walk the directory tree
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if info.IsDir() {
				return nil
			}

			// Check if file has been modified
			lastMod, exists := w.lastModTimes[filePath]
			if !exists || info.ModTime().After(lastMod) {
				w.lastModTimes[filePath] = info.ModTime()
				changed = true
			}

			return nil
		})

		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to walk path %s: %v\n", path, err)
		}
	}

	return changed
}

// IndexUpdateTrigger defines triggers for index updates
type IndexUpdateTrigger struct {
	config   *Config
	verifier *antiHallucinationVerifier
	watcher  *IndexWatcher
}

// NewIndexUpdateTrigger creates a new index update trigger
func NewIndexUpdateTrigger(config *Config, verifier *antiHallucinationVerifier) *IndexUpdateTrigger {
	trigger := &IndexUpdateTrigger{
		config:   config,
		verifier: verifier,
		watcher:  NewIndexWatcher(config, verifier),
	}

	// Add default watch paths
	trigger.addDefaultWatchPaths()

	return trigger
}

// addDefaultWatchPaths adds default paths to watch
func (t *IndexUpdateTrigger) addDefaultWatchPaths() {
	// Watch Protobuf files
	protoPath := filepath.Join(t.config.ProjectRoot, "backend", "api", "protos")
	if _, err := os.Stat(protoPath); err == nil {
		t.watcher.AddWatchPath(protoPath)
	}

	// Watch Go source files
	backendPath := filepath.Join(t.config.ProjectRoot, "backend")
	if _, err := os.Stat(backendPath); err == nil {
		t.watcher.AddWatchPath(backendPath)
	}

	// Watch config files
	configPath := filepath.Join(t.config.ProjectRoot, "backend", "app")
	if _, err := os.Stat(configPath); err == nil {
		t.watcher.AddWatchPath(configPath)
	}

	// Watch go.mod
	goModPath := filepath.Join(t.config.ProjectRoot, "backend", "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		t.watcher.AddWatchPath(goModPath)
	}

	// Watch package.json
	packageJSONPath := filepath.Join(t.config.ProjectRoot, "frontend", "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		t.watcher.AddWatchPath(packageJSONPath)
	}
}

// Start starts watching for changes
func (t *IndexUpdateTrigger) Start(interval time.Duration) {
	t.watcher.Start(interval)
}

// Stop stops watching for changes
func (t *IndexUpdateTrigger) Stop() {
	t.watcher.Stop()
}

// TriggerManualUpdate manually triggers an index update
func (t *IndexUpdateTrigger) TriggerManualUpdate() error {
	return t.verifier.RebuildIndexes()
}

// OnProtoFileChange triggers index update when a proto file changes
func (t *IndexUpdateTrigger) OnProtoFileChange(filePath string) error {
	// Rebuild API index
	return t.verifier.RebuildIndexes()
}

// OnGoFileChange triggers index update when a Go file changes
func (t *IndexUpdateTrigger) OnGoFileChange(filePath string) error {
	// Rebuild function index
	return t.verifier.RebuildIndexes()
}

// OnConfigFileChange triggers index update when a config file changes
func (t *IndexUpdateTrigger) OnConfigFileChange(filePath string) error {
	// Rebuild config index
	return t.verifier.RebuildIndexes()
}

// OnDependencyChange triggers index update when dependencies change
func (t *IndexUpdateTrigger) OnDependencyChange(filePath string) error {
	// Rebuild module index
	return t.verifier.RebuildIndexes()
}
