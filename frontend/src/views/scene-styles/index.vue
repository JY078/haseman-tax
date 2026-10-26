<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">风格库</h1>
    </div>

    <div class="split-layout">
      <!-- 左侧：场景列表 -->
      <aside class="split-layout__left">
        <div class="left-toolbar">
          <span class="left-toolbar__label">场景</span>
          <a-button type="primary" size="small" @click="openSceneModal()">
            <template #icon><icon-plus /></template>
            新建
          </a-button>
        </div>

        <div v-if="loading" class="scene-list">
          <div v-for="n in 4" :key="n" class="scene-item scene-item--skeleton" />
        </div>
        <a-empty v-else-if="scenes.length === 0" description="暂无场景" />
        <div v-else class="scene-list">
          <div
            v-for="scene in scenes"
            :key="scene.id"
            class="scene-item"
            :class="{ 'scene-item--active': selectedSceneId === scene.id }"
            @click="selectScene(scene.id)"
          >
            <div class="scene-item__main">
              <span class="scene-item__name">{{ scene.name }}</span>
              <!-- <span class="scene-item__count">{{ getSceneTagCount(scene.id) }} 风格</span> -->
            </div>
            <div class="scene-item__actions" @click.stop>
              <a-button type="text" size="mini" @click="openSceneModal(scene)">
                <template #icon><icon-edit /></template>
              </a-button>
              <a-popconfirm content="删除该场景？" @ok="handleDeleteScene(scene.id)">
                <a-button type="text" status="danger" size="mini">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-popconfirm>
            </div>
          </div>
        </div>
      </aside>

      <!-- 右侧：选中场景的详情 -->
      <main class="split-layout__right">
        <a-empty v-if="!selectedSceneId" description="请从左侧选择一个场景" />
        <template v-else>
          <!-- 风格标签 -->
          <section class="right-section">
            <div class="right-section__toolbar">
              <span class="right-section__title">风格标签</span>
              <a-button size="small" @click="fetchData">刷新</a-button>
              <a-button type="primary" size="small" @click="openTagModal()">
                <template #icon><icon-plus /></template>
                新建风格
              </a-button>
            </div>

            <div v-if="loading" class="tag-chip-grid">
              <div v-for="n in 4" :key="n" class="tag-chip tag-chip--skeleton" />
            </div>
            <a-empty v-else-if="tags.length === 0" description="暂无风格标签" />
            <div v-else class="tag-chip-grid">
              <div
                v-for="tag in tags"
                :key="tag.id"
                class="tag-chip"
                :class="{ 'tag-chip--active': selectedTagId === tag.id }"
                @click="selectedTagId = tag.id"
              >
                <div class="tag-chip__main">
                  <span class="tag-chip__name">{{ tag.name }}</span>
                  <span class="tag-chip__count">{{ tag.references?.length || 0 }} 张</span>
                </div>
                <div class="tag-chip__actions" @click.stop>
                  <a-button type="text" size="mini" @click="openTagModal(tag)">
                    <template #icon><icon-edit /></template>
                  </a-button>
                  <a-popconfirm content="删除该风格及其全部参考图？" @ok="handleDeleteTag(tag.id)">
                    <a-button type="text" status="danger" size="mini">
                      <template #icon><icon-delete /></template>
                    </a-button>
                  </a-popconfirm>
                </div>
              </div>
            </div>
          </section>

          <!-- 参考图 -->
          <section class="right-section">
            <div class="right-section__toolbar">
              <span class="right-section__title">
                参考图
                <span v-if="currentTag" class="right-section__title-sub"> · {{ currentTag.name }}</span>
              </span>
              <a-button size="small" @click="fetchData">刷新</a-button>
              <a-button type="primary" size="small" @click="openReferenceModal()" :disabled="!currentTag || deleteMode">
                <template #icon><icon-plus /></template>
                新建参考图
              </a-button>
              <a-button
                :status="deleteMode ? 'normal' : 'danger'"
                :type="deleteMode ? 'outline' : 'primary'"
                size="small"
                :disabled="!currentTag || filteredReferences.length === 0"
                @click="toggleDeleteMode"
              >
                <template #icon><icon-delete /></template>
                {{ deleteMode ? '取消' : '删除' }}
              </a-button>
              <a-button
                v-if="deleteMode"
                type="primary"
                status="danger"
                size="small"
                :disabled="selectedReferenceIds.length === 0"
                @click="handleBatchDelete"
              >
                删除已选 ({{ selectedReferenceIds.length }})
              </a-button>
            </div>

            <div v-if="loading" class="reference-grid">
              <div v-for="n in 6" :key="n" class="reference-card reference-card--skeleton" />
            </div>
            <a-empty v-else-if="!currentTag" description="请选择一个风格标签" />
            <a-empty v-else-if="filteredReferences.length === 0" description="暂无参考图" />
            <div v-else class="reference-grid">
              <div
                v-for="record in displayReferences"
                :key="record.id"
                class="reference-card"
                :class="{
                  'reference-card--selectable': deleteMode,
                  'reference-card--selected': isReferenceSelected(record.id),
                }"
                @click="handleReferenceCardClick(record)"
              >
                <div class="reference-card__media">
                  <img :src="resolveImageUrl(record.image_url || record.imageUrl)" class="reference-card__image" alt="参考图" />
                  <span v-if="deleteMode" class="reference-card__selector">
                    <span class="reference-card__selector-dot" />
                  </span>
                  <span v-else class="reference-card__zoom">查看大图</span>
                </div>
              </div>
            </div>
            <div v-if="remainingCount > 0" class="reference-load-more">
              <a-button size="small" @click="showAllReferences = true">
                加载更多 ({{ remainingCount }})
              </a-button>
            </div>
          </section>
        </template>
      </main>
    </div>

    <!-- 预览 -->
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

    <!-- 场景 Modal -->
    <a-modal
      v-model:visible="sceneModalVisible"
      :title="isEditScene ? '编辑场景' : '新建场景'"
      :ok-loading="submitLoading"
      @ok="handleSceneSubmit"
      @cancel="closeSceneModal"
    >
      <a-form :model="sceneForm" ref="sceneFormRef" layout="vertical">
        <a-form-item field="name" label="场景名称" :rules="[{ required: true, message: '请输入场景名称' }]">
          <a-input v-model="sceneForm.name" placeholder="例如：卫浴 / 厨房 / 客厅" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 风格标签 Modal -->
    <a-modal
      v-model:visible="tagModalVisible"
      :title="isEditTag ? '编辑风格标签' : '新建风格标签'"
      :ok-loading="submitLoading"
      @ok="handleTagSubmit"
      @cancel="closeTagModal"
    >
      <a-form :model="tagForm" ref="tagFormRef" layout="vertical">
        <a-form-item field="name" label="风格名称" :rules="[{ required: true, message: '请输入风格名称' }]">
          <a-input v-model="tagForm.name" placeholder="例如：极简 / 轻奢 / 北欧自然" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 参考图 Modal -->
    <a-modal
      v-model:visible="referenceModalVisible"
      :title="isEditReference ? '编辑参考图' : '新建参考图'"
      width="760px"
      :ok-loading="submitLoading"
      @ok="handleReferenceSubmit"
      @cancel="closeReferenceModal"
    >
      <a-form :model="referenceForm" ref="referenceFormRef" layout="vertical">
        <a-form-item label="所属场景">
          <span>{{ scenes.find(s => s.id === selectedSceneId)?.name || '未选择' }}</span>
        </a-form-item>
        <a-form-item field="tag_id" label="所属风格" :rules="[{ required: true, message: '请选择风格' }]">
          <a-select v-model="referenceForm.tag_id" placeholder="选择风格">
            <a-option v-for="tag in referenceFormTags" :key="tag.id" :value="tag.id">{{ tag.name || `未命名(${tag.id})` }}</a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="参考图">
          <div
            class="batch-upload-zone"
            :class="{ 'batch-upload-zone--dragover': isDragOverUpload }"
            @click="triggerReferenceFileSelect"
            @dragover.prevent="isDragOverUpload = true"
            @dragleave.prevent="isDragOverUpload = false"
            @drop.prevent="handleReferenceDrop"
          >
            <input ref="referenceFileInputRef" class="upload-input" type="file" accept="image/*" multiple @change="handleReferenceUpload" />
            <span class="batch-upload-zone__icon">+</span>
            <span class="batch-upload-zone__text">点击或拖入图片上传（支持多选）</span>
          </div>
          <div v-if="pendingImages.length > 0" class="batch-preview-grid">
            <div
              v-for="(img, idx) in pendingImages"
              :key="idx"
              class="batch-preview-card"
              :class="{ 'batch-preview-card--uploading': img.uploading }"
            >
              <img v-if="img.url" :src="resolveImageUrl(img.url)" class="batch-preview-card__img" alt="" />
              <div v-else class="batch-preview-card__loading">
                <a-spin :size="20" />
              </div>
              <button class="batch-preview-card__remove" type="button" @click="removePendingImage(idx)">×</button>
            </div>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import request, { resolveImageUrl } from '@/utils/request'

