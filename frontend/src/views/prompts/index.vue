<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">模板库</h1>
        <p class="page-subtitle">按横版、竖版、方形分类上传模板图片，供前台模板弹窗直接读取</p>
      </div>
    </div>
    <div class="content-layout">
      <aside class="sidebar">
        <div class="sidebar-header">
          <div class="sidebar-title">模板分类</div>
          <div class="sidebar-subtitle">选择分类后管理模板图片</div>
        </div>
        <div class="group-list">
          <button v-for="item in SHAPES" :key="item.value" class="group-item"
            :class="{ active: selectedShape === item.value }" @click="selectShape(item.value)">
            <div class="group-info"><icon-folder class="group-icon" /><span class="group-name">{{ item.label }}</span>
            </div>
            <span class="group-count">{{ shapeCount(item.value) }}</span>
          </button>
        </div>
      </aside>
      <main class="main-content">
        <div class="content-header">
          <div>
            <h2 class="content-title">{{ currentShapeLabel }}</h2>
            <p class="content-subtitle">当前分类 {{ prompts.length }} 个模板</p>
          </div>
          <a-button type="primary" @click="openModal()"><template #icon><icon-plus /></template>添加{{ currentShapeLabel
            }}</a-button>
        </div>
        <div class="prompt-list gallery-list">
          <article v-for="prompt in prompts" :key="prompt.id" class="prompt-card prompt-card--image-only">
            <a-popconfirm content="确定删除该模板？" @ok="removePrompt(prompt.id)">
              <a-button class="prompt-delete-btn" type="text" size="small" status="danger">
                <template #icon><icon-delete /></template>
              </a-button>
            </a-popconfirm>
            <div v-if="prompt.images?.length" class="image-grid image-grid--single">
              <button v-for="(url, idx) in prompt.images" :key="`${prompt.id}-${idx}`"
                class="image-preview-trigger"
                type="button"
                @click="openPreview(resolveImageUrl(url))">
                <div class="image-item image-item--fit"
                  :class="SHAPE_CARD_CLASS[prompt.shape] || 'image-item--square'"><img
                    :src="resolveImageUrl(url)" class="image image--contain" alt="模板图片"></div>
              </button>
            </div>
            <div v-else class="empty-copy">暂无模板图片</div>
          </article>
          <a-image-preview v-model:visible="previewVisible" :src="previewUrl" />
          <div v-if="!loading && prompts.length === 0" class="empty-state"><icon-file
              style="font-size:32px;color:var(--text-tertiary)" />
            <p>当前分类下暂无模板</p>
          </div>
        </div>
      </main>
    </div>
    <a-modal v-model:visible="modalVisible" :title="isEdit ? `编辑${shapeLabel(form.shape)}模板` : `添加${currentShapeLabel}`"
      width="720px" :mask-closable="false" @ok="submit" @cancel="closeModal">
      <div class="template-modal-shell template-modal-shell--minimal">
        <a-form ref="formRef" :model="form" layout="vertical">
          <a-form-item field="images" label="模板图"
            :rules="[{ required: true, type: 'array', min: 1, message: '请至少上传 1 张模板图片' }]">
            <div class="upload-panel upload-panel--minimal">
              <div class="upload-area upload-area--rich upload-area--centered">
                <div class="upload-dropzone-wrap">
                  <label class="upload-dropzone"
                    :class="{ uploading: imageUploading, 'upload-dropzone--dragover': isDragOverUpload, 'upload-dropzone--has-images': form.images.length > 0 }"
                    @dragover.prevent="isDragOverUpload = true" @dragleave.prevent="isDragOverUpload = false"
                    @drop.prevent="handleImageDrop">
                    <input class="file-input" type="file" multiple accept="image/png,image/jpeg,image/webp,image/gif"
                      @change="handleImageSelect">

                    <template v-if="form.images.length">
                      <div class="upload-thumb-grid">
                        <div v-for="(url, idx) in form.images" :key="idx" class="upload-thumb-card">
                          <img :src="resolveImageUrl(url)" class="upload-thumb-img">
                          <button class="remove-btn upload-thumb-remove" type="button"
                            @click.stop.prevent="removeImage(idx)"><icon-close /></button>
                        </div>
                        <div v-if="form.images.length < 9" class="upload-thumb-card upload-thumb-card--adder">
                          <div class="upload-thumb-plus"><icon-plus /></div>
                          <span>继续上传</span>
                        </div>
                      </div>
                    </template>
                    <template v-else>
                      <div class="upload-empty-state">
                        <div class="upload-dropzone-icon upload-dropzone-icon--large">
                          <icon-loading v-if="imageUploading" spin />
                          <icon-plus v-else />
                        </div>
                        <div class="upload-dropzone-copy upload-dropzone-copy--center upload-dropzone-copy--stacked">
                          <strong>{{ imageUploading ? '上传中...' : '点击或拖拽上传模板图' }}</strong>
                          <span>支持 PNG / JPG / WebP 格式</span>
                          <span>单次最多上传 9 张</span>
                        </div>
                      </div>
                    </template>

                    <div class="upload-dropzone-meta upload-dropzone-meta--center">
                      <span class="upload-meta-chip">{{ form.images.length }}/9</span>
                    </div>
                  </label>
                </div>
              </div>
            </div>
          </a-form-item>
          <div class="upload-hint upload-hint--panel">前台会按横版 / 竖版 / 方形自动归档这些模板图片，你只需要维护图片本身即可。</div>
        </a-form>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import request, { resolveImageUrl } from '@/utils/request'
