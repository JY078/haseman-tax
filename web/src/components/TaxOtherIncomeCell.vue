<template>
  <details class="other-income-cell breakdown-cell">
    <summary title="展开或收起其他收入明细" aria-label="展开或收起其他收入明细">
      <span class="summary-values"><span v-if="row.other_income == null" class="pending">待换算</span><span v-else>{{ money(row.other_income) }}</span><small class="breakdown-preview">明细合计 {{ money(row.settlement_other_income || 0) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="other-income-detail"><dl>
      <div><dt>FBT配送费赔偿</dt><dd>{{ value(row.other_income_fbt_reimbursement) }}</dd></div>
      <div><dt>退货运费赔偿</dt><dd>{{ value(row.other_income_return_shipping) }}</dd></div>
      <div><dt>配送保障赔付</dt><dd>{{ value(row.other_income_shipping_protection) }}</dd></div>
      <div><dt>FBT全体商家补贴</dt><dd>{{ value(row.other_income_fbt_subsidy) }}</dd></div>
      <div><dt>其他收入</dt><dd>{{ value(row.other_income_settlement) }}</dd></div>
    </dl></div>
  </details>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const value = value => value == null ? '待换算' : money(value)
</script>
<style scoped>
.other-income-cell { position: relative; min-width: 180px; font-variant-numeric: tabular-nums; }
.summary-values { display:inline-flex; flex-direction:column; align-items:flex-end; line-height:1.25; }
.breakdown-preview { color:#8a96a3; font-size:10px; font-weight:400; white-space:nowrap; }
summary { display:flex; align-items:center; justify-content:flex-end; gap:7px; padding:4px 0; cursor:pointer; list-style:none; border-radius:3px; }
summary::-webkit-details-marker { display:none; } summary:hover { color:#235c84; } summary:focus-visible { outline:2px solid #235c84; outline-offset:3px; }
summary svg { width:14px; height:14px; color:#7c8997; transition:transform 160ms ease; } .other-income-cell[open] summary svg { transform:rotate(180deg); }
.other-income-detail { min-width:250px; position:absolute; top:calc(100% + 5px); right:0; z-index:20; margin-top:0; padding:9px 10px; border:1px solid #dfe5ec; border-radius:6px; background:#fff; box-shadow:0 10px 24px rgb(20 35 50 / 14%); font-size:11px; text-align:left; }
dl { display:grid; gap:6px; margin:0; } dl > div { display:flex; justify-content:space-between; gap:16px; } dt { color:#64748b; white-space:nowrap; } dd { margin:0; color:#465669; white-space:nowrap; }
.pending { color:#926a28; }
</style>
