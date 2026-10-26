<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">用户管理</h1>
      <a-space>
        <a-button :loading="loading" @click="fetchUsers()">刷新状态</a-button>
        <a-button type="primary" @click="openCreateModal">
          <template #icon><icon-plus /></template>
          新建用户
        </a-button>
      </a-space>
    </div>

    <a-card>
      <p class="online-help">在线表示账号已启用、登录未过期，且用户端或管理后台在最近 90 秒内发送过心跳。超过 90 秒无心跳即离线，每 10 秒刷新。</p>

      <a-table
        :data="users"
        :loading="loading"
        :pagination="{
          total,
          current: page,
          pageSize,
          showTotal: true,
          onChange: onPageChange,
        }"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="70" />
          <a-table-column title="账号" data-index="username" :width="140" />
          <a-table-column title="所属平台" :width="120">
            <template #cell="{ record }">
              {{ getPlatformLabel(record.platform) }}
            </template>
          </a-table-column>
          <a-table-column title="部门" :width="140">
            <template #cell="{ record }">
              {{ formatDepartment(record) }}
            </template>
          </a-table-column>
          <a-table-column title="备注名" :width="160">
            <template #cell="{ record }">
              {{ record.name || '-' }}
            </template>
          </a-table-column>
          <a-table-column title="角色" :width="200">
            <template #cell="{ record }">
              <a-space wrap>
                <a-tag
                  v-for="role in (record.roles || [])"
                  :key="role.id"
                  :color="role.name === 'admin' ? 'red' : 'arcoblue'"
                  size="small"
                >
                  {{ role.name }}
                </a-tag>
                <a-tag v-if="!record.roles || record.roles.length === 0" color="gray" size="small">无角色</a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="负责平台" :width="210">
            <template #cell="{ record }">
              <a-space v-if="record.order_platforms?.length" wrap>
                <a-tag v-for="platform in record.order_platforms" :key="platform" color="arcoblue" size="small">
                  {{ getOrderPlatformLabel(platform) }}
                </a-tag>
              </a-space>
              <span v-else class="empty-platform">未配置</span>
            </template>
          </a-table-column>
          <a-table-column title="长期登录" :width="110">
            <template #cell="{ record }">
              <a-tag :color="record.long_login ? 'green' : 'gray'" size="small">
                {{ record.long_login ? '已开启' : '未开启' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="账号状态" :width="110">
            <template #cell="{ record }">
              <a-switch
                :model-value="record.enabled !== false"
                :disabled="record.id === selfId"
                checked-text="启用"
                unchecked-text="停用"
                @change="value => handleEnabledChange(record, value)"
              />
            </template>
          </a-table-column>
          <a-table-column title="在线状态" :width="90">
            <template #cell="{ record }">
              <a-tag :color="record.online ? 'green' : 'gray'" size="small">
                {{ record.online ? '在线' : '离线' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="注册时间" :width="180">
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
                v-if="record.id !== selfId"
                content="确定删除该用户？"
                @ok="handleDelete(record.id)"
              >
                <a-button type="text" size="small" status="danger">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-popconfirm>
              <a-tag v-else color="orange" size="small">当前账号</a-tag>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? `编辑用户：${form.username}` : '新建用户'"
      @ok="handleSubmit"
      @cancel="handleCancel"
      :ok-loading="submitLoading"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <template v-if="!isEdit">
          <a-form-item
            label="账号"
            field="username"
            :rules="[{ required: true, message: '请输入账号' }]"
          >
            <a-input v-model="form.username" placeholder="账号" />
          </a-form-item>
          <a-form-item label="备注名" field="name">
            <a-input v-model="form.name" placeholder="用于识别用户的备注名，例如：张三 / 设计部小王" />
          </a-form-item>
          <a-form-item label="部门" field="departmentId">
            <a-select v-model="form.departmentId" placeholder="选择部门" allow-clear @change="handleDepartmentChange">
              <a-option v-for="item in selectableDepartments" :key="item.id" :value="item.id">
                {{ formatDepartmentOption(item) }}
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item
            label="密码"
            field="password"
            :rules="[{ required: true, message: '请输入密码' }, { minLength: 6, message: '密码至少6位' }]"
          >
            <a-input-password v-model="form.password" placeholder="至少6位" />
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item label="备注名" field="name">
            <a-input v-model="form.name" placeholder="用于识别用户的备注名，例如：张三 / 设计部小王" />
          </a-form-item>
          <a-form-item label="部门" field="departmentId">
            <a-select v-model="form.departmentId" placeholder="选择部门" allow-clear @change="handleDepartmentChange">
              <a-option v-for="item in selectableDepartments" :key="item.id" :value="item.id">
                {{ formatDepartmentOption(item) }}
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="新密码" field="password">
            <a-input-password v-model="form.password" placeholder="留空则不修改" />
          </a-form-item>
        </template>
        <a-form-item label="负责平台" field="orderPlatforms">
          <a-select
            v-model="form.orderPlatforms"
            multiple
            allow-clear
            placeholder="选择该用户负责的订单平台"
          >
            <a-option v-for="item in orderPlatformOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-option>
          </a-select>
          <div class="platform-help">决定该用户可查看、维护和申请取消的订单平台，可多选。</div>
        </a-form-item>
        <a-form-item label="角色分配">
          <a-select
            v-model="form.roleIds"
            multiple
            placeholder="选择角色（可多选）"
            allow-clear
          >
            <a-option
              v-for="role in allRoles"
              :key="role.id"
              :value="role.id"
              :label="role.name + (role.description ? ` — ${role.description}` : '')"
            />
          </a-select>
        </a-form-item>
        <a-form-item label="用户端登录长期有效">
          <a-switch v-model="form.longLogin" />
          <span style="margin-left: 8px; color: #86909c; font-size: 12px">开启后用户端登录有效期为 10 年</span>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { Message } from '@arco-design/web-vue';
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon';
import { useUserStore } from '@/stores/user';
import request from '@/utils/request';

const userStore = useUserStore();
const selfId = computed(() => userStore.userInfo?.id);

const users = ref([]);
const allRoles = ref([]);
const departments = ref([]);
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
const orderPlatformOptions = [
  { label: 'TEMU', value: 'TEMU' },
  { label: 'TikTok', value: 'TIKTOK' },
  { label: "Lowe's", value: 'LOWES' },
];
const loading = ref(false);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);

const modalVisible = ref(false);
const isEdit = ref(false);
const submitLoading = ref(false);
const formRef = ref(null);
const form = ref({
  id: null,
  username: '',
  name: '',
  platform: '',
  departmentId: null,
  department: '',
  subDepartment: '',
  password: '',
  roleIds: [],
  orderPlatforms: [],
  longLogin: false,
  enabled: true,
});

let usersRequestID = 0;
let onlineRefreshTimer = null;
const fetchUsers = async (silent = false) => {
  const requestID = ++usersRequestID;
  if (!silent) loading.value = true;
  try {
    const res = await request.get('/users', {
      params: { page: page.value, page_size: pageSize.value },
    });
    if (requestID !== usersRequestID) return;
    users.value = res.data.data || [];
    total.value = res.data.total || 0;
  } catch {
    // handled
  } finally {
    if (requestID === usersRequestID) loading.value = false;
  }
};

const fetchAllRoles = async () => {
  try {
    const res = await request.get('/roles');
    allRoles.value = res.data || [];
  } catch {
    // handled
  }
};

const fetchDepartments = async () => {
  try {
    const res = await request.get('/departments');
    departments.value = res.data || [];
  } catch {
    // handled
  }
};

const selectableDepartments = computed(() => {
  return departments.value.filter(item => item.sub_name);
});

const onPageChange = (p) => {
  page.value = p;
  fetchUsers();
};

const openCreateModal = () => {
  isEdit.value = false;
  form.value = { id: null, username: '', name: '', platform: '', departmentId: null, department: '', subDepartment: '', password: '', roleIds: [], orderPlatforms: [], longLogin: false };
  modalVisible.value = true;
};

const openEditModal = (record) => {
  isEdit.value = true;
  form.value = {
    id: record.id,
    username: record.username,
    name: record.name || '',
    platform: record.platform || '',
    departmentId: findDepartmentId(record),
    department: record.department || '',
    subDepartment: record.sub_department || '',
    password: '',
    roleIds: (record.roles || []).map(r => r.id),
    orderPlatforms: [...(record.order_platforms || [])],
    longLogin: !!record.long_login,
    enabled: record.enabled !== false,
  };
  modalVisible.value = true;
};

const handleCancel = () => {
  modalVisible.value = false;
};

const handleDepartmentChange = (id) => {
  const item = departments.value.find(dep => dep.id === id);
  form.value.platform = item?.platform || '';
  form.value.department = item?.name || '';
  form.value.subDepartment = item?.sub_name || '';
};

const handleSubmit = async () => {
  if (formRef.value) {
    const err = await formRef.value.validate();
    if (err) return;
  }
  submitLoading.value = true;
  try {
    if (isEdit.value) {
      const payload = {
        role_ids: form.value.roleIds,
        long_login: form.value.longLogin,
        enabled: form.value.enabled,
        name: form.value.name,
        department_id: form.value.departmentId || 0,
        platform: form.value.platform,
        department: form.value.department,
        sub_department: form.value.subDepartment,
        order_platforms: form.value.orderPlatforms,
      };
      if (form.value.password) payload.password = form.value.password;
      await request.put(`/users/${form.value.id}`, payload);
      Message.success('用户更新成功');
    } else {
      await request.post('/users', {
        username: form.value.username,
        name: form.value.name,
        department_id: form.value.departmentId || 0,
        platform: form.value.platform,
        department: form.value.department,
        sub_department: form.value.subDepartment,
        password: form.value.password,
        role_ids: form.value.roleIds,
        order_platforms: form.value.orderPlatforms,
        long_login: form.value.longLogin,
        enabled: form.value.enabled,
      });
      Message.success('用户创建成功');
    }
    modalVisible.value = false;
    fetchUsers();
  } catch {
    // handled
  } finally {
    submitLoading.value = false;
  }
};

const formatDepartment = (record) => {
  const department = record.department || '';
  const subDepartment = record.sub_department || '';
  if (department && subDepartment) return `${department} / ${subDepartment}`;
  return department || subDepartment || '-';
};

const getPlatformLabel = (value) => {
  if (!value) return '-';
  const normalized = normalizePlatformValue(value);
  return platformOptions.find(item => normalizePlatformValue(item.value) === normalized)?.label || value;
};

const getOrderPlatformLabel = (value) => {
  return orderPlatformOptions.find(item => item.value === value)?.label || value;
};

const normalizePlatformValue = (value) => {
  return String(value || '').toLowerCase().replaceAll(' ', '').replaceAll("'", '');
};

const formatDepartmentOption = (item) => {
  const platform = getPlatformLabel(item.platform);
  const department = [item.name, item.sub_name].filter(Boolean).join(' / ') || '-';
  return `${platform} / ${department}`;
};

const findDepartmentId = (record) => {
  const item = departments.value.find(dep =>
    dep.platform === (record.platform || '') &&
    dep.name === (record.department || '') &&
    dep.sub_name === (record.sub_department || '')
  );
  return item?.id || null;
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/users/${id}`);
    Message.success('删除成功');
    fetchUsers();
  } catch {
    // handled
  }
};

const handleEnabledChange = async (record, enabled) => {
  const previous = record.enabled !== false;
  record.enabled = enabled;
  try {
    await request.put(`/users/${record.id}`, {
      name: record.name || '',
      department: record.department || '',
      sub_department: record.sub_department || '',
      platform: record.platform || '',
      role_ids: (record.roles || []).map(role => role.id),
      order_platforms: record.order_platforms || [],
      long_login: !!record.long_login,
      enabled,
    });
    Message.success(enabled ? '账号已启用' : '账号已停用');
    if (!enabled) record.online = false;
  } catch {
    record.enabled = previous;
  }
};

const refreshOnlineStatus = () => {
  if (document.visibilityState === 'visible' && !loading.value && !modalVisible.value) {
    fetchUsers(true);
  }
};

onMounted(() => {
  fetchUsers();
  fetchAllRoles();
  fetchDepartments();
  onlineRefreshTimer = setInterval(refreshOnlineStatus, 10000);
  document.addEventListener('visibilitychange', refreshOnlineStatus);
});

onUnmounted(() => {
  clearInterval(onlineRefreshTimer);
  document.removeEventListener('visibilitychange', refreshOnlineStatus);
  usersRequestID++;
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

.empty-platform,
.platform-help,
.online-help {
  color: var(--color-text-3);
  font-size: 12px;
}

.platform-help {
  margin-top: 6px;
  line-height: 1.5;
}
</style>
