#!/usr/bin/env python3
"""
STP Renderer Worker
从 Redis 队列消费任务，使用 OpenCASCADE + Open3D 渲染六面图
统一 Python 环境（同时安装 pythonocc-core 和 open3d）
"""
import os
import sys
import uuid

# Debug: 打印所有相关环境变量
print("[STP] ===== ENVIRONMENT DEBUG =====", flush=True)
for key in ['REDIS_HOST', 'REDIS_PORT', 'REDIS_DB', 'GO_API_BASE_URL', 'OSS_ACCESS_KEY', 'OSS_ENDPOINT', 'BLENDER_BIN']:
    print(f"[STP] {key}={os.getenv(key, 'NOT_SET')}", flush=True)
print("[STP] ===== END ENV DEBUG =====", flush=True)

import time
import tempfile
import traceback as tb
import gc
import json
import shutil
from contextlib import contextmanager
import requests
from redis import Redis
from config import REDIS_HOST, REDIS_PORT, REDIS_DB, GO_API_BASE_URL, RENDER_RESOLUTION, RENDER_BG_COLOR

# 添加当前目录到路径
sys.path.insert(0, os.path.dirname(__file__))

from renderer.step_loader import StepLoader
from renderer.camera import create_cameras_for_view
from renderer.image_saver import ImageRenderer
from oss.client import oss_client

# 初始化组件
redis_client = Redis(host=REDIS_HOST, port=REDIS_PORT, db=REDIS_DB, decode_responses=True)
step_loader = StepLoader()
image_renderer = ImageRenderer()

print(f"[STP] ========== MODULE IMPORTS DONE ==========", flush=True)
print(f"[STP] OSS client: {oss_client}", flush=True)
print(f"[STP] Image renderer: {image_renderer}", flush=True)

QUEUE_NAME = "stp_render_queue"
SINGLE_RENDER_QUEUE = "stp_single_render_queue"
MESH_PREVIEW_QUEUE = "stp_mesh_queue"
QUEUE_TASK_PREFIX = {
    QUEUE_NAME: "stp_render_task",
    SINGLE_RENDER_QUEUE: "stp_single_render_task",
    MESH_PREVIEW_QUEUE: "stp_mesh_task",
}
MAX_RETRIES = 3


def env_int(name: str, default: int) -> int:
    value = os.getenv(name, "")
    try:
        return int(value) if value.strip() else default
    except Exception:
        return default


def env_float(name: str, default: float) -> float:
    value = os.getenv(name, "")
    try:
        return float(value) if value.strip() else default
    except Exception:
        return default


RENDER_LOCK_KEY = os.getenv("STP_RENDER_LOCK_KEY") or "stp_render_global_lock"
RENDER_LOCK_TTL_SECONDS = env_int("STP_RENDER_LOCK_TTL_SECONDS", 1800)
RENDER_MEMORY_LIMIT_PERCENT = env_float("STP_RENDER_MEMORY_LIMIT_PERCENT", 78)
RENDER_MEMORY_WAIT_SECONDS = env_int("STP_RENDER_MEMORY_WAIT_SECONDS", 10)
RENDER_MEMORY_WAIT_MAX_SECONDS = env_int("STP_RENDER_MEMORY_WAIT_MAX_SECONDS", 180)
MODEL_RENDER_MAX_FILE_SIZE_MB = env_float("MODEL_RENDER_MAX_FILE_SIZE_MB", 100)
MODEL_RENDER_MAX_RESOLUTION = env_int("MODEL_RENDER_MAX_RESOLUTION", 1024)
MODEL_RENDER_TARGET_TRIANGLES = env_int("MODEL_RENDER_TARGET_TRIANGLES", 900000)
MODEL_RENDER_MAX_TRIANGLES = env_int("MODEL_RENDER_MAX_TRIANGLES", 1500000)


def current_memory_percent():
    """Return system memory usage percent without adding extra dependencies."""
    try:
        if os.name == "posix" and os.path.exists("/proc/meminfo"):
            info = {}
            with open("/proc/meminfo", "r", encoding="utf-8") as f:
                for line in f:
                    key, value = line.split(":", 1)
                    info[key] = int(value.strip().split()[0])
            total = info.get("MemTotal", 0)
            available = info.get("MemAvailable", 0)
            if total > 0 and available >= 0:
                return (total - available) / total * 100
        if os.name == "nt":
            import ctypes

            class MEMORYSTATUSEX(ctypes.Structure):
                _fields_ = [
                    ("dwLength", ctypes.c_ulong),
                    ("dwMemoryLoad", ctypes.c_ulong),
                    ("ullTotalPhys", ctypes.c_ulonglong),
                    ("ullAvailPhys", ctypes.c_ulonglong),
                    ("ullTotalPageFile", ctypes.c_ulonglong),
                    ("ullAvailPageFile", ctypes.c_ulonglong),
                    ("ullTotalVirtual", ctypes.c_ulonglong),
                    ("ullAvailVirtual", ctypes.c_ulonglong),
                    ("sullAvailExtendedVirtual", ctypes.c_ulonglong),
                ]

            stat = MEMORYSTATUSEX()
            stat.dwLength = ctypes.sizeof(MEMORYSTATUSEX)
            if ctypes.windll.kernel32.GlobalMemoryStatusEx(ctypes.byref(stat)):
                return float(stat.dwMemoryLoad)
    except Exception as e:
        print(f"[STP] memory check failed: {e}", flush=True)
    return None


