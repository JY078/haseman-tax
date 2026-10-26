<template>
  <MainLayout>
    <section class="suite-page">
      <aside class="setup-panel">
        <header class="setup-header">
          <p>Amazon 套图生成</p>
          <h1>产品差异化作图工作台</h1>
        </header>

        <div class="setup-scroll">
          <section class="setup-section">
            <div class="section-title">
              <span>1</span>
              <div>
                <h2>产品白底图</h2>
              </div>
            </div>

            <label class="upload-drop" :class="{ filled: !!whiteBgPreview, busy: isUploadingWhiteBg }">
              <input type="file" accept="image/*" @change="handleWhiteBgChange">
              <img v-if="whiteBgPreview" :src="whiteBgPreview" alt="产品白底图">
              <span v-else class="upload-empty">
                <CloudArrowUpIcon class="icon-lg" />
                <strong>{{ isUploadingWhiteBg ? '上传中...' : '选择白底图' }}</strong>
              </span>
            </label>

            <button v-if="whiteBgPreview" class="text-action" type="button" @click="clearWhiteBg">
              <ArrowPathIcon class="icon-sm" />
              重新上传
            </button>
          </section>

          <section class="setup-section">
            <div class="section-title">
              <span>2</span>
              <div>
                <h2>类目与品牌</h2>
              </div>
            </div>

            <div class="control-group">
              <div class="control-label">使用类目</div>
              <div class="segmented-control">
                <button
                  v-for="item in categoryOptions"
                  :key="item.value"
                  type="button"
                  :disabled="item.disabled"
                  :class="{ active: selectedCategory === item.value, disabled: item.disabled }"
                  @click="!item.disabled && (selectedCategory = item.value)"
                >
                  {{ item.label }}
                </button>
              </div>
            </div>

            <div class="control-group">
              <div class="control-label">品牌</div>
              <div class="brand-grid">
                <button
                  v-for="item in brandOptions"
                  :key="item.value"
                  type="button"
                  class="brand-card"
                  :class="[item.value, { active: selectedBrandValue === item.value }]"
                  @click="selectedBrandValue = item.value"
                >
                  <strong>{{ item.label }}</strong>
                </button>
              </div>
            </div>
          </section>

          <section class="setup-section">
            <div class="section-title">
              <span>3</span>
              <div>
                <h2>参考图方案</h2>
              </div>
            </div>

            <div v-if="selectedGallery" class="selected-gallery-card">
              <img v-if="galleryCoverImage(selectedGallery)" :src="resolveImageUrl(galleryCoverImage(selectedGallery))" alt="参考方案封面">
              <div v-else class="gallery-cover-empty">
                <PhotoIcon class="icon-sm" />
              </div>
              <span>
                <strong>{{ selectedGallery.name }}</strong>
                <small>{{ selectedGalleryMissingText || `${categoryLabel(selectedGallery.category)} · ${galleryCompletionText(selectedGallery)}` }}</small>
              </span>
            </div>
            <div v-else-if="!filteredReferenceGalleries.length" class="inline-empty">
              当前类目暂无可用参考图方案。
            </div>
            <div v-if="selectedGalleryMissingText" class="inline-empty warn">
              当前方案缺少：{{ selectedGalleryMissingText }}
            </div>
            <button class="secondary-button" type="button" @click="openGalleryPicker">
              {{ selectedGallery ? '更换参考方案' : '选择参考方案' }}
            </button>
          </section>

          <section class="setup-section">
            <div class="section-title">
              <span>4</span>
              <div>
                <h2>功能匹配</h2>
              </div>
            </div>

            <div class="control-group">
              <div class="control-label">抽拉与旋转</div>
              <div class="choice-grid">
                <button
                  v-for="item in featureModeOptions"
                  :key="item.value"
                  type="button"
                  :class="{ active: selectedFeatureMode === item.value }"
                  @click="selectedFeatureMode = item.value"
                >
                  <strong>{{ item.label }}</strong>
                </button>
              </div>
            </div>

            <div class="control-group">
              <div class="control-label">水流模式</div>
              <div class="choice-grid">
                <button
                  v-for="item in waterFlowOptions"
                  :key="item.value"
                  type="button"
                  :class="{ active: selectedWaterFlowMode === item.value }"
                  @click="selectedWaterFlowMode = item.value"
                >
                  <strong>{{ item.label }}</strong>
                </button>
              </div>
            </div>
          </section>
        </div>

        <footer class="setup-footer">
          <button class="generate-button" type="button" :disabled="isGenerating || isUploadingWhiteBg" @click="startGeneration">
            <SparklesIcon class="icon-sm" />
            {{ isGenerating ? '创建中...' : '生成套图' }}
          </button>
        </footer>
      </aside>

      <main class="result-panel">
        <section class="task-section">
          <div class="section-bar">
            <div>
              <h2>最近任务</h2>
              <p v-if="latestTask">{{ latestTask.task_id }}</p>
            </div>
            <span v-if="latestTask" class="task-status">{{ taskStatusText(latestTask.status) }}</span>
          </div>

          <div v-if="latestTask" class="task-content">
            <div class="progress-row">
              <div class="progress-track">
                <span :style="{ width: `${latestTaskProgress}%` }"></span>
              </div>
              <strong>{{ latestTask.success_image_count || 0 }}/{{ latestTask.total_image_count || 10 }}</strong>
            </div>

            <div v-for="family in latestTaskFamilies" :key="family.id" class="result-family">
              <div class="family-head">
                <strong>{{ family.name }}</strong>
                <span>{{ family.desc }} · {{ family.images.length }} 张</span>
              </div>
              <div class="result-grid">
                <article v-for="item in family.images" :key="item.id" class="result-card">
                  <button class="result-thumb" type="button" :disabled="!item.image_url" @click="openLargePreview(item.image_url)">
                    <img v-if="item.status === 'done' && item.image_url" :src="resolveImageUrl(item.image_url)" alt="生成结果">
                    <span v-else class="result-state" :class="item.status">
                      <span v-if="item.status === 'running'" class="spinner"></span>
                      {{ imageStatusText(item) }}
                    </span>
                  </button>
                  <div class="result-meta">
                    <strong>{{ item.displayTitle }}</strong>
                    <small>{{ item.subtitle || item.typeName }}</small>
                  </div>
                </article>
              </div>
            </div>
          </div>

          <div v-else class="empty-task">
            <PhotoIcon class="icon-lg" />
            <strong>暂无套图任务</strong>
          </div>
        </section>
      </main>

      <div v-if="isGalleryPickerOpen" class="modal-backdrop" @click.self="closeGalleryPicker">
        <section class="gallery-modal" role="dialog" aria-modal="true">
          <header class="modal-head">
            <div>
              <h2>选择参考图方案</h2>
              <p>{{ selectedCategoryLabel }} · {{ filteredReferenceGalleries.length }} 个方案</p>
            </div>
            <button class="icon-button" type="button" @click="closeGalleryPicker">
              ×
            </button>
          </header>

          <div class="scheme-picker">
            <aside class="scheme-sidebar">
              <button
                v-for="item in filteredReferenceGalleries"
                :key="item.id"
                type="button"
                class="scheme-card"
                :class="{ active: String(galleryDetailId) === String(item.id), selected: String(selectedGalleryId) === String(item.id), warn: !!galleryMissingText(item) }"
                @click="openGalleryDetail(item)"
              >
                <span class="scheme-cover">
                  <img v-if="galleryCoverImage(item)" :src="resolveImageUrl(galleryCoverImage(item))" alt="参考方案封面">
                  <PhotoIcon v-else class="icon-lg" />
                </span>
                <span class="scheme-info">
                  <strong>{{ item.name }}</strong>
                  <small>{{ galleryMissingText(item) || `${categoryLabel(item.category)} · ${galleryCompletionText(item)}` }}</small>
                </span>
              </button>
              <div v-if="!filteredReferenceGalleries.length" class="inline-empty">
                当前类目暂无可用参考图方案。
              </div>
            </aside>

            <section class="scheme-detail">
              <div v-if="galleryDetail" class="detail-head">
                <div>
                  <h3>{{ galleryDetail.name }}</h3>
                  <p>{{ categoryLabel(galleryDetail.category) }} · {{ galleryCompletionText(galleryDetail) }}</p>
                </div>
                <button class="secondary-button compact" type="button" @click="selectGallery(galleryDetail)">
                  选择此方案
                </button>
              </div>

              <div v-if="galleryDetail" class="asset-grid">
                <article v-for="asset in galleryAssetsForDisplay(galleryDetail)" :key="asset.key" class="asset-card" :class="{ empty: !asset.url }">
                  <button class="asset-thumb" type="button" :disabled="!asset.url" @click="asset.url && openLargePreview(asset.url)">
                    <img v-if="asset.url" :src="resolveImageUrl(asset.url)" :alt="asset.label">
                    <PhotoIcon v-else class="icon-lg" />
                  </button>
                  <div>
                    <strong>{{ asset.label }}</strong>
                    <small>{{ asset.url ? '已配置' : '未配置' }}</small>
                  </div>
                </article>
              </div>

              <div v-else class="detail-empty">
                <PhotoIcon class="icon-lg" />
                <strong>选择一个方案查看参考图</strong>
              </div>

              <footer v-if="galleryDetail" class="modal-actions">
                <span v-if="galleryMissingText(galleryDetail)" class="modal-warning">缺少：{{ galleryMissingText(galleryDetail) }}</span>
                <span v-else>方案图片已满足当前功能选择。</span>
              </footer>
            </section>
          </div>
        </section>
      </div>

      <ImagePreview :visible="!!largePreviewUrl" :imageUrl="largePreviewUrl" alt="生成结果大图" @close="closeLargePreview" />
    </section>
  </MainLayout>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ArrowPathIcon,
  CloudArrowUpIcon,
  PhotoIcon,
  SparklesIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { resolveImageUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const fixedOutputRatio = '1:1'
const fixedDifferentiationMode = 'homedepot_amazon'
const fixedImageTypeConfigs = [
  { type: 'main_image', count: 1 },
  { type: 'scene', count: 3 },
  { type: 'spec_dimension', count: 1 },
  { type: 'feature', count: 3 },
  { type: 'family', count: 1 },
  { type: 'multi_color_white_bg', count: 1 },
]

const categoryOptions = [
  { value: 'kitchen', label: '厨房' },
  { value: 'bathroom', label: '浴室', disabled: true },
]

const brandOptions = [
  { value: 'forious', label: 'Forious' },
  { value: 'androme', label: 'Androme' },
  { value: 'anza', label: 'Anza' },
]

const featureModeOptions = [
  { value: 'pull_360', label: '抽拉 + 360°', desc: '同一张图展示抽拉和旋转' },
  { value: 'pull', label: '仅抽拉', desc: '匹配抽拉功能参考图' },
  { value: 'rotate_360', label: '仅 360°', desc: '匹配 360° 旋转参考图' },
]

const waterFlowOptions = [
  { value: 'triple', label: '三模式', desc: '匹配三模式水流参考图' },
  { value: 'dual', label: '双模式', desc: '匹配双模式水流参考图' },
  { value: 'single', label: '单模式', desc: '匹配单模式水流参考图' },
]

const selectedCategory = ref('kitchen')
const selectedBrandValue = ref('forious')
const selectedGalleryId = ref('')
const selectedFeatureMode = ref('pull_360')
const selectedWaterFlowMode = ref('triple')
const whiteBgImageUrl = ref('')
const whiteBgPreview = ref('')
const isUploadingWhiteBg = ref(false)
const referenceGalleries = ref([])
const recentTasks = ref([])
const isGenerating = ref(false)
const largePreviewUrl = ref('')
const pollingTimer = ref(null)
const isFetchingTasks = ref(false)
const isGalleryPickerOpen = ref(false)
const galleryDetailId = ref('')

const selectedBrand = computed(() => brandOptions.find(item => item.value === selectedBrandValue.value) || brandOptions[0])
const selectedCategoryLabel = computed(() => categoryLabel(selectedCategory.value))
const selectedGallery = computed(() => referenceGalleries.value.find(item => String(item.id) === String(selectedGalleryId.value)) || null)
const selectedGalleryMissingText = computed(() => selectedGallery.value ? galleryMissingText(selectedGallery.value) : '')
const filteredReferenceGalleries = computed(() => {
  return referenceGalleries.value.filter(item => {
    const category = normalizeCategory(item.category)
    return !category || category === selectedCategory.value
  })
})
const galleryDetail = computed(() => filteredReferenceGalleries.value.find(item => String(item.id) === String(galleryDetailId.value)) || null)
const latestTask = computed(() => recentTasks.value[0] || null)
const latestTaskProgress = computed(() => {
  const total = Number(latestTask.value?.total_image_count || 10)
  const success = Number(latestTask.value?.success_image_count || 0)
  if (!total) return 0
  return Math.max(0, Math.min(100, Math.round((success / total) * 100)))
})
const hasPendingTasks = computed(() => {
  if (!latestTask.value) return false
  const runningStatuses = ['pending', 'planning', 'queued', 'running', 'generating']
  return runningStatuses.includes(latestTask.value.status) || latestTask.value.images?.some(item => runningStatuses.includes(item.status))
})

const imageTypeFamilies = [
  { id: 'main', name: '主图', desc: '白底主图', types: ['main_image'] },
  { id: 'scene', name: '场景图', desc: '近景 / 中景 / 远景', types: ['scene'] },
  { id: 'feature', name: '功能展示图', desc: '抽拉旋转 / 水流 / 卖点', types: ['feature'] },
  { id: 'secondary', name: '其他副图', desc: '尺寸 / 全家福 / 白底多色', types: ['spec_dimension', 'family', 'multi_color_white_bg'] },
]

const imageTypeNames = {
  main_image: '白底主图',
  scene: '场景图',
  spec_dimension: '尺寸图',
  feature: '功能展示图',
  family: '全家福图',
  multi_color_white_bg: '白底多色图',
}

const latestTaskFamilies = computed(() => {
  if (!latestTask.value?.images) return []
  const mappedImages = latestTask.value.images.map((item, index) => {
    const family = imageTypeFamilies.find(f => f.types.includes(item.image_type)) || imageTypeFamilies[1]
    const status = normalizeImageStatus(item.status)
    return {
      id: `${latestTask.value.task_id}-${item.id || index}`,
      type: item.image_type,
      typeName: imageTypeNames[item.image_type] || item.image_type || '图片',
      family: family.id,
      displayTitle: item.title || imageTypeNames[item.image_type] || '生成图片',
      subtitle: item.subtitle || item.role || '',
      status,
      image_url: item.image_url || '',
      error_message: item.error_message || '',
    }
  })
  return imageTypeFamilies
    .map(family => ({ ...family, images: mappedImages.filter(item => item.family === family.id) }))
    .filter(family => family.images.length)
})

watch(filteredReferenceGalleries, (list) => {
  const currentExists = list.some(item => String(item.id) === String(selectedGalleryId.value))
  if (!currentExists) selectedGalleryId.value = ''
}, { immediate: true })

function normalizeCategory(value) {
  const text = String(value || '').trim().toLowerCase()
  if (text === '厨房' || text === 'kitchen') return 'kitchen'
  if (text === '浴室' || text === 'bathroom') return 'bathroom'
  return ''
}

function categoryLabel(value) {
  const normalized = normalizeCategory(value)
  return categoryOptions.find(item => item.value === normalized)?.label || '未分类'
}

function galleryCompletionText(item) {
  const assets = Array.isArray(item.assets) ? item.assets : []
  const required = requiredGalleryAssets()
  const hasAsset = new Set(assets.filter(asset => asset?.url).map(asset => asset.key))
  const filled = required.filter(asset => hasAsset.has(asset.key)).length
  return `${filled}/${required.length}`
}

function galleryCoverImage(item) {
  const assets = Array.isArray(item?.assets) ? item.assets : []
  return assets.find(asset => asset?.url)?.url || ''
}

function openGalleryPicker() {
  galleryDetailId.value = selectedGalleryId.value || String(filteredReferenceGalleries.value[0]?.id || '')
  isGalleryPickerOpen.value = true
}

function closeGalleryPicker() {
  isGalleryPickerOpen.value = false
  galleryDetailId.value = ''
}

function openGalleryDetail(item) {
  galleryDetailId.value = String(item?.id || '')
}

function selectGallery(item) {
  selectedGalleryId.value = String(item?.id || '')
  closeGalleryPicker()
}

function selectedFeatureAssetKey() {
  if (selectedFeatureMode.value === 'pull_360') return 'feature_pull_360'
  if (selectedFeatureMode.value === 'pull') return 'feature_pull'
  return 'feature_360'
}

function selectedWaterFlowAssetKey() {
  if (selectedWaterFlowMode.value === 'single') return 'water_flow_single'
  if (selectedWaterFlowMode.value === 'dual') return 'water_flow_dual'
  return 'water_flow_triple'
}

function requiredGalleryAssets() {
  return [
    { key: selectedFeatureAssetKey(), label: '抽拉/360°参考图' },
    { key: selectedWaterFlowAssetKey(), label: '水流模式参考图' },
    { key: 'feature_selling_point', label: '卖点图参考图' },
    { key: 'dimension', label: '尺寸图参考' },
    { key: 'family', label: '全家福参考图' },
    { key: 'multi_color_white_bg', label: '白底多色参考图' },
  ]
}

function galleryAssetsForDisplay(item) {
  const assetMap = new Map((Array.isArray(item?.assets) ? item.assets : []).map(asset => [asset.key, asset]))
  return requiredGalleryAssets().map(required => {
    const asset = assetMap.get(required.key) || {}
    return {
      key: required.key,
      label: required.label,
      url: asset.url || '',
    }
  })
}

function galleryMissingText(item) {
  const assets = Array.isArray(item?.assets) ? item.assets : []
  const hasAsset = new Set(assets.filter(asset => asset?.url).map(asset => asset.key))
  return requiredGalleryAssets()
    .filter(asset => !hasAsset.has(asset.key))
    .map(asset => asset.label)
    .join('、')
}

function normalizeImageStatus(status) {
  if (status === 'success' || status === 'done') return 'done'
  if (status === 'failed') return 'failed'
  if (['pending', 'planning', 'queued', 'running', 'generating'].includes(status)) return 'running'
  return 'queued'
}

function imageStatusText(item) {
  if (item.status === 'done') return '已完成'
  if (item.status === 'failed') return item.error_message || '生成失败'
  if (item.status === 'running') return '生成中'
  return '等待中'
}

function taskStatusText(status) {
  const map = {
    pending: '等待中',
    planning: '规划中',
    queued: '排队中',
    running: '生成中',
    generating: '生成中',
    completed: '已完成',
    success: '已完成',
    failed: '失败',
  }
  return map[status] || status || '处理中'
}

async function fetchReferenceGalleries() {
  const res = await request.get('/reference-galleries', { params: { status: 'enabled' } })
  referenceGalleries.value = responseList(res.data)
}

async function fetchRecentTasks() {
  if (isFetchingTasks.value) return
  isFetchingTasks.value = true
  try {
    const res = await request.get('/ecom-suite/tasks')
    const list = responseList(res.data)
    if (!list.length) {
      recentTasks.value = []
      stopTaskPolling()
      return
    }
    const detailRequests = list.slice(0, 10).map(item => request.get(`/ecom-suite/tasks/${item.task_id}`))
    const detailResults = await Promise.allSettled(detailRequests)
    recentTasks.value = detailResults.filter(item => item.status === 'fulfilled').map(item => item.value.data).slice(0, 1)
    if (hasPendingTasks.value) startTaskPolling()
    else stopTaskPolling()
  } finally {
    isFetchingTasks.value = false
  }
}

function responseList(data) {
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.data)) return data.data
  return []
}

