import assert from 'node:assert/strict'
import test from 'node:test'
import {
  MCF_HEADERS,
  MCF_NON_MERGE_AMAZON_SKUS,
  DEFAULT_FIELD_RULES,
  buildMCFPreview,
  buildOrderTrackingItems,
  excludeGeneratedMCFRecords,
  createMCFBlob,
  exportableMCFRows,
  unmappedMCFNotifications,
  normalizeFieldRules,
  parseCSV,
  parseOrderRows,
  readOrderFile,
  readSkuMappingFile,
} from './orderMcf.js'

test('field rule normalization preserves updater metadata', () => {
  const [rule] = normalizeFieldRules([{
    field_key: 'sku',
    source_headers: ['Seller SKU'],
    required: true,
    updated_by: 7,
    updated_by_name: '张三',
    updated_at: '2026-07-27T08:00:00Z',
  }])
  assert.equal(rule.updated_by, 7)
  assert.equal(rule.updated_by_name, '张三')
  assert.equal(rule.updated_at, '2026-07-27T08:00:00Z')
})

const temuHeaders = [
  '订单号', '订单状态', '应履约件数', '商品名称', 'SKU货号', '收货人姓名', '收货人联系方式',
  '邮箱', '详细地址1', '详细地址2', '详细地址3', '区县', '城市', '省份', '收货地址邮编', '国家', '订单创建时间',
]

const tiktokHeaders = [
  'Order ID', 'Order SubStatus', 'Seller SKU', 'Quantity', 'Sku Quantity of return', 'Product Name',
  'Created Time', 'Delivery Option Type', 'Recipient', 'Phone #', 'Country', 'State', 'City', 'Zipcode',
  'Address Line 1', 'Address Line 2', 'Shipping Information',
]

const lowesHeaders = [
  'Date created', 'Order number', 'Quantity', 'Details', 'Status', 'Store SKU', 'Offer SKU',
  'Shipping address email', 'Shipping address first name', 'Shipping address last name',
  'Shipping address street 1', 'Shipping address street 2', 'Shipping address zip',
  'Shipping address city', 'Shipping address state', 'Shipping address country', 'Shipping address phone',
]

test('TEMU rows map Amazon SKU to both MerchantSKU and item ID', () => {
  const records = parseOrderRows('TEMU', [temuHeaders, [
    'PO-100', '待发货', '2', 'Faucet', 'FF0043BN', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'United States', '2026-07-23 22:31:28',
  ]])
  const preview = buildMCFPreview(records, { FF0043BN: { amazon_sku: 'AMZ-43BN' } })
  assert.equal(preview.length, 1)
  assert.equal(preview[0].status, 'ready')
  assert.equal(preview[0].values[3], 'AMZ-43BN')
  assert.equal(preview[0].values[5], 'AMZ-43BN')
  assert.equal(preview[0].values[0], 'PO-100')
  assert.equal(preview[0].values[2], '2026-07-23T22:31:28')
})

test('address placeholder is empty while normal hyphenated addresses are preserved', () => {
  const records = parseOrderRows('TEMU', [temuHeaders, [
    'PO-ADDRESS', '待发货', '1', 'Faucet', 'SOURCE', 'Alex', '+1 205 555 0188', '',
    '12-14 Main St', '  --  ', '--', '--', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]])
  const preview = buildMCFPreview(records, { SOURCE: { amazon_sku: 'AMZ-SOURCE' } })

  assert.equal(records[0].address1, '12-14 Main St')
  assert.equal(records[0].address2, '')
  assert.equal(records[0].address3, '')
  assert.equal(preview[0].values[12], '12-14 Main St')
  assert.equal(preview[0].values[13], '')
  assert.equal(preview[0].values[14], '')
})

