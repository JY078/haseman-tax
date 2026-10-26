<template>
  <form ref="formRef" :class="['request-form', { 'page-layout': layout === 'page' }]" @submit.prevent="submit">
    <div class="editor-left-panel">
    <section class="request-section request-settings">
      <FormField label="需求标题" field-id="request-title" required :error="errors.title">
        <input id="request-title" v-model.trim="form.title" maxlength="200" type="text" placeholder="例如：新品 30 秒卖点视频" />
      </FormField>
      <FormField label="视频总时长" required :error="errors.duration_seconds">
        <div class="duration-picker">
          <button v-for="seconds in [15, 30, 60]" :key="seconds" type="button" :class="{ active: form.content.duration_mode === seconds }" @click="selectDuration(seconds)"><strong>{{ seconds }}</strong><span>秒</span></button>
          <button type="button" :class="['custom-duration-button', { active: form.content.duration_mode === 'custom' }]" @click="selectCustomDuration">自定义</button>
        </div>
        <div v-if="form.content.duration_mode === 'custom'" class="custom-duration-row">
          <label class="number-field"><input v-model.number="customDurationMinutes" min="0.1" step="0.1" type="number" aria-label="自定义视频时长" /><span>分钟</span></label>
        </div>
      </FormField>
    </section>

    <section class="request-section video-requirements">
      <div class="requirements-grid">
        <FormField label="尺寸" field-id="video-dimensions">
          <WorkRequestSelect id="video-dimensions" v-model="form.content.dimensions" :options="dimensionOptions" placeholder="请选择尺寸" />
        </FormField>
        <FormField label="分辨率" field-id="video-resolution">
          <WorkRequestSelect id="video-resolution" v-model="form.content.resolution" :options="resolutionOptions" placeholder="请选择分辨率" />
        </FormField>
        <FormField label="视频类型" field-id="video-type">
          <WorkRequestSelect id="video-type" v-model="form.content.video_type" :options="videoTypeOptions" placeholder="请选择视频类型" />
        </FormField>
        <FormField label="声音要求" field-id="voice-requirement">
          <WorkRequestSelect id="voice-requirement" v-model="form.content.voice_requirement" :options="voiceOptions" placeholder="请选择声音要求" />
        </FormField>
        <FormField v-if="form.content.voice_requirement === '自定义'" label="声音说明" field-id="voice-requirement-detail">
          <input id="voice-requirement-detail" v-model.trim="form.content.voice_requirement_detail" maxlength="200" type="text" placeholder="填写声音或配音要求" />
        </FormField>
      </div>
      <FormField class="overall-requirement-field" label="视频总要求" field-id="overall-requirement">
        <textarea id="overall-requirement" v-model.trim="form.content.overall_requirement" rows="3" maxlength="2000" placeholder="填写整个视频的统一要求" />
      </FormField>
    </section>
    <div class="section-divider" />
    </div>

    <section class="request-section storyboard-section">
      <div class="storyboard-toolbar">
        <div class="storyboard-title">
          <h3>章节与画面</h3>
          <div :class="['duration-status', { invalid: durationDifference !== 0 }]">
            <CheckCircleIcon v-if="durationDifference === 0" class="h-4 w-4" />
            <ExclamationCircleIcon v-else class="h-4 w-4" />
            <span>{{ shotTotal }} / {{ form.content.duration_seconds || 0 }} 秒</span>
            <strong v-if="durationDifference !== 0">{{ durationDifference > 0 ? `差 ${durationDifference}` : `超 ${Math.abs(durationDifference)}` }} 秒</strong>
          </div>
        </div>
        <button type="button" class="add-group-button" @click="addGroup"><PlusIcon class="h-4 w-4" />添加章节</button>
      </div>

      <p v-if="errors.groups" class="section-error">{{ errors.groups }}</p>

      <div class="storyboard-workspace">
        <aside class="group-sidebar">
          <article v-for="(group, groupIndex) in form.content.shot_groups" :key="group.key" :class="['group-nav-item', { active: activeGroupKey === group.key, 'has-error': hasGroupError(group) }]">
            <button type="button" class="group-select" @click="selectGroup(group)">
              <span class="group-order">{{ groupIndex + 1 }}</span>
              <span class="group-copy"><strong>{{ group.title || `章节 ${groupIndex + 1}` }}</strong><span>{{ groupDuration(group) }} 秒 · {{ group.shots.length }} 个画面</span></span>
            </button>
            <div class="group-actions">
              <button type="button" title="上移章节" :disabled="groupIndex === 0" @click="moveGroup(groupIndex, -1)"><ArrowUpIcon class="h-4 w-4" /></button>
              <button type="button" title="下移章节" :disabled="groupIndex === form.content.shot_groups.length - 1" @click="moveGroup(groupIndex, 1)"><ArrowDownIcon class="h-4 w-4" /></button>
              <button type="button" class="danger-action" title="删除章节" :disabled="form.content.shot_groups.length === 1" @click="removeGroup(groupIndex)"><TrashIcon class="h-4 w-4" /></button>
            </div>
          </article>
        </aside>

        <div v-if="activeGroup" class="group-editor" :data-group-key="activeGroup.key">
          <header class="group-editor-context">
            <div><span>当前章节</span><strong>{{ activeGroup.title || `章节 ${activeGroupIndex + 1}` }}</strong></div>
            <dl><div><dt>章节时长</dt><dd>{{ groupDuration(activeGroup) }} 秒</dd></div><div><dt>画面数量</dt><dd>{{ activeGroup.shots.length }} 个</dd></div></dl>
          </header>
          <div class="group-fields">
            <FormField label="章节名称" field-id="active-group-title" required :error="groupErrors[activeGroup.key]?.title">
              <input id="active-group-title" v-model.trim="activeGroup.title" maxlength="100" type="text" placeholder="例如：产品开场" />
            </FormField>
            <FormField label="章节说明">
              <input v-model.trim="activeGroup.note" maxlength="300" type="text" placeholder="填写这一章节的统一要求" />
            </FormField>
          </div>

          <div class="small-shot-toolbar">
            <div><h4>画面列表</h4><span>{{ activeGroup.shots.length }} 个画面 · {{ groupDuration(activeGroup) }} 秒</span></div>
            <button type="button" @click="addSmallShot(activeGroup)"><PlusIcon class="h-4 w-4" />添加画面</button>
          </div>

          <div class="small-shot-list">
            <table class="shot-table">
              <colgroup><col class="shot-col-index" /><col class="shot-col-scene" /><col class="shot-col-duration" /><col class="shot-col-subtitle" /><col class="shot-col-note" /><col class="shot-col-reference" /><col class="shot-col-actions" /></colgroup>
              <thead><tr><th>画面</th><th>画面情节</th><th>时长</th><th>字幕</th><th>制作备注</th><th>参考图</th><th><span class="sr-only">操作</span></th></tr></thead>
              <tbody>
                <tr v-for="(shot, shotIndex) in activeGroup.shots" :key="shot.key" :data-shot-key="shot.key" :class="{ 'shot-row-error': Boolean(shotErrors[shot.key]) }">
                  <td class="shot-index-cell" data-label="画面"><span>{{ String(shotIndex + 1).padStart(2, '0') }}</span></td>
                  <td class="shot-scene-cell" data-label="画面情节"><textarea v-model.trim="shot.scene" rows="3" aria-label="画面情节" placeholder="描述人物、动作、场景和镜头变化" /><small v-if="shotErrors[shot.key]?.scene">{{ shotErrors[shot.key].scene }}</small></td>
                  <td class="shot-duration-cell" data-label="时长"><input v-model.number="shot.duration_seconds" class="duration-input" min="0.1" step="0.1" type="number" aria-label="画面时长" /><small v-if="shotErrors[shot.key]?.duration">{{ shotErrors[shot.key].duration }}</small></td>
                  <td data-label="字幕"><textarea v-model.trim="shot.subtitle" rows="3" aria-label="字幕" placeholder="画面字幕或配音" /></td>
                  <td data-label="制作备注"><textarea v-model.trim="shot.note" rows="3" aria-label="制作备注" placeholder="节奏、转场等要求" /></td>
                  <td class="shot-reference-cell" data-label="参考图"><ReferenceImageUploader v-model="shot.references" :require-note="true" button-text="添加参考图" /></td>
                  <td class="shot-action-cell" data-label="操作"><button type="button" title="上移画面" :disabled="shotIndex === 0" @click="moveSmallShot(activeGroup, shotIndex, -1)"><ArrowUpIcon class="h-4 w-4" /></button><button type="button" title="下移画面" :disabled="shotIndex === activeGroup.shots.length - 1" @click="moveSmallShot(activeGroup, shotIndex, 1)"><ArrowDownIcon class="h-4 w-4" /></button><button type="button" class="danger-action" title="删除画面" :disabled="activeGroup.shots.length === 1" @click="removeSmallShot(activeGroup, shotIndex)"><TrashIcon class="h-4 w-4" /></button></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div v-if="layout === 'page'" class="mobile-storyboard">
        <section v-for="(group, groupIndex) in form.content.shot_groups" :key="group.key" class="mobile-group-section">
          <header class="mobile-group-header">
            <div class="mobile-group-heading">
              <span class="mobile-group-order">{{ groupIndex + 1 }}</span>
              <div class="mobile-group-title-field">
                <span>章节名称</span>
                <input v-model.trim="group.title" maxlength="100" type="text" :aria-label="`第 ${groupIndex + 1} 章节名称`" placeholder="例如：准备工作" />
              </div>
            </div>
            <div class="mobile-group-meta"><strong>{{ groupDuration(group) }} 秒</strong><span>{{ group.shots.length }} 个画面</span></div>
          </header>
          <div class="mobile-group-note-field">
            <span>章节说明</span>
            <input v-model.trim="group.note" maxlength="300" type="text" placeholder="填写这一章节的统一要求" />
          </div>

          <div class="mobile-shot-toolbar">
            <span>画面列表</span>
            <button type="button" @click="addSmallShot(group)"><PlusIcon class="h-4 w-4" />添加画面</button>
          </div>
          <div class="mobile-shot-table">
            <div class="mobile-shot-table-head"><span>画面</span><span>画面情节</span><span>时长</span><span>字幕</span><span>制作备注</span><span>参考图</span><span><span class="sr-only">操作</span></span></div>
            <article v-for="(shot, shotIndex) in group.shots" :key="shot.key" :data-shot-key="shot.key" class="mobile-shot-row" :class="{ 'shot-row-error': Boolean(shotErrors[shot.key]) }">
              <div class="mobile-shot-index" data-label="画面"><span>{{ String(shotIndex + 1).padStart(2, '0') }}</span></div>
              <div class="mobile-shot-field mobile-shot-scene" data-label="画面情节"><textarea v-model.trim="shot.scene" rows="3" aria-label="画面情节" placeholder="描述人物、动作、场景和镜头变化" /><small v-if="shotErrors[shot.key]?.scene">{{ shotErrors[shot.key].scene }}</small></div>
              <div class="mobile-shot-field mobile-shot-duration" data-label="时长"><input v-model.number="shot.duration_seconds" class="duration-input" min="0.1" step="0.1" type="number" aria-label="画面时长" /><small v-if="shotErrors[shot.key]?.duration">{{ shotErrors[shot.key].duration }}</small></div>
              <div class="mobile-shot-field" data-label="字幕"><textarea v-model.trim="shot.subtitle" rows="3" aria-label="字幕" placeholder="画面字幕或配音" /></div>
              <div class="mobile-shot-field" data-label="制作备注"><textarea v-model.trim="shot.note" rows="3" aria-label="制作备注" placeholder="节奏、转场等要求" /></div>
              <div class="mobile-shot-field mobile-shot-reference" data-label="参考图"><ReferenceImageUploader v-model="shot.references" :require-note="true" button-text="添加参考图" /></div>
              <div class="mobile-shot-actions" data-label="操作"><button type="button" title="上移画面" :disabled="shotIndex === 0" @click="moveSmallShot(group, shotIndex, -1)"><ArrowUpIcon class="h-4 w-4" /></button><button type="button" title="下移画面" :disabled="shotIndex === group.shots.length - 1" @click="moveSmallShot(group, shotIndex, 1)"><ArrowDownIcon class="h-4 w-4" /></button><button type="button" class="danger-action" title="删除画面" :disabled="group.shots.length === 1" @click="removeSmallShot(group, shotIndex)"><TrashIcon class="h-4 w-4" /></button></div>
            </article>
          </div>
          <div class="mobile-group-actions"><button type="button" title="上移章节" :disabled="groupIndex === 0" @click="moveGroup(groupIndex, -1)"><ArrowUpIcon class="h-4 w-4" />上移章节</button><button type="button" title="下移章节" :disabled="groupIndex === form.content.shot_groups.length - 1" @click="moveGroup(groupIndex, 1)"><ArrowDownIcon class="h-4 w-4" />下移章节</button><button type="button" class="danger-action" title="删除章节" :disabled="form.content.shot_groups.length === 1" @click="removeGroup(groupIndex)"><TrashIcon class="h-4 w-4" />删除章节</button></div>
        </section>
      </div>

    </section>

    <div class="form-actions">
      <button type="button" class="secondary-button" @click="$emit('cancel')">取消</button>
      <button type="submit" class="primary-button" :disabled="submitting">
        <ArrowPathIcon v-if="submitting" class="h-4 w-4 animate-spin" />
        {{ submitting ? '正在提交' : (initialValue ? '保存修改' : '提交需求') }}
      </button>
    </div>
  </form>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ArrowDownIcon, ArrowPathIcon, ArrowUpIcon, CheckCircleIcon, ExclamationCircleIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import ReferenceImageUploader from './ReferenceImageUploader.vue'
