<template>
  <MainLayout>
    <section class="customs-page">
      <header class="heading"><div><h1>报关单</h1></div><div class="heading-actions"><button :disabled="exporting || loading || uploading || !total" @click="downloadReport">{{ exporting ? '正在生成…' : '下载报表' }}</button><button class="primary" :disabled="uploading" @click="fileInput?.click()">{{ uploading ? `正在导入 ${importCompleted}/${importResults.length}` : '批量导入报关单' }}</button></div><input ref="fileInput" class="hidden-input" type="file" accept=".pdf,application/pdf" multiple :disabled="uploading" @change="upload" /></header>
      <p v-if="error" role="alert" class="error">{{ error }}</p><p v-if="message" role="status" class="message">{{ message }}</p>
      <section v-if="importResults.length" class="import-panel" aria-label="批量导入结果" :aria-busy="uploading">
        <div class="import-heading"><strong>{{ uploading ? '批量导入中' : '批量导入完成' }}</strong><span role="status">已处理 {{ importCompleted }}/{{ importResults.length }} · 成功 {{ importCounts.success }} · 重复跳过 {{ importCounts.duplicate }} · 失败 {{ importCounts.failed }}</span><button v-if="!uploading" type="button" @click="importResults = []">关闭</button></div>
        <progress :value="importCompleted" :max="importResults.length" aria-label="导入进度"></progress>
        <p class="import-hint">每个 PDF 对应一份报关单，单文件不超过 20 MB；按海关编号去重。{{ uploading ? '正在逐份处理，请保持页面打开。' : '失败文件可修正后重新选择导入。' }}</p>
        <ul class="import-results"><li v-for="(result, index) in importResults" :key="index"><span class="import-file" :title="result.name">{{ result.name }}</span><span class="import-status" :class="result.status">{{ importStatusLabels[result.status] }}</span><span class="import-detail">{{ result.message }}</span></li></ul>
      </section>
      <div class="filters">
        <CustomsDateRangePicker v-model:from="dateFrom" v-model:to="dateTo" @change="search" />
        <Listbox v-slot="{ open }" :model-value="destinationPort" @update:model-value="selectPort">
          <div class="port-filter">
            <ListboxButton class="port-trigger" aria-label="指运港" :title="destinationPort || '指运港'"><span>{{ destinationPort ? `指运港：${destinationPort}` : '指运港' }}</span><svg class="filter-chevron" :class="{ open }" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg></ListboxButton>
            <ListboxOptions class="port-options" aria-label="指运港选项">
              <ListboxOption v-for="port in ['', ...destinationOptions]" :key="port" v-slot="{ active, selected }" :value="port" as="template"><li class="port-option" :class="{ active, selected }"><span>{{ port || '全部指运港' }}</span><svg v-if="selected" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m3 8 3 3 7-7" /></svg></li></ListboxOption>
            </ListboxOptions>
          </div>
        </Listbox>
        <div class="filter-actions"><button type="button" :disabled="loading || summaryLoading" @click="resetFilters">重置</button></div>
      </div>
      <CustomsDashboard :summary="summary" :loading="summaryLoading" :error="summaryError" @retry="loadSummary" />
      <div class="table-card">
        <div class="toolbar"><h2>报关单明细</h2><label class="column-toggle"><input v-model="showAllFields" type="checkbox" />显示全部字段</label></div>
        <div class="table-scroll" :aria-busy="loading" tabindex="0" aria-label="报关单明细列表">
          <table class="declaration-table">
            <thead><tr>
              <th class="number-cell">海关编号</th>
              <th v-for="[key, label] in fieldLabels" :key="key" :class="{ numeric: numericFields.has(key) }">{{ label }}</th>
              <th v-for="[key, label] in itemLabels" :key="key" :class="{ numeric: numericFields.has(key), 'product-cell': key === 'name', 'currency-cell': key === 'currency' }">{{ label }}</th>
              <th class="status-cell">识别状态</th><template v-if="showAllFields"><th>识别方式</th><th>核对提示</th><th>文件名</th><th>上传时间</th></template><th class="action-cell">操作</th>
            </tr></thead>
            <tbody>
              <tr v-if="loading"><td :colspan="columnCount" class="empty">正在加载报关单…</td></tr>
              <tr v-else-if="!records.length"><td :colspan="columnCount" class="empty">{{ listError ? listError : '当前筛选范围暂无报关单，上传 PDF 或调整条件后查看' }}<button v-if="listError" @click="load">重试</button></td></tr>
              <template v-else v-for="(row, rowIndex) in records" :key="row.id">
                <tr v-for="(item, index) in itemsOf(row).length ? itemsOf(row) : [{}]" :key="index" :class="{ 'alternate-row': rowIndex % 2, 'group-start': index === 0 }">
                  <template v-if="index === 0">
                    <td class="number-cell" :rowspan="Math.max(1, itemsOf(row).length)">{{ row.declaration_number }}</td>
                    <td v-for="[key] in fieldLabels" :key="key" :rowspan="Math.max(1, itemsOf(row).length)" :title="fieldValue(row, key)" :class="{ numeric: numericFields.has(key) }">{{ fieldValue(row, key) }}</td>
                  </template>
                  <td v-for="[key] in itemLabels" :key="key" :class="{ numeric: numericFields.has(key), 'product-cell': key === 'name', 'currency-cell': key === 'currency' }" :title="item[key] || ''">{{ item[key] || '—' }}</td>
                  <template v-if="index === 0">
                    <td class="status-cell" :rowspan="Math.max(1, itemsOf(row).length)"><span class="status" :class="{ warning: warningsOf(row).length }" :title="warningsOf(row).join('；')">{{ warningsOf(row).length ? '需核对' : '已识别' }}</span></td>
                    <template v-if="showAllFields">
                      <td :rowspan="Math.max(1, itemsOf(row).length)">{{ row.recognition_method === 'ocr' ? '扫描识别' : '文字提取' }}</td>
                      <td :rowspan="Math.max(1, itemsOf(row).length)"><ul v-if="warningsOf(row).length" class="warning-list"><li v-for="(warning, i) in warningsOf(row)" :key="i">{{ warning }}</li></ul><span v-else>—</span></td>
                      <td :rowspan="Math.max(1, itemsOf(row).length)" :title="row.file_name">{{ row.file_name }}</td><td :rowspan="Math.max(1, itemsOf(row).length)">{{ dateTime(row.created_at) }}</td>
                    </template>
                    <td class="action-cell" :rowspan="Math.max(1, itemsOf(row).length)"><button class="text-button" :disabled="downloading === row.id" @click="download(row)">{{ downloading === row.id ? '下载中…' : '原文件' }}</button></td>
                  </template>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <footer class="pagination"><span class="row-count">共 {{ total }} 份报关单</span><label class="page-size">每页 <select v-model.number="pageSize" :disabled="loading" @change="page = 1; load()"><option :value="20">20</option><option :value="50">50</option><option :value="100">100</option></select> 份</label><span>第 {{ page }} / {{ Math.max(1, Math.ceil(total / pageSize)) }} 页</span><button :disabled="loading || page <= 1" @click="changePage(-1)">上一页</button><button :disabled="loading || page * pageSize >= total" @click="changePage(1)">下一页</button></footer>
      </div>
    </section>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue'
