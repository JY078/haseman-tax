<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">色系库</h1>
      </div>
    </div>

    <a-card class="mb-4 style-library-card">
      <div class="style-library-toolbar">
        <div>
          <div class="section-title">产品色系</div>
          <div class="section-subtitle">管理产品色系选项与色块图，支持新增、编辑与删除</div>
        </div>
        <a-space>
          <a-button type="primary" @click="openModal()">
            <template #icon><icon-plus /></template>
            新建色系
          </a-button>
        </a-space>
      </div>

      <div v-if="loading" class="color-chip-grid color-chip-grid--loading">
        <div v-for="n in 8" :key="n" class="color-chip color-chip--skeleton" />
      </div>
      <a-empty v-else-if="items.length === 0" description="还没有色系数据，先新建一个吧" />
      <div v-else class="color-chip-grid">
        <div
          v-for="item in items"
          :key="item.id"
          class="color-chip"
        >
          <div class="color-chip__media">
            <img v-if="item.image_url" :src="resolveImageUrl(item.image_url)" class="color-chip__image" alt="色系图" />
            <span v-else class="color-chip__placeholder">{{ firstChar(item.name) }}</span>
          </div>
          <div class="color-chip__body">
            <span class="color-chip__name">{{ item.name }}</span>
            <div class="color-chip__actions" @click.stop>
              <a-button type="text" size="mini" @click="openModal(item)">
                <template #icon><icon-edit /></template>
              </a-button>
              <a-popconfirm content="确定删除该色系吗？" @ok="handleDelete(item.id)">
                <a-button type="text" status="danger" size="mini">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </a-card>

    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑色系' : '新建色系'"
      :ok-loading="submitLoading"
      @ok="handleSubmit"
      @cancel="closeModal"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <a-form-item field="name" label="色系名称" :rules="[{ required: true, message: '请输入色系名称' }]">
          <a-input v-model="form.name" placeholder="例如：拉丝金 / 磨砂银 / 玫瑰铜" />
        </a-form-item>
        <a-form-item field="image_url" label="色系图片（可选）">
          <div class="upload-box">
            <div
              v-if="form.image_url"
              class="preview-wrap"
              @dragover.prevent="isDragOverUpload = true"
              @dragleave.prevent="isDragOverUpload = false"
              @drop.prevent="handleDrop"
            >
              <img :src="resolveImageUrl(form.image_url)" class="preview-image" alt="预览" />
              <div class="preview-overlay">
                <span class="preview-badge">已上传</span>
                <div class="preview-actions">
                  <button class="preview-action" type="button" @click="previewImageUrl = resolveImageUrl(form.image_url)">预览</button>
                  <button class="preview-action" type="button" @click="triggerFileSelect">更换图片</button>
                </div>
              </div>
            </div>
            <button
              v-else
              class="upload-trigger"
              :class="{ 'upload-trigger--dragover': isDragOverUpload }"
              type="button"
              @click="triggerFileSelect"
              @dragover.prevent="isDragOverUpload = true"
              @dragleave.prevent="isDragOverUpload = false"
              @drop.prevent="handleDrop"
            >
              <span class="upload-trigger__icon">+</span>
              <span class="upload-trigger__title">点击上传色系图片</span>
              <span class="upload-trigger__hint">支持 JPG、PNG、WebP，建议上传清晰的色块或材质图</span>
              <span class="upload-trigger__subhint">也可以直接把图片拖到这里</span>
            </button>
            <input ref="fileInputRef" class="upload-input" type="file" accept="image/*" @change="handleFileChange" />
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-image-preview-group>
      <a-modal
        v-model:visible="previewVisible"
        :footer="false"
        :closable="false"
        :simple="true"
        :modal-style="{ background: 'transparent', boxShadow: 'none' }"
        :body-style="{ padding: '0', background: 'transparent' }"
        width="auto"
        @cancel="previewImageUrl = ''"
      >
        <img v-if="previewImageUrl" :src="previewImageUrl" alt="预览" class="preview-dialog-image" />
      </a-modal>
    </a-image-preview-group>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import request, { resolveImageUrl } from '@/utils/request'

const loading = ref(false)
const submitLoading = ref(false)
const items = ref([])
const modalVisible = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const fileInputRef = ref(null)
const previewImageUrl = ref('')
const isDragOverUpload = ref(false)

const form = reactive({ id: null, name: '', image_url: '' })

const previewVisible = computed(() => !!previewImageUrl.value)

