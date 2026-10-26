<template>
  <MainLayout>
    <div class="purchase-page">
      <header class="page-header">
        <div>
          <div class="page-context"><ClipboardDocumentListIcon class="h-4 w-4" />ERP · 采购</div>
          <h1>采购计划</h1>
          <p>提交采购需求并查看历史申请</p>
        </div>
        <div class="header-actions">
          <button type="button" class="quiet-button" title="刷新采购计划" :disabled="loading" @click="fetchPlans">
            <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button type="button" class="primary-button" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            添加采购计划
          </button>
        </div>
      </header>

      <section class="list-panel" aria-label="采购计划列表">
        <div class="list-toolbar">
          <div>
            <strong>采购计划列表</strong>
            <span>共 {{ total }} 条</span>
          </div>
        </div>

        <div v-if="loadError" class="error-banner" role="alert">
          <ExclamationCircleIcon class="h-4 w-4" />
          <span>{{ loadError }}</span>
          <button type="button" @click="fetchPlans">重试</button>
        </div>

        <div class="desktop-table-wrap">
          <table class="plan-table">
            <thead>
              <tr>
                <th>产品名 / SKU</th>
                <th class="number-column">计划采购量</th>
                <th class="number-column">单箱数量</th>
                <th class="number-column">箱数</th>
                <th>期望到货时间</th>
                <th>备注</th>
                <th>申请人</th>
                <th>申请时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="plan in plans" :key="plan.id">
                <td>
                  <div class="product-cell">
                    <strong :title="plan.product_name">{{ plan.product_name }}</strong>
                    <span :title="plan.sku">{{ plan.sku }}</span>
                  </div>
                </td>
                <td class="number-cell">{{ formatNumber(plan.planned_quantity) }}</td>
                <td class="number-cell">{{ formatNumber(plan.units_per_carton) }}</td>
                <td class="number-cell">{{ formatNumber(plan.carton_count) }}</td>
                <td class="date-cell">{{ formatDate(plan.expected_arrival_date) }}</td>
                <td><div class="remark-cell" :title="plan.remark || ''">{{ plan.remark || '-' }}</div></td>
                <td class="applicant-cell">{{ plan.applicant_name }}</td>
                <td class="created-cell">{{ formatDateTime(plan.created_at) }}</td>
              </tr>
              <tr v-if="loading">
                <td colspan="8" class="state-cell"><ArrowPathIcon class="h-5 w-5 animate-spin" />正在加载采购计划</td>
              </tr>
              <tr v-else-if="!plans.length">
                <td colspan="8" class="state-cell empty-state">
                  <ClipboardDocumentListIcon class="h-8 w-8" />
                  <strong>暂无采购计划</strong>
                  <span>点击“添加采购计划”提交第一条记录</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="mobile-list">
          <div v-if="loading" class="mobile-state"><ArrowPathIcon class="h-5 w-5 animate-spin" />正在加载采购计划</div>
          <article v-for="plan in plans" v-else :key="plan.id" class="plan-card">
            <header>
              <div class="product-cell"><strong>{{ plan.product_name }}</strong><span>{{ plan.sku }}</span></div>
              <time>{{ formatDate(plan.expected_arrival_date) }}</time>
            </header>
            <dl>
              <div><dt>计划采购量</dt><dd>{{ formatNumber(plan.planned_quantity) }}</dd></div>
              <div><dt>单箱数量</dt><dd>{{ formatNumber(plan.units_per_carton) }}</dd></div>
              <div><dt>箱数</dt><dd>{{ formatNumber(plan.carton_count) }}</dd></div>
            </dl>
            <p v-if="plan.remark" class="mobile-remark">{{ plan.remark }}</p>
            <footer><span>{{ plan.applicant_name }}</span><time>{{ formatDateTime(plan.created_at) }}</time></footer>
          </article>
          <div v-if="!loading && !plans.length" class="mobile-state empty-state">
            <ClipboardDocumentListIcon class="h-8 w-8" /><strong>暂无采购计划</strong><span>提交后将在这里展示</span>
          </div>
        </div>

        <footer v-if="totalPages > 1" class="pagination">
          <span>第 {{ page }} / {{ totalPages }} 页</span>
          <div>
            <button type="button" title="上一页" :disabled="page <= 1 || loading" @click="changePage(page - 1)"><ChevronLeftIcon class="h-4 w-4" /></button>
            <button type="button" title="下一页" :disabled="page >= totalPages || loading" @click="changePage(page + 1)"><ChevronRightIcon class="h-4 w-4" /></button>
          </div>
        </footer>
      </section>
    </div>

    <TransitionRoot as="template" :show="formOpen">
      <Dialog as="div" class="relative z-50" @close="closeForm">
        <TransitionChild as="template" enter="ease-out duration-200" enter-from="opacity-0" enter-to="opacity-100" leave="ease-in duration-150" leave-from="opacity-100" leave-to="opacity-0">
          <div class="fixed inset-0 bg-neutral-900/35" />
        </TransitionChild>
        <div class="fixed inset-0 overflow-hidden">
          <div class="absolute inset-0 overflow-hidden">
            <div class="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-0 sm:pl-10">
              <TransitionChild as="template" enter="transform transition ease-out duration-250" enter-from="translate-x-full" enter-to="translate-x-0" leave="transform transition ease-in duration-180" leave-from="translate-x-0" leave-to="translate-x-full">
                <DialogPanel class="form-drawer pointer-events-auto">
                  <header class="drawer-header">
                    <div><DialogTitle>添加采购计划</DialogTitle><p>申请人与申请时间将由系统自动填写</p></div>
                    <button type="button" title="关闭" aria-label="关闭" :disabled="submitting" @click="closeForm"><XMarkIcon class="h-5 w-5" /></button>
                  </header>

                  <form class="drawer-form" @submit.prevent="submitPlan">
                    <div class="form-body">
                      <div class="field-block">
                        <label for="purchase-product">产品<span>*</span></label>
                        <div ref="productPickerRef" class="product-picker">
                          <button id="purchase-product" type="button" class="product-picker-button" :class="{ selected: form.product_sku_id, invalid: fieldErrors.product_sku_id }" @click="toggleProductPicker">
                            <span v-if="selectedProduct" class="selected-product"><strong>{{ selectedProduct.product_name }}</strong><small>{{ selectedProduct.sku }}</small></span>
                            <span v-else class="placeholder">搜索并选择产品库中的产品</span>
                            <ChevronUpDownIcon class="h-4 w-4" />
                          </button>
                          <div v-if="productPickerOpen" class="product-options">
                            <div class="product-search">
                              <MagnifyingGlassIcon class="h-4 w-4" />
                              <input ref="productSearchRef" v-model="productQuery" type="search" placeholder="搜索产品名或 SKU" autocomplete="off" @input="scheduleProductSearch" />
                            </div>
                            <div class="product-results">
                              <div v-if="productsLoading" class="product-result-state"><ArrowPathIcon class="h-4 w-4 animate-spin" />正在搜索</div>
                              <button v-for="product in products" v-else :key="product.id" type="button" class="product-option" :class="{ active: product.id === form.product_sku_id }" @click="selectProduct(product)">
                                <span><strong>{{ product.product_name }}</strong><small>{{ product.sku }}</small></span>
                                <CheckIcon v-if="product.id === form.product_sku_id" class="h-4 w-4" />
                              </button>
                              <div v-if="!productsLoading && !products.length" class="product-result-state">没有匹配的产品</div>
                            </div>
                          </div>
                        </div>
                        <p v-if="fieldErrors.product_sku_id" class="field-error">{{ fieldErrors.product_sku_id }}</p>
                      </div>

                      <div class="quantity-grid">
                        <div class="field-block">
                          <label for="planned-quantity">计划采购量<span>*</span></label>
                          <input id="planned-quantity" v-model.number="form.planned_quantity" type="number" min="1" step="1" inputmode="numeric" placeholder="例如：1200" :class="{ invalid: fieldErrors.planned_quantity }" />
                          <p v-if="fieldErrors.planned_quantity" class="field-error">{{ fieldErrors.planned_quantity }}</p>
                        </div>
                        <div class="field-block">
                          <label for="units-per-carton">单箱数量<span>*</span></label>
                          <input id="units-per-carton" v-model.number="form.units_per_carton" type="number" min="1" step="1" inputmode="numeric" placeholder="例如：12" :class="{ invalid: fieldErrors.units_per_carton }" />
                          <p v-if="fieldErrors.units_per_carton" class="field-error">{{ fieldErrors.units_per_carton }}</p>
                        </div>
                        <div class="field-block">
                          <label for="carton-count">箱数<span>*</span></label>
                          <input id="carton-count" v-model.number="form.carton_count" type="number" min="1" step="1" inputmode="numeric" placeholder="例如：100" :class="{ invalid: fieldErrors.carton_count }" />
                          <p v-if="fieldErrors.carton_count" class="field-error">{{ fieldErrors.carton_count }}</p>
                        </div>
                      </div>

                      <div class="field-block">
                        <label for="arrival-date">期望到货时间<span>*</span></label>
                        <div ref="calendarRef" class="calendar-picker">
                          <button
                            id="arrival-date"
                            type="button"
                            class="date-picker-button"
                            :class="{ invalid: fieldErrors.expected_arrival_date }"
                            aria-haspopup="dialog"
                            :aria-expanded="calendarOpen"
                            @click="toggleCalendar"
                          >
                            <span :class="form.expected_arrival_date ? 'date-value' : 'date-placeholder'">
                              {{ form.expected_arrival_date ? formatDate(form.expected_arrival_date) : '请选择到货日期' }}
                            </span>
                            <CalendarDaysIcon class="h-4 w-4" aria-hidden="true" />
                          </button>
                          <div v-if="calendarOpen" class="calendar-popover" role="dialog" aria-label="选择期望到货时间">
                            <div class="calendar-toolbar">
                              <button type="button" class="calendar-nav" title="上个月" aria-label="上个月" @click="shiftCalendarMonth(-1)">
                                <ChevronLeftIcon class="h-4 w-4" />
                              </button>
                              <strong>{{ calendarMonthLabel }}</strong>
                              <button type="button" class="calendar-nav" title="下个月" aria-label="下个月" @click="shiftCalendarMonth(1)">
                                <ChevronRightIcon class="h-4 w-4" />
                              </button>
                            </div>
                            <div class="calendar-weekdays" aria-hidden="true">
                              <span v-for="label in weekLabels" :key="label">{{ label }}</span>
                            </div>
                            <div class="calendar-grid">
                              <button
                                v-for="day in calendarDays"
                                :key="day.key"
                                type="button"
                                class="calendar-day"
                                :class="{ 'outside-month': !day.isCurrentMonth, today: day.isToday, selected: day.isSelected }"
                                @click="chooseArrivalDate(day)"
                              >
                                {{ day.day }}
                              </button>
                            </div>
                            <div class="calendar-footer">
                              <button type="button" class="today-button" @click="selectToday">今天</button>
                            </div>
                          </div>
                        </div>
                        <p v-if="fieldErrors.expected_arrival_date" class="field-error">{{ fieldErrors.expected_arrival_date }}</p>
                      </div>

                      <div class="field-block">
                        <label for="purchase-remark">备注</label>
                        <textarea id="purchase-remark" v-model="form.remark" rows="5" maxlength="1000" placeholder="补充采购要求、包装说明或其他信息"></textarea>
                        <small class="character-count">{{ form.remark.length }} / 1000</small>
                      </div>
                    </div>

                    <footer class="drawer-footer">
                      <button type="button" class="cancel-button" :disabled="submitting" @click="closeForm">取消</button>
                      <button type="submit" class="submit-button" :disabled="submitting"><ArrowPathIcon v-if="submitting" class="h-4 w-4 animate-spin" />{{ submitting ? '提交中' : '提交采购计划' }}</button>
                    </footer>
                  </form>
                </DialogPanel>
              </TransitionChild>
            </div>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>
  </MainLayout>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import {
  ArrowPathIcon,
  CalendarDaysIcon,
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpDownIcon,
  ClipboardDocumentListIcon,
  ExclamationCircleIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import request from '@/utils/request'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const pageSize = 20
const plans = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadError = ref('')
const formOpen = ref(false)
const submitting = ref(false)
const products = ref([])
const productsLoading = ref(false)
const productPickerOpen = ref(false)
const productQuery = ref('')
const selectedProduct = ref(null)
const productPickerRef = ref(null)
const productSearchRef = ref(null)
const calendarOpen = ref(false)
const calendarRef = ref(null)
const calendarMonth = ref(startOfMonth(new Date()))
const fieldErrors = reactive({})
const form = reactive(defaultForm())
let productSearchTimer = null

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const weekLabels = ['一', '二', '三', '四', '五', '六', '日']
const calendarMonthLabel = computed(() => calendarMonth.value.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long' }))
const calendarDays = computed(() => buildCalendarDays(calendarMonth.value, form.expected_arrival_date))

function defaultForm() {
  return { product_sku_id: 0, planned_quantity: '', units_per_carton: '', carton_count: '', expected_arrival_date: '', remark: '' }
}

function padDatePart(value) {
  return String(value).padStart(2, '0')
}

function dateKey(date) {
  return `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())}`
}

function parseDateKey(value) {
  if (!value) return null
  const [year, month, day] = String(value).split('-').map(Number)
  if (!year || !month || !day) return null
  const date = new Date(year, month - 1, day)
  return Number.isNaN(date.getTime()) ? null : date
}

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

function buildCalendarDays(monthDate, selectedValue) {
  const firstDay = startOfMonth(monthDate)
  const mondayOffset = (firstDay.getDay() + 6) % 7
  const gridStart = new Date(firstDay.getFullYear(), firstDay.getMonth(), 1 - mondayOffset)
  const todayKey = dateKey(new Date())
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(gridStart)
    date.setDate(gridStart.getDate() + index)
    const key = dateKey(date)
    return {
      key,
      day: date.getDate(),
      isCurrentMonth: date.getMonth() === firstDay.getMonth(),
      isToday: key === todayKey,
      isSelected: key === selectedValue,
    }
  })
}

