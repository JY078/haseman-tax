<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-all duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      >
        <Transition
          enter-active-class="transition-all duration-200 ease-out"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition-all duration-150 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="visible"
            class="bg-white rounded-2xl shadow-2xl w-full flex flex-col overflow-hidden"
            style="max-width: min(1440px, 96vw); height: 94vh; max-height: 94vh;"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 py-4 border-b border-neutral-100 flex-shrink-0">
              <div>
                <h3 class="font-semibold text-neutral-900">{{ mode === 'ref' ? '编辑参考图' : '编辑图片' }}</h3>
                <p class="text-xs text-neutral-400 mt-0.5">
    {{ mode === 'ref' ? '在图片上涂抹标记需要修改的区域' : '用画笔标记修改区域，输入新的描述词后重新生成' }}
  </p>
              </div>
              <button
                class="w-8 h-8 flex items-center justify-center rounded-full bg-neutral-100 hover:bg-neutral-200 text-neutral-500 hover:text-neutral-800 transition-all duration-200 hover:scale-110"
                @click="$emit('close')"
              >
                <XMarkIcon class="w-5 h-5" />
              </button>
            </div>

            <!-- Body -->
            <div class="flex flex-1 min-h-0 overflow-hidden">
              <!-- Canvas area -->
              <div class="flex-1 flex flex-col bg-neutral-100 overflow-hidden">
                <div class="flex items-center justify-between gap-3 px-4 py-3 border-b border-neutral-200 bg-white/80 flex-shrink-0">
                  <div class="flex items-center gap-2 text-xs text-neutral-500">
                    <span class="rounded-full bg-neutral-100 px-2.5 py-1 font-semibold text-neutral-700">{{ Math.round(zoom * 100) }}%</span>
                    <span>滚轮缩放图片</span>
                  </div>
                  <div class="text-xs text-neutral-500">
                    {{ canvasW }} × {{ canvasH }}
                  </div>
                </div>

                <div ref="viewportRef" class="flex-1 overflow-auto p-4" @wheel.prevent="onViewportWheel">
                  <div v-if="isLoadingImage" class="h-full flex flex-col items-center justify-center gap-3 text-neutral-400">
                    <svg class="w-8 h-8 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    <span class="text-sm">加载图片中...</span>
                  </div>
                  <div
                    v-else
                    class="relative mx-auto rounded-xl overflow-hidden shadow-lg select-none bg-white"
                    :style="{ width: stageW + 'px', height: stageH + 'px' }"
                    ref="containerRef"
                  >
                    <!-- Bottom layer: original image -->
                    <canvas
                      ref="imageCanvasRef"
                      class="absolute left-0 top-0"
                      :width="canvasW"
                      :height="canvasH"
                      :style="{ width: stageW + 'px', height: stageH + 'px' }"
                    />

                    <!-- Movable overlay images -->
                    <div
                      v-for="item in overlayImages"
                      :key="item.id"
                      class="absolute overlay-image-layer"
                      :class="{ selected: selectedOverlayId === item.id }"
                      :style="overlayStyle(item)"
                      @mousedown.stop.prevent="startOverlayMove($event, item)"
                    >
                      <img
                        :src="item.src"
                        alt="叠加图片"
                        draggable="false"
                        class="w-full h-full object-fill pointer-events-none"
                        :style="{ transform: item.flipX ? 'scaleX(-1)' : 'none' }"
                      />
                      <button
                        v-if="selectedOverlayId === item.id"
                        class="overlay-delete"
                        type="button"
                        @mousedown.stop
                        @click.stop="removeOverlay(item.id)"
                      >×</button>
                      <button
                        v-if="selectedOverlayId === item.id"
                        class="overlay-flip"
                        type="button"
                        @mousedown.stop
                        @click.stop="toggleOverlayFlip(item)"
                      >镜</button>
                      <span
                        v-if="selectedOverlayId === item.id"
                        class="overlay-rotate"
                        @mousedown.stop.prevent="startOverlayRotate($event, item)"
                      />
                      <span
                        v-if="selectedOverlayId === item.id"
                        class="overlay-resize"
                        @mousedown.stop.prevent="startOverlayResize($event, item)"
                      />
                    </div>

                    <!-- Top layer: drawing overlay -->
                    <canvas
                      ref="drawCanvasRef"
                      class="absolute left-0 top-0"
                      :width="canvasW"
                      :height="canvasH"
                      :style="{
                        width: stageW + 'px',
                        height: stageH + 'px',
                        cursor: getCursor(activeTool),
                        pointerEvents: activeTool === 'image' ? 'none' : 'auto'
                      }"
                      @mousedown="startDraw"
                      @mousemove="onDraw"
                      @mouseup="stopDraw"
                      @mouseleave="stopDraw"
                      @touchstart.prevent="startDrawTouch"
                      @touchmove.prevent="onDrawTouch"
                      @touchend.prevent="stopDraw"
                    />
                    <!-- Draw hint overlay (shows when canvas is empty) -->
                    <div
                      v-if="isCanvasEmpty && overlayImages.length === 0 && !isDrawing"
                      class="absolute inset-0 flex items-center justify-center pointer-events-none"
                    >
                      <div class="flex flex-col items-center gap-2 bg-black/30 rounded-xl px-5 py-3">
                        <PencilSquareIcon class="w-6 h-6 text-white/80" />
                        <span class="text-xs text-white/80 font-medium">在图片上涂抹标记修改区域</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Right panel: tools + prompt -->
              <div class="w-72 flex-shrink-0 border-l border-neutral-100 flex flex-col overflow-y-auto">
                <!-- Tools -->
                <div class="p-4 border-b border-neutral-100">
                  <p class="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-3">编辑工具</p>

                  <!-- Tool toggle -->
                  <div class="flex gap-2 mb-4">
                    <button
                      v-for="tool in visibleTools"
                      :key="tool.id"
                      class="flex-1 flex flex-col items-center gap-1.5 py-2.5 rounded-lg border-2 transition-all text-xs font-medium"
                      :class="activeTool === tool.id
                        ? 'border-neutral-900 bg-neutral-900 text-white'
                        : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'"
                      @click="activeTool = tool.id"
                    >
                      <component :is="tool.icon" class="w-4 h-4" />
                      {{ tool.label }}
                    </button>
                  </div>

                  <div v-if="mode === 'ref'" class="mb-4">
                    <input ref="overlayInputRef" type="file" accept="image/*" class="hidden" @change="handleOverlayFileChange" />
                    <button
                      class="w-full py-2 text-xs font-medium rounded-lg border border-neutral-200 text-neutral-700 hover:bg-neutral-50 transition-colors flex items-center justify-center gap-1.5"
                      @click="overlayInputRef?.click()"
                    >
                      <PhotoIcon class="w-4 h-4" />
                      添加图片
                    </button>
                    <p class="mt-2 text-[11px] leading-5 text-neutral-400">添加后直接在图片上拖动；顶部手柄旋转，右下角手柄缩放，点“镜”可水平镜像。</p>
                  </div>

                  <!-- Brush size -->
                  <div v-if="activeTool !== 'image'" class="mb-4">
                    <div class="flex items-center justify-between mb-1.5">
                      <span class="text-xs text-neutral-500">笔刷大小</span>
                      <span class="text-xs font-semibold text-neutral-700">{{ brushSize }}px</span>
                    </div>
                    <input
                      type="range"
                      min="2"
                      :max="IMAGE_EDIT_BRUSH_DEFAULTS.maxSize"
                      v-model.number="brushSize"
                      class="w-full accent-neutral-900 h-1.5 rounded-full cursor-pointer"
                    />
                    <!-- Brush preview -->
                    <div class="flex justify-center mt-2">
                      <div
                        class="rounded-full bg-neutral-800 transition-all"
                        :style="{
                          width: Math.min(brushSize, 56) + 'px',
                          height: Math.min(brushSize, 56) + 'px',
                          opacity: currentBrushOpacity(),
                          backgroundColor: activeTool === 'eraser' ? '#e5e7eb' : currentBrushColor(),
                          border: activeTool === 'eraser' ? '2px dashed #9ca3af' : 'none'
                        }"
                      />
                    </div>
                  </div>

                  <!-- Color swatches (brush only) -->
                  <div v-if="activeTool === 'brush'" class="mb-4">
                    <p class="text-xs text-neutral-500 mb-2">颜色</p>
                    <div class="flex flex-wrap gap-1.5 items-center">
                      <button
                        v-for="color in presetColors"
                        :key="color"
                        class="w-6 h-6 rounded-full border-2 transition-all hover:scale-110"
                        :class="brushColor === color ? 'border-neutral-900 scale-110' : 'border-transparent'"
                        :style="{ backgroundColor: color }"
                        @click="brushColor = color"
                      />
                      <!-- Custom color picker -->
                      <label class="w-6 h-6 rounded-full border-2 border-dashed border-neutral-300 flex items-center justify-center cursor-pointer relative overflow-hidden hover:border-neutral-500 transition-colors">
                        <PlusIcon class="w-3 h-3 text-neutral-400 pointer-events-none" />
                        <input
                          type="color"
                          v-model="brushColor"
                          class="absolute inset-0 opacity-0 cursor-pointer w-full h-full"
                        />
                      </label>
                    </div>
                    <p v-if="mode === 'ref'" class="mt-2 text-[11px] leading-5 text-neutral-400">使用当前颜色的半透明标记层，不遮挡原图细节。</p>
                  </div>

                  <!-- Opacity -->
                  <div v-if="activeTool !== 'image' && !(mode === 'ref' && activeTool === 'brush')" class="mb-4">
                    <div class="flex items-center justify-between mb-1.5">
                      <span class="text-xs text-neutral-500">{{ activeTool === 'brush' ? '不透明度' : '橡皮力度' }}</span>
                      <span class="text-xs font-semibold text-neutral-700">{{ Math.round(brushOpacity * 100) }}%</span>
                    </div>
                    <input
                      type="range"
                      min="0.1"
                      max="1"
                      step="0.05"
                      v-model.number="brushOpacity"
                      class="w-full accent-neutral-900 h-1.5 rounded-full cursor-pointer"
                    />
                  </div>

                  <!-- Undo / Clear -->
                  <div class="flex gap-2">
                    <button
                      class="flex-1 py-2 text-xs font-medium rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 transition-colors flex items-center justify-center gap-1.5"
                      :disabled="undoStack.length === 0"
                      :class="undoStack.length === 0 ? 'opacity-40 cursor-not-allowed' : ''"
                      @click="undo"
                    >
                      <ArrowUturnLeftIcon class="w-3.5 h-3.5" />
                      撤销
                    </button>
                    <button
                      class="flex-1 py-2 text-xs font-medium rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 transition-colors flex items-center justify-center gap-1.5"
                      @click="clearCanvas"
                    >
                      <TrashIcon class="w-3.5 h-3.5" />
                      清除
                    </button>
                  </div>
                </div>

                <!-- Prompt -->
                <div v-if="mode === 'result'" class="p-4 flex-1 flex flex-col">
                  <p class="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-2">重新描述</p>
                  <textarea
                    v-model="editPrompt"
                    :placeholder="originalPrompt ? `原提示词：${originalPrompt.slice(0, 40)}${originalPrompt.length > 40 ? '...' : ''}` : '描述你希望如何修改这张图片...'"
                    rows="4"
                    class="w-full text-sm border border-neutral-200 rounded-lg px-3 py-2.5 resize-none bg-neutral-50 focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent transition-all placeholder:text-neutral-400 leading-relaxed"
                  />
                  <p class="text-[11px] text-neutral-400 mt-1.5">留空则沿用原有提示词</p>
                </div>

                <!-- Quality -->
                <div v-if="mode === 'result'" class="p-4 border-t border-b border-neutral-100">
                  <p class="text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-2">输出质量</p>
                  <div class="flex gap-2">
                    <button
                      v-for="q in qualityOptions"
                      :key="q.value"
                      class="flex-1 py-2 text-xs font-medium rounded-lg border-2 transition-all"
                      :class="quality === q.value
                        ? 'border-neutral-900 bg-neutral-900 text-white'
                        : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'"
                      @click="quality = q.value"
                    >
                      {{ q.label }}
                    </button>
                  </div>
                </div>

                <!-- Submit -->
                <div class="p-4 border-t border-neutral-100 flex-shrink-0">
                  <button
                    class="w-full py-3 text-sm font-semibold rounded-xl text-white transition-all flex items-center justify-center gap-2"
                    :class="isSubmitting
                      ? 'bg-neutral-400 cursor-not-allowed'
                      : 'bg-neutral-900 hover:bg-neutral-800 active:scale-[0.98] shadow-sm'"
                    :disabled="isSubmitting"
                    @click="submit"
                  >
                    <template v-if="isSubmitting">
                      <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      生成中...
                    </template>
                    <template v-else>
                      <SparklesIcon class="w-4 h-4" />
                      {{ mode === 'ref' ? '保存蒙版' : '重新生成' }}
                    </template>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import {
  XMarkIcon,
  SparklesIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
  ArrowUturnLeftIcon,
  PhotoIcon,
} from '@heroicons/vue/24/outline'
import { PaintBrushIcon, ScissorsIcon } from '@heroicons/vue/24/solid'
import { Squares2X2Icon, AdjustmentsHorizontalIcon } from '@heroicons/vue/24/outline'
import { IMAGE_EDIT_BRUSH_DEFAULTS, drawSmoothBrushStroke } from '@/utils/imageEditBrush'

