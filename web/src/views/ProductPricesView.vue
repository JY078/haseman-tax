<template>
  <MainLayout>
    <div class="price-page">
      <header class="page-heading">
        <h1>商品价格</h1>
        <div class="page-links"><router-link to="/product-library"><CubeIcon />产品库</router-link></div>
      </header>
      <section class="price-surface" aria-label="商品价格台账">
        <div class="filter-bar">
          <div class="filters" aria-label="价格记录筛选"><button v-for="tab in filters" :key="tab.value" :class="{ active: filter === tab.value }" :aria-pressed="filter === tab.value" @click="setFilter(tab.value)">{{ tab.label }}<span v-if="filter === tab.value && !loading && !error" class="filter-count">{{ total }}</span></button></div>
          <span class="currency-label" title="按 SPU 排除 CA、EU、UK 前缀">美国站 · USD</span>
        </div>
        <div class="toolbar">
          <form class="search-form" role="search" @submit.prevent="search">
            <label class="search-field"><MagnifyingGlassIcon /><input v-model="query" aria-label="搜索 SKU、ASIN、PID 或产品" placeholder="SKU / ASIN / PID / 产品名称" /><button v-if="query" type="button" class="clear-search" aria-label="清空搜索" @click="clearSearch"><XMarkIcon /></button></label>
            <button class="search-button" type="submit" :disabled="loading">查询</button><button v-if="hasFilters" class="reset-button" type="button" :disabled="loading" @click="reset">重置</button>
          </form>
          <div class="toolbar-actions"><Button variant="outline" :disabled="fetching || loading" title="获取选中商品的亚马逊价格" @click="fetchAmazonPrices">{{ fetching ? `获取中 ${fetchProgress.done} / ${fetchProgress.total}` : '获取亚马逊价格' }}</Button><button class="refresh-button" :disabled="loading" aria-label="刷新价格记录" title="刷新价格记录" @click="load"><ArrowPathIcon :class="{ spinning: loading }" /></button></div>
        </div>
        <div v-if="fetchSummary" class="fetch-status" role="status"><span>{{ fetchSummary }}</span><button v-if="fetching" class="text-link" :disabled="fetchStopped" @click="stopAmazonPriceFetch">{{ fetchStopped ? '停止中…' : '停止获取' }}</button><details v-if="fetchFailures.length"><summary>查看未更新明细（{{ fetchFailures.length }}）</summary><ul><li v-for="failure in fetchFailures" :key="failure.id">{{ failure.sku }}：{{ failure.reason }}</li></ul></details></div>
        <div v-if="error" class="state-message" role="alert"><ExclamationCircleIcon /><h2>价格记录加载失败</h2><p>{{ error }}</p><Button variant="outline" @click="load">重新加载</Button></div>
        <div v-else-if="!loading && !items.length" class="state-message"><CubeIcon /><h2>{{ hasFilters ? '没有符合条件的记录' : '还没有产品记录' }}</h2><p>{{ hasFilters ? '调整关键词或筛选条件后重试。' : '先在产品库添加 SKU，再维护平台价格。' }}</p><Button v-if="hasFilters" variant="outline" @click="reset">查看全部商品</Button><router-link v-else to="/product-library" class="text-link">前往产品库</router-link></div>
        <div v-else class="table-scroll" tabindex="0" aria-label="双平台价格表，可横向滚动" :aria-busy="loading">
          <span v-if="loading" class="sr-only" role="status">正在读取商品价格</span>
          <table>
            <caption class="sr-only">商品价格台账，金额单位为美元</caption>
            <colgroup><col class="select-col" /><col class="product-col" /><col class="amazon-col" /><col class="price-col" /><col class="price-col" /><col class="price-col" /><col class="tiktok-col" /><col class="price-col" /><col class="date-col" /><col class="action-col" /></colgroup>
            <thead><tr class="platform-head"><th rowspan="2" scope="col" class="select-cell"><input type="checkbox" :checked="fetchableItems.length > 0 && selectedIds.size === fetchableItems.length" :disabled="!fetchableItems.length" aria-label="全选当前页已匹配 TK 的商品" @change="toggleAll($event.target.checked)" /></th><th rowspan="2" scope="col" class="sticky-product">产品信息</th><th colspan="4" scope="colgroup" class="platform-divider">Amazon</th><th colspan="2" scope="colgroup" class="platform-divider">TikTok</th><th rowspan="2" scope="col" class="platform-divider">最后更新</th><th rowspan="2" scope="col" class="sticky-action">操作</th></tr><tr class="column-head"><th scope="col" class="platform-divider">SKU / ASIN</th><th scope="col" class="numeric">Regular Price</th><th scope="col" class="numeric">Prime Member</th><th scope="col" class="numeric">Deal Price</th><th scope="col" class="platform-divider">SKU / PID</th><th scope="col" class="numeric">售价</th></tr></thead>
            <tbody v-if="loading" aria-hidden="true"><tr v-for="n in 8" :key="n" class="skeleton-row"><td v-for="column in 9" :key="column"><span class="skeleton"></span><span v-if="[1, 2, 6].includes(column)" class="skeleton short"></span></td></tr></tbody>
            <tbody v-else>
              <tr v-for="item in items" :key="rowKey(item)" :class="{ 'selected-row': editorOpen && rowKey(item) === rowKey(editingItem), 'saved-row': savedRowKey === rowKey(item) }">
                <td class="select-cell"><input type="checkbox" :checked="selectedIds.has(item.id)" :disabled="!item.mapping_id" :aria-label="item.mapping_id ? `选择 ${item.sku}` : `${item.sku} 未匹配 TK SKU`" @change="toggleSelected(item.id, $event.target.checked)" /></td><td class="sticky-product"><button class="product-identity product-history-trigger" type="button" @click="historyProduct = item"><ProductThumbnail :src="item.image_url" :alt="item.sku" /><span class="product-copy"><strong class="product-name" :title="productTitle(item)">{{ productTitle(item) }}</strong><span v-if="productDetails(item)" class="secondary product-detail" :title="productDetails(item)">{{ productDetails(item) }}</span></span></button></td>
                <td class="platform-divider"><div class="identifier-row"><span class="sku-code" :title="item.sku">{{ item.sku }}</span><button class="copy-button" :aria-label="`复制 Amazon SKU ${item.sku}`" title="复制 SKU" @click="copy(item.sku)"><DocumentDuplicateIcon /></button></div><a v-if="item.asin" class="identifier-link" :href="amazonLink(item.asin)" target="_blank" rel="noopener noreferrer" :aria-label="`打开 Amazon 商品 ${item.asin}`"><span class="identifier-label">ASIN</span>{{ item.asin }}<ArrowUpRightIcon /></a><span v-else class="missing-label">ASIN 待补全</span></td>
                <td class="numeric"><button class="price-trigger" :class="{ 'missing-price': item.amazon_price == null }" :aria-label="`编辑 Amazon Regular Price ${item.sku}`" @click="edit(item, 'amazon')">{{ item.amazon_price == null ? '待录入' : money(item.amazon_price) }}<PencilSquareIcon /></button></td>
                <td class="numeric"><button class="price-trigger" :class="{ 'sale-price': item.amazon_sale_price != null, 'empty-price': item.amazon_sale_price == null }" :aria-label="`编辑 Amazon Prime Member 价 ${item.sku}`" @click="edit(item, 'amazon', 'sale')">{{ item.amazon_sale_price == null ? '—' : money(item.amazon_sale_price) }}<PencilSquareIcon /></button></td>
                <td class="numeric"><button class="price-trigger" :class="{ 'sale-price': item.amazon_best_deal != null, 'empty-price': item.amazon_best_deal == null }" :aria-label="`编辑 Amazon Deal Price 价 ${item.sku}`" @click="edit(item, 'amazon', 'deal')">{{ item.amazon_best_deal == null ? '—' : money(item.amazon_best_deal) }}<PencilSquareIcon /></button></td>
                <td class="platform-divider"><template v-if="item.mapping_id"><div class="identifier-row"><span class="sku-code" :title="item.platform_sku">{{ item.platform_sku }}</span><button class="copy-button" :aria-label="`复制 TikTok SKU ${item.platform_sku}`" title="复制 SKU" @click="copy(item.platform_sku)"><DocumentDuplicateIcon /></button></div><a v-if="item.pid" class="identifier-link" :href="tiktokLink(item.pid)" target="_blank" rel="noopener noreferrer" :aria-label="`打开 TikTok 商品 ${item.pid}`"><span class="identifier-label">PID</span>{{ item.pid }}<ArrowUpRightIcon /></a><button v-else-if="canEditTikTok" class="missing-label missing-action" @click="edit(item, 'tiktok', 'pid')">补充 PID <PencilSquareIcon /></button><span v-else class="missing-label">PID 待补全</span></template><template v-else><span class="unmapped-label">未关联 TK SKU</span><button v-if="canEditTikTok" class="text-link" @click="mappingProduct = item">添加映射 <PlusIcon /></button></template></td>
                <td class="numeric"><button v-if="item.mapping_id && canEditTikTok" class="price-trigger" :class="{ 'missing-price': item.tiktok_price == null }" :aria-label="`编辑 TikTok 售价 ${item.platform_sku}`" @click="edit(item, 'tiktok')">{{ item.tiktok_price == null ? '待录入' : money(item.tiktok_price) }}<PencilSquareIcon /></button><span v-else class="readonly-price">{{ item.mapping_id ? money(item.tiktok_price) : '—' }}</span></td>
                <td class="platform-divider"><div class="update-line"><span>AMZ</span><time v-if="item.price_updated_at" :datetime="item.price_updated_at" :title="fullDate(item.price_updated_at)">{{ shortDate(item.price_updated_at) }}</time><span v-else class="never-updated">尚未更新</span></div><div class="update-line"><span>TK</span><time v-if="item.tiktok_updated_at" :datetime="item.tiktok_updated_at" :title="fullDate(item.tiktok_updated_at)">{{ shortDate(item.tiktok_updated_at) }}</time><span v-else class="never-updated">{{ item.mapping_id ? '尚未更新' : '—' }}</span></div></td>
                <td class="sticky-action"><button class="maintain-button" :aria-label="`维护 ${item.platform_sku || item.sku}`" @click="edit(item)">维护<ChevronRightIcon /></button><span v-if="savedRowKey === rowKey(item)" class="saved-label"><CheckIcon />已保存</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <footer class="table-footer"><span>{{ rangeLabel }}</span><div class="pagination"><label class="page-size">每页 <select v-model.number="pageSize" :disabled="loading" aria-label="每页条数" @change="changePageSize"><option :value="20">20 条</option><option :value="50">50 条</option><option :value="100">100 条</option></select></label><button :disabled="loading || page <= 1" aria-label="上一页" @click="changePage(-1)"><ChevronLeftIcon /></button><span>{{ page }} / {{ pages }}</span><button :disabled="loading || page >= pages" aria-label="下一页" @click="changePage(1)"><ChevronRightIcon /></button></div></footer>
      </section>
    </div>
    <ProductPriceMappingModal :product="mappingProduct" @close="mappingProduct = null" @saved="mappingSaved" />
    <ProductPriceHistoryModal v-model:open="historyOpen" :product="historyProduct" />
    <TransitionRoot as="template" :show="editorOpen">
      <Dialog as="div" class="price-dialog" :initial-focus="priceInput" @close="requestClose">
        <TransitionChild as="template" enter="drawer-fade" enter-from="opacity-0" enter-to="opacity-100" leave="drawer-fade" leave-from="opacity-100" leave-to="opacity-0"><div class="drawer-overlay" /></TransitionChild>
        <div class="drawer-position"><TransitionChild as="template" enter="drawer-slide" enter-from="translate-x-full" enter-to="translate-x-0" leave="drawer-slide" leave-from="translate-x-0" leave-to="translate-x-full">
          <DialogPanel class="price-drawer">
            <header class="drawer-header"><div><DialogTitle as="h2">维护商品价格</DialogTitle></div><button class="icon-button" :disabled="saving" aria-label="关闭价格维护" @click="requestClose"><XMarkIcon /></button></header>
            <div class="drawer-body">
              <section class="editor-product" aria-label="当前商品"><h3>{{ productTitle(editingItem) }}</h3><p v-if="productDetails(editingItem)">{{ productDetails(editingItem) }}</p><dl><div><dt>Amazon SKU</dt><dd>{{ editingItem?.sku }}</dd></div><div><dt>TikTok SKU</dt><dd>{{ editingItem?.platform_sku || '未关联' }}</dd></div></dl></section>
              <div class="editor-tabs" aria-label="维护平台"><button :class="{ active: editingPlatform === 'amazon' }" :aria-pressed="editingPlatform === 'amazon'" :disabled="saving" @click="switchPlatform('amazon')">Amazon</button><button :class="{ active: editingPlatform === 'tiktok' }" :aria-pressed="editingPlatform === 'tiktok'" :disabled="saving || !editingItem?.mapping_id" @click="switchPlatform('tiktok')">TikTok <span v-if="!editingItem?.mapping_id || !canEditTikTok">{{ !editingItem?.mapping_id ? '未关联' : '只读' }}</span></button></div>
              <form id="price-editor" ref="editorForm" novalidate @submit.prevent="save(false)" @keydown.ctrl.enter.prevent="save(false)" @keydown.meta.enter.prevent="save(false)" @input="saveError = ''; savedMessage = ''">
                <div class="editor-fields">
                  <label v-if="editingPlatform === 'tiktok'" for="price-pid">商品 PID <span class="field-unit">选填</span><input id="price-pid" ref="pidInput" v-model.trim="form.pid" type="text" inputmode="numeric" maxlength="32" placeholder="输入商品链接末尾的数字编号" :disabled="saving || editorReadOnly" :aria-invalid="!!fieldErrors.pid" :aria-describedby="fieldErrors.pid ? 'pid-error' : undefined" /><small v-if="fieldErrors.pid" id="pid-error" class="field-error">{{ fieldErrors.pid }}</small></label>
                  <label for="price-amount">{{ editingPlatform === 'amazon' ? 'Amazon Regular Price' : 'TikTok 售价' }}<span class="field-unit">USD</span><div class="money-input"><span>$</span><input id="price-amount" ref="priceInput" v-model="form.price" type="text" inputmode="decimal" placeholder="0.00" :disabled="saving || editorReadOnly" :aria-invalid="!!fieldErrors.price" :aria-describedby="fieldErrors.price ? 'price-error' : undefined" /></div><small v-if="fieldErrors.price" id="price-error" class="field-error">{{ fieldErrors.price }}</small></label>
                  <label v-if="editingPlatform === 'amazon'" for="price-sale">Amazon Prime Member <span class="field-unit">选填 · USD</span><div class="money-input"><span>$</span><input id="price-sale" ref="saleInput" v-model="form.salePrice" type="text" inputmode="decimal" placeholder="无会员价时留空" :disabled="saving" :aria-invalid="!!fieldErrors.salePrice" :aria-describedby="fieldErrors.salePrice ? 'sale-error' : undefined" /></div><small v-if="fieldErrors.salePrice" id="sale-error" class="field-error">{{ fieldErrors.salePrice }}</small></label>
                  <label v-if="editingPlatform === 'amazon'" for="price-deal">Amazon Deal Price <span class="field-unit">选填 · USD</span><div class="money-input"><span>$</span><input id="price-deal" ref="dealInput" v-model="form.bestDeal" type="text" inputmode="decimal" placeholder="无 Deal Price 时留空" :disabled="saving" :aria-invalid="!!fieldErrors.bestDeal" :aria-describedby="fieldErrors.bestDeal ? 'deal-error' : undefined" /></div><small v-if="fieldErrors.bestDeal" id="deal-error" class="field-error">{{ fieldErrors.bestDeal }}</small></label>
                </div>
                <div class="price-preview"><span>{{ editingPlatform === 'amazon' ? '有效售价' : '平台售价' }}</span><strong>{{ previewPrice }}</strong></div>
                <div class="editor-audit"><ClockIcon /><span>上次更新：{{ editorUpdatedAt ? fullDate(editorUpdatedAt) : '尚未录入' }}</span><a v-if="editorProductLink" :href="editorProductLink" target="_blank" rel="noopener noreferrer">打开商品<ArrowUpRightIcon /></a></div>
                <p v-if="editorReadOnly" class="editor-hint">TikTok 仅可查看</p>
                <p v-if="saveError" class="form-error" role="alert">{{ saveError }}</p><p v-if="savedMessage" class="save-success" role="status"><CheckCircleIcon />{{ savedMessage }}</p>
              </form>
            </div>
            <footer class="drawer-footer">
              <div v-if="pendingAction" class="discard-confirm" role="alert"><strong>有尚未保存的修改</strong><p>继续操作将放弃本次修改。</p><div><button class="secondary-button" @click="cancelPending">继续编辑</button><button class="discard-button" @click="discardPending">放弃修改并继续</button></div></div>
              <template v-else><div class="record-navigation"><span>本页{{ editingPlatform === 'amazon' ? ' Amazon SKU' : ' TK SKU' }} <b>{{ editorIndex + 1 }} / {{ editorRecords.length }}</b></span><div><button class="icon-button" :disabled="saving || editorIndex <= 0" aria-label="维护上一条" @click="moveEditor(-1)"><ChevronLeftIcon /></button><button class="icon-button" :disabled="saving || !nextRecord" aria-label="维护下一条" @click="moveEditor(1)"><ChevronRightIcon /></button></div></div><div class="save-actions"><span class="draft-state">{{ saving ? '正在保存…' : isDirty ? '未保存' : '' }}</span><button class="secondary-button" :disabled="saving || editorReadOnly || !isDirty || !nextRecord" @click="save(true)">保存并下一条</button><button class="primary-button" title="Ctrl / ⌘ + Enter 保存" type="submit" form="price-editor" :disabled="saving || editorReadOnly || !isDirty">{{ saving ? '保存中…' : '保存' }}</button></div></template>
            </footer>
          </DialogPanel>
        </TransitionChild></div>
      </Dialog>
    </TransitionRoot>
  </MainLayout>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { ArrowPathIcon, ArrowUpRightIcon, CheckCircleIcon, CheckIcon, ChevronLeftIcon, ChevronRightIcon, ClockIcon, CubeIcon, DocumentDuplicateIcon, ExclamationCircleIcon, MagnifyingGlassIcon, PencilSquareIcon, PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import Button from '@/components/ui/Button.vue'
import ProductPriceHistoryModal from '@/components/ProductPriceHistoryModal.vue'
import ProductPriceMappingModal from '@/components/ProductPriceMappingModal.vue'
import ProductThumbnail from '@/components/ui/ProductThumbnail.vue'
import request from '@/utils/request'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const fetching = ref(false)
const selectedIds = ref(new Set())
const fetchableItems = computed(() => items.value.filter(item => item.mapping_id))
const historyProduct = ref(null)
const historyOpen = computed({ get: () => Boolean(historyProduct.value), set: value => { if (!value) historyProduct.value = null } })
function toggleSelected(id, checked) { const next = new Set(selectedIds.value); checked ? next.add(id) : next.delete(id); selectedIds.value = next }
function toggleAll(checked) { selectedIds.value = checked ? new Set(fetchableItems.value.map(item => item.id)) : new Set() }
const fetchStopped = ref(false)
const fetchFailures = ref([])
const fetchSummary = ref('')
const fetchProgress = reactive({ done: 0, total: 0, success: 0 })
const fetchJobId = ref('')
async function fetchAmazonPrices() {
  if (fetching.value) return
  fetching.value = true
  fetchStopped.value = false
  fetchFailures.value = []
  Object.assign(fetchProgress, { done: 0, total: 0, success: 0 })
  fetchSummary.value = '正在提交亚马逊价格批量任务…'
  try {
    // Firecrawl scrapes the whole list server side through one batch job, so
    // no request here has to stay open for the entire run.
    const ids = [...selectedIds.value]
    const { data } = await request.post('/product-prices/fetch-batch', ids.length ? { ids } : {})
    if (!data.started) {
      fetchSummary.value = '没有可获取的美国站商品'
      await load()
      return
    }
    fetchJobId.value = data.job.id
    fetchProgress.total = data.job.total
    fetchSummary.value = `已提交 ${data.job.total} 个商品，等待亚马逊返回…`
    let job = data.job
    while (job.status === 'running' || job.status === 'cancelling') {
      await new Promise(resolve => setTimeout(resolve, 1500))
      const { data: polled } = await request.get(`/product-prices/fetch-batch/${fetchJobId.value}`)
      job = polled.job
      Object.assign(fetchProgress, { done: job.done, total: job.total, success: job.success })
      fetchFailures.value = job.failures || []
      fetchSummary.value = job.status === 'cancelling' ? '正在停止…' : `正在获取 · ${job.done} / ${job.total}${job.asins && job.asins !== job.total ? `（已抓 ${job.scraped}/${job.asins} 个商品页）` : ''}`
    }
    Object.assign(fetchProgress, { done: job.done, total: job.total, success: job.success })
    fetchFailures.value = job.failures || []
    const pending = job.total - job.done
    fetchSummary.value = `${job.status === 'cancelled' ? '已停止' : '获取完成'} · 更新 ${job.success}，未更新 ${fetchFailures.value.length}${pending ? `，待处理 ${pending}` : ''}`
    await load()
  } catch (err) {
    fetchSummary.value = err.response?.data?.error || '无法启动批量获取，请稍后重试'
  } finally { fetching.value = false; fetchJobId.value = '' }
}
async function stopAmazonPriceFetch() {
  fetchStopped.value = true
  fetchSummary.value = '正在停止…'
  try { await request.post(`/product-prices/fetch-batch/${fetchJobId.value}/cancel`, {}) }
  catch { /* the poll loop already shows the job once it stops */ }
}
const mappingProduct = ref(null)
async function mappingSaved() { mappingProduct.value = null; toast.success('映射已保存'); await load() }
function productTitle(item) { return item?.name?.trim() || item?.spu || item?.sku || '' }
function productDetails(item) { return [...new Set([item?.spu, item?.color, item?.size].filter(value => value && value !== productTitle(item)))].join(' / ') }
const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const rangeLabel = computed(() => loading.value ? '正在加载…' : error.value ? '加载失败' : total.value ? `第 ${(page.value - 1) * pageSize.value + 1}–${Math.min(page.value * pageSize.value, total.value)} 条，共 ${total.value} 条` : '共 0 条记录')
const query = ref('')
const appliedQuery = ref('')
const filter = ref('all')
const filters = [{ value: 'all', label: '全部商品' }, { value: 'missing', label: '待补全' }, { value: 'sale', label: '有折扣' }, { value: 'unmapped', label: '未关联 TK' }]
const hasFilters = computed(() => appliedQuery.value || filter.value !== 'all')
const loading = ref(false)
const error = ref('')
const canEditTikTok = ref(false)
const editorOpen = ref(false)
const editingItem = ref(null)
const editingPlatform = ref('amazon')
const form = reactive({ price: '', salePrice: '', bestDeal: '', pid: '' })
const initialForm = ref('')
const isDirty = computed(() => editorOpen.value && JSON.stringify(form) !== initialForm.value)
const saving = ref(false)
const saveError = ref('')
const savedMessage = ref('')
const savedRowKey = ref('')
const validationAttempted = ref(false)
const pendingAction = ref(null)
const editorForm = ref(null)
const priceInput = ref(null)
const saleInput = ref(null)
const dealInput = ref(null)
const pidInput = ref(null)
const editorReadOnly = computed(() => editingPlatform.value === 'tiktok' && !canEditTikTok.value)
let pendingCancel = null
let refreshAfterEdit = false
let loadSequence = 0
let savedTimer

const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })
const money = value => value == null ? '—' : currency.format(Number(value))
const amazonLink = asin => `https://www.amazon.com/dp/${encodeURIComponent(asin)}`
const tiktokLink = pid => `https://www.tiktok.com/view/product/${encodeURIComponent(pid)}`
const fullDate = value => new Intl.DateTimeFormat('sv-SE', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(value))
const shortDate = value => fullDate(value).slice(2)
const rowKey = item => item ? `${item.id}-${item.mapping_id || 0}` : ''
const editorUpdatedAt = computed(() => editingPlatform.value === 'amazon' ? editingItem.value?.price_updated_at : editingItem.value?.tiktok_updated_at)
const editorProductLink = computed(() => editingPlatform.value === 'amazon' ? (editingItem.value?.asin ? amazonLink(editingItem.value.asin) : '') : (/^[0-9]{1,32}$/.test(form.pid) ? tiktokLink(form.pid) : ''))
const editorRecords = computed(() => {
  const seen = new Set()
  return items.value.filter(item => {
    const key = editingPlatform.value === 'amazon' ? item.id : item.mapping_id
    if (!key || seen.has(key)) return false
    seen.add(key)
    return true
  })
})
const editorIndex = computed(() => editorRecords.value.findIndex(item => editingPlatform.value === 'amazon' ? item.id === editingItem.value?.id : item.mapping_id === editingItem.value?.mapping_id))
const nextRecord = computed(() => editorRecords.value[editorIndex.value + 1])
const pricePattern = /^[0-9]{1,8}(\.[0-9]{1,2})?$/
const validationErrors = computed(() => {
  const errors = {}
  const price = form.price.trim()
  const sale = form.salePrice.trim()
  const deal = form.bestDeal.trim()
  if (price && !pricePattern.test(price)) errors.price = '输入 0–99999999.99 的金额，最多两位小数。'
  if (editingPlatform.value === 'amazon' && sale) {
    if (!pricePattern.test(sale)) errors.salePrice = 'Prime Member 价最多两位小数，且不能为负数。'
    else if (!price || Number(sale) > Number(price)) errors.salePrice = '请填写 Regular Price，且 Prime Member 价不能高于它。'
  }
  if (editingPlatform.value === 'amazon' && deal && !pricePattern.test(deal)) errors.bestDeal = 'Deal Price 价最多两位小数，且不能为负数。'
  if (editingPlatform.value === 'tiktok' && form.pid && !/^[0-9]{1,32}$/.test(form.pid)) errors.pid = '请输入最多 32 位数字，不要填写完整链接。'
  return errors
})
const fieldErrors = computed(() => validationAttempted.value ? validationErrors.value : {})
const previewPrice = computed(() => {
  if (Object.keys(validationErrors.value).some(key => key !== 'pid')) return '—'
  if (editingPlatform.value !== 'amazon') return form.price.trim() ? money(form.price.trim()) : '待录入'
  // A shopper pays whichever Amazon price is lowest right now.
  const candidates = [form.price.trim(), form.salePrice.trim(), form.bestDeal.trim()].filter(Boolean)
  return candidates.length ? money(String(Math.min(...candidates.map(Number)))) : '待录入'
})

