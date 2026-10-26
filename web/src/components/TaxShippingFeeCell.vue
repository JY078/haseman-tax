<template>
  <details class="shipping-cell">
    <summary title="展开或收起运费明细" aria-label="展开或收起运费明细">
      <span class="summary-values"><span v-if="row.shipping_fee == null" class="pending">待换算</span><span v-else>{{ money(row.shipping_fee) }}</span><small v-if="row.settlement_shipping_fee != null" class="breakdown-preview">明细合计 {{ money(row.settlement_shipping_fee) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="shipping-detail"><dl><div><dt>Customer shipping fee offset</dt><dd>{{ money(row.shipping_customer_offset) }}</dd></div><div><dt>Platform shipping fee discount</dt><dd>{{ money(row.shipping_platform_discount) }}</dd></div><div><dt>Customer shipping fee</dt><dd>{{ money(row.shipping_customer) }}</dd></div><div><dt>Refunded customer shipping fee</dt><dd>{{ money(row.shipping_refunded_customer) }}</dd></div></dl></div>
  </details>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => value == null ? '待换算' : Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
</script>
<style scoped>
.shipping-cell { position:relative; font-variant-numeric:tabular-nums; }
summary { display:flex; align-items:center; justify-content:flex-end; gap:7px; padding:4px 0; cursor:pointer; list-style:none; }
summary::-webkit-details-marker { display:none; }
summary:hover { color:#235c84; }
summary svg { width:14px; height:14px; color:#7c8997; }
.summary-values { display:inline-flex; flex-direction:column; align-items:flex-end; line-height:1.25; }
.breakdown-preview { color:#8a96a3; font-size:10px; white-space:nowrap; }
.shipping-detail { position:absolute; top:calc(100% + 5px); right:0; z-index:20; min-width:260px; padding:9px 10px; border:1px solid #dfe5ec; border-radius:6px; background:#fff; box-shadow:0 10px 24px rgb(20 35 50 / 14%); font-size:11px; text-align:left; }
dl { display:grid; gap:6px; margin:0; }
dl > div { display:flex; justify-content:space-between; gap:14px; }
dt { color:#64748b; }
dd { margin:0; color:#465669; white-space:nowrap; }
.pending { color:#926a28; }
</style>
