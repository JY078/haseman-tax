<template>
  <Modal v-model:open="open" :title="`${product?.sku || ''} · 价格历史`" size="xl">
    <div class="history-toolbar">
      <div class="range-tabs"><button v-for="option in ranges" :key="option.value" type="button" :class="{ chosen: range === option.value }" @click="range = option.value; hoverIndex = null">{{ option.label }}</button></div>
      <div class="picker-group"><div class="picker"><button class="picker-button" type="button" @click="pickerOpen = pickerOpen === 'year' ? '' : 'year'">{{ year }}年<i class="picker-caret" /></button><div v-if="pickerOpen === 'year'" class="picker-menu"><button v-for="item in years" :key="item" type="button" @click="year = item; pickerOpen = ''; hoverIndex = null">{{ item }}年</button></div></div><div v-if="range === 'month'" class="picker"><button class="picker-button" type="button" @click="pickerOpen = pickerOpen === 'month' ? '' : 'month'">{{ monthNumber }}月<i class="picker-caret" /></button><div v-if="pickerOpen === 'month'" class="picker-menu month-menu"><button v-for="item in 12" :key="item" type="button" @click="monthNumber = item; pickerOpen = ''; hoverIndex = null">{{ item }}月</button></div></div></div>
      <span class="history-product-name">{{ product?.name || product?.spu || product?.sku }}</span>
    </div>
    <div v-if="loading" class="history-empty">正在加载历史价格…</div>
    <div v-else-if="!chartItems.length" class="history-empty">该时间范围暂无历史价格</div>
    <div v-else class="chart-shell">
      <svg class="price-chart" viewBox="0 0 960 320" role="img" aria-label="商品每日价格趋势图，可使用左右方向键查看" tabindex="0" @pointermove="hoverChart" @pointerleave="hoverIndex = null" @keydown.left.prevent="stepHover(-1)" @keydown.right.prevent="stepHover(1)">
        <g v-for="tick in yTicks" :key="tick.value"><line x1="64" :y1="tick.y" x2="930" :y2="tick.y" class="grid-line" /><text x="54" :y="tick.y + 4" text-anchor="end" class="axis-label">{{ money(tick.value) }}</text></g>
        <path v-for="series in chartSeries" :key="`${series.key}-area`" :d="series.area" :class="`area area-${series.key}`" /><path v-for="series in chartSeries" :key="series.key" :d="series.line" :class="`line line-${series.key}`" />
        <text v-for="tick in dateTicks" :key="tick.index" :x="point(tick.index).x" y="292" text-anchor="middle" class="axis-label">{{ tick.label }}</text>
        <g v-if="hovered"><line :x1="point(hovered.index).x" :x2="point(hovered.index).x" y1="20" y2="260" class="hover-line" /><circle v-for="value in hovered.values" :key="value.key" :cx="point(hovered.index).x" :cy="chartY(value.value)" r="4.5" :class="`point-${value.key}`" stroke="#fff" stroke-width="2" /><g :transform="`translate(${Math.max(70, Math.min(745, point(hovered.index).x - 92))}, 4)`"><rect width="190" :height="32 + hovered.values.length * 18" rx="6" class="tooltip-box" /><text x="95" y="23" text-anchor="middle" class="tooltip-date">{{ hovered.date }}</text><text v-for="(value, index) in hovered.values" :key="value.key" x="14" :y="47 + index * 18" class="tooltip-value">{{ value.label }} {{ money(value.value) }}</text></g></g>
      </svg>
    </div>
    <div class="history-legend"><span><i class="regular-dot" />Regular Price</span><span><i class="prime-dot" />Prime Member</span><span><i class="deal-dot" />Deal Price</span></div>
  </Modal>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import request from '@/utils/request'

