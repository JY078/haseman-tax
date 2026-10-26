<template>
  <MainLayout>
    <div class="requests-page">
      <section v-if="formOpen" class="request-editor-page">
        <div class="request-editor-canvas">
          <WorkRequestForm layout="page" :initial-value="formInitialValue" :submitting="submitting" @submit="saveRequest" @cancel="closeForm" />
        </div>
      </section>

      <div v-else class="requests-shell">
      <header class="page-heading">
        <div>
          <div class="heading-kicker"><ClipboardDocumentListIcon class="h-4 w-4" />运营工具</div>
          <h1>视频需求</h1>
          <p>统一提交、分配和跟踪视频制作需求</p>
        </div>
        <div class="heading-actions">
          <button type="button" class="quiet-action" title="刷新需求列表" @click="fetchRequests"><ArrowPathIcon class="h-4 w-4" />刷新</button>
          <button type="button" class="primary-action" @click="openCreate"><PlusIcon class="h-4 w-4" />新建视频需求</button>
        </div>
      </header>

      <section class="filter-toolbar" aria-label="需求筛选">
        <div class="search-field">
          <MagnifyingGlassIcon class="h-4 w-4" />
          <input v-model="filters.keyword" type="search" placeholder="搜索工作编号或标题" @keyup.enter="applyFilters" />
        </div>
        <select v-model="filters.status" aria-label="需求状态" @change="applyFilters">
          <option value="">全部状态</option>
          <option v-for="item in statusOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
        </select>
        <select v-model="filters.creator_id" aria-label="创建人" @change="applyFilters">
          <option value="">全部创建人</option><option v-for="user in users" :key="user.id" :value="user.id">{{ userName(user) }}</option>
        </select>
        <select v-model="filters.assignee_id" aria-label="处理人" @change="applyFilters">
          <option value="">全部处理人</option><option v-for="user in users" :key="user.id" :value="user.id">{{ userName(user) }}</option>
        </select>
        <button type="button" class="filter-button" title="查询" @click="applyFilters"><MagnifyingGlassIcon class="h-4 w-4" />查询</button>
        <button v-if="hasFilters" type="button" class="reset-button" @click="resetFilters"><ArrowPathIcon class="h-4 w-4" />重置</button>
      </section>

      <section class="list-panel">
        <div class="list-meta">
          <div class="list-meta-title"><span>工作队列</span><strong>{{ total }}</strong><small>条视频需求</small></div>
          <span class="list-note">按创建时间倒序排列</span>
        </div>

        <div v-if="loading" class="loading-state" aria-live="polite">
          <div class="table-skeleton" aria-hidden="true"><i v-for="index in 6" :key="index" /></div>
          <span>正在加载需求</span>
        </div>
        <div v-else-if="loadError" class="empty-state error-state">
          <div class="empty-icon"><ExclamationTriangleIcon class="h-7 w-7" /></div>
          <strong>需求列表暂时不可用</strong>
          <p>{{ loadError }}</p>
          <button type="button" @click="fetchRequests"><ArrowPathIcon class="h-4 w-4" />重新加载</button>
        </div>
        <div v-else-if="!requests.length" class="empty-state">
          <div class="empty-icon"><ClipboardDocumentListIcon class="h-8 w-8" /></div>
          <strong>{{ hasFilters ? '没有符合条件的需求' : '还没有需求' }}</strong>
          <p>{{ hasFilters ? '调整筛选条件后再试' : '提交第一条视频制作需求' }}</p>
          <button v-if="!hasFilters" type="button" @click="openCreate"><PlusIcon class="h-4 w-4" />新建视频需求</button>
        </div>

        <template v-else>
          <div class="table-scroll">
            <table class="request-table">
              <thead><tr><th>工作编号</th><th class="title-column">标题</th><th>状态</th><th>处理人</th><th>创建时间</th><th>创建人</th><th class="progress-column">进度</th></tr></thead>
              <tbody>
                <tr v-for="requestItem in requests" :key="requestItem.id">
                  <td><span class="work-no">{{ requestItem.work_no }}</span></td>
                  <td><button type="button" class="title-link" @click="openDetail(requestItem)">{{ requestItem.title }}</button></td>
                  <td><span :class="['status-badge', `status-${requestItem.status}`]"><i />{{ statusMeta(requestItem.status).label }}</span></td>
                  <td><span v-if="requestItem.assignee" class="person-cell"><Avatar :user="requestItem.assignee" />{{ userName(requestItem.assignee) }}</span><span v-else class="muted">待分配</span></td>
                  <td><span class="date-cell">{{ formatDate(requestItem.created_at) }}</span></td>
                  <td><span class="person-cell"><Avatar :user="requestItem.creator" />{{ userName(requestItem.creator) }}</span></td>
                  <td><div class="progress-cell"><div class="progress-track"><span :style="{ width: `${requestItem.progress}%` }" /></div><strong>{{ requestItem.progress }}%</strong></div></td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="mobile-request-list">
            <button v-for="requestItem in requests" :key="requestItem.id" type="button" class="mobile-request" @click="openDetail(requestItem)">
              <div class="mobile-request-top"><span class="work-no">{{ requestItem.work_no }}</span><span :class="['status-badge', `status-${requestItem.status}`]"><i />{{ statusMeta(requestItem.status).label }}</span></div>
              <strong>{{ requestItem.title }}</strong>
              <div class="mobile-request-meta"><span>{{ requestItem.assignee ? userName(requestItem.assignee) : '待分配' }}</span><span>{{ formatDate(requestItem.created_at, true) }}</span></div>
              <div class="progress-cell"><div class="progress-track"><span :style="{ width: `${requestItem.progress}%` }" /></div><strong>{{ requestItem.progress }}%</strong></div>
            </button>
          </div>
        </template>

        <footer v-if="totalPages > 1" class="pagination">
          <span>第 {{ page }} / {{ totalPages }} 页</span>
          <div><button type="button" :disabled="page === 1" title="上一页" @click="changePage(page - 1)"><ChevronLeftIcon class="h-4 w-4" /></button><button type="button" :disabled="page === totalPages" title="下一页" @click="changePage(page + 1)"><ChevronRightIcon class="h-4 w-4" /></button></div>
        </footer>
      </section>
      </div>

      <WorkRequestDrawer
      v-model:open="detailOpen"
      :request="currentRequest"
      :users="users"
      :current-user-id="currentUserId"
      :is-admin="isAdmin"
      :action-loading="actionLoading"
      @edit="openEdit"
      @copy="openCopy"
      @action="openAction"
      @refresh="refreshDetail"
      />

      <Modal v-model:open="actionModalOpen" :title="actionModalTitle" size="md" :dismissible="!actionLoading">
      <div v-if="actionType === 'assign'" class="action-form"><label>处理人<span>*</span></label><select v-model="actionForm.assignee_id"><option value="">请选择处理人</option><option v-for="user in users" :key="user.id" :value="user.id">{{ userName(user) }}</option></select></div>
      <div v-else-if="actionType === 'progress'" class="action-form"><label>当前进度：{{ actionForm.progress }}%</label><input v-model.number="actionForm.progress" type="range" min="10" max="89" /><div class="range-labels"><span>10%</span><span>89%</span></div></div>
      <div v-else-if="actionType === 'submit-review'" class="action-form"><label>交付说明<span>*</span></label><textarea v-model.trim="actionForm.delivery_note" rows="4" placeholder="说明本次交付包含的内容" /><label>共享盘文件路径<span>*</span></label><div v-for="(_, index) in actionForm.delivery_paths" :key="index" class="path-row"><input v-model.trim="actionForm.delivery_paths[index]" type="text" placeholder="例如：\\共享盘\设计交付\REQ-..." /><button type="button" title="删除路径" :disabled="actionForm.delivery_paths.length === 1" @click="actionForm.delivery_paths.splice(index, 1)"><TrashIcon class="h-4 w-4" /></button></div><button type="button" class="add-path" @click="actionForm.delivery_paths.push('')"><PlusIcon class="h-4 w-4" />添加文件路径</button></div>
      <div v-else-if="actionType === 'reject'" class="action-form"><label>退回原因<span>*</span></label><textarea v-model.trim="actionForm.reason" rows="4" placeholder="说明需要修改的内容" /></div>
      <div v-else class="confirm-copy"><ExclamationTriangleIcon class="h-9 w-9" /><div><strong>{{ confirmContent.title }}</strong><p>{{ confirmContent.description }}</p></div></div>
      <template #footer><button type="button" class="modal-cancel" :disabled="actionLoading" @click="actionModalOpen = false">取消</button><button type="button" :class="['modal-confirm', { danger: ['cancel', 'delete', 'reject', 'return'].includes(actionType) }]" :disabled="actionLoading" @click="submitAction"><ArrowPathIcon v-if="actionLoading" class="h-4 w-4 animate-spin" />{{ actionLoading ? '处理中' : actionConfirmText }}</button></template>
      </Modal>
    </div>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ArrowPathIcon, ChevronLeftIcon, ChevronRightIcon, ClipboardDocumentListIcon, ExclamationTriangleIcon, MagnifyingGlassIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import request from '@/utils/request'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import Modal from '@/components/ui/Modal.vue'
