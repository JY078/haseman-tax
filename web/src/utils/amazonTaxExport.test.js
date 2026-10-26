import * as Vue from 'vue'
import { renderToString } from '@vue/server-renderer'
import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { ref, reactive, computed } from 'vue'
import { isExcludedTaxTotalColumn } from './taxTotals.js'

const source = readFileSync(new URL('../components/AmazonTaxTables.vue', import.meta.url), 'utf8')
const { descriptor } = parse(source)
const script = compileScript(descriptor, { id: 'amazon-tax-export-test' }).content
const createComponent = new Function(
  'ref', 'reactive', 'onMounted', 'onBeforeUnmount', 'request',
  'FilterMenu', 'AmazonProductCategories', 'AmazonSalesVolumeCell', 'AmazonSummaryAmountCell', 'AmazonValidationDot', 'TaxFormulaHelp', 'TaxColumnConfigPanel', 'TaxDateRangePicker', 'computed', 'watch', 'isExcludedTaxTotalColumn',
  script.replace(/^import .*$/gm, '').replace('export default', 'return'),
)

function setup(overrides = {}, request = {}) {
  const calls = []
  const props = reactive({ downloadBusy: '', downloadError: '', storeId: 'Qfn', siteId: 'US', siteCurrencyCode: '', dateFrom: '', dateTo: '', downloadFile: (...args) => calls.push(args), ...overrides })
  const component = createComponent(ref, reactive, () => {}, () => {}, request, {}, {}, {}, {}, {}, {}, {}, {}, computed, () => {}, isExcludedTaxTotalColumn)
  return { state: component.setup(props, { expose() {} }), props, calls }
}

test('Amazon detail exports pass all selected periods without pagination', () => {
  const { state, calls } = setup()
  state.filters.month = '2026-07,2026-08'
  state.filters.quarter = '2026-Q3'
  state.page.value = 2
  for (const type of ['shipments', 'finance']) {
    state.tab.value = type
    state.exportReport()
    assert.deepEqual(calls.at(-1), [
      '/tax-data/amazon/export', 'export', `Amazon_${type}.csv`,
      { type, month: '2026-07,2026-08', quarter: '2026-Q3', store_id: 'Qfn', site_id: 'US', site_currency: 'USD' },
    ])
  }
  assert.equal(calls.length, 2)
})

test('Amazon summary exports use the selected currency for the selected store and site', () => {
  const { state, calls } = setup()
  state.tab.value = 'summary'
  state.groupByCountry.value = true
  for (const currency of ['USD', 'CNY']) {
    state.currency.value = currency
    state.exportReport()
    assert.deepEqual(calls.at(-1), [
      '/tax-data/amazon/export', 'export', 'Amazon_quarters.csv', { type: 'quarters', date_from: '', date_to: '', currency, store_id: 'Qfn', site_id: 'US', site_currency: 'USD' },
    ])
  }
})

test('Amazon quarter summary uses the configured site currency for sales, details and export', () => {
  const { state, calls } = setup({ siteId: 'SITE_EU', siteCurrencyCode: 'EUR' })
  state.tab.value = 'summary'
  assert.equal(state.siteCurrency.value, 'EUR')
  assert.equal(state.currency.value, 'EUR')
  assert.equal(state.formatSummary('12.50', 'sales'), '€12.50')
  state.summaryRows.value = [{ sales: 12.5 }, { sales: 7.5 }]
  assert.equal(state.amazonSummaryTotal('sales'), '20.00')
  assert.equal(state.amazonSummaryDetails({ sales_product: 12.5 }, 'sales')[0].value, 12.5)
  state.exportReport()
  assert.equal(calls[0][3].currency, 'EUR')
  assert.equal(calls[0][3].site_id, 'SITE_EU')
  assert.equal(calls[0][3].store_id, 'Qfn')
  assert.equal(calls[0][3].view, undefined)
})

test('Amazon summary exports cross-store quarters for the selected outer site', () => {
  const { state, calls } = setup({ storeId: '', siteId: 'CA', siteCurrencyCode: 'CAD' })
  state.tab.value = 'summary'
  state.groupByCountry.value = true
  assert.equal(state.siteCurrency.value, 'CAD')
  assert.equal(state.currency.value, 'CAD')
  state.exportReport()
  assert.deepEqual(calls.at(-1), [
    '/tax-data/amazon/export', 'export', 'Amazon_quarters.csv', { type: 'quarters', date_from: '', date_to: '', currency: 'CAD', store_id: '', site_id: 'CA', site_currency: 'CAD' },
  ])
})

