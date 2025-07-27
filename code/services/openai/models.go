package openai

import (
	"fmt"
	"strings"
)

// ModelProvider 定义模型提供商
type ModelProvider string

const (
	ProviderOpenAI     ModelProvider = "openai"
	ProviderAnthropic  ModelProvider = "anthropic"
	ProviderQwen       ModelProvider = "qwen"
	ProviderGoogle     ModelProvider = "google"
	ProviderDeepSeek   ModelProvider = "deepseek"
	ProviderMoonshot   ModelProvider = "moonshot"
)

// ModelInfo 模型信息结构
type ModelInfo struct {
	ID           string        `json:"id"`           // 模型ID (如: openai/gpt-4o)
	Name         string        `json:"name"`         // 显示名称 (如: GPT-4o)
	Provider     ModelProvider `json:"provider"`     // 提供商
	Description  string        `json:"description"`  // 模型描述
	MaxTokens    int          `json:"max_tokens"`   // 最大token数
	IsFree       bool         `json:"is_free"`      // 是否免费
	Capabilities []string     `json:"capabilities"` // 能力列表 (text, image, vision等)
	Category     string       `json:"category"`     // 分类 (通用, 编程, 专业等)
}

// 支持的模型列表
var SupportedModels = map[string]*ModelInfo{
	"openai/gpt-4o": {
		ID:           "openai/gpt-4o",
		Name:         "GPT-4o",
		Provider:     ProviderOpenAI,
		Description:  "OpenAI最新的多模态模型，支持文本、图像和语音",
		MaxTokens:    128000,
		IsFree:       false,
		Capabilities: []string{"text", "vision", "reasoning"},
		Category:     "通用",
	},
	"openai/gpt-4.1": {
		ID:           "openai/gpt-4.1",
		Name:         "GPT-4.1",
		Provider:     ProviderOpenAI,
		Description:  "OpenAI GPT-4.1，增强的推理能力",
		MaxTokens:    128000,
		IsFree:       false,
		Capabilities: []string{"text", "reasoning"},
		Category:     "通用",
	},
	"qwen/qwen3-coder:free": {
		ID:           "qwen/qwen3-coder:free",
		Name:         "Qwen3 Coder (免费)",
		Provider:     ProviderQwen,
		Description:  "通义千问3代码专家版，专为编程任务优化",
		MaxTokens:    32768,
		IsFree:       true,
		Capabilities: []string{"text", "coding"},
		Category:     "编程",
	},
	"google/gemini-2.5-pro": {
		ID:           "google/gemini-2.5-pro",
		Name:         "Gemini 2.5 Pro",
		Provider:     ProviderGoogle,
		Description:  "Google最新的大型语言模型，性能卓越",
		MaxTokens:    1000000,
		IsFree:       false,
		Capabilities: []string{"text", "vision", "reasoning"},
		Category:     "通用",
	},
	"anthropic/claude-sonnet-4": {
		ID:           "anthropic/claude-sonnet-4",
		Name:         "claude-sonnet-4",
		Provider:     ProviderAnthropic,
		Description:  "Anthropic最新的Claude模型，擅长分析和推理",
		MaxTokens:    200000,
		IsFree:       false,
		Capabilities: []string{"text", "analysis", "reasoning"},
		Category:     "分析",
	},
	"deepseek/deepseek-chat-v3-0324:free": {
		ID:           "deepseek/deepseek-chat-v3-0324:free",
		Name:         "DeepSeek Chat V3 (免费)",
		Provider:     ProviderDeepSeek,
		Description:  "深度求索聊天模型V3版本，免费使用",
		MaxTokens:    32768,
		IsFree:       true,
		Capabilities: []string{"text", "reasoning"},
		Category:     "通用",
	},
	"moonshot/kimi-k2-0711-preview": {
		ID:           "moonshot/kimi-k2-0711-preview",
		Name:         "Kimi K2 (免费)",
		Provider:     ProviderMoonshot,
		Description:  "月之暗面Kimi K2模型，OpenRouter免费版本",
		MaxTokens:    128000,
		IsFree:       true,
		Capabilities: []string{"text", "long_context", "reasoning"},
		Category:     "长文本",
	},
}

// ModelCategory 模型分类
var ModelCategories = map[string][]string{
	"通用": {
		"openai/gpt-4o",
		"openai/gpt-4.1", 
		"google/gemini-2.5-pro",
		"deepseek/deepseek-chat-v3-0324:free",
	},
	"编程": {
		"qwen/qwen3-coder:free",
	},
	"分析": {
		"anthropic/claude-sonnet-4",
	},
	"长文本": {
		"moonshot/kimi-k2-0711-preview",
	},
	"免费": {
		"qwen/qwen3-coder:free",
		"deepseek/deepseek-chat-v3-0324:free",
		"moonshot/kimi-k2-0711-preview",
	},
}

// GetModelInfo 获取模型信息
func GetModelInfo(modelID string) (*ModelInfo, bool) {
	model, exists := SupportedModels[modelID]
	return model, exists
}

// GetModelsByCategory 按分类获取模型
func GetModelsByCategory(category string) []*ModelInfo {
	var models []*ModelInfo
	if modelIDs, exists := ModelCategories[category]; exists {
		for _, modelID := range modelIDs {
			if model, ok := SupportedModels[modelID]; ok {
				models = append(models, model)
			}
		}
	}
	return models
}

// GetAllModels 获取所有模型
func GetAllModels() []*ModelInfo {
	var models []*ModelInfo
	for _, model := range SupportedModels {
		models = append(models, model)
	}
	return models
}

// GetFreeModels 获取免费模型
func GetFreeModels() []*ModelInfo {
	var models []*ModelInfo
	for _, model := range SupportedModels {
		if model.IsFree {
			models = append(models, model)
		}
	}
	return models
}

// SearchModels 搜索模型
func SearchModels(keyword string) []*ModelInfo {
	var models []*ModelInfo
	keyword = strings.ToLower(keyword)
	
	for _, model := range SupportedModels {
		if strings.Contains(strings.ToLower(model.Name), keyword) ||
		   strings.Contains(strings.ToLower(model.Description), keyword) ||
		   strings.Contains(strings.ToLower(string(model.Provider)), keyword) {
			models = append(models, model)
		}
	}
	return models
}

// GetModelDisplayText 获取模型显示文本
func GetModelDisplayText(modelID string) string {
	if model, exists := SupportedModels[modelID]; exists {
		freeTag := ""
		if model.IsFree {
			freeTag = " 🆓"
		}
		return fmt.Sprintf("%s%s - %s", model.Name, freeTag, model.Description)
	}
	return modelID
}

// ValidateModel 验证模型是否支持
func ValidateModel(modelID string) error {
	if _, exists := SupportedModels[modelID]; !exists {
		return fmt.Errorf("不支持的模型: %s", modelID)
	}
	return nil
}

// DefaultModel 默认模型
const DefaultModel = "openai/gpt-4o"

// GetDefaultModel 获取默认模型
func GetDefaultModel() string {
	return DefaultModel
}