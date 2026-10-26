<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">部门管理</h1>
      <a-button type="primary" @click="openModal()">
        <template #icon><icon-plus /></template>
        新建部门
      </a-button>
    </div>

    <a-card>
      <a-space style="margin-bottom: 16px">
        <a-input-search
          v-model="searchKeyword"
          placeholder="搜索平台 / 一级部门 / 二级部门"
          style="width: 280px"
          allow-clear
        />
      </a-space>
      <a-table
        :data="treeRows"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 900 }"
        row-key="key"
      >
        <template #empty>
          <a-empty description="暂无部门，请点击右上角新建部门" />
        </template>
        <template #columns>
          <a-table-column title="部门结构" :width="360">
            <template #cell="{ record }">
              <div
                class="tree-name"
                :class="{ 'tree-name-active': record.key === activeKey }"
                :style="{ paddingLeft: `${record.level * 22}px` }"
              >
                <a-button
                  v-if="record.expandable"
                  type="text"
                  size="mini"
                  class="expand-button"
                  @click="toggleExpand(record.key)"
                >
                  <template #icon>
                    <icon-down v-if="expandedKeys.has(record.key)" />
                    <icon-right v-else />
                  </template>
                </a-button>
                <span v-else class="expand-placeholder"></span>
                <icon-folder v-if="record.type !== 'department'" class="tree-icon" />
                <icon-file v-else class="tree-icon tree-icon-file" />
                <span class="tree-title">{{ record.title }}</span>
                <a-tag v-if="record.type === 'platform'" size="small" color="arcoblue">
                  {{ record.count }} 个部门
                </a-tag>
                <a-tag v-if="record.type === 'primary'" size="small" color="purple">
                  {{ record.count }} 个二级部门
                </a-tag>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="平台" :width="130">
            <template #cell="{ record }">{{ record.platformLabel || '-' }}</template>
          </a-table-column>
          <a-table-column title="NAS 地区目录" :width="180">
            <template #cell="{ record }">{{ record.nas_region_path || '-' }}</template>
          </a-table-column>
          <a-table-column title="排序" :width="90" align="center">
            <template #cell="{ record }">{{ record.sort ?? '-' }}</template>
          </a-table-column>
          <a-table-column title="更新时间" :width="180">
            <template #cell="{ record }">
              {{ record.updated_at ? new Date(record.updated_at).toLocaleString() : '-' }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="220" align="center">
            <template #cell="{ record }">
              <template v-if="record.type === 'platform'">
                <a-button
                  type="text"
                  size="small"
                  @click="openModal(null, { platform: record.platform })"
                >
                  <template #icon><icon-plus /></template>
                  添加一级部门
                </a-button>
              </template>
              <template v-else-if="record.type === 'primary'">
                <a-button
                  type="text"
                  size="small"
                  @click="openModal(null, { platform: record.platform, name: record.name })"
                >
                  <template #icon><icon-plus /></template>
                  添加二级部门
                </a-button>
                <a-button
                  v-if="record.raw"
                  type="text"
                  size="small"
                  @click="openModal(record.raw)"
                >
                  <template #icon><icon-edit /></template>
                  编辑一级
                </a-button>
                <a-popconfirm
                  v-if="record.raw"
                  content="确定删除该一级部门吗？"
                  @ok="handleDelete(record.id)"
                >
                  <a-button type="text" status="danger" size="small">
                    <template #icon><icon-delete /></template>
                    删除
                  </a-button>
                </a-popconfirm>
              </template>
              <template v-else>
                <a-button type="text" size="small" @click="openModal(record.raw)">
                  <template #icon><icon-edit /></template>
                  编辑
                </a-button>
                <a-popconfirm content="确定删除该部门吗？" @ok="handleDelete(record.id)">
                  <a-button type="text" status="danger" size="small">
                    <template #icon><icon-delete /></template>
                    删除
                  </a-button>
                </a-popconfirm>
              </template>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑部门' : '新建部门'"
      :ok-loading="submitLoading"
      @ok="handleSubmit"
      @cancel="closeModal"
    >
      <a-form ref="formRef" :model="form" layout="vertical">
        <a-form-item field="platform" label="平台" :rules="[{ required: true, message: '请选择平台' }]">
          <a-select v-model="form.platform" placeholder="选择平台" :disabled="prefilled.platform">
            <a-option v-for="item in platformOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item field="name" label="一级部门" :rules="[{ required: true, message: '请输入一级部门' }]">
          <a-input v-model="form.name" placeholder="例如：US / CA / 运营部" :disabled="prefilled.name" />
        </a-form-item>
        <a-form-item field="sub_name" label="二级部门">
          <a-input v-model="form.sub_name" placeholder="例如：运营一组 / TikTok小组" />
        </a-form-item>
        <a-form-item field="nas_region_path" label="NAS 地区目录">
          <a-input v-model="form.nas_region_path" placeholder="例如：US / CA / UK，留空则保存到平台目录" />
        </a-form-item>
        <a-form-item field="sort" label="排序">
          <a-input-number v-model="form.sort" :min="0" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import {
  IconDelete,
  IconDown,
  IconEdit,
  IconFile,
  IconFolder,
  IconPlus,
  IconRight,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const platformOptions = [
  { label: 'Amazon', value: 'Amazon' },
  { label: 'FBM', value: 'FBM' },
  { label: 'HomeDepot', value: 'HomeDepot' },
  { label: "Lowe's", value: "Lowe's" },
  { label: 'TEMU', value: 'TEMU' },
  { label: 'TikTok', value: 'TikTok' },
  { label: 'Walmart', value: 'Walmart' },
  { label: 'Wayfair', value: 'Wayfair' },
  { label: 'Designer', value: 'Designer' },
  { label: 'Data', value: 'Data' },
];

const items = ref([]);
const searchKeyword = ref('');
const activeKey = ref('');
const loading = ref(false);
const submitLoading = ref(false);
const modalVisible = ref(false);
const isEdit = ref(false);
const formRef = ref(null);
const expandedKeys = ref(new Set());
const prefilled = reactive({
  platform: false,
  name: false,
});
const form = reactive({
  id: null,
  platform: '',
  name: '',
  sub_name: '',
  nas_region_path: '',
  sort: 0,
});

const filteredItems = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase();
  if (!keyword) return items.value;
  return items.value.filter((item) => {
    return [
      getPlatformLabel(item.platform),
      item.platform,
      item.name,
      item.sub_name,
      item.nas_region_path,
    ].some((value) => String(value || '').toLowerCase().includes(keyword));
  });
});