const loading = ref(false)
const submitLoading = ref(false)
const scenes = ref([])
const tags = ref([])
const selectedSceneId = ref(null)
const selectedTagId = ref(null)

const sceneModalVisible = ref(false)
const tagModalVisible = ref(false)
const referenceModalVisible = ref(false)
const isEditScene = ref(false)
const isEditTag = ref(false)
const isEditReference = ref(false)
const sceneFormRef = ref(null)
const tagFormRef = ref(null)
const referenceFormRef = ref(null)
const referenceFileInputRef = ref(null)
const previewImageUrl = ref('')
const isDragOverUpload = ref(false)
const deleteMode = ref(false)
const selectedReferenceIds = ref([])
const pendingImages = ref([])
const showAllReferences = ref(false)
const PAGE_SIZE = 12

const sceneForm = reactive({ id: null, name: '' })
const tagForm = reactive({ id: null, name: '' })
const referenceForm = reactive({ id: null, scene_id: null, tag_id: null, image_url: '' })

const previewVisible = computed(() => !!previewImageUrl.value)
const currentTag = computed(() => tags.value.find(t => t.id === selectedTagId.value) || null)
const allReferences = computed(() => tags.value.flatMap(tag => (tag.references || []).map(item => ({ ...item, tag_id: item.tag_id || tag.id }))))
const filteredReferences = computed(() => !selectedTagId.value ? [] : allReferences.value.filter(item => item.tag_id === selectedTagId.value && item.scene_id === selectedSceneId.value))
const displayReferences = computed(() => showAllReferences.value ? filteredReferences.value : filteredReferences.value.slice(0, PAGE_SIZE))
const remainingCount = computed(() => Math.max(0, filteredReferences.value.length - PAGE_SIZE))
// 参考图弹窗中显示所有风格（风格本身是跨场景通用的）
const referenceFormTags = computed(() => tags.value)