async function initLibraries() {
  try {
    await Promise.all([fetchReferenceGalleries(), fetchRecentTasks()])
  } catch {
    toast.error('获取参考图库或任务列表失败')
  }
}

function fileToDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = ev => resolve(ev.target?.result || '')
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function uploadImageFile(file) {
  const dataUrl = await fileToDataUrl(file)
  const [meta, data] = String(dataUrl).split(',')
  const mimeType = String(meta || '').split(':')[1]?.split(';')[0] || file.type || 'image/png'
  const res = await request.post('/upload/image', { data, mime_type: mimeType })
  return { url: res.data?.url || '', preview: String(dataUrl) }
}

async function handleWhiteBgChange(e) {
  const file = e.target.files?.[0]
  if (e.target) e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) return toast.error('请上传图片文件')
  isUploadingWhiteBg.value = true
  try {
    const result = await uploadImageFile(file)
    whiteBgImageUrl.value = result.url
    whiteBgPreview.value = result.preview
  } catch (err) {
    toast.error(err?.response?.data?.error || '白底图上传失败')
  } finally {
    isUploadingWhiteBg.value = false
  }
}

function clearWhiteBg() {
  whiteBgImageUrl.value = ''
  whiteBgPreview.value = ''
}

function startTaskPolling() {
  if (pollingTimer.value || !hasPendingTasks.value) return
  pollingTimer.value = window.setInterval(() => {
    fetchRecentTasks().catch(() => {})
  }, 5000)
}

