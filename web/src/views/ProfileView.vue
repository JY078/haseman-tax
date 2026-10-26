<template>
  <MainLayout>
  <div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
    <!-- Header -->
    <div class="mb-8 flex items-center gap-5">
      <div class="w-16 h-16 bg-neutral-900 rounded-full flex items-center justify-center flex-shrink-0">
        <span class="text-2xl font-bold text-white">{{ initial }}</span>
      </div>
      <div>
        <h1 class="text-2xl font-bold text-neutral-900">{{ authStore.user?.username }}</h1>
        <p class="text-sm text-neutral-500 mt-0.5">{{ authStore.user?.email }}</p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-neutral-200 mb-8">
      <nav class="flex gap-0 -mb-px">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          class="px-5 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap"
          :class="activeTab === tab.id
            ? 'border-neutral-900 text-neutral-900'
            : 'border-transparent text-neutral-500 hover:text-neutral-700'"
        >
          {{ tab.label }}
        </button>
      </nav>
    </div>

    <!-- Tab: 基本信息 -->
    <div v-if="activeTab === 'info'" class="space-y-6">

      <!-- Email missing warning -->
      <div v-if="!hasEmail" class="flex items-start gap-3 bg-amber-50 border border-amber-200 rounded-xl px-5 py-4">
        <ExclamationTriangleIcon class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-amber-800">您尚未绑定邮箱</p>
          <p class="text-xs text-amber-600 mt-0.5">绑定邮箱后，当您忘记密码时可通过邮箱找回。</p>
        </div>
        <button
          @click="showBindEmail = !showBindEmail"
          class="text-xs font-medium text-amber-700 hover:text-amber-900 border border-amber-300 rounded-lg px-3 py-1.5 transition-colors flex-shrink-0"
        >
          立即绑定
        </button>
      </div>

      <!-- Bind email form (inline) -->
      <div v-if="showBindEmail" class="bg-white border border-neutral-200 rounded-xl p-6">
        <h2 class="text-base font-semibold text-neutral-900 mb-5">{{ hasEmail ? '更换邮箱' : '绑定邮箱' }}</h2>
        <form @submit.prevent="submitBindEmail" class="space-y-4 max-w-md">
          <div>
            <label class="block text-xs font-medium text-neutral-700 mb-1.5">新邮箱地址</label>
            <input
              v-model="bindForm.email"
              type="email"
              class="w-full px-3 py-2.5 border border-neutral-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent"
              placeholder="请输入邮箱地址"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-neutral-700 mb-1.5">验证码</label>
            <div class="flex gap-2">
              <input
                v-model="bindForm.code"
                type="text"
                maxlength="6"
                class="flex-1 px-3 py-2.5 border border-neutral-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent"
                placeholder="6 位验证码"
              />
              <button
                type="button"
                @click="sendBindCode"
                :disabled="bindCooldown > 0 || bindSending"
                class="px-4 py-2.5 text-sm border border-neutral-200 rounded-lg text-neutral-700 hover:bg-neutral-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors whitespace-nowrap flex-shrink-0"
              >
                <span v-if="bindSending">发送中...</span>
                <span v-else-if="bindCooldown > 0">{{ bindCooldown }}s 后重试</span>
                <span v-else>获取验证码</span>
              </button>
            </div>
          </div>
          <div v-if="bindError" class="text-xs text-red-500">{{ bindError }}</div>
          <div v-if="bindSuccess" class="flex items-center gap-1.5 text-xs text-green-600">
            <CheckCircleIcon class="w-4 h-4" />
            邮箱绑定成功！
          </div>
          <div class="flex gap-3">
            <button
              type="submit"
              :disabled="bindLoading"
              class="px-5 py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-lg hover:bg-neutral-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {{ bindLoading ? '提交中...' : '确认绑定' }}
            </button>
            <button
              type="button"
              @click="showBindEmail = false"
              class="px-5 py-2.5 text-sm text-neutral-600 border border-neutral-200 rounded-lg hover:bg-neutral-50 transition-colors"
            >
              取消
            </button>
          </div>
        </form>
      </div>

      <div class="bg-white border border-neutral-200 rounded-xl p-6">
        <h2 class="text-base font-semibold text-neutral-900 mb-5">账号信息</h2>
        <dl class="grid grid-cols-1 sm:grid-cols-2 gap-5">
          <div>
            <dt class="text-xs font-medium text-neutral-500 mb-1">用户名</dt>
            <dd class="text-sm text-neutral-900 font-medium">{{ authStore.user?.username }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium text-neutral-500 mb-1">邮箱</dt>
            <dd class="flex items-center gap-2">
              <span class="text-sm text-neutral-900">{{ hasEmail ? authStore.user.email : '未绑定' }}</span>
              <button
                v-if="hasEmail"
                @click="showBindEmail = !showBindEmail"
                class="text-xs text-neutral-400 hover:text-neutral-700 transition-colors"
              >
                更换
              </button>
              <span v-else class="text-xs text-amber-500">⚠ 未绑定</span>
            </dd>
          </div>
          <div>
            <dt class="text-xs font-medium text-neutral-500 mb-1">注册时间</dt>
            <dd class="text-sm text-neutral-900">{{ formatDate(profile?.created_at) }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium text-neutral-500 mb-1">累计使用次数</dt>
            <dd class="text-sm text-neutral-900">{{ profile?.usage_count ?? authStore.user?.usage_count ?? 0 }} 次</dd>
          </div>
        </dl>
      </div>

      <!-- Usage stats cards -->
      <!-- <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
        <div v-for="stat in usageStats" :key="stat.label" class="bg-white border border-neutral-200 rounded-xl p-4 text-center">
          <div class="text-2xl font-bold text-neutral-900 mb-1">{{ stat.count }}</div>
          <div class="text-xs text-neutral-500">{{ stat.label }}</div>
        </div>
      </div> -->
    </div>

    <!-- Tab: 修改密码 -->
    <div v-if="activeTab === 'password'">
      <div class="bg-white border border-neutral-200 rounded-xl p-6 max-w-md">
        <h2 class="text-base font-semibold text-neutral-900 mb-5">修改密码</h2>
        <form @submit.prevent="changePassword" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-neutral-700 mb-1.5">当前密码</label>
            <div class="relative">
              <input
                v-model="pwForm.oldPassword"
                :type="showPw.old ? 'text' : 'password'"
                class="w-full px-3 py-2.5 pr-10 border border-neutral-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent"
                placeholder="请输入当前密码"
                autocomplete="current-password"
              />
              <button type="button" @click="showPw.old = !showPw.old" class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600">
                <EyeSlashIcon v-if="showPw.old" class="w-4 h-4" />
                <EyeIcon v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-neutral-700 mb-1.5">新密码</label>
            <div class="relative">
              <input
                v-model="pwForm.newPassword"
                :type="showPw.new ? 'text' : 'password'"
                class="w-full px-3 py-2.5 pr-10 border border-neutral-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent"
                placeholder="至少 6 位"
                autocomplete="new-password"
              />
              <button type="button" @click="showPw.new = !showPw.new" class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600">
                <EyeSlashIcon v-if="showPw.new" class="w-4 h-4" />
                <EyeIcon v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-neutral-700 mb-1.5">确认新密码</label>
            <div class="relative">
              <input
                v-model="pwForm.confirmPassword"
                :type="showPw.confirm ? 'text' : 'password'"
                class="w-full px-3 py-2.5 pr-10 border border-neutral-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent"
                placeholder="再次输入新密码"
                autocomplete="new-password"
              />
              <button type="button" @click="showPw.confirm = !showPw.confirm" class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600">
                <EyeSlashIcon v-if="showPw.confirm" class="w-4 h-4" />
                <EyeIcon v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div v-if="pwError" class="text-xs text-red-500 py-1">{{ pwError }}</div>
          <div v-if="pwSuccess" class="flex items-center gap-1.5 text-xs text-green-600 py-1">
            <CheckCircleIcon class="w-4 h-4" />
            密码修改成功
          </div>

          <button
            type="submit"
            :disabled="pwLoading"
            class="w-full py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-lg hover:bg-neutral-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <span v-if="pwLoading" class="inline-flex items-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              保存中...
            </span>
            <span v-else>保存修改</span>
          </button>
        </form>
      </div>
    </div>

    <!-- Tab: 登录设备 -->
    <div v-if="activeTab === 'sessions'">
      <div class="bg-white border border-neutral-200 rounded-xl overflow-hidden">
        <div class="px-6 py-4 border-b border-neutral-100 flex items-center justify-between">
          <h2 class="text-base font-semibold text-neutral-900">登录设备</h2>
          <button
            @click="loadSessions"
            class="text-xs text-neutral-500 hover:text-neutral-700 flex items-center gap-1 transition-colors"
          >
            <ArrowPathIcon class="w-3.5 h-3.5" />
            刷新
          </button>
        </div>

        <div v-if="sessionsLoading" class="py-12 flex items-center justify-center text-neutral-400 text-sm gap-2">
          <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          加载中...
        </div>

        <div v-else-if="sessions.length === 0" class="py-12 text-center text-sm text-neutral-400">
          暂无活跃会话
        </div>

        <ul v-else class="divide-y divide-neutral-100">
          <li v-for="s in sessions" :key="s.id" class="px-6 py-4 flex items-start gap-4">
            <div class="w-9 h-9 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5"
              :class="s.is_current ? 'bg-neutral-900' : 'bg-neutral-100'"
            >
              <ComputerDesktopIcon class="w-4 h-4" :class="s.is_current ? 'text-white' : 'text-neutral-500'" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-sm font-medium text-neutral-900 truncate">{{ parseUA(s.user_agent) }}</span>
                <span v-if="s.is_current" class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-green-100 text-green-700">
                  当前设备
                </span>
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium"
                  :class="s.client_type === 'admin' ? 'bg-orange-50 text-orange-600' : 'bg-neutral-100 text-neutral-600'"
                >
                  {{ s.client_type === 'admin' ? '管理后台' : 'Web 端' }}
                </span>
              </div>
              <div class="text-xs text-neutral-400 mt-1 space-x-3">
                <span v-if="s.ip">IP: {{ s.ip }}</span>
                <span>登录于 {{ formatDate(s.created_at) }}</span>
                <span v-if="s.last_active_at">最近活跃 {{ timeAgo(s.last_active_at) }}</span>
              </div>
            </div>
            <button
              v-if="!s.is_current"
              @click="revokeSession(s.id)"
              :disabled="revokingId === s.id"
              class="text-xs text-red-500 hover:text-red-700 flex-shrink-0 py-1 transition-colors disabled:opacity-50"
            >
              {{ revokingId === s.id ? '下线中...' : '强制下线' }}
            </button>
          </li>
        </ul>
      </div>
    </div>
  </div>
  </MainLayout>
</template>

<script setup>
import { ref, computed, onMounted, reactive, watch } from 'vue'
import {
  EyeIcon, EyeSlashIcon, CheckCircleIcon,
  ComputerDesktopIcon, ArrowPathIcon, ExclamationTriangleIcon
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import { useAuthStore } from '@/stores/auth'
import request from '@/utils/request'

const authStore = useAuthStore()

const hasEmail = computed(() => {
  const email = authStore.user?.email
  return !!(email && email.trim() !== '')
})

const tabs = [
  { id: 'info',     label: '基本信息' },
  { id: 'password', label: '修改密码' },
  { id: 'sessions', label: '登录设备' },
]
const activeTab = ref('info')

const initial = computed(() =>
  authStore.user?.username?.charAt(0)?.toUpperCase() || 'U'
)

// ── Profile info ─────────────────────────────────────────────────────────────
const profile = ref(null)

async function loadProfile() {
  try {
    const res = await request.get('/me')
    profile.value = res.data
  } catch { /* ignore */ }
}

// ── Usage stats ───────────────────────────────────────────────────────────────
const usageStats = ref([
  { label: 'AI 图片生成', count: 0, source: 'generate' },
  { label: '多视角',      count: 0, source: 'multi_angle' },
  { label: '消除水印',    count: 0, source: 'eraser' },
  { label: '去除背景',    count: 0, source: 'remove-bg' },
])

async function loadStats() {
  try {
    const results = await Promise.allSettled(
      usageStats.value.map(stat => {
        const params = { source: stat.source, page: 1, page_size: 1, image_only: 1 }
        return request.get('/usage-logs', { params })
      })
    )
    results.forEach((r, i) => {
      if (r.status === 'fulfilled') {
        usageStats.value[i].count = r.value.data?.total ?? 0
      }
    })
  } catch { /* ignore */ }
}

// ── Bind Email ────────────────────────────────────────────────────────────────
const showBindEmail = ref(false)
const bindForm      = reactive({ email: '', code: '' })
const bindSending   = ref(false)
const bindLoading   = ref(false)
const bindError     = ref('')
const bindSuccess   = ref(false)
const bindCooldown  = ref(0)
let bindCooldownTimer = null

function startBindCooldown(seconds = 60) {
  bindCooldown.value = seconds
  bindCooldownTimer = setInterval(() => {
    bindCooldown.value--
    if (bindCooldown.value <= 0) {
      clearInterval(bindCooldownTimer)
      bindCooldownTimer = null
    }
  }, 1000)
}

async function sendBindCode() {
  bindError.value = ''
  if (!bindForm.email) { bindError.value = '请先输入邮箱地址'; return }
  bindSending.value = true
  try {
    await request.post('/me/send-bind-email-code', { email: bindForm.email })
    startBindCooldown(60)
  } catch (err) {
    const data = err.response?.data
    bindError.value = data?.error || '发送失败，请重试'
    if (data?.wait_seconds) startBindCooldown(data.wait_seconds)
  } finally {
    bindSending.value = false
  }
}

async function submitBindEmail() {
  bindError.value   = ''
  bindSuccess.value = false
  if (!bindForm.email) { bindError.value = '请输入邮箱地址'; return }
  if (!bindForm.code)  { bindError.value = '请输入验证码'; return }
  bindLoading.value = true
  try {
    const res = await request.put('/me/email', { email: bindForm.email, code: bindForm.code })
    bindSuccess.value = true
    // Update store so UI immediately reflects the new email
    if (authStore.user) {
      authStore.user.email = res.data.email
      try {
        localStorage.setItem('user', JSON.stringify(authStore.user))
      } catch { /* ignore */ }
    }
    bindForm.email = ''
    bindForm.code  = ''
    setTimeout(() => {
      showBindEmail.value = false
      bindSuccess.value   = false
    }, 2000)
  } catch (err) {
    bindError.value = err.response?.data?.error || '绑定失败，请重试'
  } finally {
    bindLoading.value = false
  }
}

// ── Change Password ───────────────────────────────────────────────────────────
const pwForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const showPw  = reactive({ old: false, new: false, confirm: false })
const pwLoading = ref(false)
const pwError   = ref('')
const pwSuccess = ref(false)

async function changePassword() {
  pwError.value   = ''
  pwSuccess.value = false

  if (!pwForm.oldPassword) { pwError.value = '请输入当前密码'; return }
  if (pwForm.newPassword.length < 6) { pwError.value = '新密码至少 6 位'; return }
  if (pwForm.newPassword !== pwForm.confirmPassword) { pwError.value = '两次密码输入不一致'; return }

  pwLoading.value = true
  try {
    await request.put('/me/password', {
      old_password: pwForm.oldPassword,
      new_password: pwForm.newPassword,
    })
    pwSuccess.value = true
    pwForm.oldPassword = ''
    pwForm.newPassword = ''
    pwForm.confirmPassword = ''
  } catch (err) {
    pwError.value = err.response?.data?.error || '修改失败，请重试'
  } finally {
    pwLoading.value = false
  }
}

// ── Sessions ──────────────────────────────────────────────────────────────────
const sessions       = ref([])
const sessionsLoading = ref(false)
const revokingId     = ref(null)

async function loadSessions() {
  sessionsLoading.value = true
  try {
    const res = await request.get('/me/sessions')
    sessions.value = res.data || []
  } catch { sessions.value = [] }
  finally { sessionsLoading.value = false }
}

async function revokeSession(id) {
  revokingId.value = id
  try {
    await request.delete(`/me/sessions/${id}`)
    sessions.value = sessions.value.filter(s => s.id !== id)
  } catch { /* ignore */ }
  finally { revokingId.value = null }
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

function timeAgo(d) {
  if (!d) return '-'
  const diff = Date.now() - new Date(d).getTime()
  const min  = Math.floor(diff / 60000)
  if (min < 1)   return '刚刚'
  if (min < 60)  return `${min} 分钟前`
  const hr = Math.floor(min / 60)
  if (hr < 24)   return `${hr} 小时前`
  return `${Math.floor(hr / 24)} 天前`
}

function parseUA(ua = '') {
  if (!ua) return '未知浏览器'
  if (/Edg/i.test(ua))    return 'Edge 浏览器'
  if (/Chrome/i.test(ua)) return 'Chrome 浏览器'
  if (/Firefox/i.test(ua)) return 'Firefox 浏览器'
  if (/Safari/i.test(ua)) return 'Safari 浏览器'
  return '浏览器'
}

watch(activeTab, (tab) => {
  if (tab === 'sessions') loadSessions()
})

onMounted(() => {
  loadProfile()
  loadStats()
})
</script>
