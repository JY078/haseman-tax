#!/bin/bash
# STP 六面图渲染功能集成测试
#
# 前置条件：
#   1. Go 后端运行在 http://localhost:8080
#   2. Python Worker 已启动并连接 Redis
#   3. 已登录并获取 TOKEN（见下方设置）
#   4. 测试用 STP 文件放在 ./test-data/sample.stp
#
# 使用方法：
#   chmod +x stp-render-test.sh
#   export TOKEN="your_jwt_token_here"
#   ./stp-render-test.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_FILE="${TEST_FILE:-./test-data/sample.stp}"
POLL_INTERVAL=3
POLL_MAX_ATTEMPTS=60

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    cat <<EOF
用法: $0 [选项]

选项:
  -u, --url URL       API 地址 (默认: http://localhost:8080)
  -f, --file FILE     STP 测试文件路径 (默认: ./test-data/sample.stp)
  -t, --token TOKEN   JWT 访问令牌 (也可通过环境变量 TOKEN 设置)
  -p, --product-id ID 测试用产品 ID (默认: 1)
  -h, --help          显示帮助信息

示例:
  TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... $0 --product-id 5
EOF
    exit 0
}

# 解析参数
PRODUCT_ID=1
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)       BASE_URL="$2"; shift 2 ;;
        -f|--file)      TEST_FILE="$2"; shift 2 ;;
        -t|--token)     TOKEN="$2"; shift 2 ;;
        -p|--product-id) PRODUCT_ID="$2"; shift 2 ;;
        -h|--help)      usage ;;
        *)               log_error "未知参数: $1"; exit 1 ;;
    esac
done

if [[ -z "$TOKEN" ]]; then
    log_error "请设置 TOKEN 环境变量或使用 -t 参数"
    echo "提示: 登录后获取 TOKEN"
    echo "  TOKEN=\$(curl -s -X POST '${BASE_URL}/api/login' \\
    -H 'Content-Type: application/json' \\
    -d '{"email":"your@email.com","password":"xxx"}' | jq -r '.data.token')"
    exit 1
fi

log_info "=== STP 六面图渲染集成测试 ==="
log_info "API: $BASE_URL"
log_info "产品 ID: $PRODUCT_ID"
log_info "测试文件: $TEST_FILE"

# 校验测试文件
if [[ ! -f "$TEST_FILE" ]]; then
    log_error "测试文件不存在: $TEST_FILE"
    log_info "请准备一个有效的 STP/STEP 文件"
    exit 1
fi

API="${BASE_URL}/api"

# ─────────────────────────────────────────────
# 测试 1: 上传有效 STP 文件
# ─────────────────────────────────────────────
log_info ""
log_info "[1/4] 测试上传有效 STP 文件"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API}/product-prompts/${PRODUCT_ID}/stp" \
    -H "Authorization: Bearer ${TOKEN}" \
    -F "file=@${TEST_FILE};type=application/octet-stream")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

CODE=$(echo "$BODY" | jq -r '.code // "null"')
if [[ "$CODE" == "0" ]]; then
    log_info "上传成功"
    TASK_ID=$(echo "$BODY" | jq -r '.data.task_id')
    STP_FILE_URL=$(echo "$BODY" | jq -r '.data.stp_file_url')
    RENDER_STATUS=$(echo "$BODY" | jq -r '.data.render_status')
    log_info "  Task ID: $TASK_ID"
    log_info "  STP URL: $STP_FILE_URL"
    log_info "  初始状态: $RENDER_STATUS"
else
    log_error "上传失败: $(echo "$BODY" | jq -r '.error // .message')"
    log_error "HTTP 状态码: $HTTP_CODE"
    exit 1
fi

# ─────────────────────────────────────────────
# 测试 2: 轮询渲染状态直到完成
# ─────────────────────────────────────────────
log_info ""
log_info "[2/4] 测试轮询渲染状态 (最多 ${POLL_MAX_ATTEMPTS} 次轮询)"

ATTEMPT=0
LAST_STATUS=""
while (( ATTEMPT < POLL_MAX_ATTEMPTS )); do
    ((ATTEMPT++))
    STATUS_RESP=$(curl -s -X GET "${API}/product-prompts/${PRODUCT_ID}/render-status" \
        -H "Authorization: Bearer ${TOKEN}")

    CURRENT_STATUS=$(echo "$STATUS_RESP" | jq -r '.data.render_status // "unknown"')
    if [[ "$CURRENT_STATUS" != "$LAST_STATUS" ]]; then
        log_info "  [$ATTEMPT] 状态: $CURRENT_STATUS"
        LAST_STATUS="$CURRENT_STATUS"
    fi

    if [[ "$CURRENT_STATUS" == "completed" ]]; then
        log_info "渲染完成"
        break
    elif [[ "$CURRENT_STATUS" == "failed" ]]; then
        ERROR_MSG=$(echo "$STATUS_RESP" | jq -r '.data.render_error // "未知错误"')
        log_error "渲染失败: $ERROR_MSG"
        exit 1
    fi

    sleep "$POLL_INTERVAL"