async function fetchPlans() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await request.get('/purchase-plans', { params: { page: page.value, page_size: pageSize } })
    plans.value = response.data.data || []
    total.value = Number(response.data.total || 0)
  } catch (error) {
    loadError.value = error?.response?.data?.error || '采购计划加载失败，请检查网络后重试'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, defaultForm())
  Object.keys(fieldErrors).forEach(key => delete fieldErrors[key])
  selectedProduct.value = null
  productQuery.value = ''
  products.value = []
  productPickerOpen.value = false
  calendarOpen.value = false
  calendarMonth.value = startOfMonth(new Date())
  formOpen.value = true
}

function closeForm() {
  if (submitting.value) return
  productPickerOpen.value = false
  calendarOpen.value = false
  formOpen.value = false
}

function toggleProductPicker() {
  calendarOpen.value = false
  productPickerOpen.value = !productPickerOpen.value
  if (productPickerOpen.value) {
    searchProducts()
    nextTick(() => productSearchRef.value?.focus())
  }
}

function scheduleProductSearch() {
  window.clearTimeout(productSearchTimer)
  productSearchTimer = window.setTimeout(searchProducts, 220)
}

async function searchProducts() {
  productsLoading.value = true
  try {
    const response = await request.get('/purchase-plans/products', { params: productQuery.value.trim() ? { q: productQuery.value.trim() } : {} })
    products.value = response.data || []
  } catch (error) {
    products.value = []
    toast.error(error?.response?.data?.error || '产品搜索失败')
  } finally {
    productsLoading.value = false
  }
}

