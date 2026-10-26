<template>
  <div ref="root" class="date-range">
    <button ref="trigger" type="button" class="range-trigger" :aria-expanded="open" aria-haspopup="dialog" :aria-label="label" @click="toggle">
      <svg viewBox="0 0 20 20" fill="none" aria-hidden="true"><rect x="3" y="4" width="14" height="13" rx="2" /><path d="M3 8h14M7 2v4M13 2v4" /></svg>
      <strong>{{ from ? `${label}：${from} 至 ${to || from}` : label }}</strong><svg class="range-chevron" :class="{ open }" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
    </button>
    <Teleport to="body">
      <section v-if="open" ref="panel" class="range-panel" :style="panelStyle" role="dialog" aria-modal="true" :aria-label="`选择${label}范围`" @keydown="onPanelKey">
        <header><strong>{{ label }}</strong><button type="button" aria-label="关闭日期选择" @click="close">×</button></header>
        <div class="quick-options"><button type="button" @click="quick('month')">本月</button><button type="button" @click="quick('last')">上月</button><button type="button" @click="quick('year')">本年</button><button type="button" @click="draftFrom = ''; draftTo = ''">全部日期</button></div>
        <div class="calendars">
          <section v-for="(month, index) in months" :key="month.key" class="calendar" :aria-label="month.label">
            <div class="month-nav"><button type="button" :aria-label="index ? '右侧上一月' : '上一月'" @click="shift(-1)">‹</button><strong>{{ month.label }}</strong><button type="button" :aria-label="index ? '下一月' : '左侧下一月'" @click="shift(1)">›</button></div>
            <div class="days"><span v-for="day in weekdays" :key="day" class="weekday">{{ day }}</span>
              <template v-for="(day, cell) in month.days" :key="cell"><button v-if="day" type="button" :aria-label="day.value" :aria-pressed="day.value === draftFrom || day.value === draftTo" :class="{ endpoint: day.value === draftFrom || day.value === draftTo, selected: inRange(day.value), today: day.value === today }" @click="select(day.value)" @mouseenter="hover = day.value" @mouseleave="hover = ''">{{ day.label }}</button><span v-else></span></template>
            </div>
          </section>
        </div>
        <footer><span aria-live="polite">{{ draftFrom ? `${draftFrom} 至 ${draftTo || '请选择结束日期'}` : '不限日期' }}</span><button type="button" @click="close">取消</button><button type="button" class="apply" :disabled="!!draftFrom && !draftTo" @click="apply">确定</button></footer>
      </section>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