function stopTaskPolling() {
  if (pollingTimer.value) {
    window.clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

const openLargePreview = url => { largePreviewUrl.value = resolveImageUrl(url || '') }
const closeLargePreview = () => { largePreviewUrl.value = '' }
const buildImageTypeConfigs = () => fixedImageTypeConfigs.map(item => ({ ...item }))

async function startGeneration() {
  if (!whiteBgImageUrl.value) return toast.error('请先上传产品白底图')
  if (!selectedGalleryId.value) return toast.error('请选择参考图方案')
  if (selectedGalleryMissingText.value) return toast.error(`当前参考图方案缺少：${selectedGalleryMissingText.value}`)
  if (!selectedFeatureMode.value) return toast.error('请选择抽拉/360°模式')
  if (!selectedWaterFlowMode.value) return toast.error('请选择水流模式')
  isGenerating.value = true
  try {
    await request.post('/ecom-suite/tasks', {
      platform: 'amazon',
      language: 'en',
      country_code: 'us',
      country_name: '美国',
      output_ratio: fixedOutputRatio,
      image_type_configs: buildImageTypeConfigs(),
      brand: selectedBrandValue.value,
      differentiation_mode: fixedDifferentiationMode,
      white_bg_image_url: whiteBgImageUrl.value,
      reference_gallery_id: Number(selectedGalleryId.value),
      water_flow_mode: selectedWaterFlowMode.value,
      feature_mode: selectedFeatureMode.value,
    })
    toast.success('套图任务已创建')
    await fetchRecentTasks()
    startTaskPolling()
  } catch (error) {
    toast.error(error?.response?.data?.error || '创建套图任务失败')
  } finally {
    isGenerating.value = false
  }
}

onMounted(() => { initLibraries() })
onBeforeUnmount(() => { stopTaskPolling() })
</script>

<style scoped>
.suite-page {
  display: grid;
  grid-template-columns: minmax(360px, 430px) minmax(0, 1fr);
  gap: 16px;
  width: min(1480px, calc(100vw - 32px));
  height: calc(100vh - 88px);
  margin: 0 auto;
  padding: 16px 0;
  color: #172033;
}

.setup-panel,
.result-panel,
.setup-section,
.task-section {
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fff;
}

.setup-panel {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.setup-header {
  padding: 18px 18px 14px;
  border-bottom: 1px solid #edf0f5;
}

.setup-header p,
.setup-header h1,
.section-title h2,
.section-title p,
.section-bar h2,
.section-bar p {
  margin: 0;
}

.setup-header p {
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}

.setup-header h1 {
  margin-top: 4px;
  font-size: 22px;
  font-weight: 800;
}

.setup-scroll {
  display: grid;
  gap: 12px;
  min-height: 0;
  padding: 14px;
  overflow: auto;
}

.setup-section {
  display: grid;
  gap: 14px;
  padding: 14px;
}

.section-title {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.section-title > span {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: #111827;
  color: #fff;
  font-size: 12px;
  font-weight: 800;
}

.section-title h2 {
  font-size: 15px;
  font-weight: 800;
}

.section-title p {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
  line-height: 1.45;
}

.upload-drop {
  position: relative;
  display: grid;
  place-items: center;
  min-height: 188px;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border: 1px dashed #bcc6d4;
  border-radius: 8px;
  background: #f8fafc;
  cursor: pointer;
  transition: border-color .2s, background .2s, box-shadow .2s;
}

.upload-drop:hover {
  border-color: #2563eb;
  background: #fff;
  box-shadow: 0 10px 24px rgba(15, 23, 42, .08);
}

.upload-drop input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.upload-drop img {
  display: block;
  max-width: calc(100% - 24px);
  max-height: calc(100% - 24px);
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 0 0 1px rgba(15, 23, 42, .06), 0 10px 24px rgba(15, 23, 42, .08);
}

.upload-drop.busy {
  pointer-events: none;
  opacity: .7;
}

.upload-empty {
  display: grid;
  place-items: center;
  gap: 10px;
  color: #667085;
  font-size: 13px;
}

.icon-lg {
  width: 34px;
  height: 34px;
}

.icon-sm {
  width: 16px;
  height: 16px;
}

.text-action,
.icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid transparent;
  background: transparent;
  color: #2563eb;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

.text-action {
  justify-self: start;
}

.control-group {
  display: grid;
  gap: 8px;
}

.control-label {
  color: #4b5563;
  font-size: 12px;
  font-weight: 800;
}

.segmented-control {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: 3px;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #f4f6f9;
}

.segmented-control button,
.choice-grid button,
.brand-card,
.generate-button,
.icon-button {
  border-radius: 8px;
  transition: border-color .2s, box-shadow .2s, background .2s, transform .2s;
}

.segmented-control button {
  min-height: 36px;
  border: 0;
  background: transparent;
  color: #475467;
  font-size: 14px;
  font-weight: 800;
  cursor: pointer;
}

.segmented-control button.active {
  background: #fff;
  color: #111827;
  box-shadow: 0 1px 3px rgba(15, 23, 42, .1);
}

.segmented-control button.disabled {
  color: #a0a8b5;
  cursor: not-allowed;
  opacity: .55;
}

.brand-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.brand-card {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 46px;
  padding: 12px;
  border: 1px solid #e4e8ef;
  background: #fff;
  text-align: center;
  cursor: pointer;
}

.brand-card:hover,
.choice-grid button:hover,
.icon-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 18px rgba(15, 23, 42, .08);
}

.brand-card strong,
.choice-grid strong,
.result-meta strong {
  color: #172033;
  font-size: 13px;
}

.brand-card small,
.choice-grid small,
.result-meta small {
  color: #667085;
  font-size: 12px;
  line-height: 1.35;
}

.brand-card.active {
  border-color: #111827;
  background: #111827;
  box-shadow: 0 0 0 3px rgba(17, 24, 39, .1);
}

.brand-card.active strong {
  color: #fff;
}

.brand-card.androme.active {
  border-color: #111827;
  box-shadow: 0 0 0 3px rgba(17, 24, 39, .1);
}

.brand-card.anza.active {
  border-color: #111827;
  box-shadow: 0 0 0 3px rgba(17, 24, 39, .1);
}

.selected-gallery-card {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fbfcfe;
}

.selected-gallery-card img,
.gallery-cover-empty {
  width: 70px;
  height: 54px;
  border-radius: 6px;
  background: #f3f6fa;
}

.selected-gallery-card img {
  object-fit: cover;
}

.gallery-cover-empty {
  display: grid;
  place-items: center;
  color: #98a2b3;
}

.selected-gallery-card span {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.selected-gallery-card strong {
  overflow: hidden;
  color: #172033;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-gallery-card small {
  color: #667085;
  font-size: 12px;
  line-height: 1.4;
}

.inline-empty {
  padding: 12px;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  background: #f8fafc;
  color: #667085;
  font-size: 13px;
}

.inline-empty.warn {
  border-color: #fecaca;
  background: #fef2f2;
  color: #b91c1c;
}

.choice-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.choice-grid button {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 46px;
  padding: 10px;
  border: 1px solid #e4e8ef;
  background: #fff;
  text-align: center;
  cursor: pointer;
}

.choice-grid button.active {
  border-color: #111827;
  background: #f8fafc;
  box-shadow: 0 0 0 3px rgba(17, 24, 39, .08);
}

.setup-footer {
  display: grid;
  padding: 14px;
  border-top: 1px solid #edf0f5;
  background: #fff;
}

.generate-button,
.secondary-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  height: 44px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 800;
  cursor: pointer;
}

.generate-button {
  border: 0;
  background: #111827;
  color: #fff;
}

.secondary-button {
  border: 1px solid #d7dde8;
  background: #fff;
  color: #172033;
}

.generate-button:disabled {
  cursor: not-allowed;
  opacity: .62;
}

.result-panel {
  display: grid;
  grid-template-rows: minmax(0, 1fr);
  min-width: 0;
  min-height: 0;
  padding: 12px;
  background: #f7f9fc;
}

.task-section {
  padding: 14px;
}

.board-head,
.section-bar,
.family-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.board-head h2,
.section-bar h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}

.board-head p,
.section-bar p {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
}

.icon-button {
  width: 34px;
  height: 34px;
  border: 1px solid #e4e8ef;
  background: #fff;
  color: #475467;
}

.icon-button:disabled {
  opacity: .6;
}

.section-bar > span,
.task-status {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  background: #eef2ff;
  color: #3730a3;
  font-size: 12px;
  font-weight: 800;
}

.task-section {
  min-height: 0;
  overflow: auto;
}

.task-content {
  display: grid;
  gap: 18px;
  margin-top: 12px;
}

.progress-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.progress-track {
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: #e4e8ef;
}

.progress-track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #16a34a;
  transition: width .3s;
}

.progress-row strong {
  color: #172033;
  font-size: 12px;
}

.result-family {
  display: grid;
  gap: 10px;
}

.family-head strong {
  font-size: 14px;
}

.family-head span {
  color: #667085;
  font-size: 12px;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 10px;
}

.result-card {
  overflow: hidden;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fff;
}

.result-thumb {
  display: grid;
  place-items: center;
  width: 100%;
  aspect-ratio: 1;
  border: 0;
  background: #f3f6fa;
  cursor: zoom-in;
}

.result-thumb:disabled {
  cursor: default;
}

.result-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.result-state {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  color: #667085;
  font-size: 12px;
  font-weight: 800;
}

.result-state.failed {
  color: #b91c1c;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid #cbd5e1;
  border-top-color: #2563eb;
  border-radius: 999px;
  animation: spin .8s linear infinite;
}

.result-meta {
  display: grid;
  gap: 2px;
  padding: 9px 10px 10px;
}

.empty-task {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  min-height: 300px;
  color: #667085;
  text-align: center;
}

.empty-task strong {
  color: #172033;
  font-size: 15px;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(15, 23, 42, .5);
  backdrop-filter: blur(3px);
}

.gallery-modal {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  width: min(1180px, calc(100vw - 48px));
  min-width: min(980px, calc(100vw - 48px));
  min-height: min(700px, calc(100vh - 48px));
  max-height: min(860px, calc(100vh - 48px));
  overflow: hidden;
  border: 1px solid rgba(226, 232, 240, .9);
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 24px 70px rgba(15, 23, 42, .22);
}

.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 22px;
  border-bottom: 1px solid #edf0f5;
  background: linear-gradient(180deg, #fff 0%, #f8fafc 100%);
}

.modal-head h2,
.modal-head p {
  margin: 0;
}

.modal-head h2 {
  color: #172033;
  font-size: 18px;
  font-weight: 800;
}

.modal-head p {
  margin-top: 3px;
  color: #667085;
  font-size: 13px;
}

.scheme-picker {
  display: grid;
  grid-template-columns: 360px minmax(0, 1fr);
  min-height: 0;
  background: #f8fafc;
}

.scheme-sidebar {
  display: grid;
  align-content: start;
  gap: 12px;
  min-height: 0;
  padding: 16px;
  border-right: 1px solid #edf0f5;
  background: #f1f5f9;
  overflow: auto;
}

.scheme-card {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  min-height: 98px;
  padding: 12px;
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: #fff;
  text-align: left;
  cursor: pointer;
  transition: border-color .2s, box-shadow .2s, transform .2s;
}

.scheme-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 26px rgba(15, 23, 42, .1);
}

