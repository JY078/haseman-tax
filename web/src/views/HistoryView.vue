<template>
  <MainLayout>
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div class="flex items-center gap-1 mb-6 border-b border-neutral-200">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          class="flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium transition-all border-b-2 -mb-px"
          :class="activeSource === tab.value ? 'border-neutral-900 text-neutral-900' : 'border-transparent text-neutral-500 hover:text-neutral-700'"
          @click="setSource(tab.value)"
        >
          {{ tab.label }}
          <span v-if="activeSource === tab.value && filteredTotal > 0" class="ml-1 px-1.5 py-0.5 text-[11px] bg-neutral-900 text-white rounded-full leading-none">{{ filteredTotal }}</span>
        </button>
      </div>

      <div class="rounded-2xl border border-amber-200 bg-amber-50/70 px-4 py-3 mb-6 flex items-center gap-2 text-sm text-amber-800">
        <span class="inline-block w-1.5 h-1.5 rounded-full bg-amber-500"></span>
        <span>鏈繚瀛樼殑鍥剧墖浼氬湪 7 澶╁悗鑷姩娓呴櫎</span>
      </div>

      <div v-if="loading && items.length === 0" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
        <div v-for="i in pageSize" :key="i" class="rounded-xl overflow-hidden bg-white border border-neutral-200 animate-pulse">
          <div class="aspect-square bg-neutral-100" />
          <div class="p-3 space-y-2">
            <div class="h-3 bg-neutral-100 rounded w-3/4" />
            <div class="h-3 bg-neutral-100 rounded w-1/2" />
          </div>
        </div>
      </div>

      <TransitionGroup v-else-if="pagedItems.length" name="history-card" tag="div" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5 transition-opacity duration-200" :class="loading ? 'opacity-40 pointer-events-none' : 'opacity-100'">
        <div v-for="item in pagedItems" :key="`${item.kind}-${item.id}`" class="group bg-white border border-neutral-200 rounded-xl overflow-hidden cursor-pointer hover:shadow-md transition-shadow" @click="handleCardClick(item)">
          <template v-if="item.kind === 'scene_task' || item.kind === 'ecom_suite_task'">
            <div class="relative aspect-[4/5] bg-neutral-100 overflow-hidden">
              <img :src="item.cover_image" :alt="item.kind === 'ecom_suite_task' ? '套图任务封面' : '场景图任务封面'" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" loading="lazy" @error="handleImgError" @contextmenu.prevent.stop />
              <div class="absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent" />
              <div class="absolute top-3 left-3"><span class="px-2 py-1 text-[11px] font-medium rounded-full" :class="badgeOf(item.source).class">{{ badgeOf(item.source).label }}</span></div>
              <div class="absolute top-3 right-3 px-2.5 py-1 rounded-full bg-black/55 text-white text-[11px] font-semibold backdrop-blur-sm">{{ item.image_count }} 张</div>
              <div class="absolute left-4 right-4 bottom-4 text-white">
                <div class="flex items-center gap-2 mb-2"><span class="text-[11px] text-white/80">{{ formatDate(item.created_at) }}</span></div>
                <h3 class="text-base font-semibold leading-tight line-clamp-2">{{ titleOf(item) }}</h3>
                <p class="mt-1 text-xs text-white/80 line-clamp-2">{{ subtitleOf(item) }}</p>
              </div>
            </div>
            <div class="p-3 flex items-center justify-between gap-2">
              <span class="inline-flex items-center gap-1 text-[11px] text-neutral-400">
                <span class="w-1.5 h-1.5 rounded-full bg-neutral-300"></span>
                待保存
              </span>
              <div class="flex items-center gap-1">
                <button class="p-1.5 hover:bg-neutral-100 rounded-lg transition-colors" title="下载" @click.stop="downloadItem(item)">
                  <ArrowDownTrayIcon class="w-3.5 h-3.5 text-neutral-500" />
                </button>
                <button v-if="item.kind !== 'ecom_suite_task'" class="px-3 py-1.5 text-xs font-medium rounded-lg border border-emerald-200 text-emerald-700 hover:bg-emerald-50 disabled:opacity-50" :disabled="isSaving(item)" @click.stop="saveItem(item)">{{ isSaving(item) ? '保存中...' : '保存' }}</button>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="relative aspect-square bg-neutral-100 overflow-hidden">
              <video
                v-if="item.kind === 'video'"
                :src="item.image_url"
                class="w-full h-full object-cover bg-black"
                controls
                playsinline
                preload="metadata"
                @click.stop
              ></video>
              <img v-else :src="item.image_url" alt="历史图片" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" loading="lazy" @error="handleImgError" @contextmenu.prevent.stop />
              <div v-if="item.reference_image_url" class="absolute top-2 left-2 w-12 h-12 rounded-xl border border-white/70 bg-white/90 shadow overflow-hidden"><img :src="item.reference_image_url" alt="参考图" class="w-full h-full object-cover" /></div>
              <div class="absolute left-2" :class="item.reference_image_url ? 'top-16' : 'top-2'"><span class="px-2 py-1 text-[11px] font-medium rounded-full" :class="badgeOf(item.source).class">{{ badgeOf(item.source).label }}</span></div>
              <div class="absolute top-2 right-2"><span class="px-2 py-1 text-[11px] font-medium rounded-full bg-white/88 text-neutral-500 backdrop-blur-sm">待保存</span></div>
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-all duration-200" />
            </div>
            <div class="p-3">
              <div class="flex items-center justify-between">
                <span class="text-[11px] text-neutral-400">{{ formatDate(item.created_at) }}</span>
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button class="p-1.5 hover:bg-neutral-100 rounded-lg transition-colors" title="下载" @click.stop="downloadItem(item)"><ArrowDownTrayIcon class="w-3.5 h-3.5 text-neutral-500" /></button>
                  <button v-if="item.kind !== 'ecom_suite_task' && item.kind !== 'video'" class="px-3 py-1.5 text-xs font-medium rounded-lg border border-emerald-200 text-emerald-700 hover:bg-emerald-50 disabled:opacity-50" :disabled="isSaving(item)" @click.stop="saveItem(item)">{{ isSaving(item) ? '保存中...' : '保存' }}</button>
                </div>
              </div>
            </div>
          </template>
        </div>
      </TransitionGroup>

      <div v-else-if="!loading" class="flex flex-col items-center justify-center py-24 gap-4">
        <div class="w-20 h-20 bg-neutral-100 rounded-2xl flex items-center justify-center"><PhotoIcon class="w-10 h-10 text-neutral-300" /></div>
        <div class="text-center">
          <h3 class="font-semibold text-neutral-700 mb-1">暂无记录</h3>
          <p class="text-sm text-neutral-400">当前没有待保存内容</p>
        </div>
      </div>

      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 mt-8">
        <button class="px-3 py-2 text-sm border border-neutral-200 rounded-lg hover:bg-neutral-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors" :disabled="page <= 1" @click="setPage(page - 1)">上一页</button>
        <div class="flex items-center gap-1">
          <template v-for="p in pageNumbers" :key="`history-page-${p}`">
            <span v-if="p === '...'" class="w-9 h-9 text-sm flex items-center justify-center text-neutral-400">...</span>
            <button
              v-else
              class="w-9 h-9 text-sm rounded-lg transition-colors"
              :class="p === page ? 'bg-neutral-900 text-white font-medium' : 'border border-neutral-200 text-neutral-600 hover:bg-neutral-50'"
              @click="setPage(p)"
            >{{ p }}</button>
          </template>
        </div>
        <button class="px-3 py-2 text-sm border border-neutral-200 rounded-lg hover:bg-neutral-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors" :disabled="page >= totalPages" @click="setPage(page + 1)">下一页</button>
      </div>
    </div>

    <ImagePreview
      ref="viewerRef"
      :visible="viewerVisible"
      :images="viewerImages"
      alt="鍥剧墖棰勮"
      :max-width="1280"
      @close="closeViewer"
    >
      <template #footer>
        <div v-if="viewerImages.length > 1" class="flex items-center justify-center gap-2 overflow-x-auto px-1 py-1">
          <button
            v-for="(image, index) in viewerImages"
            :key="`history-thumb-${index}`"
            class="shrink-0 w-16 h-16 md:w-20 md:h-20 rounded-xl overflow-hidden border-2 transition-all"
            :class="index === viewerRef?.currentIndex
              ? 'border-neutral-900 scale-[1.03]'
              : 'border-neutral-200 opacity-60 hover:opacity-100'"
            @click="viewerRef?.goToImage(index)"
          >
            <img :src="image" class="w-full h-full object-cover" @contextmenu.prevent.stop />
          </button>
        </div>
      </template>
    </ImagePreview>
  </MainLayout>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDownTrayIcon, PhotoIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { downloadImage, resolveImageUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const pageSize = 10
const tabs = [
  { value: '', label: '全部' },
  { value: 'generate', label: 'AI 生成' },
  { value: 'video_generate', label: '视频生成' },
  { value: 'scene_generator', label: '场景图' },
  { value: 'ecom_suite', label: '套图生成' },
  { value: 'image_translate', label: '图片翻译' },
]

const activeSource = ref('')
const page = ref(1)
const loading = ref(false)
const items = ref([])
const savingKeys = ref([])
const viewerVisible = ref(false)
const viewerImages = ref([])
const viewerRef = ref(null)

const filteredItems = computed(() => items.value.filter(item => !isSaved(item)))
const filteredTotal = computed(() => filteredItems.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(filteredItems.value.length / pageSize)))
const pageNumbers = computed(() => {
  const total = totalPages.value
  const current = page.value
  const pages = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
    return pages
  }

  pages.push(1)

  if (current > 3) pages.push('...')

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) pages.push(i)

  if (current < total - 2) pages.push('...')

  pages.push(total)

  return pages
})
const pagedItems = computed(() => filteredItems.value.slice((page.value - 1) * pageSize, page.value * pageSize))