test('TikTok rows normalize Chinese state, subtract returns and use phone column only', () => {
  const records = parseOrderRows('TIKTOK', [tiktokHeaders, [
    '577491\t', '待发货', 'TK-BLACK', '3', '1', 'Faucet', '07/23/2026 8:10:09 PM\t',
    '商家物流', 'Summer Riggs', '(+1)4235342901', '美国', '田纳西', 'Jonesborough', '37659\t',
    '435 FOXGLOVE CT', '', 'Summer Riggs\n(+1)4235342901\nJonesborough',
  ]])
  const preview = buildMCFPreview(records, { 'TK-BLACK': { amazon_sku: 'AMZ-BLACK' } })
  assert.equal(records[0].quantity, 2)
  assert.equal(records[0].state, 'TN')
  assert.equal(records[0].phone, '(+1)4235342901')
  assert.equal(records[0].date, '2026-07-23T20:10:09')
  assert.equal(preview[0].values[0], '577491')
  assert.equal(preview[0].status, 'ready')
  assert.equal(preview[0].values[5], 'AMZ-BLACK')
})

test('TikTok Chinese country and state aliases normalize to two-letter codes', () => {
  const aliases = [
    ['麻萨诸塞', 'MA'],
    ['艾奥瓦', 'IA'],
    ['俄克拉荷马', 'OK'],
    ['罗德岛', 'RI'],
    ['华盛顿哥伦比亚特区', 'DC'],
  ]

  for (const [state, expected] of aliases) {
    const records = parseOrderRows('TIKTOK', [tiktokHeaders, [
      `ORDER-${expected}`, '待发货', `SKU-${expected}`, '1', '0', 'Faucet', '07/23/2026 8:10:09 PM',
      '商家物流', 'Alex', '+1 205 555 0188', '美国', state, 'City', '10001',
      '100 Main St', '', '',
    ]])
    assert.equal(records[0].country, 'US')
    assert.equal(records[0].state, expected)
  }
})

test('TikTok phone never falls back to Shipping Information', () => {
  const records = parseOrderRows('TIKTOK', [tiktokHeaders, [
    '577492', '待发货', 'TK-PHONE', '1', '0', 'Faucet', '07/23/2026 8:10:09 PM',
    '商家物流', 'Summer Riggs', '', 'US', 'TN', 'Jonesborough', '37659',
    '435 FOXGLOVE CT', '', 'Summer Riggs (+1)4235342901',
  ]])
  assert.equal(records[0].phone, '')
  const preview = buildMCFPreview(records, { 'TK-PHONE': { amazon_sku: 'AMZ-PHONE' } })
  assert.equal(preview[0].status, 'blocked')
  assert.ok(preview[0].errors.some((item) => item.includes('电话')))
})

test('TikTok merged orders share the first order DisplayableOrderDate', () => {
  const row = (orderId, sku, createdTime) => [
    orderId, '待发货', sku, '1', '0', 'Faucet', createdTime,
    '商家物流', 'Alex', '+1 205 555 0188', 'US', 'TN', 'Nashville', '37201',
    '100 Main St', '', '',
  ]
  const records = parseOrderRows('TIKTOK', [
    tiktokHeaders,
    row('TK-1001', 'TK-A', '07/23/2026 8:10:09 PM'),
    row('TK-1002', 'TK-B', '07/24/2026 8:10:09 PM'),
  ])
  const preview = buildMCFPreview(records, {
    'TK-A': { amazon_sku: 'AMZ-A' },
    'TK-B': { amazon_sku: 'AMZ-B' },
  })
  assert.equal(preview.length, 2)
  assert.ok(preview.every((row) => row.orderId === 'HB-1001-1002'))
  assert.ok(preview.every((row) => row.values[2] === '2026-07-23T20:10:09'))
})

test('TikTok non-pending substatus is excluded with order-specific reason', () => {
  const records = parseOrderRows('TIKTOK', [tiktokHeaders, [
    'TK-STATUS-1', '已取消', 'TK-STATUS', '1', '0', 'Faucet', '07/23/2026 8:10:09 PM',
    '商家物流', 'Alex', '+1 205 555 0188', 'US', 'TN', 'Nashville', '37201',
    '100 Main St', '', '',
  ]])
  assert.equal(records[0].excludedReason, '该订单号TK-STATUS-1状态不是待发货')
  assert.equal(buildMCFPreview(records, { 'TK-STATUS': { amazon_sku: 'AMZ-STATUS' } }).length, 0)
})

