#!/bin/bash
set -e

# ============================================
# 初始化脚本 - 服务器首次部署时执行一次
# ============================================

# 配置
APP_DIR="/www/wwwroot/haseman-api"
GIT_REPO="https://你的代码仓库地址.git"
SERVICE_NAME="haseman-api"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}  服务器初始化${NC}"
echo -e "${GREEN}==========================================${NC}"

# 1. 安装必要软件
echo -e "${YELLOW}[1/7] 安装 Node.js...${NC}"
curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
sudo yum install -y nodejs
node -v

echo -e "${YELLOW}[2/7] 安装 PM2...${NC}"
sudo npm install -g pm2

echo -e "${YELLOW}[3/7] 安装 Git...${NC}"
sudo yum install -y git

# 2. 创建目录
echo -e "${YELLOW}[4/7] 创建应用目录...${NC}"
sudo mkdir -p $APP_DIR
cd $APP_DIR

# 3. 克隆代码（如果目录为空）
if [ ! "$(ls -A)" ]; then
    echo -e "${YELLOW}克隆代码仓库...${NC}"
    git clone $GIT_REPO .
else
    echo -e "${YELLOW}目录已存在，跳过克隆${NC}"
fi

# 4. 配置 .env 文件
echo -e "${YELLOW}[5/7] 配置环境变量...${NC}"
if [ ! -f backend/.env ]; then
    sudo cp backend/.env.example backend/.env
    echo -e "${YELLOW}请编辑 backend/.env 配置文件${NC}"
fi

# 5. 首次编译
echo -e "${YELLOW}[6/7] 首次编译...${NC}"
cd backend
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o $SERVICE_NAME main.go
chmod +x $SERVICE_NAME
if file "$SERVICE_NAME" | grep -qE 'PE32|MS-DOS'; then
    echo "编译结果仍是 Windows 二进制，停止启动"
    exit 1
fi

# 6. 启动服务
echo -e "${YELLOW}[7/7] 启动服务...${NC}"
pm2 start ./$SERVICE_NAME --name $SERVICE_NAME
pm2 save
pm2 startup

echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}  初始化完成！${NC}"
echo -e "${GREEN}==========================================${NC}"

pm2 status
