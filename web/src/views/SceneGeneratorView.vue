<template>
  <MainLayout>
    <section class="sg">
      <div class="panel left">
        <div class="body">
          <section class="sec">
            <div class="head">
              <h3>1. 上传产品图</h3>
              <button class="asset-mini-btn" type="button" @click="openAssetPicker('product')">从资产库选择产品图</button>
            </div>
            <label :class="['drop', preview && 'has', isDraggingProduct && 'dragging']" @dragover="onProductDragOver"
              @dragleave="onProductDragLeave" @drop="onProductDrop">
              <input type="file" accept="image/*" @change="pickFile">
              <template v-if="preview"><img :src="preview" class="pv" alt="预览"><button class="rm" type="button"
                  @click.stop.prevent="clearFile">✕</button>
                <div class="ov">点击更换 / 拖拽替换</div>
              </template>
              <div v-else class="emp"><span>⬆</span><strong>上传产品白底图</strong><small>或拖拽图片到此处</small></div>
            </label>
          </section>

          <section class="sec config-sec">
            <div class="head">
              <h3>2. 生成配置</h3>
            </div>
            <div class="config-board">
              <section class="config-panel config-panel-wide">
                <div class="config-panel-head">
                  <div>
                    <strong>场景信息</strong>
                  </div>
                </div>
                <div class="config-stack">
                  <div>
                    <p class="tt">使用地点</p>
                    <div class="chips category-chips compact-grid compact-grid-3"><button v-for="i in categories"
                        :key="i" type="button" :class="['chip', category === i && 'on']" @click="category = i">{{ i
                        }}</button>
                    </div>
                    <p class="tt mt">生活人物</p>
                    <button type="button" :class="['switch-row', includePerson && 'on']"
                      @click="includePerson = !includePerson">
                      <span class="switch-text">{{ includePerson ? '出现人物' : '不出现人物' }}</span>
                      <span class="switch-track"><span class="switch-thumb"></span></span>
                    </button>
                  </div>
                  <div>
                    <p class="tt mt">产品换色（不选则为产品原色）</p>
                    <div class="color-swatches">
                      <button v-for="cf in colorFamilies" :key="cf.id" type="button"
                        :class="['color-swatch', selectedColorId === cf.id && 'on']"
                        @click="selectedColorId = selectedColorId === cf.id ? null : cf.id">
                        <span class="swatch-img">
                          <img v-if="cf.image_url" :src="resolveImageUrl(cf.image_url)" :alt="cf.name">
                          <span v-else class="swatch-initial">{{ cf.name.charAt(0).toUpperCase() }}</span>
                        </span>
                        <span class="swatch-name">{{ cf.name }}</span>
                      </button>
                    </div>
                    <div class="custom-color-section" @dragover="onColorDragOver" @dragleave="onColorDragLeave"
                      @drop="onColorDrop">
                      <p class="tt mt-sm">自定义颜色方案</p>
                      <label :class="['custom-img-upload', isDraggingColor && 'dragging']">
                        <input type="file" accept="image/*" @change="pickCustomColorImg" class="hidden-input">
                        <span class="upload-btn">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                            stroke-width="2">
                            <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M17 8l-5-5-5 5M12 3v12" />
                          </svg>
                          {{ customColorImage ? '更换参考图' : '上传颜色参考图（支持拖拽上传）' }}
                        </span>
                        <button v-if="customColorImage" type="button" class="clear-custom-btn"
                      @click.stop="clearCustomColor">✕</button>
                      </label>
                      <div v-if="customColorImage" class="custom-color-preview">
                        <img :src="customColorImage" alt="自定义颜色参考图">
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <section class="config-panel cfg-row-panel">
                <div class="cfg-row">
                  <div class="cfg-row-item">
                    <p class="cfg-row-label">画面规格</p>
                    <div class="chips compact quality-chips compact-grid compact-grid-2"><button v-for="i in qualities"
                        :key="i" type="button" :class="['chip', quality === i && 'on']" @click="quality = i">{{ i
                        }}</button>
                    </div>
                  </div>
                  <div class="cfg-row-item">
                    <p class="cfg-row-label">镜头</p>
                    <div class="chips compact lens-chips compact-grid compact-grid-3">
                      <button v-for="l in lensOptions" :key="l.value" type="button"
                        :class="['chip', lens === l.value && 'on']" @click="lens = l.value">{{ l.label }}</button>
                    </div>
                  </div>
                  <div class="cfg-row-item">
                    <p class="cfg-row-label">输出数量</p>
                    <div class="chips compact count-chips compact-grid compact-grid-4"><button v-for="i in countOptions"
                        :key="i" type="button" :class="['chip', count === i && 'on']" @click="count = i">{{ i }}
                        张</button></div>
                  </div>
                </div>
              </section>

              <section class="config-panel config-panel-wide ratio-panel">
                <div class="config-panel-head">
                  <div>
                    <strong>图片比例</strong>
                  </div>
                </div>
                <div class="ratio-visual-grid">
                  <button v-for="item in ratioOptions" :key="item.value" type="button"
                    :class="['ratio-card', ratio === item.value && 'on']" @click="ratio = item.value">
                    <span class="ratio-card-frame">
                      <span :class="['ratio-card-shape', item.shapeClass]"></span>
                    </span>
                    <span class="ratio-card-label">{{ item.value }}</span>
                  </button>
                </div>
              </section>
            </div>
          </section>

          <section class="sec style-reference-sec">
            <div class="head">
              <h3>{{ supportsStyleReference ? '3. 风格参考图' : '3. 场景细节' }}</h3>
            </div>
            <div v-if="supportsStyleReference" class="style-reference-picker">
              <button class="style asset-style-card" type="button" @click="openAssetPicker('style')">
                <img v-if="selectedRef" :src="imgOf(selectedRef)" alt="参考图">
                <span v-else class="asset-style-placeholder" aria-hidden="true">
                  <PhotoIcon />
                </span>
                <span class="asset-style-copy">
                  <strong>{{ selectedTag?.name || selectedStyleAsset?.name || '选择风格参考图' }}</strong>
                  <small>{{ selectedRef ? (selectedRef?.title || selectedStyleAsset?.category || '已从资产库选择') : '从资产库选择，或在资产库中上传新图' }}</small>
                </span>
                <ChevronRightIcon class="asset-style-chevron" aria-hidden="true" />
              </button>
              <button v-if="selectedRef" class="style-reference-clear" type="button" title="清除风格参考图，改为随机抽取" aria-label="清除风格参考图" @click="tagId = null; refId = null; selectedStyleAsset = null">
                <XMarkIcon />
              </button>
            </div>
            <label class="tx"><span>补充要求</span><textarea v-model="extra" rows="3"
                :placeholder="supportsStyleReference ? '例如：突出金属拉丝质感，或者希望背景更接近参考图的装修氛围' : '例如：房车里更紧凑明亮、氛围更温馨，或突出金属拉丝质感'"></textarea></label>
          </section>
        </div>
        <div class="foot">
          <p v-if="err" class="err">{{ err }}</p><button class="gen" type="button" @click="submit">{{
            running ? '场景图生成中（可继续创建）' : '开始生成' }}</button>
        </div>
      </div>

      <div class="panel right">
        <div class="rhd">
          <h2>最近任务</h2>
          <button class="ghost history-link batch-download-trigger" type="button" :disabled="!batchDownloadItems.length" @click="toggleBatchDownloadMode">
            {{ batchDownloadMode ? '取消选择' : '批量下载' }}
          </button>
        </div>
        <div class="result-scroll">
          <template v-if="recentTasks.length">
            <div class="blocks">
              <section v-for="t in recentTasks" :key="t.taskId" class="block">
                <div class="bhd">
                  <div class="meta">
                    <img v-if="t.productImageUrl" :src="resolveImageUrl(t.productImageUrl)" alt="原图">
                    <div>
                      <strong>{{ titleOf(t) }}</strong>
                      <span>{{ subOf(t) }}</span>
                    </div>
                  </div>
                  <div class="st">
                    <div class="st-top">
                      <button v-if="canDeleteTask(t)" class="icon-btn danger" type="button"
                        :disabled="deletingTaskId === t.taskId" @click="openDeleteModal(t)">
                        {{ deletingTaskId === t.taskId ? '删除中' : '删除' }}
                      </button>
                    </div>
                    <span class="task-time">{{ timeOf(t.createdAt) }}</span>
                    <div v-if="t.imageUrls?.length" class="archive-inline">
                      <template v-if="t.isSavedToDesign">
                        <span class="archive-state">已归档</span>
                      </template>
                      <template v-else>
                        <button class="archive-link" type="button"
                          :disabled="savingTaskId === t.taskId || t.status === 'queued' || t.status === 'running' || hasPendingCandidates(t)"
                          @click="saveTaskToDesign(t)">
                          {{ savingTaskId === t.taskId ? '保存中…' : '保存到我的设计' }}
                        </button>
                      </template>
                    </div>
                    <small v-if="t.errorMessage">{{ t.errorMessage }}</small>
                  </div>
                </div>
                <div class="rs compact-rs">
                  <div v-for="(image, index) in imageSlots(t)" :key="`${t.taskId}-${index}`" class="it">
                    <div class="tile compact-tile" :class="{ 'batch-selected': batchDownloadSelectedIds.includes(`${t.taskId}-${image.sort ?? index}`), 'batch-pickable': batchDownloadMode && image.imageUrl }" @click="batchDownloadMode && image.imageUrl ? toggleBatchDownloadItem(`${t.taskId}-${image.sort ?? index}`) : undefined">
                      <template v-if="image.imageUrl">
                        <img :src="sceneThumbnail(image.imageUrl, 640)" :alt="`${t.styleName}-${index + 1}`">
                        <div v-if="image.candidateStatus === 'generating'" class="tile-overlay pending-overlay">
                          <span class="loading-orbit" aria-hidden="true">
                            <span class="loading-ring"></span>
                            <span class="loading-core"></span>
                          </span>
                          <p>编辑效果生成中</p>
                          <span class="loading-text">可继续处理其他任务</span>
                        </div>
                        <div v-else-if="image.candidateStatus === 'ready'" class="tile-overlay ready-overlay">
                          <p>有新的编辑结果待确认</p>
                          <div class="candidate-actions-inline">
                            <button class="tile-action solid" type="button"
                              @click.prevent="previewCandidate(t, image)">查看对比</button>
                          </div>
                        </div>
                        <div v-else-if="image.candidateStatus === 'failed'" class="tile-overlay failed-overlay">
                          <p>编辑预览生成失败</p>
                          <span class="loading-text">{{ image.candidateErrorMessage || '请重新发起编辑' }}</span>
                          <button class="tile-action solid" type="button"
                            @click.prevent="rejectCandidate(t, image)">知道了</button>
                        </div>
                      </template>
                      <div v-else-if="image.candidateStatus === 'failed'" class="ld is-loading failed-slot">
                        <span class="loading-text">{{ image.candidateErrorMessage || '该位置生成失败' }}</span>
                      </div>
                      <div v-else class="ld is-loading">
                        <div class="progress-wrapper">
                          <div class="progress-bar">
                            <div class="progress-fill" :style="{ width: `${Math.min(progressTrackers[t.taskId?.toString()]?.current ?? 0, getProgressLimit(t.status))}%` }"></div>
                          </div>
                          <div class="progress-text">
                            <span>{{ t.status === 'running' ? '生成中' : '等待中' }}</span>
                            <span>{{ Math.round(Math.min(progressTrackers[t.taskId?.toString()]?.current ?? 0, getProgressLimit(t.status))) }}%</span>
                          </div>
                        </div>
                        <span class="loading-text">正在努力生成画面…</span>
                      </div>
                      <div v-if="image.imageUrl && !batchDownloadMode" class="tile-actions compact-actions"
                        :class="image.candidateStatus && image.candidateStatus !== 'failed' ? 'is-disabled' : ''">
                        <button class="tile-action" type="button"
                          @click.prevent="openImageViewer(resolveImageUrl(image.imageUrl), `${t.styleName || 'scene'}-${index + 1}`)">查看</button>
                        <button class="tile-action" type="button"
                          @click.prevent="downloadImage(resolveImageUrl(image.imageUrl), `scene-${t.taskId}-${index + 1}.png`)">下载</button>
                        <button class="tile-action" type="button"
                          :disabled="Boolean(image.candidateStatus) && image.candidateStatus !== 'failed'"
                          @click.prevent="openEdit(t, index, image)">编辑</button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
            <div v-if="batchDownloadMode" class="batch-inline-bar"><span>已选择 {{ batchDownloadSelectedItems.length }} 张图片</span><div><button type="button" class="ghost" @click="toggleBatchDownloadAll(!batchDownloadAllSelected)">{{ batchDownloadAllSelected ? '取消全选' : '全选' }}</button><button type="button" class="gen" :disabled="!batchDownloadSelectedItems.length || batchDownloading" @click="confirmBatchDownload">{{ batchDownloading ? `下载中 ${batchDownloadProgress}/${batchDownloadSelectedItems.length}` : '下载已选图片' }}</button></div></div>
          </template>
          <div v-else class="empty">
            <div class="empty-copy">
              <strong>暂无任务</strong>
              <!-- <button class="ghost history-link" type="button" @click="goMyDesigns">我的设计</button> -->
            </div>
          </div>
        </div>
      </div>

      <ImageEditModal :visible="editModalVisible" :imageUrl="editingImageUrl" :originalPrompt="editingPrompt"
        @close="closeEditModal" @submit="handleEditSubmit" />
      <ImagePreview :visible="viewerVisible" :imageUrl="viewerImageUrl" :alt="viewerTitle" @close="closeImageViewer" />
