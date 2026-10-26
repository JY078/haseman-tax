import oss2
import io
import uuid
from PIL import Image
from config import OSS_ACCESS_KEY, OSS_ACCESS_SECRET, OSS_ENDPOINT, OSS_BUCKET, OSS_DOMAIN

class OSSClient:
    def __init__(self):
        self.auth = oss2.Auth(OSS_ACCESS_KEY, OSS_ACCESS_SECRET)
        self.bucket = oss2.Bucket(self.auth, OSS_ENDPOINT, OSS_BUCKET)

    def upload_image(self, image: Image.Image, object_key: str) -> str:
        """上传 PIL Image 到 OSS"""
        buffer = io.BytesIO()
        image.save(buffer, format='PNG')
        buffer.seek(0)

        # 使用 put_object_async_result 等待上传完成
        result = self.bucket.put_object(object_key, buffer)
        if result.status != 200:
            raise RuntimeError(f"OSS upload failed: status={result.status}, resppnse_id={result.response_id}")

        if OSS_DOMAIN:
            return f"{OSS_DOMAIN}/{object_key}"
        return f"https://{OSS_BUCKET}.{OSS_ENDPOINT}/{object_key}"

    def upload_image_unique(self, image: Image.Image, prefix: str, view_name: str) -> str:
        """上传图片，使用 UUID 确保每个视角有唯一路径"""
        uid = uuid.uuid4().hex[:12]
        object_key = f"{prefix}/{uid}_{view_name}.png"
        return self.upload_image(image, object_key)

    def download_file(self, object_key: str, local_path: str) -> str:
        """
        下载 OSS 文件到本地

        Args:
            object_key: OSS 对象路径 (如 'products/stp/file.stp')
            local_path: 本地保存路径
        """
        self.bucket.get_object_to_file(object_key, local_path)
        return local_path

# 全局单例
oss_client = OSSClient() if OSS_ACCESS_KEY and OSS_ACCESS_SECRET else None