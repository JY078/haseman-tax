<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">多角度图片生成</h1>
      <p class="page-subtitle">上传产品或场景照片，AI 自动从不同摄影角度重新拍摄，生成一致的多角度视图</p>
    </div>

    <a-row :gutter="20">
      <!-- Left Panel -->
      <a-col :span="8">
        <!-- Upload -->
        <a-card class="panel-card">
          <template #title><icon-upload style="margin-right: 6px" />参考图片</template>
          <div v-if="!store.originalImage" class="upload-zone">
            <a-upload :auto-upload="false" accept="image/*" :show-file-list="false" @change="handleFileChange">
              <template #upload-button>
                <div class="upload-trigger">
                  <icon-plus :size="36" style="color: var(--color-text-4)" />
                  <div class="upload-text">点击或拖拽上传</div>
                  <div class="upload-hint">支持 JPG、PNG、WebP 格式</div>
                </div>
              </template>
            </a-upload>
          </div>
          <div v-else class="image-preview">
            <img :src="store.originalImage" alt="参考图" />
            <div class="preview-footer">
              <a-tag v-if="store.analysisStatus === 'analyzing'" color="blue" size="small">
                <a-spin :size="12" style="margin-right: 4px" />场景分析中...
              </a-tag>
              <a-tag v-else-if="store.analysisStatus === 'done'" color="green" size="small">
                <template #icon><icon-check-circle /></template>场景已分析
              </a-tag>
              <a-tag v-else-if="store.analysisStatus === 'error'" color="orange" size="small">跳过分析</a-tag>
              <a-button size="mini" status="danger" @click="store.clearImage">
                <template #icon><icon-delete /></template>清除
              </a-button>
            </div>
          </div>
        </a-card>

        <!-- Mode -->
        <a-card class="panel-card" style="margin-top: 12px">
          <template #title><icon-settings style="margin-right: 6px" />生成模式</template>
          <a-radio-group v-model="modeModel" type="button" size="large" style="width: 100%">
            <a-radio value="subject" style="flex: 1; text-align: center">
              <icon-common style="margin-right: 4px" />主体模式
            </a-radio>
            <a-radio value="camera" style="flex: 1; text-align: center">
              <icon-camera style="margin-right: 4px" />摄像头模式
            </a-radio>
          </a-radio-group>
          <div class="mode-hint">
            <a-alert v-if="store.mode === 'subject'" type="info" :show-icon="false">
              <span style="font-size: 12px"><b>主体模式</b>：产品在转台旋转，摄影棚不变。适合产品目录、电商详情页。</span>
            </a-alert>
            <a-alert v-else type="info" :show-icon="false">
              <span style="font-size: 12px"><b>摄像头模式</b>：摄像机绕场景移动，场景不变。适合场景展示、空间视觉。</span>
            </a-alert>
          </div>
        </a-card>

        <!-- Angle Selector -->
        <a-card class="panel-card" style="margin-top: 12px">
          <template #title><icon-compass style="margin-right: 6px" />拍摄角度</template>
          <div class="angle-grid">
            <div
              v-for="opt in store.currentAngleOptions"
              :key="opt.key"
              class="angle-item"
              :class="{ selected: store.selectedAngles.includes(opt.key), done: !!store.results[opt.key] }"
              @click="handleToggleAngle(opt.key)"
            >
              <span class="angle-name">{{ opt.name }}</span>
              <span class="angle-sub">{{ opt.sub }}</span>
              <icon-check-circle-fill v-if="store.results[opt.key]" class="done-badge" />
            </div>
          </div>
          <div class="angle-toolbar">
            <a-button size="small" type="text" @click="store.selectAll">全选</a-button>
            <a-divider direction="vertical" />
            <a-button size="small" type="text" @click="store.selectNone">清除</a-button>
            <span class="angle-count">{{ store.selectedAngles.length }} / {{ store.currentAngleOptions.length }}</span>
          </div>
        </a-card>

        <!-- Generate -->
        <a-button type="primary" long size="large" :loading="store.generating"
          :disabled="!store.originalImage || store.selectedAngles.length === 0"
          @click="store.startGenerate" style="margin-top: 12px">
          <template #icon><icon-robot /></template>
          {{ generateBtnText }}
        </a-button>
        <a-button v-if="store.doneCount > 0" type="outline" long @click="downloadAll" style="margin-top: 8px">
          <template #icon><icon-download /></template>下载全部 ({{ store.doneCount }}张)
        </a-button>
      </a-col>

      <!-- Right Panel -->
      <a-col :span="16">
        <!-- 3D Preview -->
        <a-card class="panel-card preview-card">
          <template #title>
            <icon-eye style="margin-right: 6px" />3D 预览
            <a-tag size="small" style="margin-left: 8px">{{ store.mode === 'subject' ? '主体' : '摄像头' }}</a-tag>
          </template>

          <!-- Subject: 3D Turntable -->
          <div v-if="store.mode === 'subject'" class="stage" @mousedown="onDragStart" @touchstart.prevent="onDragStart">
            <div class="turntable-scene">
              <div class="turntable-group" :style="subjectTransform">
                <img v-if="store.originalImage" :src="store.originalImage" class="stage-img" draggable="false" />
                <div v-else class="stage-placeholder"><icon-image :size="48" /></div>
                <div class="turntable-disc">
                  <div class="disc-surface"></div>
                  <div class="disc-edge"></div>
                </div>
              </div>
            </div>
            <div class="turntable-shadow"></div>
            <div class="stage-label">
              <a-tag size="small" color="gray">{{ previewLabel }}</a-tag>
            </div>
          </div>

          <!-- Camera: Orbit Sphere -->
          <div v-else class="stage" @mousedown="onDragStart" @touchstart.prevent="onDragStart">
            <div class="orbit-scene">
              <div class="orbit-sphere" :style="sphereTransform">
                <div v-for="deg in [0, 30, 60, 90, 120, 150]" :key="'v'+deg" class="sphere-ring" :style="{ transform: `rotateY(${deg}deg)` }"></div>
                <div class="sphere-ring equator"></div>
                <div class="sphere-ring lat-upper"></div>
                <div class="sphere-ring lat-lower"></div>
              </div>
              <img v-if="store.originalImage" :src="store.originalImage" class="stage-img orbit-img" draggable="false" />
              <div v-else class="stage-placeholder"><icon-image :size="48" /></div>
            </div>
            <div class="orbit-arrows">
              <span class="orbit-arrow" @click.stop="orbitStep(-45)">‹</span>
              <span class="orbit-arrow" @click.stop="orbitStep(45)">›</span>
            </div>
            <div class="stage-label">
              <a-tag size="small" color="gray">{{ previewLabel }}</a-tag>
            </div>
          </div>
        </a-card>

        <!-- Results -->
        <a-card class="panel-card results-card" style="margin-top: 12px">
          <template #title>
            <icon-image style="margin-right: 6px" />生成结果
            <a-tag v-if="store.doneCount > 0" size="small" style="margin-left: 8px">{{ store.doneCount }}张</a-tag>
          </template>
          <div v-if="displayList.length === 0" class="empty-state">
            <icon-camera :size="48" style="color: var(--color-text-4)" />
            <div class="empty-title">等待生成</div>
            <div class="empty-desc">上传参考图 → 选择模式和角度 → 点击生成按钮</div>
          </div>
          <div v-else class="results-grid">
            <div v-if="store.originalImage" class="result-card ref-card">
              <div class="result-label"><a-tag size="small" color="arcoblue">原图</a-tag></div>
              <div class="result-img"><img :src="store.originalImage" alt="原图" /></div>
            </div>
            <div v-for="item in displayList" :key="item.key" class="result-card" :class="item.status">
              <div class="result-label">
                <a-tag size="small" :color="tagColor(item.status)">{{ item.name }}</a-tag>
              </div>
              <div class="result-img">
                <template v-if="item.status === 'done'">
                  <img :src="store.results[item.key].url" :alt="item.name" />
                  <div class="result-actions">
                    <a-button size="mini" type="primary" shape="circle" @click="downloadOne(item.key)">
                      <icon-download />
                    </a-button>
                  </div>
                </template>
                <template v-else-if="item.status === 'generating'">
                  <div class="result-placeholder"><a-spin /><span>AI 生成中...</span></div>
                </template>
                <template v-else-if="item.status === 'pending'">
                  <div class="result-placeholder">
                    <icon-clock-circle :size="20" style="color: var(--color-text-4)" /><span>排队中</span>
                  </div>
                </template>
                <template v-else-if="item.status === 'error'">
                  <div class="result-placeholder error">
                    <icon-close-circle :size="20" /><span>未生成</span>
                    <a-button size="mini" type="text" @click="store.retryAngle(item.key)">重试</a-button>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import { useMultiAngleStore, allAngleOptions, subjectAngleOptions, cameraAngleOptions } from '@/stores/multiAngle';
