import test from 'node:test'
import assert from 'node:assert/strict'
import ExcelJS from 'exceljs'
import { fileURLToPath } from 'node:url'

import { readProductCatalogFile } from './productCatalogImport.js'

async function workbookFile(rows) {
  const workbook = new ExcelJS.Workbook()
  const sheet = workbook.addWorksheet('产品导入')
  rows.forEach(row => sheet.addRow(row))
  const buffer = await workbook.xlsx.writeBuffer()
  return { arrayBuffer: async () => buffer }
}

test('产品库导入读取必填列和选填列', async () => {
  const file = await workbookFile([
    ['SPU', 'SKU', 'ASIN', '品牌'],
    ['SPU-1', 'SKU-1', 'B012345678', 'DEAROCEAN'],
  ])
  const rows = await readProductCatalogFile(file)
  assert.deepEqual(rows, [{
    row_number: 2, spu: 'SPU-1', sku: 'SKU-1', asin: 'B012345678', spu_name: '', brand: 'DEAROCEAN', model: '',
    spu_material: '', spu_category1: '', spu_category2: '', sku_name: '', style_name: '', color: '', size: '', feature: '',
    package_qty: '', material: '', category1: '', category2: '', category3: '', listed_at: '', package_length: '',
    package_width: '', package_height: '', package_weight: '', purchase_price: '',
  }])
})

test('产品库导入拒绝缺少 ASIN 列的文件', async () => {
  const file = await workbookFile([['SPU', 'SKU'], ['SPU-1', 'SKU-1']])
  await assert.rejects(() => readProductCatalogFile(file), /ASIN/)
})

test('产品库导入允许表头上方存在说明行', async () => {
  const file = await workbookFile([
    ['填写说明'],
    ['SPU', 'SKU', 'ASIN'],
    ['SPU-2', 'SKU-2', 'B087654321'],
  ])
  const rows = await readProductCatalogFile(file)
  assert.equal(rows[0].row_number, 3)
  assert.equal(rows[0].asin, 'B087654321')
})

test('图片链接和超链接导入', async () => {
 const file = await workbookFile([['SPU','SKU','ASIN','产品图片'],['S','K','B012345678',{text:'查看图片',hyperlink:'https://example.com/product.png'}]])
 const rows = await readProductCatalogFile(file)
 assert.equal(rows[0].image_url,'https://example.com/product.png')
})

test('内嵌图片按图片列所在行匹配 SKU', async () => {
 const workbook = new ExcelJS.Workbook()
 const sheet = workbook.addWorksheet('产品')
 sheet.addRow(['SPU','SKU','ASIN','产品图片'])
 sheet.addRow(['S','K1','B012345678'])
 sheet.addRow(['S','K2','B087654321'])
 const id = workbook.addImage({buffer:Buffer.from('sample-image'),extension:'png'})
 sheet.addImage(id,{tl:{col:3,row:2},ext:{width:48,height:48}})
 const buffer = await workbook.xlsx.writeBuffer()
 const uploaded = []
 const rows = await readProductCatalogFile({arrayBuffer:async()=>buffer},{uploadImage:async blob=>{uploaded.push(await blob.text());return '/uploads/photo.png'}})
 assert.deepEqual(uploaded,['sample-image'])
 assert.equal(rows[0].image_url,'')
 assert.equal(rows[1].image_url,'/uploads/photo.png')
})

test('无效数据应在上传图片前拒绝', async () => {
 const file = await workbookFile([['SPU','SKU','ASIN','产品图片'],['S','K','','https://example.com/a.png']])
 await assert.rejects(()=>readProductCatalogFile(file),/必填/)
})

test('拒绝图片列的无效链接', async () => {
 const file = await workbookFile([['SPU','SKU','ASIN','产品图片'],['S','K','B012345678','javascript:alert(1)']])
 await assert.rejects(()=>readProductCatalogFile(file),/HTTP/)
})

test('下载模板可被导入器读取', async () => {
 const workbook = new ExcelJS.Workbook()
 await workbook.xlsx.readFile(fileURLToPath(new URL('../../public/产品库导入模板.xlsx', import.meta.url)))
 assert.equal(workbook.worksheets[0].getCell('Z2').text,'产品图片')
})
