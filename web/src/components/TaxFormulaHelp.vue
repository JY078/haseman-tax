<template>
  <button
    ref="anchor"
    class="formula-help"
    type="button"
    :aria-label="`${label}计算方式`"
    :aria-describedby="visible ? tooltipId : undefined"
    @mouseenter="open"
    @mouseleave="scheduleClose"
    @focus="open"
    @blur="scheduleClose"
    @click="toggle"
  >
    <span aria-hidden="true">!</span>
  </button>
  <Teleport to="body">
    <span
      v-if="visible"
      :id="tooltipId"
      ref="tooltip"
      class="formula-tooltip"
      role="tooltip"
      :style="position"
      @mouseenter="cancelClose"
      @mouseleave="scheduleClose"
    >{{ formula }}</span>
  </Teleport>
</template>

<script setup>
import { nextTick, onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  label: { type: String, required: true },
  formula: { type: String, required: true },
})
const anchor = ref(null)
const tooltip = ref(null)
const visible = ref(false)
const position = ref({})
const tooltipId = `tax-formula-${Math.random().toString(36).slice(2)}`
let closeTimer

function cancelClose() {
  if (closeTimer) window.clearTimeout(closeTimer)
}

function scheduleClose() {
  cancelClose()
  closeTimer = window.setTimeout(() => { visible.value = false }, 100)
}

async function open() {
  cancelClose()
  visible.value = true
  await nextTick()
  const rect = anchor.value?.getBoundingClientRect()
  const panel = tooltip.value
  if (!rect || !panel) return
  const width = Math.min(360, window.innerWidth - 24)
  const left = Math.max(12, Math.min(rect.left, window.innerWidth - width - 12))
  position.value = { width: `${width}px`, left: `${left}px`, top: `${rect.bottom + 8}px` }
  const height = panel.getBoundingClientRect().height
  if (rect.bottom + height + 16 > window.innerHeight) {
    position.value = { ...position.value, top: `${Math.max(12, rect.top - height - 8)}px` }
  }
}

function toggle() {
  if (visible.value) scheduleClose()
  else open()
}

function closeOnGlobalPointer(event) {
  if (!visible.value) return
  if (anchor.value?.contains(event.target) || tooltip.value?.contains(event.target)) return
  visible.value = false
}

function closeOnViewportChange() {
  visible.value = false
}

document.addEventListener('pointerdown', closeOnGlobalPointer)
window.addEventListener('resize', closeOnViewportChange)
window.addEventListener('scroll', closeOnViewportChange, true)
onBeforeUnmount(() => {
  cancelClose()
  document.removeEventListener('pointerdown', closeOnGlobalPointer)
  window.removeEventListener('resize', closeOnViewportChange)
  window.removeEventListener('scroll', closeOnViewportChange, true)
})
</script>

<style scoped>
.formula-help { display: inline-grid; place-items: center; width: 15px; height: 15px; margin: 0; padding: 0; border: 1px solid #aebbc6; border-radius: 50%; color: #73818c; background: #fff; font-size: 10px; font-weight: 600; line-height: 1; cursor: help; }
.formula-help:hover { border-color: #7e929f; color: #526570; background: #f4f7f8; }
.formula-help:focus-visible { outline: 2px solid #b8cbd6; outline-offset: 2px; }
.formula-tooltip { position: fixed; z-index: 1200; box-sizing: border-box; padding: 9px 11px; border: 1px solid #52636d; border-radius: 5px; color: #f8fafb; background: #35434b; box-shadow: 0 7px 18px rgb(30 44 52 / 22%); font-size: 11px; font-weight: 400; line-height: 1.55; text-align: left; white-space: normal; overflow-wrap: anywhere; pointer-events: auto; }
</style>
