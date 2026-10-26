const financeHeaderLabels = {
  'date/time': '日期/时间',
  'settlement id': '结算编号',
  type: '类型',
  'order id': '订单编号',
  sku: 'SKU',
  description: '描述',
  quantity: '数量',
  marketplace: '商城',
  'account type': '账户类型',
  fulfillment: '配送方式',
  'order city': '订单城市',
  'order state': '订单州/省',
  'order postal': '订单邮编',
  'tax collection model': '代收税模式',
  'product sales': '商品销售额',
  'product sales tax': '商品销售税',
  'shipping credits': '运费补偿',
  'shipping credits tax': '运费补偿税',
  'gift wrap credits': '礼品包装补偿',
  'giftwrap credits tax': '礼品包装补偿税',
  'regulatory fee': '合规费用',
  'tax on regulatory fee': '合规费用税',
  'promotional rebates': '促销折扣',
  'promotional rebates tax': '促销折扣税',
  'marketplace withheld tax': '商城代扣税',
  'selling fees': '销售佣金',
  'fba fees': 'FBA费用',
  'other transaction fees': '其他交易费',
  other: '其他',
  total: '总计',
  'transaction status': '交易状态',
  'transaction release date': '交易放行日期',
}

const chineseHeaderNames = new Map(Object.entries(financeHeaderLabels).map(([english, chinese]) => [chinese.toLowerCase(), english]))

export function amazonFinanceHeaderLabel(value) {
  const header = String(value ?? '').trim()
  const key = header.toLowerCase()
  if (financeHeaderLabels[key]) return `${header}（${financeHeaderLabels[key]}）`
  const english = chineseHeaderNames.get(key)
  return english ? `${english}（${header}）` : header
}