watch(selectedTagId, () => { showAllReferences.value = false })
// 参考图弹窗中自动选标签：tag_id 为空时自动选中当前选中标签
watch(() => referenceForm.tag_id, (val) => {
  if (!val || val === 0) {
    referenceForm.tag_id = selectedTagId.value || tags.value[0]?.id || null
  }
})

function getSceneTagCount(sceneId) {
  // scene_id=0 表示旧数据（未指定场景），计入会话数；新标签按 scene_id 精确匹配
  return tags.value.filter(t => t.scene_id === 0 || t.scene_id === sceneId).length
}

async function fetchData() {
  loading.value = true
  try {
    const [scenesRes, tagsRes] = await Promise.all([
      request.get('/admin/scene-style-categories'),
      request.get('/admin/scene-style-tags'),
    ])
    scenes.value = scenesRes.data || []
    tags.value = tagsRes.data || []

    if (!selectedSceneId.value && scenes.value.length) {
      selectedSceneId.value = scenes.value[0].id
    }
    if (selectedSceneId.value && !scenes.value.some(s => s.id === selectedSceneId.value)) {
      selectedSceneId.value = scenes.value[0]?.id || null
    }

    // 自动选第一个标签（优先选当前场景的，其次选 scene_id=0 的旧数据）
    if (tags.value.length && !selectedTagId.value) {
      const match = tags.value.find(t => t.scene_id === selectedSceneId.value)
      const fallback = tags.value.find(t => t.scene_id === 0)
      selectedTagId.value = match?.id || fallback?.id || tags.value[0].id
    }
    if (selectedTagId.value && !tags.value.some(t => t.id === selectedTagId.value)) {
      selectedTagId.value = tags.value[0]?.id || null
    }

    selectedReferenceIds.value = selectedReferenceIds.value.filter(id => filteredReferences.value.some(item => item.id === id))
  } finally {
    loading.value = false
  }
}

