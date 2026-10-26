import test from 'node:test'
import assert from 'node:assert/strict'
import { customSizeError, closestImageRatio, canGenerateNativeGPTSize, imageSizeTreatment, matchGPTImageSize, matchPresetImageSize } from './customImageSize.js'

test('custom sizes validate both dimensions and the pixel budget', () => {
  assert.equal(customSizeError(2928, 1200), '')
  for (const [w, h] of [['', 1200], [0, 1200], [64.5, 1200], [8193, 64], [8192, 8192], [1200, Infinity]]) {
    assert.ok(customSizeError(w, h))
  }
  assert.equal(customSizeError(8192, 64), '')
})

test('matches supported ratios symmetrically, including a restricted model', () => {
  assert.equal(closestImageRatio(2928, 1200, ['1:1', '16:9', '21:9']), '21:9')
  assert.equal(closestImageRatio(1200, 2928, ['1:1', '9:16', '9:21']), '9:21')
  assert.equal(closestImageRatio(2928, 1200, ['1:1', '16:9']), '16:9')
})

test('GPT native support is model-specific and enforces every size constraint', () => {
  assert.ok(canGenerateNativeGPTSize('openai/gpt-image-2', 2928, 1200))
  assert.ok(canGenerateNativeGPTSize('gpt-image-2-2026-04-21', 1200, 2928))
  assert.ok(canGenerateNativeGPTSize('gpt-image-2', 3840, 2160))
  for (const [w, h] of [[2929, 1200], [4096, 4096], [64, 64], [3072, 3072], [2048, 512]]) {
    assert.equal(canGenerateNativeGPTSize('gpt-image-2', w, h), false)
  }
  assert.equal(canGenerateNativeGPTSize('gpt-image-1', 2928, 1200), false)
  assert.ok(canGenerateNativeGPTSize('gpt-image-1.5', 1536, 1024))
  assert.equal(canGenerateNativeGPTSize('gemini-3-pro-image', 1024, 1024), false)
})

test('uses actual pixel sizes and distinguishes native output, scaling, and cropping', () => {
  assert.equal(closestImageRatio(2928, 1200, ['21:9', '5:2'], { '21:9': [2928, 1200] }), '21:9')
  assert.equal(imageSizeTreatment(3168, 1344, 3168, 1344), 'native')
  assert.equal(imageSizeTreatment(1584, 672, 3168, 1344), 'scale')
  assert.equal(imageSizeTreatment(2928, 1200, 3168, 1344), 'crop')
})

test('Gemini jointly matches ratio and the lowest resolution that covers the target', () => {
  const candidates = [
    { ratio: '1:1', resolution: '1K', width: 1024, height: 1024 },
    { ratio: '21:9', resolution: '1K', width: 1584, height: 672 },
    { ratio: '21:9', resolution: '2K', width: 3168, height: 1344 },
    { ratio: '21:9', resolution: '4K', width: 6336, height: 2688 },
  ]
  assert.deepEqual(matchPresetImageSize(2928, 1200, candidates), candidates[2])
  assert.deepEqual(matchPresetImageSize(1400, 600, candidates), candidates[1])
  assert.deepEqual(matchPresetImageSize(4000, 1800, candidates), candidates[3])
  assert.deepEqual(matchPresetImageSize(8000, 3000, candidates), candidates[3])
  assert.deepEqual(matchPresetImageSize(3168, 1344, candidates), candidates[2])
  assert.deepEqual(matchPresetImageSize(4000, 1800, candidates.slice(0, 3)), candidates[2])
  assert.equal(matchPresetImageSize(2928, 1200, []), null)
})

test('GPT fallback always satisfies constraints and preserves orientation', () => {
  for (const [width, height] of [[2928, 1200], [2929, 1200], [4000, 1000], [8192, 64], [64, 64], [4096, 4096], [300, 200], [5000, 4000]]) {
    const size = matchGPTImageSize('gpt-image-2', width, height)
    assert.ok(canGenerateNativeGPTSize('gpt-image-2', size.width, size.height), JSON.stringify(size))
    const portrait = matchGPTImageSize('gpt-image-2', height, width)
    assert.deepEqual(portrait, { width: size.height, height: size.width })
  }
  assert.deepEqual(matchGPTImageSize('gpt-image-2', 2928, 1200), { width: 2928, height: 1200 })
  assert.deepEqual(matchGPTImageSize('gpt-image-2', 2929, 1200), { width: 2928, height: 1200 })
  assert.deepEqual(matchGPTImageSize('gpt-image-1', 2928, 1200), { width: 1536, height: 1024 })
  assert.equal(matchGPTImageSize('gemini-3-pro-image', 2928, 1200), null)
})
