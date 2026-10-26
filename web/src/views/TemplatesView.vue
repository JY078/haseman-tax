<template>
  <MainLayout>
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div class="mb-10">
        <h1 class="text-3xl sm:text-4xl font-bold text-neutral-900 mb-3">模板库</h1>
        <p class="text-lg text-neutral-600">选择提示词模板，快速开始 AI 创作</p>
      </div>

      <!-- Search & Filter -->
      <div class="flex flex-col sm:flex-row gap-4 mb-8">
        <div class="flex-1">
          <div class="relative max-w-md">
            <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-400 pointer-events-none" />
            <input
              v-model="search"
              placeholder="搜索模板名称或内容..."
              class="w-full pl-9 pr-4 py-2 text-sm border border-neutral-200 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent transition-all placeholder:text-neutral-400"
            />
          </div>
        </div>
        <div class="flex items-center gap-2 flex-wrap">
          <button
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
            :class="selectedGroupId === null
              ? 'bg-neutral-900 text-white'
              : 'bg-white text-neutral-600 border border-neutral-200 hover:bg-neutral-50'"
            @click="selectedGroupId = null"
          >
            全部
          </button>
          <button
            v-for="group in groups"
            :key="group.id"
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
            :class="selectedGroupId === group.id
              ? 'bg-neutral-900 text-white'
              : 'bg-white text-neutral-600 border border-neutral-200 hover:bg-neutral-50'"
            @click="selectedGroupId = group.id"
          >
            {{ group.name }}
          </button>
          <!-- 用户分享 tab（仅当有已上架的未分组模板时显示） -->
          <button
            v-if="userSharedPrompts.length > 0"
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors flex items-center gap-1.5"
            :class="selectedGroupId === '__shared__'
              ? 'bg-neutral-900 text-white'
              : 'bg-white text-neutral-600 border border-neutral-200 hover:bg-neutral-50'"
            @click="selectedGroupId = '__shared__'"
          >
            <span>用户分享</span>
            <span
              class="text-[11px] px-1.5 py-0.5 rounded-full font-semibold"
              :class="selectedGroupId === '__shared__' ? 'bg-white/20 text-white' : 'bg-neutral-100 text-neutral-500'"
            >{{ userSharedPrompts.length }}</span>
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-24">
        <div class="flex flex-col items-center gap-3">
          <svg class="w-8 h-8 text-neutral-400 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          <span class="text-sm text-neutral-500">加载中...</span>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="text-center py-20">
        <div class="w-16 h-16 bg-red-50 rounded-full flex items-center justify-center mx-auto mb-4">
          <ExclamationCircleIcon class="w-8 h-8 text-red-400" />
        </div>
        <h3 class="text-lg font-semibold text-neutral-900 mb-2">加载失败</h3>
        <p class="text-neutral-500 mb-4 text-sm">{{ error }}</p>
        <button
          class="px-4 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
          @click="fetchGroups"
        >重试</button>
      </div>

      <!-- Group sections (全部) -->
      <template v-else-if="!search && selectedGroupId === null">
        <div v-for="group in groups" :key="group.id" class="mb-12">
          <div class="flex items-center justify-between mb-4">
            <div>
              <h2 class="text-lg font-semibold text-neutral-900">{{ group.name }}</h2>
              <p v-if="group.description" class="text-sm text-neutral-500 mt-0.5">{{ group.description }}</p>
            </div>
            <button
              v-if="group.prompts && group.prompts.length > 4"
              class="text-sm text-neutral-500 hover:text-neutral-900 transition-colors"
              @click="selectedGroupId = group.id"
            >
              查看全部 {{ group.prompts.length }} 个 →
            </button>
          </div>
          <div v-if="group.prompts && group.prompts.length" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <PromptCard
              v-for="prompt in group.prompts.slice(0, 8)"
              :key="prompt.id"
              :prompt="prompt"
              :group-name="group.name"
              @use="openPrompt"
            />
          </div>
          <div v-else class="py-8 text-center text-sm text-neutral-400 bg-neutral-50 rounded-lg">该分组暂无模板</div>
        </div>

        <!-- 用户分享区块（已上架的未分组模板） -->
        <div v-if="userSharedPrompts.length > 0" class="mb-12">
          <div class="flex items-center justify-between mb-4">
            <div>
              <h2 class="text-lg font-semibold text-neutral-900">用户分享</h2>
              <p class="text-sm text-neutral-500 mt-0.5">由社区用户分享的创作模板</p>
            </div>
            <button
              v-if="userSharedPrompts.length > 4"
              class="text-sm text-neutral-500 hover:text-neutral-900 transition-colors"
              @click="selectedGroupId = '__shared__'"
            >
              查看全部 {{ userSharedPrompts.length }} 个 →
            </button>
          </div>
          <div class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <PromptCard
              v-for="prompt in userSharedPrompts.slice(0, 8)"
              :key="prompt.id"
              :prompt="prompt"
              group-name="用户分享"
              @use="openPrompt"
            />
          </div>
        </div>

        <div v-if="groups.length === 0 && userSharedPrompts.length === 0" class="text-center py-20">
          <div class="w-16 h-16 bg-neutral-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <RectangleStackIcon class="w-8 h-8 text-neutral-400" />
          </div>
          <h3 class="text-lg font-semibold text-neutral-900 mb-2">暂无模板</h3>
          <p class="text-neutral-500 text-sm">请在管理后台添加提示词模板</p>
        </div>
      </template>

      <!-- 用户分享 tab 单独展示 -->
      <template v-else-if="selectedGroupId === '__shared__'">
        <div v-if="filteredPrompts.length" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <PromptCard
            v-for="prompt in filteredPrompts"
            :key="prompt.id"
            :prompt="prompt"
            group-name="用户分享"
            @use="openPrompt"
          />
        </div>
        <div v-else class="text-center py-20">
          <div class="w-16 h-16 bg-neutral-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <MagnifyingGlassIcon class="w-8 h-8 text-neutral-400" />
          </div>
          <h3 class="text-lg font-semibold text-neutral-900 mb-2">{{ search ? '未找到模板' : '暂无用户分享模板' }}</h3>
          <p class="text-neutral-500 text-sm">{{ search ? '请尝试其他关键词' : '用户提交并通过审核的模板将在此展示' }}</p>
        </div>
      </template>

      <!-- Flat list (search or filter by group) -->
      <template v-else>
        <div v-if="filteredPrompts.length" class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <PromptCard
            v-for="prompt in filteredPrompts"
            :key="prompt.id"
            :prompt="prompt"
            :group-name="getGroupName(prompt.group_id)"
            @use="openPrompt"
          />
        </div>
        <div v-else class="text-center py-20">
          <div class="w-16 h-16 bg-neutral-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <MagnifyingGlassIcon class="w-8 h-8 text-neutral-400" />
          </div>
          <h3 class="text-lg font-semibold text-neutral-900 mb-2">未找到模板</h3>
          <p class="text-neutral-500 text-sm">请尝试其他关键词</p>
        </div>
      </template>
    </div>

    <!-- Prompt Detail Modal -->
    <Teleport to="body">
      <div
        v-if="activePrompt"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @click.self="activePrompt = null"
      >
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="activePrompt = null" />
        <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col max-h-[90vh]">
          <!-- Header -->
          <div class="flex items-start justify-between p-6 border-b border-neutral-100 flex-shrink-0">
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-semibold text-neutral-900 truncate">{{ activePrompt.title }}</h3>
              <span class="text-xs text-neutral-400 mt-1 block">{{ getGroupName(activePrompt.group_id) }}</span>
            </div>
            <button class="ml-4 p-1.5 rounded-lg hover:bg-neutral-100 text-neutral-400 hover:text-neutral-600 transition-colors flex-shrink-0" @click="activePrompt = null">
              <XMarkIcon class="w-5 h-5" />
            </button>
          </div>

          <!-- Scrollable body -->
          <div class="overflow-y-auto flex-1">
            <!-- 效果图画廊 -->
            <div
              v-if="activePromptImages.length"
              class="px-6 pt-6"
            >
              <label class="text-xs font-medium text-neutral-500 mb-3 block">效果图</label>
              <!-- 单图大图展示 -->
              <div v-if="activePromptImages.length === 1" class="rounded-xl overflow-hidden border border-neutral-100">
                <img
                  :src="activePromptImages[0]"
                  alt="效果图"
                  class="w-full object-cover cursor-zoom-in"
                  @click="previewImage = activePromptImages[0]"
                />
              </div>
              <!-- 多图网格 -->
              <div
                v-else
                class="grid gap-2"
                :class="activePromptImages.length === 2 ? 'grid-cols-2' : 'grid-cols-3'"
              >
                <div
                  v-for="(img, idx) in activePromptImages"
                  :key="idx"
                  class="aspect-square rounded-lg overflow-hidden border border-neutral-100 cursor-zoom-in"
                  @click="previewImage = img"
                >
                  <img :src="img" alt="效果图" class="w-full h-full object-cover hover:opacity-90 transition-opacity" />
                </div>
              </div>
            </div>

            <!-- 提示词内容 -->
            <div class="p-6">
              <label class="text-xs font-medium text-neutral-500 mb-2 block">提示词内容</label>
              <div class="bg-neutral-50 rounded-lg p-4 text-sm text-neutral-700 leading-relaxed whitespace-pre-wrap max-h-52 overflow-y-auto border border-neutral-100">{{ activePrompt.content }}</div>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex gap-3 px-6 pb-6 flex-shrink-0">
            <button
              class="flex-1 py-2.5 text-sm font-medium rounded-lg border border-neutral-200 text-neutral-700 hover:bg-neutral-50 transition-colors flex items-center justify-center gap-2"
              @click="copyPrompt"
            >
              <ClipboardDocumentIcon v-if="!copied" class="w-4 h-4" />
              <CheckIcon v-else class="w-4 h-4 text-emerald-500" />
              {{ copied ? '已复制' : '复制内容' }}
            </button>
            <button
              class="flex-1 py-2.5 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
              @click="usePrompt"
            >
              使用此模板
            </button>
          </div>
        </div>
      </div>

      <!-- 大图预览 lightbox -->
      <div
        v-if="previewImage"
        class="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/85 backdrop-blur-sm"
        @click="previewImage = null"
      >
        <button
          class="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors"
          @click.stop="previewImage = null"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
        <img
          :src="previewImage"
          alt="效果图预览"
          class="max-w-full max-h-full object-contain rounded-lg shadow-2xl"
          @click.stop
        />
      </div>
    </Teleport>

    <AuthModal v-model:open="authModalOpen" />
  </MainLayout>
