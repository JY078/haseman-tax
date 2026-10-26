#!/usr/bin/env python3
"""诊断 OCCT BRepMesh_IncrementalMesh 构造函数签名"""
import os, sys

# 不依赖 config.py，手动设置环境变量
os.environ.setdefault('OSS_ACCESS_KEY', os.getenv('OSS_ACCESS_KEY', ''))
os.environ.setdefault('OSS_ACCESS_SECRET', os.getenv('OSS_ACCESS_SECRET', ''))
os.environ.setdefault('OSS_ENDPOINT', os.getenv('OSS_ENDPOINT', ''))
os.environ.setdefault('OSS_BUCKET', os.getenv('OSS_BUCKET', ''))

print("=" * 60)
print("BRepMesh_IncrementalMesh 构造函数诊断")
print("=" * 60)
print(f"Python: {sys.version}")
print(f"Python path: {sys.executable}")

from OCC.Core.BRepMesh import BRepMesh_IncrementalMesh

# 检查构造函数签名
print("\n[1] 构造函数签名:")
try:
    import inspect
    sig = inspect.signature(BRepMesh_IncrementalMesh.__init__)
    print(f"   __init__ {sig}")
except Exception as e:
    print(f"   Cannot get signature: {e}")

# 尝试不同的参数组合
print("\n[2] 测试不同参数组合:")

# 首先需要加载一个 shape 来测试
from OCC.Core.STEPControl import STEPControl_Reader

# 尝试列出可用的 STP 测试文件
oss_key = os.getenv('OSS_ACCESS_KEY')
oss_secret = os.getenv('OSS_ACCESS_SECRET')
oss_endpoint = os.getenv('OSS_ENDPOINT')
oss_bucket = os.getenv('OSS_BUCKET')

if all([oss_key, oss_secret, oss_endpoint, oss_bucket]):
    try:
        import oss2
        auth = oss2.Auth(oss_key, oss_secret)
        bucket = oss2.Bucket(auth, oss_endpoint, oss_bucket)
        stp_files = list(bucket.list_objects(prefix='products/stp/', max_keys=5).object_list)

        print(f"\n   可用的 STP 文件:")
        for obj in stp_files:
            print(f"   - {obj.key} ({obj.size/1024:.1f} KB)")

        if stp_files:
            test_file = '/tmp/test_step_diag.stp'
            bucket.get_object_to_file(stp_files[0].key, test_file)
            print(f"\n   下载了 {stp_files[0].key} 到 {test_file}")

            reader = STEPControl_Reader()
            status = reader.ReadFile(test_file)
            print(f"   ReadFile status: {status}")

            if status == 1:  # IFSelect_RetDone
                reader.TransferRoots()
                shape = reader.OneShape()
                print(f"   Shape null: {shape.IsNull()}")

                if not shape.IsNull():
                    # 测试不同的 mesh 参数
                    test_params = [
                        # (deflection, is_relative, algo, minSize)
                        (0.5, False, None, 0.0),
                        (1.0, False, None, 0.0),
                        (0.1, False, None, 0.0),
                        (0.01, False, None, 0.0),
                    ]

                    for deflection, is_rel, algo, min_size in test_params:
                        print(f"\n   测试: deflection={deflection}, relative={is_rel}, algo={algo}, minSize={min_size}")
                        try:
                            if algo is None:
                                # 只传 2 个参数
                                mesh = BRepMesh_IncrementalMesh(shape, deflection)
                            elif min_size > 0:
                                # 传 4 个参数
                                mesh = BRepMesh_IncrementalMesh(shape, deflection, is_rel, algo, min_size)
                            else:
                                # 传 3 个参数
                                mesh = BRepMesh_IncrementalMesh(shape, deflection, is_rel)
                            mesh.Perform()
                            print(f"   结果: 成功")
                        except Exception as e:
                            print(f"   结果: 失败 - {e}")
            else:
                print(f"   ReadFile failed (status={status}), 无法测试 mesh")
        else:
            print("\n   没有找到 STP 文件")
    except Exception as e:
        print(f"   OSS 错误: {e}")
else:
    print("\n   OSS 未配置，跳过 STP 下载测试")
    # 尝试本地文件
    local_tests = [
        '/tmp/test_model.stp',
        '/www/wwwroot/advertising-image/uploads/stp/test.stp',
    ]
    for path in local_tests:
        if os.path.exists(path):
            print(f"   找到本地测试文件: {path}")
            break

print("\n" + "=" * 60)