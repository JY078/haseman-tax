<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h1 class="page-title">仪表盘</h1>
        <p class="page-subtitle">系统数据总览</p>
      </div>
      <a-button type="primary" @click="refreshData" :loading="loading">
        <template #icon><icon-refresh /></template>
        刷新数据
      </a-button>
    </div>

    <a-row :gutter="16" class="stat-cards">
      <a-col :span="6">
        <a-card class="stat-card" :bordered="false">
          <a-spin :loading="loading">
            <div class="stat-card-content">
              <div class="stat-icon stat-icon-blue"><icon-user /></div>
              <div class="stat-info">
                <div class="stat-label">总用户数</div>
                <div class="stat-value">{{ formatNumber(stats.total_users) }}</div>
                <div class="stat-trend">
                  <span class="trend-up"><icon-arrow-rise />{{ stats.today_users || 0 }}</span>
                  <span class="trend-text">今日新增</span>
                </div>
              </div>
            </div>
          </a-spin>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" :bordered="false">
          <a-spin :loading="loading">
            <div class="stat-card-content">
              <div class="stat-icon stat-icon-purple"><icon-image /></div>
              <div class="stat-info">
                <div class="stat-label">总使用量</div>
                <div class="stat-value">{{ formatNumber(stats.total_chat + stats.total_drawing) }}</div>
                <div class="stat-trend">
                  <span class="trend-up"><icon-arrow-rise />{{ (stats.today_chat || 0) + (stats.today_drawing || 0) }}</span>
                  <span class="trend-text">今日使用</span>
                </div>
              </div>
            </div>
          </a-spin>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" :bordered="false">
          <a-spin :loading="loading">
            <div class="stat-card-content">
              <div class="stat-icon stat-icon-pink"><icon-image /></div>
              <div class="stat-info">
                <div class="stat-label">总图片生成数</div>
                <div class="stat-value">{{ formatNumber(stats.total_image_generated) }}</div>
                <div class="stat-trend">
                  <span class="trend-up"><icon-arrow-rise />{{ stats.today_image_generated || 0 }}</span>
                  <span class="trend-text">今日生成</span>
                </div>
              </div>
            </div>
          </a-spin>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" :bordered="false">
          <a-spin :loading="loading">
            <div class="stat-card-content">
              <div class="stat-icon stat-icon-gray"><icon-apps /></div>
              <div class="stat-info">
                <div class="stat-label">图片下载总数</div>
                <div class="stat-value">{{ formatNumber(stats.total_image_downloads) }}</div>
                <div class="stat-trend">
                  <span class="trend-text">基于 download_count 统计</span>
                </div>
              </div>
            </div>
          </a-spin>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="12">
        <a-card class="chart-card">
          <template #title>
            <div class="card-title">
              <icon-bar-chart class="card-title-icon" />
              <span>近 30 天使用量对比</span>
            </div>
          </template>
          <template #extra><a-tag color="arcoblue" size="small">对话 vs 绘图</a-tag></template>
          <div class="chart-wrap">
            <a-spin :loading="loading" class="chart-loading">
              <canvas id="usageChart"></canvas>
            </a-spin>
          </div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card class="chart-card">
          <template #title>
            <div class="card-title">
              <icon-list class="card-title-icon" />
              <span>图片生成与下载</span>
            </div>
          </template>
          <template #extra><a-tag color="arcoblue" size="small">生成 vs 下载</a-tag></template>
          <div class="chart-wrap chart-wrap-tall">
            <a-spin :loading="loading" class="chart-loading">
              <canvas id="imageStatsChart"></canvas>
            </a-spin>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16">
      <a-col :span="24">
        <a-card class="chart-card">
          <template #title>
            <div class="card-title">
              <icon-dashboard class="card-title-icon" />
              <span>历史数据对比</span>
            </div>
          </template>
          <template #extra><a-tag color="purple" size="small">对话 + 绘图 + 图片生成</a-tag></template>
          <div class="chart-wrap">
            <a-spin :loading="loading" class="chart-loading">
              <canvas id="historyChart"></canvas>
            </a-spin>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="24">
        <a-card class="chart-card">
          <template #title>
            <div class="card-title">
              <icon-apps class="card-title-icon" />
              <span>模型使用次数</span>
            </div>
          </template>
          <template #extra><a-tag color="arcoblue" size="small">Top 10</a-tag></template>
          <div class="chart-wrap chart-wrap-model">
            <a-spin :loading="loading" class="chart-loading">
              <canvas id="modelRankChart"></canvas>
            </a-spin>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import {
  Chart,
  LineController,
  BarController,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import {
  IconRefresh,
  IconUser,
  IconImage,
  IconBarChart,
  IconList,
  IconDashboard,
  IconArrowRise,
  IconApps,
} from '@arco-design/web-vue/es/icon';
import request from '@/utils/request';

Chart.register(LineController, BarController, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Tooltip, Legend, Filler);

const loading = ref(true);
const stats = ref({
  total_users: 0,
  total_chat: 0,
  total_drawing: 0,
  total_image_generated: 0,
  total_image_downloads: 0,
  today_users: 0,
  today_chat: 0,
  today_drawing: 0,
  today_image_generated: 0,
  model_count: 0,
  model_rank: [],
});

const chartInstances = {};

const destroyAll = () => {
  Object.values(chartInstances).forEach((chart) => chart.destroy());
  Object.keys(chartInstances).forEach((key) => delete chartInstances[key]);
};

const makeChart = (id, config) => {
  const el = document.getElementById(id);
  if (!el) return;
  if (chartInstances[id]) chartInstances[id].destroy();
  chartInstances[id] = new Chart(el, config);
};

const formatNumber = (num) => {
  if (num >= 10000) return `${(num / 10000).toFixed(1)}w`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}k`;
  return num || 0;
};

const buildCharts = (d) => {
  const gridColor = 'rgba(0,0,0,0.06)';
  const fontFamily = '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

  makeChart('usageChart', {
    type: 'line',
    data: {
      labels: d.labels || [],
      datasets: [
        {
          label: '对话',
          data: d.daily_chat || [],
          borderColor: '#3370ff',
          backgroundColor: 'rgba(51,112,255,0.1)',
          fill: true,
          tension: 0.4,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#3370ff',
          pointBorderColor: '#fff',
          pointBorderWidth: 2,
        },
        {
          label: '绘图',
          data: d.daily_drawing || [],
          borderColor: '#7b5ef8',
          backgroundColor: 'rgba(123,94,248,0.1)',
          fill: true,
          tension: 0.4,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#7b5ef8',
          pointBorderColor: '#fff',
          pointBorderWidth: 2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { position: 'top', labels: { font: { family: fontFamily, size: 12 }, usePointStyle: true, padding: 20 } },
        tooltip: { backgroundColor: 'rgba(0,0,0,0.8)', titleFont: { family: fontFamily, size: 13 }, bodyFont: { family: fontFamily, size: 12 }, cornerRadius: 8, padding: 12 },
      },
      scales: {
        x: { grid: { display: false }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c' } },
        y: { beginAtZero: true, grid: { color: gridColor }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c', precision: 0 } },
      },
    },
  });

  makeChart('imageStatsChart', {
    type: 'bar',
    data: {
      labels: ['总生成图片', '总下载次数'],
      datasets: [{
        label: '数量',
        data: [d.total_image_generated || 0, d.total_image_downloads || 0],
        backgroundColor: ['rgba(245, 63, 63, 0.8)', 'rgba(64, 128, 255, 0.8)'],
        borderRadius: 8,
        borderSkipped: false,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { display: false }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c' } },
        y: { beginAtZero: true, grid: { color: gridColor }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c', precision: 0 } },
      },
    },
  });

  makeChart('historyChart', {
    type: 'bar',
    data: {
      labels: d.labels || [],
      datasets: [
        { label: '对话', data: d.daily_chat || [], backgroundColor: 'rgba(54,162,235,0.8)', borderRadius: 6, stack: 'usage' },
        { label: '绘图', data: d.daily_drawing || [], backgroundColor: 'rgba(153,102,255,0.8)', borderRadius: 6, stack: 'usage' },
        { label: '图片生成', data: d.daily_image_generated || [], backgroundColor: 'rgba(245,63,63,0.8)', borderRadius: 6, stack: 'usage' },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { position: 'top', labels: { font: { family: fontFamily, size: 12 }, usePointStyle: true, padding: 20 } } },
      scales: {
        x: { stacked: true, grid: { display: false }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c' } },
        y: { stacked: true, beginAtZero: true, grid: { color: gridColor }, ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c', precision: 0 } },
      },
    },
  });

  const modelLabels = (d.model_rank || []).map((item) => item.model || '-');
  const modelCounts = (d.model_rank || []).map((item) => item.count || 0);
  makeChart('modelRankChart', {
    type: 'bar',
    data: {
      labels: modelLabels,
      datasets: [
        {
          label: '使用次数',
          data: modelCounts,
          backgroundColor: 'rgba(64, 128, 255, 0.82)',
          borderRadius: 8,
          borderSkipped: false,
          barThickness: 14,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      indexAxis: 'y',
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: 'rgba(0,0,0,0.8)',
          titleFont: { family: fontFamily, size: 13 },
          bodyFont: { family: fontFamily, size: 12 },
          cornerRadius: 8,
          padding: 12,
        },
      },
      scales: {
        x: {
          beginAtZero: true,
          grid: { color: gridColor },
          ticks: { font: { family: fontFamily, size: 11 }, color: '#86909c', precision: 0 },
        },
        y: {
          grid: { display: false },
          ticks: {
            font: { family: fontFamily, size: 11 },
            color: '#4e5969',
            callback(value) {
              const label = this.getLabelForValue(value);
              return label.length > 18 ? `${label.slice(0, 18)}...` : label;
            },
          },
        },
      },
    },
  });
};

const refreshData = async () => {
  loading.value = true;
  try {
    const { data } = await request.get('/dashboard/stats');
    stats.value = {
      total_users: data.total_users || 0,
      total_chat: data.total_chat || 0,
      total_drawing: data.total_drawing || 0,
      total_image_generated: data.total_image_generated || 0,
      total_image_downloads: data.total_image_downloads || 0,
      today_users: data.today_users || 0,
      today_chat: data.today_chat || 0,
      today_drawing: data.today_drawing || 0,
      today_image_generated: data.today_image_generated || 0,
      model_count: data.model_count || 0,
      model_rank: data.model_rank || [],
    };
    buildCharts(data);
  } finally {
    loading.value = false;
  }
};

onMounted(refreshData);
onBeforeUnmount(destroyAll);
</script>

<style scoped>
.page-container { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; gap: 16px; }
.page-title { margin: 0; font-size: 24px; font-weight: 700; color: #1d2129; }
.page-subtitle { margin: 6px 0 0; color: #86909c; font-size: 14px; }
.stat-cards { margin-bottom: 16px; }
.stat-card { border-radius: 14px; box-shadow: 0 6px 20px rgba(15, 35, 95, 0.06); }
.stat-card-content { display: flex; align-items: center; gap: 14px; }
.stat-icon { width: 48px; height: 48px; border-radius: 14px; display: flex; align-items: center; justify-content: center; font-size: 22px; }
.stat-icon-blue { background: rgba(64, 128, 255, 0.12); color: #4080ff; }
.stat-icon-purple { background: rgba(123, 94, 248, 0.12); color: #7b5ef8; }
.stat-icon-pink { background: rgba(245, 63, 63, 0.12); color: #f53f3f; }
.stat-icon-gray { background: rgba(134, 144, 156, 0.12); color: #4e5969; }
.stat-info { flex: 1; min-width: 0; }
.stat-label { font-size: 13px; color: #86909c; margin-bottom: 6px; }
.stat-value { font-size: 28px; line-height: 1; font-weight: 700; color: #1d2129; margin-bottom: 10px; }
.stat-trend { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.trend-up { display: inline-flex; align-items: center; gap: 4px; color: #00b42a; font-weight: 600; }
.trend-text { color: #86909c; }
.chart-card { border-radius: 16px; box-shadow: 0 10px 24px rgba(15, 35, 95, 0.06); }
.card-title { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.card-title-icon { font-size: 18px; color: #4080ff; }
.metric-number { display: inline-flex; align-items: center; justify-content: flex-end; white-space: nowrap; font-variant-numeric: tabular-nums; }
.chart-wrap { height: 320px; position: relative; }
.chart-wrap-tall { height: 360px; }
.chart-wrap-model { height: 420px; }
.chart-loading { width: 100%; height: 100%; }
</style>
