<template>
  <div class="amazon-view">
    <div class="toolbar">
      <div class="tabs"><button v-for="item in tabs" :key="item.key" :class="{ active: tab === item.key }"
          @click="switchTab(item.key)">{{ item.label }}</button></div>
      <div class="toolbar-actions"><button v-if="tab !== 'summary' && tab !== 'categories'"
          class="column-config-trigger" @click="columnConfigOpen = true">列配置</button><button @click="loadRows"
          :disabled="loading">刷新</button><button v-if="tab !== 'categories'" class="export-btn" @click="exportReport"
          :disabled="loading || !!downloadBusy">{{ downloadBusy === 'export' ? '生成中…' : '下载报表' }}</button></div>
    </div>
    <TaxColumnConfigPanel :open="columnConfigOpen" :headers="columns" :visible="visibleColumns" :pinned="pinnedColumns"
      :label="amazonHeaderLabel" @cancel="columnConfigOpen = false" @apply="applyColumnConfig" />
    <div v-if="tab === 'summary'" class="summary-toolbar">
      <div><strong>季度汇总</strong><span>{{ summaryCountryMode ? allCountries ? '按国家查看当前店铺数据，金额按人民币汇总' : '按日期范围查看当前店铺国家数据'
        : allCountries ? '全部国家数据按人民币汇总' : '按日期范围查看当前店铺与国家数据' }}</span></div>
      <div v-if="canGroupByCountry || summarySiteId" class="summary-controls"><label v-if="canGroupByCountry"
          class="country-summary-toggle"><input v-model="groupByCountry" type="checkbox"
            @change="switchSummaryGrouping" />按国家汇总</label>
        <div v-if="summarySiteId" class="currency-segment"><button :class="{ active: currency === summarySiteCurrency }"
            @click="switchCurrency(summarySiteCurrency)">{{ summarySiteCurrencyLabel }}</button><button
            :class="{ active: currency === 'CNY' }" @click="switchCurrency('CNY')">人民币</button></div>
      </div>
    </div>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p v-if="downloadError" class="error" role="alert">{{ downloadError }}</p>
    <AmazonProductCategories v-if="tab === 'categories'" ref="categoriesTable" />
    <div v-else class="table-region" :aria-busy="loading">
      <div class="table-wrap" tabindex="0" aria-label="Amazon 报税数据表">

        <table v-if="tab !== 'summary'" class="data-table" :style="{ width: `${176 + displayColumns.length * 180}px` }">
          <colgroup>
            <col style="width:86px" />
            <col style="width:90px" />
            <col v-for="column in displayColumns" :key="column" style="width:180px" />
          </colgroup>
          <thead>
            <tr>
              <th class="sticky month">
                <FilterMenu label="月份" :values="options.month" :selected="filters.month"
                  @toggle="toggleValue('month', $event)" />
              </th>
              <th class="sticky quarter">
                <FilterMenu label="季度" :values="options.quarter" :selected="filters.quarter"
                  @toggle="toggleValue('quarter', $event)" />
              </th>
              <th v-for="column in displayColumns" :key="column" :class="amazonColumnClass(column)"
                :style="amazonColumnStyle(column)" :title="amazonFinanceHeaderLabel(column)">{{
                amazonFinanceHeaderLabel(column) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id">
              <td class="sticky month">{{ row.month }}</td>
              <td class="sticky quarter">{{ row.quarter }}</td>
              <td v-for="column in displayColumns" :key="column" :class="amazonColumnClass(column)"
                :style="amazonColumnStyle(column)" :title="row.data?.[column]">{{ row.data?.[column] || '' }}</td>
            </tr>
            <tr v-if="!loading && !error && !rows.length">
              <td :colspan="displayColumns.length + 2" class="empty">暂无数据</td>
            </tr>
          </tbody>
          <tfoot v-if="rows.length">
            <tr>
              <th class="sticky month">总计</th>
              <th class="sticky quarter"></th>
              <th v-for="column in displayColumns" :key="column" class="numeric" :class="amazonColumnClass(column)"
                :style="amazonColumnStyle(column)">{{ amazonRawTotal(column) }}</th>
            </tr>
          </tfoot>
        </table>
        <table v-else class="data-table summary">
          <thead>
            <tr>
              <th>{{ summaryCountryMode ? '国家' : '季度' }}</th>
              <th v-for="column in displayedSummaryColumns" :key="column.key" class="numeric"><span
                  class="summary-header-label">{{ column.label }}
                  <TaxFormulaHelp v-if="column.formula" :label="column.label" :formula="column.formula" />
                </span></th>
            </tr>
          </thead>
          <tbody><template v-for="row in displaySummaryRows" :key="summaryRowKey(row)">
              <tr>
                <td><button v-if="row.months?.length" type="button" class="month-expander"
                    :aria-expanded="!!expandedMonths[summaryRowKey(row)]"
                    :aria-label="`${expandedMonths[summaryRowKey(row)] ? '收起' : '展开'} ${summaryCountryMode ? row.country : row.quarter} 月度明细`"
                    @click="toggleMonthDetail(summaryRowKey(row))"><span aria-hidden="true"></span></button><span>{{
                      summaryCountryMode ? row.country : row.quarter }}</span>
                  <AmazonValidationDot v-if="row.validation_errors?.length" :messages="row.validation_errors" />
                </td>
                <td v-for="column in displayedSummaryColumns" :key="column.key" class="numeric">
                  <AmazonSalesVolumeCell v-if="column.key === 'sales_volume'" :row="row" /><span
                    v-else-if="column.key === 'expense_total'">{{ formatSummary(amazonExpenseValue(row), column.key)
                    }}</span><span v-else-if="column.key === 'average_rate'">{{ averageRate(row) }}</span>
                  <AmazonSummaryAmountCell v-else :value="row[column.key]" :currency="currency"
                    :details="amazonSummaryDetails(row, column.key)" />
                </td>
              </tr><template v-if="expandedMonths[summaryRowKey(row)]">
                <tr v-for="month in row.months" :key="`${summaryRowKey(row)}-${month.month}`" class="month-detail-row">
                  <td class="month-detail-period-cell"><span class="month-detail-indent" aria-hidden="true"></span>{{
                    month.month }}</td>
                  <td v-for="column in displayedSummaryColumns" :key="column.key" class="numeric">
                    <AmazonSalesVolumeCell v-if="column.key === 'sales_volume'" :row="month" /><span
                      v-else-if="column.key === 'expense_total'">{{ formatSummary(amazonExpenseValue(month), column.key)
                      }}</span><span v-else-if="column.key === 'average_rate'">{{ averageRate(month) }}</span>
                    <AmazonSummaryAmountCell v-else :value="month[column.key]" :currency="currency"
                      :details="amazonSummaryDetails(month, column.key)" />
                  </td>
                </tr>
              </template>
            </template>
            <tr v-if="!loading && !error && !displaySummaryRows.length">
              <td :colspan="displayedSummaryColumns.length + 1" class="empty">暂无数据</td>
            </tr>
          </tbody>
          <tfoot v-if="displaySummaryRows.length">
            <tr>
              <th>总计</th>
              <th v-for="column in displayedSummaryColumns" :key="column.key" class="numeric">{{
                amazonSummaryTotal(column.key) }}
              </th>
            </tr>
          </tfoot>
        </table>
      </div>
      <div v-if="loading" class="loading" role="status"><span>正在加载数据…</span></div>
    </div>
    <div v-if="tab !== 'summary' && tab !== 'categories'" class="pager"><span class="row-count">共 {{ total }}
        条</span><button :disabled="page <= 1 || loading" @click="page--; loadRows()">上一页</button><span>{{ page }} / {{
          Math.max(1, Math.ceil(total / pageSize)) }}</span><label class="page-size-control">每页 <select
          v-model.number="pageSize" :disabled="loading" @change="page = 1; loadRows()">
          <option v-for="size in [25, 50, 100, 200]" :key="size" :value="size">{{ size }}</option>
        </select> 条</label><button :disabled="page * pageSize >= total || loading" @click="page++; loadRows()">下一页</button>
    </div>
  </div>
</template>
<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import request from '@/utils/request'
import { isExcludedTaxTotalColumn } from '@/utils/taxTotals'
import { amazonFinanceHeaderLabel } from '@/utils/amazonFinanceHeaders'
import FilterMenu from './AmazonTaxFilterMenu.vue'
import AmazonProductCategories from './AmazonProductCategories.vue'
import AmazonSalesVolumeCell from './AmazonSalesVolumeCell.vue'
import AmazonSummaryAmountCell from './AmazonSummaryAmountCell.vue'
import AmazonValidationDot from './AmazonValidationDot.vue'
import TaxFormulaHelp from './TaxFormulaHelp.vue'
import TaxColumnConfigPanel from './TaxColumnConfigPanel.vue'
const categoriesTable = ref(null)
const props = defineProps({
  downloadBusy: { type: String, default: '' },
  downloadError: { type: String, default: '' },
  downloadFile: { type: Function, required: true },
  storeId: { type: String, default: 'sunoutly' },
  siteId: { type: String, default: 'US' },
  siteCurrencyCode: { type: String, default: '' },
  countryOptions: { type: Array, default: () => [] },
  dateFrom: { type: String, default: '' },
  dateTo: { type: String, default: '' },
})
const tabs = [{ key: 'shipments', label: '亚马逊配送货件' }, { key: 'finance', label: '付款报告_联合报告_交易' }, { key: 'summary', label: '季度汇总' }, { key: 'categories', label: '商品分类' }]
const defaultSiteCurrency = site => String(site || 'US').toUpperCase() === 'CA' ? 'CAD' : 'USD'
const tab = ref('shipments'), currency = ref(props.siteId ? (props.siteCurrencyCode || defaultSiteCurrency(props.siteId)) : 'CNY'), rows = ref([]), columns = ref([]), visibleColumns = ref([]), pinnedColumns = ref([]), columnConfigOpen = ref(false), total = ref(0), page = ref(1), pageSize = ref(50), loading = ref(false), error = ref(''), rawTotals = ref({})
const displayColumns = computed(() => [...pinnedColumns.value.filter(column => visibleColumns.value.includes(column)), ...columns.value.filter(column => visibleColumns.value.includes(column) && !pinnedColumns.value.includes(column))])
const amazonColumnClass = column => pinnedColumns.value.includes(column) ? 'amazon-pinned' : ''
const amazonColumnStyle = column => { const index = pinnedColumns.value.indexOf(column); return index < 0 ? undefined : { left: `${176 + index * 180}px` } }
const siteCurrency = computed(() => props.siteCurrencyCode || defaultSiteCurrency(props.siteId))
const siteCurrencyLabel = computed(() => ({ USD: '美元', CAD: '加拿大元', EUR: '欧元', GBP: '英镑', JPY: '日元', AUD: '澳元' }[siteCurrency.value] || siteCurrency.value))
const groupByCountry = ref(false)
const canGroupByCountry = computed(() => Boolean(props.storeId) && !props.siteId)
const summaryCountryMode = computed(() => canGroupByCountry.value && groupByCountry.value)
const allCountries = computed(() => !props.siteId)
const summarySiteId = computed(() => props.siteId)
const summarySiteCurrency = computed(() => props.siteCurrencyCode || defaultSiteCurrency(props.siteId || 'US'))
const summarySiteCurrencyLabel = computed(() => ({ USD: '美元', CAD: '加拿大元', EUR: '欧元', GBP: '英镑', JPY: '日元', AUD: '澳元' }[summarySiteCurrency.value] || summarySiteCurrency.value))
let requestId = 0
const filters = reactive({ month: '', quarter: '' }), options = reactive({ month: [], quarter: [] })
const summaryColumns = [{ key: 'sales_volume', label: '销量' }, { key: 'sales', label: '销售额', formula: '商品价格 + 商品税 + 运费 + 运费税 + 礼品包装价格 + 礼品包装税费 + 商品促销折扣 + 货件促销折扣' }, { key: 'refund', label: '退货款', formula: '退款类交易的退款字段合计' }, { key: 'taxable_income', label: '净收入', formula: '销售额 + 退货款' }, { key: 'tax', label: '税费', formula: '全部财务交易的 marketplace withheld tax 合计' }, { key: 'delivery', label: '配送费', formula: '订单/退款FBA配送费 + FBA移除订单弃置费 + FBA Transaction Fees 除去 Inventory Storage Fee、Long Term Storage Fee、Removal Order Disposal Fee 的其他所有数据（取 fba fees + other 两列合计）+ Fulfilment by Amazon (FBA) Inventory Storage Fee（取 fba fees）+ Service Fee 中 FBA Inbound Placement Service Fee（取 other）' }, { key: 'selling', label: '销售费', formula: '订单/退款类交易的 selling fees 合计' }, { key: 'ads', label: '广告费', formula: 'Service Fee 交易的 other transaction fees 合计' }, { key: 'storage', label: '仓储费', formula: 'FBA Inventory Fee 全部 + FBA Transaction Fees 中 Inventory Storage Fee、Long Term Storage Fee、Removal Order Disposal Fee（均取 fba fees + other 两列合计）' }, { key: 'other', label: '其他费用', formula: '其他交易 total - marketplace withheld tax；已排除 FBA Transaction Fees、FBA Inventory Fee、仓储与弃置相关 Fulfilment by Amazon 费用；订阅服务费使用 other - marketplace withheld tax；Service Fee 中 FBA Inbound Placement Service Fee 取 other，Premium Services Fee 取 selling fees' }, { key: 'expense_total', label: '总费用', formula: '除销售额、广告费、退货款' }], summaryRows = ref([]), summaryCountryRows = ref([])
const showAverageRate = computed(() => Boolean(props.siteId) && currency.value !== 'CNY' && currency.value === summarySiteCurrency.value)
const displayedSummaryColumns = computed(() => showAverageRate.value ? [...summaryColumns, { key: 'average_rate', label: '平均汇率', formula: `人民币净收入 ÷ ${summarySiteCurrencyLabel.value}净收入` }] : summaryColumns)
function formatAverageRate(cny, local) {
  if (cny == null || local == null || cny === '' || local === '') return '待换算'
  const numerator = Number(cny), denominator = Number(local)
  if (!Number.isFinite(numerator) || !Number.isFinite(denominator)) return '待换算'
  return denominator === 0 ? '—' : (numerator / denominator).toFixed(4)
}
const averageRate = row => formatAverageRate(row.cny_taxable_income, row.taxable_income)
function attachCnyIncome(rows, cnyRows) {
  const quarters = new Map(cnyRows.map(row => [row.quarter, row]))
  return rows.map(row => {
    const cny = quarters.get(row.quarter)
    const months = new Map((cny?.months || []).map(month => [month.month, month]))
    return { ...row, cny_taxable_income: cny?.taxable_income, months: (row.months || []).map(month => ({ ...month, cny_taxable_income: months.get(month.month)?.taxable_income })) }
  })
}
const expandedMonths = reactive({})
function toggleMonthDetail(quarter) { expandedMonths[quarter] = !expandedMonths[quarter] }
const amazonSummaryDetails = (row, key) => ({
  sales: [['商品价格', row.sales_product], ['商品税', row.sales_product_tax], ['运费', row.sales_shipping], ['运费税', row.sales_shipping_tax], ['礼品包装价格', row.sales_gift_wrap], ['礼品包装税费', row.sales_gift_wrap_tax], ['商品促销折扣', row.sales_promotion], ['货件促销折扣', row.sales_shipment_promotion]],
  refund: [['商品销售额', row.refund_product_sales], ['商品销售税', row.refund_product_sales_tax], ['运费抵扣', row.refund_shipping_credits], ['运费抵扣税', row.refund_shipping_tax], ['礼品包装抵扣', row.refund_gift_wrap], ['礼品包装抵扣税', row.refund_gift_wrap_tax], ['监管费', row.refund_regulatory_fee], ['监管费税', row.refund_regulatory_tax], ['促销返利', row.refund_promotions], ['促销返利税', row.refund_promotions_tax], ['其他退款', row.refund_other]],
  tax: [['平台代扣税', row.tax_marketplace_withheld]],
  delivery: [['订单/退款FBA配送费', row.delivery_fba_fees], ['FBA移除订单弃置费', row.delivery_fulfilment_disposal], ['其余FBA Transaction Fees', row.delivery_transaction_other], ['FBA库存仓储费', row.delivery_fulfilment_storage]],
  selling: [['销售服务费', row.selling_fees]],
  ads: [['其他交易费用', row.ads_other_transaction_fees]],
  storage: [['FBA Transaction Fees 仓储类', row.storage_transaction], ['FBA Inventory Fee', row.storage_inventory]],
  other: [['其他交易总额', row.other_total], ['平台代扣税（扣除项）', row.other_marketplace_withheld], ['订阅服务费其他金额', row.other_subscription], ['FBA入库配置费', row.other_inbound_placement], ['Premium Services服务费', row.other_premium_services]],
  taxable_income: [['销售额 + 退货款', row.taxable_income]],
}[key]?.map(([label, value]) => ({ label, value })) || [])
const amazonRawTotal = column => {
  if (isExcludedTaxTotalColumn(column) || !(column in rawTotals.value)) return ''
  const value = Number(rawTotals.value[column] || 0)
  if (/quantity|数量/i.test(String(column))) return value.toLocaleString('en-US', { maximumFractionDigits: 2 })
  return value === 0 ? '-' : value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
const amazonExpenseKeys = summaryColumns.filter(column => !['sales_volume', 'sales', 'refund', 'ads', 'taxable_income'].includes(column.key)).map(column => column.key)
const amazonExpenseValue = row => amazonExpenseKeys.reduce((sum, key) => sum + numericValue(row[key]), 0)
const displaySummaryRows = computed(() => summaryCountryMode.value ? summaryCountryRows.value : summaryRows.value)
const summaryRowKey = row => summaryCountryMode.value ? row.site_id : row.quarter
const amazonSummaryTotal = key => {
  if (key === 'average_rate') {
    const rows = displaySummaryRows.value
    if (rows.some(row => row.cny_taxable_income == null || row.taxable_income == null)) return '待换算'
    return formatAverageRate(rows.reduce((sum, row) => sum + Number(row.cny_taxable_income), 0), rows.reduce((sum, row) => sum + Number(row.taxable_income), 0))
  }
  if (key === 'sales_volume') return `${displaySummaryRows.value.reduce((total, row) => total + Number(row.sales_volume || 0), 0).toLocaleString('zh-CN')}件`
  if (summaryCountryMode.value && displaySummaryRows.value.some(row => key === 'expense_total' ? amazonExpenseKeys.some(expenseKey => row[expenseKey] == null) : row[key] == null)) return '待换算'
  return totalAmount(key === 'expense_total' ? displaySummaryRows.value.reduce((total, row) => total + amazonExpenseValue(row), 0) : displaySummaryRows.value.reduce((total, row) => total + Number(row[key] || 0), 0))
}
const totalAmount = value => value == null ? '待换算' : Number(value) === 0 ? '-' : Number(value).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const numericValue = value => { if (value == null || value === '') return 0; const normalized = String(value).replace(/[$,\s]/g, '').replace(/^\((.*)\)$/, '-$1'); const result = Number(normalized); return Number.isFinite(result) ? result : 0 }
const formatSummary = (value, key) => {
  if (key === 'sales_volume') return `${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}件`
  if (value == null) return '待换算'
  if (Number(value) === 0) return '-'
  return currencySymbol(currency.value) + Number(value).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
const currencySymbol = value => ({ USD: '$', CAD: 'C$', CNY: '¥', EUR: '€', GBP: '£', JPY: '¥', AUD: 'A$' }[value] || `${value} `)
const amazonHeaderLabel = column => amazonFinanceHeaderLabel(column)
const split = v => String(v || '').split(',').filter(Boolean)
const summaryAmountKeys = ['sales', 'sales_product', 'sales_product_tax', 'sales_shipping', 'sales_shipping_tax', 'sales_gift_wrap', 'sales_gift_wrap_tax', 'sales_promotion', 'sales_shipment_promotion', 'refund', 'refund_product_sales', 'refund_product_sales_tax', 'refund_shipping_credits', 'refund_shipping_tax', 'refund_gift_wrap', 'refund_gift_wrap_tax', 'refund_regulatory_fee', 'refund_regulatory_tax', 'refund_promotions', 'refund_promotions_tax', 'refund_other', 'tax', 'tax_marketplace_withheld', 'delivery', 'delivery_fba_fees', 'delivery_fulfilment_disposal', 'delivery_transaction_other', 'delivery_fulfilment_storage', 'selling', 'selling_fees', 'ads', 'ads_other_transaction_fees', 'other', 'other_total', 'other_marketplace_withheld', 'other_subscription', 'other_inbound_placement', 'other_premium_services', 'storage', 'storage_transaction', 'storage_inventory', 'taxable_income']
function aggregateCountrySummary(site, items) {
  const row = { site_id: site.value, country: site.label || site.value, sales_volume: items.reduce((sum, item) => sum + Number(item.sales_volume || 0), 0), validation_errors: [...new Set(items.flatMap(item => item.validation_errors || []))], sales_categories: [] }
  for (const key of summaryAmountKeys) {
    const values = items.map(item => item[key]).filter(value => value !== undefined)
    row[key] = !values.length || values.some(value => value == null || value === '') ? null : values.reduce((sum, value) => sum + numericValue(value), 0)
  }
  const categories = new Map()
  for (const item of items) for (const category of item.sales_categories || []) categories.set(category.category, (categories.get(category.category) || 0) + Number(category.quantity || 0))
  row.sales_categories = [...categories.entries()].map(([category, quantity]) => ({ category, quantity }))
  const months = new Map()
  for (const item of items) for (const month of item.months || []) months.set(month.month, [...(months.get(month.month) || []), month])
  row.months = [...months.entries()].sort(([left], [right]) => left.localeCompare(right)).map(([month, monthItems]) => ({ ...aggregateCountrySummary(site, monthItems), month }))
  return row
}
async function loadOptions(kind, id) { const params = { type: kind, month: filters.month, quarter: filters.quarter, store_id: props.storeId, site_id: props.siteId }; const result = await Promise.all(['month', 'quarter'].map(field => request.get('/tax-data/amazon/filter-options', { params: { ...params, field } }))); if (id !== requestId) return; options.month = result[0].data.values || []; options.quarter = result[1].data.values || [] }
async function loadRows() {
  if (tab.value === 'categories') { ++requestId; loading.value = false; error.value = ''; categoriesTable.value?.load(); return }
  const id = ++requestId
  const kind = tab.value
  error.value = ''
  loading.value = true
  if (kind === 'summary') { summaryRows.value = []; summaryCountryRows.value = [] }
  try {
    const endpoint = kind === 'summary' ? 'quarters' : kind
    const commonParams = { store_id: props.storeId, site_id: props.siteId, site_currency: siteCurrency.value }
    if (kind === 'summary' && summaryCountryMode.value) {
      const availableSites = props.countryOptions.length ? props.countryOptions : [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }]
      const sites = props.siteId ? [availableSites.find(site => site.value === props.siteId) || { value: props.siteId, label: props.siteId, currency: props.siteCurrencyCode }] : availableSites
      const results = await Promise.all(sites.map(site => request.get(`/tax-data/amazon/${endpoint}`, { params: { currency: props.siteId ? currency.value : 'CNY', date_from: props.dateFrom, date_to: props.dateTo, store_id: props.storeId, site_id: site.value, site_currency: site.currency || defaultSiteCurrency(site.value) } })))
      if (id !== requestId) return
      summaryCountryRows.value = results.flatMap((result, index) => {
        const items = result.data.items || []
        return items.length ? [aggregateCountrySummary(sites[index], items)] : []
      })
    } else {
      const params = kind === 'summary' ? { currency: currency.value, date_from: props.dateFrom, date_to: props.dateTo, ...commonParams } : { page: page.value, page_size: pageSize.value, month: filters.month, quarter: filters.quarter, include_totals: 1, ...commonParams }
      const [r, cny] = await Promise.all([
        request.get(`/tax-data/amazon/${endpoint}`, { params }),
        kind === 'summary' && showAverageRate.value ? request.get(`/tax-data/amazon/${endpoint}`, { params: { ...params, currency: 'CNY' } }) : null,
      ])
      if (id !== requestId) return
      if (kind === 'summary') summaryRows.value = cny ? attachCnyIncome(r.data.items || [], cny.data.items || []) : r.data.items || []
      else {
        rows.value = r.data.items || []
        total.value = r.data.total || 0
        const headers = r.data.columns || []
        let saved = null
        try { saved = JSON.parse(localStorage.getItem(`amazon-columns:${props.storeId}:${kind}`) || 'null') } catch { }
        columns.value = saved?.order?.filter(column => headers.includes(column)).concat(headers.filter(column => !saved?.order?.includes(column))) || headers
        visibleColumns.value = saved?.visible?.filter(column => headers.includes(column)) || [...headers]
        pinnedColumns.value = saved?.pinned?.filter(column => headers.includes(column)) || []
        rawTotals.value = r.data.totals || {}
        await loadOptions(kind, id)
      }
    }
  } catch (e) {
    if (id === requestId) error.value = e.response?.data?.error || '数据加载失败，请重试'
  } finally { if (id === requestId) loading.value = false }
}
function switchSummaryGrouping() { for (const key of Object.keys(expandedMonths)) delete expandedMonths[key]; return loadRows() }
function switchCurrency(value) { if (currency.value === value) return; currency.value = value; loadRows() }
function toggleValue(field, value) { const a = split(filters[field]); filters[field] = !value ? '' : a.includes(value) ? a.filter(v => v !== value).join(',') : [...a, value].join(','); page.value = 1; loadRows() }
function switchTab(value) { if (tab.value === value) return; document.dispatchEvent(new CustomEvent('amazon-close-filters')); tab.value = value; rows.value = []; columns.value = []; total.value = 0; page.value = 1; filters.month = ''; filters.quarter = ''; loadRows() }
function applyColumnConfig(config) { visibleColumns.value = config.visible; pinnedColumns.value = config.pinned; columns.value = config.order.filter(column => columns.value.includes(column)); localStorage.setItem(`amazon-columns:${props.storeId}:${tab.value}`, JSON.stringify(config)); columnConfigOpen.value = false }
function exportReport() {
  if (props.downloadBusy || tab.value === 'categories') return
  const params = tab.value === 'summary' ? { type: 'quarters', date_from: props.dateFrom, date_to: props.dateTo, ...(summaryCountryMode.value ? { view: 'country' } : {}), currency: currency.value, store_id: props.storeId, site_id: summaryCountryMode.value ? summarySiteId.value : props.siteId, site_currency: summaryCountryMode.value ? summarySiteCurrency.value : siteCurrency.value } : { type: tab.value, month: filters.month, quarter: filters.quarter, store_id: props.storeId, site_id: props.siteId, site_currency: siteCurrency.value }
  props.downloadFile('/tax-data/amazon/export', 'export', `Amazon_${tab.value === 'summary' ? (summaryCountryMode.value ? 'countries' : 'quarters') : tab.value}.csv`, params)
}
function outside(e) { if (!e.target.closest('.amazon-filter, .amazon-filter-menu')) document.dispatchEvent(new CustomEvent('amazon-close-filters')) }
onMounted(() => { loadRows(); document.addEventListener('click', outside) }); onBeforeUnmount(() => document.removeEventListener('click', outside))
watch(() => [props.storeId, props.siteId, props.siteCurrencyCode], () => { currency.value = allCountries.value ? 'CNY' : siteCurrency.value; page.value = 1; rows.value = []; summaryRows.value = []; summaryCountryRows.value = []; loadRows() })
watch(() => [props.dateFrom, props.dateTo], () => { if (tab.value === 'summary') loadRows() })
watch(() => props.countryOptions.map(site => `${site.value}:${site.label}:${site.currency}`).join('|'), () => { if (tab.value === 'summary' && summaryCountryMode.value && allCountries.value) loadRows() })
</script>
<style scoped>
.amazon-view {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  color: #253447;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  flex: 0 0 auto;
  min-height: 64px;
  padding: 0 18px;
  border-bottom: 1px solid #e3e8ef;
}

.export-btn {
  color: #fff;
  background: #285d83;
  border-color: #285d83;
}

.export-btn:hover {
  background: #204e70;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

button,
select {
  height: 32px;
  padding: 0 12px;
  border: 1px solid #dce4e7;
  border-radius: 6px;
  background: #fff;
  color: #465669;
  font-size: 12px;
}

button {
  cursor: pointer;
}

button:hover {
  background: #f3f6f9;
  border-color: #bdcad7;
}

button:disabled {
  opacity: .5;
  cursor: default;
}

.tabs {
  display: flex;
  align-self: stretch;
  gap: 24px;
}

.tabs button {
  position: relative;
  height: 100%;
  min-height: 52px;
  padding: 0 2px;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: #738093;
  font-size: 13px;
}

.tabs button.active {
  color: #285d83;
  font-weight: 600;
}

.tabs button.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  background: #285d83;
}

.table-region {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
}

.table-wrap {
  overflow: auto;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  max-width: 100%;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: #aebbc8 #f5f7fa;
}

.data-table {
  table-layout: fixed;
  min-width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 12px;
}

.data-table th,
.data-table td {
  box-sizing: border-box;
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid #e8edf2;
  border-right: 1px solid #f1f3f6;
  overflow: hidden;
  text-overflow: ellipsis;
}

.data-table th {
  position: sticky;
  top: 0;
  z-index: 2;
  height: 48px;
  background: #f4f6f9;
  color: #596b7e;
  font-weight: 600;
  line-height: 1.4;
  white-space: normal;
  overflow-wrap: anywhere;
}

.data-table td {
  height: 52px;
  color: #455961;
  background: #fff;
  white-space: nowrap;
}

.data-table tbody tr:nth-child(even) td {
  background: #fcfdfe;
}

.data-table tbody tr:hover td {
  background: #f0f6fb;
}

.data-table tfoot {
  position: static;
}

.data-table tfoot th,
.data-table tfoot td {
  position: sticky;
  top: auto;
  bottom: 0;
  z-index: 2;
  height: 42px;
  padding: 9px 12px;
  color: #33485b;
  background: #edf3f7;
  border-top: 2px solid #c6d3dd;
  border-bottom: 0;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  box-shadow: 0 -5px 12px rgb(44 67 87 / 10%);
}

.data-table tfoot .sticky {
  z-index: 5;
  background: #edf3f7;
}

.data-table .sticky {
  position: sticky;
  z-index: 3;
}

.data-table thead th.sticky {
  top: 0;
  z-index: 5;
  background: #f0f3f7;
}

.month {
  left: 0;
  width: 86px;
}

.quarter {
  left: 86px;
  width: 90px;
  box-shadow: 2px 0 4px -4px #8092a6;
}

.summary {
  width: max-content;
  min-width: 100%;
  table-layout: auto;
}

.summary th,
.summary td {
  white-space: nowrap;
  overflow: visible;
  text-overflow: clip;
}

.summary thead th:first-child,
.summary tbody td:first-child,
.summary tfoot th:first-child {
  min-width: 92px;
}

.month-expander {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 22px;
  margin: 0 5px 0 -4px;
  padding: 0;
  border: 0;
  background: transparent;
  vertical-align: middle;
}

.month-expander:hover {
  border: 0;
  background: transparent;
}

.month-expander span {
  width: 6px;
  height: 6px;
  border-right: 1.5px solid #718294;
  border-bottom: 1.5px solid #718294;
  transform: rotate(-45deg);
  transition: transform .15s ease;
}

.month-expander[aria-expanded="true"] span {
  transform: rotate(45deg);
}

.month-detail-row td {
  background: #f8fafc !important;
  color: #526577;
}

.month-detail-row .month-detail-period-cell {
  font-variant-numeric: tabular-nums;
}

.month-detail-indent {
  display: inline-block;
  width: 12px;
  height: 12px;
  margin: 0 8px 0 1px;
  border-left: 1px solid #cbd5df;
  border-bottom: 1px solid #cbd5df;
  vertical-align: -2px;
}

.summary-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-bottom: 1px solid #e5ebf0;
  background: #fbfcfd;
}

.summary-toolbar strong {
  display: block;
  color: #33485b;
  font-size: 12px;
}

.summary-toolbar span {
  display: block;
  margin-top: 3px;
  color: #8794a0;
  font-size: 11px;
}

.summary-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: flex-end;
  gap: 12px;
  min-width: 0;
}

.country-summary-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #41566a;
  font-size: 12px;
  white-space: nowrap;
  cursor: pointer;
}