function initFromQuery() {
  const source = typeof route.query.source === 'string' ? route.query.source : ''
  const pageQuery = Number(route.query.page)
  activeSource.value = tabs.some(tab => tab.value === source) ? source : ''
  page.value = Number.isFinite(pageQuery) && pageQuery > 0 ? pageQuery : 1
}

function syncQuery() {
  const query = { ...route.query }
  if (activeSource.value) query.source = activeSource.value
  else delete query.source
  if (page.value > 1) query.page = String(page.value)
  else delete query.page
  router.replace({ query })
}

async function fetchEcomSuiteTasks() {
  const res = await request.get('/ecom-suite/tasks').catch(() => ({ data: [] }))
  const list = Array.isArray(res.data) ? res.data : []
  const details = await Promise.all(list.map(task => request.get(`/ecom-suite/tasks/${task.task_id}`).then(r => r.data).catch(() => null)))
  return details.filter(Boolean).map(task => {
    const images = (task.images || []).filter(img => img.image_url).map(img => ({
      ...img,
      image_url: resolveImageUrl(img.image_url),
    }))
    return {
      kind: 'ecom_suite_task',
      id: task.task_id,
      taskId: task.task_id,
      source: 'ecom_suite',
      created_at: task.created_at,
      title: '电商套图任务',
      subtitle: [task.platform, task.country_name, 比例 ].filter(Boolean).join(' / '),
      image_count: images.length || task.success_image_count || task.total_image_count || 0,
      cover_image: images[0]?.image_url || '',
      images: images.map(img => img.image_url),
      imageItems: images,
    }
  }).filter(task => task.image_count > 0)
}

