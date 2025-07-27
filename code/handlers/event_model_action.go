package handlers

import (
	"fmt"
	"start-feishubot/services"
	"start-feishubot/services/openai"
	"strings"
)

type ModelAction struct{}

func (*ModelAction) Execute(a *ActionInfo) bool {
	cmd := a.info.qParsed
	
	// 检查命令类型
	if strings.HasPrefix(cmd, "/model") || strings.HasPrefix(cmd, "/模型") {
		handleModelSwitch(a)
		return false // 阻止后续处理
	}
	
	if strings.HasPrefix(cmd, "/compare") || strings.HasPrefix(cmd, "/对比") {
		handleModelCompare(a)
		return false // 阻止后续处理
	}
	
	if strings.HasPrefix(cmd, "/models") || strings.HasPrefix(cmd, "/模型列表") {
		handleModelList(a)
		return false // 阻止后续处理
	}
	
	if strings.HasPrefix(cmd, "/current") || strings.HasPrefix(cmd, "/当前模型") {
		handleCurrentModel(a)
		return false // 阻止后续处理
	}
	
	return true // 继续处理其他消息
}

// handleModelSwitch 处理模型切换命令
func handleModelSwitch(a *ActionInfo) bool {
	cmd := a.info.qParsed
	
	// 解析命令参数
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		sendModelHelp(a)
		return true
	}
	
	modelQuery := parts[1]
	sessionId := *a.info.sessionId
	
	// 搜索匹配的模型
	var selectedModel *openai.ModelInfo
	
	// 直接匹配模型ID
	if model, exists := openai.SupportedModels[modelQuery]; exists {
		selectedModel = model
	} else {
		// 模糊搜索
		searchResults := openai.SearchModels(modelQuery)
		if len(searchResults) == 1 {
			selectedModel = searchResults[0]
		} else if len(searchResults) > 1 {
			sendModelSearchResults(a, searchResults, modelQuery)
			return true
		}
	}
	
	if selectedModel == nil {
		sendModelNotFound(a, modelQuery)
		return true
	}
	
	// 切换模型
	services.GetSessionCache().SetCurrentModel(sessionId, selectedModel.ID)
	services.GetSessionCache().SetCompareMode(sessionId, false)
	
	// 发送确认消息
	confirmMsg := fmt.Sprintf("✅ 已切换到模型: %s\n\n%s", 
		selectedModel.Name, selectedModel.Description)
	replyMsg(*a.ctx, confirmMsg, a.info.msgId)
	
	return true
}

// handleModelCompare 处理多模型对比命令
func handleModelCompare(a *ActionInfo) bool {
	cmd := a.info.qParsed
	sessionId := *a.info.sessionId
	
	// 解析命令
	parts := strings.SplitN(cmd, " ", 2)
	if len(parts) < 2 {
		sendCompareHelp(a)
		return true
	}
	
	query := parts[1]
	
	// 启用对比模式
	services.GetSessionCache().SetCompareMode(sessionId, true)
	
	// 获取所有支持的模型
	allModels := openai.GetAllModels()
	
	// 发送初始消息
	initMsg := fmt.Sprintf("🤖 多模型对比模式\n问题: %s\n\n正在请求各个模型的回答...", query)
	replyMsg(*a.ctx, initMsg, a.info.msgId)
	
	// 依次调用各个模型
	for i, model := range allModels {
		modelResponse := callModelForComparison(a, query, model.ID)
		
		responseMsg := fmt.Sprintf("【%s】\n%s\n\n---", 
			model.Name, modelResponse)
		
		// 如果是最后一个模型，添加结束标记
		if i == len(allModels)-1 {
			responseMsg += "\n\n✅ 多模型对比完成"
		}
		
		replyMsg(*a.ctx, responseMsg, a.info.msgId)
	}
	
	// 关闭对比模式
	services.GetSessionCache().SetCompareMode(sessionId, false)
	return true
}