<div v-if="candidateModalVisible" class="mask" @click.self="closeCandidateModal">
        <div class="modal candidate-modal" @click.stop>
          <div class="mhd candidate-mhd">
            <div>
              <p class="modal-kicker">Edit Preview</p>
              <h3>确认是否替换这张图</h3>
              <p>先看编辑效果，再决定是否覆盖原图。</p>
            </div>
            <button class="modal-close" type="button" @click="closeCandidateModal">✕</button>
          </div>
          <div v-if="candidateModalImage" class="candidate-grid">
            <div class="candidate-pane">
              <span class="candidate-label">原图</span>
              <div class="candidate-frame">
                <img :src="resolveImageUrl(candidateModalImage.imageUrl)" alt="原图">
              </div>
            </div>
            <div class="candidate-pane">
              <span class="candidate-label">编辑效果</span>
              <div class="candidate-frame">
                <img :src="resolveImageUrl(candidateModalImage.candidateImageUrl)" alt="编辑效果图">
              </div>
            </div>
          </div>
          <div class="mft candidate-mft">
            <button class="ghost" type="button" :disabled="candidateActionLoading"
              @click="rejectCandidate(candidateModalTask, candidateModalImage)">
              {{ candidateActionLoading ? '处理中…' : '保留原图' }}
            </button>
            <button class="gen candidate-confirm" type="button" :disabled="candidateActionLoading"
              @click="acceptCandidate(candidateModalTask, candidateModalImage)">
              {{ candidateActionLoading ? '处理中…' : '采用编辑结果' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="deleteModalVisible" class="mask" @click.self="closeDeleteModal">
        <div class="modal confirm-modal" @click.stop>
          <div class="mhd confirm-mhd">
            <div>
              <p class="modal-kicker">Delete Task</p>
              <h3>删除这条任务记录？</h3>
              <p>删除后将从当前任务列表中移除，且不可恢复。</p>
            </div>
            <button class="modal-close" type="button" @click="closeDeleteModal">✕</button>
          </div>
          <div class="confirm-body">
            <div class="confirm-card">
              <span>任务名称</span>
              <strong>{{ pendingDeleteTask ? (titleOf(pendingDeleteTask) || pendingDeleteTask.taskId) : '未选择任务'
              }}</strong>
              <small v-if="pendingDeleteTask">{{ subOf(pendingDeleteTask) }}</small>
            </div>
          </div>
          <div class="mft confirm-mft">
            <button class="ghost" type="button" @click="closeDeleteModal">取消</button>
            <button class="danger-btn" type="button"
              :disabled="!pendingDeleteTask || deletingTaskId === pendingDeleteTask.taskId" @click="confirmRemoveTask">
              {{ deletingTaskId === pendingDeleteTask?.taskId ? '删除中' : '确认删除' }}
            </button>
          </div>
        </div>
      </div>
    </section>
    <AssetPickerModal
      :visible="assetPickerVisible"
      :title="assetPickerTitle"
      :type-filter="assetPickerTypes"
      :source-types="assetPickerSourceTypes"
      :allow-style-reference-upload="assetPickerMode === 'style'"
      @close="assetPickerVisible = false"
      @select="handleAssetPickerSelect"
    />
  </MainLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronRightIcon, PhotoIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import AssetPickerModal from '@/components/assets/AssetPickerModal.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import ImageEditModal from '@/components/ui/ImageEditModal.vue'

import request, { downloadImage, getDisplayImageUrl, getDisplayThumbnailUrl, resolveImageUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'
import { useFakeProgress } from '@/composables/useFakeProgress'

const router = useRouter()
const toast = useToast()
const fakeProgress = useFakeProgress({ storageKeyPrefix: 'scene_progress' })

// 进度跟踪器类型
interface ProgressTracker {
  current: number
}
const progressTrackers = ref<Record<string, ProgressTracker>>({})

// 获取进度条上限值
function getProgressLimit(status: string): number {
  if (status === 'running' || status === 'generating' || status === 'processing') return 99
  return 50
}

const tags = ref([])
const tagsLoading = ref(false)
const tagId = ref(null)
const refId = ref(null)
const file = ref(null)
const preview = ref('')
const imgData = ref('')
const imageMimeType = ref('image/jpeg')
const productImageUrl = ref('')
const editModalVisible = ref(false)
const editingImageUrl = ref('')
const editingPrompt = ref('')
const editingTaskId = ref('')
const editingSort = ref(-1)
const viewerVisible = ref(false)
const viewerImageUrl = ref('')
const viewerTitle = ref('')
const deleteModalVisible = ref(false)
const pendingDeleteTask = ref(null)
const candidateModalVisible = ref(false)
const candidateModalTask = ref(null)
const candidateModalImage = ref(null)
const candidateActionLoading = ref(false)
const batchDownloadMode = ref(false)
const batchDownloadSelectedIds = ref([])
const batchDownloading = ref(false)
const batchDownloadProgress = ref(0)
const category = ref('厨房水槽区')
const includePerson = ref(false)
const selectedColorId = ref(null)
const ratio = ref('1:1')
const quality = ref('2K')
const count = ref(4)
const extra = ref('')
const err = ref('')
const tasks = ref([])
const deletingTaskId = ref('')
const savingTaskId = ref('')
const colorFamilies = ref([])
const lens = ref('medium')
const sceneCategories = ref([])
const sceneCategoryId = ref(null)
const customColorImage = ref('')
const customColorImageData = ref('')
const isDraggingProduct = ref(false)
const isDraggingColor = ref(false)
const assetPickerVisible = ref(false)
const assetPickerMode = ref('product')
const selectedStyleAsset = ref(null)

let timer = null
let hasLoadedOnce = false

const categories = ['厨房水槽区', '厨房岛台', '浴室台盆区', '房车空间']
const qualities = ['2K', '4K']
const lensOptions = [
  { label: '近景', value: 'close_up' },
  { label: '中景', value: 'medium' },
  { label: '远景', value: 'wide' }
]

// 按场景过滤后的风格标签
const ratioOptions = [
  { value: '1:1', shapeClass: 'ratio-shape-square' },
  { value: '4:3', shapeClass: 'ratio-shape-landscape' },
  { value: '3:4', shapeClass: 'ratio-shape-portrait' },
  { value: '16:9', shapeClass: 'ratio-shape-wide' },
  { value: '9:16', shapeClass: 'ratio-shape-tall' },
  { value: '21:9', shapeClass: 'ratio-shape-cinema' }
]
const countOptions = [1, 2, 3, 4]
const RECENT_TASK_LIMIT = 10
const DELETE_TIMEOUT_MS = 10 * 60 * 1000

const selectedTag = computed(() => tags.value.find(i => i.id === tagId.value) || null)
const selectedRef = computed(() => selectedStyleAsset.value || selectedTag.value?.references?.find(i => i.id === refId.value) || null)
const selectedColor = computed(() => colorFamilies.value.find(c => c.id === selectedColorId.value) || null)
const hasCustomColor = computed(() => Boolean(customColorImage.value))
const supportsStyleReference = computed(() => sceneSupportsStyleReference(category.value))
const list = computed(() => [...tasks.value].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()))
const recentTasks = computed(() => list.value.slice(0, RECENT_TASK_LIMIT))
const batchDownloadItems = computed(() => recentTasks.value.flatMap(task => imageSlots(task)
  .filter(image => image.imageUrl && image.candidateStatus !== 'generating')
  .map((image, index) => ({
    id: `${task.taskId}-${image.sort ?? index}`,
    url: resolveImageUrl(image.imageUrl),
    thumbnail: sceneThumbnail(image.imageUrl, 160),
    label: `${titleOf(task)} · ${index + 1}`,
  }))))
const batchDownloadSelectedItems = computed(() => batchDownloadItems.value.filter(item => batchDownloadSelectedIds.value.includes(item.id)))
const batchDownloadAllSelected = computed(() => batchDownloadItems.value.length > 0 && batchDownloadSelectedItems.value.length === batchDownloadItems.value.length)
const running = computed(() => tasks.value.some(i => i.status === 'queued' || i.status === 'running' || hasGeneratingCandidate(i)))
const assetPickerTitle = computed(() => {
  if (assetPickerMode.value === 'style') return '从资产库选择风格参考图'
  return '从资产库选择产品图'
})
const assetPickerTypes = computed(() => {
  if (assetPickerMode.value === 'style') return ['风格参考图']
  return ['产品图', '图片', '参考方案图']
})
const assetPickerSourceTypes = computed(() => {
  if (assetPickerMode.value === 'style') return ['scene_style_reference', 'asset']
  return ['asset', 'product_spu_attachment', 'reference_gallery_asset']
})

const handleKeydown = (e) => {
  if (e.key === 'Escape') {
    if (candidateModalVisible.value) closeCandidateModal()
    else if (viewerVisible.value) closeImageViewer()
    else if (deleteModalVisible.value) closeDeleteModal()
  }
}

watch(category, () => {
  tagId.value = null
  refId.value = null
  selectedStyleAsset.value = null
})

watch(customColorImage, (newVal) => {
  if (newVal && selectedColorId.value) selectedColorId.value = null
})

watch(selectedColorId, (newVal) => {
  if (newVal) {
    customColorImage.value = ''
    customColorImageData.value = ''
  }
})

function normalizeTask(task) {
  const images = (task?.images || []).map((img, index) => ({
    sort: typeof img?.sort === 'number' ? img.sort : index,
    imageUrl: img?.imageUrl || task?.imageUrls?.[index] || '',
    candidateImageUrl: img?.candidateImageUrl || '',
    candidateStatus: img?.candidateStatus || '',
    candidateErrorMessage: img?.candidateErrorMessage || '',
    candidatePrompt: img?.candidatePrompt || '',
  }))
  if (!images.length && (task?.imageUrls || []).length) {
    return {
      ...task,
      images: (task.imageUrls || []).map((url, index) => ({ sort: index, imageUrl: url, candidateImageUrl: '', candidateStatus: '', candidateErrorMessage: '', candidatePrompt: '' }))
    }
  }
  return { ...task, images }
}

function imageSlots(task) {
  const total = Math.max(task?.imageCount || 0, task?.images?.length || 0, task?.imageUrls?.length || 0, 1)
  return Array.from({ length: total }, (_, index) => task?.images?.find(img => img.sort === index) || { sort: index, imageUrl: task?.imageUrls?.[index] || '', candidateImageUrl: '', candidateStatus: '', candidateErrorMessage: '', candidatePrompt: '' })
}

function hasGeneratingCandidate(task) {
  return imageSlots(task).some(img => img.candidateStatus === 'generating')
}

function hasPendingCandidates(task) {
  return imageSlots(task).some(img => img.candidateStatus === 'generating' || img.candidateStatus === 'ready')
}

function replaceTask(nextTask) {
  const normalized = normalizeTask(nextTask)
  tasks.value = tasks.value.some(i => i.taskId === normalized.taskId)
    ? tasks.value.map(i => i.taskId === normalized.taskId ? normalized : i)
    : [normalized, ...tasks.value]
}

const hasPendingTasks = (items = tasks.value) => items.some(i => i.status === 'queued' || i.status === 'running' || hasGeneratingCandidate(i))
const isTaskTimedOut = t => { const createdAt = t?.createdAt ? new Date(t.createdAt).getTime() : NaN; return Boolean(t?.taskId) && !t?.imageUrls?.length && Number.isFinite(createdAt) && Date.now() - createdAt >= DELETE_TIMEOUT_MS }
const canDeleteTask = t => Boolean(t) && (t.status === 'failed' || Boolean(t.errorMessage) || isTaskTimedOut(t))
const canSaveTask = t => Boolean(t?.imageUrls?.length) && t.status !== 'queued' && t.status !== 'running' && !t.isSavedToDesign && !hasPendingCandidates(t)
const imgOf = r => resolveImageUrl(r?.image_url || r?.imageUrl || r?.file_url || r?.fileUrl)
function sceneThumbnail(url, width = 480) {
  const resolved = resolveImageUrl(url || '')
  if (!resolved || resolved.startsWith('data:')) return resolved
  return getDisplayThumbnailUrl(resolved, { width, height: null, fit: 'lfit', quality: 82 })
}
const titleOf = t => [t.productCategory, t.styleName, t.productColorFamily].filter(Boolean).join('-')
const subOf = t => [t.productColorFamily, `比例 ${t.aspectRatio}`].filter(Boolean).join(' · ')
const timeOf = v => !v ? '' : (isNaN(new Date(v).getTime()) ? v : new Date(v).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }))

