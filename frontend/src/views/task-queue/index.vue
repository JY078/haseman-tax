<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">任务队列</h1>
        <p class="page-subtitle">查看当前运行中与排队中的场景图任务，快速定位等待中的任务与当前执行压力。</p>
      </div>
      <a-space>
        <a-tag color="arcoblue" size="large">单用户上限 {{ overview.userRunningImageLimit }} 张</a-tag>
        <a-switch v-model="autoRefreshEnabled" size="small" />
        <span class="toolbar-hint">自动刷新（5秒）</span>
        <a-button type="primary" :loading="loading" @click="fetchOverview">刷新队列</a-button>
      </a-space>
    </div>

    <div class="stats-grid">
      <a-card class="stat-card stat-card--queued" :bordered="false">
        <div class="stat-card__label">排队任务</div>
        <div class="stat-card__value">{{ overview.queuedTaskCount }}</div>
        <div class="stat-card__meta">{{ overview.queuedImageCount }} 张图片待执行</div>
      </a-card>
      <a-card class="stat-card stat-card--running" :bordered="false">
        <div class="stat-card__label">运行任务</div>
        <div class="stat-card__value">{{ overview.runningTaskCount }}</div>
        <div class="stat-card__meta">{{ overview.runningImageCount }} 张图片执行中</div>
      </a-card>
      <a-card class="stat-card" :bordered="false">
        <div class="stat-card__label">当前筛选结果</div>
        <div class="stat-card__value">{{ filteredItems.length }}</div>
        <div class="stat-card__meta">{{ filterLabel }}</div>
      </a-card>
    </div>

    <a-card class="table-card" :bordered="false">
      <div class="table-toolbar">
        <div>
          <div class="section-title">任务列表</div>
          <div class="section-subtitle">按任务状态快速筛选，查看更多排队中的任务明细。</div>
        </div>
        <a-radio-group v-model="statusFilter" type="button" size="small">
          <a-radio value="all">全部</a-radio>
          <a-radio value="queued">排队中</a-radio>
          <a-radio value="running">运行中</a-radio>
        </a-radio-group>
      </div>

      <a-table
        :data="filteredItems"
        :loading="loading"
        :pagination="false"
        row-key="taskId"
        size="small"
        class="queue-table"
        :scroll="{ x: 1180 }"
      >
        <template #empty>
          <a-empty description="当前筛选条件下暂无任务" />
        </template>
        <template #columns>
          <a-table-column title="任务ID" data-index="taskId" :width="220" ellipsis tooltip />
          <a-table-column title="备注名" data-index="userName" :width="160" ellipsis tooltip>
            <template #cell="{ record }">
              <span class="user-name">{{ record.userName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="110" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 'running' ? 'arcoblue' : 'orange'" bordered>
                {{ record.status === 'running' ? '运行中' : '排队中' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="图片数" data-index="imageCount" :width="88" align="center">
            <template #cell="{ record }">
              <span class="count-badge">{{ record.imageCount }}</span>
            </template>
          </a-table-column>
          <a-table-column title="商品类目" data-index="productCategory" :width="190" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ record.productCategory || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="风格" data-index="styleName" :width="180" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ record.styleName || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="国家" data-index="country" :width="120" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ record.country || record.countryName || record.countryCode || '-' }}</span>
            </template>
          </a-table-column>
          <!-- <a-table-column title="排队时长" :width="140" align="center">
            <template #cell="{ record }">
              <span class="duration-text">{{ formatQueueDuration(record) }}</span>
            </template>
          </a-table-column> -->
          <!-- <a-table-column title="更新时间" :width="180">
            <template #cell="{ record }">{{ formatDateTime(record.updatedAt || record.createdAt) }}</template>
          </a-table-column> -->
          <a-table-column title="创建时间" :width="180">
            <template #cell="{ record }">{{ formatDateTime(record.createdAt) }}</template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import request from '@/utils/request'

const AUTO_REFRESH_INTERVAL = 5000

const loading = ref(false)
const statusFilter = ref('all')
const autoRefreshEnabled = ref(true)
const nowTs = ref(Date.now())
let autoRefreshTimer = null
const overview = reactive({
  userRunningImageLimit: 8,
  queuedTaskCount: 0,
  runningTaskCount: 0,
  queuedImageCount: 0,
  runningImageCount: 0,
  items: [],
})

const filteredItems = computed(() => {
  if (statusFilter.value === 'all') return overview.items
  return overview.items.filter(item => item.status === statusFilter.value)
})

const filterLabel = computed(() => {
  if (statusFilter.value === 'queued') return '仅显示排队中的任务'
  if (statusFilter.value === 'running') return '仅显示运行中的任务'
  return '显示全部任务'
})

function formatDateTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function formatQueueDuration(record) {
  if (!record?.createdAt || record?.status !== 'queued') return '-'
  const createdAt = new Date(record.createdAt)
  if (Number.isNaN(createdAt.getTime())) return '-'
  const diff = Math.max(0, nowTs.value - createdAt.getTime())
  const totalMinutes = Math.floor(diff / 60000)
  const days = Math.floor(totalMinutes / 1440)
  const hours = Math.floor((totalMinutes % 1440) / 60)
  const minutes = totalMinutes % 60
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

function stopAutoRefresh() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
}

function startAutoRefresh() {
  stopAutoRefresh()
  if (!autoRefreshEnabled.value) return
  autoRefreshTimer = setInterval(() => {
    nowTs.value = Date.now()
    fetchOverview(false)
  }, AUTO_REFRESH_INTERVAL)
}

async function fetchOverview(showLoading = true) {
  if (showLoading) loading.value = true
  nowTs.value = Date.now()
  try {
    const res = await request.get('/admin/scene-generation/queue-overview')
    const data = res.data || {}
    overview.userRunningImageLimit = Number(data.userRunningImageLimit || 8)
    overview.queuedTaskCount = Number(data.queuedTaskCount || 0)
    overview.runningTaskCount = Number(data.runningTaskCount || 0)
    overview.queuedImageCount = Number(data.queuedImageCount || 0)
    overview.runningImageCount = Number(data.runningImageCount || 0)
    overview.items = Array.isArray(data.items) ? data.items : []
  } finally {
    loading.value = false
  }
}

watch(autoRefreshEnabled, () => {
  startAutoRefresh()
})

onMounted(() => {
  fetchOverview()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
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
  gap: 16px;
  margin-bottom: 24px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
}

.page-subtitle {
  margin: 8px 0 0;
  max-width: 720px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-3);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  border-radius: 20px;
  background: linear-gradient(180deg, #ffffff 0%, #fafbff 100%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.stat-card--queued {
  background: linear-gradient(180deg, rgba(255, 247, 232, 0.95) 0%, #ffffff 100%);
}

.stat-card--running {
  background: linear-gradient(180deg, rgba(232, 244, 255, 0.95) 0%, #ffffff 100%);
}

.stat-card__label {
  font-size: 12px;
  color: var(--color-text-3);
}

.stat-card__value {
  margin-top: 10px;
  font-size: 30px;
  line-height: 1;
  font-weight: 700;
  color: var(--color-text-1);
}

.stat-card__meta {
  margin-top: 10px;
  font-size: 12px;
  color: var(--color-text-2);
}

.table-card {
  border-radius: 20px;
}

.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.section-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-1);
}

.section-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
}

.toolbar-hint {
  font-size: 12px;
  color: var(--color-text-2);
}

.queue-table :deep(.arco-table-th) {
  background: #f7f8fa;
}

.queue-table :deep(.arco-table-tr:hover .arco-table-td) {
  background: #fcfcff;
}

.user-name {
  font-weight: 600;
  color: var(--color-text-1);
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-text-1);
  font-size: 12px;
  font-weight: 700;
}

.duration-text {
  font-variant-numeric: tabular-nums;
  color: var(--color-text-1);
}

@media (max-width: 1100px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .page-header,
  .table-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
