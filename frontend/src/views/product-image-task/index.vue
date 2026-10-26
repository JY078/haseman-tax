<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">产品图生成</h1>
      <p class="page-subtitle">选择提示词配置，上传产品参考图，分两步生成产品主图分析与详情页图片</p>
    </div>

    <a-row :gutter="20">
      <!-- 左侧操作面板 -->
      <a-col :span="9">

        <!-- 选择提示词配置 -->
        <a-card class="panel-card">
          <template #title>
            <icon-file style="margin-right: 6px" />选择提示词配置
          </template>
          <a-select
            :model-value="store.selectedConfigId"
            placeholder="请选择一个产品图提示词配置"
            :loading="store.configsLoading"
            style="width: 100%"
            @change="store.setConfigId($event)"
          >
            <a-option
              v-for="cfg in store.configs"
              :key="cfg.id"
              :value="cfg.id"
            >{{ cfg.name }}</a-option>
          </a-select>

          <!-- 配置预览 -->
          <template v-if="store.selectedConfig">
            <a-divider style="margin: 12px 0" />
            <a-collapse :bordered="false" style="background: transparent">
              <a-collapse-item key="main" header="主图分析提示词" style="padding: 0">
                <div class="config-preview">{{ store.selectedConfig.main_image_prompt }}</div>
              </a-collapse-item>
              <a-collapse-item key="detail" header="详情页提示词" style="padding: 0">
                <div class="config-preview">{{ store.selectedConfig.detail_page_prompt }}</div>
              </a-collapse-item>
            </a-collapse>
          </template>
        </a-card>

        <!-- 上传参考图 -->
        <a-card class="panel-card" style="margin-top: 12px">
          <template #title>
            <icon-upload style="margin-right: 6px" />产品参考图
          </template>
          <div v-if="!store.refImage" class="upload-zone">
            <a-upload
              :auto-upload="false"
              accept="image/*"
              :show-file-list="false"
              @change="handleFileChange"
            >
              <template #upload-button>
                <div class="upload-trigger">
                  <icon-plus :size="32" style="color: var(--color-text-4)" />
                  <div class="upload-text">点击或拖拽上传</div>
                  <div class="upload-hint">支持 JPG、PNG、WebP 格式</div>
                </div>
              </template>
            </a-upload>
          </div>
          <div v-else class="image-preview">
            <img :src="store.refImage" alt="参考图" />
            <div class="preview-footer">
              <a-button size="mini" status="danger" @click="store.clearImage">
                <template #icon><icon-delete /></template>更换图片
              </a-button>
            </div>
          </div>
        </a-card>

        <!-- 自定义追加词 -->
        <a-card class="panel-card" style="margin-top: 12px">
          <template #title>
            <icon-edit style="margin-right: 6px" />自定义追加词
            <a-tag size="small" style="margin-left: 6px">可选</a-tag>
          </template>
          <a-textarea
            v-model="store.customSuffix"
            placeholder="追加到详情页提示词末尾，例如：背景需要简洁白色，产品正面展示..."
            :auto-size="{ minRows: 3, maxRows: 6 }"
            show-word-limit
            :max-length="1000"
          />
        </a-card>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button
            type="primary"
            long
            size="large"
            :loading="store.analyzing"
            :disabled="!store.canAnalyze"
            @click="store.runAnalyze"
          >
            <template #icon><icon-robot /></template>
            {{ store.analyzing ? '分析中...' : '第一步：分析产品主图' }}
          </a-button>
          <a-button
            type="primary"
            status="success"
            long
            size="large"
            style="margin-top: 8px"
            :loading="store.generating"
            :disabled="!store.canGenerate"
            @click="store.runGenerate"
          >
            <template #icon><icon-image /></template>
            {{ store.generating ? '生成中...' : '第二步：生成详情页图片' }}
          </a-button>
        </div>

      </a-col>

      <!-- 右侧结果区 -->
      <a-col :span="15">

        <!-- 分析结果 -->
        <a-card class="panel-card">
          <template #title>
            <icon-file-exclamation style="margin-right: 6px" />第一步：产品主图分析结果
            <a-tag
              v-if="store.analyzeStatus === 'done'"
              color="green"
              size="small"
              style="margin-left: 8px"
            >
              <template #icon><icon-check-circle /></template>分析完成
            </a-tag>
            <a-tag
              v-else-if="store.analyzeStatus === 'analyzing'"
              color="blue"
              size="small"
              style="margin-left: 8px"
            >
              <a-spin :size="12" style="margin-right: 4px" />分析中...
            </a-tag>
          </template>

          <div v-if="store.analyzeStatus === 'idle'" class="empty-state">
            <icon-robot :size="40" style="color: var(--color-text-4)" />
            <div class="empty-title">等待分析</div>
            <div class="empty-desc">选择配置并上传参考图后，点击「第一步：分析产品主图」</div>
          </div>

          <div v-else-if="store.analyzeStatus === 'analyzing'" class="empty-state">
            <a-spin :size="32" />
            <div class="empty-title" style="margin-top: 12px">AI 正在分析产品主图...</div>
          </div>

          <div v-else-if="store.analyzeStatus === 'error'" class="empty-state error-state">
            <icon-close-circle :size="40" />
            <div class="empty-title">AI 暂未完成分析</div>
            <div class="empty-desc">{{ store.analyzeError }}</div>
            <a-button size="small" type="outline" style="margin-top: 8px" @click="store.runAnalyze">
              重试
            </a-button>
          </div>

          <template v-else-if="store.analyzeStatus === 'done'">
            <div class="analysis-result">{{ store.analysisText }}</div>
            <div class="analysis-hint">
              <icon-info-circle style="margin-right: 4px" />
              以上分析结果将自动注入到详情页生成提示词中，点击「第二步」开始生图
            </div>
          </template>
        </a-card>

        <!-- 生成结果 -->
        <a-card class="panel-card" style="margin-top: 12px">
          <template #title>
            <icon-image style="margin-right: 6px" />第二步：详情页图片
            <a-tag
              v-if="store.generatedImage"
              color="green"
              size="small"
              style="margin-left: 8px"
            >已生成</a-tag>
          </template>

          <div v-if="store.generateStatus === 'idle'" class="empty-state">
            <icon-image :size="40" style="color: var(--color-text-4)" />
            <div class="empty-title">等待生成</div>
            <div class="empty-desc">完成第一步分析后，点击「第二步：生成详情页图片」</div>
          </div>

          <div v-else-if="store.generateStatus === 'generating'" class="empty-state">
            <a-spin :size="32" />
            <div class="empty-title" style="margin-top: 12px">AI 正在生成详情页图片...</div>
            <div class="empty-desc">大图生成通常需要 15–60 秒，请耐心等待</div>
          </div>

          <div v-else-if="store.generateStatus === 'error'" class="empty-state error-state">
            <icon-close-circle :size="40" />
            <div class="empty-title">AI 暂未完成生成</div>
            <div class="empty-desc">{{ store.generateError }}</div>
            <a-button size="small" type="outline" style="margin-top: 8px" @click="store.runGenerate">
              重试
            </a-button>
          </div>

          <div v-else-if="store.generateStatus === 'done' && store.generatedImage" class="generate-result">
            <img :src="store.generatedImage" alt="生成结果" class="result-image" />
            <div class="result-actions">
              <a-space>
                <a-button type="primary" @click="downloadImage">
                  <template #icon><icon-download /></template>下载图片
                </a-button>
                <a-button type="outline" @click="store.runGenerate">
                  <template #icon><icon-refresh /></template>重新生成
                </a-button>
              </a-space>
            </div>
          </div>
        </a-card>

      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { onMounted } from 'vue';