function selectProduct(product) {
  selectedProduct.value = product
  form.product_sku_id = product.id
  delete fieldErrors.product_sku_id
  productPickerOpen.value = false
}

function toggleCalendar() {
  productPickerOpen.value = false
  calendarOpen.value = !calendarOpen.value
  if (calendarOpen.value) {
    calendarMonth.value = startOfMonth(parseDateKey(form.expected_arrival_date) || new Date())
  }
}

function shiftCalendarMonth(offset) {
  const nextMonth = new Date(calendarMonth.value)
  nextMonth.setMonth(nextMonth.getMonth() + offset)
  calendarMonth.value = startOfMonth(nextMonth)
}

function chooseArrivalDate(day) {
  form.expected_arrival_date = day.key
  delete fieldErrors.expected_arrival_date
  calendarOpen.value = false
}

function selectToday() {
  const today = new Date()
  chooseArrivalDate({ key: dateKey(today) })
  calendarMonth.value = startOfMonth(today)
}

function validateForm() {
  Object.keys(fieldErrors).forEach(key => delete fieldErrors[key])
  if (!form.product_sku_id) fieldErrors.product_sku_id = '请选择产品'
  for (const [key, label] of [['planned_quantity', '计划采购量'], ['units_per_carton', '单箱数量'], ['carton_count', '箱数']]) {
    if (!Number.isInteger(Number(form[key])) || Number(form[key]) <= 0) fieldErrors[key] = `${label}必须为正整数`
  }
  if (!form.expected_arrival_date) fieldErrors.expected_arrival_date = '请选择期望到货时间'
  return Object.keys(fieldErrors).length === 0
}

