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
        v-if="visible && currentImage"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      >
        <div class="preview-container" ref="containerRef" :style="{ maxWidth: maxWidth + 'px' }">
          <!-- Header slot -->
          <div v-if="$slots.header" class="preview-header">
            <slot name="header" />
          </div>

          <!-- Image viewport -->
          <div
            class="preview-viewport"
            ref="viewportRef"
            :style="{
              cursor: isDragging ? 'grabbing' : (scale > 1 ? 'grab' : 'default')
            }"
            @mousedown.prevent="startDrag"
            @mousemove.prevent="onDrag"
            @mouseup.prevent="endDrag"
            @mouseleave="endDrag"
            @wheel.prevent="onWheel"
            @dblclick="reset"
          >
            <img
              v-if="currentImage"
              :src="currentImage"
              :alt="alt"
              class="preview-image"
              :style="{
                transform: `translate(${offset.x}px, ${offset.y}px) scale(${scale})`,
                transformOrigin: '0 0'
              }"
              draggable="false"
              @click.stop
            />
          </div>

          <!-- Close button (top-right) -->
          <button
            class="preview-close-btn"
            type="button"
            @click="$emit('close')"
            title="关闭"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>

          <!-- Footer slot -->
          <div v-if="$slots.footer" class="preview-footer">
            <slot name="footer" />
          </div>

          <!-- Controls -->
          <div v-if="showControls" class="preview-controls">
            <button class="ctrl-btn" @click="adjustScale(-0.25)" title="缩小">−</button>
            <span class="ctrl-level">{{ Math.round(scale * 100) }}%</span>
            <button class="ctrl-btn" @click="adjustScale(0.25)" title="放大">+</button>
          </div>

          <!-- Navigation arrows (multi-image) -->
          <template v-if="hasMultiple">
            <button
              v-if="!isFirst"
              class="nav-btn nav-prev"
              type="button"
              @click="prevImage"
              title="上一张"
            >‹</button>
            <button
              v-if="!isLast"
              class="nav-btn nav-next"
              type="button"
              @click="nextImage"
              title="下一张"
            >›</button>
            <div v-if="hasMultiple" class="nav-indicator">
              {{ currentIndex + 1 }} / {{ images.length }}
            </div>
          </template>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { resolveImageUrl } from '@/utils/request'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  imageUrl: {
    type: String,
    default: '',
  },
  images: {
    type: Array,
    default: () => [],
  },
  initialIndex: {
    type: Number,
    default: 0,
  },
  alt: {
    type: String,
    default: '',
  },
  maxWidth: {
    type: Number,
    default: 1280,
  },
  maxHeight: {
    type: Number,
    default: 0,
  },
  showControls: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits(['close'])

const viewportRef = ref(null)
const containerRef = ref(null)
const scale = ref(1)
const offset = ref({ x: 0, y: 0 })
const isDragging = ref(false)
const dragStart = ref({ x: 0, y: 0 })
const lastOffset = ref({ x: 0, y: 0 })

// Anchor: display size at scale=1
const anchorDW = ref(0)
const anchorDH = ref(0)

const MIN_SCALE = 0.5
const MAX_SCALE = 5

// Multi-image support
const currentIndex = ref(0)
const hasMultiple = computed(() => props.images.length > 1)
const isFirst = computed(() => currentIndex.value === 0)
const isLast = computed(() => currentIndex.value === props.images.length - 1)
const currentImage = computed(() => {
  const raw = props.images.length > 0 ? props.images[currentIndex.value] : props.imageUrl
  return resolveImageUrl(raw)
})

function captureDimensions() {
  const container = containerRef.value
  const vp = viewportRef.value
  if (!container || !vp) return
  const imgEl = vp.querySelector('img')
  if (!imgEl) return

  // Use rendered dimensions (actual display size at scale=1)
  const imgW = imgEl.offsetWidth
  const imgH = imgEl.offsetHeight
  if (!imgW || !imgH) return

  anchorDW.value = imgW
  anchorDH.value = imgH
}

watch(() => props.visible, async (val) => {
  if (!val) {
    reset()
    currentIndex.value = 0
    return
  }
  await nextTick()
  captureDimensions()
})

