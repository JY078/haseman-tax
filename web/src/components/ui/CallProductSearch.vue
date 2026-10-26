<template>
  <div class="product-search" @focusout="leave">
    <label :for="inputId">SKU</label>
    <input :id="inputId" :value="modelValue" :disabled="disabled" maxlength="200" autocomplete="off" role="combobox" aria-autocomplete="list" :aria-expanded="open" :aria-controls="`${inputId}-options`" :aria-activedescendant="open && active >= 0 ? `${inputId}-${active}` : undefined" :placeholder="en ? 'Search SKU or ASIN' : '搜索 SKU 或 ASIN'" @input="input" @focus="search(modelValue)" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)" @keydown.enter="confirm" @keydown.esc="escape" />
    <div v-if="open" class="suggestions">
      <p v-if="busy" role="status">{{ en ? 'Searching…' : '正在搜索…' }}</p>
      <p v-else-if="failed" role="status">{{ en ? 'Search failed. Type again to retry.' : '搜索失败，请重新输入重试' }}</p>
      <p v-else-if="!items.length" role="status">{{ en ? 'No matches. You can enter a SKU manually.' : '未找到匹配产品，可手动填写 SKU' }}</p>
      <ul :id="`${inputId}-options`" role="listbox" :aria-label="en ? 'Product matches' : '匹配产品'">
        <li v-for="(item, index) in items" :id="`${inputId}-${index}`" :key="item.id" role="option" :aria-selected="active === index" :class="{ active: active === index }" @pointerdown.prevent @click="choose(item)"><strong>{{ item.sku || '-' }}</strong><span>ASIN · {{ item.asin || '-' }}</span></li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import request from '@/utils/request'
const props = defineProps({ modelValue: { type: String, default: '' }, disabled: Boolean, locale: { type: String, default: 'zh' }, inputId: { type: String, default: 'call-product-sku' } })
const emit = defineEmits(['update:modelValue', 'select'])
const en = computed(() => props.locale === 'en')
const open = ref(false), busy = ref(false), failed = ref(false), items = ref([]), active = ref(-1)
let timer, version = 0
function close() { clearTimeout(timer); version++; open.value = false; items.value = []; active.value = -1; busy.value = false }
function search(value) {
  close()
  if (!value.trim() || props.disabled) return
  open.value = true; busy.value = true; failed.value = false
  const current = version
  timer = setTimeout(async () => {
    try {
      const { data } = await request.get('/product-skus', { params: { q: value.trim() } })
      if (current !== version) return
      items.value = Array.isArray(data) ? data : []
    } catch { if (current === version) failed.value = true }
    finally { if (current === version) busy.value = false }
  }, 250)
}
function input(event) { emit('update:modelValue', event.target.value); search(event.target.value) }
function choose(item) { emit('update:modelValue', item.sku || ''); emit('select', item); close() }
function move(delta) { if (!open.value) { search(props.modelValue); return } if (items.value.length) { active.value = (active.value + delta + items.value.length) % items.value.length; document.getElementById(`${props.inputId}-${active.value}`)?.scrollIntoView({ block: 'nearest' }) } }
function confirm(event) { if (!open.value) return; event.preventDefault(); if (active.value >= 0) choose(items.value[active.value]) }
function escape(event) { if (open.value) { event.preventDefault(); event.stopPropagation(); close() } }
function leave(event) { if (!event.currentTarget.contains(event.relatedTarget)) close() }
onBeforeUnmount(close)
</script>

<style scoped>
.product-search { position: relative; min-width: 0; }
label { display: block; font-size: 12px; color: #555; }
input { display: block; width: 100%; margin-top: 8px; border: 1px solid #d5d9d6; border-radius: 5px; padding: 10px 11px; background: white; color: #282828; font-size: 13px; line-height: 1.6; }
input:focus { outline: 2px solid #bdd4ca; border-color: #608b7c; outline-offset: 1px; }
.suggestions { position: absolute; top: 100%; left: 0; right: 0; z-index: 10; margin-top: 5px; max-height: 240px; overflow-y: auto; border: 1px solid #d5d9d6; border-radius: 6px; background: white; box-shadow: 0 8px 24px rgb(0 0 0 / 12%); }
.suggestions p { padding: 12px; color: #777; font-size: 12px; }
li { padding: 10px 12px; cursor: pointer; overflow-wrap: anywhere; }
li:hover, li.active { background: #edf3ef; }
li strong { display: block; font-size: 13px; color: #245e4e; }
li span { display: block; margin-top: 3px; font-size: 12px; color: #666; }
</style>
