from OCC.Core.BRepMesh import BRepMesh_IncrementalMesh
from OCC.Core.BRepBndLib import brepbndlib
from OCC.Core.Bnd import Bnd_Box
import gc
import os
import numpy as np


class StepLoader:
    def __init__(self):
        pass

    def load(self, file_path: str) -> tuple:
        """加载 STP 文件，返回 shape 和包围盒"""
        from OCC.Core.IFSelect import IFSelect_RetDone
        from OCC.Core.STEPControl import STEPControl_Reader

        reader = STEPControl_Reader()
        status = reader.ReadFile(file_path)

        if status != IFSelect_RetDone:
            raise ValueError(f"Failed to read STEP file (status={status}): {file_path}")

        try:
            reader.TransferRoots()
            shape = reader.OneShape()
        except Exception as e:
            raise ValueError(f"TransferRoots/OneShape failed: {e}")

        if shape.IsNull():
            raise ValueError(f"OneShape returned null shape for: {file_path}")

        # 计算包围盒
        bbox = Bnd_Box()
        brepbndlib.Add(shape, bbox)

        return shape, bbox

    def is_step_file(self, file_path: str) -> bool:
        ext = os.path.splitext(file_path)[1].lower()
        return ext in ('.stp', '.step')

    def load_mesh_file(self, file_path: str, simplify: bool = True, target_triangles: int = 50000,
                       max_file_size_mb: int = 200, max_triangles: int = 200000) -> tuple:
        """Load OBJ/FBX mesh files and return mesh_info + mesh_data.

        Args:
            file_path: Path to mesh file
            simplify: Whether to simplify the mesh (recommended for large files)
            target_triangles: Target triangle count after simplification (default 50k)
            max_file_size_mb: Maximum file size in MB to prevent OOM (default 200MB)
            max_triangles: Hard triangle limit after loading/simplification.
        """
        ext = os.path.splitext(file_path)[1].lower()
        if ext not in ('.obj', '.fbx'):
            raise ValueError(f"Unsupported mesh file format: {ext}")

        # 检查文件大小，防止 OOM
        file_size_mb = os.path.getsize(file_path) / (1024 * 1024)
        print(f"[StepLoader] File size: {file_size_mb:.1f} MB", flush=True)
        if file_size_mb > max_file_size_mb:
            raise ValueError(f"文件过大（{file_size_mb:.1f}MB），最大支持 {max_file_size_mb}MB。请尝试将 FBX 导出为 OBJ 格式。")

        vertices = []
        triangles = []
        loaded_mesh = None  # 用于释放原始 mesh 对象

        try:
            import open3d as o3d
            print(f"[StepLoader] Loading {ext} with open3d...", flush=True)
            loaded_mesh = o3d.io.read_triangle_mesh(file_path, enable_post_processing=True)
            if loaded_mesh is not None and len(loaded_mesh.vertices) > 0 and len(loaded_mesh.triangles) > 0:
                source_triangles = len(loaded_mesh.triangles)
                if simplify and source_triangles > target_triangles:
                    print(f"[StepLoader] Simplifying mesh: {source_triangles} -> {target_triangles} triangles", flush=True)
                    mesh = loaded_mesh.simplify_quadric_decimation(target_triangles)
                    print(f"[StepLoader] Simplification complete: {len(mesh.triangles)} triangles", flush=True)
                    del loaded_mesh
                    loaded_mesh = None
                    if max_triangles > 0 and len(mesh.triangles) > max_triangles:
                        raise ValueError(
                            f"Mesh is too complex after simplification: {len(mesh.triangles)} triangles > {max_triangles}"
                        )
                    vertices = np.asarray(mesh.vertices, dtype=np.float64).tolist()
                    triangles = np.asarray(mesh.triangles, dtype=np.int64).tolist()
                    del mesh  # 释放简化后的 mesh
                else:
                    if max_triangles > 0 and source_triangles > max_triangles:
                        raise ValueError(
                            f"Mesh is too complex: {source_triangles} triangles > {max_triangles}"
                        )
                    vertices = np.asarray(loaded_mesh.vertices, dtype=np.float64).tolist()
                    triangles = np.asarray(loaded_mesh.triangles, dtype=np.int64).tolist()
                print(f"[StepLoader] open3d loaded: {len(vertices)} verts, {len(triangles)} tris", flush=True)
            if loaded_mesh is not None:
                del loaded_mesh
            loaded_mesh = None
        except Exception as e:
            print(f"[StepLoader] open3d mesh load failed for {file_path}: {e}", flush=True)
            if loaded_mesh is not None:
                del loaded_mesh
                loaded_mesh = None
            if "too complex" in str(e):
                raise

        if not vertices or not triangles:
            try:
                import trimesh
                print(f"[StepLoader] Trying trimesh for {file_path}...", flush=True)
                loaded = trimesh.load(file_path, force='scene')
                meshes = []
                if hasattr(loaded, 'geometry'):
                    meshes = list(loaded.geometry.values())
                elif hasattr(loaded, 'vertices') and hasattr(loaded, 'faces'):
                    meshes = [loaded]

                vertex_offset = 0
                item_vertices = None
                item_faces = None
                for item in meshes:
                    if not hasattr(item, 'vertices') or not hasattr(item, 'faces'):
                        continue

                    item_vertices = np.asarray(item.vertices, dtype=np.float64)
                    item_faces = np.asarray(item.faces, dtype=np.int64)
                    if item_vertices.size == 0 or item_faces.size == 0:
                        continue

                    # 简化大网格
                    if simplify and len(item_faces) > target_triangles:
                        try:
                            print(f"[StepLoader] Simplifying trimesh: {len(item_faces)} -> {target_triangles}", flush=True)
                            simplified = item.simplify_quadric_decimation(target_triangles)
                            item_vertices = np.asarray(simplified.vertices, dtype=np.float64)
                            item_faces = np.asarray(simplified.faces, dtype=np.int64)
                            del simplified  # 释放简化后的对象
                        except Exception as simp_err:
                            print(f"[StepLoader] Simplification failed, using original: {simp_err}", flush=True)

                    if max_triangles > 0 and len(triangles) + len(item_faces) > max_triangles:
                        raise ValueError(
                            f"Mesh is too complex after simplification: "
                            f"{len(triangles) + len(item_faces)} triangles > {max_triangles}"
                        )

                    vertices.extend(item_vertices.tolist())
                    triangles.extend((item_faces + vertex_offset).tolist())
                    vertex_offset += len(item_vertices)

                # 释放大对象
                del loaded
                del meshes
                if item_vertices is not None:
                    del item_vertices
                if item_faces is not None:
                    del item_faces
                gc.collect()

                print(f"[StepLoader] trimesh loaded: {len(vertices)} verts, {len(triangles)} tris", flush=True)
            except Exception as e:
                raise ValueError(
                    f"Failed to read {ext} file. Install trimesh with an assimp-capable runtime for FBX, or convert to OBJ/STP first: {e}"
                )

        if not vertices or not triangles:
            raise ValueError(f"Failed to read mesh file or mesh is empty: {file_path}")
        if max_triangles > 0 and len(triangles) > max_triangles:
            raise ValueError(
                f"Mesh is too complex after simplification: {len(triangles)} triangles > {max_triangles}"
            )

        verts_np = np.array(vertices, dtype=np.float64)
        min_p = verts_np.min(axis=0)
        max_p = verts_np.max(axis=0)
        size = max_p - min_p
        mesh_info = {
            'min': (float(min_p[0]), float(min_p[1]), float(min_p[2])),
            'max': (float(max_p[0]), float(max_p[1]), float(max_p[2])),
            'size': (float(size[0]), float(size[1]), float(size[2])),
        }
        mesh_data = {'vertices': vertices, 'triangles': triangles}
        print(f"[StepLoader] Mesh file loaded: {len(vertices)} verts, {len(triangles)} tris", flush=True)

        # 释放中间 numpy 数组
        del verts_np
        gc.collect()

        return mesh_info, mesh_data

    def get_mesh_data(self, shape, bbox, deflection: float = 0.3) -> dict:
        """将 shape 三角化并返回顶点数据（供多视角复用）"""
        # 使用相对较大的 deflection 减少三角形数量，提升渲染性能
        # 0.3mm 对大多数机械零件足够精细，同时保证渲染速度
        mesh = BRepMesh_IncrementalMesh(shape, deflection)
        mesh.Perform()

        min_p = bbox.CornerMin()
        max_p = bbox.CornerMax()

        dx = max_p.X() - min_p.X()
        dy = max_p.Y() - min_p.Y()
        dz = max_p.Z() - min_p.Z()

        return {
            'min': (min_p.X(), min_p.Y(), min_p.Z()),
            'max': (max_p.X(), max_p.Y(), max_p.Z()),
            'size': (dx, dy, dz)
        }

    def extract_vertices_triangles(self, shape, bbox, deflection: float = 0.3) -> dict:
        """
        从 OCC shape 提取顶点 + 三角形索引，供渲染复用。
        只生成一次，6 个视角共享同一个 mesh 数据。
        """
        from OCC.Core.TopAbs import TopAbs_FACE
        from OCC.Core.TopExp import TopExp_Explorer
        from OCC.Core.BRep import BRep_Tool
        from OCC.Core.TopLoc import TopLoc_Location
        from OCC.Core.gp import gp_Pnt

        mesh_gen = BRepMesh_IncrementalMesh(shape, deflection)
        mesh_gen.Perform()

        vertices = []
        triangles = []
        vertex_index = 0
        faces_checked = 0
        faces_with_mesh = 0

        explorer = TopExp_Explorer(shape, TopAbs_FACE)
        while explorer.More():
            face = explorer.Current()
            location = face.Location()
            faces_checked += 1

            triangulation = BRep_Tool.Triangulation(face, location)
            if triangulation is None:
                explorer.Next()
                continue

            faces_with_mesh += 1
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

        print(f"[StepLoader] Faces: {faces_checked}, mesh'd: {faces_with_mesh}, "
              f"verts: {len(vertices)}, tris: {len(triangles)}", flush=True)

        return {'vertices': vertices, 'triangles': triangles}