async function loadTags() {
  tagsLoading.value = true
  try {
    const params = {}
    if (sceneCategoryId.value) params.scene_id = sceneCategoryId.value
    const r = await request.get('/scene-style-tags', { params })
    tags.value = r.data || []
  } catch (e) {
    tags.value = []
    console.error('loadTags failed:', e)
  } finally {
    tagsLoading.value = false
  }
}
async function loadSceneCategories() {
  try {
    const r = await request.get('/scene-style-categories')
    sceneCategories.value = r.data || []
  } catch {
    sceneCategories.value = []
  }
}
function resolveSceneType(cat) {
  const t = String(cat || '').toLowerCase()
  if (t.includes('厨房') || t.includes('岛台') || t.includes('kitchen') || t.includes('island') || t.includes('厨')) return 'kitchen'
  if (t.includes('浴室') || t.includes('卫生间') || t.includes('洗手间') || t.includes('bathroom') || t.includes('浴')) return 'bathroom'
  return 'generic'
}
function sceneSupportsStyleReference(cat) {
  const t = String(cat || '').toLowerCase()
  return !(t.includes('房车') || t.includes('rv'))
}
async function loadColorFamilies() { try { const r = await request.get('/product-color-families'); colorFamilies.value = r.data || [] } catch { colorFamilies.value = [] } }
function syncSceneCategoryId() {
  const sceneType = resolveSceneType(category.value)
  if (sceneType === 'kitchen') sceneCategoryId.value = 4
  else if (sceneType === 'bathroom') sceneCategoryId.value = 3
  else sceneCategoryId.value = null
}
function openAssetPicker(mode) {
  assetPickerMode.value = mode
  if (mode === 'style') syncSceneCategoryId()
  assetPickerVisible.value = true
}

function toggleBatchDownloadMode() {
  batchDownloadMode.value = !batchDownloadMode.value
  batchDownloadSelectedIds.value = []
  batchDownloadProgress.value = 0
  
}

function toggleBatchDownloadAll(checked) {
  batchDownloadSelectedIds.value = checked ? batchDownloadItems.value.map(item => item.id) : []
}

function toggleBatchDownloadItem(id) {
  batchDownloadSelectedIds.value = batchDownloadSelectedIds.value.includes(id)
    ? batchDownloadSelectedIds.value.filter(itemId => itemId !== id)
    : [...batchDownloadSelectedIds.value, id]
}

async function confirmBatchDownload() {
  if (batchDownloading.value || !batchDownloadSelectedItems.value.length) return
  batchDownloading.value = true
  batchDownloadProgress.value = 0
  let failed = 0
  let downloadMeta = {}
  for (const [index, item] of batchDownloadSelectedItems.value.entries()) {
    try {
      const result = await downloadImage(item.url, `scene-batch-${index + 1}.png`, index === 0 ? {} : { brand: downloadMeta.brand, sku: downloadMeta.sku, skipSkuPicker: true })
      if (index === 0 && result === null) break
      if (index === 0 && result) downloadMeta = result
    } catch {
      failed += 1
    }
    batchDownloadProgress.value = index + 1
    await new Promise(resolve => setTimeout(resolve, 180))
  }
  batchDownloading.value = false
  batchDownloadMode.value = false
  toast[failed ? 'error' : 'success'](failed ? `${failed} 张图片下载失败，请重试` : `已逐张下载 ${batchDownloadProgress.value} 张图片`)
}
function clearFile() { file.value = null; preview.value = ''; imgData.value = ''; productImageUrl.value = ''; imageMimeType.value = 'image/jpeg'; closeEditModal() }
function onProductDragOver(e) { e.preventDefault(); isDraggingProduct.value = true }
function onProductDragLeave() { isDraggingProduct.value = false }
function onProductDrop(e) { e.preventDefault(); isDraggingProduct.value = false; const f = e.dataTransfer?.files?.[0]; if (f && f.type.startsWith('image/')) { file.value = f; productImageUrl.value = ''; imageMimeType.value = f.type || 'image/jpeg'; const r = new FileReader(); r.onload = x => { const s = String(x.target?.result || ''); preview.value = s; imgData.value = s.includes(',') ? s.split(',')[1] : s }; r.readAsDataURL(f) } }
function pickCustomColorImg(e) { const f = e.target.files?.[0]; if (!f) return; const r = new FileReader(); r.onload = x => { const s = String(x.target?.result || ''); customColorImage.value = s; customColorImageData.value = s.includes(',') ? s.split(',')[1] : s }; r.readAsDataURL(f) }
function clearCustomColor() { customColorImage.value = ''; customColorImageData.value = '' }
function onColorDragOver(e) { e.preventDefault(); isDraggingColor.value = true }
function onColorDragLeave() { isDraggingColor.value = false }
function onColorDrop(e) { e.preventDefault(); isDraggingColor.value = false; const f = e.dataTransfer?.files?.[0]; if (f && f.type.startsWith('image/')) { const r = new FileReader(); r.onload = x => { const s = String(x.target?.result || ''); customColorImage.value = s; customColorImageData.value = s.includes(',') ? s.split(',')[1] : s }; r.readAsDataURL(f) } }
function pickFile(e) { const f = e.target.files?.[0]; if (!f) return; file.value = f; productImageUrl.value = ''; imageMimeType.value = f.type || 'image/jpeg'; const r = new FileReader(); r.onload = x => { const s = String(x.target?.result || ''); preview.value = s; imgData.value = s.includes(',') ? s.split(',')[1] : s }; r.readAsDataURL(f) }
async function handleAssetPickerSelect(asset) {
  if (!asset?.fileUrl) return
  try {
    if (assetPickerMode.value === 'style') {
      if (asset.sourceType === 'asset') {
        selectedStyleAsset.value = asset
        tagId.value = null
        refId.value = null
      } else {
        await loadTags()
        const nextRefId = Number(asset.sourceId) || null
        const nextTag = tags.value.find(t => t.id === asset.referenceTagId && (t.references || []).some(r => r.id === nextRefId))
        if (!nextTag || !nextRefId) {
          toast.warning('这张风格图不属于当前使用地点')
          return
        }
        selectedStyleAsset.value = null
        tagId.value = nextTag.id
        refId.value = nextRefId
      }
      assetPickerVisible.value = false
      toast.success('已选择风格参考图')
      return
    }
    file.value = null
    productImageUrl.value = asset.fileUrl
    preview.value = getDisplayImageUrl(asset.fileUrl)
    imgData.value = ''
    imageMimeType.value = mimeTypeFromUrl(asset.fileUrl)
    toast.success('已应用产品图')
    assetPickerVisible.value = false
  } catch {
    toast.error('资产图片读取失败，请换一张试试')
  }
}

