export const MCF_HEADERS = [
  'MerchantFulfillmentOrderID',
  'DisplayableOrderID',
  'DisplayableOrderDate',
  'MerchantSKU',
  'Quantity',
  'MerchantFulfillmentOrderItemID',
  'GiftMessage',
  'DisplayableComment',
  'PerUnitDeclaredValue',
  'DisplayableOrderComment',
  'DeliverySLA',
  'AddressName',
  'AddressFieldOne',
  'AddressFieldTwo',
  'AddressFieldThree',
  'AddressCity',
  'AddressCountryCode',
  'AddressStateOrRegion',
  'AddressPostalCode',
  'AddressPhoneNumber',
  'NotificationEmail',
  'FulfillmentAction',
  'MarketplaceID',
  'CarrierPreferences',
]

export const MCF_ORDER_COMMENT = 'Thank you for choosing FORIOUS. We hope you enjoy your purchase!'

export const MCF_NON_MERGE_AMAZON_SKUS = [
  'FB01010B-10-Black', 'FB01010BN-10-Brushed Nickel', 'FB01010CH-10-Chrome', 'FB01010G-10-Brushed Gold',
  'FB01011B-10-Black', 'FB01011BN-10-Brushed Nickel', 'FB01011CH-10-Chrome', 'FB01011G-10-Brushed Gold',
  'FB01020B', 'FB01020BN', 'FB01020G',
  'FB02010B-12-Black', 'FB02010BN-12-Brushed Nickel', 'FB02010CH-12-Chrome', 'FB02010G-12-Brushed Gold',
  'FB02011B-12-Black', 'FB02011BN-12-Brushed Nickel', 'FB02011CH-12-Chrome', 'FB02011G-12-Brushed Gold',
]

const mcfNonMergeAmazonSkuSet = new Set(MCF_NON_MERGE_AMAZON_SKUS)

export const DEFAULT_FIELD_RULES = {
  TEMU: [
    ['orderId', ['订单号'], true],
    ['status', ['订单状态'], true],
    ['quantity', ['应履约件数'], true],
    ['productName', ['商品名称'], false],
    ['sku', ['SKU货号'], true],
    ['recipient', ['收货人姓名'], true],
    ['phone', ['收货人联系方式'], true],
    ['email', ['邮箱'], false],
    ['address1', ['详细地址1'], true],
    ['address2', ['详细地址2'], false],
    ['address3', ['详细地址3'], false],
    ['district', ['区县'], false],
    ['city', ['城市'], true],
    ['state', ['省份'], true],
    ['postalCode', ['收货地址邮编'], true],
    ['country', ['国家'], true],
    ['createdTime', ['订单创建时间'], true],
  ],
  TIKTOK: [
    ['orderId', ['Order ID'], true],
    ['status', ['Order SubStatus'], true],
    ['sku', ['Seller SKU'], true],
    ['quantity', ['Quantity'], true],
    ['returnQuantity', ['Sku Quantity of return'], false],
    ['fulfillmentType', ['Delivery Option Type'], true],
    ['productName', ['Product Name'], false],
    ['recipient', ['Recipient'], true],
    ['phone', ['Phone #', 'PHONE', 'Phone'], true],
    ['country', ['Country'], true],
    ['state', ['State'], true],
    ['city', ['City'], true],
    ['postalCode', ['Zipcode'], true],
    ['address1', ['Address Line 1'], true],
    ['address2', ['Address Line 2'], false],
    ['createdTime', ['Created Time'], true],
  ],
  LOWES: [
    ['orderId', ['Order number'], true],
    ['status', ['Status'], true],
    ['quantity', ['Quantity'], true],
    ['productName', ['Details'], false],
    ['sku', ['Store SKU', 'Offer SKU'], true],
    ['recipientFirstName', ['Shipping address first name'], true],
    ['recipientLastName', ['Shipping address last name'], false],
    ['phone', ['Shipping address phone'], true],
    ['email', ['Shipping address email'], false],
    ['address1', ['Shipping address street 1'], true],
    ['address2', ['Shipping address street 2'], false],
    ['city', ['Shipping address city'], true],
    ['state', ['Shipping address state'], true],
    ['postalCode', ['Shipping address zip'], true],
    ['country', ['Shipping address country'], true],
    ['createdTime', ['Date created'], true],
  ],
}

