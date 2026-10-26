<template>
  <MainLayout>
    <main class="keyword-page">
      <header class="keyword-header">
        <div>
          <span class="keyword-kicker">Listing intelligence</span>
          <h1>关键词库</h1>
          <p>拓展、过滤、标记后入库；Listing 生成与评估会自动读取已入库关键词。</p>
        </div>
        <button class="refresh-button" type="button" :disabled="loading" title="刷新关键词库" @click="loadKeywords">
          <ArrowPathIcon :class="{ spinning: loading }" />
          <span>刷新</span>
        </button>
      </header>

      <section class="expand-panel">
        <div class="panel-heading">
          <span class="step-index">01</span>
          <div><strong>拓展候选词</strong><small>人工整理或 Agent 回流的词都先进入候选区</small></div>
        </div>
        <div class="scope-grid">
          <label><span>平台</span><select v-model="expandForm.platform"><option v-for="item in platforms" :key="item.value" :value="item.value">{{ item.label }}</option></select></label>
          <label><span>站点代码</span><input v-model.trim="expandForm.countryCode" maxlength="20" placeholder="US"></label>
          <label><span>产品品类</span><input v-model.trim="expandForm.category" placeholder="可留空，作为平台通用词"></label>
        </div>
        <label class="terms-field"><span>候选关键词</span><textarea v-model="expandForm.text" rows="5" placeholder="每行一个关键词，也支持逗号或分号分隔"></textarea></label>
        <div class="expand-actions">
          <span>{{ splitTerms(expandForm.text).length }} 个待加入词</span>
          <button type="button" :disabled="saving || !splitTerms(expandForm.text).length" @click="expandKeywords"><PlusIcon />加入拓展区</button>
        </div>
      </section>

      <section class="library-workspace">
        <div class="workflow-tabs" aria-label="关键词处理阶段">
          <button v-for="stage in stages" :key="stage.value" type="button" :class="{ active: activeStage === stage.value }" @click="selectStage(stage.value)">
            <span>{{ stage.index }}</span><strong>{{ stage.label }}</strong><small>{{ stage.description }}</small>
          </button>
        </div>

        <div class="library-toolbar">
          <div class="search-field"><MagnifyingGlassIcon /><input v-model.trim="filters.keyword" placeholder="搜索关键词" @keyup.enter="applyFilters"></div>
          <select v-model="filters.platform" @change="applyFilters"><option value="">全部平台</option><option v-for="item in platforms" :key="item.value" :value="item.value">{{ item.label }}</option></select>
          <span>共 {{ total }} 条</span>
        </div>

        <div v-if="notice" class="notice">{{ notice }}</div>
        <div v-if="errorMessage" class="error-message">{{ errorMessage }}</div>

        <div class="keyword-table">
          <div class="keyword-row keyword-row-head">
            <span>关键词</span><span>适用范围</span><span>标记</span><span>沉淀数据</span><span>操作</span>
          </div>
          <div v-if="!loading && !items.length" class="empty-state">当前阶段还没有关键词</div>
          <div v-for="item in items" :key="item.id" class="keyword-row">
            <div class="term-cell"><strong>{{ item.term }}</strong><small>{{ sourceLabel(item.source) }}</small><small v-if="item.notes" class="candidate-reason" :title="item.notes">{{ item.notes }}</small></div>
            <div class="scope-cell"><strong>{{ platformLabel(item.platform) }}</strong><small>{{ [item.country_code, item.category].filter(Boolean).join(' · ') || '平台通用' }}</small></div>
            <div class="tags-cell">
              <input v-if="activeStage === 'filtered'" v-model="tagDrafts[item.id]" placeholder="意图、属性、场景等，用逗号分隔">
              <template v-else><span v-for="tag in item.tags || []" :key="tag">{{ tag }}</span><small v-if="!(item.tags || []).length">未标记</small></template>
            </div>
            <div class="usage-cell"><strong>{{ item.usage_count || 0 }} 次</strong><small>{{ item.last_used_at ? `最近使用 ${formatDate(item.last_used_at)}` : '尚未使用' }}</small></div>
            <div class="row-actions">
              <template v-if="activeStage === 'expanded'">
                <button type="button" title="通过过滤" @click="moveKeyword(item, 'filtered')"><CheckIcon />通过</button>
                <button class="danger" type="button" title="淘汰关键词" @click="moveKeyword(item, 'rejected')"><XMarkIcon />淘汰</button>
              </template>
              <button v-else-if="activeStage === 'filtered'" type="button" @click="saveTags(item)"><TagIcon />完成标记</button>
              <button v-else-if="activeStage === 'tagged'" type="button" @click="moveKeyword(item, 'library')"><ArchiveBoxArrowDownIcon />入库</button>
              <button v-else-if="activeStage === 'library'" type="button" @click="moveKeyword(item, 'tagged')"><ArrowUturnLeftIcon />撤回</button>
              <template v-else>
                <button type="button" @click="moveKeyword(item, 'expanded')"><ArrowUturnLeftIcon />恢复候选</button>
                <button class="danger" type="button" @click="removeKeyword(item)"><TrashIcon />删除</button>
              </template>
            </div>
          </div>
        </div>

        <footer class="pagination">
          <button type="button" :disabled="page <= 1 || loading" @click="page--; loadKeywords()">上一页</button>
          <span>第 {{ page }} / {{ totalPages }} 页</span>
          <button type="button" :disabled="page >= totalPages || loading" @click="page++; loadKeywords()">下一页</button>
        </footer>
      </section>
    </main>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ArchiveBoxArrowDownIcon,
  ArrowPathIcon,
  ArrowUturnLeftIcon,
  CheckIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  TagIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import request from '@/utils/request'