async function fetchItems() {
  loading.value = true
  try {
    const sceneTasksPromise = (!activeSource.value || activeSource.value === 'scene_generator')
      ? request.get('/scene-generation/tasks', { params: { page: 1, page_size: 10 } }).then(r => (r.data?.items || []).map(t => ({
          kind: 'scene_task',
          id: t.taskId,
          taskId: t.taskId,
          source: 'scene_generator',
          created_at: t.createdAt,
          title: [t.productCategory, t.styleName].filter(Boolean).join(' / ') || '场景图任务',
          subtitle: [t.productColorFamily, 比例 ].filter(Boolean).join(' / '),
          image_count: t.imageUrls?.length || t.imageCount || 0,
          cover_image: resolveImageUrl(t.imageUrls?.[0] || t.productImageUrl),
          images: (t.imageUrls || []).map(resolveImageUrl),
          isSavedToDesign: !!t.isSavedToDesign,
        }))).catch(() => [])
      : Promise.resolve([])

    const ecomTasksPromise = (!activeSource.value || activeSource.value === 'ecom_suite')
      ? fetchEcomSuiteTasks()
      : Promise.resolve([])

    const imagesPromise = ['scene_generator', 'ecom_suite'].includes(activeSource.value)
      ? Promise.resolve([])
      : request.get('/usage-logs', {
          params: { image_only: 1, page: 1, page_size: 10, ...(activeSource.value ? { source: activeSource.value } : {}) },
        }).then(r => (r.data?.data || [])
          .filter(i => !['scene_generator', 'sceneGenerator', 'scene-generator', 'ecom_suite'].includes(i.source))
          .map(i => ({
            ...i,
            kind: isVideoUsageLog(i) ? 'video' : 'image',
            image_url_raw: i.image_url,
            image_url: resolveImageUrl(i.image_url),
            reference_image_url: resolveImageUrl(i.reference_image_url || ''),
            title: i.prompt || badgeOf(i.source).label,
            subtitle: i.prompt || '',
          }))).catch(() => [])

    const [sceneTasks, ecomTasks, images] = await Promise.all([sceneTasksPromise, ecomTasksPromise, imagesPromise])
    const groupedImageSet = new Set([...sceneTasks, ...ecomTasks].flatMap(task => task.images || []).filter(Boolean))
    const dedupedImages = images.filter(image => !groupedImageSet.has(image.image_url))
    items.value = [...sceneTasks, ...ecomTasks, ...dedupedImages].sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
    if (page.value > totalPages.value) page.value = 1
  } finally {
    loading.value = false
  }
}

