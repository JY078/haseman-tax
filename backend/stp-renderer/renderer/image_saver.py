"""
OCC-based 3D renderer for STP/STEP files.
Uses OpenCASCADE AIS viewer for headless rendering (no GLFW/EGL required).
Pre-extracts mesh once, reuses for all 6 views.
"""
import json
import os
import subprocess
import tempfile

import numpy as np
from PIL import Image, ImageDraw
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D
from mpl_toolkits.mplot3d.art3d import Poly3DCollection
from mpl_toolkits.mplot3d.art3d import Line3DCollection

from config import RENDER_RESOLUTION, RENDER_BG_COLOR

# matplotlib elev/azim 坐标系说明：
# - elev: 仰角，从 XY 平面向上旋转的角度 (0-180)
# - azim: 方位角，在 XY 平面内从 X 轴逆时针旋转的角度 (-180 到 180)
#
# 标准工程视图角度定义：
# - front (正视): elev=0, azim=0   → 从 -Y 方向看，Z 向上
# - back (后视): elev=0, azim=180  → 从 +Y 方向看，Z 向上
# - left (左视): elev=0, azim=90   → 从 -X 方向看，Z 向上
# - right (右视): elev=0, azim=-90 → 从 +X 方向看，Z 向上
# - top (俯视): elev=90, azim=0    → 从 +Z 方向看，Y 向上
# - bottom (仰视): elev=-90, azim=0 → 从 -Z 方向看，Y 向上
VIEW_ANGLES = {
    'front':  (0, 0),
    'back':   (0, 180),
    'left':   (0, 90),
    'right':  (0, -90),
    'top':    (90, 0),
    'bottom': (-90, 0),
}

def get_matplotlib_view(camera, view_name: str) -> tuple:
    if camera is not None and hasattr(camera, 'pos'):
        pos = np.array(camera.pos, dtype=np.float64)
        norm = np.linalg.norm(pos)
        if norm > 1e-6:
            elev = np.degrees(np.arcsin(pos[2] / norm))
            azim = np.degrees(np.arctan2(pos[1], pos[0]))
            return elev, azim

    return VIEW_ANGLES.get(view_name, (0, 0))


def _normalize_mesh_vertices(vertices, target_size: float = 2.0) -> np.ndarray:
    verts_np = np.array(vertices, dtype=np.float64)
    v_min = verts_np.min(axis=0)
    v_max = verts_np.max(axis=0)
    v_center = (v_min + v_max) / 2.0
    v_size = max(v_max - v_min)
    if v_size > 0:
        verts_np = (verts_np - v_center) * (target_size / v_size)
    return verts_np


def _shift_control_image(img: Image.Image, bg_value: int, shift_x_ratio: float = -0.04) -> Image.Image:
    w, h = img.size
    shifted = Image.new("RGB", (w, h), (bg_value, bg_value, bg_value))
    dx = int(round(w * shift_x_ratio))
    shifted.paste(img, (dx, 0))
    return shifted


def _finalize_control_size(img: Image.Image, size: int, resample=Image.LANCZOS) -> Image.Image:
    if img.size == (size, size):
        return img
    return img.resize((size, size), resample)


