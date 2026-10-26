#!/bin/sh
set -e

# 创建必要的目录
mkdir -p /app/logs /app/uploads /app/data

# 从环境变量或挂载的配置文件加载配置
# 如果存在 /app/config/.env 则使用它
if [ -f /app/config/.env ]; then
    cp /app/config/.env /app/.env
fi

# 启动应用
exec ./server