const props = defineProps({ product: { type: Object, default: null } })
const open = defineModel('open', { type: Boolean, default: false })
const today = new Date()
const range = ref('month'); const year = ref(String(today.getFullYear())); const monthNumber = ref(today.getMonth() + 1); const pickerOpen = ref(''); const allItems = ref([]); const loading = ref(false); const hoverIndex = ref(null)
const ranges = [{ value: 'month', label: '月度' }, { value: 'year', label: '年度' }]
const fields = [{ key: 'regular', label: 'Regular Price', field: 'amazon_price' }, { key: 'prime', label: 'Prime Member', field: 'amazon_sale_price' }, { key: 'deal', label: 'Deal Price', field: 'amazon_best_deal' }]
const years = computed(() => [...new Set([year.value, ...allItems.value.map(item => item.price_date.slice(0, 4))])].sort().reverse())
const month = computed(() => `${year.value}-${String(monthNumber.value).padStart(2, '0')}`)
const chartItems = computed(() => allItems.value.filter(item => item.price_date.startsWith(range.value === 'month' ? month.value : year.value)).sort((a, b) => a.price_date.localeCompare(b.price_date)))
// 每天固定绘制三条价格系列；当天没有记录的价格按 0 展示，避免面积图断裂或只剩孤立点。
const points = computed(() => chartItems.value.map(item => ({ date: item.price_date.slice(0, 10), values: fields.map(field => ({ key: field.key, label: field.label, value: item[field.field] == null ? 0 : Number(item[field.field]) })) })))
const values = computed(() => points.value.flatMap(point => point.values.map(value => value.value)))
const chartMin = computed(() => values.value.length ? Math.max(0, Math.min(...values.value) - Math.max(1, (Math.max(...values.value) - Math.min(...values.value)) * 0.15)) : 0)
const chartMax = computed(() => values.value.length ? Math.max(...values.value) + Math.max(1, (Math.max(...values.value) - Math.min(...values.value)) * 0.15) : 1)
const chartSpan = computed(() => Math.max(1, chartMax.value - chartMin.value))
const yTicks = computed(() => Array.from({ length: 5 }, (_, index) => { const value = chartMax.value - chartSpan.value * index / 4; return { value, y: 20 + index * 60 } }))
function chartY(value) { return 260 - (value - chartMin.value) / chartSpan.value * 240 }
const timestamp = item => Date.parse(`${item.date}T00:00:00Z`)
function point(index) { const rows = points.value; if (!rows.length) return { x: 497, y: 140 }; const span = timestamp(rows[rows.length - 1]) - timestamp(rows[0]); return { x: span ? 64 + (timestamp(rows[index]) - timestamp(rows[0])) / span * 866 : 497, y: chartY(rows[index].values[0]?.value || chartMin.value) } }
const linePointsFor = field => points.value.map((item, index) => { const value = item.values.find(itemValue => itemValue.key === field.key).value; return `${point(index).x},${chartY(value)}` })
const chartSeries = computed(() => fields.map(field => { const pts = linePointsFor(field); if (!pts.length) return { key: field.key, line: '', area: '' }; const line = `M ${pts.join(' L ')}`; const first = pts[0].split(','); const last = pts.at(-1).split(','); return { key: field.key, line, area: `M ${first[0]} 260 L ${pts.join(' L ')} L ${last[0]} 260 Z` } }))
const dateTicks = computed(() => [...new Set(Array.from({ length: Math.min(5, points.value.length) }, (_, index) => Math.round(index * (points.value.length - 1) / Math.max(1, Math.min(5, points.value.length) - 1))))].map(index => ({ index, label: range.value === 'year' ? points.value[index].date.slice(5) : points.value[index].date.slice(8) })))
const hovered = computed(() => { const index = Math.min(hoverIndex.value ?? points.value.length - 1, points.value.length - 1); return index >= 0 ? { index, date: points.value[index].date, values: points.value[index].values } : null })
function money(value) { return value == null ? '—' : `$${Number(value).toFixed(2)}` }
function hoverChart(event) { const svg = event.currentTarget; const pt = svg.createSVGPoint(); pt.x = event.clientX; pt.y = event.clientY; const x = pt.matrixTransform(svg.getScreenCTM().inverse()).x; let nearest = 0; for (let index = 1; index < points.value.length; index++) if (Math.abs(point(index).x - x) < Math.abs(point(nearest).x - x)) nearest = index; hoverIndex.value = nearest }
function stepHover(delta) { hoverIndex.value = Math.max(0, Math.min(points.value.length - 1, (hoverIndex.value ?? points.value.length - 1) + delta)) }
async function load() { if (!open.value || !props.product?.id) return; loading.value = true; try { const from = `${today.getFullYear() - 5}-01-01`; const to = `${today.getFullYear() + 1}-01-01`; allItems.value = (await request.get('/product-prices/history', { params: { product_sku_id: props.product.id, from, to } })).data.items || []; const latest = [...allItems.value].sort((a, b) => b.price_date.localeCompare(a.price_date))[0]; if (latest) { year.value = latest.price_date.slice(0, 4); monthNumber.value = Number(latest.price_date.slice(5, 7)) } } finally { loading.value = false } }
watch(() => [open.value, props.product?.id], load)
</script>

