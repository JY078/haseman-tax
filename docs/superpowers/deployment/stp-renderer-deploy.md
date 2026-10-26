# STP 渲染 Worker 部署指南

## 概述

STP 渲染 Worker 是一个独立的 Python 进程，负责从 Redis 队列消费 STP 文件渲染任务，使用 OpenCASCADE (OCCT) 生成产品六面图。

## 环境要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| Python | 3.10+ | 3.10+ |
| 内存 | 4 GB | 8 GB+ |
| 磁盘 | 2 GB 可用 | SSD，5 GB+ |
| Redis | 3.2+ | 6.0+ |
| 网络 | 常规互联网连接 | 稳定的企业网络 |

## 前置依赖

### 1. Go 后端已部署

确保以下 API 可用：

- `POST /api/product-prompts/:id/stp` — 上传 STP 文件
- `GET /api/product-prompts/:id/render-status` — 查询渲染状态
- `POST /api/internal/stp-render/complete` — Worker 回调

### 2. Redis 已配置

确认 Go 后端的 Redis 配置（`backend/internal/cache/redis.go`）已连接同一 Redis 实例。

### 3. OSS 已配置

Worker 和 Go 后端使用同一个 OSS Bucket，用于存储 STP 文件和渲染结果图片。

---

## 快速开始

### Step 1: 安装 Python 依赖

```bash
cd backend/stp-renderer
pip install -r requirements.txt
```

核心依赖：

```
opencascade >= 7.7.0   # STEP 文件解析和 3D 渲染
numpy >= 1.24.0        # 数学运算
Pillow >= 10.0.0       # 图像处理
oss2 >= 2.18.0          # 阿里云 OSS SDK
redis >= 5.0.0          # Redis 客户端
requests >= 2.31.0      # HTTP 请求
```

### Step 2: 配置环境变量

```bash
# Redis 配置
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_DB=0

# OSS 配置（与 Go 后端使用相同的凭证）
export ALIBABA_CLOUD_ACCESS_KEY_ID=your_key_id
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=your_key_secret
export OSS_BUCKET=advertising-image
export OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
export OSS_DOMAIN=https://your-cdn-domain.com    # 可选，CDN 域名

# Go 后端 API（Worker 回调地址）
export GO_API_BASE_URL=http://localhost:8080
```

> **注意：** 如果 Go 后端和 Worker 部署在不同服务器，请将 `GO_API_BASE_URL` 替换为 Go 后端可被 Worker 访问的地址（内网 IP 或公网域名）。

### Step 3: 启动 Worker

```bash
# 前台运行（测试用）
python main.py

# 后台运行
nohup python main.py > worker.log 2>&1 &

# 使用 supervisor 或 systemd 管理（推荐生产环境）
```

---

## 生产环境部署

### 使用 systemd 管理

创建服务文件 `/etc/systemd/system/stp-renderer.service`：

```ini
[Unit]
Description=STP Renderer Worker
After=network.target redis.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/app/backend/stp-renderer
Environment="REDIS_HOST=localhost"
Environment="REDIS_PORT=6379"
Environment="REDIS_DB=0"
Environment="OSS_ACCESS_KEY=your_key"
Environment="OSS_ACCESS_SECRET=your_secret"
Environment="OSS_BUCKET=advertising-image"
Environment="OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com"
Environment="GO_API_BASE_URL=http://localhost:8080"
ExecStart=/usr/bin/python3 /opt/app/backend/stp-renderer/main.py
Restart=always
RestartSec=10
StandardOutput=append:/var/log/stp-renderer/worker.log
StandardError=append:/var/log/stp-renderer/worker.log

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable stp-renderer
sudo systemctl start stp-renderer
sudo systemctl status stp-renderer
```

### 使用 supervisor 管理

创建配置 `/etc/supervisor/conf.d/stp-renderer.conf`：

```ini
[program:stp-renderer]
command=python main.py
directory=/opt/app/backend/stp-renderer
user=www-data
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/var/log/stp-renderer/worker.log
environment=REDIS_HOST="localhost",REDIS_PORT="6379",OSS_ACCESS_KEY="your_key",OSS_ACCESS_SECRET="your_secret"
```

```bash
supervisorctl reread
supervisorctl add stp-renderer
supervisorctl start stp-renderer
```

---

## 高并发配置

### 多实例部署

一个 Worker 串行处理任务。40 人团队，建议部署 **2-3 个 Worker 实例**：

```bash
# 启动多个实例（自动分配不同端口无需配置，Redis 队列共享）
for i in {1..3}; do
    nohup python main.py > worker_${i}.log 2>&1 &
done
```

使用 systemd 时，复制多份 service 或使用模板：

```ini
# /etc/systemd/system/stp-renderer@.service
[Unit]
Description=STP Renderer Worker Instance %i

[Service]
Type=simple
ExecStart=/usr/bin/python3 /opt/app/backend/stp-renderer/main.py
Restart=always
RestartSec=10
```

```bash
sudo systemctl enable stp-renderer@{1..3}
sudo systemctl start stp-renderer@{1..3}
```

