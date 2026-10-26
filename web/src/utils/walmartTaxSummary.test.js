import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import { renderToString } from '@vue/server-renderer'
import { isExcludedTaxTotalColumn } from './taxTotals.js'

const { descriptor } = parse(readFileSync(new URL('../views/TaxDataView.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'walmart-summary-test' })
const imports = [...script.content.matchAll(/^import ([A-Za-z]+) from .*$/gm)].map(match => match[1])
const stub = { setup: (_, { slots }) => () => slots.default?.({}) }
const create = new Function('Vue', ...imports, 'isExcludedTaxTotalColumn', script.content
  .replace(/^import {([^}]+)} from 'vue'$/gm, 'const {$1} = Vue')
  .replace(/^import {([^}]+)} from '@headlessui.vue'$/gm, 'const {$1} = new Proxy({}, { get: () => ({render: () => null}) })')
  .replace(/^import .*$/gm, '').replace('export default', 'return'))
const template = compileTemplate({ source: descriptor.template.content, filename: 'TaxDataView.vue', id: 'walmart-summary-test', compilerOptions: { bindingMetadata: script.bindings } })
const render = new Function('Vue', template.code.replace(/import {([^}]+)} from .vue./g, (_, names) => 'const {' + names.replace(/ as /g, ': ') + '} = Vue').replace('export function render', 'return function render'))(Vue)
function component(request = { get: async () => ({ data: { items: [] } }) }) {
  return create(Vue, ...imports.map(name => name === 'request' ? request : stub), isExcludedTaxTotalColumn)
}
const rows = [
  { quarter: '2025-Q2', usd_totals: { 销售额: '100', 退款: '-10', 佣金: '-5', 配送费: '-15' }, usd_taxable_income: '90', sales_volume: '2' },
  { quarter: '2026-Q3', usd_totals: { 销售额: '200', 退款: '-20', 佣金: '-8', 配送费: '-22' }, usd_taxable_income: '180', sales_volume: '3' },
]

test('Walmart renders each quarter expense and income under the correct columns', async () => {
  const original = component()
  const app = Vue.createSSRApp({ ...original, render, setup(props, context) {
    const state = original.setup(props, context)
    state.tab.value = 'quarters'; state.quarters.value = rows
    state.categories.value = ['销售额', '退款', '佣金', '配送费']
    return state
  } })
  const html = await renderToString(app)
  const body = html.split('<tbody>')[1].split('</tbody>')[0]
  const cells = [...body.matchAll(new RegExp('<tr[^>]*>(.*?)</tr>', 'gs'))].map(row => [...row[1].matchAll(new RegExp('<td[^>]*>(.*?)</td>', 'gs'))].map(cell => cell[1].replace(/<[^>]*>/g, '').trim()))
  assert.deepEqual(cells.map(row => row.slice(-2)), [['$-20.00', '$90.00'], ['$-30.00', '$180.00']])
  const footer = html.split('<tfoot>')[1].split('</tfoot>')[0]
  const totals = [...footer.matchAll(new RegExp('<th[^>]*>(.*?)</th>', 'gs'))].map(cell => cell[1].replace(/<[^>]*>/g, '').trim())
  assert.deepEqual(totals.slice(-2), ['-50.00', '$270.00'])
})

test('Walmart summary requests do not inherit a raw-ledger quarter filter', async () => {
  const calls = []
  const original = component({ get: async (url, options) => { calls.push({ url, params: options.params }); return { data: { items: rows } } } })
  let state
  await renderToString(Vue.createSSRApp({ setup(_, context) { state = original.setup({}, context); return () => null } }))
  state.tab.value = 'quarters'; state.quarter.value = '2026-Q3'; state.storeId.value = 'TVOFEK'
  await state.loadData()
  assert.equal(calls[0].params.store_id, 'TVOFEK')
  assert.equal(calls[0].params.quarter, undefined)
  assert.equal(state.filteredQuarters.value.length, 2)
})
test('Walmart export sends the displayed currency and summary date range', async () => {
  for (const currency of ['USD', 'CNY']) {
    const calls = []
    const original = component({ get: (url, options) => { calls.push({ url, ...options }); return new Promise(() => {}) } })
    let state
    await renderToString(Vue.createSSRApp({ setup(_, context) { state = original.setup({}, context); return () => null } }))
    state.tab.value = 'quarters'
    state.storeId.value = 'TVOFEK'
    state.countryId.value = 'US'
    state.summaryCurrency.value = currency
    state.dateFrom.value = '2026-01-01'
    state.dateTo.value = '2026-03-31'
    state.exportReport()
    assert.equal(calls.length, 1)
    assert.equal(calls[0].params.currency, currency)
    assert.equal(calls[0].params.store_id, 'TVOFEK')
    assert.equal(calls[0].params.site_id, 'US')
    assert.equal(calls[0].params.date_from, '2026-01-01')
    assert.equal(calls[0].params.date_to, '2026-03-31')
    assert.equal(calls[0].responseType, 'blob')
  }
})
