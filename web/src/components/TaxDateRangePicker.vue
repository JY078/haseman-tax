<template>
  <div ref="root" class="tax-period-picker">
    <button ref="trigger" type="button" class="tax-period-trigger" :class="{ open, active: from || to }" :aria-expanded="open" aria-haspopup="dialog" :aria-label="`${label}：${triggerText}`" @click="toggle">
      <span class="period-field-label">{{ label }}</span><strong>{{ triggerText }}</strong>
      <svg class="period-chevron" :class="{ open }" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </button>
    <Teleport to="body">
      <Transition name="period-panel">
        <section v-if="open" ref="panel" class="tax-period-panel" :style="panelStyle" role="dialog" aria-label="选择报税日期范围" @mouseleave="hoverPeriod = ''">
          <header class="panel-head"><div><strong>选择范围</strong><p>按季度或月份选择，确认后生效</p></div><button type="button" class="panel-close" aria-label="关闭范围选择器" @click="close(true)">×</button></header>
          <div class="panel-body">
            <div class="period-mode" role="group" aria-label="期间类型">
              <button ref="firstControl" type="button" :aria-pressed="mode === 'quarter'" :class="{ active: mode === 'quarter' }" @click="changeMode('quarter')">按季度</button>
              <button type="button" :aria-pressed="mode === 'month'" :class="{ active: mode === 'month' }" @click="changeMode('month')">按月份</button>
            </div>
            <div class="quick-row" role="group" aria-label="快捷期间"><span>快捷选择</span><button v-for="option in quickOptions" :key="option.key" type="button" :aria-pressed="quickActive(option.key)" :class="{ active: quickActive(option.key) }" @click="quickSelect(option.key)">{{ option.label }}</button></div>
            <div class="year-nav">
              <button type="button" class="year-arrow" :disabled="viewYear <= minYear" aria-label="上一年" @click="shiftYear(-1)">‹</button>
              <select v-model.number="viewYear" aria-label="选择年份" @change="hoverPeriod = ''"><option v-for="year in years" :key="year" :value="year">{{ year }} 年{{ year === currentYear ? ' · 今年' : '' }}</option></select>
              <button type="button" class="year-arrow" :disabled="viewYear >= maxYear" aria-label="下一年" @click="shiftYear(1)">›</button>
            </div>
            <div class="period-grid" :class="mode === 'quarter' ? 'quarters' : 'months'" role="group" :aria-label="`${viewYear} 年${periodUnit}`">
              <button v-for="period in visiblePeriods" :key="period.value" type="button" class="period-cell" :class="cellClass(period.value)" :aria-pressed="!!cellClass(period.value).selected" :aria-label="`${periodLabel(period.value)}${isToday(period.value) ? '，当前期间' : ''}`" @click="selectPeriod(period.value)" @mouseenter="hoverPeriod = period.value" @focus="hoverPeriod = period.value" @blur="hoverPeriod = ''">
                <b>{{ period.label }}</b><small v-if="period.hint">{{ period.hint }}</small>
              </button>
            </div>
            <p class="selection-hint" aria-live="polite">{{ footText }}</p>
            <div class="range-preview"><div><span>开始日期</span><strong>{{ draftRange.from || '不限' }}</strong></div><span class="range-separator" aria-hidden="true">→</span><div><span>结束日期</span><strong>{{ draftRange.to || '不限' }}</strong></div></div>
          </div>
          <footer class="panel-foot"><button type="button" class="clear-button" @click="clear">重置</button><div><button type="button" class="cancel-button" @click="close(true)">取消</button><button type="button" class="apply-button" @click="apply">应用范围</button></div></footer>
        </section>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
