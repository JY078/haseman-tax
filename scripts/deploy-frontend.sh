#!/bin/bash
set -e

# Configuration
OSS_REGION="cn-hangzhou"
WEB_BUCKET="haseman-web-prod"
ADMIN_BUCKET="haseman-admin-prod"

echo "=========================================="
echo "  Deploying Frontend to OSS"
echo "=========================================="

# Install ossutil if not exists
if ! command -v ossutil &> /dev/null; then
    echo "[1/5] Installing ossutil..."
    wget -O /tmp/ossutil64.gz https://help-static-aliyun-doc.aliyunicloud.com/doc仁云文档-oss工具/ossutil_v1.7.19.zip
    gunzip -f /tmp/ossutil64.gz
    chmod +x /tmp/ossutil64
    sudo mv /tmp/ossutil64 /usr/local/bin/ossutil
fi

# Configure ossutil (一次性配置)
echo "[2/5] Configuring ossutil..."
ossutil config \
  -e oss-cn-hangzhou.aliyuncs.com \
  -i ${ALIYUN_ACCESS_KEY_ID} \
  -k ${ALIYUN_ACCESS_KEY_SECRET}

# Build frontend
echo "[3/5] Building frontend..."
cd frontend
npm install
npm run build

echo "[4/5] Uploading frontend to OSS..."
ossutil rm -rf oss://${WEB_BUCKET}/ --force
ossutil cp -r dist oss://${WEB_BUCKET}/ --force

# Build admin web
echo "[5/5] Building and uploading admin web..."
cd ../web
npm install
npm run build
ossutil rm -rf oss://${ADMIN_BUCKET}/ --force
ossutil cp -r dist oss://${ADMIN_BUCKET}/ --force

echo "=========================================="
echo "  Frontend deployment complete!"
echo "=========================================="