async function submitPlan() {
  if (!validateForm()) return
  submitting.value = true
  try {
    await request.post('/purchase-plans', {
      product_sku_id: form.product_sku_id,
      planned_quantity: Number(form.planned_quantity),
      units_per_carton: Number(form.units_per_carton),
      carton_count: Number(form.carton_count),
      expected_arrival_date: form.expected_arrival_date,
      remark: form.remark.trim(),
    })
    formOpen.value = false
    page.value = 1
    await fetchPlans()
    toast.success('采购计划已提交')
  } catch (error) {
    toast.error(error?.response?.data?.error || '采购计划提交失败')
  } finally {
    submitting.value = false
  }
}

function changePage(nextPage) {
  if (nextPage < 1 || nextPage > totalPages.value) return
  page.value = nextPage
  fetchPlans()
}

function formatNumber(value) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatDate(value) {
  if (!value) return '-'
  const raw = String(value).slice(0, 10)
  const [year, month, day] = raw.split('-')
  return year && month && day ? `${year}-${month}-${day}` : raw
}

function formatDateTime(value) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).replaceAll('/', '-')
}

function handleDocumentPointerDown(event) {
  if (productPickerOpen.value && !productPickerRef.value?.contains(event.target)) productPickerOpen.value = false
  if (calendarOpen.value && !calendarRef.value?.contains(event.target)) calendarOpen.value = false
}

