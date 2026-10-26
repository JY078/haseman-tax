export const financeFields = Object.entries({ contract_number: '合同协议号', export_date: '进出口日期', declaration_date: '申报日期', bill_number: '提运单号', trade_mode: '贸易 / 监管方式', destination_port: '指运 / 经停港', freight: '运费', insurance: '保费', packages: '件数', gross_weight: '毛重（千克）', net_weight: '净重（千克）' })
export const financeItems = Object.entries({ name: '商品名称', quantity: '法定数量 / 单位', declared_quantity: '申报数量 / 单位', unit_price: '单价', total_price: '总价', currency: '币制' })
export const numericFields = new Set(['packages', 'gross_weight', 'net_weight', 'quantity', 'secondary_quantity', 'declared_quantity', 'unit_price', 'total_price', 'freight', 'insurance'])

export function customsDate(value) {
  const raw = String(value || '').trim()
  const match = /^(\d{4})-?(\d{2})-?(\d{2})$/.exec(raw)
  return match ? `${match[1]}-${match[2]}-${match[3]}` : raw || '—'
}

export function customsNumber(value, digits = 2) {
  const raw = String(value ?? '').trim()
  if (!/^-?\d+(?:\.\d+)?$/.test(raw)) return '—'
  return Number(raw).toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: digits })
}

export function customsMeasure(measure, digits = 2) {
  return measure?.count ? customsNumber(measure.value, digits) : '—'
}

export function customsFeeAmount(value) {
  const parts = String(value ?? '').trim().replaceAll('／', '/').split('/')
  const amount = (parts.length === 3 ? parts[1] : parts.length === 1 ? parts[0] : '').trim()
  return /^(?:\d+|\d{1,3}(?:,\d{3})+)(?:\.\d+)?$/.test(amount) ? amount.replaceAll(',', '') : null
}

export function customsTradeMode(value) {
  const raw = String(value ?? '').trim()
  const prefix = /^[（(]\s*(\d{4})\s*[）)]\s*(.*)$/.exec(raw)
  if (prefix) return `${prefix[2].trim()}(${prefix[1]})`
  return raw.replace(/\s*[（(]\s*(\d{4})\s*[）)]$/, '($1)')
}
