#!/bin/bash
set -e

# ============================================
# 后端部署脚本 - 从代码仓库拉取并部署
# ============================================

# 配置
APP_DIR="/www/wwwroot/haseman-api"
GIT_REPO="https://你的代码仓库地址.git"
BRANCH="master"
SERVICE_NAME="haseman-api"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}  开始部署后端服务${NC}"
echo -e "${GREEN}==========================================${NC}"

# 进入应用目录
cd $APP_DIR || exit 1

# 1. 拉取最新代码
echo -e "${YELLOW}[1/5] 拉取最新代码...${NC}"
git pull origin $BRANCH

# 2. 安装 Go 依赖
echo -e "${YELLOW}[2/5] 安装 Go 依赖...${NC}"
cd backend
go mod tidy

# 3. 编译
echo -e "${YELLOW}[3/5] 编译项目...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o $SERVICE_NAME main.go
chmod +x $SERVICE_NAME
if file "$SERVICE_NAME" | grep -qE 'PE32|MS-DOS'; then
    echo "编译结果仍是 Windows 二进制，停止部署"
    exit 1
fi

# Start the CPU cutout service before switching the backend to the new version.
bash "$APP_DIR/scripts/start-rembg.sh"

# 4. 重启服务
echo -e "${YELLOW}[4/5] 重启服务...${NC}"

# 停止旧进程
pm2 stop $SERVICE_NAME 2>/dev/null || true
pm2 delete $SERVICE_NAME 2>/dev/null || true

# 启动新进程
pm2 start ./$SERVICE_NAME --name $SERVICE_NAME

# 5. 保存并设置开机自启
echo -e "${YELLOW}[5/5] 保存进程状态...${NC}"
pm2 save
pm2 startup

echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}  部署完成！${NC}"
echo -e "${GREEN}==========================================${NC}"

# 显示状态
pm2 status
