export const videoModelOptions = [
  { label: 'Seedance 2.5', value: 'seedance-2-5', description: '长镜头生成，支持 4–30 秒' },
]

export const videoResolutionOptions = ['480p', '720p', '1080p']
export const videoDurationOptions = [5, 10, 15]

export function videoDurationRangeForModel(model) {
  return model === 'seedance-2-5' ? { min: 4, max: 30 } : { min: 4, max: 15 }
}
export const videoAspectRatioOptions = [
  { label: '21:9 横向电影', value: '21:9' },
  { label: '16:9 宽屏', value: '16:9' },
  { label: '4:3 经典', value: '4:3' },
  { label: '1:1 方形', value: '1:1' },
  { label: '3:4 竖图', value: '3:4' },
  { label: '9:16 竖屏', value: '9:16' },
]

export function videoResolutionsForModel(model) {
  return videoResolutionOptions
}

export function isVideoTaskCompleteStatus(status) {
  return ['COMPLETED', 'COMPLETE', 'SUCCESS', 'SUCCEEDED', 'FINISHED', 'DONE'].includes(String(status || '').toUpperCase())
}

export function isVideoTaskFailedStatus(status) {
  return ['FAILED', 'FAIL', 'ERROR', 'CANCELED', 'CANCELLED', 'EXPIRED'].includes(String(status || '').toUpperCase())
}

export function extractVideoResultUrl(data = {}) {
  return data.video_url || data.videoUrl || data.content?.video_url || data.content?.videoUrl ||
    data.output?.video_url || data.output?.videoUrl || data.data?.video_url || data.data?.videoUrl ||
    data.result?.video_url || data.result?.videoUrl || data.output?.url || data.data?.url ||
    data.result?.url || data.url || data.source_url || data.sourceUrl || ''
}

export function imageFileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('读取图片失败'))
    reader.readAsDataURL(file)
  })
}
