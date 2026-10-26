<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">参考图库</h1>
        <p class="page-subtitle">管理套图生成可选的参考图方案。</p>
      </div>
      <a-button type="primary" @click="openCreate">
        <template #icon><icon-plus /></template>
        新建方案
      </a-button>
    </div>

    <a-card class="filter-card">
      <a-space wrap>
        <a-input-search v-model="keyword" placeholder="搜索方案名称 / 类目" allow-clear class="search-input" />
        <a-select v-model="statusFilter" placeholder="状态" allow-clear class="status-select">
          <a-option value="enabled">启用</a-option>
          <a-option value="disabled">停用</a-option>
        </a-select>
        <a-button @click="fetchData">刷新</a-button>
      </a-space>
    </a-card>

    <a-card class="global-assets-card">
      <div class="global-assets-head">
        <div>
          <strong>全局主图素材</strong>
          <span>所有 Forious / Androme 主图都会自动引用这里的水管素材，不跟随单个方案变化</span>
        </div>
        <a-button type="primary" :loading="globalAssetSaving" @click="saveGlobalMainAssets">保存全局素材</a-button>
      </div>
      <div class="global-assets-grid">
        <article v-for="slot in globalMainAssetSlots" :key="slot.key" class="global-asset-item">
          <button class="global-asset-preview" type="button" @click="preview(globalAssetMap[slot.key])">
            <img v-if="globalAssetMap[slot.key]" :src="resolveImageUrl(globalAssetMap[slot.key])" :alt="slot.label">
            <span v-else>未上传</span>
          </button>
          <div class="global-asset-info">
            <div>
              <strong>{{ slot.label }}</strong>
              <p>{{ slot.desc }}</p>
            </div>
            <a-space>
              <a-button size="small" @click="triggerUpload(slot.key, 'global')">
                <template #icon><icon-upload /></template>
                上传
              </a-button>
              <a-button size="small" @click="preview(globalAssetMap[slot.key])" :disabled="!globalAssetMap[slot.key]">预览</a-button>
              <a-button size="small" status="danger" @click="clearGlobalAsset(slot.key)" :disabled="!globalAssetMap[slot.key]">清空</a-button>
            </a-space>
          </div>
        </article>
      </div>
    </a-card>

    <a-card>
      <a-table :data="filteredGalleries" :loading="loading" :pagination="false" row-key="id">
        <template #columns>
          <a-table-column title="方案名称" :width="260">
            <template #cell="{ record }">
              <div class="scheme-name-cell">
                <strong>{{ record.name }}</strong>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="适用类目" :width="140">
            <template #cell="{ record }">
              <a-tag v-if="record.category" size="small" color="arcoblue">{{ categoryLabel(record.category) }}</a-tag>
              <span v-else class="text-secondary">未分类</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status === 'enabled' ? 'green' : 'gray'">
                {{ record.status === 'enabled' ? '启用' : '停用' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="参考图完整度" :width="220">
            <template #cell="{ record }">
              <div class="progress-cell">
                <a-progress :percent="assetCompletion(record)" size="small" />
                <span>{{ countFilledAssets(record) }}/{{ assetSlots.length || 10 }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="排序" data-index="sort" :width="80" />
          <a-table-column title="更新时间" :width="180">
            <template #cell="{ record }">
              {{ formatTime(record.updated_at || record.created_at) }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="230" fixed="right">
            <template #cell="{ record }">
              <a-button type="text" size="small" @click="openEdit(record)">
                <template #icon><icon-edit /></template>
                编辑
              </a-button>
              <a-button type="text" size="small" @click="openEdit(record, true)">
                <template #icon><icon-image /></template>
                参考图
              </a-button>
              <a-popconfirm content="确定删除该参考图方案吗？" @ok="deleteGallery(record.id)">
                <a-button type="text" status="danger" size="small">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-popconfirm>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-drawer
      :visible="drawerVisible"
      placement="right"
      :width="880"
      unmount-on-close
      ok-text="确定"
      cancel-text="取消"
      :confirm-loading="submitLoading"
      @ok="saveGallery"
      @cancel="closeDrawer"
    >
      <template #title>
        <div class="drawer-head">
          <div>
            <div class="drawer-title">{{ form.id ? '编辑参考图方案' : '新建参考图方案' }}</div>
            <div class="drawer-subtitle">先维护方案信息，再配置每个用途对应的参考图。</div>
          </div>
        </div>
      </template>

      <a-tabs v-model:active-key="activeTab" class="drawer-tabs">
        <a-tab-pane key="base" title="方案信息">
          <a-form ref="formRef" :model="form" layout="vertical">
            <a-row :gutter="16">
              <a-col :span="14">
                <a-form-item field="name" label="方案名称" :rules="[{ required: true, message: '请输入方案名称' }]">
                  <a-input v-model="form.name" placeholder="例如：厨房龙头 Amazon 差异化方案" />
                </a-form-item>
              </a-col>
              <a-col :span="10">
                <a-form-item field="category" label="适用类目">
                  <a-select v-model="form.category" placeholder="请选择类目" allow-clear>
                    <a-option v-for="item in categoryOptions" :key="item.value" :value="item.value">
                      {{ item.label }}
                    </a-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="status" label="状态">
                  <a-select v-model="form.status">
                    <a-option value="enabled">启用</a-option>
                    <a-option value="disabled">停用</a-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="sort" label="排序">
                  <a-input-number v-model="form.sort" :min="0" hide-button :style="{ width: '100%' }" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="assets" title="参考图配置">
          <div class="asset-toolbar">
            <div>
              <strong>参考图槽位</strong>
              <span>{{ filledAssetCount }}/{{ assetSlots.length }} 已配置</span>
            </div>
            <a-progress :percent="completionPercent" size="small" class="asset-progress" />
          </div>

          <section v-for="group in groupedSlots" :key="group.name" class="asset-group">
            <div class="group-title">{{ group.name }}</div>
            <div class="asset-grid">
              <article v-for="slot in group.items" :key="slot.key" class="asset-item">
                <button class="asset-preview" type="button" @click="preview(assetMap[slot.key]?.url)">
                  <img v-if="assetMap[slot.key]?.url" :src="resolveImageUrl(assetMap[slot.key].url)" :alt="slot.label">
                  <span v-else>未上传</span>
                </button>
                <div class="asset-info">
                  <div class="asset-title">
                    <strong>{{ slot.label }}</strong>
                    <a-tag size="small" :color="assetMap[slot.key]?.url ? 'green' : 'gray'">
                      {{ assetMap[slot.key]?.url ? '已配置' : '缺图' }}
                    </a-tag>
                  </div>
                  <a-space>
                    <a-button size="mini" @click="triggerUpload(slot.key)">
                      <template #icon><icon-upload /></template>
                      上传
                    </a-button>
                    <a-button size="mini" @click="preview(assetMap[slot.key]?.url)" :disabled="!assetMap[slot.key]?.url">预览</a-button>
                    <a-button size="mini" status="danger" @click="clearAsset(slot.key)" :disabled="!assetMap[slot.key]?.url">清空</a-button>
                  </a-space>
                </div>
              </article>
            </div>
          </section>
        </a-tab-pane>
      </a-tabs>
    </a-drawer>

    <input ref="fileInputRef" class="hidden-file" type="file" accept="image/*" @change="handleFileChange" />

    <a-modal
      v-model:visible="previewVisible"
      :footer="false"
      :closable="false"
      :simple="true"
      :modal-style="{ background: 'transparent', boxShadow: 'none' }"
      :body-style="{ padding: '0', background: 'transparent' }"
      width="auto"
      @cancel="previewUrl = ''"
    >
      <img v-if="previewUrl" :src="previewUrl" alt="预览" class="preview-image" />
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconUpload, IconEdit, IconDelete, IconImage } from '@arco-design/web-vue/es/icon'
import request, { resolveImageUrl } from '@/utils/request'

const loading = ref(false)
const submitLoading = ref(false)
const globalAssetSaving = ref(false)
const drawerVisible = ref(false)
const activeTab = ref('base')
const galleries = ref([])
const assetSlots = ref([])
const assetMap = reactive({})
const globalAssetMap = reactive({})
const formRef = ref(null)
const fileInputRef = ref(null)
const uploadTargetKey = ref('')
const uploadTargetScope = ref('scheme')
const previewUrl = ref('')
const keyword = ref('')
const statusFilter = ref('')
const previewVisible = computed(() => !!previewUrl.value)
const categoryOptions = [
  { label: '厨房', value: 'kitchen' },
  { label: '浴室', value: 'bathroom' },
]
const globalMainAssetSlots = [
  {
    key: 'ecom_suite_androme_pipes_reference_url',
    label: '主图水管参考图',
    desc: 'Forious 和 Androme 主图生成两个水管时都会参考这张图',
  },
]

const form = reactive({
  id: null,
  name: '',
  category: '',
  version: '',
  description: '',
  status: 'enabled',
  sort: 0,
})

const filteredGalleries = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return galleries.value.filter(item => {
    const matchKeyword = !q || [item.name, item.category, categoryLabel(item.category)].some(value => String(value || '').toLowerCase().includes(q))
    const matchStatus = !statusFilter.value || item.status === statusFilter.value
    return matchKeyword && matchStatus
  })
})

const groupedSlots = computed(() => {
  const groups = [
    { name: '功能展示', keys: ['feature_pull_360', 'feature_pull', 'feature_360'] },
    { name: '水流模式', keys: ['water_flow_single', 'water_flow_dual', 'water_flow_triple'] },
    { name: '版式参考', keys: ['feature_selling_point', 'dimension', 'family', 'multi_color_white_bg'] },
  ]
  return groups.map(group => ({
    ...group,
    items: group.keys.map(key => assetSlots.value.find(slot => slot.key === key)).filter(Boolean),
  })).filter(group => group.items.length)
})

const filledAssetCount = computed(() => assetSlots.value.filter(slot => assetMap[slot.key]?.url).length)
const completionPercent = computed(() => {
  if (!assetSlots.value.length) return 0
  return Math.round((filledAssetCount.value / assetSlots.value.length) * 100) / 100
})

function resetForm() {
  form.id = null
  form.name = ''
  form.category = ''
  form.version = ''
  form.description = ''
  form.status = 'enabled'
  form.sort = 0
  formRef.value?.clearValidate()
  resetAssets()
}

function resetAssets(assets = []) {
  for (const slot of assetSlots.value) {
    const matched = assets.find(item => item.key === slot.key)
    assetMap[slot.key] = {
      key: slot.key,
      url: matched?.url || '',
      description: slot.label,
    }
  }
}

function countFilledAssets(item) {
  return Array.isArray(item.assets) ? item.assets.filter(asset => asset.url).length : 0
}

function assetCompletion(item) {
  if (!assetSlots.value.length) return 0
  return Math.round((countFilledAssets(item) / assetSlots.value.length) * 100) / 100
}

function formatTime(value) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function categoryLabel(value) {
  const matched = categoryOptions.find(item => item.value === value || item.label === value)
  return matched?.label || value || '未分类'
}

function categoryValue(value) {
  const matched = categoryOptions.find(item => item.value === value || item.label === value)
  return matched?.value || ''
}

async function fetchAssetSlots() {
  const res = await request.get('/reference-gallery-asset-keys')
  assetSlots.value = Array.isArray(res.data) ? res.data : []
  resetAssets()
}

async function fetchData() {
  loading.value = true
  try {
    const res = await request.get('/admin/reference-galleries')
    galleries.value = Array.isArray(res.data) ? res.data : []
  } finally {
    loading.value = false
  }
}

async function fetchGlobalMainAssets() {
  const res = await request.get('/settings')
  const settings = res.data || {}
  for (const slot of globalMainAssetSlots) {
    globalAssetMap[slot.key] = settings[slot.key] || ''
  }
}

function fillForm(item) {
  form.id = item.id
  form.name = item.name || ''
  form.category = categoryValue(item.category)
  form.version = item.version || ''
  form.description = ''
  form.status = item.status || 'enabled'
  form.sort = item.sort || 0
  resetAssets(Array.isArray(item.assets) ? item.assets : [])
}

function openCreate() {
  resetForm()
  activeTab.value = 'base'
  drawerVisible.value = true
}

function openEdit(record, showAssets = false) {
  fillForm(record)
  activeTab.value = showAssets ? 'assets' : 'base'
  drawerVisible.value = true
}

function closeDrawer() {
  drawerVisible.value = false
}

function buildPayload() {
  return {
    name: form.name,
    category: form.category,
    version: form.version,
    description: '',
    status: form.status,
    sort: Number(form.sort) || 0,
    assets: assetSlots.value
      .map(slot => ({
        key: slot.key,
        url: String(assetMap[slot.key]?.url || '').trim(),
        description: slot.label,
      }))
      .filter(item => item.url),
  }
}

async function saveGallery() {
  const err = await formRef.value?.validate()
  if (err) return
  submitLoading.value = true
  try {
    if (form.id) {
      await request.put(`/admin/reference-galleries/${form.id}`, buildPayload())
      Message.success('参考图方案更新成功')
    } else {
      await request.post('/admin/reference-galleries', buildPayload())
      Message.success('参考图方案创建成功')
    }
    drawerVisible.value = false
    await fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function deleteGallery(id) {
  await request.delete(`/admin/reference-galleries/${id}`)
  Message.success('参考图方案已删除')
  await fetchData()
}

function triggerUpload(key, scope = 'scheme') {
  uploadTargetKey.value = key
  uploadTargetScope.value = scope
  fileInputRef.value?.click()
}

function handleFileChange(event) {
  const file = event.target.files?.[0]
  if (event.target) event.target.value = ''
  if (!file || !uploadTargetKey.value) return
  if (!file.type.startsWith('image/')) {
    Message.error('请上传图片文件')
    return
  }
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const res = await request.post('/upload/image', {
        data: String(e.target?.result || ''),
        mime_type: file.type || 'image/png',
      })
      const url = res.data?.url || ''
      if (uploadTargetScope.value === 'global') {
        globalAssetMap[uploadTargetKey.value] = url
      } else {
        assetMap[uploadTargetKey.value].url = url
      }
      Message.success('图片上传成功')
    } catch {
      Message.error('图片上传失败')
    }
  }
  reader.readAsDataURL(file)
}

function clearAsset(key) {
  assetMap[key].url = ''
}

function clearGlobalAsset(key) {
  globalAssetMap[key] = ''
}

async function saveGlobalMainAssets() {
  globalAssetSaving.value = true
  try {
    const payload = {}
    for (const slot of globalMainAssetSlots) {
      payload[slot.key] = String(globalAssetMap[slot.key] || '').trim()
    }
    await request.post('/settings', payload)
    Message.success('全局主图素材已保存')
  } finally {
    globalAssetSaving.value = false
  }
}

function preview(url) {
  if (!url) return
  previewUrl.value = resolveImageUrl(url)
}

onMounted(async () => {
  await fetchAssetSlots()
  await fetchGlobalMainAssets()
  await fetchData()
})
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 18px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-1);
}

