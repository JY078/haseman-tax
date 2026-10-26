import { defineStore } from 'pinia';
import { ref, reactive, computed } from 'vue';
import { Message } from '@arco-design/web-vue';
import request from '../utils/request';

const STORAGE_KEY = 'multiAngle_session';
const TTL_MS      = 30 * 60 * 1000; // 30 分钟

// ── Angle option configs（供 store 内部使用，也导出供组件使用）──
export const subjectAngleOptions = [
  { key: 'front',       name: '正面',    sub: 'Front'    },
  { key: 'frontRight45',name: '右前45°', sub: '3/4 Right'},
  { key: 'right',       name: '右侧',    sub: 'Right Side'},
  { key: 'top',         name: '俯视',    sub: 'Top-down' },
  { key: 'frontLeft45', name: '左前45°', sub: '3/4 Left' },
  { key: 'left',        name: '左侧',    sub: 'Left Side' },
  { key: 'back',        name: '背面',    sub: 'Back'     },
  { key: 'bottom',      name: '仰视',    sub: 'Low Angle'},
];

export const cameraAngleOptions = [
  { key: 'camFront',   name: '正面平视',   sub: 'Eye Level'  },
  { key: 'camRight45', name: '右前45°',    sub: 'Orbit Right'},
  { key: 'camRight',   name: '右侧',       sub: 'Side Right' },
  { key: 'camHigh',    name: '高角度俯拍', sub: 'High Angle' },
  { key: 'camLeft45',  name: '左前45°',    sub: 'Orbit Left' },
  { key: 'camLeft',    name: '左侧',       sub: 'Side Left'  },
  { key: 'camBehind',  name: '背后视角',   sub: 'Behind'     },
  { key: 'camLow',     name: '低角度仰拍', sub: 'Low Angle'  },
  { key: 'camBirdEye', name: '鸟瞰',       sub: "Bird's Eye" },
  { key: 'camCloseup', name: '特写镜头',   sub: 'Close-up'   },
];

export const allAngleOptions = [...subjectAngleOptions, ...cameraAngleOptions];

// ── Prompt templates ──
const angleLabels = {
  front: {
    zh: '正面',
    output: '生成结果：人物正脸完全朝向镜头，双肩左右对称，手持物品朝向镜头，物品标签文字清晰正对镜头可见。',
  },
  frontRight45: {
    zh: '右前方45°',
    output: '生成结果：人物身体轻微向右偏转约45°，右肩比左肩更靠近镜头，人物头部轻微右转，右侧脸颊更突出、左侧脸颊变窄；手持物品跟随身体右转，物品标签文字因45°偏转而从正对变为侧对镜头，标签面积明显缩小、产生透视形变。',
  },
  right: {
    zh: '右侧面',
    output: '生成结果：人物完全展示右侧面，右脸侧面轮廓、右肩、右臂侧面可见，人物正脸方向朝向画面左侧（朝左看）；手持物品也完全侧面，只能看到物品的右侧边缘，物品正面标签文字完全不可见。',
  },
  frontLeft45: {
    zh: '左前方45°',
    output: '生成结果：人物身体轻微向左偏转约45°，左肩比右肩更靠近镜头，人物头部轻微左转，左侧脸颊更突出、右侧脸颊变窄；手持物品跟随身体左转，物品标签文字因45°偏转变为侧对镜头，标签面积明显缩小、产生透视形变。（注意：此视角与右前方45°的偏转方向相反）',
  },
  left: {
    zh: '左侧面',
    output: '生成结果：人物完全展示左侧面，左脸侧面轮廓、左肩、左臂侧面可见，人物正脸方向朝向画面右侧（朝右看）；手持物品也完全侧面，只能看到物品的左侧边缘，物品正面标签文字完全不可见。（与右侧面镜像相反）',
  },
  back: {
    zh: '背面',
    output: '生成结果：只能看到人物后背、后脑勺、发型背面，完全看不到人物正脸和五官；手持物品也呈现背面，物品正面标签文字绝对不可见，只能看到物品背面。',
  },
  top: {
    zh: '俯视',
    output: '生成结果：从正上方向下俯瞰完整人物，可见人物头顶的完整发型、两侧肩膀、两条手臂、衣服顶部；画面中人物整体可见，不要只拍物品特写；物品只能看到其顶面，标签文字不可见。',
  },
  bottom: {
    zh: '仰视',
    output: '生成结果：相机在人物腰部以下位置朝上仰拍整个人物，可见人物下巴底部、颈部下方、胸部、两侧手臂及手持物品；物品从仰视角度可见其底部，标签不可见；背景呈现仰视时上方的背景色。',
  },
  camFront: {
    zh: '正面平视',
    output: '生成结果：人物正脸完全朝向镜头，双肩对称，物品标签正对镜头，整体与参考图视角一致。',
  },
  camRight45: {
    zh: '右前方45°',
    output: '生成结果：人物身体右转约45°，右肩更靠近镜头，右侧脸颊更明显；物品跟随右转，标签侧对镜头面积缩小。',
  },
  camRight: {
    zh: '右侧面',
    output: '生成结果：人物完全右侧面轮廓，正脸朝向画面左侧，物品只见右侧边缘，标签完全不可见。',
  },
  camLeft45: {
    zh: '左前方45°',
    output: '生成结果：人物身体左转约45°，左肩更靠近镜头，左侧脸颊更明显；物品跟随左转，标签侧对镜头面积缩小。（与右前方45°方向相反）',
  },
  camLeft: {
    zh: '左侧面',
    output: '生成结果：人物完全左侧面轮廓，正脸朝向画面右侧，物品只见左侧边缘，标签完全不可见。',
  },
  camBehind: {
    zh: '背面',
    output: '生成结果：只见人物背部和后脑勺，看不到正脸，物品只见背面，标签完全不可见。',
  },
  camHigh: {
    zh: '高角度俯拍',
    output: '生成结果：从高处向下俯拍整个场景，可见人物头顶及肩膀，周围环境从高角度呈现，物品顶部局部可见。',
  },
  camLow: {
    zh: '低角度仰拍',
    output: '生成结果：相机在腰部以下仰拍，人物显得高大，可见人物下方轮廓、手臂和物品下方，背景向上延伸。',
  },
  camBirdEye: {
    zh: '鸟瞰',
    output: '生成结果：从正上方90°俯瞰完整场景，可见头顶、肩膀顶部、衣物，物品只见顶面，标签不可见。',
  },
  camCloseup: {
    zh: '近景特写',
    output: '生成结果：相机靠近主体，物品和人物局部占据画面主要区域，细节清晰，物品标签文字清晰可见。',
  },
};

