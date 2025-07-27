package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"start-feishubot/handlers"
	"start-feishubot/initialization"
	"start-feishubot/logger"

	"github.com/gin-gonic/gin"
	sdkginext "github.com/larksuite/oapi-sdk-gin"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/spf13/pflag"
	"start-feishubot/services/openai"
)

// 解密飞书加密数据 (根据飞书官方文档的AES-256-CBC解密算法)
func decryptFeishuData(encryptKey, encryptStr string) ([]byte, error) {
	// 1. 使用 SHA256 对 Encrypt Key 进行哈希，得到密钥 key
	hash := sha256.Sum256([]byte(encryptKey))
	key := hash[:]

	// 2. Base64解码加密数据
	data, err := base64.StdEncoding.DecodeString(encryptStr)
	if err != nil {
		return nil, err
	}

	// 3. 提取IV（前16字节）和加密内容
	if len(data) < 16 {
		return nil, err
	}
	iv := data[:16]
	encrypted := data[16:]

	// 4. AES-256-CBC 解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// 5. 去除PKCS7Padding
	padding := int(decrypted[len(decrypted)-1])
	return decrypted[:len(decrypted)-padding], nil
}

// 处理卡片回调的URL验证和业务逻辑
func handleCardCallback(config *initialization.Config, cardHandler *larkcard.CardActionHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取原始请求体
		bodyBytes, err := c.GetRawData()
		if err != nil {
			logger.Errorf("读取请求体失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
			return
		}

		var body map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			logger.Errorf("解析JSON失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		// 检查是否为加密的URL验证请求
		if encryptStr, exists := body["encrypt"]; exists {
			logger.Info("🔐 收到飞书加密的URL验证请求")
			
			// 解密数据
			decrypted, err := decryptFeishuData(config.FeishuAppEncryptKey, encryptStr.(string))
			if err != nil {
				logger.Errorf("解密失败: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Decrypt failed"})
				return
			}

			// 解析解密后的内容
			var decryptedBody map[string]interface{}
			if err := json.Unmarshal(decrypted, &decryptedBody); err != nil {
				logger.Errorf("解析解密数据失败: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid decrypted data"})
				return
			}

			// 检查是否为URL验证
			if reqType, exists := decryptedBody["type"]; exists && reqType == "url_verification" {
				logger.Info("✅ 确认为URL验证请求")
				
				// 验证token
				if token, exists := decryptedBody["token"]; exists {
					if token != config.FeishuAppVerificationToken {
						logger.Errorf("❌ Token验证失败")
						c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
						return
					}
				}

				// 返回challenge
				if challenge, exists := decryptedBody["challenge"]; exists {
					logger.Info("🎉 URL验证成功，返回challenge: %v", challenge)
					c.JSON(http.StatusOK, gin.H{"challenge": challenge})
					return
				}
			}
		}

		// 检查明文URL验证请求
		if reqType, exists := body["type"]; exists && reqType == "url_verification" {
			logger.Info("🔐 收到飞书明文URL验证请求")
			
			if token, exists := body["token"]; exists {
				if token != config.FeishuAppVerificationToken {
					logger.Errorf("❌ Token验证失败")
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
					return
				}
			}

			if challenge, exists := body["challenge"]; exists {
				logger.Info("🎉 URL验证成功，返回challenge: %v", challenge)
				c.JSON(http.StatusOK, gin.H{"challenge": challenge})
				return
			}
		}

		// 处理正常的卡片动作回调
		logger.Info("📋 处理卡片动作回调")
		
		// 重建请求体给SDK处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Request.ContentLength = int64(len(bodyBytes))
		
		// 使用飞书SDK处理卡片动作
		sdkginext.NewCardActionHandlerFunc(cardHandler)(c)
	}
}

// 创建包装的卡片处理器，添加日志记录
func createCardHandlerWithLogging() func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
	return func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
		logger.Info("📋 收到卡片动作回调")
		logger.Debugf("卡片动作详情: RequestId=%s, OpenID=%s", 
			cardAction.RequestId(), cardAction.OpenID)
		
		// 调用原始的卡片处理器
		return handlers.CardHandler()(ctx, cardAction)
	}
}

func main() {
	initialization.InitRoleList()
	pflag.Parse()
	config := initialization.GetConfig()
	initialization.LoadLarkClient(*config)
	gpt := openai.NewChatGPT(*config)
	handlers.InitHandlers(gpt, *config)

	eventHandler := dispatcher.NewEventDispatcher(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey).
		OnP2MessageReceiveV1(handlers.Handler).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			logger.Debugf("收到请求 %v", event.RequestURI)
			return handlers.ReadHandler(ctx, event)
		})

	// 根据飞书官方文档，创建卡片处理器应该能自动处理URL验证
	// 但是为了确保URL验证正确，我们需要手动处理
	cardHandler := larkcard.NewCardActionHandler(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey,
		createCardHandlerWithLogging())

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.POST("/webhook/event",
		sdkginext.NewEventHandlerFunc(eventHandler))
	r.POST("/webhook/card",
		handleCardCallback(config, cardHandler))

	if err := initialization.StartServer(*config, r); err != nil {
		logger.Fatalf("failed to start server: %v", err)
	}
}
