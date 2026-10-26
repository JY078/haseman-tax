<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">公告管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><icon-plus /></template>
        新建公告
      </a-button>
    </div>

    <a-card>
      <a-table
        :data="announcements"
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
          <a-table-column title="标题" data-index="title" :width="240" ellipsis />
          <a-table-column title="状态" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status === 'published' ? 'green' : 'gray'" size="small">
                {{ record.status === 'published' ? '已发布' : '草稿' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" :width="180">
            <template #cell="{ record }">
              {{ new Date(record.created_at).toLocaleString() }}
            </template>
          </a-table-column>
          <a-table-column title="更新时间" :width="180">
            <template #cell="{ record }">
              {{ new Date(record.updated_at).toLocaleString() }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="220">
            <template #cell="{ record }">
              <a-button type="text" size="small" @click="openEditModal(record)">
                <template #icon><icon-edit /></template>
                编辑
              </a-button>
              <a-button
                type="text"
                size="small"
                :status="record.status === 'published' ? 'warning' : 'success'"
                @click="toggleStatus(record)"
              >
                <template #icon>
                  <icon-eye-invisible v-if="record.status === 'published'" />
                  <icon-eye v-else />
                </template>
                {{ record.status === 'published' ? '下架' : '发布' }}
              </a-button>
              <a-popconfirm content="确定删除该公告？" @ok="handleDelete(record.id)">
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
      :title="isEdit ? '编辑公告' : '新建公告'"
      :width="640"
      @ok="handleSubmit"
      @cancel="modalVisible = false"
      :ok-loading="submitLoading"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <a-form-item
          label="标题"
          field="title"
          :rules="[{ required: true, message: '请输入公告标题' }]"
        >
          <a-input v-model="form.title" placeholder="公告标题" />
        </a-form-item>
        <a-form-item
          label="内容"
          field="content"
          :rules="[{ required: true, message: '请输入公告内容' }]"
        >
          <a-textarea
            v-model="form.content"
            placeholder="公告内容，支持多行文本"
            :auto-size="{ minRows: 4, maxRows: 12 }"
          />
        </a-form-item>
        <a-form-item label="状态" field="status">
          <a-radio-group v-model="form.status" type="button">
            <a-radio value="draft">草稿</a-radio>
            <a-radio value="published">立即发布</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconEye,
  IconEyeInvisible,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const announcements = ref([]);
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
  title: '',
  content: '',
  status: 'draft',
});

const fetchAnnouncements = async () => {
  loading.value = true;
  try {
    const res = await request.get('/announcements', {
      params: { page: page.value, page_size: pageSize.value },
    });
    announcements.value = res.data.data || [];
    total.value = res.data.total || 0;
  } catch {
    // handled
  } finally {
    loading.value = false;
  }
};

const onPageChange = (p) => {
  page.value = p;
  fetchAnnouncements();
};

const openCreateModal = () => {
  isEdit.value = false;
  form.value = { id: null, title: '', content: '', status: 'draft' };
  modalVisible.value = true;
};

const openEditModal = (record) => {
  isEdit.value = true;
  form.value = {
    id: record.id,
    title: record.title,
    content: record.content,
    status: record.status,
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
    if (isEdit.value) {
      await request.put(`/announcements/${form.value.id}`, {
        title: form.value.title,
        content: form.value.content,
        status: form.value.status,
      });
      Message.success('公告更新成功');
    } else {
      await request.post('/announcements', {
        title: form.value.title,
        content: form.value.content,
        status: form.value.status,
      });
      Message.success('公告创建成功');
    }
    modalVisible.value = false;
    fetchAnnouncements();
  } catch {
    // handled
  } finally {
    submitLoading.value = false;
  }
};

const toggleStatus = async (record) => {
  const newStatus = record.status === 'published' ? 'draft' : 'published';
  try {
    await request.put(`/announcements/${record.id}`, { status: newStatus });
    Message.success(newStatus === 'published' ? '公告已发布' : '公告已下架');
    fetchAnnouncements();
  } catch {
    // handled
  }
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/announcements/${id}`);
    Message.success('删除成功');
    fetchAnnouncements();
  } catch {
    // handled
  }
};

onMounted(() => {
  fetchAnnouncements();
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
