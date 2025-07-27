# 🚀 飞书AI机器人开源项目（不需要魔法）

> 基于 [ConnectAI-E/Feishu-OpenAI](https://github.com/ConnectAI-E/Feishu-OpenAI) 的深度优化版本，专为国内用户打造的飞书AI智能机器人。使用OpenRouter一个API密钥即可调用多个AI模型，无需魔法上网。

## ✨ 主要优化亮点

### 🎨 界面与交互优化
- **移除冗余复制按钮**，简化操作界面，提升用户体验
- **优化视觉元素**：👻️ → 🌟 提供更积极正面的视觉反馈
- **精简模型选择**，移除不必要按钮，聚焦核心功能

### 🔧 技术架构增强  
- **🌏 无需魔法上网**：使用OpenRouter统一API，国内直连访问
- **🔑 一键多模型**：一个API密钥调用GPT-4、Claude、Gemini等多个模型
- **修复KIMI模型配置**，更新为正确的OpenRouter端点格式
- **完善错误处理**，提升系统稳定性和异常恢复能力
- **代码质量优化**，清理未使用代码，提升维护性

## 🚀 核心功能

- **🤖 多模型支持**: OpenAI GPT-4o、Claude、Gemini、DeepSeek等
- **🎨 图像生成**: DALL·E-3 高质量图像创作
- **👁️ 图像理解**: GPT-4V 智能图像分析  
- **🎤 语音交互**: Whisper 语音转文字
- **💬 多话题对话**: 群聊和私聊上下文管理
- **🎭 角色扮演**: 内置多种AI角色模式
- **📊 使用统计**: Token用量追踪和管理

## 📦 快速开始

### 环境要求
- Go 1.19+
- 飞书/Lark开发者账号
- **OpenRouter API Key**（一个密钥调用所有模型，无需魔法）

### 1. 克隆项目
```bash
git clone https://github.com/alenwakeyouknow/feishu-ai-robot.git
cd feishu-ai-robot/code
```

### 2. 配置环境
复制并编辑配置文件：
```bash
cp config.yaml config.yaml.example
nano config.yaml
```

配置必要参数：
```yaml
# 飞书应用配置
APP_ID: cli_xxxxxxxxxxxxxxxx
APP_SECRET: your_app_secret_here
APP_ENCRYPT_KEY: your_encrypt_key_here
APP_VERIFICATION_TOKEN: your_verification_token_here

# ✨ OpenRouter配置（无需魔法，一个Key调用所有模型）
OPENAI_KEY: sk-or-v1-your_openrouter_api_key_here
API_URL: https://openrouter.ai/api/v1
USE_OPENROUTER: true

# 流式模式配置
STREAM_MODE: true
```

### 3. 构建运行
```bash
go mod tidy
go build -o feishubot .
./feishubot

```
### 4. 飞书详细配置
https://github.com/ConnectAI-E/Feishu-OpenAI?tab=readme-ov-file#%E8%AF%A6%E7%BB%86%E9%85%8D%E7%BD%AE%E6%AD%A5%E9%AA%A4

## 🛠️ 部署方式

### Docker 部署
```bash
docker build -t feishubot .
docker run -d --name feishubot -p 9000:9000 \
  --env APP_ID=your_app_id \
  --env APP_SECRET=your_app_secret \
  --env OPENAI_KEY=your_openrouter_key \
  feishubot
```

### 生产环境
推荐使用 systemd 服务部署，具体配置请参考项目文档。

## 🔧 飞书机器人配置

### 1. 创建应用
- 前往[飞书开放平台](https://open.feishu.cn/)创建应用
- 获取 APP_ID 和 APP_SECRET

### 2. 配置回调地址
- 事件回调: `http://your-domain:9000/webhook/event`
- 卡片回调: `http://your-domain:9000/webhook/card`

### 3. 权限配置
添加以下权限：
- `im:message` - 接收消息
- `im:message.group_at_msg` - 群聊@消息
- `im:message.p2p_msg` - 私聊消息
- `im:resource` - 图片文件资源

### 4. 事件订阅
配置以下事件：
- 机器人进群
- 接收消息  
- 消息已读

## 📊 优化对比

| 优化项目 | 原版本 | 优化版本 | 改进说明 |
|----------|--------|----------|----------|
| **API访问** | OpenAI官方API | **OpenRouter统一API** | **🌏 无需魔法，一键多模型** |
| 界面复杂度 | 多个复制按钮 | 简化界面 | 移除冗余功能 |
| 视觉体验 | 👻️ 图标 | 🌟 图标 | 更积极正面 |
| KIMI模型 | 配置错误 | 正确配置 | 提升稳定性 |
| 代码质量 | 冗余代码 | 精简优化 | 提升维护性 |

## 🛡️ 稳定性改进

- **🌏 无魔法访问**: 基于OpenRouter统一API，国内用户直连使用
- **🔑 多模型支持**: 一个API密钥调用GPT-4、Claude、Gemini、DeepSeek等
- **API端点修复**: 使用正确的OpenRouter模型格式
- **错误处理增强**: 完善异常捕获和恢复机制  
- **代码清理**: 移除未使用的imports和函数
- **配置优化**: 简化配置文件结构

## 🔒 开源协议

本项目采用 [GPL-3.0](LICENSE) 开源协议。

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发环境
```bash
git clone https://github.com/alenwakeyouknow/feishu-ai-robot.git
cd feishu-ai-robot/code
go mod download
go run main.go
```

### 测试
```bash
go test ./...
```
### 效果
<img width="1304" height="1502" alt="image" src="https://github.com/user-attachments/assets/a3149f95-c665-423b-add0-4336da773809" />
<img width="1100" height="340" alt="image" src="https://github.com/user-attachments/assets/4461a591-01a3-4a60-a665-777b9415a923" />


## 📞 支持与反馈

- 🐛 **Bug报告**: [提交Issue](https://github.com/alenwakeyouknow/feishu-ai-robot/issues)
- 💡 **功能建议**: [功能请求](https://github.com/alenwakeyouknow/feishu-ai-robot/issues)

## 该项目部署以及飞书开发联系
![bab1d0c2a2ec377e9984c5900a023f3b](https://github.com/user-attachments/assets/49fab9cb-8fea-4e04-a703-85b38950de39)

## 请作者喝杯咖啡
![641b563e51968c6a5f7a4699814bf2ce](https://github.com/user-attachments/assets/918a836c-4b03-4e60-bcf9-a1f8c09b45b0)


## 🙏 致谢

感谢 [ConnectAI-E/Feishu-OpenAI](https://github.com/ConnectAI-E/Feishu-OpenAI) 提供的优秀基础框架。

本项目在原有基础上进行了专项优化，专注于：
- ✅ 界面简化与用户体验提升
- ✅ 模型配置修复与稳定性增强  
- ✅ 代码质量改进与维护性提升
- ✅ 功能精简与性能优化

---

**🌟 如果这个项目对你有帮助，请给我们一个Star！**
