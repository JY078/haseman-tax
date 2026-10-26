#!/usr/bin/env bash
# Usage: sudo bash scripts/install-rembg-native.sh [Python executable]
set -euo pipefail
if [ "$(id -u)" -ne 0 ]; then
  echo '请使用 root 或 sudo 执行此安装脚本。' >&2
  exit 1
fi
command -v systemctl >/dev/null || { echo '需要支持 systemd 的 Linux 系统。' >&2; exit 1; }
PYTHON_BIN="${1:-${AI_AGENT_PYTHON:-python3}}"
command -v "$PYTHON_BIN" >/dev/null || { echo "未找到 $PYTHON_BIN，请传入 Python 3.11 或 3.12 的绝对路径，或设置 AI_AGENT_PYTHON。" >&2; exit 1; }
"$PYTHON_BIN" -c 'import sys; assert sys.version_info[:2] in ((3, 11), (3, 12)), "请使用 Python 3.11 或 3.12"; import venv; print("使用 Python:", sys.executable, sys.version.split()[0])'
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SOURCE_DIR=""
for candidate in "$SCRIPT_DIR/../backend/rembg-service" "$SCRIPT_DIR/rembg-service" "$SCRIPT_DIR"; do
  if [ -f "$candidate/requirements.txt" ] && [ -f "$candidate/app.py" ]; then
    SOURCE_DIR="$(cd "$candidate" && pwd)"
    break
  fi
done
if [ -z "$SOURCE_DIR" ]; then
  echo '安装文件不完整：未找到 rembg-service/requirements.txt 和 app.py。' >&2
  echo "请将完整 rembg-service 文件夹上传至 $SCRIPT_DIR/rembg-service，再执行本脚本。" >&2
  exit 1
fi
echo "服务源码目录：$SOURCE_DIR"
SERVICE_DIR=/opt/haseman-rembg
MODEL_DIR=/www/wwwroot/haseman-api/rembg-models
MODEL_FILE=birefnet-general-efficient.onnx
SOURCE_MODEL="$SOURCE_DIR/$MODEL_FILE"
[ -f "$SOURCE_MODEL" ] || SOURCE_MODEL="$MODEL_DIR/$MODEL_FILE"
if [ ! -f "$SOURCE_MODEL" ]; then
  echo "请先上传离线模型至 $SOURCE_DIR/$MODEL_FILE 或 $MODEL_DIR/$MODEL_FILE。" >&2
  exit 1
fi
# Check before changing the running installation; never retry a blocked download.
"$PYTHON_BIN" - "$SOURCE_MODEL" <<'PY'
import hashlib, sys
with open(sys.argv[1], 'rb') as model:
    if hashlib.file_digest(model, 'sha256').hexdigest() != 'b6cc66609e98f76e66c0d7df0eea477d80f08df4cf59e86540b8bd52fe0d7546':
        sys.exit('模型校验失败，请重新上传完整的 birefnet-general-efficient.onnx')
PY
UNIT=/etc/systemd/system/haseman-rembg.service
if [ -f "$UNIT" ] && ! grep -q '^# Managed by HaiErMan install-rembg-native.sh$' "$UNIT"; then
  echo "已有非本脚本管理的 $UNIT，请先检查，未覆盖。" >&2
  exit 1
fi
if ! id haseman-rembg >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/haseman-rembg --shell /usr/sbin/nologin haseman-rembg
fi
install -d -m 755 "$SERVICE_DIR"
install -d -m 755 -o haseman-rembg -g haseman-rembg \
  /var/lib/haseman-rembg \
  /var/lib/haseman-rembg/models \
  /var/lib/haseman-rembg/numba
install -d -m 755 -o haseman-rembg -g haseman-rembg "$MODEL_DIR"
if [ "$SOURCE_MODEL" != "$MODEL_DIR/$MODEL_FILE" ]; then
  install -m 644 "$SOURCE_MODEL" "$MODEL_DIR/$MODEL_FILE.new"
  mv -f "$MODEL_DIR/$MODEL_FILE.new" "$MODEL_DIR/$MODEL_FILE"
fi
chmod 644 "$MODEL_DIR/$MODEL_FILE"
"$PYTHON_BIN" -m venv "$SERVICE_DIR/venv"
"$SERVICE_DIR/venv/bin/python" -m pip install -r "$SOURCE_DIR/requirements.txt"
install -m 644 "$SOURCE_DIR/app.py" "$SERVICE_DIR/app.py"
cat > "$UNIT" <<'UNIT'
# Managed by HaiErMan install-rembg-native.sh
[Unit]
Description=Haseman CPU background removal
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
User=haseman-rembg
Group=haseman-rembg
WorkingDirectory=/opt/haseman-rembg
Environment=PYTHONUNBUFFERED=1
Environment=U2NET_HOME=/www/wwwroot/haseman-api/rembg-models
Environment=NUMBA_CACHE_DIR=/var/lib/haseman-rembg/numba
Environment=OMP_NUM_THREADS=3
Environment=NUMBA_NUM_THREADS=3
Environment=OPENBLAS_NUM_THREADS=3
Environment=MKL_NUM_THREADS=3
ExecStart=/opt/haseman-rembg/venv/bin/python -m uvicorn app:app --host 127.0.0.1 --port 8011 --workers 1
Restart=on-failure
RestartSec=10
CPUQuota=300%
MemoryMax=5637144576
MemorySwapMax=0
TasksMax=128
Nice=10
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/haseman-rembg
ReadWritePaths=/www/wwwroot/haseman-api/rembg-models

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable haseman-rembg
systemctl reset-failed haseman-rembg
systemctl restart haseman-rembg
echo '服务已启动，离线加载完整版模型。查看进度：journalctl -u haseman-rembg -f'
echo '就绪检查：curl -f http://127.0.0.1:8011/health'
echo '宝塔后端环境变量设置 REMBG_SERVICE_URL=http://127.0.0.1:8011，然后重启后端。'
echo '必须同步更新并重新编译 Go 后端：旧后端只认 ISNet，会拒绝新模型的健康检查。'
echo '资源检查：systemctl show haseman-rembg -p CPUQuotaPerSecUSec -p MemoryMax -p MemorySwapMax'
echo 'MemorySwapMax 需要 cgroup v2；旧系统请确认限制实际生效。'
