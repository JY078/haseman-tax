<template>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">模型管理</h1>
      <a-space>
        <a-select 
          v-model="filterType" 
          placeholder="筛选类型" 
          style="width: 160px" 
          allow-clear
          @change="fetchModels"
        >
          <a-option value="chat">对话</a-option>
          <a-option value="drawing">绘画</a-option>
          <a-option value="video">视频</a-option>
        </a-select>
        <a-button type="primary" @click="openCreateModal">
          <template #icon><icon-plus /></template>
          新建模型
        </a-button>
      </a-space>
    </div>

    <a-card>

      <a-table 
        :data="models" 
        :loading="loading" 
        :pagination="false"
        style="margin-top: 16px"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="80" />
          <a-table-column title="模型名称" data-index="name" />
          <a-table-column title="模型类型" data-index="types">
            <template #cell="{ record }">
              <a-space wrap>
                <a-tag v-for="t in record.types" :key="t" :color="getTypeColor(t)">
                  {{ getTypeLabel(t) }}
                </a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="支持分辨率" data-index="supported_resolutions" :width="220">
            <template #cell="{ record }">
              <template v-if="(record.types || []).includes('drawing')">
                <template v-if="record.supported_resolutions && record.supported_resolutions.length > 0">
                  <a-space wrap>
                    <a-tag
                      v-for="r in record.supported_resolutions"
                      :key="r"
                      color="purple"
                      size="small"
                    >{{ r }}</a-tag>
                  </a-space>
                </template>
                <span v-else style="color: #999; font-size: 12px;">不限制</span>
              </template>
              <span v-else style="color: #ccc;">—</span>
            </template>
          </a-table-column>

          <a-table-column title="状态" data-index="is_active">
            <template #cell="{ record }">
              <a-switch 
                v-model="record.is_active" 
                :loading="record.updating"
                @change="(val) => handleStatusChange(record, val)"
              />
            </template>
          </a-table-column>
          <a-table-column title="默认模型" data-index="is_default">
            <template #cell="{ record }">
              <a-switch 
                v-model="record.is_default" 
                :loading="record.updatingDefault"
                @change="(val) => handleDefaultChange(record, val)"
              />
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="created_at">
            <template #cell="{ record }">
              {{ new Date(record.created_at).toLocaleString() }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="200">
            <template #cell="{ record }">
              <a-space>
                <!-- 仅对话和绘画类型显示测试按钮 -->
                <a-button
                  v-if="hasTestableType(record)"
                  type="text"
                  size="small"
                  status="success"
                  @click="openTestModal(record)"
                >
                  <template #icon><icon-experiment /></template>
                  测试
                </a-button>
                <a-button type="text" size="small" @click="openEditModal(record)">
                  编辑
                </a-button>
                <a-popconfirm content="确定要删除该模型吗？" @ok="handleDelete(record.id)">
                  <a-button type="text" status="danger" size="small">
                    删除
                  </a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 新建/编辑弹窗 -->
    <a-modal 
      v-model:visible="modalVisible" 
      :title="isEdit ? '编辑模型' : '新建模型'" 
      @ok="handleSubmit" 
      @cancel="handleCancel"
      width="560px"
    >
      <a-form :model="form" ref="formRef" layout="vertical">
        <a-form-item field="name" label="模型名称" :rules="[{ required: true, message: '请输入模型名称' }]">
          <a-input v-model="form.name" placeholder="请输入模型名称" />
        </a-form-item>
        <a-form-item field="types" label="模型类型" :rules="[{ required: true, type: 'array', minLength: 1, message: '请选择至少一个模型类型' }]">
          <a-select v-model="form.types" placeholder="请选择模型类型" multiple allow-create>
            <a-option value="chat">对话模型 (Chat)</a-option>
            <a-option value="drawing">绘画模型 (Drawing)</a-option>
            <a-option value="video">视频模型 (Video)</a-option>
          </a-select>
        </a-form-item>

        <!-- 绘画模型专属：支持的分辨率配置 -->
        <a-form-item
          v-if="isDrawingModel"
          field="supported_resolutions"
          label="支持的分辨率"
        >
          <div class="resolution-config">
            <a-input-tag
              v-model="form.supported_resolutions"
              placeholder="手动填写，回车确认"
              style="width: 100%"
            />
            <div class="resolution-presets">
              <span class="preset-hint">快速添加：</span>
              <a-tag
                v-for="r in presetResolutions"
                :key="r.value"
                :checkable="true"
                :checked="form.supported_resolutions.includes(r.value)"
                @check="(checked) => toggleResolution(r.value, checked)"
                size="small"
                style="cursor: pointer"
              >{{ r.value }}</a-tag>
            </div>
            <div style="color: #999; font-size: 12px; margin-top: 4px;">
              留空表示不限制分辨率。
            </div>
          </div>
        </a-form-item>

        <a-form-item field="is_active" label="状态">
          <a-switch v-model="form.is_active" />
        </a-form-item>
        <a-form-item field="is_default" label="默认模型">
          <a-switch v-model="form.is_default" />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">开启后，该模型将成为对应类型的默认模型</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 测试弹窗 -->
    <a-modal
      v-model:visible="testModalVisible"
      :title="`测试模型：${testModel?.name}`"
      :footer="false"
      width="640px"
      @cancel="closeTestModal"
    >
      <template v-if="testModel">
        <!-- 类型切换（如果模型同时支持对话和绘画） -->
        <a-radio-group
          v-if="hasChat(testModel) && hasDrawing(testModel)"
          v-model="testType"
          type="button"
          style="margin-bottom: 16px"
        >
          <a-radio value="chat">对话测试</a-radio>
          <a-radio value="drawing">绘画测试</a-radio>
        </a-radio-group>

        <!-- 对话测试 -->
        <template v-if="testType === 'chat'">
          <a-textarea
            v-model="chatPrompt"
            placeholder="请输入对话内容..."
            :auto-size="{ minRows: 3, maxRows: 6 }"
            style="margin-bottom: 12px"
          />
          <a-button
            type="primary"
            :loading="testLoading"
            @click="runChatTest"
            long
          >
            <template #icon><icon-send /></template>
            发送
          </a-button>
          <div v-if="testResult" class="test-result">
            <a-divider>回复</a-divider>
            <div class="result-content">{{ testResult }}</div>
            <div v-if="testUsage" class="result-usage">
              Token 用量：↑ {{ testUsage.prompt_tokens }} ↓ {{ testUsage.completion_tokens }}，共 {{ testUsage.total_tokens }}
            </div>
          </div>
          <a-result
            v-if="testError"
            status="error"
            :title="testError"
            style="padding: 12px 0"
          />
        </template>

        <!-- 绘画测试 -->
        <template v-if="testType === 'drawing'">
          <a-textarea
            v-model="drawPrompt"
            placeholder="请输入图片描述..."
            :auto-size="{ minRows: 3, maxRows: 6 }"
            style="margin-bottom: 8px"
          />
          <a-select
            v-model="drawAspectRatio"
            style="width: 160px; margin-bottom: 12px"
            placeholder="宽高比"
          >
            <a-option value="1:1">1:1（方形）</a-option>
            <a-option value="16:9">16:9（横屏）</a-option>
            <a-option value="9:16">9:16（竖屏）</a-option>
            <a-option value="4:3">4:3</a-option>
          </a-select>
          <a-button
            type="primary"
            :loading="testLoading"
            @click="runDrawTest"
            long
          >
            <template #icon><icon-image /></template>
            生成图片
          </a-button>
          <div v-if="drawResult" class="test-result">
                <a-divider>生成结果</a-divider>
                <img
                  :src="drawResult.image_url || `data:${drawResult.mime_type};base64,${drawResult.image_data}`"
                  style="max-width: 100%; border-radius: 6px"
                  alt="生成图片"
                />
                <div v-if="drawResult.text" class="result-content" style="margin-top: 8px">{{ drawResult.text }}</div>
              </div>
          <a-result
            v-if="testError"
            status="error"
            :title="testError"
            style="padding: 12px 0"
          />
        </template>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import request from '@/utils/request';
import { IconPlus, IconExperiment, IconSend, IconImage } from '@arco-design/web-vue/es/icon';

const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';
const models = ref([]);
const loading = ref(false);
const filterType = ref('');

const modalVisible = ref(false);
const isEdit = ref(false);
const formRef = ref(null);
const form = reactive({
  id: null,
  name: '',
  types: [],
  supported_resolutions: [],
  is_active: true,
  is_default: false,
});

/** 当前表单中是否包含绘画类型 */
const isDrawingModel = computed(() => (form.types || []).includes('drawing'));

/** 预设分辨率选项（常见图片生成规格） */
const presetResolutions = [
  { value: '512' },
  { value: '1K' },
  { value: '2K' },
  { value: '4K' },
];

/** 切换预设分辨率（快速添加/移除） */
const toggleResolution = (value, checked) => {
  const list = form.supported_resolutions;
  if (checked && !list.includes(value)) {
    list.push(value);
  } else if (!checked) {
    const idx = list.indexOf(value);
    if (idx !== -1) list.splice(idx, 1);
  }
};

// 测试弹窗状态
const testModalVisible = ref(false);
const testModel = ref(null);
const testType = ref('chat');
const testLoading = ref(false);
const testResult = ref('');
const testUsage = ref(null);
const testError = ref('');
const chatPrompt = ref('你好，请简单介绍一下你自己。');
const drawPrompt = ref('');
const drawAspectRatio = ref('1:1');
const drawResult = ref(null);

const getTypeColor = (type) => {
  switch (type) {
    case 'chat': return 'arcoblue';
    case 'drawing': return 'purple';
    case 'video': return 'orange';
    default: return 'gray';
  }
};

const getTypeLabel = (type) => {
  switch (type) {
    case 'chat': return '对话';
    case 'drawing': return '绘画';
    case 'video': return '视频';
    default: return type;
  }
};

const hasChat = (record) => (record.types || []).includes('chat');
const hasDrawing = (record) => (record.types || []).includes('drawing');
const hasTestableType = (record) => hasChat(record) || hasDrawing(record);

const fetchModels = async () => {
  loading.value = true;
  try {
    const params = {};
    if (filterType.value) {
      params.type = filterType.value;
    }
    const res = await request.get('/models', { params });
    models.value = res.data;
  } catch (error) {
    // Error handled by interceptor
  } finally {
    loading.value = false;
  }
};

const openCreateModal = () => {
  isEdit.value = false;
  form.id = null;
  form.name = '';
  form.types = [];
  form.supported_resolutions = [];
  form.is_active = true;
  form.is_default = false;
  modalVisible.value = true;
};

const openEditModal = (record) => {
  isEdit.value = true;
  form.id = record.id;
  form.name = record.name;
  form.types = [...(record.types || [])];
  form.supported_resolutions = [...(record.supported_resolutions || [])];
  form.is_active = record.is_active;
  form.is_default = record.is_default;
  modalVisible.value = true;
};

const handleCancel = () => {
  modalVisible.value = false;
};

const handleSubmit = async () => {
  const err = await formRef.value.validate();
  if (err) return;

  try {
    if (isEdit.value) {
      await request.put(`/models/${form.id}`, form);
      Message.success('更新成功');
    } else {
      await request.post('/models', form);
      Message.success('创建成功');
    }
    modalVisible.value = false;
    fetchModels();
  } catch (error) {
    // Error handled
  }
};

const handleStatusChange = async (record, val) => {
  record.updating = true;
  try {
    await request.put(`/models/${record.id}`, { is_active: val });
    Message.success('状态更新成功');
    await fetchModels();
  } catch (error) {
    record.is_active = !val;
  } finally {
    record.updating = false;
  }
};

const handleDefaultChange = async (record, val) => {
  record.updatingDefault = true;
  try {
    await request.put(`/models/${record.id}`, { is_default: val });
    Message.success('默认模型更新成功');
    await fetchModels(); // await 确保 UI 与服务器状态严格同步
  } catch (error) {
    record.is_default = !val;
  } finally {
    record.updatingDefault = false;
  }
};

const handleDelete = async (id) => {
  try {
    await request.delete(`/models/${id}`);
    Message.success('删除成功');
    fetchModels();
  } catch (error) {
    // Error handled
  }
};

const openTestModal = (record) => {
  testModel.value = record;
  // 默认选择该模型支持的第一个可测类型
  testType.value = hasChat(record) ? 'chat' : 'drawing';
  // 重置结果
  testResult.value = '';
  testUsage.value = null;
  testError.value = '';
  drawResult.value = null;
  chatPrompt.value = '你好，请简单介绍一下你自己。';
  drawPrompt.value = '';
  testModalVisible.value = true;
};

const closeTestModal = () => {
  testModalVisible.value = false;
  testModel.value = null;
};

const runChatTest = async () => {
  if (!chatPrompt.value.trim()) {
    Message.warning('请输入对话内容');
    return;
  }
  testLoading.value = true;
  testResult.value = '';
  testUsage.value = null;
  testError.value = '';

  try {
    const token = localStorage.getItem('token');
    const response = await fetch(apiBase + '/chat', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({
        model: testModel.value.name,
        messages: [{ role: 'user', content: chatPrompt.value }],
        stream: true,
      }),
    });

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}));
      testError.value = errData.error || `请求失败 (${response.status})，请检查 API Key 配置`;
      return;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop(); // 保留末尾不完整的行

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        const data = line.slice(6).trim();
        if (data === '[DONE]') continue;
        try {
          const chunk = JSON.parse(data);
          const delta = chunk.choices?.[0]?.delta?.content ?? '';
          if (delta) testResult.value += delta;
        } catch {
          // 忽略非 JSON 行
        }
      }
    }
  } catch (err) {
    testError.value = err?.message || '请求失败，请检查 API Key 配置';
  } finally {
    testLoading.value = false;
  }
};

const runDrawTest = async () => {
  if (!drawPrompt.value.trim()) {
    Message.warning('请输入图片描述');
    return;
  }
  testLoading.value = true;
  drawResult.value = null;
  testError.value = '';
  try {
    const res = await request.post('/image/generate', {
      model: testModel.value.name,
      prompt: drawPrompt.value,
      aspect_ratio: drawAspectRatio.value,
    });
    drawResult.value = res.data;
  } catch (err) {
    testError.value = err?.response?.data?.error || '生成失败，请检查绘图 API Key 配置';
  } finally {
    testLoading.value = false;
  }
};

onMounted(() => {
  fetchModels();
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

.test-result {
  margin-top: 12px;
}

.result-content {
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  padding: 12px;
  font-size: 14px;
  line-height: 1.6;
  max-height: 300px;
  overflow-y: auto;
}

.result-usage {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-tertiary);
  text-align: right;
}

.resolution-config {
  width: 100%;
}

.resolution-presets {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.preset-hint {
  font-size: 12px;
  color: #999;
  flex-shrink: 0;
}
</style>