export const useMultiAngleStore = defineStore('multiAngle', () => {
  // ── 图片状态 ──
  const originalImage     = ref('');
  const originalImageData = ref('');
  const originalMimeType  = ref('');

  // ── 分析状态 ──
  const analysisStatus  = ref('idle'); // idle | analyzing | done | error
  const productAnalysis = ref('');

  // ── 生成状态 ──
  const generating     = ref(false);
  const mode           = ref('subject'); // subject | camera
  const selectedAngles = ref(['front', 'frontRight45', 'right', 'back']);
  const results        = reactive({});   // angleKey → { url, data, mimeType }
  const statusMap      = reactive({});   // angleKey → pending | generating | done | error

  // ── Computed ──
  const currentAngleOptions = computed(() =>
    mode.value === 'subject' ? subjectAngleOptions : cameraAngleOptions
  );
  const doneCount = computed(() => Object.keys(results).length);
  const generatedCount = computed(() =>
    Object.values(statusMap).filter(s => s === 'done' || s === 'error').length
  );
  const totalCount = computed(() => Object.keys(statusMap).length);

  // ── 图片操作 ──
  function setImage(dataUrl, data, mimeType) {
    originalImage.value     = dataUrl;
    originalImageData.value = data;
    originalMimeType.value  = mimeType;
    Object.keys(results).forEach(k => delete results[k]);
    Object.keys(statusMap).forEach(k => delete statusMap[k]);
    productAnalysis.value = '';
    analysisStatus.value  = 'idle';
    analyzeProduct();
  }

  function clearImage() {
    originalImage.value     = '';
    originalImageData.value = '';
    originalMimeType.value  = '';
    productAnalysis.value   = '';
    analysisStatus.value    = 'idle';
    Object.keys(results).forEach(k => delete results[k]);
    Object.keys(statusMap).forEach(k => delete statusMap[k]);
    localStorage.removeItem(STORAGE_KEY);
  }

  // ── 模式切换 ──
  function switchMode(newMode) {
    mode.value = newMode;
    selectedAngles.value = (newMode === 'subject' ? subjectAngleOptions : cameraAngleOptions)
      .slice(0, 4).map(a => a.key);
    Object.keys(results).forEach(k => delete results[k]);
    Object.keys(statusMap).forEach(k => delete statusMap[k]);
  }

  // ── 角度选择 ──
  function toggleAngle(key) {
    const idx = selectedAngles.value.indexOf(key);
    if (idx >= 0) selectedAngles.value.splice(idx, 1);
    else selectedAngles.value.push(key);
  }

  function selectAll() {
    selectedAngles.value = currentAngleOptions.value.map(a => a.key);
  }

  function selectNone() {
    selectedAngles.value = [];
  }

  // ── 场景分析 ──
  async function analyzeProduct() {
    if (!originalImageData.value || productAnalysis.value) return;
    analysisStatus.value = 'analyzing';
    try {
      const res = await request.post('/multi-angle/analyze', {
        image_data: originalImageData.value,
        mime_type:  originalMimeType.value,
      });
      productAnalysis.value = res.data.analysis || '';
      analysisStatus.value  = 'done';
    } catch {
      analysisStatus.value = 'error';
    }
  }

  // ── Prompt 构建 ──
  function buildPrompt(angleKey) {
    const label  = angleLabels[angleKey];
    const zh     = label?.zh ?? angleKey;
    const output = label?.output ?? `生成结果：从【${zh}】视角拍摄，人物姿势和物品数量与参考图一致。`;

    const poseAnchor = productAnalysis.value
      ? `【参考图姿势锚点，不得改变】${productAnalysis.value}`
      : `【不得改变】参考图中人物的体位、手臂动作、手持物品方式、服装外观。`;

    const outputDesc    = `【目标效果】${output}`;
    const prohibitions  =
      `【严格禁止】重复物品、断肢、浮空手臂、多余肢体、身体部位穿透；` +
      `图中物品数量必须与参考图完全一致。`;
    const style = `【保持不变】参考图的光线方向、色调、蓝色背景和整体摄影风格。`;

    return [poseAnchor, outputDesc, prohibitions, style].join('\n');
  }

  // ── 单角度生成 ──
  async function generateAngle(angleKey) {
    const prompt = buildPrompt(angleKey);
    const res = await request.post('/multi-angle/generate', {
      image_data: originalImageData.value,
      mime_type:  originalMimeType.value,
      prompt,
      mode:  mode.value,
      angle: { x: 0, y: 0, z: 0 },
    });
    if (res.data.image_data || res.data.image_url) {
      const url = res.data.image_url
        || `data:${res.data.mime_type};base64,${res.data.image_data}`;
      results[angleKey] = { url, data: res.data.image_data, mimeType: res.data.mime_type };
      return true;
    }
    return false;
  }

  // ── 批量生成（在 store 中执行，离开页面后仍继续运行）──
  async function startGenerate() {
    if (!originalImageData.value || selectedAngles.value.length === 0) return;
    generating.value = true;
    for (const key of selectedAngles.value) statusMap[key] = 'pending';

    // 等待分析完成（如果正在分析中）
    if (analysisStatus.value === 'analyzing') {
      await new Promise(resolve => {
        const id = setInterval(() => {
          if (analysisStatus.value !== 'analyzing') {
            clearInterval(id);
            resolve();
          }
        }, 200);
      });
    }

    try {
      for (const angleKey of selectedAngles.value) {
        statusMap[angleKey] = 'generating';
        try {
          const ok = await generateAngle(angleKey);
          statusMap[angleKey] = ok ? 'done' : 'error';
        } catch {
          statusMap[angleKey] = 'error';
        }
      }
      const done   = Object.values(statusMap).filter(s => s === 'done').length;
      const failed = Object.values(statusMap).filter(s => s === 'error').length;
      if (done > 0 && failed === 0) Message.success(`全部 ${done} 个角度生成成功`);
      else if (done > 0)            Message.warning(`完成 ${done} 个，失败 ${failed} 个`);
      else                          Message.error('本次未能获取到结果，请稍后重试');
      saveToStorage();
    } finally {
      generating.value = false;
    }
  }

  // ── 单角度重试 ──
  async function retryAngle(key) {
    statusMap[key] = 'generating';
    try {
      const ok = await generateAngle(key);
      statusMap[key] = ok ? 'done' : 'error';
      if (ok) {
        const name = allAngleOptions.find(a => a.key === key)?.name || key;
        Message.success(`${name} 重新生成成功`);
      }
    } catch {
      statusMap[key] = 'error';
      Message.error('重试失败');
    } finally {
      saveToStorage();
    }
  }

  // ── localStorage 持久化（30 分钟 TTL）──
  function saveToStorage() {
    try {
      // results 只保存 URL，跳过 base64 数据避免超出 5MB 限制
      const savedResults = {};
      for (const [key, val] of Object.entries(results)) {
        if (val?.url && !val.url.startsWith('data:')) {
          savedResults[key] = { url: val.url };
        }
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        timestamp:   Date.now(),
        mode:        mode.value,
        statusMap:   { ...statusMap },
        results:     savedResults,
      }));
    } catch { /* storage 写失败时静默忽略 */ }
  }

  function loadFromStorage() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const saved = JSON.parse(raw);
      if (Date.now() - saved.timestamp > TTL_MS) {
        localStorage.removeItem(STORAGE_KEY);
        return;
      }
      if (saved.mode) mode.value = saved.mode;
      const sm = saved.statusMap || {};
      Object.keys(sm).forEach(k => { statusMap[k] = sm[k]; });
      const rs = saved.results || {};
      Object.keys(rs).forEach(k => { results[k] = rs[k]; });
    } catch { /* 解析失败时静默忽略 */ }
  }

  // 初始化时尝试恢复上一次的会话（30 分钟内）
  loadFromStorage();

  return {
    // state
    originalImage, originalImageData, originalMimeType,
    analysisStatus, productAnalysis,
    generating, mode, selectedAngles, results, statusMap,
    // computed
    currentAngleOptions, doneCount, generatedCount, totalCount,
    // actions
    setImage, clearImage, switchMode,
    toggleAngle, selectAll, selectNone,
    analyzeProduct, startGenerate, retryAngle,
  };
});