const STATE_ROWS = [
  ['AL', 'Alabama', '阿拉巴马', '阿拉巴马州'],
  ['AK', 'Alaska', '阿拉斯加', '阿拉斯加州'],
  ['AZ', 'Arizona', '亚利桑那', '亚利桑那州'],
  ['AR', 'Arkansas', '阿肯色', '阿肯色州'],
  ['CA', 'California', '加利福尼亚', '加利福尼亚州', '加州'],
  ['CO', 'Colorado', '科罗拉多', '科罗拉多州'],
  ['CT', 'Connecticut', '康涅狄格', '康涅狄格州'],
  ['DE', 'Delaware', '特拉华', '特拉华州'],
  ['FL', 'Florida', '佛罗里达', '佛罗里达州'],
  ['GA', 'Georgia', '佐治亚', '佐治亚州', '乔治亚', '乔治亚州'],
  ['HI', 'Hawaii', '夏威夷', '夏威夷州'],
  ['ID', 'Idaho', '爱达荷', '爱达荷州'],
  ['IL', 'Illinois', '伊利诺伊', '伊利诺伊州'],
  ['IN', 'Indiana', '印第安纳', '印第安纳州'],
  ['IA', 'Iowa', '爱荷华', '爱荷华州', '艾奥瓦', '艾奥瓦州'],
  ['KS', 'Kansas', '堪萨斯', '堪萨斯州'],
  ['KY', 'Kentucky', '肯塔基', '肯塔基州'],
  ['LA', 'Louisiana', '路易斯安那', '路易斯安那州'],
  ['ME', 'Maine', '缅因', '缅因州'],
  ['MD', 'Maryland', '马里兰', '马里兰州'],
  ['MA', 'Massachusetts', '马萨诸塞', '马萨诸塞州', '麻萨诸塞', '麻萨诸塞州'],
  ['MI', 'Michigan', '密歇根', '密歇根州'],
  ['MN', 'Minnesota', '明尼苏达', '明尼苏达州'],
  ['MS', 'Mississippi', '密西西比', '密西西比州'],
  ['MO', 'Missouri', '密苏里', '密苏里州'],
  ['MT', 'Montana', '蒙大拿', '蒙大拿州'],
  ['NE', 'Nebraska', '内布拉斯加', '内布拉斯加州'],
  ['NV', 'Nevada', '内华达', '内华达州'],
  ['NH', 'New Hampshire', '新罕布什尔', '新罕布什尔州'],
  ['NJ', 'New Jersey', '新泽西', '新泽西州'],
  ['NM', 'New Mexico', '新墨西哥', '新墨西哥州'],
  ['NY', 'New York', '纽约', '纽约州'],
  ['NC', 'North Carolina', '北卡罗来纳', '北卡罗来纳州'],
  ['ND', 'North Dakota', '北达科他', '北达科他州'],
  ['OH', 'Ohio', '俄亥俄', '俄亥俄州'],
  ['OK', 'Oklahoma', '俄克拉何马', '俄克拉何马州', '俄克拉荷马', '俄克拉荷马州'],
  ['OR', 'Oregon', '俄勒冈', '俄勒冈州'],
  ['PA', 'Pennsylvania', '宾夕法尼亚', '宾夕法尼亚州'],
  ['RI', 'Rhode Island', '罗得岛', '罗得岛州', '罗德岛', '罗德岛州'],
  ['SC', 'South Carolina', '南卡罗来纳', '南卡罗来纳州'],
  ['SD', 'South Dakota', '南达科他', '南达科他州'],
  ['TN', 'Tennessee', '田纳西', '田纳西州'],
  ['TX', 'Texas', '得克萨斯', '得克萨斯州', '德克萨斯', '德克萨斯州'],
  ['UT', 'Utah', '犹他', '犹他州'],
  ['VT', 'Vermont', '佛蒙特', '佛蒙特州'],
  ['VA', 'Virginia', '弗吉尼亚', '弗吉尼亚州', '维吉尼亚', '维吉尼亚州'],
  ['WA', 'Washington', '华盛顿', '华盛顿州'],
  ['WV', 'West Virginia', '西弗吉尼亚', '西弗吉尼亚州'],
  ['WI', 'Wisconsin', '威斯康星', '威斯康星州'],
  ['WY', 'Wyoming', '怀俄明', '怀俄明州'],
  ['DC', 'District of Columbia', 'Washington DC', '哥伦比亚特区', '华盛顿特区', '华盛顿哥伦比亚特区'],
]

const STATE_MAP = new Map()
for (const [code, ...aliases] of STATE_ROWS) {
  STATE_MAP.set(normalizeLookupKey(code), code)
  for (const alias of aliases) STATE_MAP.set(normalizeLookupKey(alias), code)
}

export async function readOrderFile(file, platform, fieldRules) {
  const normalizedPlatform = normalizePlatform(platform)
  if (!normalizedPlatform) throw new Error('请选择来源平台')
  if (normalizedPlatform === 'TIKTOK' && !sanitizeText(file?.name).includes('待发货')) {
    throw new Error('TikTok 订单文件名必须包含“待发货”')
  }
  const rows = await readTableRows(file)
  return parseOrderRows(normalizedPlatform, rows, fieldRules)
}

