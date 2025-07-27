#!/bin/bash

# 飞书OpenAI机器人WSL部署脚本
# 使用方法: ./deploy-to-wsl.sh

SERVER_IP="192.168.200.106"
SERVER_USER="mc"
SERVER_PASS="morpho123"
PROJECT_NAME="feishu-openai"

echo "🚀 开始部署飞书OpenAI机器人到WSL Ubuntu 22.04"
echo "服务器: $SERVER_USER@$SERVER_IP"
echo ""

# 检查必要文件
if [ ! -f "feishu-openai-upgraded.tar.gz" ]; then
    echo "❌ 找不到项目打包文件，请先运行打包命令"
    exit 1
fi

echo "📤 传输文件到服务器..."
scp feishu-openai-upgraded.tar.gz deploy-config.yaml manage.sh $SERVER_USER@$SERVER_IP:~/

echo "🔧 在服务器上设置环境..."
ssh $SERVER_USER@$SERVER_IP << 'ENDSSH'
    # 解压项目
    echo "📂 解压项目文件..."
    tar -xzf feishu-openai-upgraded.tar.gz
    
    # 复制配置文件
    cp deploy-config.yaml code/config.yaml
    cp manage.sh code/
    chmod +x code/manage.sh
    
    # 进入项目目录
    cd code
    
    # 安装依赖并编译
    echo "🔨 编译项目..."
    go mod tidy
    go build -o feishu-chatgpt main.go
    
    # 创建systemd服务文件 (可选)
    echo "📋 创建系统服务..."
    sudo tee /etc/systemd/system/feishu-openai.service > /dev/null << EOF
[Unit]
Description=Feishu OpenAI Bot
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$HOME/code
ExecStart=$HOME/code/feishu-chatgpt
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
    
    # 重载systemd
    sudo systemctl daemon-reload
    
    echo "✅ 部署完成！"
    echo ""
    echo "📖 使用说明:"
    echo "1. 启动机器人: ./manage.sh start"
    echo "2. 查看状态: ./manage.sh status"
    echo "3. 查看日志: ./manage.sh logs"
    echo "4. 停止机器人: ./manage.sh stop"
    echo ""
    echo "🌐 或者使用系统服务:"
    echo "1. 启动: sudo systemctl start feishu-openai"
    echo "2. 开机自启: sudo systemctl enable feishu-openai"
    echo "3. 查看状态: sudo systemctl status feishu-openai"
    
ENDSSH

echo ""
echo "🎉 部署完成！"
echo "请按照以下步骤完成配置:"
echo ""
echo "1. 连接到服务器:"
echo "   ssh $SERVER_USER@$SERVER_IP"
echo ""
echo "2. 启动机器人:"
echo "   cd code && ./manage.sh start"
echo ""
echo "3. 设置公网访问 (选择一种方式):"
echo "   a) 使用cpolar: cpolar http 9000"
echo "   b) 配置端口转发到Windows主机"
echo ""
echo "4. 配置飞书回调地址:"
echo "   事件回调: https://your-domain/webhook/event"
echo "   卡片回调: https://your-domain/webhook/card"