def wait_for_memory_budget(reason: str):
    deadline = time.time() + RENDER_MEMORY_WAIT_MAX_SECONDS
    while True:
        percent = current_memory_percent()
        if percent is None or percent < RENDER_MEMORY_LIMIT_PERCENT:
            return True
        if time.time() >= deadline:
            print(
                f"[STP] memory still high for {reason}: {percent:.1f}% >= {RENDER_MEMORY_LIMIT_PERCENT:.1f}%, keep task queued",
                flush=True,
            )
            return False
        print(
            f"[STP] memory high before {reason}: {percent:.1f}% >= {RENDER_MEMORY_LIMIT_PERCENT:.1f}%, waiting...",
            flush=True,
        )
        time.sleep(RENDER_MEMORY_WAIT_SECONDS)


def _bbox_info(bbox) -> dict:
    min_p = bbox.CornerMin()
    max_p = bbox.CornerMax()
    dx = max_p.X() - min_p.X()
    dy = max_p.Y() - min_p.Y()
    dz = max_p.Z() - min_p.Z()
    return {
        "min": (min_p.X(), min_p.Y(), min_p.Z()),
        "max": (max_p.X(), max_p.Y(), max_p.Z()),
        "size": (dx, dy, dz),
    }


def _clamp_resolution(value: int, cap: int) -> int:
    try:
        requested = int(value)
    except Exception:
        requested = MODEL_RENDER_MAX_RESOLUTION
    requested = max(256, requested)
    return min(requested, cap, MODEL_RENDER_MAX_RESOLUTION)


def _max_model_bytes() -> int:
    return int(MODEL_RENDER_MAX_FILE_SIZE_MB * 1024 * 1024)


def _check_local_model_size(path: str) -> None:
    file_size_mb = os.path.getsize(path) / (1024 * 1024)
    if file_size_mb > MODEL_RENDER_MAX_FILE_SIZE_MB:
        raise ValueError(
            f"Model file is too large: {file_size_mb:.1f}MB > {MODEL_RENDER_MAX_FILE_SIZE_MB:.1f}MB"
        )


def select_model_profile(model_path: str, purpose: str = "render", requested_resolution: int = 1024) -> dict:
    file_size_mb = os.path.getsize(model_path) / (1024 * 1024)
    if file_size_mb > MODEL_RENDER_MAX_FILE_SIZE_MB:
        raise ValueError(
            f"Model file is too large: {file_size_mb:.1f}MB > {MODEL_RENDER_MAX_FILE_SIZE_MB:.1f}MB"
        )

    memory_percent = current_memory_percent()
    profile = {
        "name": "universal",
        "deflection": 0.04,
        "probe_deflection": 0.08,
        "fallback_deflection": 0.15,
        "target_triangles": MODEL_RENDER_TARGET_TRIANGLES,
        "max_triangles": MODEL_RENDER_MAX_TRIANGLES,
        "resolution": _clamp_resolution(requested_resolution, MODEL_RENDER_MAX_RESOLUTION),
    }

    profile["file_size_mb"] = file_size_mb
    profile["memory_percent"] = memory_percent
    profile["max_file_size_mb"] = MODEL_RENDER_MAX_FILE_SIZE_MB
    profile["purpose"] = purpose
    print(
        "[STP] Universal model profile: "
        f"name={profile['name']} purpose={purpose} file={file_size_mb:.1f}MB "
        f"memory={memory_percent if memory_percent is not None else 'unknown'} "
        f"deflection={profile['deflection']} target_triangles={profile['target_triangles']} "
        f"max_triangles={profile['max_triangles']} resolution={profile['resolution']}",
        flush=True,
    )
    return profile


def _count_step_topology(shape) -> dict:
    from OCC.Core.TopAbs import TopAbs_EDGE, TopAbs_FACE
    from OCC.Core.TopExp import TopExp_Explorer

    counts = {"faces": 0, "edges": 0}
    explorer = TopExp_Explorer(shape, TopAbs_FACE)
    while explorer.More():
        counts["faces"] += 1
        explorer.Next()

    explorer = TopExp_Explorer(shape, TopAbs_EDGE)
    while explorer.More():
        counts["edges"] += 1
        explorer.Next()

    return counts


def adjust_step_profile(profile: dict, shape) -> dict:
    counts = _count_step_topology(shape)
    adjusted = dict(profile)
    adjusted["face_count"] = counts["faces"]
    adjusted["edge_count"] = counts["edges"]
    print(
        f"[STP] STEP topology: faces={counts['faces']} edges={counts['edges']} profile={adjusted['name']}",
        flush=True,
    )
    return adjusted


def _clean_step_mesh_cache(shape) -> None:
    try:
        from OCC.Core.BRepTools import breptools
        breptools.Clean(shape)
    except Exception:
        try:
            from OCC.Core.BRepTools import BRepTools
            BRepTools.Clean(shape)
        except Exception as e:
            print(f"[STP] STEP mesh cache cleanup skipped: {e}", flush=True)


def _precompute_step_mesh(shape, deflection: float) -> dict:
    _clean_step_mesh_cache(shape)
    return image_renderer.precompute_mesh(shape, deflection)


