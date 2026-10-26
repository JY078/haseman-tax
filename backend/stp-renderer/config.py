import os

# Redis 配置
# Go backend passes REDIS_HOST=host:port (or just host) 和 REDIS_PORT=port separately.
# We extract host/port from REDIS_HOST if it contains a colon, otherwise use REDIS_PORT.
_redis_host_env = os.getenv("REDIS_HOST", "localhost")
if ":" in _redis_host_env:
    _parts = _redis_host_env.rsplit(":", 1)
    REDIS_HOST = _parts[0]
    REDIS_PORT = int(_parts[1])
else:
    REDIS_HOST = _redis_host_env
    REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
REDIS_DB = int(os.getenv("REDIS_DB", "0"))

# OSS 配置 - 优先从环境变量，fallback 到 backend/.env
_oss_key = os.getenv("OSS_ACCESS_KEY") or os.getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
_oss_secret = os.getenv("OSS_ACCESS_SECRET") or os.getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")
_oss_endpoint = os.getenv("OSS_ENDPOINT")
_oss_bucket = os.getenv("OSS_BUCKET")
_oss_domain = os.getenv("OSS_DOMAIN")

# 如果 env 里没有，尝试从 backend/.env 读取
if not _oss_key or not _oss_secret:
    _backend_env = os.path.join(os.path.dirname(os.path.dirname(__file__)), ".env")
    if os.path.exists(_backend_env):
        with open(_backend_env, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if "=" in line:
                    k, v = line.split("=", 1)
                    v = v.strip().strip('"').strip("'")
                    if k == "ALIBABA_CLOUD_ACCESS_KEY_ID" and not _oss_key:
                        _oss_key = v
                    elif k == "ALIBABA_CLOUD_ACCESS_KEY_SECRET" and not _oss_secret:
                        _oss_secret = v
                    elif k == "OSS_ENDPOINT" and not _oss_endpoint:
                        _oss_endpoint = v
                    elif k == "OSS_BUCKET" and not _oss_bucket:
                        _oss_bucket = v

OSS_ACCESS_KEY = _oss_key
OSS_ACCESS_SECRET = _oss_secret
OSS_ENDPOINT = _oss_endpoint or "oss-cn-shenzhen.aliyuncs.com"
OSS_BUCKET = _oss_bucket or "haseman-upload-prod"
OSS_DOMAIN = _oss_domain or ""

# Go API 配置
GO_API_BASE_URL = os.getenv("GO_API_BASE_URL", "http://localhost:8080")

# 渲染配置
RENDER_RESOLUTION = 1024
RENDER_BG_COLOR = (255, 255, 255)  # 白色背景
CAMERA_DISTANCE_FACTOR = 2.5  # d = max(bbox) * factor