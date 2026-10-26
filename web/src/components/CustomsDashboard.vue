<template>
  <section class="dashboard" aria-label="报关单 BI 看板" :aria-busy="loading">
    <header class="dashboard-heading"><div><h2>报关单 BI 看板</h2><p>当前筛选范围 · 全部报关单 · 按申报日期统计</p></div><span class="scope">{{ loading ? '正在汇总…' : '全量汇总' }}</span></header>
    <p v-if="error" role="alert" class="notice error">{{ error }} <button @click="$emit('retry')">重试</button></p>
    <div v-else-if="loading" class="metrics skeleton" role="status" aria-label="正在加载看板"><div v-for="n in 6" :key="n"><span></span><strong></strong></div></div>
    <template v-else-if="summary">
      <div class="metrics">
        <div><span>报关单数</span><strong>{{ summary.total }}<small>份</small></strong></div>
        <div><span>商品明细</span><strong>{{ summary.item_count }}<small>项</small></strong></div>
        <div><span>件数合计</span><strong>{{ customsMeasure(summary.packages, 0) }}<small>件</small></strong><em v-if="summary.packages.missing">{{ summary.packages.missing }} 份未计入</em></div>
        <div><span>毛重合计</span><strong>{{ customsMeasure(summary.gross_weight, 3) }}<small>kg</small></strong><em v-if="summary.gross_weight.missing">{{ summary.gross_weight.missing }} 份未计入</em></div>
        <div><span>净重合计</span><strong>{{ customsMeasure(summary.net_weight, 3) }}<small>kg</small></strong><em v-if="summary.net_weight.missing">{{ summary.net_weight.missing }} 份未计入</em></div>
        <div :class="{ review: summary.review_count }"><span>待核对报关单</span><strong>{{ summary.review_count }}<small>份</small></strong><em>含识别提示或无法汇总的数据</em></div>
      </div>
      <p v-if="!summary.total" class="empty">当前筛选范围暂无报关单，上传 PDF 或调整筛选条件后查看汇总。</p>
      <template v-else>
        <div class="analysis-grid">
          <section class="amount-panel"><div class="panel-heading"><h3>金额汇总</h3><span>按原币分列</span></div>
            <div class="amount-scroll" tabindex="0" aria-label="按币种汇总金额"><table><thead><tr><th>币制</th><th>商品总价</th><th>运费</th><th>保费</th></tr></thead><tbody><tr v-for="amount in summary.amounts" :key="amount.currency"><th>{{ amount.currency }}</th><td>{{ customsMeasure(amount.goods) }}</td><td>{{ customsMeasure(amount.freight) }}</td><td>{{ customsMeasure(amount.insurance) }}</td></tr><tr v-if="!summary.amounts.length"><td colspan="4" class="empty">暂无可汇总金额</td></tr></tbody></table></div>
            <p class="footnote">运费、保费仅汇总标记为“总价”的金额，按每份报关单计一次。</p>
            <p v-if="summary.unaggregated_amounts" class="amount-warning">{{ summary.unaggregated_amounts }} 项金额缺失或币种、计价方式不明确，未计入汇总。</p>
          </section>
          <section class="chart-panel"><div class="panel-heading"><h3>月度申报趋势</h3><span>份 / 月</span></div><div class="bars"><div v-for="entry in summary.months" :key="entry.name" class="bar-row"><span>{{ entry.name }}</span><div class="bar-track"><i :style="{ width: barWidth(entry.count, summary.months) }"></i></div><strong>{{ entry.count }}</strong></div></div></section>
          <section class="chart-panel"><div class="panel-heading"><h3>指运 / 经停港分布</h3><span>报关单数 · TOP 6</span></div><div class="bars"><div v-for="entry in summary.destinations.slice(0, 6)" :key="entry.name" class="bar-row"><span :title="entry.name">{{ entry.name }}</span><div class="bar-track"><i :style="{ width: barWidth(entry.count, summary.destinations) }"></i></div><strong>{{ entry.count }}</strong></div></div></section>
        </div>
      </template>
    </template>
  </section>
</template>

<script setup>
import { customsMeasure } from '@/utils/customsDeclarations'
defineProps({ summary: { type: Object, default: null }, loading: Boolean, error: { type: String, default: '' } })
defineEmits(['retry'])
const barWidth = (count, rows) => `${count / Math.max(1, ...rows.map(row => row.count)) * 100}%`
</script>