const platforms = [
  { value: 'tiktok', label: 'TikTok Shop' },
  { value: 'home_depot', label: 'Home Depot' },
  { value: 'lowes', label: 'Lowe’s' },
  { value: 'wayfair', label: 'Wayfair' },
  { value: 'amazon', label: 'Amazon' },
]
const stages = [
  { value: 'expanded', index: '01', label: '拓展', description: '候选词池' },
  { value: 'filtered', index: '02', label: '过滤', description: '相关且合规' },
  { value: 'tagged', index: '03', label: '标记', description: '意图与属性' },
  { value: 'library', index: '04', label: '入库', description: '任务自动使用' },
  { value: 'rejected', index: '×', label: '已淘汰', description: '保留判断记录' },
]

const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const loading = ref(false)
const saving = ref(false)
const activeStage = ref('expanded')
const notice = ref('')
const errorMessage = ref('')
const tagDrafts = reactive({})
const filters = reactive({ platform: '', keyword: '' })
const expandForm = reactive({ platform: 'tiktok', countryCode: 'US', category: '', text: '' })
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

onMounted(loadKeywords)

function splitTerms(value) {
  return [...new Set(String(value || '').split(/[\n\r,;，；]+/).map(item => item.trim()).filter(Boolean))]
}

function platformLabel(value) {
  return platforms.find(item => item.value === value)?.label || value
}

function sourceLabel(value) {
  if (value === 'listing_agent') return 'Listing Agent 拓展'
  if (value === 'listing_evaluator') return '评估 Agent 回流'
  return '人工录入'
}

function formatDate(value) {
  return new Date(value).toLocaleDateString('zh-CN')
}

function applyFilters() {
  page.value = 1
  loadKeywords()
}

function selectStage(value) {
  activeStage.value = value
  page.value = 1
  loadKeywords()
}

async function loadKeywords() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await request.get('/admin/listing-keywords', { params: { stage: activeStage.value, platform: filters.platform, keyword: filters.keyword, page: page.value, page_size: pageSize } })
    items.value = response.data?.data || []
    total.value = Number(response.data?.total || 0)
    for (const item of items.value) tagDrafts[item.id] = (item.tags || []).join(', ')
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '关键词库加载失败'
  } finally {
    loading.value = false
  }
}