done

if (( ATTEMPT >= POLL_MAX_ATTEMPTS )); then
    log_warn "轮询超时，当前状态仍为: $CURRENT_STATUS"
fi

# ─────────────────────────────────────────────
# 测试 3: 验证六面图数量
# ─────────────────────────────────────────────
log_info ""
log_info "[3/4] 验证六面图"

SIX_VIEWS_COUNT=$(echo "$STATUS_RESP" | jq -r '.data.six_views | length')
EXPECTED_VIEWS=6

if [[ "$SIX_VIEWS_COUNT" == "$EXPECTED_VIEWS" ]]; then
    log_info "六面图数量正确: $SIX_VIEWS_COUNT"
else
    log_warn "六面图数量异常: $SIX_VIEWS_COUNT (期望: $EXPECTED_VIEWS)"
fi

log_info "六面图详情:"
echo "$STATUS_RESP" | jq -r '.data.six_views[] | "  - \(.description) -> \(.url)"' 2>/dev/null || true

# ─────────────────────────────────────────────
# 测试 4: 验证六面图 URL 可访问
# ─────────────────────────────────────────────
log_info ""
log_info "[4/4] 验证六面图 URL 可访问性"

VALID_COUNT=0
INVALID_COUNT=0
URLS=$(echo "$STATUS_RESP" | jq -r '.data.six_views[].url')
for url in $URLS; do
    if [[ -z "$url" ]]; then
        ((INVALID_COUNT++))
        continue
    fi

    # 转换 OSS URL 为可访问的 URL (假设使用代理或公网 OSS)
    # 如果是 oss:// 前缀，尝试通过 Go 代理访问
    if [[ "$url" == oss://* ]]; then
        PROXY_URL="${BASE_URL}/api/proxy?url=${url}"
    else
        PROXY_URL="$url"
    fi

    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$PROXY_URL")
    if [[ "$HTTP_CODE" == "200" ]]; then
        ((VALID_COUNT++))
    else
        log_warn "  URL 不可访问 (HTTP $HTTP_CODE): $url"
        ((INVALID_COUNT++))
    fi
done

log_info "可访问: $VALID_COUNT / $SIX_VIEWS_COUNT"

# ─────────────────────────────────────────────
# 测试 5: 错误处理测试 - 上传非 STP 文件
# ─────────────────────────────────────────────
log_info ""
log_info "[5/5] 测试错误处理 - 上传非 STP 文件"

ERROR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API}/product-prompts/${PRODUCT_ID}/stp" \
    -H "Authorization: Bearer ${TOKEN}" \
    -F "file=@${TEST_FILE};type=text/plain;filename=test.txt")

ERROR_CODE=$(echo "$ERROR_RESPONSE" | head -n1 | jq -r '.code // "null"')
ERROR_BODY=$(echo "$ERROR_RESPONSE" | sed '$d')

# 尝试不同的响应格式
if [[ "$ERROR_CODE" == "null" ]]; then
    ERROR_MSG=$(echo "$ERROR_BODY" | jq -r '.error // .message // "unknown"')
else
    ERROR_MSG=$(echo "$ERROR_BODY" | jq -r '.error // .message // "unknown"')
fi

if [[ "$ERROR_CODE" == "0" ]]; then
    log_warn "意外：非 STP 文件上传成功 (应该被拒绝)"
elif [[ -n "$ERROR_MSG" && "$ERROR_MSG" != "null" && "$ERROR_MSG" != "unknown" ]]; then
    log_info "正确拒绝非 STP 文件: $ERROR_MSG"
else
    log_warn "无法验证错误响应格式"
fi

# ─────────────────────────────────────────────
# 总结
# ─────────────────────────────────────────────
log_info ""
log_info "=== 测试完成 ==="
log_info "渲染状态: $(echo "$STATUS_RESP" | jq -r '.data.render_status')"
log_info "六面图数量: $SIX_VIEWS_COUNT"
log_info "可访问图片: $VALID_COUNT / $SIX_VIEWS_COUNT"

if [[ "$SIX_VIEWS_COUNT" == "$EXPECTED_VIEWS" && "$VALID_COUNT" -ge "$((EXPECTED_VIEWS/2))" ]]; then
    log_info "整体结果: 通过"
    exit 0
else
    log_warn "整体结果: 部分通过"
    exit 1
fi
