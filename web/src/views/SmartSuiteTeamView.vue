<template>
  <MainLayout>
      <div class="review-panel">
        <header class="review-header">
          <div class="page-title"><RouterLink to="/smart-suite" title="返回我的智能套图" aria-label="返回我的智能套图"><ChevronLeftIcon /></RouterLink><div><span class="review-eyebrow">智能套图 · 管理员只读</span><h1>团队操作记录</h1></div></div>
          <div class="header-actions"><label class="member-filter">使用人<select v-model="member" aria-label="使用人"><option value="">全部其他用户</option><option v-for="u in members" :key="u.id" :value="String(u.id)">{{ u.name }}</option></select></label><button :disabled="loading" title="刷新任务与结果" aria-label="刷新任务与结果" @click="refresh"><ArrowPathIcon /></button></div>
        </header>
        <div class="review-workspace" :class="{ 'mobile-detail': mobileDetail }">
          <aside class="review-browser">
            <div class="filters">
              <h2 class="history-title">操作记录</h2>
              <label>搜索最近 200 条任务<input v-model="search" placeholder="商品、用户名或任务编号" type="search" /></label>
              <div class="filter-pair"><label>状态<select v-model="status"><option value="">全部状态</option><option value="success">已完成</option><option value="active">进行中</option><option value="failed">失败</option><option value="stopped">已停止</option></select></label></div>
              <span class="result-count">其他用户 · {{ filtered.length }} 个任务</span>
            </div>
            <p v-if="listError" class="notice error" role="alert">{{ listError }} <button @click="refresh">重试</button></p>
            <div v-else-if="loading" class="skeletons" aria-label="正在加载"><div v-for="n in 4" :key="n" /></div>
            <div v-else class="task-list">
              <p v-if="!filtered.length" class="notice">没有匹配的任务</p>
              <button v-for="item in filtered" :key="item.task_id" class="task-item" :class="{ selected: selected?.task_id === item.task_id }" @click="selectTask(item)">
                <img v-if="item.thumbnail" :src="resolveImageUrl(item.thumbnail)" alt="" /><span v-else class="thumb-placeholder"><PhotoIcon /></span>
                <span class="task-copy"><strong>{{ item.product_name || '未命名商品' }}</strong><span>{{ item.user_name || `用户 #${item.user_id}` }} · {{ item.platform }}</span><small><i :class="tone(item.status)" />{{ statusText(item.status) }} · {{ date(item.created_at) }}</small></span>
              </button>
            </div>
          </aside>
          <main class="review-detail">
            <button class="mobile-back" @click="mobileDetail = false"><ChevronLeftIcon />任务列表</button>
            <div v-if="!selected" class="detail-empty"><DocumentTextIcon /><p>选择一个任务查看成果</p></div>
            <template v-else>
              <div class="detail-heading"><div><span class="record-owner">使用人：{{ selected.user_name || `用户 #${selected.user_id}` }}</span><h3>{{ selected.product_name || '未命名商品' }}</h3><small>{{ selected.platform }} · {{ date(selected.created_at) }} · {{ selected.task_id }}</small></div><span class="status-label" :class="tone(task?.status || selected.status)">{{ statusText(task?.status || selected.status) }}</span></div>
              <nav class="detail-tabs" aria-label="任务详情"><button v-for="view in views" :key="view.id" :class="{ active: tab === view.id }" :aria-pressed="tab === view.id" @click="tab = view.id"><component :is="view.icon" />{{ view.name }}</button></nav>
              <p v-if="detailError" class="notice error" role="alert">{{ detailError }} <button @click="selectTask(selected)">重试</button></p>
              <div v-else-if="detailLoading" class="detail-loading skeletons" aria-label="正在加载任务详情"><div /><div /></div>
              <div v-else-if="task" class="detail-content">
                <p v-if="task.error_message" class="notice error">{{ task.error_message }}</p>
                <template v-if="tab === 'images'">
                  <p v-if="!task.images?.length" class="notice">该任务暂无生成图片</p>
                  <section v-for="group in groups" :key="group.name" class="image-section"><h4>{{ group.name }} <span>{{ group.images.length }}</span></h4><div class="image-grid"><figure v-for="img in group.images" :key="img.id"><button :disabled="!img.image_url" :title="img.image_url ? '放大图片' : '尚未生成'" @click="preview = img.image_url"><img v-if="img.image_url" :src="resolveImageUrl(img.image_url)" :alt="img.title || group.name" /><span v-else class="image-pending"><PhotoIcon />{{ statusText(img.status) }}</span></button><figcaption>{{ img.title || `${group.name} ${img.image_index}` }}</figcaption><details v-if="img.candidate_prompt || img.error_message"><summary>修改与异常</summary><p>{{ img.candidate_prompt }}</p><p>{{ img.error_message }}</p></details></figure></div></section>
                </template>
                <template v-else-if="tab === 'listing'">
                  <p v-if="!listing" class="notice">该任务暂无 Listing 文案</p>
                  <template v-else><div class="listing-toolbar"><span>{{ listing.platform || selected.platform }}</span><button @click="copyListing"><ClipboardDocumentIcon />{{ copyState || '复制文案' }}</button></div><section v-for="field in listingFields" :key="field.key" class="listing-section"><h4>{{ field.label }}</h4><ul v-if="Array.isArray(field.value)"><li v-for="(line, index) in field.value" :key="index">{{ line }}</li></ul><p v-else>{{ field.value }}</p></section></template>
                </template>
                <template v-else>
                  <dl class="task-facts"><div><dt>当前进展</dt><dd>{{ task.current_action || statusText(task.status) }}</dd></div><div><dt>目标站点</dt><dd>{{ task.country_name || task.country_code || '未填写' }}</dd></div><div><dt>语言</dt><dd>{{ task.language || '未填写' }}</dd></div></dl>
                  <section class="input-section"><h4>产品原图</h4><button v-if="task.white_bg_image_url" class="input-image" title="放大产品原图" @click="preview = task.white_bg_image_url"><img :src="resolveImageUrl(task.white_bg_image_url)" alt="产品原图" /></button></section>
                  <section v-for="field in inputFields" :key="field.key" class="input-section"><h4>{{ field.label }}</h4><p>{{ inputs[field.key] || '未填写' }}</p></section>
                </template>
              </div>
            </template>
          </main>
        </div>
        <Dialog :open="Boolean(preview)" class="image-dialog" @close="preview = ''"><div class="review-backdrop" /><div class="preview-position"><DialogPanel class="preview-panel"><DialogTitle class="sr-only">图片预览</DialogTitle><button title="关闭图片" aria-label="关闭图片" @click="preview = ''"><XMarkIcon /></button><img :src="resolveImageUrl(preview)" alt="生成图片预览" /></DialogPanel></div></Dialog>
      </div>
  </MainLayout>
