<template>
  <details v-if="canExpand" class="commission-cell breakdown-cell">
    <summary title="展开或收起平台佣金明细" aria-label="展开或收起平台佣金明细">
      <span class="summary-values"><span v-if="row.commission == null" class="pending">待换算</span><span v-else>{{ money(row.commission) }}</span><small v-if="detailTotal != null" class="breakdown-preview">明细合计 {{ money(detailTotal) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="commission-detail"><dl><div><dt>平台佣金</dt><dd>{{ value(row.settlement_commission_referral) }}</dd></div><div><dt>物流相关佣金</dt><dd>{{ value(row.settlement_commission_shipping) }}</dd></div><div><dt>活动服务费</dt><dd>{{ value(row.settlement_commission_campaign) }}</dd></div><div><dt>退货运费</dt><dd>{{ value(row.settlement_commission_return_shipping) }}</dd></div><div><dt>FBT客户运费抵扣调整</dt><dd>{{ value(row.settlement_commission_fbt_shipping_offset) }}</dd></div><div><dt>FBT配送费赔偿调整</dt><dd>{{ value(row.settlement_commission_fbt_reimbursement) }}</dd></div><div><dt>平台佣金退款</dt><dd>{{ value(row.settlement_commission_refund) }}</dd></div></dl></div>
  </details>
  <span v-else class="commission-plain"><span class="summary-values"><span v-if="row.commission == null" class="pending">待换算</span><span v-else>{{ money(row.commission) }}</span><small v-if="detailTotal != null" class="breakdown-preview">明细合计 {{ money(detailTotal) }}</small></span></span>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const value = value => value == null ? '待换算' : money(value)
const detailTotal = computed(() => props.row.settlement_commission_total ?? props.row.settlement_commission)
const canExpand = computed(() => [props.row.settlement_commission_referral, props.row.settlement_commission_shipping, props.row.settlement_commission_campaign, props.row.settlement_commission_return_shipping, props.row.settlement_commission_fbt_shipping_offset, props.row.settlement_commission_fbt_reimbursement, props.row.settlement_commission_refund].filter(item => item != null).length > 1)
</script>
<style scoped>
.commission-cell { position: relative; width: 100%; min-width: 180px; max-width: 240px; margin-left:auto; font-variant-numeric: tabular-nums; }
.commission-plain { display:block; min-width:180px; text-align:right; font-variant-numeric:tabular-nums; }
summary { display:flex; min-width:0; align-items:center; justify-content:flex-end; gap:7px; padding:4px 0; overflow:hidden; cursor:pointer; list-style:none; border-radius:3px; }
summary::-webkit-details-marker { display:none; } summary:hover { color:#235c84; } summary:focus-visible { outline:2px solid #235c84; outline-offset:3px; }
.summary-values { display:inline-flex; flex-direction:column; align-items:flex-end; line-height:1.25; }
.breakdown-preview { color:#8a96a3; font-size:10px; font-weight:400; white-space:nowrap; }
summary svg { flex:0 0 14px; width:14px; height:14px; color:#7c8997; transition:transform 160ms ease; } .commission-cell[open] summary svg { transform:rotate(180deg); }
.commission-detail { position:absolute; top:calc(100% + 5px); right:0; z-index:20; width:236px; max-width:236px; box-sizing:border-box; margin-top:0; padding:9px 10px; border:1px solid #dfe5ec; border-radius:6px; background:#fff; box-shadow:0 10px 24px rgb(20 35 50 / 14%); font-size:11px; text-align:left; }
dl { display:grid; grid-template-columns:max-content max-content; gap:6px 16px; margin:0; } dl > div { display:contents; } dt { color:#64748b; white-space:nowrap; } dd { min-width:0; max-width:126px; overflow-wrap:anywhere; text-align:right; margin:0; color:#465669; white-space:nowrap; }
.pending { color:#926a28; }
</style>