function badgeOf(source) {
  return ({
    generate: { label: 'AI 生成', class: 'bg-violet-100 text-violet-700' },
    video_generate: { label: '视频生成', class: 'bg-rose-100 text-rose-700' },
    scene_generator: { label: '场景图', class: 'bg-cyan-100 text-cyan-700' },
    ecom_suite: { label: '套图生成', class: 'bg-orange-100 text-orange-700' },
    smart_suite_listing_keyword_research: { label: 'SEO 关键词研究', class: 'bg-emerald-100 text-emerald-700' },
    image_translate: { label: '图片翻译', class: 'bg-indigo-100 text-indigo-700' },
    multi_angle: { label: '多视角', class: 'bg-amber-100 text-amber-700' },
    eraser: { label: '消除水印', class: 'bg-rose-100 text-rose-700' },
    'remove-bg': { label: '去除背景', class: 'bg-sky-100 text-sky-700' },
  })[source] || { label: source || '其他', class: 'bg-neutral-100 text-neutral-600' }
}

function sourceAction(source) {
  return ({
    generate: { label: '去生成页', path: '/generate' },
    video_generate: { label: '查看视频', path: '/history?source=video_generate' },
    scene_generator: { label: '去场景图', path: '/scene-generator' },
    ecom_suite: { label: '去套图生成', path: '/product-image' },
    smart_suite_listing_keyword_research: { label: '查看智能套图', path: '/product-image' },
    image_translate: { label: '去图片翻译', path: '/image-translate' },
    multi_angle: { label: '去多视角', path: '/multi-angle' },
    eraser: { label: '去消除水印', path: '/eraser' },
    'remove-bg': { label: '去除背景', path: '/remove-bg' },
  })[source] || { label: '查看详情', path: '/' }
}
function isSaved(item) {
  return item.kind === 'scene_task' ? !!item.isSavedToDesign : item.kind === 'ecom_suite_task' ? false : !!item.is_saved_to_design
}

