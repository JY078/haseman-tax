<template>
  <Dialog :open="!!media" :style="{ position: 'fixed', inset: 0, zIndex: 120 }" @close="$emit('close')">
    <div class="media-backdrop" aria-hidden="true" />
    <div class="media-position">
      <DialogPanel class="media-panel" @keydown="onKeydown">
        <DialogTitle class="sr-only">{{ media?.type === 'video' ? '评论视频' : '评论图片' }}</DialogTitle>
        <button class="media-control media-close" type="button" aria-label="关闭预览" @click="$emit('close')"><XMarkIcon /></button>
        <div class="media-stage">
          <video v-if="media?.type === 'video'" :key="currentUrl" :src="currentUrl" controls autoplay playsinline preload="metadata" @error="failed = true" />
          <img v-else-if="currentUrl" :key="currentUrl" :src="currentUrl" :alt="'评论图片 ' + (index + 1)" @error="failed = true">
          <p v-if="failed" class="media-error" role="alert">{{ media?.type === 'video' ? '视频无法播放，请检查文件链接或浏览器是否支持该视频编码。' : '图片加载失败，请稍后重试。' }}</p>
        </div>
        <template v-if="media && media.urls.length > 1">
          <button class="media-control media-prev" type="button" :disabled="index === 0" :aria-label="media.type === 'video' ? '上一个' : '上一张'" @click="move(-1)"><ChevronLeftIcon /></button>
          <button class="media-control media-next" type="button" :disabled="index === media.urls.length - 1" :aria-label="media.type === 'video' ? '下一个' : '下一张'" @click="move(1)"><ChevronRightIcon /></button>
          <span class="media-counter" aria-live="polite">{{ index + 1 }} <span>/</span> {{ media.urls.length }}</span>
        </template>
      </DialogPanel>
    </div>
  </Dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/vue'
import { ChevronLeftIcon, ChevronRightIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const props = defineProps({ media: { type: Object, default: null } })
defineEmits(['close'])
const index = ref(0)
const failed = ref(false)
const currentUrl = computed(() => props.media?.urls[index.value] || '')
watch(() => props.media, value => { index.value = value?.index || 0; failed.value = false })
watch(currentUrl, () => { failed.value = false })
function move(step) { if (props.media) index.value = Math.max(0, Math.min(props.media.urls.length - 1, index.value + step)) }
function onKeydown(event) {
  if (event.target?.tagName === 'VIDEO') return
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') { event.preventDefault(); move(event.key === 'ArrowLeft' ? -1 : 1) }
}
</script>

<style scoped>
.media-backdrop{position:fixed;inset:0;background:rgba(12,17,24,.86);backdrop-filter:blur(3px)}
.media-position{position:fixed;inset:0;display:flex;align-items:center;justify-content:center;padding:72px 80px}
.media-panel{position:relative;max-width:100%;color:#fff;background:transparent;border:0;box-shadow:none}
.media-stage{position:relative;display:flex;flex-direction:column;align-items:center;justify-content:center}
.media-stage img,.media-stage video{display:block;max-width:min(1400px,calc(100vw - 160px));max-height:calc(100dvh - 144px);object-fit:contain;border-radius:0}
.media-stage video{width:min(1100px,calc(100vw - 160px));background:#000}
.media-control{position:fixed;display:grid;place-items:center;width:44px;height:44px;border:1px solid #ffffff26;border-radius:50%;background:#ffffff0d;color:#fff;cursor:pointer;transition:background .15s,border-color .15s}
.media-control svg{width:22px;height:22px;stroke-width:1.7}.media-control:hover:not(:disabled){background:#ffffff26;border-color:#ffffff59}.media-control:focus-visible{outline:2px solid #a9cff5;outline-offset:4px}.media-control:disabled{opacity:.22;cursor:default}
.media-close{top:20px;right:24px}.media-prev{left:24px;top:calc(50% - 22px)}.media-next{right:24px;top:calc(50% - 22px)}
.media-counter{position:fixed;bottom:22px;left:50%;transform:translateX(-50%);padding:7px 15px;border-radius:20px;background:#00000040;color:#f1f4f8;font-size:12px;font-variant-numeric:tabular-nums;letter-spacing:2px}.media-counter span{color:#85909f;margin:0 4px}
.media-error{max-width:440px;padding:12px 16px;margin-top:12px;border-radius:6px;background:#21191a;color:#f3d2ce;font-size:13px;line-height:1.6;text-align:center}
@media(max-width:560px){.media-position{padding:68px 12px 88px}.media-stage img,.media-stage video{max-width:calc(100vw - 24px);max-height:calc(100dvh - 156px)}.media-stage video{width:calc(100vw - 24px)}.media-close{top:16px;right:16px}.media-prev,.media-next{top:auto;bottom:20px;width:38px;height:38px}.media-prev{left:calc(50% - 100px)}.media-next{right:calc(50% - 100px)}.media-counter{bottom:23px}}
@media(prefers-reduced-motion:reduce){.media-control{transition:none}}
</style>
