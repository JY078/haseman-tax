<template>
  <MainLayout>
    <section class="cutout" @dragover.prevent @drop.prevent="onDrop">
      <header class="page-head">
        <div><h1>去背景</h1><p>上传即抠图，自由换底色。</p></div>
        <button v-if="active" class="button secondary" @click="chooseFile"><ArrowUpTrayIcon />上传图片</button>
      </header>
      <input ref="fileInput" class="sr-only" type="file" accept="image/png,image/jpeg,image/webp" @change="pickFile" />
      <p v-if="error" class="notice" role="alert">{{ error }}<button aria-label="关闭提示" @click="error = ''"><XMarkIcon /></button></p>

      <div class="workspace">
        <div class="editor">
          <div class="editor-bar">
            <div class="tabs" aria-label="查看模式">
              <button :class="{ selected: view === 'result' }" :aria-pressed="view === 'result'" @click="view = 'result'">抠图效果</button>
              <button :class="{ selected: view === 'original' }" :aria-pressed="view === 'original'" :disabled="!active" @click="view = 'original'">原图</button>
            </div>
            <span class="image-info">{{ dimensions || 'PNG · JPG · WebP' }}</span>
          </div>
          <div class="stage" :class="{ checker: view === 'result' && background === 'transparent' }" :style="stageStyle">
            <div v-if="!active" class="upload-empty">
              <div class="upload-symbol"><PhotoIcon /></div>
              <h2>放进图片，留下主体</h2>
              <button class="button primary" @click="chooseFile"><ArrowUpTrayIcon />上传图片</button>
              <span>或拖拽图片到这里</span>
            </div>
            <template v-else>
              <img v-if="displayUrl" class="main-image" :class="{ faded: pending && view === 'result' }" :src="displayUrl" :alt="view === 'original' ? '原始图片' : '抠图结果'" @load="measureImage" />
              <div v-if="pending && view === 'result'" class="stage-overlay" role="status">
                <div class="processing-lines"><i></i><i></i><i></i></div><strong>{{ active.status === 'queued' ? '等待处理' : '正在抠图' }}</strong>
              </div>
              <div v-else-if="active.status === 'failed' && view === 'result'" class="stage-overlay failure" role="alert">
                <ExclamationCircleIcon /><strong>这张还没处理好</strong><p>{{ cutoutError(active.error_message) }}</p>
                <button v-if="active.inputData" class="button primary" @click="retryActive"><ArrowPathIcon />重试</button>
                <button v-else class="button primary" @click="chooseFile">重新上传</button>
              </div>
            </template>
          </div>
          <div class="editor-footer"><span>{{ active?.subject || '图片工作台' }}</span><span v-if="ready"><CheckIcon />已完成</span><span v-else-if="pending">图片依次处理</span></div>
        </div>

        <aside class="controls">
          <div class="control-content">
            <h2>背景</h2>
            <div class="background-grid">
              <button v-for="item in backgrounds" :key="item.value" class="background-choice" :class="{ active: background === item.value }" :disabled="legacy" :aria-pressed="background === item.value" @click="background = item.value; view = 'result'">
                <span class="swatch" :class="{ checker: item.value === 'transparent' }" :style="{ backgroundColor: item.value }"><CheckIcon v-if="background === item.value" :class="{ light: item.value === '#202124' }" /></span><span>{{ item.label }}</span>
              </button>
            </div>
            <label class="custom-color" :class="{ disabled: legacy }"><span>自定义颜色</span><span class="color-value">{{ customColor.toUpperCase() }}</span><input type="color" v-model="customColor" :disabled="legacy" aria-label="自定义背景颜色" @input="background = customColor; view = 'result'" /></label>
            <p v-if="legacy" class="subtle">旧版结果仅支持原背景</p>
          </div>
          <div class="export-controls">
            <div class="export-meta"><span>导出格式</span><strong>PNG <span>原尺寸</span></strong></div>
            <button class="button primary download" :disabled="!ready || exporting" @click="downloadResult"><ArrowDownTrayIcon />{{ exporting ? '准备下载…' : '下载图片' }}</button>
          </div>
        </aside>
      </div>

      <section v-if="tasks.length" class="recent">
        <div class="recent-head"><h2>最近处理</h2><span>{{ tasks.length }} 张</span><span v-if="historyError" class="history-error">{{ historyError }} <button @click="fetchTasks">刷新</button></span></div>
        <div class="filmstrip">
          <button v-for="task in tasks" :key="task.task_id" class="thumbnail" :class="{ current: activeId === task.task_id }" :aria-label="task.subject || '查看图片'" :aria-pressed="activeId === task.task_id" @click="selectTask(task)">
            <img :src="imageUrl(task.preview_image_url || task.preview || task.result_image_url)" :alt="task.subject || '图片缩略图'" />
            <span v-if="task.status !== 'done'" class="thumb-status" :class="{ failed: task.status === 'failed' }">{{ task.status === 'failed' ? '未完成' : '处理中' }}</span>
          </button>
        </div>
      </section>
    </section>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { ArrowUpTrayIcon, ArrowDownTrayIcon, ArrowPathIcon, PhotoIcon, CheckIcon, XMarkIcon, ExclamationCircleIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { downloadImage, getDisplayImageUrl, resolveImageUrl } from '@/utils/request'
import { cutoutError, paintCutout } from '@/utils/cutout'

const fileInput = ref(null), tasks = ref([]), activeId = ref(''), background = ref('#ffffff'), customColor = ref('#ded8cd')
const view = ref('result'), error = ref(''), historyError = ref(''), dimensions = ref(''), exporting = ref(false)
const active = computed(() => tasks.value.find(task => task.task_id === activeId.value))
const pending = computed(() => ['queued', 'running'].includes(active.value?.status))
const ready = computed(() => active.value?.status === 'done' && !!active.value?.result_image_url)
const legacy = computed(() => !!active.value && active.value.background !== 'transparent')
const backgrounds = [
  { label: '透明', value: 'transparent' }, { label: '白色', value: '#ffffff' }, { label: '黑色', value: '#202124' },
  { label: '浅灰', value: '#e8e9eb' }, { label: '雾蓝', value: '#dce6ed' }, { label: '柔粉', value: '#f0dedb' },
]
const imageUrl = url => url ? getDisplayImageUrl(url) : ''
const displayUrl = computed(() => imageUrl(view.value === 'original' || !ready.value ? active.value?.preview || active.value?.preview_image_url : active.value?.result_image_url))
const stageStyle = computed(() => ({ backgroundColor: view.value === 'original' || legacy.value ? '#f1f2f3' : background.value }))
let pollTimer, disposed = false, fetching = false, uploadSequence = 0

function chooseFile() { fileInput.value?.click() }
function pickFile(event) { const file = event.target.files?.[0]; event.target.value = ''; if (file) upload(file) }
function onDrop(event) { const file = event.dataTransfer?.files?.[0]; if (file) upload(file) }
function loadImage(url) { return new Promise((resolve, reject) => { const image = new Image(); image.onload = () => resolve(image); image.onerror = () => reject(new Error('图片加载失败，请重试')); image.src = url }) }
function readFile(file) { return new Promise((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result); reader.onerror = () => reject(new Error('图片读取失败')); reader.readAsDataURL(file) }) }
function measureImage(event) { const image = event.target; dimensions.value = `${image.naturalWidth} × ${image.naturalHeight}` }
function selectTask(task) { activeId.value = task.task_id; view.value = 'result'; error.value = ''; dimensions.value = ''; if (task.background !== 'transparent') background.value = '#ffffff' }