function mimeTypeFromUrl(url = '') {
  const path = String(url).split('?')[0].split('#')[0].toLowerCase()
  if (path.endsWith('.jpg') || path.endsWith('.jpeg')) return 'image/jpeg'
  if (path.endsWith('.webp')) return 'image/webp'
  if (path.endsWith('.gif')) return 'image/gif'
  if (path.endsWith('.bmp')) return 'image/bmp'
  if (path.endsWith('.avif')) return 'image/avif'
  if (path.endsWith('.svg')) return 'image/svg+xml'
  return 'image/png'
}
function openImageViewer(url, title = '场景图预览') { viewerImageUrl.value = url; viewerTitle.value = title; viewerVisible.value = true }
function closeImageViewer() { viewerVisible.value = false; viewerImageUrl.value = ''; viewerTitle.value = '' }
function goMyDesigns() { router.push({ path: '/my-designs', query: { source: 'scene_generator' } }) }
function notifySyncedToDesigns() { toast.success('任务已创建，右侧会持续刷新进度') }
function openDeleteModal(task) { if (!task?.taskId || deletingTaskId.value) return; pendingDeleteTask.value = task; deleteModalVisible.value = true }
function closeDeleteModal() { if (deletingTaskId.value) return; deleteModalVisible.value = false; pendingDeleteTask.value = null }
function openEdit(task, index, image) { editingTaskId.value = task.taskId; editingSort.value = index; editingImageUrl.value = resolveImageUrl(image?.imageUrl || task?.imageUrls?.[index] || ''); editingPrompt.value = image?.candidatePrompt || extra.value || `${titleOf(task)} · 第${index + 1}张`; editModalVisible.value = true }
function closeEditModal() { editModalVisible.value = false; editingImageUrl.value = ''; editingPrompt.value = ''; editingTaskId.value = ''; editingSort.value = -1 }
function previewCandidate(task, image) { candidateModalTask.value = task; candidateModalImage.value = image; candidateModalVisible.value = true }
function closeCandidateModal() { if (candidateActionLoading.value) return; candidateModalVisible.value = false; candidateModalTask.value = null; candidateModalImage.value = null }

async function confirmRemoveTask() {
  const task = pendingDeleteTask.value
  if (!task?.taskId || deletingTaskId.value) return
  deletingTaskId.value = task.taskId
  try {
    await request.delete(`/scene-generation/tasks/${task.taskId}`)
    tasks.value = tasks.value.filter(i => i.taskId !== task.taskId)
    deleteModalVisible.value = false
    pendingDeleteTask.value = null
    toast.success('任务已删除')
    if (tasks.value.length === 0) stop()
  } catch (e) {
    toast.error(e?.response?.data?.error || '删除任务失败，请稍后重试')
  } finally {
    deletingTaskId.value = ''
  }
}

async function saveTaskToDesign(task) {
  if (!task?.taskId || savingTaskId.value || !canSaveTask(task)) return
  savingTaskId.value = task.taskId
  try {
    const r = await request.post(`/scene-generation/tasks/${task.taskId}/save`)
    const savedTask = r.data?.task
    if (savedTask?.taskId) replaceTask(savedTask)
    else tasks.value = tasks.value.map(i => i.taskId === task.taskId ? { ...i, isSavedToDesign: true, savedAt: new Date().toISOString() } : i)
    toast.success(r.data?.message || '已保存到我的设计')
  } catch (e) {
    toast.error(e?.response?.data?.error || '保存失败，请稍后重试')
  } finally {
    savingTaskId.value = ''
  }
}

async function handleEditSubmit({ prompt: editPromptText, imageData, mimeType, quality }) {
  const taskId = editingTaskId.value
  const sort = editingSort.value
  closeEditModal()
  if (!taskId || sort < 0) return
  try {
    const r = await request.post(`/scene-generation/tasks/${taskId}/images/${sort}/edit`, { prompt: editPromptText?.trim() || '', imageData, mimeType: mimeType || 'image/png', quality: quality || '2k' })
    if (r.data?.task) replaceTask(r.data.task)
    toast.success(r.data?.message || '已开始生成编辑预览')
    await fetchTasks()
    poll()
  } catch (e) {
    toast.error(e?.response?.data?.error || '编辑预览生成失败，请稍后重试')
  }
}

async function acceptCandidate(task, image) {
  if (!task?.taskId || candidateActionLoading.value) return
  candidateActionLoading.value = true
  let shouldCloseModal = false
  try {
    const r = await request.post(`/scene-generation/tasks/${task.taskId}/images/${image.sort}/accept`)
    if (r.data?.task) replaceTask(r.data.task)
    shouldCloseModal = Boolean(candidateModalVisible.value && candidateModalTask.value?.taskId === task.taskId && candidateModalImage.value?.sort === image.sort)
    toast.success(r.data?.message || '已采用编辑结果')
  } catch (e) {
    toast.error(e?.response?.data?.error || '采用失败，请稍后重试')
  } finally {
    candidateActionLoading.value = false
    if (shouldCloseModal) closeCandidateModal()
  }
}

async function rejectCandidate(task, image) {
  if (!task?.taskId || candidateActionLoading.value) return
  candidateActionLoading.value = true
  let shouldCloseModal = false
  try {
    const r = await request.post(`/scene-generation/tasks/${task.taskId}/images/${image.sort}/reject`)
    if (r.data?.task) replaceTask(r.data.task)
    shouldCloseModal = Boolean(candidateModalVisible.value && candidateModalTask.value?.taskId === task.taskId && candidateModalImage.value?.sort === image.sort)
    toast.success(r.data?.message || '已保留原图')
  } catch (e) {
    toast.error(e?.response?.data?.error || '操作失败，请稍后重试')
  } finally {
    candidateActionLoading.value = false
    if (shouldCloseModal) closeCandidateModal()
  }
}

async function submit() {
  err.value = ''
  if (!imgData.value && !productImageUrl.value) return err.value = '请先上传或从资产库选择 1 张产品图'
  try {
    const useStyle = supportsStyleReference.value
    const r = await request.post('/scene-generation/tasks', { productCategory: category.value, styleTagId: useStyle && !selectedStyleAsset.value ? tagId.value : 0, styleReferenceId: useStyle && !selectedStyleAsset.value ? (refId.value || undefined) : undefined, styleReferenceImageUrl: useStyle ? (selectedStyleAsset.value?.fileUrl || '') : '', includePerson: includePerson.value, aspectRatio: ratio.value, productColorFamily: selectedColor.value?.name || (hasCustomColor.value ? '自定义' : '') || '', productColorFamilyImageUrl: selectedColor.value?.image_url || '', productColorFamilyImageData: hasCustomColor.value ? customColorImageData.value : '', productColorFamilyImageMimeType: hasCustomColor.value ? (customColorImage.value.startsWith('data:image/png') ? 'image/png' : 'image/jpeg') : '', imageSize: quality.value, imageCount: count.value, extraPrompt: extra.value.trim(), productImageData: imgData.value, productImageUrl: productImageUrl.value, productImageMimeType: imageMimeType.value || file.value?.type || 'image/jpeg', lens: lens.value })

    // 优先使用后端返回的完整任务数据（如果后端立即返回）
    if (r.data?.task) {
      replaceTask(normalizeTask(r.data.task))
    } else if (r.data?.taskId) {
      replaceTask({ taskId: r.data.taskId, status: 'queued', imageUrls: [], images: [], createdAt: new Date().toISOString(), styleName: useStyle ? (selectedStyleAsset.value?.name || selectedTag.value?.name || '未命名风格') : '内置场景风格', productCategory: category.value, productColorFamily: selectedColor.value?.name || (hasCustomColor.value ? '自定义' : '') || '', aspectRatio: ratio.value, productImageUrl: preview.value, imageCount: count.value, errorMessage: '' })
    }
    notifySyncedToDesigns()
    poll()
  } catch (e) {
    err.value = e?.response?.data?.error || '提交任务失败，请稍后重试'
  }
}

async function fetchTasks() {
  try {
    const r = await request.get('/scene-generation/tasks')
    const nextTasks = (r.data?.items || []).map(normalizeTask)
    tasks.value = nextTasks
    hasLoadedOnce = true

    // 进度追踪：检查所有任务的状态
    nextTasks.forEach(t => {
      const taskId = t.taskId?.toString()
      if (!taskId) return

      if (t.status === 'queued' || t.status === 'running') {
        if (!progressTrackers.value[taskId]) {
          progressTrackers.value[taskId] = { current: 0 }
          const persisted = fakeProgress.getPersistedProgress(taskId)
          if (persisted) {
            progressTrackers.value[taskId].current = persisted.progress
          }
          if (t.status === 'running') {
            fakeProgress.startProgress(taskId, (p) => {
              const tracker = progressTrackers.value[taskId]
              if (tracker) tracker.current = p
            })
          }
        }
      } else if (progressTrackers.value[taskId]) {
        fakeProgress.completeProgress(taskId)
        delete progressTrackers.value[taskId]
      }
    })

    if (!hasPendingTasks(nextTasks)) stop()
  } catch (e) {
    if (!hasLoadedOnce) err.value = e?.response?.data?.error || '获取最近任务失败'
    tasks.value = []
    stop()
  }
}