function selectScene(id) {
  selectedSceneId.value = id
  selectedTagId.value = null // 切场景时清空标签选择，避免旧场景标签影响操作
  deleteMode.value = false
  selectedReferenceIds.value = []
}

// --- Scene ---
function resetSceneForm() {
  sceneForm.id = null
  sceneForm.name = ''
  sceneFormRef.value?.clearValidate()
}

function openSceneModal(record = null) {
  isEditScene.value = !!record
  resetSceneForm()
  if (record) {
    sceneForm.id = record.id
    sceneForm.name = record.name
  }
  sceneModalVisible.value = true
}

function closeSceneModal() { sceneModalVisible.value = false }

async function handleSceneSubmit() {
  const err = await sceneFormRef.value?.validate()
  if (err) return
  submitLoading.value = true
  try {
    const payload = { name: sceneForm.name }
    if (isEditScene.value) {
      await request.put(`/admin/scene-style-categories/${sceneForm.id}`, payload)
      Message.success('场景已更新')
    } else {
      await request.post('/admin/scene-style-categories', payload)
      Message.success('场景已创建')
    }
    sceneModalVisible.value = false
    await fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDeleteScene(id) {
  await request.delete(`/admin/scene-style-categories/${id}`)
  Message.success('场景已删除')
  if (selectedSceneId.value === id) {
    selectedSceneId.value = null
    selectedTagId.value = null
  }
  await fetchData()
}

// --- Tag ---
function resetTagForm() {
  tagForm.id = null
  tagForm.name = ''
  tagFormRef.value?.clearValidate()
}

function openTagModal(record = null) {
  isEditTag.value = !!record
  resetTagForm()
  if (record) {
    tagForm.id = record.id
    tagForm.name = record.name
  }
  tagModalVisible.value = true
}

function closeTagModal() { tagModalVisible.value = false }

async function handleTagSubmit() {
  const err = await tagFormRef.value?.validate()
  if (err) return
  submitLoading.value = true
  try {
    const payload = { name: tagForm.name }
    if (isEditTag.value) {
      await request.put(`/admin/scene-style-tags/${tagForm.id}`, payload)
      Message.success('风格标签已更新')
    } else {
      await request.post('/admin/scene-style-tags', payload)
      Message.success('风格标签已创建')
    }
    tagModalVisible.value = false
    await fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDeleteTag(id) {
  await request.delete(`/admin/scene-style-tags/${id}`)
  Message.success('风格标签已删除')
  if (selectedTagId.value === id) {
    selectedTagId.value = null
    deleteMode.value = false
    selectedReferenceIds.value = []
  }
  await fetchData()
}

function resetReferenceForm() {
  referenceForm.id = null
  referenceForm.scene_id = selectedSceneId.value || null
  referenceForm.tag_id = selectedTagId.value || tags.value[0]?.id || null
  referenceForm.image_url = ''
  pendingImages.value = []
  referenceFormRef.value?.clearValidate()
}

function openReferenceModal(record = null) {
  isEditReference.value = !!record
  resetReferenceForm()
  if (record) {
    referenceForm.id = record.id
    referenceForm.scene_id = record.scene_id || null
    const tag = tags.value.find(t => t.id === record.tag_id)
    referenceForm.tag_id = tag?.id || record.tag_id
    referenceForm.image_url = record.image_url || ''
  }
  referenceModalVisible.value = true
}

function closeReferenceModal() {
  referenceModalVisible.value = false
  isDragOverUpload.value = false
}

function isReferenceSelected(id) {
  return selectedReferenceIds.value.includes(id)
}

function toggleDeleteMode() {
  deleteMode.value = !deleteMode.value
  selectedReferenceIds.value = []
}

function handleReferenceCardClick(record) {
  if (deleteMode.value) {
    selectedReferenceIds.value = isReferenceSelected(record.id)
      ? selectedReferenceIds.value.filter(id => id !== record.id)
      : [...selectedReferenceIds.value, record.id]
    return
  }
  previewImageUrl.value = resolveImageUrl(record.image_url || record.imageUrl)
}

async function handleBatchDelete() {
  if (selectedReferenceIds.value.length === 0) return
  await Promise.all(selectedReferenceIds.value.map(id => request.delete(`/admin/scene-style-references/${id}`)))
  Message.success(`已删除 ${selectedReferenceIds.value.length} 张参考图`)
  selectedReferenceIds.value = []
  deleteMode.value = false
  if (previewImageUrl.value) previewImageUrl.value = ''
  await fetchData()
}

function triggerReferenceFileSelect() {
  referenceFileInputRef.value?.click()
}

async function uploadReferenceFile(file) {
  return new Promise((resolve, reject) => {
    if (!file) { reject(new Error('no file')); return }
    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const base64 = String(e.target?.result || '')
        console.log('Uploading file:', file.name, 'size:', file.size, 'type:', file.type)
        const res = await request.post('/upload/image', { data: base64, mime_type: file.type || 'image/png' })
        console.log('Upload success:', res.data?.url)
        resolve(res.data?.url || '')
      } catch (err) {
        console.error('Upload failed:', err)
        reject(new Error('upload failed'))
      }
    }
    reader.readAsDataURL(file)
  })
}

async function uploadPendingFiles(files) {
  for (const file of files) {
    const idx = pendingImages.value.length
    pendingImages.value.push({ url: '', uploading: true })
    try {
      const url = await uploadReferenceFile(file)
      pendingImages.value[idx] = { url, uploading: false }
    } catch {
      pendingImages.value.splice(idx, 1)
      Message.error('部分图片上传失败')
    }
  }
}

async function handleReferenceUpload(event) {
  const files = Array.from(event.target.files || [])
  console.log('Files selected:', files.length)
  if (files.length) await uploadPendingFiles(files)
  if (referenceFileInputRef.value) referenceFileInputRef.value.value = ''
  isDragOverUpload.value = false
}

async function handleReferenceDrop(event) {
  const files = Array.from(event.dataTransfer?.files || []).filter(f => f.type.startsWith('image/'))
  if (files.length) await uploadPendingFiles(files)
  isDragOverUpload.value = false
}

function removePendingImage(idx) {
  pendingImages.value.splice(idx, 1)
}

async function handleReferenceSubmit() {
  const err = await referenceFormRef.value?.validate()
  if (err) {
    console.log('Form validation failed:', err)
    return
  }
  console.log('pendingImages:', pendingImages.value)
  const readyImages = pendingImages.value.filter(img => !img.uploading && img.url)
  console.log('readyImages:', readyImages)

  // 新建模式：必须有已上传的图片
  if (!isEditReference.value && readyImages.length === 0) {
    Message.warning('请至少上传一张参考图')
    return
  }
  // 编辑模式：没有新上传图片时使用原图
  const imageUrl = readyImages.length > 0 ? readyImages[0].url : referenceForm.image_url

  submitLoading.value = true
  try {
    if (isEditReference.value) {
      await request.put(`/admin/scene-style-references/${referenceForm.id}`, {
        tag_id: referenceForm.tag_id,
        image_url: imageUrl
      })
    } else {
      console.log('Creating', readyImages.length, 'references with scene_id:', selectedSceneId.value, 'tag_id:', referenceForm.tag_id)
      const results = await Promise.all(readyImages.map(img =>
        request.post('/admin/scene-style-references', {
          scene_id: selectedSceneId.value,
          tag_id: referenceForm.tag_id,
          image_url: img.url
        })
      ))
      console.log('Created results:', results)
    }
    Message.success(isEditReference.value ? '参考图已更新' : `已添加 ${readyImages.length} 张参考图`)
    referenceModalVisible.value = false
    await fetchData()
  } catch (error) {
    console.error('Submit error:', error)
    Message.error('提交失败: ' + (error?.response?.data?.error || error.message))
  } finally {
    submitLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  margin-bottom: 20px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
}

/* --- Split Layout --- */
.split-layout {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.split-layout__left {
  width: 260px;
  flex-shrink: 0;
  background: #fff;
  border-radius: 20px;
  padding: 16px;
  border: 1px solid var(--color-border-2);
  position: sticky;
  top: 24px;
}

.split-layout__right {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* --- Left Panel --- */
.left-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-1);
}

.left-toolbar__label {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-1);
}

.scene-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.scene-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.18s ease;
}

.scene-item:hover {
  background: rgba(var(--primary-1), 0.6);
}

.scene-item--active {
  background: rgba(var(--primary-1), 0.95);
  border-color: rgba(var(--primary-5), 0.4);
}

.scene-item--skeleton {
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(110deg, #f6f7fb 8%, #eef1f7 18%, #f6f7fb 33%);
  background-size: 200% 100%;
  animation: shimmer 1.1s linear infinite;
  pointer-events: none;
}

.scene-item__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.scene-item__name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.scene-item__count {
  font-size: 11px;
  color: var(--color-text-3);
}

.scene-item__actions {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.scene-item:hover .scene-item__actions {
  opacity: 1;
}

/* --- Right Section --- */
.right-section {
  background: #fff;
  border-radius: 20px;
  padding: 20px;
  border: 1px solid var(--color-border-2);
}

.right-section__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.right-section__title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-1);
  margin-right: auto;
}

.right-section__title-sub {
  font-size: 13px;
  font-weight: 400;
  color: var(--color-text-3);
}

/* --- Tag Chips --- */
.tag-chip-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}

