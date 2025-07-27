#!/bin/bash

# 指定日志文件路径
LOGFILE="/home/mc/feishu-openai-master/code/mygoapp.log"

# 检查日志文件是否存在，存在则清空
if [ -f "$LOGFILE" ]; then
    > "$LOGFILE"  # 清空日志文件
    echo "$(date) - logfile.log has been cleared." >> /home/mc/clear_log_history.log  # 记录清理历史
else
    echo "$(date) - logfile.log does not exist." >> /home/mc/clear_log_history.log
fi