test('TikTok non-merchant delivery option is excluded with order-specific reason', () => {
  const records = parseOrderRows('TIKTOK', [tiktokHeaders, [
    'TK-DELIVERY-1', '待发货', 'TK-DELIVERY', '1', '0', 'Faucet', '07/23/2026 8:10:09 PM',
    '平台物流', 'Alex', '+1 205 555 0188', 'US', 'TN', 'Nashville', '37201',
    '100 Main St', '', '',
  ]])
  assert.equal(records[0].excludedReason, '该订单号TK-DELIVERY-1的配送类型不是商家物流')
  assert.equal(buildMCFPreview(records, { 'TK-DELIVERY': { amazon_sku: 'AMZ-DELIVERY' } }).length, 0)
})

test("Lowe's rows use Store SKU and merge recipient name", () => {
  const records = parseOrderRows('LOWES', [lowesHeaders, [
    '07/31/2026 12:38:03 AM', '414363965-A', '1', 'Toilet Paper Holder', 'Awaiting shipment',
    'WF12609BN', 'OFFER-FALLBACK', 'notify@example.com', 'Jody', 'Young', '759 Campbell Rd', '',
    '35747', 'Grant', 'AL', 'US', '256 6828045',
  ]])
  const preview = buildMCFPreview(records, { WF12609BN: { amazon_sku: 'AMZ-WF12609BN' } })

  assert.equal(records[0].sourceSku, 'WF12609BN')
  assert.equal(records[0].recipient, 'Jody Young')
  assert.equal(records[0].address2, '')
  assert.equal(records[0].date, '2026-07-31T00:38:03')
  assert.equal(preview[0].status, 'ready')
  assert.equal(preview[0].values[0], '414363965-A')
  assert.equal(preview[0].values[3], 'AMZ-WF12609BN')
  assert.equal(preview[0].values[5], 'AMZ-WF12609BN')
  assert.equal(preview[0].values[10], 'Standard')
})

