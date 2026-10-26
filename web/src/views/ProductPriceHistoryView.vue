<template>
  <MainLayout><div class="history-page">
    <header class="history-header"><div><h1>价格历史</h1><p>按天查看美国站 SKU 的 Amazon 价格快照</p></div><router-link class="back-link" to="/product-prices">返回商品价格</router-link></header>
    <section class="history-surface"><div class="history-toolbar"><label>月份 <input v-model="month" type="month" @change="load" /></label><input v-model.trim="keyword" placeholder="搜索 SKU 或商品" @input="load" /></div>
      <div v-if="loading" class="history-state">正在加载历史价格…</div><div v-else-if="!days.length" class="history-state">该月份暂无历史价格</div>
      <div v-else class="calendar"><div class="calendar-head"><span>SKU / 商品</span><span v-for="day in days" :key="day" :class="{ today: day === today }">{{ day.slice(8) }}</span></div><div v-for="row in rows" :key="row.sku" class="calendar-row"><div class="history-product"><ProductThumbnail :src="row.image_url" :alt="row.sku" /><div><strong>{{ row.name || row.sku }}</strong><small>{{ row.sku }}</small></div></div><div v-for="day in days" :key="day" class="day-cell" :class="{ empty: !row.dates[day] }"> <template v-if="row.dates[day]"><b>{{ money(row.dates[day].amazon_price) }}</b><small v-if="row.dates[day].amazon_sale_price">促 {{ money(row.dates[day].amazon_sale_price) }}</small></template><span v-else>—</span></div></div></div>
    </section>
  </div></MainLayout>
</template>
<script setup>
import { computed, onMounted, ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import ProductThumbnail from '@/components/ui/ProductThumbnail.vue'
import request from '@/utils/request'
const now = new Date(); const month = ref(`${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}`); const keyword = ref(''); const loading = ref(false); const items = ref([])
const today = `${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}-${String(now.getDate()).padStart(2,'0')}`
const days = computed(() => { const [y,m] = month.value.split('-').map(Number); const count = new Date(y,m,0).getDate(); return Array.from({length:count},(_,i)=>`${month.value}-${String(i+1).padStart(2,'0')}`) })
const rows = computed(() => { const grouped = new Map(); for (const item of items.value) { if (keyword.value && !`${item.sku} ${item.name}`.toLowerCase().includes(keyword.value.toLowerCase())) continue; if (!grouped.has(item.sku)) grouped.set(item.sku,{...item,dates:{}}); grouped.get(item.sku).dates[item.price_date.slice(0,10)] = item } return [...grouped.values()] })
function money(value) { return value == null ? '—' : `$${Number(value).toFixed(2)}` }
async function load() { loading.value=true; try { const [y,m] = month.value.split('-'); const from=`${month.value}-01`; const to=`${month.value}-${String(new Date(Number(y),Number(m),0).getDate()).padStart(2,'0')}`; items.value=(await request.get('/product-prices/history',{params:{from,to}})).data.items||[] } finally { loading.value=false } }
onMounted(load)
</script>
<style scoped>
.history-page{padding:28px;max-width:1800px;margin:auto}.history-header{display:flex;justify-content:space-between;align-items:end;margin-bottom:20px}.history-header h1{font-size:26px;color:#17211b}.history-header p{color:#7b877e;margin-top:5px}.back-link,.history-link{color:#276749;font-weight:650}.history-surface{background:#fff;border:1px solid #e5ebe6;border-radius:14px;overflow:auto;box-shadow:0 8px 30px #173b2410}.history-toolbar{display:flex;gap:14px;padding:16px;border-bottom:1px solid #edf1ee}.history-toolbar label,.history-toolbar input{height:36px;border:1px solid #d8e0da;border-radius:8px;padding:0 10px;color:#34463a}.calendar{min-width:1250px}.calendar-head,.calendar-row{display:grid;grid-template-columns:220px repeat(31,minmax(44px,1fr));}.calendar-head{position:sticky;top:0;background:#f6f9f6;color:#708074;font-size:11px;font-weight:700}.calendar-head span{padding:10px 4px;text-align:center;border-right:1px solid #edf1ee}.calendar-head span:first-child{text-align:left;padding-left:16px}.calendar-head .today{color:#217346;background:#e4f2e7}.calendar-row{min-height:76px;border-top:1px solid #edf1ee}.history-product{display:flex;align-items:center;gap:9px;padding:10px 12px;position:sticky;left:0;background:#fff;z-index:1}.history-product strong,.history-product small{display:block;max-width:150px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.history-product small{color:#849188;font-size:11px;margin-top:3px}.day-cell{display:flex;flex-direction:column;align-items:center;justify-content:center;border-right:1px solid #f0f3f1;font-size:11px;color:#34463a}.day-cell b{font-size:12px}.day-cell small{color:#c56b29;margin-top:3px}.day-cell.empty{color:#cbd4ce}.history-state{padding:80px;text-align:center;color:#87938a}
</style>
