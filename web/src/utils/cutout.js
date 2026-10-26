export function cutoutError(message) {
  if (/connection refused|connectex|dial tcp|Cannot GET|抠图服务请求失败/.test(message || '')) {
    return '抠图服务暂未就绪，请稍后重试'
  }
  return message || '处理未完成，请重试'
}

export function paintCutout(canvas, image, background) {
  canvas.width = image.naturalWidth || image.width
  canvas.height = image.naturalHeight || image.height
  const context = canvas.getContext('2d')
  context.clearRect(0, 0, canvas.width, canvas.height)
  if (background !== 'transparent') {
    context.fillStyle = background
    context.fillRect(0, 0, canvas.width, canvas.height)
  }
  context.drawImage(image, 0, 0)
}
