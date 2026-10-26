<template>
  <details class="transaction-cell">
    <summary title="展开或收起交易数量明细" aria-label="展开或收起交易数量明细">
      <span class="summary-values">
        <span v-if="row.transaction_count == null" class="pending">待导入</span>
        <span v-else>{{ count(row.transaction_count) }}笔</span>
        <small class="breakdown-preview">销量 {{ count(row.sales_volume) }}件</small>
      </span>
      <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </summary>
    <div class="transaction-detail">
      <dl>
        <div><dt>销量</dt><dd>{{ count(row.sales_volume) }}件</dd></div>
        <div><dt>样品数</dt><dd>{{ count(row.sample_count) }}件</dd></div>
      </dl>
    </div>
  </details>
</template>

<script setup>
defineProps({ row: { type: Object, required: true } })
const count = value => Number(value || 0).toLocaleString('zh-CN')
</script>

<style scoped>
.transaction-cell { position: relative; width: 100%; min-width: 120px; font-variant-numeric: tabular-nums; }
summary { display: flex; min-width: 0; align-items: center; justify-content: flex-end; gap: 7px; padding: 4px 0; cursor: pointer; list-style: none; }
summary::-webkit-details-marker { display: none; }
summary:hover { color: #235c84; }
summary:focus-visible { outline: 2px solid #235c84; outline-offset: 3px; }
.summary-values { display: inline-flex; flex-direction: column; align-items: flex-end; line-height: 1.25; }
.breakdown-preview { color: #8a96a3; font-size: 10px; font-weight: 400; white-space: nowrap; }
summary svg { flex: 0 0 14px; width: 14px; height: 14px; color: #7c8997; transition: transform 160ms ease; }
.transaction-cell[open] summary svg { transform: rotate(180deg); }
.transaction-detail { position: absolute; top: calc(100% + 5px); right: 0; z-index: 20; width: 150px; box-sizing: border-box; padding: 9px 10px; border: 1px solid #dfe5ec; border-radius: 6px; background: #fff; box-shadow: 0 10px 24px rgb(20 35 50 / 14%); font-size: 11px; text-align: left; }
dl { display: grid; gap: 6px; margin: 0; }
dl > div { display: flex; justify-content: space-between; gap: 16px; }
dt { color: #64748b; }
dd { margin: 0; color: #465669; white-space: nowrap; }
.pending { color: #926a28; }
</style>