async function expandKeywords() {
  const terms = splitTerms(expandForm.text)
  if (!terms.length) return
  saving.value = true
  errorMessage.value = ''
  try {
    await request.post('/admin/listing-keywords/expand', { platform: expandForm.platform, country_code: expandForm.countryCode, category: expandForm.category, terms, source: 'operator' })
    expandForm.text = ''
    activeStage.value = 'expanded'
    page.value = 1
    notice.value = `已加入 ${terms.length} 个候选词`
    await loadKeywords()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '候选词加入失败'
  } finally {
    saving.value = false
  }
}

async function moveKeyword(item, stage) {
  errorMessage.value = ''
  try {
    await request.put(`/admin/listing-keywords/${item.id}`, { stage })
    await loadKeywords()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '关键词状态更新失败'
  }
}

async function saveTags(item) {
  const tags = splitTerms(tagDrafts[item.id])
  if (!tags.length) {
    errorMessage.value = '至少添加一个意图、属性或场景标记'
    return
  }
  try {
    await request.put(`/admin/listing-keywords/${item.id}`, { tags, stage: 'tagged' })
    await loadKeywords()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '关键词标记失败'
  }
}

async function removeKeyword(item) {
  try {
    await request.delete(`/admin/listing-keywords/${item.id}`)
    await loadKeywords()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '关键词删除失败'
  }
}
</script>

