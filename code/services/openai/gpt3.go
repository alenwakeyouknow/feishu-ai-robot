package openai

import (
	"errors"
	"start-feishubot/logger"
	"strings"

	"github.com/pandodao/tokenizer-go"
)

type AIMode float64

const (
	Fresh      AIMode = 0.1
	Warmth     AIMode = 0.7
	Balance    AIMode = 1.2
	Creativity AIMode = 1.7
)

var AIModeMap = map[string]AIMode{
	"严谨": Fresh,
	"简洁": Warmth,
	"标准": Balance,
	"发散": Creativity,
}

var AIModeStrs = []string{
	"严谨",
	"简洁",
	"标准",
	"发散",
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatGPTChoiceItem    `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ChatGPTChoiceItem struct {
	Message      Messages `json:"message"`
	Index        int      `json:"index"`
	FinishReason string   `json:"finish_reason"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model            string     `json:"model"`
	Messages         []Messages `json:"messages"`
	MaxTokens        int        `json:"max_tokens"`
	Temperature      AIMode     `json:"temperature"`
	TopP             int        `json:"top_p"`
	FrequencyPenalty int        `json:"frequency_penalty"`
	PresencePenalty  int        `json:"presence_penalty"`
}

func (msg *Messages) CalculateTokenLength() int {
	text := strings.TrimSpace(msg.Content)
	return tokenizer.MustCalToken(text)
}

func (gpt *ChatGPT) Completions(msg []Messages, aiMode AIMode) (resp Messages,
	err error) {
	return gpt.CompletionsWithModel(msg, aiMode, gpt.Model)
}

// CompletionsWithModel 使用指定模型进行对话
func (gpt *ChatGPT) CompletionsWithModel(msg []Messages, aiMode AIMode, model string) (resp Messages,
	err error) {
	requestBody := ChatGPTRequestBody{
		Model:            model,
		Messages:         msg,
		MaxTokens:        gpt.MaxTokens,
		Temperature:      aiMode,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	gptResponseBody := &ChatGPTResponseBody{}
	url := gpt.FullUrl("chat/completions")
	logger.Debug(url)
	logger.Debugf("request body with model %s: %+v", model, requestBody)
	
	if url == "" {
		return resp, errors.New("无法获取openai请求地址")
	}
	
	err = gpt.sendRequestWithBodyType(url, "POST", jsonBody, requestBody, gptResponseBody)
	if err == nil && len(gptResponseBody.Choices) > 0 {
		resp = gptResponseBody.Choices[0].Message
	} else {
		logger.Errorf("ERROR %v", err)
		resp = Messages{}
		err = errors.New("模型 " + model + " 请求失败: " + err.Error())
	}
	return resp, err
}