import { IconPlus, IconFolder, IconDelete, IconFile, IconClose, IconLoading } from '@arco-design/web-vue/es/icon'
const SHAPES = [{ value: 'horizontal', label: '横版' }, { value: 'vertical', label: '竖版' }, { value: 'square', label: '方形' }]
const SHAPE_CARD_CLASS = { horizontal: 'image-item--horizontal', vertical: 'image-item--vertical', square: 'image-item--square' }
const loading = ref(false), imageUploading = ref(false), modalVisible = ref(false), isEdit = ref(false), formRef = ref(null), selectedShape = ref('horizontal'), allPrompts = ref([]), prompts = ref([]), isDragOverUpload = ref(false), previewVisible = ref(false), previewUrl = ref('')
const form = reactive({ id: null, shape: 'horizontal', images: [] })
const currentShapeLabel = computed(() => SHAPES.find(i => i.value === selectedShape.value)?.label || '模板')
const shapeLabel = shape => SHAPES.find(i => i.value === shape)?.label || '模板'
const shapeCount = shape => allPrompts.value.filter(i => i.shape === shape).length
const syncPrompts = () => { prompts.value = allPrompts.value.filter(i => i.shape === selectedShape.value) }
const selectShape = shape => { selectedShape.value = shape; syncPrompts() }
async function fetchPrompts() { loading.value = true; try { const res = await request.get('/admin/prompts'); allPrompts.value = Array.isArray(res.data) ? res.data.filter(i => i.status !== 'pending') : []; syncPrompts() } finally { loading.value = false } }
function resetForm() { form.id = null; form.shape = selectedShape.value; form.images = []; formRef.value?.clearValidate() }
function openModal(prompt = null) { isEdit.value = !!prompt; resetForm(); if (prompt) { form.id = prompt.id; form.shape = prompt.shape || 'square'; form.images = Array.isArray(prompt.images) ? [...prompt.images] : [] } modalVisible.value = true }
function closeModal() { modalVisible.value = false }
function openPreview(url) { previewUrl.value = url; previewVisible.value = true }
function removeImage(idx) { form.images.splice(idx, 1) }
const readFileAsBase64 = file => new Promise((resolve, reject) => { const reader = new FileReader(); reader.onload = () => { const result = String(reader.result || ''); resolve(result.includes(',') ? result.split(',')[1] : result) }; reader.onerror = reject; reader.readAsDataURL(file) })
async function uploadFiles(files) { if (!files.length) return; const remaining = 9 - form.images.length; if (remaining <= 0) { Message.warning('最多上传 9 张模板图片'); return } const toUpload = files.slice(0, remaining); if (files.length > remaining) Message.warning(`已超出上限，仅上传前 ${remaining} 张`); imageUploading.value = true; try { for (const file of toUpload) { const base64 = await readFileAsBase64(file); const res = await request.post('/upload/image', { data: base64, mime_type: file.type }); if (res.data?.url) form.images.push(res.data.url) } } catch { Message.error('部分图片上传失败，请重试') } finally { imageUploading.value = false } }
async function handleImageSelect(e) { const files = Array.from(e.target.files || []); await uploadFiles(files); e.target.value = '' }
async function handleImageDrop(e) { isDragOverUpload.value = false; const files = Array.from(e.dataTransfer?.files || []).filter(file => file.type.startsWith('image/')); await uploadFiles(files) }
async function submit() {
  const err = await formRef.value?.validate()
  if (err) return
  try {
    if (isEdit.value) {
      const payload = { group_id: 0, shape: selectedShape.value, title: '', content: '', images: form.images }
      await request.put(`/prompts/${form.id}`, payload)
      Message.success('模板更新成功')
    } else {
      await Promise.all(
        form.images.map((imageUrl) => request.post('/prompts', {
          group_id: 0,
          shape: selectedShape.value,
          title: '',
          content: '',
          images: [imageUrl]
        }))
      )
      Message.success(`成功创建 ${form.images.length} 个模板`)
    }
    modalVisible.value = false
    await fetchPrompts()
    syncPrompts()
  } catch { }
}
async function removePrompt(id) { try { await request.delete(`/prompts/${id}`); Message.success('模板删除成功'); await fetchPrompts() } catch { } }
onMounted(fetchPrompts)
</script>