<style scoped>
.keyword-page { width: min(1180px, calc(100% - 32px)); margin: 0 auto; padding: 42px 0 64px; color: #18181b; }
.keyword-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 26px; }
.keyword-kicker { display: block; color: #71717a; font-size: 11px; font-weight: 700; text-transform: uppercase; }
.keyword-header h1 { margin: 7px 0 6px; font-size: 32px; line-height: 1.15; font-weight: 700; }
.keyword-header p { margin: 0; color: #71717a; font-size: 14px; }
.refresh-button, .row-actions button, .expand-actions button, .pagination button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; border: 1px solid #d4d4d8; border-radius: 6px; background: #fff; color: #3f3f46; font-size: 13px; font-weight: 600; }
.refresh-button { height: 36px; padding: 0 13px; }.refresh-button svg, .row-actions svg, .expand-actions svg { width: 15px; height: 15px; }
.expand-panel { border: 1px solid #e4e4e7; background: #fff; border-radius: 8px; padding: 20px; }
.panel-heading { display: flex; align-items: center; gap: 12px; margin-bottom: 18px; }.panel-heading div { display: flex; flex-direction: column; gap: 2px; }.panel-heading strong { font-size: 15px; }.panel-heading small { color: #a1a1aa; }
.step-index { display: grid; place-items: center; width: 30px; height: 30px; border: 1px solid #d4d4d8; border-radius: 50%; font-size: 11px; font-weight: 700; }
.scope-grid { display: grid; grid-template-columns: 1fr 0.7fr 1.6fr; gap: 12px; }.scope-grid label, .terms-field { display: flex; flex-direction: column; gap: 6px; font-size: 12px; font-weight: 600; color: #52525b; }
.scope-grid input, .scope-grid select, .terms-field textarea, .library-toolbar input, .library-toolbar select, .tags-cell input { width: 100%; border: 1px solid #d4d4d8; border-radius: 6px; background: #fff; padding: 9px 10px; color: #18181b; font-size: 13px; outline: none; }
.scope-grid input:focus, .scope-grid select:focus, .terms-field textarea:focus, .library-toolbar input:focus, .tags-cell input:focus { border-color: #71717a; }
.terms-field { margin-top: 12px; }.terms-field textarea { resize: vertical; line-height: 1.55; }
.expand-actions { display: flex; justify-content: space-between; align-items: center; margin-top: 12px; }.expand-actions > span { color: #a1a1aa; font-size: 12px; }.expand-actions button { height: 36px; padding: 0 14px; border-color: #18181b; background: #18181b; color: #fff; }
.expand-actions button:disabled, button:disabled { cursor: not-allowed; opacity: .45; }
.library-workspace { margin-top: 24px; }
.workflow-tabs { display: grid; grid-template-columns: repeat(5, 1fr); border: 1px solid #e4e4e7; border-radius: 8px 8px 0 0; overflow: hidden; background: #fafafa; }.workflow-tabs button { min-height: 68px; display: grid; grid-template-columns: 24px 1fr; grid-template-rows: auto auto; column-gap: 8px; align-content: center; text-align: left; border: 0; border-right: 1px solid #e4e4e7; background: transparent; color: #71717a; padding: 10px 12px; }.workflow-tabs button:last-child { border-right: 0; }.workflow-tabs button > span { grid-row: 1 / 3; align-self: center; font-size: 11px; }.workflow-tabs button strong { font-size: 13px; }.workflow-tabs button small { margin-top: 2px; color: #a1a1aa; font-size: 11px; }.workflow-tabs button.active { background: #18181b; color: #fff; }.workflow-tabs button.active small { color: #a1a1aa; }
.library-toolbar { display: flex; align-items: center; gap: 10px; padding: 12px; border: 1px solid #e4e4e7; border-top: 0; background: #fff; }.library-toolbar > span { margin-left: auto; color: #71717a; font-size: 12px; }.library-toolbar select { width: 170px; }.search-field { position: relative; width: 260px; }.search-field svg { position: absolute; left: 10px; top: 50%; width: 15px; transform: translateY(-50%); color: #a1a1aa; }.search-field input { padding-left: 32px; }
.keyword-table { border: 1px solid #e4e4e7; border-top: 0; background: #fff; }.keyword-row { display: grid; grid-template-columns: 1.35fr 1fr 1.45fr .8fr 1fr; min-height: 66px; border-top: 1px solid #ededf0; }.keyword-row:first-child { border-top: 0; }.keyword-row > * { min-width: 0; padding: 12px; border-right: 1px solid #ededf0; }.keyword-row > *:last-child { border-right: 0; }
.keyword-row-head { min-height: 36px; background: #fafafa; color: #71717a; font-size: 11px; font-weight: 700; }.keyword-row-head > span { display: flex; align-items: center; padding-top: 8px; padding-bottom: 8px; }
.term-cell, .scope-cell, .usage-cell { display: flex; flex-direction: column; justify-content: center; gap: 4px; }.term-cell strong, .scope-cell strong, .usage-cell strong { overflow: hidden; font-size: 13px; text-overflow: ellipsis; }.term-cell small, .scope-cell small, .usage-cell small, .tags-cell small { color: #a1a1aa; font-size: 11px; }
.candidate-reason { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.tags-cell { display: flex; align-items: center; flex-wrap: wrap; gap: 5px; }.tags-cell > span { border-radius: 4px; background: #f4f4f5; padding: 3px 6px; font-size: 11px; color: #52525b; }.tags-cell input { padding: 7px 8px; }
.row-actions { display: flex; align-items: center; gap: 6px; }.row-actions button { min-height: 30px; padding: 0 9px; font-size: 11px; }.row-actions button.danger { border-color: #fecaca; color: #b91c1c; }
.empty-state { padding: 48px 16px; text-align: center; color: #a1a1aa; font-size: 13px; }.notice, .error-message { border: 1px solid #bbf7d0; border-top: 0; background: #f0fdf4; padding: 9px 12px; color: #166534; font-size: 12px; }.error-message { border-color: #fecaca; background: #fef2f2; color: #b91c1c; }
.pagination { display: flex; align-items: center; justify-content: flex-end; gap: 12px; margin-top: 12px; color: #71717a; font-size: 12px; }.pagination button { min-height: 32px; padding: 0 11px; }
.spinning { animation: spin .8s linear infinite; } @keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 860px) { .keyword-page { width: min(100% - 20px, 1180px); padding-top: 24px; }.scope-grid { grid-template-columns: 1fr; }.workflow-tabs { grid-template-columns: 1fr 1fr; border-radius: 8px; }.workflow-tabs button { border-bottom: 1px solid #e4e4e7; }.library-toolbar { flex-wrap: wrap; border-top: 1px solid #e4e4e7; margin-top: 12px; }.search-field { width: 100%; }.keyword-table { overflow-x: auto; border-top: 1px solid #e4e4e7; }.keyword-row { min-width: 880px; }.keyword-header { align-items: flex-start; }.refresh-button span { display: none; } }
</style>
