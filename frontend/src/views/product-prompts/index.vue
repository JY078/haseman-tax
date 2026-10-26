<template>
  <div class="catalog-page">
    <header class="catalog-toolbar">
      <div class="toolbar-title">
        <strong>产品库</strong>
        <span>按 SPU 管理模型、认证资料和下属 SKU</span>
      </div>

      <div class="toolbar-actions">
        <a-input-search
          v-model="spuKeyword"
          class="toolbar-search"
          placeholder="搜索 SPU / 品牌 / 型号"
          allow-clear
          @search="handleSearch"
          @clear="handleSearch"
        />
        <a-button type="primary" @click="openSpuEditor()">
          <template #icon><icon-plus /></template>
          新建 SPU
        </a-button>
        <a-button @click="fetchSpus">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
      </div>
    </header>

    <section class="summary-strip">
      <div class="summary-item">
        <span>SPU</span>
        <strong>{{ spuPagination.total }}</strong>
      </div>
      <div class="summary-item">
        <span>SKU</span>
        <strong>{{ skuCount }}</strong>
      </div>
      <div class="summary-item">
        <span>模型</span>
        <strong>{{ modelCount }}</strong>
      </div>
      <div class="summary-item">
        <span>认证</span>
        <strong>{{ attachmentCount }}</strong>
      </div>
    </section>

    <section class="catalog-list">
      <div class="list-head">
        <div></div>
        <div>SPU / 品名</div>
        <div>品牌 / 型号</div>
        <div>分类</div>
        <div>资料</div>
        <div>SKU</div>
        <div>操作</div>
      </div>

      <div v-if="spuLoading" class="loading-row">正在加载产品库...</div>

      <template v-else>
        <div v-for="spu in spuItems" :key="spu.id" class="spu-group">
          <div class="spu-list-row" :class="{ expanded: expandedSpuId === spu.id }">
            <button
              type="button"
              class="expand-btn"
              :class="{ open: expandedSpuId === spu.id }"
              @click="toggleSpuExpand(spu)"
            >
              <icon-down />
            </button>

            <button type="button" class="spu-code" @click="toggleSpuExpand(spu)">
              <strong>{{ spu.spu }}</strong>
              <span>{{ spu.name || '-' }}</span>
            </button>

            <div class="cell-stack">
              <strong>{{ spu.brand || '-' }}</strong>
              <span>{{ [spu.model, spu.material].filter(Boolean).join(' / ') || '-' }}</span>
            </div>

            <div class="cell-stack">
              <span>{{ spu.category1 || '-' }}</span>
              <span>{{ spu.category2 || '-' }}</span>
            </div>

            <div class="asset-cell">
              <a-tag size="small" :color="spu.model_url ? 'green' : 'gray'">
                {{ spu.model_url ? '有模型' : '无模型' }}
              </a-tag>
              <a-tag size="small" :color="(spu.attachments || []).length ? 'arcoblue' : 'gray'">
                {{ (spu.attachments || []).length }} 认证
              </a-tag>
            </div>

            <div class="cell-muted">{{ spu.sku_count || spu.skus?.length || 0 }} 个</div>

            <div class="row-actions">
              <a-button class="row-action-btn" size="small" @click="openSpuEditor(spu)">编辑</a-button>
              <a-button class="row-action-btn primary" size="small" @click="openSkuEditor(spu)">新增 SKU</a-button>
              <a-popconfirm
                content="确定删除这个 SPU 吗？该 SPU 下的 SKU 和认证资料也会一起删除。"
                @ok="deleteSpu(spu)"
              >
                <a-button class="row-action-btn danger" size="small">删除</a-button>
              </a-popconfirm>
            </div>
          </div>

          <div v-show="expandedSpuId === spu.id" class="sku-panel">
            <div v-if="getFilteredSkus(spu).length === 0" class="sku-empty-row">暂无 SKU</div>
            <div v-for="sku in getFilteredSkus(spu)" :key="sku.id" class="sku-list-row">
              <div class="cell-stack">
                <strong>{{ sku.sku || '-' }}</strong>
                <span>{{ sku.name || '-' }}</span>
              </div>
              <div class="cell-muted">{{ sku.style_name || '-' }}</div>
              <div class="cell-muted">{{ [sku.color, sku.size].filter(Boolean).join(' / ') || '-' }}</div>
              <div class="cell-stack">
                <span>{{ sku.feature || '-' }}</span>
                <span>{{ sku.package_qty || '-' }}</span>
              </div>
              <div class="sku-actions">
                <a-button class="sku-action-btn" size="mini" @click="openSkuEditor(spu, sku)">编辑</a-button>
                <a-popconfirm content="确定删除这个 SKU 吗？" @ok="deleteSku(sku)">
                  <a-button class="sku-action-btn danger" size="mini">删除</a-button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <a-empty v-if="spuItems.length === 0" description="暂无 SPU" />
      </template>
    </section>

    <div class="pager">
      <div class="pager-info">
        共 {{ spuPagination.total }} 条，每页 {{ spuPagination.pageSize }} 条
      </div>
      <a-pagination
        :total="spuPagination.total"
        :current="spuPagination.current"
        :page-size="spuPagination.pageSize"
        size="small"
        show-total
        show-jumper
        @change="handlePageChange"
      />
    </div>

    <a-drawer :visible="spuEditorVisible" placement="right" :width="820" unmount-on-close @cancel="spuEditorVisible = false">
      <template #title>
        <div class="drawer-head">
          <div>
            <div class="drawer-title">{{ spuEditorForm.id ? '编辑 SPU' : '新建 SPU' }}</div>
            <div class="drawer-subtitle">维护 SPU 基础信息、产品模型和认证资料</div>
          </div>
        </div>
      </template>

      <a-form ref="spuFormRef" :model="spuEditorForm" layout="vertical">
        <section class="drawer-section">
          <h3>基础信息</h3>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="spu" label="SPU" :rules="[{ required: true, message: '请输入 SPU' }]">
                <a-input v-model="spuEditorForm.spu" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="name" label="品名">
                <a-input v-model="spuEditorForm.name" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="brand" label="品牌">
                <a-input v-model="spuEditorForm.brand" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="model" label="型号">
                <a-input v-model="spuEditorForm.model" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item field="material" label="产品材质">
                <a-input v-model="spuEditorForm.material" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item field="category1" label="一级分类">
                <a-input v-model="spuEditorForm.category1" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item field="category2" label="二级分类">
                <a-input v-model="spuEditorForm.category2" />
              </a-form-item>
            </a-col>
          </a-row>
        </section>

        <section v-if="spuEditorForm.id" class="drawer-section">
          <div class="section-title-row">
            <h3>产品模型</h3>
            <a-button size="small" @click="triggerSpuFile({ id: spuEditorForm.id }, 'model')">
              <template #icon><icon-upload /></template>
              {{ spuEditorForm.model_url ? '更换模型' : '上传模型' }}
            </a-button>
          </div>
          <div v-if="spuEditorForm.model_url" class="file-line">
            <span>{{ getFileName(spuEditorForm.model_url) }}</span>
            <a-button type="text" size="small" @click="openFile(spuEditorForm.model_url)">查看</a-button>
          </div>
          <div v-else class="empty-note">支持 `.stp` / `.step` / `.obj` / `.fbx`。</div>
        </section>

        <section v-if="spuEditorForm.id" class="drawer-section">
          <div class="section-title-row">
            <h3>认证资料</h3>
            <a-button size="small" @click="openAttachmentCreate({ id: spuEditorForm.id })">
              <template #icon><icon-upload /></template>
              新增认证
            </a-button>
          </div>

          <div class="attachment-list">
            <div v-for="attachment in spuEditorForm.attachments || []" :key="attachment.id" class="attachment-item">
              <div class="attachment-meta">
                <a-tag size="small" color="arcoblue">{{ attachment.label || '未命名标签' }}</a-tag>
                <span>{{ attachment.file_name || getFileName(attachment.file_url) }}</span>
              </div>
              <div class="attachment-actions">
                <a-button type="text" size="small" @click="openFile(attachment.file_url)">查看</a-button>
                <a-button type="text" size="small" @click="openAttachmentLabelEditor({ id: spuEditorForm.id }, attachment)">改标签</a-button>
                <a-popconfirm content="确定删除这条认证资料吗？" @ok="deleteAttachment({ id: spuEditorForm.id }, attachment)">
                  <a-button type="text" size="small" status="danger">删除</a-button>
                </a-popconfirm>
              </div>
            </div>
            <div v-if="(spuEditorForm.attachments || []).length === 0" class="empty-note">还没有上传认证资料。</div>
          </div>
        </section>
      </a-form>

      <div class="drawer-footer">
        <a-button @click="spuEditorVisible = false">取消</a-button>
        <a-button type="primary" :loading="spuSubmitLoading" @click="saveSpu">确定</a-button>
      </div>
    </a-drawer>

    <a-drawer :visible="skuEditorVisible" placement="right" :width="860" unmount-on-close @cancel="skuEditorVisible = false">
      <template #title>
        <div class="drawer-head">
          <div>
            <div class="drawer-title">{{ skuEditorForm.id ? '编辑 SKU' : '新建 SKU' }}</div>
            <div class="drawer-subtitle">SKU 归属当前 SPU，模型与认证资料由 SPU 统一维护</div>
          </div>
        </div>
      </template>

      <a-form ref="skuFormRef" :model="skuEditorForm" layout="vertical">
        <section class="drawer-section">
          <h3>基础信息</h3>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="sku" label="SKU" :rules="[{ required: true, message: '请输入 SKU' }]">
                <a-input v-model="skuEditorForm.sku" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="name" label="名称">
                <a-input v-model="skuEditorForm.name" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="spu" label="SPU">
                <a-input v-model="skuEditorForm.spu" disabled />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="style_name" label="款名">
                <a-input v-model="skuEditorForm.style_name" />
              </a-form-item>
            </a-col>
          </a-row>
        </section>

        <section class="drawer-section">
          <h3>规格</h3>
          <a-row :gutter="16">
            <a-col :span="8"><a-form-item field="color" label="颜色"><a-input v-model="skuEditorForm.color" /></a-form-item></a-col>
            <a-col :span="8"><a-form-item field="size" label="尺寸"><a-input v-model="skuEditorForm.size" /></a-form-item></a-col>
            <a-col :span="8"><a-form-item field="feature" label="功能"><a-input v-model="skuEditorForm.feature" /></a-form-item></a-col>
          </a-row>
        </section>

        <section class="drawer-section">
          <h3>包装与价格</h3>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="package_qty" label="包装数量"><a-input v-model="skuEditorForm.package_qty" /></a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="listed_at" label="上架时间">
                <a-date-picker v-model="skuEditorForm.listed_at" style="width: 100%" format="YYYY-MM-DD" value-format="YYYY-MM-DD" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item field="category3" label="三级分类"><a-input v-model="skuEditorForm.category3" /></a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="purchase_price" label="采购价格">
                <a-input-number
                  :model-value="skuEditorForm.purchase_price === '' ? undefined : Number(skuEditorForm.purchase_price)"
                  :min="0"
                  :precision="2"
                  style="width: 100%"
                  @update:model-value="setSkuNumberField('purchase_price', $event)"
                >
                  <template #prefix>￥</template>
                </a-input-number>
              </a-form-item>
            </a-col>
          </a-row>
        </section>
      </a-form>

      <div class="drawer-footer">
        <a-button @click="skuEditorVisible = false">取消</a-button>
        <a-button type="primary" :loading="skuSubmitLoading" @click="saveSku">确定</a-button>
      </div>
    </a-drawer>

    <a-modal
      v-model:visible="attachmentEditorVisible"
      title="认证标签"
      @ok="saveAttachmentLabel"
      @cancel="attachmentEditorVisible = false"
    >
      <a-form layout="vertical">
        <a-form-item label="标签">
          <a-input v-model="attachmentEditorForm.label" placeholder="例如：FCC / CE / 说明书" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="attachmentCreateVisible"
      title="新增认证资料"
      :confirm-loading="attachmentUploadLoading"
      @ok="uploadAttachment"
      @cancel="attachmentCreateVisible = false"
    >
      <a-form layout="vertical">
        <a-form-item label="标签">
          <a-input v-model="attachmentCreateForm.label" placeholder="例如：FCC / CE / 说明书" />
        </a-form-item>
        <a-form-item label="文件">
          <input ref="attachmentInputRef" class="file-input" type="file" accept="image/*,.pdf" @change="handleAttachmentFileChange" />
          <div class="upload-hint">{{ attachmentCreateForm.file?.name || '支持图片或 PDF' }}</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="previewVisible"
      :footer="false"
      :width="previewWidth"
      :body-style="{ padding: '0' }"
      @cancel="previewVisible = false"
    >
      <div class="preview-shell">
        <img v-if="previewType === 'image'" :src="previewUrl" class="preview-image" alt="预览" />
        <iframe v-else-if="previewType === 'pdf'" :src="previewUrl" class="preview-frame" />
        <div v-else class="preview-fallback">
          <p>当前文件暂不支持内嵌预览。</p>
          <a-button type="primary" @click="openFile(previewUrl)">新窗口打开</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import request, { resolveImageUrl } from '@/utils/request';