def _precompute_step_mesh_for_current_profile(shape, profile: dict) -> tuple:
    probe_deflection = profile.get("probe_deflection", profile["fallback_deflection"])
    mesh_data = _precompute_step_mesh(shape, probe_deflection)
    triangle_count = len(mesh_data.get("triangles") or [])
    current_deflection = probe_deflection

    if triangle_count > profile["max_triangles"] and probe_deflection < profile["fallback_deflection"]:
        del mesh_data
        gc.collect()
        current_deflection = profile["fallback_deflection"]
        mesh_data = _precompute_step_mesh(shape, current_deflection)
        triangle_count = len(mesh_data.get("triangles") or [])

    return mesh_data, triangle_count, current_deflection, probe_deflection


def _simplify_step_preview_mesh(mesh_data: dict, profile: dict) -> dict:
    triangle_count = len(mesh_data.get("triangles") or [])
    max_triangles = int(profile.get("max_triangles") or 0)
    if profile.get("purpose") != "preview" or max_triangles <= 0 or triangle_count <= max_triangles:
        return mesh_data

    target_triangles = int(profile.get("target_triangles") or max_triangles)
    target_triangles = max(10000, min(target_triangles, int(max_triangles * 0.9)))
    print(
        f"[STP] Simplifying STEP preview mesh: {triangle_count} -> {target_triangles} triangles",
        flush=True,
    )

    try:
        import numpy as np
        import open3d as o3d

        mesh = o3d.geometry.TriangleMesh()
        mesh.vertices = o3d.utility.Vector3dVector(np.asarray(mesh_data["vertices"], dtype=np.float64))
        mesh.triangles = o3d.utility.Vector3iVector(np.asarray(mesh_data["triangles"], dtype=np.int32))
        mesh.remove_duplicated_vertices()
        mesh.remove_degenerate_triangles()
        mesh.remove_duplicated_triangles()
        mesh.remove_non_manifold_edges()
        simplified = mesh.simplify_quadric_decimation(target_triangles)
        simplified.remove_degenerate_triangles()
        simplified.remove_duplicated_triangles()
        simplified.remove_unreferenced_vertices()

        simplified_count = len(simplified.triangles)
        if simplified_count > max_triangles:
            raise RuntimeError(
                f"simplified mesh still too complex: {simplified_count} triangles > {max_triangles}"
            )

        simplified_data = {
            "vertices": np.asarray(simplified.vertices, dtype=np.float64).tolist(),
            "triangles": np.asarray(simplified.triangles, dtype=np.int64).tolist(),
        }
        print(
            f"[STP] STEP preview simplification complete: "
            f"{len(simplified_data['vertices'])} verts, {simplified_count} tris",
            flush=True,
        )
        del mesh
        del simplified
        return simplified_data
    except Exception as e:
        print(f"[STP] STEP preview simplification failed: {e}", flush=True)
        raise RuntimeError(
            f"STEP model is too complex: {triangle_count} triangles > {max_triangles} "
            f"and preview simplification failed"
        ) from e


def _precompute_step_mesh_with_profile(shape, profile: dict) -> dict:
    mesh_data, triangle_count, current_deflection, probe_deflection = _precompute_step_mesh_for_current_profile(shape, profile)

    if triangle_count > profile["max_triangles"]:
        mesh_data = _simplify_step_preview_mesh(mesh_data, profile)
        triangle_count = len(mesh_data.get("triangles") or [])

    if triangle_count > profile["max_triangles"]:
        raise RuntimeError(
            f"STEP model is too complex: {triangle_count} triangles > {profile['max_triangles']} "
            f"after universal profile simplification"
        )

    if profile["deflection"] >= current_deflection:
        return mesh_data

    if current_deflection > probe_deflection:
        print(
            f"[STP] Keep coarse STEP mesh for {profile['name']}: "
            f"{triangle_count} triangles at deflection={current_deflection}",
            flush=True,
        )
        return mesh_data

    refine_budget = int(profile["max_triangles"] * 0.55)
    if triangle_count > refine_budget:
        print(
            f"[STP] Keep STEP probe mesh for {profile['name']}: "
            f"{triangle_count} probe triangles > refine budget {refine_budget}",
            flush=True,
        )
        return mesh_data

    print(
        f"[STP] Refining STEP mesh for {profile['name']}: "
        f"probe={triangle_count} triangles, deflection={profile['deflection']}",
        flush=True,
    )
    del mesh_data
    gc.collect()
    refined = _precompute_step_mesh(shape, profile["deflection"])
    refined_count = len(refined.get("triangles") or [])
    if refined_count > profile["max_triangles"]:
        del refined
        gc.collect()
        raise RuntimeError(
            f"STEP model is too complex after refinement: {refined_count} triangles > {profile['max_triangles']}"
        )
    return refined


def load_model_auto(model_path: str, purpose: str = "render", requested_resolution: int = 1024) -> tuple:
    profile = select_model_profile(model_path, purpose, requested_resolution)
    if step_loader.is_step_file(model_path):
        shape, bbox = step_loader.load(model_path)
        profile = adjust_step_profile(profile, shape)
        mesh_info = _bbox_info(bbox)
        mesh_data = _precompute_step_mesh_with_profile(shape, profile)
    else:
        shape = None
        mesh_info, mesh_data = step_loader.load_mesh_file(
            model_path,
            simplify=True,
            target_triangles=profile["target_triangles"],
            max_file_size_mb=int(profile["max_file_size_mb"]),
            max_triangles=profile["max_triangles"],
        )

    return shape, mesh_info, mesh_data, profile


