package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"start-feishubot/logger"

	"start-feishubot/initialization"
	"start-feishubot/services"
	"start-feishubot/services/openai"

	"github.com/google/uuid"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type CardKind string
type CardChatType string

var (
	ClearCardKind        = CardKind("clear")            // 清空上下文
	PicModeChangeKind    = CardKind("pic_mode_change")  // 切换图片创作模式
	VisionModeChangeKind = CardKind("vision_mode")      // 切换图片解析模式
	PicResolutionKind    = CardKind("pic_resolution")   // 图片分辨率调整
	PicStyleKind         = CardKind("pic_style")        // 图片风格调整
	VisionStyleKind      = CardKind("vision_style")     // 图片推理级别调整
	PicTextMoreKind      = CardKind("pic_text_more")    // 重新根据文本生成图片
	PicVarMoreKind       = CardKind("pic_var_more")     // 变量图片
	RoleTagsChooseKind   = CardKind("role_tags_choose") // 内置角色所属标签选择
	RoleChooseKind       = CardKind("role_choose")      // 内置角色选择
	AIModeChooseKind     = CardKind("ai_mode_choose")   // AI模式选择
	ModelSwitchKind      = CardKind("model_switch")     // 模型切换
	AllModelsKind        = CardKind("all_models")       // 主流模型回答
	MoreModelsKind       = CardKind("more_models")      // 查看更多模型
)

var (
	GroupChatType = CardChatType("group")
	UserChatType  = CardChatType("personal")
)

type CardMsg struct {
	Kind      CardKind    `json:"kind"`
	ChatType  CardChatType `json:"chatType"`
	Value     interface{} `json:"value"`
	SessionId string      `json:"sessionId"` // 使用json tag确保字段名一致
	MsgId     string      `json:"msgId"`
}

type MenuOption struct {
	value string
	label string
}

func replyCard(ctx context.Context,
	msgId *string,
	cardContent string,
) error {
	client := initialization.GetLarkClient()
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		logger.Errorf("服务端错误 resp code[%v], msg [%v] requestId [%v] ", resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func newSendCard(
	header *larkcard.MessageCardHeader,
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(true). // 启用共享卡片
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Header(header).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

// newStreamingCard 创建支持流式更新的卡片 (遵循飞书官方标准)
func newStreamingCard(
	header *larkcard.MessageCardHeader,
	elements ...larkcard.MessageCardElement) (string, error) {
	
	// 根据飞书官方文档，使用官方SDK创建支持更新的卡片
	// 流式效果通过PatchCard API实现，而非自定义streaming_mode
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(true). // 关键：启用卡片更新支持
		Build()
		
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	
	// 使用官方SDK创建标准卡片结构
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Header(header).
		Elements(aElementPool).
		String()
		
	return cardContent, err
}

func newSimpleSendCard(
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

// withSplitLine 用于生成分割线
func withSplitLine() larkcard.MessageCardElement {
	splitLine := larkcard.NewMessageCardHr().
		Build()
	return splitLine
}

// withHeader 用于生成消息头
func withHeader(title string, color string) *larkcard.
	MessageCardHeader {
	if title == "" {
		title = "🤖️机器人提醒"
	}
	header := larkcard.NewMessageCardHeader().
		Template(color).
		Title(larkcard.NewMessageCardPlainText().
			Content(title).
			Build()).
		Build()
	return header
}

// withNote 用于生成纯文本脚注
func withNote(note string) larkcard.MessageCardElement {
	noteElement := larkcard.NewMessageCardNote().
		Elements([]larkcard.MessageCardNoteElement{larkcard.NewMessageCardPlainText().
			Content(note).
			Build()}).
		Build()
	return noteElement
}

// withMainMd 用于生成markdown消息体（保持markdown格式）
func withMainMd(msg string) larkcard.MessageCardElement {
	// 不使用processMessage以避免JSON转义破坏markdown格式
	msg = strings.TrimSpace(msg)
	// 只处理换行符，保持其他markdown符号
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardLarkMd().
				Content(msg).
				Build()).
			IsShort(false). // 设置为false以支持更长的markdown内容
			Build()}).
		Build()
	return mainElement
}

// withEnhancedMd 用于生成增强markdown消息体（优化markdown支持）
func withEnhancedMd(msg string) larkcard.MessageCardElement {
	// 保持原始内容，仅进行必要的清理
	msg = strings.TrimSpace(msg)
	
	// 处理特殊的换行情况
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// 处理双换行为段落分隔
	msg = strings.ReplaceAll(msg, "\n\n", "\n\n")
	
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardLarkMd().
				Content(msg).
				Build()).
			IsShort(false). // 支持长内容
			Build()}).
		Build()
	return mainElement
}

