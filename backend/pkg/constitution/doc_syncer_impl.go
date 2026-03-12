package constitution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// documentationSyncer 文档同步器实现
type documentationSyncer struct {
	config         *Config
	templateDir    string
	outputDir      string
	protoParser    ProtoParser
	vueParser      VueParser
	searchIndex    *SearchIndex
	versionStore   *VersionStore
	changeDetector *ChangeDetector
	mu             sync.RWMutex
}

// NewDocumentationSyncer 创建文档同步器
func NewDocumentationSyncer(config *Config) DocumentationSyncer {
	return &documentationSyncer{
		config:         config,
		templateDir:    ".ai/templates",
		outputDir:      "docs",
		protoParser:    NewProtoParser(),
		vueParser:      NewVueParser(),
		searchIndex:    NewSearchIndex(),
		versionStore:   NewVersionStore(),
		changeDetector: NewChangeDetector(),
	}
}

// SyncAPIDocumentation 同步 API 文档
func (s *documentationSyncer) SyncAPIDocumentation(ctx context.Context, protoFile string) (*DocumentationResult, error) {
	// 解析 Protobuf 文件
	apiDoc, err := s.protoParser.Parse(protoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto file: %w", err)
	}

	// 添加源代码链接
	apiDoc.SourceFile = protoFile
	apiDoc.SourceLink = s.generateSourceLink(protoFile)

	// 加载模板
	tmpl, err := s.loadTemplate("api-doc.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// 生成文档内容
	var content strings.Builder
	if err := tmpl.Execute(&content, apiDoc); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// 计算内容哈希
	hash := s.calculateHash(content.String())

	// 检测变更
	outputPath := s.getAPIDocPath(apiDoc.ServiceName)
	diff, err := s.changeDetector.Detect(outputPath, content.String())
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	// 写入文档文件
	if diff.Changed {
		if err := s.writeDocumentation(outputPath, content.String()); err != nil {
			return nil, fmt.Errorf("failed to write documentation: %w", err)
		}

		// 保存版本
		version := &DocumentationVersion{
			Version:   s.generateVersion(),
			DocPath:   outputPath,
			Content:   content.String(),
			Hash:      hash,
			CreatedAt: time.Now(),
			Author:    "constitution-syncer",
			Message:   fmt.Sprintf("Update API documentation for %s", apiDoc.ServiceName),
		}
		if err := s.versionStore.Save(version); err != nil {
			return nil, fmt.Errorf("failed to save version: %w", err)
		}

		// 更新搜索索引
		if err := s.searchIndex.Index(outputPath, content.String(), apiDoc); err != nil {
			return nil, fmt.Errorf("failed to index documentation: %w", err)
		}
	}

	return &DocumentationResult{
		FilePath:       outputPath,
		Content:        content.String(),
		SourceFile:     protoFile,
		SourceLink:     apiDoc.SourceLink,
		GeneratedAt:    time.Now(),
		Hash:           hash,
		Changed:        diff.Changed,
		PreviousHash:   diff.OldHash,
		ChangesSummary: s.summarizeChanges(diff),
	}, nil
}

// SyncComponentDocumentation 同步组件文档
func (s *documentationSyncer) SyncComponentDocumentation(ctx context.Context, componentFile string) (*DocumentationResult, error) {
	// 解析 Vue 组件
	compDoc, err := s.vueParser.Parse(componentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse component file: %w", err)
	}

	// 添加源代码链接
	compDoc.SourceFile = componentFile
	compDoc.SourceLink = s.generateSourceLink(componentFile)

	// 加载模板
	tmpl, err := s.loadTemplate("component-doc.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// 生成文档内容
	var content strings.Builder
	if err := tmpl.Execute(&content, compDoc); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// 计算内容哈希
	hash := s.calculateHash(content.String())

	// 检测变更
	outputPath := s.getComponentDocPath(compDoc.Name)
	diff, err := s.changeDetector.Detect(outputPath, content.String())
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	// 写入文档文件
	if diff.Changed {
		if err := s.writeDocumentation(outputPath, content.String()); err != nil {
			return nil, fmt.Errorf("failed to write documentation: %w", err)
		}

		// 保存版本
		version := &DocumentationVersion{
			Version:   s.generateVersion(),
			DocPath:   outputPath,
			Content:   content.String(),
			Hash:      hash,
			CreatedAt: time.Now(),
			Author:    "constitution-syncer",
			Message:   fmt.Sprintf("Update component documentation for %s", compDoc.Name),
		}
		if err := s.versionStore.Save(version); err != nil {
			return nil, fmt.Errorf("failed to save version: %w", err)
		}

		// 更新搜索索引
		if err := s.searchIndex.Index(outputPath, content.String(), compDoc); err != nil {
			return nil, fmt.Errorf("failed to index documentation: %w", err)
		}
	}

	return &DocumentationResult{
		FilePath:       outputPath,
		Content:        content.String(),
		SourceFile:     componentFile,
		SourceLink:     compDoc.SourceLink,
		GeneratedAt:    time.Now(),
		Hash:           hash,
		Changed:        diff.Changed,
		PreviousHash:   diff.OldHash,
		ChangesSummary: s.summarizeChanges(diff),
	}, nil
}

// SyncFeatureDocumentation 同步功能文档
func (s *documentationSyncer) SyncFeatureDocumentation(ctx context.Context, featureName string, changes []CodeChange) (*DocumentationResult, error) {
	// 加载模板
	tmpl, err := s.loadTemplate("feature-doc.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// 准备模板数据
	data := map[string]interface{}{
		"FeatureName": featureName,
		"Changes":     changes,
		"GeneratedAt": time.Now(),
	}

	// 生成文档内容
	var content strings.Builder
	if err := tmpl.Execute(&content, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// 计算内容哈希
	hash := s.calculateHash(content.String())

	// 检测变更
	outputPath := s.getFeatureDocPath(featureName)
	diff, err := s.changeDetector.Detect(outputPath, content.String())
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	// 写入文档文件
	if diff.Changed {
		if err := s.writeDocumentation(outputPath, content.String()); err != nil {
			return nil, fmt.Errorf("failed to write documentation: %w", err)
		}

		// 保存版本
		version := &DocumentationVersion{
			Version:   s.generateVersion(),
			DocPath:   outputPath,
			Content:   content.String(),
			Hash:      hash,
			CreatedAt: time.Now(),
			Author:    "constitution-syncer",
			Message:   fmt.Sprintf("Update feature documentation for %s", featureName),
		}
		if err := s.versionStore.Save(version); err != nil {
			return nil, fmt.Errorf("failed to save version: %w", err)
		}
	}

	return &DocumentationResult{
		FilePath:       outputPath,
		Content:        content.String(),
		GeneratedAt:    time.Now(),
		Hash:           hash,
		Changed:        diff.Changed,
		PreviousHash:   diff.OldHash,
		ChangesSummary: s.summarizeChanges(diff),
	}, nil
}

// GenerateAPIReference 生成完整 API 参考文档
func (s *documentationSyncer) GenerateAPIReference(ctx context.Context, outputPath string) error {
	// 查找所有 proto 文件
	protoFiles, err := s.findProtoFiles()
	if err != nil {
		return fmt.Errorf("failed to find proto files: %w", err)
	}

	// 并发生成文档
	var wg sync.WaitGroup
	errChan := make(chan error, len(protoFiles))

	for _, protoFile := range protoFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			if _, err := s.SyncAPIDocumentation(ctx, file); err != nil {
				errChan <- fmt.Errorf("failed to sync %s: %w", file, err)
			}
		}(protoFile)
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateDocumentation 验证文档完整性
func (s *documentationSyncer) ValidateDocumentation(ctx context.Context) (*DocumentationReport, error) {
	report := &DocumentationReport{
		Timestamp: time.Now(),
	}

	// 统计 API 文档
	protoFiles, err := s.findProtoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find proto files: %w", err)
	}
	report.TotalAPIs = len(protoFiles)

	for _, protoFile := range protoFiles {
		apiDoc, err := s.protoParser.Parse(protoFile)
		if err != nil {
			continue
		}
		docPath := s.getAPIDocPath(apiDoc.ServiceName)
		if s.fileExists(docPath) {
			report.DocumentedAPIs++
		} else {
			report.MissingDocs = append(report.MissingDocs, docPath)
		}
	}

	// 统计组件文档
	vueFiles, err := s.findVueFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find vue files: %w", err)
	}
	report.TotalComponents = len(vueFiles)

	for _, vueFile := range vueFiles {
		compDoc, err := s.vueParser.Parse(vueFile)
		if err != nil {
			continue
		}
		docPath := s.getComponentDocPath(compDoc.Name)
		if s.fileExists(docPath) {
			report.DocumentedComponents++
		} else {
			report.MissingDocs = append(report.MissingDocs, docPath)
		}
	}

	// 计算覆盖率
	totalDocs := report.TotalAPIs + report.TotalComponents
	documentedDocs := report.DocumentedAPIs + report.DocumentedComponents
	if totalDocs > 0 {
		report.CoveragePercent = float64(documentedDocs) / float64(totalDocs) * 100
	}

	return report, nil
}

// DetectChanges 检测文档变更
func (s *documentationSyncer) DetectChanges(ctx context.Context, filePath string) (*DocumentationDiff, error) {
	return s.changeDetector.Detect(filePath, "")
}

// BuildSearchIndex 构建文档搜索索引
func (s *documentationSyncer) BuildSearchIndex(ctx context.Context) error {
	return s.searchIndex.Build()
}

// SearchDocumentation 搜索文档
func (s *documentationSyncer) SearchDocumentation(ctx context.Context, query string) ([]*SearchResult, error) {
	return s.searchIndex.Search(query)
}

// GetDocumentationVersion 获取文档版本
func (s *documentationSyncer) GetDocumentationVersion(ctx context.Context, docPath string) (*DocumentationVersion, error) {
	return s.versionStore.Get(docPath)
}

// ListDocumentationVersions 列出文档版本历史
func (s *documentationSyncer) ListDocumentationVersions(ctx context.Context, docPath string) ([]*DocumentationVersion, error) {
	return s.versionStore.List(docPath)
}

// Helper methods

func (s *documentationSyncer) loadTemplate(name string) (*template.Template, error) {
	tmplPath := filepath.Join(s.templateDir, name)
	return template.ParseFiles(tmplPath)
}

func (s *documentationSyncer) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func (s *documentationSyncer) generateSourceLink(filePath string) string {
	// 生成 GitHub/GitLab 源代码链接
	// 这里简化处理，实际应该从 git remote 获取
	return fmt.Sprintf("https://github.com/your-org/your-repo/blob/main/%s", filePath)
}

func (s *documentationSyncer) getAPIDocPath(serviceName string) string {
	return filepath.Join(s.outputDir, "api", fmt.Sprintf("%s.md", strings.ToLower(serviceName)))
}

func (s *documentationSyncer) getComponentDocPath(componentName string) string {
	return filepath.Join(s.outputDir, "components", fmt.Sprintf("%s.md", strings.ToLower(componentName)))
}

func (s *documentationSyncer) getFeatureDocPath(featureName string) string {
	return filepath.Join(s.outputDir, "features", fmt.Sprintf("%s.md", strings.ToLower(featureName)))
}

func (s *documentationSyncer) writeDocumentation(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *documentationSyncer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *documentationSyncer) findProtoFiles() ([]string, error) {
	var files []string
	err := filepath.Walk("backend/api/protos", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (s *documentationSyncer) findVueFiles() ([]string, error) {
	var files []string
	err := filepath.Walk("frontend/apps/admin/src", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".vue") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (s *documentationSyncer) generateVersion() string {
	return fmt.Sprintf("v%s", time.Now().Format("20060102-150405"))
}

func (s *documentationSyncer) summarizeChanges(diff *DocumentationDiff) string {
	if !diff.Changed {
		return "No changes"
	}
	return fmt.Sprintf("Added %d lines, removed %d lines, modified %d sections",
		len(diff.AddedLines), len(diff.RemovedLines), len(diff.ModifiedSections))
}
