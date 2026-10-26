import test from 'node:test'
import assert from 'node:assert/strict'
import { extractVideoResultUrl, isVideoTaskCompleteStatus, isVideoTaskFailedStatus, videoResolutionsForModel } from './videoGeneration.js'

test('视频模型统一提供完整清晰度选项', () => {
  assert.deepEqual(videoResolutionsForModel('seedance-2-5'), ['480p', '720p', '1080p'])
})

test('解析 Ark 视频任务状态和结果', () => {
  assert.equal(extractVideoResultUrl({ content: { video_url: 'https://example.com/video.mp4' } }), 'https://example.com/video.mp4')
  assert.equal(isVideoTaskCompleteStatus('succeeded'), true)
  assert.equal(isVideoTaskFailedStatus('expired'), true)
})