import FormField from './WorkRequestFormField.vue'
import WorkRequestSelect from './WorkRequestSelect.vue'

const props = defineProps({ initialValue: { type: Object, default: null }, submitting: { type: Boolean, default: false }, layout: { type: String, default: 'drawer' } })
const emit = defineEmits(['submit', 'cancel'])
const key = () => `${Date.now()}-${Math.random().toString(36).slice(2)}`
const dimensionOptions = [{ value: '16:9', label: '16:9' }, { value: '9:16', label: '9:16' }]
const resolutionOptions = [{ value: '480p', label: '480p' }, { value: '720p', label: '720p' }, { value: '1080p', label: '1080p' }]
const videoTypeOptions = [{ value: '产品宣传', label: '产品宣传' }, { value: '安装视频', label: '安装视频' }, { value: '产品广告', label: '产品广告' }]
const voiceOptions = [{ value: '男声', label: '男声' }, { value: '女声', label: '女声' }, { value: '老人', label: '老人' }, { value: '自定义', label: '自定义' }]
const errors = reactive({})
const groupErrors = reactive({})
const shotErrors = reactive({})
const formRef = ref(null)
const activeGroupKey = ref(null)

const blankSmallShot = duration => ({ key: key(), duration_seconds: duration, scene: '', subtitle: '', note: '', references: [] })
const blankGroup = (duration, index = 1) => ({ key: key(), title: `章节 ${index}`, note: '', shots: [blankSmallShot(duration)] })
const blankContent = () => ({
  duration_mode: 15,
  duration_seconds: 15,
  dimensions: '',
  resolution: '',
  video_type: '',
  voice_requirement: '',
  voice_requirement_detail: '',
  overall_requirement: '',
  shot_groups: [blankGroup(15)],
})
const form = reactive({ type: 'video', title: '', content: blankContent() })

const activeGroup = computed(() => form.content.shot_groups.find(group => group.key === activeGroupKey.value) || form.content.shot_groups[0])
const activeGroupIndex = computed(() => Math.max(0, form.content.shot_groups.findIndex(group => group.key === activeGroup.value?.key)))
const allSmallShots = computed(() => form.content.shot_groups.flatMap(group => group.shots))
const shotTotal = computed(() => Number(allSmallShots.value.reduce((total, shot) => total + Number(shot.duration_seconds || 0), 0).toFixed(1)))
const durationDifference = computed(() => Number((Number(form.content.duration_seconds || 0) - shotTotal.value).toFixed(1)))
const customDurationMinutes = computed({
  get: () => Number((Number(form.content.duration_seconds || 0) / 60).toFixed(2)),
  set: value => {
    const seconds = Number((Number(value || 0) * 60).toFixed(1))
    form.content.duration_seconds = seconds
    syncSingleSmallShotDuration(seconds)
  },
})

watch(() => props.initialValue, loadInitial, { immediate: true })

function normalizeSmallShot(shot) {
  return {
    key: key(),
    duration_seconds: Number(shot.duration_seconds || 0),
    scene: shot.scene || shot.display_requirement || '',
    subtitle: shot.subtitle || shot.dialogue || '',
    note: shot.note || '',
    references: Array.isArray(shot.references) ? shot.references : [],
  }
}

function loadInitial(value) {
  clearValidation()
  if (!value) {
    form.type = 'video'
    form.title = ''
    form.content = blankContent()
    selectGroup(form.content.shot_groups[0])
    return
  }
  form.type = 'video'
  form.title = value.title || ''
  const content = JSON.parse(JSON.stringify(value.content || {}))
  const duration = Number(content.duration_seconds || 15)
  let groups
  if (Array.isArray(content.shot_groups) && content.shot_groups.length) {
    groups = content.shot_groups.map((group, index) => ({
      key: key(),
      title: group.title || `章节 ${index + 1}`,
      note: group.note || '',
      shots: (group.shots?.length ? group.shots : [blankSmallShot(duration)]).map(normalizeSmallShot),
    }))
  } else {
    const legacyShots = Array.isArray(content.shots) && content.shots.length ? content.shots : [blankSmallShot(duration)]
    groups = [{ key: key(), title: '章节 1', note: '', shots: legacyShots.map(normalizeSmallShot) }]
  }
  form.content = {
    duration_mode: [15, 30, 60].includes(duration) ? duration : 'custom',
    duration_seconds: duration,
    dimensions: content.dimensions || '',
    resolution: content.resolution || '',
    video_type: content.video_type || '',
    voice_requirement: content.voice_requirement || '',
    voice_requirement_detail: content.voice_requirement_detail || '',
    overall_requirement: content.overall_requirement || '',
    shot_groups: groups,
  }
  selectGroup(groups[0])
}

