import test from 'node:test'
import assert from 'node:assert/strict'
import { fitExpandLayoutForExport, getExpandLayout, parseImageRatio } from './imageExpand.js'

test('parseImageRatio supports original and explicit ratios', () => {
  assert.equal(parseImageRatio('original', 1200, 800), 1.5)
  assert.equal(parseImageRatio('16:9', 1200, 800), 16 / 9)
  assert.equal(parseImageRatio('21:9', 1200, 800), 21 / 9)
})

test('getExpandLayout keeps the source inside the target canvas', () => {
  const layout = getExpandLayout(1200, 800, '16:9', 1.25, { x: 1, y: 0 })

  assert.equal(Math.round(layout.targetWidth), 1778)
  assert.equal(layout.targetHeight, 1000)
  assert.equal(Math.round(layout.imageX), 578)
  assert.equal(layout.imageY, 0)
})

test('fitExpandLayoutForExport caps the long edge and preserves placement', () => {
  const fitted = fitExpandLayoutForExport({
    targetWidth: 6000,
    targetHeight: 3000,
    imageX: 1500,
    imageY: 750,
    imageWidth: 3000,
    imageHeight: 1500,
  })

  assert.deepEqual(fitted, {
    targetWidth: 2048,
    targetHeight: 1024,
    imageX: 512,
    imageY: 256,
    imageWidth: 1024,
    imageHeight: 512,
  })
})
