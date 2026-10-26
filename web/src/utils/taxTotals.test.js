import test from 'node:test'
import assert from 'node:assert/strict'
import { isExcludedTaxTotalColumn } from './taxTotals.js'

test('all previously excluded columns remain excluded with header formatting variants', () => {
  const columns = [
    'Customer Order #', 'Customer Order line #', 'Purchase Order #', 'Purchase Order line #',
    'Commission Rate', 'Base Commission Rate', 'Partner Item Id', 'Partner GTIN', 'Product Tax Code', 'Ship to Zipcode',
    'Order ID', 'SKU ID', 'Product ID', 'Zipcode', 'Weight(kg)', 'Package ID',
    '亚马逊订单商品编号', '卖家订单编号', 'Seller Order ID', '配送邮政编码', '追踪编码', 'settlement id', 'order postal',
    'Weight (kg)', '重量（千克）', '合作方商品编号', '\uFEFFsettlement\u00a0id ', 'ＳＫＵ ＩＤ',
  ]
  for (const column of columns) {
    assert.equal(isExcludedTaxTotalColumn(column), true, column)
    assert.equal(isExcludedTaxTotalColumn(` ${column.toUpperCase()} `), true, column)
  }
})

test('amounts and quantities are still additive', () => {
  for (const column of ['Amount', 'quantity', '已发货数量', 'Ship Qty', 'Order Amount', 'fba fees', 'total']) {
    assert.equal(isExcludedTaxTotalColumn(column), false, column)
  }
})

