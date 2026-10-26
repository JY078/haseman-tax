#!/usr/bin/env python3
"""
STP 渲染诊断脚本
在服务器上运行此脚本测试 STP 文件读取和渲染
"""
import os
import sys
import tempfile
import shutil

print("=" * 60)
print("STP 渲染诊断脚本")
print("=" * 60)

# 检查 Python 版本和模块
print(f"\n[1] Python 环境:")
print(f"    Python: {sys.version}")
print(f"    Executable: {sys.executable}")

print(f"\n[2] 依赖模块检查:")
modules = [
    ('redis', 'Redis'),
    ('oss2', 'OSS'),
    ('PIL', 'Pillow'),
    ('numpy', 'NumPy'),
    ('open3d', 'Open3D'),
    ('trimesh', 'trimesh'),
    ('pyassimp', 'pyassimp'),
    ('OCC', 'OpenCASCADE'),
]

for module_name, display_name in modules:
    try:
        if module_name == 'OCC':
            from OCC.Core import *
            print(f"    {display_name}: OK")
        elif module_name == 'open3d':
            import open3d as o3d
            print(f"    {display_name}: OK (v{o3d.__version__ if hasattr(o3d, '__version__') else 'unknown'})")
        else:
            mod = __import__(module_name)
            ver = getattr(mod, '__version__', 'unknown')
            print(f"    {display_name}: OK (v{ver})")
    except ImportError as e:
        print(f"    {display_name}: MISSING ({e})")
    except Exception as e:
        print(f"    {display_name}: ERROR ({e})")

# 检查配置文件
print(f"\n[3] 配置文件检查:")
config_vars = [
    'REDIS_HOST', 'REDIS_PORT', 'REDIS_DB',
    'GO_API_BASE_URL', 'OSS_ACCESS_KEY', 'OSS_ACCESS_SECRET',
    'OSS_ENDPOINT', 'OSS_BUCKET', 'OSS_DOMAIN'
]
for var in config_vars:
    val = os.getenv(var, 'NOT_SET')
    if var in ('OSS_ACCESS_KEY', 'OSS_ACCESS_SECRET'):
        val = '***' if val and val != 'NOT_SET' else val
    print(f"    {var}={val}")

# 测试 STP 文件读取
print(f"\n[4] STP 文件读取测试:")

# 先检查是否有测试文件
test_stp = None
possible_paths = [
    '/tmp/test_model.stp',
    './test_model.stp',
    '/uploads/stp/test_model.stp',
]

for path in possible_paths:
    if os.path.exists(path):
        test_stp = path
        break

if not test_stp:
    print("    没有找到测试 STP 文件，跳过读取测试")
    print("    提示: 请先上传一个 STP 文件，然后运行此脚本")
else:
    print(f"    使用测试文件: {test_stp}")
    print(f"    文件大小: {os.path.getsize(test_stp)} bytes")

    # 读取文件头部
    with open(test_stp, 'rb') as f:
        header = f.read(200)
    print(f"    文件头部 (前100字节):")
    print(f"    {header[:100]}")

    # 检查编码
    try:
        with open(test_stp, 'r', encoding='utf-8') as f:
            lines = f.readlines()
        print(f"    编码检查: UTF-8 OK (共 {len(lines)} 行)")
    except UnicodeDecodeError as e:
        print(f"    编码检查: UTF-8 失败 - {e}")
        try:
            with open(test_stp, 'r', encoding='latin-1') as f:
                lines = f.readlines()
            print(f"    编码检查: Latin-1 OK (共 {len(lines)} 行)")
        except Exception as e2:
            print(f"    编码检查: Latin-1 也失败 - {e2}")

    # 尝试用 OCCT 读取
    print(f"\n[5] OCCT STEP 文件读取测试:")
    try:
        from OCC.Core.IFSelect import IFSelect_RetDone
        from OCC.Core.STEPControl import STEPControl_Reader

        reader = STEPControl_Reader()
        status = reader.ReadFile(test_stp)
        print(f"    ReadFile status: {status}")

        if status == IFSelect_RetDone:
            print(f"    ReadFile: 成功")
            reader.TransferRoots()
            shape = reader.OneShape()

            if shape.IsNull():
                print(f"    OneShape: 返回空 Shape!")
            else:
                print(f"    OneShape: 成功获取 Shape")
        else:
            print(f"    ReadFile: 失败 (status={status})")
            # 尝试获取错误信息
            print(f"    尝试获取更多诊断信息...")
            reader.TransferRoots()  # 尝试继续
            shape = reader.OneShape()
            if shape.IsNull():
                print(f"    Shape 为空")
            else:
                print(f"    但 Shape 不为空?")

    except Exception as e:
        print(f"    OCCT 错误: {e}")
        import traceback
        traceback.print_exc()

# 测试 Redis 连接
print(f"\n[6] Redis 连接测试:")
redis_host = os.getenv('REDIS_HOST', 'localhost')
redis_port = int(os.getenv('REDIS_PORT', '6379'))
redis_db = int(os.getenv('REDIS_DB', '0'))

try:
    from redis import Redis
    r = Redis(host=redis_host, port=redis_port, db=redis_db, decode_responses=True)
    r.ping()
    print(f"    Redis 连接: OK")
    # 检查队列
    queue_len = r.llen('stp_render_queue')
    print(f"    stp_render_queue 长度: {queue_len}")
except Exception as e:
    print(f"    Redis 错误: {e}")

# 测试 OSS 连接
print(f"\n[7] OSS 连接测试:")
oss_key = os.getenv('OSS_ACCESS_KEY')
oss_secret = os.getenv('OSS_ACCESS_SECRET')
oss_endpoint = os.getenv('OSS_ENDPOINT')
oss_bucket = os.getenv('OSS_BUCKET')

if all([oss_key, oss_secret, oss_endpoint, oss_bucket]):
    try:
        import oss2
        auth = oss2.Auth(oss_key, oss_secret)
        bucket = oss2.Bucket(auth, oss_endpoint, oss_bucket)
        print(f"    OSS 认证: OK")
        # 尝试列出文件
        try:
            result = bucket.list_objects(max_keys=1)
            print(f"    OSS Bucket 访问: OK (包含 {result.object_list.total_count} 个对象)")
        except Exception as e:
            print(f"    OSS Bucket 访问: 警告 ({e})")
    except Exception as e:
        print(f"    OSS 错误: {e}")
else:
    print(f"    OSS 未配置完整，跳过测试")

print("\n" + "=" * 60)
print("诊断完成")
print("=" * 60)
