import test from 'node:test'
import assert from 'node:assert/strict'
import { amazonFinanceHeaderLabel } from './amazonFinanceHeaders.js'

test('Amazon finance headers show the English source and Chinese translation', () => {
  assert.equal(amazonFinanceHeaderLabel('product sales'), 'product sales（商品销售额）')
  assert.equal(amazonFinanceHeaderLabel('商品销售额'), 'product sales（商品销售额）')
})

test('unknown Amazon finance headers remain unchanged', () => {
  assert.equal(amazonFinanceHeaderLabel('New Amazon Field'), 'New Amazon Field')
})
