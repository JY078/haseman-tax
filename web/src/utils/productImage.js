import request from './request'

export async function uploadProductImage(file) {
  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) throw new Error('请使用 JPG、PNG、WebP 或 GIF 图片')
  if (!file.size || file.size > 30 * 1024 * 1024) throw new Error('图片大小须在 30 MB 以内')
  const data = new FormData()
  data.append('image', file, file.name || 'product.png')
  const response = await request.post('/upload/image', data)
  if (!response.data?.url) throw new Error('上传未返回图片地址')
  return response.data.url
}
