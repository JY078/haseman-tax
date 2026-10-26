const clamp = (value, min, max) => Math.min(max, Math.max(min, value))

export function parseImageRatio(ratio, sourceWidth, sourceHeight) {
  if (ratio === 'original') return sourceWidth / sourceHeight

  const [width, height] = String(ratio).split(':').map(Number)
  if (!width || !height) return sourceWidth / sourceHeight
  return width / height
}

export function getExpandLayout(sourceWidth, sourceHeight, ratio, factor, position = { x: 0.5, y: 0.5 }) {
  const safeWidth = Math.max(1, Number(sourceWidth) || 1)
  const safeHeight = Math.max(1, Number(sourceHeight) || 1)
  const safeFactor = Math.max(1, Number(factor) || 1)
  const targetRatio = parseImageRatio(ratio, safeWidth, safeHeight)

  let targetWidth = safeWidth * safeFactor
  let targetHeight = targetWidth / targetRatio
  const minimumHeight = safeHeight * safeFactor

  if (targetHeight < minimumHeight) {
    targetHeight = minimumHeight
    targetWidth = targetHeight * targetRatio
  }

  const extraWidth = Math.max(0, targetWidth - safeWidth)
  const extraHeight = Math.max(0, targetHeight - safeHeight)

  return {
    targetWidth,
    targetHeight,
    imageX: extraWidth * clamp(Number(position.x) || 0, 0, 1),
    imageY: extraHeight * clamp(Number(position.y) || 0, 0, 1),
    imageWidth: safeWidth,
    imageHeight: safeHeight,
  }
}

export function fitExpandLayoutForExport(layout, maxEdge = 2048) {
  const scale = Math.min(1, maxEdge / Math.max(layout.targetWidth, layout.targetHeight))
  return Object.fromEntries(
    Object.entries(layout).map(([key, value]) => [key, Math.max(key.startsWith('image') ? 0 : 1, Math.round(value * scale))])
  )
}