.country-summary-toggle input {
  width: 14px;
  height: 14px;
  margin: 0;
  accent-color: #285d83;
  cursor: pointer;
}

.country-summary-toggle input:focus-visible {
  outline: 2px solid #285d83;
  outline-offset: 2px;
}

.summary-controls span {
  display: inline;
  margin-top: 0;
}

.currency-segment {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid #d8e1e8;
  border-radius: 7px;
  background: #f1f4f7;
}

.currency-segment button {
  min-width: 68px;
  height: 32px;
  padding: 0 10px;
  border: 0;
  border-radius: 5px;
  color: #6d7c89;
  background: transparent;
  font-size: 11px;
}

.currency-segment button.active {
  color: #285d83;
  background: #fff;
  box-shadow: 0 1px 4px rgb(38 56 74 / 12%);
  font-weight: 700;
}

.data-table .empty {
  height: 132px;
  text-align: center;
  color: #89949b;
  padding: 40px;
  font-size: 13px;
}

.data-table .numeric {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.data-table .amazon-pinned {
  position: sticky;
  z-index: 3;
  background: #fff;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.data-table thead .amazon-pinned {
  z-index: 6;
  background: #f0f3f7;
}

.data-table tfoot .amazon-pinned {
  z-index: 5;
  background: #edf3f7;
}

.header-formula {
  display: block;
  margin-top: 4px;
  font-size: 10px;
  font-weight: 400;
  color: #8794a0;
}

.summary-header-label {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.loading {
  position: absolute;
  inset: 0;
  z-index: 7;
  display: grid;
  place-items: center;
  background: rgb(255 255 255 / 64%);
  pointer-events: none;
}

.loading span {
  padding: 8px 14px;
  border: 1px solid #dce5ed;
  border-radius: 16px;
  color: #62788b;
  background: rgb(255 255 255 / 94%);
  box-shadow: 0 2px 8px rgb(30 50 70 / 8%);
  font-size: 11px;
}

.pager {
  display: flex;
  flex: 0 0 auto;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  padding: 10px 18px;
  box-sizing: border-box;
  border-top: 1px solid #e3e8ef;
  font-size: 12px;
  color: #738093;
}

.row-count {
  margin-right: auto;
  white-space: nowrap;
}

.page-size-control {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.page-size-control select {
  min-width: 64px;
  height: 30px;
  padding: 0 7px;
}

.error {
  margin: 16px 20px;
  color: #a34635;
  font-size: 13px;
}

button:focus-visible,
select:focus-visible,
.table-wrap:focus-visible {
  outline: 2px solid #285d83;
  outline-offset: 2px;
}

@media (max-width:600px) {
  .toolbar {
    padding: 0 12px;
    gap: 12px;
  }

  .tabs {
    gap: 16px;
  }

  .tabs button {
    font-size: 12px;
  }

  .pager {
    flex-wrap: wrap;
    gap: 8px;
    padding: 10px 12px;
  }

  .row-count {
    flex: 1;
  }
}

@media (max-width:760px) {
  .summary-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .summary-controls {
    justify-content: flex-start;
    width: 100%;
  }
}

@media (max-width:520px) {
  .summary-controls {
    align-items: stretch;
    gap: 10px;
  }
}
</style>