</template>

<script setup>
import { ref, computed, onMounted, defineComponent, h } from 'vue'
import { useRouter } from 'vue-router'
import {
  MagnifyingGlassIcon,
  RectangleStackIcon,
  ExclamationCircleIcon,
  XMarkIcon,
  ClipboardDocumentIcon,
  CheckIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import AuthModal from '@/components/ui/AuthModal.vue'
import { useAuthStore } from '@/stores/auth'
import request, { resolveImageUrl } from '@/utils/request'

const router = useRouter()
const authStore = useAuthStore()

const groups = ref([])
const userSharedPrompts = ref([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const selectedGroupId = ref(null)
const activePrompt = ref(null)
const authModalOpen = ref(false)
const copied = ref(false)
const previewImage = ref(null)

const activePromptImages = computed(() => {
  if (!activePrompt.value) return []
  const imgs = activePrompt.value.images
  if (!Array.isArray(imgs)) return []
  return imgs.filter(Boolean).map(resolveImageUrl)
})

// PromptCard 子组件
const PromptCard = defineComponent({
  name: 'PromptCard',
  props: {
    prompt: Object,
    groupName: String,
  },
  emits: ['use'],
  setup(props, { emit }) {
    return () => {
      const images = Array.isArray(props.prompt.images) ? props.prompt.images.filter(Boolean) : []
      const hasImages = images.length > 0
      const extraCount = images.length - 1

      return h('div', {
        class: 'group bg-white border border-neutral-200 rounded-xl cursor-pointer hover:border-neutral-400 hover:shadow-md transition-all duration-200 flex flex-col overflow-hidden',
        onClick: () => emit('use', props.prompt),
      }, [
        // 封面图区域（有效果图时显示）
        hasImages
          ? h('div', { class: 'relative aspect-[16/9] bg-neutral-100 overflow-hidden flex-shrink-0' }, [
              h('img', {
                src: resolveImageUrl(images[0]),
                alt: '',
                class: 'w-full h-full object-cover transition-transform duration-300 group-hover:scale-105',
              }),
              extraCount > 0
                ? h('div', {
                    class: 'absolute bottom-2 right-2 bg-black/55 text-white text-[10px] font-medium px-2 py-0.5 rounded-full backdrop-blur-sm',
                  }, `+${extraCount} 张`)
                : null,
            ])
          : null,
        // 卡片内容区
        h('div', { class: 'p-4 flex flex-col gap-3 flex-1' }, [
          h('div', { class: 'flex items-start justify-between gap-2' }, [
            h('h3', { class: 'text-sm font-semibold text-neutral-900 leading-snug flex-1' }, props.prompt.title),
            h('span', { class: 'text-[10px] px-2 py-0.5 bg-neutral-100 text-neutral-500 rounded-full flex-shrink-0 font-medium' }, props.groupName),
          ]),
          h('p', { class: 'text-xs text-neutral-500 leading-relaxed line-clamp-3 flex-1' }, props.prompt.content),
          h('div', { class: 'flex items-center justify-end' }, [
            h('span', { class: 'text-xs text-neutral-400 group-hover:text-neutral-700 transition-colors font-medium' }, '点击使用 →'),
          ]),
        ]),
      ])
    }
  },
})

// 所有 prompt 的扁平列表（分组内 + 用户分享），用于搜索/筛选
const allPrompts = computed(() => [
  ...groups.value.flatMap(g => (g.prompts || []).map(p => ({ ...p, group_id: p.group_id || g.id }))),
  ...userSharedPrompts.value.map(p => ({ ...p, group_id: 0 })),
])

const filteredPrompts = computed(() => {
  let list = allPrompts.value
  if (search.value.trim()) {
    const kw = search.value.trim().toLowerCase()
    list = list.filter(p => p.title.toLowerCase().includes(kw) || p.content.toLowerCase().includes(kw))
  }
  if (selectedGroupId.value === '__shared__') {
    // 用户分享 tab：仅显示 group_id=0 的模板
    list = list.filter(p => !p.group_id || p.group_id === 0)
  } else if (selectedGroupId.value !== null) {
    list = list.filter(p => p.group_id === selectedGroupId.value)
  }
  return list
})

function getGroupName(groupId) {
  return groups.value.find(g => g.id === groupId)?.name || '用户分享'
}

async function fetchGroups() {
  loading.value = true
  error.value = ''
  try {
    const [groupsRes, sharedRes] = await Promise.all([
      request.get('/prompt-groups'),
      request.get('/prompts', { params: { group_id: 0 } }),
    ])
    const raw = groupsRes.data || []
    groups.value = [...raw]
      .sort((a, b) => {
        const sa = Number(a.sort) || 0
        const sb = Number(b.sort) || 0
        if (sa !== sb) return sa - sb
        return (Number(a.id) || 0) - (Number(b.id) || 0)
      })
      .map((g) => {
        const plist = Array.isArray(g.prompts) ? [...g.prompts] : []
        plist.sort((a, b) => {
          const sa = Number(a.sort) || 0
          const sb = Number(b.sort) || 0
          if (sa !== sb) return sa - sb
          return (Number(a.id) || 0) - (Number(b.id) || 0)
        })
        return { ...g, prompts: plist }
      })
    const shared = sharedRes.data || []
    userSharedPrompts.value = [...shared].sort((a, b) => {
      const sa = Number(a.sort) || 0
      const sb = Number(b.sort) || 0
      if (sa !== sb) return sa - sb
      return (Number(a.id) || 0) - (Number(b.id) || 0)
    })
  } catch (e) {
    error.value = e?.response?.data?.error || '请求失败，请检查网络或登录状态'
  } finally {
    loading.value = false
  }
}

function openPrompt(prompt) {
  activePrompt.value = prompt
  copied.value = false
}

async function copyPrompt() {
  if (!activePrompt.value) return
  try {
    await navigator.clipboard.writeText(activePrompt.value.content)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // fallback
  }
}

function usePrompt() {
  if (!activePrompt.value) return
  if (!authStore.isLoggedIn) { authModalOpen.value = true; return }
  router.push({ path: '/generate', query: { prompt: activePrompt.value.content } })
  activePrompt.value = null
}

onMounted(fetchGroups)
</script>