import CustomsDashboard from '@/components/CustomsDashboard.vue'
import CustomsDateRangePicker from '@/components/CustomsDateRangePicker.vue'
import { financeFields, financeItems, numericFields, customsDate, customsFeeAmount, customsTradeMode } from '@/utils/customsDeclarations'
import MainLayout from '@/components/layout/MainLayout.vue'
import request from '@/utils/request'
import { buildCustomsReport } from '@/utils/customsReport'
import { importCustomsBatch } from '@/utils/customsBatchImport'
const endpoint = '/finance/customs-declarations'
const records = ref([]), total = ref(0), page = ref(1), pageSize = ref(20)
const exporting = ref(false), loading = ref(false), uploading = ref(false), downloading = ref(null)
const listError = ref(''), error = ref(''), message = ref(''), fileInput = ref(null)
const importResults = ref([])
const importStatusLabels = { pending: '等待处理', uploading: '导入中', success: '成功', duplicate: '重复跳过', failed: '失败' }
const importCounts = computed(() => importResults.value.reduce((counts, result) => { counts[result.status]++; return counts }, { pending: 0, uploading: 0, success: 0, duplicate: 0, failed: 0 }))
const importCompleted = computed(() => importCounts.value.success + importCounts.value.duplicate + importCounts.value.failed)
let listRequest = 0
let summaryRequest = 0
const showAllFields = ref(false), destinationPort = ref(''), destinationOptions = ref([]), dateFrom = ref(''), dateTo = ref(''), appliedFilters = ref({})
const summary = ref(null), summaryLoading = ref(false), summaryError = ref('')
const fieldValue = (row, key) => {
  const value = row.fields?.[key] || (key === 'declaration_date' ? row.declaration_date : '')
  if (key === 'freight' || key === 'insurance') return customsFeeAmount(value) ?? '—'
  if (key === 'trade_mode') return customsTradeMode(value) || '—'
  return key.endsWith('_date') ? customsDate(value) : value || '—'
}
async function loadSummary() {
  const requestId = ++summaryRequest
  summaryLoading.value = true; summaryError.value = ''; summary.value = null
  try {
    const { data } = await request.get(`${endpoint}/summary`, { params: appliedFilters.value })
    if (requestId === summaryRequest) { summary.value = data; destinationOptions.value = data.destination_options || [] }
  } catch (e) {
    if (requestId === summaryRequest) summaryError.value = e.response?.data?.error || '看板加载失败，请重试'
  } finally { if (requestId === summaryRequest) summaryLoading.value = false }
}
const refresh = () => Promise.all([load(), loadSummary()])
function resetFilters() { destinationPort.value = ''; dateFrom.value = ''; dateTo.value = ''; search() }
function selectPort(value) { if (destinationPort.value === value) return; destinationPort.value = value; search() }
const allFieldLabels = Object.entries({ pre_entry_number: '预录入编号', port: '进出口口岸', record_number: '备案号', export_date: '进出口日期', declaration_date: '申报日期', consignor: '境内收发货人', overseas_consignee: '境外收发货人', transport_mode: '运输方式', vessel: '运输工具名称及航次', bill_number: '提运单号', manufacturer: '生产销售 / 消费使用单位', trade_mode: '贸易 / 监管方式', tax_exemption: '征免性质', settlement: '结汇方式', license_number: '许可证号', destination: '运抵 / 启运国', destination_port: '指运 / 经停港', source_region: '境内货源 / 目的地', approval_number: '批准文号', trade_terms: '成交方式', freight: '运费', insurance: '保费', other_fees: '杂费', contract_number: '合同协议号', packages: '件数', packaging: '包装种类', gross_weight: '毛重（千克）', net_weight: '净重（千克）', containers: '集装箱号', attachments: '随附单据', remarks: '标记唛码及备注' })
const allItemLabels = Object.entries({ sequence: '项号', hs_code: '商品编号', name: '商品名称', specification: '规格型号', quantity: '法定数量 / 单位', secondary_quantity: '第二数量 / 单位', declared_quantity: '申报数量 / 单位', destination: '目的国', unit_price: '单价', total_price: '总价', currency: '币制', exemption: '征免' })
const fieldLabels = computed(() => showAllFields.value ? allFieldLabels : financeFields)
const itemLabels = computed(() => showAllFields.value ? allItemLabels : financeItems)
const columnCount = computed(() => 1 + fieldLabels.value.length + itemLabels.value.length + (showAllFields.value ? 6 : 2))
const itemsOf = row => Array.isArray(row.items) ? row.items : []
const warningsOf = row => Array.isArray(row.warnings) ? row.warnings : []
const dateTime = value => new Date(value).toLocaleString('zh-CN', { hour12: false })
async function load() { const requestId = ++listRequest; loading.value = true; listError.value = ''; try { const { data } = await request.get(endpoint, { params: { page: page.value, page_size: pageSize.value, ...appliedFilters.value } }); if (requestId !== listRequest) return; records.value = data.items || []; total.value = data.total || 0 } catch (e) { if (requestId === listRequest) { records.value = []; total.value = 0; listError.value = e.response?.data?.error || '报关单列表加载失败' } } finally { if (requestId === listRequest) loading.value = false } }
function search() {
  error.value = ''
  if (dateFrom.value && dateTo.value && dateFrom.value > dateTo.value) { error.value = '开始日期不能晚于结束日期'; return }
  appliedFilters.value = { destination_port: destinationPort.value, date_from: dateFrom.value, date_to: dateTo.value }
  page.value = 1
  refresh()
}
function changePage(delta) { error.value = ''; page.value += delta; load() }
async function upload(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length || uploading.value) return
  error.value = ''; message.value = ''; uploading.value = true
  importResults.value = files.map(file => ({ name: file.name, status: 'pending', message: '' }))
  try {
    await importCustomsBatch(files, async file => {
      const body = new FormData(); body.append('file', file)
      const { data } = await request.post(endpoint, body, { timeout: 150000 })
      return data
    }, (index, result) => { importResults.value[index] = { name: files[index].name, ...result } })
    page.value = 1
    await refresh()
  } finally { uploading.value = false }
}
async function download(row) { downloading.value = row.id; error.value = ''; try { const { data } = await request.get(`${endpoint}/${row.id}/file`, { responseType: 'blob' }); const url = URL.createObjectURL(data), link = document.createElement('a'); link.href = url; link.download = row.file_name; document.body.appendChild(link); link.click(); link.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000) } catch { error.value = '原文件下载失败，请重试' } finally { downloading.value = null } }
async function downloadReport() {
  if (exporting.value || loading.value || !total.value) return
  const options = { fields: [...fieldLabels.value], items: [...itemLabels.value], allFields: showAllFields.value, filters: { ...appliedFilters.value }, exportedAt: new Date() }
  exporting.value = true; error.value = ''; message.value = ''
  try {
    const { data } = await request.get(`${endpoint}/export`, { params: options.filters, timeout: 120000 })
    if (!data.items?.length) { message.value = '当前筛选范围暂无可导出的报关单'; return }
    const workbook = await buildCustomsReport(data.items, options)
    const buffer = await workbook.xlsx.writeBuffer()
    const url = URL.createObjectURL(new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `报关单报表_${options.exportedAt.toLocaleDateString('sv-SE')}.xlsx`
    document.body.appendChild(link); link.click(); link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    message.value = `已导出 ${data.items.length} 份报关单`
  } catch (e) { error.value = e.response?.data?.error || '报表下载失败，请重试' }
  finally { exporting.value = false }
}
onMounted(refresh)
</script>

<style scoped>
.customs-page { width:100%; box-sizing:border-box; padding:16px; color:#253447; }
.heading { display:flex; justify-content:space-between; align-items:center; gap:16px; margin-bottom:16px; }
.heading-actions { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.import-panel { margin-bottom:16px; padding:14px 16px; border:1px solid #dce4e7; border-radius:4px; background:#fff; font-size:12px; }
.import-heading { display:flex; align-items:center; flex-wrap:wrap; gap:12px; } .import-heading>span { color:#718091; } .import-heading>button { margin-left:auto; }
.import-panel progress { display:block; width:100%; height:6px; margin-top:12px; accent-color:#285d83; } .import-hint { margin:10px 0; color:#718091; line-height:1.6; }
.import-results { max-height:240px; overflow:auto; padding:0; margin:0; list-style:none; } .import-results li { display:grid; grid-template-columns:minmax(100px,1fr) 80px minmax(160px,2fr); align-items:start; gap:12px; padding:9px 0; border-top:1px solid #edf0f3; line-height:1.6; }
.import-file,.import-detail { overflow-wrap:anywhere; } .import-status { color:#718091; } .import-status.success { color:#23654b; } .import-status.duplicate { color:#8c6321; } .import-status.failed { color:#9d3c3c; } .import-status.uploading { color:#285d83; } .import-detail { color:#718091; }
@media(max-width:600px) { .import-results li { grid-template-columns:minmax(0,1fr) 80px; } .import-detail { grid-column:1/-1; } }
.eyebrow { margin:0 0 6px; color:#7890a3; font-size:11px; letter-spacing:.08em; } .heading h1 { margin:0; font-size:20px; font-weight:600; } .subtitle { color:#7b8997; font-size:13px; margin:8px 0 0; }
button,select { height:32px; padding:0 12px; border:1px solid #dce4e7; border-radius:6px; background:#fff; color:#465669; font-size:12px; } button { cursor:pointer; transition:background .15s,border-color .15s; } button:hover { background:#f3f6f9; border-color:#bdcad7; } button:disabled { opacity:.5; cursor:default; } .primary { background:#285d83; color:#fff; border-color:#285d83; white-space:nowrap; } .primary:hover { background:#204e70; } .hidden-input { display:none; }
.error,.message { padding:12px 14px; border-radius:6px; font-size:13px; line-height:1.7; margin-bottom:16px; } .error { color:#9d3c3c; background:#fff1f1; } .message { color:#23654b; background:#edf8f2; }
.filters { display:flex; align-items:center; flex-wrap:wrap; gap:12px; padding:12px 16px; margin-bottom:16px; border:1px solid #e3e8ef; border-radius:4px; background:#fff; }
.port-filter { position:relative; width:180px; max-width:100%; min-width:0; font-size:12px; }
.port-trigger { display:flex; align-items:center; justify-content:space-between; gap:10px; width:100%; height:34px; border-radius:4px; text-align:left; } .port-trigger>span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.filter-chevron,.port-option svg { display:block; width:16px; height:16px; flex:0 0 16px; stroke:currentColor; stroke-width:1.5; stroke-linecap:round; stroke-linejoin:round; } .filter-chevron { transition:transform .15s; } .filter-chevron.open { transform:rotate(180deg); }
.port-options { position:absolute; top:calc(100% + 6px); left:0; z-index:30; box-sizing:border-box; width:100%; max-height:260px; overflow:auto; margin:0; padding:4px; list-style:none; border:1px solid #d4dfe7; border-radius:4px; background:#fff; box-shadow:0 8px 24px rgb(37 52 71 / 12%); outline:none; }
.port-option { display:flex; align-items:center; justify-content:space-between; gap:8px; min-height:32px; padding:6px 8px; border-radius:3px; color:#465669; cursor:pointer; overflow-wrap:anywhere; } .port-option.active { background:#f0f6fb; } .port-option.selected { color:#285d83; background:#eaf2f8; font-weight:500; } .filter-actions { display:flex; gap:8px; margin-left:auto; }
.table-card { border:1px solid #e3e8ef; border-radius:4px; background:#fff; overflow:hidden; } .toolbar { display:flex; align-items:center; justify-content:space-between; gap:16px; min-height:44px; padding:0 16px; border-bottom:1px solid #e3e8ef; } .toolbar h2 { font-size:13px; font-weight:600; margin:0; } .column-toggle { display:flex; align-items:center; gap:6px; font-size:12px; color:#718091; cursor:pointer; } .column-toggle input { accent-color:#285d83; }
.table-scroll { max-width:100%; max-height:calc(100dvh - 220px); min-height:260px; overflow:auto; overscroll-behavior:contain; scrollbar-width:thin; scrollbar-color:#aebbc8 #f5f7fa; } .declaration-table { width:max-content; min-width:100%; border-collapse:separate; border-spacing:0; font-size:12px; }
th,td { box-sizing:border-box; min-width:115px; padding:8px 12px; text-align:left; border-bottom:1px solid #e8edf2; border-right:1px solid #f1f3f6; white-space:nowrap; } th { position:sticky; top:0; z-index:2; height:48px; background:#f4f6f9; color:#596b7e; font-weight:600; line-height:1.4; } td { height:44px; max-width:300px; overflow:hidden; text-overflow:ellipsis; color:#455961; background:#fff; vertical-align:middle; } .alternate-row td { background:#fcfdfe; } tbody tr:hover td { background:#f0f6fb; } .group-start:not(:first-child) td { border-top:1px solid #d9e3eb; }
.number-cell { min-width:196px; font-variant-numeric:tabular-nums; position:sticky; left:0; z-index:3; box-shadow:3px 0 7px rgb(31 54 73 / 6%); } thead .number-cell { z-index:5; background:#f0f3f7; } .product-cell { min-width:150px; font-weight:500; } .numeric { text-align:right; font-variant-numeric:tabular-nums; } .action-cell { min-width:86px; position:sticky; right:0; z-index:3; border-left:1px solid #e3e8ef; } thead .action-cell { z-index:5; }
.declaration-table .currency-cell { width:64px; min-width:64px; max-width:64px; padding-left:8px; padding-right:8px; text-align:center; }
.declaration-table .status-cell { width:84px; min-width:84px; max-width:84px; padding-left:8px; padding-right:8px; text-align:center; }
.text-button { height:auto; padding:0; border:0; background:transparent; color:#285d83; } .status { padding:3px 7px; border-radius:4px; background:#edf8f2; color:#23654b; font-size:11px; } .status.warning { color:#8c6321; background:#fff8e8; } .warning-list { margin:0; padding-left:18px; width:240px; white-space:normal; color:#8c6321; font-size:12px; line-height:1.7; } .empty { height:150px; padding:36px; text-align:left; color:#89949b; } .empty button { margin-left:10px; }
.pagination { display:flex; justify-content:flex-end; align-items:center; flex-wrap:wrap; gap:12px; padding:12px 18px; min-height:56px; box-sizing:border-box; font-size:12px; color:#738093; border-top:1px solid #e3e8ef; } .row-count { margin-right:auto; } .page-size { display:flex; align-items:center; gap:5px; } .page-size select { min-width:62px; padding:0 6px; }
button:focus-visible,input:focus-visible,select:focus-visible,.table-scroll:focus-visible { outline:2px solid #285d83; outline-offset:2px; }
@media(max-width:700px) { .customs-page { padding:12px; } .heading { gap:12px; } .filters { padding:12px; gap:10px; } .port-filter { width:180px; } .pagination { gap:8px; padding:12px; } .row-count { width:100%; } .number-cell { min-width:170px; } }
</style>
