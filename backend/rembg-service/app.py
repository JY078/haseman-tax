"""Internal CPU-only background removal service. Run exactly one worker."""

import base64
import binascii
import hashlib
import io
import logging
import os
import threading
from pathlib import Path
from contextlib import asynccontextmanager
from typing import Literal

from fastapi import FastAPI, HTTPException, Response
import onnxruntime as ort
from PIL import Image, ImageChops, ImageOps, UnidentifiedImageError
from pydantic import BaseModel, Field
from rembg import remove
from rembg.sessions.birefnet_general import BiRefNetSessionGeneral

MODEL = "birefnet-general"
MODEL_FILE = "birefnet-general-efficient.onnx"
MODEL_SHA256 = "b6cc66609e98f76e66c0d7df0eea477d80f08df4cf59e86540b8bd52fe0d7546"
Image.MAX_IMAGE_PIXELS = None
inference_lock = threading.Lock()


class EfficientBiRefNetSession(BiRefNetSessionGeneral):
    def predict(self, image, *args, **kwargs):
        mask = super().predict(image, *args, **kwargs)[0]
        bounds = refinement_bounds(mask)
        if bounds is None:
            return [mask]
        refined = super().predict(image.crop(bounds), *args, **kwargs)[0]
        # Preserve low-confidence objects outside the refined region.
        mask.paste(refined, bounds[:2])
        return [mask]

    @classmethod
    def download_models(cls, *args, **kwargs):
        # Use the verified offline graph; the upstream export needs much more RAM.
        path = Path(cls.rembg_home()) / MODEL_FILE
        if not path.is_file():
            raise FileNotFoundError(f"请先上传完整版离线模型至 {path}")
        with path.open("rb") as model:
            if hashlib.file_digest(model, "sha256").hexdigest() != MODEL_SHA256:
                raise ValueError(f"模型校验失败，请重新上传 {path}")
        return str(path)


def refinement_bounds(mask):
    bounds = mask.point(lambda alpha: 255 if alpha > 128 else 0).getbbox()
    if bounds is None:
        return None
    left, top, right, bottom = bounds
    padding = int(max(right - left, bottom - top) * 0.12)
    bounds = (max(0, left - padding), max(0, top - padding),
              min(mask.width, right + padding), min(mask.height, bottom + padding))
    width, height = bounds[2] - bounds[0], bounds[3] - bounds[1]
    # Only spend a second inference when cropping materially increases detail.
    if width * height >= mask.width * mask.height * 0.6 or max(width, height) <= 1024:
        return None
    return bounds


@asynccontextmanager
async def lifespan(app):
    options = ort.SessionOptions()
    options.enable_cpu_mem_arena = False
    options.enable_mem_pattern = False
    options.intra_op_num_threads = int(os.getenv("OMP_NUM_THREADS", "3"))
    options.inter_op_num_threads = 1
    app.state.session = EfficientBiRefNetSession(MODEL, options, providers=["CPUExecutionProvider"])
    yield


app = FastAPI(lifespan=lifespan)


class RemoveRequest(BaseModel):
    image_data: str = Field(min_length=1, max_length=35_000_000)
    background: Literal["transparent", "white", "black"] = "transparent"


def decode_image(data):
    if data.startswith("data:"):
        data = data.split(",", 1)[-1]
    try:
        raw = base64.b64decode(data, validate=True)
        with Image.open(io.BytesIO(raw)) as source:
            image = ImageOps.exif_transpose(source).convert("RGBA")
            image.load()
            return image
    except (binascii.Error, ValueError, UnidentifiedImageError, OSError):
        raise HTTPException(400, "图片数据无效，请上传 PNG、JPEG 或 WebP 图片")


def cutout(image, session, background):
    # White is only the inference canvas for existing transparent pixels.
    original_alpha = image.getchannel("A")
    rgb = Image.alpha_composite(Image.new("RGBA", image.size, "white"), image).convert("RGB")
    result = remove(
        rgb, session=session, decontaminate=True, post_process_mask=False,
    ).convert("RGBA")
    # Foreground estimation is only useful at the edge. Keep the original
    # product pixels in confident foreground areas (no smoothing/repainting).
    mask = result.getchannel("A")
    foreground = mask.point(lambda alpha: 255 if alpha >= 250 else 0)
    result = Image.composite(image.convert("RGB"), result.convert("RGB"), foreground).convert("RGBA")
    # Preserve existing transparency and soft edges instead of thresholding the mask.
    result.putalpha(ImageChops.multiply(mask, original_alpha))
    if background != "transparent":
        result = Image.alpha_composite(Image.new("RGBA", result.size, background), result)
    output = io.BytesIO()
    result.save(output, format="PNG")
    return output.getvalue()


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL, "provider": "CPUExecutionProvider"}


@app.post("/remove")
def remove_background(request: RemoveRequest):
    # Reject overlapping work rather than loading multiple large inference tensors.
    if not inference_lock.acquire(blocking=False):
        raise HTTPException(503, "抠图服务繁忙，请稍后重试")
    try:
        image = decode_image(request.image_data)
        return Response(cutout(image, app.state.session, request.background), media_type="image/png")
    except HTTPException:
        raise
    except Exception:
        logging.exception("Background removal failed")
        raise HTTPException(500, "抠图失败，请重试或更换图片")
    finally:
        inference_lock.release()
