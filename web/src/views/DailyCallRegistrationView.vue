<template>
  <MainLayout :locale="locale">
    <div class="calls-page" :lang="locale">
      <header class="page-heading">
        <div class="page-title"><PhoneIcon class="page-icon" /><h1>{{ t('每日来电登记') }}</h1></div>
        <div class="heading-actions"><Button variant="outline" :disabled="loading || exporting || forbidden || !total" :title="t('导出全部筛选结果')" @click="exportReport"><ClipboardDocumentListIcon class="h-4 w-4" />{{ exporting ? t('正在导出...') : t('导出报表') }}</Button><Button v-if="permissions.can_register" :disabled="loading" @click="openCreate"><PlusIcon class="h-4 w-4" />{{ t('新增登记') }}</Button></div>
      </header>

      <div class="toolbar">
        <div class="status-tabs" :aria-label="t('表类型')">
          <button v-for="option in tableTypes" :key="option.value" type="button" :aria-pressed="kind === option.value" :class="{ active: kind === option.value }" @click="setKind(option.value)">{{ option.label }}</button>
        </div>
        <div class="date-filter"><DatePicker v-model="dateValue" v-model:mode="dateMode" :locale="locale" :placeholder="t('选择时间')" :aria-label="t('时间范围')" @change="applyFilters" /><button v-if="dateValue" type="button" class="date-clear" :aria-label="t('清除时间筛选')" :title="t('清除时间筛选')" @click="dateValue = ''; applyFilters()"><XMarkIcon class="h-4 w-4" /></button></div>
        <form class="search" @submit.prevent="search"><MagnifyingGlassIcon class="h-4 w-4" /><input v-model="keyword" type="text" :aria-label="t('搜索登记')" :placeholder="t('搜索姓名、订单、邮箱或电话')" /><button v-if="keyword || appliedKeyword" type="button" class="icon-button" :aria-label="t('清除搜索')" @click="keyword = ''; search()"><XMarkIcon class="h-4 w-4" /></button><button type="submit" class="search-submit" :disabled="loading">{{ t('搜索') }}</button></form>
        <button type="button" class="icon-button" :title="t('刷新')" :aria-label="t('刷新')" :disabled="loading" @click="loadCalls"><ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" /></button>
      </div>

      <div v-if="loadError" class="empty-state" role="alert"><ExclamationCircleIcon /><h2>{{ forbidden ? t('暂无访问权限') : t('暂时无法加载登记') }}</h2><p>{{ loadError }}</p><Button variant="outline" @click="loadCalls">{{ t('重新加载') }}</Button></div>
      <template v-else>
        <div class="table-meta"><span>{{ t('来电记录') }} <strong>{{ total }}</strong></span><span>{{ t('最新登记优先') }}</span></div>
        <div class="table-scroll" tabindex="0" :aria-label="t('来电登记表')" :aria-busy="loading">
          <table>
            <colgroup><col v-for="(width, index) in columnWidths" :key="index" :style="{ width }" /></colgroup>
            <thead><tr><th v-for="label in columns" :key="label" scope="col">{{ label }}</th></tr></thead>
            <tbody v-if="loading"><tr v-for="n in 6" :key="n" class="skeleton-row"><td v-for="label in columns" :key="label"><span /></td></tr></tbody>
            <tbody v-for="item in loading ? [] : calls" :key="item.id" class="call-group">
              <tr class="call-summary">
                <td><div class="date-cell"><time :datetime="item.created_at">{{ formatDate(item.created_at, true) }}</time><button type="button" class="icon-button people-button" :title="t('查看填写人员')" :aria-label="`${t('查看填写人员')}: ${item.name}`" @click="openDetail(item, 'people')"><UserCircleIcon class="h-4 w-4" /></button></div></td>
                <td><button type="button" class="name-link" @click="openDetail(item)">{{ item.name }}</button></td>
                <td class="mono">{{ item.order_number || '-' }}</td><td>{{ item.email || '-' }}</td>
                <template v-if="kind === 'inbound'"><td class="mono">{{ item.sku || '-' }}</td><td class="mono">{{ item.asin || '-' }}</td><td>{{ item.issue_type || '-' }}</td></template>
                <td>{{ item.bought_in || '-' }}</td><td class="mono">{{ item.phone_no || '-' }}</td><td>{{ item.country || '-' }}</td>
                <template v-if="kind === 'outbound'">
                  <td><p class="cell-copy" :title="item.information_requested">{{ item.information_requested || '-' }}</p></td>
                  <td><p class="cell-copy" :title="item.result">{{ item.result || '-' }}</p></td>
                </template>
                <template v-else>
                  <td><p class="cell-copy" :class="{ 'quality-copy': item.quality_result?.trim() }" :title="item.quality_result">{{ item.quality_result?.trim() || '-' }}</p></td>
                  <td><p class="cell-copy" :title="item.reason">{{ item.reason || '-' }}</p></td>
                </template>
              </tr>
              <tr class="call-issue"><td :colspan="columns.length"><div class="issue-inline"><span class="issue-label">{{ t('问题简述') }}</span><p class="issue-full">{{ item.brief_of_issue || '-' }}</p></div></td></tr>
            </tbody>
          </table>
        </div>
        <div v-if="!loading && !calls.length" class="empty-state"><PhoneIcon /><h2>{{ keyword || status ? t('没有符合条件的登记') : t('暂无来电登记') }}</h2><Button v-if="permissions.can_register && !keyword && !status" variant="outline" @click="openCreate"><PlusIcon class="h-4 w-4" />{{ t('新增登记') }}</Button></div>
        <footer class="pagination"><span class="pagination-count">{{ recordCount }}</span><div class="pagination-controls"><label class="page-size"><span>{{ t('每页') }}</span><select v-model.number="pageSize" :aria-label="t('每页条数')" :disabled="loading" @change="changePageSize"><option :value="20">20</option><option :value="50">50</option><option :value="100">100</option></select></label><button class="pagination-button" type="button" :title="t('上一页')" :aria-label="t('上一页')" :disabled="page <= 1 || loading" @click="changePage(page - 1)"><ChevronLeftIcon class="h-4 w-4" /></button><form class="pagination-jump" @submit.prevent="jumpToPage"><input v-model="pageInput" type="text" inputmode="numeric" :aria-label="t('页码')" :disabled="loading || !total" @keydown.esc="pageInput = String(page)" /><span class="pagination-divider">/</span><span>{{ totalPages }}</span><button type="submit" :disabled="loading || !total">{{ t('跳转') }}</button></form><button class="pagination-button" type="button" :title="t('下一页')" :aria-label="t('下一页')" :disabled="page >= totalPages || loading" @click="changePage(page + 1)"><ChevronRightIcon class="h-4 w-4" /></button></div></footer>
      </template>
    </div>

    <Dialog v-if="readingIssue" :open="true" :initial-focus="issueBody" class="relative z-[80] calls-dialog" @close="readingIssue = null">
      <div class="drawer-backdrop" />
      <div class="issue-position">
        <DialogPanel class="w-full max-w-3xl" :lang="locale">
          <div class="issue-reader">
            <header class="drawer-heading"><div><span class="drawer-caption">{{ t('来电') }} / {{ readingIssue.id }}</span><DialogTitle>{{ t('问题描述全文') }}</DialogTitle></div><button type="button" class="icon-button" :aria-label="t('关闭')" @click="readingIssue = null"><XMarkIcon class="h-5 w-5" /></button></header>
            <div class="issue-context"><strong>{{ readingIssue.name }}</strong><time>{{ formatDate(readingIssue.created_at, true) }}</time><span v-if="readingIssue.order_number">{{ t('订单号') }}: {{ readingIssue.order_number }}</span></div>
            <div ref="issueBody" class="issue-body" tabindex="0" role="region" :aria-label="t('问题描述全文')"><p>{{ readingIssue.brief_of_issue }}</p></div>
            <footer class="issue-footer"><button type="button" class="issue-copy" @click="copyIssue"><ClipboardDocumentListIcon class="h-4 w-4" />{{ t('复制全文') }}</button><Button variant="outline" @click="readingIssue = null">{{ t('关闭') }}</Button></footer>
          </div>
        </DialogPanel>
      </div>
    </Dialog>

    <TransitionRoot as="template" :show="drawerOpen">
      <Dialog class="relative z-[70] calls-dialog" @close="closeDrawer">
        <TransitionChild as="template" enter="transition-opacity duration-200" enter-from="opacity-0" enter-to="opacity-100" leave="transition-opacity duration-150" leave-from="opacity-100" leave-to="opacity-0"><div class="drawer-backdrop" /></TransitionChild>
        <div class="drawer-position" :class="{ 'create-position': creatingDialog }">
          <TransitionChild as="template" enter="transform transition duration-200 ease-out" :enter-from="creatingDialog ? 'opacity-0 scale-95' : 'translate-x-full'" enter-to="translate-x-0 opacity-100 scale-100" leave="transform transition duration-150 ease-in" leave-from="translate-x-0 opacity-100 scale-100" :leave-to="creatingDialog ? 'opacity-0 scale-95' : 'translate-x-full'">
            <DialogPanel class="drawer-panel" :class="{ 'create-panel': creatingDialog }" :lang="locale">
              <header class="drawer-heading"><div><span class="drawer-caption">{{ isNew ? t('新登记') : `${t('来电')} / ${selected?.id || ''}` }}</span><DialogTitle>{{ isNew ? t('新增来电登记') : selected?.name || t('来电详情') }}</DialogTitle></div><button type="button" class="icon-button" :title="t('关闭')" :aria-label="t('关闭')" :disabled="saving" @click="closeDrawer"><XMarkIcon class="h-5 w-5" /></button></header>
              <div v-if="!isNew" class="drawer-tabs" role="tablist" :aria-label="t('登记详情')">
                <button id="call-record-tab" type="button" role="tab" :aria-selected="drawerTab === 'record'" aria-controls="call-record-panel" :tabindex="drawerTab === 'record' ? 0 : -1" :class="{ active: drawerTab === 'record' }" @click="drawerTab = 'record'" @keydown.right.prevent="switchDrawerTab('people')"><ClipboardDocumentListIcon class="h-4 w-4" />{{ t('登记详情') }}</button>
                <button id="call-people-tab" type="button" role="tab" :aria-selected="drawerTab === 'people'" aria-controls="call-people-panel" :tabindex="drawerTab === 'people' ? 0 : -1" :class="{ active: drawerTab === 'people' }" @click="drawerTab = 'people'" @keydown.left.prevent="switchDrawerTab('record')"><UserCircleIcon class="h-4 w-4" />{{ t('填写人员') }}</button>
              </div>
              <div class="drawer-body">
                <div v-if="detailLoading" class="drawer-state" role="status">{{ t('正在加载登记...') }}</div>
                <div v-else-if="detailError" class="drawer-state" role="alert"><p>{{ detailError }}</p><Button variant="outline" @click="loadDetail(selected.id)">{{ t('重试') }}</Button></div>
                <div v-else-if="drawerTab === 'people' && selected" id="call-people-panel" role="tabpanel" aria-labelledby="call-people-tab" class="people-timeline">
                  <section class="person-event"><div class="person-marker"><PhoneIcon class="h-5 w-5" /></div><div><h3>{{ t('首次登记') }}</h3><strong>{{ selected.created_by_name }} <small>#{{ selected.created_by_id }}</small></strong><time>{{ formatDate(selected.created_at) }}</time></div></section>
                  <section v-if="selected.registration_updated_at !== selected.created_at" class="person-event"><div class="person-marker"><PencilSquareIcon class="h-5 w-5" /></div><div><h3>{{ t('登记信息 · 最近填写') }}</h3><strong>{{ selected.registration_updated_by_name }} <small>#{{ selected.registration_updated_by_id }}</small></strong><time>{{ formatDate(selected.registration_updated_at) }}</time></div></section>
                  <section class="person-event" :class="{ 'event-pending': !selected.reviewed_at }"><div class="person-marker review-marker"><CheckBadgeIcon class="h-5 w-5" /></div><div><h3>{{ t(kind === 'outbound' ? '外呼结果' : '质检信息 · 最近填写') }}</h3><template v-if="selected.reviewed_at"><strong>{{ selected.reviewed_by_name }} <small>#{{ selected.reviewed_by_id }}</small></strong><time>{{ formatDate(selected.reviewed_at) }}</time></template><strong v-else>-</strong></div></section>
                </div>
                <div v-else id="call-record-panel" :role="isNew ? undefined : 'tabpanel'" :aria-labelledby="isNew ? undefined : 'call-record-tab'">
                  <section class="detail-section">
                    <div class="section-heading"><h3><PhoneIcon class="h-4 w-4" />{{ t('来电登记') }}</h3><button v-if="!isNew && permissions.can_register && !editing" type="button" class="icon-button" :title="t('编辑登记信息')" :aria-label="t('编辑登记信息')" @click="startEdit('registration')"><PencilSquareIcon class="h-4 w-4" /></button><LockClosedIcon v-else-if="!permissions.can_register" class="h-4 w-4 muted" :title="t('由海外客服填写')" /></div>
                    <div class="date-detail"><span>{{ t('日期') }}</span><time>{{ isNew ? t('保存时自动记录') : formatDate(selected?.created_at) }}</time></div>
                    <form v-if="editing === 'registration'" class="entry-form" @invalid.capture="localizeValidation" @input.capture="clearValidation" @submit.prevent="saveRegistration">
                      <label>{{ t('表类型') }}<input :value="tableTypes.find(option => option.value === kind)?.label" readonly disabled /></label>
                      <div class="form-grid">
                        <div v-for="field in fields" :key="field.key" :class="{ 'field-select': field.options }">
                          <Listbox v-if="field.options" v-model="registration[field.key]" :disabled="saving">
                            <ListboxLabel class="field-label">{{ field.label }}</ListboxLabel>
                            <ListboxButton class="select-trigger"><span :class="{ muted: !registration[field.key] }">{{ registration[field.key] || t('请选择') }}</span><ChevronDownIcon class="h-4 w-4" /></ListboxButton>
                            <ListboxOptions class="select-options">
                              <ListboxOption v-for="option in field.options" :key="option" v-slot="{ active, selected: isSelected }" :value="option" as="template">
                                <li class="select-option" :class="{ 'option-active': active, 'option-selected': isSelected }"><span>{{ option }}</span><CheckIcon v-if="isSelected" class="h-4 w-4" /></li>
                              </ListboxOption>
                            </ListboxOptions>
                          </Listbox>
                          <template v-else><label :for="`call-${field.key}`">{{ field.label }}<span v-if="field.required" class="required"> *</span></label><input :id="`call-${field.key}`" v-model="registration[field.key]" :type="field.type || 'text'" :maxlength="field.max" :required="field.required" :disabled="saving" /></template>
                        </div>
                      </div>
                      <label>{{ t('问题简述') }}<span class="required"> *</span><textarea v-model="registration.brief_of_issue" rows="5" maxlength="10000" required :disabled="saving" /></label>
                      <p v-if="saveError" class="form-error" role="alert">{{ saveError }}</p>
                      <div class="form-actions"><Button v-if="!isNew" type="button" variant="outline" :disabled="saving" @click="cancelEdit">{{ t('取消') }}</Button><Button type="submit" :disabled="saving || (!isNew && !dirty)"><CheckIcon class="h-4 w-4" />{{ saving ? t('保存中...') : isNew ? t('创建登记') : t('保存登记') }}</Button></div>
                    </form>
                    <dl v-else class="detail-grid"><div v-for="field in fields" :key="field.key"><dt>{{ field.label }}</dt><dd>{{ selected?.[field.key] || '-' }}</dd></div><div class="full-width"><dt class="issue-detail-heading"><span>{{ t('问题简述') }}</span><button v-if="selected?.brief_of_issue" type="button" class="issue-read" @click="readingIssue = selected">{{ t('查看全文') }}<ArrowRightIcon class="h-3 w-3" /></button></dt><dd>{{ selected?.brief_of_issue || '-' }}</dd></div></dl>
                  </section>
                  <section v-if="!isNew" class="detail-section review-section">
                    <div class="section-heading"><h3><CheckBadgeIcon class="h-4 w-4" />{{ t(kind === 'outbound' ? '外呼结果' : '质检记录') }}</h3><button v-if="permissions.can_review && !editing" type="button" class="icon-button" :title="t('编辑登记信息')" :aria-label="t('编辑登记信息')" @click="startEdit('review')"><PencilSquareIcon class="h-4 w-4" /></button></div>
                    <form v-if="editing === 'review'" class="entry-form" @submit.prevent="saveReview">
                      <template v-for="field in responseFields" :key="field.key"><CallProductSearch v-if="field.key === 'sku'" v-model="review.sku" :locale="locale" :disabled="saving" @select="review.asin = $event.asin || ''" /><label v-else>{{ t(field.label) }}<input v-if="field.max === 200" v-model="review[field.key]" :maxlength="field.max" :disabled="saving" /><textarea v-else v-model="review[field.key]" rows="4" :maxlength="field.max" :disabled="saving" /></label></template>
                      <p v-if="saveError" class="form-error" role="alert">{{ saveError }}</p><div class="form-actions"><Button type="button" variant="outline" :disabled="saving" @click="cancelEdit">{{ t('取消') }}</Button><Button type="submit" :disabled="saving || !dirty">{{ saving ? t('保存中...') : t('保存登记') }}</Button></div>
                    </form>
                    <dl v-else class="detail-grid"><div v-for="field in responseFields" :key="field.key" :class="{ 'full-width': field.max > 200 }"><dt>{{ t(field.label) }}</dt><dd>{{ selected?.[field.key]?.trim() || '-' }}</dd></div></dl>
                  </section>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </TransitionRoot>
  </MainLayout>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, Listbox, ListboxButton, ListboxLabel, ListboxOption, ListboxOptions, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { ArrowPathIcon, ArrowRightIcon, CheckBadgeIcon, CheckIcon, ChevronDownIcon, ChevronLeftIcon, ChevronRightIcon, ClipboardDocumentListIcon, ExclamationCircleIcon, LockClosedIcon, MagnifyingGlassIcon, PencilSquareIcon, PhoneIcon, PlusIcon, UserCircleIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import Button from '@/components/ui/Button.vue'
import DatePicker from '@/components/ui/DatePicker.vue'
import CallProductSearch from '@/components/ui/CallProductSearch.vue'
import request from '@/utils/request'
import { useToast } from '@/composables/useToast'

import { useAuthStore } from '@/stores/auth'
import { dailyCallLocale, dailyCallText, dailyCallError } from '@/utils/dailyCallLocale'

const authStore = useAuthStore()
const locale = computed(() => dailyCallLocale(authStore.user))
const t = text => dailyCallText(locale.value, text)
watch(locale, () => { document.title = t('每日来电登记') }, { immediate: true })
const toast = useToast()
const exporting = ref(false)
const appliedKeyword = ref('')
async function exportReport() {
  if (exporting.value) return
  exporting.value = true
  const filters = { kind: kind.value, keyword: appliedKeyword.value, date: dateValue.value, status: status.value, export: 'true' }
  const exportLocale = locale.value
  const label = value => dailyCallText(exportLocale, value)
  try {
    const { data } = await request.get('/daily-calls', { params: filters })
    const items = data.data || []
    if (!items.length) { toast.info(label('没有可导出的记录')); return }
    const { default: ExcelJS } = await import('exceljs')
    const workbook = new ExcelJS.Workbook()
    const sheet = workbook.addWorksheet(label(filters.kind === 'outbound' ? '主动外呼表' : '来电登记表'))
    const fields = [
      ['created_at', '日期'], ['name', '姓名'], ['order_number', '订单号'], ['email', '邮箱'],
      ...(filters.kind === 'inbound' ? [['sku', 'SKU'], ['asin', 'ASIN'], ['issue_type', '问题类型']] : []),
      ['bought_in', '购买渠道'], ['phone_no', '电话号码'], ['country', '国家'], ['brief_of_issue', '问题简述'],
      ...(filters.kind === 'outbound' ? [['information_requested', '需要客户提供的信息'], ['result', '外呼结果']] : [['quality_result', '质检结果'], ['reason', '理由']]),
    ]
    sheet.columns = fields.map(([key, title]) => ({ key, header: label(title), width: ['brief_of_issue', 'information_requested', 'reason', 'result'].includes(key) ? 55 : key === 'email' ? 30 : 22 }))
    for (const item of items) sheet.addRow(Object.fromEntries(fields.map(([key]) => [key, key === 'created_at' ? (item[key] ? formatDate(item[key]) : '') : String(item[key] ?? '')])))
    sheet.eachRow(row => { row.alignment = { vertical: 'top', wrapText: true } })
    sheet.getRow(1).font = { bold: true, color: { argb: 'FFFFFFFF' } }
    sheet.getRow(1).fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF285D50' } }
    sheet.views = [{ state: 'frozen', ySplit: 1 }]
    sheet.autoFilter = { from: { row: 1, column: 1 }, to: { row: items.length + 1, column: fields.length } }
    const bytes = await workbook.xlsx.writeBuffer()
    const url = URL.createObjectURL(new Blob([bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `${filters.kind}-calls-${filters.date || 'all'}-${Date.now()}.xlsx`
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    toast.success(label('报表已导出'))
  } catch (error) { toast.error(errorMessage(error, '导出失败，请重试')) }
  finally { exporting.value = false }
}
const purchaseChannels = ['Amazon', 'TikTok', 'Walmart', 'Wayfair', 'HomeDepot', 'Temu', "Lowe's", 'eBay', 'Shopify', 'Shein', 'AliExpress', 'Shoppee']
const countries = ['United States', 'Canada', 'Mexico', 'United Kingdom', 'Germany', 'France', 'Italy', 'Spain', 'Japan', 'India', 'Australia', 'Norway', 'Sweden', 'Poland', 'Netherlands', 'Belgium', 'Middle East']
const responseFields = computed(() => kind.value === 'outbound' ? [{ key: 'information_requested', label: '需要客户提供的信息', max: 10000 }, { key: 'result', label: '外呼结果', max: 10000 }] : [{ key: 'sku', label: 'SKU', max: 200 }, { key: 'asin', label: 'ASIN', max: 200 }, { key: 'issue_type', label: '问题类型', max: 200 }, { key: 'quality_result', label: '质检结果', max: 2000 }, { key: 'reason', label: '理由', max: 10000 }])
const columns = computed(() => (kind.value === 'outbound' ? ['日期', '姓名', '订单号', '邮箱', '购买渠道', '电话号码', '国家', '需要客户提供的信息', '外呼结果'] : ['日期', '姓名', '订单号', '邮箱', 'SKU', 'ASIN', '问题类型', '购买渠道', '电话号码', '国家', '质检结果', '理由']).map(t))
const columnWidths = computed(() => kind.value === 'outbound' ? ['130px', '110px', '145px', '180px', '100px', '90px', '110px', '220px', '170px'] : ['130px', '110px', '145px', '180px', '110px', '110px', '130px', '100px', '90px', '110px', '170px', '170px'])
const fields = computed(() => [
  { key: 'name', label: t('姓名'), max: 200, required: true },
  { key: 'order_number', label: t('订单号'), max: 200 },
  { key: 'email', label: t('邮箱'), type: 'email', max: 254 },
  { key: 'bought_in', label: t('购买渠道'), options: purchaseChannels },
  { key: 'phone_no', label: t('电话号码'), type: 'tel', max: 100 },
  { key: 'country', label: t('国家'), options: countries },
])
const tableTypes = computed(() => [{ value: 'inbound', label: t('来电登记表') }, { value: 'outbound', label: t('主动外呼表') }])
const calls = ref([]), total = ref(0), page = ref(1), pageSize = ref(20), keyword = ref(''), status = ref(''), kind = ref('inbound'), dateMode = ref('day'), dateValue = ref('')
const recordCount = computed(() => locale.value === 'en' ? `${total.value} ${total.value === 1 ? 'record' : 'records'}` : `共 ${total.value} 条`)
const loading = ref(true), loadError = ref(''), forbidden = ref(false)
const permissions = ref({ can_register: false, can_review: false })
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const pageInput = ref('1')
watch(page, value => { pageInput.value = String(value) })
function jumpToPage() {
  const value = String(pageInput.value).trim()
  if (!/^\d+$/.test(value) || !Number.isSafeInteger(Number(value))) {
    pageInput.value = String(page.value)
    return
  }
  const target = Math.min(totalPages.value, Math.max(1, Number(value)))
  pageInput.value = String(target)
  changePage(target)
}
const drawerOpen = ref(false), drawerTab = ref('record'), selected = ref(null), isNew = ref(false)
const creatingDialog = ref(false)
const readingIssue = ref(null)
const issueBody = ref(null)
async function copyIssue() {
  try {
    await navigator.clipboard.writeText(readingIssue.value.brief_of_issue)
    toast.success(t('全文已复制'))
  } catch { toast.error(t('复制失败，请选择文字后复制')) }
}
const detailLoading = ref(false), detailError = ref(''), editing = ref(''), saving = ref(false), saveError = ref('')
const registration = reactive({}), review = reactive({ sku: '', asin: '', issue_type: '', quality_result: '', reason: '' })
let listRequest = 0, detailRequest = 0, editSnapshot = ''
function formSnapshot(value) { return JSON.stringify(Object.fromEntries(Object.entries(value).map(([key, text]) => [key, text.trim()]))) }
const dirty = computed(() => editing.value && formSnapshot(editing.value === 'review' ? review : registration) !== editSnapshot)

function localizeValidation(event) {
  const input = event.target
  input.setCustomValidity(t(input.validity.valueMissing ? '请填写此字段' : input.validity.typeMismatch ? '请输入有效的邮箱地址' : '请检查填写内容'))
}
function clearValidation(event) { event.target.setCustomValidity?.('') }
function formatDate(value, dateOnly = false) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', ...(dateOnly ? {} : { hour: '2-digit', minute: '2-digit', second: '2-digit' }) }).format(new Date(value))
}
function errorMessage(error, fallback) { return dailyCallError(locale.value, error, fallback) }
async function loadCalls() {
  const version = ++listRequest
  loading.value = true
  loadError.value = ''
  try {
    const { data } = await request.get('/daily-calls', { params: { page: page.value, page_size: pageSize.value, keyword: appliedKeyword.value, status: status.value, kind: kind.value, date: dateValue.value } })
    if (version !== listRequest) return
    calls.value = data.data || []
    total.value = data.total
    permissions.value = data.permissions
    forbidden.value = false
  } catch (error) {
    if (version !== listRequest) return
    loadError.value = errorMessage(error, '请检查网络连接后重试')
    forbidden.value = error?.response?.status === 403
    permissions.value = { can_register: false, can_review: false }
  } finally { if (version === listRequest) loading.value = false }
}
function applyFilters() { page.value = 1; loadCalls() }
function search() { if (loading.value) return; keyword.value = keyword.value.trim(); appliedKeyword.value = keyword.value; applyFilters() }
function setStatus(value) { status.value = value; search() }
function setKind(value) { kind.value = value; page.value = 1; status.value = ''; loadCalls() }
function changePage(value) { if (loading.value || value === page.value) return; page.value = value; loadCalls() }
function changePageSize() { page.value = 1; loadCalls() }
function startEdit(section) {
  for (const key of Object.keys(review)) delete review[key]
  for (const field of responseFields.value) review[field.key] = selected.value?.[field.key] || ''
  for (const field of [...fields.value, { key: 'brief_of_issue' }]) registration[field.key] = selected.value?.[field.key] || ''
  editing.value = section
  saveError.value = ''
  editSnapshot = formSnapshot(section === 'review' ? review : registration)
}
function cancelEdit() {
  if (dirty.value && !window.confirm(t('放弃尚未保存的修改？'))) return
  editing.value = ''
  saveError.value = ''
}
function openCreate() {
  ++detailRequest
  creatingDialog.value = true
  selected.value = null
  isNew.value = true
  drawerTab.value = 'record'
  detailLoading.value = false
  detailError.value = ''
  startEdit('registration')
  drawerOpen.value = true
}
async function loadDetail(id) {
  const version = ++detailRequest
  detailLoading.value = true
  detailError.value = ''
  try {
    const { data } = await request.get(`/daily-calls/${id}`, { params: { kind: kind.value } })
    if (version !== detailRequest) return
    selected.value = data.data
    permissions.value = data.permissions
  } catch (error) {
    if (version === detailRequest) detailError.value = errorMessage(error, '加载登记详情失败')
  } finally { if (version === detailRequest) detailLoading.value = false }
}
function openDetail(item, tab = 'record') {
  creatingDialog.value = false
  selected.value = item
  isNew.value = false
  editing.value = ''
  saveError.value = ''
  drawerTab.value = tab
  drawerOpen.value = true
  loadDetail(item.id)
}
function closeDrawer() {
  if (saving.value || (dirty.value && !window.confirm(t('放弃尚未保存的修改？')))) return
  ++detailRequest
  drawerOpen.value = false
  editing.value = ''
}
async function switchDrawerTab(tab) {
  drawerTab.value = tab
  await nextTick()
  document.getElementById(`call-${tab}-tab`)?.focus()
}
async function saveRegistration() {
  await saveSection('registration', { ...registration })
}
async function saveReview() {
  await saveSection('review', { ...review })
}
async function saveSection(section, payload) {
  if (saving.value) return
  if (!isNew.value && !dirty.value) return
  saving.value = true
  saveError.value = ''
  try {
    if (isNew.value) {
      const { data } = await request.post('/daily-calls', payload, { params: { kind: kind.value } })
      selected.value = data
      drawerOpen.value = false
      page.value = 1
      keyword.value = ''
      appliedKeyword.value = ''
      status.value = ''
    } else {
      await request.put(`/daily-calls/${selected.value.id}/${section}`, payload, { params: { kind: kind.value } })
      await loadDetail(selected.value.id)
    }
    editing.value = ''
    toast.success(section === 'review' ? t('质检已保存') : t('登记已保存'))
    loadCalls()
  } catch (error) { saveError.value = errorMessage(error, '保存失败，请重试') }
  finally { saving.value = false }
}
onMounted(loadCalls)
</script>

<style scoped>
.calls-page { max-width: 1800px; margin: 0 auto; padding: 36px 32px; color: #252525; letter-spacing: 0; }
.page-heading, .page-title, .toolbar, .table-meta, .pagination, .pagination > div, .drawer-heading, .section-heading, .section-heading h3, .form-actions { display: flex; align-items: center; }
.page-heading { justify-content: space-between; gap: 20px; margin-bottom: 30px; }
.heading-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.search:focus-within { border-color: #608b7c; box-shadow: 0 0 0 2px #e5eeeb; }
.date-filter:focus-within { border-radius: 7px; box-shadow: 0 0 0 2px #e5eeeb; }
.page-title { gap: 12px; min-width: 0; }
h1 { font-size: 24px; font-weight: 650; line-height: 1.3; overflow-wrap: anywhere; }
.page-icon { width: 24px; height: 24px; flex-shrink: 0; color: #436d65; }
.toolbar { gap: 12px; padding-bottom: 18px; border-bottom: 1px solid #dedede; flex-wrap: wrap; }
.status-tabs { display: flex; gap: 4px; margin-right: auto; }
.status-tabs button { padding: 8px 13px; font-size: 13px; color: #686868; border-radius: 5px; white-space: nowrap; }
.status-tabs button.active { color: #214c44; background: #e5eeeb; font-weight: 600; }
.search { display: flex; align-items: center; gap: 8px; width: 330px; max-width: 100%; padding-left: 12px; border: 1px solid #d8d8d8; border-radius: 6px; background: white; color: #777; }
.search input { min-width: 0; width: 100%; font-size: 13px; padding: 9px 0; outline: none; }
.date-filter { position: relative; display: flex; align-items: center; gap: 4px; flex: 0 0 252px; width: 252px; max-width: 100%; min-height: 40px; }
.date-filter :deep(.calendar-picker) { flex: 1 1 0; min-width: 0; }
.date-filter :deep(.date-picker-button) { width: 100%; }
.search { min-height: 40px; width: 360px; }
.search:focus-within { border-color: #527b6e; box-shadow: 0 0 0 2px #e5eeeb; }
.search > svg { flex-shrink: 0; }
.search-submit { align-self: stretch; padding: 0 14px; border-left: 1px solid #d8d8d8; color: #285d50; font-size: 13px; white-space: nowrap; }
.search-submit:hover:not(:disabled) { background: #edf4f0; }
.export-action { padding-left: 12px; border-left: 1px solid #dedede; }
.date-filter .date-clear { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; flex: 0 0 auto; color: #7a8580; }
.date-filter .date-clear:hover { color: #285d50; background: #edf4f0; border-radius: 5px; }
.date-clear { width: 28px; height: 36px; color: #84918b; font-size: 18px; }
.icon-button { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 5px; flex-shrink: 0; color: #626262; transition: background 150ms; }
.icon-button:hover:not(:disabled) { background: #e9eeeb; color: #214c44; }
button:disabled { opacity: .45; cursor: not-allowed; }
button:focus-visible, .table-scroll:focus-visible { outline: 2px solid #43776d; outline-offset: 3px; }
.table-meta { justify-content: space-between; padding: 19px 0 13px; font-size: 12px; color: #777; }
.table-meta strong { margin-left: 8px; color: #333; font-variant-numeric: tabular-nums; }
.table-scroll { overflow: auto; width: 100%; max-height: 65vh; background: #fff; border-top: 1px solid #ddd; border-bottom: 1px solid #ddd; }
table { table-layout: fixed; border-collapse: separate; border-spacing: 0; width: 1155px; min-width: 100%; font-size: 12px; }
th { overflow-wrap: anywhere; position: sticky; top: 0; z-index: 2; background: #f1f3f2; color: #5c625f; font-size: 11px; font-weight: 600; text-align: left; padding: 13px 14px; white-space: normal; border-bottom: 1px solid #ddd; }
td { padding: 14px; height: 65px; vertical-align: top; border-bottom: 1px solid #efefef; overflow-wrap: anywhere; background: white; }
tr:last-child td { border-bottom: 0; }
tr:hover td { background: #f7faf8; }
th:first-child, td:first-child { position: sticky; left: 0; z-index: 1; border-right: 1px solid #eceeec; }
th:first-child { z-index: 3; }
.date-cell { display: flex; align-items: center; gap: 4px; white-space: nowrap; font-variant-numeric: tabular-nums; }
.people-button { width: 25px; height: 25px; margin-top: -3px; color: #668078; }
.name-link { text-align: left; color: #215e50; font-weight: 600; text-decoration: underline; text-decoration-color: #bed1ca; text-underline-offset: 4px; overflow-wrap: anywhere; }
.mono { font-variant-numeric: tabular-nums; }
.cell-copy { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; line-height: 1.65; white-space: pre-wrap; }
.quality-copy { color: #275e51; }
.table-scroll { container-type: inline-size; }
.call-group .call-summary td { border-bottom: 0; height: auto; padding-bottom: 10px; }
.call-group .call-issue td { position: static; border-right: 0; border-bottom: 1px solid #cdd8d2; padding: 0 14px 20px; height: auto; }
.call-group:last-child .call-issue td { border-bottom: 0; }
.call-group:hover td { background: #f7faf8; }
.issue-inline { position: sticky; left: 14px; width: calc(100cqw - 28px); max-width: 100%; padding-top: 12px; border-top: 1px solid #edf0ee; }
.issue-label { display: block; margin-bottom: 6px; font-size: 11px; color: #68776e; font-weight: 600; }
.issue-full { max-width: 100ch; margin: 0; font-size: 13px; line-height: 1.8; white-space: pre-wrap; overflow-wrap: anywhere; }
.issue-read { display: inline-flex; align-items: center; gap: 6px; min-height: 32px; margin-top: 6px; color: #215e50; font-size: 12px; font-weight: 600; }
.issue-read:hover { text-decoration: underline; text-underline-offset: 4px; }
.issue-position { position: fixed; inset: 0; display: flex; align-items: center; justify-content: center; padding: 16px; }
.issue-reader { display: flex; flex-direction: column; max-height: calc(100dvh - 32px); overflow: hidden; border-radius: 8px; background: white; color: #292929; box-shadow: 0 20px 60px rgb(0 0 0 / 16%); }
.issue-reader .drawer-heading { flex-shrink: 0; }
.issue-footer { display: flex; flex-shrink: 0; justify-content: space-between; align-items: center; gap: 16px; border-top: 1px solid #e7e7e7; padding: 14px 28px; }
.issue-copy { display: inline-flex; align-items: center; gap: 8px; min-height: 36px; color: #215e50; font-size: 13px; }
.issue-copy:hover { text-decoration: underline; text-underline-offset: 4px; }
.issue-detail-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.issue-detail-heading .issue-read { margin-top: 0; }
.issue-context { display: flex; flex: 0 0 auto; flex-wrap: wrap; gap: 6px 20px; padding: 14px 28px; border-bottom: 1px solid #e7e7e7; background: #f7faf8; font-size: 12px; overflow-wrap: anywhere; }
.issue-context > * { min-width: 0; max-width: 100%; }
.issue-context strong { font-weight: 600; }
.issue-context time, .issue-context span { color: #626262; }
.issue-body { min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: 24px 28px 32px; }
.issue-body p { font-size: 15px; line-height: 1.85; white-space: pre-wrap; overflow-wrap: anywhere; }
.issue-body:focus-visible { outline: 2px solid #43776d; outline-offset: -3px; }
.pending { color: #866725; background: #fbf3db; padding: 3px 7px; border-radius: 4px; font-size: 11px; white-space: nowrap; }
.pagination { justify-content: space-between; padding: 18px 0 4px; border-top: 1px solid #e3e8e5; font-size: 12px; color: #777; }
.pagination-count { font-variant-numeric: tabular-nums; color: #69736e; }
.pagination-controls { gap: 6px; padding: 3px; border: 1px solid #d9dfdc; border-radius: 8px; background: #f7faf8; font-variant-numeric: tabular-nums; }
.page-size { display: inline-flex; align-items: center; gap: 6px; padding-left: 8px; color: #69736e; white-space: nowrap; }
.page-size select { height: 32px; min-width: 58px; padding: 0 22px 0 9px; border: 1px solid #d9dfdc; border-radius: 5px; appearance: none; background: #fff; color: #285d50; font-weight: 600; cursor: pointer; background-image: linear-gradient(45deg, transparent 50%, #668078 50%), linear-gradient(135deg, #668078 50%, transparent 50%); background-position: calc(100% - 11px) 13px, calc(100% - 7px) 13px; background-size: 4px 4px, 4px 4px; background-repeat: no-repeat; }
.page-size select:focus-visible { outline: 2px solid #608b7c; outline-offset: 1px; }
.pagination-jump { display: flex; align-items: center; gap: 7px; }
.pagination-jump input { width: 48px; height: 32px; border: 1px solid #d9dfdc; border-radius: 5px; background: #fff; color: #285d50; font-weight: 600; text-align: center; }
.pagination-jump input:focus-visible { outline: 2px solid #608b7c; outline-offset: 1px; }
.pagination-jump button { height: 32px; padding: 0 9px; border-radius: 5px; color: #285d50; font-weight: 600; }
.pagination-jump button:hover:not(:disabled) { background: #e5eeeb; }
.pagination-divider { color: #c4ccc8; }
.pagination-button { display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; border: 0; border-radius: 5px; background: transparent; color: #526c63; transition: border-color 150ms, background 150ms, color 150ms; }
.pagination-button:hover:not(:disabled) { border-color: #8eafa3; background: #edf4f0; color: #214c44; }
.pagination-button:focus-visible { outline: 2px solid #bdd4ca; outline-offset: 2px; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 15px; min-height: 320px; text-align: center; padding: 30px; }
.empty-state > svg { width: 32px; height: 32px; color: #789188; }
.empty-state h2 { font-size: 16px; font-weight: 600; }
.empty-state p { font-size: 13px; color: #777; }
.skeleton-row span { display: block; height: 12px; margin-top: 7px; background: #efefef; border-radius: 3px; }
:global(.calls-dialog) { position: relative; z-index: 70; letter-spacing: 0; }
.drawer-backdrop { position: fixed; inset: 0; background: rgb(20 25 23 / 30%); }
.drawer-position { position: fixed; inset: 0; display: flex; justify-content: flex-end; pointer-events: none; }
.drawer-panel { pointer-events: auto; width: 600px; max-width: 100%; height: 100%; display: flex; flex-direction: column; background: white; box-shadow: -12px 0 45px rgb(0 0 0 / 8%); color: #292929; }
.create-position { align-items: center; justify-content: center; padding: 24px; }
.create-panel { width: 680px; height: auto; max-height: calc(100dvh - 48px); border-radius: 8px; overflow: hidden; box-shadow: 0 20px 60px rgb(0 0 0 / 16%); }
.drawer-heading { justify-content: space-between; gap: 16px; padding: 26px 28px 22px; border-bottom: 1px solid #eee; }
.drawer-heading > div { min-width: 0; }
.drawer-caption { color: #777; font-size: 10px; }
.drawer-heading h2 { font-size: 21px; font-weight: 600; margin-top: 5px; overflow-wrap: anywhere; }
.drawer-tabs { display: flex; gap: 24px; padding: 0 28px; border-bottom: 1px solid #e8e8e8; }
.drawer-tabs button { display: flex; align-items: center; gap: 7px; padding: 15px 0; font-size: 13px; border-bottom: 2px solid transparent; color: #777; }
.drawer-tabs button.active { border-color: #396c5f; color: #285d50; font-weight: 600; }
.drawer-body { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: 0 28px 30px; }
.drawer-state { padding: 70px 0; text-align: center; color: #777; display: grid; justify-items: center; gap: 18px; }
.detail-section { padding: 24px 0; }
.detail-section + .detail-section { border-top: 1px solid #e7e7e7; }
.section-heading { justify-content: space-between; min-height: 34px; margin-bottom: 18px; }
.section-heading h3 { gap: 8px; font-size: 14px; font-weight: 650; }
.section-heading h3 svg { color: #527b6e; }
.date-detail { display: flex; justify-content: space-between; gap: 12px; margin-bottom: 20px; font-size: 12px; color: #777; }
.date-detail time { color: #555; font-variant-numeric: tabular-nums; }
.detail-grid, .form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px 24px; }
.detail-grid dt { font-size: 11px; color: #777; margin-bottom: 7px; }
.detail-grid dd { font-size: 13px; line-height: 1.8; white-space: pre-wrap; overflow-wrap: anywhere; }
.full-width { grid-column: 1 / -1; }
.entry-form { display: grid; gap: 20px; }
.entry-form label { display: block; font-size: 12px; color: #555; }
.field-select { position: relative; min-width: 0; }
.field-label { display: block; font-size: 12px; color: #555; }
.select-trigger { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; min-height: 43px; margin-top: 8px; padding: 10px 11px; border: 1px solid #d5d9d6; border-radius: 5px; background: white; color: #282828; text-align: left; font-size: 13px; line-height: 1.6; }
.select-trigger span { min-width: 0; overflow-wrap: anywhere; }
.select-trigger svg, .select-option svg { flex-shrink: 0; }
.select-trigger:focus-visible { outline: 2px solid #bdd4ca; border-color: #608b7c; outline-offset: 1px; }
.select-trigger:disabled { background: #f5f6f5; }
.select-options { position: absolute; top: 100%; left: 0; z-index: 5; width: 100%; margin-top: 5px; padding: 4px; max-height: min(240px, 30vh); overflow-y: auto; overscroll-behavior: contain; border: 1px solid #d5d9d6; border-radius: 6px; background: white; box-shadow: 0 8px 24px rgb(0 0 0 / 12%); outline: none; }
.select-option { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 9px 8px; border-radius: 3px; color: #444; font-size: 13px; line-height: 1.5; cursor: pointer; overflow-wrap: anywhere; }
.option-active { background: #edf3ef; }
.option-selected { color: #245e4e; font-weight: 600; }
.entry-form input, .entry-form textarea { display: block; width: 100%; min-width: 0; margin-top: 8px; border: 1px solid #d5d9d6; border-radius: 5px; padding: 10px 11px; background: white; color: #282828; font-size: 13px; line-height: 1.6; }
.entry-form textarea { resize: vertical; min-height: 80px; }
.entry-form input:focus, .entry-form textarea:focus { outline: 2px solid #bdd4ca; border-color: #608b7c; outline-offset: 1px; }
.entry-form input:disabled, .entry-form textarea:disabled { background: #f5f6f5; }
.required { color: #a34e45; }
.form-actions { justify-content: flex-end; gap: 10px; }
.form-error { color: #a13932; font-size: 13px; }
.muted { color: #888; }
.people-timeline { padding: 32px 0; }
.person-event { display: grid; grid-template-columns: 38px minmax(0, 1fr); gap: 17px; padding-bottom: 38px; position: relative; }
.person-event:not(:last-child)::after { content: ''; position: absolute; top: 41px; bottom: 6px; left: 18px; width: 1px; background: #e4e8e5; }
.person-marker { display: flex; align-items: center; justify-content: center; width: 38px; height: 38px; border-radius: 50%; background: #eef1ef; color: #58766a; }
.review-marker { background: #e7f1eb; color: #32664d; }
.person-event h3 { font-size: 12px; color: #777; margin-bottom: 10px; }
.person-event strong { font-size: 15px; font-weight: 600; display: block; overflow-wrap: anywhere; }
.person-event small { color: #888; font-size: 11px; font-weight: 400; margin-left: 5px; }
.person-event time { display: block; font-size: 12px; margin-top: 7px; color: #888; }
.event-pending strong { color: #888; font-weight: 400; font-size: 13px; }
@media (max-width: 767px) {
  .issue-position { padding: 12px; }
  .issue-reader { max-height: calc(100dvh - 24px); }
  .issue-context { padding: 12px 20px; }
  .issue-body { padding: 20px; }
  .issue-footer { padding: 12px 20px; }
  .create-position { padding: 12px; }
  .create-panel { max-height: calc(100dvh - 24px); }
  .calls-page { padding: 24px 16px; }
  .page-heading { align-items: flex-start; flex-wrap: wrap; gap: 16px; margin-bottom: 24px; }
  h1 { font-size: 20px; }
  .page-title { gap: 9px; }
  .page-icon { width: 20px; height: 20px; }
  .status-tabs { width: 100%; }
  .search { flex: 1 1 100%; width: 100%; }
  .heading-actions { width: 100%; }
  .drawer-heading { padding: 22px 20px; }
  .drawer-tabs { padding: 0 20px; }
  .drawer-body { padding: 0 20px 24px; }
  .detail-grid, .form-grid { gap: 20px 16px; }
  .pagination { align-items: center; flex-wrap: wrap; gap: 14px; padding-top: 14px; }
  .pagination-count { max-width: 45%; line-height: 1.5; }
  .pagination-controls { flex-shrink: 0; }
  .page-size { padding-left: 4px; }
}
@media (max-width: 380px) { .form-grid, .detail-grid { grid-template-columns: minmax(0, 1fr); } }
@media (prefers-reduced-motion: reduce) { :global(.calls-dialog *) { transition-duration: 0ms !important; animation: none !important; } }
</style>