watch(currentIndex, async () => {
  reset()
  await nextTick()
  captureDimensions()
})

watch(() => props.images, (newImages) => {
  if (newImages.length > 0 && currentIndex.value >= newImages.length) {
    currentIndex.value = 0
  }
})

function onWheel(e) {
  if (!anchorDW.value || !anchorDH.value) return

  const delta = e.deltaY > 0 ? -0.15 : 0.15
  const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale.value + delta))

  // Image visual center = viewport center + offset + anchorDW/2
  // To keep image centered: offset = -anchorDW/2 * (newScale - 1)
  offset.value = {
    x: -anchorDW.value / 2 * (newScale - 1),
    y: -anchorDH.value / 2 * (newScale - 1),
  }
  scale.value = newScale
}

function startDrag(e) {
  if (scale.value <= 1) return
  isDragging.value = true
  dragStart.value = { x: e.clientX, y: e.clientY }
  lastOffset.value = { ...offset.value }
}

function onDrag(e) {
  if (!isDragging.value) return
  offset.value = {
    x: lastOffset.value.x + (e.clientX - dragStart.value.x),
    y: lastOffset.value.y + (e.clientY - dragStart.value.y),
  }
}

function endDrag() {
  isDragging.value = false
}

function adjustScale(delta) {
  if (!anchorDW.value || !anchorDH.value) return
  const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale.value + delta))
  offset.value = {
    x: -anchorDW.value / 2 * (newScale - 1),
    y: -anchorDH.value / 2 * (newScale - 1),
  }
  scale.value = newScale
}

function prevImage() {
  if (currentIndex.value > 0) currentIndex.value--
}

function nextImage() {
  if (currentIndex.value < props.images.length - 1) currentIndex.value++
}

async function goToImage(index) {
  if (index < 0 || index >= props.images.length) return
  currentIndex.value = index
  reset()
  await nextTick()
  captureDimensions()
}

function reset() {
  scale.value = 1
  offset.value = { x: 0, y: 0 }
}

function handleKeydown(e) {
  if (!props.visible) return
  if (e.key === 'ArrowLeft') prevImage()
  if (e.key === 'ArrowRight') nextImage()
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))

defineExpose({ adjustScale, reset, prevImage, nextImage, goToImage, currentIndex })
</script>

<style scoped>
.preview-container {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0;
  height: calc(100vh - 32px);
  max-height: calc(100vh - 32px);
  overflow: hidden;
}

.preview-header {
  flex-shrink: 0;
  position: relative;
  width: 100%;
  margin-bottom: 12px;
}

.preview-close {
  position: absolute;
  top: -18px;
  right: -18px;
  z-index: 2;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 999px;
  background: rgba(17, 24, 39, .92);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: background .15s ease;
}

.preview-close:hover {
  background: rgba(17, 24, 39, 1);
}

.preview-viewport {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 32px;
  border-radius: 18px;
}

.preview-image {
  border-radius: 18px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, .36);
  background: #fff;
  will-change: transform;
  user-select: none;
  -webkit-user-drag: none;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.preview-footer {
  flex-shrink: 0;
  width: 100%;
  margin-top: 12px;
}

.preview-close-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: rgba(17, 24, 39, .75);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all .2s ease;
  backdrop-filter: blur(4px);
}

.preview-close-btn:hover {
  background: rgba(17, 24, 39, .95);
  transform: scale(1.1);
}

.preview-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 6px 12px;
  background: rgba(17, 24, 39, .92);
  border-radius: 999px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}

.ctrl-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.15);
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all .15s ease;
}

.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.25);
}

.ctrl-level {
  min-width: 52px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
}

.nav-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 3;
  width: 44px;
  height: 44px;
  border: none;
  border-radius: 999px;
  background: rgba(17, 24, 39, .6);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26px;
  font-weight: 300;
  transition: background .15s ease;
}

.nav-btn:hover {
  background: rgba(17, 24, 39, .85);
}

.nav-prev { left: 16px; }
.nav-next { right: 16px; }

.nav-indicator {
  position: absolute;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 3;
  padding: 4px 12px;
  background: rgba(17, 24, 39, .6);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  border-radius: 999px;
  backdrop-filter: blur(4px);
}
</style>