const props = defineProps({
  visible: { type: Boolean, default: false },
  imageUrl: { type: String, default: '' },
  originalPrompt: { type: String, default: '' },
  mode: { type: String, default: 'result' },  // 'ref' | 'result'
})

const emit = defineEmits(['close', 'submit'])

const isLoadingImage = ref(true)
const workingImageUrl = ref('')
const canvasW = ref(400)
const canvasH = ref(400)
const baseZoom = ref(1)
const zoom = ref(1)

const imageCanvasRef = ref(null)
const drawCanvasRef = ref(null)
const containerRef = ref(null)
const viewportRef = ref(null)
const overlayInputRef = ref(null)

const activeTool = ref('brush')
const brushSize = ref(IMAGE_EDIT_BRUSH_DEFAULTS.size)
const brushColor = ref(IMAGE_EDIT_BRUSH_DEFAULTS.color)
const brushOpacity = ref(IMAGE_EDIT_BRUSH_DEFAULTS.opacity)
const REF_MASK_BRUSH_OPACITY = 0.34
const editPrompt = ref('')
const quality = ref('2k')

const qualityOptions = [
  { label: '1K', value: '1k' },
  { label: '2K', value: '2k' },
  { label: '4K', value: '4k' },
]

const isDrawing = ref(false)
const isSubmitting = ref(false)
const isCanvasEmpty = ref(true)
const overlayImages = ref([])
const selectedOverlayId = ref('')
const stageW = computed(() => Math.max(1, Math.round(canvasW.value * zoom.value)))
const stageH = computed(() => Math.max(1, Math.round(canvasH.value * zoom.value)))

