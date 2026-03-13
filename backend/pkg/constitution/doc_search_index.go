package constitution

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// SearchIndex 搜索索引
type SearchIndex struct {
	index map[string]*IndexEntry
	mu    sync.RWMutex
}

// IndexEntry 索引条目
type IndexEntry struct {
	DocPath    string
	Title      string
	Content    string
	SourceFile string
	SourceLink string
	Keywords   []string
}

// NewSearchIndex 创建搜索索引
func NewSearchIndex() *SearchIndex {
	return &SearchIndex{
		index: make(map[string]*IndexEntry),
	}
}

// Index 索引文档
func (s *SearchIndex) Index(docPath, content string, metadata interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &IndexEntry{
		DocPath:  docPath,
		Content:  content,
		Keywords: s.extractKeywords(content),
	}

	// 根据元数据类型提取信息
	switch meta := metadata.(type) {
	case *APIDocumentation:
		entry.Title = meta.ServiceName
		entry.SourceFile = meta.SourceFile
		entry.SourceLink = meta.SourceLink
	case *ComponentDocumentation:
		entry.Title = meta.Name
		entry.SourceFile = meta.SourceFile
		entry.SourceLink = meta.SourceLink
	}

	s.index[docPath] = entry
	return nil
}

// Build 构建索引
func (s *SearchIndex) Build() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 遍历文档目录
	return filepath.Walk("docs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// 读取文档内容
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// 提取标题
		title := s.extractTitle(string(content))

		entry := &IndexEntry{
			DocPath:  path,
			Title:    title,
			Content:  string(content),
			Keywords: s.extractKeywords(string(content)),
		}

		s.index[path] = entry
		return nil
	})
}

// Search 搜索文档
func (s *SearchIndex) Search(query string) ([]*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	results := []*SearchResult{}

	for _, entry := range s.index {
		score := s.calculateScore(entry, queryWords)
		if score > 0 {
			snippet := s.extractSnippet(entry.Content, queryWords)
			result := &SearchResult{
				DocPath:    entry.DocPath,
				Title:      entry.Title,
				Snippet:    snippet,
				Score:      score,
				SourceFile: entry.SourceFile,
				SourceLink: entry.SourceLink,
			}
			results = append(results, result)
		}
	}

	// 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 返回前 20 个结果
	if len(results) > 20 {
		results = results[:20]
	}

	return results, nil
}

func (s *SearchIndex) extractKeywords(content string) []string {
	// 简单的关键词提取（实际应该使用更复杂的算法）
	words := strings.Fields(strings.ToLower(content))
	keywordMap := make(map[string]int)

	for _, word := range words {
		// 过滤短词和常见词
		if len(word) < 3 {
			continue
		}
		if s.isStopWord(word) {
			continue
		}
		keywordMap[word]++
	}

	// 按频率排序
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range keywordMap {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// 返回前 20 个关键词
	keywords := []string{}
	for i := 0; i < len(sorted) && i < 20; i++ {
		keywords = append(keywords, sorted[i].Key)
	}

	return keywords
}

func (s *SearchIndex) extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}

func (s *SearchIndex) calculateScore(entry *IndexEntry, queryWords []string) float64 {
	score := 0.0
	contentLower := strings.ToLower(entry.Content)
	titleLower := strings.ToLower(entry.Title)

	for _, word := range queryWords {
		// 标题匹配权重更高
		if strings.Contains(titleLower, word) {
			score += 10.0
		}

		// 内容匹配
		count := strings.Count(contentLower, word)
		score += float64(count)

		// 关键词匹配
		for _, keyword := range entry.Keywords {
			if strings.Contains(keyword, word) {
				score += 2.0
			}
		}
	}

	return score
}

func (s *SearchIndex) extractSnippet(content string, queryWords []string) string {
	lines := strings.Split(content, "\n")
	contentLower := strings.ToLower(content)

	// 找到第一个匹配的位置
	for _, word := range queryWords {
		index := strings.Index(contentLower, word)
		if index >= 0 {
			// 提取上下文
			start := index - 50
			if start < 0 {
				start = 0
			}
			end := index + 100
			if end > len(content) {
				end = len(content)
			}

			snippet := content[start:end]
			// 清理片段
			snippet = strings.TrimSpace(snippet)
			if start > 0 {
				snippet = "..." + snippet
			}
			if end < len(content) {
				snippet = snippet + "..."
			}

			return snippet
		}
	}

	// 如果没有匹配，返回前 150 个字符
	if len(content) > 150 {
		return content[:150] + "..."
	}
	return content
}

func (s *SearchIndex) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "is": true, "at": true, "which": true, "on": true,
		"and": true, "or": true, "but": true, "in": true, "with": true,
		"to": true, "for": true, "of": true, "as": true, "by": true,
		"this": true, "that": true, "from": true, "are": true, "was": true,
	}
	return stopWords[word]
}