test('Amazon selected-store summary aggregates quarter rows into country totals', () => {
  const { state } = setup({ siteId: '' })
  state.groupByCountry.value = true
  const row = state.aggregateCountrySummary({ value: 'US', label: '美国' }, [
    { sales_volume: 2, sales: '10.25', sales_product: '8', sales_categories: [{ category: '厨具', quantity: 1 }], months: [{ month: '2026-01', sales_volume: 1, sales: '4.25', sales_product: '4', sales_categories: [{ category: '厨具', quantity: 1 }] }] },
    { sales_volume: 3, sales: '5.75', sales_product: '4', sales_categories: [{ category: '厨具', quantity: 2 }], months: [{ month: '2026-04', sales_volume: 2, sales: '5.75', sales_product: '4', sales_categories: [{ category: '厨具', quantity: 2 }] }] },
  ])
  assert.equal(row.country, '美国')
  assert.equal(row.sales_volume, 5)
  assert.equal(row.sales, 16)
  assert.equal(row.sales_product, 12)
  assert.deepEqual(row.sales_categories, [{ category: '厨具', quantity: 3 }])
  assert.deepEqual(row.months.map(month => [month.month, month.sales_volume, month.sales]), [['2026-01', 1, 4.25], ['2026-04', 2, 5.75]])
  state.summaryCountryRows.value = [row]
  assert.deepEqual(state.displaySummaryRows.value, [row])
})

test('Amazon selected-store summary shows every country in CNY when the outer country filter is all', async () => {
  const calls = []
  const { state, calls: exports } = setup({
    storeId: 'Qfn',
    siteId: '',
    countryOptions: [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }],
  }, {
    get: async (url, options) => {
      calls.push({ url, params: options.params })
      const amount = options.params.site_id === 'US' ? '12.50' : '25.00'
      return { data: { items: [{ sales_volume: 2, sales: amount, sales_product: amount, sales_categories: [] }] } }
    },
  })
  state.tab.value = 'summary'
  state.groupByCountry.value = true
  assert.equal(state.summarySiteId.value, '')
  await state.loadRows()
  assert.deepEqual(calls.map(call => [call.params.site_id, call.params.currency, call.params.store_id]), [['US', 'CNY', 'Qfn'], ['CA', 'CNY', 'Qfn']])
  assert.deepEqual(state.displaySummaryRows.value.map(row => [row.site_id, row.country, row.sales_volume, row.sales]), [['US', '美国', 2, 12.5], ['CA', '加拿大', 2, 25]])
  state.exportReport()
  assert.deepEqual(exports.at(-1)[3], { type: 'quarters', view: 'country', date_from: '', date_to: '', currency: 'CNY', store_id: 'Qfn', site_id: '', site_currency: 'USD' })
})

test('Amazon selected-store summary only loads the explicitly selected country', async () => {
  const calls = []
  const { state } = setup({
    storeId: 'Qfn',
    siteId: 'CA',
    siteCurrencyCode: 'CAD',
    countryOptions: [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }],
  }, {
    get: async (url, options) => {
      calls.push(options.params.site_id)
      return { data: { items: [{ sales_volume: 1, sales: '15', months: [] }] } }
    },
  })
  state.tab.value = 'summary'
  state.groupByCountry.value = true
  await state.loadRows()
  assert.deepEqual(calls, ['CA', 'CA'])
  assert.equal(state.canGroupByCountry.value, false)
  assert.equal(state.summaryCountryMode.value, false)
  assert.equal(state.summaryRows.value.length, 1)
})

test('Amazon all-countries summary defaults to CNY', () => {
  const { state } = setup({ storeId: '', siteId: '', siteCurrencyCode: '' })
  assert.equal(state.allCountries.value, true)
  assert.equal(state.currency.value, 'CNY')
})

test('Amazon export respects shared busy state and excludes product categories', () => {
  const { state, props, calls } = setup()
  for (const busy of ['export', 'downloads-1']) {
    props.downloadBusy = busy
    state.exportReport()
  }
  props.downloadBusy = ''
  state.tab.value = 'categories'
  state.exportReport()
  assert.equal(calls.length, 0)
  state.tab.value = 'shipments'
  state.exportReport()
  assert.equal(calls.length, 1)
})

test('Amazon footer leaves excluded and nonnumeric columns blank, quantities have no currency', () => {
  const { state } = setup()
  state.rawTotals.value = { 'Settlement ID': 12345, '亚马逊订单商品编号': 12345, '卖家订单编号': 12345, 'quantity': 12, 'total': 10 }
  assert.equal(state.amazonRawTotal('Settlement ID'), '')
  assert.equal(state.amazonRawTotal('亚马逊订单商品编号'), '')
  assert.equal(state.amazonRawTotal('卖家订单编号'), '')
  assert.equal(state.amazonRawTotal('description'), '')
  assert.equal(state.amazonRawTotal('quantity'), '12')
  assert.equal(state.amazonRawTotal('total'), '10.00')
})