const props = defineProps({ from: { type: String, default: '' }, to: { type: String, default: '' }, label: { type: String, default: '范围' } })
const emit = defineEmits(['update:from', 'update:to', 'change'])
const root = ref(null), trigger = ref(null), panel = ref(null), firstControl = ref(null), panelStyle = ref({})
const open = ref(false), mode = ref('quarter'), startPeriod = ref(''), endPeriod = ref(''), hoverPeriod = ref(''), draftChanged = ref(false)
const currentYear = new Date().getFullYear()
const minYear = computed(() => Math.min(currentYear - 5, Number((props.from || props.to || String(currentYear)).slice(0, 4))))
const maxYear = computed(() => Math.max(currentYear + 1, Number((props.to || props.from || String(currentYear)).slice(0, 4))))
const years = computed(() => Array.from({ length: maxYear.value - minYear.value + 1 }, (_, index) => minYear.value + index))
const viewYear = ref(currentYear)
const quickOptions = [{ key: 'all', label: '全部' }, { key: 'this', label: '本年' }, { key: 'last', label: '去年' }]
const monthValue = (year, month) => `${year}-${String(month).padStart(2, '0')}`
const periodUnit = computed(() => mode.value === 'quarter' ? '季度' : '月份')
const triggerText = computed(() => !props.from && !props.to ? '全部时间' : `${props.from || '不限'} 至 ${props.to || '不限'}`)
const visiblePeriods = computed(() => Array.from({ length: mode.value === 'quarter' ? 4 : 12 }, (_, index) => {
  const value = index + 1
  return mode.value === 'quarter' ? { value: `${viewYear.value}-Q${value}`, label: `第${['一', '二', '三', '四'][index]}季度`, hint: `${index * 3 + 1}月 — ${value * 3}月` } : { value: monthValue(viewYear.value, value), label: `${value}月` }
}))
const draftRange = computed(() => {
  if (!draftChanged.value) return { from: props.from, to: props.to }
  if (!startPeriod.value) return { from: '', to: '' }
  return { from: periodRange(startPeriod.value).from, to: periodRange(endPeriod.value || startPeriod.value).to }
})
const footText = computed(() => !startPeriod.value ? `选择一个${periodUnit.value}，或连续选择起止期间` : !endPeriod.value ? `已选 ${periodLabel(startPeriod.value)}；可直接应用，或选择结束${periodUnit.value}` : `${periodLabel(startPeriod.value)} 至 ${periodLabel(endPeriod.value)}，可跨年选择`)
function periodLabel(period) { return period.includes('-Q') ? period.replace('-Q', ' 年第 ') + ' 季度' : period.replace('-', ' 年 ') + ' 月' }
function periodRange(period) {
  const [year, value] = period.split(mode.value === 'quarter' ? '-Q' : '-').map(Number)
  const first = mode.value === 'quarter' ? (value - 1) * 3 + 1 : value
  const last = mode.value === 'quarter' ? first + 2 : value
  return { from: `${monthValue(year, first)}-01`, to: `${monthValue(year, last)}-${new Date(year, last, 0).getDate()}` }
}
function periodIndex(period) { const [year, value] = period.split(mode.value === 'quarter' ? '-Q' : '-').map(Number); return year * (mode.value === 'quarter' ? 4 : 12) + value }
function datePeriod(date) { if (!date) return ''; return mode.value === 'quarter' ? `${date.slice(0, 4)}-Q${Math.ceil(Number(date.slice(5, 7)) / 3)}` : date.slice(0, 7) }
function isToday(period) { const now = new Date(); return period === (mode.value === 'quarter' ? `${now.getFullYear()}-Q${Math.floor(now.getMonth() / 3) + 1}` : monthValue(now.getFullYear(), now.getMonth() + 1)) }
function syncDraft() { startPeriod.value = datePeriod(props.from); endPeriod.value = datePeriod(props.to); hoverPeriod.value = ''; draftChanged.value = false; viewYear.value = Number((props.from || props.to || String(currentYear)).slice(0, 4)) }
async function toggle() {
  if (open.value) { close(); return }
  syncDraft(); open.value = true; await nextTick(); positionPanel(); firstControl.value?.focus({ preventScroll: true })
}
function close(restoreFocus = false) { open.value = false; hoverPeriod.value = ''; if (restoreFocus) trigger.value?.focus({ preventScroll: true }) }
function positionPanel() {
  if (!open.value || !trigger.value) return
  const rect = trigger.value.getBoundingClientRect(), width = Math.min(400, window.innerWidth - 24)
  const height = Math.min(panel.value?.offsetHeight || 480, window.innerHeight - 24)
  const top = rect.bottom + 8 + height <= window.innerHeight - 12 ? rect.bottom + 8 : Math.max(12, rect.top - height - 8)
  panelStyle.value = { width: `${width}px`, left: `${Math.max(12, Math.min(rect.left, window.innerWidth - width - 12))}px`, top: `${top}px`, maxHeight: `${window.innerHeight - 24}px` }
}
function shiftYear(offset) { viewYear.value = Math.min(maxYear.value, Math.max(minYear.value, viewYear.value + offset)); hoverPeriod.value = '' }
function changeMode(next) { if (mode.value === next) return; const range = draftRange.value; mode.value = next; startPeriod.value = datePeriod(range.from); endPeriod.value = datePeriod(range.to); hoverPeriod.value = ''; nextTick(positionPanel) }
function selectPeriod(period) {
  draftChanged.value = true
  if (!startPeriod.value || endPeriod.value) { startPeriod.value = period; endPeriod.value = ''; return }
  if (periodIndex(period) < periodIndex(startPeriod.value)) { endPeriod.value = startPeriod.value; startPeriod.value = period } else endPeriod.value = period
  hoverPeriod.value = ''
}
function quickSelect(key) {
  if (key === 'all') { clear(); return }
  const year = key === 'this' ? currentYear : currentYear - 1
  viewYear.value = year; draftChanged.value = true
  startPeriod.value = mode.value === 'quarter' ? `${year}-Q1` : monthValue(year, 1)
  endPeriod.value = mode.value === 'quarter' ? `${year}-Q4` : monthValue(year, 12)
  hoverPeriod.value = ''
}
function quickActive(key) { const range = draftRange.value; if (key === 'all') return !range.from && !range.to; const year = key === 'this' ? currentYear : currentYear - 1; return range.from === `${year}-01-01` && range.to === `${year}-12-31` }
function cellClass(period) {
  const start = startPeriod.value ? periodIndex(startPeriod.value) : null
  const end = endPeriod.value ? periodIndex(endPeriod.value) : start != null && hoverPeriod.value ? periodIndex(hoverPeriod.value) : start
  const index = periodIndex(period)
  return { today: isToday(period), selected: start != null && end != null && index >= Math.min(start, end) && index <= Math.max(start, end), start: period === startPeriod.value, end: period === endPeriod.value, 'preview-end': !endPeriod.value && !!startPeriod.value && hoverPeriod.value === period && period !== startPeriod.value }
}
function clear() { startPeriod.value = ''; endPeriod.value = ''; hoverPeriod.value = ''; draftChanged.value = true }
function apply() { const range = draftRange.value; if (range.from !== props.from || range.to !== props.to) { emit('update:from', range.from); emit('update:to', range.to); emit('change') } close(true) }
function onDocClick(event) { if (open.value && !root.value?.contains(event.target) && !panel.value?.contains(event.target)) close() }
function onDocKeydown(event) { if (event.key === 'Escape' && open.value) { event.preventDefault(); close(true) } }
watch(() => [props.from, props.to], syncDraft)
onMounted(() => { document.addEventListener('click', onDocClick); document.addEventListener('keydown', onDocKeydown); window.addEventListener('resize', positionPanel); window.addEventListener('scroll', positionPanel, true) })
onBeforeUnmount(() => { document.removeEventListener('click', onDocClick); document.removeEventListener('keydown', onDocKeydown); window.removeEventListener('resize', positionPanel); window.removeEventListener('scroll', positionPanel, true) })
</script>

