import test from 'node:test'
import assert from 'node:assert/strict'
import ExcelJS from 'exceljs'
import { buildCustomsReport } from './customsReport.js'

test('贸易方式导出与页面一致，代码显示在名称后', async () => {
  const workbook = await buildCustomsReport([{ declaration_number: '310120260000665664', fields: { trade_mode: '(9810) 跨境电商出口海外仓' }, items: [], warnings: [] }], { fields: [['trade_mode', '贸易方式']], items: [] })
  assert.equal(workbook.getWorksheet('报关单明细').getCell('B2').value, '跨境电商出口海外仓(9810)')
})

test('运保费导出为数值且不改写原始费用', async () => {
  const fields = { freight: 'USD/3908.95/总价', insurance: 'USD/101.89/总价' }
  const workbook = await buildCustomsReport([{ declaration_number: '310120260000665664', fields, items: [], warnings: [] }], { fields: [['freight', '运费'], ['insurance', '保费']], items: [] })
  const saved = new ExcelJS.Workbook()
  await saved.xlsx.load(await workbook.xlsx.writeBuffer())
  assert.equal(saved.getWorksheet('报关单明细').getCell('B2').value, 3908.95)
  assert.equal(saved.getWorksheet('报关单明细').getCell('C2').value, 101.89)
  assert.equal(fields.freight, 'USD/3908.95/总价')
  assert.equal(fields.insurance, 'USD/101.89/总价')
})

test('报关单报表保留长编号和金额精度，多商品单据合并且文本不会执行为公式', async () => {
  const records = [{ declaration_number: '031012026000066566', fields: { packages: '205' }, items: [{ name: '=1+1', total_price: '37629.41', unit_price: '22.8057' }, { name: '配件', total_price: '1234567890123456.78' }], warnings: [] }]
  const workbook = await buildCustomsReport(records, { fields: [['packages', '件数']], items: [['name', '商品名称'], ['total_price', '总价'], ['unit_price', '单价']], filters: { date_from: '2026-09-01', destination_port: '加拿大' } })
  const saved = new ExcelJS.Workbook()
  await saved.xlsx.load(await workbook.xlsx.writeBuffer())
  const sheet = saved.getWorksheet('报关单明细')
  assert.equal(sheet.rowCount, 3)
  assert.equal(sheet.getCell('A2').value, '031012026000066566')
  assert.equal(sheet.getCell('A3').master.address, 'A2')
  assert.equal(sheet.getCell('B3').master.address, 'B2')
  assert.equal(sheet.getCell('B2').value, 205)
  assert.equal(sheet.getCell('C2').value, '=1+1')
  assert.equal(sheet.getCell('C2').formula, undefined)
  assert.equal(sheet.getCell('D2').value, 37629.41)
  assert.equal(sheet.getCell('E2').value, 22.8057)
  assert.equal(sheet.getCell('D3').value, '1234567890123456.78')
  assert.equal(saved.getWorksheet('导出说明').getCell('B4').value, '加拿大')
})

test('导出全部匹配记录和全部字段，空商品单据仍保留', async () => {
  const records = Array.from({ length: 121 }, (_, id) => ({ declaration_number: String(id), fields: {}, items: [], warnings: ['需核对'], file_name: 'sample.pdf', recognition_method: 'ocr' }))
  const workbook = await buildCustomsReport(records, { fields: [], items: [['name', '商品名称']], allFields: true })
  const sheet = workbook.getWorksheet('报关单明细')
  assert.equal(sheet.rowCount, 122)
  assert.equal(sheet.getCell('A122').value, '120')
  assert.equal(sheet.getCell('C2').value, '需核对')
  assert.equal(sheet.getCell('D2').value, '扫描识别')
  assert.equal(sheet.getCell('F2').value, 'sample.pdf')
  assert.equal(workbook.getWorksheet('导出说明').getCell('B5').value, 121)
})