// withMainText 用于生成纯文本消息体
func withMainText(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = cleanTextBlock(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardLarkMd().
				Content(msg).
				Build()).
			IsShort(false).
			Build()}).
		Build()
	return mainElement
}

func withImageDiv(imageKey string) larkcard.MessageCardElement {
	imageElement := larkcard.NewMessageCardImage().
		ImgKey(imageKey).
		Alt(larkcard.NewMessageCardPlainText().Content("").
			Build()).
		Preview(true).
		Mode(larkcard.MessageCardImageModelCropCenter).
		CompactWidth(true).
		Build()
	return imageElement
}

// withMdAndExtraBtn 用于生成带有额外按钮的消息体
func withMdAndExtraBtn(msg string, btn *larkcard.
	MessageCardEmbedButton) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields(
			[]*larkcard.MessageCardField{
				larkcard.NewMessageCardField().
					Text(larkcard.NewMessageCardLarkMd().
						Content(msg).
						Build()).
					IsShort(true).
					Build()}).
		Extra(btn).
		Build()
	return mainElement
}

func newBtn(content string, value map[string]interface{},
	typename larkcard.MessageCardButtonType) *larkcard.
	MessageCardEmbedButton {
	btn := larkcard.NewMessageCardEmbedButton().
		Type(typename).
		Value(value).
		Text(larkcard.NewMessageCardPlainText().
			Content(content).
			Build())
	return btn
}

func newMenu(
	placeHolder string,
	value map[string]interface{},
	options ...MenuOption,
) *larkcard.
	MessageCardEmbedSelectMenuStatic {
	var aOptionPool []*larkcard.MessageCardEmbedSelectOption
	for _, option := range options {
		aOption := larkcard.NewMessageCardEmbedSelectOption().
			Value(option.value).
			Text(larkcard.NewMessageCardPlainText().
				Content(option.label).
				Build())
		aOptionPool = append(aOptionPool, aOption)

	}
	btn := larkcard.NewMessageCardEmbedSelectMenuStatic().
		MessageCardEmbedSelectMenuStatic(larkcard.NewMessageCardEmbedSelectMenuBase().
			Options(aOptionPool).
			Placeholder(larkcard.NewMessageCardPlainText().
				Content(placeHolder).
				Build()).
			Value(value).
			Build()).
		Build()
	return btn
}