<style scoped>
.history-toolbar{display:flex;align-items:center;gap:12px;margin-bottom:18px;flex-wrap:wrap}.range-tabs{display:flex;gap:4px}.range-tabs button{height:30px;padding:0 12px;border:1px solid transparent;border-radius:6px;background:transparent;color:#627166;font-size:12px;cursor:pointer}.range-tabs button.chosen{border-color:#c8ddd0;background:#edf7f0;color:#286346;font-weight:600}.picker-group{display:flex;gap:6px}.picker{position:relative}.picker-button{display:inline-flex;align-items:center;gap:7px;height:30px;padding:0 11px;border:1px solid #d7e0d9;border-radius:6px;background:#fff;color:#496257;font-size:12px}.picker-caret{width:7px;height:7px;border-right:1.5px solid #627b6b;border-bottom:1.5px solid #627b6b;transform:rotate(45deg)}.picker-menu{position:absolute;z-index:5;top:36px;left:0;display:grid;grid-template-columns:repeat(2,1fr);min-width:112px;padding:5px;border:1px solid #d9e3dc;border-radius:8px;background:#fff;box-shadow:0 10px 24px rgb(28 55 35 / 13%)}.picker-menu button{height:30px;padding:0 10px;border:0;border-radius:5px;background:transparent;color:#4d6255;font-size:12px;cursor:pointer}.picker-menu button:hover{background:#edf7f0;color:#286346}.month-menu{grid-template-columns:repeat(3,1fr);min-width:150px}.history-product-name{margin-left:auto;max-width:260px;overflow:hidden;color:#738078;font-size:13px;text-overflow:ellipsis;white-space:nowrap}.chart-shell{overflow:auto;border:1px solid #e2e9e3;border-radius:10px;background:#fbfdfb}.price-chart{display:block;width:100%;min-width:680px;height:auto;padding:10px;outline:none}.grid-line{stroke:#e3ebe5;stroke-width:1}.axis-label{fill:#7b887e;font-size:11px}.hover-line{stroke:#9bb6a4;stroke-dasharray:4 4}.area{opacity:.12}.area-regular{fill:#356b4b}.area-prime{fill:#b4863e}.area-deal{fill:#c45e29}.line{fill:none;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}.line-regular{stroke:#356b4b}.line-prime{stroke:#b4863e}.line-deal{stroke:#c45e29}.point-regular{fill:#356b4b}.point-prime{fill:#b4863e}.point-deal{fill:#c45e29}.tooltip-box{fill:#edf7f0;stroke:#c8ddd0}.tooltip-date{fill:#315b43;font-size:12px;font-weight:600}.tooltip-value{fill:#496257;font-size:12px;font-variant-numeric:tabular-nums}.history-empty{padding:56px 0;color:#8a968d;text-align:center}.history-legend{display:flex;gap:16px;margin-top:14px;color:#7d887f;font-size:11px}.history-legend i{display:inline-block;width:7px;height:7px;border-radius:50%;margin-right:5px}.regular-dot{background:#356b4b}.prime-dot{background:#b4863e}.deal-dot{background:#c45e29}
</style>
