import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { isCustomerServiceUser, isTaxDataUser, userLandingRoute } from '@/utils/userLandingRoute'
import { useSeo } from '@/composables/useSeo'

const routes = [
  {
    path: '/finance/customs-declarations',
    name: 'customs-declarations',
    component: () => import('@/views/CustomsDeclarationsView.vue'),
    meta: { requiresAuth: true, taxDataOnly: true, seo: { title: '报关单', path: '/finance/customs-declarations' } },
  },
  {
    path: '/smart-suite/team',
    name: 'smart-suite-team',
    component: () => import('@/views/SmartSuiteTeamView.vue'),
    meta: { requiresAuth: true, adminOnly: true, seo: { title: '团队操作记录', path: '/smart-suite/team' } },
  },
  {
    path: '/finance/exchange-rates',
    name: 'finance-exchange-rates',
    component: () => import('@/views/ExchangeRatesView.vue'),
    meta: { requiresAuth: true, exchangeRateOnly: true, seo: { title: '汇率管理', path: '/finance/exchange-rates' } },
  },
  {
    path: '/tax-data',
    name: 'tax-data',
    component: () => import('@/views/TaxDataView.vue'),
    meta: { requiresAuth: true, taxDataOnly: true, seo: { title: '报税数据', description: '原始账单与季度报税数据管理。', path: '/tax-data' } },
  },
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: {
      seo: {
        title: 'HaSeMan 内部工作台',
        description: '面向运营与设计团队的 AI 图片生产工作台，集中提供场景图、产品图、多角度生成与图片处理能力。',
        keywords: '内部工作台,场景图生成,产品图生成,多角度生成,图片处理',
        titleFull: true,
      },
    },
  },
  {
    path: '/features',
    name: 'features',
    component: () => import('@/views/FeaturesView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '功能总览',
        description: '查看内部 AI 图片工作台的主要能力与适用场景。',
        keywords: '功能总览,AI图片工作台,内部设计工具',
        path: '/features',
      },
    },
  },
  {
    path: '/templates',
    name: 'templates',
    component: () => import('@/views/TemplatesView.vue'),
    meta: {
      seo: {
        title: '模板库',
        description: '统一查看和复用内部模板与提示词资源。',
        keywords: '模板库,提示词模板,内部模板',
        path: '/templates',
      },
    },
  },
  {
    path: '/assets',
    name: 'assets',
    component: () => import('@/views/AssetsView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '资产库',
        description: '统一查看和管理系统资产、部门资产、个人资产与共享资产。',
        keywords: '资产库,素材库,部门资产,个人资产,系统资产',
        path: '/assets',
      },
    },
  },
  {
    path: '/daily-call-registration',
    name: 'daily-call-registration',
    component: () => import('@/views/DailyCallRegistrationView.vue'),
    meta: {
      requiresAuth: true,
      seo: { title: 'Daily Call Registration', description: '客户来电登记与质检记录', path: '/daily-call-registration' },
    },
  },
  {
    path: '/product-library',
    name: 'product-library',
    component: () => import('@/views/ProductLibraryView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '产品库',
        description: '按 SPU 管理产品、SKU、模型和认证资料。',
        keywords: '产品库,SPU,SKU,产品模型,认证资料',
        path: '/product-library',
      },
    },
  },
  {
    path: '/product-prices',
    name: 'product-prices',
    component: () => import('@/views/ProductPricesView.vue'),
    meta: { requiresAuth: true, seo: { title: '商品价格', description: '维护 Amazon 与 TikTok 商品价格、PID 和商品链接。', path: '/product-prices' } },
  },
  { path: '/product-price-history', name: 'product-price-history', component: () => import('@/views/ProductPriceHistoryView.vue'), meta: { requiresAuth: true, seo: { title: '价格历史', description: '按天查看商品价格历史。', path: '/product-price-history' } } },
  {
    path: '/scene-generator',
    name: 'scene-generator',
    component: () => import('@/views/SceneGeneratorView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '场景图生成',
        description: '上传产品图、选择风格标签与参考图，生成用于业务出图的高质量场景图。',
        keywords: '场景图生成,产品场景图,AI场景图',
        path: '/scene-generator',
      },
    },
  },
  {
    path: '/image-translate',
    name: 'image-translate',
    component: () => import('@/views/ImageTranslateView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '图片文字翻译',
        description: '上传包含文字的图片，将图片中的语言文字翻译为目标语言，保持原图风格。',
        keywords: '图片翻译,文字翻译,图片本地化,图片语言转换',
        path: '/image-translate',
      },
    },
  },
  {
    path: '/multi-angle',
    name: 'multi-angle',
    component: () => import('@/views/MultiAngleView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '多角度产品图',
        description: 'AI 一键生成多角度产品展示图，全方位展示商品细节。',
        keywords: '多角度产品图,产品展示图,商品效果图,产品图制作',
        path: '/multi-angle',
      },
    },
  },
  {
    path: '/depth',
    name: 'depth',
    component: () => import('@/views/DepthView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '景深图生成',
        description: '上传图片并生成主体清晰、背景自然虚化的景深效果图。',
        keywords: '景深图生成,背景虚化,AI景深图,产品景深图',
        path: '/depth',
      },
    },
  },
  {
    path: '/light-shadow',
    name: 'light-shadow',
    component: () => import('@/views/LightShadowView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '光影优化',
        description: '上传图片，优化产品与背景之间的高光、阴影和接触区域融合效果。',
        keywords: '光影优化,产品图优化,AI图片优化,阴影优化',
        path: '/light-shadow',
      },
    },
  },
  {
    path: '/image-to-prompt',
    name: 'image-to-prompt',
    component: () => import('@/views/ImageToPromptView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '图片转提示词',
        description: '上传图片，由视觉模型分析画面并生成可复用的提示词。',
        keywords: '图片转提示词,反推提示词,AI提示词,图生文',
        path: '/image-to-prompt',
      },
    },
  },
  {
    path: '/tools',
    name: 'tools',
    component: () => import('@/views/ToolsView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '图片工具箱',
        description: '集合抠图、擦除等内部常用图片处理工具。',
        keywords: '图片工具,背景消除,图片擦除,图片处理',
        path: '/tools',
      },
    },
  },
  {
    path: '/eraser',
    name: 'eraser',
    component: () => import('@/views/EraserView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: 'AI图片擦除',
        description: '去除图片中的水印、杂物、文字等不需要元素。',
        keywords: 'AI擦除,图片去水印,图片修复,去除杂物,图片清除',
        path: '/eraser',
      },
    },
  },
  {
    path: '/remove-bg',
    name: 'remove-bg',
    component: () => import('@/views/RemoveBgView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: 'AI智能抠图',
        description: '一键智能抠图，适用于产品和素材加工。',
        keywords: 'AI抠图,背景消除,一键去背景,智能抠图',
        path: '/remove-bg',
      },
    },
  },
  {
    path: '/generate',
    name: 'generate',
    component: () => import('@/views/GenerateView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: 'AI图像生成',
        description: '输入描述或上传参考图，快速生成业务所需图像。',
        keywords: 'AI生图,AI绘画,智能图像生成,AI图片生成',
        path: '/generate',
      },
    },
  },
  {
    path: '/product-replace',
    name: 'product-replace',
    component: () => import('@/views/ProductReplaceView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '产品替换',
        description: '上传产品图进行AI产品替换，快速生成多种场景效果。',
        keywords: '产品替换,AI产品替换,场景替换,产品场景图',
        path: '/product-replace',
      },
    },
  },
  {
    path: '/white-bg',
    name: 'white-bg',
    component: () => import('@/views/WhiteBgView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '白底图生成',
        description: '上传商品图片，快速生成适合电商与素材库使用的白底产品图。',
        keywords: '白底图生成,产品白底图,电商白底图,商品主图',
        path: '/white-bg',
      },
    },
  },
  {
    path: '/product-image',
    name: 'product-image',
    component: () => import('@/views/ProductImageView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '套图生成',
        description: '上传商品图片，自动生成专业产品展示图。',
        keywords: '套图生成,产品效果图,商品图片制作,产品展示图',
        path: '/product-image',
      },
    },
  },
  {
    path: '/video-generate',
    name: 'video-generate',
    component: () => import('@/views/VideoGenerateView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '视频生成',
        description: '使用火山方舟 Seedance 生成产品视频。',
        keywords: '视频生成,Seedance,产品视频,AI视频',
        path: '/video-generate',
      },
    },
  },
  {
    path: '/smart-suite',
    name: 'smart-suite',
    component: () => import('@/views/SmartSuiteView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '智能生成套图',
        description: '基于产品白底图与核心信息，规划副图和 A+ 图内容。',
        keywords: '智能生成套图,副图,A+图,电商内容生成',
        path: '/smart-suite',
      },
    },
  },
  {
    path: '/listing-keywords',
    name: 'listing-keywords',
    component: () => import('@/views/ListingKeywordLibraryView.vue'),
    meta: {
      requiresAuth: true,
      adminOnly: true,
      seo: {
        title: 'Listing 关键词库',
        description: '维护各平台 Listing 关键词的拓展、过滤、标记与入库流程。',
        keywords: 'Listing关键词,关键词库,电商运营',
        path: '/listing-keywords',
      },
    },
  },
  {
    path: '/order-to-mcf',
    name: 'order-to-mcf',
    component: () => import('@/views/OrderToMcfView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '订单转 MCF',
        description: '内部订单转换工具。',
        keywords: '订单转换,MCF',
        path: '/order-to-mcf',
      },
    },
  },
  {
    path: '/review-analysis',
    name: 'review-analysis',
    component: () => import('@/views/ReviewAnalysisView.vue'),
    meta: { requiresAuth: true, seo: { title: '评论分析', description: '导入买家评论并进行 VOC 浏览与筛选。', path: '/review-analysis' } },
  },
  {
    path: '/order-tracking',
    name: 'order-tracking',
    component: () => import('@/views/OrderTrackingView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '订单汇总',
        description: '平台订单与物流追踪管理。',
        keywords: '订单汇总,物流追踪,追踪号',
        path: '/order-tracking',
      },
    },
  },
  {
    path: '/work-requests',
    name: 'work-requests',
    component: () => import('@/views/WorkRequestsView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '需求管理',
        description: '提交并跟踪视频制作需求。',
        keywords: '需求管理,视频需求,镜头需求',
        path: '/work-requests',
      },
    },
  },
  {
    path: '/purchase-plans',
    name: 'purchase-plans',
    component: () => import('@/views/PurchasePlansView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '采购计划',
        description: '提交并查看 ERP 采购计划。',
        keywords: '采购计划,ERP,采购管理',
        path: '/purchase-plans',
      },
    },
  },
  {
    path: '/brand-settings',
    name: 'brand-settings',
    component: () => import('@/views/BrandSettingsView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '品牌设置',
        description: '配置全局品牌名称与 Logo，用于 A+ 图等品牌化内容生成。',
        keywords: '品牌设置,品牌Logo,A+图,品牌资产',
        path: '/brand-settings',
      },
    },
  },
  {
    path: '/history',
    name: 'history',
    component: () => import('@/views/HistoryView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '历史记录',
        description: '统一查看各工具的生成与处理历史，并可选择保存到我的设计。',
        keywords: '历史记录,AI生成历史,图片处理历史',
        path: '/history',
      },
    },
  },
  {
    path: '/ai-image-metadata',
    name: 'ai-image-metadata',
    component: () => import('@/views/SyntheticPerformerView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: 'AI 人物元数据',
        description: '批量为本地图片添加电商平台要求的 AI 人物 XMP 元数据。',
        keywords: 'AI人物元数据,XMP,IPTC,contains-synthetic-performer',
        path: '/ai-image-metadata',
      },
    },
  },
  {
    path: '/my-designs',
    name: 'my-designs',
    component: () => import('@/views/MyDesignsView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '我的作品',
        description: '查看和管理个人历史生成记录。',
        keywords: '我的作品,图像管理,作品下载,设计管理',
        path: '/my-designs',
      },
    },
  },
  {
    path: '/profile',
    name: 'profile',
    component: () => import('@/views/ProfileView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '个人中心',
        description: '管理个人账户与偏好配置。',
        keywords: '个人中心,账户管理,用户信息',
        path: '/profile',
      },
    },
  },
  {
    path: '/stp-white-bg',
    name: 'stp-white-bg',
    component: () => import('@/views/StpWhiteBgView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: '专业画布',
        description: '上传 STP 文件，通过 3D 预览选择角度，生成白底产品图。',
        keywords: 'STP,白底图,3D,CAD',
        path: '/stp-white-bg',
      },
    },
  },
  {
    path: '/maintenance',
    name: 'maintenance',
    component: () => import('@/views/MaintenanceView.vue'),
    meta: {
      seo: {
        title: '系统维护中',
        description: '系统正在进行维护，请稍后再访问。',
        path: '/maintenance',
      },
    },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const FINANCE_PATHS = new Set(['/tax-data', '/finance/exchange-rates', '/finance/customs-declarations'])

function isFinancePath(path) {
  return FINANCE_PATHS.has(path)
}

let maintenanceCache = { enabled: false, checkedAt: 0 }
const CACHE_TTL = 30_000
const MAINTENANCE_POLL_INTERVAL = 10_000

function isAdmin(authStore) {
  return (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin')
}

async function checkMaintenance(force = false) {
  const now = Date.now()
  if (!force && now - maintenanceCache.checkedAt < CACHE_TTL) {
    return maintenanceCache.enabled
  }
  try {
    const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'
    const res = await fetch(`${baseURL}/maintenance-status`, { cache: 'no-store' })
    const data = await res.json()
    maintenanceCache = { enabled: !!data.enabled, checkedAt: now }
    return maintenanceCache.enabled
  } catch {
    return maintenanceCache.enabled
  }
}

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // 这是财务专用 Web 副本。保留首页作为登录入口和维护页，业务页面只允许财务目录。
  if (to.name !== 'home' && to.name !== 'maintenance' && !isFinancePath(to.path)) {
    next({ name: authStore.isLoggedIn ? 'tax-data' : 'home', replace: true })
    return
  }

  if (to.meta.taxDataOnly && authStore.isLoggedIn) {
    try {
      const { default: request } = await import('@/utils/request')
      const response = await request.get('/me')
      authStore.user = response.data
    } catch {
      next({ name: 'generate' })
      return
    }
  }

  if (to.meta.exchangeRateOnly && authStore.isLoggedIn && isCustomerServiceUser(authStore.user) && !isAdmin(authStore)) {
    next({ name: 'daily-call-registration' })
    return
  }

  if (isAdmin(authStore)) {
    if (to.name === 'maintenance') {
      next(userLandingRoute(authStore.user))
      return
    }
  } else {
    const isMaintenance = await checkMaintenance()

    if (isMaintenance && to.name !== 'maintenance') {
      next({ name: 'maintenance' })
      return
    }

    if (!isMaintenance && to.name === 'maintenance') {
      next({ name: 'home' })
      return
    }
  }

  if (authStore.isLoggedIn && isCustomerServiceUser(authStore.user) && !(to.meta.taxDataOnly && isTaxDataUser(authStore.user)) && to.name !== 'daily-call-registration' && to.name !== 'maintenance') {
    next({ name: 'daily-call-registration', replace: true })
    return
  }

  if (to.name === 'home' && authStore.isLoggedIn) {
    next(userLandingRoute(authStore.user))
    return
  }

  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    next({ name: 'home' })
  } else if (to.meta.taxDataOnly && !isTaxDataUser(authStore.user)) {
    next({ name: 'generate' })
  } else if (to.meta.adminOnly && !isAdmin(authStore)) {
    next({ name: 'generate' })
  } else {
    next()
  }
})

router.afterEach((to) => {
  if (to.meta?.seo) {
    const seoConfig = to.meta.seo
    useSeo({
      title: seoConfig.titleFull ? undefined : seoConfig.title,
      description: seoConfig.description,
      keywords: seoConfig.keywords,
      path: seoConfig.path || to.path,
    })
    if (seoConfig.titleFull && seoConfig.title) {
      document.title = seoConfig.title
    }
  }
})

export function startMaintenancePolling() {
  const checkAndRedirect = async () => {
    const authStore = useAuthStore()
    if (isAdmin(authStore)) return

    const isMaintenance = await checkMaintenance(true)
    if (isMaintenance && router.currentRoute.value.name !== 'maintenance') {
      await router.replace({ name: 'maintenance' })
    } else if (!isMaintenance && router.currentRoute.value.name === 'maintenance') {
      await router.replace({ name: 'home' })
    }
  }

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') checkAndRedirect()
  }

  const timer = window.setInterval(checkAndRedirect, MAINTENANCE_POLL_INTERVAL)
  document.addEventListener('visibilitychange', handleVisibilityChange)

  return () => {
    window.clearInterval(timer)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }
}

export default router