const props = defineProps({ from: { type: String, default: '' }, to: { type: String, default: '' }, label: { type: String, default: '申报日期' } })
const emit = defineEmits(['update:from', 'update:to', 'change'])
const root = ref(null), trigger = ref(null), panel = ref(null), open = ref(false), panelStyle = ref({})
const draftFrom = ref(''), draftTo = ref(''), hover = ref('')
const dateKey = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
const today = dateKey(new Date()), baseMonth = ref(new Date(new Date().getFullYear(), new Date().getMonth(), 1))
const weekdays = ['一', '二', '三', '四', '五', '六', '日']
const months = computed(() => [0, 1].map(offset => {
  const date = new Date(baseMonth.value.getFullYear(), baseMonth.value.getMonth() + offset, 1)
  const count = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate()
  const days = Array.from({ length: (date.getDay() + 6) % 7 }, () => null)
  for (let i = 1; i <= count; i++) days.push({ label: i, value: dateKey(new Date(date.getFullYear(), date.getMonth(), i)) })
  while (days.length < 42) days.push(null)
  return { key: dateKey(date), label: `${date.getFullYear()} 年 ${date.getMonth() + 1} 月`, days }
}))
function select(value) {
  if (!draftFrom.value || draftTo.value) { draftFrom.value = value; draftTo.value = ''; return }
  ;[draftFrom.value, draftTo.value] = [draftFrom.value, value].sort()
  hover.value = ''
}
function inRange(value) { const end = draftTo.value || hover.value; return !!draftFrom.value && !!end && value >= [draftFrom.value, end].sort()[0] && value <= [draftFrom.value, end].sort()[1] }
function shift(offset) { baseMonth.value = new Date(baseMonth.value.getFullYear(), baseMonth.value.getMonth() + offset, 1) }
function quick(kind) {
  const now = new Date(), year = now.getFullYear(), month = now.getMonth()
  const start = new Date(year, kind === 'year' ? 0 : month - (kind === 'last' ? 1 : 0), 1)
  const end = kind === 'year' ? new Date(year, 11, 31) : new Date(start.getFullYear(), start.getMonth() + 1, 0)
  draftFrom.value = dateKey(start); draftTo.value = dateKey(end); baseMonth.value = start
}
function position() {
  if (!open.value || !trigger.value) return
  const rect = trigger.value.getBoundingClientRect(), width = Math.min(660, innerWidth - 24), height = Math.min(panel.value?.offsetHeight || 440, innerHeight - 24)
  panelStyle.value = { width: `${width}px`, left: `${Math.max(12, Math.min(rect.left, innerWidth - width - 12))}px`, top: `${Math.max(12, Math.min(rect.bottom + 6, innerHeight - height - 12))}px`, maxHeight: `${innerHeight - 24}px` }
}
async function toggle() {
  if (open.value) { close(); return }
  draftFrom.value = props.from; draftTo.value = props.to; hover.value = ''
  const initial = props.from || today
  baseMonth.value = new Date(Number(initial.slice(0, 4)), Number(initial.slice(5, 7)) - 1, 1)
  open.value = true; await nextTick(); position(); panel.value?.querySelector('button')?.focus({ preventScroll: true })
}
function close() { open.value = false; trigger.value?.focus({ preventScroll: true }) }
function apply() { emit('update:from', draftFrom.value); emit('update:to', draftTo.value); emit('change'); close() }
function outside(event) { if (open.value && !root.value?.contains(event.target) && !panel.value?.contains(event.target)) close() }
function onPanelKey(event) {
  if (event.key === 'Escape') { event.preventDefault(); close() }
  if (event.key !== 'Tab') return
  const buttons = [...panel.value.querySelectorAll('button:not(:disabled)')], first = buttons[0], last = buttons.at(-1)
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
onMounted(() => { document.addEventListener('pointerdown', outside); window.addEventListener('resize', position); window.addEventListener('scroll', position, true) })
onBeforeUnmount(() => { document.removeEventListener('pointerdown', outside); window.removeEventListener('resize', position); window.removeEventListener('scroll', position, true) })
</script>

<style scoped>
.range-trigger { display:flex; align-items:center; gap:10px; height:34px; padding:0 12px; border:1px solid #dce4e7; border-radius:4px; background:#fff; color:#596b7e; font:inherit; font-size:12px; cursor:pointer; }
.range-trigger strong { font-weight:500; color:#33485b; font-variant-numeric:tabular-nums; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; } .range-trigger svg { display:block; width:16px; height:16px; flex:0 0 16px; stroke:currentColor; stroke-width:1.5; stroke-linecap:round; stroke-linejoin:round; }
.range-trigger .range-chevron { margin-left:auto; transition:transform .15s; transform-origin:center; } .range-chevron.open { transform:rotate(180deg); }
.range-panel { position:fixed; z-index:1100; box-sizing:border-box; overflow:auto; background:white; border:1px solid #d4dfe7; border-radius:6px; box-shadow:0 8px 28px #25344724; color:#33485b; font-size:12px; }
.range-panel header { display:flex; align-items:center; justify-content:space-between; padding:10px 16px; border-bottom:1px solid #e8edf2; }
.range-panel button { border:0; background:transparent; color:inherit; cursor:pointer; font:inherit; border-radius:4px; height:30px; padding:0 10px; } .range-panel button:hover { background:#edf4f9; } .range-panel button:disabled { opacity:.4; cursor:default; }
.quick-options { display:flex; gap:8px; padding:10px 16px 0; } .quick-options button { color:#285d83; background:#f3f6f9; }
.calendars { display:grid; grid-template-columns:1fr 1fr; padding:12px 16px; gap:24px; } .calendar { min-width:0; } .month-nav { display:flex; align-items:center; justify-content:space-between; margin-bottom:8px; } .month-nav strong { font-weight:600; } .month-nav button { font-size:20px; }
.days { display:grid; grid-template-columns:repeat(7,minmax(0,1fr)); row-gap:3px; text-align:center; font-variant-numeric:tabular-nums; } .days button { padding:0; height:32px; border-radius:2px; } .weekday { padding:6px 0; color:#8996a3; } .days .selected { background:#edf4fa; color:#285d83; } .days .endpoint, .days .endpoint:hover { background:#285d83; color:white; } .days .today { box-shadow:inset 0 0 0 1px #92b2c9; }
.range-panel footer { display:flex; align-items:center; gap:8px; padding:12px 16px; border-top:1px solid #e8edf2; } footer>span { margin-right:auto; color:#718091; font-variant-numeric:tabular-nums; } .range-panel .apply { background:#285d83; color:#fff; }
button:focus-visible { outline:2px solid #285d83; outline-offset:2px; }
@media(max-width:560px) { .calendars { grid-template-columns:1fr; gap:16px; } .date-range { max-width:100%; min-width:0; } .range-trigger { max-width:100%; gap:6px; } .range-trigger strong { font-size:11px; } footer { flex-wrap:wrap; } footer>span { flex-basis:100%; } }
</style>