import { IconDown, IconPlus, IconRefresh, IconUpload } from '@arco-design/web-vue/es/icon';

const spuItems = ref([]);
const spuLoading = ref(false);
const spuKeyword = ref('');
const expandedSpuId = ref(null);
const skuKeyword = ref('');
const spuFormRef = ref(null);
const skuFormRef = ref(null);
const spuSubmitLoading = ref(false);
const skuSubmitLoading = ref(false);
const spuEditorVisible = ref(false);
const skuEditorVisible = ref(false);
const attachmentEditorVisible = ref(false);
const attachmentCreateVisible = ref(false);
const attachmentUploadLoading = ref(false);
const previewVisible = ref(false);
const previewUrl = ref('');
const previewType = ref('');
const previewWidth = ref(900);

const spuPagination = reactive({ current: 1, pageSize: 12, total: 0 });
const spuEditorForm = reactive(createDefaultSpuForm());
const skuEditorForm = reactive(createDefaultSkuForm());
const attachmentEditorForm = reactive({ spuId: null, id: null, label: '' });
const attachmentCreateForm = reactive({ spuId: null, label: '', file: null });

const skuCount = computed(() => spuItems.value.reduce((sum, item) => sum + (item.sku_count || item.skus?.length || 0), 0));
const modelCount = computed(() => spuItems.value.filter((item) => item.model_url).length);
const attachmentCount = computed(() => spuItems.value.reduce((sum, item) => sum + (item.attachments?.length || 0), 0));