test('order tracking only includes mapped exportable rows', () => {
  const mapped = [
    'PO-MAPPED', '待发货', '1', 'Faucet', 'MAPPED', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const unmapped = [...mapped]
  unmapped[0] = 'PO-UNMAPPED'
  unmapped[4] = 'UNMAPPED'
  unmapped[5] = 'Taylor'
  const records = parseOrderRows('TEMU', [temuHeaders, mapped, unmapped])
  const preview = buildMCFPreview(records, { MAPPED: { amazon_sku: 'AMZ-MAPPED' } })
  const items = buildOrderTrackingItems(preview)

  assert.deepEqual(items.map((item) => item.order_id), ['PO-MAPPED'])
  assert.deepEqual(items.map((item) => item.sku), ['MAPPED'])
})

test('partial export, reupload and mapping completion only export remaining orders', async () => {
  const first = ['PO-FIRST', '待发货', '2', 'Faucet', 'SKU-A', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28']
  const second = [...first]
  second[0] = 'PO-SECOND'
  second[4] = 'SKU-B'
  const records = parseOrderRows('TEMU', [temuHeaders, first, second])
  const mappings = { 'SKU-A': { amazon_sku: 'AMZ-A' } }
  const preview = buildMCFPreview(records, mappings)
  const rows = exportableMCFRows(preview)
  assert.deepEqual(rows.map((row) => row.orderId), ['PO-FIRST'])
  assert.deepEqual(rows[0].originalOrderIds, ['PO-FIRST'])
  assert.deepEqual(unmappedMCFNotifications(preview), [{ orderId: 'PO-SECOND', excludedReason: '该订单SKU未映射：SKU-B' }])
  const text = new TextDecoder('utf-16le').decode(await createMCFBlob(rows).arrayBuffer())
  assert.ok(text.includes('PO-FIRST'))
  assert.ok(!text.includes('PO-SECOND'))
  const saved = buildOrderTrackingItems(rows)
  assert.deepEqual(saved.map((item) => item.order_id), ['PO-FIRST'])
  const reuploaded = excludeGeneratedMCFRecords(records, saved)
  assert.equal(exportableMCFRows(buildMCFPreview(reuploaded, mappings)).length, 0)
  mappings['SKU-B'] = { amazon_sku: 'AMZ-B' }
  const remaining = buildMCFPreview(reuploaded, mappings)
  assert.deepEqual(exportableMCFRows(remaining).map((row) => row.orderId), ['PO-SECOND'])
  assert.deepEqual(unmappedMCFNotifications(remaining), [])
})

test('generated MCF order items are excluded by order and platform SKU', () => {
  const records = [
    { sourceOrderId: 'ORDER-1', sourceSku: 'SKU-1', excludedReason: '' },
    { sourceOrderId: 'ORDER-1', sourceSku: 'SKU-2', excludedReason: '' },
    { sourceOrderId: 'ORDER-2', sourceSku: 'SKU-1', excludedReason: '' },
  ]
  const result = excludeGeneratedMCFRecords(records, [{ order_id: 'ORDER-1', sku: 'SKU-1' }])
  assert.match(result[0].excludedReason, /已生成过 MCF/)
  assert.equal(result[1].excludedReason, '')
  assert.equal(result[2].excludedReason, '')
})

test('saved field aliases apply to future imports', () => {
  const rules = DEFAULT_FIELD_RULES.TEMU.map(([fieldKey, aliases, required]) => ({
    field_key: fieldKey,
    source_headers: fieldKey === 'orderId' ? ['新订单编号'] : aliases,
    required,
  }))
  const renamedHeaders = [...temuHeaders]
  renamedHeaders[0] = '新订单编号'
  const records = parseOrderRows('TEMU', [renamedHeaders, [
    'PO-ALIAS', '待发货', '1', 'Faucet', 'SOURCE', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]], rules)
  assert.equal(records[0].orderId, 'PO-ALIAS')
})

test('field mismatch reports missing standard fields and detected headers', () => {
  const renamedHeaders = [...temuHeaders]
  renamedHeaders[0] = '新的订单编号'
  assert.throws(
    () => parseOrderRows('TEMU', [renamedHeaders, []]),
    (error) => {
      assert.equal(error.code, 'FIELD_MAPPING_MISMATCH')
      assert.deepEqual(error.missingFields, ['orderId'])
      assert.ok(error.detectedHeaders.includes('新的订单编号'))
      return true
    },
  )
})

test('same order and Amazon SKU merge into one MCF item', () => {
  const base = [
    'PO-200', '待发货', '1', 'Faucet', 'SOURCE-A', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const second = [...base]
  second[4] = 'SOURCE-B'
  second[2] = '2'
  const records = parseOrderRows('TEMU', [temuHeaders, base, second])
  const preview = buildMCFPreview(records, {
    'SOURCE-A': { amazon_sku: 'AMZ-ONE' },
    'SOURCE-B': { amazon_sku: 'AMZ-ONE' },
  })
  assert.equal(preview.length, 1)
  assert.equal(preview[0].quantity, 3)
  assert.equal(preview[0].values[4], '3')
  assert.equal(preview[0].values[5], 'AMZ-ONE')
})

test('same recipient and address merge different orders with HB order ID', () => {
  const first = [
    'ORDER-1234', '待发货', '1', 'Faucet', 'SOURCE-A', 'Alex Smith', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const second = [...first]
  second[0] = 'ORDER-5678'
  second[4] = 'SOURCE-B'
  second[8] = '100 MAIN-ST.'
  const records = parseOrderRows('TEMU', [temuHeaders, first, second])
  const preview = buildMCFPreview(records, {
    'SOURCE-A': { amazon_sku: 'AMZ-A' },
    'SOURCE-B': { amazon_sku: 'AMZ-B' },
  })

  assert.equal(preview.length, 2)
  assert.ok(preview.every((row) => row.orderId === 'HB-1234-5678'))
  assert.ok(preview.every((row) => row.values[0] === 'HB-1234-5678' && row.values[1] === 'HB-1234-5678'))
  assert.deepEqual(preview[0].originalOrderIds, ['ORDER-1234', 'ORDER-5678'])
  assert.ok(preview.every((row) => row.isMergedOrder))

  const trackingItems = buildOrderTrackingItems(preview)
  assert.deepEqual(trackingItems.map((item) => item.sku).sort(), ['SOURCE-A', 'SOURCE-B'])
  assert.ok(trackingItems.every((item) => item.order_id === 'HB-1234-5678'))
  assert.ok(trackingItems.every((item) => item.is_merged_order))
  assert.deepEqual(trackingItems[0].original_order_ids, ['ORDER-1234', 'ORDER-5678'])
  assert.deepEqual(trackingItems.flatMap((item) => item.original_items).map((item) => [item.original_order_id, item.amazon_sku, item.quantity]), [
    ['ORDER-1234', 'AMZ-A', 1],
    ['ORDER-5678', 'AMZ-B', 1],
  ])
})

test('protected Amazon SKU prevents cross-order merging', () => {
  const first = [
    'ORDER-PROTECTED', '待发货', '1', 'Faucet', 'SOURCE-A', 'Alex Smith', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const second = [...first]
  second[0] = 'ORDER-NORMAL'
  second[4] = 'SOURCE-B'
  const records = parseOrderRows('TEMU', [temuHeaders, first, second])
  const preview = buildMCFPreview(records, {
    'SOURCE-A': { amazon_sku: MCF_NON_MERGE_AMAZON_SKUS[0] },
    'SOURCE-B': { amazon_sku: 'AMZ-NORMAL' },
  })

  assert.equal(preview.length, 2)
  assert.deepEqual(preview.map((row) => row.orderId), ['ORDER-PROTECTED', 'ORDER-NORMAL'])
  assert.ok(preview.every((row) => !row.isMergedOrder))
})

test('protected Amazon SKU quantity is split into twos with the original order ID', () => {
  const records = parseOrderRows('TEMU', [temuHeaders, [
    'ORDER-SPLIT', '待发货', '7', 'Faucet', 'SOURCE-SPLIT', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]])
  const preview = buildMCFPreview(records, { 'SOURCE-SPLIT': { amazon_sku: MCF_NON_MERGE_AMAZON_SKUS[1] } })

  assert.deepEqual(preview.map((row) => row.quantity), [2, 2, 2, 1])
  assert.ok(preview.every((row) => row.orderId === 'ORDER-SPLIT'))
  assert.ok(preview.every((row) => row.amazonSku === MCF_NON_MERGE_AMAZON_SKUS[1]))
  assert.deepEqual(preview.map((row) => row.values[4]), ['2', '2', '2', '1'])
  const trackingItems = buildOrderTrackingItems(preview)
  assert.deepEqual(trackingItems.map((item) => item.quantity), [2, 2, 2, 1])
  assert.equal(new Set(trackingItems.map((item) => item.package_key)).size, 4)
  assert.deepEqual(trackingItems.map((item) => item.package_key), ['part-1', 'part-2', 'part-3', 'part-4'])
  assert.ok(trackingItems.every((item) => !/[\u0000-\u001f\u007f]/.test(item.package_key)))
  assert.ok(trackingItems.every((item) => item.order_id === 'ORDER-SPLIT'))
  assert.deepEqual(trackingItems.map((item) => item.package_key), preview.map((row) => row.packageKey))
})

test('merged orders combine identical Amazon SKU into one MCF row and add quantity', () => {
  const first = [
    'ORDER-1234', '待发货', '1', 'Faucet', 'SOURCE-A', 'Alex Smith', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const second = [...first]
  second[0] = 'ORDER-5678'
  second[4] = 'SOURCE-B'
  const preview = buildMCFPreview(parseOrderRows('TEMU', [temuHeaders, first, second]), {
    'SOURCE-A': { amazon_sku: 'AMZ-SAME' },
    'SOURCE-B': { amazon_sku: 'AMZ-SAME' },
  })
  assert.equal(preview.length, 1)
  assert.equal(preview[0].orderId, 'HB-1234-5678')
  assert.equal(preview[0].quantity, 2)
  assert.equal(preview[0].values[4], '2')
  assert.deepEqual(preview[0].sourceRows, [2, 3])
})

test('three merged orders append all original order suffixes', () => {
  const row = (orderId, sku) => [
    orderId, '待发货', '1', 'Faucet', sku, 'Alex Smith', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const records = parseOrderRows('TEMU', [temuHeaders, row('ORDER-1234', 'SOURCE-A'), row('ORDER-5678', 'SOURCE-B'), row('ORDER-9012', 'SOURCE-C')])
  const preview = buildMCFPreview(records, {
    'SOURCE-A': { amazon_sku: 'AMZ-SAME' },
    'SOURCE-B': { amazon_sku: 'AMZ-SAME' },
    'SOURCE-C': { amazon_sku: 'AMZ-SAME' },
  })
  assert.equal(preview.length, 1)
  assert.equal(preview[0].orderId, 'HB-1234-5678-9012')
  assert.equal(preview[0].quantity, 3)
  assert.equal(preview[0].values[4], '3')
})

test('different recipient does not merge orders at the same address', () => {
  const first = [
    'ORDER-1234', '待发货', '1', 'Faucet', 'SOURCE-A', 'Alex Smith', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]
  const second = [...first]
  second[0] = 'ORDER-5678'
  second[4] = 'SOURCE-B'
  second[5] = 'Taylor Smith'
  const preview = buildMCFPreview(parseOrderRows('TEMU', [temuHeaders, first, second]), {
    'SOURCE-A': { amazon_sku: 'AMZ-A' },
    'SOURCE-B': { amazon_sku: 'AMZ-B' },
  })
  assert.deepEqual(preview.map((row) => row.orderId), ['ORDER-1234', 'ORDER-5678'])
})

test('non-current platform SKU blocks MCF and points users to mapping maintenance', () => {
  const records = parseOrderRows('TEMU', [temuHeaders, [
    'PO-201', '待发货', '1', 'Faucet', 'SOURCE-OLD', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]])
  const preview = buildMCFPreview(records, {
    'SOURCE-OLD': { amazon_sku: 'AMZ-ONE', selection_required: true },
  })
  assert.equal(preview[0].status, 'blocked')
  assert.ok(preview[0].errors.some((message) => message.includes('SKU 映射')))
})

test('mapped MCF rows remain exportable when another SKU is unmapped', () => {
  const rows = buildMCFPreview(parseOrderRows('TEMU', [temuHeaders,
    ['PO-MIXED', '待发货', '1', 'Faucet', 'SOURCE-MAPPED', 'Alex', '+1 205 555 0188', '', '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28'],
    ['PO-MIXED', '待发货', '1', 'Faucet', 'SOURCE-UNMAPPED', 'Alex', '+1 205 555 0188', '', '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28'],
  ]), { 'SOURCE-MAPPED': { amazon_sku: 'AMZ-MAPPED' } })
  assert.equal(exportableMCFRows(rows).length, 1)
  assert.deepEqual(unmappedMCFNotifications(rows), [{ orderId: 'PO-MIXED', excludedReason: '该订单SKU未映射：SOURCE-UNMAPPED' }])
})

test('MCF output is UTF-16 LE with exact 24-column header', async () => {
  const records = parseOrderRows('TEMU', [temuHeaders, [
    'PO-300', '待发货', '1', 'Faucet', 'SOURCE', 'Alex', '+1 205 555 0188', '',
    '100 Main St', '', '', '', 'Birmingham', 'AL', '35215', 'US', '2026-07-23 22:31:28',
  ]])
  const preview = buildMCFPreview(records, { SOURCE: { amazon_sku: 'AMZ-SOURCE' } })
  const bytes = new Uint8Array(await createMCFBlob(preview).arrayBuffer())
  assert.deepEqual([...bytes.slice(0, 2)], [0xff, 0xfe])
  const text = new TextDecoder('utf-16le').decode(bytes.slice(2))
  const lines = text.split('\r\n').filter((line) => line !== '')
  assert.deepEqual(lines[0].split('\t'), MCF_HEADERS)
  assert.equal(lines[1].split('\t').length, 24)
})

test('CSV parser keeps commas and newlines inside quoted cells', () => {
  assert.deepEqual(parseCSV('A,B\r\n"one, two","line 1\nline 2"\r\n'), [
    ['A', 'B'],
    ['one, two', 'line 1\nline 2'],
  ])
})

test('order upload ignores extension and detects tab-delimited text', async () => {
  const row = [
    '577493', '待发货', 'TK-TAB', '1', '0', 'Faucet', '07/24/2026 8:10:09 PM',
    '商家物流', 'Summer Riggs', '+1 423 534 2901', 'US', 'TN', 'Jonesborough', '37659',
    '435 FOXGLOVE CT', '', '',
  ]
  const content = `${tiktokHeaders.join('\t')}\r\n${row.join('\t')}\r\n`
  const bytes = new TextEncoder().encode(content)
  const file = { name: 'TikTok待发货订单.unknown', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  const records = await readOrderFile(file, 'TIKTOK')
  assert.equal(records[0].sourceSku, 'TK-TAB')
  assert.equal(records[0].phone, '+1 423 534 2901')
})

test('TikTok order upload requires pending-shipment text in filename', async () => {
  const bytes = new TextEncoder().encode('content should not be parsed')
  const file = { name: 'TikTok订单.csv', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  await assert.rejects(() => readOrderFile(file, 'TIKTOK'), /文件名必须包含“待发货”/)
})

test('SKU mapping table import accepts aliases and removes identical duplicates', async () => {
  const content = 'Seller SKU,亚马逊 SKU,Non-Amazon SKU\r\nTK-A,AMZ-A,NON-A\r\nTK-A,AMZ-A,NON-A\r\nTK-B,AMZ-B,NON-B\r\n'
  const bytes = new TextEncoder().encode(content)
  const file = { name: 'mapping.table', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  const rows = await readSkuMappingFile(file)
  assert.deepEqual(rows.map(({ platform_sku, amazon_sku, non_amazon_sku }) => [platform_sku, amazon_sku, non_amazon_sku]), [
    ['TK-A', 'AMZ-A', 'NON-A'],
    ['TK-B', 'AMZ-B', 'NON-B'],
  ])
})

test('SKU mapping table import reads optional remarks', async () => {
  const content = '平台 SKU,Amazon SKU,非亚 SKU,备注\r\nTK-A,AMZ-A,NON-A,主推款\r\n'
  const bytes = new TextEncoder().encode(content)
  const file = { name: 'mapping.csv', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  const rows = await readSkuMappingFile(file)
  assert.equal(rows[0].remark, '主推款')
})

test('SKU mapping table import allows multiple Amazon SKUs for one platform SKU', async () => {
  const content = '平台 SKU,Amazon SKU,非亚 SKU\r\nTK-A,AMZ-A,NON-A\r\nTK-A,AMZ-B,NON-A\r\n'
  const bytes = new TextEncoder().encode(content)
  const file = { name: 'mapping.csv', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  const rows = await readSkuMappingFile(file)
  assert.deepEqual(rows.map((row) => [row.platform_sku, row.amazon_sku]), [['TK-A', 'AMZ-A'], ['TK-A', 'AMZ-B']])
})

test('SKU mapping table import requires non-Amazon SKU column', async () => {
  const content = '平台 SKU,Amazon SKU\r\nTK-A,AMZ-A\r\n'
  const bytes = new TextEncoder().encode(content)
  const file = { name: 'mapping.csv', size: bytes.byteLength, arrayBuffer: async () => bytes.buffer }
  await assert.rejects(() => readSkuMappingFile(file), /非亚 SKU/)
})
