<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">使用记录</h1>
        <p class="page-subtitle">查看各类功能的使用明细</p>
      </div>
      <a-button class="refresh-btn" @click="refreshAll" :loading="loading">
        <template #icon><icon-refresh /></template>
        刷新
      </a-button>
    </div>

    <div class="toolbar-card">
      <a-space wrap>
        <a-input-search
          v-model="filterUserName"
          placeholder="搜索用户"
          style="width: 200px"
          allow-clear
          @search="fetchLogs"
          @clear="fetchLogs"
        />
        <a-select
          v-model="filterType"
          placeholder="全部类型"
          style="width: 140px"
          allow-clear
          @change="fetchLogs"
        >
          <a-option value="chat">对话</a-option>
          <a-option value="drawing">绘图</a-option>
          <a-option value="video">视频</a-option>
        </a-select>
      </a-space>
    </div>

    <div class="content-layout">
      <section class="main-panel">
        <a-card class="panel-card" :bordered="false">
          <template #title>
            <div class="card-title">
              <icon-list class="card-title-icon" />
              <span>使用明细</span>
            </div>
          </template>

          <a-table
            :data="logs"
            :loading="loading"
            :pagination="{
              total,
              current: page,
              pageSize,
              showTotal: true,
              onChange: onPageChange,
            }"
            :scroll="{ x: 1020 }"
            size="small"
            row-key="id"
          >
            <template #columns>
              <a-table-column title="ID" data-index="id" :width="78" align="right">
                <template #cell="{ record }">
                  <span class="id-pill metric-number">{{ record.id }}</span>
                </template>
              </a-table-column>
              <a-table-column title="用户" data-index="user_name" :width="116">
                <template #cell="{ record }">{{ record.user_name || '-' }}</template>
              </a-table-column>
              <a-table-column title="类型" :width="86" align="center">
                <template #cell="{ record }">
                  <a-tag :color="getTypeColor(record.usage_type)" size="small">
                    {{ getTypeLabel(record.usage_type) }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="下载" :width="92" align="center">
                <template #cell="{ record }">
                  <a-tag :color="record.download_count > 0 ? 'green' : 'gray'" size="small">
                    {{ record.download_count || 0 }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="模型" data-index="model" :width="132">
                <template #cell="{ record }">
                  <a-typography-text :ellipsis="{ rows: 1, showTooltip: true }" style="max-width: 110px">
                    {{ record.model || '-' }}
                  </a-typography-text>
                </template>
              </a-table-column>
              <a-table-column title="提示词" :width="220" :ellipsis="true">
                <template #cell="{ record }">
                  <a-typography-text :ellipsis="{ rows: 1, showTooltip: true }" style="max-width: 200px">
                    {{ record.prompt || '-' }}
                  </a-typography-text>
                </template>
              </a-table-column>
              <a-table-column title="Token" :width="150">
                <template #cell="{ record }">
                  <span v-if="record.usage_type === 'chat'" class="token-usage">
                    <a-tag color="arcoblue" size="small">{{ record.total_tokens }}</a-tag>
                    <span class="token-detail">{{ record.prompt_tokens }} / {{ record.completion_tokens }}</span>
                  </span>
                  <span v-else class="text-muted">-</span>
                </template>
              </a-table-column>
              <a-table-column title="分辨率" data-index="resolution" :width="88" align="center">
                <template #cell="{ record }">{{ record.resolution || '-' }}</template>
              </a-table-column>
              <a-table-column title="状态" :width="82" align="center">
                <template #cell="{ record }">
                  <a-tag :color="record.status === 'success' ? 'green' : 'red'" size="small">
                    {{ record.status === 'success' ? '成功' : '失败' }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="时间" :width="160">
                <template #cell="{ record }">
                  <span class="time-text">{{ formatDateTime(record.created_at) }}</span>
                </template>
              </a-table-column>
              <a-table-column title="耗时" data-index="duration" :width="96" align="right">
                <template #cell="{ record }">{{ formatDurationSeconds(record.duration) }}</template>
              </a-table-column>
              <a-table-column title="操作" :width="72" align="center">
                <template #cell="{ record }">
                  <a-button type="text" size="small" @click="viewDetail(record)">详情</a-button>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </a-card>
      </section>

    </div>

    <a-modal v-model:visible="detailVisible" title="使用记录详情" :footer="false" width="760px">
      <a-descriptions v-if="currentLog" :column="2" bordered>
        <a-descriptions-item label="ID"><span class="metric-number">{{ currentLog.id }}</span></a-descriptions-item>
        <a-descriptions-item label="时间">{{ formatDateTime(currentLog.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="用户">{{ currentLog.user_name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="类型">
          <a-tag :color="getTypeColor(currentLog.usage_type)">{{ getTypeLabel(currentLog.usage_type) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="模型" :span="2">{{ currentLog.model || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="currentLog.status === 'success' ? 'green' : 'red'">
            {{ currentLog.status === 'success' ? '成功' : '失败' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="分辨率">{{ currentLog.resolution || '-' }}</a-descriptions-item>
        <a-descriptions-item label="下载次数">
          <a-tag :color="currentLog.download_count > 0 ? 'green' : 'gray'">
            {{ currentLog.download_count || 0 }}
          </a-tag>
        </a-descriptions-item>
        <template v-if="currentLog.usage_type === 'chat'">
          <a-descriptions-item label="提示 Token">{{ currentLog.prompt_tokens }}</a-descriptions-item>
          <a-descriptions-item label="回复 Token">{{ currentLog.completion_tokens }}</a-descriptions-item>
          <a-descriptions-item label="总 Token" :span="2">{{ currentLog.total_tokens }}</a-descriptions-item>
        </template>
        <a-descriptions-item label="提示词" :span="2">
          <div class="prompt-wrapper" :class="{ collapsed: !promptExpanded }">
            <div class="prompt-content">{{ currentLog.prompt || '-' }}</div>
            <a-button type="text" size="small" class="prompt-toggle" @click="promptExpanded = !promptExpanded">
              <template #icon>
                <icon-up v-if="promptExpanded" />
                <icon-down v-else />
              </template>
              {{ promptExpanded ? '收起' : '展开全部' }}
            </a-button>
          </div>
        </a-descriptions-item>
        <template v-if="currentReferenceImages.length">
          <a-descriptions-item label="参考图" :span="2">
            <div class="reference-gallery">
              <div class="reference-stage">
                <a-button
                  v-if="currentReferenceImages.length > 1"
                  class="reference-nav reference-nav-left"
                  shape="circle"
                  size="small"
                  @click="showPreviousReferenceImage"
                >
                  <template #icon><icon-left /></template>
                </a-button>
                <img :src="activeReferenceImage" class="preview-image ref-preview-image" alt="参考图" />
                <a-button
                  v-if="currentReferenceImages.length > 1"
                  class="reference-nav reference-nav-right"
                  shape="circle"
                  size="small"
                  @click="showNextReferenceImage"
                >
                  <template #icon><icon-right /></template>
                </a-button>
                <div v-if="currentReferenceImages.length > 1" class="reference-counter">
                  {{ activeReferenceImageIndex + 1 }} / {{ currentReferenceImages.length }}
                </div>
              </div>
              <div v-if="currentReferenceImages.length > 1" class="reference-thumbs">
                <button
                  v-for="(url, index) in currentReferenceImages"
                  :key="`${url}-${index}`"
                  type="button"
                  class="reference-thumb"
                  :class="{ active: index === activeReferenceImageIndex }"
                  @click="activeReferenceImageIndex = index"
                >
                  <img :src="url" alt="参考图" />
                </button>
              </div>
            </div>
          </a-descriptions-item>
        </template>
        <template v-if="currentLog.image_url">
          <a-descriptions-item :label="isVideoUsageLog(currentLog) ? '生成视频' : '生成图片'" :span="2">
            <div class="image-preview-wrapper">
              <video
                v-if="isVideoUsageLog(currentLog)"
                :src="currentLog.image_url"
                class="preview-video"
                controls
                playsinline
                preload="metadata"
              ></video>
              <img v-else :src="currentLog.image_url" class="preview-image" alt="生成图片" />
              <a-button type="text" size="small" class="image-download-link" @click="downloadUsageImage(currentLog)">
                <template #icon><icon-download /></template>
                {{ isVideoUsageLog(currentLog) ? '下载原视频' : '下载原图' }}
              </a-button>
            </div>
          </a-descriptions-item>
        </template>
        <a-descriptions-item v-if="currentLog.usage_type === 'chat' && currentLog.result" label="回复内容" :span="2">
          <a-typography-paragraph style="margin: 0; white-space: pre-wrap; max-height: 220px; overflow-y: auto;">
            {{ currentLog.result }}
          </a-typography-paragraph>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import {
  IconDownload,
  IconDown,
  IconLeft,
  IconList,
  IconRefresh,
  IconRight,
  IconUp,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const logs = ref([]);
const loading = ref(false);
const filterUserName = ref('');
const filterType = ref('');
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);

const detailVisible = ref(false);
const currentLog = ref(null);
const promptExpanded = ref(false);
const activeReferenceImageIndex = ref(0);

const currentReferenceImages = computed(() => currentLog.value?.reference_image_urls || []);
const activeReferenceImage = computed(() => currentReferenceImages.value[activeReferenceImageIndex.value] || '');

const fetchLogs = async () => {
  loading.value = true;
  try {
    const params = { page: page.value, page_size: pageSize.value };
    if (filterType.value) params.type = filterType.value;
    if (filterUserName.value) params.user_name = filterUserName.value;
    const res = await request.get('/usage-logs', { params });
    logs.value = (res.data.data || []).map((item) => {
      const referenceImageUrls = resolveReferenceImageUrls(item);
      return {
        ...item,
        image_url: resolveImageUrl(item.image_url),
        reference_image_url: referenceImageUrls[0] || '',
        reference_image_urls: referenceImageUrls,
      };
    });
    total.value = res.data.total || 0;
  } finally {
    loading.value = false;
  }
};

const refreshAll = async () => {
  page.value = 1;
  await fetchLogs();
};

const onPageChange = (p) => {
  page.value = p;
  fetchLogs();
};


const viewDetail = (record) => {
  currentLog.value = record;
  promptExpanded.value = false;
  activeReferenceImageIndex.value = 0;
  detailVisible.value = true;
};

const showPreviousReferenceImage = () => {
  const total = currentReferenceImages.value.length;
  if (total <= 1) return;
  activeReferenceImageIndex.value = (activeReferenceImageIndex.value - 1 + total) % total;
};

const showNextReferenceImage = () => {
  const total = currentReferenceImages.value.length;
  if (total <= 1) return;
  activeReferenceImageIndex.value = (activeReferenceImageIndex.value + 1) % total;
};

const downloadUsageImage = async (record) => {
  if (!record?.id) {
    Message.error('下载失败');
    return;
  }

  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';
  const filename = isVideoUsageLog(record)
    ? `video-${record.id || Date.now()}.mp4`
    : `image-${record.id || Date.now()}.png`;
  const downloadUrl = `${apiBase}/usage-logs/${record.id}/download?filename=${encodeURIComponent(filename)}`;
  try {
    const token = localStorage.getItem('token');
    const res = await fetch(downloadUrl, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const blob = await res.blob();
    triggerBlobDownload(blob, getDownloadFilename(res) || filename);
    await fetchLogs();
    if (currentLog.value?.id === record.id) {
      const updated = logs.value.find((item) => item.id === record.id);
      if (updated) currentLog.value = updated;
    }
  } catch {
    Message.error('下载失败');
  }
};

const resolveImageUrl = (url) => {
  if (!url || url.startsWith('data:') || url.startsWith('http://') || url.startsWith('https://')) return url;
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';
  if (apiBase.startsWith('http')) return new URL(apiBase).origin + url;
  return url;
};

const resolveReferenceImageUrls = (item) => {
  const rawList = item.reference_image_urls;
  let urls = [];
  if (Array.isArray(rawList)) {
    urls = rawList;
  } else if (typeof rawList === 'string' && rawList.trim()) {
    try {
      const parsed = JSON.parse(rawList);
      if (Array.isArray(parsed)) urls = parsed;
    } catch {
      urls = [];
    }
  }
  if (!urls.length && item.reference_image_url) {
    urls = [item.reference_image_url];
  }
  const seen = new Set();
  return urls
    .map((url) => resolveImageUrl(String(url || '').trim()))
    .filter((url) => {
      if (!url || seen.has(url)) return false;
      seen.add(url);
      return true;
    });
};

const formatDurationSeconds = (duration) => {
  if (duration == null || duration === '') return '-';
  const ms = Number(duration);
  if (!Number.isFinite(ms)) return '-';
  return `${(ms / 1000).toFixed(2)} s`;
};

const triggerBlobDownload = (blob, filename) => {
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = objectUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(objectUrl);
};

const getDownloadFilename = (response) => {
  const disposition = response.headers.get('content-disposition');
  if (!disposition) return '';
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) return decodeURIComponent(utf8Match[1]);
  const match = disposition.match(/filename="?([^"]+)"?/i);
  return match?.[1] || '';
};

const getTypeLabel = (type) => {
  if (type === 'chat') return '对话';
  if (type === 'drawing') return '绘图';
  if (type === 'video') return '视频';
  if (type === 'image_translate') return '图片翻译';
  return type || '-';
};

const getTypeColor = (type) => {
  if (type === 'chat') return 'arcoblue';
  if (type === 'drawing') return 'purple';
  if (type === 'video') return 'orangered';
  if (type === 'image_translate') return 'cyan';
  return 'gray';
};

const isVideoUsageLog = (record) => {
  return record?.usage_type === 'video' ||
    record?.source === 'video_generate' ||
    /\.(mp4|webm|mov|m4v|avi)(\?|#|$)/i.test(String(record?.image_url || ''));
};

const formatDateTime = (value) => {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date);
};

onMounted(() => {
  refreshAll();
});
</script>

<style scoped>
.page-container { padding: 24px; background: #f7f8fa; min-height: 100%; }
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 16px; }
.page-title { margin: 0; font-size: 22px; font-weight: 700; color: #1d2129; }
.page-subtitle { margin: 6px 0 0; color: #86909c; font-size: 13px; line-height: 1.6; }
.toolbar-card { margin-bottom: 16px; padding: 14px 16px; background: #fff; border: 1px solid #eef0f3; border-radius: 12px; box-shadow: 0 6px 18px rgba(15, 35, 95, 0.04); }
.content-layout { min-width: 0; }
.main-panel { min-width: 0; }
.panel-card { border-radius: 12px; box-shadow: 0 6px 18px rgba(15, 35, 95, 0.05); border: 1px solid #eef0f3; }
.card-title { display: flex; align-items: center; gap: 8px; font-weight: 600; color: #1d2129; }
.card-title-icon { font-size: 18px; color: #4080ff; }
.id-pill { display: inline-flex; align-items: center; justify-content: center; min-width: 34px; height: 24px; padding: 0 8px; border-radius: 999px; background: rgba(15, 23, 42, 0.06); font-variant-numeric: tabular-nums; }
.metric-number { display: inline-flex; align-items: center; justify-content: flex-end; white-space: nowrap; font-variant-numeric: tabular-nums; }
.text-muted { color: #86909c; }
.time-text { white-space: nowrap; font-variant-numeric: tabular-nums; color: #4e5969; }
.token-usage { display: inline-flex; align-items: center; gap: 8px; }
.token-detail { font-size: 12px; color: #86909c; white-space: nowrap; }
.image-preview-wrapper { display: flex; flex-direction: column; gap: 8px; }
.preview-image { max-width: 100%; max-height: 400px; object-fit: contain; border-radius: 10px; border: 1px solid #e5e8ef; }
.preview-video { width: 100%; max-height: 460px; object-fit: contain; border-radius: 10px; border: 1px solid #e5e8ef; background: #000; }
.ref-preview-image { max-height: 280px; }
.reference-gallery { display: flex; flex-direction: column; gap: 10px; width: 100%; }
.reference-stage { position: relative; display: flex; align-items: center; justify-content: center; min-height: 180px; max-height: 280px; overflow: hidden; border-radius: 10px; background: #f7f8fa; border: 1px solid #e5e8ef; }
.reference-stage .ref-preview-image { max-height: 280px; border: 0; border-radius: 0; background: transparent; }
.reference-nav { position: absolute; top: 50%; z-index: 2; transform: translateY(-50%); background: rgba(255, 255, 255, 0.92); box-shadow: 0 4px 12px rgba(15, 23, 42, 0.12); }
.reference-nav-left { left: 10px; }
.reference-nav-right { right: 10px; }
.reference-counter { position: absolute; right: 10px; bottom: 10px; padding: 2px 8px; border-radius: 999px; background: rgba(15, 23, 42, 0.68); color: #fff; font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; }
.reference-thumbs { display: flex; gap: 8px; max-width: 100%; overflow-x: auto; padding: 2px 2px 6px; }
.reference-thumb { flex: 0 0 58px; width: 58px; height: 58px; padding: 0; border: 2px solid transparent; border-radius: 8px; background: #f7f8fa; overflow: hidden; cursor: pointer; transition: border-color 0.15s ease, box-shadow 0.15s ease; }
.reference-thumb:hover { border-color: #94bfff; }
.reference-thumb.active { border-color: #165dff; box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.12); }
.reference-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
.image-download-link { display: inline-flex; align-items: center; gap: 4px; align-self: flex-start; font-size: 13px; padding-left: 0; }
.prompt-wrapper { display: flex; flex-direction: column; gap: 8px; }
.prompt-content { white-space: pre-wrap; word-break: break-all; font-size: 13px; line-height: 1.6; color: #1d2129; }
.prompt-wrapper.collapsed .prompt-content { max-height: 120px; overflow: hidden; position: relative; }
.prompt-wrapper.collapsed .prompt-content::after { content: ''; position: absolute; bottom: 0; left: 0; right: 0; height: 40px; background: linear-gradient(transparent, #fff); }
.prompt-toggle { align-self: flex-start; padding-left: 0; }
.arco-table :deep(.arco-table-th) { background: #f7f8fa; }
.arco-table :deep(.arco-table-tr:hover .arco-table-td) { background: #fcfcff; }
@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: stretch; }
}
</style>
