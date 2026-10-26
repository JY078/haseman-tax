import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { Message } from '@arco-design/web-vue';
import request from '../utils/request';

const STORAGE_KEY = 'productImage_session';
const TTL_MS      = 30 * 60 * 1000; // 30 分钟

export const useProductImageStore = defineStore('productImage', () => {
  // ── 提示词配置列表 ──
  const configs        = ref([]);
  const configsLoading = ref(false);
  const selectedConfigId = ref(null);

  // ── 参考图 ──
  const refImage     = ref('');
  const refImageData = ref('');
  const refMimeType  = ref('');

  // ── 自定义追加词 ──
  const customSuffix = ref('');

  // ── 分析状态 ──
  const analyzeStatus = ref('idle'); // idle | analyzing | done | error
  const analysisText  = ref('');
  /** 与后端 analyze 返回的 screens 一致，生图时需注入每屏 image_prompt */
  const analysisScreens = ref([]);
  const analyzeError  = ref('');

  // ── 生成状态 ──
  const generateStatus = ref('idle'); // idle | generating | done | error
  const generatedImage = ref('');
  const generateError  = ref('');

  // ── Computed ──
  const selectedConfig = computed(() =>
    configs.value.find(c => c.id === selectedConfigId.value) || null
  );
  const analyzing  = computed(() => analyzeStatus.value === 'analyzing');
  const generating = computed(() => generateStatus.value === 'generating');

  const canAnalyze = computed(() =>
    !!selectedConfig.value && !!refImageData.value && !analyzing.value
  );
  const canGenerate = computed(() =>
    !!selectedConfig.value && !!refImageData.value &&
    analyzeStatus.value === 'done' && !generating.value
  );

  // ── localStorage 持久化（30 分钟 TTL）──
  function saveToStorage() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        timestamp:       Date.now(),
        selectedConfigId: selectedConfigId.value,
        analyzeStatus:   analyzeStatus.value,
        analysisText:    analysisText.value,
        analyzeError:    analyzeError.value,
        generateStatus:  generateStatus.value,
        generateError:   generateError.value,
        generatedImage:  generatedImage.value?.startsWith('data:') ? '' : (generatedImage.value || ''),
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
      selectedConfigId.value = saved.selectedConfigId ?? null;
      analyzeStatus.value    = saved.analyzeStatus   || 'idle';
      analysisText.value     = saved.analysisText    || '';
      analyzeError.value     = saved.analyzeError    || '';
      generateStatus.value   = saved.generateStatus  || 'idle';
      generateError.value    = saved.generateError   || '';
      generatedImage.value   = saved.generatedImage  || '';
    } catch { /* 解析失败时静默忽略 */ }
  }

  // ── 加载配置列表 ──
  async function fetchConfigs() {
    if (configs.value.length > 0) return; // 已加载则跳过
    configsLoading.value = true;
    try {
      const res = await request.get('/product-prompts');
      configs.value = res.data || [];
    } catch {
      // 错误由拦截器统一处理
    } finally {
      configsLoading.value = false;
    }
  }

  // 强制刷新配置列表（提示词管理页面修改后调用）
  async function refreshConfigs() {
    configsLoading.value = true;
    try {
      const res = await request.get('/product-prompts');
      configs.value = res.data || [];
    } catch {
      // ignore
    } finally {
      configsLoading.value = false;
    }
  }

  // ── 切换配置 ──
  function setConfigId(id) {
    selectedConfigId.value = id;
    analysisText.value  = '';
    analysisScreens.value = [];
    analyzeStatus.value = 'idle';
    generateStatus.value = 'idle';
    generatedImage.value = '';
  }

  // ── 图片操作 ──
  function setImage(dataUrl, data, mimeType) {
    refImage.value     = dataUrl;
    refImageData.value = data;
    refMimeType.value  = mimeType;
    analysisText.value  = '';
    analysisScreens.value = [];
    analyzeStatus.value = 'idle';
    generateStatus.value = 'idle';
    generatedImage.value = '';
  }

  function clearImage() {
    refImage.value       = '';
    refImageData.value   = '';
    refMimeType.value    = '';
    analysisText.value   = '';
    analysisScreens.value = [];
    analyzeStatus.value  = 'idle';
    generateStatus.value = 'idle';
    generatedImage.value = '';
    localStorage.removeItem(STORAGE_KEY);
  }

  // ── 第一步：分析 ──
  async function runAnalyze() {
    if (!canAnalyze.value) return;
    analyzeStatus.value = 'analyzing';
    analyzeError.value  = '';
    analysisText.value  = '';
    analysisScreens.value = [];
    generateStatus.value = 'idle';
    generatedImage.value = '';

    try {
      const res = await request.post('/product-image/analyze', {
        image_data: refImageData.value,
        mime_type:  refMimeType.value,
        prompt:     selectedConfig.value.main_image_prompt,
        detail_page_prompt: selectedConfig.value.detail_page_prompt || '',
      });
      analysisText.value  = res.data.analysis || '';
      analysisScreens.value = Array.isArray(res.data.screens) ? res.data.screens : [];
      analyzeStatus.value = 'done';
    } catch (err) {
      analyzeError.value  = '本次未能获取到结果，请稍后重试';
      analyzeStatus.value = 'error';
    } finally {
      saveToStorage();
    }
  }

  // ── 第二步：生成 ──
  async function runGenerate() {
    if (!canGenerate.value) return;
    generateStatus.value = 'generating';
    generateError.value  = '';
    generatedImage.value = '';

    const parts = [selectedConfig.value.detail_page_prompt];
    const screens = analysisScreens.value;
    if (screens.length > 0) {
      const s = screens[0];
      parts.push(`\n\n【第 ${s.index} 屏｜${s.title}】`);
      parts.push(`\n【本屏排版与内容说明】\n${s.description}`);
      parts.push(`\n\n【本屏生图指令（必须严格按此画面与构图生成）】\n${s.image_prompt}`);
    } else if (analysisText.value.trim()) {
      parts.push(`\n\n【产品分析参考】\n${analysisText.value.trim()}`);
    }
    if (analysisText.value.trim() && screens.length > 0) {
      parts.push(`\n\n【产品整体概述】\n${analysisText.value.trim()}`);
    }
    if (customSuffix.value.trim()) {
      parts.push(`\n\n${customSuffix.value.trim()}`);
    }
    const finalPrompt = parts.join('');

    try {
      const res = await request.post('/image/generate', {
        prompt:         finalPrompt,
        ref_image_data: refImageData.value,
        ref_mime_type:  refMimeType.value,
      });
      const data = res.data;
      generatedImage.value = data.image_url
        || (data.image_data ? `data:${data.mime_type};base64,${data.image_data}` : '');
      if (!generatedImage.value) throw new Error('AI 未返回图片数据');
      generateStatus.value = 'done';
      Message.success('详情页图片生成成功');
    } catch (err) {
      generateError.value  = '本次未能获取到结果，请稍后重试';
      generateStatus.value = 'error';
    } finally {
      saveToStorage();
    }
  }

  // 初始化时尝试恢复上一次的会话（30 分钟内）
  loadFromStorage();

  return {
    // state
    configs, configsLoading, selectedConfigId,
    refImage, refImageData, refMimeType,
    customSuffix,
    analyzeStatus, analysisText, analysisScreens, analyzeError,
    generateStatus, generatedImage, generateError,
    // computed
    selectedConfig, analyzing, generating, canAnalyze, canGenerate,
    // actions
    fetchConfigs, refreshConfigs,
    setConfigId, setImage, clearImage,
    runAnalyze, runGenerate,
  };
});