### 队列积压告警

当队列长度超过阈值时发送告警（可接入企业微信/钉钉）：

```bash
# 监控脚本，放在 cron 中每分钟执行
QUEUE_LEN=$(redis-cli LLEN stp_render_queue)
if [ "$QUEUE_LEN" -gt 50 ]; then
    curl -X POST "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY" \
        -H "Content-Type: application/json" \
        -d "{\"msgtype\":\"text\",\"text\":{\"content\":\"[警告] STP 渲染队列积压: ${QUEUE_LEN} 个任务\"}}"
fi
```

---

## 运维监控

### 查看队列状态

```bash
# 队列长度
redis-cli LLEN stp_render_queue

# 待处理任务数
redis-cli LLEN stp_render_queue

# 查看最近任务状态
redis-cli KEYS "stp_render_task:*" | head -10 | \
  xargs -I{} redis-cli HGETALL {}

# 清空队列（慎用，仅测试环境）
redis-cli DEL stp_render_queue
```

### 日志分析

```bash
# 实时查看 Worker 日志
tail -f /var/log/stp-renderer/worker.log

# 搜索错误
grep -i "error\|failed\|exception" /var/log/stp-renderer/worker.log

# 统计任务处理时间
grep "Processing task" worker.log | wc -l

# 平均渲染时间（需要日志格式支持）
grep "completed successfully" worker.log | \
  awk -F'time=' '{sum+=$2; count++} END {print "平均:", sum/count "s, 总数:", count}'
```

### 健康检查

Worker 进程会在启动时输出以下日志：

```
STP Renderer Worker started
Redis: localhost:6379/0
Go API: http://localhost:8080
```

如果 Worker 异常退出，systemd/supervisor 会自动重启并记录退出码。

---

## 常见问题

### Q1: OCCT 导入失败

**错误：** `ImportError: DLL load failed` 或 `ModuleNotFoundError: No module named 'OCCP'`

**原因：** OpenCASCADE 在 Windows 上依赖 Visual C++ Redistributable，或安装路径问题。

**解决：**

```bash
# Windows: 安装 Visual C++ Redistributable 2015-2022
# 或使用 conda 安装预编译包
conda install -c conda-forge opencascade-python
```

### Q2: OSS 上传失败

**错误：** `oss2.exceptions.RequestError` 或 ` oss2.exceptions.ServerError`

**解决：**

1. 确认 AccessKey 权限（需要读写 Bucket 权限）
2. 检查 OSS Endpoint 是否正确
3. 确认 Bucket 名称无误
4. 查看 OSS 控制台是否有流量限制

### Q3: 渲染超时

**原因：** 复杂 STP 文件渲染耗时超过 60s

**解决：** Worker 默认无限等待，可通过添加超时控制：

```python
# 在 main.py 的 process_task 中添加
import signal

def timeout_handler(signum, frame):
    raise TimeoutError("Rendering timeout")

signal.signal(signal.SIGALRM, timeout_handler)
signal.alarm(120)  # 2 分钟超时

try:
    six_views = render_six_views(stp_path)
finally:
    signal.alarm(0)
```

### Q4: 六面图方向不正确

**原因：** STP 文件坐标系与预期不符（有些 CAD 软件使用 Z-up，有些 Y-up）

**解决：** 在 `renderer/camera.py` 中调整 `VIEW_CONFIGS` 的坐标系定义，或在 `step_loader.py` 中添加坐标系转换。

### Q5: 内存持续增长

**原因：** 临时文件未清理或 OCCT 内存泄漏

**解决：** Worker 已使用 `tempfile.TemporaryDirectory` 自动清理。如问题持续，考虑定期重启 Worker（配合 supervisor 的 `autorestart=true`）。

---

## 配置参考

### 环境变量完整列表

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `REDIS_HOST` | 是 | localhost | Redis 主机地址 |
| `REDIS_PORT` | 是 | 6379 | Redis 端口 |
| `REDIS_DB` | 否 | 0 | Redis 数据库编号 |
| `OSS_ACCESS_KEY` / `ALIBABA_CLOUD_ACCESS_KEY_ID` | 是 | — | OSS AccessKey ID |
| `OSS_ACCESS_SECRET` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | 是 | — | OSS AccessKey Secret |
| `OSS_ENDPOINT` | 是 | oss-cn-hangzhou.aliyuncs.com | OSS Endpoint |
| `OSS_BUCKET` | 是 | advertising-image | OSS Bucket 名称 |
| `OSS_DOMAIN` | 否 | — | CDN 域名（用于生成 URL） |
| `GO_API_BASE_URL` | 是 | http://localhost:8080 | Go 后端地址 |

### 渲染配置（修改 `config.py`）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `RENDER_RESOLUTION` | 1024 | 输出图片分辨率（像素） |
| `RENDER_BG_COLOR` | (255, 255, 255) | 背景色（白色） |
| `CAMERA_DISTANCE_FACTOR` | 2.5 | 相机距离 = max(bbox) * factor |