function createDefaultSpuForm() {
  return {
    id: null,
    spu: '',
    name: '',
    brand: '',
    model: '',
    material: '',
    category1: '',
    category2: '',
    model_url: '',
    attachments: [],
  };
}

function createDefaultSkuForm() {
  return {
    id: null,
    spu_id: null,
    sku: '',
    spu: '',
    name: '',
    style_name: '',
    color: '',
    size: '',
    feature: '',
    package_qty: '',
    material: '',
    category1: '',
    category2: '',
    category3: '',
    listed_at: '',
    package_length: '',
    package_width: '',
    package_height: '',
    package_weight: '',
    purchase_price: '',
  };
}

function assignSpuForm(record = {}) {
  Object.assign(spuEditorForm, createDefaultSpuForm(), record, {
    attachments: Array.isArray(record.attachments) ? record.attachments : [],
  });
}

function assignSkuForm(spu, record = {}) {
  Object.assign(skuEditorForm, createDefaultSkuForm(), record, {
    spu_id: spu?.id || record.spu_id || null,
    spu: spu?.spu || record.spu || '',
  });
}

async function fetchSpus() {
  spuLoading.value = true;
  try {
    const res = await request.get('/admin/product-spus', {
      params: {
        keyword: spuKeyword.value,
        page: spuPagination.current,
        page_size: spuPagination.pageSize,
      },
    });
    spuItems.value = res.data?.items || [];
    spuPagination.total = res.data?.total || 0;
    if (expandedSpuId.value && !spuItems.value.some((item) => item.id === expandedSpuId.value)) {
      expandedSpuId.value = null;
    }
    syncOpenSpuEditorData();
  } finally {
    spuLoading.value = false;
  }
}

