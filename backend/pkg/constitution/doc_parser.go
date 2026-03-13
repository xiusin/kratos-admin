package constitution

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ProtoParser Protobuf 解析器接口
type ProtoParser interface {
	Parse(filePath string) (*APIDocumentation, error)
}

// VueParser Vue 组件解析器接口
type VueParser interface {
	Parse(filePath string) (*ComponentDocumentation, error)
}

// protoParser Protobuf 解析器实现
type protoParser struct{}

// NewProtoParser 创建 Protobuf 解析器
func NewProtoParser() ProtoParser {
	return &protoParser{}
}

// Parse 解析 Protobuf 文件
func (p *protoParser) Parse(filePath string) (*APIDocumentation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc := &APIDocumentation{
		SourceFile: filePath,
		Methods:    []*APIMethod{},
		Messages:   []*APIMessage{},
	}

	scanner := bufio.NewScanner(file)
	var currentComment strings.Builder
	var inService bool
	var inMessage bool
	var currentMessage *APIMessage

	serviceRegex := regexp.MustCompile(`service\s+(\w+)\s*{`)
	rpcRegex := regexp.MustCompile(`rpc\s+(\w+)\s*\((\w+)\)\s*returns\s*\((\w+)\)`)
	messageRegex := regexp.MustCompile(`message\s+(\w+)\s*{`)
	fieldRegex := regexp.MustCompile(`\s*(\w+)\s+(\w+)\s*=\s*(\d+);`)
	packageRegex := regexp.MustCompile(`package\s+([\w.]+);`)
	commentRegex := regexp.MustCompile(`^\s*//\s*(.*)`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 提取注释
		if matches := commentRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			if currentComment.Len() > 0 {
				currentComment.WriteString(" ")
			}
			currentComment.WriteString(matches[1])
			continue
		}

		// 提取 package
		if matches := packageRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			doc.Package = matches[1]
		}

		// 提取 service
		if matches := serviceRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			doc.ServiceName = matches[1]
			doc.Description = currentComment.String()
			currentComment.Reset()
			inService = true
			continue
		}

		// 提取 rpc 方法
		if inService {
			if matches := rpcRegex.FindStringSubmatch(trimmed); len(matches) > 3 {
				method := &APIMethod{
					Name:         matches[1],
					Description:  currentComment.String(),
					RequestType:  matches[2],
					ResponseType: matches[3],
				}
				doc.Methods = append(doc.Methods, method)
				currentComment.Reset()
				continue
			}
		}

		// 提取 message
		if matches := messageRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			if currentMessage != nil {
				doc.Messages = append(doc.Messages, currentMessage)
			}
			currentMessage = &APIMessage{
				Name:        matches[1],
				Description: currentComment.String(),
				Fields:      []*APIField{},
			}
			currentComment.Reset()
			inMessage = true
			continue
		}

		// 提取 message 字段
		if inMessage && currentMessage != nil {
			if matches := fieldRegex.FindStringSubmatch(trimmed); len(matches) > 3 {
				field := &APIField{
					Type:        matches[1],
					Name:        matches[2],
					Description: currentComment.String(),
				}
				currentMessage.Fields = append(currentMessage.Fields, field)
				currentComment.Reset()
				continue
			}
		}

		// 检测块结束
		if trimmed == "}" {
			if inMessage && currentMessage != nil {
				doc.Messages = append(doc.Messages, currentMessage)
				currentMessage = nil
				inMessage = false
			}
			if inService {
				inService = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return doc, nil
}

// vueParser Vue 组件解析器实现
type vueParser struct{}

// NewVueParser 创建 Vue 组件解析器
func NewVueParser() VueParser {
	return &vueParser{}
}

// Parse 解析 Vue 组件文件
func (p *vueParser) Parse(filePath string) (*ComponentDocumentation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc := &ComponentDocumentation{
		SourceFile: filePath,
		Props:      []*ComponentProp{},
		Events:     []*ComponentEvent{},
		Slots:      []*ComponentSlot{},
	}

	// 从文件名提取组件名
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]
	doc.Name = strings.TrimSuffix(fileName, ".vue")

	scanner := bufio.NewScanner(file)
	var currentComment strings.Builder
	var inScript bool
	var inProps bool

	commentRegex := regexp.MustCompile(`^\s*//\s*(.*)`)
	propsRegex := regexp.MustCompile(`defineProps<\{`)
	propRegex := regexp.MustCompile(`\s*(\w+)(\?)?:\s*(\w+);`)
	emitRegex := regexp.MustCompile(`emit\(['"](\w+)['"]`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 检测 script 块
		if strings.Contains(trimmed, "<script") {
			inScript = true
			continue
		}
		if strings.Contains(trimmed, "</script>") {
			inScript = false
			continue
		}

		if !inScript {
			continue
		}

		// 提取注释
		if matches := commentRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			if currentComment.Len() > 0 {
				currentComment.WriteString(" ")
			}
			currentComment.WriteString(matches[1])
			continue
		}

		// 检测 props 定义
		if propsRegex.MatchString(trimmed) {
			inProps = true
			continue
		}

		// 提取 prop
		if inProps {
			if matches := propRegex.FindStringSubmatch(trimmed); len(matches) > 3 {
				prop := &ComponentProp{
					Name:        matches[1],
					Type:        matches[3],
					Description: currentComment.String(),
					Required:    matches[2] == "",
				}
				doc.Props = append(doc.Props, prop)
				currentComment.Reset()
				continue
			}
			if trimmed == "}>" || trimmed == "})" {
				inProps = false
			}
		}

		// 提取 emit 事件
		if matches := emitRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			event := &ComponentEvent{
				Name:        matches[1],
				Description: currentComment.String(),
			}
			doc.Events = append(doc.Events, event)
			currentComment.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return doc, nil
}
