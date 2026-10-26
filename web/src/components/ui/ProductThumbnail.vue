<script setup>
import { ref, watch } from 'vue'
import { PhotoIcon } from '@heroicons/vue/24/outline'
import Modal from './Modal.vue'
import { resolveImageUrl } from '@/utils/request'

const props = defineProps({ src: { type: String, default: '' }, alt: { type: String, default: '产品图片' } })
const failed = ref(false)
const open = ref(false)
watch(() => props.src, () => { failed.value = false; open.value = false })
</script>

<template>
  <button v-if="src && !failed" type="button" class="product-thumbnail" :aria-label="`查看${alt}`" title="查看大图" @click.stop="open = true">
    <img :src="resolveImageUrl(src)" :alt="alt" loading="lazy" @error="failed = true" />
  </button>
  <span v-else class="product-thumbnail placeholder" :title="failed ? '图片加载失败' : '暂无图片'" :aria-label="failed ? '图片加载失败' : '暂无图片'"><PhotoIcon /></span>
  <Modal v-model:open="open" :title="alt" size="lg"><img class="product-full-image" :src="resolveImageUrl(src)" :alt="alt" /></Modal>
</template>

<style scoped>
.product-thumbnail { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 48px; width: 48px; height: 48px; overflow: hidden; border: 1px solid #e5e7eb; border-radius: 8px; background: #fff; padding: 4px; }
button.product-thumbnail:hover { border-color: #9ca3af; }
button.product-thumbnail:focus-visible { outline: 2px solid #475569; outline-offset: 2px; }
.product-thumbnail img { width: 100%; height: 100%; object-fit: contain; }
.placeholder { background: #f8fafc; color: #b4bdc9; }
.placeholder svg { width: 22px; height: 22px; }
.product-full-image { display: block; width: 100%; max-height: 70vh; object-fit: contain; }
</style>
