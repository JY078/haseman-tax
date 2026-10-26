export function customSizeError(width, height) {
  if (![width, height].every(value => Number.isInteger(Number(value)) && Number(value) >= 64 && Number(value) <= 8192)) {
    return '宽和高请输入 64–8192 之间的整数'
  }
  if (Number(width) * Number(height) > 24000000) return '总像素不能超过 2400 万，请减小宽或高'
  return ''
}

export function closestImageRatio(width, height, ratios, pixelSizes = {}) {
  const target = Number(width) / Number(height)
  const distance = ratio => {
    const [w, h] = pixelSizes[ratio] || ratio.split(':').map(Number)
    return Math.abs(Math.log(w / h / target))
  }
  return ratios.reduce((closest, ratio) => distance(ratio) < distance(closest) ? ratio : closest)
}

// Keep aligned with backend/internal/adapter/gpt_image_size.go.
export function canGenerateNativeGPTSize(model, width, height) {
  const name = String(model).toLowerCase().trim().split('/').pop()
  const w = Number(width), h = Number(height)
  if (!Number.isInteger(w) || !Number.isInteger(h) || w <= 0 || h <= 0) return false
  if (name === 'gpt-image-2' || name.startsWith('gpt-image-2-')) {
    return w <= 3840 && h <= 3840 && w % 16 === 0 && h % 16 === 0 && w <= h * 3 && h <= w * 3 && w * h >= 655360 && w * h <= 8294400
  }
  return ['gpt-image-1', 'gpt-image-1-mini', 'gpt-image-1.5'].includes(name)
    && ((w === 1024 && h === 1024) || (w === 1536 && h === 1024) || (w === 1024 && h === 1536))
}

export function imageSizeTreatment(width, height, sourceWidth, sourceHeight) {
  if (Number(width) === sourceWidth && Number(height) === sourceHeight) return 'native'
  return Number(width) * sourceHeight === Number(height) * sourceWidth ? 'scale' : 'crop'
}

// Prefer the closest actual aspect ratio, then the smallest resolution that avoids upscaling.
export function matchPresetImageSize(width, height, candidates) {
  if (!candidates.length) return null
  const target = Number(width) / Number(height)
  const distance = item => Math.abs(Math.log(item.width / item.height / target))
  const closest = candidates.reduce((best, item) => distance(item) < distance(best) ? item : best)
  const matching = candidates.filter(item => item.ratio === closest.ratio).sort((a, b) => a.width * a.height - b.width * b.height)
  return matching.find(item => item.width >= width && item.height >= height) || matching[matching.length - 1]
}

// Minimize relative changes to both dimensions among legal sizes. Normalize orientation for symmetry.
export function matchGPTImageSize(model, width, height) {
  width = Number(width); height = Number(height)
  if (width <= 0 || height <= 0 || !Number.isFinite(width + height)) return null
  if (canGenerateNativeGPTSize(model, width, height)) return { width, height }
  const name = String(model).toLowerCase().trim().split('/').pop()
  const landscape = width >= height
  const long = Math.max(width, height), short = Math.min(width, height)
  let best = null, bestScore = Infinity
  const consider = (w, h) => {
    if (width === height && w !== h) return
    const score = Math.log(w / long) ** 2 + Math.log(h / short) ** 2
    if (score < bestScore) { bestScore = score; best = landscape ? { width: w, height: h } : { width: h, height: w } }
  }
  if (name === 'gpt-image-2' || name.startsWith('gpt-image-2-')) {
    for (let w = 16; w <= 3840; w += 16) {
      for (let h = Math.ceil(Math.max(w / 3, 655360 / w) / 16) * 16; h <= Math.min(w, 8294400 / w); h += 16) consider(w, h)
    }
  } else if (['gpt-image-1', 'gpt-image-1-mini', 'gpt-image-1.5'].includes(name)) {
    consider(1024, 1024); consider(1536, 1024)
  }
  return best
}