import { useProductImageStore } from '@/stores/productImage';
import {
  IconPlus, IconDelete, IconUpload, IconRobot, IconImage, IconFile,
  IconEdit, IconDownload, IconRefresh, IconCheckCircle, IconCloseCircle,
  IconInfoCircle,
} from '@arco-design/web-vue/es/icon';

const store = useProductImageStore();

// ── 图片上传 ──
const handleFileChange = (_, fileItem) => {
  const file = fileItem.file;
  if (!file) return;
  const reader = new FileReader();
  reader.onload = (e) => {
    const dataUrl   = e.target.result;
    const data      = dataUrl.split(',')[1];
    const mimeType  = dataUrl.split(';')[0].split(':')[1];
    store.setImage(dataUrl, data, mimeType);
  };
  reader.readAsDataURL(file);
};

// ── 下载 ──
const downloadImage = () => {
  if (!store.generatedImage) return;
  const link = document.createElement('a');
  link.href = store.generatedImage;
  link.download = `product-detail-${Date.now()}.${imageExtension(store.generatedImage)}`;
  link.click();
};

const imageExtension = (source) => {
  const dataMime = String(source || '').match(/^data:image\/([^;,]+)/i)?.[1]?.toLowerCase();
  if (dataMime) return dataMime === 'jpeg' || dataMime === 'jpg' ? 'jpg' : dataMime;
  const path = String(source || '').split('?')[0].split('#')[0];
  return path.match(/\.(jpe?g|png|webp|gif|bmp|avif)$/i)?.[1].toLowerCase() || 'png';
};

onMounted(() => {
  store.fetchConfigs();
});
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-size: 13px;
  color: var(--text-tertiary);
  margin: 0;
}

.panel-card {
  border-radius: var(--radius-lg);
}

/* 上传区 */
.upload-zone {
  display: flex;
  justify-content: center;
}

.upload-trigger {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 160px;
  border: 2px dashed var(--color-border-3);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.2s;
}

.upload-trigger:hover {
  border-color: rgb(var(--primary-6));
}

.upload-text {
  margin-top: 8px;
  color: var(--color-text-3);
  font-size: 13px;
}

.upload-hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-text-4);
}

.image-preview {
  text-align: center;
}

.image-preview img {
  max-width: 100%;
  max-height: 200px;
  border-radius: 6px;
  border: 1px solid var(--color-border-2);
  object-fit: contain;
}

.preview-footer {
  display: flex;
  justify-content: center;
  margin-top: 10px;
}

/* 配置预览 */
.config-preview {
  font-size: 12px;
  color: var(--color-text-3);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 120px;
  overflow-y: auto;
  background: var(--color-fill-1);
  border-radius: 4px;
  padding: 8px;
}

/* 操作按钮 */
.action-buttons {
  margin-top: 12px;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  color: var(--color-text-4);
}

.empty-title {
  margin-top: 12px;
  font-size: 14px;
  color: var(--color-text-3);
  font-weight: 500;
}

.empty-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-4);
  text-align: center;
}

.error-state {
  color: rgb(var(--red-6));
}

/* 分析结果 */
.analysis-result {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-1);
  background: var(--color-fill-1);
  border-radius: 6px;
  padding: 12px 16px;
  max-height: 320px;
  overflow-y: auto;
}

.analysis-hint {
  margin-top: 10px;
  font-size: 12px;
  color: var(--color-text-4);
  display: flex;
  align-items: center;
}

/* 生成结果 */
.generate-result {
  text-align: center;
}

.result-image {
  max-width: 100%;
  border-radius: 8px;
  border: 1px solid var(--color-border-2);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.result-actions {
  margin-top: 14px;
  display: flex;
  justify-content: center;
}
</style>