<style scoped>
.tax-period-picker { position:relative; min-width:0; }
.tax-period-trigger { display:inline-flex; align-items:center; gap:12px; min-width:178px; max-width:100%; height:36px; padding:0 12px; border:1px solid #dce4ec; border-radius:6px; background:#fff; color:#253f54; font:inherit; font-size:12px; cursor:pointer; transition:border-color .15s, background .15s; }
.tax-period-trigger:hover, .tax-period-trigger.active { border-color:#8daec4; }
.tax-period-trigger.open { border-color:#285d83; background:#f5f9fc; }
.period-field-label { color:#7b8997; font-weight:400; white-space:nowrap; }
.tax-period-trigger strong { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-weight:600; font-variant-numeric:tabular-nums; }
.period-chevron { flex:0 0 16px; width:16px; height:16px; margin-left:auto; color:#7b8997; transition:transform .15s; }
.period-chevron.open { transform:rotate(180deg); }
.tax-period-panel { position:fixed; z-index:1100; box-sizing:border-box; display:flex; flex-direction:column; overflow:auto; width:400px; max-width:calc(100vw - 24px); border:1px solid #d3dde6; border-radius:10px; background:#fff; box-shadow:0 12px 32px rgb(25 45 65 / 16%); color:#41566a; font-size:12px; line-height:1.5; font-family:inherit; }
.tax-period-panel button, .tax-period-panel select { box-sizing:border-box; font-family:inherit; font-size:12px; cursor:pointer; transition:background .15s, border-color .15s, color .15s; }
.panel-head { display:flex; align-items:flex-start; justify-content:space-between; padding:16px 18px 12px; flex-shrink:0; }
.panel-head strong { color:#253f54; font-size:14px; font-weight:600; }
.panel-head p { margin:3px 0 0; color:#7b8997; font-size:12px; }
.panel-close { width:28px; height:28px; border:0; border-radius:5px; background:transparent; color:#718294; font-size:20px !important; }
.panel-close:hover { background:#edf3f7; color:#285d83; }
.panel-body { padding:0 18px 16px; }
.period-mode { display:flex; gap:3px; padding:3px; border-radius:6px; background:#eef2f6; }
.period-mode button { flex:1; height:32px; border:0; border-radius:4px; background:transparent; color:#66788a; }
.period-mode button.active { color:#285d83; background:#fff; box-shadow:0 1px 3px rgb(30 48 64 / 10%); font-weight:600; }
.quick-row { display:flex; align-items:center; flex-wrap:wrap; gap:6px; margin:12px 0 16px; }
.quick-row>span { margin-right:auto; color:#7b8997; font-size:11px; }
.quick-row button { height:26px; padding:0 10px; border:1px solid #dce4ec; border-radius:4px; background:#fff; color:#596b7e; }
.quick-row button.active { border-color:#9bb8cb; color:#285d83; background:#edf5fa; }
.year-nav { display:flex; align-items:center; justify-content:space-between; margin-bottom:10px; }
.year-nav select { border:0; background:#fff; color:#253f54; font-size:14px; font-weight:600; padding:5px; border-radius:4px; font-variant-numeric:tabular-nums; }
.year-arrow { width:28px; height:28px; border:1px solid #dce4ec; border-radius:5px; background:#fff; color:#596b7e; font-size:20px !important; line-height:1; }
.year-arrow:disabled { opacity:.35; cursor:default; }
.period-grid { display:grid; gap:8px; }
.period-grid.quarters { grid-template-columns:repeat(2,minmax(0,1fr)); }
.period-grid.months { grid-template-columns:repeat(4,minmax(0,1fr)); }
.period-cell { position:relative; display:flex; flex-direction:column; align-items:center; justify-content:center; gap:3px; height:60px; border:1px solid #dce4ec; border-radius:6px; background:#fff; color:#41566a; }
.months .period-cell { height:38px; }
.period-cell b { font-size:13px; font-weight:600; font-variant-numeric:tabular-nums; }
.period-cell small { font-size:11px; color:#7b8997; }
.period-cell:hover, .quick-row button:hover, .year-arrow:hover:not(:disabled) { border-color:#8daec4; background:#f2f7fb; }
.period-cell.selected { background:#edf5fa; border-color:#bcd6e6; color:#285d83; }
.period-cell.start, .period-cell.end { background:#285d83; border-color:#285d83; color:#fff; }
.period-cell.start small, .period-cell.end small { color:#e0ecf4; }
.period-cell.preview-end { border-style:dashed; border-color:#285d83; }
.period-cell.today::after { content:''; position:absolute; top:6px; right:6px; width:4px; height:4px; border-radius:50%; background:currentColor; }
.selection-hint { min-height:34px; margin:12px 0 8px; color:#718294; font-size:11px; line-height:1.6; }
.range-preview { display:grid; grid-template-columns:1fr auto 1fr; align-items:center; gap:12px; padding:10px 12px; border:1px solid #e5ebf0; border-radius:6px; background:#f8fafc; }
.range-preview>div { display:flex; flex-direction:column; gap:4px; }
.range-preview span { color:#7b8997; font-size:11px; }
.range-preview strong { color:#33485b; font-size:12px; font-weight:600; font-variant-numeric:tabular-nums; }
.panel-foot { display:flex; align-items:center; justify-content:space-between; gap:8px; padding:12px 18px; border-top:1px solid #e5ebf0; flex-shrink:0; }
.panel-foot>div { display:flex; gap:8px; }
.panel-foot button { height:32px; padding:0 14px; border-radius:5px; }
.clear-button { border:0; background:transparent; color:#718294; padding-left:0 !important; }
.cancel-button { border:1px solid #dce4ec; background:#fff; color:#596b7e; }
.apply-button { border:1px solid #285d83; background:#285d83; color:#fff; font-weight:600; }
.apply-button:hover { background:#204e70; }
.cancel-button:hover { background:#f2f7fb; }
.clear-button:hover { color:#285d83; }
.tax-period-trigger:focus-visible, .tax-period-panel :is(button,select):focus-visible { outline:2px solid #285d83; outline-offset:2px; }
.period-panel-enter-active, .period-panel-leave-active { transition:opacity .15s, transform .15s; }
.period-panel-enter-from, .period-panel-leave-to { opacity:0; transform:translateY(-4px); }
@media (max-width:520px) { .tax-period-picker { flex:1 1 100%; } .tax-period-trigger { width:100%; } .panel-head,.panel-foot { padding-left:14px; padding-right:14px; } .panel-body { padding-left:14px; padding-right:14px; } }
@media (prefers-reduced-motion:reduce) { *, .period-panel-enter-active, .period-panel-leave-active { transition:none !important; } }
</style>
