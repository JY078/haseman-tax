<template>
  <main class="page-container">
    <header class="page-header">
      <div class="title-block">
        <h1 class="page-title">图片统计</h1>
      </div>
      <div class="report-actions">
        <a-button type="outline" :loading="reportPreviewLoading" @click="previewReport">
          <template #icon><icon-file-pdf /></template>
          预览 PDF
        </a-button>
        <a-button type="primary" :loading="reportSendLoading" @click="sendReport">
          <template #icon><icon-send /></template>
          发送到飞书
        </a-button>
      </div>
    </header>

    <section class="filter-bar" aria-label="统计筛选">
      <div class="filter-group">
        <div class="filter-field">
          <label class="filter-label">统计月份</label>
          <a-month-picker
            v-model="selectedMonth"
            value-format="YYYY-MM"
            placeholder="全部月份"
            allow-clear
            style="width: 180px"
            @change="applyFilters"
            @clear="applyFilters"
          />
        </div>
        <div class="filter-field">
          <label class="filter-label">人员</label>
          <a-select
            v-model="selectedUserID"
            placeholder="全部人员"
            allow-clear
            allow-search
            style="width: 220px"
            @change="applyFilters"
            @clear="applyFilters"
          >
            <a-option v-for="user in filterUsers" :key="user.id" :value="user.id">
              {{ user.name || user.username }}{{ user.name && user.username ? `（${user.username}）` : '' }}
            </a-option>
          </a-select>
        </div>
      </div>
    </section>

    <section class="overview-grid" aria-label="统计概览">
      <div class="metrics-rail" aria-label="核心指标">
        <article class="metric-item generated-metric">
          <div class="summary-label">总生成量</div>
          <div class="summary-value">{{ formatCount(summary.generated_count) }}</div>
        </article>
        <article class="metric-item download-metric">
          <div class="summary-label">总下载量</div>
          <div class="summary-value">{{ formatCount(summary.download_count) }}</div>
        </article>
        <article class="metric-item rate-metric">
          <div class="summary-label">总下载率</div>
          <div class="summary-value">{{ formatRate(summary.download_rate) }}</div>
        </article>
      </div>

      <section class="trend-panel">
        <div class="panel-heading">
          <h2>整体趋势</h2>
        </div>
        <div class="trend-chart-wrap">
          <canvas ref="trendCanvas" aria-label="图片统计趋势图"></canvas>
          <div v-if="!trend.length" class="trend-empty">暂无趋势数据</div>
        </div>
      </section>
    </section>

    <section class="department-panel">
      <div class="panel-heading department-heading">
        <h2>部门趋势</h2>
      </div>
      <div class="department-grid">
        <article v-for="department in departments" :key="department" class="department-chart-card">
          <div class="department-chart-heading">
            <h3>{{ department }}</h3>
            <span>{{ departmentTotals[department] || 0 }} 张</span>
          </div>
          <div class="department-chart-wrap">
            <canvas :ref="(el) => setDepartmentCanvas(department, el)" :aria-label="`${department}图片统计趋势图`"></canvas>
            <div v-if="!departmentTrends[department]?.length" class="trend-empty">暂无数据</div>
          </div>
        </article>
      </div>
    </section>

    <section class="table-panel">
      <div class="table-heading">
        <h2>人员明细</h2>
        <span class="record-count">共 {{ total }} 人</span>
      </div>

      <a-table
        :data="rows"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 720, y: 560 }"
        size="medium"
        row-key="user_id"
      >
        <template #columns>
          <a-table-column title="人员" :width="320">
            <template #cell="{ record }">
              <div class="user-cell">
                <a-avatar :size="34" class="user-avatar">{{ getInitial(record) }}</a-avatar>
                <div class="user-info">
                  <span class="user-name">{{ record.name || record.username || '-' }}</span>
                  <span class="user-account">{{ record.username ? `@${record.username}` : '未设置用户名' }}</span>
                </div>
              </div>
            </template>
          </a-table-column>
          <a-table-column :width="160" align="right">
            <template #title>
              <sort-button field="generated_count" label="生成量" />
            </template>
            <template #cell="{ record }">
              <span class="metric-number">{{ formatCount(record.generated_count) }}</span>
            </template>
          </a-table-column>
          <a-table-column :width="160" align="right">
            <template #title>
              <sort-button field="download_count" label="下载量" />
            </template>
            <template #cell="{ record }">
              <span class="metric-number">{{ formatCount(record.download_count) }}</span>
            </template>
          </a-table-column>
          <a-table-column :width="200" align="right">
            <template #title>
              <sort-button field="download_rate" label="下载率" />
            </template>
            <template #cell="{ record }">
              <div class="rate-cell">
                <span class="rate-track" aria-hidden="true">
                  <span :style="{ width: rateProgressWidth(record.download_rate) }"></span>
                </span>
                <span class="rate-value">{{ formatRate(record.download_rate) }}</span>
              </div>
            </template>
          </a-table-column>
        </template>
        <template #empty>
          <div class="empty-state">
            <icon-empty />
            <span>暂无统计数据</span>
          </div>
        </template>
      </a-table>
    </section>
  </main>