<style scoped>
.template-modal-shell :deep(.arco-form-item-content){width:100%;}
.template-modal-shell :deep(.arco-form-item-content-flex){width:100%;display:flex;justify-content:center;align-items:flex-start;}

.page-container {
  padding: 24px;
  display: flex;
  flex-direction: column;
  min-height: 100%
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px
}

.page-title {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary)
}

.page-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--text-tertiary)
}

.content-layout {
  display: flex;
  gap: 20px;
  flex: 1;
  min-height: 400px
}

.sidebar {
  width: 260px;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color)
}

.sidebar-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary)
}

.sidebar-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-tertiary)
}

.group-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px
}

.group-item {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: all .15s;
  margin-bottom: 4px
}

.group-item:hover {
  background: var(--bg-tertiary)
}

.group-item.active {
  background: var(--accent-color);
  color: #fff
}

.group-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0
}

.group-icon {
  color: var(--accent-color);
  font-size: 16px;
  flex-shrink: 0
}

.group-item.active .group-icon {
  color: #fff
}

.group-name {
  font-size: 14px;
  font-weight: 500
}

.group-count {
  font-size: 12px;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: 10px
}

.group-item.active .group-count {
  background: rgba(255, 255, 255, .2);
  color: #fff
}

.main-content {
  flex: 1;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden
}

.content-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color)
}

.content-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  color: var(--text-primary)
}

.content-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 2px 0 0
}

.prompt-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px
}

.gallery-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 18px;
  align-content: start
}

.prompt-card {
  position: relative;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 14px;
  transition: all .18s;
  box-shadow: 0 8px 24px rgba(15, 23, 42, .04)
}

.prompt-card:hover {
  border-color: rgba(22, 93, 255, .24);
  box-shadow: 0 14px 34px rgba(15, 23, 42, .08)
}

.prompt-card--image-only {
  padding: 14px
}

.prompt-delete-btn {
  position: absolute;
  top: 18px;
  right: 18px;
  z-index: 3;
  width: 30px;
  height: 30px;
  border-radius: 999px;
  background: rgba(255, 255, 255, .72);
  color: rgba(15, 23, 42, .58);
  border: 1px solid rgba(148, 163, 184, .18);
  backdrop-filter: blur(10px);
  opacity: 0;
  transform: translateY(-4px) scale(.96);
  transition: opacity .18s ease, transform .18s ease, background .18s ease, color .18s ease, border-color .18s ease, box-shadow .18s ease;
  box-shadow: 0 4px 14px rgba(15, 23, 42, .06)
}

.prompt-card:hover .prompt-delete-btn {
  opacity: 1;
  transform: translateY(0) scale(1)
}

.prompt-delete-btn:hover {
  background: rgba(255, 255, 255, .94) !important;
  color: rgba(220, 38, 38, .82) !important;
  border-color: rgba(220, 38, 38, .14);
  box-shadow: 0 8px 18px rgba(220, 38, 38, .10)
}

.image-preview-trigger {
  display: block;
  width: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: zoom-in
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px
}

.image-grid--single {
  grid-template-columns: 1fr
}

.image-item {
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  aspect-ratio: 1/1;
  transition: transform .18s ease, box-shadow .18s ease, border-color .18s ease
}

.image-preview-trigger:hover .image-item {
  transform: translateY(-2px);
  border-color: rgba(22, 93, 255, .26);
  box-shadow: 0 10px 24px rgba(15, 23, 42, .08)
}

.image-item--fit {
  background: #f7f8fb
}

.image-item--horizontal {
  aspect-ratio: 16/9
}

.image-item--vertical {
  aspect-ratio: 4/5
}

.image-item--square {
  aspect-ratio: 1/1
}

.image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block
}

.image--contain {
  object-fit: contain;
  background: #f7f8fb
}

.empty-copy,
.upload-hint {
  font-size: 12px;
  color: var(--text-tertiary)
}

.empty-state {
  grid-column: 1/-1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-tertiary);
  text-align: center
}