const groupedDepartments = computed(() => {
  const platformMap = new Map();

  for (const item of filteredItems.value) {
    if (!platformMap.has(item.platform)) {
      platformMap.set(item.platform, {
        label: getPlatformLabel(item.platform),
        value: item.platform,
        departments: new Map(),
      });
    }
    const platformGroup = platformMap.get(item.platform);
    const name = item.name || '未命名一级部门';
    if (!platformGroup.departments.has(name)) {
      platformGroup.departments.set(name, []);
    }
    platformGroup.departments.get(name).push(item);
  }

  return Array.from(platformMap.values())
    .map((platform) => {
      const departments = Array.from(platform.departments.entries())
        .map(([name, children]) => {
          const sortedChildren = children.slice().sort(sortDepartment);
          return {
            name,
            raw: sortedChildren.find((item) => !item.sub_name) || null,
            children: sortedChildren.filter((item) => item.sub_name),
          };
        })
        .sort((a, b) => sortDepartment(a.raw || a.children[0], b.raw || b.children[0]) || a.name.localeCompare(b.name));
      return {
        ...platform,
        departments,
      };
    })
    .sort((a, b) => a.label.localeCompare(b.label));
});

const treeRows = computed(() => {
  const rows = [];
  for (const platform of groupedDepartments.value) {
    const platformKey = `platform:${platform.value}`;
    rows.push({
      key: platformKey,
      type: 'platform',
      level: 0,
      title: platform.label,
      platform: platform.value,
      platformLabel: platform.label,
      count: platform.departments.reduce((total, group) => total + group.children.length + (group.raw ? 1 : 0), 0),
      expandable: platform.departments.length > 0,
    });

    if (!expandedKeys.value.has(platformKey)) continue;
    for (const group of platform.departments) {
      const primaryKey = `primary:${platform.value}:${group.name}`;
      rows.push({
        key: primaryKey,
        type: 'primary',
        level: 1,
        title: group.name,
        platform: platform.value,
        platformLabel: platform.label,
        name: group.name,
        count: group.children.length,
        id: group.raw?.id,
        nas_region_path: group.raw?.nas_region_path,
        sort: group.raw?.sort,
        updated_at: group.raw?.updated_at,
        expandable: group.children.length > 0,
        raw: group.raw,
      });

      if (!expandedKeys.value.has(primaryKey)) continue;
      for (const department of group.children) {
        rows.push({
          key: `department:${department.id}`,
          type: 'department',
          level: 2,
          title: department.sub_name || '未设置二级部门',
          platform: department.platform,
          platformLabel: getPlatformLabel(department.platform),
          name: department.name,
          id: department.id,
          nas_region_path: department.nas_region_path,
          sort: department.sort,
          updated_at: department.updated_at,
          expandable: false,
          raw: department,
        });
      }
    }
  }
  return rows;
});