export async function readSkuMappingFile(file) {
  const rows = await readTableRows(file)
  const platformHeaders = ['平台 SKU', '平台SKU', 'Platform SKU', 'Seller SKU', 'Source SKU']
  const amazonHeaders = ['Amazon SKU', 'AmazonSKU', '亚马逊 SKU', '亚马逊SKU']
  const nonAmazonHeaders = ['非亚 SKU', '非亚SKU', 'Non-Amazon SKU', 'Non Amazon SKU', 'NonAmazonSKU']
  const remarkHeaders = ['备注', 'Remark', 'Notes']
  let headerIndex = -1
  let platformIndex = -1
  let amazonIndex = -1
  let nonAmazonIndex = -1
  let remarkIndex = -1
  for (let index = 0; index < Math.min(rows.length, 10); index += 1) {
    const headers = new Map((rows[index] || []).map((header, column) => [normalizeHeaderKey(header), column]))
    platformIndex = platformHeaders.map((header) => headers.get(normalizeHeaderKey(header))).find((column) => column !== undefined) ?? -1
    amazonIndex = amazonHeaders.map((header) => headers.get(normalizeHeaderKey(header))).find((column) => column !== undefined) ?? -1
    nonAmazonIndex = nonAmazonHeaders.map((header) => headers.get(normalizeHeaderKey(header))).find((column) => column !== undefined) ?? -1
    remarkIndex = remarkHeaders.map((header) => headers.get(normalizeHeaderKey(header))).find((column) => column !== undefined) ?? -1
    if (platformIndex >= 0 && amazonIndex >= 0 && nonAmazonIndex >= 0) {
      headerIndex = index
      break
    }
  }
  if (headerIndex < 0) throw new Error('映射表必须包含“平台 SKU”“Amazon SKU”和“非亚 SKU”三列')

  const result = []
  const seen = new Map()
  for (let index = headerIndex + 1; index < rows.length; index += 1) {
    const platformSku = sanitizeText(rows[index]?.[platformIndex])
    const amazonSku = sanitizeText(rows[index]?.[amazonIndex])
    const nonAmazonSku = sanitizeText(rows[index]?.[nonAmazonIndex])
    const remark = remarkIndex >= 0 ? sanitizeText(rows[index]?.[remarkIndex]) : ''
    if (!platformSku && !amazonSku && !nonAmazonSku) continue
    if (!platformSku || !amazonSku || !nonAmazonSku) throw new Error(`映射表第 ${index + 1} 行存在空 SKU`)
    const key = `${platformSku}\u0000${amazonSku}`
    if (seen.has(key)) {
      const existing = seen.get(key)
      continue
    }
    seen.set(key, { amazonSku, nonAmazonSku })
    result.push({ platform_sku: platformSku, amazon_sku: amazonSku, non_amazon_sku: nonAmazonSku, remark, row_number: index + 1 })
  }
  if (!result.length) throw new Error('映射表中没有可导入的数据')
  if (result.length > 2000) throw new Error('单次最多导入 2,000 条 SKU 映射')
  return result
}

async function readTableRows(file) {
  if (!file) throw new Error('请选择表格文件')
  if (file.size > 10 * 1024 * 1024) throw new Error('文件不能超过 10 MB')
  const buffer = await file.arrayBuffer()
  const bytes = new Uint8Array(buffer)
  return isZipWorkbook(bytes) ? readExcelRows(buffer) : parseCSV(decodeTextTable(bytes))
}

async function readExcelRows(buffer) {
  const ExcelJSModule = await import('exceljs')
  const Workbook = ExcelJSModule.Workbook || ExcelJSModule.default?.Workbook
  if (!Workbook) throw new Error('Excel 解析组件加载失败')
  const workbook = new Workbook()
  await workbook.xlsx.load(buffer, { ignoreNodes: ['dataValidations'] })
  const worksheet = workbook.worksheets[0]
  if (!worksheet) throw new Error('Excel 文件中没有工作表')
  const rows = []
  for (let rowIndex = 1; rowIndex <= worksheet.actualRowCount; rowIndex += 1) {
    const row = worksheet.getRow(rowIndex)
    const values = []
    for (let colIndex = 1; colIndex <= worksheet.actualColumnCount; colIndex += 1) {
      values.push(row.getCell(colIndex).text)
    }
    rows.push(values)
  }
  return rows
}

export function parseOrderRows(platform, rows, fieldRules) {
  const normalizedPlatform = normalizePlatform(platform)
  const rules = normalizeFieldRules(fieldRules || DEFAULT_FIELD_RULES[normalizedPlatform])
  const headerIndex = findBestHeaderIndex(rows, rules)
  if (headerIndex < 0) throw createFieldMappingError(normalizedPlatform, rules, [])
  const headers = rows[headerIndex].map(normalizeHeader)
  const headerKeys = new Set(headers.map(normalizeHeaderKey))
  const missingFields = rules
    .filter((rule) => rule.required && !rule.source_headers.some((header) => headerKeys.has(normalizeHeaderKey(header))))
    .map((rule) => rule.field_key)
  if (missingFields.length) throw createFieldMappingError(normalizedPlatform, rules, headers, missingFields)

  const indexes = new Map(headers.map((header, index) => [normalizeHeaderKey(header), index]))
  const records = []
  for (let index = headerIndex + 1; index < rows.length; index += 1) {
    const row = rows[index] || []
    if (!row.some((value) => sanitizeText(value) !== '')) continue
    const get = (fieldKey) => {
      const rule = rules.find((item) => item.field_key === fieldKey)
      const sourceIndex = rule?.source_headers
        .map((header) => indexes.get(normalizeHeaderKey(header)))
        .find((value) => value !== undefined)
      return sanitizeText(row[sourceIndex] ?? '')
    }
    const parser = {
      TEMU: parseTemuRow,
      TIKTOK: parseTikTokRow,
      LOWES: parseLowesRow,
    }[normalizedPlatform]
    const record = parser(get, index + 1)
    records.push(record)
  }
  if (!records.length) throw new Error('文件中没有订单明细')
  if (records.length > 20000) throw new Error('单次最多处理 20,000 条订单明细')
  return records
}