function keyOf(item) {
  return `${item.kind}-${item.id}`
}

function isSaving(item) {
  return savingKeys.value.includes(keyOf(item))
}

async function saveItem(item) {
  if (isSaving(item)) return
  savingKeys.value.push(keyOf(item))
  try {
    if (item.kind === 'scene_task') {
      await request.post(`/scene-generation/tasks/${item.taskId}/save`)
      item.isSavedToDesign = true
    } else {
      await request.post(`/usage-logs/${item.id}/save`)
      item.is_saved_to_design = true
    }
    toast.success('已保存')
    closeViewer()
  } catch (e) {
    toast.error(e?.response?.data?.error || '保存失败，请稍后重试')
  } finally {
    savingKeys.value = savingKeys.value.filter(k => k !== keyOf(item))
  }
}

function titleOf(item) { return item.title || item.prompt || '未命名记录' }
function subtitleOf(item) { return item.subtitle || item.prompt || '鏆傛棤鎻忚堪' }
function handleCardClick(item) {
  if (item.kind === 'video') return
  if (item.kind === 'scene_task' || item.kind === 'ecom_suite_task') {
    viewerImages.value = item.images || []
    viewerVisible.value = true
    viewerRef.value?.goToImage(0)
    return
  }
  openViewer(item)
}
function openViewer(item) { viewerImages.value = [item.image_url]; viewerVisible.value = true; viewerRef.value?.goToImage(0) }
function closeViewer() { viewerVisible.value = false }
function downloadItem(item) {
  const index = viewerRef.value?.currentIndex ?? 0
  const url = (item.kind === 'scene_task' || item.kind === 'ecom_suite_task')
    ? (item.images?.[index] || item.images?.[0] || item.cover_image)
    : item.image_url
  if (url) downloadImage(url, `${item.kind === 'video' ? 'video' : 'ai'}-${item.id}.${item.kind === 'video' ? 'mp4' : 'png'}`)
}
function isVideoUsageLog(item) {
  return item?.source === 'video_generate' || /\.(mp4|webm|mov|m4v|avi)(\?|#|$)/i.test(String(item?.image_url || ''))
}
function handleImgError(e) { e.target.style.opacity = '0.2' }
function handleModalImgError(e) { e.target.style.display = 'none' }
function setSource(v) { activeSource.value = v; page.value = 1; syncQuery(); fetchItems() }
function setPage(v) { page.value = v; syncQuery(); window.scrollTo({ top: 0, behavior: 'smooth' }) }
function formatDate(v) {
  if (!v) return ''
  const d = new Date(v), diff = Date.now() - d.getTime()
  if (diff < 6e4) return '刚刚'
  if (diff < 36e5) return `${Math.floor(diff / 6e4)} 分钟前`
  if (diff < 864e5) return `${Math.floor(diff / 36e5)} 小时前`
  if (diff < 6048e5) return `${Math.floor(diff / 864e5)} 天前`
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', year: 'numeric' })
}
function handleKeydown(e) {
  if (e.key === 'Escape' && viewerVisible.value) {
    closeViewer()
  }
}

onMounted(() => { initFromQuery(); fetchItems(); window.addEventListener('keydown', handleKeydown) })
onBeforeUnmount(() => { window.removeEventListener('keydown', handleKeydown) })
</script>

<style scoped>
.history-card-enter-active,.history-card-leave-active{transition:opacity .24s ease,transform .24s ease}.history-card-enter-from,.history-card-leave-to{opacity:0;transform:translateY(10px) scale(.98)}.history-card-move{transition:transform .24s ease}
</style>
