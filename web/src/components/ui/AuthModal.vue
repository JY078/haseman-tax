<template>
  <TransitionRoot as="template" :show="open">
    <Dialog as="div" class="relative z-50" @close="$emit('update:open', false)">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-black/60" />
      </TransitionChild>

      <div class="fixed inset-0 overflow-y-auto">
        <div class="flex min-h-full items-center justify-center p-4">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <DialogPanel class="w-full max-w-md">
              <div class="bg-white rounded-3xl overflow-hidden">
                <div class="relative">
                  <button
                    @click="$emit('update:open', false)"
                    class="absolute top-5 right-5 p-2 text-neutral-400 hover:text-neutral-600 transition-colors z-10"
                  >
                    <XMarkIcon class="w-5 h-5" />
                  </button>

                  <div class="px-10 pt-12 pb-4">
                    <!-- <div class="flex items-center gap-2.5 mb-10">
                      <div class="w-10 h-10 bg-neutral-900 rounded-xl flex items-center justify-center">
                        <span class="text-white font-bold text-xs leading-none">DH</span>
                      </div>
                      <span class="text-xl font-semibold text-neutral-900">DH</span>
                    </div> -->

                    <h2 class="text-2xl font-bold text-neutral-900 mb-2">
                      {{ mode === 'login' ? '登录你的账号' : '重置密码' }}
                    </h2>
                    <div
                      v-if="gateHint && mode !== 'forgot'"
                      class="mb-6 rounded-xl bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-950"
                      role="status"
                    >
                      {{ gateHint }}
                    </div>
                    <!-- <p class="text-neutral-500 mb-8">
                      {{ mode === 'login' ? '输入你的账号和密码以继续'
                        : forgotStep === 1 ? '输入账号绑定的邮箱，获取重置验证码'
                        : forgotStep === 2 ? '输入验证码并设置新密码'
                        : '' }}
                    </p> -->

                    <!-- ── Login / Register form ── -->
                    <form v-if="mode !== 'forgot'" @submit.prevent="handleSubmit" class="space-y-5">
                      <div>
                        <label class="block text-sm font-medium text-neutral-700 mb-2">账号</label>
                        <input
                          v-model="form.username"
                          type="text"
                          placeholder="输入你的账号"
                          class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors"
                        />
                      </div>

                      <div>
                        <div class="flex items-center justify-between mb-2">
                          <label class="text-sm font-medium text-neutral-700">密码</label>
                          <!-- <button
                            v-if="mode === 'login'"
                            type="button"
                            class="text-sm text-neutral-500 hover:text-neutral-700 transition-colors"
                            @click="enterForgot"
                          >忘记密码？</button> -->
                        </div>
                        <input
                          v-model="form.password"
                          type="password"
                          placeholder="••••••••"
                          class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors"
                        />
                      </div>

                      <div v-if="error" class="text-red-500 text-sm">{{ error }}</div>

                      <button
                        type="submit"
                        :disabled="loading"
                        class="w-full py-3.5 bg-neutral-900 text-white font-medium rounded-xl hover:bg-neutral-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed mt-2"
                      >
                        {{ loading ? '处理中...' : '登录' }}
                      </button>
                    </form>

                    <!-- ── Forgot password form ── -->
                    <div v-else class="space-y-5">

                      <!-- Step 1: enter email -->
                      <template v-if="forgotStep === 1">
                        <div>
                          <label class="block text-sm font-medium text-neutral-700 mb-2">注册邮箱</label>
                          <input
                            v-model="forgot.email"
                            type="email"
                            placeholder="输入账号绑定的邮箱"
                            class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors"
                          />
                        </div>
                        <div v-if="error" class="text-red-500 text-sm">{{ error }}</div>
                        <button
                          :disabled="loading || !forgot.email"
                          class="w-full py-3.5 bg-neutral-900 text-white font-medium rounded-xl hover:bg-neutral-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          @click="sendResetCode"
                        >
                          {{ loading ? '发送中...' : '发送验证码' }}
                        </button>
                      </template>

                      <!-- Step 2: enter code + new password -->
                      <template v-else-if="forgotStep === 2">
                        <div class="p-3 bg-neutral-50 rounded-xl text-sm text-neutral-500">
                          验证码已发送至 <span class="font-medium text-neutral-700">{{ forgot.email }}</span>，有效期 10 分钟
                        </div>
                        <div>
                          <div class="flex items-center justify-between mb-2">
                            <label class="text-sm font-medium text-neutral-700">验证码</label>
                            <button
                              type="button"
                              :disabled="resetCooldown > 0 || loading"
                              class="text-sm text-neutral-500 hover:text-neutral-700 disabled:opacity-40 transition-colors"
                              @click="sendResetCode"
                            >{{ resetCooldown > 0 ? `${resetCooldown}s 后重发` : '重新发送' }}</button>
                          </div>
                          <input
                            v-model="forgot.code"
                            type="text"
                            maxlength="6"
                            placeholder="6位验证码"
                            class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors tracking-widest text-center text-lg font-medium"
                          />
                        </div>
                        <div>
                          <label class="block text-sm font-medium text-neutral-700 mb-2">新密码</label>
                          <input
                            v-model="forgot.password"
                            type="password"
                            placeholder="至少 6 位"
                            class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors"
                          />
                        </div>
                        <div>
                          <label class="block text-sm font-medium text-neutral-700 mb-2">确认新密码</label>
                          <input
                            v-model="forgot.confirmPassword"
                            type="password"
                            placeholder="再次输入新密码"
                            class="w-full px-4 py-3.5 bg-neutral-50 border border-neutral-200 rounded-xl text-neutral-900 placeholder-neutral-400 focus:outline-none focus:border-neutral-400 transition-colors"
                          />
                        </div>
                        <div v-if="error" class="text-red-500 text-sm">{{ error }}</div>
                        <button
                          :disabled="loading || !forgot.code || !forgot.password"
                          class="w-full py-3.5 bg-neutral-900 text-white font-medium rounded-xl hover:bg-neutral-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          @click="doResetPassword"
                        >
                          {{ loading ? '重置中...' : '确认重置密码' }}
                        </button>
                      </template>

                      <!-- Step 3: success -->
                      <template v-else-if="forgotStep === 3">
                        <div class="flex flex-col items-center gap-4 py-4">
                          <div class="w-14 h-14 bg-emerald-100 rounded-full flex items-center justify-center">
                            <svg class="w-7 h-7 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          </div>
                          <div class="text-center">
                            <p class="font-semibold text-neutral-900 mb-1">密码重置成功</p>
                            <p class="text-sm text-neutral-500">请使用新密码重新登录</p>
                          </div>
                          <button
                            class="w-full py-3.5 bg-neutral-900 text-white font-medium rounded-xl hover:bg-neutral-800 transition-colors"
                            @click="backToLogin"
                          >立即登录</button>
                        </div>
                      </template>
                    </div>
                  </div>

                  <!-- Footer -->
                  <div class="px-10 py-5 bg-neutral-50 text-center">
                    <template v-if="mode === 'forgot'">
                      <button
                        class="text-sm text-neutral-500 hover:text-neutral-700 transition-colors flex items-center gap-1 mx-auto"
                        @click="backToLogin"
                      >
                        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
                        </svg>
                        返回登录
                      </button>
                    </template>
                    <!-- <template v-else>
                      <span class="text-sm text-neutral-500">暂不支持在线注册</span>
                    </template> -->
                  </div>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { useAuthStore } from '@/stores/auth'