.tag-chip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid var(--color-border-2);
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #fafbff 100%);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.04);
  cursor: pointer;
  transition: all 0.2s ease;
}

.tag-chip:hover {
  transform: translateY(-1px);
  border-color: rgba(var(--primary-5), 0.4);
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
}

.tag-chip--active {
  border-color: rgb(var(--primary-6));
  background: linear-gradient(180deg, rgba(var(--primary-1), 0.95) 0%, #ffffff 100%);
  box-shadow: 0 12px 24px rgba(var(--primary-6), 0.12);
}

.tag-chip--skeleton {
  height: 56px;
  background: linear-gradient(110deg, #f6f7fb 8%, #eef1f7 18%, #f6f7fb 33%);
  background-size: 200% 100%;
  animation: shimmer 1.1s linear infinite;
}

.tag-chip__main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.tag-chip__name {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-1);
}

.tag-chip__count {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-text-2);
  font-size: 11px;
  font-weight: 600;
}

.tag-chip__actions {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
}

/* --- Reference Grid --- */
.reference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.reference-card {
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.05);
  cursor: zoom-in;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.reference-card:hover {
  transform: translateY(-2px);
  border-color: rgba(var(--primary-5), 0.4);
  box-shadow: 0 16px 32px rgba(15, 23, 42, 0.08);
}

