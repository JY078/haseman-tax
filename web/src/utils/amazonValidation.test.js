import test from 'node:test'
import assert from 'node:assert/strict'
import { parseAmazonValidationMessage, parseAmazonValidationMessages } from './amazonValidation.js'

test('parses a full validation message with period, values and currency', () => {
  const issue = parseAmazonValidationMessage('2026-01-01 至 2026-03-31 税费校验不一致：明细 123.45，PDF 122.00，差额 1.45 USD')
  assert.deepEqual(issue, { range: '2026-01-01 ~ 2026-03-31', metric: '税费', detail: '123.45', pdf: '122.00', difference: '1.45', currency: 'USD' })
})

test('parses delivery and storage metrics with negative differences', () => {
  const delivery = parseAmazonValidationMessage('2025-07-01 至 2025-09-30 配送费校验不一致：明细 -12.00，PDF -10.00，差额 -2.00 CAD')
  assert.equal(delivery.metric, '配送费')
  assert.equal(delivery.difference, '-2.00')
  assert.equal(delivery.currency, 'CAD')
  const storage = parseAmazonValidationMessage('仓储费校验不一致：明细 5.00，PDF 4.00，差额 1.00')
  assert.equal(storage.metric, '仓储费')
  assert.equal(storage.range, '')
  assert.equal(storage.currency, '')
})

test('returns null for unrelated messages and keeps the raw text', () => {
  assert.equal(parseAmazonValidationMessage('数据加载失败'), null)
  const parsed = parseAmazonValidationMessages(['数据加载失败', '2026-01-01 至 2026-03-31 税费校验不一致：明细 1.00，PDF 2.00，差额 -1.00 USD'])
  assert.equal(parsed[0].issue, null)
  assert.equal(parsed[0].message, '数据加载失败')
  assert.equal(parsed[1].issue.metric, '税费')
})