@contextmanager
def render_slot(task_id: str, queue_name: str):
    token = f"{os.getpid()}:{task_id}:{time.time()}"
    acquired = False
    try:
        while not acquired:
            if not wait_for_memory_budget(task_id):
                task_key = f"{QUEUE_TASK_PREFIX.get(queue_name, 'stp_render_task')}:{task_id}"
                update_task_status(task_key, "pending")
                redis_client.rpush(queue_name, task_id)
                yield False
                return
            acquired = bool(redis_client.set(RENDER_LOCK_KEY, token, nx=True, ex=RENDER_LOCK_TTL_SECONDS))
            if not acquired:
                print(f"[STP] another render is running, requeue task={task_id}", flush=True)
                task_key = f"{QUEUE_TASK_PREFIX.get(queue_name, 'stp_render_task')}:{task_id}"
                update_task_status(task_key, "pending")
                redis_client.rpush(queue_name, task_id)
                yield False
                return
        yield True
    finally:
        if acquired:
            try:
                if redis_client.get(RENDER_LOCK_KEY) == token:
                    redis_client.delete(RENDER_LOCK_KEY)
            except Exception as e:
                print(f"[STP] release render lock failed: {e}", flush=True)
        image_renderer.close()
        gc.collect()


def download_stp_file(oss_url: str) -> str:
    """下载 STP 文件到本地临时目录"""
    import shutil
    from urllib.parse import urlparse

    tmp_dir = tempfile.mkdtemp(prefix="stp_")
    parsed = urlparse(oss_url)
    ext = os.path.splitext(parsed.path)[1].lower()
    if ext not in ('.stp', '.step', '.obj', '.fbx'):
        ext = '.stp'
    local_path = os.path.join(tmp_dir, f"model{ext}")

    if oss_client is None:
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise RuntimeError("OSS client not initialized. Please check OSS configuration.")

    try:
        # 从完整 URL 中提取 OSS object key
        # 格式: https://bucket.endpoint/path/to/file.stp
        # 或: https://CDN_DOMAIN/path/to/file.stp
        parsed = urlparse(oss_url)
        # object_key 是 path 部分，去掉前导 /
        object_key = parsed.path.lstrip('/')

        if object_key:
            # 使用 OSS SDK 下载
            oss_client.download_file(object_key, local_path)
            _check_local_model_size(local_path)
        else:
            # fallback: 使用 requests 下载完整 URL
            response = requests.get(oss_url, timeout=60, stream=True)
            response.raise_for_status()
            downloaded = 0
            max_bytes = _max_model_bytes()
            with open(local_path, 'wb') as f:
                for chunk in response.iter_content(chunk_size=1024 * 1024):
                    if not chunk:
                        continue
                    downloaded += len(chunk)
                    if downloaded > max_bytes:
                        raise ValueError(
                            f"Model file is too large while downloading: "
                            f"{downloaded / (1024 * 1024):.1f}MB > {MODEL_RENDER_MAX_FILE_SIZE_MB:.1f}MB"
                        )
                    f.write(chunk)
            _check_local_model_size(local_path)
    except Exception as e:
        # 下载失败时清理临时目录
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise RuntimeError(f"Failed to download STP file: {e}")

    return local_path


def render_views(shape, cameras: list, mesh_data: dict) -> dict:
    """直接渲染多个视角（复用预计算的 mesh 数据）"""
    results = {}
    for cam_info in cameras:
        view_name = cam_info['name']
        try:
            img = image_renderer.create_view_image(shape, cam_info['camera'], view_name, mesh_data)
            if img:
                results[view_name] = img
                print(f"[STP] Rendered {view_name}: {img.size}", flush=True)
            else:
                print(f"[STP] WARNING: No image for {view_name}", flush=True)
        except Exception as e:
            print(f"[STP] ERROR rendering {view_name}: {e}", flush=True)
    return results