</template>

<script setup>
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { Chart, registerables } from 'chart.js';
Chart.register(...registerables);
import {
  IconCaretDown,
  IconCaretUp,
  IconEmpty,
  IconFilePdf,
  IconSend,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

const rows = ref([]);
const loading = ref(false);
const selectedMonth = ref('');
const selectedUserID = ref(null);
const filterUsers = ref([]);
const total = ref(0);
const summary = ref({ generated_count: 0, download_count: 0, download_rate: 0 });
const sort = ref({ field: 'generated_count', direction: 'descend' });
const trend = ref([]);
const trendCanvas = ref(null);
let trendChart;
const departments = ['美国1部', '美国2部', '美国3部', '欧洲1部', '设计部', '多平台', '其他'];
const departmentTrends = ref({});
const departmentCanvases = {};
const departmentCharts = {};
const departmentTotals = computed(() => Object.fromEntries(departments.map((department) => [
  department,
  (departmentTrends.value[department] || []).reduce((totalValue, item) => totalValue + Number(item.generated_count || 0), 0),
])));
const reportPreviewLoading = ref(false);
const reportSendLoading = ref(false);

const SortButton = (props) => h(
  'button',
  {
    type: 'button',
    class: 'sort-button',
    'aria-sort': sort.value.field === props.field
      ? (sort.value.direction === 'ascend' ? 'ascending' : 'descending')
      : 'none',
    onClick: () => toggleSort(props.field),
  },
  [
    h('span', props.label),
    h('span', { class: 'sort-arrows', 'aria-hidden': 'true' }, [
      h(IconCaretUp, {
        class: ['sort-arrow', { active: sort.value.field === props.field && sort.value.direction === 'ascend' }],
      }),
      h(IconCaretDown, {
        class: ['sort-arrow', { active: sort.value.field === props.field && sort.value.direction === 'descend' }],
      }),
    ]),
  ],
);

const fetchStats = async () => {
  loading.value = true;
  try {
    const params = {
      unpaged: 1,
      sort_field: sort.value.field,
      sort_direction: sort.value.direction,
    };
    if (selectedMonth.value) params.month = selectedMonth.value;
    if (selectedUserID.value) params.user_id = selectedUserID.value;
    const res = await request.get('/usage-logs/image-user-stats', { params });
    rows.value = res.data.data || [];
    total.value = res.data.total || 0;
    summary.value = res.data.summary || { generated_count: 0, download_count: 0, download_rate: 0 };
    trend.value = res.data.trend || [];
    departmentTrends.value = res.data.department_trends || {};
    filterUsers.value = res.data.filter_users || [];
    await nextTick();
    renderAllCharts();
  } catch {
    Message.error('图片统计加载失败');
  } finally {
    loading.value = false;
  }
};

const applyFilters = () => {
  fetchStats();
};

const toggleSort = (field) => {
  const direction = sort.value.field === field && sort.value.direction === 'descend' ? 'ascend' : 'descend';
  sort.value = { field, direction };
  fetchStats();
};

const formatCount = (value) => Number(value || 0).toLocaleString('zh-CN');
const formatRate = (value) => `${(Number(value || 0) * 100).toFixed(1)}%`;
const rateProgressWidth = (value) => `${Math.min(Math.max(Number(value || 0) * 100, 0), 100)}%`;
const getInitial = (record) => String(record.name || record.username || '?').trim().slice(0, 1).toUpperCase();

const chartConfig = (data, compact = false) => ({
    type: 'line',
    data: {
      labels: data.map((item) => item.period),
      datasets: [
        { label: '生成量', data: data.map((item) => Number(item.generated_count || 0)), borderColor: '#165DFF', borderWidth: 2, pointRadius: data.length > 20 ? 0 : compact ? 2 : 3, pointHoverRadius: 5, tension: 0.32, fill: false, yAxisID: 'quantity' },
        { label: '下载量', data: data.map((item) => Number(item.download_count || 0)), borderColor: '#00B42A', borderWidth: 2, pointRadius: data.length > 20 ? 0 : compact ? 2 : 3, pointHoverRadius: 5, tension: 0.32, fill: false, yAxisID: 'quantity' },
        { label: '下载率', data: data.map((item) => Number(item.download_rate || 0) * 100), borderColor: '#FF7D00', borderWidth: 2, pointRadius: data.length > 20 ? 0 : compact ? 2 : 3, pointHoverRadius: 5, tension: 0.32, fill: false, yAxisID: 'rate' },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { display: !compact, position: 'top', align: 'end', labels: { usePointStyle: true, boxWidth: 8, color: '#4E5969' } },
        tooltip: { callbacks: { label: (ctx) => `${ctx.dataset.label}: ${ctx.dataset.yAxisID === 'rate' ? ctx.parsed.y.toFixed(1) + '%' : ctx.parsed.y.toLocaleString('zh-CN')}` } },
      },
      scales: {
        x: { grid: { display: false }, ticks: { color: '#86909C', maxTicksLimit: compact ? 5 : 10, maxRotation: 0 } },
        rate: { beginAtZero: true, position: 'left', grid: { color: '#EEF0F3' }, ticks: { color: '#FF7D00', maxTicksLimit: 5, callback: (value) => `${value}%` } },
        quantity: { beginAtZero: true, position: 'right', grid: { drawOnChartArea: false }, ticks: { color: '#86909C', maxTicksLimit: 5, callback: (value) => Number(value).toLocaleString('zh-CN') } },
      },
    },
  });

const renderTrendChart = () => {
  if (!trendCanvas.value) return;
  if (trendChart) trendChart.destroy();
  trendChart = new Chart(trendCanvas.value, chartConfig(trend.value));
};

const setDepartmentCanvas = (department, element) => {
  if (element) departmentCanvases[department] = element;
};

const renderDepartmentCharts = () => {
  departments.forEach((department) => {
    if (departmentCharts[department]) departmentCharts[department].destroy();
    const canvas = departmentCanvases[department];
    if (canvas) departmentCharts[department] = new Chart(canvas, chartConfig(departmentTrends.value[department] || [], true));
  });
};

const renderAllCharts = () => {
  renderTrendChart();
  renderDepartmentCharts();
};

const reportQuery = () => (selectedMonth.value ? { month: selectedMonth.value } : {});

const previewReport = async () => {
  reportPreviewLoading.value = true;
  const previewWindow = window.open('', '_blank');
  try {
    const res = await request.get('/usage-logs/image-monthly-report/pdf', {
      params: reportQuery(),
      responseType: 'blob',
    });
    const url = URL.createObjectURL(res.data);
    if (previewWindow) {
      previewWindow.location.href = url;
      window.setTimeout(() => URL.revokeObjectURL(url), 60 * 1000);
    } else {
      const link = document.createElement('a');
      link.href = url;
      link.download = `图片统计月报-${selectedMonth.value || '上月'}.pdf`;
      link.click();
      URL.revokeObjectURL(url);
    }
  } catch {
    if (previewWindow) previewWindow.close();
    Message.error('PDF 预览生成失败');
  } finally {
    reportPreviewLoading.value = false;
  }
};

const sendReport = async () => {
  reportSendLoading.value = true;
  try {
    await request.post('/usage-logs/image-monthly-report/send', null, { params: reportQuery() });
    Message.success(`${selectedMonth.value || '上月'}图片统计月报已发送到飞书`);
  } catch {
    // request interceptor already surfaces the server error.
  } finally {
    reportSendLoading.value = false;
  }
};

onMounted(fetchStats);
onBeforeUnmount(() => {
  if (trendChart) trendChart.destroy();
  Object.values(departmentCharts).forEach((chart) => chart.destroy());
});
</script>

<style scoped>
.page-container {
  --accent: #2468d8;
  --canvas: #f5f6f8;
  --surface: #fff;
  --line: #e4e7eb;
  --line-soft: #eef0f3;
  --text: #1f2329;
  --text-secondary: #5e6673;
  --text-muted: #8a919e;
  min-height: 100%;
  padding: 22px clamp(20px, 3vw, 36px) 40px;
  background: var(--canvas);
  color: var(--text);
}
.page-container > * { width: min(100%, 1600px); margin-right: auto; margin-left: auto; }
.page-header { display: flex; min-height: 40px; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 14px; }
.title-block { min-width: 0; }
.page-title { margin: 0; color: var(--text); font-size: 22px; font-weight: 700; line-height: 1.3; }
.report-actions { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px; }
.filter-bar { display: flex; min-height: 68px; align-items: center; margin-bottom: 14px; padding: 12px 16px; background: var(--surface); border: 1px solid var(--line); border-radius: 6px; }
.filter-group { display: flex; align-items: flex-end; gap: 12px; min-width: 0; }
.filter-field { display: flex; flex-direction: column; gap: 6px; }
.filter-label { color: var(--text-secondary); font-size: 12px; font-weight: 600; line-height: 1; }
.overview-grid { display: grid; grid-template-columns: minmax(220px, 270px) minmax(0, 1fr); gap: 14px; margin-bottom: 20px; }
.metrics-rail { display: grid; overflow: hidden; background: var(--surface); border: 1px solid var(--line); border-radius: 6px; }
.metric-item { position: relative; display: flex; min-width: 0; flex-direction: column; justify-content: center; padding: 20px 22px 18px; }
.metric-item + .metric-item { border-top: 1px solid var(--line-soft); }
.metric-item::before { position: absolute; top: 20px; bottom: 20px; left: 0; width: 3px; content: ''; }
.generated-metric::before { background: #2468d8; }
.download-metric::before { background: #16865c; }
.rate-metric::before { background: #c76a1b; }
.summary-label { margin-bottom: 7px; color: var(--text-muted); font-size: 12px; font-weight: 600; }
.summary-value { overflow: hidden; color: var(--text); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: clamp(27px, 2.2vw, 34px); font-weight: 650; line-height: 1.1; font-variant-numeric: tabular-nums; text-overflow: ellipsis; white-space: nowrap; }
.trend-panel, .department-panel, .table-panel { overflow: hidden; background: var(--surface); border: 1px solid var(--line); border-radius: 6px; }
.trend-panel { min-width: 0; padding-bottom: 12px; }
.panel-heading, .table-heading { display: flex; min-height: 52px; align-items: center; justify-content: space-between; gap: 16px; padding: 0 18px; border-bottom: 1px solid var(--line-soft); }
.panel-heading h2, .table-heading h2 { margin: 0; color: var(--text); font-size: 15px; font-weight: 650; }
.trend-chart-wrap { position: relative; height: 324px; padding: 14px 16px 0; }
.trend-empty { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 13px; }
.department-panel { margin-bottom: 20px; }
.department-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.department-chart-card { min-width: 0; padding: 16px 18px 12px; border-right: 1px solid var(--line-soft); border-bottom: 1px solid var(--line-soft); }
.department-chart-card:nth-child(even) { border-right: 0; }
.department-chart-card:nth-last-child(-n + 2) { border-bottom: 0; }
.department-chart-card:last-child:nth-child(odd) { grid-column: 1 / -1; border-right: 0; border-bottom: 0; }
.department-chart-heading { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-bottom: 7px; }
.department-chart-heading h3 { margin: 0; color: var(--text-secondary); font-size: 13px; font-weight: 650; }
.department-chart-heading span { color: var(--text-muted); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.department-chart-wrap { position: relative; height: 214px; }
.table-heading { min-height: 54px; }
.record-count { color: var(--text-muted); font-size: 12px; white-space: nowrap; }
.user-cell { display: flex; align-items: center; gap: 11px; min-width: 0; }
.user-avatar { flex: 0 0 auto; background: #edf3fc; color: var(--accent); font-size: 14px; font-weight: 600; }
.user-info { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.user-name { overflow: hidden; color: var(--text); font-size: 14px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.user-account { overflow: hidden; color: var(--text-muted); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.metric-number, .rate-value { color: var(--text); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 13px; font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap; }
.rate-cell { display: flex; align-items: center; justify-content: flex-end; gap: 12px; }
.rate-track { display: block; width: 64px; height: 4px; overflow: hidden; border-radius: 2px; background: #edf0f4; }
.rate-track span { display: block; height: 100%; border-radius: inherit; background: var(--accent); }
.rate-value { min-width: 48px; color: var(--accent); text-align: right; }
.sort-button { display: inline-flex; align-items: center; justify-content: flex-end; gap: 4px; width: 100%; padding: 0; border: 0; background: transparent; color: var(--text-secondary); font: inherit; font-weight: 600; cursor: pointer; }
.sort-button:hover { color: var(--accent); }
.sort-arrows { display: inline-flex; flex-direction: column; justify-content: center; margin-left: 1px; line-height: 0; }
.sort-arrow { width: 10px; height: 8px; color: #c5cad1; font-size: 10px; opacity: 0.9; }
.sort-arrow + .sort-arrow { margin-top: -2px; }
.sort-arrow.active { color: var(--accent); opacity: 1; }
.empty-state { display: flex; min-height: 220px; align-items: center; justify-content: center; flex-direction: column; gap: 10px; color: #c5cad1; font-size: 13px; }
.empty-state svg { font-size: 34px; }
.table-panel :deep(.arco-table-th) { height: 42px; background: #f7f8fa; color: var(--text-secondary); font-weight: 600; }
.table-panel :deep(.arco-table-td) { height: 56px; border-color: var(--line-soft); }
.table-panel :deep(.arco-table-tr:hover .arco-table-td) { background: #f6f9fe; }
.table-panel :deep(.arco-scrollbar-track-direction-vertical) { right: 3px; }
@media (max-width: 1024px) {
  .overview-grid { grid-template-columns: 1fr; }
  .metrics-rail { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .metric-item { min-height: 104px; }
  .metric-item + .metric-item { border-top: 0; border-left: 1px solid var(--line-soft); }
  .metric-item::before { top: 0; right: 20px; bottom: auto; left: 20px; width: auto; height: 3px; }
}
@media (max-width: 768px) {
  .page-container { padding: 16px 14px 28px; }
  .page-header { align-items: stretch; flex-direction: column; gap: 12px; }
  .report-actions { justify-content: stretch; }
  .report-actions :deep(.arco-btn) { flex: 1 1 0; }
  .filter-bar { padding: 14px; }
  .filter-group { align-items: stretch; flex-direction: column; gap: 12px; }
  .filter-field :deep(.arco-picker), .filter-field :deep(.arco-input-wrapper), .filter-field :deep(.arco-select-view) { width: 100% !important; }
  .metric-item { min-height: 92px; padding: 18px 14px 14px; }
  .summary-value { font-size: 23px; }
  .panel-heading, .table-heading { padding: 0 14px; }
  .trend-chart-wrap { height: 260px; padding: 12px 8px 0; }
  .department-grid { grid-template-columns: 1fr; }
  .department-chart-card, .department-chart-card:nth-child(even), .department-chart-card:nth-last-child(-n + 2) { border-right: 0; border-bottom: 1px solid var(--line-soft); }
  .department-chart-card:last-child:nth-child(odd), .department-chart-card:last-child { grid-column: auto; border-bottom: 0; }
  .department-chart-wrap { height: 210px; }
}
</style>
