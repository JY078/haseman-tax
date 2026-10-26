import test from 'node:test'
import assert from 'node:assert/strict'
import { createReviewProductImageLoader, matchReviewProductImage, reviewDateKey } from './reviewAnalysis.js'

test('normalizes imported dates without shifting their calendar day', () => {
  for (const value of ['2026-1-9', '2026/01/09', '2026年1月9日', '2026-01-09T23:59:00-08:00', '2026-01-09 23:59:59']) {
    assert.equal(reviewDateKey(value), '2026-01-09')
  }
  assert.equal(reviewDateKey('2024-02-29'), '2024-02-29')
})

test('rejects invalid or missing dates', () => {
  for (const value of ['', null, undefined, 'not a date', '2026-02-29', '2026-04-31', '2026-13-01', '2026-01-00']) {
    assert.equal(reviewDateKey(value), '')
  }
})

test('catalog image matching uses ASIN only and ignores empty duplicate images', () => {
  const products = [
    { sku: 'FF01124BBN', asin: 'B0BVHM9Q4D', image_url: '/uploads/product.png' },
    { sku: 'OTHER', asin: 'B0BVHM9Q4D', image_url: '' },
    { sku: 'B0BVHM9Q4D-extra', asin: 'OTHER', image_url: '/uploads/wrong.png' },
  ]
  assert.equal(matchReviewProductImage(' b0bvhm9q4d ', products), '/uploads/product.png')
  assert.equal(matchReviewProductImage('ff01124bbn', products), '')
  assert.equal(matchReviewProductImage('B0BVHM9', products), '')
  assert.equal(matchReviewProductImage('', products), '')
  assert.equal(matchReviewProductImage('B0BVHM9Q4D', [...products, { sku: 'B0BVHM9Q4D', image_url: '/uploads/sku.png' }]), '/uploads/product.png')
  assert.equal(matchReviewProductImage('B0BVHM9Q4D', [...products, { sku: 'B0BVHM9Q4D', image_url: '' }]), '/uploads/product.png')
  assert.equal(matchReviewProductImage('B0BVHM9Q4D', [...products, { asin: 'B0BVHM9Q4D', image_url: '/uploads/conflict.png' }]), '')
})

test('catalog fallback paginates and caches repeated keys', async () => {
  const calls = []
  const load = createReviewProductImageLoader({ get: async (url, { params }) => {
    calls.push({ url, ...params })
    return { data: { total: 101, items: [{ skus: params.page === 1 ? [{ sku: 'OTHER', image_url: '/wrong.png' }] : [{ asin: 'B0BVHM9Q4D', image_url: '/right.png' }] }] } }
  } })
  assert.deepEqual(await Promise.all([load('b0bvhm9q4d'), load(' B0BVHM9Q4D ')]), ['/right.png', '/right.png'])
  assert.equal(calls.length, 2)
  assert.deepEqual(calls.map(call => call.page), [1, 2])
  assert(calls.every(call => call.url === '/product-catalog/spus' && call.keyword === 'B0BVHM9Q4D'))
})

test('failed catalog requests can retry and empty keys never request', async () => {
  let calls = 0
  const load = createReviewProductImageLoader({ get: async () => {
    if (++calls === 1) throw new Error('offline')
    return { data: { items: [], total: 0 } }
  } })
  assert.equal(await load(''), '')
  assert.equal(calls, 0)
  await assert.rejects(load('SKU'), /offline/)
  assert.equal(await load('SKU'), '')
  assert.equal(calls, 2)
})