async function fetchItems() {
  loading.value = true
  try {
    const res = await request.get('/admin/product-color-families')
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.id = null
  form.name = ''
  form.image_url = ''
  formRef.value?.clearValidate()
}

function openModal(record = null) {
  isEdit.value = !!record
  resetForm()
  if (record) {
    form.id = record.id
    form.name = record.name
    form.image_url = record.image_url || ''
  }
  modalVisible.value = true
}

function closeModal() {
  modalVisible.value = false
  isDragOverUpload.value = false
}

async function handleSubmit() {
  const err = await formRef.value?.validate()
  if (err) return
  submitLoading.value = true
  try {
    const payload = { name: form.name, image_url: form.image_url }
    if (isEdit.value) {
      await request.put(`/admin/product-color-families/${form.id}`, payload)
      Message.success('色系更新成功')
    } else {
      await request.post('/admin/product-color-families', payload)
      Message.success('色系创建成功')
    }
    modalVisible.value = false
    await fetchItems()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id) {
  await request.delete(`/admin/product-color-families/${id}`)
  Message.success('色系已删除')
  await fetchItems()
}

function triggerFileSelect() {
  fileInputRef.value?.click()
}

async function uploadFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const res = await request.post('/upload/image', { data: String(e.target?.result || ''), mime_type: file.type || 'image/png' })
      form.image_url = res.data?.url || ''
      Message.success('图片上传成功')
    } catch {
      Message.error('图片上传失败')
    } finally {
      isDragOverUpload.value = false
      if (fileInputRef.value) fileInputRef.value.value = ''
    }
  }
  reader.readAsDataURL(file)
}

async function handleFileChange(event) {
  const file = event.target.files?.[0]
  await uploadFile(file)
}

async function handleDrop(event) {
  const file = event.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) {
    await uploadFile(file)
  }
}

function firstChar(str) {
  return str ? str.charAt(0).toUpperCase() : '?'
}

onMounted(fetchItems)
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
}

.mb-4 {
  margin-bottom: 16px;
}

.style-library-card {
  border-radius: 20px;
}

.style-library-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.section-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-1);
}

.section-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
}

.color-chip-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.color-chip-grid--loading {
  pointer-events: none;
}

.color-chip {
  display: flex;
  flex-direction: column;
  width: 100%;
  border: 1px solid var(--color-border-2);
  border-radius: 18px;
  background: #fff;
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.06);
  overflow: hidden;
  transition: transform 0.22s ease, box-shadow 0.22s ease, border-color 0.22s ease;
}

.color-chip:hover {
  transform: translateY(-3px);
  border-color: rgba(var(--primary-5), 0.45);
  box-shadow: 0 22px 40px rgba(15, 23, 42, 0.1);
}

.color-chip--skeleton {
  min-height: 240px;
  background: linear-gradient(110deg, #f6f7fb 8%, #eef1f7 18%, #f6f7fb 33%);
  background-size: 200% 100%;
  animation: shimmer 1.1s linear infinite;
}

.color-chip__media {
  width: 100%;
  aspect-ratio: 1 / 1;
  background: #f7f8fa;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.color-chip__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.color-chip__placeholder {
  font-size: 48px;
  font-weight: 700;
  color: var(--color-text-3);
}

.color-chip__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  gap: 8px;
}

.color-chip__name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.color-chip__actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.upload-box {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-input {
  display: none;
}

.upload-trigger {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 200px;
  width: 100%;
  padding: 28px 20px;
  border: 1px dashed rgb(var(--primary-6));
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(var(--primary-1), 0.7) 0%, #fff 100%);
  cursor: pointer;
  transition: all 0.22s ease;
}

.upload-trigger:hover,
.upload-trigger--dragover {
  border-color: rgb(var(--primary-5));
  transform: translateY(-1px);
  box-shadow: 0 16px 34px rgba(15, 23, 42, 0.08);
}

.upload-trigger__icon {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--primary-6));
  color: #fff;
  font-size: 28px;
  line-height: 1;
  box-shadow: 0 10px 24px rgba(var(--primary-6), 0.28);
}

.upload-trigger__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-1);
}

.upload-trigger__hint {
  font-size: 12px;
  color: var(--color-text-3);
}

.upload-trigger__subhint {
  margin-top: 6px;
  font-size: 12px;
  color: rgb(var(--primary-6));
  font-weight: 600;
}

.preview-wrap {
  position: relative;
  width: 100%;
  max-width: 320px;
  border-radius: 18px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: #fff;
  box-shadow: 0 16px 34px rgba(15, 23, 42, 0.08);
}

.preview-image {
  width: 100%;
  aspect-ratio: 1 / 1;
  object-fit: cover;
  display: block;
}

.preview-overlay {
  position: absolute;
  inset: auto 0 0 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0) 0%, rgba(15, 23, 42, 0.72) 100%);
}

.preview-badge {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.18);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  backdrop-filter: blur(6px);
}

.preview-actions {
  display: flex;
  gap: 8px;
}

.preview-action {
  border: none;
  border-radius: 999px;
  padding: 7px 12px;
  background: #fff;
  color: var(--color-text-1);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.preview-action:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.16);
}

.preview-dialog-image {
  width: auto;
  max-width: min(72vw, 920px);
  max-height: 68vh;
  object-fit: contain;
  border-radius: 14px;
  display: block;
  margin: 0 auto;
  background: transparent;
}

@keyframes shimmer {
  to {
    background-position-x: -200%;
  }
}
</style>