function poll() { if (!hasPendingTasks()) return stop(); stop(); timer = window.setInterval(fetchTasks, 1500) }
function stop() { if (timer) { window.clearInterval(timer); timer = null } }

onMounted(async () => { window.addEventListener('keydown', handleKeydown); await Promise.all([loadTags(), loadSceneCategories(), loadColorFamilies(), fetchTasks()]); if (hasPendingTasks()) poll() })
onBeforeUnmount(() => { window.removeEventListener('keydown', handleKeydown); stop() })
</script>

<style scoped>
.sg {
  width: auto;
  max-width: none;
  margin: 0 8px;
  padding: 16px 0 22px;
  display: grid;
  grid-template-columns: minmax(320px, 368px) minmax(0, 1fr);
  gap: 18px;
  align-items: stretch;
  height: calc(100vh - 92px);
  overflow: hidden
}

.panel {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 30px;
  box-shadow: 0 18px 50px rgba(15, 23, 42, .05)
}

.left {
  display: flex;
  flex-direction: column;
  position: sticky;
  top: 18px;
  max-height: calc(100vh - 110px);
  overflow: hidden
}

.body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: auto
}

.sec {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border: 1px solid #edf0f3;
  border-radius: 24px;
  background: #fcfcfd
}

.head h3,
.rhd h2,
.mhd h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: .01em
}

.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f1f5f9
}

.asset-mini-btn,
.asset-inline-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  border: 1px solid #d7dde8;
  border-radius: 8px;
  background: #fff;
  color: #172033;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
  transition: border-color .2s, box-shadow .2s, transform .2s;
}

.asset-mini-btn {
  padding: 0 12px;
  flex-shrink: 0;
}

.asset-inline-btn {
  width: 100%;
  margin-top: 8px;
  padding: 0 14px;
}

.asset-mini-btn:hover,
.asset-inline-btn:hover {
  border-color: #111827;
  box-shadow: 0 8px 18px rgba(15, 23, 42, .08);
  transform: translateY(-1px);
}

.head span,
.mhd p,
.rfh span,
.meta span,
.st span,
.st small,
.rt span {
  font-size: 12px;
  color: #6b7280
}

.drop {
  position: relative;
  display: flex;
  min-height: 190px;
  border: 2px dashed #d1d5db;
  border-radius: 24px;
  background: #f8fafc;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: pointer
}

.drop input {
  position: absolute;
  inset: 0;
  opacity: 0
}

.drop.has {
  border-color: #111827;
  background: #eef2ff
}

.drop.dragging {
  border-color: #3b82f6;
  background: #eff6ff;
  border-style: solid
}

.pv {
  width: 100%;
  height: 190px;
  object-fit: contain
}

.emp {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #6b7280
}

.emp strong {
  font-size: 15px;
  color: #111827
}

.rm {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 999px;
  background: rgba(17, 24, 39, .78);
  color: #fff;
  z-index: 2
}

.ov {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px;
  background: linear-gradient(to top, rgba(17, 24, 39, .8), transparent);
  color: #fff;
  text-align: center
}

.card,
.style,
.tx,
.rt,
.sc {
  border: 1px solid #e5e7eb;
  border-radius: 20px;
  background: #fafafa
}

.card {
  padding: 14px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, .03)
}

.config-sec {
  gap: 14px
}

.config-board {
  display: flex;
  flex-direction: column;
  gap: 12px
}

.config-panel {
  padding: 16px 18px;
  border: 1px solid #e8ebf0;
  border-radius: 22px;
  background: linear-gradient(180deg, #ffffff, #fbfcfe);
  box-shadow: 0 10px 26px rgba(15, 23, 42, .04);
  min-width: 0;
}

.config-panel-wide {
  padding-bottom: 16px
}

.cfg-row-panel {
  padding: 16px 18px;
}

.cfg-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px;
}

.cfg-row-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cfg-row-label {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  color: #374151;
  letter-spacing: .01em;
}

.compact-panel {
  padding: 14px 16px 14px
}

.config-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px
}

.config-panel-head strong {
  display: block;
  font-size: 16px;
  line-height: 1.35;
  color: #111827
}

.config-stack {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 4px;
  min-width: 0;
}

.color-swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.color-swatch {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
  cursor: pointer;
  transition: all .2s ease;
  min-width: 0;
}

.color-swatch:hover {
  border-color: #9ca3af;
  box-shadow: 0 4px 12px rgba(15, 23, 42, .08)
}

.color-swatch.on {
  border-color: #111827;
  box-shadow: 0 0 0 1px #111827 inset;
  background: #fafafa
}

.swatch-img {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  overflow: hidden;
  background: #f3f4f6;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.swatch-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.swatch-initial {
  font-size: 14px;
  font-weight: 600;
  color: #6b7280
}

.swatch-name {
  font-size: 12px;
  font-weight: 500;
  color: #374151;
  text-align: left;
  line-height: 1.3
}

.config-dual {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px
}

.card-kicker {
  margin-bottom: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .1em;
  text-transform: uppercase;
  color: #9ca3af
}

.category-chips .chip {
  min-width: 0
}

.switch-row {
  width: 100%;
  min-height: 42px;
  margin-top: 8px;
  padding: 8px 10px 8px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #fff;
  color: #374151;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  cursor: pointer;
  transition: border-color .2s ease, box-shadow .2s ease, background .2s ease
}

.switch-row:hover {
  border-color: #9ca3af;
  box-shadow: 0 8px 18px rgba(15, 23, 42, .06)
}

.switch-row.on {
  border-color: #111827;
  background: #f8fafc
}

.switch-text {
  font-size: 13px;
  font-weight: 600
}

.switch-track {
  width: 42px;
  height: 24px;
  padding: 2px;
  border-radius: 999px;
  background: #d1d5db;
  transition: background .2s ease;
  flex-shrink: 0
}

.switch-thumb {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #fff;
  display: block;
  box-shadow: 0 1px 4px rgba(15, 23, 42, .22);
  transition: transform .2s ease
}

.switch-row.on .switch-track {
  background: #111827
}

.switch-row.on .switch-thumb {
  transform: translateX(18px)
}

.compact-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px
}

.compact-grid-2 .chip {
  min-height: 40px
}

.compact-grid-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr))
}

.compact-grid-4 {
  grid-template-columns: repeat(4, minmax(0, 1fr))
}

.compact-grid-4 .chip {
  min-height: 40px;
  padding-inline: 8px
}

.category-chips.compact-grid-2 .chip {
  padding-inline: 10px
}

.quality-chips,
.count-chips,
.ratio-chips {
  margin-top: 0
}

.ratio-panel {
  overflow: hidden
}

.ratio-visual-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px
}

.ratio-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 124px;
  padding: 12px 10px;
  border: 1px solid #dddfe4;
  border-radius: 18px;
  background: #fff;
  color: #374151;
  cursor: pointer;
  transition: all .2s ease
}

.ratio-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(15, 23, 42, .08)
}

.ratio-card.on {
  background: #171717;
  border-color: #171717;
  color: #fff;
  box-shadow: none
}

.ratio-card-frame {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 54px
}

.ratio-card-shape {
  display: block;
  border: 3px solid currentColor;
  border-radius: 7px;
  opacity: .76
}

.ratio-card.on .ratio-card-shape {
  opacity: 1
}

.ratio-card-label {
  font-size: 14px;
  font-weight: 500;
  letter-spacing: .01em
}

.ratio-shape-square {
  width: 34px;
  height: 34px
}

.ratio-shape-landscape {
  width: 46px;
  height: 34px
}

.ratio-shape-wide {
  width: 50px;
  height: 28px
}

.ratio-shape-cinema {
  width: 54px;
  height: 24px
}

.ratio-shape-portrait {
  width: 28px;
  height: 42px
}

.ratio-shape-tall {
  width: 24px;
  height: 46px
}

.ratio-shape-poster {
  width: 28px;
  height: 40px
}

.spec-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: linear-gradient(180deg, #ffffff, #fbfcff)
}

.spec-topline {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 4px
}

.spec-title {
  display: block;
  font-size: 15px;
  color: #111827;
  line-height: 1.35
}

.compact-bh {
  padding-bottom: 8px
}

.spec-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px
}

.spec-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0
}

.spec-field-label {
  font-size: 12px;
  font-weight: 700;
  color: #374151
}

.select-wrap {
  position: relative
}

.select-wrap::after {
  content: '▾';
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 12px;
  color: #6b7280;
  pointer-events: none
}

.spec-select {
  width: 100%;
  height: 40px;
  padding: 0 36px 0 12px;
  border: 1px solid #d1d5db;
  border-radius: 14px;
  background: #fff;
  color: #111827;
  font-size: 13px;
  font-weight: 600;
  outline: none;
  appearance: none;
  cursor: pointer;
  transition: border-color .2s ease, box-shadow .2s ease
}

.spec-select:focus {
  border-color: #111827;
  box-shadow: 0 0 0 3px rgba(17, 24, 39, .08)
}

.slim-row {
  padding: 10px 12px;
  gap: 8px
}

.compact-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px
}

.compact-label .tt {
  margin: 0
}

.compact-label span {
  font-size: 11px;
  color: #9ca3af
}

.bh span {
  font-size: 12px;
  line-height: 1.5;
  color: #6b7280;
  max-width: 180px;
  text-align: right
}

.ratio-chips .chip,
.count-chips .chip,
.quality-chips .chip {
  min-width: 0
}

.tt {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  color: #374151
}

.mt {
  margin-top: 14px
}

.mt-sm {
  margin-top: 10px
}

.custom-color-section {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed #e5e7eb;
}

.custom-img-upload {
  position: relative;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
}

.hidden-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  overflow: hidden;
}

.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid #d1d5db;
  border-radius: 10px;
  background: #fff;
  color: #374151;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s ease;
  white-space: nowrap;
}

.upload-btn:hover {
  border-color: #9ca3af;
  box-shadow: 0 4px 12px rgba(15, 23, 42, .08);
  transform: translateY(-1px);
}

.custom-img-upload.dragging .upload-btn {
  border-color: #3b82f6;
  background: #eff6ff;
  transform: translateY(-1px);
}