.scheme-card.active {
  border-color: #2563eb;
  background: #f8fbff;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, .12), 0 12px 26px rgba(15, 23, 42, .08);
}

.scheme-card.selected:not(.active) {
  border-color: #93c5fd;
}

.scheme-card.warn {
  border-color: #fecaca;
}

.scheme-cover {
  display: grid;
  place-items: center;
  width: 104px;
  height: 76px;
  overflow: hidden;
  border-radius: 8px;
  background: #f3f6fa;
  color: #98a2b3;
}

.scheme-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.scheme-info {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.scheme-info strong,
.asset-card strong {
  color: #172033;
  font-size: 13px;
}

.scheme-info small,
.asset-card small {
  color: #667085;
  font-size: 12px;
  line-height: 1.35;
}

.scheme-detail {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 16px;
  min-height: 0;
  padding: 20px;
  background: #fff;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: #fbfcfe;
}

.detail-head h3,
.detail-head p {
  margin: 0;
}

.detail-head h3 {
  color: #172033;
  font-size: 17px;
  font-weight: 800;
}

.detail-head p {
  margin-top: 4px;
  color: #667085;
  font-size: 13px;
}

.secondary-button.compact {
  width: auto;
  min-width: 132px;
  height: 38px;
  padding: 0 16px;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 14px;
  min-height: 0;
  padding-right: 4px;
  overflow: auto;
}

.asset-card {
  display: grid;
  grid-template-rows: auto minmax(44px, auto);
  gap: 10px;
  padding: 12px;
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 8px 22px rgba(15, 23, 42, .06);
}

.asset-card > div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.asset-card.empty {
  border-style: dashed;
  background: #f8fafc;
  box-shadow: none;
}

.asset-thumb {
  display: grid;
  place-items: center;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  border: 0;
  border-radius: 8px;
  background: #f3f6fa;
  color: #98a2b3;
  cursor: zoom-in;
}

.asset-thumb:disabled {
  cursor: default;
}

.asset-thumb img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #fff;
}

