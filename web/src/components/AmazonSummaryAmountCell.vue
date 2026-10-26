<template>
  <button v-if="canExpand" ref="anchor" class="amazon-summary-trigger" :class="{ active: open }" :aria-expanded="open" aria-label="查看金额明细" @click="toggle">
    <span class="summary-values"><span v-if="value == null" class="pending">待换算</span><span v-else>{{ money(value) }}</span></span>
    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
  </button>
  <span v-else class="amazon-summary-plain"><span v-if="value == null" class="pending">待换算</span><span v-else>{{ money(value) }}</span></span>
  <Teleport to="body">
    <section v-if="open" ref="panel" class="amazon-summary-popover" :style="position" aria-label="金额明细">
      <dl><div v-for="detail in details" :key="detail.label"><dt>{{ detail.label }}</dt><dd>{{ detail.value == null ? '待换算' : money(detail.value) }}</dd></div></dl>
    </section>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  value: { type: [Number, String], default: null },
  details: { type: Array, default: () => [] },
  currency: { type: String, required: true },
})
const symbol = computed(() => ({ USD: '$', CAD: 'C$', CNY: '¥', EUR: '€', GBP: '£', JPY: '¥', AUD: 'A$' }[props.currency] || `${props.currency} `))
const canExpand = computed(() => props.details.length > 1)
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const anchor = ref(null), panel = ref(null), open = ref(false), position = ref({})

async function toggle() {
  open.value = !open.value
  if (!open.value) return
  await nextTick()
  const rect = anchor.value?.getBoundingClientRect()
  const width = Math.min(300, window.innerWidth - 24)
  if (!rect) return
  position.value = { width: `${width}px`, left: `${Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))}px`, top: `${rect.bottom + 8}px` }
  await nextTick()
  const height = panel.value?.getBoundingClientRect().height || 0
  if (rect.bottom + height + 20 > window.innerHeight) position.value = { ...position.value, top: `${Math.max(12, rect.top - height - 8)}px` }
}
function outside(event) { if (!anchor.value?.contains(event.target) && !panel.value?.contains(event.target)) open.value = false }
function close() { open.value = false }
function keydown(event) { if (event.key === 'Escape' && open.value) { open.value = false; anchor.value?.focus() } }
onMounted(() => { document.addEventListener('click', outside); document.addEventListener('keydown', keydown); window.addEventListener('scroll', close, true); window.addEventListener('resize', close) })
onBeforeUnmount(() => { document.removeEventListener('click', outside); document.removeEventListener('keydown', keydown); window.removeEventListener('scroll', close, true); window.removeEventListener('resize', close) })
</script>

<style scoped>
.amazon-summary-trigger { display: inline-flex; align-items: center; justify-content: flex-end; gap: 4px; max-width: 100%; padding: 4px 0; border: 0; color: #455961; background: transparent; font: inherit; font-variant-numeric: tabular-nums; cursor: pointer; }
.amazon-summary-plain { display: inline-block; max-width: 100%; text-align: right; font-variant-numeric: tabular-nums; }
.amazon-summary-trigger:hover, .amazon-summary-trigger.active { color: #235c84; }
.amazon-summary-trigger svg { width: 14px; height: 14px; color: #7c8997; transition: transform 160ms ease; }
.amazon-summary-trigger.active svg { transform: rotate(180deg); }
.summary-values { display: inline-flex; flex-direction: column; align-items: flex-end; line-height: 1.25; }
.amazon-summary-popover { position: fixed; z-index: 1100; box-sizing: border-box; padding: 9px 10px; border: 1px solid #dfe5ec; border-radius: 6px; color: #465669; background: #fff; box-shadow: 0 10px 24px rgb(20 35 50 / 14%); font-size: 11px; text-align: left; }
dl { display: grid; gap: 6px; margin: 0; }
dl > div { display: flex; justify-content: space-between; gap: 16px; }
dt { color: #64748b; white-space: normal; }
dd { margin: 0; color: #465669; white-space: nowrap; }
.pending { color: #926a28; }
button:focus-visible { outline: 2px solid #235c84; outline-offset: 3px; }
</style>
