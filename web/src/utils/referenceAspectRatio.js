export async function resolveReferenceAspectRatio(references, supportedRatios) {
  const first = references[0]
  if (!first) throw new Error('请先添加参考图，再使用原图比例')
  const source = first.data
    ? `data:${first.mimeType || 'image/png'};base64,${first.data}`
    : first.url || first.remoteUrl
  const image = await new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('无法读取第一张参考图的尺寸，请重新添加参考图'))
    image.src = source
  })
  if (!image.naturalWidth || !image.naturalHeight) {
    throw new Error('无法读取第一张参考图的尺寸，请重新添加参考图')
  }
  const originalRatio = image.naturalWidth / image.naturalHeight
  // 使用对数距离，使横图与竖图的匹配规则对称。
  const distance = ratio => {
    const [width, height] = ratio.split(':').map(Number)
    return Math.abs(Math.log(width / height / originalRatio))
  }
  return supportedRatios.reduce((closest, ratio) => (
    distance(ratio) < distance(closest) ? ratio : closest
  ))
}