async function upload(file) {
  error.value = ''
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) { error.value = '请上传 PNG、JPG 或 WebP 图片'; return }
  if (file.size > 25 * 1024 * 1024) { error.value = '图片不能超过 25MB'; return }
  const sequence = ++uploadSequence
  try {
    const preview = await readFile(file)
    await loadImage(preview)
    if (disposed || sequence !== uploadSequence) return
    const task = { task_id: `local-${crypto.randomUUID()}`, status: 'queued', subject: file.name, preview, inputData: preview.split(',')[1], mime: file.type, background: 'transparent', created_at: new Date().toISOString() }
    tasks.value.unshift(task)
    selectTask(task)
    await createTask(task)
  } catch (e) { error.value = e.message || '上传失败，请重试' }
}

async function createTask(task) {
  try {
    const { data } = await request.post('/white-bg/tasks', { image_data: task.inputData, mime_type: task.mime, subject: task.subject, preview_image: task.preview, background: 'transparent' })
    const oldId = task.task_id
    const target = tasks.value.find(item => item.task_id === oldId)
    if (target) {
      tasks.value = tasks.value.filter(item => item === target || item.task_id !== data.task_id)
      target.task_id = data.task_id; target.status = data.status || 'queued'
    }
    if (activeId.value === oldId) activeId.value = data.task_id
    schedulePoll()
  } catch (e) {
    const target = tasks.value.find(item => item.task_id === task.task_id)
    if (target) { target.status = 'failed'; target.error_message = cutoutError(e.response?.data?.error) }
  }
}
async function retryActive() {
  const task = active.value
  if (!task?.inputData || pending.value) return
  task.status = 'queued'; task.error_message = ''; task.task_id = `local-${crypto.randomUUID()}`; activeId.value = task.task_id
  await createTask(task)
}
function schedulePoll() {
  clearTimeout(pollTimer)
  if (!disposed && tasks.value.some(task => ['queued', 'running'].includes(task.status))) pollTimer = setTimeout(fetchTasks, 2500)
}
async function fetchTasks() {
  if (fetching || disposed) return
  fetching = true
  try {
    const { data } = await request.get('/white-bg/tasks')
    if (disposed) return
    const local = new Map(tasks.value.map(task => [task.task_id, task]))
    const incoming = (data.items || []).map(task => ({ ...local.get(task.task_id), ...task }))
    const ids = new Set(incoming.map(task => task.task_id))
    tasks.value = [...tasks.value.filter(task => !ids.has(task.task_id)), ...incoming].sort((a, b) => new Date(b.created_at) - new Date(a.created_at)).slice(0, 50)
    // Release large upload buffers after completion; failed tasks retain them for retry.
    tasks.value.forEach(task => { if (task.status === 'done') delete task.inputData })
    if (!activeId.value && tasks.value.length) selectTask(tasks.value[0])
    historyError.value = ''
  } catch { historyError.value = '状态更新失败' }
  finally { fetching = false; schedulePoll() }
}