function syncOpenSpuEditorData() {
  if (!spuEditorVisible.value || !spuEditorForm.id) return;
  const current = spuItems.value.find((item) => item.id === spuEditorForm.id);
  if (current) assignSpuForm(current);
}

function handleSearch() {
  spuPagination.current = 1;
  fetchSpus();
}

function handlePageChange(page) {
  spuPagination.current = page;
  fetchSpus();
}

function toggleSpuExpand(spu) {
  expandedSpuId.value = expandedSpuId.value === spu.id ? null : spu.id;
  skuKeyword.value = '';
}

function openSpuEditor(record = null) {
  assignSpuForm(record || {});
  spuEditorVisible.value = true;
}

async function saveSpu() {
  const err = await spuFormRef.value?.validate();
  if (err) return;
  spuSubmitLoading.value = true;
  try {
    const payload = {
      spu: spuEditorForm.spu,
      name: spuEditorForm.name,
      brand: spuEditorForm.brand,
      model: spuEditorForm.model,
      material: spuEditorForm.material,
      category1: spuEditorForm.category1,
      category2: spuEditorForm.category2,
    };
    let res;
    if (spuEditorForm.id) {
      res = await request.put(`/admin/product-spus/${spuEditorForm.id}`, payload);
      Message.success('SPU 更新成功');
    } else {
      res = await request.post('/admin/product-spus', payload);
      Message.success('SPU 创建成功');
    }
    spuEditorVisible.value = false;
    await fetchSpus();
    if (res?.data?.id) expandedSpuId.value = res.data.id;
  } finally {
    spuSubmitLoading.value = false;
  }
}

