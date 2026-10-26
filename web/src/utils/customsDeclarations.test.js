import test from 'node:test'
import assert from 'node:assert/strict'
import { financeFields, financeItems, customsDate, customsMeasure, customsFeeAmount, customsTradeMode } from './customsDeclarations.js'

test('贸易方式代码统一置于名称后且保留前导零', () => {
  assert.equal(customsTradeMode('(9810) 跨境电商出口海外仓'), '跨境电商出口海外仓(9810)')
  assert.equal(customsTradeMode('跨境电商出口海外仓（9810）'), '跨境电商出口海外仓(9810)')
  assert.equal(customsTradeMode('跨境电商出口海外仓(9810)'), '跨境电商出口海外仓(9810)')
  assert.equal(customsTradeMode('（0110）一般贸易'), '一般贸易(0110)')
  assert.equal(customsTradeMode('一般贸易'), '一般贸易')
  assert.equal(customsTradeMode(null), '')
})

test('运保费只展示原始数值，兼容零值和全角分隔符', () => {
  assert.equal(customsFeeAmount('USD/3908.95/总价'), '3908.95')
  assert.equal(customsFeeAmount('USD／101.89／总价'), '101.89')
  assert.equal(customsFeeAmount(' USD / 3,908.950 / 总价 '), '3908.950')
  assert.equal(customsFeeAmount('/0/'), '0')
  assert.equal(customsFeeAmount(0), '0')
  assert.equal(customsFeeAmount('101.89'), '101.89')
  for (const value of ['', null, 'USD//总价', 'USD/invalid/总价']) assert.equal(customsFeeAmount(value), null)
})

test('财务视图完整覆盖红框字段', () => {
  assert.deepEqual(financeFields.map(([key]) => key), ['contract_number', 'export_date', 'declaration_date', 'bill_number', 'trade_mode', 'destination_port', 'freight', 'insurance', 'packages', 'gross_weight', 'net_weight'])
  assert.deepEqual(financeItems.map(([key]) => key), ['name', 'quantity', 'declared_quantity', 'unit_price', 'total_price', 'currency'])
})
test('日期兼容报关单原始格式和已归一化格式', () => {
  assert.equal(customsDate('20260919'), '2026-09-19')
  assert.equal(customsDate('2026-09-18'), '2026-09-18')
  assert.equal(customsDate(''), '—')
})
test('财务汇总区分缺失数据和真实零值', () => {
  assert.equal(customsMeasure({ count: 0, value: '0' }), '—')
  assert.equal(customsMeasure({ count: 1, value: '0' }), '0')
  assert.equal(customsMeasure({ count: 2, value: '39635.83' }), '39,635.83')
  assert.equal(customsMeasure({ count: 1, value: 'invalid' }), '—')
})