// handleModelList 处理模型列表命令
func handleModelList(a *ActionInfo) bool {
	// 按分类显示模型
	categories := []string{"通用", "编程", "分析", "长文本", "免费"}
	
	var msgBuilder strings.Builder
	msgBuilder.WriteString("🤖 **支持的模型列表**\n\n")
	
	for _, category := range categories {
		models := openai.GetModelsByCategory(category)
		if len(models) > 0 {
			msgBuilder.WriteString(fmt.Sprintf("**%s类**\n", category))
			for _, model := range models {
				freeTag := ""
				if model.IsFree {
					freeTag = " 🆓"
				}
				msgBuilder.WriteString(fmt.Sprintf("• %s%s - %s\n", 
					model.Name, freeTag, model.Description))
			}
			msgBuilder.WriteString("\n")
		}
	}
	
	msgBuilder.WriteString("**使用方法:**\n")
	msgBuilder.WriteString("• `/model <模型名>` - 切换到指定模型\n")
	msgBuilder.WriteString("• `/compare <问题>` - 多模型对比\n")
	msgBuilder.WriteString("• `/current` - 查看当前模型")
	
	replyMsg(*a.ctx, msgBuilder.String(), a.info.msgId)
	return true
}

// handleCurrentModel 处理查看当前模型命令
func handleCurrentModel(a *ActionInfo) bool {
	sessionId := *a.info.sessionId
	
	currentModelID := services.GetSessionCache().GetCurrentModel(sessionId)
	compareMode := services.GetSessionCache().GetCompareMode(sessionId)
	
	var msg string
	if compareMode {
		msg = "🤖 当前处于多模型对比模式"
	} else if model, exists := openai.SupportedModels[currentModelID]; exists {
		freeTag := ""
		if model.IsFree {
			freeTag = " 🆓"
		}
		msg = fmt.Sprintf("🤖 **当前模型**: %s%s\n\n%s\n\n**能力**: %s\n**分类**: %s", 
			model.Name, freeTag, model.Description, 
			strings.Join(model.Capabilities, ", "), model.Category)
	} else {
		msg = fmt.Sprintf("🤖 **当前模型**: %s (未知模型)", currentModelID)
	}
	
	replyMsg(*a.ctx, msg, a.info.msgId)
	return true
}

// 辅助函数
func sendModelHelp(a *ActionInfo) {
	helpMsg := `🤖 **模型切换帮助**

**使用方法:**
• ` + "`/model <模型名>`" + ` - 切换到指定模型
• ` + "`/model gpt-4o`" + ` - 切换到GPT-4o
• ` + "`/model qwen`" + ` - 模糊搜索包含"qwen"的模型

**查看模型:**
• ` + "`/models`" + ` - 查看所有支持的模型
• ` + "`/current`" + ` - 查看当前使用的模型

**多模型对比:**
• ` + "`/compare <问题>`" + ` - 让所有模型回答同一问题`

	replyMsg(*a.ctx, helpMsg, a.info.msgId)
}

func sendCompareHelp(a *ActionInfo) {
	helpMsg := `🤖 **多模型对比帮助**

**使用方法:**
• ` + "`/compare <问题>`" + ` - 让所有模型回答同一问题

**示例:**
• ` + "`/compare 什么是人工智能?`" + `
• ` + "`/compare 写一个Python排序函数`" + `

**注意:** 多模型对比会依次调用所有支持的模型，可能需要较长时间。`

	replyMsg(*a.ctx, helpMsg, a.info.msgId)
}

func sendModelNotFound(a *ActionInfo, query string) {
	msg := fmt.Sprintf("❌ 未找到模型: %s\n\n使用 `/models` 查看所有支持的模型", query)
	replyMsg(*a.ctx, msg, a.info.msgId)
}

func sendModelSearchResults(a *ActionInfo, models []*openai.ModelInfo, query string) {
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("🔍 找到多个匹配 \"%s\" 的模型:\n\n", query))
	
	for _, model := range models {
		freeTag := ""
		if model.IsFree {
			freeTag = " 🆓"
		}
		msgBuilder.WriteString(fmt.Sprintf("• `/model %s` - %s%s\n", 
			model.ID, model.Name, freeTag))
	}
	
	msgBuilder.WriteString("\n请使用完整的模型ID进行切换。")
	replyMsg(*a.ctx, msgBuilder.String(), a.info.msgId)
}

// callModelForComparison 调用指定模型进行对比
func callModelForComparison(a *ActionInfo, query, modelID string) string {
	// 构建消息
	msg := []openai.Messages{
		{Role: "user", Content: query},
	}
	
	// 使用指定模型调用
	response, err := a.handler.gpt.CompletionsWithModel(msg, openai.Balance, modelID)
	if err != nil {
		return fmt.Sprintf("调用失败: %s", err.Error())
	}
	
	return response.Content
}