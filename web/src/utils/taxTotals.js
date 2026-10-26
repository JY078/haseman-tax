const normalize = value => String(value || '').normalize('NFKC').toLowerCase().replace(/[\s\p{P}\uFEFF]/gu, '')
const excluded = new Set([
  'Customer Order #', 'Customer Order line #', 'Purchase Order #', 'Purchase Order line #',
  'Commission Rate', 'Base Commission Rate', 'Partner Item Id', 'Partner GTIN', 'Product Tax Code', 'Ship to Zipcode',
  'Order ID', 'SKU ID', 'Product ID', 'Zipcode', 'Weight(kg)', 'Package ID',
  '亚马逊订单商品编号', '卖家订单编号', 'seller order id', '配送邮政编码', '追踪编码', 'settlement id', 'order postal',
  '客户单号', '客户订单号', '客户订单行号', '采购订单号', '采购订单行号', '佣金比例',
  '基础佣金比例', '合作方商品编号', '商品条码', '商品税码', '收货邮编', '订单ID', '产品ID', '邮编', '重量(千克)', '包裹ID',
].map(normalize))

export const isExcludedTaxTotalColumn = column => excluded.has(normalize(column))