test('Amazon delivery details include both Fulfilment by Amazon fees', () => {
  const { state } = setup()
  const details = state.amazonSummaryDetails({ delivery_fba_fees: -13139.94, delivery_fulfilment_disposal: -24.91, delivery_transaction_other: -463.35, delivery_fulfilment_storage: -305.87 }, 'delivery')
  assert.equal(details.length, 4)
  assert.equal(Math.round(details.reduce((sum, item) => sum + item.value, 0) * 100), -1393407)
  assert.deepEqual(details.at(3), { label: 'FBA库存仓储费', value: -305.87 })
})

test('Amazon storage details split transaction and inventory fee sources', () => {
  const { state } = setup()
  const details = state.amazonSummaryDetails({ storage_transaction: -500, storage_inventory: -88.13 }, 'storage')
  assert.deepEqual(details, [{ label: 'FBA Transaction Fees 仓储类', value: -500 }, { label: 'FBA Inventory Fee', value: -88.13 }])
})



test('Amazon selected store defaults to quarters and toggles country grouping with matching exports', async () => {
  const requests = []
  const quarterRows = [
    { quarter: '2026-Q1', sales: '10', months: [{ month: '2026-01', sales: '10' }] },
    { quarter: '2026-Q2', sales: '20', months: [{ month: '2026-04', sales: '20' }] },
  ]
  const { state, calls } = setup({ siteId: '', dateFrom: '2026-01-01', dateTo: '2026-06-30', countryOptions: [{ value: 'US', label: '美国', currency: 'USD' }] }, {
    get: async (url, options) => { requests.push(options.params); return { data: { items: quarterRows } } },
  })
  state.tab.value = 'summary'
  assert.equal(state.groupByCountry.value, false)
  assert.equal(state.summaryCountryMode.value, false)
  await state.loadRows()
  assert.deepEqual(state.displaySummaryRows.value.map(row => row.quarter), ['2026-Q1', '2026-Q2'])
  assert.equal(state.summaryRowKey(state.displaySummaryRows.value[0]), '2026-Q1')
  state.toggleMonthDetail('2026-Q1')
  state.exportReport()
  assert.equal(calls.at(-1)[2], 'Amazon_quarters.csv')
  assert.equal(calls.at(-1)[3].view, undefined)
  assert.equal(calls.at(-1)[3].date_from, '2026-01-01')
  state.groupByCountry.value = true
  await state.switchSummaryGrouping()
  assert.equal(state.summaryCountryMode.value, true)
  assert.deepEqual(Object.keys(state.expandedMonths), [])
  assert.equal(state.displaySummaryRows.value.length, 1)
  assert.equal(state.displaySummaryRows.value[0].country, '美国')
  assert.equal(state.displaySummaryRows.value[0].sales, 30)
  assert.equal(state.displaySummaryRows.value[0].months.length, 2)
  state.exportReport()
  assert.equal(calls.at(-1)[2], 'Amazon_countries.csv')
  assert.equal(calls.at(-1)[3].view, 'country')
  state.groupByCountry.value = false
  await state.switchSummaryGrouping()
  assert.deepEqual(state.displaySummaryRows.value.map(row => row.quarter), ['2026-Q1', '2026-Q2'])
  assert.equal(state.summaryCountryRows.value.length, 0)
  assert.equal(requests.length, 3)
  assert.deepEqual(requests.map(params => params.site_id), ['', 'US', ''])
  for (const params of requests) {
    assert.equal(params.store_id, 'Qfn')
    assert.equal(params.date_from, '2026-01-01')
    assert.equal(params.date_to, '2026-06-30')
  }
})

test('Amazon country grouping requires a selected store', () => {
  const { state } = setup({ storeId: '' })
  state.groupByCountry.value = true
  assert.equal(state.summaryCountryMode.value, false)
})


test('Amazon summary controls follow store and country filter visibility', async () => {
  const toolbar = descriptor.template.content.split('<div v-if="tab === ')[1]
  const toolbarSource = '<div v-if="tab === ' + toolbar.split('<p v-if="error"')[0]
  const compiled = compileTemplate({ source: toolbarSource, filename: 'SummaryToolbar.vue', id: 'summary-controls-test' })
  assert.deepEqual(compiled.errors, [])
  const render = new Function('Vue', compiled.code.replace(/import {([^}]+)} from .vue./g, (_, names) => 'const {' + names.replace(/ as /g, ': ') + '} = Vue').replace('export function render', 'return function render'))(Vue)
  for (const [storeId, siteId, checkbox, currencyButtons] of [
    ['', '', false, false], ['Qfn', '', true, false], ['', 'US', false, true], ['Qfn', 'US', false, true],
  ]) {
    const { state } = setup({ storeId, siteId })
    state.tab.value = 'summary'
    state.groupByCountry.value = true
    const html = await renderToString(Vue.createSSRApp({ render, setup: () => Object.fromEntries(['tab', 'summaryCountryMode', 'allCountries', 'canGroupByCountry', 'summarySiteId', 'groupByCountry', 'switchSummaryGrouping', 'currency', 'summarySiteCurrency', 'summarySiteCurrencyLabel', 'switchCurrency'].map(key => [key, state[key]])) }))
    assert.equal(html.includes('type="checkbox"'), checkbox)
    assert.equal(html.includes('currency-segment'), currencyButtons)
    assert.equal(state.summaryCountryMode.value, checkbox)
  }
})