.page-subtitle {
  margin: 6px 0 0;
  color: var(--color-text-3);
  font-size: 13px;
}

.filter-card {
  margin-bottom: 14px;
}

.global-assets-card {
  margin-bottom: 14px;
}

.global-assets-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 14px;
}

.global-assets-head > div {
  display: grid;
  gap: 5px;
}

.global-assets-head strong {
  color: var(--color-text-1);
  font-size: 15px;
}

.global-assets-head span {
  color: var(--color-text-3);
  font-size: 12px;
}

.global-assets-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.global-asset-item {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: #fbfcff;
}

.global-asset-preview {
  display: grid;
  place-items: center;
  width: 104px;
  aspect-ratio: 1;
  overflow: hidden;
  border: 0;
  border-radius: 8px;
  background: #f2f3f5;
  color: var(--color-text-3);
  cursor: pointer;
}

.global-asset-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.global-asset-info {
  display: grid;
  align-content: space-between;
  gap: 12px;
  min-width: 0;
}

.global-asset-info strong {
  color: var(--color-text-1);
}

.global-asset-info p {
  margin: 5px 0 0;
  color: var(--color-text-3);
  font-size: 12px;
  line-height: 1.5;
}

.search-input {
  width: 280px;
}

.status-select {
  width: 130px;
}

