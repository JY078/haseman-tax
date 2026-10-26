<template>
  <MainLayout>
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">

      <!-- Header -->
      <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
        <h1 class="text-3xl font-bold text-neutral-900">我的作品</h1>
        <button
          class="inline-flex items-center gap-2 px-4 py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-xl hover:bg-neutral-800 transition-colors"
          @click="$router.push('/generate')"
        >
          <SparklesIcon class="w-4 h-4" />
          开始创作
        </button>
      </div>

      <!-- Expiry Notice Banner -->
      <!-- <div class="flex items-start gap-3 px-4 py-3 mb-6 bg-amber-50 border border-amber-200 rounded-xl">
        <ClockIcon class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
        <div class="text-sm font-semibold text-amber-800">长期留存</div>
      </div> -->

      <!-- Tabs -->
      <div class="flex items-center gap-1 mb-6 border-b border-neutral-200">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          class="flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium transition-all border-b-2 -mb-px"
          :class="activeSource === tab.value
            ? 'border-neutral-900 text-neutral-900'
            : 'border-transparent text-neutral-500 hover:text-neutral-700'"
          @click="switchSource(tab.value)"
        >
          <component :is="tab.icon" class="w-4 h-4" />
          {{ tab.label }}
          <span
            v-if="activeSource === tab.value && total > 0"
            class="ml-1 px-1.5 py-0.5 text-[11px] bg-neutral-900 text-white rounded-full leading-none"
          >{{ total }}</span>
        </button>
      </div>

      <!-- Loading skeleton -->
      <div v-if="loading && items.length === 0" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
        <div v-for="i in pageSize" :key="i" class="rounded-xl overflow-hidden bg-white border border-neutral-200 animate-pulse">
          <div class="aspect-square bg-neutral-100" />
          <div class="p-3 space-y-2">
            <div class="h-3 bg-neutral-100 rounded w-3/4" />
            <div class="h-3 bg-neutral-100 rounded w-1/2" />
          </div>
        </div>
      </div>

      <!-- Grid -->
      <div
        v-else-if="items.length > 0"
        class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5 transition-opacity duration-200"
        :class="loading ? 'opacity-40 pointer-events-none' : 'opacity-100'"
      >
        <div
          v-for="item in items"
          :key="item.id"
          class="group bg-white border border-neutral-200 rounded-xl overflow-hidden cursor-pointer hover:shadow-md transition-shadow"
          @click="handleCardClick(item)"
        >
          <template v-if="item.kind === 'scene_task' || item.kind === 'ecom_suite_task'">
            <div class="relative aspect-[4/5] bg-neutral-100 overflow-hidden">
              <img
                :src="item.cover_image"
                :alt="item.kind === 'ecom_suite_task' ? '套图任务封面' : '场景图任务封面'"
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                loading="lazy"
                @error="handleImgError"
              />
              <div class="absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent" />
              <div class="absolute top-3 left-3 w-16 h-16 rounded-2xl border border-white/70 bg-white/90 shadow-lg overflow-hidden backdrop-blur-sm">
                <img v-if="item.reference_image_url" :src="item.reference_image_url" alt="参考图" class="w-full h-full object-cover" />
                <div v-else class="w-full h-full flex items-center justify-center bg-neutral-100 text-neutral-400">
                  <SwatchIcon class="w-6 h-6" />
                </div>
              </div>
              <div class="absolute top-3 right-3 px-2.5 py-1 rounded-full bg-black/55 text-white text-[11px] font-semibold backdrop-blur-sm">
                {{ item.image_count }} 张
              </div>
              <div class="absolute left-4 right-4 bottom-4 text-white">
                <div class="flex items-center gap-2 mb-2">
                  <span class="text-[11px] text-white/80">{{ formatDate(item.created_at) }}</span>
                </div>
                <h3 class="text-base font-semibold leading-tight line-clamp-2">{{ item.title }}</h3>
                <p class="mt-1 text-xs text-white/80 line-clamp-2">{{ item.subtitle }}</p>
              </div>
            </div>  
          </template>

          <template v-else>
            <div class="relative aspect-square bg-neutral-100 overflow-hidden">
              <img
                :src="item.image_url"
                alt="设计作品"
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                loading="lazy"
                @error="handleImgError"
              />
              <div v-if="item.reference_image_url" class="absolute top-2 left-2 w-12 h-12 rounded-xl border border-white/70 bg-white/90 shadow overflow-hidden">
                <img :src="item.reference_image_url" alt="参考图" class="w-full h-full object-cover" />
              </div>
              <div class="absolute left-2" :class="item.reference_image_url ? 'top-16' : 'top-2'">
                <span
                  class="px-2 py-1 text-[11px] font-medium rounded-full"
                  :class="sourceBadge(item.source).class"
                >{{ sourceBadge(item.source).label }}</span>
              </div>
              <div class="absolute top-2 right-2">
                <span
                  class="px-2 py-1 text-[11px] font-medium rounded-full flex items-center gap-1"
                  :class="expiryInfo().class"
                >
                  <ClockIcon class="w-3 h-3" />
                  {{ expiryInfo().label }}
                </span>
              </div>
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-all duration-200" />
            </div>

            <div class="p-3">
              <div class="flex items-center justify-between">
                <span class="text-[11px] text-neutral-400">{{ formatDate(item.created_at) }}</span>
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    v-if="!isExpired(item)"
                    class="p-1.5 hover:bg-neutral-100 rounded-lg transition-colors"
                    title="下载"
                    @click.stop="downloadItem(item)"
                  >
                    <ArrowDownTrayIcon class="w-3.5 h-3.5 text-neutral-500" />
                  </button>
                  <button
                    v-if="canShare(item)"
                    class="p-1.5 hover:bg-violet-50 rounded-lg transition-colors"
                    title="分享为模板"
                    @click.stop="openShareModal(item)"
                  >
                    <ShareIcon class="w-3.5 h-3.5 text-neutral-500 group-hover:text-violet-500 transition-colors" />
                  </button>
                  <button
                      class="p-1.5 hover:bg-neutral-100 rounded-lg transition-colors"
                      :title="sourceAction(item.source).label"
                    @click.stop="goToSource(item)"
                  >
                    <ArrowTopRightOnSquareIcon class="w-3.5 h-3.5 text-neutral-500" />
                  </button>
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>

      <!-- Empty：仅在不处于加载状态且确实无数据时显示 -->
      <div v-else-if="!loading" class="flex flex-col items-center justify-center py-24 gap-4">
        <!-- <div class="w-20 h-20 bg-neutral-100 rounded-2xl flex items-center justify-center">
          <PhotoIcon class="w-10 h-10 text-neutral-300" />
        </div> -->
        <div class="text-center">
          <h3 class="font-semibold text-neutral-700 mb-1">暂无作品</h3>
          <p class="text-sm text-neutral-400">暂无保存内容</p>
        </div>
        <!-- <button
          class="mt-2 px-4 py-2 bg-neutral-900 text-white text-sm font-medium rounded-lg hover:bg-neutral-800 transition-colors"
          @click="$router.push('/generate')"
        >开始创作</button> -->
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 mt-8">
        <button
          class="px-3 py-2 text-sm border border-neutral-200 rounded-lg hover:bg-neutral-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="page <= 1"
          @click="goPage(page - 1)"
        >上一页</button>
        <div class="flex items-center gap-1">
          <template v-for="p in pageNumbers" :key="p">
            <span v-if="p === '...'" class="w-9 h-9 text-sm flex items-center justify-center text-neutral-400">...</span>
            <button
              v-else
              class="w-9 h-9 text-sm rounded-lg transition-colors"
              :class="p === page
                ? 'bg-neutral-900 text-white font-medium'
                : 'border border-neutral-200 text-neutral-600 hover:bg-neutral-50'"
              @click="goPage(p)"
            >{{ p }}</button>
          </template>
        </div>
        <button
          class="px-3 py-2 text-sm border border-neutral-200 rounded-lg hover:bg-neutral-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="page >= totalPages"
          @click="goPage(page + 1)"
        >下一页</button>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="detailItem"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
        @click.self="closeDetail"
      >
        <div v-if="detailItem.kind === 'scene_task'" class="bg-white rounded-3xl overflow-hidden shadow-2xl w-full max-w-6xl max-h-[90vh] flex flex-col">
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100 flex-shrink-0">
            <div class="flex items-center gap-2 min-w-0">
              <span class="px-2 py-1 text-[11px] font-medium rounded-full bg-cyan-100 text-cyan-700">场景图</span>
              <span class="text-sm text-neutral-500 truncate">{{ formatDate(detailItem.created_at) }}</span>
            </div>
            <button class="p-1.5 rounded-lg hover:bg-neutral-100 transition-colors" @click="closeDetail">
              <XMarkIcon class="w-5 h-5 text-neutral-500" />
            </button>
          </div>
          <div class="flex-1 overflow-y-auto bg-neutral-50 p-5 space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h3 class="text-lg font-semibold text-neutral-900">整组作品</h3>
                <p class="text-sm text-neutral-500 mt-1">{{ detailItem.images.length }} 张</p>
              </div>
              <div class="hidden sm:flex items-center gap-2 text-xs text-neutral-400">
                <span>点击查看大图</span>
              </div>
            </div>
            <div class="grid sm:grid-cols-2 xl:grid-cols-3 gap-4">
              <button
                v-for="(image, index) in detailItem.images"
                :key="`${detailItem.task_id}-${index}`"
                class="group relative aspect-[4/3] rounded-3xl overflow-hidden border border-neutral-200 bg-white shadow-sm text-left hover:shadow-lg hover:border-neutral-300 transition-all"
                @click="openViewer(detailItem.images, index)"
              >
                <img :src="image" class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-300" @error="handleImgError" />
                <div class="absolute inset-0 bg-gradient-to-t from-black/30 via-transparent to-transparent opacity-80 group-hover:opacity-100 transition-opacity" />
                <div class="absolute inset-x-3 bottom-3 flex items-center justify-between gap-3">
                  <span class="inline-flex items-center px-2.5 py-1 rounded-full bg-black/35 backdrop-blur-sm text-xs font-medium text-white">{{ index + 1 }} / {{ detailItem.images.length }}</span>
                  <span class="inline-flex items-center px-2.5 py-1 rounded-full bg-white/16 backdrop-blur-sm text-[11px] font-medium text-white opacity-0 group-hover:opacity-100 transition-opacity">查看</span>
                </div>
              </button>
            </div>
          </div>
        </div>

        <div v-else class="bg-white rounded-2xl overflow-hidden shadow-2xl w-full max-w-3xl max-h-[90vh] flex flex-col">
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100 flex-shrink-0">
            <div class="flex items-center gap-2">
              <span class="px-2 py-1 text-[11px] font-medium rounded-full" :class="sourceBadge(detailItem.source).class">
                {{ sourceBadge(detailItem.source).label }}
              </span>
              <span class="text-sm text-neutral-500">{{ formatDate(detailItem.created_at) }}</span>
              <span
                class="px-2 py-1 text-[11px] font-medium rounded-full flex items-center gap-1"
                :class="expiryInfo(detailItem).class"
              >
                <ClockIcon class="w-3 h-3" />
                {{ expiryInfo(detailItem).label }}
              </span>
            </div>
            <div class="flex items-center gap-2">
              <button
                v-if="!isExpired(detailItem)"
                class="px-3 py-1.5 text-xs font-medium rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 transition-colors flex items-center gap-1.5"
                @click="downloadItem(detailItem)"
              >
                <ArrowDownTrayIcon class="w-3.5 h-3.5" />
                下载
              </button>
              <button
                v-if="canShare(detailItem)"
                class="px-3 py-1.5 text-xs font-medium rounded-lg border border-violet-200 text-violet-600 hover:bg-violet-50 transition-colors flex items-center gap-1.5"
                @click="openShareModal(detailItem)"
              >
                <ShareIcon class="w-3.5 h-3.5" />
                分享为模板
              </button>
              <button
                class="px-3 py-1.5 text-xs font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors flex items-center gap-1.5"
                @click="goToSource(detailItem); detailItem = null"
              >
                <ArrowTopRightOnSquareIcon class="w-3.5 h-3.5" />
                {{ sourceAction(detailItem.source).label }}
              </button>
            </div>
          </div>

          <div class="flex-1 overflow-y-auto bg-neutral-50">
            <button class="w-full h-full flex items-center justify-center" @click="openViewer([detailItem.image_url], 0)">
              <img
                :src="detailItem.image_url"
                class="max-w-full max-h-[60vh] object-contain cursor-zoom-in"
                @error="handleModalImgError"
              />
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <ImagePreview
      ref="viewerRef"
      :visible="viewerVisible"
      :images="viewerImages"
      alt="图片预览"
      :max-width="1280"
      @close="closeViewer"
    >
      <template #footer>
        <div v-if="viewerImages.length > 1" class="flex items-center justify-center gap-2 overflow-x-auto px-1 py-1">
          <button
            v-for="(image, index) in viewerImages"
            :key="`design-thumb-${index}`"
            class="shrink-0 w-16 h-16 md:w-20 md:h-20 rounded-xl overflow-hidden border-2 transition-all"
            :class="index === viewerRef?.currentIndex
              ? 'border-neutral-900 scale-[1.03]'
              : 'border-neutral-200 opacity-60 hover:opacity-100'"
            @click="viewerRef?.goToImage(index)"
          >
            <img :src="image" class="w-full h-full object-cover" />
          </button>
        </div>
      </template>
    </ImagePreview>

    <!-- 分享为模板确认弹窗 -->
    <Teleport to="body">
      <div
        v-if="shareModal.visible"
        class="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
        @click.self="closeShareModal"
      >
        <div class="bg-white rounded-2xl shadow-2xl w-full max-w-md flex flex-col overflow-hidden">
          <!-- Header -->
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100">
            <div class="flex items-center gap-2">
              <div class="w-7 h-7 bg-violet-100 rounded-lg flex items-center justify-center">
                <ShareIcon class="w-4 h-4 text-violet-600" />
              </div>
              <h3 class="text-sm font-semibold text-neutral-900">分享为模板</h3>
            </div>
            <button class="p-1.5 rounded-lg hover:bg-neutral-100 transition-colors" @click="closeShareModal">
              <XMarkIcon class="w-4 h-4 text-neutral-400" />
            </button>
          </div>

          <!-- Body -->
          <div class="p-5 space-y-4">
            <!-- 预览图 -->
            <div class="flex gap-3 items-start">
              <img
                v-if="shareModal.item"
                :src="shareModal.item.image_url"
                class="w-16 h-16 object-cover rounded-lg border border-neutral-100 flex-shrink-0"
              />
              <div class="flex-1 min-w-0">
                <p class="text-xs text-neutral-500 mb-1">提示词内容（将作为模板内容）</p>
                <p class="text-xs text-neutral-700 leading-relaxed bg-neutral-50 rounded-lg p-2 border border-neutral-100 max-h-32 overflow-y-auto whitespace-pre-wrap break-words">{{ shareModal.item?.prompt || '（无提示词）' }}</p>
              </div>
            </div>

            <!-- 模板标题 -->
            <div>
              <label class="text-xs font-medium text-neutral-700 mb-1.5 block">
                模板标题 <span class="text-red-400">*</span>
              </label>
              <input
                v-model="shareModal.title"
                type="text"
                placeholder="请输入模板标题，便于其他用户识别"
                maxlength="60"
                class="w-full px-3 py-2 text-sm border border-neutral-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-400 focus:border-transparent transition-all placeholder:text-neutral-400"
                @keydown.enter="submitShare"
              />
            </div>

            <!-- 提示 -->
            <div class="flex items-start gap-2 px-3 py-2.5 bg-amber-50 border border-amber-100 rounded-lg">
              <svg class="w-3.5 h-3.5 text-amber-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
              </svg>
              <p class="text-[11px] text-amber-700 leading-relaxed">分享后需经管理员审核，审核通过后将在模板库公开展示</p>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex gap-2.5 px-5 pb-5">
            <button
              class="flex-1 py-2 text-sm font-medium rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 transition-colors"
              @click="closeShareModal"
            >取消</button>
            <button
              class="flex-1 py-2 text-sm font-medium rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-1.5"
              :disabled="!shareModal.title.trim() || shareModal.submitting"
              @click="submitShare"
            >
              <svg v-if="shareModal.submitting" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {{ shareModal.submitting ? '提交中...' : '确认分享' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </MainLayout>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  SparklesIcon, XMarkIcon,
  ArrowDownTrayIcon, ArrowTopRightOnSquareIcon,
  ClockIcon, ShareIcon,
  ScissorsIcon, CubeTransparentIcon, SquaresPlusIcon,
  PaintBrushIcon, SwatchIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { resolveImageUrl, downloadImage } from '@/utils/request'

const route = useRoute()
const router = useRouter()

// ---- Tabs ----
const DEFAULT_SOURCE = 'generate'
const tabs = [
  { value: 'generate', label: 'AI 生成', icon: SparklesIcon },
  { value: 'scene_generator', label: '场景图', icon: SwatchIcon },
  { value: 'ecom_suite', label: '套图生成', icon: SparklesIcon },
  // { value: 'multi_angle', label: '多视角', icon: CubeTransparentIcon },
  // { value: 'eraser', label: '消除水印', icon: PaintBrushIcon },
  // { value: 'remove-bg', label: '去除背景', icon: ScissorsIcon },
]

// ---- State ----
const activeSource = ref(DEFAULT_SOURCE)
const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const loading = ref(false)
const detailItem = ref(null)
const viewerRef = ref(null)
const viewerVisible = ref(false)
const viewerImages = ref([])

// ---- Computed ----
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const isSceneTab = computed(() => activeSource.value === 'scene_generator')
const isEcomSuiteTab = computed(() => activeSource.value === 'ecom_suite')
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

async function fetchEcomSuiteTasks() {
  const [taskRes, productRes] = await Promise.all([
    request.get('/ecom-suite/tasks').catch(() => ({ data: [] })),
    request.get('/product-library').catch(() => ({ data: [] })),
  ])
  const list = Array.isArray(taskRes.data) ? taskRes.data : []
  const productImageMap = new Map((Array.isArray(productRes.data) ? productRes.data : []).map(product => [Number(product.id), product.image_url]))
  const details = await Promise.all(list.map(task => request.get(`/ecom-suite/tasks/${task.task_id}`).then(r => r.data).catch(() => null)))
  return details.filter(Boolean).map(task => {
    const images = (task.images || []).filter(img => img.image_url).map(img => ({
      ...img,
      image_url: resolveImageUrl(img.image_url),
    }))
    return {
      kind: 'ecom_suite_task',
      id: task.task_id,
      task_id: task.task_id,
      source: 'ecom_suite',
      created_at: task.created_at,
      title: '电商套图任务',
      subtitle: [task.platform, task.country_name, `比例 ${task.output_ratio}`].filter(Boolean).join(' · '),
      cover_image: images[0]?.image_url || '',
      images: images.map(img => img.image_url),
      image_count: images.length || task.success_image_count || task.total_image_count || 0,
      reference_image_url: resolveImageUrl(productImageMap.get(Number(task.product_id)) || ''),
    }
  }).filter(task => task.image_count > 0)
}

// ---- Fetch ----
async function fetchItems() {
  loading.value = true
  try {
    if (isSceneTab.value) {
      const res = await request.get('/scene-generation/tasks', { params: { saved_only: 1, page: page.value, page_size: pageSize } })
      const tasks = (res.data.items || []).filter(item => item.isSavedToDesign)
      items.value = tasks.map(task => ({
        kind: 'scene_task',
        id: task.taskId,
        task_id: task.taskId,
        source: 'scene_generator',
        created_at: task.savedAt || task.createdAt,
        title: [task.productCategory, task.styleName].filter(Boolean).join(' · ') || '场景图任务',
        subtitle: [task.productColorFamily, `比例 ${task.aspectRatio}`].filter(Boolean).join(' · '),
        product_color_family: task.productColorFamily,
        reference_image_title: task.referenceImageTitle || '',
        reference_image_url: resolveImageUrl(task.referenceImageUrl || task.productImageUrl),
        cover_image: resolveImageUrl(task.imageUrls?.[0] || task.productImageUrl),
        images: (task.imageUrls || []).map(resolveImageUrl),
        image_count: task.imageUrls?.length || task.imageCount || 0,
      }))
      total.value = res.data.total || items.value.length
      return
    }

    if (isEcomSuiteTab.value) {
      items.value = await fetchEcomSuiteTasks()
      total.value = items.value.length
      return
    }

    const params = { image_only: 1, saved_only: 1, page: page.value, page_size: pageSize }
    if (activeSource.value) params.source = activeSource.value
    const res = await request.get('/usage-logs', { params })
    items.value = (res.data.data || []).map(item => ({
      ...item,
      kind: 'image',
      image_url_raw: item.image_url,
      image_url: resolveImageUrl(item.image_url),
      reference_image_url: resolveImageUrl(item.reference_image_url || ''),
    }))
    total.value = res.data.total || 0
  } catch {
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function switchSource(val) {
  activeSource.value = val
  page.value = 1
  syncQuery()
  fetchItems()
}

function goPage(p) {
  page.value = p
  syncQuery()
  fetchItems()
  if (!isSceneTab.value) {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

function syncQuery() {
  const query = { ...route.query }
  if (activeSource.value && activeSource.value !== DEFAULT_SOURCE) query.source = activeSource.value
  else delete query.source
  if (page.value > 1) query.page = String(page.value)
  else delete query.page
  router.replace({ query })
}

// ---- Helpers ----
function sourceBadge(source) {
  const map = {
    generate: { label: 'AI 生成', class: 'bg-violet-100 text-violet-700' },
    scene_generator: { label: '场景图', class: 'bg-cyan-100 text-cyan-700' },
    sceneGenerator: { label: '场景图', class: 'bg-cyan-100 text-cyan-700' },
    'scene-generator': { label: '场景图', class: 'bg-cyan-100 text-cyan-700' },
    ecom_suite: { label: '套图生成', class: 'bg-orange-100 text-orange-700' },
    multi_angle: { label: '多视角', class: 'bg-amber-100 text-amber-700' },
    eraser: { label: '消除水印', class: 'bg-rose-100 text-rose-700' },
    'remove-bg': { label: '去除背景', class: 'bg-sky-100 text-sky-700' },
  }
  return map[source] || { label: source || '其他', class: 'bg-neutral-100 text-neutral-600' }
}

function sourceAction(source) {
  const map = {
    generate:    { label: '去生成页',   path: '/generate' },
    scene_generator: { label: '去场景图', path: '/scene-generator' },
    ecom_suite: { label: '去套图生成', path: '/product-image' },
    sceneGenerator: { label: '去场景图', path: '/scene-generator' },
    'scene-generator': { label: '去场景图', path: '/scene-generator' },
    multi_angle: { label: '去多视角',   path: '/multi-angle' },
    eraser:      { label: '去消除水印', path: '/eraser' },
    'remove-bg': { label: '去去除背景', path: '/remove-bg' },
  }
  return map[source] || { label: '查看详情', path: '/' }
}

function goToSource(item) {
  const action = sourceAction(item.source)
  if (item.kind === 'scene_task') {
    router.push({ path: '/scene-generator' })
    return
  }
  if (item.kind === 'ecom_suite_task') {
    router.push({ path: '/product-image' })
    return
  }
  if (item.source === 'generate' && item.prompt) {
    router.push({ path: '/generate', query: { prompt: item.prompt } })
  } else if (['scene_generator', 'sceneGenerator', 'scene-generator'].includes(item.source)) {
    router.push({ path: '/scene-generator' })
  } else {
    router.push(action.path)
  }
}

function handleCardClick(item) {
  if (item.kind === 'scene_task' || item.kind === 'ecom_suite_task') {
    openViewer(item.images || [], 0)
    return
  }
  openDetail(item)
}

function openDetail(item) {
  detailItem.value = item
}

function closeDetail() {
  detailItem.value = null
  closeViewer()
}

function openViewer(images, index) {
  viewerImages.value = images
  viewerVisible.value = true
  nextTick(() => viewerRef.value?.goToImage(index))
}

function closeViewer() {
  viewerVisible.value = false
}

function downloadItem(item) {
  const images = viewerImages.value
  const idx = viewerRef.value?.currentIndex ?? 0
  const url = (item.kind === 'scene_task' || item.kind === 'ecom_suite_task')
    ? (images[idx] || images[0] || item.cover_image)
    : item.image_url
  if (url) downloadImage(url, `ai-${item.id}.png`)
}

const IMAGE_TTL_DAYS = 90

function isExpired(item) {
  if (item?.kind === 'scene_task' || item?.kind === 'ecom_suite_task') return false
  if (!item?.created_at) return false
  const created = new Date(item.created_at)
  const expiry = new Date(created.getTime() + IMAGE_TTL_DAYS * 24 * 60 * 60 * 1000)
  return expiry - new Date() <= 0
}

function expiryInfo() {
  return { label: '已保存', class: 'bg-emerald-100 text-emerald-700' }
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now - d
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  if (diff < 86400000 * 7) return `${Math.floor(diff / 86400000)} 天前`
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', year: 'numeric' })
}

function handleImgError(e) {
  e.target.style.display = 'none'
  const placeholder = document.createElement('div')
  placeholder.className = 'w-full h-full flex flex-col items-center justify-center bg-neutral-100 text-neutral-400'
  placeholder.innerHTML = `
    <svg class="w-10 h-10 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
    </svg>
    <span class="text-xs">图片已过期</span>
  `
  e.target.parentElement.appendChild(placeholder)
}

function handleModalImgError(e) {
  e.target.style.display = 'none'
  const placeholder = document.createElement('div')
  placeholder.className = 'w-full h-[60vh] flex flex-col items-center justify-center bg-neutral-100 text-neutral-400'
  placeholder.innerHTML = `
    <svg class="w-16 h-16 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
    </svg>
    <span class="text-sm font-medium">图片已过期</span>
    <span class="text-xs mt-1">该图片已被系统自动清理</span>
  `
  e.target.parentElement.appendChild(placeholder)
}

function handleKeydown(e) {
  if (e.key === 'Escape') {
    if (viewerVisible.value) {
      closeViewer()
      return
    }
    if (detailItem.value) {
      closeDetail()
    }
    return
  }
}

// ---- Share as Template ----
const shareModal = ref({ visible: false, item: null, title: '', submitting: false })

function canShare(item) {
  return item?.kind !== 'scene_task' && !!item?.prompt && !isExpired(item)
}

function openShareModal(item) {
  shareModal.value = {
    visible: true,
    item,
    title: '',
    submitting: false,
  }
}

function closeShareModal() {
  shareModal.value.visible = false
}

function initFromQuery() {
  const source = typeof route.query.source === 'string' ? route.query.source : ''
  const pageQuery = Number(route.query.page)
  activeSource.value = tabs.some(tab => tab.value === source) ? source : DEFAULT_SOURCE
  page.value = Number.isFinite(pageQuery) && pageQuery > 0 ? pageQuery : 1
}

async function submitShare() {
  const { item, title } = shareModal.value
  if (!title.trim() || !item) return
  shareModal.value.submitting = true
  try {
    await request.post('/prompts', {
      group_id: 0,
      title: title.trim(),
      content: item.prompt,
      images: item.image_url_raw ? [item.image_url_raw] : [],
      status: 'pending',
      source: 'user',
    })
    closeShareModal()
    // 简单成功提示（不依赖 UI 库）
    const toast = document.createElement('div')
    toast.className = 'fixed top-5 left-1/2 -translate-x-1/2 z-[100] px-4 py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-xl shadow-lg transition-all'
    toast.textContent = '✓ 已提交，等待管理员审核'
    document.body.appendChild(toast)
    setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 400) }, 2500)
  } catch {
    shareModal.value.submitting = false
  }
}

onMounted(() => {
  initFromQuery()
  fetchItems()
  window.addEventListener('keydown', handleKeydown)
})

watch(() => route.query.source, (next, prev) => {
  if (next === prev) return
  initFromQuery()
  fetchItems()
})

watch(() => route.query.page, (next, prev) => {
  if (next === prev) return
  initFromQuery()
  fetchItems()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.line-clamp-2{
  display:-webkit-box;
  -webkit-line-clamp:2;
  -webkit-box-orient:vertical;
  overflow:hidden;
}
</style>
