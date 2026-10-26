import test from 'node:test'
import assert from 'node:assert/strict'
import { paintCutout, cutoutError } from './cutout.js'

test('transparent export skips fill, colored export fills before drawing at original dimensions', () => {
  const events = []
  const context = { clearRect: () => events.push('clear'), fillRect: () => events.push('fill'), drawImage: () => events.push('image') }
  const canvas = { getContext: () => context }
  const image = { naturalWidth: 3200, naturalHeight: 1600 }
  paintCutout(canvas, image, '#f0dedb')
  assert.equal(canvas.width, 3200)
  assert.equal(canvas.height, 1600)
  assert.equal(context.fillStyle, '#f0dedb')
  assert.deepEqual(events, ['clear', 'fill', 'image'])
  events.length = 0
  paintCutout(canvas, image, 'transparent')
  assert.deepEqual(events, ['clear', 'image'])
})

test('old connection failures are readable, actionable image errors are retained', () => {
  assert.equal(cutoutError('dial tcp 127.0.0.1:8001: connect: connection refused'), '抠图服务暂未就绪，请稍后重试')
  assert.equal(cutoutError('图片数据无效'), '图片数据无效')
})