import {
  IconPlus, IconDelete, IconUpload, IconCamera, IconRobot, IconSettings,
  IconImage, IconDownload, IconCheckCircle, IconCheckCircleFill,
  IconCloseCircle, IconClockCircle, IconCommon, IconEye, IconCompass,
} from '@arco-design/web-vue/es/icon';

const store = useMultiAngleStore();

// ── 模式双向绑定（通过 store.switchMode 切换，同步重置角度和结果）──
const modeModel = computed({
  get: () => store.mode,
  set: (val) => store.switchMode(val),
});

// ── 3D 预览状态（纯视觉，无需持久化）──
const previewX  = ref(0);
const previewY  = ref(0);
const isDragging = ref(false);
const dragStart  = reactive({ cx: 0, cy: 0, ax: 0, ay: 0 });

// ── Angle → preview rotation mapping ──
const anglePreviewMap = {
  front: { x: 0, y: 0 }, back: { x: 0, y: 180 }, left: { x: 0, y: -90 }, right: { x: 0, y: 90 },
  top: { x: 65, y: 0 }, bottom: { x: -45, y: 0 }, frontRight45: { x: 15, y: 45 }, frontLeft45: { x: 15, y: -45 },
  camFront: { x: 0, y: 0 }, camRight45: { x: 15, y: 45 }, camRight: { x: 0, y: 90 },
  camLeft45: { x: 15, y: -45 }, camLeft: { x: 0, y: -90 }, camBehind: { x: 0, y: 180 },
  camHigh: { x: 50, y: 0 }, camLow: { x: -35, y: 0 }, camBirdEye: { x: 80, y: 0 }, camCloseup: { x: 5, y: 0 },
};

