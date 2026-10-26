<template>
  <button v-if="canExpand" ref="anchor" class="summary-amount-trigger" :class="{ active: open }" :aria-expanded="open" aria-label="查看金额明细" @click="toggle">
      <span class="summary-values"><span v-if="value == null" class="pending">{{ pendingText }}</span><span v-else>{{ money(value) }}</span><small v-if="detailValue != null" class="breakdown-preview">明细合计 {{ money(detailValue) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
  </button>
  <span v-else class="summary-amount-plain"><span class="summary-values"><span v-if="value == null" class="pending">{{ pendingText }}</span><span v-else>{{ money(value) }}</span><small v-if="detailValue != null" class="breakdown-preview">明细合计 {{ money(detailValue) }}</small></span></span>
  <Teleport to="body"><section v-if="open" ref="panel" class="summary-amount-popover" :style="position" aria-label="金额明细"><dl><div v-for="detail in details" :key="detail.label"><dt>{{ detail.label }}</dt><dd>{{ detail.value == null ? pendingText : money(detail.value) }}</dd></div></dl></section></Teleport>
</template>
<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
const props = defineProps({ value: { type: [Number, String], default: null }, detailValue: { type: [Number, String], default: null }, detailLabel: { type: String, required: true }, details: { type: Array, default: () => [] }, pendingText: { type: String, default: '待换算' }, currency: { type: String, default: 'CNY' } })
const canExpand = computed(() => props.details.length > 1)
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const anchor = ref(null), panel = ref(null), open = ref(false), position = ref({})
async function toggle() { open.value = !open.value; if (!open.value) return; await nextTick(); const rect = anchor.value?.getBoundingClientRect(); if (!rect) return; const width = Math.min(320, window.innerWidth - 24); position.value = { width: `${width}px`, left: `${Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))}px`, top: `${rect.bottom + 8}px` }; await nextTick(); const height = panel.value?.getBoundingClientRect().height || 0; if (rect.bottom + height + 20 > window.innerHeight) position.value = { ...position.value, top: `${Math.max(12, rect.top - height - 8)}px` } }
function outside(event) { if (!anchor.value?.contains(event.target) && !panel.value?.contains(event.target)) open.value = false }
function close() { open.value = false }
function keydown(event) { if (event.key === 'Escape' && open.value) { open.value = false; anchor.value?.focus() } }
onMounted(() => { document.addEventListener('click', outside); document.addEventListener('keydown', keydown); window.addEventListener('scroll', close, true); window.addEventListener('resize', close) })
onBeforeUnmount(() => { document.removeEventListener('click', outside); document.removeEventListener('keydown', keydown); window.removeEventListener('scroll', close, true); window.removeEventListener('resize', close) })
</script>
<style scoped>
.summary-amount-cell { position: relative; min-width: 150px; font-variant-numeric: tabular-nums; }
.summary-amount-plain { display: inline-block; min-width: 150px; text-align: right; font-variant-numeric: tabular-nums; }
.summary-amount-trigger { display: flex; align-items: center; justify-content: flex-end; gap: 7px; width: 100%; min-width: 150px; padding: 4px 0; border: 0; color: inherit; background: transparent; font: inherit; font-variant-numeric: tabular-nums; cursor: pointer; }
.summary-amount-trigger:hover, .summary-amount-trigger.active { color: #235c84; }
.summary-amount-trigger svg { flex: 0 0 14px; width: 14px; height: 14px; color: #7c8997; transition: transform 160ms ease; }
.summary-amount-trigger.active svg { transform: rotate(180deg); }
.summary-values { display: inline-flex; flex-direction: column; align-items: flex-end; line-height: 1.25; }
.breakdown-preview { color: #8a96a3; font-size: 10px; font-weight: 400; white-space: nowrap; }
.summary-amount-trigger:focus-visible { outline: 2px solid #235c84; outline-offset: 3px; }
.summary-amount-popover { position: fixed; z-index: 1100; box-sizing: border-box; min-width: 190px; padding: 9px 10px; border: 1px solid #dfe5ec; border-radius: 6px; color: #465669; background: #fff; box-shadow: 0 10px 24px rgb(20 35 50 / 14%); font-size: 11px; text-align: left; }
dl { display: grid; gap: 6px; margin: 0; } dl > div { display: flex; justify-content: space-between; gap: 16px; } dt { color: #64748b; } dd { margin: 0; color: #465669; white-space: nowrap; } .pending { color: #926a28; }
</style>
