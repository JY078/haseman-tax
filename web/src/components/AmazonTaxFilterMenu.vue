<template>
  <span class="amazon-filter">{{ label }}<button ref="trigger" class="filter-trigger" :class="{ active: selected }" :aria-label="`筛选${label}`" :aria-expanded="open" @click.stop="toggle">⌕</button></span>
  <Teleport to="body">
    <div v-if="open" class="amazon-filter-menu" :style="position" @click.stop>
      <div class="popover-title">筛选{{ label }}</div>
      <button class="filter-option" :class="{ active: !selected }" @click="$emit('toggle','')">全部</button>
      <button v-for="value in values" :key="value" class="filter-option" :class="{ active: selectedValues.includes(value) }" @click="$emit('toggle',value)"><span class="filter-check">{{ selectedValues.includes(value) ? '✓' : '' }}</span>{{ value }}</button>
      <div v-if="!values.length" class="filter-loading">暂无可选值</div>
    </div>
  </Teleport>
</template>
<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
const props=defineProps({label:String,values:{type:Array,default:()=>[]},selected:String})
defineEmits(['toggle'])
const open=ref(false), trigger=ref(null), position=ref({})
const selectedValues=computed(()=>String(props.selected||'').split(',').filter(Boolean))
function close(){open.value=false}
function toggle(){
  const wasOpen=open.value
  document.dispatchEvent(new CustomEvent('amazon-close-filters'))
  if(wasOpen)return
  const rect=trigger.value.getBoundingClientRect()
  position.value={top:`${rect.bottom+6}px`,left:`${Math.max(8,Math.min(rect.left,window.innerWidth-220))}px`,maxHeight:`${Math.max(100,Math.min(320,window.innerHeight-rect.bottom-18))}px`}
  open.value=true
}
function scroll(event){if(!event.target.closest?.('.amazon-filter-menu'))close()}
function key(event){if(event.key==='Escape'){close();trigger.value?.focus()}}
onMounted(()=>{document.addEventListener('amazon-close-filters',close);window.addEventListener('resize',close);window.addEventListener('scroll',scroll,true);document.addEventListener('keydown',key)})
onBeforeUnmount(()=>{document.removeEventListener('amazon-close-filters',close);window.removeEventListener('resize',close);window.removeEventListener('scroll',scroll,true);document.removeEventListener('keydown',key)})
</script>
<style scoped>
.amazon-filter{display:inline-flex;align-items:center;white-space:nowrap;gap:3px}
.filter-trigger{height:28px;border:0;background:transparent;color:#7c8b9a;padding:0 3px;font-size:14px;cursor:pointer}
.filter-trigger.active{color:#285d83;background:#eaf2f8;border-radius:4px}
.amazon-filter-menu{position:fixed;z-index:10001;display:flex;flex-direction:column;gap:3px;width:190px;overflow-y:auto;padding:7px;background:#fff;border:1px solid #d7e0e8;border-radius:8px;box-shadow:0 12px 30px rgb(20 35 50 / 17%);font-family:inherit}
.popover-title{padding:6px 8px 7px;border-bottom:1px solid #edf1f5;color:#506174;font-size:11px;font-weight:700}
.filter-option{flex-shrink:0;display:flex;align-items:center;width:100%;height:30px;padding:0 9px;border:0;border-radius:5px;color:#526274;background:transparent;text-align:left;font-size:12px;cursor:pointer}
.filter-option:hover{color:#285d83;background:#f1f6fa}.filter-option.active{color:#285d83;background:#eaf2f8;font-weight:600}
.filter-check{display:inline-flex;width:16px;height:16px;flex:0 0 16px;align-items:center;justify-content:center;margin-right:8px;border:1px solid #bac8d6;border-radius:3px;color:#235c84}
.filter-loading{padding:9px 8px;color:#8997a4;font-size:11px}
button:focus-visible{outline:2px solid #285d83;outline-offset:1px}
</style>