/** Undo stack: array of ImageData snapshots */
const undoStack = ref([])

const tools = [
  { id: 'brush', label: '画笔', icon: PaintBrushIcon },
  { id: 'eraser', label: '橡皮', icon: ScissorsIcon },
  { id: 'line', label: '直线', icon: AdjustmentsHorizontalIcon },
  { id: 'rect', label: '框选', icon: Squares2X2Icon },
  { id: 'image', label: '图片', icon: PhotoIcon },
]
const visibleTools = computed(() => props.mode === 'ref' ? tools : tools.filter(tool => tool.id !== 'image'))

/** 获取工具对应的光标 */
function getCursor(tool) {
  switch (tool) {
    case 'brush':
    case 'eraser':
      return 'crosshair'
    case 'line':
    case 'rect':
      return 'crosshair'
    default:
      return 'cell'
  }
}

const presetColors = [
  '#ff3b30', '#ff9500', '#ffcc00', '#34c759',
  '#007aff', '#5856d6', '#ff2d55', '#ffffff',
  '#000000', '#636366',
]

function currentBrushColor() {
  return brushColor.value
}

function currentBrushOpacity() {
  return props.mode === 'ref' && activeTool.value === 'brush' ? REF_MASK_BRUSH_OPACITY : brushOpacity.value
}

