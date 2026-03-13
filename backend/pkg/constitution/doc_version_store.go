package constitution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// VersionStore 版本存储
type VersionStore struct {
	storePath string
	mu        sync.RWMutex
}

// NewVersionStore 创建版本存储
func NewVersionStore() *VersionStore {
	return &VersionStore{
		storePath: ".ai/doc-versions",
	}
}

// Save 保存文档版本
func (s *VersionStore) Save(version *DocumentationVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 创建存储目录
	if err := os.MkdirAll(s.storePath, 0755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// 生成版本文件路径
	versionFile := s.getVersionFilePath(version.DocPath, version.Version)

	// 序列化版本数据
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// Get 获取最新版本
func (s *VersionStore) Get(docPath string) (*DocumentationVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions, err := s.List(docPath)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for %s", docPath)
	}

	return versions[0], nil
}

// List 列出所有版本
func (s *VersionStore) List(docPath string) ([]*DocumentationVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 获取文档的版本目录
	docHash := s.hashDocPath(docPath)
	versionDir := filepath.Join(s.storePath, docHash)

	// 检查目录是否存在
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return []*DocumentationVersion{}, nil
	}

	// 读取所有版本文件
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read version directory: %w", err)
	}

	versions := []*DocumentationVersion{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		versionFile := filepath.Join(versionDir, entry.Name())
		data, err := os.ReadFile(versionFile)
		if err != nil {
			continue
		}

		var version DocumentationVersion
		if err := json.Unmarshal(data, &version); err != nil {
			continue
		}

		versions = append(versions, &version)
	}

	// 按创建时间倒序排序
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	return versions, nil
}

// GetVersion 获取指定版本
func (s *VersionStore) GetVersion(docPath, version string) (*DocumentationVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versionFile := s.getVersionFilePath(docPath, version)

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	var docVersion DocumentationVersion
	if err := json.Unmarshal(data, &docVersion); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &docVersion, nil
}

// Delete 删除版本
func (s *VersionStore) Delete(docPath, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionFile := s.getVersionFilePath(docPath, version)
	return os.Remove(versionFile)
}

// Cleanup 清理旧版本（保留最近 N 个版本）
func (s *VersionStore) Cleanup(docPath string, keepCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.List(docPath)
	if err != nil {
		return err
	}

	// 如果版本数量不超过保留数量，不需要清理
	if len(versions) <= keepCount {
		return nil
	}

	// 删除多余的旧版本
	for i := keepCount; i < len(versions); i++ {
		versionFile := s.getVersionFilePath(docPath, versions[i].Version)
		if err := os.Remove(versionFile); err != nil {
			return fmt.Errorf("failed to delete version %s: %w", versions[i].Version, err)
		}
	}

	return nil
}

func (s *VersionStore) getVersionFilePath(docPath, version string) string {
	docHash := s.hashDocPath(docPath)
	return filepath.Join(s.storePath, docHash, fmt.Sprintf("%s.json", version))
}

func (s *VersionStore) hashDocPath(docPath string) string {
	// 简单的路径哈希（实际应该使用更好的哈希算法）
	return strings.ReplaceAll(strings.ReplaceAll(docPath, "/", "_"), ".", "_")
}
