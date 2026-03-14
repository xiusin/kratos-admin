package sms

import (
	"fmt"
	"sync"
)

// TemplateType 短信模板类型
type TemplateType string

const (
	TemplateTypeVerificationCode TemplateType = "verification_code" // 验证码
	TemplateTypeNotification     TemplateType = "notification"      // 通知
	TemplateTypeMarketing        TemplateType = "marketing"         // 营销
)

// TemplateManager 短信模板管理器
type TemplateManager struct {
	mu        sync.RWMutex
	templates map[string]*Template
}

// NewTemplateManager 创建短信模板管理器
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*Template),
	}
}

// Register 注册短信模板
func (tm *TemplateManager) Register(name string, template *Template) error {
	if name == "" {
		return fmt.Errorf("template name is required")
	}
	if template == nil {
		return fmt.Errorf("template is required")
	}
	if template.Code == "" {
		return fmt.Errorf("template code is required")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.templates[name] = template
	return nil
}

// Get 获取短信模板
func (tm *TemplateManager) Get(name string) (*Template, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	template, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	return template, nil
}

// GetCode 获取模板代码
func (tm *TemplateManager) GetCode(name string) (string, error) {
	template, err := tm.Get(name)
	if err != nil {
		return "", err
	}
	return template.Code, nil
}

// List 列出所有模板
func (tm *TemplateManager) List() map[string]*Template {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[string]*Template, len(tm.templates))
	for k, v := range tm.templates {
		result[k] = v
	}
	return result
}

// Delete 删除模板
func (tm *TemplateManager) Delete(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.templates[name]; !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	delete(tm.templates, name)
	return nil
}

// Count 获取模板数量
func (tm *TemplateManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.templates)
}
