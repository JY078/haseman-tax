<template>
  <TransitionRoot as="template" :show="open">
    <Dialog as="div" class="relative z-50" @close="closeDrawer">
      <TransitionChild as="template" enter="ease-out duration-220" enter-from="opacity-0" enter-to="opacity-100" leave="ease-in duration-160" leave-from="opacity-100" leave-to="opacity-0">
        <div class="fixed inset-0 bg-neutral-900/35" />
      </TransitionChild>
      <div class="fixed inset-0 overflow-hidden">
        <div class="absolute inset-0 overflow-hidden">
          <div class="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-0 sm:pl-10">
            <TransitionChild as="template" enter="transform transition ease-out duration-250" enter-from="translate-x-full" enter-to="translate-x-0" leave="transform transition ease-in duration-180" leave-from="translate-x-0" leave-to="translate-x-full">
              <DialogPanel :class="['drawer-panel', { 'drawer-panel-expanded': expanded }]" class="pointer-events-auto">
                <template v-if="request">
                  <header class="drawer-header">
                    <div class="drawer-headline">
                      <div class="drawer-eyebrow"><span class="video-label">视频需求</span><span class="work-no">{{ request.work_no }}</span></div>
                      <DialogTitle>{{ request.title }}</DialogTitle>
                    </div>
                    <div class="drawer-header-actions">
                      <button type="button" class="expand-button" :title="expanded ? '收起全屏' : '展开全屏'" @click="expanded = !expanded">
                        <ArrowsPointingInIcon v-if="expanded" class="h-5 w-5" />
                        <ArrowsPointingOutIcon v-else class="h-5 w-5" />
                      </button>
                      <button type="button" class="close-button" title="关闭" @click="closeDrawer"><XMarkIcon class="h-5 w-5" /></button>
                    </div>
                  </header>

                  <div class="drawer-scroll">
                    <section class="status-strip">
                      <div><span>状态</span><strong :class="[`status-${request.status}`]"><i />{{ statusMeta.label }}</strong></div>
                      <div><span>当前进度</span><div class="large-progress"><div><b :style="{ width: `${request.progress}%` }" /></div><strong>{{ request.progress }}%</strong></div></div>
                    </section>

                    <section class="detail-section metadata-section">
                      <h3>任务信息</h3>
                      <dl class="metadata-grid">
                        <div><dt>创建人</dt><dd><Avatar :user="request.creator" />{{ userName(request.creator) }}</dd></div>
                        <div><dt>创建时间</dt><dd>{{ formatDate(request.created_at) }}</dd></div>
                        <div><dt>处理人</dt><dd v-if="request.assignee"><Avatar :user="request.assignee" />{{ userName(request.assignee) }}</dd><dd v-else class="muted">待管理员分配</dd></div>
                        <div><dt>更新时间</dt><dd>{{ formatDate(request.updated_at) }}</dd></div>
                      </dl>
                    </section>

                    <section class="detail-section">
                      <h3>需求内容</h3>
                      <div class="video-detail-content">
                        <div class="video-duration"><ClockIcon class="h-5 w-5" /><div><span>视频总时长</span><strong>{{ request.content?.duration_seconds }} 秒</strong></div></div>
                        <div v-if="request.content?.dimensions || request.content?.resolution || request.content?.video_type || request.content?.voice_requirement" class="video-spec-grid">
                          <div v-if="request.content?.dimensions"><span>尺寸</span><strong>{{ request.content.dimensions }}</strong></div>
                          <div v-if="request.content?.resolution"><span>分辨率</span><strong>{{ request.content.resolution }}</strong></div>
                          <div v-if="request.content?.video_type"><span>视频类型</span><strong>{{ request.content.video_type }}</strong></div>
                          <div v-if="request.content?.voice_requirement"><span>声音要求</span><strong>{{ request.content.voice_requirement }}{{ request.content.voice_requirement === '自定义' && request.content.voice_requirement_detail ? ` · ${request.content.voice_requirement_detail}` : '' }}</strong></div>
                        </div>
                        <InfoBlock v-if="request.content?.overall_requirement" class="overall-requirement-detail" label="视频总要求" :value="request.content.overall_requirement" />
                        <div class="shot-group-list">
                          <article v-for="(group, groupIndex) in storyboardGroups" :key="groupIndex" class="detail-shot-group">
                            <header class="shot-group-head"><div><span>{{ groupIndex + 1 }}</span><strong>{{ group.title }}</strong></div><b>{{ groupDuration(group) }} 秒 · {{ group.shots.length }} 个画面</b></header>
                            <p v-if="group.note" class="group-note">{{ group.note }}</p>
                            <div class="detail-shot-table">
                              <div class="detail-shot-table-head"><span>画面</span><span>画面情节</span><span>时长</span><span>字幕</span><span>制作备注</span><span>参考图</span></div>
                              <div v-for="(shot, shotIndex) in group.shots" :key="shotIndex" class="detail-shot-row">
                                <span class="detail-shot-index" data-label="画面">{{ String(shotIndex + 1).padStart(2, '0') }}</span>
                                <p class="detail-cell-copy" data-label="画面情节">{{ shot.scene || '未填写' }}</p>
                                <span class="detail-shot-duration" data-label="时长">{{ shot.duration_seconds }} 秒</span>
                                <p class="detail-cell-copy muted-copy" data-label="字幕">{{ shot.subtitle || '—' }}</p>
                                <p class="detail-cell-copy muted-copy" data-label="制作备注">{{ shot.note || '—' }}</p>
                                <div class="detail-shot-references" data-label="参考图"><ImageGallery v-if="shot.references?.length" title="" :images="shot.references" compact @preview="previewImage = $event" /><span v-else>—</span></div>
                              </div>
                            </div>
                          </article>
                        </div>
                      </div>
                    </section>

                    <section v-if="request.delivery_note || deliveryPaths.length" class="detail-section delivery-section">
                      <h3>交付内容</h3>
                      <InfoBlock label="交付说明" :value="request.delivery_note" />
                      <div v-if="deliveryPaths.length" class="info-block"><span>共享盘文件路径</span><button v-for="path in deliveryPaths" :key="path" type="button" class="delivery-path" title="复制路径" @click="copyPath(path)"><FolderOpenIcon class="h-4 w-4" /><code>{{ path }}</code><ClipboardDocumentIcon class="h-4 w-4" /></button></div>
                    </section>
                    <section v-if="request.rejection_reason" class="rejection-box"><ExclamationCircleIcon class="h-5 w-5" /><div><strong>最近一次验收退回原因</strong><p>{{ request.rejection_reason }}</p></div></section>
                  </div>

                  <footer class="drawer-actions">
                    <div class="secondary-actions">
                      <button type="button" title="复制为新需求" @click="emit('copy', request)"><DocumentDuplicateIcon class="h-4 w-4" />复制</button>
                      <button v-if="canEdit" type="button" title="编辑需求" @click="emit('edit', request)"><PencilSquareIcon class="h-4 w-4" />编辑</button>
                      <Menu v-if="moreActions.length" as="div" class="relative">
                        <MenuButton class="more-button" title="更多操作"><EllipsisHorizontalIcon class="h-5 w-5" /></MenuButton>
                        <MenuItems class="more-menu">
                          <MenuItem v-for="action in moreActions" :key="action.type" v-slot="{ active }"><button type="button" :class="[{ active }, action.danger ? 'danger' : '']" @click="emit('action', action.type)"><component :is="action.icon" class="h-4 w-4" />{{ action.label }}</button></MenuItem>
                        </MenuItems>
                      </Menu>
                    </div>
                    <div class="primary-actions">
                      <button v-if="canAssign" type="button" class="solid-action" @click="emit('action', 'assign')"><UserPlusIcon class="h-4 w-4" />{{ request.assignee ? '更换处理人' : '分配处理人' }}</button>
                      <button v-if="canProgress" type="button" class="outline-action" @click="emit('action', 'progress')"><ChartBarIcon class="h-4 w-4" />更新进度</button>
                      <button v-if="canSubmitReview" type="button" class="solid-action" @click="emit('action', 'submit-review')"><PaperAirplaneIcon class="h-4 w-4" />提交验收</button>
                      <button v-if="canReview" type="button" class="reject-action" @click="emit('action', 'reject')">退回修改</button>
                      <button v-if="canReview" type="button" class="approve-action" @click="emit('action', 'approve')"><CheckIcon class="h-4 w-4" />验收通过</button>
                    </div>
                  </footer>
                </template>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>

  <Teleport to="body"><div v-if="previewImage" class="image-preview-overlay" role="dialog" aria-modal="true" @click.self="previewImage = ''"><button type="button" title="关闭预览" @click="previewImage = ''"><XMarkIcon class="h-6 w-6" /></button><img :src="displayImage(previewImage)" alt="参考图预览" /></div></Teleport>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, Menu, MenuButton, MenuItem, MenuItems, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { ArrowsPointingInIcon, ArrowsPointingOutIcon, ChartBarIcon, CheckIcon, ClipboardDocumentIcon, ClockIcon, DocumentDuplicateIcon, EllipsisHorizontalIcon, ExclamationCircleIcon, FolderOpenIcon, PaperAirplaneIcon, PencilSquareIcon, TrashIcon, UserPlusIcon, XCircleIcon, XMarkIcon, ArrowUturnLeftIcon } from '@heroicons/vue/24/outline'
import { getDisplayImageUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'
import Avatar from './WorkRequestAvatar.vue'
import InfoBlock from './WorkRequestInfoBlock.vue'
import ImageGallery from './WorkRequestImageGallery.vue'

const props = defineProps({ open: { type: Boolean, default: false }, request: { type: Object, default: null }, users: { type: Array, default: () => [] }, currentUserId: { type: Number, default: 0 }, isAdmin: { type: Boolean, default: false }, actionLoading: { type: Boolean, default: false } })
const emit = defineEmits(['update:open', 'edit', 'copy', 'action', 'refresh'])
const toast = useToast(), previewImage = ref(''), expanded = ref(false)
const statusMap = { pending: '待受理', in_progress: '进行中', pending_review: '待验收', completed: '已完成', canceled: '已取消' }
const statusMeta = computed(() => ({ label: statusMap[props.request?.status] || props.request?.status }))
const deliveryPaths = computed(() => Array.isArray(props.request?.delivery_paths) ? props.request.delivery_paths : [])
const storyboardGroups = computed(() => {
  const content = props.request?.content || {}
  if (Array.isArray(content.shot_groups) && content.shot_groups.length) {
    return content.shot_groups.map((group, index) => ({
      title: group.title || `章节 ${index + 1}`,
      note: group.note || '',
      shots: (group.shots || []).map(shot => ({ ...shot, scene: shot.scene || shot.display_requirement || '', subtitle: shot.subtitle || shot.dialogue || '', note: shot.note || '' })),
    }))
  }
  const legacyShots = Array.isArray(content.shots) ? content.shots : []
  return legacyShots.length ? [{ title: '章节 1', note: '', shots: legacyShots.map(shot => ({ ...shot, scene: shot.display_requirement || '', subtitle: shot.dialogue || '', note: '' })) }] : []
})
const isCreator = computed(() => props.request?.creator_id === props.currentUserId)
const isAssignee = computed(() => props.request?.assignee_id === props.currentUserId)
const canEdit = computed(() => props.request?.status === 'pending' && (isCreator.value || props.isAdmin))
const canAssign = computed(() => props.isAdmin && ['pending', 'in_progress'].includes(props.request?.status))
const canProgress = computed(() => props.request?.status === 'in_progress' && (isAssignee.value || props.isAdmin))
const canSubmitReview = computed(() => props.request?.status === 'in_progress' && (isAssignee.value || props.isAdmin))
const canReview = computed(() => props.request?.status === 'pending_review' && (isCreator.value || props.isAdmin))
const moreActions = computed(() => {
  const actions = []
  if (props.isAdmin && props.request?.status === 'in_progress') actions.push({ type: 'return', label: '退回创建人修改', icon: ArrowUturnLeftIcon, danger: true })
  if ((props.isAdmin && !['completed', 'canceled'].includes(props.request?.status)) || (isCreator.value && props.request?.status === 'pending')) actions.push({ type: 'cancel', label: '取消需求', icon: XCircleIcon, danger: true })
  if (props.isAdmin) actions.push({ type: 'delete', label: '删除需求', icon: TrashIcon, danger: true })
  return actions
})
watch(() => props.open, value => { if (!value) expanded.value = false })
function closeDrawer() { expanded.value = false; emit('update:open', false) }
function userName(user) { return user?.name || user?.username || `用户 #${user?.id || ''}` }
function formatDate(value) { if (!value) return '—'; return new Date(value).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replaceAll('/', '-') }
function displayImage(url) { return getDisplayImageUrl(url) }
function groupDuration(group) { return Number((group.shots || []).reduce((total, shot) => total + Number(shot.duration_seconds || 0), 0).toFixed(1)) }
async function copyPath(path) { try { await navigator.clipboard.writeText(path); toast.success('文件路径已复制') } catch { toast.error('复制失败，请手动选择路径') } }
</script>

<style scoped>
.drawer-panel { display: flex; width: min(880px, 100vw); height: 100vh; flex-direction: column; background: #fff; box-shadow: -16px 0 50px rgb(0 0 0 / 16%); }
.drawer-header { display: flex; min-height: 93px; align-items: flex-start; justify-content: space-between; gap: 20px; border-bottom: 1px solid #e5e5e5; padding: 19px 23px; }.drawer-eyebrow { display: flex; align-items: center; gap: 9px; margin-bottom: 9px; }.drawer-header h2 { max-width: 620px; margin: 0; color: #171717; font-size: 19px; font-weight: 750; line-height: 1.4; }.close-button { display: grid; width: 33px; height: 33px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #737373; }.close-button:hover { background: #f5f5f5; }.work-no { color: #a3a3a3; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; }.video-label { display: inline-flex; height: 23px; align-items: center; border-radius: 5px; padding: 0 8px; color: #9a3412; background: #ffedd5; font-size: 11px; font-weight: 700; }
.drawer-scroll { flex: 1; overflow-y: auto; padding: 22px 23px 30px; }.status-strip { display: grid; grid-template-columns: 160px 1fr; gap: 20px; border: 1px solid #e5e5e5; border-radius: 7px; padding: 14px 16px; background: #fafafa; }.status-strip > div { display: flex; flex-direction: column; gap: 7px; }.status-strip span { color: #a3a3a3; font-size: 11px; font-weight: 600; }.status-strip > div > strong { display: inline-flex; width: fit-content; align-items: center; gap: 6px; font-size: 13px; }.status-strip > div > strong i { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }.status-pending { color: #a16207; }.status-in_progress { color: #2563eb; }.status-pending_review { color: #9333ea; }.status-completed { color: #16a34a; }.status-canceled { color: #737373; }.large-progress { display: flex; align-items: center; gap: 10px; }.large-progress > div { overflow: hidden; height: 7px; flex: 1; border-radius: 4px; background: #e5e5e5; }.large-progress b { display: block; height: 100%; border-radius: inherit; background: #22c55e; }.large-progress strong { width: 38px; color: #525252; font-size: 12px; text-align: right; }
.detail-section { margin-top: 26px; }.detail-section > h3 { margin: 0 0 14px; border-bottom: 1px solid #ededed; padding-bottom: 9px; color: #404040; font-size: 13px; font-weight: 800; }.metadata-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px 22px; margin: 0; }.metadata-grid div { min-width: 0; }.metadata-grid dt { margin-bottom: 5px; color: #a3a3a3; font-size: 11px; }.metadata-grid dd { display: flex; align-items: center; gap: 7px; margin: 0; color: #404040; font-size: 12px; }.metadata-grid dd.muted { color: #a3a3a3; }.video-duration { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; border-radius: 6px; padding: 11px 13px; color: #2563eb; background: #eff6ff; }.video-duration div { display: flex; flex-direction: column; }.video-duration span { color: #60a5fa; font-size: 10px; }.video-duration strong { font-size: 14px; }.shot-group-list { display: flex; flex-direction: column; gap: 14px; }.detail-shot-group { border: 1px solid #dedede; border-radius: 7px; padding: 14px; background: #fafafa; }.shot-group-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }.shot-group-head > div { display: flex; min-width: 0; align-items: center; gap: 9px; }.shot-group-head > div > span { display: grid; width: 26px; height: 26px; flex: 0 0 auto; place-items: center; border-radius: 5px; color: #fff; background: #262626; font-size: 11px; font-weight: 800; }.shot-group-head strong { color: #262626; font-size: 14px; }.shot-group-head b { color: #525252; font-size: 12px; }.group-note { margin: 8px 0 0 35px; color: #737373; font-size: 12px; line-height: 1.6; }.small-shot-detail-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 13px; }.small-shot-detail { min-width: 0; border: 1px solid #e5e5e5; border-radius: 6px; padding: 12px; background: #fff; }.small-shot-detail > header { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; }.small-shot-detail > header strong { color: #404040; font-size: 12px; }.small-shot-detail > header span { border-radius: 4px; padding: 3px 6px; color: #525252; background: #f5f5f5; font-size: 10px; }.delivery-section { border-top: 1px solid #e5e5e5; padding-top: 22px; }.delivery-path { display: flex; width: 100%; align-items: center; gap: 8px; border: 1px solid #d4d4d4; border-radius: 6px; padding: 10px 11px; color: #525252; background: #fafafa; text-align: left; }.delivery-path:hover { border-color: #a3a3a3; background: #f5f5f5; }.delivery-path code { overflow: hidden; flex: 1; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }.rejection-box { display: flex; align-items: flex-start; gap: 10px; margin-top: 20px; border: 1px solid #fed7aa; border-radius: 7px; padding: 12px; color: #c2410c; background: #fff7ed; }.rejection-box strong { font-size: 12px; }.rejection-box p { margin: 4px 0 0; color: #9a3412; font-size: 12px; line-height: 1.6; }
.video-detail-content + :deep(.info-block) { margin-top: 18px; }
.drawer-actions { display: flex; min-height: 66px; align-items: center; justify-content: space-between; gap: 12px; border-top: 1px solid #e5e5e5; padding: 12px 18px; background: #fff; }.secondary-actions, .primary-actions { display: flex; align-items: center; gap: 7px; }.drawer-actions button { display: inline-flex; height: 36px; align-items: center; justify-content: center; gap: 6px; border-radius: 6px; padding: 0 11px; font-size: 12px; font-weight: 700; }.secondary-actions > button, .more-button, .outline-action, .reject-action { border: 1px solid #d4d4d4; color: #525252; background: #fff; }.secondary-actions > button:hover, .outline-action:hover { background: #fafafa; }.solid-action, .approve-action { color: #fff; background: #171717; }.approve-action { background: #15803d; }.reject-action { color: #b91c1c; border-color: #fecaca; }.more-button { width: 36px; padding: 0 !important; }.more-menu { position: absolute; right: 0; bottom: 42px; width: 180px; overflow: hidden; border: 1px solid #e5e5e5; border-radius: 7px; padding: 4px; background: #fff; box-shadow: 0 10px 35px rgb(0 0 0 / 14%); }.more-menu button { width: 100%; justify-content: flex-start; }.more-menu button.active { background: #f5f5f5; }.more-menu button.danger { color: #b91c1c; }
.image-preview-overlay { position: fixed; inset: 0; z-index: 80; display: flex; align-items: center; justify-content: center; padding: 30px; background: rgb(0 0 0 / 82%); }.image-preview-overlay > button { position: absolute; top: 18px; right: 18px; display: grid; width: 40px; height: 40px; place-items: center; border-radius: 6px; color: #fff; background: rgb(255 255 255 / 12%); }.image-preview-overlay img { max-width: 92vw; max-height: 90vh; object-fit: contain; }
@media (max-width: 640px) { .drawer-panel { width: 100vw; }.drawer-header { padding: 16px; }.drawer-scroll { padding: 17px 16px 25px; }.status-strip { grid-template-columns: 120px 1fr; }.metadata-grid, .small-shot-detail-list { grid-template-columns: 1fr; }.drawer-actions { align-items: stretch; flex-direction: column-reverse; }.secondary-actions, .primary-actions { width: 100%; }.primary-actions button { flex: 1; }.secondary-actions > button { flex: 1; }.more-button { flex: 0 0 auto !important; } }
/* Taste-skill: detail view reads like a delivery brief, not a stack of modal cards. */
.drawer-panel { width: min(920px, 100vw); color: #292826; box-shadow: -18px 0 56px rgb(65 58 49 / 13%); transition: width .22s ease; }
.drawer-panel-expanded { width: 100vw; max-width: none; }
.drawer-header { min-height: 100px; border-color: #e4e1dc; padding: 21px 28px; }.drawer-eyebrow { margin-bottom: 10px; }.video-label { color: #7c4b26; background: #f4e9df; }.work-no { color: #8c8780; }.drawer-header h2 { color: #201f1d; font-size: 21px; }.drawer-header-actions { display: flex; flex: 0 0 auto; align-items: center; gap: 3px; }.expand-button { display: grid; width: 33px; height: 33px; place-items: center; border-radius: 6px; color: #77736d; transition: background .16s ease, color .16s ease; }.expand-button:hover { color: #292826; background: #f3f1ed; }.expand-button:focus-visible { outline: 2px solid #736c63; outline-offset: 2px; }.close-button { color: #77736d; transition: background .16s ease, color .16s ease; }.close-button:hover { color: #292826; background: #f3f1ed; }.close-button:focus-visible { outline: 2px solid #736c63; outline-offset: 2px; }
.drawer-scroll { padding: 25px 28px 32px; }.status-strip { grid-template-columns: 170px 1fr; border-color: #e2dfda; border-radius: 8px; padding: 15px 17px; background: #faf9f7; }.status-strip span { color: #96918a; }.status-strip > div > strong { font-variant-numeric: tabular-nums; }.status-pending { color: #8a620f; }.status-in_progress { color: #35617d; }.status-pending_review { color: #76517b; }.status-completed { color: #486d4c; }.status-canceled { color: #77736d; }.large-progress > div { background: #e7e4df; }.large-progress b { background: #5d7f63; }
.detail-section { margin-top: 30px; }.detail-section > h3 { margin-bottom: 16px; border-color: #ebe9e5; color: #3a3733; font-size: 13px; }.metadata-grid { gap: 16px 28px; }.metadata-grid dt { color: #96918a; }.metadata-grid dd { color: #4d4944; }
.video-duration { margin-bottom: 18px; border-radius: 7px; color: #35617d; background: #e8f0f4; }.video-duration span { color: #6d93aa; }.video-duration strong { font-variant-numeric: tabular-nums; }.shot-group-list { gap: 0; }.detail-shot-group { border: 0; border-top: 1px solid #e4e1dc; border-radius: 0; padding: 20px 0 22px; background: transparent; }.detail-shot-group:first-child { border-top: 0; padding-top: 0; }.shot-group-head > div > span { border-radius: 4px; color: #fff; background: #2b2926; }.shot-group-head strong { color: #353330; }.shot-group-head b { color: #5a5650; font-variant-numeric: tabular-nums; }.group-note { color: #77736d; }.small-shot-detail-list { gap: 9px; margin-top: 14px; }.small-shot-detail { border-color: #e6e3de; border-radius: 7px; padding: 13px; }.small-shot-detail > header { margin-bottom: 11px; }.small-shot-detail > header strong { color: #4a4641; }.small-shot-detail > header span { color: #66615b; background: #f0eeeb; font-variant-numeric: tabular-nums; }
.delivery-section { border-color: #e4e1dc; }.delivery-path { border-color: #d8d5cf; background: #faf9f7; transition: border-color .16s ease, background .16s ease; }.delivery-path:hover { border-color: #aaa59d; background: #fff; }.drawer-actions { min-height: 70px; border-color: #e4e1dc; padding: 13px 23px; }.drawer-actions button { transition: background .16s ease, border-color .16s ease, color .16s ease, transform .12s ease; }.drawer-actions button:active { transform: translateY(1px); }.solid-action { background: #2b2926; }.solid-action:hover { background: #45413c; }.approve-action { background: #4e7454; }.approve-action:hover { background: #3f6145; }.outline-action:hover { border-color: #aaa59d; background: #faf9f7; }.reject-action:hover { background: #f8e9e6; }.more-button:focus-visible, .solid-action:focus-visible, .approve-action:focus-visible, .outline-action:focus-visible, .reject-action:focus-visible { outline: 2px solid #736c63; outline-offset: 2px; }
@media (max-width: 640px) { .drawer-header { padding: 17px 16px; }.drawer-header h2 { font-size: 19px; }.drawer-scroll { padding: 19px 16px 25px; }.status-strip { grid-template-columns: 1fr 1fr; }.drawer-actions { padding: 12px 16px; } }
/* Read-only scene table */
.detail-shot-table { overflow-x: auto; margin-top: 14px; border: 1px solid #e4e1dc; border-radius: 7px; }
.detail-shot-table-head, .detail-shot-row { display: grid; min-width: 800px; grid-template-columns: 52px minmax(190px, 1.6fr) 72px minmax(130px, 1fr) minmax(130px, 1fr) 150px; }
.detail-shot-table-head { min-height: 36px; align-items: center; color: #858078; background: #f4f2ee; font-size: 10px; font-weight: 750; }.detail-shot-table-head span { padding: 0 9px; }
.detail-shot-row { align-items: start; border-top: 1px solid #e9e6e1; background: #fff; }.detail-shot-row > * { min-width: 0; padding: 11px 9px; }.detail-shot-row > * + * { border-left: 1px solid #efede9; }
.detail-shot-index { display: block; color: #77736d; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; font-weight: 750; text-align: center; }.detail-shot-duration { color: #4d4944; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; white-space: nowrap; }
.detail-cell-copy { margin: 0; color: #4d4944; font-size: 11px; line-height: 1.65; white-space: pre-wrap; word-break: break-word; }.muted-copy { color: #77736d; }.detail-shot-references > span { color: #aaa59d; font-size: 11px; }
.detail-shot-references :deep(.image-gallery) { margin: 0; }.detail-shot-references :deep(.gallery-title) { display: none; }
@media (max-width: 900px) {
  .detail-shot-table { overflow: visible; border: 0; }
  .detail-shot-table-head { display: none; }
  .detail-shot-row { display: grid; min-width: 0; grid-template-columns: 50px minmax(0, 1fr) 80px; grid-template-areas: 'index scene duration' 'index subtitle subtitle' 'index note note' 'index reference reference'; overflow: hidden; margin-top: 10px; border: 1px solid #e4e1dc; border-radius: 7px; }
  .detail-shot-row:first-of-type { margin-top: 0; }.detail-shot-row > * { padding: 10px; }.detail-shot-row > * + * { border-left: 0; }.detail-shot-row > *::before { display: block; margin-bottom: 5px; color: #96918a; content: attr(data-label); font-size: 9px; font-weight: 700; }
  .detail-shot-index { grid-area: index; background: #f7f6f3; }.detail-shot-index::before { text-align: center; }.detail-shot-row > :nth-child(2) { grid-area: scene; }.detail-shot-duration { grid-area: duration; }.detail-shot-row > :nth-child(4) { grid-area: subtitle; border-top: 1px solid #efede9; }.detail-shot-row > :nth-child(5) { grid-area: note; border-top: 1px solid #efede9; }.detail-shot-references { grid-area: reference; border-top: 1px solid #efede9; }
}
@media (max-width: 520px) {
  .detail-shot-row { grid-template-columns: 42px minmax(0, 1fr); grid-template-areas: 'index scene' 'index duration' 'index subtitle' 'index note' 'index reference'; }.detail-shot-index { grid-row: 1 / span 5; }.detail-shot-duration { border-top: 1px solid #efede9; }
}
.video-spec-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin: 0 0 14px; }
.video-spec-grid > div { min-width: 0; border: 1px solid #e6e3de; border-radius: 6px; padding: 10px 11px; background: #faf9f7; }
.video-spec-grid span { display: block; margin-bottom: 5px; color: #96918a; font-size: 10px; }
.video-spec-grid strong { display: block; overflow: hidden; color: #4d4944; font-size: 12px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.overall-requirement-detail { margin-top: 0; margin-bottom: 16px; }
.overall-requirement-detail :deep(.info-block p) { white-space: pre-wrap; }
@media (max-width: 720px) { .video-spec-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
