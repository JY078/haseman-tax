<template>
  <MainLayout>
    <section class="bg-white py-16 border-b border-neutral-200">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="max-w-3xl">
          <p class="text-xs font-semibold tracking-[0.18em] uppercase text-neutral-400 mb-4">内部工作台</p>
          <h1 class="text-4xl sm:text-5xl font-bold text-neutral-900 leading-tight mb-5">聚焦内部出图流程的能力总览</h1>
          <p class="text-lg text-neutral-500 leading-relaxed">本页只保留实际要用的内部生产能力，不再按营销站点方式组织页面与文案。</p>
        </div>
      </div>
    </section>

    <section class="bg-neutral-50 py-14">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-6">
        <div v-for="item in features" :key="item.title" class="grid lg:grid-cols-[1.2fr_0.8fr] gap-6 rounded-3xl border border-neutral-200 bg-white p-7">
          <div>
            <div class="inline-flex px-3 py-1 rounded-full text-xs font-semibold mb-4" :class="item.badgeClass">{{ item.badge }}</div>
            <h2 class="text-2xl sm:text-3xl font-bold text-neutral-900 mb-4">{{ item.title }}</h2>
            <p class="text-neutral-500 leading-relaxed mb-6">{{ item.description }}</p>
            <ul class="space-y-3">
              <li v-for="point in item.points" :key="point" class="flex items-start gap-3 text-sm text-neutral-600 leading-relaxed">
                <span class="mt-1 h-2 w-2 rounded-full bg-neutral-900 flex-shrink-0" />
                <span>{{ point }}</span>
              </li>
            </ul>
          </div>
          <div class="rounded-3xl border border-dashed border-neutral-200 bg-neutral-50 p-6 flex flex-col justify-between">
            <div>
              <p class="text-xs font-semibold text-neutral-400 uppercase tracking-[0.18em] mb-3">适用场景</p>
              <p class="text-sm text-neutral-600 leading-relaxed">{{ item.scene }}</p>
            </div>
            <router-link :to="item.to" class="inline-flex items-center justify-center rounded-2xl bg-neutral-900 px-5 py-3 text-sm font-semibold text-white hover:bg-neutral-800 transition-colors mt-6">
              进入功能
            </router-link>
          </div>
        </div>
      </div>
    </section>
  </MainLayout>
</template>

<script setup>
import MainLayout from '@/components/layout/MainLayout.vue'

const features = [
  {
    badge: '主工作流',
    badgeClass: 'bg-emerald-50 text-emerald-700',
    title: '场景图生成',
    description: '上传产品图后，结合风格标签与参考图，生成用于业务沟通、提案和内部素材制作的高质量场景图。',
    points: ['沿用现有用户体系与任务记录', '直接接入当前 Go 后端，无额外服务拆分', '适合当前迁移方案中的核心新增能力'],
    scene: '适用于新品视觉探索、产品线风格统一、内部方案评审与素材准备。',
    to: '/scene-generator',
  },
  {
    badge: '已有能力',
    badgeClass: 'bg-amber-50 text-amber-700',
    title: '产品图生成',
    description: '从现有参考图与配置出发，快速生成电商主图、详情图等标准产品图。',
    points: ['适合固定模板化产出', '支持批量处理', '保留现有分析与生成链路'],
    scene: '适用于详情页、主图、运营活动物料的常规出图。',
    to: '/product-image',
  },
  {
    badge: '辅助能力',
    badgeClass: 'bg-sky-50 text-sky-700',
    title: '图片工具',
    description: '保留与图片加工相关的高频辅助工具，用于支持主工作流。',
    points: ['图片擦除和去背景适合二次加工', '图片转提示词方便沉淀模板'],
    scene: '适用于主流程之外的补图、修图和复用提效。',
    to: '/tools',
  },
]
</script>
