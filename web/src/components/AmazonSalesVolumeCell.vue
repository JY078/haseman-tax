<template>
  <button v-if="row.sales_categories?.length" ref="anchor" class="volume-trigger" :class="{ active: open }" :aria-expanded="open" aria-label="查看商品分类销量" @click="toggle">
    <span>{{ quantity(row.sales_volume) }}件</span>
    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
  </button>
  <span v-else class="volume-plain">{{ quantity(row.sales_volume) }}件</span>
  <Teleport to="body">
    <section v-if="open" ref="panel" class="amazon-volume-popover" :style="position" aria-label="商品分类销量明细">
      <dl v-if="row.sales_categories?.length"><div v-for="item in row.sales_categories" :key="item.category"><dt>{{ item.category }}</dt><dd>{{ quantity(item.quantity) }}件</dd></div></dl>
      <p v-else>暂无货件数据</p>
    </section>
  </Teleport>
</template>
<script setup>
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
defineProps({ row: { type: Object, required: true } })
const anchor=ref(null), panel=ref(null), open=ref(false), position=ref({})
const quantity=value=>Number(value||0).toLocaleString('zh-CN',{maximumFractionDigits:0})
async function toggle() {
  open.value=!open.value
  if(!open.value)return
  document.dispatchEvent(new CustomEvent('amazon-volume-open',{detail:anchor.value}))
  const rect=anchor.value.getBoundingClientRect(), width=Math.min(320,window.innerWidth-24)
  position.value={width:`${width}px`,left:`${Math.max(12,Math.min(rect.right-width,window.innerWidth-width-12))}px`,top:`${rect.bottom+8}px`}
  await nextTick()
  if(!panel.value)return
  const height=panel.value.getBoundingClientRect().height
  if(rect.bottom+height+20>window.innerHeight) position.value={...position.value,top:`${Math.max(12,rect.top-height-8)}px`}
}
function outside(event){if(!anchor.value?.contains(event.target)&&!panel.value?.contains(event.target))open.value=false}
function keydown(event){if(event.key==='Escape'&&open.value){open.value=false;anchor.value?.focus()}}
function scroll(event){if(!panel.value?.contains(event.target))open.value=false}
function close(){open.value=false}
function otherOpened(event){if(event.detail!==anchor.value)close()}
onMounted(()=>{document.addEventListener('click',outside);document.addEventListener('keydown',keydown);document.addEventListener('amazon-volume-open',otherOpened);window.addEventListener('scroll',scroll,true);window.addEventListener('resize',close)})
onBeforeUnmount(()=>{document.removeEventListener('click',outside);document.removeEventListener('keydown',keydown);document.removeEventListener('amazon-volume-open',otherOpened);window.removeEventListener('scroll',scroll,true);window.removeEventListener('resize',close)})
</script>
<style scoped>
.volume-trigger{display:inline-flex;align-items:center;justify-content:flex-end;gap:7px;max-width:100%;padding:4px 0;border:0;background:transparent;color:#455961;font:inherit;font-variant-numeric:tabular-nums;cursor:pointer}
.volume-plain{display:inline-block;font-variant-numeric:tabular-nums}
.volume-trigger:hover,.volume-trigger.active{color:#285d83}.volume-trigger svg{width:14px;height:14px;flex-shrink:0;color:#7c8997;transition:transform .15s}.volume-trigger.active svg{transform:rotate(180deg)}
 .amazon-volume-popover{position:fixed;z-index:1100;box-sizing:border-box;min-width:220px;padding:9px 10px;border:1px solid #dfe5ec;border-radius:6px;background:#fff;box-shadow:0 10px 24px rgb(20 35 50 / 14%);color:#465669;font-size:11px;text-align:left}
dl{display:grid;gap:6px;margin:0;max-height:min(300px,50vh);overflow:auto}dl>div{display:flex;justify-content:space-between;gap:16px}dt{color:#64748b;white-space:normal;overflow-wrap:anywhere}dd{margin:0;color:#465669;white-space:nowrap;font-variant-numeric:tabular-nums}p{margin:0;color:#89949b}
button:focus-visible{outline:2px solid #285d83;outline-offset:3px}
</style>