def render_six_views(stp_path: str) -> list:
    """渲染六面图，返回 6 张图像的 OSS URL"""
    # 加载 STP 文件
    print(f"[STP] Loading 3D model file: {stp_path}")
    shape, mesh_info, mesh_data, profile = load_model_auto(stp_path, "render", RENDER_RESOLUTION)
    image_renderer.resolution = profile["resolution"]

    # 提取 mesh 数据用于相机距离计算
    print(f"[STP] STP bounding box:")
    print(f"[STP]   Min: {mesh_info['min']}")
    print(f"[STP]   Max: {mesh_info['max']}")
    print(f"[STP]   Size (dx, dy, dz): {mesh_info['size']}")
    print(f"[STP]   Max dimension: {max(mesh_info['size'])}")

    # 预计算 mesh 数据（一次提取，供 6 个视角复用）
    # 使用较大的 deflection (0.3) 减少三角形数量，提升渲染速度
    print(
        f"[STP] Mesh precomputed: {len(mesh_data['vertices'])} vertices, "
        f"{len(mesh_data['triangles'])} triangles, profile={profile['name']}",
        flush=True,
    )

    if not mesh_data['vertices'] or not mesh_data['triangles']:
        print(f"[STP] CRITICAL: Mesh data is EMPTY! vertices={len(mesh_data['vertices'])}, triangles={len(mesh_data['triangles'])}", flush=True)
        raise RuntimeError(f"Mesh extraction failed: got 0 vertices and/or 0 triangles")

    # 创建 6 个相机
    cameras = create_cameras_for_view(mesh_info)
    print(f"[STP] Created {len(cameras)} cameras")

    # 直接渲染（复用预计算的 mesh 数据）
    print(f"[STP] Starting direct Open3D rendering")
    images = render_views(shape, cameras, mesh_data)
    print(f"[STP] Received {len(images)} rendered images")

    results = []
    for cam_info in cameras:
        view_name = cam_info['name']
        description = cam_info['description']
        img = images.get(view_name)

        if img is None:
            print(f"[STP] WARNING: Missing image for {view_name}, creating placeholder")
            from PIL import Image, ImageDraw
            img = Image.new('RGB', (image_renderer.resolution, image_renderer.resolution), RENDER_BG_COLOR)
            draw = ImageDraw.Draw(img)
            draw.text((20, 20), f"Render failed: {view_name}", fill=(200, 100, 100))
            # 标记这个视角失败了（用特殊标记）
            results.append({
                'url': None,  # None 表示上传失败占位图
                'description': description,
                'view_name': view_name,
                'placeholder': True,
            })
            continue

        # 上传到 OSS
        if oss_client is None:
            raise RuntimeError("OSS client not initialized. Cannot upload rendered image.")

        object_key = f"products/six-views/{int(time.time())}_{view_name}.png"
        uid = uuid.uuid4().hex[:8]
        url = oss_client.upload_image(img, f"products/six-views/{uid}_{view_name}.png")
        print(f"[STP] Uploaded {view_name}: {url}", flush=True)

        results.append({
            'url': url,
            'description': description,
            'view_name': view_name,
        })

    # 清理 matplotlib figure，释放内存
    image_renderer.close()

    # 清理下载的 STP 文件
    stp_parent_dir = os.path.dirname(stp_path)
    shutil.rmtree(stp_parent_dir, ignore_errors=True)
    print(f"[STP] Rendering complete, {len(results)} views", flush=True)

    # 检查是否有占位图（渲染失败的视角）
    placeholders = [r for r in results if r.get('placeholder')]
    if placeholders:
        print(f"[STP] WARNING: {len(placeholders)} views have placeholders: {[r['view_name'] for r in placeholders]}", flush=True)

    return results


def update_task_status(task_key: str, status: str, error_msg: str = ""):
    """更新 Redis 中的任务状态"""
    task_data = {
        'status': status,
        'error_message': error_msg,
        'updated_at': time.strftime('%Y-%m-%dT%H:%M:%SZ')
    }
    redis_client.hset(task_key, mapping=task_data)


def notify_go_backend(task_id: str, status: str, six_views: list = None, error_msg: str = ""):
    """通知 Go 后端渲染结果"""
    payload = {
        'task_id': task_id,
        'status': status,
        'error_message': error_msg
    }

    if six_views:
        payload['six_views'] = six_views

    headers = {}
    internal_secret = os.getenv("INTERNAL_API_SECRET", "")
    if internal_secret:
        headers['Authorization'] = f'Bearer {internal_secret}'

    callback_url = f"{GO_API_BASE_URL}/api/internal/stp-render/complete"
    print(f"[STP] === NOTIFY BACKEND ===", flush=True)
    print(f"[STP] Callback URL: {callback_url}", flush=True)
    print(f"[STP] INTERNAL_API_SECRET set: {bool(internal_secret)}", flush=True)
    print(f"[STP] Payload: task_id={task_id}, status={status}, six_views_count={len(six_views) if six_views else 0}, error_msg='{error_msg}'", flush=True)
    print(f"[STP] Headers: {headers}", flush=True)

    try:
        response = requests.post(
            callback_url,
            json=payload,
            headers=headers,
            timeout=10
        )
        print(f"[STP] Backend callback HTTP status: {response.status_code}", flush=True)
        print(f"[STP] Backend callback response body: {response.text[:500]}", flush=True)
        print(f"[STP] Backend callback success: {response.status_code == 200}", flush=True)
        print(f"[STP] === END NOTIFY ===", flush=True)
        return response.status_code == 200
    except requests.exceptions.ConnectionError as e:
        print(f"[STP] Connection ERROR: {e}", flush=True)
        print(f"[STP] Is GO_API_BASE_URL correct? {GO_API_BASE_URL}", flush=True)
        print(f"[STP] === END NOTIFY (CONNECTION FAILED) ===", flush=True)
        return False
    except requests.exceptions.Timeout as e:
        print(f"[STP] Timeout ERROR: {e}", flush=True)
        print(f"[STP] === END NOTIFY (TIMEOUT) ===", flush=True)
        return False
    except Exception as e:
        print(f"[STP] Failed to notify Go backend: {e}", flush=True)
        print(f"[STP] === END NOTIFY (ERROR) ===", flush=True)
        return False


