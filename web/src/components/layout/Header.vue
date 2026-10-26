<template>
  <header class="sticky top-0 z-40 bg-white/95 backdrop-blur-sm border-b border-neutral-200">
    <div class="w-full px-4 sm:px-6 lg:px-8 xl:px-10">
      <div class="flex items-center justify-between h-16">
        <router-link to="/tax-data" class="flex items-center gap-2">
          <div class="w-8 h-8 bg-neutral-900 rounded-lg flex items-center justify-center">
            <span class="text-white font-bold text-xs leading-none">HSM</span>
          </div>
          <div class="flex flex-col leading-none">
            <span class="text-lg font-semibold text-neutral-900">HSM </span>
            <span class="text-[10px] text-neutral-400">{{ t('内部工作台') }}</span>
          </div>
        </router-link>

        <!-- 桌面端导航 -->
        <nav class="items-center gap-1" :class="navigationLocale === 'en' ? 'hidden lg:flex' : 'hidden md:flex'">
          <template v-for="item in navigation" :key="item.name">

            <!-- 含子菜单的下拉项 -->
            <Menu v-if="item.children" v-slot="{ open }" as="div" class="relative">
              <MenuButton
                class="inline-flex items-center gap-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors"
                :class="open
                  ? 'bg-neutral-100 text-neutral-900'
                  : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900'">
                {{ t(item.name) }}
                <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" />
                </svg>
              </MenuButton>

                <MenuItems
                  class="absolute left-0 mt-1 bg-white rounded-xl shadow-lg ring-1 ring-black/5 focus:outline-none z-50"
                  :class="item.children.some(child => child.section) ? 'w-[560px] p-3' : 'w-64 p-1'">
                  <div class="grid gap-3" :class="item.children.some(child => child.section) ? 'grid-cols-2' : 'grid-cols-1'">
                    <div v-for="group in groupNavChildren(item.children)" :key="group.name" class="min-w-0"
                      :class="group.name === '图像资源' ? 'col-span-2 border-t border-neutral-100 pt-2' : ''">
                      <div v-if="group.name !== '功能'" class="px-3 pb-2 pt-1 text-xs font-semibold text-neutral-400">{{ t(group.name) }}</div>
                      <MenuItem v-for="child in group.items" :key="child.name" v-slot="{ active }">
                        <router-link :to="child.to" class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors"
                          :class="[
                            active ? 'bg-neutral-50' : '',
                            isChildActive(child.to) ? 'text-neutral-900 font-medium' : 'text-neutral-600'
                          ]">
                          <component :is="child.icon" class="h-5 w-5 flex-shrink-0 text-neutral-500" aria-hidden="true" />
                          <div class="flex-1 min-w-0">
                            <div class="leading-tight">{{ t(child.name) }}</div>
                            <div class="text-xs text-neutral-400 truncate mt-0.5">{{ t(child.desc) }}</div>
                          </div>
                          <span v-if="isChildActive(child.to)"
                            class="w-1.5 h-1.5 rounded-full bg-neutral-900 flex-shrink-0" />
                        </router-link>
                      </MenuItem>
                    </div>
                  </div>
                </MenuItems>
            </Menu>

            <!-- 普通导航项：已登录直接链接 -->
            <router-link v-else-if="!item.requiresAuth || authStore.isLoggedIn" :to="item.to"
              class="relative inline-flex items-center px-4 py-2 text-sm font-medium rounded-lg transition-colors" :class="[
                isActive(item.to)
                  ? 'bg-neutral-100 text-neutral-900'
                  : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900',
              ]">
              <span>{{ t(item.name) }}</span>
              <span v-if="item.badge"
                class="pointer-events-none absolute right-0 top-0 z-20 translate-x-1/2 -translate-y-1/2 inline-flex items-center gap-0.5 rounded-full bg-cyan-600 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-white shadow-sm ring-1 ring-white">
                <span class="h-1 w-1 rounded-full bg-white/90" />
                {{ item.badge }}
              </span>
            </router-link>

            <!-- 普通导航项：未登录弹出登录框 -->
            <button v-else type="button"
              class="relative inline-flex items-center px-4 py-2 text-sm font-medium rounded-lg transition-colors text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900"
              @click="openLoginModalForNav(item)">
              <span>{{ t(item.name) }}</span>
              <span v-if="item.badge"
                class="pointer-events-none absolute right-0 top-0 z-20 translate-x-1/2 -translate-y-1/2 inline-flex items-center gap-0.5 rounded-full bg-cyan-600 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-white shadow-sm ring-1 ring-white">
                <span class="h-1 w-1 rounded-full bg-white/90" />
                {{ item.badge }}
              </span>
            </button>

          </template>
        </nav>

        <div class="flex items-center gap-3">
          <Menu v-if="authStore.isLoggedIn" as="div" class="relative">
            <MenuButton class="flex items-center gap-2 p-1.5 rounded-lg hover:bg-neutral-100 transition-colors">
              <div class="w-8 h-8 bg-neutral-200 rounded-full flex items-center justify-center">
                <span class="text-sm font-medium text-neutral-700">
                  {{ (authStore.user?.name || authStore.user?.username)?.charAt(0)?.toUpperCase() || 'U' }}
                </span>
              </div>
            </MenuButton>
            <transition enter-active-class="transition ease-out duration-100"
              enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100"
              leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100"
              leave-to-class="transform opacity-0 scale-95">
              <MenuItems
                class="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none overflow-hidden">
                <div class="px-4 py-3 border-b border-neutral-100">
                  <p class="text-sm font-medium text-neutral-900">{{ authStore.user?.name || authStore.user?.username }}
                  </p>
                  <p class="text-xs text-neutral-500 truncate">{{ authStore.user?.username }}</p>
                </div>
                <div class="py-1">
                  <MenuItem v-slot="{ active }">
                    <button @click="handleLogout"
                      :class="[active ? 'bg-neutral-50' : '', 'block w-full px-4 py-2 text-sm text-red-600 text-left']">
                      {{ t('退出登录') }}
                    </button>
                  </MenuItem>
                </div>
              </MenuItems>
            </transition>
          </Menu>

          <Button v-else variant="outline" size="sm" @click="authModalOpen = true">
            {{ t('登录') }}
          </Button>

          <button @click="mobileMenuOpen = !mobileMenuOpen"
             :class="navigationLocale === 'en' ? 'lg:hidden' : 'md:hidden'" class="p-2 rounded-lg text-neutral-600 hover:bg-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300"
            :aria-label="mobileMenuOpen ? t('关闭导航') : t('打开导航')">
            <Bars3Icon v-if="!mobileMenuOpen" class="w-6 h-6" />
            <XMarkIcon v-else class="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端菜单 -->
    <TransitionRoot :show="mobileMenuOpen" enter="transition-opacity duration-200" enter-from="opacity-0"
      enter-to="opacity-100" leave="transition-opacity duration-150" leave-from="opacity-100" leave-to="opacity-0">
      <div :class="navigationLocale === 'en' ? 'lg:hidden' : 'md:hidden'" class="border-t border-neutral-200 bg-white">
        <nav class="px-4 py-3 space-y-1">
          <template v-for="item in navigation" :key="item.name">

            <!-- 含子菜单：展开显示子项 -->
            <template v-if="item.children">
              <div class="px-4 pt-3 pb-1 text-sm font-semibold text-neutral-900">
                {{ t(item.name) }}
              </div>
              <div v-for="group in groupNavChildren(item.children)" :key="group.name">
                <div v-if="group.name !== '功能'" class="px-6 py-1.5 text-xs font-semibold text-neutral-400">{{ t(group.name) }}</div>
                <template v-for="child in group.items" :key="child.name">
                  <router-link v-if="!item.requiresAuth || authStore.isLoggedIn" :to="child.to"
                    class="flex items-center gap-3 pl-6 pr-4 py-2.5 text-sm font-medium rounded-lg transition-colors"
                    :class="[
                      isChildActive(child.to)
                        ? 'bg-neutral-100 text-neutral-900'
                        : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900'
                    ]" @click="mobileMenuOpen = false">
                    <component :is="child.icon" class="h-5 w-5 flex-shrink-0 text-neutral-500" aria-hidden="true" />
                    {{ t(child.name) }}
                  </router-link>
                  <button v-else type="button"
                    class="flex items-center gap-3 pl-6 pr-4 py-2.5 w-full text-left text-sm font-medium rounded-lg transition-colors text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900"
                    @click="openLoginModalForNav(item)">
                    <component :is="child.icon" class="h-5 w-5 flex-shrink-0 text-neutral-500" aria-hidden="true" />
                    {{ t(child.name) }}
                  </button>
                </template>
              </div>
            </template>

            <!-- 普通项 -->
            <template v-else>
              <router-link v-if="!item.requiresAuth || authStore.isLoggedIn" :to="item.to"
                class="flex items-center justify-between gap-3 px-4 py-2.5 text-sm font-medium rounded-lg transition-colors" :class="[
                  isActive(item.to)
                    ? 'bg-neutral-100 text-neutral-900'
                    : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900'
                ]" @click="mobileMenuOpen = false">
                <span>{{ t(item.name) }}</span>
                <span v-if="item.badge"
                  class="inline-flex items-center gap-0.5 rounded-full bg-cyan-50 px-1.5 py-0.5 text-[10px] font-semibold text-cyan-700 ring-1 ring-cyan-100">
                  <span class="h-1 w-1 rounded-full bg-cyan-500" />
                  {{ item.badge }}
                </span>
              </router-link>
              <button v-else type="button"
                class="flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm font-medium rounded-lg transition-colors text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900"
                @click="openLoginModalForNav(item)">
                <span>{{ t(item.name) }}</span>
                <span v-if="item.badge"
                  class="inline-flex items-center gap-0.5 rounded-full bg-cyan-50 px-1.5 py-0.5 text-[10px] font-semibold text-cyan-700 ring-1 ring-cyan-100">
                  <span class="h-1 w-1 rounded-full bg-cyan-500" />
                  {{ item.badge }}
                </span>
              </button>
            </template>

          </template>

          <div v-if="!authStore.isLoggedIn" class="pt-2 border-t border-neutral-100 mt-2">
            <button @click="authModalOpen = true; mobileMenuOpen = false"
              class="w-full px-4 py-2.5 text-sm font-medium text-center bg-neutral-900 text-white rounded-lg">
              {{ t('登录') }}
            </button>
          </div>
        </nav>
      </div>
    </TransitionRoot>
  </header>

  <AuthModal v-model:open="authModalOpen" :gate-hint="authGateHint" />
  <FeedbackModal v-model:open="feedbackOpen" />
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TransitionRoot, Menu, MenuButton, MenuItems, MenuItem } from '@headlessui/vue'
import {
  ArrowPathRoundedSquareIcon,
  Bars3Icon,
  ChartBarIcon,
  ChatBubbleLeftRightIcon,
  ClipboardDocumentListIcon,
  CubeTransparentIcon,
  DocumentMagnifyingGlassIcon,
  DocumentArrowDownIcon,
  FolderOpenIcon,
  LanguageIcon,
  PhotoIcon,
  RectangleStackIcon,
  RectangleGroupIcon,
  ShoppingCartIcon,
  ScissorsIcon,
  SparklesIcon,
  TagIcon,
  ViewfinderCircleIcon,
  VideoCameraIcon,
  WrenchScrewdriverIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import Button from '@/components/ui/Button.vue'
import AuthModal from '@/components/ui/AuthModal.vue'
import FeedbackModal from '@/components/ui/FeedbackModal.vue'
import { useAuthStore } from '@/stores/auth'
import { dailyCallLocale, dailyCallText } from '@/utils/dailyCallLocale'
import { isTaxDataUser } from '@/utils/userLandingRoute'

const props = defineProps({ locale: { type: String, default: '' } })
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const navigationLocale = computed(() => props.locale || dailyCallLocale(authStore.user))
const t = text => dailyCallText(navigationLocale.value, text)
const mobileMenuOpen = ref(false)
const authModalOpen = ref(false)
const authGateHint = ref('')
const feedbackOpen = ref(false)
const isAdmin = computed(() => (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin'))

const allNavigation = [
  {
    name: '图像创作',
    requiresAuth: true,
    children: [
      { name: '专业模式', to: '/stp-white-bg', icon: WrenchScrewdriverIcon, section: '核心生成', desc: '使用专业画布完成复杂出图流程' },
      { name: '水龙头场景图生成', to: '/scene-generator', icon: PhotoIcon, section: '核心生成', desc: '生成水龙头场景图' },
      { name: 'AI 图像生成', to: '/generate', icon: SparklesIcon, section: '核心生成', desc: '通过描述或参考图自由生图' },
      { name: '智能生成', to: '/smart-suite', icon: RectangleStackIcon, section: '核心生成', desc: '智能生成listing和套图' },
      { name: '多角度生成', to: '/multi-angle', icon: CubeTransparentIcon, section: '图像处理', desc: '基于现有产品图补全更多展示角度' },
      { name: '景深图生成', to: '/depth', icon: ViewfinderCircleIcon, section: '图像处理', desc: '生成主体清晰、背景虚化的景深图' },
      // { name: '产品融合', to: '/light-shadow', icon: RectangleGroupIcon, section: '图像处理', desc: '产品融合到场景中，光影自然真实' },
      { name: '图片文字翻译', to: '/image-translate', icon: LanguageIcon, section: '图像处理', desc: '将图片中的文字翻译为目标语言' },
      { name: '图片转提示词', to: '/image-to-prompt', icon: DocumentMagnifyingGlassIcon, section: '图像处理', desc: '根据图片反推可复用提示词' },
      { name: '去背景', to: '/white-bg', icon: ScissorsIcon, section: '图像处理', desc: '智能移除背景' },
      { name: '资产库', to: '/assets', icon: FolderOpenIcon, section: '图像资源', desc: '管理和复用图片与设计素材' },
      { name: 'AI图像标注', to: '/ai-image-metadata', icon: DocumentArrowDownIcon,  section: '图像处理', desc: '批量处理 AI 生成的人物图像元数据'},
    ],
  },
  // { name: '视频生成', to: '/video-generate', icon: VideoCameraIcon, requiresAuth: true },
  {
    name: '运营工具',
    requiresAuth: true,
    children: [
      {
        name: '订单转 MCF',
        to: '/order-to-mcf',
        icon: ArrowPathRoundedSquareIcon,
        desc: '将平台订单转换为 Amazon MCF 模板',
      },
      {
        name: '订单汇总',
        to: '/order-tracking',
        icon: ClipboardDocumentListIcon,
        desc: '查看平台订单并回填物流追踪号',
      },
      {
        name: '商品价格',
        to: '/product-prices',
        icon: TagIcon,
        desc: '维护 Amazon 与 TikTok 售价',
      },
      {
        name: '评论分析',
        to: '/review-analysis',
        icon: ChatBubbleLeftRightIcon,
        desc: '导入并分析买家评论 VOC',
      },

      // {
      //   name: '视频需求',
      //   to: '/work-requests',
      //   icon: RectangleStackIcon,
      //   desc: '提交并跟踪视频制作需求',
      // },
      // {
      //   name: '采购计划',
      //   to: '/purchase-plans',
      //   icon: ShoppingCartIcon,
      //   desc: '提交并查看 ERP 采购计划',
      // },
      // {
      //   name: 'Listing 关键词库',
      //   to: '/listing-keywords',
      //   icon: TagIcon,
      //   desc: '维护拓展、过滤、标记与入库流程',
      //   adminOnly: true,
      // },
    ],
  },
  { name: 'Daily Call Registration', to: '/daily-call-registration', requiresAuth: true },
  {
    name: '财务',
    requiresAuth: true,
    children: [
      { name: '汇率管理', to: '/finance/exchange-rates', icon: ChartBarIcon, desc: '查看月度汇率与变化趋势', exchangeRateOnly: true },
      { name: '报税数据', to: '/tax-data', icon: ClipboardDocumentListIcon, desc: '管理原始账单与季度报税汇总', taxDataOnly: true },
      { name: '报关单', to: '/finance/customs-declarations', icon: ClipboardDocumentListIcon, desc: '上传 PDF、识别并保存报关单', taxDataOnly: true },
    ],
  },
  { name: '产品库', to: '/product-library', requiresAuth: true },
]
const navigation = computed(() => {
  // 此副本只提供财务目录，避免继续暴露原系统的设计、运营和商品功能。
  return authStore.isLoggedIn ? allNavigation.filter(item => item.name === '财务') : []
})

function groupNavChildren(children = []) {
  const groups = []
  for (const child of children.filter(item => !item.hidden && (!item.adminOnly || isAdmin.value) && (!item.taxDataOnly || (authStore.isLoggedIn && isTaxDataUser(authStore.user))) && (!item.exchangeRateOnly || authStore.isLoggedIn))) {
    const name = child.section || '功能'
    let group = groups.find(item => item.name === name)
    if (!group) {
      group = { name, items: [] }
      groups.push(group)
    }
    group.items.push(child)
  }
  return groups
}

watch(authModalOpen, (open) => {
  if (!open) authGateHint.value = ''
})

function openLoginModalForNav(item) {
  authGateHint.value = `使用「${item.name}」需要先登录账号`
  authModalOpen.value = true
  mobileMenuOpen.value = false
}

function isActive(path) {
  if (typeof path !== 'string') return false
  if (path === '/') {
    return route.path === '/'
  }
  return route.path === path
}

function normalizePath(target) {
  if (!target) return ''
  if (typeof target === 'string') return target
  return target.path || ''
}

function isChildActive(to) {
  const path = normalizePath(to)
  if (!path) return false
  if (path === '/tools') {
    return route.path === '/tools' || route.path === '/eraser' || route.path === '/remove-bg' || route.path === '/product-replace' || route.path === '/depth' || route.path === '/light-shadow'
  }
  if (typeof to !== 'string' && to.path && to.query?.t) {
    return route.path === to.path && route.query.t === to.query.t
  }
  return route.path === path
}

function handleLogout() {
  authStore.logout()
  mobileMenuOpen.value = false
  router.push('/')
}
</script>