.empty-state p {
  margin: 12px 0;
  font-size: 14px
}

.template-modal-shell {
  display: flex;
  flex-direction: column;
  gap: 14px
}

.template-modal-shell--minimal {
  padding: 6px 0 2px
}

.upload-panel {
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  background: linear-gradient(180deg, #fff, var(--color-fill-1))
}

.upload-panel--minimal {
  padding: 0;
  border: none;
  background: transparent
}

.upload-area--rich {
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: 100%
}

.upload-area--centered {
  align-items: center
}

.upload-dropzone-wrap {
  display: flex;
  justify-content: center;
  width: 100%
}

.upload-dropzone {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 32px 24px;
  border: 1px dashed #d9dee8;
  border-radius: 18px;
  background: linear-gradient(180deg, #fbfcfe, #f7f9fc);
  cursor: pointer;
  transition: all .16s;
  min-height: 236px;
  justify-content: center;
  width: 392px;
  max-width: 100%;
  margin: 0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .9)
}

.upload-dropzone:hover {
  border-color: #c7d2e5;
  background: linear-gradient(180deg, #fcfdff, #f5f8fc)
}

.upload-dropzone.uploading {
  opacity: .7;
  cursor: not-allowed
}

.upload-dropzone--dragover {
  border-color: var(--accent-color);
  background: #f3f7ff;
  box-shadow: 0 0 0 3px rgba(var(--primary-6), .08)
}

.upload-dropzone--has-images {
  padding: 14px;
  min-height: auto;
  gap: 12px
}

.upload-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18px;
  min-height: 170px;
  text-align: center
}

.upload-dropzone-visual {
  display: flex;
  align-items: center;
  gap: 18px
}

.upload-dropzone-visual--center {
  flex-direction: column;
  justify-content: center;
  text-align: center
}

.upload-dropzone-icon {
  width: 60px;
  height: 60px;
  border-radius: 16px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
  font-size: 22px;
  border: 1px solid #e6eaf0;
  flex-shrink: 0
}

.upload-dropzone-icon--large {
  width: 64px;
  height: 64px;
  border-radius: 18px;
  font-size: 24px;
  box-shadow: 0 6px 18px rgba(59, 130, 246, .08)
}

.upload-dropzone-copy {
  display: flex;
  flex-direction: column;
  gap: 6px
}

.upload-dropzone-copy--center {
  align-items: center
}

.upload-dropzone-copy--stacked {
  gap: 4px
}

.upload-dropzone-copy strong {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.4
}

.upload-dropzone-copy span {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-tertiary)
}

.upload-dropzone-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px
}

.upload-dropzone-meta--center {
  justify-content: center
}

.upload-meta-chip {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  background: #fff;
  border: 1px solid #e6eaf0;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-secondary);
  box-shadow: 0 1px 2px rgba(15, 23, 42, .04)
}

.upload-thumb-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  width: 100%
}

.upload-thumb-card {
  position: relative;
  aspect-ratio: 1/1;
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid #e7ebf2;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, .04)
}

.upload-thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block
}

.upload-thumb-remove {
  position: absolute;
  top: 6px;
  right: 6px
}

.upload-thumb-card--adder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed #d7dfea;
  color: var(--text-tertiary);
  background: linear-gradient(180deg, #ffffff, #f8fafc)
}

.upload-thumb-plus {
  width: 32px;
  height: 32px;
  border-radius: 11px;
  border: 1px solid #e6eaf0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, .04)
}

.upload-thumb-card--adder span {
  font-size: 12px;
  font-weight: 500
}

.remove-btn {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(17, 24, 39, .58);
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 11px;
  padding: 0;
  line-height: 1;
  transition: all .15s
}

.remove-btn:hover {
  background: rgba(220, 38, 38, .92)
}

.upload-hint--panel {
  display: none
}

.file-input {
  display: none
}

@media (max-width:768px) {

  .page-header,
  .content-layout,
  .content-header {
    flex-direction: column;
    align-items: stretch
  }

  .sidebar {
    width: 100%
  }

  .gallery-list {
    grid-template-columns: 1fr
  }

  .upload-dropzone {
    width: 100%;
    min-height: auto;
    padding: 24px 16px
  }

  .upload-dropzone-icon {
    width: 52px;
    height: 52px;
    font-size: 20px
  }

  .upload-dropzone-icon--large {
    width: 56px;
    height: 56px;
    font-size: 22px
  }

  .upload-dropzone-copy strong {
    font-size: 16px
  }

  .upload-dropzone-copy span {
    font-size: 12px
  }

  .upload-thumb-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr))
  }
}
</style>
