import numpy as np

# 6 个视角配置 - 基于标准工程坐标系 (Z-up)
# 每个视角由"眼睛位置"和"目标位置"定义
VIEW_CONFIGS = [
    {'name': 'front',   'pos': (0, -1, 0),   'target': (0, 0, 0),  'up': (0, 0, 1)},
    {'name': 'back',    'pos': (0, 1, 0),    'target': (0, 0, 0),  'up': (0, 0, 1)},
    {'name': 'left',    'pos': (-1, 0, 0),   'target': (0, 0, 0),  'up': (0, 0, 1)},
    {'name': 'right',   'pos': (1, 0, 0),     'target': (0, 0, 0),  'up': (0, 0, 1)},
    {'name': 'top',     'pos': (0, 0, 1),     'target': (0, 0, 0),  'up': (0, -1, 0)},
    {'name': 'bottom',  'pos': (0, 0, -1),    'target': (0, 0, 0),  'up': (0, 1, 0)},
]


def compute_model_center_and_scale(mesh_info: dict) -> tuple:
    """
    根据模型包围盒计算归一化后的中心点和缩放因子。
    用于将任意 STP 文件归一化到统一的视图空间。
    """
    min_p = mesh_info['min']
    max_p = mesh_info['max']

    # 模型中心点
    center = (
        (min_p[0] + max_p[0]) / 2.0,
        (min_p[1] + max_p[1]) / 2.0,
        (min_p[2] + max_p[2]) / 2.0,
    )

    # 最大尺寸（用于归一化）
    size = (
        max_p[0] - min_p[0],
        max_p[1] - min_p[1],
        max_p[2] - min_p[2],
    )
    max_dim = max(size)

    return center, size, max_dim

class Camera:
    def __init__(self, pos: tuple, target: tuple, up: tuple, distance: float):
        # 将模型中心平移到原点，然后设置相机位置
        # pos 和 target 已经是相对于模型中心的方向
        self.pos = np.array(pos) * distance
        self.target = np.array(target)  # 通常是 (0, 0, 0)，模型中心
        self.up = np.array(up)
        # 确保 up 向量与视线方向正交
        forward = self.target - self.pos
        if np.linalg.norm(forward) > 1e-6:
            forward = forward / np.linalg.norm(forward)
            self.up = self.up - np.dot(self.up, forward) * forward
            if np.linalg.norm(self.up) > 1e-6:
                self.up = self.up / np.linalg.norm(self.up)
            else:
                # 如果 up 与 forward 平行，使用默认 up
                self.up = np.array([0, 0, 1])

    def get_extrinsic_matrix(self) -> np.ndarray:
        """计算相机外参矩阵 (Open3D 格式: [R|t])"""
        f = self.target - self.pos
        f = f / np.linalg.norm(f)
        s = np.cross(f, self.up)
        s = s / np.linalg.norm(s)
        u = np.cross(s, f)

        # Open3D extrinsic: [R|t] where R is column-major
        # Row-major: [s, u, -f, 0] with translation
        extrinsic = np.zeros((4, 4))
        extrinsic[0, :3] = s
        extrinsic[1, :3] = u
        extrinsic[2, :3] = -f
        extrinsic[3, 3] = 1.0

        # 相机位置作为平移向量（负号因为相机看向目标）
        extrinsic[0, 3] = -np.dot(s, self.pos)
        extrinsic[1, 3] = -np.dot(u, self.pos)
        extrinsic[2, 3] = np.dot(f, self.pos)

        return extrinsic

    def get_view_matrix(self) -> np.ndarray:
        """计算 View Matrix (look-at matrix)"""
        f = self.target - self.pos
        f = f / np.linalg.norm(f)
        s = np.cross(f, self.up)
        s = s / np.linalg.norm(s)
        u = np.cross(s, f)

        view = np.identity(4)
        view[0, :3] = s
        view[1, :3] = u
        view[2, :3] = -f
        view[0, 3] = -np.dot(s, self.pos)
        view[1, 3] = -np.dot(u, self.pos)
        view[2, 3] = np.dot(f, self.pos)

        return view

    @staticmethod
    def calculate_distance(bbox_size: tuple, factor: float = 2.5) -> float:
        """根据包围盒计算相机距离"""
        max_dim = max(bbox_size)
        return max_dim * factor

    def get_position(self) -> tuple:
        """获取相机位置"""
        return tuple(self.pos)

    def get_up_vector(self) -> tuple:
        """获取相机 up 向量"""
        return tuple(self.up)

    def get_rotation_matrix(self) -> list:
        """获取相机旋转矩阵（用于视角计算）"""
        f = self.target - self.pos
        f = f / np.linalg.norm(f)
        s = np.cross(f, self.up)
        s = s / np.linalg.norm(s)
        u = np.cross(s, f)
        # 返回列主序的 3x3 旋转矩阵
        return [[s[0], u[0], -f[0]],
                [s[1], u[1], -f[1]],
                [s[2], u[2], -f[2]]]

    def get_camera_object(self):
        """获取 OCC 相机对象（用于 OCC 渲染）"""
        try:
            from OCC.Core.Graphic3d import Graphic3d_Camera
            from OCC.Core.gp import gp_Pnt, gp_Dir, gp_Ax3

            cam = Graphic3d_Camera()
            # 设置相机位置和方向
            eye = gp_Pnt(self.pos[0], self.pos[1], self.pos[2])
            target = gp_Pnt(self.target[0], self.target[1], self.target[2])

            # 计算视线方向
            forward = self.target - self.pos
            forward = forward / np.linalg.norm(forward)
            fwd_dir = gp_Dir(forward[0], forward[1], forward[2])

            # 设置相机轴
            cam.SetAxialTrihedron(gp_Ax3(target, fwd_dir))
            return cam
        except Exception as e:
            print(f"[Camera] get_camera_object failed: {e}")
            return None

def create_cameras_for_view(mesh_info: dict, factor: float = 2.5) -> list:
    """为 6 个视角创建相机"""
    dx, dy, dz = mesh_info['size']
    distance = max(dx, dy, dz) * factor

    cameras = []
    for config in VIEW_CONFIGS:
        cam = Camera(
            pos=config['pos'],
            target=config['target'],
            up=config['up'],
            distance=distance
        )
        cameras.append({
            'name': config['name'],
            'camera': cam,
            'description': f"[六面-{config['name']}]"
        })

    return cameras


def create_camera_from_angle(mesh_info: dict, azimuth: float, elevation: float, distance: float = None) -> dict:
    """
    根据方位角和仰角创建相机。
    - azimuth: 方位角（度），0=正Z方向，90=正X方向
    - elevation: 仰角（度），0=水平面，90=正上方
    - distance: 相机到模型中心的距离，默认根据模型大小计算
    """
    import math
    dx, dy, dz = mesh_info['size']
    if distance is None:
        distance = max(dx, dy, dz) * 2.5

    # 转换为弧度
    az_rad = math.radians(azimuth)
    el_rad = math.radians(elevation)

    # 计算相机位置（球坐标转笛卡尔坐标）
    # 使用 OpenGL/Open3D 坐标系：Y-up，Z 向外
    pos = (
        distance * math.cos(el_rad) * math.sin(az_rad),
        distance * math.cos(el_rad) * math.cos(az_rad),
        distance * math.sin(el_rad)
    )

    # 默认 up 向量
    up = (0, 0, 1)

    cam = Camera(pos=pos, target=(0, 0, 0), up=up, distance=1)

    return {
        'name': 'single',
        'camera': cam,
        'description': '[单视角]'
    }