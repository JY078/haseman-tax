<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">意见反馈</h1>
    </div>

    <a-card>
      <!-- 筛选栏 -->
      <div class="filter-bar">
        <a-radio-group v-model="statusFilter" type="button" @change="onFilterChange">
          <a-radio value="">全部</a-radio>
          <a-radio value="pending">待处理</a-radio>
          <a-radio value="reviewed">已处理</a-radio>
        </a-radio-group>
      </div>

      <a-table
        :data="feedbacks"
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
          <a-table-column title="用户" :width="140">
            <template #cell="{ record }">
              <span class="text-sm">{{ record.user?.username || `#${record.user_id}` }}</span>
              <div class="text-xs text-gray-400 truncate max-w-[120px]">{{ record.user?.email }}</div>
            </template>
          </a-table-column>
          <a-table-column title="反馈内容" :width="320">
            <template #cell="{ record }">
              <div class="feedback-content">{{ record.content }}</div>
            </template>
          </a-table-column>
          <a-table-column title="联系方式" data-index="contact_info" :width="160">
            <template #cell="{ record }">
              <span class="text-sm text-gray-500">{{ record.contact_info || '—' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status === 'reviewed' ? 'green' : 'orange'" size="small">
                {{ record.status === 'reviewed' ? '已处理' : '待处理' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="提交时间" :width="175">
            <template #cell="{ record }">
              {{ new Date(record.created_at).toLocaleString() }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="180">
            <template #cell="{ record }">
              <a-button type="text" size="small" @click="openDetail(record)">
                <template #icon><icon-eye /></template>
                查看
              </a-button>
              <a-button
                v-if="record.status === 'pending'"
                type="text"
                size="small"
                status="success"
                @click="markReviewed(record)"
              >
                <template #icon><icon-check /></template>
                标记已处理
              </a-button>
              <a-popconfirm content="确定删除该反馈？" @ok="handleDelete(record.id)">
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

    <!-- 详情 / 备注弹窗 -->
    <a-modal
      v-model:visible="detailVisible"
      title="反馈详情"
      :width="560"
      @ok="handleSaveNote"
      @cancel="detailVisible = false"
      :ok-loading="saveLoading"
      ok-text="保存备注"
    >
      <div v-if="current" class="detail-body">
        <div class="detail-row">
          <span class="detail-label">用户</span>
          <span>{{ current.user?.username || `#${current.user_id}` }}（{{ current.user?.email }}）</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">联系方式</span>
          <span>{{ current.contact_info || '未填写' }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">状态</span>
          <a-tag :color="current.status === 'reviewed' ? 'green' : 'orange'" size="small">
            {{ current.status === 'reviewed' ? '已处理' : '待处理' }}
          </a-tag>
        </div>
        <div class="detail-row">
          <span class="detail-label">提交时间</span>
          <span>{{ new Date(current.created_at).toLocaleString() }}</span>
        </div>
        <div class="detail-section">
          <div class="detail-label mb-2">反馈内容</div>
          <div class="feedback-content-box">{{ current.content }}</div>
        </div>
        <div class="detail-section">
          <div class="detail-label mb-2">处理备注</div>
          <a-textarea
            v-model="noteInput"
            placeholder="填写处理备注（可选）"
            :auto-size="{ minRows: 3, maxRows: 8 }"
          />
        </div>
        <div class="detail-section">
          <a-radio-group v-model="statusInput" type="button">
            <a-radio value="pending">待处理</a-radio>
            <a-radio value="reviewed">已处理</a-radio>
          </a-radio-group>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import { IconEye, IconDelete, IconCheck } from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const feedbacks = ref([]);
const loading = ref(false);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const statusFilter = ref('');

const detailVisible = ref(false);
const saveLoading = ref(false);
const current = ref(null);
const noteInput = ref('');
const statusInput = ref('pending');

const fetchFeedbacks = async () => {
  loading.value = true;
  try {
    const params = { page: page.value, page_size: pageSize.value };
    if (statusFilter.value) params.status = statusFilter.value;
    const res = await request.get('/admin/feedbacks', { params });
    feedbacks.value = res.data.data || [];
    total.value = res.data.total || 0;
  } catch {
    // handled
  } finally {
    loading.value = false;
  }
};

const onPageChange = (p) => {
  page.value = p;
  fetchFeedbacks();
};

const onFilterChange = () => {
  page.value = 1;
  fetchFeedbacks();
};

const openDetail = (record) => {
  current.value = record;
  noteInput.value = record.admin_note || '';
  statusInput.value = record.status || 'pending';
  detailVisible.value = true;
};

const handleSaveNote = async () => {
  if (!current.value) return;
  saveLoading.value = true;
  try {
    await request.put(`/admin/feedbacks/${current.value.id}`, {
      status: statusInput.value,
      admin_note: noteInput.value,
    });
    Message.success('保存成功');
    detailVisible.value = false;
    fetchFeedbacks();
  } catch {
    // handled
  } finally {
    saveLoading.value = false;
  }
};

const markReviewed = async (record) => {
  try {
    await request.put(`/admin/feedbacks/${record.id}`, {
      status: 'reviewed',
      admin_note: record.admin_note || '',
    });
    Message.success('已标记为已处理');
    fetchFeedbacks();
  } catch {
    // handled
  }
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/admin/feedbacks/${id}`);
    Message.success('删除成功');
    fetchFeedbacks();
  } catch {
    // handled
  }
};

onMounted(() => {
  fetchFeedbacks();
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

.filter-bar {
  margin-bottom: 16px;
}

.feedback-content {
  font-size: 13px;
  color: #374151;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 72px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}

.detail-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #374151;
}

.detail-label {
  font-size: 13px;
  color: #6b7280;
  min-width: 64px;
  flex-shrink: 0;
}

.detail-section {
  padding-top: 4px;
}

.feedback-content-box {
  font-size: 13px;
  color: #374151;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  max-height: 200px;
  overflow-y: auto;
}
</style>