function parseTemuRow(get, rowNumber) {
  const sourceOrderId = cleanIdentifier(get('orderId'))
  const status = get('status')
  const record = {
    id: `TEMU-${rowNumber}`,
    rowNumber,
    platform: 'TEMU',
    status,
    sourceOrderId,
    orderId: sourceOrderId,
    date: normalizeDate(get('createdTime')),
    sourceSku: cleanIdentifier(get('sku')),
    productName: get('productName'),
    quantity: parseWholeNumber(get('quantity')),
    recipient: get('recipient'),
    address1: sanitizeAddressText(get('address1')),
    address2: sanitizeAddressText(get('address2')),
    address3: joinPresentValues(sanitizeAddressText(get('address3')), sanitizeAddressText(get('district'))),
    city: get('city'),
    country: normalizeCountry(get('country')),
    state: normalizeState(get('state')),
    postalCode: cleanIdentifier(get('postalCode')),
    phone: get('phone'),
    email: get('email'),
    warnings: [],
    excludedReason: isPendingStatus(status) ? '' : `订单状态为“${status || '空'}”`,
  }
  return record
}

function parseTikTokRow(get, rowNumber) {
  const sourceOrderId = cleanIdentifier(get('orderId'))
  const status = get('status')
  const fulfillmentType = get('fulfillmentType')
  const quantity = parseWholeNumber(get('quantity'))
  const returnQuantity = parseWholeNumber(get('returnQuantity'), 0)
  const finalQuantity = Number.isInteger(quantity) && Number.isInteger(returnQuantity)
    ? quantity - returnQuantity
    : null
  let excludedReason = ''
  if (status !== '待发货') excludedReason = `该订单号${sourceOrderId || '空'}状态不是待发货`
  else if (fulfillmentType !== '商家物流') excludedReason = `该订单号${sourceOrderId || '空'}的配送类型不是商家物流`
  else if (Number.isInteger(finalQuantity) && finalQuantity <= 0) excludedReason = '扣除退货数量后无需履约'

  return {
    id: `TIKTOK-${rowNumber}`,
    rowNumber,
    platform: 'TIKTOK',
    status,
    fulfillmentType,
    sourceOrderId,
    orderId: sourceOrderId,
    date: normalizeDate(get('createdTime')),
    sourceSku: cleanIdentifier(get('sku')),
    productName: get('productName'),
    quantity: finalQuantity,
    recipient: get('recipient'),
    address1: sanitizeAddressText(get('address1')),
    address2: sanitizeAddressText(get('address2')),
    address3: '',
    city: get('city'),
    country: normalizeCountry(get('country')),
    state: normalizeState(get('state')),
    postalCode: cleanIdentifier(get('postalCode')),
    phone: get('phone'),
    email: '',
    warnings: [],
    excludedReason,
  }
}

function parseLowesRow(get, rowNumber) {
  const sourceOrderId = cleanIdentifier(get('orderId'))
  const status = get('status')
  return {
    id: `LOWES-${rowNumber}`,
    rowNumber,
    platform: 'LOWES',
    status,
    sourceOrderId,
    orderId: sourceOrderId,
    date: normalizeDate(get('createdTime')),
    sourceSku: cleanIdentifier(get('sku')),
    productName: get('productName'),
    quantity: parseWholeNumber(get('quantity')),
    recipient: joinPresentValues(get('recipientFirstName'), get('recipientLastName')),
    address1: sanitizeAddressText(get('address1')),
    address2: sanitizeAddressText(get('address2')),
    address3: '',
    city: get('city'),
    country: normalizeCountry(get('country')),
    state: normalizeState(get('state')),
    postalCode: cleanIdentifier(get('postalCode')),
    phone: get('phone'),
    email: get('email'),
    warnings: [],
    excludedReason: isPendingStatus(status) ? '' : `订单状态为“${status || '空'}”`,
  }
}