import { userLandingRoute } from '@/utils/userLandingRoute'
import request from '@/utils/request'

const props = defineProps({
  open: { type: Boolean, required: true },
  /** 从需登录入口打开时展示的简短说明（登录/注册模式显示，忘记密码时隐藏） */
  gateHint: { type: String, default: '' },
})
const emit = defineEmits(['update:open'])

const authStore = useAuthStore()
const router = useRouter()

// mode: 'login' | 'forgot'
const mode = ref('login')
const loading = ref(false)
const error = ref('')

const form = reactive({
  username: '',
  password: '',
})

// ── Forgot password ───────────────────────────────────────────────────────
const forgotStep = ref(1)   // 1=email input, 2=code+new password, 3=success
const resetCooldown = ref(0)
let resetCooldownTimer = null

const forgot = reactive({
  email: '', code: '', password: '', confirmPassword: '',
})

// ── Watchers ─────────────────────────────────────────────────────────────
watch(() => props.open, (val) => {
  if (!val) resetAll()
})

// ── Helpers ───────────────────────────────────────────────────────────────
function resetAll() {
  Object.assign(form, { username: '', password: '' })
  Object.assign(forgot, { email: '', code: '', password: '', confirmPassword: '' })
  mode.value = 'login'
  forgotStep.value = 1
  error.value = ''
  resetCooldown.value = 0
  if (resetCooldownTimer) { clearInterval(resetCooldownTimer); resetCooldownTimer = null }
}

function enterForgot() {
  mode.value = 'forgot'
  forgotStep.value = 1
  error.value = ''
}

function backToLogin() {
  mode.value = 'login'
  forgotStep.value = 1
  error.value = ''
  Object.assign(forgot, { email: '', code: '', password: '', confirmPassword: '' })
}

function startCountdown(counter, seconds) {
  counter.value = seconds
  const id = setInterval(() => {
    counter.value--
    if (counter.value <= 0) clearInterval(id)
  }, 1000)
  return id
}

// ── Login submit ───────────────────────────────────────────────────────────
async function handleSubmit() {
  error.value = ''
  if (!form.username) { error.value = '请输入账号'; return }
  if (!form.password) { error.value = '请输入密码'; return }

  loading.value = true
  try {
    const res = await request.post('/login', {
      username: form.username, password: form.password, client_type: 'web',
    })
    authStore.login(res.data.user, res.data.token)
    emit('update:open', false)
    await router.replace(userLandingRoute(res.data.user))
  } catch (e) {
    error.value = e.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}

// ── Forgot: send reset code ───────────────────────────────────────────────
async function sendResetCode() {
  if (!forgot.email) { error.value = '请输入邮箱地址'; return }
  error.value = ''
  loading.value = true
  try {
    await request.post('/forgot-password', { email: forgot.email })
    forgotStep.value = 2
    resetCooldownTimer = startCountdown(resetCooldown, 60)
  } catch (e) {
    error.value = e.response?.data?.error || '发送失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

// ── Forgot: confirm reset ─────────────────────────────────────────────────
async function doResetPassword() {
  error.value = ''
  if (!forgot.code)     { error.value = '请输入验证码'; return }
  if (forgot.password.length < 6)                      { error.value = '新密码至少 6 位'; return }
  if (forgot.password !== forgot.confirmPassword)       { error.value = '两次密码不一致'; return }

  loading.value = true
  try {
    await request.post('/reset-password', {
      email:    forgot.email,
      code:     forgot.code,
      password: forgot.password,
    })
    forgotStep.value = 3
  } catch (e) {
    error.value = e.response?.data?.error || '重置失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>