</template>

<script setup>
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/vue'
import { ArrowPathIcon, XMarkIcon, ChevronLeftIcon, PhotoIcon, DocumentTextIcon, ClipboardDocumentIcon, AdjustmentsHorizontalIcon } from '@heroicons/vue/24/outline'
import request, { resolveImageUrl } from '@/utils/request'

const records = ref([]), selected = ref(null), task = ref(null), inputs = ref({})
const search = ref(''), member = ref(''), status = ref(''), tab = ref('images'), preview = ref(''), copyState = ref('')
const loading = ref(false), detailLoading = ref(false), listError = ref(''), detailError = ref(''), mobileDetail = ref(false)
let detailVersion = 0, listVersion = 0
const views = [{ id: 'images', name: '生成图片', icon: PhotoIcon }, { id: 'listing', name: 'Listing 文案', icon: DocumentTextIcon }, { id: 'input', name: '提交资料', icon: AdjustmentsHorizontalIcon }]
const inputFields = [{ key: 'product_features', label: '产品信息' }, { key: 'product_proof', label: '证明与合规' }, { key: 'package_contents', label: '包装内容' }, { key: 'style_requirements', label: '视觉要求' }]
const members = computed(() => Array.from(new Map(records.value.map(x => [x.user_id, { id: x.user_id, name: x.user_name || `用户 #${x.user_id}` }])).values()))
const activeStatuses = ['pending', 'running', 'planning', 'generating', 'reviewing', 'analyzing', 'processing']
const filtered = computed(() => records.value.filter(x => (!member.value || String(x.user_id) === member.value) && (!status.value || (status.value === 'active' ? activeStatuses.includes(x.status) : x.status === status.value)) && `${x.product_name} ${x.user_name} ${x.task_id}`.toLowerCase().includes(search.value.trim().toLowerCase())))
const listing = computed(() => task.value?.listing || null)
const listingFields = computed(() => [{ key: 'title', label: 'Title' }, { key: 'about_this_product', label: 'About This Product' }, { key: 'description', label: 'Description' }, { key: 'features', label: 'Features' }, { key: 'bullets', label: 'Bullets' }, { key: 'salient_bullets', label: 'Salient Bullets' }, { key: 'bullet_points', label: 'Bullet Points' }].map(x => ({ ...x, value: listing.value?.[x.key] })).filter(x => x.value?.length && (listing.value?.platform !== 'wayfair' || ['about_this_product', 'features'].includes(x.key))))
const groups = computed(() => {
  const labels = { smart_secondary: '副图', smart_desktop_a_plus: '桌面 A+', smart_mobile_a_plus: '移动 A+' }
  return Object.entries(Object.groupBy(task.value?.images || [], x => x.image_type)).map(([key, images]) => ({ name: labels[key] || key, images }))
})
function statusText(s) { return ({ success: '已完成', failed: '失败', stopped: '已停止', cancelled: '已取消', pending: '排队中' })[s] || (activeStatuses.includes(s) ? '进行中' : s || '未开始') }
function tone(s) { return s === 'success' ? 'success' : s === 'failed' ? 'failed' : activeStatuses.includes(s) ? 'active-status' : 'neutral' }
function date(value) { const d = new Date(value); return Number.isNaN(d.getTime()) ? '未知时间' : d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }
async function selectTask(item, changeTab = true) {
  const version = ++detailVersion
  selected.value = item; task.value = null; inputs.value = {}; detailError.value = ''; detailLoading.value = true; copyState.value = ''; mobileDetail.value = true
  if (changeTab) tab.value = item.output_target === 'listing' ? 'listing' : 'images'
  try { const { data } = await request.get(`/admin/ecom-suite/tasks/${encodeURIComponent(item.task_id)}`); if (version === detailVersion) { task.value = data.task; inputs.value = data.input || {} } }
  catch { if (version === detailVersion) detailError.value = '无法加载任务详情，请重试' }
  finally { if (version === detailVersion) detailLoading.value = false }
}
async function refresh() {
  const version = ++listVersion
  loading.value = true; listError.value = ''
  try { const { data } = await request.get('/admin/ecom-suite/tasks', { params: { limit: 200, scope: 'others' } }); if (version !== listVersion) return; records.value = Array.isArray(data) ? data : []; const current = records.value.find(x => x.task_id === selected.value?.task_id); if (current) await selectTask(current, false); else { selected.value = null; task.value = null; if (filtered.value.length) { await selectTask(filtered.value[0]); mobileDetail.value = false } } }
  catch { if (version === listVersion) listError.value = '无法加载团队任务，请重试' }
  finally { if (version === listVersion) loading.value = false }
}
async function copyListing() { try { await navigator.clipboard.writeText(listingFields.value.map(x => `${x.label}\n${Array.isArray(x.value) ? x.value.join('\n') : x.value}`).join('\n\n')); copyState.value = '已复制' } catch { copyState.value = '复制失败' } }
watch(filtered, items => {
  if (selected.value && !items.some(item => item.task_id === selected.value.task_id)) {
    ++detailVersion
    selected.value = null; task.value = null; inputs.value = {}; detailError.value = ''; detailLoading.value = false; mobileDetail.value = false
  }
})
onMounted(refresh)
onBeforeUnmount(() => { ++detailVersion; ++listVersion })
</script>

