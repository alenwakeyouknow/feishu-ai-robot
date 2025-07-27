#!/bin/bash

# 飞书OpenAI机器人管理脚本

PROJECT_DIR="$HOME/code"
APP_NAME="feishu-chatgpt"
PID_FILE="$PROJECT_DIR/$APP_NAME.pid"
LOG_FILE="$PROJECT_DIR/app.log"

case "$1" in
    start)
        echo "启动飞书OpenAI机器人..."
        cd $PROJECT_DIR
        if [ -f $PID_FILE ]; then
            echo "机器人已在运行 (PID: $(cat $PID_FILE))"
            exit 1
        fi
        nohup ./$APP_NAME > $LOG_FILE 2>&1 &
        echo $! > $PID_FILE
        echo "机器人已启动 (PID: $!)"
        ;;
    stop)
        echo "停止飞书OpenAI机器人..."
        if [ -f $PID_FILE ]; then
            PID=$(cat $PID_FILE)
            kill $PID
            rm $PID_FILE
            echo "机器人已停止"
        else
            echo "机器人未运行"
        fi
        ;;
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
    status)
        if [ -f $PID_FILE ]; then
            PID=$(cat $PID_FILE)
            if ps -p $PID > /dev/null; then
                echo "机器人正在运行 (PID: $PID)"
            else
                echo "PID文件存在但进程不在运行"
                rm $PID_FILE
            fi
        else
            echo "机器人未运行"
        fi
        ;;
    logs)
        echo "查看最新50行日志:"
        tail -50 $LOG_FILE
        ;;
    follow)
        echo "实时查看日志 (Ctrl+C退出):"
        tail -f $LOG_FILE
        ;;
    build)
        echo "重新编译项目..."
        cd $PROJECT_DIR
        go build -o $APP_NAME main.go
        echo "编译完成"
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs|follow|build}"
        echo ""
        echo "命令说明:"
        echo "  start   - 启动机器人"
        echo "  stop    - 停止机器人"
        echo "  restart - 重启机器人"
        echo "  status  - 查看运行状态"
        echo "  logs    - 查看最新日志"
        echo "  follow  - 实时跟踪日志"
        echo "  build   - 重新编译项目"
        exit 1
        ;;
esac