def process_task(task_id: str):
    """处理单个渲染任务"""
    import sys
    print(f"\n[STP] ========== Processing task: {task_id} ==========", flush=True)
    sys.stdout.flush()

    task_key = f"stp_render_task:{task_id}"
    task_info = redis_client.hgetall(task_key)
    print(f"[STP] Task key: {task_key}, task_info: {task_info}", flush=True)

    if not task_info:
        print(f"[STP] Task not found: {task_id}", flush=True)
        return

    stp_file_url = task_info.get('stp_file_url')
    product_id = task_info.get('product_id')
    print(f"[STP] stp_file_url: {stp_file_url}", flush=True)
    print(f"[STP] product_id: {product_id}", flush=True)

    update_task_status(task_key, 'processing')
    sys.stdout.flush()

    six_views = []
    error_msg = ""
    stp_path = ""

    try:
        # 下载 STP 文件
        print(f"[STP] Downloading STP file from: {stp_file_url}", flush=True)
        sys.stdout.flush()
        stp_path = download_stp_file(stp_file_url)
        print(f"[STP] STP downloaded to: {stp_path}", flush=True)
        sys.stdout.flush()

        # 渲染六面图
        print(f"[STP] Rendering six views for product {product_id}", flush=True)
        sys.stdout.flush()
        six_views = render_six_views(stp_path)
        print(f"[STP] Rendering returned {len(six_views)} views", flush=True)
        sys.stdout.flush()

        if not six_views:
            error_msg = "No views rendered (empty result)"
            print(f"[STP] WARNING: {error_msg}", flush=True)
            sys.stdout.flush()

    except Exception as e:
        error_msg = str(e)
        print(f"[STP] Task {task_id} failed: {error_msg}", flush=True)
        tb.print_exc()
        sys.stdout.flush()
    finally:
        if stp_path:
            shutil.rmtree(os.path.dirname(stp_path), ignore_errors=True)

    print(f"[STP] Final decision: six_views={len(six_views)}, error_msg='{error_msg}'", flush=True)
    sys.stdout.flush()

    # 检查是否有占位图（渲染失败的视角）
    placeholders = [v for v in six_views if v.get('placeholder')]
    failed_view_names = [v['view_name'] for v in placeholders]
    partial_failure = bool(placeholders)

    if placeholders:
        error_msg = f"部分视角渲染失败: {', '.join(failed_view_names)}"
        print(f"[STP] Partial render failure: {error_msg}", flush=True)

    # 过滤掉 placeholder 记录，只保留成功的视图
    successful_views = [v for v in six_views if not v.get('placeholder')]

    # 只有有渲染结果且没有占位图才标记完成
    if successful_views and not error_msg:
        print(f"[STP] Marking task as completed, sending callback", flush=True)
        sys.stdout.flush()
        # 发送成功的视图给后端（不含 placeholder 标记）
        callback_views = [{'url': v['url'], 'description': v['description']} for v in successful_views]
        success = notify_go_backend(task_id, 'completed', callback_views)
        print(f"[STP] Callback result: success={success}", flush=True)
        sys.stdout.flush()
        if success:
            update_task_status(task_key, 'completed')
            print(f"[STP] Task {task_id} completed successfully", flush=True)
        else:
            update_task_status(task_key, 'failed', 'Failed to notify backend')
            print(f"[STP] Task {task_id} completed but backend notification failed", flush=True)
    else:
        msg = error_msg if error_msg else "No views rendered (empty result)"
        print(f"[STP] Marking task as failed: {msg}", flush=True)
        sys.stdout.flush()
        notify_go_backend(task_id, 'failed', error_msg=msg)
        update_task_status(task_key, 'failed', msg)

    print(f"[STP] ========== Task {task_id} done ==========\n", flush=True)
    sys.stdout.flush()


def render_single_view(shape, camera_info: dict, mesh_data: dict, render_mode: str = 'photo',
                       model_path: str = '') -> 'PIL.Image.Image':
    """
    渲染单个视角，根据 render_mode 选择渲染方法。

    Args:
        shape: OpenCASCADE TopoDS_Shape
        camera_info: Camera info dict
        mesh_data: Pre-extracted mesh data
        render_mode: 'photo' | 'wireframe' | 'depth' | 'silhouette'

    Returns:
        PIL Image
    """
    from PIL import Image, ImageDraw

    view_name = camera_info.get('view_name', 'single')

    try:
        if render_mode == 'wireframe':
            img = image_renderer.create_wireframe_image(
                camera_info['camera'], view_name, mesh_data, shape, model_path
            )
            print(f"[STP] Rendered wireframe for {view_name}", flush=True)
        elif render_mode == 'depth':
            img = image_renderer.create_depth_map(
                camera_info['camera'], view_name, mesh_data, model_path
            )
            print(f"[STP] Rendered depth map for {view_name}", flush=True)
        elif render_mode == 'silhouette':
            img = image_renderer.create_silhouette(
                camera_info['camera'], view_name, mesh_data
            )
            print(f"[STP] Rendered silhouette for {view_name}", flush=True)
        else:
            # 默认照片模式
            img = image_renderer.create_view_image(
                shape, camera_info['camera'], view_name, mesh_data
            )
            print(f"[STP] Rendered photo for {view_name}", flush=True)

        if img:
            return img
    except Exception as e:
        print(f"[STP] render_single_view error: {e}", flush=True)

    # 返回占位图
    resolution = camera_info.get('resolution', 1024)
    img = Image.new('RGB', (resolution, resolution), RENDER_BG_COLOR)
    draw = ImageDraw.Draw(img)
    draw.text((20, 20), f"Render failed: {render_mode}", fill=(200, 100, 100))
    return img