.reference-card--selectable {
  cursor: pointer;
}

.reference-card--selected {
  border-color: rgb(var(--danger-6));
  box-shadow: 0 0 0 2px rgba(var(--danger-2), 0.8), 0 12px 24px rgba(var(--danger-6), 0.14);
}

.reference-card--skeleton {
  height: 160px;
  background: linear-gradient(110deg, #f6f7fb 8%, #eef1f7 18%, #f6f7fb 33%);
  background-size: 200% 100%;
  animation: shimmer 1.1s linear infinite;
}

.reference-card__media {
  position: relative;
  background: #f7f8fa;
}

.reference-card__image {
  width: 100%;
  aspect-ratio: 1 / 1;
  object-fit: cover;
  display: block;
}

.reference-card__zoom,
.reference-card__selector {
  position: absolute;
  right: 10px;
  bottom: 10px;
}

.reference-card__zoom {
  padding: 5px 9px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.72);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  opacity: 0;
  transform: translateY(4px);
  transition: all 0.18s ease;
}

.reference-card:hover .reference-card__zoom {
  opacity: 1;
  transform: translateY(0);
}

.reference-card__selector {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 6px 14px rgba(15, 23, 42, 0.1);
}

.reference-card__selector-dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  border: 2px solid var(--color-border-3);
  background: #fff;
}