<style scoped>
.dashboard { margin-bottom:16px; color:#253447; }
.dashboard-heading,.panel-heading { display:flex; align-items:center; justify-content:space-between; gap:12px; }
h2,h3,p { margin:0; } h2 { font-size:16px; font-weight:600; } h3 { font-size:13px; font-weight:600; }
.dashboard-heading { margin-bottom:14px; } .dashboard-heading p { margin-top:5px; font-size:12px; color:#7b8997; }
.scope { padding:5px 9px; border:1px solid #dbe5ec; border-radius:4px; color:#627d91; font-size:11px; white-space:nowrap; }
.metrics { display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); border:1px solid #e3e8ef; border-radius:4px; background:#fff; overflow:hidden; }
.metrics>div { padding:14px 16px; border-right:1px solid #e8edf2; min-width:0; } .metrics>div:last-child { border:0; }
.metrics span { font-size:12px; color:#718091; } .metrics strong { display:block; margin-top:10px; font-size:23px; line-height:1.3; font-weight:600; font-variant-numeric:tabular-nums; overflow-wrap:anywhere; }
.metrics small { margin-left:7px; font-size:11px; font-weight:400; color:#8794a0; } .metrics em { display:block; font-style:normal; margin-top:6px; font-size:10px; color:#8794a0; }
.metrics .review strong,.metrics .review em { color:#986a25; }
.analysis-grid { display:grid; grid-template-columns:1.35fr 1fr 1fr; gap:12px; margin-top:12px; }
.analysis-grid>section { min-width:0; border:1px solid #e3e8ef; border-radius:4px; background:#fff; padding:16px; }
.panel-heading { margin-bottom:15px; } .panel-heading>span { font-size:10px; color:#8794a0; }
.amount-scroll { overflow:auto; } table { width:100%; border-collapse:collapse; white-space:nowrap; font-size:12px; } th,td { padding:9px 10px; text-align:right; border-bottom:1px solid #e8edf2; font-variant-numeric:tabular-nums; } th { font-weight:500; color:#718091; background:#f4f6f9; } th:first-child { text-align:left; } tbody th { background:transparent; font-weight:600; color:#285d83; }
.footnote,.amount-warning { margin-top:10px; color:#8794a0; font-size:11px; line-height:1.6; } .amount-warning { color:#986a25; }
.bars { max-height:180px; overflow:auto; } .bar-row { display:grid; grid-template-columns:84px minmax(20px,1fr) 28px; gap:10px; align-items:center; margin:12px 0; font-size:11px; } .bar-row>span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:#6a7d8c; } .bar-row strong { text-align:right; font-weight:500; font-variant-numeric:tabular-nums; } .bar-track { height:7px; background:#edf2f6; border-radius:2px; } .bar-track i { display:block; height:100%; background:#6b91ac; border-radius:2px; }
.empty { padding:24px; font-size:12px; color:#8794a0; text-align:center; } .error { padding:16px; font-size:13px; color:#9d3c3c; background:#fff1f1; } button { margin-left:8px; background:white; border:1px solid #dce4e7; border-radius:4px; cursor:pointer; padding:4px 10px; }
.skeleton span,.skeleton strong { display:block; border-radius:3px; background:#edf2f6; height:12px; width:60%; } .skeleton strong { height:29px; width:85%; }
@media(max-width:1100px) { .metrics { grid-template-columns:repeat(3,minmax(0,1fr)); } .metrics>div:nth-child(-n+3) { border-bottom:1px solid #e8edf2; } .metrics>div:nth-child(3n) { border-right:0; } .analysis-grid { grid-template-columns:1fr 1fr; } .amount-panel { grid-column:1/-1; } }
@media(max-width:600px) { .metrics { grid-template-columns:repeat(2,minmax(0,1fr)); } .metrics>div { padding:16px 12px; border-bottom:1px solid #e8edf2; } .metrics>div:nth-child(3) { border-right:1px solid #e8edf2; } .metrics>div:nth-child(2n) { border-right:0; } .metrics strong { font-size:22px; } .analysis-grid { grid-template-columns:1fr; } }
</style>