.scheme-name-cell {
  display: grid;
  gap: 4px;
}

.scheme-name-cell strong {
  color: var(--color-text-1);
}

.scheme-name-cell span,
.text-secondary {
  color: var(--color-text-3);
  font-size: 12px;
}

.progress-cell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 48px;
  align-items: center;
  gap: 10px;
}

.drawer-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.drawer-title {
  color: var(--color-text-1);
  font-size: 16px;
  font-weight: 700;
}

.drawer-subtitle {
  margin-top: 4px;
  color: var(--color-text-3);
  font-size: 12px;
}

.drawer-tabs {
  padding-bottom: 8px;
}

.asset-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.asset-toolbar > div {
  display: grid;
  gap: 4px;
}

.asset-toolbar span {
  color: var(--color-text-3);
  font-size: 12px;
}

.asset-progress {
  width: 220px;
}

.asset-group {
  display: grid;
  gap: 10px;
  margin-bottom: 18px;
}

.group-title {
  color: var(--color-text-1);
  font-weight: 700;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.asset-item {
  display: grid;
  grid-template-columns: 118px minmax(0, 1fr);
  gap: 12px;
  padding: 10px;
  border: 1px solid var(--color-border-2);
  border-radius: 12px;
}

.asset-preview {
  display: grid;
  place-items: center;
  width: 118px;
  aspect-ratio: 1;
  overflow: hidden;
  border: 0;
  border-radius: 10px;
  background: #f2f3f5;
  color: var(--color-text-3);
  cursor: pointer;
}

.asset-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-info {
  display: grid;
  align-content: start;
  gap: 9px;
}

.asset-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.asset-title strong {
  min-width: 0;
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hidden-file {
  display: none;
}

.preview-image {
  display: block;
  width: auto;
  max-width: min(72vw, 920px);
  max-height: 70vh;
  object-fit: contain;
  border-radius: 14px;
  background: transparent;
}

@media (max-width: 1180px) {
  .global-assets-grid {
    grid-template-columns: 1fr;
  }

  .asset-grid {
    grid-template-columns: 1fr;
  }
}
</style>