test('Selecting a country disables country grouping and exports quarters even if previously checked', () => {
  const { state, props, calls } = setup({ siteId: '' })
  state.tab.value = 'summary'
  state.groupByCountry.value = true
  assert.equal(state.summaryCountryMode.value, true)
  props.siteId = 'CA'
  assert.equal(state.canGroupByCountry.value, false)
  assert.equal(state.summaryCountryMode.value, false)
  state.exportReport()
  assert.equal(calls.at(-1)[2], 'Amazon_quarters.csv')
  assert.equal(calls.at(-1)[3].view, undefined)
  assert.equal(calls.at(-1)[3].site_id, 'CA')
})


test('Amazon average rates match quarters and months by key with four decimals', async () => {
  const calls = []
  const local = [
    { quarter: '2026-Q1', taxable_income: '100', months: [{ month: '2026-01', taxable_income: '40' }, { month: '2026-02', taxable_income: '60' }] },
    { quarter: '2026-Q2', taxable_income: '300', months: [{ month: '2026-04', taxable_income: '300' }] },
  ]
  const cny = [
    { quarter: '2026-Q2', taxable_income: '2100', months: [{ month: '2026-04', taxable_income: '2100' }] },
    { quarter: '2026-Q1', taxable_income: '712.345', months: [{ month: '2026-02', taxable_income: '432.345' }, { month: '2026-01', taxable_income: '280' }] },
  ]
  const { state } = setup({ dateFrom: '2026-01-01', dateTo: '2026-06-30' }, { get: async (_, options) => { calls.push(options.params); return { data: { items: options.params.currency === 'CNY' ? cny : local } } } })
  state.tab.value = 'summary'
  await state.loadRows()
  assert.deepEqual(calls.map(call => call.currency), ['USD', 'CNY'])
  for (const call of calls) { assert.equal(call.store_id, 'Qfn'); assert.equal(call.site_id, 'US'); assert.equal(call.date_from, '2026-01-01'); assert.equal(call.date_to, '2026-06-30') }
  assert.deepEqual(state.displayedSummaryColumns.value.slice(-2).map(column => column.key), ['taxable_income', 'average_rate'])
  assert.deepEqual(state.summaryRows.value.map(state.averageRate), ['7.1235', '7.0000'])
  assert.deepEqual(state.summaryRows.value[0].months.map(state.averageRate), ['7.0000', '7.2058'])
  assert.equal(state.amazonSummaryTotal('average_rate'), '7.0309')
  assert.equal(state.formatAverageRate('0', '0'), '—')
  assert.equal(state.formatAverageRate(null, '100'), '待换算')
  assert.equal(state.formatAverageRate('-700', '-100'), '7.0000')
  const missing = state.attachCnyIncome(local, [])
  assert.equal(state.averageRate(missing[0]), '待换算')
  assert.equal(state.averageRate(missing[0].months[0]), '待换算')
})

test('Amazon average rate is visible only for a selected country in its local currency', () => {
  const { state, props } = setup()
  assert.equal(state.showAverageRate.value, true)
  state.currency.value = 'CNY'
  assert.equal(state.showAverageRate.value, false)
  assert.equal(state.displayedSummaryColumns.value.at(-1).key, 'taxable_income')
  props.siteId = 'CA'; props.siteCurrencyCode = 'CAD'; state.currency.value = 'CAD'
  assert.equal(state.showAverageRate.value, true)
  props.siteId = ''
  assert.equal(state.showAverageRate.value, false)
})


test('Amazon service placement and premium fees appear only in other expense details', () => {
  const { state } = setup()
  const row = { other_inbound_placement: -12.34, other_premium_services: -5.67 }
  assert.deepEqual(state.amazonSummaryDetails(row, 'other').slice(-2), [
    { label: 'FBA入库配置费', value: -12.34 },
    { label: 'Premium Services服务费', value: -5.67 },
  ])
  const country = state.aggregateCountrySummary({ value: 'US', label: '美国' }, [row, row])
  assert.equal(country.other_inbound_placement, -24.68)
  assert.equal(country.other_premium_services, -11.34)
})