async function deleteSpu(spu) {
  if (!spu?.id) return;
  await request.delete(`/admin/product-spus/${spu.id}`);
  if (expandedSpuId.value === spu.id) {
    expandedSpuId.value = null;
  }
  Message.success('SPU 已删除');
  await fetchSpus();
}

function openSkuEditor(spu, record = null) {
  if (!spu) {
    Message.warning('请先选择 SPU');
    return;
  }
  assignSkuForm(spu, record || {});
  skuEditorVisible.value = true;
}

async function saveSku() {
  const err = await skuFormRef.value?.validate();
  if (err) return;
  skuSubmitLoading.value = true;
  try {
    const payload = { ...skuEditorForm };
    if (skuEditorForm.id) {
      await request.put(`/admin/product-skus/${skuEditorForm.id}`, payload);
      Message.success('SKU 更新成功');
    } else {
      await request.post('/admin/product-skus', payload);
      Message.success('SKU 创建成功');
    }
    skuEditorVisible.value = false;
    await fetchSpus();
  } finally {
    skuSubmitLoading.value = false;
  }
}

async function deleteSku(sku) {
  if (!sku?.id) return;
  await request.delete(`/admin/product-skus/${sku.id}`);
  Message.success('SKU 已删除');
  await fetchSpus();
}