.clear-custom-btn {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 999px;
  background: rgba(17, 24, 39, .7);
  color: #fff;
  font-size: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
  line-height: 1;
}

.custom-color-preview {
  margin-top: 8px;
}

.custom-color-preview img {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  object-fit: cover;
  border: 1px solid #e5e7eb;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px
}

.chips.compact {
  gap: 6px
}

.chip,
.gen,
.ghost {
  border: none;
  border-radius: 999px;
  padding: 7px 12px;
  transition: all .2s ease
}

.chip {
  background: #fff;
  border: 1px solid #d1d5db;
  color: #374151;
  cursor: pointer;
  min-height: 40px;
  font-size: 13px;
  font-weight: 600
}

.gen {
  width: 100%;
  justify-content: center;
  display: inline-flex;
  align-items: center;
  min-height: 48px;
  cursor: pointer
}

.chip:hover,
.ghost:hover,
.gen:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(15, 23, 42, .08)
}

.chip.on,
.gen {
  background: #111827;
  color: #fff
}

.ipt,
.tx textarea {
  width: 100%;
  border: 1px solid #d1d5db;
  border-radius: 14px;
  padding: 12px;
  background: #fff
}

.style {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #fff;
  cursor: pointer;
  transition: border-color .2s ease, box-shadow .2s ease, transform .2s ease;
  min-height: 72px
}

.style:hover,
.tx:focus-within,
.drop:hover {
  border-color: #9ca3af;
  box-shadow: 0 12px 26px rgba(15, 23, 42, .06)
}

.style:hover {
  transform: translateY(-1px)
}

.style img,
.meta img {
  width: 54px;
  height: 54px;
  border-radius: 14px;
  object-fit: cover
}

.asset-style-card {
  flex: 1;
  min-width: 0;
  width: 100%;
  border-radius: 16px;
  text-align: left
}

.style-reference-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0
}

.asset-style-placeholder {
  display: grid;
  place-items: center;
  flex: 0 0 54px;
  width: 54px;
  height: 54px;
  border: 1px dashed #cbd5e1;
  border-radius: 12px;
  background: #f8fafc;
  color: #64748b
}

.asset-style-placeholder svg {
  width: 22px;
  height: 22px
}

.asset-style-chevron {
  flex: 0 0 18px;
  width: 18px;
  height: 18px;
  color: #98a2b3
}

.style-reference-clear {
  display: grid;
  place-items: center;
  flex: 0 0 40px;
  width: 40px;
  height: 40px;
  border: 1px solid #d7dde8;
  border-radius: 10px;
  background: #fff;
  color: #667085;
  cursor: pointer;
  transition: border-color .2s, color .2s, background .2s
}

.style-reference-clear:hover {
  border-color: #98a2b3;
  background: #f8fafc;
  color: #111827
}

.style-reference-clear svg {
  width: 17px;
  height: 17px
}

.asset-style-card .asset-style-copy {
  display: grid;
  gap: 4px;
  min-width: 0
}

.asset-style-card strong,
.asset-style-card small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap
}

.asset-style-card strong {
  color: #111827;
  font-size: 14px;
  font-weight: 800
}

.asset-style-card small {
  color: #667085;
  font-size: 12px;
  font-weight: 700
}

.asset-inline-btn.subtle {
  color: #667085;
  background: #f8fafc
}

.tx {
  padding: 12px 14px;
  background: #fff;
  transition: border-color .2s ease, box-shadow .2s ease
}

.tx span {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700
}

.foot {
  padding: 18px 24px 20px;
  border-top: 1px solid #e5e7eb;
  background: linear-gradient(to top, #fff, rgba(255, 255, 255, .96));
  backdrop-filter: blur(8px);
  position: sticky;
  bottom: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12px
}

.err {
  margin: 0 0 10px;
  color: #dc2626;
  font-size: 13px
}

.ghost {
  background: #f3f4f6;
  color: #111827;
  cursor: pointer
}

.right {
  padding: 0;
  min-width: 0;
  max-height: calc(100vh - 110px);
  position: sticky;
  top: 18px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  isolation: isolate
}

.result-scroll {
  flex: 1;
  overflow: auto;
  overflow-x: hidden;
  padding-bottom: 8px
}

.rhd,
.bhd,
.mhd,
.rfh,
.mft {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px
}

.rhd {
  padding: 12px 20px 10px;
  margin: 0;
  border-bottom: 1px solid #eef2f7;
  background: #fff;
  z-index: 12;
  box-shadow: 0 10px 18px -18px rgba(15, 23, 42, .18)
}

.rhd h2 {
  margin-bottom: 2px
}

.result-tip {
  font-size: 11px;
  color: #9ca3af;
  display: block;
  line-height: 1.5
}

.history-link {
  width: auto;
  min-height: 36px;
  padding: 0 14px;
  flex-shrink: 0
}

.blocks {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 0;
  padding: 10px 14px 12px;
  overflow: auto;
  flex: 1;
  min-height: 0
}

.block {
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  padding: 10px;
  background: linear-gradient(180deg, #fff, #fcfcfd);
  overflow: hidden;
  transition: box-shadow .2s ease, border-color .2s ease
}

.block:hover {
  border-color: #d1d5db;
  box-shadow: 0 10px 24px rgba(15, 23, 42, .05)
}

.task-time {
  font-size: 11px;
  font-weight: 600
}

.task-time {
  color: #9ca3af;
  white-space: nowrap
}

.bhd {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid #eef2f7
}

.meta {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0
}

.meta strong,
.rfh h4,
.rt strong {
  display: block
}

.meta strong {
  font-size: 13px;
  line-height: 1.25;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden
}

.meta span {
  margin-top: 2px;
  display: block;
  line-height: 1.3;
  font-size: 11px;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden
}

.st {
  text-align: right;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 3px;
  min-width: 84px;
  flex-shrink: 0
}

.st-top {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px
}

.archive-inline {
  display: flex;
  justify-content: flex-end;
  margin-top: 2px
}

.archive-link {
  border: none;
  background: transparent;
  color: #111827;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
  transition: opacity .2s ease, color .2s ease
}

.archive-link:hover:not(:disabled) {
  color: #0f172a;
  text-decoration: underline
}

.archive-link:disabled {
  opacity: .45;
  cursor: not-allowed;
  text-decoration: none
}

.archive-state {
  display: inline-flex;
  align-items: center;
  font-size: 11px;
  font-weight: 600;
  color: #059669
}

.icon-btn {
  border: none;
  border-radius: 999px;
  padding: 5px 9px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s ease;
  background: #f3f4f6;
  color: #374151
}

.icon-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 10px 20px rgba(15, 23, 42, .08)
}

.icon-btn:disabled {
  opacity: .6;
  cursor: not-allowed
}

.icon-btn.danger {
  background: #fff1f2;
  color: #be123c
}

.rs,
.grid,
.rg,
.mbd {
  display: grid;
  gap: 6px
}

.rs {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: 8px;
  align-items: start
}

.compact-rs {
  grid-auto-rows: minmax(0, 1fr)
}

.grid {
  grid-template-columns: 1fr 1fr
}

.it {
  position: relative;
  min-width: 0
}

.tile {
  position: relative;
  aspect-ratio: 1/1;
  border-radius: 14px;
  overflow: hidden;
  background: #f3f4f6;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 0 0 1px rgba(229, 231, 235, .8)
}

.batch-selected { box-shadow: 0 0 0 2px #111827, 0 10px 24px rgba(15,23,42,.12); }
.batch-pickable { cursor: pointer; }
.batch-inline-bar { position: sticky; bottom: 10px; z-index: 20; display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 12px 14px 4px; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 14px; background: rgba(255,255,255,.96); box-shadow: 0 10px 28px rgba(15,23,42,.12); font-size: 12px; font-weight: 700; color: #475569; }
.batch-inline-bar > div { display: flex; gap: 8px; align-items: center; }
.compact-tile {
  aspect-ratio: 16/9
}

.tile img {
  width: 100%;
  height: 100%;
  object-fit: cover
}

.tile-actions {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(17, 24, 39, 0);
  transition: background .2s ease, opacity .2s ease;
  opacity: 0;
  pointer-events: none
}

.compact-actions {
  gap: 4px;
  padding: 8px;
  flex-wrap: wrap
}

.tile:hover .tile-actions {
  background: rgba(17, 24, 39, .38);
  opacity: 1;
  pointer-events: auto
}

.tile-actions.is-disabled,
.tile:hover .tile-actions.is-disabled {
  background: transparent;
  opacity: 0;
  pointer-events: none
}

.tile-action {
  border: none;
  border-radius: 999px;
  padding: 6px 9px;
  background: rgba(255, 255, 255, .94);
  color: #111827;
  cursor: pointer;
  transition: transform .2s ease, background .2s ease, box-shadow .2s ease;
  font-size: 11px;
  font-weight: 600;
  line-height: 1
}

.tile-action:hover {
  transform: translateY(-1px);
  background: #fff;
  box-shadow: 0 12px 24px rgba(15, 23, 42, .18)
}

.tile-action.solid {
  background: rgba(255, 255, 255, .14);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, .22)
}

.tile-action.solid:hover {
  background: rgba(255, 255, 255, .2)
}

.ld {
  text-align: center;
  color: #6b7280;
  font-size: 11px
}

.ld.is-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 12px;
  text-align: center
}

.ld.is-loading p {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: #111827
}

.loading-orbit {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px
}

.loading-ring,
.loading-core {
  border-radius: 999px
}

.loading-ring {
  width: 42px;
  height: 42px;
  border: 3px solid rgba(17, 24, 39, .14);
  border-top-color: #111827;
  animation: task-spin .9s linear infinite
}

.loading-core {
  position: absolute;
  width: 14px;
  height: 14px;
  background: rgba(17, 24, 39, .12);
  animation: task-pulse 1.2s ease-in-out infinite
}

.loading-text {
  font-size: 11px;
  line-height: 1.4;
  color: #6b7280
}

.progress-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 0 8px
}

.progress-bar {
  width: 100%;
  height: 6px;
  background: #e5e7eb;
  border-radius: 999px;
  overflow: hidden
}

.progress-fill {
  height: 100%;
  background: #111827;
  border-radius: 999px;
  transition: width .2s ease
}

.progress-text {
  display: flex;
  justify-content: space-between;
  width: 100%;
  font-size: 10px;
  color: #9ca3af
}

.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #111827
}

