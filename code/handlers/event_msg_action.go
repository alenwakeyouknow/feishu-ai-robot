package handlers

import (
	"fmt"
	"log"
	"time"

	"start-feishubot/logger"
	"start-feishubot/services/openai"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
)

func setDefaultPrompt(msg []openai.Messages) []openai.Messages {
	if !hasSystemRole(msg) {
		msg = append(msg, openai.Messages{
			Role: "system", Content: "你是一个智能助手，请默认使用中文回答用户的问题，除非用户明确要求使用其他语言。请尽可能详细和准确地回答用户的问题。当前日期：" + time.Now().Format("2006年01月02日"),
		})
	}
	return msg
}

//func setDefaultVisionPrompt(msg []openai.VisionMessages) []openai.VisionMessages {
//	if !hasSystemRole(msg) {
//		msg = append(msg, openai.VisionMessages{
//			Role: "system", Content: []openai.ContentType{
//				{Type: "text", Text: "You are ChatGPT4V, " +
//					"You are ChatGPT4V, " +
//					"a large language and picture model trained by" +
//					" OpenAI. " +
//					"Answer in user's language as cdetailed and accurate as " +
//					" possible. Knowledge cutoff: 20230601 " +
//					"Current date" + time.Now().Format("20060102"),
//				}},
//		})
//	}
//	return msg
//}

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
	if a.handler.config.StreamMode {
		return true
	}
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	// 如果没有提示词，默认模拟ChatGPT
	msg = setDefaultPrompt(msg)
	msg = append(msg, openai.Messages{
		Role: "user", Content: a.info.qParsed,
	})

	// get ai mode as temperature
	aiMode := a.handler.sessionCache.GetAIMode(*a.info.sessionId)
	// get current selected model
	currentModel := a.handler.sessionCache.GetCurrentModel(*a.info.sessionId)
	fmt.Println("msg: ", msg)
	fmt.Println("aiMode: ", aiMode)
	fmt.Println("currentModel: ", currentModel)
	
	// use specified model for completion
	completions, err := a.handler.gpt.CompletionsWithModel(msg, aiMode, currentModel)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	msg = append(msg, completions)
	a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
	//if new topic
	if len(msg) == 3 {
		//fmt.Println("new topic", msg[1].Content)
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId,
			completions.Content)
		return false
	}
	if len(msg) != 3 {
		sendOldTopicCard(*a.ctx, a.info.sessionId, a.info.msgId,
			completions.Content)
		return false
	}
	// 使用卡片格式回复以支持markdown
	cardContent, _ := newSendCard(
		withHeader("🤖 AI回答", larkcard.TemplateGreen),
		withMainMd(completions.Content),
		withModelSwitchButtons(a.info.sessionId),
		withNote("点击按钮可切换其他模型回答"))
	err = replyCard(*a.ctx, a.info.msgId, cardContent)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	return true
}

// 判断msg中的是否包含system role
func hasSystemRole(msg []openai.Messages) bool {
	for _, m := range msg {
		if m.Role == "system" {
			return true
		}
	}
	return false
}

type StreamMessageAction struct { /*消息*/
}