let lastPos = { x: 0, y: 0 }
let overlayAction = null
let shapeStartPos = { x: 0, y: 0 }  // 记录直线/矩形起点
let tempCanvas = null  // 临时画布用于预览
let strokePoints = []

/** 缓存已转换的 base64 图片，避免重复请求 */
const base64Cache = new Map()

async function ensureEditableImageUrl(url) {
  if (!url || url.startsWith('data:') || url.startsWith('/')) return url
  if (!url.startsWith('http://') && !url.startsWith('https://')) return url

  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  const token = localStorage.getItem('token')
  const res = await fetch(`${apiBase}/images/ensure-oss`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ imageUrl: url }),
  })
  if (!res.ok) throw new Error(`Ensure image failed: ${res.status}`)
  const data = await res.json()
  return data?.imageUrl || url
}

/** Convert imageUrl to base64 data URL (uses proxy for OSS to avoid CORS) */
async function imageUrlToBase64(url) {
  if (url.startsWith('data:')) return url

  // 缓存命中
  if (base64Cache.has(url)) {
    return base64Cache.get(url)
  }

  // OSS 图片走后端代理，避免跨域
  // 非 OSS 图片直接 fetch
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  const fetchUrl = url.startsWith('http://') || url.startsWith('https://')
    ? `${apiBase}/proxy?url=${encodeURIComponent(url)}`
    : url
  const res = await fetch(fetchUrl)
  if (!res.ok) throw new Error(`Image fetch failed: ${res.status}`)
  const blob = await res.blob()
  const result = await blobToBase64(blob)
  base64Cache.set(url, result)
  return result
}