function clearValidation() {
  Object.keys(errors).forEach(item => delete errors[item])
  Object.keys(groupErrors).forEach(item => delete groupErrors[item])
  Object.keys(shotErrors).forEach(item => delete shotErrors[item])
}
function groupDuration(group) { return Number(group.shots.reduce((total, shot) => total + Number(shot.duration_seconds || 0), 0).toFixed(1)) }
function syncSingleSmallShotDuration(seconds) { if (allSmallShots.value.length === 1) allSmallShots.value[0].duration_seconds = seconds }
function selectDuration(seconds) { form.content.duration_mode = seconds; form.content.duration_seconds = seconds; syncSingleSmallShotDuration(seconds) }
function selectCustomDuration() {
  if (form.content.duration_mode !== 'custom' && [15, 30, 60].includes(Number(form.content.duration_seconds))) {
    form.content.duration_seconds = 60
    syncSingleSmallShotDuration(60)
  }
  form.content.duration_mode = 'custom'
}
function selectGroup(group) { activeGroupKey.value = group.key }
function splitAvailableDuration(shots = allSmallShots.value) {
  if (durationDifference.value > 0) return durationDifference.value
  const source = shots.at(-1)
  const sourceDuration = Number(source?.duration_seconds || 0)
  if (source && sourceDuration > 0.1) {
    const duration = Number((sourceDuration / 2).toFixed(1))
    source.duration_seconds = Number((sourceDuration - duration).toFixed(1))
    return duration
  }
  return 0.1
}
function addGroup() {
  const group = blankGroup(splitAvailableDuration(), form.content.shot_groups.length + 1)
  form.content.shot_groups.push(group)
  selectGroup(group)
  nextTick(() => formRef.value?.querySelector(`[data-group-key="${group.key}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}
function removeGroup(index) {
  if (form.content.shot_groups.length === 1) return
  const removedDuration = groupDuration(form.content.shot_groups[index])
  const [removed] = form.content.shot_groups.splice(index, 1)
  delete groupErrors[removed.key]
  removed.shots.forEach(shot => delete shotErrors[shot.key])
  const nextGroup = form.content.shot_groups[Math.min(index, form.content.shot_groups.length - 1)]
  if (nextGroup?.shots[0]) nextGroup.shots[0].duration_seconds = Number((Number(nextGroup.shots[0].duration_seconds || 0) + removedDuration).toFixed(1))
  selectGroup(nextGroup)
}
function moveGroup(index, offset) {
  const target = index + offset
  if (target < 0 || target >= form.content.shot_groups.length) return
  const [group] = form.content.shot_groups.splice(index, 1)
  form.content.shot_groups.splice(target, 0, group)
}
function addSmallShot(group) {
  const shot = blankSmallShot(splitAvailableDuration(group.shots))
  group.shots.push(shot)
  nextTick(() => formRef.value?.querySelector(`[data-shot-key="${shot.key}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' }))
}
function removeSmallShot(group, index) {
  if (group.shots.length === 1) return
  const [removed] = group.shots.splice(index, 1)
  delete shotErrors[removed.key]
  const target = group.shots[Math.min(index, group.shots.length - 1)]
  target.duration_seconds = Number((Number(target.duration_seconds || 0) + Number(removed.duration_seconds || 0)).toFixed(1))
}
function moveSmallShot(group, index, offset) {
  const target = index + offset
  if (target < 0 || target >= group.shots.length) return
  const [shot] = group.shots.splice(index, 1)
  group.shots.splice(target, 0, shot)
}
function hasGroupError(group) { return Boolean(groupErrors[group.key] || group.shots.some(shot => shotErrors[shot.key])) }

function validate() {
  clearValidation()
  if (!form.title) errors.title = '请填写标题'
  if (!(form.content.duration_seconds > 0)) errors.duration_seconds = '请选择或填写视频总时长'
  let firstInvalidGroup = null
  form.content.shot_groups.forEach(group => {
    if (!group.title) {
      groupErrors[group.key] = { title: '请填写章节名称' }
      if (!firstInvalidGroup) {
        firstInvalidGroup = group.key
      }
    }
    group.shots.forEach(shot => {
      const shotError = {}
      if (!(shot.duration_seconds > 0)) shotError.duration = '请填写时长'
      if (!shot.scene) shotError.scene = '请填写画面情节'
      if (Object.keys(shotError).length) {
        shotErrors[shot.key] = shotError
        if (!firstInvalidGroup) {
          firstInvalidGroup = group.key
        }
      }
    })
  })
  if (durationDifference.value !== 0) errors.groups = '所有画面时长合计需等于视频总时长'
  if (firstInvalidGroup) {
    activeGroupKey.value = firstInvalidGroup
  }
  const valid = Object.keys(errors).length === 0 && Object.keys(groupErrors).length === 0 && Object.keys(shotErrors).length === 0
  if (!valid) nextTick(() => formRef.value?.querySelector('.field-error, .section-error, .has-error')?.scrollIntoView({ behavior: 'smooth', block: 'center' }))
  return valid
}

function submit() {
  if (!validate()) return
  const legacyShots = form.content.shot_groups.flatMap(group => group.shots.map(shot => ({
    duration_seconds: shot.duration_seconds,
    display_requirement: shot.scene,
    dialogue: shot.subtitle,
    references: shot.references,
  })))
  const content = {
    duration_seconds: form.content.duration_seconds,
    dimensions: form.content.dimensions,
    resolution: form.content.resolution,
    video_type: form.content.video_type,
    voice_requirement: form.content.voice_requirement,
    voice_requirement_detail: form.content.voice_requirement === '自定义' ? form.content.voice_requirement_detail : '',
    overall_requirement: form.content.overall_requirement,
    shots: legacyShots,
    shot_groups: form.content.shot_groups.map(group => ({
      title: group.title,
      note: group.note,
      shots: group.shots.map(shot => ({ duration_seconds: shot.duration_seconds, scene: shot.scene, subtitle: shot.subtitle, note: shot.note, references: shot.references })),
    })),
  }
  emit('submit', { type: 'video', title: form.title, content })
}
</script>

<style scoped>
.request-form { display: flex; min-height: 100%; flex-direction: column; background: #fff; }
.request-section { padding: 26px 30px; }
.request-settings { display: grid; grid-template-columns: minmax(0, 1fr) 360px; gap: 24px; align-items: start; }
.section-divider { height: 1px; margin: 0 30px; background: #ededed; }
.request-form :deep(.field-label label) { color: #262626; font-size: 14px; font-weight: 700; }
.request-form :deep(.field-label small) { display: none; }
.request-form :deep(input:not([type='file'])), .request-form :deep(textarea) { width: 100%; border: 1px solid #d4d4d4; border-radius: 6px; color: #262626; background: #fff; outline: none; transition: border-color .15s ease, box-shadow .15s ease; }
.request-form :deep(input:not([type='file'])) { height: 44px; padding: 0 13px; }
.request-form :deep(textarea) { resize: vertical; padding: 11px 13px; font-size: 14px; line-height: 1.6; }
.request-form :deep(input:focus), .request-form :deep(textarea:focus) { border-color: #525252; box-shadow: 0 0 0 3px rgb(23 23 23 / 6%); }
.request-form :deep(input::placeholder), .request-form :deep(textarea::placeholder) { color: #b0b0b0; }
.number-field { position: relative; display: block; }
.number-field input { padding-right: 45px !important; }
.number-field span { position: absolute; top: 50%; right: 12px; color: #737373; font-size: 13px; transform: translateY(-50%); }
.duration-picker { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 7px; }
.duration-picker > button { display: inline-flex; height: 44px; align-items: center; justify-content: center; gap: 3px; border: 1px solid #d4d4d4; border-radius: 6px; color: #525252; background: #fff; font-size: 13px; font-weight: 650; line-height: 1; }
.duration-picker > button strong { color: #262626; font-size: 15px; line-height: 1; }.duration-picker > button span { font-size: 12px; line-height: 1; }
.duration-picker > button:hover { border-color: #8a8a8a; }.duration-picker > button.active { border-color: #171717; color: #fff; background: #171717; }.duration-picker > button.active strong { color: #fff; }.duration-picker > .custom-duration-button { align-items: center; }
.custom-duration-row { margin-top: 8px; }
.storyboard-section { display: flex; flex-direction: column; gap: 14px; }
.storyboard-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; }.storyboard-title { display: flex; align-items: center; gap: 12px; }.storyboard-title h3 { margin: 0; color: #171717; font-size: 17px; font-weight: 750; }
.duration-status { display: inline-flex; height: 28px; align-items: center; gap: 5px; border-radius: 5px; padding: 0 8px; color: #166534; background: #f0fdf4; font-size: 13px; white-space: nowrap; }.duration-status.invalid { color: #9a3412; background: #fff7ed; }.duration-status strong { font-weight: 700; }
.add-group-button, .small-shot-toolbar button { display: inline-flex; height: 38px; align-items: center; justify-content: center; gap: 6px; border: 1px solid #d4d4d4; border-radius: 6px; padding: 0 12px; color: #262626; background: #fff; font-size: 13px; font-weight: 700; }.add-group-button:hover, .small-shot-toolbar button:hover { border-color: #8a8a8a; background: #fafafa; }
.section-error { margin: -2px 0 0; color: #dc2626; font-size: 13px; }
.storyboard-workspace { display: grid; min-height: 570px; grid-template-columns: 260px minmax(0, 1fr); overflow: hidden; border: 1px solid #dcdcdc; border-radius: 7px; background: #fff; }
.mobile-storyboard { display: none; }
.group-sidebar { display: flex; min-width: 0; flex-direction: column; gap: 7px; border-right: 1px solid #e5e5e5; padding: 12px; background: #fafafa; }
.group-nav-item { overflow: hidden; border: 1px solid transparent; border-radius: 6px; background: transparent; }.group-nav-item:hover { background: #f1f1f1; }.group-nav-item.active { border-color: #d4d4d4; background: #fff; box-shadow: 0 2px 8px rgb(0 0 0 / 4%); }.group-nav-item.has-error { border-color: #fca5a5; }
.group-select { display: flex; width: 100%; min-width: 0; align-items: center; gap: 10px; padding: 12px 10px 8px; text-align: left; }.group-order { display: grid; width: 26px; height: 26px; flex: 0 0 auto; place-items: center; border-radius: 5px; color: #525252; background: #e5e5e5; font-size: 12px; font-weight: 800; }.group-nav-item.active .group-order { color: #fff; background: #171717; }.group-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 3px; }.group-copy strong, .group-copy span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.group-copy strong { color: #262626; font-size: 13px; }.group-copy span { color: #8a8a8a; font-size: 11px; }
.group-actions, .small-shot-actions { display: flex; justify-content: flex-end; gap: 2px; padding: 0 7px 7px; }.group-actions button, .small-shot-actions button { display: grid; width: 30px; height: 30px; place-items: center; border-radius: 5px; color: #737373; }.group-actions button:hover:not(:disabled), .small-shot-actions button:hover:not(:disabled) { color: #171717; background: #e5e5e5; }.group-actions .danger-action:hover:not(:disabled), .small-shot-actions .danger-action:hover:not(:disabled) { color: #b91c1c; background: #fee2e2; }.group-actions button:disabled, .small-shot-actions button:disabled { cursor: not-allowed; opacity: .25; }
.group-editor { min-width: 0; padding: 22px; }.group-fields { display: grid; grid-template-columns: minmax(220px, .8fr) minmax(0, 1.2fr); gap: 16px; padding-bottom: 22px; border-bottom: 1px solid #ededed; }
.small-shot-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px 0 12px; }.small-shot-toolbar h4 { margin: 0; color: #262626; font-size: 15px; font-weight: 750; }.small-shot-toolbar button { height: 34px; padding: 0 10px; }
.small-shot-list { display: flex; flex-direction: column; gap: 9px; }.small-shot-card { overflow: hidden; border: 1px solid #dedede; border-radius: 7px; }.small-shot-card.open { border-color: #a3a3a3; box-shadow: 0 4px 16px rgb(0 0 0 / 4%); }.small-shot-card.has-error { border-color: #fca5a5; }
.small-shot-head { display: flex; min-height: 54px; align-items: center; background: #fafafa; }.small-shot-card.open .small-shot-head { border-bottom: 1px solid #e5e5e5; }.small-shot-toggle { display: flex; min-width: 0; height: 54px; flex: 1; align-items: center; gap: 10px; padding: 0 10px 0 14px; text-align: left; }.small-shot-toggle strong { flex: 0 0 auto; color: #262626; font-size: 13px; }.scene-preview { overflow: hidden; flex: 1; color: #737373; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }.small-shot-duration { flex: 0 0 auto; color: #404040; font-size: 13px; font-weight: 700; }.small-shot-toggle > svg { flex: 0 0 auto; color: #737373; }.small-shot-actions { flex: 0 0 auto; padding: 0 8px 0 0; }
.form-actions { position: sticky; bottom: 0; z-index: 5; display: flex; justify-content: flex-end; gap: 10px; margin-top: auto; border-top: 1px solid #e5e5e5; padding: 16px 30px; background: rgb(255 255 255 / 96%); backdrop-filter: blur(8px); }.primary-button, .secondary-button { display: inline-flex; height: 42px; align-items: center; justify-content: center; gap: 7px; border-radius: 6px; padding: 0 18px; font-size: 14px; font-weight: 700; }.primary-button { color: #fff; background: #171717; }.primary-button:hover:not(:disabled) { background: #404040; }.primary-button:disabled { cursor: not-allowed; opacity: .55; }.secondary-button { border: 1px solid #d4d4d4; color: #404040; background: #fff; }.secondary-button:hover { background: #fafafa; }
@media (max-width: 900px) { .storyboard-workspace { grid-template-columns: 220px minmax(0, 1fr); } }
@media (max-width: 720px) { .request-section { padding: 22px 18px; }.request-settings { grid-template-columns: 1fr; gap: 20px; }.section-divider { margin: 0 18px; }.storyboard-toolbar { align-items: stretch; flex-direction: column; }.storyboard-title { justify-content: space-between; }.add-group-button { width: 100%; }.storyboard-workspace { min-height: 0; grid-template-columns: 1fr; overflow: visible; }.group-sidebar { overflow-x: auto; flex-direction: row; border-right: 0; border-bottom: 1px solid #e5e5e5; }.group-nav-item { min-width: 200px; flex: 0 0 auto; }.group-editor { padding: 18px; }.group-fields { grid-template-columns: 1fr; }.form-actions { padding-right: 18px; padding-left: 18px; } }
@media (max-width: 430px) { .duration-picker { grid-template-columns: repeat(2, minmax(0, 1fr)); }.storyboard-title { align-items: flex-start; flex-direction: column; }.group-editor { padding: 14px 12px; }.small-shot-toolbar { align-items: stretch; flex-direction: column; }.small-shot-toolbar button { width: 100%; }.small-shot-toggle > svg { display: none; }.small-shot-actions { gap: 0; padding-right: 5px; }.small-shot-actions button { width: 28px; }.small-shot-editor { padding: 15px 12px; }.form-actions .secondary-button, .form-actions .primary-button { flex: 1; } }
/* Taste-skill: expose hierarchy and keep controls quiet in a dense editor. */
.request-form { color: #292826; background: #fff; }
.request-section { padding: 28px 32px; }.request-settings { grid-template-columns: minmax(0, 1fr) 390px; gap: 28px; }
.request-form :deep(.field-label label) { color: #353330; font-size: 13px; font-weight: 700; }
.request-form :deep(input:not([type='file'])), .request-form :deep(textarea) { border-color: #d8d5cf; color: #353330; transition: border-color .18s ease, box-shadow .18s ease, background .18s ease; }
.request-form :deep(input:focus), .request-form :deep(textarea:focus) { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.duration-picker > button { border-color: #d8d5cf; color: #67635d; transition: border-color .18s ease, background .18s ease, color .18s ease, transform .12s ease; }.duration-picker > button:hover { border-color: #aaa59d; background: #faf9f7; }.duration-picker > button.active { border-color: #2b2926; background: #2b2926; }.duration-picker > button:active, .add-group-button:active, .small-shot-toolbar button:active { transform: translateY(1px); }
.storyboard-section { gap: 16px; }.storyboard-title h3 { color: #252421; font-size: 18px; }.duration-status { color: #486d4c; background: #ebf1ea; font-variant-numeric: tabular-nums; }.duration-status.invalid { color: #98523f; background: #f8ebe7; }
.add-group-button, .small-shot-toolbar button { border-color: #d8d5cf; color: #373532; transition: border-color .18s ease, background .18s ease, transform .12s ease; }.add-group-button:hover, .small-shot-toolbar button:hover { border-color: #aaa59d; background: #faf9f7; }
.storyboard-workspace { min-height: 590px; grid-template-columns: 270px minmax(0, 1fr); border-color: #dfdcd7; border-radius: 8px; }
.group-sidebar { gap: 5px; border-color: #e4e1dc; padding: 10px; background: #f7f6f3; }
.group-nav-item { border-radius: 6px; transition: background .16s ease, border-color .16s ease; }.group-nav-item:hover { background: #efede9; }.group-nav-item.active { border-color: #d3cfc8; box-shadow: none; }.group-order { color: #6d6963; background: #e9e6e1; }.group-nav-item.active .group-order { background: #2b2926; }.group-copy strong { color: #353330; }.group-copy span { color: #8c8780; font-variant-numeric: tabular-nums; }
.group-editor { padding: 0 24px 26px; }.group-editor-context { display: flex; min-height: 72px; align-items: center; justify-content: space-between; gap: 18px; margin: 0 -24px 22px; border-bottom: 1px solid #ebe9e5; padding: 0 24px; background: #fcfcfb; }.group-editor-context > div { display: flex; min-width: 0; flex-direction: column; gap: 4px; }.group-editor-context span, .group-editor-context dt { color: #928d86; font-size: 10px; font-weight: 650; }.group-editor-context strong { overflow: hidden; color: #2d2b28; font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }.group-editor-context dl { display: flex; gap: 24px; margin: 0; }.group-editor-context dl > div { display: flex; flex-direction: column; gap: 4px; text-align: right; }.group-editor-context dd { margin: 0; color: #45423e; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-weight: 700; font-variant-numeric: tabular-nums; }
.group-fields { border-color: #ebe9e5; padding-bottom: 24px; }.small-shot-toolbar { padding: 20px 0 12px; }.small-shot-toolbar h4 { color: #353330; }
.small-shot-list { gap: 8px; }.small-shot-card { border-color: #dfdcd7; border-radius: 7px; transition: border-color .18s ease, box-shadow .18s ease; }.small-shot-card.open { border-color: #bdb8b0; box-shadow: 0 3px 14px rgb(65 58 49 / 4%); }.small-shot-head { background: #faf9f7; }.small-shot-card.open .small-shot-head { border-color: #e7e4df; }.shot-order { display: grid; width: 24px; height: 24px; flex: 0 0 auto; place-items: center; border-radius: 4px; color: #77736d; background: #ece9e4; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; }.small-shot-toggle strong { color: #353330; }.scene-preview { color: #77736d; }.reference-count { flex: 0 0 auto; color: #8a6a51; font-size: 10px; }.small-shot-duration { color: #45423e; font-variant-numeric: tabular-nums; }
.small-shot-editor { padding: 20px; background: #fff; }.form-actions { border-color: #e4e1dc; padding: 14px 32px; background: rgb(255 255 255 / 97%); }.primary-button { background: #2b2926; transition: background .18s ease, transform .12s ease; }.primary-button:hover:not(:disabled) { background: #45413c; }.secondary-button { border-color: #d8d5cf; }.primary-button:active:not(:disabled), .secondary-button:active { transform: translateY(1px); }.primary-button:focus-visible, .secondary-button:focus-visible, .add-group-button:focus-visible, .small-shot-toolbar button:focus-visible, .group-actions button:focus-visible, .small-shot-actions button:focus-visible { outline: 2px solid #736c63; outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { .duration-picker > button, .add-group-button, .small-shot-toolbar button, .group-nav-item, .small-shot-card, .primary-button { transition: none; } }
@media (max-width: 720px) { .request-section { padding: 22px 17px; }.request-settings { grid-template-columns: 1fr; gap: 20px; }.storyboard-workspace { min-height: 0; grid-template-columns: 1fr; }.group-editor { padding: 0 14px 20px; }.group-editor-context { margin: 0 -14px 18px; padding: 0 14px; }.group-editor-context dl { gap: 14px; }.reference-count { display: none; }.form-actions { padding-right: 17px; padding-left: 17px; } }
@media (max-width: 480px) { .group-editor-context { align-items: stretch; flex-direction: column; justify-content: center; padding-top: 12px; padding-bottom: 12px; }.group-editor-context dl { justify-content: flex-start; }.group-editor-context dl > div { text-align: left; } }
/* Shot timeline */
.small-shot-toolbar { padding: 22px 0 13px; }
.small-shot-toolbar > div { display: flex; min-width: 0; align-items: baseline; gap: 9px; }
.small-shot-toolbar > div > span { color: #928d86; font-size: 11px; font-variant-numeric: tabular-nums; }
.small-shot-list { gap: 0; }
.small-shot-card { position: relative; display: grid; grid-template-columns: 48px minmax(0, 1fr); overflow: visible; border: 0; border-radius: 0; background: transparent; box-shadow: none; }
.small-shot-card::before { position: absolute; top: 0; bottom: 0; left: 23px; width: 1px; background: #ddd9d2; content: ''; }
.small-shot-card:first-child::before { top: 26px; }
.small-shot-card:last-child::before { bottom: calc(100% - 27px); }
.small-shot-card.open, .small-shot-card.has-error { border: 0; box-shadow: none; }
.shot-rail { position: relative; z-index: 1; display: flex; justify-content: center; padding-top: 16px; }
.shot-rail span { display: grid; width: 28px; height: 28px; place-items: center; border: 1px solid #d4d0c9; border-radius: 5px; color: #77736d; background: #f7f6f3; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; font-weight: 700; transition: border-color .18s ease, color .18s ease, background .18s ease; }
.small-shot-card.open .shot-rail span { border-color: #2b2926; color: #fff; background: #2b2926; }
.small-shot-card.has-error:not(.open) .shot-rail span { border-color: #d4a396; color: #98523f; background: #f9eeea; }
.shot-main { min-width: 0; margin-bottom: 10px; overflow: hidden; border: 1px solid #dfdcd7; border-radius: 8px; background: #fff; transition: border-color .18s ease, box-shadow .18s ease; }
.small-shot-card.open .shot-main { border-color: #bdb8b0; box-shadow: 0 4px 18px rgb(65 58 49 / 4%); }
.small-shot-card.has-error .shot-main { border-color: #e0b1a6; }
.small-shot-head { min-height: 68px; border: 0; background: #faf9f7; }
.small-shot-card.open .small-shot-head { border-bottom: 1px solid #e7e4df; background: #f7f6f3; }
.small-shot-toggle { height: auto; min-height: 68px; gap: 12px; padding: 10px 12px 10px 16px; }
.shot-summary { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 5px; }
.shot-summary strong { color: #353330; font-size: 13px; font-weight: 750; }
.scene-preview { display: block; overflow: hidden; color: #6f6a64; font-size: 12px; line-height: 1.45; text-overflow: ellipsis; white-space: nowrap; }
.scene-preview.empty { color: #aaa59d; }
.shot-meta { display: flex; flex: 0 0 auto; align-items: center; justify-content: flex-end; gap: 6px; }
.small-shot-duration, .reference-count, .shot-error-mark { display: inline-flex; height: 24px; align-items: center; border-radius: 4px; padding: 0 7px; font-size: 10px; font-weight: 700; white-space: nowrap; }
.small-shot-duration { color: #4d4944; background: #ece9e4; font-variant-numeric: tabular-nums; }
.reference-count { color: #7d5d45; background: #f4ebe4; }
.shot-error-mark { color: #98523f; background: #f8e9e6; }
.shot-chevron { flex: 0 0 auto; color: #8c8780; }
.small-shot-actions { gap: 1px; border-left: 1px solid #e7e4df; padding: 0 9px; }
.small-shot-actions button { width: 29px; height: 29px; color: #858078; }
.small-shot-actions button:hover:not(:disabled) { color: #353330; background: #ece9e4; }
.small-shot-actions .danger-action:hover:not(:disabled) { color: #98523f; background: #f8e9e6; }
.small-shot-editor { padding: 20px 22px 22px; background: #fff; }
.shot-primary-fields { display: grid; grid-template-columns: minmax(0, 1fr) 130px; gap: 18px; align-items: start; }
.shot-secondary-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin-top: 18px; padding-top: 18px; border-top: 1px solid #efede9; }
.shot-reference-field { margin-top: 18px; padding-top: 18px; border-top: 1px solid #efede9; }
.scene-field textarea { min-height: 132px; }
.duration-field .number-field input { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-variant-numeric: tabular-nums; }
@media (max-width: 900px) {
  .small-shot-toggle { gap: 9px; padding-left: 13px; }.reference-count { display: none; }.small-shot-actions { padding: 0 5px; }.shot-primary-fields { grid-template-columns: minmax(0, 1fr) 112px; }
}
@media (max-width: 720px) {
  .small-shot-toolbar > div > span { display: none; }.small-shot-card { grid-template-columns: 38px minmax(0, 1fr); }.small-shot-card::before { left: 18px; }.shot-rail span { width: 26px; height: 26px; }.shot-meta { display: none; }.small-shot-head { align-items: stretch; flex-direction: column; }.small-shot-toggle { min-height: 64px; }.small-shot-actions { justify-content: flex-end; border-top: 1px solid #ebe8e3; border-left: 0; padding: 5px 8px; background: #faf9f7; }.shot-primary-fields { grid-template-columns: 1fr; }.duration-field { max-width: 180px; }.shot-secondary-fields { grid-template-columns: 1fr; }
}
@media (max-width: 430px) {
  .small-shot-card { grid-template-columns: 32px minmax(0, 1fr); }.small-shot-card::before { left: 15px; }.shot-rail { padding-top: 14px; }.shot-rail span { width: 24px; height: 24px; font-size: 9px; }.shot-main { border-radius: 7px; }.small-shot-editor { padding: 17px 14px 19px; }.small-shot-toggle { padding-right: 10px; padding-left: 12px; }.small-shot-toggle > svg { display: block; }
}
/* Full-page editor */
.page-layout { height: 100%; min-height: 0; overflow: hidden; border: 1px solid #e5e5e5; border-radius: 8px; background: #fff; }
.page-layout .request-section { padding: 18px 32px; }
.page-layout .request-settings { grid-template-columns: minmax(0, 1.5fr) minmax(390px, .7fr); gap: 48px; }
.page-layout .request-settings :deep(.field-label) { gap: 7px; }
.page-layout .storyboard-section { min-height: 0; flex: 1; gap: 10px; padding-top: 14px; padding-bottom: 12px; }
.page-layout .storyboard-toolbar { min-height: 38px; }
.page-layout .storyboard-title h3 { font-size: 16px; }
.page-layout .storyboard-workspace { min-height: 0; flex: 1; grid-template-columns: 230px minmax(0, 1fr); }
.page-layout .group-sidebar { overflow-x: hidden; overflow-y: auto; padding: 10px; scrollbar-width: thin; }
.page-layout .group-nav-item { border-radius: 5px; }
.page-layout .group-select { gap: 8px; padding: 8px 8px 5px; }
.page-layout .group-order { width: 22px; height: 22px; border-radius: 4px; font-size: 10px; }
.page-layout .group-copy { gap: 2px; }
.page-layout .group-copy strong { font-size: 12px; }
.page-layout .group-copy span { font-size: 10px; }
.page-layout .group-actions { padding: 0 5px 5px; }
.page-layout .group-actions button { width: 26px; height: 25px; }
.page-layout .group-editor { display: flex; min-height: 0; flex-direction: column; overflow: hidden; padding: 0 32px 20px; }
.page-layout .group-editor-context { min-height: 64px; margin-right: -32px; margin-left: -32px; padding-right: 32px; padding-left: 32px; }
.page-layout .group-fields { flex: 0 0 auto; grid-template-columns: minmax(280px, .65fr) minmax(420px, 1.35fr); gap: 24px; padding-bottom: 16px; }
.page-layout .small-shot-toolbar { flex: 0 0 auto; padding-top: 14px; }
.page-layout .small-shot-list { min-height: 0; flex: 1; overflow-y: auto; overscroll-behavior: contain; padding-right: 8px; scrollbar-gutter: stable; }
.page-layout .small-shot-editor { padding: 20px 24px 22px; }
.page-layout .shot-primary-fields { grid-template-columns: minmax(0, 1fr) 150px; gap: 24px; }
.page-layout .shot-secondary-fields { gap: 24px; }
.page-layout .scene-field textarea { min-height: 150px; }
.page-layout .form-actions { position: static; flex: 0 0 auto; border-radius: 0 0 8px 8px; padding: 11px 32px; }
@media (max-width: 1100px) {
  .page-layout .request-settings { grid-template-columns: minmax(0, 1fr) 360px; gap: 28px; }
  .page-layout .storyboard-workspace { grid-template-columns: 210px minmax(0, 1fr); }
  .page-layout .group-editor { padding-right: 22px; padding-left: 22px; }
  .page-layout .group-editor-context { margin-right: -22px; margin-left: -22px; padding-right: 22px; padding-left: 22px; }
  .page-layout .group-fields { grid-template-columns: minmax(220px, .8fr) minmax(0, 1.2fr); gap: 18px; }
}
@media (max-width: 720px) {
  .page-layout { border-right: 0; border-left: 0; border-radius: 0; }
  .page-layout .request-section { padding: 22px 17px; }
  .page-layout .request-settings, .page-layout .group-fields { grid-template-columns: 1fr; gap: 20px; }
  .page-layout .storyboard-workspace { min-height: 0; grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); }
  .page-layout .group-sidebar { display: none; }
  .page-layout .group-editor { padding: 0 14px 20px; }
  .page-layout .group-editor-context { margin-right: -14px; margin-left: -14px; padding-right: 14px; padding-left: 14px; }
  .page-layout .form-actions { border-radius: 0; padding-right: 17px; padding-left: 17px; }
}
/* Spreadsheet-like scene editor */
.page-layout .small-shot-list { display: block; overflow: auto; padding-right: 0; }
.shot-table { width: 100%; min-width: 1040px; border-collapse: separate; border-spacing: 0; table-layout: fixed; }
.shot-col-index { width: 70px; }.shot-col-scene { width: 27%; }.shot-col-duration { width: 112px; }.shot-col-subtitle { width: 18%; }.shot-col-note { width: 18%; }.shot-col-reference { width: 210px; }.shot-col-actions { width: 104px; }
.shot-table th { position: sticky; top: 0; z-index: 3; height: 38px; border-bottom: 1px solid #dcd8d1; padding: 0 10px; color: #77736d; background: #f4f2ee; font-size: 11px; font-weight: 750; text-align: left; }
.shot-table th:first-child { border-radius: 6px 0 0 0; padding-left: 14px; }.shot-table th:last-child { border-radius: 0 6px 0 0; }
.shot-table td { border-bottom: 1px solid #ebe8e3; padding: 10px; background: #fff; vertical-align: top; }.shot-table tbody tr:hover td { background: #fcfbf9; }.shot-table tbody tr:last-child td { border-bottom: 0; }
.shot-table textarea, .shot-table input { width: 100%; border: 1px solid #dedbd5; border-radius: 5px; color: #353330; background: #fff; outline: none; transition: border-color .16s ease, box-shadow .16s ease, background .16s ease; }
.shot-table textarea { min-height: 72px; resize: vertical; padding: 9px 10px; font-size: 12px; line-height: 1.55; }.shot-table input { height: 36px; padding: 0 9px; font-size: 12px; }
.shot-table textarea:focus, .shot-table input:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }.shot-table textarea::placeholder, .shot-table input::placeholder { color: #aaa59d; }
.shot-index-cell { padding-top: 18px !important; text-align: center; }.shot-index-cell span { display: inline-grid; width: 30px; height: 26px; place-items: center; border: 1px solid #dedbd5; border-radius: 5px; color: #77736d; background: #f7f6f3; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; font-weight: 750; }
.shot-duration-cell .number-field { display: grid; width: 100%; height: 36px; grid-template-columns: minmax(0, 1fr) 30px; overflow: hidden; border: 1px solid #dedbd5; border-radius: 5px; background: #fff; transition: border-color .16s ease, box-shadow .16s ease; }
.shot-duration-cell .number-field:focus-within { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.page-layout .shot-duration-cell .number-field input { min-width: 0; height: 34px; border: 0; border-radius: 0; padding: 0 6px 0 8px !important; background: transparent; box-shadow: none; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-variant-numeric: tabular-nums; text-align: right; }
.shot-duration-cell .number-field span { position: static; display: flex; align-items: center; justify-content: center; border-left: 1px solid #ebe8e3; color: #77736d; background: #f7f6f3; font-size: 11px; line-height: 1; transform: none; }
.shot-table small { display: block; margin-top: 5px; color: #a04f40; font-size: 10px; line-height: 1.4; }.shot-row-error td { background: #fffaf8; }.shot-row-error .shot-index-cell span { border-color: #dfb6ab; color: #98523f; background: #f9eeea; }
.shot-reference-cell :deep(.reference-uploader) { gap: 7px; }.shot-reference-cell :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }.shot-reference-cell :deep(.reference-preview) { border-radius: 4px; }.shot-reference-cell :deep(.reference-item input) { height: 28px; margin-top: 4px; padding: 0 6px; font-size: 10px; }.shot-reference-cell :deep(.reference-remove) { top: 3px; right: 3px; width: 20px; height: 20px; }.shot-reference-cell :deep(.reference-remove svg) { width: 12px; height: 12px; }.shot-reference-cell :deep(.upload-trigger) { height: 34px; gap: 5px; font-size: 11px; }.shot-reference-cell :deep(.upload-trigger svg) { width: 14px; height: 14px; }
.shot-action-cell { display: flex; align-items: flex-start; gap: 2px; padding-top: 14px !important; }.shot-action-cell button { display: grid; width: 27px; height: 27px; place-items: center; border-radius: 4px; color: #858078; }.shot-action-cell button:hover:not(:disabled) { color: #353330; background: #ece9e4; }.shot-action-cell .danger-action:hover:not(:disabled) { color: #98523f; background: #f8e9e6; }.shot-action-cell button:disabled { cursor: not-allowed; opacity: .25; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
@media (max-width: 900px) {
  .page-layout .small-shot-list { overflow-x: hidden; }
  .shot-table, .shot-table tbody { display: block; width: 100%; min-width: 0; }
  .shot-table colgroup, .shot-table thead { display: none; }
  .shot-table tbody { display: flex; flex-direction: column; gap: 10px; padding: 1px; }
  .shot-table tbody tr { display: grid; grid-template-columns: 54px minmax(0, 1.4fr) 112px minmax(190px, .8fr); grid-template-areas: 'index scene duration actions' 'index subtitle note reference'; overflow: hidden; border: 1px solid #dedbd5; border-radius: 7px; background: #fff; }
  .shot-table td { border-bottom: 0; padding: 10px; background: #fff; }.shot-table td + td { border-left: 1px solid #efede9; }.shot-table tr:hover td { background: #fff; }
  .shot-table td::before { display: block; margin-bottom: 6px; color: #8c8780; content: attr(data-label); font-size: 10px; font-weight: 700; }
  .shot-index-cell { grid-area: index; padding: 14px 8px !important; background: #f7f6f3 !important; }.shot-index-cell::before { text-align: center; }.shot-scene-cell { grid-area: scene; }.shot-duration-cell { grid-area: duration; }.shot-table td:nth-child(4) { grid-area: subtitle; border-top: 1px solid #efede9; }.shot-table td:nth-child(5) { grid-area: note; border-top: 1px solid #efede9; }.shot-reference-cell { grid-area: reference; border-top: 1px solid #efede9; }.shot-action-cell { grid-area: actions; align-items: flex-end; justify-content: flex-end; flex-direction: row; padding: 10px !important; }
  .shot-table textarea { min-height: 64px; }.shot-reference-cell :deep(.reference-grid) { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 760px) {
  .page-layout .small-shot-list { margin-right: -14px; margin-left: -14px; padding: 0 14px; }
  .shot-table tbody tr { grid-template-columns: 46px minmax(0, 1fr); grid-template-areas: 'index scene' 'index duration' 'index subtitle' 'index note' 'index reference' 'index actions'; }
  .shot-table td + td { border-top: 1px solid #efede9; border-left: 0; }.shot-scene-cell { border-top: 0 !important; }.shot-index-cell { grid-row: 1 / span 6; }
  .shot-action-cell { align-items: center; justify-content: flex-start; }.shot-action-cell::before { margin: 0 auto 0 0 !important; }.shot-reference-cell :deep(.reference-grid) { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 480px) {
  .shot-reference-cell :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .page-layout .storyboard-title { align-items: flex-start; gap: 7px; }.page-layout .duration-status { height: auto; min-height: 26px; flex-wrap: wrap; padding-top: 4px; padding-bottom: 4px; }
}
/* Mobile editor follows the same readable hierarchy as the detail view. */
@media (max-width: 720px) {
  .page-layout .storyboard-workspace { border-color: #e4e1dc; background: #faf9f7; }
  .page-layout .group-editor { background: #fff; }
  .page-layout .group-editor-context { min-height: 0; align-items: flex-start; margin-bottom: 0; border-bottom: 1px solid #ebe9e5; padding-top: 16px; padding-bottom: 16px; background: #fff; }
  .page-layout .group-editor-context > div { gap: 5px; }
  .page-layout .group-editor-context span, .page-layout .group-editor-context dt { font-size: 10px; }
  .page-layout .group-editor-context strong { font-size: 15px; }
  .page-layout .group-editor-context dl { gap: 18px; }
  .page-layout .group-fields { gap: 14px; padding-top: 16px; padding-bottom: 18px; }
  .page-layout .small-shot-toolbar { margin: 0; border-top: 1px solid #ebe9e5; padding-top: 16px; }
  .page-layout .small-shot-list { overflow-x: hidden; overflow-y: auto; padding-right: 0; }
  .page-layout .shot-table tbody { gap: 12px; padding: 0; }
  .page-layout .shot-table tbody tr { border-color: #e4e1dc; border-radius: 7px; }
  .page-layout .shot-table td { padding: 11px 12px; }
  .page-layout .shot-table textarea { min-height: 82px; }
  .page-layout .shot-table td::before { margin-bottom: 6px; color: #96918a; font-size: 9px; letter-spacing: .02em; }
  .page-layout .shot-index-cell { padding: 13px 7px !important; }
  .page-layout .shot-action-cell { padding-top: 9px !important; padding-bottom: 9px !important; background: #faf9f7 !important; }
}
@media (max-width: 900px) {
  .page-layout { height: 100%; overflow-x: hidden; overflow-y: auto; }
  .page-layout .request-settings { grid-template-columns: 1fr; gap: 18px; }
  .page-layout .storyboard-workspace { display: none; }
  .page-layout .mobile-storyboard { display: flex; flex-direction: column; gap: 0; }
  .page-layout .mobile-group-section { border-top: 1px solid #e4e1dc; padding: 25px 0 28px; }
  .page-layout .mobile-group-section:first-child { border-top: 0; padding-top: 0; }
  .page-layout .mobile-group-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; }
  .page-layout .mobile-group-heading { display: flex; min-width: 0; align-items: flex-start; gap: 12px; }
  .page-layout .mobile-group-order { display: grid; width: 38px; height: 38px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #fff; background: #2b2926; font-size: 16px; font-weight: 800; }
  .page-layout .mobile-group-title-field, .page-layout .mobile-group-note-field { display: flex; min-width: 0; flex-direction: column; gap: 5px; }
  .page-layout .mobile-group-title-field span, .page-layout .mobile-group-note-field > span { color: #96918a; font-size: 10px; font-weight: 700; }
  .page-layout .mobile-group-title-field input { width: min(360px, 100%); height: 34px; border: 0; border-bottom: 1px solid #d8d5cf; border-radius: 0; padding: 0 0 4px; color: #353330; background: transparent; font-size: 19px; font-weight: 750; outline: none; }
  .page-layout .mobile-group-title-field input:focus { border-color: #77736d; box-shadow: none; }
  .page-layout .mobile-group-meta { display: flex; flex: 0 0 auto; align-items: baseline; gap: 7px; padding-top: 8px; color: #77736d; white-space: nowrap; }
  .page-layout .mobile-group-meta strong { color: #4d4944; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 14px; font-variant-numeric: tabular-nums; }
  .page-layout .mobile-group-meta span { font-size: 12px; }
  .page-layout .mobile-group-note-field { margin: 13px 0 0 50px; }
  .page-layout .mobile-group-note-field input { height: 32px; border: 0; border-bottom: 1px solid #ebe9e5; border-radius: 0; padding: 0; color: #77736d; background: transparent; font-size: 13px; outline: none; }
  .page-layout .mobile-group-note-field input:focus { border-color: #aaa59d; box-shadow: none; }
  .page-layout .mobile-shot-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 20px; border-bottom: 1px solid #e4e1dc; padding-bottom: 9px; }
  .page-layout .mobile-shot-toolbar > span { color: #4d4944; font-size: 13px; font-weight: 750; }
  .page-layout .mobile-shot-toolbar button, .page-layout .mobile-add-group, .page-layout .mobile-group-actions button { display: inline-flex; height: 32px; align-items: center; justify-content: center; gap: 5px; border: 1px solid #d8d5cf; border-radius: 5px; padding: 0 9px; color: #5f5a54; background: #fff; font-size: 11px; font-weight: 700; }
  .page-layout .mobile-shot-toolbar button:hover, .page-layout .mobile-add-group:hover, .page-layout .mobile-group-actions button:hover { border-color: #aaa59d; background: #faf9f7; }
  .page-layout .mobile-shot-table { margin-top: 13px; }
  .page-layout .mobile-shot-table-head { display: none; }
  .page-layout .mobile-shot-row { display: grid; grid-template-columns: 48px minmax(0, 1fr) 82px; grid-template-areas: 'index scene duration' 'index subtitle subtitle' 'index note note' 'index reference reference' 'index actions actions'; overflow: hidden; border: 1px solid #e4e1dc; border-radius: 7px; background: #fff; }
  .page-layout .mobile-shot-row + .mobile-shot-row { margin-top: 10px; }
  .page-layout .mobile-shot-row > * { min-width: 0; padding: 11px 10px; }
  .page-layout .mobile-shot-row > *::before { display: block; margin-bottom: 6px; color: #96918a; content: attr(data-label); font-size: 9px; font-weight: 700; }
  .page-layout .mobile-shot-index { grid-area: index; background: #f7f6f3; text-align: center; }
  .page-layout .mobile-shot-index::before { text-align: center; }
  .page-layout .mobile-shot-index > span { display: inline-grid; width: 28px; height: 26px; place-items: center; border: 1px solid #dedbd5; border-radius: 5px; color: #77736d; background: #fff; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; font-weight: 750; }
  .page-layout .mobile-shot-scene { grid-area: scene; }
  .page-layout .mobile-shot-duration { grid-area: duration; }
  .page-layout .mobile-shot-row > :nth-child(4) { grid-area: subtitle; border-top: 1px solid #efede9; }
  .page-layout .mobile-shot-row > :nth-child(5) { grid-area: note; border-top: 1px solid #efede9; }
  .page-layout .mobile-shot-reference { grid-area: reference; border-top: 1px solid #efede9; }
  .page-layout .mobile-shot-actions { display: flex; grid-area: actions; align-items: center; justify-content: flex-end; gap: 3px; border-top: 1px solid #efede9; background: #faf9f7; }
  .page-layout .mobile-shot-actions button { display: grid; width: 28px; height: 28px; place-items: center; border-radius: 4px; color: #77736d; }
  .page-layout .mobile-shot-actions button:hover:not(:disabled) { color: #353330; background: #ece9e4; }
  .page-layout .mobile-shot-actions .danger-action:hover:not(:disabled) { color: #98523f; background: #f8e9e6; }
  .page-layout .mobile-shot-actions button:disabled { cursor: not-allowed; opacity: .3; }
  .page-layout .mobile-shot-row textarea { width: 100%; min-height: 82px; border: 1px solid #dedbd5; border-radius: 5px; padding: 9px 10px; color: #4d4944; background: #fff; font-size: 12px; line-height: 1.6; outline: none; resize: vertical; }
  .page-layout .mobile-shot-row textarea:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
  .page-layout .mobile-shot-duration .number-field { display: grid; width: 100%; height: 36px; grid-template-columns: minmax(0, 1fr) 28px; overflow: hidden; border: 1px solid #dedbd5; border-radius: 5px; background: #fff; }
  .page-layout .mobile-shot-duration .number-field input { min-width: 0; height: 34px; border: 0; border-radius: 0; padding: 0 5px 0 3px !important; background: transparent; box-shadow: none; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-variant-numeric: tabular-nums; text-align: right; }
  .page-layout .mobile-shot-duration .number-field span { position: static; display: flex; align-items: center; justify-content: center; border-left: 1px solid #ebe8e3; color: #77736d; background: #f7f6f3; font-size: 10px; line-height: 1; transform: none; }
  .page-layout .mobile-shot-reference :deep(.reference-grid) { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 6px; }
  .page-layout .mobile-shot-reference :deep(.reference-item input) { height: 28px; margin-top: 4px; padding: 0 6px; font-size: 10px; }
  .page-layout .mobile-shot-reference :deep(.upload-trigger) { height: 34px; font-size: 11px; }
  .page-layout .mobile-group-actions { display: flex; justify-content: flex-end; gap: 5px; margin-top: 10px; }
  .page-layout .mobile-group-actions button { height: 29px; border-color: transparent; background: transparent; }
  .page-layout .mobile-group-actions button:hover { background: #f3f1ed; }
  .page-layout .mobile-group-actions .danger-action { color: #98523f; }
  .page-layout .mobile-add-group { width: 100%; height: 40px; margin-top: 4px; border-style: dashed; }
  .page-layout .form-actions { position: sticky; bottom: 0; z-index: 5; }
}
@media (max-width: 480px) {
  .page-layout .mobile-group-header { gap: 10px; }
  .page-layout .mobile-group-order { width: 34px; height: 34px; font-size: 14px; }
  .page-layout .mobile-group-title-field input { width: min(220px, 100%); font-size: 17px; }
  .page-layout .mobile-group-meta { align-items: flex-end; flex-direction: column; gap: 2px; padding-top: 4px; }
  .page-layout .mobile-group-note-field { margin-left: 44px; }
  .page-layout .mobile-shot-reference :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
/* All-chapter desktop editor: every chapter stays in the reading flow. */
.page-layout { height: 100%; overflow-x: hidden; overflow-y: auto; }
.page-layout .storyboard-workspace { display: none; }
.page-layout .mobile-storyboard { display: flex; flex-direction: column; gap: 0; }
.page-layout .mobile-group-section { border-top: 1px solid #e4e1dc; padding: 24px 0 28px; }
.page-layout .mobile-group-section:first-child { border-top: 0; padding-top: 0; }
.page-layout .mobile-group-header { display: flex; min-height: 46px; align-items: flex-start; justify-content: space-between; gap: 20px; }
.page-layout .mobile-group-heading { display: flex; min-width: 0; align-items: center; gap: 12px; }
.page-layout .mobile-group-order { display: grid; width: 34px; height: 34px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #fff; background: #2b2926; font-size: 14px; font-weight: 800; }
.page-layout .mobile-group-title-field, .page-layout .mobile-group-note-field { display: flex; min-width: 0; flex-direction: column; gap: 4px; }
.page-layout .mobile-group-title-field span, .page-layout .mobile-group-note-field > span { color: #96918a; font-size: 10px; font-weight: 700; }
.page-layout .mobile-group-title-field input { width: min(420px, 42vw); height: 30px; border: 0; border-bottom: 1px solid #d8d5cf; border-radius: 0; padding: 0 0 3px; color: #353330; background: transparent; font-size: 18px; font-weight: 750; outline: none; }
.page-layout .mobile-group-title-field input:focus, .page-layout .mobile-group-note-field input:focus { border-color: #77736d; box-shadow: none; }
.page-layout .mobile-group-meta { display: flex; flex: 0 0 auto; align-items: baseline; gap: 7px; padding-top: 17px; color: #77736d; white-space: nowrap; }
.page-layout .mobile-group-meta strong { color: #4d4944; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 14px; font-variant-numeric: tabular-nums; }
.page-layout .mobile-group-meta span { font-size: 12px; }
.page-layout .mobile-group-note-field { margin: 9px 0 0 46px; }
.page-layout .mobile-group-note-field input { width: min(680px, 62vw); height: 28px; border: 0; border-bottom: 1px solid #ebe9e5; border-radius: 0; padding: 0; color: #77736d; background: transparent; font-size: 13px; outline: none; }
.page-layout .mobile-shot-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 20px 0 9px; border-bottom: 1px solid #e4e1dc; padding-bottom: 8px; }
.page-layout .mobile-shot-toolbar > span { color: #4d4944; font-size: 13px; font-weight: 750; }
.page-layout .mobile-shot-toolbar button, .page-layout .mobile-add-group, .page-layout .mobile-group-actions button { display: inline-flex; height: 30px; align-items: center; justify-content: center; gap: 5px; border: 1px solid #d8d5cf; border-radius: 5px; padding: 0 9px; color: #5f5a54; background: #fff; font-size: 11px; font-weight: 700; }
.page-layout .mobile-shot-toolbar button:hover, .page-layout .mobile-add-group:hover, .page-layout .mobile-group-actions button:hover { border-color: #aaa59d; background: #faf9f7; }
.page-layout .mobile-shot-table { overflow-x: auto; border: 1px solid #e4e1dc; border-radius: 7px; scrollbar-width: thin; }
.page-layout .mobile-shot-table-head, .page-layout .mobile-shot-row { display: grid; min-width: 1100px; grid-template-columns: 54px minmax(260px, 1.55fr) 96px minmax(180px, 1fr) minmax(180px, 1fr) 210px 94px; grid-template-areas: none; }
.page-layout .mobile-shot-table-head { min-height: 36px; align-items: center; color: #858078; background: #f4f2ee; font-size: 10px; font-weight: 750; }
.page-layout .mobile-shot-table-head > span { padding: 0 10px; }
.page-layout .mobile-shot-row { align-items: stretch; border-top: 1px solid #ebe8e3; background: #fff; }
.page-layout .mobile-shot-row > * { min-width: 0; grid-area: auto !important; padding: 10px; }
.page-layout .mobile-shot-row > * + * { border-left: 1px solid #efede9; }
.page-layout .mobile-shot-row:hover { background: #fcfbf9; }
.page-layout .mobile-shot-row > *::before { display: none; }
.page-layout .mobile-shot-index { display: flex; align-items: flex-start; justify-content: center; background: #faf9f7; }
.page-layout .mobile-shot-index > span { display: inline-grid; width: 28px; height: 26px; place-items: center; border: 1px solid #dedbd5; border-radius: 5px; color: #77736d; background: #fff; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; font-weight: 750; }
.page-layout .mobile-shot-field textarea { width: 100%; min-height: 74px; border: 1px solid #dedbd5; border-radius: 5px; padding: 8px 9px; color: #4d4944; background: #fff; font-size: 12px; line-height: 1.55; outline: none; resize: vertical; }
.page-layout .mobile-shot-field textarea:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.page-layout .mobile-shot-duration .number-field { display: grid; width: 100%; height: 34px; grid-template-columns: minmax(0, 1fr) 27px; overflow: hidden; border: 1px solid #dedbd5; border-radius: 5px; background: #fff; }
.page-layout .mobile-shot-duration .number-field input { min-width: 0; height: 32px; border: 0; border-radius: 0; padding: 0 4px 0 3px !important; background: transparent; box-shadow: none; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-variant-numeric: tabular-nums; text-align: right; }
.page-layout .mobile-shot-duration .number-field span { position: static; display: flex; align-items: center; justify-content: center; border-left: 1px solid #ebe8e3; color: #77736d; background: #f7f6f3; font-size: 10px; line-height: 1; transform: none; }
.page-layout .mobile-shot-reference :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
.page-layout .mobile-shot-reference :deep(.reference-item input) { height: 28px; margin-top: 4px; padding: 0 6px; font-size: 10px; }
.page-layout .mobile-shot-reference :deep(.upload-trigger) { height: 34px; gap: 5px; font-size: 11px; }
.page-layout .mobile-shot-actions { display: flex; align-items: flex-start; justify-content: flex-end; gap: 2px; padding-right: 8px !important; }
.page-layout .mobile-shot-actions button { display: grid; width: 26px; height: 26px; place-items: center; border-radius: 4px; color: #77736d; }
.page-layout .mobile-shot-actions button:hover:not(:disabled) { color: #353330; background: #ece9e4; }
.page-layout .mobile-shot-actions .danger-action:hover:not(:disabled) { color: #98523f; background: #f8e9e6; }
.page-layout .mobile-shot-actions button:disabled { cursor: not-allowed; opacity: .3; }
.page-layout .mobile-group-actions { display: flex; justify-content: flex-end; gap: 4px; margin-top: 8px; }
.page-layout .mobile-group-actions button { height: 28px; border-color: transparent; background: transparent; }
.page-layout .mobile-group-actions .danger-action { color: #98523f; }
.page-layout .mobile-add-group { width: 100%; height: 38px; margin-top: 4px; border-style: dashed; }
.page-layout .form-actions { position: sticky; bottom: 0; z-index: 5; }
@media (max-width: 900px) {
  .page-layout .mobile-group-title-field input { width: 280px; }
  .page-layout .mobile-group-note-field input { width: min(560px, 70vw); }
}
@media (max-width: 600px) {
  .page-layout .mobile-group-header { align-items: flex-start; flex-direction: column; gap: 6px; }
  .page-layout .mobile-group-meta { padding-top: 0; padding-left: 46px; }
  .page-layout .mobile-group-title-field input, .page-layout .mobile-group-note-field input { width: min(100%, 420px); }
}
/* Professional desktop pass: quiet surfaces, clear document rhythm, and stable controls. */
.page-layout.request-form { color: #34322f; background: #f6f5f2; }
.page-layout .request-section { padding: 24px 38px; }
.page-layout .request-settings { grid-template-columns: minmax(0, 1fr) 360px; gap: 32px; border-bottom: 1px solid #e6e3de; background: #fff; }
.page-layout .request-settings :deep(.field-label) { gap: 8px; }
.page-layout .request-settings :deep(.field-label label) { color: #4a4742; font-size: 12px; letter-spacing: .01em; }
.page-layout .request-settings :deep(input:not([type='file'])) { height: 42px; border-color: #d9d6d0; border-radius: 6px; color: #292724; background: #fff; font-size: 14px; }
.page-layout .request-settings :deep(input:not([type='file'])):hover { border-color: #bdb8b0; }
.page-layout .duration-picker { gap: 8px; }
.page-layout .duration-picker > button { height: 42px; border-color: #d9d6d0; border-radius: 6px; color: #66615b; background: #fff; }
.page-layout .duration-picker > button strong { font-size: 14px; }
.page-layout .duration-picker > button.active { border-color: #2b2926; color: #fff; background: #2b2926; box-shadow: 0 2px 5px rgb(43 41 38 / 12%); }
.page-layout .storyboard-section { gap: 0; padding-top: 27px; background: #fff; }
.page-layout .storyboard-toolbar { min-height: 44px; margin-bottom: 18px; }
.page-layout .storyboard-title { gap: 12px; }
.page-layout .storyboard-title h3 { color: #292724; font-size: 17px; letter-spacing: .01em; }
.page-layout .duration-status { height: 27px; border: 1px solid #dce7dc; border-radius: 5px; padding: 0 9px; color: #4e7454; background: #f1f6f1; font-size: 11px; }
.page-layout .duration-status.invalid { border-color: #ecd8d2; color: #98523f; background: #fdf3f0; }
.page-layout .add-group-button, .page-layout .mobile-shot-toolbar button { height: 34px; border-color: #d6d2cb; border-radius: 6px; color: #4f4b45; background: #fff; font-size: 12px; }
.page-layout .add-group-button:hover, .page-layout .mobile-shot-toolbar button:hover { border-color: #a8a29a; color: #292724; background: #faf9f7; }
.page-layout .mobile-group-section { padding: 26px 0 30px; }
.page-layout .mobile-group-section + .mobile-group-section { border-top-color: #dedbd5; }
.page-layout .mobile-group-heading { gap: 11px; }
.page-layout .mobile-group-order { width: 32px; height: 32px; border-radius: 5px; background: #34312d; font-size: 13px; }
.page-layout .mobile-group-title-field span, .page-layout .mobile-group-note-field > span { color: #9a958d; font-size: 10px; letter-spacing: .02em; }
.page-layout .mobile-group-title-field input { height: 32px; border-bottom-color: #cfcac2; color: #292724; font-size: 17px; }
.page-layout .mobile-group-title-field input:hover, .page-layout .mobile-group-note-field input:hover { border-bottom-color: #aaa39a; }
.page-layout .mobile-group-meta { gap: 8px; color: #8b857d; }
.page-layout .mobile-group-meta strong { color: #555049; font-size: 13px; }
.page-layout .mobile-group-note-field input { border-bottom-color: #e5e1db; color: #706a63; font-size: 12px; }
.page-layout .mobile-shot-toolbar { margin-top: 22px; border-bottom-color: #dedbd5; }
.page-layout .mobile-shot-toolbar > span { color: #5b5650; font-size: 12px; letter-spacing: .02em; }
.page-layout .mobile-shot-table { border-color: #dcd8d1; border-radius: 7px; background: #fff; }
.page-layout .mobile-shot-table-head { display: grid; min-height: 38px; color: #817b73; background: #f5f3ef; font-size: 10px; letter-spacing: .03em; }
.page-layout .mobile-shot-row { border-top-color: #e8e4de; }
.page-layout .mobile-shot-row:hover { background: #fdfcfb; }
.page-layout .mobile-shot-row > * + * { border-left-color: #eeeae5; }
.page-layout .mobile-shot-index { background: #faf9f7; }
.page-layout .mobile-shot-index > span { border-color: #d8d3cb; color: #625d56; }
.page-layout .mobile-shot-field textarea { min-height: 78px; border-color: #d8d4cd; border-radius: 5px; color: #45413c; background: #fff; font-size: 12px; }
.page-layout .mobile-shot-field textarea::placeholder { color: #aaa49c; }
.page-layout .mobile-shot-field textarea:hover { border-color: #bdb7ae; }
.page-layout .mobile-shot-duration .number-field { border-color: #d8d4cd; }
.page-layout .mobile-shot-duration .number-field span { color: #817b73; background: #f7f5f2; }
.page-layout .mobile-shot-reference :deep(.upload-trigger) { border-color: #c8c2b9; color: #655f58; background: #faf9f7; }
.page-layout .mobile-shot-reference :deep(.upload-trigger:hover) { border-color: #9f978d; background: #fff; }
.page-layout .mobile-shot-actions { background: #fcfbf9; }
.page-layout .mobile-shot-actions button { color: #8b857d; }
.page-layout .mobile-group-actions button { color: #7d766e; }
.page-layout .form-actions { border-top-color: #dedbd5; padding: 12px 38px; background: rgb(255 255 255 / 96%); }
.page-layout .secondary-button, .page-layout .primary-button { height: 38px; border-radius: 6px; font-size: 12px; }
.page-layout .secondary-button { border-color: #d6d2cb; color: #5f5952; background: #fff; }
.page-layout .primary-button { color: #fff; background: #2b2926; box-shadow: 0 2px 5px rgb(43 41 38 / 12%); }
.page-layout .primary-button:hover:not(:disabled) { background: #45413c; }
@media (max-width: 900px) {
  .page-layout .request-section { padding-right: 22px; padding-left: 22px; }
  .page-layout .form-actions { padding-right: 22px; padding-left: 22px; }
}
/* Final editor layout: the scene area scrolls independently of the actions. */
.page-layout.request-form { height: 100%; min-height: 0; overflow: hidden; }
.page-layout.request-form { display: grid; grid-template-columns: minmax(310px, 360px) minmax(0, 1fr); grid-template-rows: minmax(0, 1fr) auto; grid-template-areas: 'left storyboard' 'actions actions'; }
.page-layout .editor-left-panel { grid-area: left; min-width: 0; min-height: 0; overflow-y: auto; border-right: 1px solid #e6e3de; background: #fff; scrollbar-width: thin; }
.page-layout .editor-left-panel .request-section { padding-right: 24px; padding-left: 24px; }
.page-layout .editor-left-panel .section-divider { margin-right: 24px; margin-left: 24px; }
.page-layout .request-settings,
.page-layout .video-requirements { display: flex; flex-direction: column; gap: 22px; }
.page-layout .request-settings { grid-template-columns: 1fr; }
.page-layout .video-requirements { display: flex; flex-direction: column; gap: 18px; border-bottom: 1px solid #e6e3de; background: #fff; }
.page-layout .requirements-grid { display: grid; grid-template-columns: 1fr; gap: 14px; }
.page-layout .requirements-grid :deep(.field-label) { min-width: 0; }
.page-layout .overall-requirement-field { max-width: 980px; }
.page-layout .overall-requirement-field :deep(textarea) { min-height: 76px; border-color: #d9d6d0; border-radius: 6px; color: #45413c; background: #fff; font-size: 13px; }
.page-layout .storyboard-section { grid-area: storyboard; min-width: 0; min-height: 0; flex: 1 1 auto; overflow: hidden; padding-right: 24px; padding-left: 24px; }
.page-layout .storyboard-workspace { min-height: 0; flex: 1 1 auto; }
.page-layout .mobile-storyboard { min-height: 0; flex: 1 1 auto; overflow-y: auto; padding-bottom: 12px; }
.page-layout .form-actions { grid-area: actions; position: relative; bottom: auto; z-index: 10; margin-top: 0; border-radius: 0 0 8px 8px; }
.page-layout .shot-table { min-width: 1180px; }
.page-layout .shot-col-reference { width: 280px; }
.page-layout .shot-reference-cell :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(112px, 1fr)); gap: 8px; }
.page-layout .shot-reference-cell :deep(.reference-preview) { min-height: 92px; aspect-ratio: 4 / 3; }
.page-layout .mobile-shot-table-head,
.page-layout .mobile-shot-row { min-width: 1060px; grid-template-columns: 52px minmax(260px, 1.35fr) 112px minmax(160px, 1fr) minmax(160px, 1fr) 240px 76px; }
.page-layout .mobile-shot-reference :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(104px, 1fr)); gap: 8px; }
.page-layout .mobile-shot-reference :deep(.reference-preview) { min-height: 100px; aspect-ratio: 4 / 3; }
.page-layout .mobile-shot-reference :deep(.reference-preview img),
.page-layout .shot-reference-cell :deep(.reference-preview img) { object-fit: contain; background: #f3f1ed; }
.page-layout .mobile-shot-reference :deep(.reference-item input) { height: 30px; font-size: 11px; }
.page-layout .duration-input { appearance: textfield; width: 100%; height: 36px; border: 1px solid #dedbd5; border-radius: 5px; padding: 0 8px; color: #353330; background: #fff; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-variant-numeric: tabular-nums; text-align: center; outline: none; }
.page-layout .duration-input:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.page-layout .duration-input::-webkit-outer-spin-button,
.page-layout .duration-input::-webkit-inner-spin-button { margin: 0; appearance: none; }
.page-layout .shot-duration-cell .duration-input { min-width: 0; }
.page-layout .mobile-shot-duration .duration-input { height: 34px; }
.page-layout .mobile-shot-table-head > span:nth-child(3),
.page-layout .mobile-shot-duration { text-align: center; }
@media (max-width: 1100px) {
  .page-layout.request-form { grid-template-columns: minmax(280px, 320px) minmax(0, 1fr); }
}
@media (max-width: 720px) {
  .page-layout.request-form { display: flex; flex-direction: column; overflow-y: auto; }
  .page-layout .editor-left-panel { overflow: visible; border-right: 0; }
  .page-layout .storyboard-section { overflow: visible; }
  .page-layout .mobile-storyboard { overflow: visible; }
  .page-layout .mobile-shot-table-head,
  .page-layout .mobile-shot-row { min-width: 0; grid-template-columns: 48px minmax(0, 1fr) 82px; }
  .page-layout .mobile-shot-reference :deep(.reference-grid) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .page-layout .mobile-shot-reference :deep(.reference-preview) { min-height: 0; }
  .page-layout .requirements-grid { grid-template-columns: 1fr; }
  .page-layout .form-actions { position: sticky; bottom: 0; }
}
</style>