.modal-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 36px;
  padding: 12px 14px;
  border: 1px solid #dcfce7;
  border-radius: 10px;
  background: #f0fdf4;
  color: #047857;
  font-size: 12px;
  font-weight: 700;
}

.modal-warning {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  background: #fef2f2;
  color: #b91c1c;
  font-size: 12px;
  font-weight: 700;
}

.detail-empty {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  min-height: 360px;
  border: 1px dashed #d7dde8;
  border-radius: 8px;
  background: #f8fafc;
  color: #667085;
  text-align: center;
}

.detail-empty strong {
  color: #172033;
  font-size: 15px;
}

.detail-empty span {
  font-size: 13px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1180px) {
  .suite-page {
    grid-template-columns: 1fr;
    height: auto;
  }

  .setup-panel,
  .result-panel {
    min-height: auto;
  }

}

@media (max-width: 760px) {
  .suite-page {
    width: calc(100vw - 20px);
    padding: 10px 0;
  }

  .brand-grid,
  .choice-grid {
    grid-template-columns: 1fr;
  }

  .modal-backdrop {
    padding: 10px;
  }

  .gallery-modal {
    width: calc(100vw - 20px);
    min-width: 0;
    min-height: calc(100vh - 20px);
    max-height: calc(100vh - 20px);
  }

  .scheme-picker {
    grid-template-columns: 1fr;
  }

  .scheme-sidebar {
    max-height: 260px;
    border-right: 0;
    border-bottom: 1px solid #edf0f5;
  }

  .detail-head {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
