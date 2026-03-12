package constitution

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// ChangeDetector 变更检测器
type ChangeDetector struct{}

// NewChangeDetector 创建变更检测器
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{}
}

// Detect 检测文档变更
func (d *ChangeDetector) Detect(filePath, newContent string) (*DocumentationDiff, error) {
	diff := &DocumentationDiff{
		FilePath:         filePath,
		Changed:          false,
		AddedLines:       []string{},
		RemovedLines:     []string{},
		ModifiedSections: []string{},
		Timestamp:        time.Now(),
	}

	// 读取旧文件内容
	oldContent, err := d.readFile(filePath)
	if err != nil {
		// 文件不存在，视为新文件
		if os.IsNotExist(err) {
			diff.Changed = true
			diff.NewHash = d.calculateHash(newContent)
			diff.AddedLines = strings.Split(newContent, "\n")
			return diff, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 计算哈希
	oldHash := d.calculateHash(oldContent)
	newHash := d.calculateHash(newContent)

	diff.OldHash = oldHash
	diff.NewHash = newHash

	// 如果哈希相同，没有变更
	if oldHash == newHash {
		return diff, nil
	}

	diff.Changed = true

	// 计算行级差异
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// 简单的行级 diff（实际应该使用更复杂的算法）
	oldLineMap := make(map[string]bool)
	for _, line := range oldLines {
		oldLineMap[line] = true
	}

	newLineMap := make(map[string]bool)
	for _, line := range newLines {
		newLineMap[line] = true
	}

	// 找出新增的行
	for _, line := range newLines {
		if !oldLineMap[line] && strings.TrimSpace(line) != "" {
			diff.AddedLines = append(diff.AddedLines, line)
		}
	}

	// 找出删除的行
	for _, line := range oldLines {
		if !newLineMap[line] && strings.TrimSpace(line) != "" {
			diff.RemovedLines = append(diff.RemovedLines, line)
		}
	}

	// 检测修改的章节
	diff.ModifiedSections = d.detectModifiedSections(oldContent, newContent)

	return diff, nil
}

func (d *ChangeDetector) readFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (d *ChangeDetector) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func (d *ChangeDetector) detectModifiedSections(oldContent, newContent string) []string {
	sections := []string{}

	// 检测 Markdown 标题变更
	oldSections := d.extractSections(oldContent)
	newSections := d.extractSections(newContent)

	for section := range newSections {
		if _, exists := oldSections[section]; !exists {
			sections = append(sections, section)
		}
	}

	return sections
}

func (d *ChangeDetector) extractSections(content string) map[string]bool {
	sections := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			sections[line] = true
		}
	}

	return sections
}