function openAttachmentCreate(spu) {
  attachmentCreateForm.spuId = spu.id;
  attachmentCreateForm.label = '';
  attachmentCreateForm.file = null;
  attachmentCreateVisible.value = true;
}

function handleAttachmentFileChange(event) {
  attachmentCreateForm.file = event.target.files?.[0] || null;
}

function openAttachmentLabelEditor(spu, record) {
  attachmentEditorForm.spuId = spu.id;
  attachmentEditorForm.id = record.id;
  attachmentEditorForm.label = record.label || '';
  attachmentEditorVisible.value = true;
}

async function saveAttachmentLabel() {
  if (!attachmentEditorForm.spuId || !attachmentEditorForm.id) return;
  await request.put(`/admin/product-spus/${attachmentEditorForm.spuId}/attachments/${attachmentEditorForm.id}`, {
    label: attachmentEditorForm.label,
  });
  attachmentEditorVisible.value = false;
  await fetchSpus();
  Message.success('标签已更新');
}

async function uploadAttachment() {
  if (!attachmentCreateForm.spuId) return;
  if (!attachmentCreateForm.label.trim()) {
    Message.error('请先填写标签');
    return;
  }
  if (!attachmentCreateForm.file) {
    Message.error('请选择文件');
    return;
  }
  attachmentUploadLoading.value = true;
  try {
    const fd = new FormData();
    fd.append('file', attachmentCreateForm.file);
    fd.append('label', attachmentCreateForm.label);
    await request.post(`/admin/product-spus/${attachmentCreateForm.spuId}/attachments`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    attachmentCreateVisible.value = false;
    await fetchSpus();
    Message.success('认证资料已上传');
  } finally {
    attachmentUploadLoading.value = false;
  }
}

async function deleteAttachment(spu, record) {
  await request.delete(`/admin/product-spus/${spu.id}/attachments/${record.id}`);
  await fetchSpus();
  Message.success('认证资料已删除');
}

async function triggerSpuFile(spu, kind) {
  if (!spu?.id || kind !== 'model') return;
  const input = document.createElement('input');
  input.type = 'file';
  input.onchange = async () => {
    const file = input.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('file', file);
    await request.post(`/admin/product-spus/${spu.id}/model`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    await fetchSpus();
    Message.success('模型已更新');
  };
  input.click();
}

function getFilteredSkus(spu) {
  const items = Array.isArray(spu?.skus) ? spu.skus : [];
  const keyword = skuKeyword.value.trim().toLowerCase();
  if (!keyword) return items;
  return items.filter((item) => [
    item.sku,
    item.name,
    item.style_name,
    item.color,
    item.size,
    item.material,
  ].some((value) => String(value || '').toLowerCase().includes(keyword)));
}

function openFile(url) {
  if (!url) return;
  const resolved = resolveImageUrl(url);
  const type = getFileType(url);
  previewUrl.value = resolved;
  previewType.value = type;
  previewWidth.value = type === 'image' ? 980 : 920;
  if (type === 'file') {
    window.open(resolved, '_blank');
    return;
  }
  previewVisible.value = true;
}

function getFileType(url) {
  const lower = String(url || '').toLowerCase();
  if (lower.match(/\.(png|jpg|jpeg|gif|webp|bmp)(\?|$)/)) return 'image';
  if (lower.match(/\.pdf(\?|$)/)) return 'pdf';
  return 'file';
}

function getFileName(url) {
  if (!url) return '';
  try {
    return new URL(url).pathname.split('/').pop() || url;
  } catch {
    return url.split('/').pop() || url;
  }
}

function setSkuNumberField(field, value) {
  skuEditorForm[field] = value === null || value === undefined ? '' : String(value);
}

function noop() {}

onMounted(fetchSpus);
</script>

<style scoped>
.catalog-page {
  height: 100%;
  min-height: 0;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #f3f5f8;
  overflow: hidden;
  color: #1f2937;
}

.catalog-toolbar,
.summary-strip,
.catalog-list {
  border: 1px solid #dfe4ec;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.catalog-toolbar {
  flex: 0 0 auto;
  min-height: 64px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-radius: 8px;
}

.toolbar-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 240px;
}

.toolbar-title strong {
  font-size: 18px;
  line-height: 24px;
  color: #172033;
  letter-spacing: 0;
}

.toolbar-title span,
.cell-muted,
.upload-hint {
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-search {
  width: min(420px, 34vw);
}

.summary-strip {
  flex: 0 0 auto;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-radius: 8px;
  overflow: hidden;
}

.summary-item {
  padding: 11px 16px;
  border-right: 1px solid #e8edf4;
  background: linear-gradient(180deg, #fff 0%, #fbfcfe 100%);
}

.summary-item:last-child {
  border-right: 0;
}

.summary-item span {
  display: block;
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.summary-item strong {
  margin-top: 2px;
  display: block;
  font-size: 20px;
  line-height: 26px;
  color: #172033;
  font-weight: 700;
}

.catalog-list {
  flex: 1 1 auto;
  min-height: 0;
  border-radius: 8px;
  overflow: auto;
  scrollbar-gutter: stable;
}

.list-head,
.spu-list-row,
.sku-list-row,
.sku-empty-row {
  display: grid;
  grid-template-columns: 34px minmax(210px, 1.2fr) minmax(160px, 0.85fr) minmax(170px, 0.9fr) minmax(130px, 0.7fr) 70px 260px;
  align-items: center;
  column-gap: 12px;
  min-width: 1080px;
}

.list-head {
  position: sticky;
  top: 0;
  z-index: 2;
  height: 44px;
  padding: 0 16px;
  background: #f7f9fc;
  border-bottom: 1px solid #dfe4ec;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.spu-group {
  border-bottom: 1px solid #e8edf4;
  min-width: 1080px;
  background: #fff;
}

.spu-list-row {
  min-height: 58px;
  padding: 0 16px;
  background: #fff;
  transition: background-color 0.16s ease, box-shadow 0.16s ease;
}

.spu-list-row:hover {
  background: #fafcff;
}

.spu-list-row.expanded {
  background: #f8fbff;
  box-shadow: inset 3px 0 0 #2563eb;
}

.expand-btn {
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: #667085;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.18s ease, color 0.18s ease;
}

.expand-btn :deep(svg) {
  width: 14px;
  height: 14px;
  transform: rotate(-90deg);
  transition: transform 0.18s ease;
}

.expand-btn:hover {
  background: #eef4ff;
  color: #1d4ed8;
}

.expand-btn.open {
  background: #eff6ff;
  color: #1d4ed8;
}

.expand-btn.open :deep(svg) {
  transform: rotate(0deg);
}

.spu-code {
  border: 0;
  background: transparent;
  padding: 0;
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
  text-align: left;
  color: #172033;
  font-weight: 650;
  line-height: 20px;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spu-code strong,
.spu-code span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spu-code span {
  color: #667085;
  font-size: 12px;
  font-weight: 400;
}

.spu-code:hover {
  color: #1d4ed8;
}

.asset-cell,
.row-actions,
.attachment-actions,
.section-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.row-actions {
  justify-content: flex-end;
  flex-wrap: nowrap;
  min-width: 0;
  overflow: hidden;
}

.row-action-btn {
  flex: 0 0 auto;
  border-color: #cfd8e6;
  background: #fff;
  color: #344054;
  font-weight: 600;
  white-space: nowrap;
}

.row-action-btn:hover {
  border-color: #2563eb;
  color: #1d4ed8;
  background: #eff6ff;
}

.row-action-btn.primary {
  border-color: #2563eb;
  background: #2563eb;
  color: #fff;
}

.row-action-btn.primary:hover {
  border-color: #1d4ed8;
  background: #1d4ed8;
  color: #fff;
}

.row-action-btn.danger {
  border-color: #fecaca;
  background: #fff;
  color: #b42318;
}

.row-action-btn.danger:hover {
  border-color: #dc2626;
  background: #fef2f2;
  color: #b42318;
}

.asset-cell {
  gap: 6px;
}

.sku-panel {
  background: #fff;
  border-top: 0;
}

.sku-list-row,
.sku-empty-row {
  min-height: 48px;
  padding: 0 16px 0 62px;
  background: #fbfdff;
  border-top: 1px solid #e8edf4;
  box-shadow: inset 3px 0 0 #bfdbfe;
  grid-template-columns: minmax(210px, 1.2fr) minmax(160px, 0.85fr) minmax(170px, 0.9fr) minmax(130px, 0.7fr) 260px;
  min-width: 1080px;
}

.sku-list-row:hover {
  background: #f6faff;
}

.sku-empty-row {
  color: #667085;
}

.sku-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}

.sku-action-btn {
  flex: 0 0 auto;
  border-color: #cfd8e6;
  background: #fff;
  color: #344054;
  font-weight: 600;
  white-space: nowrap;
}

.sku-action-btn:hover {
  border-color: #2563eb;
  color: #1d4ed8;
  background: #eff6ff;
}

.sku-action-btn.danger {
  border-color: #fecaca;
  background: #fff;
  color: #b42318;
}

.sku-action-btn.danger:hover {
  border-color: #dc2626;
  background: #fef2f2;
  color: #b42318;
}

.loading-row {
  padding: 28px 16px;
  color: #667085;
}

.cell-stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cell-stack strong {
  color: #172033;
  font-size: 13px;
  line-height: 20px;
}

.cell-stack span {
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.pager {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
  padding: 8px 12px;
  border: 1px solid #dfe4ec;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.pager-info {
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.drawer-head {
  width: 100%;
}

.drawer-title {
  color: #172033;
  font-size: 18px;
  font-weight: 650;
  line-height: 24px;
}

.drawer-subtitle {
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.drawer-section {
  margin-bottom: 14px;
  padding: 16px;
  border: 1px solid #dfe4ec;
  border-radius: 8px;
  background: #fbfcfe;
}

.drawer-section h3 {
  margin: 0 0 12px;
  color: #172033;
  font-size: 14px;
  line-height: 20px;
  font-weight: 700;
}

.section-title-row {
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid #e8edf4;
}

.file-line,
.attachment-item {
  min-height: 42px;
  padding: 10px 12px;
  border: 1px solid #e8edf4;
  border-radius: 7px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.file-line span,
.attachment-meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-note {
  padding: 14px;
  border: 1px dashed #cbd5e1;
  border-radius: 7px;
  background: #fff;
  color: #667085;
  font-size: 12px;
  line-height: 18px;
}

.attachment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.attachment-meta {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
}

.attachment-actions {
  flex: 0 0 auto;
  justify-content: flex-end;
  flex-wrap: nowrap;
}

.drawer-footer {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid #dfe4ec;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background: #fff;
}

.file-input {
  width: 100%;
}

.preview-shell {
  max-height: 80vh;
  background: #fff;
}

.preview-image {
  display: block;
  width: 100%;
  max-height: 80vh;
  object-fit: contain;
}

.preview-frame {
  display: block;
  width: 100%;
  height: 80vh;
  border: 0;
}

.preview-fallback {
  min-height: 280px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
  justify-content: center;
}

:deep(.catalog-toolbar .arco-btn),
:deep(.drawer-footer .arco-btn),
:deep(.row-actions .arco-btn) {
  border-radius: 6px;
}

:deep(.arco-input-wrapper),
:deep(.arco-picker),
:deep(.arco-input-number) {
  border-radius: 6px;
}

@media (max-width: 1200px) {
  .catalog-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .toolbar-search {
    width: 100%;
  }

  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