export function buildMCFPreview(records, mappings = {}) {
  const mergeOrderIDs = buildMergedOrderIDs(records.filter((record) => {
    const mapping = mappings[record.sourceSku]
    return !mapping?.selection_required && validateRecord(record, mappingAmazonSku(mapping)).length === 0
  }), mappings)
  const groups = new Map()
  for (const record of records) {
    if (record.excludedReason) continue
    const mapping = mappings[record.sourceSku]
    const amazonSku = sanitizeText(typeof mapping === 'string' ? mapping : mapping?.amazon_sku)
    const errors = validateRecord(record, amazonSku)
    if (mapping?.selection_required) {
      errors.push(`SKU ${record.sourceSku} 不是当前平台 SKU，请到 SKU 映射中将正确项设为当前`)
    }
    const warnings = [...(record.warnings || [])]
    const merged = mergeOrderIDs.get(record.sourceOrderId)
    const outputOrderID = merged?.orderId || record.orderId
    const outputOrderDate = merged?.platform === 'TIKTOK' ? merged.date : record.date
    // 合并单统一订单号；同一合并单内的相同 Amazon SKU 合并为一条明细并累加数量。
    const key = amazonSku ? `${outputOrderID}\u0000${amazonSku}` : `${outputOrderID}\u0000__row_${record.rowNumber}`

    if (!groups.has(key)) {
      groups.set(key, {
        id: key,
        sourceRows: [record.rowNumber],
        sourceSkus: [record.sourceSku],
        sourceItems: [{ orderId: record.sourceOrderId, sku: record.sourceSku, quantity: record.quantity }],
        platform: record.platform,
        orderId: outputOrderID,
        originalOrderIds: merged?.originalOrderIds || [record.sourceOrderId],
        isMergedOrder: Boolean(merged),
        date: outputOrderDate,
        amazonSku,
        quantity: Number.isInteger(record.quantity) ? record.quantity : 0,
        recipient: record.recipient,
        address1: record.address1,
        address2: record.address2,
        address3: record.address3,
        city: record.city,
        country: record.country,
        state: record.state,
        postalCode: record.postalCode,
        phone: record.phone,
        email: record.email,
        errors,
        warnings,
      })
      continue
    }

    const group = groups.get(key)
    group.sourceRows.push(record.rowNumber)
    if (!group.sourceSkus.includes(record.sourceSku)) group.sourceSkus.push(record.sourceSku)
    group.sourceItems.push({ orderId: record.sourceOrderId, sku: record.sourceSku, quantity: record.quantity })
    if (Number.isInteger(record.quantity)) group.quantity += record.quantity
    group.errors = uniqueStrings([...group.errors, ...errors])
    group.warnings = uniqueStrings([...group.warnings, ...warnings, '同一合并单相同 Amazon SKU 已合并数量'])
    if (!hasSameAddress(group, record)) group.errors.push('同一订单的收件信息不一致')
  }

  return [...groups.values()].flatMap((item) => {
    item.errors = uniqueStrings(item.errors)
    item.warnings = uniqueStrings(item.warnings)
    item.status = item.errors.length ? 'blocked' : item.warnings.length ? 'warning' : 'ready'
    item.values = [
      item.orderId,
      item.orderId,
      item.date,
      item.amazonSku,
      String(item.quantity),
      item.amazonSku,
      '',
      '',
      '',
      MCF_ORDER_COMMENT,
      'Standard',
      sanitizeText(item.recipient),
      sanitizeAddressText(item.address1),
      sanitizeAddressText(item.address2),
      sanitizeAddressText(item.address3),
      sanitizeText(item.city),
      item.country,
      item.state,
      sanitizeText(item.postalCode),
      sanitizeText(item.phone),
      sanitizeText(item.email),
      '',
      '',
      '',
    ]
    return splitMCFPreviewRow(item)
  })
}

export function excludeGeneratedMCFRecords(records, generatedItems) {
  const generated = new Set((generatedItems || []).flatMap((item) => {
    const orderIDs = item.original_order_ids?.length ? item.original_order_ids : [item.order_id]
    return orderIDs.map((orderID) => `${orderID}\u0000${item.sku}`)
  }))
  return records.map((record) => !record.excludedReason && generated.has(`${record.sourceOrderId}\u0000${record.sourceSku}`)
    ? { ...record, excludedReason: '该订单商品已生成过 MCF，本次已排除' }
    : record)
}

function validateRecord(record, amazonSku) {
  const errors = []
  if (isMissing(record.sourceOrderId)) errors.push('订单号为空')
  if (!record.date) errors.push('订单日期无效')
  if (isMissing(record.sourceSku)) errors.push('平台 SKU 为空')
  else if (!amazonSku) errors.push(`SKU ${record.sourceSku} 未映射`)
  if (!Number.isInteger(record.quantity) || record.quantity <= 0) errors.push('履约数量必须是正整数')
  if (isMissing(record.recipient)) errors.push('收件人为空或被遮罩')
  if (isMissing(record.address1)) errors.push('地址 1 为空或被遮罩')
  if (isMissing(record.city)) errors.push('城市为空或被遮罩')
  if (!record.country) errors.push('国家不是美国或无法识别')
  if (!record.state) errors.push('州无法识别')
  if (isMissing(record.postalCode)) errors.push('邮编为空或被遮罩')
  if (!isValidPhone(record.phone)) errors.push('电话为空、被遮罩或位数不足')
  return errors
}

export function exportableMCFRows(previewRows) {
  return previewRows.filter((row) => ['ready', 'warning'].includes(row.status) && row.amazonSku)
}

export function unmappedMCFNotifications(previewRows) {
  const notifications = new Map()
  previewRows
    .filter((row) => row.status === 'blocked' && !row.amazonSku)
    .flatMap((row) => (row.sourceItems || []).map((source) => ({ orderId: source.orderId, sku: source.sku })))
    .filter((item) => item.orderId && item.sku)
    .forEach(({ orderId, sku }) => {
      const key = `${orderId}\u0000${sku}`
      notifications.set(key, { orderId, excludedReason: `该订单SKU未映射：${sku}` })
    })
  return [...notifications.values()]
}