async function fetchItems() {
  loading.value = true;
  try {
    const res = await request.get('/departments');
    items.value = res.data || [];
    expandExistingGroups();
  } finally {
    loading.value = false;
  }
}

function expandExistingGroups() {
  const next = new Set(expandedKeys.value);
  for (const item of items.value) {
    next.add(`platform:${item.platform}`);
    next.add(`primary:${item.platform}:${item.name || '未命名一级部门'}`);
  }
  expandedKeys.value = next;
}

function resetForm() {
  form.id = null;
  form.platform = '';
  form.name = '';
  form.sub_name = '';
  form.nas_region_path = '';
  form.sort = 0;
  prefilled.platform = false;
  prefilled.name = false;
  formRef.value?.clearValidate();
}

function openModal(record = null, defaults = {}) {
  isEdit.value = !!record;
  resetForm();
  if (record) {
    form.id = record.id;
    form.platform = record.platform || '';
    form.name = record.name || '';
    form.sub_name = record.sub_name || '';
    form.nas_region_path = record.nas_region_path || '';
    form.sort = record.sort || 0;
  } else {
    form.platform = defaults.platform || '';
    form.name = defaults.name || '';
    prefilled.platform = !!defaults.platform;
    prefilled.name = !!defaults.name;
  }
  modalVisible.value = true;
}

function closeModal() {
  modalVisible.value = false;
}

async function handleSubmit() {
  const err = await formRef.value?.validate();
  if (err) return;
  submitLoading.value = true;
  try {
    const payload = {
      platform: form.platform,
      name: form.name,
      sub_name: form.sub_name,
      nas_region_path: form.nas_region_path,
      sort: form.sort,
    };
    let saved = null;
    if (isEdit.value) {
      const res = await request.put(`/departments/${form.id}`, payload);
      saved = res.data;
      Message.success('部门更新成功');
    } else {
      const res = await request.post('/departments', payload);
      saved = res.data;
      Message.success('部门创建成功');
    }
    modalVisible.value = false;
    focusDepartment(saved);
    await fetchItems();
  } finally {
    submitLoading.value = false;
  }
}

function focusDepartment(item) {
  if (!item) return;
  const platformKey = `platform:${item.platform}`;
  const primaryKey = `primary:${item.platform}:${item.name || '未命名一级部门'}`;
  const next = new Set(expandedKeys.value);
  next.add(platformKey);
  next.add(primaryKey);
  expandedKeys.value = next;
  activeKey.value = item.sub_name ? `department:${item.id}` : primaryKey;
  searchKeyword.value = item.sub_name || item.name || getPlatformLabel(item.platform);
}

async function handleDelete(id) {
  await request.delete(`/departments/${id}`);
  Message.success('部门已删除');
  await fetchItems();
}

function toggleExpand(key) {
  const next = new Set(expandedKeys.value);
  if (next.has(key)) {
    next.delete(key);
  } else {
    next.add(key);
  }
  expandedKeys.value = next;
}

function sortDepartment(a, b) {
  return (a?.sort || 0) - (b?.sort || 0) || (a?.id || 0) - (b?.id || 0);
}

function getPlatformLabel(value) {
  if (!value) return '-';
  const normalized = normalizePlatformValue(value);
  return platformOptions.find(item => normalizePlatformValue(item.value) === normalized)?.label || value;
}

function normalizePlatformValue(value) {
  return String(value || '').toLowerCase().replaceAll(' ', '').replaceAll("'", '');
}

onMounted(fetchItems);
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.tree-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
}

.expand-button {
  width: 22px;
  height: 22px;
  padding: 0;
}

.expand-placeholder {
  width: 22px;
  height: 22px;
  flex: 0 0 auto;
}

.tree-icon {
  color: rgb(var(--arcoblue-6));
  flex: 0 0 auto;
}

.tree-icon-file {
  color: var(--color-text-3);
}

.tree-title {
  font-weight: 500;
}

.tree-name-active {
  margin: -4px 0;
  padding-top: 4px;
  padding-bottom: 4px;
  border-radius: 4px;
  background: var(--color-fill-2);
}
</style>