// ── Computed ──
const transitionVal = computed(() => isDragging.value ? 'none' : 'transform 0.35s ease');

const subjectTransform = computed(() => ({
  transform: `rotateX(${15 + previewX.value * 0.3}deg) rotateY(${previewY.value}deg)`,
  transition: transitionVal.value,
}));

const sphereTransform = computed(() => ({
  transform: `rotateX(${previewX.value}deg) rotateY(${previewY.value}deg)`,
  transition: transitionVal.value,
}));

const previewLabel = computed(() => {
  const parts = [];
  const y = previewY.value % 360;
  const x = previewX.value;
  if (Math.abs(y) > 5) parts.push(`${y > 0 ? '右' : '左'}转${Math.abs(Math.round(y))}°`);
  if (Math.abs(x) > 5) parts.push(`${x > 0 ? '俯' : '仰'}${Math.abs(Math.round(x))}°`);
  return parts.length > 0 ? parts.join('，') : '正面视角';
});

const generateBtnText = computed(() => {
  if (store.generating) return `生成中 (${store.generatedCount}/${store.selectedAngles.length})`;
  return `AI 生成 ${store.selectedAngles.length} 个角度`;
});

const displayList = computed(() => {
  const keys = Object.keys(store.statusMap).length > 0
    ? Object.keys(store.statusMap)
    : Object.keys(store.results);
  if (keys.length === 0) return [];
  return keys.map(key => ({
    key,
    name: allAngleOptions.find(a => a.key === key)?.name || key,
    status: store.statusMap[key] || (store.results[key] ? 'done' : 'pending'),
  }));
});

// ── 文件上传 ──
function handleFileChange(_, fileItem) {
  const file = fileItem.file;
  if (!file) return;
  const reader = new FileReader();
  reader.onload = (e) => {
    const dataUrl = e.target.result;
    const data     = dataUrl.split(',')[1];
    const mimeType = dataUrl.split(';')[0].split(':')[1];
    store.setImage(dataUrl, data, mimeType);
  };
  reader.readAsDataURL(file);
}

// ── 角度切换（同步更新 3D 预览旋转）──
function handleToggleAngle(key) {
  store.toggleAngle(key);
  const mapping = anglePreviewMap[key];
  if (mapping) {
    previewX.value = mapping.x;
    previewY.value = mapping.y;
  }
}

// ── 3D 预览拖拽 ──
function getPointer(e) {
  return e.touches ? e.touches[0] : e;
}

function onDragStart(e) {
  if (!store.originalImage) return;
  isDragging.value = true;
  const p = getPointer(e);
  dragStart.cx = p.clientX;
  dragStart.cy = p.clientY;
  dragStart.ax = previewX.value;
  dragStart.ay = previewY.value;
}

