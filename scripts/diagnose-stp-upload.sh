#!/bin/bash
# STP Upload 路由诊断脚本
# 在服务器上运行，用于排查 /api/stp/model POST 请求无法到达 Go 应用的问题

set -e

API_HOST="127.0.0.1"
API_PORT="8080"
API_BASE="http://${API_HOST}:${API_PORT}/api"

echo "=========================================="
echo "STP Upload 路由诊断"
echo "=========================================="
echo ""

# 检查是否以 root 运行
if [ "$EUID" -ne 0 ]; then
    echo "提示: 某些检查可能需要 root 权限"
    echo ""
fi

echo "[1] 检查 Go 进程是否在运行..."
if pgrep -f "haseman-api" > /dev/null; then
    echo "    ✓ haseman-api 进程正在运行"
    ps aux | grep haseman-api | grep -v grep | head -1
else
    echo "    ✗ haseman-api 进程未运行"
fi
echo ""

echo "[2] 检查端口 8080 监听状态..."
if command -v ss &> /dev/null; then
    if ss -tlnp 2>/dev/null | grep -q ":8080"; then
        echo "    ✓ 端口 8080 正在监听"
        ss -tlnp | grep 8080
    else
        echo "    ✗ 端口 8080 未监听"
    fi
elif command -v netstat &> /dev/null; then
    if netstat -tlnp 2>/dev/null | grep -q ":8080"; then
        echo "    ✓ 端口 8080 正在监听"
    else
        echo "    ✗ 端口 8080 未监听"
    fi
else
    echo "    无法检查端口状态"
fi
echo ""

echo "[3] 测试已知的正常端点 (GET /api/stp/mesh-preview)..."
RESPONSE=$(curl -s -w "\n---HTTP_STATUS:%{http_code}---" "${API_BASE}/stp/mesh-preview?stp_url=test" 2>&1)
echo "$RESPONSE" | head -3
HTTP_CODE=$(echo "$RESPONSE" | grep -oP "HTTP_STATUS:\K[0-9]+" || echo "unknown")
echo "    HTTP状态码: $HTTP_CODE"
echo ""

echo "[4] 测试 STP Model 端点 (POST /api/stp/model，无文件)..."
RESPONSE=$(curl -s -w "\n---HTTP_STATUS:%{http_code}---" -X POST "${API_BASE}/stp/model" 2>&1)
echo "$RESPONSE" | head -3
HTTP_CODE=$(echo "$RESPONSE" | grep -oP "HTTP_STATUS:\K[0-9]+" || echo "unknown")
echo "    HTTP状态码: $HTTP_CODE"
if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "400" ]; then
    echo "    ✓ 请求已到达 Go 应用（需要认证或参数）"
else
    echo "    ✗ 请求未能到达 Go 应用"
fi
echo ""

echo "[5] 测试 STP Model 端点 (POST /api/stp/model，带 multipart/form-data)..."
TMPFILE="/tmp/test-stp-$(date +%s).txt"
echo "test content" > "$TMPFILE"
RESPONSE=$(curl -s -w "\n---HTTP_STATUS:%{http_code}---" -X POST \
    -F "file=@${TMPFILE}" \
    "${API_BASE}/stp/model" 2>&1)
echo "$RESPONSE" | head -3
HTTP_CODE=$(echo "$RESPONSE" | grep -oP "HTTP_STATUS:\K[0-9]+" || echo "unknown")
echo "    HTTP状态码: $HTTP_CODE"
rm -f "$TMPFILE"
echo ""

echo "[6] 检查 nginx 配置中是否有 stp/model 相关规则..."
NGINX_CONF_FILES=$(find /etc /www /usr/local/etc 2>/dev/null -name "nginx.conf" -o -name "*.conf" 2>/dev/null | head -20)
for conf in $NGINX_CONF_FILES; do
    if [ -f "$conf" ] && grep -qi "stp" "$conf" 2>/dev/null; then
        echo "    在 $conf 中找到 stp 相关配置:"
        grep -n "stp" "$conf" | head -10
    fi
done
echo ""

echo "[7] 检查 nginx/proxy 配置中是否有上传限制..."
for conf in $NGINX_CONF_FILES; do
    if [ -f "$conf" ]; then
        if grep -qi "client_max_body_size\|upload\|multipart" "$conf" 2>/dev/null; then
            echo "    在 $conf 中找到相关配置:"
            grep -in "client_max_body_size\|multipart" "$conf" | head -10
        fi
    fi
done
echo ""

echo "[8] 查看 Go 应用最近 50 行日志..."
pm2 logs haseman-api --nostream --lines 50 2>/dev/null | tail -50 || echo "    无法获取 pm2 日志"
echo ""

echo "[9] 查看是否有 [STP] UploadStp 相关的日志..."
pm2 logs haseman-api --nostream --lines 200 2>/dev/null | grep -i "UploadStp\|stp.*upload" || echo "    未找到相关日志"
echo ""

echo "=========================================="
echo "诊断完成"
echo "=========================================="
echo ""
echo "检查结果解读:"
echo "  - 如果 [4] 返回 401 或 400: 请求正常到达 Go 应用"
echo "  - 如果 [4] 返回 404: 路由未注册，需要重新部署"
echo "  - 如果 [4] 返回其他错误: 检查 nginx 配置"
echo ""
echo "如果所有测试都显示请求未到达 Go:"
echo "  1. 检查 nginx 中 /api/stp/model 的代理配置"
echo "  2. 检查是否有重定向规则"
echo "  3. 尝试手动 curl 本地地址测试"
echo "     curl -X POST http://127.0.0.1:8080/api/stp/model"