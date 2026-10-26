<template>
  <Modal :open="!!product" title="添加 TikTok 映射" :dismissible="!saving" @update:open="close">
    <form id="price-mapping-form" class="mapping-form" @submit.prevent="save">
      <label>Amazon SKU<input :value="product?.sku" readonly /></label>
      <label>TikTok SKU<input v-model.trim="form.sku" required maxlength="160" :disabled="saving" @input="error = ''" /></label>
      <label>非亚马逊 SKU <span>选填</span><input v-model.trim="form.nonAmazon" maxlength="160" :disabled="saving" /></label>
      <label>备注 <span>选填</span><input v-model.trim="form.remark" maxlength="1000" :disabled="saving" /></label>
      <p v-if="error" class="mapping-error" role="alert">{{ error }}</p>
    </form>
    <template #footer><Button variant="outline" :disabled="saving" @click="close">取消</Button><Button type="submit" form="price-mapping-form" :disabled="saving">{{ saving ? '保存中…' : '保存映射' }}</Button></template>
  </Modal>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import Button from '@/components/ui/Button.vue'
import request from '@/utils/request'

const props = defineProps({ product: { type: Object, default: null } })
const emit = defineEmits(['close', 'saved'])
const form = reactive({ sku: '', nonAmazon: '', remark: '' })
const saving = ref(false)
const error = ref('')
watch(() => props.product, () => {
  Object.assign(form, { sku: '', nonAmazon: '', remark: '' })
  error.value = ''
})
function close() { if (!saving.value) emit('close') }
async function save() {
  if (saving.value || !form.sku || !props.product || props.product.mapping_id) return
  saving.value = true
  error.value = ''
  try {
    const { data } = await request.get('/order-sku-mappings/group', { params: { platform: 'TIKTOK', platform_sku: form.sku } })
    if (data.items.length) {
      error.value = '此 TK SKU 已有映射，请在订单转 MCF 中维护。'
      return
    }
    await request.put('/order-sku-mappings', {
      platform: 'TIKTOK', platform_sku: form.sku, amazon_sku: props.product.sku,
      non_amazon_sku: form.nonAmazon,
      remark: form.remark, is_current: true, only_unmapped: true,
    })
    emit('saved')
  } catch (err) { error.value = err.response?.data?.error || '映射保存失败，请重试。' }
  finally { saving.value = false }
}
</script>

<style scoped>
.mapping-form { display: grid; gap: 18px; }.mapping-form label { font-size: 13px; color: #46534b; }.mapping-form label > span { float: right; color: #879188; font-size: 11px; }.mapping-form input { width: 100%; margin-top: 7px; height: 40px; padding: 0 12px; border: 1px solid #dce4dc; border-radius: 5px; outline: none; }.mapping-form input:focus { border-color: #568368; box-shadow: 0 0 0 3px #eef4ef; }.mapping-form input[readonly] { background: #f5f7f5; color: #78857c; }.mapping-error { color: #b36547; font-size: 12px; }
</style>