export function createMCFBlob(previewRows) {
  if (!previewRows.length) throw new Error('没有可以导出的订单')
  if (previewRows.some((row) => row.status === 'blocked')) throw new Error('请先处理全部阻断问题')
  const lines = [MCF_HEADERS, ...previewRows.map((row) => row.values)]
  const text = lines.map((row) => row.map(sanitizeText).join('\t')).join('\r\n') + '\r\n'
  const bytes = new Uint8Array(2 + text.length * 2)
  bytes[0] = 0xff
  bytes[1] = 0xfe
  const view = new DataView(bytes.buffer)
  for (let index = 0; index < text.length; index += 1) {
    view.setUint16(2 + index * 2, text.charCodeAt(index), true)
  }
  return new Blob([bytes], { type: 'text/plain;charset=utf-16le' })
}

export function updateRecordAddress(record, changes) {
  return {
    ...record,
    recipient: sanitizeText(changes.recipient),
    address1: sanitizeAddressText(changes.address1),
    address2: sanitizeAddressText(changes.address2),
    address3: sanitizeAddressText(changes.address3),
    city: sanitizeText(changes.city),
    state: normalizeState(changes.state),
    postalCode: cleanIdentifier(changes.postalCode),
    country: normalizeCountry(changes.country),
    phone: sanitizeText(changes.phone),
    email: sanitizeText(changes.email),
  }
}

export function summarizeRecords(records, previewRows) {
  const excluded = records.filter((record) => record.excludedReason).length
  const active = records.length - excluded
  return {
    sourceRows: records.length,
    activeRows: active,
    excludedRows: excluded,
    outputRows: previewRows.length,
    blockedRows: previewRows.filter((row) => row.status === 'blocked').length,
    warningRows: previewRows.filter((row) => row.status === 'warning').length,
    quantity: previewRows.reduce((sum, row) => sum + (Number(row.quantity) || 0), 0),
  }
}

export function buildOrderTrackingItems(previewRows) {
  const result = new Map()
  for (const row of previewRows) {
    if (!['ready', 'warning'].includes(row.status) || !row.amazonSku) continue
    for (const source of row.sourceItems || []) {
      if (!source.sku) continue
      const packageKey = row.packageKey || ''
      const key = `${row.orderId}\u0000${source.sku}\u0000${packageKey}`
      if (!result.has(key)) {
        result.set(key, {
          order_id: row.orderId,
          sku: source.sku,
          package_key: packageKey,
          quantity: 0,
          recipient: row.recipient,
          phone: row.phone,
          address: [row.address1, row.address2, row.address3, row.city, row.state, row.postalCode, row.country].filter(Boolean).join(', '),
          original_order_ids: row.originalOrderIds || [source.orderId],
          original_items: [],
          is_merged_order: Boolean(row.isMergedOrder),
        })
      }
      const trackingItem = result.get(key)
      const sourceQuantity = Number(source.quantity) || 0
      const existingOriginalItem = trackingItem.original_items.find((item) => (
        item.original_order_id === source.orderId && item.sku === source.sku && item.amazon_sku === row.amazonSku
      ))
      if (existingOriginalItem) {
        existingOriginalItem.quantity += sourceQuantity
      } else {
        trackingItem.original_items.push({
          original_order_id: source.orderId,
          sku: source.sku,
          amazon_sku: row.amazonSku,
          quantity: sourceQuantity,
        })
      }
      trackingItem.quantity += sourceQuantity
    }
  }
  return [...result.values()]
}

function buildMergedOrderIDs(records, mappings = {}) {
  const addressGroups = new Map()
  const firstRecordByOrderID = new Map()
  const nonMergeOrderIDs = new Set(records
    .filter((record) => !record.excludedReason && record.sourceOrderId && isNonMergeAmazonSku(mappingAmazonSku(mappings[record.sourceSku])))
    .map((record) => record.sourceOrderId))
  for (const record of records) {
    if (record.excludedReason || !record.sourceOrderId) continue
    if (nonMergeOrderIDs.has(record.sourceOrderId)) continue
    if (!firstRecordByOrderID.has(record.sourceOrderId)) firstRecordByOrderID.set(record.sourceOrderId, record)
    const key = mergeAddressKey(record)
    if (!key) continue
    if (!addressGroups.has(key)) addressGroups.set(key, [])
    const orderIDs = addressGroups.get(key)
    if (!orderIDs.includes(record.sourceOrderId)) orderIDs.push(record.sourceOrderId)
  }
  const result = new Map()
  for (const orderIDs of addressGroups.values()) {
    if (orderIDs.length < 2) continue
    const orderId = `HB-${orderIDs.map(orderIDLastFour).join('-')}`
    const firstRecord = firstRecordByOrderID.get(orderIDs[0])
    for (const originalOrderID of orderIDs) {
      result.set(originalOrderID, {
        orderId,
        originalOrderIds: [...orderIDs],
        platform: firstRecord?.platform || '',
        date: firstRecord?.date || '',
      })
    }
  }
  return result
}

