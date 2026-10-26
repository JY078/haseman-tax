<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">角色管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><icon-plus /></template>
        新建角色
      </a-button>
    </div>

    <a-card>

      <a-table :data="roles" :loading="loading" :pagination="false">
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="70" />
          <a-table-column title="角色名" data-index="name" :width="140" />
          <a-table-column title="描述" data-index="description" />
          <a-table-column title="菜单权限">
            <template #cell="{ record }">
              <a-space wrap>
                <a-tag v-for="menu in (record.menus || [])" :key="menu.id" color="arcoblue" size="small">
                  {{ menu.title }}
                </a-tag>
                <a-tag v-if="!record.menus || record.menus.length === 0" color="gray" size="small">无</a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="180">
            <template #cell="{ record }">
              <a-button type="text" size="small" @click="openEditModal(record)">
                <template #icon><icon-edit /></template>
                编辑权限
              </a-button>
              <a-popconfirm
                v-if="record.name !== 'admin' && record.name !== 'user'"
                content="确定删除该角色？"
                @ok="handleDelete(record.id)"
              >
                <a-button type="text" size="small" status="danger">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-popconfirm>
              <a-tag v-else color="orange" size="small">内置</a-tag>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? `编辑角色：${form.name}` : '新建角色'"
      @ok="handleSubmit"
      @cancel="handleCancel"
      :ok-loading="submitLoading"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <a-form-item
          v-if="!isEdit"
          label="角色名"
          field="name"
          :rules="[{ required: true, message: '请输入角色名' }]"
        >
          <a-input v-model="form.name" placeholder="如 editor、viewer" />
        </a-form-item>
        <a-form-item label="描述" field="description">
          <a-input v-model="form.description" placeholder="角色描述（可选）" />
        </a-form-item>
        <a-form-item label="菜单权限">
          <a-select
            v-model="form.menuIds"
            multiple
            placeholder="选择菜单权限（可多选）"
            allow-clear
          >
            <a-option
              v-for="menu in allMenus"
              :key="menu.id"
              :value="menu.id"
              :label="menu.title"
            />
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const roles = ref([]);
const allMenus = ref([]);
const loading = ref(false);
const modalVisible = ref(false);
const isEdit = ref(false);
const submitLoading = ref(false);
const formRef = ref(null);

const form = ref({
  id: null,
  name: '',
  description: '',
  menuIds: [],
});

const fetchRoles = async () => {
  loading.value = true;
  try {
    const res = await request.get('/roles');
    roles.value = res.data || [];
  } catch {
    // handled by interceptor
  } finally {
    loading.value = false;
  }
};

const fetchAllMenus = async () => {
  try {
    const res = await request.get('/all-menus');
    allMenus.value = res.data || [];
  } catch {
    // handled by interceptor
  }
};

const openCreateModal = () => {
  isEdit.value = false;
  form.value = { id: null, name: '', description: '', menuIds: [] };
  modalVisible.value = true;
};

const openEditModal = (record) => {
  isEdit.value = true;
  form.value = {
    id: record.id,
    name: record.name,
    description: record.description || '',
    menuIds: (record.menus || []).map(m => m.id),
  };
  modalVisible.value = true;
};

const handleCancel = () => {
  modalVisible.value = false;
};

const handleSubmit = async () => {
  if (formRef.value) {
    const err = await formRef.value.validate();
    if (err) return;
  }
  submitLoading.value = true;
  try {
    if (isEdit.value) {
      await request.put(`/roles/${form.value.id}`, {
        description: form.value.description,
        menu_ids: form.value.menuIds,
      });
      Message.success('角色更新成功');
    } else {
      await request.post('/roles', {
        name: form.value.name,
        description: form.value.description,
        menu_ids: form.value.menuIds,
      });
      Message.success('角色创建成功');
    }
    modalVisible.value = false;
    fetchRoles();
  } catch {
    // handled by interceptor
  } finally {
    submitLoading.value = false;
  }
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/roles/${id}`);
    Message.success('删除成功');
    fetchRoles();
  } catch {
    // handled
  }
};

onMounted(() => {
  fetchRoles();
  fetchAllMenus();
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
</style>
