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
      <div v-if="visible" class="expand-backdrop" @click.self="emit('close')">
        <div class="expand-dialog" role="dialog" aria-modal="true" aria-labelledby="expand-title">
          <header class="expand-header">
            <div>
              <h3 id="expand-title">AI 扩图</h3>
              <!-- <p v-if="naturalWidth">{{ naturalWidth }} × {{ naturalHeight }} → {{ outputSizeText }}</p> -->
            </div>
            <button class="icon-button" type="button" aria-label="关闭" title="关闭" @click="emit('close')">
              <XMarkIcon />
            </button>
          </header>

          <div class="expand-body">
            <section class="preview-panel">
              <div v-if="loading" class="preview-state">
                <span class="spinner" />
                <span>图片加载中</span>
              </div>
              <div v-else-if="loadError" class="preview-state preview-state--error">{{ loadError }}</div>
              <div v-else class="preview-wrap">
                <div
                  ref="stageRef"
                  class="preview-stage"
                  :class="{ dragging }"
                  :style="previewStageStyle"
                  @pointerdown="startDrag"
                  @pointermove="moveDrag"
                  @pointerup="stopDrag"
                  @pointercancel="stopDrag"
                >
                  <div class="expand-zone" />
                  <img :src="imageUrl" alt="待扩展图片" draggable="false" :style="previewImageStyle" />
                  <span class="frame-corner frame-corner--tl" />
                  <span class="frame-corner frame-corner--tr" />
                  <span class="frame-corner frame-corner--bl" />
                  <span class="frame-corner frame-corner--br" />
                </div>
                <span class="preview-hint">拖动图片调整扩图位置</span>
              </div>
            </section>

            <aside class="settings-panel">
              <fieldset>
                <legend>扩图比例</legend>
                <div class="ratio-grid">
                  <button
                    v-for="option in ratioOptions"
                    :key="option.value"
                    type="button"
                    :class="{ active: selectedRatio === option.value }"
                    @click="selectRatio(option.value)"
                  >
                    <span class="ratio-shape" :style="ratioShapeStyle(option.value)" />
                    <span>{{ option.label }}</span>
                  </button>
                </div>
              </fieldset>

              <fieldset>
                <legend>扩图倍数</legend>
                <div class="factor-grid">
                  <button
                    v-for="factor in factorOptions"
                    :key="factor"
                    type="button"
                    :class="{ active: selectedFactor === factor }"
                    @click="selectFactor(factor)"
                  >
                    {{ Math.round(factor * 100) }}%
                  </button>
                </div>
              </fieldset>

              <fieldset>
                <legend>清晰度</legend>
                <div class="quality-grid">
                  <button
                    v-for="quality in resolutionOptions"
                    :key="quality"
                    type="button"
                    :class="{ active: selectedResolution === quality }"
                    @click="selectedResolution = quality"
                  >
                    {{ quality }}
                  </button>
                </div>
              </fieldset>
            </aside>
          </div>

          <footer class="expand-footer">
            <button type="button" class="secondary-button" @click="emit('close')">取消</button>
            <button type="button" class="primary-button" :disabled="loading || !!loadError || submitting" @click="submitExpand">
              <SparklesIcon />
              {{ submitting ? '处理中…' : '开始扩图' }}
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { SparklesIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { fitExpandLayoutForExport, getExpandLayout, parseImageRatio } from '@/utils/imageExpand'

const props = defineProps({
  visible: { type: Boolean, default: false },
  imageUrl: { type: String, default: '' },
  originalPrompt: { type: String, default: '' },
  originalAspectRatio: { type: String, default: '1:1' },
  resolutions: { type: Array, default: () => ['1K', '2K'] },
  defaultResolution: { type: String, default: '1K' },
})

const emit = defineEmits(['close', 'submit'])
const ratioOptions = [
  { label: '原比例', value: 'original' },
  { label: '1:1', value: '1:1' },
  { label: '3:2', value: '3:2' },
  { label: '2:3', value: '2:3' },
  { label: '4:3', value: '4:3' },
  { label: '3:4', value: '3:4' },
  { label: '16:9', value: '16:9' },
  { label: '9:16', value: '9:16' },
  { label: '21:9', value: '21:9' },
]
const factorOptions = [1, 1.25, 1.5, 2, 3]

const stageRef = ref(null)
const naturalWidth = ref(0)
const naturalHeight = ref(0)
const selectedRatio = ref('original')
const selectedFactor = ref(1.25)
const selectedResolution = ref('1K')
const position = ref({ x: 0.5, y: 0.5 })
const loading = ref(false)
const loadError = ref('')
const submitting = ref(false)
const loadedImage = ref(null)
const dragging = ref(false)
const dragStart = ref(null)

const layout = computed(() => getExpandLayout(
  naturalWidth.value,
  naturalHeight.value,
  selectedRatio.value,
  selectedFactor.value,
  position.value
))

const targetAspectRatio = computed(() => parseImageRatio(
  selectedRatio.value,
  naturalWidth.value || 1,
  naturalHeight.value || 1
))

const apiAspectRatio = computed(() => selectedRatio.value === 'original'
  ? props.originalAspectRatio
  : selectedRatio.value)

const resolutionOptions = computed(() => {
  const values = props.resolutions.map(value => String(value).toUpperCase()).filter(Boolean)
  return [...new Set(values.length ? values : ['1K', '2K'])]
})

const outputSizeText = computed(() => `${Math.round(layout.value.targetWidth)} × ${Math.round(layout.value.targetHeight)}`)

const previewStageStyle = computed(() => {
  const maxWidth = 760
  const maxHeight = 540
  const scale = Math.min(maxWidth / layout.value.targetWidth, maxHeight / layout.value.targetHeight)
  return {
    width: `${Math.max(1, layout.value.targetWidth * scale)}px`,
    height: `${Math.max(1, layout.value.targetHeight * scale)}px`,
  }
})

const previewImageStyle = computed(() => ({
  left: `${(layout.value.imageX / layout.value.targetWidth) * 100}%`,
  top: `${(layout.value.imageY / layout.value.targetHeight) * 100}%`,
  width: `${(layout.value.imageWidth / layout.value.targetWidth) * 100}%`,
  height: `${(layout.value.imageHeight / layout.value.targetHeight) * 100}%`,
}))

watch(() => [props.visible, props.imageUrl], ([visible]) => {
  if (!visible || !props.imageUrl) return
  selectedRatio.value = 'original'
  selectedFactor.value = 1.25
  const requestedResolution = String(props.defaultResolution || '').toUpperCase()
  selectedResolution.value = resolutionOptions.value.includes(requestedResolution)
    ? requestedResolution
    : resolutionOptions.value[0]
  resetPosition()
  loadImage()
}, { immediate: true })

function loadImage() {
  loading.value = true
  loadError.value = ''
  const image = new Image()
  image.onload = () => {
    loadedImage.value = image
    naturalWidth.value = image.naturalWidth
    naturalHeight.value = image.naturalHeight
    loading.value = false
  }
  image.onerror = () => {
    loadedImage.value = null
    loading.value = false
    loadError.value = '图片加载失败，请重试'
  }
  image.src = props.imageUrl
}

function selectRatio(value) {
  selectedRatio.value = value
  resetPosition()
}

function selectFactor(value) {
  selectedFactor.value = value
  resetPosition()
}

function resetPosition() {
  position.value = { x: 0.5, y: 0.5 }
}

function startDrag(event) {
  if (!stageRef.value) return
  dragging.value = true
  event.currentTarget.setPointerCapture(event.pointerId)
  dragStart.value = { x: event.clientX, y: event.clientY, position: { ...position.value } }
}

function moveDrag(event) {
  if (!dragging.value || !dragStart.value || !stageRef.value) return
  const rect = stageRef.value.getBoundingClientRect()
  const imageWidth = rect.width * layout.value.imageWidth / layout.value.targetWidth
  const imageHeight = rect.height * layout.value.imageHeight / layout.value.targetHeight
  const rangeX = Math.max(0, rect.width - imageWidth)
  const rangeY = Math.max(0, rect.height - imageHeight)

  position.value = {
    x: rangeX ? Math.min(1, Math.max(0, dragStart.value.position.x + (event.clientX - dragStart.value.x) / rangeX)) : 0.5,
    y: rangeY ? Math.min(1, Math.max(0, dragStart.value.position.y + (event.clientY - dragStart.value.y) / rangeY)) : 0.5,
  }
}

function stopDrag(event) {
  dragging.value = false
  dragStart.value = null
  if (event.currentTarget?.hasPointerCapture?.(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
}

function ratioShapeStyle(value) {
  const ratio = value === 'original'
    ? (naturalWidth.value || 1) / (naturalHeight.value || 1)
    : parseImageRatio(value, 1, 1)
  const width = ratio >= 1 ? 22 : 22 * ratio
  const height = ratio >= 1 ? 22 / ratio : 22
  return { width: `${Math.max(7, width)}px`, height: `${Math.max(7, height)}px` }
}

async function submitExpand() {
  if (!loadedImage.value || submitting.value) return
  submitting.value = true

  try {
    const exportLayout = fitExpandLayoutForExport(layout.value)
    const canvas = document.createElement('canvas')
    canvas.width = exportLayout.targetWidth
    canvas.height = exportLayout.targetHeight
    const context = canvas.getContext('2d')
    context.fillStyle = '#000000'
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(
      loadedImage.value,
      exportLayout.imageX,
      exportLayout.imageY,
      exportLayout.imageWidth,
      exportLayout.imageHeight
    )

    const prompt = `基于参考图无缝扩展画面。完整保留原图主体、商品、人物、文字、颜色、光影与细节，不裁剪、不改动原图内容；仅生成黑色留白区域的自然延展内容，使边缘、透视、景深和光线连续一致。`

    emit('submit', {
      prompt,
      imageData: canvas.toDataURL('image/png').split(',')[1],
      mimeType: 'image/png',
      aspectRatio: apiAspectRatio.value,
      resolution: selectedResolution.value,
    })
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.expand-backdrop { position: fixed; inset: 0; z-index: 60; display: flex; align-items: center; justify-content: center; padding: 16px; background: rgba(0,0,0,.62); backdrop-filter: blur(6px); }
.expand-dialog { display: flex; width: min(1180px, 96vw); height: min(780px, 92vh); flex-direction: column; overflow: hidden; border: 1px solid rgba(255,255,255,.6); border-radius: 18px; background: #fff; box-shadow: 0 28px 80px rgba(15,23,42,.28); }
.expand-header { display: flex; min-height: 70px; align-items: center; justify-content: space-between; padding: 14px 20px; border-bottom: 1px solid #eceef1; }
.expand-header h3 { margin: 0; color: #171717; font-size: 17px; font-weight: 700; }
.expand-header p { margin: 3px 0 0; color: #9ca3af; font-size: 11px; }
.icon-button { display: inline-flex; width: 34px; height: 34px; align-items: center; justify-content: center; border: 0; border-radius: 50%; background: #f3f4f6; color: #6b7280; }
.icon-button:hover { background: #e5e7eb; color: #171717; }
.icon-button svg { width: 18px; height: 18px; }
.expand-body { display: grid; min-height: 0; flex: 1; grid-template-columns: minmax(0, 1fr) 300px; }
.preview-panel { min-width: 0; min-height: 0; overflow: hidden; background: #f0f1f3; }
.preview-wrap, .preview-state { display: flex; width: 100%; height: 100%; align-items: center; justify-content: center; padding: 24px; color: #71717a; font-size: 13px; }
.preview-wrap { position: relative; }
.preview-hint { position: absolute; bottom: 14px; left: 50%; z-index: 4; padding: 6px 10px; border: 1px solid rgba(255,255,255,.72); border-radius: 999px; background: rgba(255,255,255,.86); box-shadow: 0 5px 14px rgba(15,23,42,.1); color: #52525b; font-size: 11px; pointer-events: none; transform: translateX(-50%); white-space: nowrap; }
.preview-state { flex-direction: column; gap: 10px; }
.preview-state--error { color: #b91c1c; }
.spinner { width: 30px; height: 30px; border: 3px solid #d4d4d8; border-top-color: #27272a; border-radius: 50%; animation: spin .8s linear infinite; }
.preview-stage { position: relative; max-width: 100%; max-height: 100%; overflow: hidden; cursor: grab; touch-action: none; background-color: #e4e6e9; background-image: linear-gradient(45deg,#d7d9dd 25%,transparent 25%),linear-gradient(-45deg,#d7d9dd 25%,transparent 25%),linear-gradient(45deg,transparent 75%,#d7d9dd 75%),linear-gradient(-45deg,transparent 75%,#d7d9dd 75%); background-position: 0 0,0 8px,8px -8px,-8px 0; background-size: 16px 16px; box-shadow: 0 14px 38px rgba(15,23,42,.16); }
.preview-stage.dragging { cursor: grabbing; }
.preview-stage img { position: absolute; z-index: 2; object-fit: fill; pointer-events: none; user-select: none; box-shadow: 0 3px 14px rgba(15,23,42,.12); }
.expand-zone { position: absolute; inset: 0; z-index: 1; border: 1px dashed rgba(31,41,55,.5); background: rgba(255,255,255,.25); }
.frame-corner { position: absolute; z-index: 3; width: 18px; height: 18px; border-color: #fff; filter: drop-shadow(0 1px 2px rgba(0,0,0,.5)); pointer-events: none; }
.frame-corner--tl { top: 5px; left: 5px; border-top: 2px solid; border-left: 2px solid; }
.frame-corner--tr { top: 5px; right: 5px; border-top: 2px solid; border-right: 2px solid; }
.frame-corner--bl { bottom: 5px; left: 5px; border-bottom: 2px solid; border-left: 2px solid; }
.frame-corner--br { right: 5px; bottom: 5px; border-right: 2px solid; border-bottom: 2px solid; }
.settings-panel { min-height: 0; overflow-y: auto; padding: 20px; border-left: 1px solid #eceef1; background: #fff; }
.settings-panel fieldset { margin: 0 0 24px; padding: 0; border: 0; }
.settings-panel legend { margin-bottom: 10px; color: #52525b; font-size: 12px; font-weight: 700; }
.ratio-grid { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 7px; }
.ratio-grid button { display: flex; min-width: 0; height: 62px; flex-direction: column; align-items: center; justify-content: center; gap: 6px; border: 1px solid #e4e4e7; border-radius: 7px; background: #fafafa; color: #71717a; font-size: 10px; }
.ratio-grid button:hover, .factor-grid button:hover { border-color: #a1a1aa; color: #27272a; }
.ratio-grid button.active, .factor-grid button.active { border-color: #27272a; background: #27272a; color: #fff; }
.ratio-shape { display: block; border: 1.5px solid currentColor; border-radius: 2px; }
.factor-grid { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: 7px; }
.factor-grid button { height: 36px; border: 1px solid #e4e4e7; border-radius: 7px; background: #fafafa; color: #52525b; font-size: 11px; font-weight: 650; }
.quality-grid { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: 7px; }
.quality-grid button { height: 36px; border: 1px solid #e4e4e7; border-radius: 7px; background: #fafafa; color: #52525b; font-size: 11px; font-weight: 650; }
.quality-grid button:hover { border-color: #a1a1aa; color: #27272a; }
.quality-grid button.active { border-color: #27272a; background: #27272a; color: #fff; }
.expand-footer { display: flex; min-height: 70px; align-items: center; justify-content: flex-end; gap: 10px; padding: 12px 20px; border-top: 1px solid #eceef1; background: #fff; }
.secondary-button, .primary-button { display: inline-flex; height: 40px; align-items: center; justify-content: center; gap: 7px; padding: 0 18px; border-radius: 8px; font-size: 13px; font-weight: 650; }
.secondary-button { border: 1px solid #e4e4e7; background: #fff; color: #52525b; }
.primary-button { border: 1px solid #18181b; background: #18181b; color: #fff; }
.primary-button:hover { background: #27272a; }
.primary-button:disabled { cursor: not-allowed; opacity: .45; }
.primary-button svg { width: 16px; height: 16px; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 760px) {
  .expand-dialog { height: min(860px, 96vh); }
  .expand-body { grid-template-columns: 1fr; grid-template-rows: minmax(260px,1fr) auto; }
  .settings-panel { max-height: 310px; border-top: 1px solid #eceef1; border-left: 0; }
  .preview-wrap { padding: 14px; }
}
</style>