function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(e.target.result)
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

async function setupCanvas() {
  if (!props.imageUrl) return
  isLoadingImage.value = true

  try {
    workingImageUrl.value = await ensureEditableImageUrl(props.imageUrl)
    const dataUrl = await imageUrlToBase64(workingImageUrl.value)
    const img = await new Promise((resolve, reject) => {
      const i = new Image()
      i.onload = () => resolve(i)
      i.onerror = reject
      i.src = dataUrl
    })

    canvasW.value = img.naturalWidth || img.width
    canvasH.value = img.naturalHeight || img.height
    overlayImages.value = []
    selectedOverlayId.value = ''

    isLoadingImage.value = false
    await nextTick()

    /** Draw base image */
    const imgCtx = imageCanvasRef.value.getContext('2d')
    imgCtx.clearRect(0, 0, canvasW.value, canvasH.value)
    imgCtx.drawImage(img, 0, 0, canvasW.value, canvasH.value)

    /** Reset drawing layer */
    const drawCtx = drawCanvasRef.value.getContext('2d')
    drawCtx.clearRect(0, 0, canvasW.value, canvasH.value)
    isCanvasEmpty.value = true
    undoStack.value = []
    await nextTick()
    fitToView()
  } catch (err) {
    console.error('ImageEditModal: 加载图片失败', err)
    isLoadingImage.value = false
  }
}

watch(() => props.visible, (val) => {
  if (val) {
    editPrompt.value = ''
    isSubmitting.value = false
    setupCanvas()
  }
})

watch(() => props.imageUrl, (val) => {
  if (props.visible && val) {
    setupCanvas()
  }
})

/** Get canvas-space coordinates from a mouse/touch event */
function getPos(e) {
  const canvas = drawCanvasRef.value
  const rect = canvas.getBoundingClientRect()
  return {
    x: (e.clientX - rect.left) / zoom.value,
    y: (e.clientY - rect.top) / zoom.value,
  }
}

function configureCtx(ctx) {
  ctx.lineWidth = brushSize.value
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  if (activeTool.value === 'eraser') {
    ctx.globalCompositeOperation = 'destination-out'
    ctx.strokeStyle = 'rgba(0,0,0,1)'
    ctx.fillStyle = 'rgba(0,0,0,1)'
    ctx.globalAlpha = brushOpacity.value
  } else {
    ctx.globalCompositeOperation = 'source-over'
    ctx.strokeStyle = currentBrushColor()
    ctx.fillStyle = currentBrushColor()
    ctx.globalAlpha = currentBrushOpacity()
  }
}

function saveUndoSnapshot() {
  const ctx = drawCanvasRef.value.getContext('2d')
  const snapshot = ctx.getImageData(0, 0, canvasW.value, canvasH.value)
  undoStack.value.push(snapshot)
  if (undoStack.value.length > 20) undoStack.value.shift()
}

function startDraw(e) {
  saveUndoSnapshot()
  isDrawing.value = true
  const pos = getPos(e)
  lastPos = pos
  shapeStartPos = pos
  strokePoints = [pos]

  // 直线和矩形需要保存当前画布状态用于预览
  if (activeTool.value === 'line' || activeTool.value === 'rect') {
    const ctx = drawCanvasRef.value.getContext('2d')
    // 保存当前画布状态
    tempCanvas = ctx.getImageData(0, 0, canvasW.value, canvasH.value)
  } else if (activeTool.value === 'brush' || activeTool.value === 'eraser') {
    const ctx = drawCanvasRef.value.getContext('2d')
    tempCanvas = ctx.getImageData(0, 0, canvasW.value, canvasH.value)
    configureCtx(ctx)
    drawSmoothBrushStroke(ctx, strokePoints, brushSize.value)
  }

  isCanvasEmpty.value = false
}