def process_single_render_task(task_id: str):
    """处理单张渲染任务"""
    import sys
    print(f"\n[STP] ========== Processing single render task: {task_id} ==========", flush=True)
    sys.stdout.flush()

    task_key = f"stp_single_render_task:{task_id}"
    task_info = redis_client.hgetall(task_key)
    print(f"[STP] Single task key: {task_key}, info: {task_info}", flush=True)

    if not task_info:
        print(f"[STP] Single task not found: {task_id}", flush=True)
        return

    stp_file_url = task_info.get('stp_file_url')
    azimuth = float(task_info.get('azimuth', 0))
    elevation = float(task_info.get('elevation', 30))
    distance = float(task_info.get('distance', 5))
    resolution = int(task_info.get('resolution', 1024))
    render_mode = task_info.get('render_mode', 'photo')  # 新增参数
    camera_params_raw = task_info.get('camera_params', '')
    print(f"[STP] azimuth={azimuth}, elevation={elevation}, distance={distance}, resolution={resolution}, render_mode={render_mode}", flush=True)

    update_task_status(task_key, 'processing')
    sys.stdout.flush()

    error_msg = ""
    image_url = ""
    stp_path = ""

    try:
        # 下载 STP 文件
        print(f"[STP] Downloading STP file from: {stp_file_url}", flush=True)
        stp_path = download_stp_file(stp_file_url)
        print(f"[STP] STP downloaded to: {stp_path}", flush=True)

        # 加载模型
        shape, mesh_info, mesh_data, profile = load_model_auto(stp_path, "render", resolution)
        resolution = profile["resolution"]
        image_renderer.resolution = resolution
        print(f"[STP] Model loaded, bbox: {mesh_info['size']}", flush=True)

        # 预计算 mesh
        if not mesh_data['vertices'] or not mesh_data['triangles']:
            raise RuntimeError("Mesh extraction failed")
        vertex_count = len(mesh_data['vertices'])
        triangle_count = len(mesh_data['triangles'])

        # 创建单个相机
        from renderer.camera import create_camera_from_angle
        camera_info = create_camera_from_angle(mesh_info, azimuth, elevation, distance)
        if camera_params_raw:
            try:
                camera_info['camera'].frontend_params = json.loads(camera_params_raw)
                print(f"[STP] frontend camera params applied", flush=True)
            except Exception as e:
                print(f"[STP] invalid frontend camera params: {e}", flush=True)
        camera_info['resolution'] = resolution
        camera_info['view_name'] = 'single'

        # 渲染（根据 render_mode 选择渲染方法）
        img = render_single_view(shape, camera_info, mesh_data, render_mode, stp_path)

        # 上传到 OSS（根据渲染模式选择目录）
        uid = uuid.uuid4().hex[:8]
        sub_dir = "stp-white-bg" if render_mode == 'photo' else f"stp-{render_mode}"
        image_url = oss_client.upload_image(img, f"products/{sub_dir}/{uid}_single.png")
        print(f"[STP] Single render uploaded: {image_url}", flush=True)

        # 清理
        image_renderer.close()
        stp_parent_dir = os.path.dirname(stp_path)
        shutil.rmtree(stp_parent_dir, ignore_errors=True)

    except Exception as e:
        error_msg = str(e)
        print(f"[STP] Single task {task_id} failed: {error_msg}", flush=True)
        tb.print_exc()
        sys.stdout.flush()
    finally:
        if stp_path:
            shutil.rmtree(os.path.dirname(stp_path), ignore_errors=True)

    # 通知后端
    if image_url:
        print(f"[STP] Marking single task as completed", flush=True)
        success = notify_single_render_complete(task_id, 'completed', image_url)
        if success:
            update_task_status(task_key, 'completed')
        else:
            update_task_status(task_key, 'failed', 'Failed to notify backend')
    else:
        msg = error_msg if error_msg else "Render failed"
        notify_single_render_complete(task_id, 'failed', error_msg=msg)
        update_task_status(task_key, 'failed', msg)

    print(f"[STP] ========== Single task {task_id} done ==========\n", flush=True)
    sys.stdout.flush()