.more-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 0 14px 12px;
  padding: 10px 12px;
  border: 1px dashed #d1d5db;
  border-radius: 14px;
  background: #fafafa;
  color: #6b7280;
  font-size: 12px
}

.dl {
  position: absolute;
  right: 10px;
  bottom: 10px;
  border: none;
  border-radius: 999px;
  padding: 6px 10px;
  background: rgba(17, 24, 39, .85);
  color: #fff;
  cursor: pointer;
  transition: transform .2s ease, box-shadow .2s ease;
  font-size: 12px
}

.dl:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 24px rgba(15, 23, 42, .18)
}

.empty {
  display: flex;
  min-height: 0;
  flex: 1;
  align-items: center;
  justify-content: center;
  color: #6b7280;
  border: 1px dashed #d1d5db;
  border-radius: 26px;
  background: linear-gradient(180deg, #fcfcfd, #f9fafb);
  margin: 10px 20px 16px;
  padding: 24px;
  overflow: auto
}

.empty-copy {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  max-width: 320px;
  text-align: center
}

.empty-copy strong {
  font-size: 15px;
  color: #111827
}

.empty-copy span {
  font-size: 13px;
  line-height: 1.6;
  color: #6b7280
}

.viewer-mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, .78);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  z-index: 70
}

.simple-viewer-mask {
  cursor: zoom-out
}

.simple-viewer-image {
  display: block;
  max-width: min(92vw, 1180px);
  max-height: 90vh;
  object-fit: contain;
  border-radius: 20px;
  box-shadow: 0 32px 80px rgba(15, 23, 42, .35);
  background: #111827
}

.viewer-hint {
  position: fixed;
  left: 50%;
  bottom: 28px;
  transform: translateX(-50%);
  padding: 8px 14px;
  border-radius: 999px;
  background: rgba(255, 255, 255, .12);
  border: 1px solid rgba(255, 255, 255, .18);
  color: rgba(255, 255, 255, .92);
  font-size: 12px;
  letter-spacing: .01em;
  backdrop-filter: blur(10px)
}

@keyframes task-spin {
  to {
    transform: rotate(360deg)
  }
}

@keyframes task-pulse {

  0%,
  100% {
    transform: scale(.72);
    opacity: .55
  }

  50% {
    transform: scale(1);
    opacity: 1
  }
}

.tile-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 14px;
  background: rgba(15, 23, 42, .62);
  color: #fff;
  text-align: center;
  z-index: 2
}

.tile-overlay p {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.5
}

.pending-overlay {
  backdrop-filter: blur(1px)
}

.ready-overlay {
  background: linear-gradient(180deg, rgba(15, 23, 42, .28), rgba(15, 23, 42, .78))
}

.failed-overlay {
  background: rgba(127, 29, 29, .78)
}

.candidate-actions-inline {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px
}

.tile-actions.is-disabled {
  opacity: 0;
  pointer-events: none
}

.candidate-modal {
  width: min(920px, 100%);
  display: flex;
  flex-direction: column;
  gap: 18px
}

.candidate-mhd {
  padding-bottom: 12px;
  border-bottom: 1px solid #eef2f7
}

.candidate-mhd p:last-child {
  margin: 6px 0 0;
  font-size: 13px;
  color: #6b7280;
  line-height: 1.6
}

.candidate-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px
}

.candidate-pane {
  display: flex;
  flex-direction: column;
  gap: 10px
}

.candidate-label {
  font-size: 12px;
  font-weight: 700;
  color: #6b7280
}

.candidate-frame {
  position: relative;
  border: 1px solid #e5e7eb;
  border-radius: 22px;
  overflow: hidden;
  background: #f8fafc;
  aspect-ratio: 1/1;
  display: flex;
  align-items: center;
  justify-content: center
}

.candidate-frame img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #fff
}

.candidate-mft {
  margin-top: 0;
  padding-top: 4px;
  justify-content: flex-end
}

.candidate-confirm {
  width: auto;
  padding-inline: 18px;
  min-height: 42px
}

@media (max-width:640px) {
  .candidate-grid {
    grid-template-columns: 1fr
  }

  .candidate-frame {
    aspect-ratio: 4/3
  }

  .candidate-actions-inline {
    flex-direction: column;
    width: 100%
  }

  .candidate-actions-inline .tile-action {
    width: 100%;
    justify-content: center
  }
}

.mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, .46);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  z-index: 60
}

.modal {
  width: min(1120px, 100%);
  max-height: 86vh;
  overflow: auto;
  background: #fff;
  border-radius: 28px;
  padding: 22px
}

.style-modal {
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12);
  overflow: hidden;
  min-height: 460px;
}

.style-mhd {
  padding: 18px 22px 16px;
  border-bottom: 1px solid #f0f2f7;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-shrink: 0;
}

.mhd-info {
  flex: 1;
  min-width: 0;
}

.modal-kicker {
  margin: 0 0 4px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: .08em;
  text-transform: uppercase;
  color: #9ca3af;
}

.style-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: #111827;
  letter-spacing: -.3px;
}

.style-summary-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: #f3f4f6;
  border-radius: 999px;
  flex-shrink: 0;
}

.summary-tag {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  color: #6b7280;
  font-weight: 500;
}

.summary-tag svg {
  flex-shrink: 0;
  color: #9ca3af;
}

.summary-tag--selected {
  color: #111827;
  font-weight: 600;
}

.summary-tag--selected svg {
  color: #111827;
}

.summary-ref {
  font-size: 12px;
  color: #9ca3af;
}

.modal-close {
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 10px;
  background: #f3f4f6;
  color: #6b7280;
  cursor: pointer;
  transition: all .15s;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.modal-close:hover {
  background: #e5e7eb;
  color: #111827;
}

.style-lr {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.style-left {
  background: #fafbfc;
  border-right: 1px solid #edf0f5;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.left-section {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.left-header {
  padding: 14px 18px 12px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .07em;
  text-transform: uppercase;
  color: #9ca3af;
  border-bottom: 1px solid #edf0f5;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.left-count {
  background: #e5e7eb;
  color: #6b7280;
  font-size: 10px;
  font-weight: 700;
  padding: 1px 7px;
  border-radius: 999px;
  letter-spacing: 0;
  text-transform: none;
}

.tag-list {
  padding: 8px;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tag-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  gap: 10px;
  color: #9ca3af;
  font-size: 13px;
}

.loading-dots {
  display: flex;
  gap: 5px;
}

.loading-dots span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #d1d5db;
  animation: dotPulse 1.4s ease-in-out infinite;
}

.loading-dots span:nth-child(2) {
  animation-delay: .2s;
}

.loading-dots span:nth-child(3) {
  animation-delay: .4s;
}

@keyframes dotPulse {

  0%,
  80%,
  100% {
    opacity: .3;
    transform: scale(.8);
  }

  40% {
    opacity: 1;
    transform: scale(1);
  }
}

.tag-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border: 1.5px solid transparent;
  border-radius: 12px;
  background: transparent;
  color: #374151;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: all .15s;
  width: 100%;
}

.tag-item:hover {
  background: #fff;
  border-color: #e5e7eb;
  color: #111827;
}

.tag-item.on {
  background: #111827;
  color: #fff;
  border-color: #111827;
  box-shadow: 0 4px 12px rgba(17, 24, 39, .18);
}

.tag-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-count {
  font-size: 11px;
  font-weight: 600;
  opacity: .45;
  margin-left: 8px;
  flex-shrink: 0;
}

.tag-item.on .tag-count {
  opacity: .65;
}

.tag-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  gap: 8px;
  color: #9ca3af;
  font-size: 13px;
  text-align: center;
}

.tag-empty svg {
  color: #d1d5db;
  flex-shrink: 0;
}

.style-right {
  padding: 20px 22px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.ref-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.ref-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  flex-shrink: 0;
}

.ref-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .06em;
  text-transform: uppercase;
  color: #9ca3af;
}

.ref-count {
  font-size: 11px;
  font-weight: 600;
  color: #9ca3af;
  background: #f3f4f6;
  padding: 2px 8px;
  border-radius: 999px;
}

.ref-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: 60px;
}

.ref-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
  padding-right: 4px;
  align-content: start;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.ref-card {
  width: 180px;
  height: 180px;
  border: 1.5px solid #e5e7eb;
  border-radius: 16px;
  background: #fff;
  cursor: pointer;
  transition: all .2s;
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}

.ref-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 8px 24px rgba(15, 23, 42, .1);
  transform: translateY(-3px);
}

.ref-card.on {
  border-color: #111827;
  box-shadow: 0 0 0 2.5px #111827, 0 8px 24px rgba(17, 24, 39, .14);
}

.ref-card-body {
  position: relative;
  overflow: hidden;
  background: #f8f9fb;
  border-radius: 14px;
  aspect-ratio: 1;
}

.ref-card-body img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform .3s;
}

.ref-card:hover .ref-card-body img {
  transform: scale(1.04);
}

.ref-check {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #111827;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(17, 24, 39, .3);
}

.ref-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 50px 20px;
  gap: 10px;
  font-size: 14px;
  color: #9ca3af;
  border: 1.5px dashed #e5e7eb;
  border-radius: 18px;
  background: #fafbfc;
  text-align: center;
}

.ref-empty svg {
  color: #d1d5db;
  flex-shrink: 0;
}

.ref-empty--prompt {
  color: #c4c9d4;
}

.ref-empty--prompt svg {
  color: #e5e7eb;
}

.style-mft {
  padding: 14px 22px;
  border-top: 1px solid #f0f2f7;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-shrink: 0;
}

.style-mft-left {
  flex: 1;
  min-width: 0;
}

.sel-hint {
  font-size: 13px;
  color: #6b7280;
  font-weight: 500;
}

.sel-hint strong {
  font-weight: 600;
  color: #111827;
}

.sel-hint--muted {
  color: #9ca3af;
}

.style-mft-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.btn-ghost {
  padding: 9px 22px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all .15s;
  background: #fff;
  border: 1.5px solid #e5e7eb;
  color: #6b7280;
}

.btn-ghost:hover {
  background: #f9fafb;
  border-color: #d1d5db;
  color: #374151;
}

.btn-primary {
  padding: 9px 24px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all .15s;
  background: #111827;
  border: none;
  color: #fff;
  box-shadow: 0 4px 14px rgba(17, 24, 39, .25);
}

.btn-primary:hover {
  background: #1f2937;
  box-shadow: 0 6px 18px rgba(17, 24, 39, .3);
}

.btn-primary:active {
  transform: scale(.97);
}