<style scoped>
.page-title{display:flex;align-items:center;gap:12px}.page-title a{display:grid;place-items:center;width:32px;height:32px;color:inherit;border:1px solid #e1e6e3;border-radius:5px}.header-actions{align-items:center}.member-filter{display:flex;align-items:center;gap:8px;font-size:12px;color:#64746c}.member-filter select{width:170px;padding:8px;border:1px solid #d6dfda;border-radius:5px;background:#fff;color:#28372e}.history-title{font-size:13px;margin:0 0 16px;font-weight:600}.record-owner{font-size:13px;font-weight:600;color:#23715b}
.review-backdrop{position:fixed;inset:0;background:rgba(20,31,27,.4)}.review-panel{height:calc(100dvh - 64px);min-height:0;background:#fff;display:flex;flex-direction:column;overflow:hidden;color:#25312d;letter-spacing:0}button,input,select{font:inherit}button{cursor:pointer}button:disabled{cursor:default;opacity:.5}button:focus-visible,input:focus-visible,select:focus-visible{outline:2px solid #24846c;outline-offset:2px}svg{width:20px;height:20px;flex-shrink:0}button{color:inherit}.review-header{display:flex;justify-content:space-between;align-items:center;padding:18px 24px;border-bottom:1px solid #dce2df}.review-eyebrow{font-size:11px;color:#687d73}.review-header h1{margin:4px 0 0;font-size:21px;font-weight:650}.header-actions{display:flex;gap:8px}.header-actions button,.preview-panel>button{width:36px;height:36px;display:grid;place-items:center;border:1px solid #dce2df;background:#fff;border-radius:6px}.review-workspace{display:grid;grid-template-columns:260px minmax(0,1fr);flex:1;min-height:0}.review-browser{background:#f6f8f7;border-right:1px solid #dce2df;display:flex;flex-direction:column;min-height:0}.filters{padding:18px 16px 12px;border-bottom:1px solid #e1e6e3}.filters label{display:flex;flex-direction:column;gap:6px;font-size:11px;color:#64746c}.filters input,.filters select{width:100%;min-width:0;box-sizing:border-box;background:white;border:1px solid #d6dfda;border-radius:5px;padding:8px;font-size:12px;color:#28372e}.filter-pair{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin:12px 0}.result-count{font-size:11px;color:#78877e}.task-list{overflow:auto;padding:8px}.task-item{width:100%;display:flex;align-items:center;gap:12px;border:1px solid transparent;border-radius:6px;background:transparent;padding:12px 8px;text-align:left;margin-bottom:4px}.task-item:hover{background:#ecf1ee}.task-item.selected{background:#fff;border-color:#aac9bb;box-shadow:inset 3px 0 #29856a}.task-item img,.thumb-placeholder{width:48px;height:56px;object-fit:contain;flex-shrink:0;background:#fff;border-radius:4px}.thumb-placeholder{display:grid;place-items:center;color:#92a59a}.task-copy{min-width:0;display:flex;flex-direction:column;gap:6px;font-size:12px}.task-copy strong{overflow:hidden;white-space:nowrap;text-overflow:ellipsis;font-size:13px}.task-copy>span{color:#697a70}.task-copy small{font-size:10px;color:#7d8b83;display:flex;align-items:center;gap:5px}.task-copy i{width:6px;height:6px;border-radius:50%;background:#95a299;flex-shrink:0}.task-copy i.success{background:#28856a}.task-copy i.failed{background:#ba534c}.task-copy i.active-status{background:#ce993c}.review-detail{min-width:0;display:flex;flex-direction:column;min-height:0}.detail-heading{padding:22px 28px 16px;display:flex;justify-content:space-between;gap:16px}.detail-heading h3{font-size:19px;margin:6px 0 8px;overflow-wrap:anywhere}.detail-heading small{color:#8a958e;font-size:10px;overflow-wrap:anywhere}.status-label{align-self:flex-start;border:1px solid #dce2df;border-radius:4px;padding:4px 7px;white-space:nowrap;font-size:11px;color:#66756c}.status-label.success{color:#23715b;background:#eef7f1}.status-label.failed{color:#a24842;background:#fdf1f0}.status-label.active-status{color:#926520;background:#fff8e8}.detail-tabs{display:flex;gap:24px;padding:0 28px;border-bottom:1px solid #e1e6e3}.detail-tabs button{display:flex;align-items:center;gap:6px;border:0;border-bottom:2px solid transparent;background:transparent;padding:12px 0;font-size:12px;color:#7d887f}.detail-tabs svg{width:16px;height:16px}.detail-tabs .active{border-color:#29856a;color:#23715b}.detail-content{overflow:auto;padding:24px 28px;flex:1}.image-section h4,.input-section h4,.listing-section h4{font-size:12px;font-weight:650;margin:0 0 12px;color:#64766c}.image-section h4 span{margin-left:6px;color:#9aa69f}.image-section{margin-bottom:28px}.image-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.image-grid figure{margin:0;min-width:0}.image-grid button{width:100%;aspect-ratio:1;border:1px solid #e2e7e4;border-radius:6px;background:#f7f9f8;padding:0;overflow:hidden;display:grid;place-items:center;cursor:zoom-in}.image-grid img{width:100%;height:100%;object-fit:contain}.image-grid figcaption{font-size:11px;margin-top:8px;overflow-wrap:anywhere;color:#62756a}.image-pending{display:grid;justify-items:center;gap:8px;font-size:12px;color:#8b9c92}details{font-size:11px;margin-top:8px;overflow-wrap:anywhere}details p{white-space:pre-wrap}.listing-toolbar{display:flex;justify-content:space-between;align-items:center;font-size:12px;color:#829087;margin-bottom:24px}.listing-toolbar button{border:1px solid #dce2df;border-radius:5px;background:#fff;padding:6px 10px;display:flex;align-items:center;gap:6px}.listing-toolbar svg{width:15px}.listing-section{padding-bottom:22px;margin-bottom:22px;border-bottom:1px solid #ecf0ed}.listing-section p,.listing-section li{font-size:14px;line-height:1.8;white-space:pre-wrap;overflow-wrap:anywhere;margin:0}.listing-section ul{padding-left:18px;margin:0}.listing-section li+li{margin-top:12px}.task-facts{margin:0 0 24px;display:grid;gap:12px;font-size:12px}.task-facts>div{display:grid;grid-template-columns:85px 1fr}.task-facts dt{color:#7b8c81}.task-facts dd{margin:0;overflow-wrap:anywhere}.input-section{border-top:1px solid #e7ece8;padding:20px 0}.input-section p{white-space:pre-wrap;overflow-wrap:anywhere;font-size:13px;line-height:1.8}.input-image{width:160px;height:160px;background:white;border:1px solid #e1e6e3;border-radius:6px;cursor:zoom-in}.input-image img{width:100%;height:100%;object-fit:contain}.notice{padding:24px;color:#849188;font-size:13px;line-height:1.7}.error{color:#a24842}.error button{border:0;background:transparent;text-decoration:underline}.detail-empty{display:grid;align-content:center;justify-items:center;flex:1;color:#92a197;font-size:14px}.skeletons{padding:20px}.skeletons>div{height:74px;background:#edf1ee;border-radius:5px;margin-bottom:12px}.detail-loading>div{height:160px}.mobile-back{display:none}.preview-position{position:fixed;inset:0;display:grid;place-items:center;padding:24px}.preview-panel{position:relative;max-width:95vw;max-height:90dvh}.preview-panel>button{position:absolute;right:8px;top:8px}.preview-panel>img{display:block;max-width:90vw;max-height:85dvh;object-fit:contain;background:white}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0,0,0,0)}
@media(max-width:900px){.review-workspace{grid-template-columns:250px minmax(0,1fr)}.image-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.detail-content{padding:18px}.detail-heading{padding:18px}.detail-tabs{padding:0 18px;gap:16px}}
@media(max-width:640px){.review-panel{height:calc(100dvh - 64px);min-height:0;background:#fff;display:flex;flex-direction:column;overflow:hidden;color:#25312d;letter-spacing:0}.review-header{padding:14px 16px}.review-workspace{grid-template-columns:1fr}.review-detail{display:none}.mobile-detail .review-browser{display:none}.mobile-detail .review-detail{display:flex}.mobile-back{display:flex;align-items:center;gap:4px;border:0;background:#f6f8f7;padding:10px 16px;font-size:12px}.detail-heading{gap:8px}.detail-heading h3{font-size:16px}.detail-tabs{gap:14px}.detail-tabs button{font-size:11px}.image-grid{gap:10px}}
:global(.image-dialog){position:fixed;inset:0;z-index:5000;color:#25312d;font-family:inherit;letter-spacing:0}
:global(.image-dialog){z-index:5010}
</style>
<style scoped>
@media(max-width:640px){.review-header{flex-wrap:wrap;gap:12px}.review-header h1{font-size:18px}.header-actions{width:100%;justify-content:space-between}.member-filter{flex:1}.member-filter select{flex:1;max-width:240px}.review-header .header-actions{gap:12px}.filter-pair{grid-template-columns:1fr}}
</style>