function onDraw(e) {
  if (!isDrawing.value) return

  // 直线和矩形：使用预览方式
  if (activeTool.value === 'line' || activeTool.value === 'rect') {
    const ctx = drawCanvasRef.value.getContext('2d')
    const pos = getPos(e)

    // 恢复之前保存的状态
    if (tempCanvas) {
      ctx.putImageData(tempCanvas, 0, 0)
    }

    // 绘制预览
    configureCtx(ctx)
    ctx.beginPath()

    if (activeTool.value === 'line') {
      // 绘制直线
      ctx.moveTo(shapeStartPos.x, shapeStartPos.y)
      ctx.lineTo(pos.x, pos.y)
      ctx.stroke()
    } else if (activeTool.value === 'rect') {
      // 绘制矩形
      const x = Math.min(shapeStartPos.x, pos.x)
      const y = Math.min(shapeStartPos.y, pos.y)
      const w = Math.abs(pos.x - shapeStartPos.x)
      const h = Math.abs(pos.y - shapeStartPos.y)
      ctx.strokeRect(x, y, w, h)
    }
    lastPos = pos
    return
  }

  // 画笔和橡皮：正常绘制
  const ctx = drawCanvasRef.value.getContext('2d')
  const pos = getPos(e)
  strokePoints.push(pos)
  if (tempCanvas) {
    ctx.putImageData(tempCanvas, 0, 0)
  }
  configureCtx(ctx)
  drawSmoothBrushStroke(ctx, strokePoints, brushSize.value)
  lastPos = pos
}

function stopDraw() {
  isDrawing.value = false
  // 直线和矩形完成后清理临时画布
  if (activeTool.value === 'line' || activeTool.value === 'rect' || activeTool.value === 'brush' || activeTool.value === 'eraser') {
    tempCanvas = null
    strokePoints = []
  }
}

function startDrawTouch(e) {
  const touch = e.touches[0]
  startDraw(touch)
}

function onDrawTouch(e) {
  const touch = e.touches[0]
  onDraw(touch)
}

function adjustZoom(delta, anchorEvent = null) {
  const viewport = viewportRef.value
  const oldZoom = zoom.value
  const nextZoom = Math.max(0.1, Math.min(4, Number((oldZoom + delta).toFixed(2))))
  if (nextZoom === oldZoom) return

  if (viewport && anchorEvent) {
    const rect = viewport.getBoundingClientRect()
    const anchorX = anchorEvent.clientX - rect.left + viewport.scrollLeft
    const anchorY = anchorEvent.clientY - rect.top + viewport.scrollTop
    const ratio = nextZoom / oldZoom
    zoom.value = nextZoom
    nextTick(() => {
      viewport.scrollLeft = anchorX * ratio - (anchorEvent.clientX - rect.left)
      viewport.scrollTop = anchorY * ratio - (anchorEvent.clientY - rect.top)
    })
    return
  }

  zoom.value = nextZoom
}

function onViewportWheel(e) {
  const delta = e.deltaY > 0 ? -0.12 : 0.12
  adjustZoom(delta, e)
}

function fitToView() {
  const viewport = viewportRef.value
  const availableW = Math.max(1, (viewport?.clientWidth || 960) - 32)
  const availableH = Math.max(1, (viewport?.clientHeight || 720) - 32)
  const nextZoom = Math.min(availableW / canvasW.value, availableH / canvasH.value)
  baseZoom.value = Number(Math.max(0.05, nextZoom).toFixed(3))
  zoom.value = baseZoom.value
}

function handleWindowResize() {
  if (!props.visible || isLoadingImage.value) return
  fitToView()
}

