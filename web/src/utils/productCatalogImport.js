import ExcelJS from 'exceljs'

const importColumns = [
  ['spu', 'SPU'], ['sku', 'SKU'], ['asin', 'ASIN'], ['spu_name', 'SPU名称'], ['brand', '品牌'], ['model', '型号'],
  ['spu_material', 'SPU材质'], ['spu_category1', 'SPU一级分类'], ['spu_category2', 'SPU二级分类'], ['sku_name', 'SKU名称'],
  ['style_name', '款式名'], ['color', '颜色'], ['size', '尺寸'], ['feature', '功能'], ['package_qty', '包装数量'],
  ['material', 'SKU材质'], ['category1', 'SKU一级分类'], ['category2', 'SKU二级分类'], ['category3', 'SKU三级分类'],
  ['listed_at', '上架日期'], ['package_length', '包装长'], ['package_width', '包装宽'], ['package_height', '包装高'],
  ['package_weight', '包装重量'], ['purchase_price', '采购价'],
]

function cellText(value) {
  if (value == null) return ''
  if (value instanceof Date) {
    const year = value.getFullYear()
    const month = String(value.getMonth() + 1).padStart(2, '0')
    const day = String(value.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }
  if (typeof value === 'object') {
    if ('text' in value) return String(value.text).trim()
    if ('result' in value) return cellText(value.result)
    if (Array.isArray(value.richText)) return value.richText.map(item => item.text || '').join('').trim()
  }
  return String(value).trim()
}

export async function readProductCatalogFile(file, { uploadImage } = {}) {
  const workbook = new ExcelJS.Workbook()
  await workbook.xlsx.load(await file.arrayBuffer())
  const worksheet = workbook.worksheets[0]
  if (!worksheet) throw new Error('导入文件中没有工作表')

  let headerRowNumber = 0
  let headerIndexes = new Map()
  for (let rowNumber = 1; rowNumber <= Math.min(10, worksheet.rowCount); rowNumber += 1) {
    const indexes = new Map()
    worksheet.getRow(rowNumber).eachCell((cell, columnNumber) => indexes.set(cellText(cell.value), columnNumber))
    if (['SPU', 'SKU', 'ASIN'].every(header => indexes.has(header))) {
      headerRowNumber = rowNumber
      headerIndexes = indexes
      break
    }
  }
  const missing = importColumns.slice(0, 3).filter(([, header]) => !headerIndexes.has(header)).map(([, header]) => header)
  if (missing.length) throw new Error(`导入文件缺少必填列：${missing.join('、')}`)

  const rows = []
  for (let rowNumber = headerRowNumber + 1; rowNumber <= worksheet.rowCount; rowNumber += 1) {
    const source = worksheet.getRow(rowNumber)
    const row = { row_number: rowNumber }
    importColumns.forEach(([key, header]) => {
      const columnNumber = headerIndexes.get(header)
      row[key] = columnNumber ? cellText(source.getCell(columnNumber).value) : ''
    })
    const imageColumn = headerIndexes.get('产品图片') || headerIndexes.get('图片链接') || headerIndexes.get('图片')
    if (imageColumn) {
      const value = source.getCell(imageColumn).value
      row.image_url = typeof value === 'object' && value?.hyperlink ? value.hyperlink : cellText(value)
      if (row.image_url && !/^https?:\/\//i.test(row.image_url)) throw new Error(`第 ${rowNumber} 行的产品图片须为 HTTP/HTTPS 图片链接，或在该列插入图片`)
    }
    if (Object.entries(row).some(([key, value]) => key !== 'row_number' && value)) rows.push(row)
  }
  if (!rows.length) throw new Error('导入文件中没有产品数据')
  if (rows.length > 5000) throw new Error('单次最多导入 5,000 行')
  const seen = new Set()
  for (const row of rows) {
    if (!row.spu || !row.sku || !row.asin) throw new Error(`第 ${row.row_number} 行的 SPU、SKU、ASIN 均为必填项`)
    if (seen.has(row.sku)) throw new Error(`第 ${row.row_number} 行 SKU 重复：${row.sku}`)
    seen.add(row.sku)
  }
  const imageColumn = headerIndexes.get('产品图片') || headerIndexes.get('图片链接') || headerIndexes.get('图片')
  const embedded = new Map()
  for (const placement of worksheet.getImages()) {
    const anchor = placement.range.tl
    if (Math.floor(anchor.col) + 1 !== imageColumn) continue
    const rowNumber = Math.floor(anchor.row) + 1
    if (!rows.some(row => row.row_number === rowNumber)) continue
    if (embedded.has(rowNumber)) throw new Error(`第 ${rowNumber} 行存在多张产品图片，请仅保留一张`)
    const image = workbook.getImage(placement.imageId)
    const extension = image.extension?.toLowerCase()
    const mime = { png: 'image/png', jpeg: 'image/jpeg', jpg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp' }[extension]
    if (!mime || !image.buffer) throw new Error(`第 ${rowNumber} 行图片格式不支持，请使用普通插入图片或图片链接`)
    const blob = new Blob([image.buffer], { type: mime })
    if (!blob.size || blob.size > 30 * 1024 * 1024) throw new Error(`第 ${rowNumber} 行图片大小须在 30 MB 以内`)
    embedded.set(rowNumber, blob)
  }
  if (embedded.size && !uploadImage) throw new Error('导入内嵌图片需要图片上传服务')
  // Validate the whole sheet before uploading; sequential uploads avoid bursts.
  for (const row of rows) {
    if (embedded.has(row.row_number)) row.image_url = await uploadImage(embedded.get(row.row_number))
  }
  return rows
}