func (m *StreamMessageAction) Execute(a *ActionInfo) bool {
	if !a.handler.config.StreamMode {
		return true
	}
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	// 如果没有提示词，默认模拟ChatGPT
	msg = setDefaultPrompt(msg)
	msg = append(msg, openai.Messages{
		Role: "user", Content: a.info.qParsed,
	})
	//if new topic
	var ifNewTopic bool
	if len(msg) <= 3 {
		ifNewTopic = true
	} else {
		ifNewTopic = false
	}

	// 🔥 关键修复：立即发送"正在处理"卡片，然后异步处理AI调用
	cardId, err2 := sendOnProcess(a, ifNewTopic)
	if err2 != nil {
		return false
	}

	// 🔥 完全异步处理AI调用，不阻塞责任链执行
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("StreamMessageAction panic: %v", err)
				updateFinalCardWithSession(*a.ctx, "处理异常，请稍后重试", cardId, a.info.sessionId, ifNewTopic)
			}
		}()

		answer := ""
		chatResponseStream := make(chan string)
		done := make(chan struct{}) // 添加 done 信号，保证 goroutine 正确退出
		noContentTimeout := time.AfterFunc(10*time.Second, func() {
			log.Println("no content timeout")
			close(done)
			err := updateFinalCardWithSession(*a.ctx, "请求超时", cardId, a.info.sessionId, ifNewTopic)
			if err != nil {
				return
			}
			return
		})
		defer noContentTimeout.Stop()

		go func() {
			defer func() {
				if err := recover(); err != nil {
					err := updateFinalCardWithSession(*a.ctx, "聊天失败", cardId, a.info.sessionId, ifNewTopic)
					if err != nil {
						return
					}
				}
			}()

			//log.Printf("UserId: %s , Request: %s", a.info.userId, msg)
			aiMode := a.handler.sessionCache.GetAIMode(*a.info.sessionId)
			//fmt.Println("msg: ", msg)
			//fmt.Println("aiMode: ", aiMode)
			if err := a.handler.gpt.StreamChat(*a.ctx, msg, aiMode,
				chatResponseStream); err != nil {
				err := updateFinalCardWithSession(*a.ctx, "聊天失败", cardId, a.info.sessionId, ifNewTopic)
				if err != nil {
					return
				}
				close(done) // 关闭 done 信号
			}

			close(done) // 关闭 done 信号
		}()
		
		// 🎯 符合飞书官方要求的流式卡片更新机制
		// 基于飞书API最佳实践：适中的更新频率，避免过于频繁的API调用
		streamTicker := time.NewTicker(800 * time.Millisecond) // 800ms间隔，确保稳定性
		defer streamTicker.Stop()
		
		var lastUpdateLength int // 记录上次更新的内容长度
		var isStreaming bool = true
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("流式更新协程异常:", r)
				}
			}()
			
			for isStreaming {
				select {
				case <-done:
					isStreaming = false
					return
				case <-streamTicker.C:
					// 📝 按块更新内容，给用户流式输出的感觉
					if len(answer) > lastUpdateLength {
						// 🎭 形式上的流式：显示当前内容 + 正在输入指示器
						streamingContent := answer
						if len(answer) > 0 {
							streamingContent = answer + "\n\n⏳ *正在生成中...*"
						}
						
						err := updateTextCard(*a.ctx, streamingContent, cardId, ifNewTopic)
						if err != nil {
							logger.Error("流式更新失败:", err)
							// 遇到错误时适当延迟，避免频繁重试
							time.Sleep(200 * time.Millisecond)
							continue
						}
						
						lastUpdateLength = len(answer)
						logger.Debug("✨ 流式展示：当前内容长度", len(answer), "字符")
					}
				}
			}
		}()
		
		for {
			select {
			case res, ok := <-chatResponseStream:
				if !ok {
					return
				}
				noContentTimeout.Stop()
				answer += res
				//pp.Println("answer", answer)
			case <-done: // ✅ 流式更新完成处理
				// 🎯 停止流式更新，标记完成状态
				isStreaming = false
				streamTicker.Stop()
				
				// 📋 发送最终完整卡片 - 移除"正在生成中"提示，显示完整回答和操作按钮
				err := updateFinalCardWithSession(*a.ctx, answer, cardId, a.info.sessionId, ifNewTopic)
				if err != nil {
					logger.Error("最终卡片更新失败:", err)
					return
				}
				
				// 💾 保存对话记录到缓存
				msg := append(msg, openai.Messages{
					Role: "assistant", Content: answer,
				})
				a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
				close(chatResponseStream)
				
				logger.Info("🎉 流式回答完成 - 总字符数:", len(answer))
				return
			}
		}
	}()
	
	// 🔥 立即返回false，表示处理完成（异步处理已启动）
	return false
}

func sendOnProcess(a *ActionInfo, ifNewTopic bool) (*string, error) {
	// send 正在处理中
	cardId, err := sendOnProcessCard(*a.ctx, a.info.sessionId,
		a.info.msgId, ifNewTopic)
	if err != nil {
		return nil, err
	}
	return cardId, nil

}
