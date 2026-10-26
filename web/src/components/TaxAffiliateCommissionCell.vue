<template>
  <details v-if="canExpand" class="affiliate-cell breakdown-cell">
    <summary title="展开或收起联盟佣金明细" aria-label="展开或收起联盟佣金明细">
      <span class="summary-values"><span>{{ money(row.affiliate_commission || 0) }}</span><small v-if="row.settlement_affiliate_commission != null" class="breakdown-preview">明细合计 {{ money(row.settlement_affiliate_commission) }}</small></span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="affiliate-detail"><dl>
      <div><dt>联盟佣金</dt><dd>{{ value(row.affiliate_direct) }}</dd></div>
      <div><dt>联盟伙伴佣金</dt><dd>{{ value(row.affiliate_partner) }}</dd></div>
      <div><dt>联盟店广告佣金</dt><dd>{{ value(row.affiliate_shop_ads) }}</dd></div>
      <div><dt>联盟佣金预留金额</dt><dd>{{ value(row.affiliate_deposit) }}</dd></div>
      <div><dt>联盟佣金退款</dt><dd>{{ value(row.affiliate_refund) }}</dd></div>
      <div><dt>联盟伙伴广告佣金</dt><dd>{{ value(row.affiliate_partner_shop_ads) }}</dd></div>
    </dl></div>
  </details>
  <span v-else class="affiliate-plain"><span class="summary-values"><span>{{ money(row.affiliate_commission || 0) }}</span><small v-if="row.settlement_affiliate_commission != null" class="breakdown-preview">明细合计 {{ money(row.settlement_affiliate_commission) }}</small></span></span>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const value = value => value == null ? '待换算' : money(value)
const canExpand = computed(() => [props.row.affiliate_direct, props.row.affiliate_partner, props.row.affiliate_shop_ads, props.row.affiliate_deposit, props.row.affiliate_refund, props.row.affiliate_partner_shop_ads].filter(item => item != null).length > 1)
</script>
<style scoped>
.affiliate-cell { position: relative; min-width: 180px; font-variant-numeric: tabular-nums; }
.affiliate-plain { display:block; min-width:180px; text-align:right; font-variant-numeric:tabular-nums; }
.summary-values { display:inline-flex; flex-direction:column; align-items:flex-end; line-height:1.25; }
.breakdown-preview { color:#8a96a3; font-size:10px; font-weight:400; white-space:nowrap; }
summary { display:flex; align-items:center; justify-content:flex-end; gap:7px; padding:4px 0; cursor:pointer; list-style:none; border-radius:3px; }
summary::-webkit-details-marker { display:none; } summary:hover { color:#235c84; } summary:focus-visible { outline:2px solid #235c84; outline-offset:3px; }
summary svg { width:14px; height:14px; color:#7c8997; transition:transform 160ms ease; } .affiliate-cell[open] summary svg { transform:rotate(180deg); }
.affiliate-detail { min-width:250px; position:absolute; top:calc(100% + 5px); right:0; z-index:20; margin-top:0; padding:9px 10px; border:1px solid #dfe5ec; border-radius:6px; background:#fff; box-shadow:0 10px 24px rgb(20 35 50 / 14%); font-size:11px; text-align:left; }
dl { display:grid; gap:6px; margin:0; } dl > div { display:flex; justify-content:space-between; gap:16px; } dt { color:#64748b; white-space:nowrap; } dd { margin:0; color:#465669; white-space:nowrap; }
.pending { color:#926a28; }
</style>