function onDragMove(e) {
  if (!isDragging.value) return;
  const p = getPointer(e);
  previewY.value = dragStart.ay + (p.clientX - dragStart.cx) * 0.5;
  previewX.value = Math.max(-80, Math.min(80, dragStart.ax - (p.clientY - dragStart.cy) * 0.5));
}

function onDragEnd() {
  isDragging.value = false;
}

function orbitStep(deg) {
  previewY.value += deg;
}

onMounted(() => {
  document.addEventListener('mousemove', onDragMove);
  document.addEventListener('mouseup', onDragEnd);
  document.addEventListener('touchmove', onDragMove, { passive: true });
  document.addEventListener('touchend', onDragEnd);
});

onUnmounted(() => {
  document.removeEventListener('mousemove', onDragMove);
  document.removeEventListener('mouseup', onDragEnd);
  document.removeEventListener('touchmove', onDragMove);
  document.removeEventListener('touchend', onDragEnd);
});

// ── 下载 ──
function downloadOne(key) {
  const r = store.results[key];
  if (!r) return;
  const name = allAngleOptions.find(a => a.key === key)?.name || key;
  const link = document.createElement('a');
  link.href = r.url;
  link.download = `multi-angle-${name}-${Date.now()}.${imageExtension(r.url)}`;
  link.click();
}

function imageExtension(source) {
  const dataMime = String(source || '').match(/^data:image\/([^;,]+)/i)?.[1]?.toLowerCase();
  if (dataMime) return dataMime === 'jpeg' || dataMime === 'jpg' ? 'jpg' : dataMime;
  const path = String(source || '').split('?')[0].split('#')[0];
  return path.match(/\.(jpe?g|png|webp|gif|bmp|avif)$/i)?.[1].toLowerCase() || 'png';
}

function downloadAll() {
  Object.keys(store.results).forEach(downloadOne);
}

function tagColor(status) {
  return { done: 'green', generating: 'blue', error: 'red', pending: 'gray' }[status] || 'gray';
}
</script>

<style scoped>
.page-container {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.panel-card { border-radius: var(--radius-lg); }

/* ── Upload ── */
.upload-zone { display: flex; justify-content: center; }
.upload-trigger {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  width: 100%; min-height: 180px; border: 2px dashed var(--color-border-3);
  border-radius: 8px; cursor: pointer; transition: border-color 0.2s;
}
.upload-trigger:hover { border-color: rgb(var(--primary-6)); }
.upload-text { margin-top: 8px; color: var(--color-text-3); }
.upload-hint { margin-top: 2px; font-size: 12px; color: var(--color-text-4); }
.image-preview { text-align: center; }
.image-preview img { max-width: 100%; max-height: 200px; border-radius: 6px; border: 1px solid var(--color-border-2); }
.preview-footer { display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 10px; }
.mode-hint { margin-top: 10px; }

/* ── Angle Grid ── */
.angle-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.angle-item {
  display: flex; flex-direction: column; align-items: center; padding: 10px 4px;
  border: 2px solid var(--color-border-2); border-radius: 6px; cursor: pointer;
  transition: all 0.15s; position: relative; user-select: none;
}
.angle-item:hover { border-color: rgb(var(--primary-5)); background: var(--color-primary-light-1); }
.angle-item.selected { border-color: rgb(var(--primary-6)); background: var(--color-primary-light-1); }
.angle-item.done { border-color: rgb(var(--green-6)); }
.angle-name { font-size: 13px; font-weight: 500; color: var(--color-text-1); }
.angle-sub { font-size: 11px; color: var(--color-text-3); margin-top: 2px; }
.done-badge { position: absolute; top: -5px; right: -5px; color: rgb(var(--green-6)); font-size: 14px; background: #fff; border-radius: 50%; }
.angle-toolbar { display: flex; align-items: center; margin-top: 10px; font-size: 13px; }
.angle-count { margin-left: auto; color: var(--color-text-3); font-size: 12px; }

/* ── 3D Preview Stage ── */
.preview-card :deep(.arco-card-body) { padding: 16px; }

.stage {
  position: relative;
  height: 320px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, var(--color-fill-1) 0%, var(--color-fill-2) 100%);
  border-radius: 10px;
  cursor: grab;
  overflow: hidden;
  user-select: none;
}
.stage:active { cursor: grabbing; }

.stage-img {
  max-width: 160px;
  max-height: 160px;
  object-fit: contain;
  pointer-events: none;
}

.stage-placeholder {
  width: 160px; height: 160px;
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-4);
}