.reference-card--selected .reference-card__selector-dot {
  border-color: rgb(var(--danger-6));
  background: rgb(var(--danger-6));
}

.reference-load-more {
  display: flex;
  justify-content: center;
  padding: 16px 0 4px;
}

/* --- Upload --- */
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
  min-height: 180px;
  width: 100%;
  padding: 24px 20px;
  border: 1px dashed rgb(var(--primary-6));
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(var(--primary-1), 0.7) 0%, #fff 100%);
  cursor: pointer;
  transition: all 0.2s ease;
}

.upload-trigger:hover,
.upload-trigger--dragover {
  border-color: rgb(var(--primary-5));
  transform: translateY(-1px);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
}

.upload-trigger__icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--primary-6));
  color: #fff;
  font-size: 24px;
  line-height: 1;
  box-shadow: 0 8px 20px rgba(var(--primary-6), 0.26);
}

.upload-trigger__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
}

.upload-trigger__hint {
  font-size: 12px;
  color: var(--color-text-3);
}

.preview-wrap {
  position: relative;
  width: 100%;
  max-width: 280px;
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: #fff;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
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
  gap: 10px;
  padding: 12px;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0) 0%, rgba(15, 23, 42, 0.72) 100%);
}

.preview-badge {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 9px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.18);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  backdrop-filter: blur(6px);
}

.preview-actions {
  display: flex;
  gap: 6px;
}

.preview-action {
  border: none;
  border-radius: 999px;
  padding: 6px 10px;
  background: #fff;
  color: var(--color-text-1);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.16s ease;
}

.preview-action:hover {
  transform: translateY(-1px);
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
  to { background-position-x: -200%; }
}

/* --- Batch Upload --- */
.batch-upload-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 120px;
  padding: 24px 20px;
  border: 2px dashed rgba(var(--primary-6), 0.5);
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}
.batch-upload-zone:hover,
.batch-upload-zone--dragover {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-1), 0.5);
}
.batch-upload-zone__icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(var(--primary-6));
  color: #fff;
  font-size: 20px;
  line-height: 1;
}
.batch-upload-zone__text {
  font-size: 13px;
  color: var(--color-text-2);
}

.batch-preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 10px;
  margin-top: 12px;
}
.batch-preview-card {
  position: relative;
  aspect-ratio: 1;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: #f7f8fa;
}
.batch-preview-card__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.batch-preview-card__loading {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.batch-preview-card__remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: none;
  background: rgba(15, 23, 42, 0.6);
  color: #fff;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.batch-preview-card:hover .batch-preview-card__remove {
  opacity: 1;
}
</style>
