package constitution

import (
	"context"
	"time"
)

// DocumentationSyncer 文档同步器接口
type DocumentationSyncer interface {
	// SyncAPIDocumentation 同步 API 文档（从 .proto 文件提取）
	SyncAPIDocumentation(ctx context.Context, protoFile string) (*DocumentationResult, error)

	// SyncComponentDocumentation 同步组件文档（从 Vue 组件提取）
	SyncComponentDocumentation(ctx context.Context, componentFile string) (*DocumentationResult, error)

	// SyncFeatureDocumentation 同步功能文档
	SyncFeatureDocumentation(ctx context.Context, featureName string, changes []CodeChange) (*DocumentationResult, error)

	// GenerateAPIReference 生成完整 API 参考文档
	GenerateAPIReference(ctx context.Context, outputPath string) error

	// ValidateDocumentation 验证文档完整性
	ValidateDocumentation(ctx context.Context) (*DocumentationReport, error)

	// DetectChanges 检测文档变更
	DetectChanges(ctx context.Context, filePath string) (*DocumentationDiff, error)

	// BuildSearchIndex 构建文档搜索索引
	BuildSearchIndex(ctx context.Context) error

	// SearchDocumentation 搜索文档
	SearchDocumentation(ctx context.Context, query string) ([]*SearchResult, error)

	// GetDocumentationVersion 获取文档版本
	GetDocumentationVersion(ctx context.Context, docPath string) (*DocumentationVersion, error)

	// ListDocumentationVersions 列出文档版本历史
	ListDocumentationVersions(ctx context.Context, docPath string) ([]*DocumentationVersion, error)
}

// DocumentationResult 文档生成结果
type DocumentationResult struct {
	FilePath       string    `json:"file_path"`
	Content        string    `json:"content"`
	SourceFile     string    `json:"source_file"`
	SourceLink     string    `json:"source_link"`
	GeneratedAt    time.Time `json:"generated_at"`
	Hash           string    `json:"hash"`
	Changed        bool      `json:"changed"`
	PreviousHash   string    `json:"previous_hash,omitempty"`
	ChangesSummary string    `json:"changes_summary,omitempty"`
}

// DocumentationReport 文档完整性报告
type DocumentationReport struct {
	TotalAPIs            int       `json:"total_apis"`
	DocumentedAPIs       int       `json:"documented_apis"`
	TotalComponents      int       `json:"total_components"`
	DocumentedComponents int       `json:"documented_components"`
	MissingDocs          []string  `json:"missing_docs"`
	OutdatedDocs         []string  `json:"outdated_docs"`
	InconsistentDocs     []string  `json:"inconsistent_docs"`
	CoveragePercent      float64   `json:"coverage_percent"`
	Timestamp            time.Time `json:"timestamp"`
}

// DocumentationDiff 文档变更差异
type DocumentationDiff struct {
	FilePath         string    `json:"file_path"`
	Changed          bool      `json:"changed"`
	OldHash          string    `json:"old_hash"`
	NewHash          string    `json:"new_hash"`
	AddedLines       []string  `json:"added_lines"`
	RemovedLines     []string  `json:"removed_lines"`
	ModifiedSections []string  `json:"modified_sections"`
	Timestamp        time.Time `json:"timestamp"`
}

// SearchResult 搜索结果
type SearchResult struct {
	DocPath    string  `json:"doc_path"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	SourceFile string  `json:"source_file"`
	SourceLink string  `json:"source_link"`
}

// DocumentationVersion 文档版本
type DocumentationVersion struct {
	Version      string    `json:"version"`
	DocPath      string    `json:"doc_path"`
	Content      string    `json:"content"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"created_at"`
	Author       string    `json:"author"`
	Message      string    `json:"message"`
	SourceCommit string    `json:"source_commit,omitempty"`
}

// APIDocumentation API 文档结构
type APIDocumentation struct {
	ServiceName string        `json:"service_name"`
	Description string        `json:"description"`
	Methods     []*APIMethod  `json:"methods"`
	Messages    []*APIMessage `json:"messages"`
	SourceFile  string        `json:"source_file"`
	SourceLink  string        `json:"source_link"`
	Package     string        `json:"package"`
}

// APIMethod API 方法
type APIMethod struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RequestType  string   `json:"request_type"`
	ResponseType string   `json:"response_type"`
	HTTPMethod   string   `json:"http_method,omitempty"`
	HTTPPath     string   `json:"http_path,omitempty"`
	Examples     []string `json:"examples,omitempty"`
	ErrorCodes   []string `json:"error_codes,omitempty"`
}

// APIMessage API 消息
type APIMessage struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Fields      []*APIField `json:"fields"`
}

// APIField API 字段
type APIField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Repeated    bool   `json:"repeated"`
}

// ComponentDocumentation 组件文档结构
type ComponentDocumentation struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Props       []*ComponentProp  `json:"props"`
	Events      []*ComponentEvent `json:"events"`
	Slots       []*ComponentSlot  `json:"slots"`
	Examples    []string          `json:"examples,omitempty"`
	SourceFile  string            `json:"source_file"`
	SourceLink  string            `json:"source_link"`
}

// ComponentProp 组件属性
type ComponentProp struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// ComponentEvent 组件事件
type ComponentEvent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Payload     string `json:"payload,omitempty"`
}

// ComponentSlot 组件插槽
type ComponentSlot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Props       string `json:"props,omitempty"`
}
