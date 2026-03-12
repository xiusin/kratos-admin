package constitution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewRollbackManager(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	// Check if backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Errorf("Backup directory was not created")
	}
}

func TestCreateBackup(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.txt")
	content1 := []byte("test content 1")
	content2 := []byte("test content 2")

	if err := os.WriteFile(testFile1, content1, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, content2, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	taskID := "test-task-001"
	backupID, err := manager.CreateBackup(taskID, []string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if backupID == "" {
		t.Error("Expected non-empty backup ID")
	}

	// Verify backup was created
	backup, err := manager.GetBackup(backupID)
	if err != nil {
		t.Fatalf("Failed to get backup: %v", err)
	}

	if backup.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, backup.TaskID)
	}

	if len(backup.Files) != 2 {
		t.Errorf("Expected 2 backup files, got %d", len(backup.Files))
	}

	// Verify backup files exist
	for _, backupFile := range backup.Files {
		if backupFile.BackupPath == "" {
			continue
		}
		if _, err := os.Stat(backupFile.BackupPath); os.IsNotExist(err) {
			t.Errorf("Backup file does not exist: %s", backupFile.BackupPath)
		}
	}
}

func TestCreateBackupEmptyTaskID(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	_, err = manager.CreateBackup("", []string{"test.txt"})
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

func TestCreateBackupEmptyFiles(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	_, err = manager.CreateBackup("test-task", []string{})
	if err == nil {
		t.Error("Expected error for empty files list")
	}
}

func TestCreateBackupNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create backup with non-existent file
	taskID := "test-task-002"
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	backupID, err := manager.CreateBackup(taskID, []string{nonExistentFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup was created with empty backup path
	backup, err := manager.GetBackup(backupID)
	if err != nil {
		t.Fatalf("Failed to get backup: %v", err)
	}

	if len(backup.Files) != 1 {
		t.Errorf("Expected 1 backup file, got %d", len(backup.Files))
	}

	if backup.Files[0].BackupPath != "" {
		t.Errorf("Expected empty backup path for non-existent file, got %s", backup.Files[0].BackupPath)
	}
}

func TestRollback(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("original content")
	if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	taskID := "test-task-003"
	_, err = manager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify file
	modifiedContent := []byte("modified content")
	if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Rollback
	if err := manager.Rollback(taskID, "test rollback"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file was restored
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != string(originalContent) {
		t.Errorf("Expected content %s, got %s", originalContent, restoredContent)
	}
}

func TestRollbackEmptyTaskID(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	err = manager.Rollback("", "test")
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

func TestRollbackNoBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	err = manager.Rollback("nonexistent-task", "test")
	if err == nil {
		t.Error("Expected error for task with no backups")
	}
}

func TestRollbackCreatedFile(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create backup with non-existent file
	testFile := filepath.Join(tempDir, "newfile.txt")
	taskID := "test-task-004"
	_, err = manager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Create the file
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Rollback should delete the file
	if err := manager.Rollback(taskID, "test rollback"); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted")
	}
}

func TestListBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create multiple backups
	taskID := "test-task-005"
	_, err = manager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup 1: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = manager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup 2: %v", err)
	}

	// List backups
	backups, err := manager.ListBackups(taskID)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}
}

func TestListBackupsEmptyTaskID(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	_, err = manager.ListBackups("")
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

func TestListBackupsNonExistentTask(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	backups, err := manager.ListBackups("nonexistent-task")
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(backups))
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	taskID := "test-task-006"
	backupID, err := manager.CreateBackup(taskID, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Manually modify backup timestamp to be old
	backup, err := manager.GetBackup(backupID)
	if err != nil {
		t.Fatalf("Failed to get backup: %v", err)
	}

	backup.Timestamp = time.Now().AddDate(0, 0, -100)
	taskBackupDir := filepath.Join(backupDir, taskID)
	metadataPath := filepath.Join(taskBackupDir, backupID+".json")
	rm := manager.(*rollbackManager)
	if err := rm.saveBackupMetadata(metadataPath, backup); err != nil {
		t.Fatalf("Failed to save modified backup: %v", err)
	}

	// Cleanup old backups
	if err := manager.CleanupOldBackups(90); err != nil {
		t.Fatalf("Failed to cleanup old backups: %v", err)
	}

	// Verify backup was deleted
	_, err = manager.GetBackup(backupID)
	if err == nil {
		t.Error("Expected backup to be deleted")
	}
}

func TestCleanupOldBackupsInvalidRetention(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	traceDir := filepath.Join(tempDir, "traces")

	manager, err := NewRollbackManager(backupDir, traceDir)
	if err != nil {
		t.Fatalf("Failed to create rollback manager: %v", err)
	}

	err = manager.CleanupOldBackups(0)
	if err == nil {
		t.Error("Expected error for invalid retention days")
	}

	err = manager.CleanupOldBackups(-1)
	if err == nil {
		t.Error("Expected error for negative retention days")
	}
}

func TestCalculateHash(t *testing.T) {
	content := []byte("test content")
	hash1 := calculateHash(content)
	hash2 := calculateHash(content)

	if hash1 != hash2 {
		t.Error("Expected same hash for same content")
	}

	differentContent := []byte("different content")
	hash3 := calculateHash(differentContent)

	if hash1 == hash3 {
		t.Error("Expected different hash for different content")
	}
}