.stage-label {
  position: absolute;
  bottom: 10px;
  left: 50%;
  transform: translateX(-50%);
}

/* Subject: 3D turntable */
.turntable-scene {
  perspective: 600px;
  perspective-origin: center 60%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.turntable-group {
  transform-style: preserve-3d;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.turntable-group .stage-img {
  position: relative;
  z-index: 1;
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.15));
  margin-bottom: -50px;
}

.turntable-disc {
  position: relative;
  width: 180px;
  height: 180px;
  margin-top: 0;
  transform: rotateX(70deg) translateY(-10px);
  transform-style: preserve-3d;
}

.disc-surface {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: radial-gradient(ellipse at 45% 45%, #f8f8f8 0%, #ececec 50%, #e0e0e0 100%);
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: inset 0 0 30px rgba(0, 0, 0, 0.04);
}

.disc-edge {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  transform: translateZ(-8px);
  background: radial-gradient(ellipse at 45% 45%, #ddd 0%, #ccc 100%);
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.turntable-shadow {
  width: 160px;
  height: 20px;
  background: radial-gradient(ellipse at center, rgba(0, 0, 0, 0.12) 0%, rgba(0, 0, 0, 0.05) 50%, transparent 80%);
  border-radius: 50%;
  margin-top: -15px;
  filter: blur(3px);
}

/* Camera: orbit sphere */
.orbit-scene {
  position: relative;
  width: 260px;
  height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.orbit-sphere {
  position: absolute;
  width: 240px;
  height: 240px;
  transform-style: preserve-3d;
}

.sphere-ring {
  position: absolute;
  top: 0; left: 0;
  width: 100%; height: 100%;
  border: 1.2px solid rgba(130, 160, 200, 0.35);
  border-radius: 50%;
  pointer-events: none;
}

.sphere-ring.equator {
  transform: rotateX(90deg);
  border-color: rgba(100, 140, 200, 0.45);
  border-width: 1.5px;
}

.sphere-ring.lat-upper {
  width: 78%;
  height: 78%;
  top: 11%;
  left: 11%;
  transform: rotateX(90deg) translateZ(38px);
  border-style: dashed;
  border-color: rgba(130, 160, 200, 0.25);
}

.sphere-ring.lat-lower {
  width: 78%;
  height: 78%;
  top: 11%;
  left: 11%;
  transform: rotateX(90deg) translateZ(-38px);
  border-style: dashed;
  border-color: rgba(130, 160, 200, 0.25);
}

.orbit-img {
  position: relative;
  z-index: 1;
  filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.12));
}

.orbit-arrows {
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  transform: translateY(-50%);
  display: flex;
  justify-content: space-between;
  padding: 0 12px;
  pointer-events: none;
}

.orbit-arrow {
  width: 32px; height: 32px;
  display: flex; align-items: center; justify-content: center;
  font-size: 22px; font-weight: 300;
  color: var(--color-text-3);
  background: rgba(255,255,255,0.8);
  border: 1px solid var(--color-border-2);
  border-radius: 50%;
  cursor: pointer;
  pointer-events: all;
  transition: all 0.15s;
  user-select: none;
}
.orbit-arrow:hover {
  color: var(--color-text-1);
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

/* ── Results ── */
.results-card { min-height: 200px; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 180px; }
.empty-title { margin-top: 12px; font-size: 15px; color: var(--color-text-3); }
.empty-desc { margin-top: 4px; font-size: 13px; color: var(--color-text-4); }
.results-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.result-card { border: 1px solid var(--color-border-2); border-radius: 8px; overflow: hidden; transition: box-shadow 0.2s; }
.result-card:hover { box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08); }
.result-card.ref-card { border-color: rgb(var(--primary-5)); }
.result-label { padding: 5px 6px; background: var(--color-fill-1); }
.result-img {
  position: relative; width: 100%; aspect-ratio: 1;
  display: flex; align-items: center; justify-content: center; background: var(--color-fill-2);
}
.result-img img { width: 100%; height: 100%; object-fit: cover; }
.result-actions { position: absolute; bottom: 6px; right: 6px; opacity: 0; transition: opacity 0.2s; }
.result-card:hover .result-actions { opacity: 1; }
.result-placeholder {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 6px; height: 100%; color: var(--color-text-3); font-size: 12px;
}
.result-placeholder.error { color: rgb(var(--red-6)); }
</style>