function mappingAmazonSku(mapping) {
  return sanitizeText(typeof mapping === 'string' ? mapping : mapping?.amazon_sku)
}

function isNonMergeAmazonSku(amazonSku) {
  return mcfNonMergeAmazonSkuSet.has(sanitizeText(amazonSku))
}

function splitMCFPreviewRow(item) {
  if (!isNonMergeAmazonSku(item.amazonSku) || !Number.isInteger(item.quantity) || item.quantity < 3) {
    return [item]
  }

  const quantities = splitMCFQuantity(item.quantity)
  const sourceItems = splitMCFSourceItems(item.sourceItems, quantities)
  return quantities.map((quantity, index) => {
    const row = {
      ...item,
      id: `${item.id}\u0000part-${index + 1}`,
      packageKey: `part-${index + 1}`,
      quantity,
      sourceItems: sourceItems[index],
      values: [...item.values],
    }
    row.values[4] = String(quantity)
    return row
  })
}

function splitMCFQuantity(quantity) {
  const result = []
  let remaining = quantity
  while (remaining >= 3) {
    result.push(2)
    remaining -= 2
  }
  if (remaining > 0) result.push(remaining)
  return result
}

function splitMCFSourceItems(sourceItems, quantities) {
  const normalized = (sourceItems || [])
    .map((source) => ({ ...source, quantity: Number(source.quantity) || 0 }))
    .filter((source) => source.quantity > 0)
  const parts = []
  let sourceIndex = 0
  let sourceRemaining = normalized[0]?.quantity || 0

  for (const quantity of quantities) {
    let remaining = quantity
    const part = []
    while (remaining > 0 && sourceIndex < normalized.length) {
      if (sourceRemaining <= 0) {
        sourceIndex += 1
        sourceRemaining = normalized[sourceIndex]?.quantity || 0
        continue
      }
      const allocated = Math.min(remaining, sourceRemaining)
      part.push({ ...normalized[sourceIndex], quantity: allocated })
      remaining -= allocated
      sourceRemaining -= allocated
    }
    if (remaining > 0 && normalized.length > 0) {
      part.push({ ...normalized[normalized.length - 1], quantity: remaining })
    }
    parts.push(part)
  }
  return parts
}

function mergeAddressKey(record) {
  const fields = ['recipient', 'address1', 'address2', 'address3', 'city', 'state', 'postalCode', 'country']
  const values = fields.map((field) => normalizeMergeValue(record[field]))
  if (!values[0] || !values[1] || !values[4] || !values[5] || !values[6] || !values[7]) return ''
  return values.join('\u0000')
}

function normalizeMergeValue(value) {
  return sanitizeText(value).toLowerCase().replace(/[\s._,#-]+/g, '')
}

function orderIDLastFour(value) {
  const clean = cleanIdentifier(value)
  return clean.slice(-4).padStart(4, '0')
}

export function normalizePlatform(value) {
  const normalized = sanitizeText(value).toUpperCase()
  if (normalized === 'TEMU') return 'TEMU'
  if (['TIKTOK', 'TIKTOK SHOP', 'TK'].includes(normalized)) return 'TIKTOK'
  if (['LOWES', "LOWE'S", 'LOWE’S', 'LOWE'].includes(normalized)) return 'LOWES'
  return ''
}

export function normalizeState(value) {
  return STATE_MAP.get(normalizeLookupKey(value)) || ''
}

export function normalizeCountry(value) {
  const key = normalizeLookupKey(value)
  return ['us', 'usa', 'unitedstates', 'unitedstatesofamerica', '美国', '美國'].includes(key) ? 'US' : ''
}

export function normalizeDate(value) {
  const clean = sanitizeText(value)
  if (!clean) return ''
  const dateOnly = clean.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})$/)
  if (dateOnly) return `${dateOnly[1]}-${pad2(dateOnly[2])}-${pad2(dateOnly[3])}T00:00:00`
  let match = clean.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})[ T](\d{1,2}):(\d{2}):(\d{2})/)
  if (match) return `${match[1]}-${pad2(match[2])}-${pad2(match[3])}T${pad2(match[4])}:${match[5]}:${match[6]}`
  match = clean.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})\s+(\d{1,2}):(\d{2}):(\d{2})\s*(AM|PM)$/i)
  if (!match) return ''
  let hour = Number(match[4])
  const period = match[7].toUpperCase()
  if (period === 'AM' && hour === 12) hour = 0
  if (period === 'PM' && hour !== 12) hour += 12
  return `${match[3]}-${pad2(match[1])}-${pad2(match[2])}T${pad2(hour)}:${match[5]}:${match[6]}`
}

export function parseCSV(text, delimiter = detectDelimiter(text)) {
  const rows = []
  let row = []
  let field = ''
  let quoted = false
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]
    if (quoted) {
      if (char === '"' && text[index + 1] === '"') {
        field += '"'
        index += 1
      } else if (char === '"') {
        quoted = false
      } else {
        field += char
      }
      continue
    }
    if (char === '"') quoted = true
    else if (char === delimiter) {
      row.push(field)
      field = ''
    } else if (char === '\n') {
      row.push(field)
      rows.push(row)
      row = []
      field = ''
    } else if (char !== '\r') {
      field += char
    }
  }
  if (field !== '' || row.length) {
    row.push(field)
    rows.push(row)
  }
  return rows
}