function handleDocumentKeydown(event) {
  if (event.key === 'Escape') {
    productPickerOpen.value = false
    calendarOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown)
  document.addEventListener('keydown', handleDocumentKeydown)
  fetchPlans()
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown)
  document.removeEventListener('keydown', handleDocumentKeydown)
  window.clearTimeout(productSearchTimer)
})
</script>

<style scoped>
.purchase-page { width: min(100%, 1540px); margin: 0 auto; padding: 24px; color: #1f2937; }
.page-header { min-height: 72px; margin-bottom: 18px; display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; border-bottom: 1px solid #dfe3e8; padding-bottom: 18px; }
.page-context { margin-bottom: 6px; display: flex; align-items: center; gap: 6px; color: #667085; font-size: 12px; line-height: 18px; font-weight: 600; }
.page-header h1 { color: #111827; font-size: 24px; line-height: 32px; font-weight: 700; }
.page-header p { margin-top: 3px; color: #667085; font-size: 13px; line-height: 20px; }
.header-actions { display: flex; align-items: center; gap: 8px; }
.quiet-button, .primary-button, .cancel-button, .submit-button { height: 38px; display: inline-flex; align-items: center; justify-content: center; gap: 7px; border-radius: 7px; padding: 0 13px; font-size: 13px; line-height: 18px; font-weight: 650; transition: border-color .15s, background-color .15s, color .15s, transform .1s; white-space: nowrap; }
.quiet-button:active, .primary-button:active, .cancel-button:active, .submit-button:active { transform: translateY(1px); }
.quiet-button { border: 1px solid #d5dae1; background: #fff; color: #475467; }
.quiet-button:hover:not(:disabled) { border-color: #aeb6c2; background: #f9fafb; color: #111827; }
.primary-button, .submit-button { border: 1px solid #20252b; background: #20252b; color: #fff; }
.primary-button:hover, .submit-button:hover:not(:disabled) { border-color: #111827; background: #111827; }
button:disabled { cursor: not-allowed; opacity: .55; }
.list-panel { overflow: hidden; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; box-shadow: 0 1px 2px rgba(16,24,40,.04); }
.list-toolbar { min-height: 54px; padding: 0 16px; display: flex; align-items: center; border-bottom: 1px solid #e8ebef; background: #fafbfc; }
.list-toolbar > div { display: flex; align-items: baseline; gap: 10px; }
.list-toolbar strong { color: #20252b; font-size: 13px; font-weight: 700; }
.list-toolbar span { color: #7a8493; font-size: 12px; }
.error-banner { min-height: 44px; padding: 10px 16px; display: flex; align-items: center; gap: 8px; border-bottom: 1px solid #fecaca; background: #fff7f7; color: #b42318; font-size: 12px; }
.error-banner button { margin-left: auto; font-weight: 700; text-decoration: underline; text-underline-offset: 2px; }
.desktop-table-wrap { min-height: 360px; max-height: calc(100dvh - 285px); overflow: auto; scrollbar-gutter: stable; }
.plan-table { width: 100%; min-width: 1180px; border-collapse: collapse; table-layout: fixed; }
.plan-table th { position: sticky; top: 0; z-index: 2; height: 44px; border-bottom: 1px solid #dfe3e8; background: #f4f6f8; padding: 0 14px; color: #667085; font-size: 11px; line-height: 16px; font-weight: 700; text-align: left; white-space: nowrap; }
.plan-table th:nth-child(1) { width: 22%; }.plan-table th:nth-child(2) { width: 10%; }.plan-table th:nth-child(3) { width: 9%; }.plan-table th:nth-child(4) { width: 7%; }.plan-table th:nth-child(5) { width: 12%; }.plan-table th:nth-child(6) { width: 17%; }.plan-table th:nth-child(7) { width: 10%; }.plan-table th:nth-child(8) { width: 13%; }
.plan-table td { height: 68px; border-bottom: 1px solid #edf0f2; padding: 11px 14px; color: #344054; font-size: 12px; line-height: 18px; vertical-align: middle; }
.plan-table tbody tr:last-child td { border-bottom: 0; }
.plan-table tbody tr:hover { background: #fafbfc; }
.number-column, .number-cell { text-align: right !important; }
.number-cell { color: #111827 !important; font-variant-numeric: tabular-nums; font-weight: 650; }
.product-cell { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.product-cell strong, .product-cell span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.product-cell strong { color: #111827; font-size: 13px; line-height: 19px; font-weight: 650; }
.product-cell span { color: #667085; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; line-height: 16px; }
.remark-cell { display: -webkit-box; overflow: hidden; line-height: 18px; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.date-cell, .created-cell { color: #475467 !important; font-variant-numeric: tabular-nums; white-space: nowrap; }.applicant-cell { color: #111827 !important; font-weight: 600; }
.state-cell { height: 260px !important; color: #667085 !important; text-align: center; }
.state-cell, .empty-state { align-items: center; justify-content: center; gap: 8px; }
.state-cell { display: table-cell; }.empty-state { flex-direction: column; }.state-cell.empty-state { display: table-cell; }
.state-cell svg { margin: 0 auto 8px; }.state-cell strong, .state-cell span { display: block; }.state-cell strong { color: #344054; font-size: 13px; }.state-cell span { margin-top: 4px; color: #98a2b3; font-size: 12px; }
.pagination { min-height: 52px; padding: 8px 14px; display: flex; align-items: center; justify-content: space-between; border-top: 1px solid #dfe3e8; background: #fafbfc; color: #667085; font-size: 12px; }
.pagination div { display: flex; gap: 6px; }.pagination button { width: 32px; height: 32px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid #d7dce2; border-radius: 6px; background: #fff; color: #475467; }.pagination button:hover:not(:disabled) { border-color: #9ca3af; background: #f4f5f7; color: #111827; }
.mobile-list { display: none; }
.form-drawer { width: min(580px, 100vw); height: 100%; display: flex; flex-direction: column; overflow: hidden; background: #fff; box-shadow: -12px 0 34px rgba(16,24,40,.15); }
.drawer-header { min-height: 78px; padding: 16px 20px; display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; border-bottom: 1px solid #dfe3e8; }
.drawer-header h2 { color: #111827; font-size: 18px; line-height: 25px; font-weight: 700; }.drawer-header p { margin-top: 3px; color: #667085; font-size: 12px; line-height: 18px; }.drawer-header button { width: 34px; height: 34px; display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; border-radius: 6px; color: #667085; }.drawer-header button:hover:not(:disabled) { background: #f1f3f5; color: #111827; }
.drawer-form { min-height: 0; flex: 1; display: flex; flex-direction: column; }.form-body { min-height: 0; flex: 1; overflow-y: auto; padding: 22px 20px 32px; }.drawer-footer { min-height: 68px; padding: 14px 20px; display: flex; align-items: center; justify-content: flex-end; gap: 8px; border-top: 1px solid #dfe3e8; background: #fafbfc; }.cancel-button { border: 1px solid #d5dae1; background: #fff; color: #475467; }.cancel-button:hover:not(:disabled) { border-color: #aeb6c2; background: #f4f5f7; }
.field-block { position: relative; margin-bottom: 20px; }.field-block > label { min-height: 20px; margin-bottom: 7px; display: block; color: #475467; font-size: 12px; line-height: 20px; font-weight: 650; }.field-block > label span { margin-left: 3px; color: #dc2626; }
.field-block > input, .field-block > textarea { width: 100%; border: 1px solid #d5dae1; border-radius: 7px; background: #fff; color: #111827; font-size: 13px; outline: none; box-shadow: 0 1px 2px rgba(16,24,40,.03); transition: border-color .15s, box-shadow .15s; }.field-block > input { height: 40px; padding: 0 12px; }.field-block > textarea { min-height: 112px; resize: vertical; padding: 10px 12px; line-height: 20px; }.field-block > input::placeholder, .field-block > textarea::placeholder { color: #a1a8b3; }.field-block > input:hover, .field-block > textarea:hover { border-color: #aeb6c2; }.field-block > input:focus, .field-block > textarea:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1); }.field-block > input.invalid, .product-picker-button.invalid { border-color: #e05252; }.field-block > input.invalid:focus, .product-picker-button.invalid:focus { box-shadow: 0 0 0 3px rgba(220,38,38,.08); }
.quantity-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }.field-error { margin-top: 5px; color: #b42318; font-size: 11px; line-height: 16px; }.character-count { position: absolute; right: 1px; bottom: -18px; color: #98a2b3; font-size: 10px; }
.calendar-picker { position: relative; }
.date-picker-button { width: 100%; height: 40px; padding: 0 12px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #d5dae1; border-radius: 7px; background: #fff; color: #667085; text-align: left; outline: none; box-shadow: 0 1px 2px rgba(16,24,40,.03); transition: border-color .15s, box-shadow .15s, background-color .15s; }
.date-picker-button:hover { border-color: #aeb6c2; background: #fcfcfd; }.date-picker-button:focus-visible { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1); }.date-picker-button.invalid { border-color: #e05252; }.date-picker-button.invalid:focus-visible { box-shadow: 0 0 0 3px rgba(220,38,38,.08); }.date-picker-button > svg { flex: 0 0 auto; color: #667085; }.date-placeholder { color: #a1a8b3; font-size: 13px; }.date-value { color: #111827; font-size: 13px; font-variant-numeric: tabular-nums; }
.calendar-popover { position: absolute; z-index: 30; top: calc(100% + 6px); left: 0; width: min(320px, calc(100vw - 32px)); padding: 12px; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; box-shadow: 0 14px 34px rgba(16,24,40,.16), 0 2px 8px rgba(16,24,40,.06); }
.calendar-toolbar { min-height: 34px; padding-bottom: 10px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #edf0f2; }.calendar-toolbar strong { color: #20252b; font-size: 13px; line-height: 20px; font-weight: 700; }.calendar-nav { width: 30px; height: 30px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid transparent; border-radius: 6px; color: #667085; }.calendar-nav:hover { border-color: #dfe3e8; background: #f4f5f7; color: #111827; }.calendar-nav:focus-visible, .calendar-day:focus-visible, .today-button:focus-visible { outline: 2px solid #667085; outline-offset: 1px; }
.calendar-weekdays, .calendar-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 3px; }.calendar-weekdays { margin-top: 10px; color: #98a2b3; font-size: 10px; line-height: 18px; text-align: center; }.calendar-grid { margin-top: 3px; }.calendar-day { aspect-ratio: 1; min-width: 0; display: inline-flex; align-items: center; justify-content: center; border-radius: 5px; color: #344054; font-size: 12px; line-height: 1; font-variant-numeric: tabular-nums; transition: background-color .12s, color .12s; }.calendar-day:hover { background: #f1f3f5; color: #111827; }.calendar-day.outside-month { color: #b5bbc3; }.calendar-day.today { box-shadow: inset 0 0 0 1px #98a2b3; }.calendar-day.selected { background: #20252b; color: #fff; box-shadow: none; font-weight: 700; }.calendar-day.selected:hover { background: #111827; color: #fff; }
.calendar-footer { margin-top: 10px; padding-top: 10px; display: flex; justify-content: flex-end; border-top: 1px solid #edf0f2; }.today-button { height: 28px; padding: 0 9px; border-radius: 5px; color: #475467; font-size: 11px; line-height: 16px; font-weight: 650; }.today-button:hover { background: #f1f3f5; color: #111827; }
.product-picker { position: relative; }.product-picker-button { width: 100%; min-height: 48px; padding: 7px 11px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #d5dae1; border-radius: 7px; background: #fff; color: #667085; text-align: left; outline: none; box-shadow: 0 1px 2px rgba(16,24,40,.03); transition: border-color .15s, box-shadow .15s; }.product-picker-button:hover { border-color: #aeb6c2; }.product-picker-button:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1); }.product-picker-button > svg { flex: 0 0 auto; }.placeholder { color: #98a2b3; font-size: 13px; }.selected-product { min-width: 0; display: flex; flex-direction: column; gap: 2px; }.selected-product strong, .selected-product small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.selected-product strong { color: #111827; font-size: 13px; line-height: 18px; font-weight: 650; }.selected-product small { color: #667085; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; line-height: 16px; }
.product-options { position: absolute; z-index: 20; top: calc(100% + 6px); width: 100%; overflow: hidden; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; box-shadow: 0 14px 34px rgba(16,24,40,.16), 0 2px 8px rgba(16,24,40,.06); }.product-search { position: relative; padding: 8px; border-bottom: 1px solid #e8ebef; }.product-search svg { position: absolute; left: 19px; top: 20px; color: #98a2b3; }.product-search input { width: 100%; height: 38px; border: 1px solid #d5dae1; border-radius: 6px; padding: 0 10px 0 34px; color: #111827; font-size: 12px; outline: none; }.product-search input:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.08); }.product-results { max-height: 260px; overflow-y: auto; padding: 5px; }.product-option { width: 100%; min-height: 50px; padding: 7px 9px; display: flex; align-items: center; justify-content: space-between; gap: 10px; border-radius: 5px; color: #475467; text-align: left; }.product-option:hover, .product-option.active { background: #f1f3f5; color: #111827; }.product-option > span { min-width: 0; display: flex; flex-direction: column; gap: 2px; }.product-option strong, .product-option small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.product-option strong { font-size: 12px; line-height: 18px; font-weight: 650; }.product-option small { color: #7a8493; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; line-height: 15px; }.product-result-state { min-height: 90px; display: flex; align-items: center; justify-content: center; gap: 7px; color: #7a8493; font-size: 12px; }
@media (max-width: 768px) {
  .purchase-page { padding: 16px 12px; }.page-header { min-height: 0; align-items: flex-start; }.page-header h1 { font-size: 21px; line-height: 28px; }.header-actions { flex-direction: column-reverse; align-items: stretch; }.header-actions button { width: 42px; overflow: hidden; padding: 0; gap: 0; font-size: 0; }.header-actions .quiet-button { color: #475467; }.header-actions .primary-button { color: #fff; }
  .desktop-table-wrap { display: none; }.mobile-list { display: block; }.plan-card { padding: 15px 14px; border-bottom: 1px solid #e8ebef; }.plan-card:last-child { border-bottom: 0; }.plan-card header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }.plan-card header .product-cell { min-width: 0; flex: 1; }.plan-card header time { flex: 0 0 auto; color: #475467; font-size: 11px; line-height: 17px; font-variant-numeric: tabular-nums; }.plan-card dl { margin-top: 13px; display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); border: 1px solid #e1e5ea; border-radius: 7px; background: #fafbfc; }.plan-card dl div { min-width: 0; padding: 9px 10px; border-right: 1px solid #e1e5ea; }.plan-card dl div:last-child { border-right: 0; }.plan-card dt { color: #7a8493; font-size: 10px; line-height: 15px; }.plan-card dd { margin-top: 2px; overflow: hidden; color: #111827; font-size: 14px; line-height: 20px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }.mobile-remark { margin-top: 11px; color: #475467; font-size: 12px; line-height: 18px; white-space: pre-wrap; word-break: break-word; }.plan-card footer { margin-top: 12px; padding-top: 10px; display: flex; justify-content: space-between; gap: 14px; border-top: 1px solid #edf0f2; color: #7a8493; font-size: 10px; line-height: 16px; }.plan-card footer span { color: #475467; font-weight: 600; }.mobile-state { min-height: 230px; padding: 32px 16px; display: flex; align-items: center; justify-content: center; gap: 7px; color: #667085; font-size: 12px; }.mobile-state.empty-state { flex-direction: column; }.mobile-state strong { color: #344054; font-size: 13px; }.mobile-state span { color: #98a2b3; }
  .quantity-grid { grid-template-columns: 1fr; gap: 0; }.form-drawer { width: 100vw; }.drawer-header, .form-body, .drawer-footer { padding-left: 16px; padding-right: 16px; }.calendar-popover { position: fixed; top: auto; right: 16px; bottom: 16px; left: 16px; width: auto; }
}
</style>