def process_mesh_preview_task(task_id: str):
    """处理 mesh 预览数据提取任务"""
    import sys
    print(f"\n[STP] ========== Processing mesh preview task: {task_id} ==========", flush=True)
    sys.stdout.flush()

    task_key = f"stp_mesh_task:{task_id}"
    task_info = redis_client.hgetall(task_key)
    print(f"[STP] Mesh preview task key: {task_key}, info: {task_info}", flush=True)

    if not task_info:
        print(f"[STP] Mesh preview task not found: {task_id}", flush=True)
        return

    stp_file_url = task_info.get('stp_file_url')
    print(f"[STP] Mesh preview stp_url: {stp_file_url}", flush=True)

    update_task_status(task_key, 'processing')
    sys.stdout.flush()

    error_msg = ""
    mesh_data_json = ""
    stp_path = ""

    try:
        # 下载 STP 文件
        print(f"[STP] Downloading STP file for mesh preview: {stp_file_url}", flush=True)
        stp_path = download_stp_file(stp_file_url)
        print(f"[STP] STP downloaded to: {stp_path}", flush=True)

        # 加载模型
        shape, mesh_info, mesh_data, profile = load_model_auto(stp_path, "preview", MODEL_RENDER_MAX_RESOLUTION)
        print(f"[STP] Model loaded, bbox: {mesh_info['size']}", flush=True)

        # 提取 mesh 数据（使用较小的 deflection 提高精度）
        if not mesh_data['vertices'] or not mesh_data['triangles']:
            raise RuntimeError("Mesh extraction failed")
        vertex_count = len(mesh_data['vertices'])
        triangle_count = len(mesh_data['triangles'])

        # 清理
        image_renderer.close()
        stp_parent_dir = os.path.dirname(stp_path)
        shutil.rmtree(stp_parent_dir, ignore_errors=True)

        # 构建返回数据 - 使用扁平数组格式减少 JSON 大小
        # vertices: [x1, y1, z1, x2, y2, z2, ...] 而不是 [[x,y,z], [x,y,z], ...]
        # triangles: [i1, j1, k1, i2, j2, k2, ...] 而不是 [[i,j,k], [i,j,k], ...]
        vertices_flat = []
        for v in mesh_data['vertices']:
            vertices_flat.extend([float(v[0]), float(v[1]), float(v[2])])

        triangles_flat = []
        for t in mesh_data['triangles']:
            triangles_flat.extend([int(t[0]), int(t[1]), int(t[2])])

        result = {
            'vertices': vertices_flat,
            'triangles': triangles_flat,
            'bounding_box': {
                'center': [
                    (mesh_info['min'][0] + mesh_info['max'][0]) / 2,
                    (mesh_info['min'][1] + mesh_info['max'][1]) / 2,
                    (mesh_info['min'][2] + mesh_info['max'][2]) / 2
                ],
                'size': [
                    mesh_info['size'][0],
                    mesh_info['size'][1],
                    mesh_info['size'][2]
                ]
            }
        }

        # 释放原始 mesh_data，减小内存峰值
        del mesh_data
        del vertices_flat
        del triangles_flat
        gc.collect()

        # 尝试使用 orjson（更快更省内存），否则用标准 json
        try:
            import orjson
            mesh_data_json = orjson.dumps(result).decode('utf-8')
        except ImportError:
            import json
            mesh_data_json = json.dumps(result)
        del result
        gc.collect()
        print(f"[STP] Mesh preview extracted: {vertex_count} verts, {triangle_count} tris", flush=True)

    except Exception as e:
        error_msg = str(e)
        print(f"[STP] Mesh preview task {task_id} failed: {error_msg}", flush=True)
        tb.print_exc()
        sys.stdout.flush()
    finally:
        if stp_path:
            shutil.rmtree(os.path.dirname(stp_path), ignore_errors=True)

    # 更新任务状态
    if mesh_data_json:
        update_task_status(task_key, 'completed')
        redis_client.hset(task_key, 'mesh_data', mesh_data_json)
        print(f"[STP] Mesh preview task {task_id} completed", flush=True)
    else:
        update_task_status(task_key, 'failed', error_msg)
        print(f"[STP] Mesh preview task {task_id} failed", flush=True)

    print(f"[STP] ========== Mesh preview task {task_id} done ==========\n", flush=True)
    sys.stdout.flush()


def notify_single_render_complete(task_id: str, status: str, image_url: str = "", error_msg: str = ""):
    """通知 Go 后端单张渲染结果"""
    payload = {
        'task_id': task_id,
        'status': status,
        'image_url': image_url,
        'error_message': error_msg
    }

    headers = {}
    internal_secret = os.getenv("INTERNAL_API_SECRET", "")
    if internal_secret:
        headers['Authorization'] = f'Bearer {internal_secret}'

    callback_url = f"{GO_API_BASE_URL}/api/internal/stp-render/single/complete"
    print(f"[STP] Single render callback URL: {callback_url}", flush=True)

    try:
        response = requests.post(
            callback_url,
            json=payload,
            headers=headers,
            timeout=10
        )
        print(f"[STP] Single render callback HTTP status: {response.status_code}", flush=True)
        return response.status_code == 200
    except Exception as e:
        print(f"[STP] Single render callback failed: {e}", flush=True)
        return False


def worker_loop():
    """Worker 主循环，同时处理六面图队列和单张渲染队列"""
    import sys
    print("STP Renderer Worker started", flush=True)
    print(f"Redis: {REDIS_HOST}:{REDIS_PORT}/{REDIS_DB}", flush=True)
    print(f"Go API: {GO_API_BASE_URL}", flush=True)
    print(f"OSS Client initialized: {oss_client is not None}", flush=True)
    print(f"Python: {sys.executable}", flush=True)
    sys.stdout.flush()

    # 测试 Redis 连接
    try:
        redis_client.ping()
        print(f"[STP] Redis ping OK", flush=True)
    except Exception as e:
        print(f"[STP] Redis ping FAILED: {e}", flush=True)
        print(f"[STP] Will retry connection in loop...", flush=True)
    sys.stdout.flush()

    while True:
        try:
            # 使用 BLPOP 监听三个队列
            result = redis_client.blpop([QUEUE_NAME, SINGLE_RENDER_QUEUE, MESH_PREVIEW_QUEUE], timeout=30)
            print(f"[STP] BLPOP returned: {result}", flush=True)
            if result:
                queue_name, task_id = result
                print(f"[STP] Got task from queue '{queue_name}': {task_id}", flush=True)
                sys.stdout.flush()
                # 根据队列名决定处理函数
                with render_slot(task_id, queue_name) as can_render:
                    if not can_render:
                        time.sleep(RENDER_MEMORY_WAIT_SECONDS)
                        continue
                    if queue_name == SINGLE_RENDER_QUEUE:
                        process_single_render_task(task_id)
                    elif queue_name == MESH_PREVIEW_QUEUE:
                        process_mesh_preview_task(task_id)
                    else:
                        process_task(task_id)
        except KeyboardInterrupt:
            print("Worker stopped")
            break
        except Exception as e:
            print(f"Worker error: {e}")
            tb.print_exc()
            time.sleep(5)


if __name__ == "__main__":
    worker_loop()