import WorkRequestForm from '@/components/work-requests/WorkRequestForm.vue'
import WorkRequestDrawer from '@/components/work-requests/WorkRequestDrawer.vue'
import Avatar from '@/components/work-requests/WorkRequestAvatar.vue'
import MainLayout from '@/components/layout/MainLayout.vue'

const authStore = useAuthStore()
const toast = useToast()
const requests = ref([]), users = ref([]), total = ref(0), page = ref(1), loading = ref(false), loadError = ref(''), submitting = ref(false), actionLoading = ref(false)
const pageSize = 20
const filters = reactive({ keyword: '', status: '', creator_id: '', assignee_id: '' })
const formOpen = ref(false), editingRequest = ref(null), copyingRequest = ref(null)
const detailOpen = ref(false), currentRequest = ref(null)
const actionModalOpen = ref(false), actionType = ref(''), actionForm = reactive({ assignee_id: '', progress: 10, delivery_note: '', delivery_paths: [''], reason: '' })
const statusOptions = [{ value: 'pending', label: '待受理' }, { value: 'in_progress', label: '进行中' }, { value: 'pending_review', label: '待验收' }, { value: 'completed', label: '已完成' }, { value: 'canceled', label: '已取消' }]

const currentUserId = computed(() => Number(authStore.user?.id || 0))
const isAdmin = computed(() => (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin'))
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const hasFilters = computed(() => Object.values(filters).some(Boolean))
const formInitialValue = computed(() => editingRequest.value || copyingRequest.value)
const actionModalTitle = computed(() => ({ assign: '分配处理人', progress: '更新任务进度', 'submit-review': '提交验收', approve: '确认验收通过', reject: '退回修改', return: '退回需求修改', cancel: '取消需求', delete: '删除需求' })[actionType.value] || '确认操作')
const actionConfirmText = computed(() => ({ assign: '确认分配', progress: '保存进度', 'submit-review': '提交验收', approve: '验收通过', reject: '确认退回', return: '确认退回', cancel: '确认取消', delete: '确认删除' })[actionType.value] || '确认')
const confirmContent = computed(() => ({ approve: { title: '确认该需求已达到交付要求？', description: '通过后任务将完成，进度更新为 100%。' }, return: { title: '退回给创建人修改需求？', description: '任务将恢复为待受理并清空当前处理人。' }, cancel: { title: '确认取消这条需求？', description: '取消后需求将停止流转，但仍保留在列表中。' }, delete: { title: '永久删除这条需求？', description: '此操作不可恢复，请确认不再需要这条记录。' } })[actionType.value] || { title: '', description: '' })

function userName(user) { return user?.name || user?.username || `用户 #${user?.id || ''}` }
function statusMeta(status) { return statusOptions.find(item => item.value === status) || { label: status } }
function formatDate(value, compact = false) { if (!value) return '—'; const date = new Date(value); return date.toLocaleString('zh-CN', compact ? { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' } : { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replaceAll('/', '-') }

async function fetchRequests() {
  loading.value = true
  loadError.value = ''
  try { const params = { page: page.value, page_size: pageSize }; Object.entries(filters).forEach(([key, value]) => { if (value) params[key] = value }); const response = await request.get('/work-requests', { params }); requests.value = response.data.data || []; total.value = response.data.total || 0 } catch (error) { loadError.value = error?.response?.data?.error || '请检查网络连接后重试'; toast.error(error?.response?.data?.error || '获取需求列表失败') } finally { loading.value = false }
}
async function fetchUsers() { try { users.value = (await request.get('/work-requests/users')).data || [] } catch { users.value = [] } }
function applyFilters() { page.value = 1; fetchRequests() }
function resetFilters() { Object.assign(filters, { keyword: '', status: '', creator_id: '', assignee_id: '' }); applyFilters() }
function changePage(value) { page.value = value; fetchRequests(); window.scrollTo({ top: 0, behavior: 'smooth' }) }
function openCreate() { editingRequest.value = null; copyingRequest.value = null; formOpen.value = true }
function openEdit(requestItem) { editingRequest.value = JSON.parse(JSON.stringify(requestItem)); copyingRequest.value = null; detailOpen.value = false; formOpen.value = true }
function openCopy(requestItem) { copyingRequest.value = JSON.parse(JSON.stringify(requestItem)); editingRequest.value = null; detailOpen.value = false; formOpen.value = true }
function closeForm() { if (submitting.value) return; formOpen.value = false }
async function saveRequest(payload) { submitting.value = true; try { if (editingRequest.value) await request.put(`/work-requests/${editingRequest.value.id}`, payload); else await request.post('/work-requests', payload); toast.success(editingRequest.value ? '需求已保存' : '需求已提交'); formOpen.value = false; page.value = 1; await fetchRequests() } catch (error) { toast.error(error?.response?.data?.error || '提交需求失败') } finally { submitting.value = false } }
async function openDetail(requestItem) { detailOpen.value = true; currentRequest.value = requestItem; await refreshDetail(requestItem.id) }
async function refreshDetail(id = currentRequest.value?.id) { if (!id) return; try { currentRequest.value = (await request.get(`/work-requests/${id}`)).data } catch (error) { toast.error(error?.response?.data?.error || '获取需求详情失败') } }

function openAction(type) {
  actionType.value = type
  Object.assign(actionForm, { assignee_id: currentRequest.value?.assignee_id || '', progress: currentRequest.value?.progress || 10, delivery_note: currentRequest.value?.delivery_note || '', delivery_paths: currentRequest.value?.delivery_paths?.length ? [...currentRequest.value.delivery_paths] : [''], reason: '' })
  actionModalOpen.value = true
}

async function submitAction() {
  if (!currentRequest.value) return
  let method = 'post', endpoint = `/work-requests/${currentRequest.value.id}/${actionType.value}`, payload = {}
  if (actionType.value === 'assign') { if (!actionForm.assignee_id) return toast.error('请选择处理人'); payload = { assignee_id: Number(actionForm.assignee_id) } }
  else if (actionType.value === 'progress') payload = { progress: Number(actionForm.progress) }
  else if (actionType.value === 'submit-review') { const paths = actionForm.delivery_paths.map(item => item.trim()).filter(Boolean); if (!actionForm.delivery_note || !paths.length) return toast.error('请填写交付说明和至少一个共享盘文件路径'); payload = { delivery_note: actionForm.delivery_note, delivery_paths: paths } }
  else if (actionType.value === 'reject') { if (!actionForm.reason) return toast.error('请填写退回原因'); payload = { reason: actionForm.reason } }
  else if (actionType.value === 'delete') { method = 'delete'; endpoint = `/work-requests/${currentRequest.value.id}` }
  actionLoading.value = true
  try { await request[method](endpoint, method === 'delete' ? undefined : payload); toast.success(({ assign: '需求已分配', progress: '进度已更新', 'submit-review': '已提交验收', approve: '验收已通过', reject: '已退回修改', return: '需求已退回', cancel: '需求已取消', delete: '需求已删除' })[actionType.value]); actionModalOpen.value = false; if (actionType.value === 'delete') { detailOpen.value = false; currentRequest.value = null } else await refreshDetail(); await fetchRequests() } catch (error) { toast.error(error?.response?.data?.error || '操作失败') } finally { actionLoading.value = false }
}

onMounted(() => { fetchRequests(); fetchUsers() })
</script>

<style scoped>
.requests-page { min-height: calc(100vh - 64px); color: #262626; background: #fafafa; }
.requests-shell { width: 100%; padding: 28px 32px 48px; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 23px; }
.heading-kicker { display: flex; align-items: center; gap: 6px; margin-bottom: 5px; color: #737373; font-size: 12px; font-weight: 700; }
.page-heading h1 { margin: 0; color: #171717; font-size: 25px; font-weight: 750; letter-spacing: 0; line-height: 1.2; }
.page-heading p { margin: 6px 0 0; color: #737373; font-size: 13px; }
.primary-action { display: inline-flex; height: 40px; align-items: center; gap: 7px; border-radius: 6px; padding: 0 15px; color: #fff; background: #171717; font-size: 13px; font-weight: 700; box-shadow: 0 1px 2px rgb(0 0 0 / 8%); }
.primary-action:hover { background: #404040; }
.filter-toolbar { display: grid; grid-template-columns: minmax(220px, 1.5fr) repeat(3, minmax(125px, .72fr)) auto auto; gap: 8px; border: 1px solid #e5e5e5; border-bottom: 0; padding: 12px; background: #fff; }
.search-field { position: relative; }
.search-field svg { position: absolute; top: 50%; left: 11px; color: #a3a3a3; transform: translateY(-50%); }
.search-field input, .filter-toolbar select, .action-form input[type='text'], .action-form select, .action-form textarea { width: 100%; border: 1px solid #d4d4d4; border-radius: 5px; color: #404040; background: #fff; outline: none; }
.search-field input { height: 36px; padding: 0 10px 0 34px; font-size: 13px; }
.filter-toolbar select { height: 36px; padding: 0 28px 0 10px; font-size: 12px; }
.search-field input:focus, .filter-toolbar select:focus, .action-form input:focus, .action-form select:focus, .action-form textarea:focus { border-color: #737373; box-shadow: 0 0 0 3px rgb(23 23 23 / 6%); }
.filter-button, .reset-button { display: inline-flex; height: 36px; align-items: center; justify-content: center; gap: 5px; border-radius: 5px; padding: 0 12px; font-size: 12px; font-weight: 700; }
.filter-button { color: #fff; background: #262626; }
.reset-button { border: 1px solid #d4d4d4; color: #737373; background: #fff; }
.list-panel { border: 1px solid #e5e5e5; background: #fff; }
.list-meta { display: flex; height: 45px; align-items: center; gap: 8px; border-bottom: 1px solid #e5e5e5; padding: 0 14px; color: #525252; font-size: 12px; }
.list-meta strong { display: grid; min-width: 21px; height: 21px; place-items: center; border-radius: 5px; color: #525252; background: #f0f0f0; font-size: 11px; }
.list-note { margin-left: auto; color: #a3a3a3; }
.table-scroll { overflow-x: auto; }
.request-table { width: 100%; min-width: 1080px; border-collapse: collapse; table-layout: fixed; }
.request-table th { height: 42px; border-bottom: 1px solid #e5e5e5; padding: 0 13px; color: #737373; background: #fafafa; font-size: 11px; font-weight: 700; text-align: left; }
.request-table td { height: 58px; border-bottom: 1px solid #ededed; padding: 0 13px; color: #525252; font-size: 12px; vertical-align: middle; }
.request-table tbody tr:last-child td { border-bottom: 0; }
.request-table tbody tr:hover { background: #fcfcfc; }
.request-table th:nth-child(1) { width: 145px; }.request-table .title-column { width: 290px; }.request-table th:nth-child(3) { width: 115px; }.request-table th:nth-child(4) { width: 145px; }.request-table th:nth-child(5) { width: 155px; }.request-table th:nth-child(6) { width: 140px; }.request-table .progress-column { width: 145px; }
.work-no { color: #737373; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; font-weight: 600; }
.title-link { display: block; overflow: hidden; max-width: 100%; color: #262626; font-size: 13px; font-weight: 700; text-align: left; text-overflow: ellipsis; white-space: nowrap; }
.title-link:hover { color: #0369a1; text-decoration: underline; text-underline-offset: 3px; }
.status-badge { display: inline-flex; height: 25px; align-items: center; gap: 5px; border-radius: 5px; padding: 0 8px; font-size: 11px; font-weight: 700; white-space: nowrap; }
.status-badge { color: #525252; background: #f5f5f5; }.status-badge i { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.status-pending { color: #854d0e; background: #fef9c3; }.status-in_progress { color: #1d4ed8; background: #dbeafe; }.status-pending_review { color: #7e22ce; background: #f3e8ff; }.status-completed { color: #15803d; background: #dcfce7; }.status-canceled { color: #737373; background: #e5e5e5; }
.person-cell { display: inline-flex; max-width: 100%; align-items: center; gap: 7px; color: #404040; white-space: nowrap; }.muted { color: #a3a3a3; }.date-cell { color: #737373; white-space: nowrap; }
.progress-cell { display: flex; align-items: center; gap: 8px; }.progress-track { overflow: hidden; height: 6px; flex: 1; border-radius: 3px; background: #e5e5e5; }.progress-track span { display: block; height: 100%; border-radius: inherit; background: #22c55e; transition: width .3s ease; }.progress-cell strong { width: 31px; color: #737373; font-size: 10px; text-align: right; }
.loading-state, .empty-state { display: flex; min-height: 330px; flex-direction: column; align-items: center; justify-content: center; color: #a3a3a3; }.loading-state { gap: 9px; font-size: 13px; }.empty-icon { display: grid; width: 56px; height: 56px; margin-bottom: 13px; place-items: center; border-radius: 7px; color: #a3a3a3; background: #f5f5f5; }.empty-state strong { color: #525252; font-size: 14px; }.empty-state p { margin: 4px 0 15px; font-size: 12px; }.empty-state button { display: inline-flex; height: 36px; align-items: center; gap: 6px; border-radius: 6px; padding: 0 12px; color: #fff; background: #262626; font-size: 12px; font-weight: 700; }
.pagination { display: flex; height: 53px; align-items: center; justify-content: flex-end; gap: 13px; border-top: 1px solid #e5e5e5; padding: 0 14px; color: #737373; font-size: 11px; }.pagination div { display: flex; gap: 5px; }.pagination button { display: grid; width: 30px; height: 30px; place-items: center; border: 1px solid #d4d4d4; border-radius: 5px; background: #fff; }.pagination button:disabled { cursor: not-allowed; opacity: .35; }
.mobile-request-list { display: none; }
.action-form { display: flex; flex-direction: column; gap: 8px; }.action-form > label { color: #404040; font-size: 13px; font-weight: 700; }.action-form > label span { margin-left: 3px; color: #dc2626; }.action-form select, .action-form input[type='text'] { height: 42px; padding: 0 11px; }.action-form textarea { resize: vertical; padding: 10px 11px; line-height: 1.6; }.action-form input[type='range'] { width: 100%; accent-color: #171717; }.range-labels { display: flex; justify-content: space-between; color: #a3a3a3; font-size: 11px; }.path-row { display: flex; gap: 7px; }.path-row button { display: grid; width: 40px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #737373; }.path-row button:hover:not(:disabled) { color: #b91c1c; background: #fee2e2; }.path-row button:disabled { opacity: .3; }.add-path { display: inline-flex; width: fit-content; align-items: center; gap: 5px; margin-top: 2px; color: #525252; font-size: 12px; font-weight: 700; }.confirm-copy { display: flex; align-items: flex-start; gap: 13px; color: #b45309; }.confirm-copy div { color: #404040; }.confirm-copy strong { font-size: 14px; }.confirm-copy p { margin: 4px 0 0; color: #737373; font-size: 12px; line-height: 1.6; }.modal-cancel, .modal-confirm { display: inline-flex; height: 38px; align-items: center; justify-content: center; gap: 6px; border-radius: 6px; padding: 0 15px; font-size: 13px; font-weight: 700; }.modal-cancel { border: 1px solid #d4d4d4; color: #525252; background: #fff; }.modal-confirm { color: #fff; background: #171717; }.modal-confirm.danger { background: #b91c1c; }.modal-cancel:disabled, .modal-confirm:disabled { opacity: .55; }
@media (max-width: 1200px) { .filter-toolbar { grid-template-columns: minmax(220px, 1.4fr) repeat(2, minmax(130px, .7fr)) auto auto; }.filter-toolbar select:nth-of-type(3) { display: none; } }
@media (max-width: 760px) { .requests-shell { padding: 20px 16px 36px; }.page-heading { align-items: center; }.page-heading h1 { font-size: 22px; }.page-heading p { display: none; }.primary-action { padding: 0 12px; }.filter-toolbar { grid-template-columns: 1fr 1fr; border-bottom: 1px solid #e5e5e5; }.search-field { grid-column: 1 / -1; }.filter-toolbar select { display: block !important; }.filter-toolbar select:nth-of-type(3) { display: none !important; }.filter-button, .reset-button { display: none; }.list-panel { margin-top: 10px; }.list-meta { border-bottom: 0; }.table-scroll { display: none; }.mobile-request-list { display: flex; flex-direction: column; border-top: 1px solid #e5e5e5; }.mobile-request { display: flex; flex-direction: column; gap: 10px; border-bottom: 1px solid #e5e5e5; padding: 15px 13px; text-align: left; }.mobile-request:last-child { border-bottom: 0; }.mobile-request:hover { background: #fafafa; }.mobile-request-top, .mobile-request-meta { display: flex; align-items: center; justify-content: space-between; gap: 8px; }.mobile-request > strong { overflow: hidden; color: #262626; font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }.mobile-request-meta { justify-content: flex-start; color: #737373; font-size: 11px; }.mobile-request-meta span:not(:last-child)::after { margin-left: 8px; color: #d4d4d4; content: '·'; } }
/* Taste-skill: warm, document-like operations workspace. */
.requests-page { min-height: calc(100dvh - 64px); color: #292826; background: #fafafa; }
.requests-shell { width: min(100%, 1680px); margin: 0 auto; padding: 34px 36px 56px; }
.page-heading { margin-bottom: 24px; }
.heading-kicker { margin-bottom: 7px; color: #85817a; font-size: 11px; font-weight: 650; }
.page-heading h1 { color: #1f1e1c; font-size: 28px; font-weight: 760; line-height: 1.15; }
.page-heading p { max-width: 65ch; margin-top: 7px; color: #77736d; line-height: 1.5; }
.heading-actions { display: flex; align-items: center; gap: 8px; }
.primary-action, .quiet-action { display: inline-flex; height: 40px; align-items: center; justify-content: center; gap: 7px; border-radius: 6px; padding: 0 14px; font-size: 13px; font-weight: 700; transition: background .18s ease, border-color .18s ease, color .18s ease, transform .12s ease; }
.primary-action { color: #fff; background: #232220; box-shadow: none; }.primary-action:hover { background: #3b3935; }
.quiet-action { border: 1px solid #dedbd5; color: #69655f; background: #fbfbfa; }.quiet-action:hover { border-color: #c3bfb8; color: #292826; background: #fff; }
.primary-action:active, .quiet-action:active, .filter-button:active, .reset-button:active { transform: translateY(1px); }
.primary-action:focus-visible, .quiet-action:focus-visible, .filter-button:focus-visible, .reset-button:focus-visible, .title-link:focus-visible, .pagination button:focus-visible { outline: 2px solid #736c63; outline-offset: 2px; }
.filter-toolbar { border-color: #e2dfda; border-radius: 8px 8px 0 0; padding: 11px; background: #fbfbfa; }
.search-field input, .filter-toolbar select, .action-form input[type='text'], .action-form select, .action-form textarea { border-color: #d8d5cf; color: #373532; transition: border-color .18s ease, box-shadow .18s ease; }
.search-field input:focus, .filter-toolbar select:focus, .action-form input:focus, .action-form select:focus, .action-form textarea:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.filter-button { background: #2d2b28; }.filter-button:hover { background: #494640; }
.reset-button { border-color: #d8d5cf; color: #77736d; }.reset-button:hover { color: #292826; background: #f6f5f2; }
.list-panel { overflow: hidden; border-color: #e2dfda; border-top: 0; border-radius: 0 0 8px 8px; }
.list-meta { height: 49px; justify-content: space-between; border-color: #ebe9e5; padding: 0 15px; }
.list-meta-title { display: flex; align-items: baseline; gap: 7px; }.list-meta-title > span { color: #302e2b; font-weight: 750; }.list-meta-title strong { min-width: 0; height: auto; display: inline; color: #302e2b; background: transparent; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-variant-numeric: tabular-nums; }.list-meta small, .list-note { color: #96918a; font-size: 11px; }
.request-table th { height: 40px; border-color: #e8e5e0; color: #85817a; background: #f8f7f5; font-weight: 650; }
.request-table td { height: 60px; border-color: #efede9; color: #55514c; }
.request-table tbody tr { transition: background .16s ease; }.request-table tbody tr:hover { background: #faf9f7; }
.work-no { color: #77736d; font-variant-numeric: tabular-nums; }
.title-link { color: #292826; transition: color .16s ease; }.title-link:hover { color: #8b4c26; text-decoration-color: #d4b9a6; }
.status-badge { color: #66615b; background: #f2f1ef; }.status-badge i { width: 5px; height: 5px; }
.status-pending { color: #8a620f; background: #faf2dc; }.status-in_progress { color: #35617d; background: #e8f0f4; }.status-pending_review { color: #76517b; background: #f1eaf2; }.status-completed { color: #486d4c; background: #ebf1ea; }.status-canceled { color: #77736d; background: #efeeec; }
.progress-track span { background: #5d7f63; }.progress-cell strong { color: #77736d; font-variant-numeric: tabular-nums; }
.loading-state { gap: 16px; font-size: 12px; }.table-skeleton { display: flex; width: min(720px, calc(100% - 48px)); flex-direction: column; gap: 12px; }.table-skeleton i { height: 34px; border-radius: 5px; background: #ece9e4; animation: request-pulse 1.5s ease-in-out infinite alternate; }
.empty-icon { width: 52px; height: 52px; border-radius: 8px; color: #8d8881; background: #f1efeb; }.empty-state strong { color: #494640; }.empty-state p { max-width: 420px; line-height: 1.6; text-align: center; }.empty-state button { background: #2d2b28; }.error-state .empty-icon { color: #9f4f42; background: #f8e9e6; }
.request-editor-page { height: calc(100dvh - 64px); overflow: hidden; background: #fafafa; }
.request-editor-canvas { width: min(100%, 1680px); height: 100%; margin: 0 auto; overflow: hidden; padding: 28px 36px 36px; }
@keyframes request-pulse { to { opacity: .48; } }
@media (prefers-reduced-motion: reduce) { .table-skeleton i { animation: none; }.primary-action, .quiet-action, .filter-button, .reset-button, .request-table tbody tr { transition: none; } }
@media (max-width: 760px) {
  .requests-shell { padding: 22px 15px 38px; }.page-heading { align-items: flex-end; }.page-heading h1 { font-size: 23px; }.page-heading p, .quiet-action { display: none; }.primary-action { padding: 0 11px; }
  .filter-toolbar { border-radius: 7px; }.list-panel { border: 1px solid #e2dfda; border-radius: 7px; }.list-note { display: none; }
  .mobile-request-list { border-color: #e8e5e0; }.mobile-request { border-color: #ebe9e5; padding: 16px 14px; transition: background .16s ease; }.mobile-request:hover { background: #faf9f7; }.mobile-request:active { background: #f4f2ee; }.mobile-request > strong { color: #292826; }.mobile-request-meta { color: #77736d; }
}
@media (max-width: 900px) { .request-editor-canvas { padding: 18px 15px 24px; } }
</style>