.ref-name {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ref-card.on .ref-name {
  color: #111827;
  font-weight: 600;
}

.ref-empty {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  font-size: 14px;
  color: #9ca3af;
  background: #f8fafc;
  border: 1px dashed #e5e7eb;
  border-radius: 18px;
}

.style-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px
}

.summary-item {
  padding: 12px 14px;
  border: 1px solid #e8ebf0;
  border-radius: 18px;
  background: #fff
}

.summary-item span {
  display: block;
  font-size: 11px;
  color: #9ca3af;
  margin-bottom: 6px
}

.summary-item strong {
  display: block;
  font-size: 14px;
  line-height: 1.5;
  color: #111827
}

.style-layout {
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 16px;
  margin-top: 0
}

.style-rail {
  padding: 14px;
  border: 1px solid #edf0f3;
  border-radius: 22px;
  background: #f8fafc
}

.rail-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 4px 10px
}

.rail-title strong {
  font-size: 13px;
  color: #111827
}

.scene-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid #edf0f3;
  border-radius: 18px;
  background: #f8fafc;
}

.scene-chip {
  padding: 6px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  background: #fff;
  color: #374151;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s ease;
}

.scene-chip:hover {
  border-color: #9ca3af;
  box-shadow: 0 4px 12px rgba(15, 23, 42, .08);
}

.scene-chip.on {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.style-empty-hint {
  padding: 18px 12px;
  text-align: center;
  font-size: 13px;
  color: #9ca3af;
}

.rail-title span {
  font-size: 11px;
  color: #9ca3af
}

.style-tag {
  padding: 12px 14px;
  border-radius: 18px;
  background: #fff
}

.style-main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  border: 1px solid #edf0f3;
  border-radius: 22px;
  background: #fff;
  min-width: 0
}

.style-rfh {
  align-items: flex-start;
  padding-bottom: 10px;
  border-bottom: 1px solid #f1f5f9
}

.style-rfh h4 {
  margin: 0 0 4px;
  font-size: 18px
}

.picked-ref {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid #e8ebf0;
  border-radius: 18px;
  background: #f8fafc
}

.picked-ref img {
  width: 72px;
  height: 72px;
  border-radius: 14px;
  object-fit: cover
}

.picked-ref span {
  display: block;
  font-size: 11px;
  color: #9ca3af;
  margin-bottom: 4px
}

.picked-ref strong {
  display: block;
  font-size: 14px;
  color: #111827;
  line-height: 1.5
}

.subtle-ghost {
  padding: 9px 14px;
  font-size: 12px
}

.style-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 0
}

.style-card {
  padding: 10px;
  border-radius: 18px;
  background: #fff
}

.style-card img {
  width: 100%;
  aspect-ratio: 1/1;
  object-fit: cover;
  border-radius: 14px;
  margin-bottom: 10px
}

.style-card-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  text-align: left
}

.style-card-meta strong {
  font-size: 13px;
  line-height: 1.45;
  color: #111827
}

.style-card-meta span {
  font-size: 11px;
  color: #9ca3af
}

.style-empty {
  padding: 18px;
  border: 1px dashed #d1d5db;
  border-radius: 18px;
  background: #fafafa;
  color: #6b7280;
  font-size: 13px;
  text-align: center
}

.confirm-modal {
  width: min(520px, 100%);
  padding: 20px 20px 18px
}

.confirm-mhd {
  padding-bottom: 12px;
  border-bottom: 1px solid #eef2f7
}

.confirm-body {
  padding: 6px 0 2px
}

.confirm-card {
  padding: 14px 16px;
  border: 1px solid #eceff3;
  border-radius: 18px;
  background: #f8fafc
}

.confirm-card span {
  display: block;
  font-size: 11px;
  color: #9ca3af;
  margin-bottom: 6px
}

.confirm-card strong {
  display: block;
  font-size: 15px;
  line-height: 1.45;
  color: #111827
}

.confirm-card small {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: #6b7280;
  line-height: 1.5
}

.danger-btn {
  border: none;
  border-radius: 999px;
  padding: 10px 16px;
  background: #be123c;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s ease
}

.danger-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 12px 24px rgba(190, 18, 60, .18)
}

.danger-btn:disabled {
  opacity: .6;
  cursor: not-allowed
}

.confirm-mft {
  margin-top: 0;
  padding-top: 4px;
  justify-content: flex-end
}

.style-mft {
  margin-top: 0;
  padding-top: 4px;
  justify-content: flex-end
}

.mbd {
  grid-template-columns: 260px 1fr;
  margin-top: 18px
}

.rail {
  display: flex;
  flex-direction: column;
  gap: 10px
}

.rt {
  text-align: left;
  cursor: pointer;
  transition: border-color .2s ease, box-shadow .2s ease, transform .2s ease
}

.rt:hover,
.sc:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 24px rgba(15, 23, 42, .08)
}

.rt.on,
.sc.on {
  border-color: #111827;
  box-shadow: 0 0 0 1px #111827 inset
}

.rg {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 14px
}

.sc {
  padding: 10px;
  text-align: left;
  cursor: pointer;
  transition: border-color .2s ease, box-shadow .2s ease, transform .2s ease
}

.sc img {
  width: 100%;
  aspect-ratio: 1/1;
  object-fit: cover;
  border-radius: 14px;
  margin-bottom: 8px
}

.mft {
  margin-top: 18px;
  justify-content: flex-end
}

@media (max-width:1280px) {
  .rs {
    grid-template-columns: repeat(2, minmax(0, 1fr))
  }

  .style-summary,
  .style-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr))
  }

  .config-dual {
    grid-template-columns: 1fr
  }

  .ratio-visual-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr))
  }

  .compact-grid-4 {
    grid-template-columns: repeat(4, minmax(0, 1fr))
  }

  .rhd {
    padding: 12px 16px 10px
  }

  .blocks {
    padding: 10px 12px 12px
  }

  .more-bar {
    margin: 0 12px 12px
  }

  .empty {
    margin: 10px 16px 16px
  }
}

@media (max-width:1100px) {
  .sg {
    margin: 0 12px;
    grid-template-columns: 1fr;
    padding: 16px 0 24px;
    height: auto;
    overflow: visible
  }

  .left,
  .right {
    position: static;
    max-height: none;
    overflow: visible
  }

  .body,
  .result-scroll,
  .blocks,
  .empty {
    overflow: visible
  }

  .mbd,
  .grid,
  .rg,
  .spec-form-grid,
  .style-layout {
    grid-template-columns: 1fr
  }

  .style-summary {
    grid-template-columns: repeat(3, minmax(0, 1fr))
  }

  .rs {
    grid-template-columns: repeat(2, minmax(0, 1fr))
  }

  .config-dual,
  .config-stack {
    grid-template-columns: 1fr
  }

  .ratio-visual-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr))
  }

  .compact-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr))
  }

  .compact-grid-4 {
    grid-template-columns: repeat(4, minmax(0, 1fr))
  }

  .rhd {
    box-shadow: none;
    padding: 14px 20px 12px;
    align-items: flex-start
  }

  .result-tip {
    font-size: 12px
  }

  .history-link {
    min-height: 38px
  }

  .blocks {
    padding: 10px 20px 16px
  }

  .block {
    padding: 12px
  }

  .task-topline {
    margin-bottom: 10px
  }

  .bhd {
    gap: 10px;
    padding-bottom: 8px
  }

  .meta strong {
    font-size: 14px;
    line-height: 1.35;
    display: block;
    -webkit-line-clamp: unset;
    overflow: visible
  }

  .meta span {
    font-size: 12px;
    line-height: 1.4;
    display: block;
    -webkit-line-clamp: unset;
    overflow: visible
  }

  .st {
    gap: 4px;
    min-width: 104px
  }

  .icon-btn {
    padding: 6px 10px;
    font-size: 12px
  }

  .rs,
  .grid,
  .rg,
  .mbd {
    gap: 8px
  }

  .rs {
    margin-top: 10px
  }

  .batch-selected { box-shadow: 0 0 0 2px #111827, 0 10px 24px rgba(15,23,42,.12); }
.compact-tile {
    aspect-ratio: 4/3
  }

  .compact-actions {
    gap: 6px;
    padding: 10px
  }

  .tile-action {
    padding: 7px 11px;
    font-size: 12px;
    line-height: normal
  }

  .ld {
    font-size: 12px
  }

  .dot {
    width: 10px;
    height: 10px
  }

  .more-bar {
    margin: 0 20px 16px
  }

  .empty {
    margin: 12px 20px 20px
  }

  .empty-copy span {
    font-size: 14px
  }
}

@media (max-width:640px) {
  .sg {
    margin: 0 8px;
    padding: 12px 0 20px
  }

  .body,
  .foot,
  .modal {
    padding: 16px
  }

  .right {
    padding: 0
  }

  .rhd,
  .bhd,
  .bh,
  .viewer-bar,
  .compact-label,
  .style-mhd,
  .style-rfh,
  .more-bar {
    flex-direction: column;
    align-items: flex-start
  }

  .history-link {
    width: 100%;
    justify-content: center
  }

  .result-tip {
    display: none
  }

  .task-topline {
    width: 100%;
    align-items: flex-start
  }

  .task-time {
    white-space: normal
  }

  .st,
  .compact-label span {
    align-items: flex-start;
    text-align: left;
    min-width: 0
  }

  .st-top {
    width: 100%;
    justify-content: space-between
  }

  .archive-inline {
    justify-content: flex-start
  }

  .viewer-actions {
    width: 100%
  }

  .viewer-actions .tile-action {
    flex: 1;
    justify-content: center
  }

  .style-summary,
  .style-grid {
    grid-template-columns: 1fr
  }

  .ratio-visual-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px
  }

  .ratio-card {
    min-height: 112px;
    border-radius: 16px;
    padding: 10px 8px
  }

  .compact-grid,
  .compact-grid-4 {
    grid-template-columns: repeat(2, minmax(0, 1fr))
  }

  .picked-ref {
    align-items: flex-start
  }

  .asset-style-card {
    padding: 12px
  }

  .asset-style-card small {
    white-space: normal;
    line-height: 1.45
  }

  .rs {
    grid-template-columns: 1fr
  }

  .config-panel {
    padding: 14px 14px 16px
  }

  .blocks {
    padding: 10px 16px 12px
  }

  .more-bar {
    margin: 0 16px 12px
  }

  .empty {
    margin: 12px 16px 16px
  }

  .archive-link {
    padding: 4px 0
  }
}
</style>
