import { customsDate, customsFeeAmount, customsTradeMode } from './customsDeclarations.js'

const numberKeys = new Set(['packages', 'gross_weight', 'net_weight', 'unit_price', 'total_price', 'freight', 'insurance'])
function cellValue(key, value) {
  if (key === 'trade_mode') value = customsTradeMode(value)
  if (key === 'freight' || key === 'insurance') value = customsFeeAmount(value)
  if (value == null || value === '') return null
  if (key.endsWith('_date')) return customsDate(value)
  const raw = String(value)
  // Excel stores at most 15 significant digits; keep longer values as exact text.
  if (numberKeys.has(key) && /^\d+(?:\.\d+)?$/.test(raw) && raw.replace('.', '').replace(/^0+/, '').length <= 15) return Number(raw)
  return raw
}

export async function buildCustomsReport(records, { fields, items, allFields = false, filters = {}, exportedAt = new Date() }) {
  const module = await import('exceljs')
  const workbook = new (module.default || module).Workbook()
  const sheet = workbook.addWorksheet('报关单明细')
  const extra = allFields ? ['识别方式', '核对提示', '文件名', '上传时间'] : []
  const headers = ['海关编号', ...fields.map(([, label]) => label), ...items.map(([, label]) => label), '识别状态', ...extra]
  sheet.columns = headers.map(label => ({ header: label, width: label === '海关编号' ? 24 : label === '币制' ? 9 : label === '识别状态' ? 12 : 21 }))
  sheet.views = [{ state: 'frozen', xSplit: 1, ySplit: 1 }]
  for (const record of records) {
    const products = record.items?.length ? record.items : [{}]
    const warnings = Array.isArray(record.warnings) ? record.warnings : []
    const firstRow = sheet.rowCount + 1
    const common = [String(record.declaration_number || ''), ...fields.map(([key]) => cellValue(key, record.fields?.[key] || (key === 'declaration_date' ? record.declaration_date : '')))]
    const tail = [warnings.length ? '需核对' : '已识别', ...(allFields ? [record.recognition_method === 'ocr' ? '扫描识别' : '文字提取', warnings.join('；'), record.file_name || '', record.created_at ? new Date(record.created_at).toLocaleString('zh-CN', { hour12: false }) : ''] : [])]
    for (const product of products) sheet.addRow([...common, ...items.map(([key]) => cellValue(key, product[key])), ...tail])
    if (products.length > 1) {
      for (let col = 1; col <= headers.length; col++) {
        if (col <= common.length || col > common.length + items.length) sheet.mergeCells(firstRow, col, sheet.rowCount, col)
      }
    }
  }
  sheet.eachRow((row, index) => {
    row.height = index === 1 ? 30 : 26
    row.eachCell(cell => {
      cell.font = { name: 'Calibri', size: 11, bold: index === 1, color: { argb: index === 1 ? 'FFFFFFFF' : 'FF33485B' } }
      cell.alignment = { vertical: 'middle', horizontal: typeof cell.value === 'number' ? 'right' : 'left', wrapText: true }
      cell.border = { bottom: { style: 'thin', color: { argb: 'FFE3E8EF' } } }
      if (index === 1) cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF285D83' } }
      if (typeof cell.value === 'number') cell.numFmt = '#,##0.###############'
    })
  })
  sheet.getColumn(1).numFmt = '@'
  const info = workbook.addWorksheet('导出说明')
  info.columns = [{ width: 22 }, { width: 85 }]
  info.addRows([['报表', '报关单明细'], ['申报日期起', filters.date_from || '不限'], ['申报日期止', filters.date_to || '不限'], ['指运港', filters.destination_port || '全部'], ['报关单数量', records.length], ['商品明细数量', records.reduce((sum, row) => sum + (row.items?.length || 0), 0)], ['字段范围', allFields ? '全部字段' : '当前显示字段'], ['导出时间', exportedAt.toLocaleString('zh-CN', { hour12: false })], ['数据口径', '导出当前筛选条件下的全部报关单，不受页面分页限制。单据信息按海关编号合并；运保费、件数及重量每单计一次。金额保留原币种，未进行汇率换算。']])
  info.getColumn(2).alignment = { wrapText: true, vertical: 'top' }
  info.getRow(9).height = 46
  return workbook
}