async function downloadResult() {
  if (!ready.value || exporting.value) return
  exporting.value = true; error.value = ''
  try {
    const task = active.value, color = background.value
    const filename = `${task.subject.replace(/\.[^.]+$/, '') || 'cutout'}-${color === 'transparent' ? 'transparent' : 'background'}.png`
    await downloadImage(resolveImageUrl(task.result_image_url), filename, { forceSkuPicker: true, filename }, async blob => {
      const objectUrl = URL.createObjectURL(blob)
      try {
        const image = await loadImage(objectUrl)
        const canvas = document.createElement('canvas')
        paintCutout(canvas, image, color)
        return await new Promise((resolve, reject) => canvas.toBlob(value => value ? resolve(value) : reject(new Error('导出失败')), 'image/png'))
      } finally { URL.revokeObjectURL(objectUrl) }
    })
  } catch (e) { error.value = e.response?.data?.error || e.message || '下载失败，请重试' }
  finally { exporting.value = false }
}
watch(activeId, () => { dimensions.value = '' })
onMounted(fetchTasks)
onBeforeUnmount(() => { disposed = true; clearTimeout(pollTimer) })
</script>

<style scoped>
.cutout{--ink:#242827;--muted:#666d69;--line:#e2e5e2;--surface:#fff;--canvas:#f4f5f3;max-width:1440px;margin:0 auto;padding:28px 32px 32px;color:var(--ink)}
.page-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:22px}.page-head h1{font-size:26px;font-weight:650;letter-spacing:-.8px;margin:0}.page-head p{font-size:13px;color:var(--muted);margin:6px 0 0}
button,input{font:inherit}button{cursor:pointer}button:disabled{cursor:not-allowed;opacity:.45}button:focus-visible,input:focus-visible{outline:2px solid #32765a;outline-offset:4px}svg{width:19px;height:19px;stroke-width:1.7;flex-shrink:0}.button{display:inline-flex;align-items:center;justify-content:center;gap:9px;min-height:43px;padding:0 19px;border-radius:9px;font-size:13px;font-weight:600;white-space:nowrap;transition:background .15s,transform .15s}.button:active:not(:disabled){transform:translateY(1px)}.primary{background:#263e33;color:#fff;border:1px solid #263e33}.primary:hover:not(:disabled){background:#345343}.secondary{background:var(--surface);color:var(--ink);border:1px solid var(--line)}.secondary:hover:not(:disabled){background:var(--canvas)}
.workspace{display:grid;grid-template-columns:minmax(0,1fr) 256px;background:var(--surface);border:1px solid var(--line);border-radius:16px;overflow:hidden}.editor{min-width:0}.editor-bar{height:65px;display:flex;align-items:center;justify-content:space-between;padding:0 22px;border-bottom:1px solid var(--line);gap:12px}.tabs{display:flex;padding:3px;background:var(--canvas);border-radius:8px;gap:2px}.tabs button{border:0;background:transparent;color:var(--muted);font-size:12px;padding:8px 15px;border-radius:6px}.tabs .selected{background:var(--surface);color:var(--ink);box-shadow:0 1px 4px #0000000d}.image-info{font-size:11px;color:var(--muted);font-variant-numeric:tabular-nums;white-space:nowrap}
.stage{position:relative;display:flex;align-items:center;justify-content:center;min-height:380px;height:clamp(380px,55vh,650px);overflow:hidden;transition:background-color .15s}.checker{background-image:conic-gradient(#dde0df 25%,transparent 0 50%,#dde0df 0 75%,transparent 0);background-size:20px 20px;background-color:#f9faf9}.main-image{width:100%;height:100%;object-fit:contain;padding:28px;box-sizing:border-box}.main-image.faded{opacity:.3}.upload-empty{display:flex;flex-direction:column;align-items:center;gap:14px;padding:30px;text-align:center}.upload-symbol{display:grid;place-items:center;width:68px;height:78px;border:1px solid #d5dcd7;border-radius:12px;background:#fff;transform:rotate(-7deg);margin-bottom:10px;box-shadow:8px 7px 0 #e8ece8;color:#5e7869}.upload-symbol svg{width:30px;height:30px}.upload-empty h2{font-size:23px;letter-spacing:-.6px;font-weight:550;margin:0 0 8px}.upload-empty>span{color:var(--muted);font-size:13px}.upload-empty small{color:var(--muted);font-size:11px;margin-top:12px}.editor-footer{height:45px;padding:0 22px;display:flex;align-items:center;justify-content:space-between;gap:18px;border-top:1px solid var(--line);font-size:11px;color:var(--muted)}.editor-footer>span:first-child{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.editor-footer>span:last-child{display:flex;align-items:center;gap:5px;white-space:nowrap}.editor-footer svg{width:14px;height:14px}
.controls{display:flex;flex-direction:column;border-left:1px solid var(--line);padding:24px 20px;gap:30px}.control-content h2{font-size:14px;font-weight:600;margin:0 0 20px}.background-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px 12px}.background-choice{padding:0;border:0;background:none;color:var(--muted);display:flex;flex-direction:column;align-items:center;gap:8px;font-size:11px}.swatch{width:48px;height:48px;border-radius:10px;border:1px solid #00000014;display:grid;place-items:center;box-sizing:border-box}.active .swatch{outline:2px solid #375f49;outline-offset:3px}.swatch svg{width:18px;height:18px;color:#24382b}.swatch .light{color:#fff}.background-choice.active{color:var(--ink)}.custom-color{position:relative;display:flex;align-items:center;gap:8px;margin-top:25px;padding-top:20px;border-top:1px solid var(--line);font-size:12px;cursor:pointer}.color-value{font-size:10px;font-variant-numeric:tabular-nums;color:var(--muted);margin-left:auto}.custom-color input{width:25px;height:25px;padding:0;overflow:hidden;border:0;border-radius:6px;background:transparent;cursor:pointer}.disabled{opacity:.45}.export-controls{margin-top:auto;padding-top:24px;display:flex;flex-direction:column;gap:10px}.export-meta{display:flex;justify-content:space-between;font-size:11px;color:var(--muted);margin-bottom:6px}.export-meta strong{font-size:11px;font-weight:500;color:var(--ink)}.export-meta strong span{font-weight:400;color:var(--muted);margin-left:5px}.subtle{font-size:11px;color:var(--muted);line-height:1.6}
.stage-overlay{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;flex-direction:column;gap:16px;background:#f5f6f4c9;backdrop-filter:blur(5px);padding:25px;text-align:center}.stage-overlay strong{font-size:16px;font-weight:550}.stage-overlay p{font-size:13px;color:var(--muted);max-width:360px;margin:0;line-height:1.7;overflow-wrap:anywhere}.failure>svg{width:29px;height:29px;color:#a3563f}.processing-lines{display:flex;gap:5px;align-items:center;height:28px}.processing-lines i{width:4px;height:24px;border-radius:3px;background:#496a57;animation:pulse 1s ease-in-out infinite}.processing-lines i:nth-child(2){animation-delay:.15s}.processing-lines i:nth-child(3){animation-delay:.3s}@keyframes pulse{0%,100%{transform:scaleY(.4);opacity:.4}50%{transform:scaleY(1);opacity:1}}
.notice{display:flex;align-items:center;justify-content:space-between;background:#fcf0eb;border:1px solid #edd0c3;color:#904e36;border-radius:8px;padding:10px 14px;font-size:13px;margin:0 0 16px}.notice button{border:0;background:none;color:inherit;display:flex}.recent{margin-top:25px}.recent-head{display:flex;align-items:center;gap:9px;margin-bottom:13px}.recent-head h2{font-size:13px;font-weight:550;margin:0}.recent-head>span{font-size:11px;color:var(--muted)}.history-error{margin-left:auto}.history-error button{border:0;background:none;color:var(--ink);text-decoration:underline}.filmstrip{display:flex;gap:12px;overflow-x:auto;padding:3px 3px 12px}.thumbnail{position:relative;width:82px;height:82px;flex-shrink:0;padding:4px;background:var(--surface);border:1px solid var(--line);border-radius:9px;overflow:hidden}.thumbnail.current{outline:2px solid #375f49;outline-offset:1px}.thumbnail img{width:100%;height:100%;object-fit:contain;border-radius:5px}.thumb-status{position:absolute;bottom:0;left:0;right:0;font-size:10px;color:#fff;background:#405649d9;padding:3px}.thumb-status.failed{background:#8b5848e6}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap}
@media(max-width:900px){.cutout{padding:22px 18px}.workspace{grid-template-columns:minmax(0,1fr) 225px}.controls{padding:22px 16px}.swatch{width:44px;height:44px}.stage{height:440px}}
@media(max-width:640px){.cutout{padding:20px 12px}.page-head h1{font-size:23px}.workspace{grid-template-columns:1fr}.controls{border-left:0;border-top:1px solid var(--line);padding:20px;gap:10px}.background-grid{grid-template-columns:repeat(6,1fr);gap:9px}.swatch{width:38px;height:38px}.control-content h2{margin-bottom:14px}.custom-color{margin-top:18px;padding-top:15px}.stage{min-height:320px;height:360px}.editor-bar{padding:0 12px;height:57px}.image-info{font-size:10px}.tabs button{padding:7px 10px}.main-image{padding:16px}.export-controls{padding-top:15px}.upload-empty h2{font-size:20px}.recent{margin-top:20px}}
@media(prefers-reduced-motion:reduce){*{animation:none!important;transition:none!important}}
</style>
