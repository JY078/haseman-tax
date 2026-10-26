import test from 'node:test'
import assert from 'node:assert/strict'

import { IMAGE_EDIT_BRUSH_DEFAULTS, drawSmoothBrushStroke } from './imageEditBrush.js'

test('image editors share the expected brush defaults', () => {
  assert.deepEqual(IMAGE_EDIT_BRUSH_DEFAULTS, {
    color: '#ff3b30',
    opacity: 0.7,
    size: 20,
    maxSize: 80,
  })
})

test('smooth brush stroke uses quadratic midpoints for multi-point input', () => {
  const calls = []
  const ctx = {
    beginPath: () => calls.push(['beginPath']),
    moveTo: (...args) => calls.push(['moveTo', ...args]),
    lineTo: (...args) => calls.push(['lineTo', ...args]),
    quadraticCurveTo: (...args) => calls.push(['quadraticCurveTo', ...args]),
    stroke: () => calls.push(['stroke']),
  }

  drawSmoothBrushStroke(ctx, [
    { x: 0, y: 0 },
    { x: 10, y: 4 },
    { x: 20, y: 8 },
  ], IMAGE_EDIT_BRUSH_DEFAULTS.size)

  assert.deepEqual(calls, [
    ['beginPath'],
    ['moveTo', 0, 0],
    ['quadraticCurveTo', 10, 4, 15, 6],
    ['lineTo', 20, 8],
    ['stroke'],
  ])
})