async function handleOverlayFileChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  const src = await blobToBase64(file)
  const img = await loadImageElement(src)
  const maxW = canvasW.value * 0.45
  const maxH = canvasH.value * 0.45
  const scale = Math.min(maxW / img.naturalWidth, maxH / img.naturalHeight, 1)
  const w = Math.max(80, Math.round(img.naturalWidth * scale))
  const h = Math.max(80, Math.round(img.naturalHeight * scale))
  const item = {
    id: `overlay-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    src,
    x: Math.round((canvasW.value - w) / 2),
    y: Math.round((canvasH.value - h) / 2),
    w,
    h,
    ratio: img.naturalHeight / img.naturalWidth,
    rotation: 0,
    flipX: false,
  }
  overlayImages.value.push(item)
  selectedOverlayId.value = item.id
  activeTool.value = 'image'
  isCanvasEmpty.value = false
}

function loadImageElement(src) {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}

function overlayStyle(item) {
  return {
    width: `${item.w * zoom.value}px`,
    height: `${item.h * zoom.value}px`,
    transform: `translate(${item.x * zoom.value}px, ${item.y * zoom.value}px) rotate(${item.rotation}deg)`,
    transformOrigin: 'center center',
    cursor: activeTool.value === 'image' ? 'move' : 'default',
    pointerEvents: activeTool.value === 'image' ? 'auto' : 'none',
  }
}

function startOverlayMove(e, item) {
  selectedOverlayId.value = item.id
  activeTool.value = 'image'
  overlayAction = {
    type: 'move',
    id: item.id,
    startX: e.clientX,
    startY: e.clientY,
    x: item.x,
    y: item.y,
  }
  window.addEventListener('mousemove', onOverlayPointerMove)
  window.addEventListener('mouseup', stopOverlayAction)
}

function startOverlayResize(e, item) {
  selectedOverlayId.value = item.id
  activeTool.value = 'image'
  overlayAction = {
    type: 'resize',
    id: item.id,
    startX: e.clientX,
    startY: e.clientY,
    w: item.w,
    h: item.h,
  }
  window.addEventListener('mousemove', onOverlayPointerMove)
  window.addEventListener('mouseup', stopOverlayAction)
}

function startOverlayRotate(e, item) {
  selectedOverlayId.value = item.id
  activeTool.value = 'image'
  const rect = containerRef.value.getBoundingClientRect()
  const centerX = rect.left + (item.x + item.w / 2) * zoom.value
  const centerY = rect.top + (item.y + item.h / 2) * zoom.value
  overlayAction = {
    type: 'rotate',
    id: item.id,
    centerX,
    centerY,
    rotation: item.rotation,
    startAngle: Math.atan2(e.clientY - centerY, e.clientX - centerX),
  }
  window.addEventListener('mousemove', onOverlayPointerMove)
  window.addEventListener('mouseup', stopOverlayAction)
}

function onOverlayPointerMove(e) {
  if (!overlayAction) return
  const item = overlayImages.value.find(entry => entry.id === overlayAction.id)
  if (!item) return
  const dx = (e.clientX - overlayAction.startX) / zoom.value
  const dy = (e.clientY - overlayAction.startY) / zoom.value
  if (overlayAction.type === 'move') {
    item.x = overlayAction.x + dx
    item.y = overlayAction.y + dy
  } else if (overlayAction.type === 'resize') {
    const nextW = Math.max(40, overlayAction.w + dx)
    item.w = nextW
    item.h = Math.max(40, nextW * item.ratio)
  } else if (overlayAction.type === 'rotate') {
    const angle = Math.atan2(e.clientY - overlayAction.centerY, e.clientX - overlayAction.centerX)
    item.rotation = overlayAction.rotation + ((angle - overlayAction.startAngle) * 180) / Math.PI
  }
}

function stopOverlayAction() {
  overlayAction = null
  window.removeEventListener('mousemove', onOverlayPointerMove)
  window.removeEventListener('mouseup', stopOverlayAction)
}

function syncOverlayHeight(item) {
  item.h = Math.max(40, item.w * item.ratio)
}

function toggleOverlayFlip(item) {
  item.flipX = !item.flipX
}

function removeOverlay(id) {
  overlayImages.value = overlayImages.value.filter(item => item.id !== id)
  if (selectedOverlayId.value === id) selectedOverlayId.value = ''
}

async function drawOverlayImages(ctx) {
  for (const item of overlayImages.value) {
    const img = await loadImageElement(item.src)
    ctx.save()
    ctx.translate(item.x + item.w / 2, item.y + item.h / 2)
    ctx.rotate((item.rotation * Math.PI) / 180)
    ctx.scale(item.flipX ? -1 : 1, 1)
    ctx.drawImage(img, -item.w / 2, -item.h / 2, item.w, item.h)
    ctx.restore()
  }
}

function undo() {
  if (undoStack.value.length === 0) return
  const snapshot = undoStack.value.pop()
  const ctx = drawCanvasRef.value.getContext('2d')
  ctx.putImageData(snapshot, 0, 0)
  isCanvasEmpty.value = undoStack.value.length === 0
}

function clearCanvas() {
  saveUndoSnapshot()
  const ctx = drawCanvasRef.value.getContext('2d')
  ctx.clearRect(0, 0, canvasW.value, canvasH.value)
  isCanvasEmpty.value = true
}

async function submit() {
  if (isSubmitting.value) return
  isSubmitting.value = true

  try {
    if (props.mode === 'ref') {
      // ref 模式：发送合成后的参考图，同时保留蒙版数据
      const mergeCanvas = document.createElement('canvas')
      mergeCanvas.width = canvasW.value
      mergeCanvas.height = canvasH.value
      const ctx = mergeCanvas.getContext('2d')
      ctx.drawImage(imageCanvasRef.value, 0, 0)
      ctx.drawImage(drawCanvasRef.value, 0, 0)
      await drawOverlayImages(ctx)
      const dataUrl = mergeCanvas.toDataURL('image/png')
      const maskData = drawCanvasRef.value.toDataURL('image/png').split(',')[1]
      emit('submit', {
        imageUrl: workingImageUrl.value || props.imageUrl,
        dataUrl,
        imageData: dataUrl.split(',')[1],
        mimeType: 'image/png',
        maskData: maskData,
        originalPrompt: props.originalPrompt,
      })
    } else {
      // result 模式：保持现有逻辑
      /** Merge image layer + drawing layer into one canvas */
      const mergeCanvas = document.createElement('canvas')
      mergeCanvas.width = canvasW.value
      mergeCanvas.height = canvasH.value
      const ctx = mergeCanvas.getContext('2d')
      ctx.drawImage(imageCanvasRef.value, 0, 0)
      ctx.drawImage(drawCanvasRef.value, 0, 0)

      const dataUrl = mergeCanvas.toDataURL('image/png')
      const base64 = dataUrl.split(',')[1]

      emit('submit', {
        prompt: editPrompt.value.trim() || props.originalPrompt,
        imageData: base64,
        mimeType: 'image/png',
        quality: quality.value,
      })
    }
  } finally {
    isSubmitting.value = false
  }
}

onMounted(() => window.addEventListener('resize', handleWindowResize))
onBeforeUnmount(() => {
  stopOverlayAction()
  window.removeEventListener('resize', handleWindowResize)
})
</script>

<style scoped>
.overlay-image-layer {
  left: 0;
  top: 0;
  z-index: 4;
  box-sizing: border-box;
}

.overlay-image-layer.selected {
  outline: 2px solid #111827;
  outline-offset: 2px;
}

.overlay-delete {
  position: absolute;
  top: -12px;
  right: -12px;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 999px;
  background: #111827;
  color: #fff;
  font-size: 16px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 6px 16px rgba(0, 0, 0, .2);
}

.overlay-flip {
  position: absolute;
  top: -12px;
  left: -12px;
  min-width: 28px;
  height: 24px;
  padding: 0 6px;
  border: 0;
  border-radius: 999px;
  background: #111827;
  color: #fff;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 6px 16px rgba(0, 0, 0, .2);
}

.overlay-rotate {
  position: absolute;
  left: 50%;
  top: -42px;
  width: 22px;
  height: 22px;
  transform: translateX(-50%);
  border: 2px solid #fff;
  border-radius: 999px;
  background: #111827;
  cursor: grab;
  box-shadow: 0 6px 16px rgba(0, 0, 0, .2);
}

.overlay-rotate::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 20px;
  width: 2px;
  height: 18px;
  transform: translateX(-50%);
  background: #111827;
}

.overlay-resize {
  position: absolute;
  right: -9px;
  bottom: -9px;
  width: 18px;
  height: 18px;
  border: 2px solid #fff;
  border-radius: 999px;
  background: #111827;
  cursor: nwse-resize;
  box-shadow: 0 6px 16px rgba(0, 0, 0, .2);
}
</style>