// 清除卡片按钮
func withClearDoubleCheckBtn(sessionID *string) larkcard.MessageCardElement {
	confirmBtn := newBtn("确认清除", map[string]interface{}{
		"name":      "clear_confirm_btn",
		"value":     "1",
		"kind":      ClearCardKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"name":      "clear_cancel_btn",
		"value":     "0",
		"kind":      ClearCardKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withPicModeDoubleCheckBtn(sessionID *string) larkcard.
	MessageCardElement {
	confirmBtn := newBtn("切换模式", map[string]interface{}{
		"name":      "pic_mode_confirm_btn",
		"value":     "1",
		"kind":      PicModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"name":      "pic_mode_cancel_btn",
		"value":     "0",
		"kind":      PicModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}
func withVisionModeDoubleCheckBtn(sessionID *string) larkcard.
	MessageCardElement {
	confirmBtn := newBtn("切换模式", map[string]interface{}{
		"name":      "vision_mode_confirm_btn",
		"value":     "1",
		"kind":      VisionModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"name":      "vision_mode_cancel_btn",
		"value":     "0",
		"kind":      VisionModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withOneBtn(btn *larkcard.MessageCardEmbedButton) larkcard.
	MessageCardElement {
	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{btn}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

//新建对话按钮

func withPicResolutionBtn(sessionID *string) larkcard.
	MessageCardElement {
	resolutionMenu := newMenu("默认分辨率",
		map[string]interface{}{
			"value":     "0",
			"kind":      PicResolutionKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		// dall-e-2 256, 512, 1024
		//MenuOption{
		//	label: "256x256",
		//	value: string(services.Resolution256),
		//},
		//MenuOption{
		//	label: "512x512",
		//	value: string(services.Resolution512),
		//},
		// dall-e-3
		MenuOption{
			label: "1024x1024",
			value: string(services.Resolution1024),
		},
		MenuOption{
			label: "1024x1792",
			value: string(services.Resolution10241792),
		},
		MenuOption{
			label: "1792x1024",
			value: string(services.Resolution17921024),
		},
	)

	styleMenu := newMenu("风格",
		map[string]interface{}{
			"value":     "0",
			"kind":      PicStyleKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		MenuOption{
			label: "生动风格",
			value: string(services.PicStyleVivid),
		},
		MenuOption{
			label: "自然风格",
			value: string(services.PicStyleNatural),
		},
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{resolutionMenu, styleMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withVisionDetailLevelBtn(sessionID *string) larkcard.
	MessageCardElement {
	detailMenu := newMenu("选择图片解析度，默认为高",
		map[string]interface{}{
			"value":     "0",
			"kind":      VisionStyleKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		MenuOption{
			label: "高",
			value: string(services.VisionDetailHigh),
		},
		MenuOption{
			label: "低",
			value: string(services.VisionDetailLow),
		},
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{detailMenu}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}
func withRoleTagsBtn(sessionID *string, tags ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range tags {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu("选择角色分类",
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleTagsChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withRoleBtn(sessionID *string, titles ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range titles {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu("查看内置角色",
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withAIModeBtn(sessionID *string, aiModeStrs []string) larkcard.MessageCardElement {
	var menuOptions []MenuOption
	for _, label := range aiModeStrs {
		menuOptions = append(menuOptions, MenuOption{
			label: label,
			value: label,
		})
	}

	cancelMenu := newMenu("选择模式",
		map[string]interface{}{
			"value":     "0",
			"kind":      AIModeChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

// withModelSwitchButtons 根据飞书卡片文档创建模型切换按钮
func withModelSwitchButtons(sessionID *string) larkcard.MessageCardElement {
	// Claude按钮
	claudeBtn := newBtn("claude-sonnet-4", map[string]interface{}{
		"name":      "claude_btn",
		"value":     "anthropic/claude-sonnet-4",
		"kind":      ModelSwitchKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypePrimary)
	
	// DeepSeek按钮
	deepseekBtn := newBtn("DeepSeek", map[string]interface{}{
		"name":      "deepseek_btn",
		"value":     "deepseek/deepseek-chat-v3-0324:free",
		"kind":      ModelSwitchKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypePrimary)
	
	// Gemini按钮
	geminiBtn := newBtn("Gemini 2.5", map[string]interface{}{
		"name":      "gemini_btn",
		"value":     "google/gemini-2.5-pro",
		"kind":      ModelSwitchKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypePrimary)
	
	
	// 主流模型按钮
	allModelsBtn := newBtn("主流模型回答", map[string]interface{}{
		"name":      "all_models_btn",
		"value":     "all_models",
		"kind":      AllModelsKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDefault)
	
	// 查看更多模型按钮
	moreModelsBtn := newBtn("查看更多模型", map[string]interface{}{
		"name":      "more_models_btn",
		"value":     "more_models",
		"kind":      MoreModelsKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDefault)

	// 创建按钮组
	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{
			claudeBtn, deepseekBtn, geminiBtn, allModelsBtn, moreModelsBtn,
		}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()

	return actions
}


func replyMsg(ctx context.Context, msg string, msgId *string) error {
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := initialization.GetLarkClient()
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func uploadImage(base64Str string) (*string, error) {
	imageBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	client := initialization.GetLarkClient()
	resp, err := client.Im.Image.Create(context.Background(),
		larkim.NewCreateImageReqBuilder().
			Body(larkim.NewCreateImageReqBodyBuilder().
				ImageType(larkim.ImageTypeMessage).
				Image(bytes.NewReader(imageBytes)).
				Build()).
			Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}
	return resp.Data.ImageKey, nil
}

func replyImage(ctx context.Context, ImageKey *string,
	msgId *string) error {
	//fmt.Println("sendMsg", ImageKey, msgId)

	msgImage := larkim.MessageImage{ImageKey: *ImageKey}
	content, err := msgImage.String()
	if err != nil {
		fmt.Println(err)
		return err
	}
	client := initialization.GetLarkClient()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeImage).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func replayImageCardByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string, question string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendImageCard(ctx, *imageKey, msgId, sessionId, question)
	if err != nil {
		return err
	}
	return nil
}

func replayImagePlainByBase64(ctx context.Context, base64Str string,
	msgId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = replyImage(ctx, imageKey, msgId)
	if err != nil {
		return err
	}
	return nil
}

func replayVariantImageByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendVarImageCard(ctx, *imageKey, msgId, sessionId)
	if err != nil {
		return err
	}
	return nil
}

func sendMsg(ctx context.Context, msg string, chatId *string) error {
	//fmt.Println("sendMsg", msg, chatId)
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := initialization.GetLarkClient()
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	//fmt.Println("content", content)

	resp, err := client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(*chatId).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func sendClearCacheCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🆑 机器人提醒", larkcard.TemplateBlue),
		withMainMd("您确定要清除对话上下文吗？"),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"),
		withClearDoubleCheckBtn(sessionId))
	replyCard(ctx, msgId, newCard)
}

func sendSystemInstructionCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🥷  已进入角色扮演模式", larkcard.TemplateIndigo),
		withMainMd(content),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"))
	replyCard(ctx, msgId, newCard)
}

func sendPicCreateInstructionCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🖼️ 已进入图片创作模式", larkcard.TemplateBlue),
		withPicResolutionBtn(sessionId),
		withNote("提醒：回复文本或图片，让AI生成相关的图片。"))
	replyCard(ctx, msgId, newCard)
}

func sendVisionInstructionCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🕵️️ 已进入图片推理模式", larkcard.TemplateBlue),
		withVisionDetailLevelBtn(sessionId),
		withNote("提醒：回复图片，让LLM和你一起推理图片的内容。"))
	replyCard(ctx, msgId, newCard)
}

func sendPicModeCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🖼️ 机器人提醒", larkcard.TemplateBlue),
		withMainMd("收到图片，是否进入图片创作模式？"),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"),
		withPicModeDoubleCheckBtn(sessionId))
	replyCard(ctx, msgId, newCard)
}
func sendVisionModeCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🕵️ 机器人提醒", larkcard.TemplateBlue),
		withMainMd("检测到图片，是否进入图片推理模式？"),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"),
		withVisionModeDoubleCheckBtn(sessionId))
	replyCard(ctx, msgId, newCard)
}

func sendNewTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🌟 已开启新的话题", larkcard.TemplateBlue),
		withMainMd(content),
		withModelSwitchButtons(sessionId),
		withNote("提醒：点击按钮可切换模型回答，或继续对话保持话题连贯"))
	replyCard(ctx, msgId, newCard)
}

func sendOldTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
		withMainMd(content),
		withModelSwitchButtons(sessionId),
		withNote("提醒：点击按钮可切换模型回答，或继续对话保持话题连贯"))
	replyCard(ctx, msgId, newCard)
}

func sendVisionTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🕵️图片推理结果", larkcard.TemplateBlue),
		withMainMd(content),
		withNote("让LLM和你一起推理图片的内容~"))
	replyCard(ctx, msgId, newCard)
}

func sendHelpCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🎒需要帮助吗？", larkcard.TemplateBlue),
		withMainMd("**🤠你好呀~ 我来自企联AI，一款基于OpenAI的智能助手！**"),
		withSplitLine(),
		withMdAndExtraBtn(
			"** 🆑 清除话题上下文**\n文本回复 *清除* 或 */clear*",
			newBtn("立刻清除", map[string]interface{}{
				"name":      "help_clear_btn",
				"value":     "1",
				"kind":      ClearCardKind,
				"chatType":  UserChatType,
				"sessionId": *sessionId,
			}, larkcard.MessageCardButtonTypeDanger)),
		withSplitLine(),
		withMainMd("🤖 **发散模式选择** \n"+" 文本回复 *发散模式* 或 */ai_mode*"),
		withSplitLine(),
		withMainMd("🛖 **内置角色列表** \n"+" 文本回复 *角色列表* 或 */roles*"),
		withSplitLine(),
		withMainMd("🥷 **角色扮演模式**\n文本回复*角色扮演* 或 */system*+空格+角色信息"),
		withSplitLine(),
		withMainMd("🎤 **AI语音对话**\n私聊模式下直接发送语音"),
		withSplitLine(),
		withMainMd("🎨 **图片创作模式**\n回复*图片创作* 或 */picture*"),
		withSplitLine(),
		withMainMd("🕵️ **图片推理模式** \n"+" 文本回复 *图片推理* 或 */vision*"),
		withSplitLine(),
		withMainMd("🎰 **Token余额查询**\n回复*余额* 或 */balance*"),
		withSplitLine(),
		withMainMd("🔃️ **历史话题回档** 🚧\n"+" 进入话题的回复详情页,文本回复 *恢复* 或 */reload*"),
		withSplitLine(),
		withMainMd("📤 **话题内容导出** 🚧\n"+" 文本回复 *导出* 或 */export*"),
		withSplitLine(),
		withMainMd("🎰 **连续对话与多话题模式**\n"+" 点击对话框参与回复，可保持话题连贯。同时，单独提问即可开启全新新话题"),
		withSplitLine(),
		withMainMd("🎒 **需要更多帮助**\n文本回复 *帮助* 或 */help*"),
	)
	replyCard(ctx, msgId, newCard)
}

func sendImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string, question string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn("再来一张", map[string]interface{}{
			"name":      "pic_text_more_btn",
			"value":     question,
			"kind":      PicTextMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(ctx, msgId, newCard)
	return nil
}

func sendVarImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn("再来一张", map[string]interface{}{
			"name":      "pic_var_more_btn",
			"value":     imageKey,
			"kind":      PicVarMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(ctx, msgId, newCard)
	return nil
}

func sendBalanceCard(ctx context.Context, msgId *string,
	balance openai.BalanceResponse) {
	newCard, _ := newSendCard(
		withHeader("🎰️ 余额查询", larkcard.TemplateBlue),
		withMainMd(fmt.Sprintf("总额度: %.2f$", balance.TotalGranted)),
		withMainMd(fmt.Sprintf("已用额度: %.2f$", balance.TotalUsed)),
		withMainMd(fmt.Sprintf("可用额度: %.2f$",
			balance.TotalAvailable)),
		withNote(fmt.Sprintf("有效期: %s - %s",
			balance.EffectiveAt.Format("2006-01-02 15:04:05"),
			balance.ExpiresAt.Format("2006-01-02 15:04:05"))),
	)
	replyCard(ctx, msgId, newCard)
}

func SendRoleTagsCard(ctx context.Context,
	sessionId *string, msgId *string, roleTags []string) {
	newCard, _ := newSendCard(
		withHeader("🛖 请选择角色类别", larkcard.TemplateIndigo),
		withRoleTagsBtn(sessionId, roleTags...),
		withNote("提醒：选择角色所属分类，以便我们为您推荐更多相关角色。"))
	err := replyCard(ctx, msgId, newCard)
	if err != nil {
		logger.Errorf("选择角色出错 %v", err)
	}
}

func SendRoleListCard(ctx context.Context,
	sessionId *string, msgId *string, roleTag string, roleList []string) {
	newCard, _ := newSendCard(
		withHeader("🛖 角色列表"+" - "+roleTag, larkcard.TemplateIndigo),
		withRoleBtn(sessionId, roleList...),
		withNote("提醒：选择内置场景，快速进入角色扮演模式。"))
	replyCard(ctx, msgId, newCard)
}

func SendAIModeListsCard(ctx context.Context,
	sessionId *string, msgId *string, aiModeStrs []string) {
	newCard, _ := newSendCard(
		withHeader("🤖 发散模式选择", larkcard.TemplateIndigo),
		withAIModeBtn(sessionId, aiModeStrs),
		withNote("提醒：选择内置模式，让AI更好的理解您的需求。"))
	replyCard(ctx, msgId, newCard)
}

func sendOnProcessCard(ctx context.Context,
	sessionId *string, msgId *string, ifNewTopic bool) (*string,
	error) {
	var newCard string
	// 使用流式卡片创建初始"正在处理"状态的卡片
	if ifNewTopic {
		newCard, _ = newStreamingCard(
			withHeader("🌟 已开启新的话题", larkcard.TemplateBlue),
			withNote("正在思考，请稍等..."))
	} else {
		newCard, _ = newStreamingCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withNote("正在思考，请稍等..."))
	}

	id, err := replyCardWithBackId(ctx, msgId, newCard)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func updateTextCard(ctx context.Context, msg string,
	msgId *string, ifNewTopic bool) error {
	var newCard string
	// 使用流式卡片更新中间状态
	if ifNewTopic {
		newCard, _ = newStreamingCard(
			withHeader("🌟 已开启新的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withNote("正在生成，请稍等..."))
	} else {
		newCard, _ = newStreamingCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withNote("正在生成，请稍等..."))
	}
	err := PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}
func updateFinalCard(
	ctx context.Context,
	msg string,
	msgId *string,
	ifNewSession bool,
) error {
	var newCard string
	// 从msgId获取sessionId，这里简化处理，实际应该从上下文获取
	sessionId := msgId // 简化处理
	
	if ifNewSession {
		newCard, _ = newSendCard(
			withHeader("🌟 已开启新的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withModelSwitchButtons(sessionId),
			withNote("已完成，点击按钮可切换模型回答，或继续提问保持话题连贯。"))
	} else {
		newCard, _ = newSendCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withModelSwitchButtons(sessionId),
			withNote("已完成，点击按钮可切换模型回答，或继续提问保持话题连贯。"))
	}
	err := PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}

// updateFinalCardWithSession 带sessionId的更新最终卡片
func updateFinalCardWithSession(
	ctx context.Context,
	msg string,
	msgId *string,
	sessionId *string,
	ifNewSession bool,
) error {
	var newCard string
	
	if ifNewSession {
		newCard, _ = newSendCard(
			withHeader("🌟 已开启新的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withModelSwitchButtons(sessionId),
			withNote("已完成，点击按钮可切换模型回答，或继续提问保持话题连贯。"))
	} else {
		newCard, _ = newSendCard(
			withHeader("🔃️ 上下文的话题", larkcard.TemplateBlue),
			withMainMd(msg),
			withModelSwitchButtons(sessionId),
			withNote("已完成，点击按钮可切换模型回答，或继续提问保持话题连贯。"))
	}
	err := PatchCard(ctx, msgId, newCard)
	if err != nil {
		return err
	}
	return nil
}

func newSendCardWithOutHeader(
	elements ...larkcard.MessageCardElement) (string, error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(true).
		Build()
	var aElementPool []larkcard.MessageCardElement
	aElementPool = append(aElementPool, elements...)
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

func PatchCard(ctx context.Context, msgId *string,
	cardContent string) error {
	client := initialization.GetLarkClient()

	// 按照飞书API规范执行PATCH更新
	resp, err := client.Im.Message.Patch(ctx, larkim.NewPatchMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewPatchMessageReqBodyBuilder().
			Content(cardContent).
			Build()).
		Build())

	// 处理网络错误
	if err != nil {
		return fmt.Errorf("飞书API调用失败: %v", err)
	}

	// 处理飞书服务端错误，包括API限流
	if !resp.Success() {
		// 检查是否是API限流错误 (通常返回码为99991400)
		if resp.Code == 99991400 {
			return fmt.Errorf("飞书API限流，请降低更新频率: %s", resp.Msg)
		}
		return fmt.Errorf("飞书服务端错误 [%v]: %s (RequestId: %v)", 
			resp.Code, resp.Msg, resp.RequestId())
	}
	return nil
}

func replyCardWithBackId(ctx context.Context,
	msgId *string,
	cardContent string,
) (*string, error) {
	client := initialization.GetLarkClient()
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}

	//ctx = context.WithValue(ctx, "SendMsgId", *resp.Data.MessageId)
	//SendMsgId := ctx.Value("SendMsgId")
	//pp.Println(SendMsgId)
	return resp.Data.MessageId, nil
}
