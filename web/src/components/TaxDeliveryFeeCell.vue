<template>
  <details class="delivery-cell breakdown-cell">
    <summary :aria-label="`${row.quarter} 配送费，展开或收起物流金额`" title="展开或收起物流金额">
      <span class="summary-values"><span v-if="row.delivery == null" class="pending">待换算</span><span v-else>{{ money(row.delivery) }}</span><small v-if="breakdownTotal != null" class="breakdown-preview">明细合计 {{ money(breakdownTotal) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="delivery-detail">
      <dl><div v-for="item in items" :key="item.key"><dt>{{ item.label }}</dt><dd><span v-if="row[item.key] == null" class="pending">待换算</span><template v-else>{{ money(row[item.key]) }}</template></dd></div></dl>
      <p v-if="row.delivery_unmatched_rows" class="unmatched">{{ row.delivery_unmatched_rows }} 条订单待匹配</p>
    </div>
  </details>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const items = [{ key: 'delivery_mcf', label: 'MCF 总计' }, { key: 'delivery_fbt', label: 'FBT 总计' }, { key: 'delivery_overseas', label: '海外仓总计' }]
const breakdownTotal = computed(() => {
  if (items.some(item => props.row[item.key] == null)) return null
  return items.reduce((sum, item) => sum + Number(props.row[item.key] || 0), 0)
})
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
</script>

<style scoped>
.delivery-cell { position: relative; font-variant-numeric: tabular-nums; }
.summary-values { display: inline-flex; flex-direction: column; align-items: flex-end; line-height: 1.25; }
.breakdown-preview { color: #8a96a3; font-size: 10px; font-weight: 400; white-space: nowrap; }
summary { display: flex; align-items: center; justify-content: flex-end; gap: 7px; padding: 4px 0; cursor: pointer; list-style: none; border-radius: 3px; }
summary::-webkit-details-marker { display: none; }
summary:hover { color: #235c84; }
summary:focus-visible { outline: 2px solid #235c84; outline-offset: 3px; }
summary svg { flex: 0 0 14px; width: 14px; height: 14px; color: #7c8997; transition: transform 160ms ease; }
.delivery-cell[open] summary svg { transform: rotate(180deg); }
.delivery-detail { position: absolute; top: calc(100% + 5px); right: 0; z-index: 20; min-width: 210px; box-sizing: border-box; margin-top: 0; padding: 9px 10px; border: 1px solid #dfe5ec; border-radius: 6px; background: #fff; box-shadow: 0 10px 24px rgb(20 35 50 / 14%); font-size: 11px; text-align: left; }
dl { display: grid; gap: 6px; margin: 0; }
dl > div { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
dt { color: #64748b; }
dd { margin: 0; color: #465669; }
.pending, .unmatched { color: #926a28; }
.unmatched { margin: 8px 0 0; font-size: 10px; white-space: normal; }
@media (prefers-reduced-motion: reduce) { summary svg { transition: none; } }
</style>
