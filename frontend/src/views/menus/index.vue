<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">菜单管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><icon-plus /></template>
        新建菜单
      </a-button>
    </div>

    <a-card>
      <a-table
        :data="menus"
        :loading="loading"
        :pagination="{
          total,
          current: page,
          pageSize,
          showTotal: true,
          onChange: onPageChange,
        }"
        row-key="id"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="70" />
          <a-table-column title="菜单名称" :width="200">
            <template #cell="{ record }">
              <span v-if="record.parent_id !== 0" class="sub-menu-indent">└ </span>
              <span>{{ record.title }}</span>
              <a-tag v-if="record.parent_id === 0" size="small" color="blue" style="margin-left:6px">一级</a-tag>
              <a-tag v-else size="small" color="green" style="margin-left:6px">二级</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="上级菜单" :width="130">
            <template #cell="{ record }">
              <span v-if="record.parent_id !== 0">{{ getParentTitle(record.parent_id) }}</span>
              <span v-else class="text-secondary">—</span>
            </template>
          </a-table-column>
          <a-table-column title="路径" :width="180">
            <template #cell="{ record }">
              <span v-if="record.path">{{ record.path }}</span>
              <a-tag v-else size="small" color="gray">分类目录</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="图标" data-index="icon" :width="150">
            <template #cell="{ record }">
              <a-tag size="small" color="arcoblue">{{ record.icon || '—' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="排序" data-index="sort" :width="70" />
          <a-table-column title="创建时间" :width="180">
            <template #cell="{ record }">
              {{ new Date(record.created_at).toLocaleString() }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="160">
            <template #cell="{ record }">
              <a-button type="text" size="small" @click="openEditModal(record)">
                <template #icon><icon-edit /></template>
                编辑
              </a-button>
              <a-popconfirm
                content="确定删除该菜单？删除后相关角色将失去该菜单权限。"
                @ok="handleDelete(record.id)"
              >
                <a-button type="text" size="small" status="danger">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-popconfirm>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑菜单' : '新建菜单'"
      :width="520"
      @ok="handleSubmit"
      @cancel="modalVisible = false"
      :ok-loading="submitLoading"
      ok-text="确认"
      cancel-text="取消"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <a-form-item label="上级菜单" field="parent_id">
          <a-select
            v-model="form.parent_id"
            placeholder="不选则为一级菜单"
            allow-clear
            :options="parentMenuOptions"
          />
          <div class="field-tip">选择上级菜单后，该菜单将作为其子菜单显示</div>
        </a-form-item>

        <a-form-item
          label="菜单名称"
          field="title"
          :rules="[{ required: true, message: '请输入菜单名称' }]"
        >
          <a-input v-model="form.title" placeholder="例如：用户管理" allow-clear />
        </a-form-item>

        <a-form-item
          label="路由路径"
          field="path"
          :rules="[
            {
              validator: (value, cb) => {
                if (value && !/^\/\S+/.test(value)) {
                  cb('路径必须以 / 开头且不含空格');
                } else {
                  cb();
                }
              },
            },
          ]"
        >
          <a-input v-model="form.path" placeholder="例如：/users（一级分类菜单可留空）" allow-clear />
          <div class="field-tip">留空表示该菜单仅作为分类标题，点击不跳转页面</div>
        </a-form-item>

        <a-form-item label="图标标识" field="icon">
          <a-select
            v-model="form.icon"
            placeholder="请选择图标（可选）"
            allow-clear
            allow-search
          >
            <a-option v-for="icon in iconOptions" :key="icon.value" :value="icon.value">
              {{ icon.label }}
            </a-option>
          </a-select>
          <div class="field-tip">图标将显示在侧边栏菜单中</div>
        </a-form-item>

        <a-form-item
          label="排序"
          field="sort"
          :rules="[{ type: 'number', min: 0, message: '排序值不能为负数' }]"
        >
          <a-input-number
            v-model="form.sort"
            placeholder="数值越小越靠前"
            :min="0"
            :step="1"
            :style="{ width: '100%' }"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import {
  IconPlus,
  IconEdit,
  IconDelete,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const menus = ref([]);
const loading = ref(false);
const total = ref(0);
const page = ref(1);
const pageSize = ref(50);

const modalVisible = ref(false);
const isEdit = ref(false);
const submitLoading = ref(false);
const formRef = ref(null);

const defaultForm = () => ({
  id: null,
  parent_id: null,
  title: '',
  path: '',
  icon: '',
  sort: 0,
});
const form = ref(defaultForm());

/** 只取一级菜单作为可选父菜单（parent_id === 0） */
const parentMenuOptions = computed(() =>
  menus.value
    .filter(m => m.parent_id === 0)
    .map(m => ({ label: m.title, value: m.id }))
);

/** 根据 parent_id 查找父菜单标题 */
const getParentTitle = (parentId) => {
  const parent = menus.value.find(m => m.id === parentId);
  return parent?.title || `ID:${parentId}`;
};

// 与 default-layout.vue 中 iconMap 保持一致，并补充常用图标
const iconOptions = [
  { label: 'icon-dashboard（仪表盘）', value: 'icon-dashboard' },
  { label: 'icon-user（用户）', value: 'icon-user' },
  { label: 'icon-settings（设置）', value: 'icon-settings' },
  // { label: 'icon-pay-circle（支付）', value: 'icon-pay-circle' },
  { label: 'icon-history（历史）', value: 'icon-history' },
  { label: 'icon-safe（安全/角色）', value: 'icon-safe' },
  { label: 'icon-robot（模型/机器人）', value: 'icon-robot' },
  { label: 'icon-image（图片）', value: 'icon-image' },
  { label: 'icon-file（文件/模板）', value: 'icon-file' },
  { label: 'icon-book（风格/资料库）', value: 'icon-book' },
  { label: 'icon-notification（公告）', value: 'icon-notification' },
  { label: 'icon-storage（存储）', value: 'icon-storage' },
  // { label: 'icon-gift（套餐/礼物）', value: 'icon-gift' },
  { label: 'icon-menu（菜单）', value: 'icon-menu' },
];

const fetchMenus = async () => {
  loading.value = true;
  try {
    const res = await request.get('/admin/menus', {
      params: { page: page.value, page_size: pageSize.value },
    });
    menus.value = res.data.data || [];
    total.value = res.data.total || 0;
  } catch {
    // 错误由拦截器统一处理
  } finally {
    loading.value = false;
  }
};

const onPageChange = (p) => {
  page.value = p;
  fetchMenus();
};

const openCreateModal = () => {
  isEdit.value = false;
  form.value = defaultForm();
  modalVisible.value = true;
};

const openEditModal = (record) => {
  isEdit.value = true;
  form.value = {
    id: record.id,
    parent_id: record.parent_id || null,
    title: record.title,
    path: record.path,
    icon: record.icon || '',
    sort: record.sort,
  };
  modalVisible.value = true;
};

const handleSubmit = async () => {
  if (formRef.value) {
    const err = await formRef.value.validate();
    if (err) return;
  }

  submitLoading.value = true;
  try {
    const payload = {
      parent_id: form.value.parent_id || 0,
      title: form.value.title,
      path: form.value.path,
      icon: form.value.icon,
      sort: form.value.sort,
    };

    if (isEdit.value) {
      await request.put(`/admin/menus/${form.value.id}`, payload);
      Message.success('菜单更新成功');
    } else {
      await request.post('/admin/menus', payload);
      Message.success('菜单创建成功');
    }
    modalVisible.value = false;
    fetchMenus();
  } catch {
    // 错误由拦截器统一处理
  } finally {
    submitLoading.value = false;
  }
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/admin/menus/${id}`);
    Message.success('删除成功');
    fetchMenus();
  } catch {
    // 错误由拦截器统一处理
  }
};

onMounted(() => {
  fetchMenus();
});
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

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.field-tip {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
}

.sub-menu-indent {
  color: var(--color-text-3);
  margin-right: 2px;
}

.text-secondary {
  color: var(--color-text-3);
}
</style>
