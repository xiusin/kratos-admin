package constitution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// rollbackManager implements the RollbackManager interface
type rollbackManager struct {
	backupDir string
	traceDir  string
}

// NewRollbackManager creates a new RollbackManager instance
func NewRollbackManager(backupDir, traceDir string) (RollbackManager, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &rollbackManager{
		backupDir: backupDir,
		traceDir:  traceDir,
	}, nil
}

// CreateBackup creates a backup of files before modification
func (r *rollbackManager) CreateBackup(taskID string, files []string) (string, error) {
	if taskID == "" {
		return "", fmt.Errorf("taskID cannot be empty")
	}

	if len(files) == 0 {
		return "", fmt.Errorf("files list cannot be empty")
	}

	// Generate backup ID
	backupID := uuid.New().String()
	timestamp := time.Now()

	// Create backup directory for this task
	taskBackupDir := filepath.Join(r.backupDir, taskID)
	if err := os.MkdirAll(taskBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create task backup directory: %w", err)
	}

	// Backup each file
	backupFiles := make([]BackupFile, 0, len(files))
	for _, filePath := range files {
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// File doesn't exist, skip backup but record it
			backupFiles = append(backupFiles, BackupFile{
				OriginalPath: filePath,
				BackupPath:   "",
				Hash:         "",
			})
			continue
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		// Calculate hash
		hash := calculateHash(content)

		// Create backup file path
		backupFileName := fmt.Sprintf("%s_%s", backupID, filepath.Base(filePath))
		backupPath := filepath.Join(taskBackupDir, backupFileName)

		// Create backup file directory if needed
		backupFileDir := filepath.Dir(backupPath)
		if err := os.MkdirAll(backupFileDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create backup file directory: %w", err)
		}

		// Write backup file
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return "", fmt.Errorf("failed to write backup file %s: %w", backupPath, err)
		}

		backupFiles = append(backupFiles, BackupFile{
			OriginalPath: filePath,
			BackupPath:   backupPath,
			Hash:         hash,
		})
	}

	// Create backup metadata
	backup := Backup{
		BackupID:  backupID,
		TaskID:    taskID,
		Timestamp: timestamp,
		Files:     backupFiles,
	}

	// Save backup metadata
	metadataPath := filepath.Join(taskBackupDir, fmt.Sprintf("%s.json", backupID))
	if err := r.saveBackupMetadata(metadataPath, &backup); err != nil {
		return "", fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return backupID, nil
}

// Rollback rolls back changes for a task
func (r *rollbackManager) Rollback(taskID string, reason string) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}

	// Get all backups for this task
	backups, err := r.ListBackups(taskID)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found for task %s", taskID)
	}

	// Use the most recent backup
	backup := backups[len(backups)-1]

	// Restore each file
	restoredFiles := make([]string, 0, len(backup.Files))
	conflicts := make([]string, 0)

	for _, backupFile := range backup.Files {
		// If backup path is empty, the file didn't exist originally
		if backupFile.BackupPath == "" {
			// Delete the file if it was created
			if _, err := os.Stat(backupFile.OriginalPath); err == nil {
				if err := os.Remove(backupFile.OriginalPath); err != nil {
					return fmt.Errorf("failed to remove created file %s: %w", backupFile.OriginalPath, err)
				}
				restoredFiles = append(restoredFiles, backupFile.OriginalPath)
			}
			continue
		}

		// Check if current file has been modified since backup
		if _, err := os.Stat(backupFile.OriginalPath); err == nil {
			currentContent, err := os.ReadFile(backupFile.OriginalPath)
			if err != nil {
				return fmt.Errorf("failed to read current file %s: %w", backupFile.OriginalPath, err)
			}

			currentHash := calculateHash(currentContent)
			if currentHash != backupFile.Hash {
				// File has been modified, check if it's different from backup
				backupContent, err := os.ReadFile(backupFile.BackupPath)
				if err != nil {
					return fmt.Errorf("failed to read backup file %s: %w", backupFile.BackupPath, err)
				}

				backupHash := calculateHash(backupContent)
				if currentHash != backupHash {
					conflicts = append(conflicts, backupFile.OriginalPath)
				}
			}
		}

		// Restore file from backup
		backupContent, err := os.ReadFile(backupFile.BackupPath)
		if err != nil {
			return fmt.Errorf("failed to read backup file %s: %w", backupFile.BackupPath, err)
		}

		// Create directory if needed
		originalDir := filepath.Dir(backupFile.OriginalPath)
		if err := os.MkdirAll(originalDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", originalDir, err)
		}

		// Write restored file
		if err := os.WriteFile(backupFile.OriginalPath, backupContent, 0644); err != nil {
			return fmt.Errorf("failed to restore file %s: %w", backupFile.OriginalPath, err)
		}

		restoredFiles = append(restoredFiles, backupFile.OriginalPath)
	}

	// Record rollback in task trace if trace manager is available
	if r.traceDir != "" {
		rollbackInfo := RollbackInfo{
			Triggered:     true,
			Reason:        reason,
			Timestamp:     time.Now(),
			RestoredFiles: restoredFiles,
		}

		// Load task trace
		taskTrace, err := r.loadTaskTrace(taskID)
		if err == nil {
			taskTrace.Rollback = &rollbackInfo
			taskTrace.Status = TaskStatusRolledBack
			if err := r.saveTaskTrace(taskID, taskTrace); err != nil {
				// Log error but don't fail rollback
				fmt.Printf("Warning: failed to update task trace: %v\n", err)
			}
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("rollback completed with conflicts in files: %v", conflicts)
	}

	return nil
}