def _letterbox_control_image(img: Image.Image, size: int, bg_value: int) -> Image.Image:
    if img.size == (size, size):
        return img

    w, h = img.size
    scale = min(size / max(w, 1), size / max(h, 1))
    new_w = max(1, int(round(w * scale)))
    new_h = max(1, int(round(h * scale)))
    resized = img.resize((new_w, new_h), Image.LANCZOS)
    boxed = Image.new("RGB", (size, size), (bg_value, bg_value, bg_value))
    boxed.paste(resized, ((size - new_w) // 2, (size - new_h) // 2))
    return boxed


def _fit_control_subject_image(
    img: Image.Image,
    bg_value: int,
    target_fill: float = 0.74,
    padding_ratio: float = 0.08,
) -> Image.Image:
    rgb = np.array(img.convert("RGB"), dtype=np.int16)
    gray = np.array(img.convert("L"), dtype=np.uint8)

    # Keep this conservative: the bbox must include faint antialiasing, dark
    # depth details, and tiny product features. Extra whitespace is better than
    # cropping real geometry.
    if bg_value >= 128:
        bg = np.array([255, 255, 255], dtype=np.int16)
        diff = np.max(np.abs(rgb - bg), axis=2)
        mask = diff > 3
    else:
        mask = gray > 2

    ys, xs = np.where(mask)
    if len(xs) < 64:
        return img

    size = img.size[0]
    min_x, max_x = int(xs.min()), int(xs.max())
    min_y, max_y = int(ys.min()), int(ys.max())
    subject_w = max_x - min_x + 1
    subject_h = max_y - min_y + 1
    if subject_w < 16 or subject_h < 16:
        return img

    pad = max(int(round(max(subject_w, subject_h) * padding_ratio)), int(round(size * 0.04)), 16)
    min_x = max(0, min_x - pad)
    max_x = min(img.size[0] - 1, max_x + pad)
    min_y = max(0, min_y - pad)
    max_y = min(img.size[1] - 1, max_y + pad)

    crop_w = max_x - min_x + 1
    crop_h = max_y - min_y + 1
    crop_size = min(size, max(crop_w, crop_h))
    cx = (min_x + max_x) / 2.0
    cy = (min_y + max_y) / 2.0

    left = int(round(cx - crop_size / 2.0))
    top = int(round(cy - crop_size / 2.0))
    left = max(0, min(left, img.size[0] - crop_size))
    top = max(0, min(top, img.size[1] - crop_size))
    return img.crop((left, top, left + crop_size, top + crop_size))


def _strengthen_lineart_image(img: Image.Image) -> Image.Image:
    gray = np.array(img.convert("L"), dtype=np.uint8)
    line = np.where(gray < 210, 0, 255).astype(np.uint8)
    line_img = Image.fromarray(line, "L")
    return Image.merge("RGB", [line_img, line_img, line_img])


def _lineart_components(mask: np.ndarray, max_components: int = 2000) -> list:
    visited = np.zeros_like(mask, dtype=bool)
    components = []
    height, width = mask.shape
    for start_y, start_x in zip(*np.where(mask)):
        if visited[start_y, start_x]:
            continue
        stack = [(int(start_y), int(start_x))]
        visited[start_y, start_x] = True
        points = []
        while stack:
            y, x = stack.pop()
            points.append((y, x))
            for ny in (y - 1, y, y + 1):
                for nx in (x - 1, x, x + 1):
                    if ny == y and nx == x:
                        continue
                    if 0 <= ny < height and 0 <= nx < width and mask[ny, nx] and not visited[ny, nx]:
                        visited[ny, nx] = True
                        stack.append((ny, nx))
        if points:
            ys = np.array([p[0] for p in points], dtype=np.int32)
            xs = np.array([p[1] for p in points], dtype=np.int32)
            components.append({
                "points": points,
                "min_x": int(xs.min()),
                "max_x": int(xs.max()),
                "min_y": int(ys.min()),
                "max_y": int(ys.max()),
                "pixels": len(points),
            })
        if len(components) >= max_components:
            break
    return components


def _log_lineart_control_quality(mask: np.ndarray, components: list, bbox: tuple) -> None:
    min_x, max_x, min_y, max_y = bbox
    width = max_x - min_x + 1
    height = max_y - min_y + 1
    min_visible = int(np.clip(min(width, height) * 0.012, 5, 14))
    weak = 0
    detached = 0
    for comp in components:
        comp_w = comp["max_x"] - comp["min_x"] + 1
        comp_h = comp["max_y"] - comp["min_y"] + 1
        if comp["pixels"] < min_visible or max(comp_w, comp_h) < min_visible:
            weak += 1
        if comp["pixels"] < 80 and comp_w < min_visible * 3 and comp_h < min_visible * 3:
            detached += 1
    print(
        f"[Renderer] Lineart control quality: bbox={width}x{height}, "
        f"min_visible_px={min_visible}, components={len(components)}, "
        f"weak_small_features={weak}, tiny_detached_features={detached}",
        flush=True,
    )


def _reinforce_small_lineart_features(img: Image.Image) -> Image.Image:
    gray = np.array(img.convert("L"), dtype=np.uint8)
    mask = gray < 128
    ys, xs = np.where(mask)
    if len(xs) < 64:
        return img

    min_x, max_x = int(xs.min()), int(xs.max())
    min_y, max_y = int(ys.min()), int(ys.max())
    width = max_x - min_x + 1
    height = max_y - min_y + 1
    if width < 80 or height < 80:
        return img

    components = _lineart_components(mask)
    _log_lineart_control_quality(mask, components, (min_x, max_x, min_y, max_y))

    out = img.copy()
    draw = ImageDraw.Draw(out)
    max_feature = int(np.clip(min(width, height) * 0.065, 18, 48))
    for comp in components:
        comp_w = comp["max_x"] - comp["min_x"] + 1
        comp_h = comp["max_y"] - comp["min_y"] + 1
        if comp["pixels"] < 6:
            continue
        if comp_w > max_feature or comp_h > max_feature:
            continue
        if comp["min_x"] < min_x or comp["max_x"] > max_x or comp["min_y"] < min_y or comp["max_y"] > max_y:
            continue
        for y, x in comp["points"]:
            draw.point((x, y), fill=(0, 0, 0))
            if comp["pixels"] < 260:
                draw.point((x + 1, y), fill=(0, 0, 0))
                draw.point((x, y + 1), fill=(0, 0, 0))

    padded = np.pad(mask, 1, mode="constant", constant_values=False)
    neighbor_count = np.zeros(mask.shape, dtype=np.uint8)
    for dy in (-1, 0, 1):
        for dx in (-1, 0, 1):
            if dy == 0 and dx == 0:
                continue
            neighbor_count += padded[1 + dy:1 + dy + mask.shape[0], 1 + dx:1 + dx + mask.shape[1]]

    endpoints = mask & (neighbor_count <= 2)
    endpoints[:min_y, :] = False
    endpoints[max_y + 1:, :] = False
    endpoints[:, :min_x] = False
    endpoints[:, max_x + 1:] = False
    endpoint_points = np.column_stack(np.where(endpoints))
    if len(endpoint_points) > 900:
        endpoint_points = endpoint_points[::max(1, len(endpoint_points) // 900)]

    max_gap = int(np.clip(min(width, height) * 0.025, 4, 9))
    for index, (y1, x1) in enumerate(endpoint_points):
        best = None
        best_dist = None
        for y2, x2 in endpoint_points[index + 1:]:
            dx = int(x2 - x1)
            dy = int(y2 - y1)
            dist = float(np.hypot(dx, dy))
            if dist < 3 or dist > max_gap:
                continue

            mx = int(round((x1 + x2) / 2))
            my = int(round((y1 + y2) / 2))
            y0 = max(my - 2, 0)
            y3 = min(my + 3, mask.shape[0])
            x0 = max(mx - 2, 0)
            x3 = min(mx + 3, mask.shape[1])
            if mask[y0:y3, x0:x3].sum() > 1:
                continue

            if best is None or dist < best_dist:
                best = (int(x2), int(y2))
                best_dist = dist
        if best is not None:
            draw.line((int(x1), int(y1), best[0], best[1]), fill=(0, 0, 0), width=1)
    return out


class ImageRenderer:
    def __init__(self, resolution: int = RENDER_RESOLUTION, deflection: float = 0.3):
        self.resolution = resolution
        self.bg_color = RENDER_BG_COLOR
        self.deflection = deflection
        self._fig = None
        self._ax = None

    def precompute_mesh(self, shape, deflection: float = None) -> dict:
        """
        一次性从 shape 提取 mesh 数据，供 6 个视角复用。
        使用较大的 deflection 减少三角形数量，提升渲染速度。
        """
        from OCC.Core.BRepMesh import BRepMesh_IncrementalMesh
        from OCC.Core.TopAbs import TopAbs_FACE
        from OCC.Core.TopExp import TopExp_Explorer
        from OCC.Core.BRep import BRep_Tool
        from OCC.Core.TopLoc import TopLoc_Location
        from OCC.Core.gp import gp_Pnt

        mesh_deflection = self.deflection if deflection is None else deflection
        mesh_gen = BRepMesh_IncrementalMesh(shape, mesh_deflection)
        mesh_gen.Perform()

        vertices = []
        triangles = []
        vertex_index = 0

        explorer = TopExp_Explorer(shape, TopAbs_FACE)
        while explorer.More():
            face = explorer.Current()
            location = face.Location()
            triangulation = BRep_Tool.Triangulation(face, location)
            if triangulation is None:
                explorer.Next()
                continue

            np3d = triangulation.NbNodes()
            for i in range(1, np3d + 1):
                p = triangulation.Node(i)
                p_xyz = p.XYZ()
                if not location.IsIdentity():
                    p_pnt = gp_Pnt(p_xyz.X(), p_xyz.Y(), p_xyz.Z())
                    p_pnt_t = p_pnt.Transformed(location.Transformation())
                    vertices.append([p_pnt_t.X(), p_pnt_t.Y(), p_pnt_t.Z()])
                else:
                    vertices.append([p_xyz.X(), p_xyz.Y(), p_xyz.Z()])

            ntri = triangulation.NbTriangles()
            for i in range(1, ntri + 1):
                t = triangulation.Triangle(i)
                idx1, idx2, idx3 = t.Get()
                triangles.append([
                    vertex_index + idx1 - 1,
                    vertex_index + idx2 - 1,
                    vertex_index + idx3 - 1
                ])
            vertex_index += np3d
            explorer.Next()

        print(f"[Renderer] Mesh: {len(vertices)} verts, {len(triangles)} tris", flush=True)
        return {'vertices': vertices, 'triangles': triangles}

    def create_view_image(self, shape, camera, view_name: str,
                          mesh_data: dict = None) -> Image.Image:
        """
        渲染单个视角。

        Args:
            shape: OpenCASCADE TopoDS_Shape (used only if mesh_data is None)
            camera: dict with camera object
            view_name: Name of the view (front, back, left, right, top, bottom)
            mesh_data: Pre-extracted mesh dict with 'vertices' and 'triangles',
                       or None to extract on-the-fly
        """
        try:
            return self._render_with_matplotlib(camera, view_name, mesh_data)
        except Exception as e:
            print(f"[Renderer] Render failed for {view_name}: {e}", flush=True)
            return self._create_placeholder(view_name, str(e)[:40])

    def _render_with_matplotlib(self, camera, view_name: str,
                                 mesh_data: dict) -> Image.Image:
        """
        使用 matplotlib Poly3DCollection 直接绘制预提取的 mesh，
        比每次重新生成 mesh 快 6 倍。

        坐标系说明:
        - matplotlib 使用 Z-up 坐标系
        - 模型在渲染前已经归一化到 [-0.5, 0.5] 范围，中心在原点
        - VIEW_ANGLES 定义了标准工程视图的 elev/azim 角度
        """
        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']

        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = np.array(vertices, dtype=np.float64)

        # 打印诊断信息（每个视角打印一次）
        if not hasattr(self, '_diagnostic_printed'):
            v_min = verts_np.min(axis=0)
            v_max = verts_np.max(axis=0)
            v_center = (v_min + v_max) / 2.0
            v_size = max(v_max - v_min)
            print(f"[Renderer] Original bounds: min={v_min}, max={v_max}", flush=True)
            print(f"[Renderer] Original center: {v_center}, size: {v_size}", flush=True)
            self._diagnostic_printed = True

        # 归一化到 [-0.5, 0.5]
        v_min = verts_np.min(axis=0)
        v_max = verts_np.max(axis=0)
        v_center = (v_min + v_max) / 2.0
        v_size = max(v_max - v_min)

        if v_size > 0:
            verts_np = (verts_np - v_center) / v_size

        # 打印归一化后的边界
        norm_min = verts_np.min(axis=0)
        norm_max = verts_np.max(axis=0)
        print(f"[Renderer] Normalized bounds: min={norm_min}, max={norm_max}", flush=True)

        elev, azim = get_matplotlib_view(camera, view_name)

        # 复用同一个 figure，只更新视角（避免重复创建 figure 的开销）
        if self._fig is None:
            self._fig = plt.figure(figsize=(res / 100, res / 100), dpi=100)
            self._ax = self._fig.add_subplot(111, projection='3d')
            self._ax.set_box_aspect([1, 1, 1])
            self._ax.axis('off')
            self._ax.set_xlim(-0.6, 0.6)
            self._ax.set_ylim(-0.6, 0.6)
            self._ax.set_zlim(-0.6, 0.6)
            self._ax.dist = 8

        # 更新视角
        self._ax.view_init(elev=elev, azim=azim)

        # 清除旧 mesh，重新绘制
        self._ax.cla()
        self._ax.set_box_aspect([1, 1, 1])
        self._ax.axis('off')
        self._ax.set_xlim(-0.6, 0.6)
        self._ax.set_ylim(-0.6, 0.6)
        self._ax.set_zlim(-0.6, 0.6)
        self._ax.dist = 8

        mesh_triangles = []
        for tri in triangles:
            try:
                mesh_triangles.append([verts_np[tri[0]], verts_np[tri[1]], verts_np[tri[2]]])
            except IndexError:
                continue

        if mesh_triangles:
            collection = Poly3DCollection(
                mesh_triangles,
                alpha=1.0,
                edgecolor='none',
                facecolor=[0.85, 0.85, 0.85],
                linewidth=0,
            )
            self._ax.add_collection3d(collection)

        self._fig.tight_layout(pad=0)
        self._fig.canvas.draw()

        # ARGB -> RGB
        buf = self._fig.canvas.tostring_argb()
        w, h = self._fig.canvas.get_width_height()
        arr = np.frombuffer(buf, dtype=np.uint8).reshape((h, w, 4))
        arr = arr[:, :, 1:]
        img = Image.fromarray(arr, 'RGB')

        # 缩放到目标分辨率
        img = img.resize((res, res), Image.LANCZOS)

        return img

    def _create_placeholder(self, view_name: str, reason: str = "") -> Image.Image:
        """创建占位图"""
        img = Image.new('RGB', (self.resolution, self.resolution), self.bg_color)
        draw = ImageDraw.Draw(img)

        center = self.resolution // 2
        radius = int(self.resolution * 0.15)
        draw.ellipse(
            [center - radius, center - radius, center + radius, center + radius],
            outline=(200, 100, 100),
            width=3
        )
        draw.text((center - 30, center + radius + 20), "Render Failed", fill=(200, 100, 100))
        draw.text((20, 20), f"View: {view_name}", fill=(100, 100, 100))
        if reason:
            text = f"Reason: {reason}"
            lines = [text[i:i + 90] for i in range(0, len(text), 90)]
            draw.multiline_text((20, 60), "\n".join(lines[:6]), fill=(150, 50, 50), spacing=6)

        return img

    def close(self):
        """关闭 matplotlib figure，释放内存"""
        if self._fig is not None:
            plt.close(self._fig)
            self._fig = None
            self._ax = None

    def save_image(self, image: Image.Image, output_path: str):
        """保存图像到文件"""
        image.save(output_path, format='PNG')

    def _get_camera_vectors(self, camera):
        if hasattr(camera, 'frontend_params') and camera.frontend_params:
            params = camera.frontend_params
            pos = np.array(params.get('position', [0, 0, 3]), dtype=np.float64)
            target = np.array(params.get('target', [0, 0, 0]), dtype=np.float64)
            up = np.array(params.get('up', [0, 1, 0]), dtype=np.float64)
            fov = float(params.get('fov', 45))
            aspect = float(params.get('aspect', 1))
        else:
            pos = np.array(getattr(camera, 'pos', [0, 0, 3]), dtype=np.float64)
            target = np.array(getattr(camera, 'target', [0, 0, 0]), dtype=np.float64)
            up = np.array(getattr(camera, 'up', [0, 0, 1]), dtype=np.float64)
            fov = 45.0
            aspect = 1.0

        forward = target - pos
        if np.linalg.norm(forward) < 1e-6:
            forward = np.array([0, 0, -1], dtype=np.float64)
        forward = forward / np.linalg.norm(forward)

        right = np.cross(forward, up)
        if np.linalg.norm(right) < 1e-6:
            right = np.array([1, 0, 0], dtype=np.float64)
        right = right / np.linalg.norm(right)

        true_up = np.cross(right, forward)
        true_up = true_up / np.linalg.norm(true_up)

        return pos, forward, right, true_up, fov, max(aspect, 1e-6)

    def _project_mesh(self, verts_np: np.ndarray, camera, res: int):
        pos, forward, right, up, fov, aspect = self._get_camera_vectors(camera)
        rel = verts_np - pos
        x_cam = rel @ right
        y_cam = rel @ up
        z_cam = rel @ forward

        tan_half_fov = np.tan(np.deg2rad(fov) / 2.0)
        if tan_half_fov <= 0:
            tan_half_fov = np.tan(np.deg2rad(45) / 2.0)

        ndc_x = x_cam / (z_cam * tan_half_fov * aspect)
        ndc_y = y_cam / (z_cam * tan_half_fov)
        pts = np.stack([
            (ndc_x + 1.0) * 0.5 * (res - 1),
            (1.0 - (ndc_y + 1.0) * 0.5) * (res - 1),
        ], axis=1)

        return pts, z_cam

    def _edge_segments_for_view(self, verts_np: np.ndarray, tri_np: np.ndarray, camera):
        _, forward, _, _, _, _ = self._get_camera_vectors(camera)
        v0 = verts_np[tri_np[:, 0]]
        v1 = verts_np[tri_np[:, 1]]
        v2 = verts_np[tri_np[:, 2]]
        normals = np.cross(v1 - v0, v2 - v0)
        lengths = np.linalg.norm(normals, axis=1)
        valid_normals = lengths > 1e-10
        normals[valid_normals] = normals[valid_normals] / lengths[valid_normals, None]
        facing = normals @ forward

        edge_faces = {}
        for face_idx, tri in enumerate(tri_np):
            for a, b in ((tri[0], tri[1]), (tri[1], tri[2]), (tri[2], tri[0])):
                key = tuple(sorted((int(a), int(b))))
                edge_faces.setdefault(key, []).append(face_idx)

        selected = []
        open_edges = []
        hard_edge_cos = np.cos(np.deg2rad(35))
        for edge, faces in edge_faces.items():
            if len(faces) == 1:
                open_edges.append(edge)
                continue

            if len(faces) >= 2:
                f1, f2 = faces[0], faces[1]
                one_front = facing[f1] >= 0
                other_front = facing[f2] >= 0
                if one_front != other_front:
                    selected.append(edge)
                    continue

                if one_front and other_front:
                    dot = float(np.dot(normals[f1], normals[f2]))
                    if dot < hard_edge_cos:
                        selected.append(edge)

        selected_np = np.array(selected, dtype=np.int64) if selected else np.empty((0, 2), dtype=np.int64)
        open_np = np.array(open_edges, dtype=np.int64) if open_edges else np.empty((0, 2), dtype=np.int64)
        return selected_np, open_np

    def _rasterize_depth(self, projected: np.ndarray, z_cam: np.ndarray, tri_np: np.ndarray, res: int) -> np.ndarray:
        depth = np.full((res, res), np.inf, dtype=np.float64)

        for tri in tri_np:
            if np.any(z_cam[tri] <= 1e-6):
                continue
            pts = projected[tri]
            if not np.isfinite(pts).all():
                continue

            min_x = max(int(np.floor(np.min(pts[:, 0]))), 0)
            max_x = min(int(np.ceil(np.max(pts[:, 0]))), res - 1)
            min_y = max(int(np.floor(np.min(pts[:, 1]))), 0)
            max_y = min(int(np.ceil(np.max(pts[:, 1]))), res - 1)
            if min_x > max_x or min_y > max_y:
                continue

            p0, p1, p2 = pts
            denom = ((p1[1] - p2[1]) * (p0[0] - p2[0]) +
                     (p2[0] - p1[0]) * (p0[1] - p2[1]))
            if abs(denom) < 1e-8:
                continue

            xs = np.arange(min_x, max_x + 1)
            ys = np.arange(min_y, max_y + 1)
            xx, yy = np.meshgrid(xs, ys)
            w0 = ((p1[1] - p2[1]) * (xx - p2[0]) +
                  (p2[0] - p1[0]) * (yy - p2[1])) / denom
            w1 = ((p2[1] - p0[1]) * (xx - p2[0]) +
                  (p0[0] - p2[0]) * (yy - p2[1])) / denom
            w2 = 1.0 - w0 - w1
            mask = (w0 >= 0) & (w1 >= 0) & (w2 >= 0)
            if not np.any(mask):
                continue

            z_vals = w0 * z_cam[tri[0]] + w1 * z_cam[tri[1]] + w2 * z_cam[tri[2]]
            region = depth[min_y:max_y + 1, min_x:max_x + 1]
            update = mask & (z_vals < region)
            region[update] = z_vals[update]

        return depth

    def _edge_is_visible(self, projected: np.ndarray, z_cam: np.ndarray, depth: np.ndarray,
                         a: int, b: int, res: int, tolerance: float = 0.06) -> bool:
        for t in np.linspace(0.1, 0.9, 5):
            x = projected[a, 0] * (1.0 - t) + projected[b, 0] * t
            y = projected[a, 1] * (1.0 - t) + projected[b, 1] * t
            if not np.isfinite([x, y]).all():
                continue

            px = int(round(x))
            py = int(round(y))
            if px < 0 or px >= res or py < 0 or py >= res:
                continue

            visible_z = depth[py, px]
            edge_z = z_cam[a] * (1.0 - t) + z_cam[b] * t
            if not np.isfinite(visible_z) or edge_z <= visible_z + tolerance:
                return True

        return False

    def _draw_visible_edge(self, draw: ImageDraw.ImageDraw, projected: np.ndarray, z_cam: np.ndarray,
                           depth: np.ndarray, a: int, b: int, res: int, color: int,
                           tolerance: float = 0.08, samples: int = 18):
        visible_points = []
        for t in np.linspace(0.0, 1.0, samples):
            x = projected[a, 0] * (1.0 - t) + projected[b, 0] * t
            y = projected[a, 1] * (1.0 - t) + projected[b, 1] * t
            edge_z = z_cam[a] * (1.0 - t) + z_cam[b] * t
            if edge_z <= 1e-6 or not np.isfinite([x, y, edge_z]).all():
                if len(visible_points) >= 2:
                    draw.line(visible_points, fill=color, width=1)
                visible_points = []
                continue

            px = int(round(x))
            py = int(round(y))
            if px < 0 or px >= res or py < 0 or py >= res:
                if len(visible_points) >= 2:
                    draw.line(visible_points, fill=color, width=1)
                visible_points = []
                continue

            x0 = max(px - 1, 0)
            x1 = min(px + 2, res)
            y0 = max(py - 1, 0)
            y1 = min(py + 2, res)
            local_depth = depth[y0:y1, x0:x1]
            finite_depth = local_depth[np.isfinite(local_depth)]
            is_visible = len(finite_depth) > 0 and edge_z <= float(finite_depth.min()) + tolerance

            if is_visible:
                visible_points.append((float(x), float(y)))
            else:
                if len(visible_points) >= 2:
                    draw.line(visible_points, fill=color, width=1)
                visible_points = []

        if len(visible_points) >= 2:
            draw.line(visible_points, fill=color, width=1)

    def _open_edge_has_background(self, projected: np.ndarray, depth: np.ndarray,
                                  a: int, b: int, res: int, min_length: float = 8.0) -> bool:
        p1 = projected[a]
        p2 = projected[b]
        if not np.isfinite([p1[0], p1[1], p2[0], p2[1]]).all():
            return False

        vec = p2 - p1
        length = float(np.linalg.norm(vec))
        if length < min_length:
            return False

        normal = np.array([-vec[1], vec[0]], dtype=np.float64) / max(length, 1e-6)
        for t in np.linspace(0.2, 0.8, 4):
            p = p1 * (1.0 - t) + p2 * t
            for sign in (-1.0, 1.0):
                sample = p + normal * sign * 3.0
                x = int(round(sample[0]))
                y = int(round(sample[1]))
                if x < 0 or x >= res or y < 0 or y >= res:
                    return True
                if not np.isfinite(depth[y, x]):
                    return True

        return False

    # ========== 新的渲染模式 ==========

    def create_wireframe_image(self, camera, view_name: str,
                               mesh_data: dict = None, shape=None, model_path: str = "") -> Image.Image:
        """
        渲染线框图 - 用于 AI ControlNet Lineart 控制。

        Args:
            camera: dict with camera object
            view_name: Name of the view (front, back, left, right, top, bottom)
            mesh_data: Pre-extracted mesh dict with 'vertices' and 'triangles'

        Returns:
            黑白线框图，白色背景，黑色线条
        """
        try:
            if mesh_data:
                blender_bin = os.getenv("BLENDER_BIN", "").strip() or "blender"
                try:
                    return self._render_blender_image(blender_bin, model_path, camera, view_name, mesh_data, "lineart")
                except Exception as e:
                    print(f"[Renderer] Blender lineart failed for {view_name}: {e}", flush=True)
                    return self._create_placeholder(view_name, f"Blender: {str(e)[:160]}")

            return self._create_placeholder(view_name, "No mesh data")
        except Exception as e:
            print(f"[Renderer] Wireframe render failed for {view_name}: {e}", flush=True)
            return self._create_placeholder(view_name, f"Wireframe: {str(e)[:30]}")

    def _write_mesh_obj(self, mesh_data: dict, output_path: str) -> None:
        vertices = mesh_data.get('vertices') if mesh_data else None
        triangles = mesh_data.get('triangles') if mesh_data else None
        if not vertices or not triangles:
            raise RuntimeError("No mesh data to export for Blender")

        verts_np = _normalize_mesh_vertices(vertices, target_size=2.0)
        with open(output_path, "w", encoding="utf-8") as f:
            f.write("# temporary mesh exported for Blender lineart\n")
            for v in verts_np:
                f.write(f"v {float(v[0])} {float(v[1])} {float(v[2])}\n")
            for tri in triangles:
                f.write(f"f {int(tri[0]) + 1} {int(tri[1]) + 1} {int(tri[2]) + 1}\n")

    def _render_blender_image(self, blender_bin: str, model_path: str, camera, view_name: str,
                              mesh_data: dict = None, mode: str = "lineart") -> Image.Image:
        script_path = os.path.join(os.path.dirname(__file__), "blender_lineart.py")
        if not os.path.exists(script_path):
            raise RuntimeError(f"Blender lineart script not found: {script_path}")

        if hasattr(camera, 'frontend_params') and camera.frontend_params:
            camera_params = dict(camera.frontend_params)
        else:
            camera_params = {
                "position": np.array(getattr(camera, 'pos', [0, 0, 6]), dtype=float).tolist(),
                "target": np.array(getattr(camera, 'target', [0, 0, 0]), dtype=float).tolist(),
                "up": np.array(getattr(camera, 'up', [0, 1, 0]), dtype=float).tolist(),
                "fov": 35,
                "aspect": 1,
            }
        pos, forward, right, up, fov, aspect = self._get_camera_vectors(camera)
        camera_params.update({
            "position": pos.tolist(),
            "target": (pos + forward).tolist(),
            "forward": forward.tolist(),
            "right": right.tolist(),
            "true_up": up.tolist(),
            "fov": fov,
            "aspect": aspect,
        })
        with tempfile.TemporaryDirectory(prefix="blender_lineart_") as tmp_dir:
            input_path = os.path.join(tmp_dir, "model.obj")
            self._write_mesh_obj(mesh_data, input_path)
            normalize_input = "0"
            last_error = ""

            for render_width in (self.resolution,):
                render_height = render_width
                output_path = os.path.join(tmp_dir, f"{view_name}_{mode}_{render_width}.png")
                cmd = [
                    blender_bin,
                    "--background",
                    "--factory-startup",
                    "--python", script_path,
                    "--",
                    "--input", input_path,
                    "--output", output_path,
                    "--camera", json.dumps(camera_params),
                    "--resolution", str(self.resolution),
                    "--width", str(render_width),
                    "--height", str(render_height),
                    "--normalize", normalize_input,
                    "--mode", mode,
                ]
                print(
                    f"[Renderer] Using Blender {mode}: {blender_bin} {input_path} "
                    f"render={render_width}x{render_height} final={self.resolution}",
                    flush=True,
                )
                result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=180)
                if result.returncode != 0:
                    last_error = f"Blender exited with {result.returncode}: {(result.stderr or result.stdout)[-1200:]}"
                    print(f"[Renderer] Blender {mode} failed at {render_width}x{render_height}: {last_error}", flush=True)
                    continue
                if not os.path.exists(output_path):
                    png_files = [
                        os.path.join(tmp_dir, name)
                        for name in os.listdir(tmp_dir)
                        if name.lower().endswith(".png")
                    ]
                    if png_files:
                        output_path = png_files[0]
                    else:
                        log_tail = ((result.stderr or "") + "\n" + (result.stdout or ""))[-1200:]
                        last_error = f"Blender did not produce output image. Log: {log_tail}"
                        print(f"[Renderer] Blender {mode} failed at {render_width}x{render_height}: {last_error}", flush=True)
                        continue

                img = Image.open(output_path).convert("RGB")
                if mode == "lineart":
                    img = _letterbox_control_image(img, render_width, bg_value=255)
                    img = _strengthen_lineart_image(img)
                    img = _shift_control_image(img, bg_value=255)
                elif mode == "depth":
                    img = _letterbox_control_image(img, render_width, bg_value=0)
                    img = _shift_control_image(img, bg_value=0)
                elif img.size != (self.resolution, self.resolution):
                    img = img.resize((self.resolution, self.resolution), Image.LANCZOS)
                return img

        raise RuntimeError(last_error or "Blender render failed")

    def _render_hlr_wireframe(self, shape, camera, view_name: str) -> Image.Image:
        from OCC.Core.HLRBRep import HLRBRep_Algo, HLRBRep_HLRToShape
        from OCC.Core.HLRAlgo import HLRAlgo_Projector
        from OCC.Core.TopAbs import TopAbs_EDGE
        from OCC.Core.TopExp import TopExp_Explorer
        from OCC.Core.BRepAdaptor import BRepAdaptor_Curve
        from OCC.Core.gp import gp_Ax2, gp_Dir, gp_Pnt

        res = self.resolution
        _, forward, right, up, _, _ = self._get_camera_vectors(camera)
        view_dir = -forward
        axis = gp_Ax2(
            gp_Pnt(0, 0, 0),
            gp_Dir(float(view_dir[0]), float(view_dir[1]), float(view_dir[2])),
            gp_Dir(float(right[0]), float(right[1]), float(right[2])),
        )

        algo = HLRBRep_Algo()
        algo.Add(shape)
        algo.Projector(HLRAlgo_Projector(axis))
        algo.Update()
        algo.Hide()
        hlr_shape = HLRBRep_HLRToShape(algo)

        edge_shapes = []
        for compound in (
            hlr_shape.VCompound(),
            hlr_shape.OutLineVCompound(),
            hlr_shape.Rg1LineVCompound(),
            hlr_shape.RgNLineVCompound(),
            hlr_shape.IsoLineVCompound(),
            hlr_shape.HCompound(),
            hlr_shape.OutLineHCompound(),
            hlr_shape.Rg1LineHCompound(),
            hlr_shape.RgNLineHCompound(),
            hlr_shape.IsoLineHCompound(),
        ):
            explorer = TopExp_Explorer(compound, TopAbs_EDGE)
            while explorer.More():
                edge_shapes.append(explorer.Current())
                explorer.Next()

        if not edge_shapes:
            raise RuntimeError("No HLR edges")

        polylines = []
        bounds = []
        for edge in edge_shapes:
            try:
                curve = BRepAdaptor_Curve(edge)
                first = float(curve.FirstParameter())
                last = float(curve.LastParameter())
            except Exception as e:
                print(f"[Renderer] HLR edge skipped for {view_name}: {e}", flush=True)
                continue

            if not np.isfinite(first) or not np.isfinite(last) or abs(last - first) <= 1e-12:
                continue

            params = np.linspace(first, last, 32)
            pts = []
            for param in params:
                try:
                    p = curve.Value(float(param))
                    v = np.array([p.X(), p.Y(), p.Z()], dtype=np.float64)
                    x = float(v @ right)
                    y = float(v @ up)
                    pts.append((x, y))
                    bounds.append((x, y))
                except Exception:
                    continue
            if len(pts) >= 2:
                polylines.append(pts)

        if not bounds:
            raise RuntimeError("No valid HLR edges")

        b = np.array(bounds, dtype=np.float64)
        min_xy = b.min(axis=0)
        max_xy = b.max(axis=0)
        center = (min_xy + max_xy) / 2.0
        size = max(max_xy - min_xy)
        if size <= 1e-9:
            return self._create_placeholder(view_name, "Empty HLR bounds")

        margin = res * 0.08
        scale = (res - margin * 2) / size
        img = Image.new('L', (res, res), 255)
        draw = ImageDraw.Draw(img)
        for pts in polylines:
            screen = []
            for x, y in pts:
                sx = (x - center[0]) * scale + res / 2.0
                sy = res / 2.0 - (y - center[1]) * scale
                screen.append((float(sx), float(sy)))
            draw.line(screen, fill=0, width=1)

        return Image.merge('RGB', [img, img, img])

    def _render_wireframe_legacy(self, camera, view_name: str,
                         mesh_data: dict) -> Image.Image:
        """
        使用 matplotlib 渲染线框图。
        只显示边缘线，不填充面，产生干净的线稿效果。
        """
        import matplotlib.gridspec as gridspec

        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']

        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = np.array(vertices, dtype=np.float64)

        # 归一化到 [-0.5, 0.5]
        v_min = verts_np.min(axis=0)
        v_max = verts_np.max(axis=0)
        v_center = (v_min + v_max) / 2.0
        v_size = max(v_max - v_min)

        if v_size > 0:
            verts_np = (verts_np - v_center) / v_size

        elev, azim = VIEW_ANGLES.get(view_name, (0, 0))

        # 创建新 figure 渲染线框
        fig = plt.figure(figsize=(res / 100, res / 100), dpi=100)
        ax = fig.add_subplot(111, projection='3d')
        ax.set_box_aspect([1, 1, 1])
        ax.axis('off')
        ax.set_xlim(-0.6, 0.6)
        ax.set_ylim(-0.6, 0.6)
        ax.set_zlim(-0.6, 0.6)
        ax.dist = 8
        ax.view_init(elev=elev, azim=azim)

        tri_np = np.array(triangles, dtype=np.int64)
        valid = np.all((tri_np >= 0) & (tri_np < len(verts_np)), axis=1)
        tri_np = tri_np[valid]

        if len(tri_np) > 0:
            edges = np.concatenate([
                tri_np[:, [0, 1]],
                tri_np[:, [1, 2]],
                tri_np[:, [2, 0]],
            ], axis=0)
            edges = np.sort(edges, axis=1)
            edges = np.unique(edges, axis=0)
            segments = verts_np[edges]
            ax.add_collection3d(Line3DCollection(segments, colors='black', linewidths=0.25))

        fig.tight_layout(pad=0)
        fig.canvas.draw()

        # 转换为灰度图（白底黑线）
        buf = fig.canvas.tostring_argb()
        w, h = fig.canvas.get_width_height()
        arr = np.frombuffer(buf, dtype=np.uint8).reshape((h, w, 4))
        arr = arr[:, :, 1:]  # ARGB -> RGB

        # RGB 转灰度：L = 0.299*R + 0.587*G + 0.114*B
        gray = (0.299 * arr[:, :, 0] + 0.587 * arr[:, :, 1] + 0.114 * arr[:, :, 2]).astype(np.uint8)

        # 二值化：线条变黑，背景变白
        # 阈值 200：低于 200 认为是线条，高于 200 认为是背景
        binary = np.where(gray < 200, 0, 255).astype(np.uint8)

        img = Image.fromarray(binary, 'L')  # L = 灰度模式
        img = img.resize((res, res), Image.LANCZOS)
        img_rgb = Image.merge('RGB', [img, img, img])  # 转回 RGB（黑白图）

        plt.close(fig)
        return img_rgb

    def _render_wireframe(self, camera, view_name: str,
                         mesh_data: dict) -> Image.Image:
        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']
        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = _normalize_mesh_vertices(vertices)
        tri_np = np.array(triangles, dtype=np.int64)
        valid = np.all((tri_np >= 0) & (tri_np < len(verts_np)), axis=1)
        tri_np = tri_np[valid]
        if len(tri_np) == 0:
            return self._create_placeholder(view_name, "No triangles")

        scale = 3
        render_res = res * scale
        projected, z_cam = self._project_mesh(verts_np, camera, render_res)
        edge_np, open_edge_np = self._edge_segments_for_view(verts_np, tri_np, camera)
        depth = self._rasterize_depth(projected, z_cam, tri_np, render_res)

        img = Image.new('L', (render_res, render_res), 255)
        draw = ImageDraw.Draw(img)
        for edges, require_background in ((edge_np, False), (open_edge_np, True)):
            for a, b in edges:
                if z_cam[a] <= 1e-6 or z_cam[b] <= 1e-6:
                    continue
                if not np.isfinite([projected[a, 0], projected[a, 1], projected[b, 0], projected[b, 1]]).all():
                    continue

                if require_background and not self._open_edge_has_background(projected, depth, a, b, render_res):
                    continue

                self._draw_visible_edge(draw, projected, z_cam, depth, a, b, render_res, 0, tolerance=0.08)

        img = img.resize((res, res), Image.LANCZOS)
        return Image.merge('RGB', [img, img, img])

    def create_depth_map(self, camera, view_name: str,
                         mesh_data: dict = None, model_path: str = "") -> Image.Image:
        """
        渲染深度图 - 用于 AI ControlNet Depth 控制。

        Args:
            camera: dict with camera object
            view_name: Name of the view (front, back, left, right, top, bottom)
            mesh_data: Pre-extracted mesh dict with 'vertices' and 'triangles'

        Returns:
            深度图，白色=近，黑色=远
        """
        try:
            if mesh_data:
                blender_bin = os.getenv("BLENDER_BIN", "").strip() or "blender"
                return self._render_blender_image(blender_bin, model_path, camera, view_name, mesh_data, "depth")

            return self._create_placeholder(view_name, "No mesh data")
        except Exception as e:
            print(f"[Renderer] Depth render failed for {view_name}: {e}", flush=True)
            return self._create_placeholder(view_name, f"Depth: {str(e)[:30]}")

    def _render_depth_map_legacy(self, camera, view_name: str,
                         mesh_data: dict) -> Image.Image:
        """
        使用 matplotlib 渲染深度图。
        每个像素的值表示该点到相机的距离（越近越白，越远越黑）。
        """
        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']

        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = np.array(vertices, dtype=np.float64)

        # 归一化到 [-0.5, 0.5]
        v_min = verts_np.min(axis=0)
        v_max = verts_np.max(axis=0)
        v_center = (v_min + v_max) / 2.0
        v_size = max(v_max - v_min)

        if v_size > 0:
            verts_np = (verts_np - v_center) / v_size

        elev, azim = get_matplotlib_view(camera, view_name)

        # 创建新 figure 渲染深度
        fig = plt.figure(figsize=(res / 100, res / 100), dpi=100)
        ax = fig.add_subplot(111, projection='3d')
        ax.set_box_aspect([1, 1, 1])
        ax.axis('off')
        ax.set_xlim(-0.6, 0.6)
        ax.set_ylim(-0.6, 0.6)
        ax.set_zlim(-0.6, 0.6)
        ax.dist = 8
        ax.view_init(elev=elev, azim=azim)

        # 绘制 mesh，用 z 值作为颜色（模拟深度）
        mesh_triangles = []
        for tri in triangles:
            try:
                mesh_triangles.append([verts_np[tri[0]], verts_np[tri[1]], verts_np[tri[2]]])
            except IndexError:
                continue

        if mesh_triangles:
            collection = Poly3DCollection(
                mesh_triangles,
                alpha=1.0,
                edgecolor='none',
                facecolor='gray',  # 统一灰色，后续处理深度
                linewidth=0,
            )
            ax.add_collection3d(collection)

        fig.tight_layout(pad=0)
        fig.canvas.draw()

        # 获取渲染缓冲区
        buf = fig.canvas.tostring_argb()
        w, h = fig.canvas.get_width_height()
        arr = np.frombuffer(buf, dtype=np.uint8).reshape((h, w, 4))
        arr = arr[:, :, 1:]  # ARGB -> RGB

        # 转换为灰度图
        gray = (0.299 * arr[:, :, 0] + 0.587 * arr[:, :, 1] + 0.114 * arr[:, :, 2]).astype(np.uint8)

        # 反转深度：产品部分变白，背景保持黑/深灰
        # 原始渲染的背景是白色 (255)，产品是灰色 (约 200)
        # 我们需要：产品 = 白 (255)，背景 = 黑 (0)
        depth = np.where(gray < 200, gray, 0)  # 只保留产品部分

        # 线性拉伸到 0-255 范围
        if depth.max() > depth.min():
            depth_norm = ((depth - depth.min()) / (depth.max() - depth.min()) * 255).astype(np.uint8)
        else:
            depth_norm = depth

        img = Image.fromarray(depth_norm, 'L')
        img = img.resize((res, res), Image.LANCZOS)
        img_rgb = Image.merge('RGB', [img, img, img])

        plt.close(fig)
        return img_rgb

    def _render_depth_map(self, camera, view_name: str,
                         mesh_data: dict) -> Image.Image:
        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']
        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = _normalize_mesh_vertices(vertices)
        tri_np = np.array(triangles, dtype=np.int64)
        valid = np.all((tri_np >= 0) & (tri_np < len(verts_np)), axis=1)
        tri_np = tri_np[valid]
        if len(tri_np) == 0:
            return self._create_placeholder(view_name, "No triangles")

        projected, z_cam = self._project_mesh(verts_np, camera, res)
        depth = np.full((res, res), np.inf, dtype=np.float64)

        for tri in tri_np:
            if np.any(z_cam[tri] <= 1e-6):
                continue
            pts = projected[tri]
            if not np.isfinite(pts).all():
                continue

            min_x = max(int(np.floor(np.min(pts[:, 0]))), 0)
            max_x = min(int(np.ceil(np.max(pts[:, 0]))), res - 1)
            min_y = max(int(np.floor(np.min(pts[:, 1]))), 0)
            max_y = min(int(np.ceil(np.max(pts[:, 1]))), res - 1)
            if min_x > max_x or min_y > max_y:
                continue

            p0, p1, p2 = pts
            denom = ((p1[1] - p2[1]) * (p0[0] - p2[0]) +
                     (p2[0] - p1[0]) * (p0[1] - p2[1]))
            if abs(denom) < 1e-8:
                continue

            xs = np.arange(min_x, max_x + 1)
            ys = np.arange(min_y, max_y + 1)
            xx, yy = np.meshgrid(xs, ys)
            w0 = ((p1[1] - p2[1]) * (xx - p2[0]) +
                  (p2[0] - p1[0]) * (yy - p2[1])) / denom
            w1 = ((p2[1] - p0[1]) * (xx - p2[0]) +
                  (p0[0] - p2[0]) * (yy - p2[1])) / denom
            w2 = 1.0 - w0 - w1
            mask = (w0 >= 0) & (w1 >= 0) & (w2 >= 0)
            if not np.any(mask):
                continue

            z_vals = w0 * z_cam[tri[0]] + w1 * z_cam[tri[1]] + w2 * z_cam[tri[2]]
            region = depth[min_y:max_y + 1, min_x:max_x + 1]
            update = mask & (z_vals < region)
            region[update] = z_vals[update]

        img_arr = np.zeros((res, res), dtype=np.uint8)
        finite = np.isfinite(depth)
        if np.any(finite):
            near = depth[finite].min()
            far = depth[finite].max()
            if far > near:
                img_arr[finite] = ((far - depth[finite]) / (far - near) * 255).astype(np.uint8)
            else:
                img_arr[finite] = 255

        img = Image.fromarray(img_arr, 'L')
        return Image.merge('RGB', [img, img, img])

    def create_silhouette(self, camera, view_name: str,
                          mesh_data: dict = None) -> Image.Image:
        """
        渲染剪影图 - 纯产品轮廓，白色背景，黑色产品。

        Args:
            camera: dict with camera object
            view_name: Name of the view (front, back, left, right, top, bottom)
            mesh_data: Pre-extracted mesh dict with 'vertices' and 'triangles'

        Returns:
            剪影图，白色背景，黑色产品轮廓
        """
        try:
            return self._render_silhouette(camera, view_name, mesh_data)
        except Exception as e:
            print(f"[Renderer] Silhouette render failed for {view_name}: {e}", flush=True)
            return self._create_placeholder(view_name, f"Silhouette: {str(e)[:30]}")

    def _render_silhouette(self, camera, view_name: str,
                           mesh_data: dict) -> Image.Image:
        """
        渲染剪影图 - 只保留产品轮廓，不渲染内部细节。
        """
        vertices = mesh_data['vertices']
        triangles = mesh_data['triangles']

        if not vertices:
            return self._create_placeholder(view_name, "No mesh data")

        res = self.resolution
        verts_np = np.array(vertices, dtype=np.float64)

        # 归一化到 [-0.5, 0.5]
        v_min = verts_np.min(axis=0)
        v_max = verts_np.max(axis=0)
        v_center = (v_min + v_max) / 2.0
        v_size = max(v_max - v_min)

        if v_size > 0:
            verts_np = (verts_np - v_center) / v_size

        elev, azim = get_matplotlib_view(camera, view_name)

        # 创建新 figure
        fig = plt.figure(figsize=(res / 100, res / 100), dpi=100)
        ax = fig.add_subplot(111, projection='3d')
        ax.set_box_aspect([1, 1, 1])
        ax.axis('off')
        ax.set_xlim(-0.6, 0.6)
        ax.set_ylim(-0.6, 0.6)
        ax.set_zlim(-0.6, 0.6)
        ax.dist = 8
        ax.view_init(elev=elev, azim=azim)

        # 绘制 mesh 为纯黑色
        mesh_triangles = []
        for tri in triangles:
            try:
                mesh_triangles.append([verts_np[tri[0]], verts_np[tri[1]], verts_np[tri[2]]])
            except IndexError:
                continue

        if mesh_triangles:
            collection = Poly3DCollection(
                mesh_triangles,
                alpha=1.0,
                edgecolor='none',
                facecolor='black',
                linewidth=0,
            )
            ax.add_collection3d(collection)

        fig.tight_layout(pad=0)
        fig.canvas.draw()

        # 获取缓冲区并转为灰度
        buf = fig.canvas.tostring_argb()
        w, h = fig.canvas.get_width_height()
        arr = np.frombuffer(buf, dtype=np.uint8).reshape((h, w, 4))
        arr = arr[:, :, 1:]  # ARGB -> RGB

        gray = (0.299 * arr[:, :, 0] + 0.587 * arr[:, :, 1] + 0.114 * arr[:, :, 2]).astype(np.uint8)

        # 二值化：产品变黑，背景变白
        binary = np.where(gray < 200, 0, 255).astype(np.uint8)

        img = Image.fromarray(binary, 'L')
        img = img.resize((res, res), Image.LANCZOS)
        img_rgb = Image.merge('RGB', [img, img, img])

        plt.close(fig)
        return img_rgb