function decodeTextTable(bytes) {
  if (bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) {
    return new TextDecoder('utf-8').decode(bytes.subarray(3))
  }
  if (bytes[0] === 0xff && bytes[1] === 0xfe) {
    return new TextDecoder('utf-16le').decode(bytes.subarray(2))
  }
  if (bytes[0] === 0xfe && bytes[1] === 0xff) {
    return new TextDecoder('utf-16be').decode(bytes.subarray(2))
  }
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch {
    return new TextDecoder('gb18030', { fatal: true }).decode(bytes)
  }
}

function isZipWorkbook(bytes) {
  return bytes.length >= 4 && bytes[0] === 0x50 && bytes[1] === 0x4b
}

function detectDelimiter(text) {
  const line = String(text || '').split(/\r?\n/).find((item) => item.trim()) || ''
  const counts = new Map([[',', 0], ['\t', 0], [';', 0]])
  let quoted = false
  for (let index = 0; index < line.length; index += 1) {
    const char = line[index]
    if (char === '"') {
      if (quoted && line[index + 1] === '"') index += 1
      else quoted = !quoted
    } else if (!quoted && counts.has(char)) {
      counts.set(char, counts.get(char) + 1)
    }
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1])[0]?.[0] || ','
}

function findBestHeaderIndex(rows, rules) {
  let bestIndex = -1
  let bestScore = 0
  for (let index = 0; index < Math.min(rows.length, 10); index += 1) {
    const headers = new Set((rows[index] || []).map(normalizeHeaderKey))
    const score = rules.filter((rule) => rule.source_headers.some((header) => headers.has(normalizeHeaderKey(header)))).length
    if (score > bestScore) {
      bestIndex = index
      bestScore = score
    }
  }
  return bestIndex
}

function createFieldMappingError(platform, rules, detectedHeaders, missingFields) {
  const label = { TEMU: 'TEMU', TIKTOK: 'TikTok', LOWES: "Lowe's" }[platform] || platform
  const error = new Error(`文件字段与 ${label} 规则不匹配`)
  error.code = 'FIELD_MAPPING_MISMATCH'
  error.missingFields = missingFields || rules.filter((item) => item.required).map((item) => item.field_key)
  error.detectedHeaders = detectedHeaders.filter(Boolean)
  return error
}

function normalizeHeader(value) {
  return sanitizeText(value).replace(/^\uFEFF/, '')
}

function normalizeHeaderKey(value) {
  return normalizeHeader(value).toLocaleLowerCase()
}

export function normalizeFieldRules(rules) {
  if (!Array.isArray(rules)) return []
  return rules.map((rule) => {
    if (Array.isArray(rule)) {
      return { field_key: rule[0], source_headers: [...rule[1]], required: Boolean(rule[2]) }
    }
    return {
      field_key: sanitizeText(rule.field_key),
      source_headers: Array.isArray(rule.source_headers) ? rule.source_headers.map(sanitizeText).filter(Boolean) : [],
      required: Boolean(rule.required),
      updated_by: Number(rule.updated_by || 0),
      updated_by_name: sanitizeText(rule.updated_by_name),
      updated_at: rule.updated_at || '',
    }
  })
}

function isPendingStatus(value) {
  const normalized = sanitizeText(value).toLowerCase()
  return normalized === '待发货' || normalized === 'awaiting shipment' || normalized === 'to ship'
}

function parseWholeNumber(value, emptyFallback = null) {
  const clean = sanitizeText(value)
  if (!clean) return emptyFallback
  if (!/^\d+$/.test(clean)) return null
  return Number(clean)
}

function cleanIdentifier(value) {
  return sanitizeText(value).replace(/\.0+$/, '')
}

function joinPresentValues(...values) {
  return values.filter((value) => !isMissing(value)).map(sanitizeText).join(' ')
}

function isMissing(value) {
  const clean = sanitizeText(value)
  return !clean || clean === '--' || /\*{3,}/.test(clean)
}

function isValidPhone(value) {
  if (isMissing(value)) return false
  const digits = String(value).replace(/\D/g, '')
  return digits.length >= 10 && digits.length <= 15
}

function hasSameAddress(group, record) {
  const fields = ['recipient', 'address1', 'address2', 'address3', 'city', 'country', 'state', 'postalCode']
  return fields.every((field) => normalizeMergeValue(group[field]) === normalizeMergeValue(record[field]))
}

function normalizeLookupKey(value) {
  return sanitizeText(value).toLowerCase().replace(/[\s._-]/g, '')
}

function uniqueStrings(values) {
  return [...new Set(values.filter(Boolean))]
}

function pad2(value) {
  return String(value).padStart(2, '0')
}

function sanitizeText(value) {
  return String(value ?? '')
    .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/g, '')
    .replace(/[\t\r\n]+/g, ' ')
    .trim()
}

function sanitizeAddressText(value) {
  const clean = sanitizeText(value)
  return clean === '--' ? '' : clean
}