// GetBackup retrieves backup information
func (r *rollbackManager) GetBackup(backupID string) (*Backup, error) {
	if backupID == "" {
		return nil, fmt.Errorf("backupID cannot be empty")
	}

	// Search for backup metadata in all task directories
	taskDirs, err := os.ReadDir(r.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, taskDir := range taskDirs {
		if !taskDir.IsDir() {
			continue
		}

		metadataPath := filepath.Join(r.backupDir, taskDir.Name(), fmt.Sprintf("%s.json", backupID))
		if _, err := os.Stat(metadataPath); err == nil {
			return r.loadBackupMetadata(metadataPath)
		}
	}

	return nil, fmt.Errorf("backup %s not found", backupID)
}

// ListBackups lists backups for a task
func (r *rollbackManager) ListBackups(taskID string) ([]*Backup, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	taskBackupDir := filepath.Join(r.backupDir, taskID)
	if _, err := os.Stat(taskBackupDir); os.IsNotExist(err) {
		return []*Backup{}, nil
	}

	files, err := os.ReadDir(taskBackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read task backup directory: %w", err)
	}

	backups := make([]*Backup, 0)
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		metadataPath := filepath.Join(taskBackupDir, file.Name())
		backup, err := r.loadBackupMetadata(metadataPath)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to load backup metadata %s: %v\n", metadataPath, err)
			continue
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

// CleanupOldBackups removes backups older than retention period
func (r *rollbackManager) CleanupOldBackups(retentionDays int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retentionDays must be positive")
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	taskDirs, err := os.ReadDir(r.backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, taskDir := range taskDirs {
		if !taskDir.IsDir() {
			continue
		}

		taskBackupDir := filepath.Join(r.backupDir, taskDir.Name())
		files, err := os.ReadDir(taskBackupDir)
		if err != nil {
			fmt.Printf("Warning: failed to read task backup directory %s: %v\n", taskBackupDir, err)
			continue
		}

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
				continue
			}

			metadataPath := filepath.Join(taskBackupDir, file.Name())
			backup, err := r.loadBackupMetadata(metadataPath)
			if err != nil {
				fmt.Printf("Warning: failed to load backup metadata %s: %v\n", metadataPath, err)
				continue
			}

			if backup.Timestamp.Before(cutoffTime) {
				// Delete backup files
				for _, backupFile := range backup.Files {
					if backupFile.BackupPath != "" {
						if err := os.Remove(backupFile.BackupPath); err != nil {
							fmt.Printf("Warning: failed to remove backup file %s: %v\n", backupFile.BackupPath, err)
						}
					}
				}

				// Delete metadata file
				if err := os.Remove(metadataPath); err != nil {
					fmt.Printf("Warning: failed to remove metadata file %s: %v\n", metadataPath, err)
				}
			}
		}

		// Remove empty task backup directory
		files, _ = os.ReadDir(taskBackupDir)
		if len(files) == 0 {
			os.Remove(taskBackupDir)
		}
	}

	return nil
}

// Helper functions

func calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func (r *rollbackManager) saveBackupMetadata(path string, backup *Backup) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup metadata: %w", err)
	}

	return nil
}

func (r *rollbackManager) loadBackupMetadata(path string) (*Backup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup metadata: %w", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup metadata: %w", err)
	}

	return &backup, nil
}

func (r *rollbackManager) loadTaskTrace(taskID string) (*TaskTrace, error) {
	// Find task trace file
	files, err := os.ReadDir(r.traceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(r.traceDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var trace TaskTrace
		if err := json.Unmarshal(data, &trace); err != nil {
			continue
		}

		if trace.TaskID == taskID {
			return &trace, nil
		}
	}

	return nil, fmt.Errorf("task trace not found for task %s", taskID)
}

func (r *rollbackManager) saveTaskTrace(taskID string, trace *TaskTrace) error {
	// Find task trace file
	files, err := os.ReadDir(r.traceDir)
	if err != nil {
		return fmt.Errorf("failed to read trace directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(r.traceDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var existingTrace TaskTrace
		if err := json.Unmarshal(data, &existingTrace); err != nil {
			continue
		}

		if existingTrace.TaskID == taskID {
			// Update trace file
			data, err := json.MarshalIndent(trace, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal task trace: %w", err)
			}

			if err := os.WriteFile(path, data, 0644); err != nil {
				return fmt.Errorf("failed to write task trace: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("task trace file not found for task %s", taskID)
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
