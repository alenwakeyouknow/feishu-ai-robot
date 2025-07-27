package handlers

import (
	"context"
	"fmt"
	"strings"

	"start-feishubot/services"
	"start-feishubot/services/openai"

	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
)

// NewModelSwitchCardHandler 处理模型切换按钮
func NewModelSwitchCardHandler(cardMsg CardMsg, m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
		if cardMsg.Kind != ModelSwitchKind {
			return nil, ErrNextHandler
		}

		// 获取选择的模型
		modelID, ok := cardMsg.Value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid model ID value")
		}
		sessionId := cardMsg.SessionId
		
		// 切换到选择的模型
		services.GetSessionCache().SetCurrentModel(sessionId, modelID)
		services.GetSessionCache().SetCompareMode(sessionId, false)
		
		// 获取模型信息
		var modelName string
		if model, exists := openai.SupportedModels[modelID]; exists {
			modelName = model.Name
		} else {
			modelName = modelID
		}
		
		// 获取用户的原始问题（从消息中提取）
		msg := services.GetSessionCache().GetMsg(sessionId)
		var userQuestion string
		if len(msg) > 0 {
			// 找到最后一个用户消息
			for i := len(msg) - 1; i >= 0; i-- {
				if msg[i].Role == "user" {
					userQuestion = msg[i].Content
					break
				}
			}
		}
		
		if userQuestion == "" {
			userQuestion = "请介绍一下你自己" // 默认问题
		}
		
		// 先返回"正在处理"的卡片
		processingCard, _ := newSendCard(
			withHeader(fmt.Sprintf("🤖 正在使用 %s 回答", modelName), larkcard.TemplateBlue),
			withMainMd(fmt.Sprintf("**问题:** %s\n\n⏳ 正在思考中，请稍候...", userQuestion)),
			withNote("请稍等，正在获取回答"))
		
		// 在后台异步处理
		go func() {
			// 使用新模型重新回答问题
			newMsg := []openai.Messages{
				{Role: "user", Content: userQuestion},
			}
			
			// 调用新模型
			completions, err := m.gpt.CompletionsWithModel(newMsg, openai.Balance, modelID)
			if err != nil {
				// 发送错误卡片
				errorCard, _ := newSendCard(
					withHeader("❌ 模型切换错误", larkcard.TemplateRed),
					withMainText(fmt.Sprintf("切换到 %s 时出错: %v", modelName, err)),
					withModelSwitchButtons(&sessionId),
					withNote("请稍后重试或选择其他模型"))
				replyCard(ctx, &cardAction.OpenMessageID, errorCard)
				return
			}
			
			// 发送最终回答卡片
			finalCard, _ := newSendCard(
				withHeader(fmt.Sprintf("🤖 %s 的回答", modelName), larkcard.TemplateGreen),
				withMainMd(fmt.Sprintf("**问题:** %s", userQuestion)),
				withMainMd(completions.Content),
				withModelSwitchButtons(&sessionId),
				withNote("点击按钮可切换其他模型回答"))
			replyCard(ctx, &cardAction.OpenMessageID, finalCard)
		}()
		
		return processingCard, nil
	}
}

// NewAllModelsCardHandler 处理主流模型回答按钮
func NewAllModelsCardHandler(cardMsg CardMsg, m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
		if cardMsg.Kind != AllModelsKind {
			return nil, ErrNextHandler
		}

		sessionId := cardMsg.SessionId
		
		// 获取用户的原始问题
		msg := services.GetSessionCache().GetMsg(sessionId)
		var userQuestion string
		if len(msg) > 0 {
			for i := len(msg) - 1; i >= 0; i-- {
				if msg[i].Role == "user" {
					userQuestion = msg[i].Content
					break
				}
			}
		}
		
		if userQuestion == "" {
			userQuestion = "请介绍一下你自己"
		}
		
		// 启用对比模式
		services.GetSessionCache().SetCompareMode(sessionId, true)
		
		// 先返回初始提示卡片
		initialCard, _ := newSendCard(
			withHeader("🚀 主流模型回答", larkcard.TemplateBlue),
			withMainMd(fmt.Sprintf("**问题:** %s\n\n⏳ 正在请求GPT、Claude、DeepSeek三个主流模型回答，将分别发送3个卡片...", userQuestion)),
			withNote("请稍等，正在获取各模型回答"))
		
		// 在后台异步处理3个主流模型
		go func() {
			// 主流模型定义：GPT、Claude、DeepSeek
			mainModels := []struct{
				ID string
				DisplayName string
				Emoji string
			}{
				{"openai/gpt-4o", "GPT-4o", "🧠"},
				{"anthropic/claude-sonnet-4", "Claude-Sonnet-4", "🎭"},
				{"deepseek/deepseek-chat-v3-0324:free", "DeepSeek-V3", "🔍"},
			}
			
			// 为每个模型创建单独的卡片
			for _, model := range mainModels {
				newMsg := []openai.Messages{
					{Role: "user", Content: userQuestion},
				}
				
				completions, err := m.gpt.CompletionsWithModel(newMsg, openai.Balance, model.ID)
				var response string
				var cardColor string
				if err != nil {
					response = fmt.Sprintf("❌ 调用失败: %s", err.Error())
					cardColor = larkcard.TemplateRed
				} else {
					response = completions.Content
					cardColor = larkcard.TemplateGreen
				}
				
				// 为每个模型创建独立的回答卡片
				modelCard, _ := newSendCard(
					withHeader(fmt.Sprintf("%s %s 的回答", model.Emoji, model.DisplayName), cardColor),
					withMainMd(fmt.Sprintf("**问题:** %s\n\n**回答:**\n%s", userQuestion, response)),
					withModelSwitchButtons(&sessionId),
					withNote(fmt.Sprintf("来自 %s 的回答，点击按钮可切换其他模型", model.DisplayName)))
				
				// 发送新的回答卡片
				replyCard(ctx, &cardAction.OpenMessageID, modelCard)
			}
			
			// 关闭对比模式
			services.GetSessionCache().SetCompareMode(sessionId, false)
		}()
		
		return initialCard, nil
	}
}

// NewMoreModelsCardHandler 处理查看更多模型按钮
func NewMoreModelsCardHandler(cardMsg CardMsg, m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
		if cardMsg.Kind != MoreModelsKind {
			return nil, ErrNextHandler
		}

		sessionId := cardMsg.SessionId
		
		// 获取前10个模型
		allModels := openai.GetAllModels()
		var modelList strings.Builder
		modelList.WriteString("**📋 支持的模型列表:**\n\n")
		
		for i, model := range allModels {
			if i >= 10 { // 只显示前10个
				break
			}
			freeTag := ""
			if model.IsFree {
				freeTag = " 🆓"
			}
			modelList.WriteString(fmt.Sprintf("%d. **%s**%s\n   *%s*\n\n", 
				i+1, model.Name, freeTag, model.Description))
		}
		
		modelList.WriteString("💡 **使用方法:**\n")
		modelList.WriteString("• 发送 `/model <模型名>` 切换模型\n")
		modelList.WriteString("• 发送 `/compare <问题>` 多模型对比")
		
		newCard, _ := newSendCard(
			withHeader("📚 模型库", larkcard.TemplateBlue),
			withMainMd(modelList.String()),
			withModelSwitchButtons(&sessionId),
			withNote("使用命令或点击按钮即可切换模型"))
		
		return newCard, nil
	}
}