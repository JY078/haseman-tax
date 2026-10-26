<template>
  <div class="reference-uploader" tabindex="0" @click="focusPasteTarget" @paste="handlePaste">
    <div v-if="modelValue.length" class="reference-grid">
      <article v-for="(image, index) in modelValue" :key="`${image.url}-${index}`" class="reference-item">
        <div class="reference-preview">
          <img :src="displayUrl(image.url)" :alt="image.note || '参考图'" />
          <div v-if="image.uploading" class="reference-loading">
            <ArrowPathIcon class="h-5 w-5 animate-spin" />
          </div>
          <button type="button" class="reference-remove" title="移除图片" @click="remove(index)">
            <XMarkIcon class="h-4 w-4" />
          </button>
        </div>
        <input
          v-if="requireNote"
          :value="image.note"
          type="text"
          maxlength="100"
          placeholder="填写这张图的备注"
          @input="updateNote(index, $event.target.value)"
        />
      </article>
    </div>

    <label class="upload-trigger" :class="{ disabled: uploading }" title="支持粘贴截图">
      <PhotoIcon class="h-5 w-5" />
      <span>{{ uploading ? '正在上传...' : buttonText }}</span>
      <input type="file" accept="image/jpeg,image/png,image/webp" multiple :disabled="uploading" @change="handleFiles" />
    </label>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ArrowPathIcon, PhotoIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import request, { getDisplayThumbnailUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  requireNote: { type: Boolean, default: false },
  buttonText: { type: String, default: '添加参考图' },
})
const emit = defineEmits(['update:modelValue'])
const toast = useToast()
const uploading = ref(false)

function displayUrl(url) {
  return getDisplayThumbnailUrl(url, { width: 280, height: 180, quality: 82, fit: 'lfit' })
}

function updateNote(index, note) {
  const next = props.modelValue.map((item, itemIndex) => itemIndex === index ? { ...item, note } : item)
  emit('update:modelValue', next)
}

function remove(index) {
  emit('update:modelValue', props.modelValue.filter((_, itemIndex) => itemIndex !== index))
}

function focusPasteTarget(event) {
  if (event.target.closest('.reference-remove, .reference-item input')) return
  event.currentTarget.focus()
}

async function uploadFiles(files) {
  if (!files.length) return
  const allowed = new Set(['image/jpeg', 'image/png', 'image/webp'])
  const invalid = files.find(file => !allowed.has(file.type) || file.size > 30 * 1024 * 1024)
  if (invalid) {
    toast.error(!allowed.has(invalid.type) ? '仅支持 JPG、PNG、WebP 图片' : `${invalid.name} 超过 30MB`)
    return
  }

  uploading.value = true
  try {
    const uploaded = []
    for (const file of files) {
      const form = new FormData()
      form.append('image', file)
      const response = await request.post('/work-requests/reference-images', form)
      uploaded.push({ url: response.data.url, note: '' })
    }
    emit('update:modelValue', [...props.modelValue, ...uploaded])
  } catch (error) {
    toast.error(error?.response?.data?.error || '参考图上传失败')
  } finally {
    uploading.value = false
  }
}

async function handleFiles(event) {
  const files = [...(event.target.files || [])]
  event.target.value = ''
  await uploadFiles(files)
}

async function handlePaste(event) {
  const items = [...(event.clipboardData?.items || [])]
  const files = items
    .filter(item => item.kind === 'file' && item.type.startsWith('image/'))
    .map(item => item.getAsFile())
    .filter(Boolean)
  if (!files.length) return
  event.preventDefault()
  await uploadFiles(files)
}
</script>

<style scoped>
.reference-uploader { display: flex; width: 100%; flex-direction: column; gap: 10px; outline: none; }
.reference-uploader:focus-visible { border-radius: 6px; box-shadow: 0 0 0 2px #736c63; }
.reference-grid { display: grid; width: 100%; grid-template-columns: repeat(auto-fill, minmax(148px, 1fr)); gap: 12px; }
.reference-item { min-width: 0; }
.reference-preview { position: relative; overflow: hidden; width: 100%; aspect-ratio: 4 / 3; border: 1px solid #e5e7eb; border-radius: 6px; background: #f5f5f5; }
.reference-preview img { width: 100%; height: 100%; object-fit: cover; }
.reference-loading { position: absolute; inset: 0; display: grid; place-items: center; color: #fff; background: rgb(23 23 23 / 55%); }
.reference-remove { position: absolute; top: 7px; right: 7px; display: grid; width: 27px; height: 27px; place-items: center; border-radius: 6px; color: #fff; background: rgb(23 23 23 / 72%); }
.reference-remove:hover { background: #171717; }
.reference-item input { width: 100%; height: 36px; margin-top: 7px; border: 1px solid #d4d4d4; border-radius: 6px; padding: 0 10px; font-size: 13px; outline: none; }
.reference-item input:focus { border-color: #737373; box-shadow: 0 0 0 3px rgb(23 23 23 / 7%); }
.upload-trigger { display: flex; width: 100%; height: 44px; cursor: pointer; align-items: center; justify-content: center; gap: 7px; border: 1px dashed #bdbdbd; border-radius: 6px; color: #525252; background: #fafafa; font-size: 13px; font-weight: 650; }
.upload-trigger:hover { border-color: #737373; color: #171717; background: #fff; }
.upload-trigger.disabled { cursor: wait; opacity: .6; }
.upload-trigger input { position: absolute; width: 1px; height: 1px; overflow: hidden; opacity: 0; }
.reference-preview { border-color: #dfdcd7; background: #f3f1ed; }.reference-preview img { transition: transform .25s ease; }.reference-preview:hover img { transform: scale(1.015); }.reference-remove { background: rgb(43 41 38 / 78%); transition: background .16s ease, transform .12s ease; }.reference-remove:active { transform: translateY(1px); }.reference-item input { border-color: #d8d5cf; color: #353330; }.reference-item input:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }.upload-trigger { border-color: #bdb8b0; color: #625e58; background: #faf9f7; transition: border-color .16s ease, background .16s ease, color .16s ease; }.upload-trigger:hover { border-color: #8f8981; color: #292826; background: #fff; }.upload-trigger:focus-within { outline: 2px solid #736c63; outline-offset: 2px; }
</style>