async function load() {
  const sequence = ++loadSequence
  loading.value = true
  error.value = ''
  try {
    const { data } = await request.get('/product-prices', { params: { q: appliedQuery.value, filter: filter.value, page: page.value, page_size: pageSize.value } })
    if (sequence !== loadSequence) return
    items.value = data.items
    total.value = data.total
    canEditTikTok.value = data.can_edit_tiktok
    if (page.value > pages.value) { page.value = pages.value; return load() }
  } catch (err) {
    if (sequence === loadSequence) error.value = err.response?.data?.error || '请检查网络后重试。'
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}
function search() { appliedQuery.value = query.value.trim(); page.value = 1; load() }
function clearSearch() { query.value = ''; search() }
function setFilter(value) { if (filter.value === value) return; filter.value = value; page.value = 1; load() }
function reset() { query.value = ''; appliedQuery.value = ''; filter.value = 'all'; page.value = 1; load() }
function changePage(offset) { page.value += offset; load() }
function changePageSize() { page.value = 1; load() }
async function copy(value) {
  try { await navigator.clipboard.writeText(value); toast.success('编号已复制') }
  catch { toast.error('复制失败，请选中编号手动复制') }
}
function fillForm() {
  const item = editingItem.value
  form.price = String((editingPlatform.value === 'amazon' ? item.amazon_price : item.tiktok_price) ?? '')
  form.salePrice = String(item.amazon_sale_price ?? '')
  form.bestDeal = String(item.amazon_best_deal ?? '')
  form.pid = item.pid || ''
  initialForm.value = JSON.stringify(form)
  validationAttempted.value = false
  saveError.value = ''
  savedMessage.value = ''
}
async function focusField(field = 'price') {
  await nextTick()
  const input = field === 'sale' ? saleInput.value : field === 'deal' ? dealInput.value : field === 'pid' ? pidInput.value : priceInput.value
  input?.focus()
  input?.select()
}
function edit(item, platform = 'amazon', field = 'price') {
  editingItem.value = item
  editingPlatform.value = platform
  fillForm()
  editorOpen.value = true
  focusField(field)
}
function guardEdit(action) {
  if (saving.value || pendingAction.value) return
  if (isDirty.value) pendingAction.value = action
  else action()
}
function cancelPending() { pendingAction.value = null; pendingCancel?.(); pendingCancel = null; focusField() }
function discardPending() { const action = pendingAction.value; pendingAction.value = null; pendingCancel = null; action?.() }
function requestClose() {
  guardEdit(() => {
    editorOpen.value = false
    if (refreshAfterEdit) { refreshAfterEdit = false; load() }
  })
}
function switchPlatform(platform) {
  if (editingPlatform.value === platform) return
  guardEdit(() => { editingPlatform.value = platform; fillForm(); focusField() })
}
function moveEditor(offset) {
  const target = editorRecords.value[editorIndex.value + offset]
  if (target) guardEdit(() => { editingItem.value = target; fillForm(); focusField() })
}
async function save(andNext = false) {
  if (saving.value || editorReadOnly.value || !isDirty.value || pendingAction.value) return
  validationAttempted.value = true
  if (Object.keys(validationErrors.value).length) {
    await nextTick()
    editorForm.value?.querySelector('[aria-invalid="true"]')?.focus()
    return
  }
  saveError.value = ''
  const isAmazon = editingPlatform.value === 'amazon'
  const price = form.price.trim() || null
  const salePrice = form.salePrice.trim() || null
  const bestDeal = form.bestDeal.trim() || null
  const item = editingItem.value
  const target = andNext ? nextRecord.value : null
  saving.value = true
  try {
    const id = isAmazon ? item.id : item.mapping_id
    const { data } = await request.put(`/product-prices/${editingPlatform.value}/${id}`, isAmazon ? { price, sale_price: salePrice, best_deal: bestDeal } : { price, pid: form.pid })
    const updates = isAmazon ? { amazon_price: price, amazon_sale_price: salePrice, amazon_best_deal: bestDeal, price_updated_at: data.updated_at } : { tiktok_price: price, pid: form.pid, tiktok_updated_at: data.updated_at }
    for (const row of items.value) {
      if (isAmazon ? row.id === item.id : row.mapping_id === item.mapping_id) Object.assign(row, updates)
    }
    savedRowKey.value = rowKey(item)
    clearTimeout(savedTimer)
    savedTimer = setTimeout(() => { savedRowKey.value = '' }, 3500)
    refreshAfterEdit = true
    fillForm()
    savedMessage.value = '已保存，更新时间已同步'
    if (target) { editingItem.value = target; fillForm(); savedMessage.value = '上一条已保存，可继续维护当前商品'; focusField() }
  } catch (err) { saveError.value = err.response?.data?.error || '保存失败，输入内容已保留，请重试。' }
  finally { saving.value = false }
}
function beforeUnload(event) { if (isDirty.value || saving.value || fetching.value) { event.preventDefault(); event.returnValue = '' } }
onBeforeRouteLeave(() => {
  if (fetching.value) { toast.error('请等待获取完成，或先停止获取'); return false }
  if (saving.value || pendingAction.value) return false
  if (!isDirty.value) return true
  return new Promise(resolve => { pendingAction.value = () => resolve(true); pendingCancel = () => resolve(false) })
})
onMounted(() => { load(); window.addEventListener('beforeunload', beforeUnload) })
onBeforeUnmount(() => { ++loadSequence; clearTimeout(savedTimer); pendingCancel?.(); window.removeEventListener('beforeunload', beforeUnload) })
</script>

<style scoped>
.product-identity { display: flex; align-items: center; gap: 12px; min-width: 0; }
.product-copy { min-width: 0; flex: 1; }

.fetch-status { padding: 12px 20px; border-top: 1px solid #e2e6e3; background: #f5f8f6; color: #46534b; font-size: 12px; overflow-wrap: anywhere; }
.fetch-status > button { margin-left: 16px; }.fetch-status details { margin-top: 8px; }.fetch-status summary { cursor: pointer; }.fetch-status ul { max-height: 180px; overflow-y: auto; margin-top: 8px; padding-left: 18px; }.fetch-status li { margin: 4px 0; }
.price-page { --line: #e2e6e3; --muted: #758078; --accent: #246a50; max-width: 1680px; margin: 0 auto; padding: 24px 24px 20px; color: #25302b; font-size: 13px; }
.page-heading, .page-links, .filter-bar, .toolbar, .toolbar-actions, .search-form, .search-field, .table-footer, .pagination { display: flex; align-items: center; }
.page-heading { justify-content: space-between; gap: 16px; margin-bottom: 19px; }h1 { font-size: 24px; font-weight: 650; letter-spacing: -.5px; }
.page-links { gap: 15px; }.page-links a { display: flex; align-items: center; gap: 6px; color: #66736a; font-size: 12px; padding: 7px 0; }.page-links a:hover { color: var(--accent); }svg { width: 16px; height: 16px; flex-shrink: 0; }
.price-surface { border: 1px solid var(--line); border-radius: 10px; background: #fff; overflow: hidden; }.filter-bar { justify-content: space-between; gap: 12px; padding: 0 20px; border-bottom: 1px solid var(--line); }.filters { display: flex; align-self: stretch; gap: 26px; }.filters button { display: flex; align-items: center; gap: 7px; min-height: 49px; border-bottom: 2px solid transparent; padding-top: 2px; color: #68716c; white-space: nowrap; }.filters button.active { border-bottom-color: var(--accent); color: #205a44; font-weight: 600; }.filter-count { padding: 1px 6px; border-radius: 4px; font-size: 11px; line-height: 17px; background: #edf3ef; font-variant-numeric: tabular-nums; }.currency-label { color: #45594b; font-size: 12px; font-weight: 600; white-space: nowrap; }
.toolbar { justify-content: space-between; gap: 12px; padding: 14px 20px; }.search-form { gap: 8px; }.search-field { gap: 8px; border: 1px solid #d6ded8; border-radius: 5px; padding: 0 10px; color: #8b968e; }.search-field input { height: 34px; width: 280px; min-width: 0; background: transparent; color: #25302b; outline: none; font-size: 12px; }.clear-search { color: #8b968e; }.clear-search svg { width: 13px; height: 13px; }.search-button, .refresh-button, .reset-button { min-height: 36px; border-radius: 5px; padding: 0 13px; font-size: 12px; }.search-button { background: #2d5945; color: #fff; }.search-button:hover { background: #234735; }.reset-button { color: #7d887f; padding: 0 7px; }.refresh-button { border: 1px solid var(--line); padding: 0 9px; color: #64756a; }.toolbar-actions { gap: 17px; }

.table-scroll { overflow: auto; max-height: max(280px, calc(100dvh - 340px)); scrollbar-gutter: stable; overscroll-behavior: contain; }table { width: 100%; min-width: 1260px; table-layout: fixed; border-collapse: separate; border-spacing: 0; text-align: left; }.select-col { width: 42px; }.select-cell { width: 42px; text-align: center; }.product-col { width: 16%; }.amazon-col { width: 17%; }.price-col { width: 8.3%; }.tiktok-col { width: 19%; }.date-col { width: 16%; }.action-col { width: 7.1%; }
th, td { padding: 12px 14px; }thead th { position: sticky; background: #f3f5f4; border-bottom: 1px solid var(--line); font-weight: 500; z-index: 2; }.platform-head th { top: 0; height: 36px; padding-top: 8px; padding-bottom: 8px; font-size: 12px; border-top: 1px solid var(--line); }.platform-head th[rowspan] { color: #637169; }.platform-head span { color: #909a93; font-size: 11px; margin-left: 8px; }.column-head th { top: 36px; height: 34px; padding-top: 7px; padding-bottom: 7px; color: #7a867e; font-size: 11px; background: #fafbfa; }.platform-divider { border-left: 1px solid #e8ece9; }
td { height: 70px; vertical-align: middle; border-bottom: 1px solid #edf0ed; background: #fff; }tbody tr:last-child td { border-bottom: 0; }.sticky-product { position: sticky; left: 0; z-index: 1; box-shadow: 1px 0 #e8ece9; }.sticky-action { position: sticky; right: 0; z-index: 1; border-left: 1px solid #e8ece9; text-align: center; }thead .sticky-product, thead .sticky-action { z-index: 3; }tbody tr:hover td { background: #f8fbf9; }tbody .selected-row td, tbody .saved-row td { background: #f0f7f2; }
.product-name { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; font-weight: 550; }.secondary { display: block; font-size: 11px; color: #7c857f; margin-top: 7px; }.product-detail { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.identifier-row { display: flex; align-items: center; gap: 5px; }.sku-code { font-size: 13px; line-height: 1.6; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }.copy-button { display: inline-flex; padding: 3px; color: #a3ada6; border-radius: 3px; }.copy-button svg { width: 12px; height: 12px; }.copy-button:hover { color: #246a50; background: #e6f0e9; }.identifier-link { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: #4b7964; margin-top: 6px; white-space: nowrap; }.identifier-link svg { width: 12px; height: 12px; }.identifier-link:hover { text-decoration: underline; }.identifier-label { color: #9aa49d; font-size: 9px; letter-spacing: .2px; }.numeric { text-align: right; }.price-trigger { display: inline-flex; align-items: center; gap: 5px; position: relative; min-height: 30px; padding: 3px 0 3px 5px; color: #2d3932; font-size: 14px; font-weight: 600; white-space: nowrap; font-variant-numeric: tabular-nums; }.price-trigger svg { width: 11px; height: 11px; color: #8a9a8e; opacity: 0; position: absolute; left: -10px; }.price-trigger:hover svg, .price-trigger:focus-visible svg { opacity: 1; }.price-trigger:hover { color: var(--accent); text-decoration: underline; }.sale-price { color: #286b4f; }.missing-price { color: #a27c45; font-size: 12px; font-weight: 400; }.empty-price { color: #a7afa9; }.readonly-price { font-size: 13px; color: #7c8a80; font-variant-numeric: tabular-nums; }.missing-label { display: flex; align-items: center; gap: 5px; margin-top: 6px; font-size: 11px; color: #a17b43; }.missing-action:hover { text-decoration: underline; }.missing-action svg { width: 11px; height: 11px; }.unmapped-label { font-size: 12px; color: #8b958e; }.text-link { display: flex; align-items: center; gap: 4px; font-size: 11px; color: #527b62; margin-top: 5px; }.text-link svg { width: 12px; height: 12px; }
.update-line { display: flex; align-items: center; gap: 8px; font-size: 12px; white-space: nowrap; color: #738077; line-height: 1.8; }.update-line + .update-line { margin-top: 3px; }.update-line > span:first-child { width: 26px; font-size: 9px; color: #849188; font-weight: 500; }.update-line time { font-variant-numeric: tabular-nums; }.never-updated { color: #a0aaa3; }.maintain-button { display: inline-flex; align-items: center; gap: 3px; color: #39644c; font-size: 12px; padding: 6px 0; white-space: nowrap; }.maintain-button svg { width: 12px; height: 12px; }.maintain-button:hover { color: #173c28; }.saved-label { display: flex; align-items: center; justify-content: center; gap: 2px; color: #428160; font-size: 10px; }.saved-label svg { width: 11px; height: 11px; }
.table-footer { justify-content: space-between; gap: 14px; border-top: 1px solid var(--line); padding: 11px 20px; font-size: 12px; color: #7c8980; }.pagination { gap: 13px; white-space: nowrap; font-variant-numeric: tabular-nums; }.pagination button { padding: 5px; border: 1px solid var(--line); border-radius: 4px; }.page-size { display: flex; align-items: center; gap: 8px; margin-right: 10px; font-size: 11px; }.page-size select { height: 28px; padding: 0 7px; border: 1px solid var(--line); border-radius: 4px; background: #fff; color: #55675a; }
.state-message { min-height: 320px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; border-top: 1px solid var(--line); padding: 30px; text-align: center; color: #849188; }.state-message > svg { width: 32px; height: 32px; color: #9eafa3; }.state-message h2 { color: #52655a; font-size: 15px; font-weight: 500; }.skeleton-row td { height: 70px; }.skeleton { display: block; height: 10px; background: #eef2ee; border-radius: 3px; width: 80%; animation: pulse 1.4s ease-in-out infinite; }.skeleton.short { width: 50%; margin-top: 12px; height: 7px; }.spinning { animation: spin 1s linear infinite; }
:global(.price-dialog) { position: fixed; inset: 0; z-index: 50; font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif; color: #29332d; }.drawer-overlay { position: fixed; inset: 0; background: rgb(24 39 29 / .18); }.drawer-position { position: fixed; inset: 0; display: flex; justify-content: flex-end; pointer-events: none; }.price-drawer { width: min(100%, 460px); height: 100dvh; background: #fff; display: flex; flex-direction: column; pointer-events: auto; box-shadow: -8px 0 35px rgb(34 52 41 / .08); }.drawer-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding: 20px 24px; border-bottom: 1px solid #e7ece8; }.drawer-header h2 { font-size: 18px; font-weight: 600; }.icon-button { display: inline-flex; align-items: center; justify-content: center; padding: 6px; border-radius: 4px; color: #78897c; }.icon-button:hover { background: #f0f4f0; }.icon-button svg { width: 18px; height: 18px; }.drawer-body { flex: 1; min-height: 0; overflow-y: auto; padding: 22px 24px; overscroll-behavior: contain; }.editor-product h3 { font-size: 17px; font-weight: 600; margin-top: 0; overflow-wrap: anywhere; }.editor-product > p { font-size: 11px; color: #7f8982; margin-top: 5px; }.editor-product dl { margin-top: 16px; }.editor-product dl > div { display: grid; grid-template-columns: 84px 1fr; gap: 8px; font-size: 12px; margin-top: 8px; line-height: 1.6; }.editor-product dt { color: #7f8982; }.editor-product dd { color: #526159; overflow-wrap: anywhere; }
.editor-tabs { display: grid; grid-template-columns: 1fr 1fr; padding: 3px; margin: 24px 0 22px; border-radius: 6px; background: #f2f5f2; }.editor-tabs button { display: flex; align-items: center; justify-content: center; gap: 8px; min-height: 37px; border: 1px solid transparent; border-radius: 4px; color: #68776d; font-size: 13px; font-weight: 550; }.editor-tabs button.active { background: #fff; color: #2b6348; border-color: #dce5dd; box-shadow: 0 1px 2px rgb(29 56 38 / .04); }.editor-tabs span { font-size: 10px; color: #7c8980; font-weight: 400; }.editor-fields { display: grid; gap: 22px; }.editor-fields label { display: block; color: #46534b; font-size: 13px; font-weight: 500; }.field-unit { float: right; font-size: 10px; font-weight: 400; color: #849188; }.editor-fields input { display: block; width: 100%; height: 42px; margin-top: 8px; padding: 0 12px; border: 1px solid #dce4dc; border-radius: 5px; color: #2d4333; font-size: 14px; font-weight: 400; background: #fff; outline: none; font-variant-numeric: tabular-nums; }.editor-fields input:disabled { background: #f7f9f7; color: #8a998e; }.editor-fields input:focus { border-color: #568368; box-shadow: 0 0 0 3px #eef4ef; }.money-input { position: relative; }.money-input > span { position: absolute; left: 13px; top: 12px; color: #849188; font-size: 14px; font-weight: 400; }.money-input input { padding-left: 30px; }.editor-fields small { display: block; color: #94a098; font-size: 10px; line-height: 1.7; margin-top: 7px; font-weight: 400; }.editor-fields .field-error { color: #b76541; }.editor-fields input[aria-invalid="true"] { border-color: #c99477; }.price-preview { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 24px; padding: 17px 16px; border-radius: 6px; background: #f5f8f5; }.price-preview span { font-size: 12px; color: #748279; }.price-preview strong { color: #37684b; font-size: 23px; font-weight: 600; font-variant-numeric: tabular-nums; }.editor-audit { display: flex; flex-wrap: wrap; align-items: center; gap: 5px; color: #849188; font-size: 11px; margin-top: 17px; }.editor-audit svg { width: 12px; height: 12px; }.editor-audit a { display: inline-flex; align-items: center; gap: 3px; margin-left: auto; color: #527f5f; }.editor-hint { font-size: 10px; line-height: 1.8; color: #849188; margin-top: 9px; }.form-error { margin-top: 15px; color: #b36547; font-size: 12px; line-height: 1.7; }.save-success { display: flex; align-items: center; gap: 6px; margin-top: 15px; color: #4f8460; font-size: 11px; }
.drawer-footer { flex-shrink: 0; border-top: 1px solid #e4ebe5; padding: 14px 24px 20px; background: #fff; }.record-navigation { display: flex; justify-content: space-between; align-items: center; margin-bottom: 13px; color: #7c8980; font-size: 10px; }.record-navigation b { color: #59765f; font-weight: 500; margin-left: 5px; }.record-navigation > div { display: flex; gap: 4px; }.record-navigation .icon-button { border: 1px solid #e1e8e1; padding: 4px; }.record-navigation svg { width: 14px; height: 14px; }.save-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }.draft-state { margin-right: auto; font-size: 10px; color: #85968a; }.primary-button, .secondary-button, .discard-button { min-height: 36px; padding: 0 13px; border-radius: 5px; font-size: 12px; white-space: nowrap; }.primary-button { background: #315e44; border: 1px solid #315e44; color: #fff; min-width: 67px; }.primary-button:hover { background: #244c35; }.secondary-button { border: 1px solid #dbe4dc; color: #59735f; background: #fff; }.secondary-button:hover { background: #f0f5f0; }.discard-confirm strong { font-size: 13px; font-weight: 550; color: #876c45; }.discard-confirm p { font-size: 11px; color: #9a8c78; margin-top: 5px; }.discard-confirm > div { display: flex; justify-content: flex-end; gap: 8px; margin-top: 14px; }.discard-button { border: 1px solid #d4bfa2; color: #89643d; background: #fcf9f4; }
button { cursor: pointer; transition: background-color .15s, color .15s, border-color .15s; }button:disabled { opacity: .4; cursor: not-allowed; }button:focus-visible, a:focus-visible, select:focus-visible, .table-scroll:focus-visible { outline: 2px solid #578168; outline-offset: 3px; }.search-field:focus-within { border-color: #618773; box-shadow: 0 0 0 2px #edf3ef; }.drawer-slide { transition: transform .2s ease; }.drawer-fade { transition: opacity .2s ease; }@keyframes pulse { 50% { opacity: .4; } }@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 767px) { .price-page { padding: 17px 12px; }.page-heading { margin-bottom: 15px; }h1 { font-size: 21px; }.page-links { gap: 10px; }.page-links a { font-size: 11px; }.page-links svg { width: 13px; height: 13px; }.filter-bar { padding: 0 12px; }.filters { gap: 18px; }.filters button { min-height: 45px; font-size: 12px; }.filter-count { font-size: 10px; }.currency-label { display: none; }.toolbar { padding: 12px; gap: 8px; }.search-form { flex: 1; min-width: 0; }.search-field { flex: 1; min-width: 0; }.search-field input { width: 100%; font-size: 11px; }.search-button { padding: 0 10px; }.reset-button { display: none; }.toolbar-actions { gap: 0; }.sticky-product, .sticky-action { left: auto; right: auto; }tbody .sticky-product { position: static; box-shadow: none; }tbody .sticky-action { position: static; }.table-footer { flex-wrap: wrap; padding: 10px 12px; gap: 10px; }.pagination { gap: 9px; }.page-size { margin-right: 2px; }.draft-state { display: none; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
</style>
