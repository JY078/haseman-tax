<template>
  <MainLayout>
    <div class="min-h-[calc(100vh-4rem)] bg-neutral-50">
      <section class="border-b border-neutral-200 bg-white">
        <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h1 class="text-2xl font-bold text-neutral-900 sm:text-3xl">资产库</h1>
              <div class="mt-3 flex flex-wrap gap-2">
                <span
                  v-for="stat in libraryStats"
                  :key="stat.label"
                  class="rounded-md bg-neutral-100 px-2.5 py-1 text-xs font-semibold text-neutral-600"
                >
                  {{ stat.label }} {{ stat.value }}
                </span>
              </div>
            </div>
            <button
              class="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-neutral-900 px-4 text-sm font-medium text-white transition-colors hover:bg-neutral-800"
              @click="openUpload"
            >
              <CloudArrowUpIcon class="h-4 w-4" />
              上传资产
            </button>
          </div>
        </div>
      </section>

      <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div class="mb-5 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex min-w-0 flex-1 flex-col gap-3 xl:flex-row xl:items-center">
            <div class="relative sm:w-80">
              <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
              <input
                v-model="search"
                class="h-10 w-full rounded-lg border border-neutral-200 bg-white pl-9 pr-3 text-sm text-neutral-900 outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
                placeholder="搜索名称、标签、归属"
              />
            </div>
            <div class="flex gap-2 overflow-x-auto pb-1 xl:pb-0">
              <button
                v-for="scope in scopes"
                :key="scope.value"
                class="inline-flex h-10 shrink-0 items-center gap-2 rounded-lg px-3 text-sm font-medium transition-colors"
                :class="activeScope === scope.value ? 'bg-neutral-900 text-white' : 'border border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'"
                @click="setScope(scope.value)"
              >
                <component :is="scope.icon" class="h-4 w-4" />
                {{ scope.label }}
              </button>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <select
              v-model="sortBy"
              class="h-10 rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-700 outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
            >
              <option value="updated">最近更新</option>
              <option value="used">使用最多</option>
              <option value="name">名称排序</option>
            </select>
          </div>
        </div>

        <div class="mb-5 flex flex-wrap items-center gap-2 text-sm">
          <!-- <button
            class="font-medium transition"
            :class="!activeMajorCategory ? 'text-neutral-900' : 'text-neutral-500 hover:text-neutral-900'"
            @click="goMajorHome"
          >
            资产库
          </button> -->
          <template v-if="activeMajorCategory">
            <span class="text-neutral-300">/</span>
            <button
              class="font-medium transition"
              :class="!activeCategory ? 'text-neutral-900' : 'text-neutral-500 hover:text-neutral-900'"
              @click="activeCategory = ''; activeSubCategory = ''"
            >
              {{ activeMajorCategoryLabel }}
            </button>
          </template>
          <template v-if="activeCategory">
            <span class="text-neutral-300">/</span>
            <span class="font-medium text-neutral-900">{{ activeCategory }}</span>
          </template>
          <template v-if="activeSubCategory">
            <span class="text-neutral-300">/</span>
            <span class="font-medium text-neutral-900">{{ activeSubCategory }}</span>
          </template>
        </div>

        <section v-if="!activeMajorCategory" class="mb-6">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-neutral-900">资产分类</h2>
            </div>
            <button
              v-if="hasActiveFilters"
              class="text-sm font-medium text-neutral-500 transition hover:text-neutral-900"
              @click="resetFilters"
            >
              清空筛选
            </button>
          </div>
          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <button
              v-for="group in majorCategoryCards"
              :key="group.value"
              type="button"
              class="group rounded-lg border bg-white p-3 text-left transition hover:-translate-y-0.5 hover:shadow-md"
              :class="activeMajorCategory === group.value ? 'border-neutral-900 ring-2 ring-neutral-900/10' : 'border-neutral-200'"
              @click="setMajorCategory(group.value)"
            >
              <div class="mb-3 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h3 class="truncate text-base font-semibold text-neutral-900">{{ group.label }}</h3>
                  <p class="mt-1 text-xs text-neutral-500">{{ group.count }} 个资产 · {{ group.subCategoryCount }} 个小分类</p>
                </div>
                <span class="shrink-0 rounded-md bg-neutral-100 px-2 py-1 text-xs font-semibold text-neutral-600">{{ group.shortLabel }}</span>
              </div>
              <div
                class="grid grid-cols-4 gap-1 overflow-hidden rounded-lg bg-neutral-100"
              >
                <div
                  v-for="(preview, index) in group.previews"
                  :key="`${group.value}-${index}`"
                  class="aspect-square min-w-0 bg-neutral-200"
                >
                  <img
                    :src="preview.src"
                    :alt="preview.alt"
                    class="h-full w-full"
                    :class="preview.generated ? 'object-contain p-2' : 'object-cover'"
                    loading="lazy"
                  />
                </div>
              </div>
            </button>
          </div>
        </section>

        <section v-else-if="!activeCategory" class="mb-6">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 shadow-sm">
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-neutral-900">{{ activeMajorCategoryLabel }}</h2>
              <p class="mt-1 text-xs text-neutral-500">{{ categoryLevelHint }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <button
                class="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-semibold text-neutral-700 shadow-sm transition hover:border-neutral-300 hover:bg-neutral-50 hover:text-neutral-900"
                @click="goMajorHome"
              >
                <ChevronLeftIcon class="h-4 w-4" />
                返回大类
              </button>
              <button
                v-if="canRepairCurrentCategories"
                class="inline-flex h-9 items-center justify-center rounded-lg border border-neutral-200 bg-white px-3 text-sm font-semibold text-neutral-700 shadow-sm transition hover:border-neutral-300 hover:bg-neutral-50 hover:text-neutral-900 disabled:cursor-not-allowed disabled:bg-neutral-100 disabled:text-neutral-400"
                :disabled="repairingCategories"
                @click="repairCurrentCategories"
              >
                {{ repairingCategories ? '修复中' : '修复历史分类' }}
              </button>
              <button
                v-if="canCreateCategoryInCurrentView"
                class="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-neutral-900 px-3 text-sm font-semibold text-white shadow-sm transition hover:bg-neutral-800"
                @click="openCategoryCreate"
              >
                <PlusIcon class="h-4 w-4" />
                新建分类
              </button>
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <article
              v-for="category in categoryCards"
              :key="category.value"
              class="group rounded-lg border bg-white p-3 text-left transition hover:-translate-y-0.5 hover:shadow-md"
              :class="activeCategory === category.value ? 'border-neutral-900 ring-2 ring-neutral-900/10' : 'border-neutral-200'"
            >
              <div class="mb-3 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h3 class="truncate text-base font-semibold text-neutral-900">{{ category.label }}</h3>
                  <p class="mt-1 text-xs text-neutral-500">{{ category.description }}</p>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <span class="rounded-md bg-neutral-100 px-2 py-1 text-xs font-semibold text-neutral-600">{{ category.shortLabel }}</span>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md p-1.5 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="管理分类"
                    @click.stop="openCategoryManage(category)"
                  >
                    <Cog6ToothIcon class="h-4 w-4" />
                  </button>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md px-1.5 py-1 text-xs font-semibold text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="上移"
                    @click.stop="moveCategorySort(category, -1)"
                  >
                    ↑
                  </button>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md px-1.5 py-1 text-xs font-semibold text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="下移"
                    @click.stop="moveCategorySort(category, 1)"
                  >
                    ↓
                  </button>
                </div>
              </div>
              <button
                type="button"
                class="block w-full text-left"
                @click="setCategory(category.value)"
              >
                <div
                class="grid grid-cols-4 gap-1 overflow-hidden rounded-lg bg-neutral-100"
                >
                  <div
                    v-for="(preview, index) in category.previews"
                    :key="`${category.value}-${index}`"
                    class="aspect-square min-w-0 bg-neutral-200"
                  >
                    <img
                      :src="preview.src"
                      :alt="preview.alt"
                      class="h-full w-full"
                      :class="preview.generated ? 'object-contain p-2' : 'object-cover'"
                      loading="lazy"
                    />
                  </div>
                </div>
              </button>
            </article>
          </div>
        </section>

        <section v-else-if="isStyleReferenceMajor && !activeSubCategory" class="mb-6">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 shadow-sm">
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-neutral-900">{{ activeCategory }}</h2>
              <p class="mt-1 text-xs text-neutral-500">{{ styleCategoryCards.length }} 个风格，选择后查看参考图。</p>
            </div>
            <button
              class="inline-flex h-9 shrink-0 items-center justify-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-semibold text-neutral-700 shadow-sm transition hover:border-neutral-300 hover:bg-neutral-50 hover:text-neutral-900"
              @click="activeCategory = ''"
            >
              <ChevronLeftIcon class="h-4 w-4" />
              返回空间
            </button>
          </div>

          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <article
              v-for="category in styleCategoryCards"
              :key="category.value"
              class="group rounded-lg border bg-white p-3 text-left transition hover:-translate-y-0.5 hover:shadow-md"
              :class="activeSubCategory === category.value ? 'border-neutral-900 ring-2 ring-neutral-900/10' : 'border-neutral-200'"
            >
              <div class="mb-3 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h3 class="truncate text-base font-semibold text-neutral-900">{{ category.label }}</h3>
                  <p class="mt-1 text-xs text-neutral-500">{{ category.description }}</p>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <span class="rounded-md bg-neutral-100 px-2 py-1 text-xs font-semibold text-neutral-600">{{ category.shortLabel }}</span>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md p-1.5 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="管理风格"
                    @click.stop="openCategoryManage(category)"
                  >
                    <Cog6ToothIcon class="h-4 w-4" />
                  </button>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md px-1.5 py-1 text-xs font-semibold text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="上移"
                    @click.stop="moveCategorySort(category, -1)"
                  >
                    ↑
                  </button>
                  <button
                    v-if="canManageCategoryCard(category)"
                    class="rounded-md px-1.5 py-1 text-xs font-semibold text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                    title="下移"
                    @click.stop="moveCategorySort(category, 1)"
                  >
                    ↓
                  </button>
                </div>
              </div>
              <button type="button" class="block w-full text-left" @click="setSubCategory(category.value)">
                <div class="grid grid-cols-4 gap-1 overflow-hidden rounded-lg bg-neutral-100">
                  <div
                    v-for="(preview, index) in category.previews"
                    :key="`${category.value}-${index}`"
                    class="aspect-square min-w-0 bg-neutral-200"
                  >
                    <img
                      :src="preview.src"
                      :alt="preview.alt"
                      class="h-full w-full"
                      :class="preview.generated ? 'object-contain p-2' : 'object-cover'"
                      loading="lazy"
                    />
                  </div>
                </div>
              </button>
            </article>
          </div>
        </section>

        <main v-else>
            <div class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 shadow-sm">
              <div class="min-w-0">
                <h2 class="truncate text-base font-semibold text-neutral-900">{{ assetListTitle }}</h2>
                <p class="mt-1 text-xs text-neutral-500">共 {{ filteredAssets.length }} 个资产</p>
              </div>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <div v-if="selectedAssets.length" class="flex flex-wrap items-center gap-2 rounded-lg bg-neutral-50 px-2 py-1">
                  <span class="px-1 text-xs font-semibold text-neutral-600">已选 {{ selectedAssets.length }}</span>
                  <select
                    v-model="batchMoveTargetId"
                    class="h-8 rounded-md border border-neutral-200 bg-white px-2 text-xs text-neutral-700 outline-none focus:border-neutral-900"
                  >
                    <option value="">移动到</option>
                    <option v-for="target in batchMoveTargets" :key="target.id" :value="target.id">{{ target.name }}</option>
                  </select>
                  <button
                    class="h-8 rounded-md bg-neutral-900 px-3 text-xs font-semibold text-white disabled:cursor-not-allowed disabled:bg-neutral-300"
                    :disabled="!batchMoveTargetId"
                    @click="moveSelectedAssets"
                  >
                    移动选中
                  </button>
                  <button class="h-8 rounded-md px-2 text-xs font-semibold text-neutral-500 hover:bg-white hover:text-neutral-900" @click="clearAssetSelection">取消选择</button>
                </div>
                <button
                  class="inline-flex h-9 shrink-0 items-center justify-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 text-sm font-semibold text-neutral-700 shadow-sm transition hover:border-neutral-300 hover:bg-neutral-50 hover:text-neutral-900"
                  @click="goBackFromAssetList"
                >
                  <ChevronLeftIcon class="h-4 w-4" />
                  {{ assetListBackText }}
                </button>
              </div>
            </div>

            <div v-if="filteredAssets.length" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              <article
                v-for="asset in filteredAssets"
                :key="asset.id"
                class="group overflow-hidden rounded-lg border border-neutral-200 bg-white transition hover:-translate-y-0.5 hover:shadow-md"
              >
                <button class="block w-full text-left" @click="openDetail(asset)">
                  <div class="relative aspect-[4/3] bg-neutral-100">
                    <label
                      v-if="canMoveAsset(asset)"
                      class="absolute right-3 top-3 z-10 flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg bg-white/90 text-neutral-700 shadow-sm backdrop-blur-sm transition hover:bg-white"
                      title="选择资产"
                      @click.stop
                    >
                      <input
                        type="checkbox"
                        class="h-4 w-4 rounded border-neutral-300 text-neutral-900 focus:ring-neutral-900"
                        :checked="selectedAssetIds.includes(asset.id)"
                        @change="toggleAssetSelection(asset)"
                      />
                    </label>
                    <img
                      :src="asset.cover"
                      :alt="asset.name"
                      class="h-full w-full"
                      :class="asset.coverGenerated ? 'object-contain p-8' : 'object-cover'"
                      loading="lazy"
                    />
                    <div class="absolute left-3 top-3 flex items-center gap-2">
                      <span
                        class="inline-flex items-center rounded-md px-2 py-1 text-[11px] font-semibold"
                        :class="ownerMeta(asset.ownerType).badgeClass"
                      >
                        {{ ownerMeta(asset.ownerType).label }}
                      </span>
                      <span class="rounded-md bg-white/90 px-2 py-1 text-[11px] font-medium text-neutral-600 backdrop-blur-sm">{{ asset.type }}</span>
                    </div>
                    <div class="absolute bottom-3 right-3 rounded-md bg-black/60 px-2 py-1 text-[11px] font-medium text-white backdrop-blur-sm">
                      {{ asset.usage }} 次使用
                    </div>
                  </div>
                  <div class="p-4">
                    <div class="mb-2 flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <h2 class="truncate text-sm font-semibold text-neutral-900">{{ asset.name }}</h2>
                        <p class="mt-1 truncate text-xs text-neutral-500">{{ asset.ownerName }} · {{ asset.updatedAt }}</p>
                      </div>
                      <component :is="ownerMeta(asset.ownerType).icon" class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
                    </div>
                    <div class="flex min-h-6 flex-wrap gap-1.5">
                      <span
                        v-for="tag in asset.tags"
                        :key="tag"
                        class="rounded-md bg-neutral-100 px-2 py-1 text-[11px] font-medium text-neutral-500"
                      >
                        {{ tag }}
                      </span>
                    </div>
                  </div>
                </button>
                <div class="flex items-center justify-between border-t border-neutral-100 px-4 py-3">
                  <span class="text-xs text-neutral-400">{{ asset.permission }}</span>
                  <div class="flex items-center gap-1">
                    <button
                      class="rounded-lg p-1.5 transition hover:bg-neutral-100"
                      :class="asset.favorite ? 'text-amber-500' : 'text-neutral-500 hover:text-neutral-900'"
                      :title="asset.favorite ? '取消收藏' : '收藏'"
                      @click.stop="toggleFavorite(asset)"
                    >
                      <StarIcon class="h-4 w-4" />
                    </button>
                    <button
                      class="rounded-lg p-1.5 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                      title="下载"
                      @click.stop="downloadAsset(asset)"
                    >
                      <ArrowDownTrayIcon class="h-4 w-4" />
                    </button>
                    <button
                      v-if="asset.canManage"
                      class="rounded-lg p-1.5 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900"
                      title="管理"
                      @click.stop="openDetail(asset, true)"
                    >
                      <Cog6ToothIcon class="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </article>
            </div>

            <div v-else class="flex min-h-96 flex-col items-center justify-center rounded-lg border border-dashed border-neutral-300 bg-white text-center">
              <FolderOpenIcon class="mb-3 h-10 w-10 text-neutral-300" />
              <h2 class="text-sm font-semibold text-neutral-800">没有匹配的资产</h2>
              <p class="mt-1 text-sm text-neutral-400">调整筛选条件后再试</p>
            </div>
        </main>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="selectedAsset" class="fixed inset-0 z-50 flex justify-end bg-black/30 backdrop-blur-sm" @click.self="closeDetail">
        <aside class="flex h-full w-full max-w-xl flex-col bg-white shadow-2xl">
          <div class="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
            <div class="min-w-0">
              <h2 class="truncate text-base font-semibold text-neutral-900">{{ selectedAsset.name }}</h2>
              <p class="mt-1 text-xs text-neutral-500">{{ selectedAsset.ownerName }} · {{ selectedAsset.type }}</p>
            </div>
            <button class="rounded-lg p-2 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900" @click="closeDetail">
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>

          <div class="flex-1 overflow-y-auto p-5">
            <img
              :src="selectedAsset.cover"
              :alt="selectedAsset.name"
              class="mb-5 aspect-[4/3] w-full rounded-lg bg-neutral-100"
              :class="selectedAsset.coverGenerated ? 'object-contain p-10' : 'object-cover'"
            />

            <template v-if="detailEditing">
              <div class="space-y-4">
                <div>
                  <label class="mb-1.5 block text-sm font-medium text-neutral-700">资产名称</label>
                  <input v-model="editForm.name" class="h-10 w-full rounded-lg border border-neutral-200 px-3 text-sm outline-none focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10" />
                </div>
                <div class="grid gap-3 sm:grid-cols-2">
                  <div>
                    <label class="mb-1.5 block text-sm font-medium text-neutral-700">类型</label>
                    <select v-model="editForm.type" class="h-10 w-full rounded-lg border border-neutral-200 px-3 text-sm text-neutral-700 outline-none focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10">
                      <option v-for="type in formTypes" :key="type" :value="type">{{ type }}</option>
                    </select>
                  </div>
                  <div>
                    <label class="mb-1.5 block text-sm font-medium text-neutral-700">分类</label>
                    <input
                      v-model="editForm.category"
                      list="edit-category-options"
                      class="h-10 w-full rounded-lg border border-neutral-200 px-3 text-sm outline-none focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
                      placeholder="输入即可在该资产库内创建分类"
                    />
                    <datalist id="edit-category-options">
                      <option v-for="category in editTargetCategories" :key="category" :value="category" />
                    </datalist>
                  </div>
                </div>
                <div>
                  <label class="mb-1.5 block text-sm font-medium text-neutral-700">标签</label>
                  <input v-model="editForm.tagsText" class="h-10 w-full rounded-lg border border-neutral-200 px-3 text-sm outline-none focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10" placeholder="用逗号分隔" />
                </div>
              </div>
            </template>

            <template v-else>
              <div class="grid gap-3 sm:grid-cols-2">
                <div v-for="item in detailRows" :key="item.label" class="rounded-lg border border-neutral-200 p-3">
                  <div class="text-xs text-neutral-400">{{ item.label }}</div>
                  <div class="mt-1 text-sm font-medium text-neutral-900">{{ item.value }}</div>
                </div>
              </div>
              <div class="mt-5">
                <h3 class="mb-2 text-sm font-semibold text-neutral-900">标签</h3>
                <div class="flex flex-wrap gap-2">
                  <span v-for="tag in selectedAsset.tags" :key="tag" class="rounded-lg bg-neutral-100 px-2.5 py-1.5 text-xs font-medium text-neutral-600">{{ tag }}</span>
                </div>
              </div>
            </template>
          </div>

          <div class="flex flex-wrap gap-2 border-t border-neutral-100 p-5">
            <template v-if="detailEditing">
              <button class="flex-1 rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-neutral-800" @click="saveAssetEdit">保存</button>
              <button class="flex-1 rounded-lg border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-700 transition hover:bg-neutral-50" @click="detailEditing = false">取消</button>
            </template>
            <template v-else>
              <button
                v-if="selectedAsset.fileUrl"
                class="flex-1 rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-neutral-800"
                @click="downloadAsset(selectedAsset)"
              >
                下载资产
              </button>
              <button
                v-else
                class="flex-1 cursor-not-allowed rounded-lg bg-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-500"
                disabled
              >
                暂无可下载文件
              </button>
              <button
                v-if="selectedAsset.canManage"
                class="flex-1 rounded-lg border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-700 transition hover:bg-neutral-50"
                @click="startEdit(selectedAsset)"
              >
                编辑信息
              </button>
              <button
                v-if="selectedAsset.canManage"
                class="rounded-lg border border-red-200 px-4 py-2.5 text-sm font-medium text-red-600 transition hover:bg-red-50"
                @click="assetPendingDelete = selectedAsset"
              >
                删除
              </button>
            </template>
          </div>
        </aside>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="categoryManageOpen" class="fixed inset-0 z-[56] flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm" @click.self="closeCategoryManage">
        <div class="w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-neutral-100 px-6 py-5">
            <div>
              <h2 class="text-lg font-semibold text-neutral-900">管理分类</h2>
              <p class="mt-1 text-sm text-neutral-500">{{ categoryManageTarget?.ownerName }} / {{ activeMajorCategoryLabel }}</p>
            </div>
            <button class="rounded-lg p-2 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900" @click="closeCategoryManage">
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>
          <div class="space-y-5 p-6">
            <section class="rounded-xl border border-neutral-200 bg-neutral-50 p-4">
              <div class="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h3 class="text-sm font-semibold text-neutral-900">编辑分类名称</h3>
                  <p class="mt-1 text-xs text-neutral-500">改名后，该分类下的资产会同步归到新名称。</p>
                </div>
                <span class="rounded-md bg-white px-2.5 py-1 text-xs font-semibold text-neutral-600">{{ categoryManageTarget?.count || 0 }} 个资产</span>
              </div>
              <div class="flex flex-col gap-2 sm:flex-row">
                <input
                  v-model.trim="categoryManageForm.name"
                  class="h-10 min-w-0 flex-1 rounded-lg border border-neutral-200 bg-white px-3 text-sm outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
                  placeholder="分类名称"
                />
                <button
                  class="h-10 rounded-lg bg-neutral-900 px-4 text-sm font-semibold text-white transition hover:bg-neutral-800 disabled:cursor-not-allowed disabled:bg-neutral-300"
                  :disabled="categoryManageSaving || !categoryNameChanged"
                  @click="renameCategory"
                >
                  保存名称
                </button>
              </div>
            </section>

            <section v-if="categoryManageTarget?.kind !== 'style-space'" class="rounded-xl border border-neutral-200 bg-white p-4">
              <div class="mb-3">
                <h3 class="text-sm font-semibold text-neutral-900">移动分类资产</h3>
                <p class="mt-1 text-xs text-neutral-500">将当前分类下的资产移动到同资产库、同大类的其他分类。</p>
              </div>
              <div class="flex flex-col gap-2 sm:flex-row">
                <select
                  v-model="categoryManageForm.targetCategoryId"
                  class="h-10 min-w-0 flex-1 rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-700 outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
                >
                  <option value="">选择目标分类</option>
                  <option v-for="target in categoryMoveTargets" :key="target.id" :value="target.id">{{ target.name }} · {{ target.assetCount }} 个资产</option>
                </select>
                <button
                  class="h-10 rounded-lg border border-neutral-200 px-4 text-sm font-semibold text-neutral-700 transition hover:bg-neutral-50 disabled:cursor-not-allowed disabled:bg-neutral-100 disabled:text-neutral-400"
                  :disabled="categoryManageSaving || !categoryManageForm.targetCategoryId || !(categoryManageTarget?.count > 0)"
                  @click="moveCategoryAssets"
                >
                  移动资产
                </button>
              </div>
              <p v-if="!categoryMoveTargets.length" class="mt-2 text-xs text-neutral-400">当前大类下还没有其他可移动的分类。</p>
            </section>

            <section v-if="categoryManageTarget?.kind === 'asset'" class="rounded-xl border border-red-100 bg-red-50 p-4">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 class="text-sm font-semibold text-red-700">删除分类</h3>
                  <p class="mt-1 text-xs text-red-500">{{ categoryManageTarget?.count > 0 ? '分类内还有资产，需先移动资产后才能删除。' : '删除后该分类不会再出现在上传和资产库中。' }}</p>
                </div>
                <button
                  class="h-10 rounded-lg bg-red-600 px-4 text-sm font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:bg-red-200"
                  :disabled="categoryManageSaving || (categoryManageTarget?.count > 0)"
                  @click="deleteCategory"
                >
                  删除分类
                </button>
              </div>
            </section>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="categoryOpen" class="fixed inset-0 z-[55] flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm" @click.self="closeCategoryCreate">
        <div class="w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-neutral-100 px-6 py-5">
            <div>
              <h2 class="text-lg font-semibold text-neutral-900">新建分类</h2>
              <p class="mt-1 text-sm text-neutral-500">{{ categoryCreateHint }}</p>
            </div>
            <button class="rounded-lg p-2 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900" @click="closeCategoryCreate">
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>
          <div class="space-y-5 p-6">
            <div class="grid gap-3 rounded-xl border border-neutral-200 bg-neutral-50 p-4 sm:grid-cols-2">
              <div class="rounded-lg bg-white p-3 shadow-sm">
                <span class="block text-xs font-medium text-neutral-500">资产库</span>
                <div class="mt-2 flex items-center gap-2 text-sm font-semibold text-neutral-900">
                  <component :is="categoryCreateOwner.icon" class="h-4 w-4 text-neutral-500" />
                  {{ categoryCreateOwner.label }}
                </div>
              </div>
              <div class="rounded-lg bg-white p-3 shadow-sm">
                <span class="block text-xs font-medium text-neutral-500">资产大类</span>
                <div class="mt-2 text-sm font-semibold text-neutral-900">{{ categoryCreateMajor.label }}</div>
              </div>
            </div>
            <div>
              <label class="mb-2 block text-sm font-semibold text-neutral-800">分类名称</label>
              <input v-model.trim="categoryForm.name" class="h-11 w-full rounded-lg border border-neutral-200 px-3 text-sm outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10" placeholder="例如：厨房白底图" />
            </div>
          </div>
          <div class="flex justify-end gap-2 border-t border-neutral-100 bg-neutral-50 px-6 py-4">
            <button class="rounded-lg border border-neutral-200 px-4 py-2 text-sm font-medium text-neutral-700 transition hover:bg-neutral-50" @click="closeCategoryCreate">取消</button>
            <button
              class="rounded-lg bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-neutral-800 disabled:cursor-not-allowed disabled:bg-neutral-300"
              :disabled="creatingCategory"
              @click="createAssetCategory"
            >
              {{ creatingCategory ? '创建中' : '创建分类' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="uploadOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm" @click.self="closeUpload">
        <div class="max-h-[92vh] w-full max-w-6xl overflow-hidden rounded-xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-neutral-100 px-6 py-5">
            <div>
              <h2 class="text-lg font-semibold text-neutral-900">上传资产</h2>
              <p class="mt-1 text-sm text-neutral-500">{{ uploadLibraryHint }}</p>
            </div>
            <button class="rounded-lg p-2 text-neutral-500 transition hover:bg-neutral-100 hover:text-neutral-900" @click="closeUpload">
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>

          <div class="grid max-h-[calc(92vh-9rem)] overflow-y-auto lg:grid-cols-[360px_minmax(0,1fr)]">
            <aside class="border-b border-neutral-100 bg-neutral-50 p-6 lg:border-b-0 lg:border-r">
              <div class="space-y-6">
                <div>
                  <label class="mb-2 block text-sm font-semibold text-neutral-800">上传到</label>
                  <div class="grid gap-2">
                    <button
                      v-for="target in availableUploadTargets"
                      :key="target.value"
                      type="button"
                      class="flex items-center gap-3 rounded-lg border px-3 py-3 text-left transition"
                      :class="uploadForm.ownerType === target.value ? 'border-neutral-900 bg-neutral-900 text-white shadow-sm' : 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50'"
                      @click="uploadForm.ownerType = target.value"
                    >
                      <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" :class="uploadForm.ownerType === target.value ? 'bg-white/10' : 'bg-neutral-100'">
                        <component :is="target.icon" class="h-5 w-5" />
                      </span>
                      <span class="block min-w-0 flex-1">
                        <span class="block truncate text-sm font-semibold">{{ target.label }}</span>
                        <span class="mt-0.5 block truncate text-xs opacity-70">{{ target.scopeText }}</span>
                      </span>
                      <CheckIcon v-if="uploadForm.ownerType === target.value" class="h-4 w-4 shrink-0" />
                    </button>
                  </div>
                </div>

                <div>
                  <div class="mb-2 flex items-center justify-between gap-3">
                    <label class="block text-sm font-semibold text-neutral-800">{{ uploadCategoryPickerTitle }}</label>
                    <button v-if="canCreateUploadCategory" type="button" class="text-xs font-semibold text-neutral-500 transition hover:text-neutral-900" @click="openCategoryCreate">
                      新建
                    </button>
                  </div>
                  <div class="relative">
                    <button
                      type="button"
                      class="flex min-h-14 w-full items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white px-3 text-left text-sm outline-none transition hover:border-neutral-300 focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10"
                      @click="categoryDropdownOpen = !categoryDropdownOpen"
                    >
                      <span class="min-w-0">
                        <span class="block truncate font-semibold text-neutral-900">{{ selectedUploadCategory?.name || '选择分类' }}</span>
                        <span class="mt-0.5 block truncate text-xs text-neutral-500">{{ selectedUploadCategoryMeta }}</span>
                      </span>
                      <ChevronDownIcon class="h-4 w-4 shrink-0 text-neutral-400" :class="{ 'rotate-180': categoryDropdownOpen }" />
                    </button>
                    <div
                      v-if="categoryDropdownOpen"
                      class="absolute z-20 mt-2 max-h-80 w-full overflow-y-auto rounded-lg border border-neutral-200 bg-white p-1.5 shadow-xl"
                    >
                      <button
                        v-for="category in uploadCategoryOptions"
                        :key="category.key"
                        type="button"
                        class="flex w-full items-center gap-3 rounded-md px-3 py-3 text-left transition hover:bg-neutral-50"
                        :class="uploadCategoryKey === category.key ? 'bg-neutral-100' : ''"
                        @click="selectUploadCategory(category)"
                      >
                        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-neutral-100 text-xs font-semibold text-neutral-600">
                          {{ majorCategoryLabelFromType(category.type).slice(0, 2) }}
                        </span>
                        <span class="min-w-0 flex-1">
                          <span class="block truncate text-sm font-semibold text-neutral-900">{{ category.name }}</span>
                          <span class="mt-0.5 block truncate text-xs text-neutral-500">{{ categoryOptionMeta(category) }}</span>
                        </span>
                        <CheckIcon v-if="uploadCategoryKey === category.key" class="h-4 w-4 shrink-0 text-neutral-900" />
                      </button>
                      <button
                        v-if="canCreateUploadCategory && !uploadCategoryOptions.length"
                        type="button"
                        class="w-full rounded-md px-3 py-3 text-left text-sm font-medium text-neutral-600 hover:bg-neutral-50"
                        @click="openCategoryCreate"
                      >
                        新建第一个分类
                      </button>
                    </div>
                  </div>
                  <p class="mt-2 text-xs leading-5 text-neutral-500">{{ uploadCategoryRuleText }}</p>
                </div>

                <div>
                  <label class="mb-2 block text-sm font-semibold text-neutral-800">标签</label>
                  <input v-model="uploadForm.tagsText" class="h-11 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm outline-none transition focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10" placeholder="用逗号分隔" />
                </div>
              </div>
            </aside>

            <section class="space-y-5 p-6">
              <div class="rounded-xl border border-neutral-200 bg-white p-4">
                <div class="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <h3 class="text-sm font-semibold text-neutral-900">文件</h3>
                    <p class="mt-1 text-xs text-neutral-500">{{ uploadFileSummary }}</p>
                  </div>
                  <span class="rounded-md bg-neutral-100 px-2.5 py-1 text-xs font-semibold text-neutral-600">{{ uploadFiles.length }} 个文件</span>
                </div>
                <label class="flex min-h-[360px] cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed border-neutral-300 bg-neutral-50 px-5 py-8 text-center transition hover:border-neutral-400 hover:bg-white">
                  <input class="hidden" type="file" multiple accept="image/*,video/*,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx" :disabled="uploading" @change="handleFileChange" />
                  <div v-if="uploadPreviewItems.length" class="mb-5 grid max-h-[360px] w-full max-w-2xl grid-cols-3 gap-3 overflow-y-auto p-1 sm:grid-cols-4">
                    <div
                      v-for="(item, index) in uploadPreviewItems"
                      :key="item.id"
                      class="group relative aspect-square overflow-hidden rounded-lg bg-white ring-1 ring-neutral-200"
                    >
                      <img v-if="item.preview" :src="item.preview" :alt="item.name" class="h-full w-full object-cover" />
                      <div v-else class="flex h-full items-center justify-center px-2 text-center text-[11px] font-medium text-neutral-500">{{ item.name }}</div>
                      <button
                        type="button"
                        class="absolute right-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-white/95 text-neutral-500 opacity-0 shadow-sm ring-1 ring-neutral-200 transition hover:text-neutral-900 group-hover:opacity-100"
                        :disabled="uploading"
                        @click.prevent="removeUploadFile(index)"
                      >
                        <XMarkIcon class="h-4 w-4" />
                      </button>
                      <div class="absolute inset-x-0 bottom-0 bg-white/90 px-2 py-1 text-left">
                        <p class="truncate text-[11px] font-medium text-neutral-700">{{ item.name }}</p>
                      </div>
                    </div>
                  </div>
                  <div v-else class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-white ring-1 ring-neutral-200">
                    <CloudArrowUpIcon class="h-8 w-8 text-neutral-500" />
                  </div>
                  <p class="text-sm font-semibold text-neutral-900">选择或拖拽文件到这里</p>
                  <p class="mt-1 text-xs text-neutral-500">支持图片、模型文件、文档和视频</p>
                </label>
                <div v-if="uploadFiles.length" class="mt-3 flex flex-wrap items-center justify-between gap-2">
                  <p class="text-xs text-neutral-500">{{ uploadProgressText }}</p>
                  <div class="flex gap-2">
                    <label class="cursor-pointer rounded-lg border border-neutral-200 bg-white px-3 py-2 text-xs font-semibold text-neutral-700 transition hover:bg-neutral-50" :class="{ 'pointer-events-none opacity-50': uploading }">
                      继续添加
                      <input class="hidden" type="file" multiple accept="image/*,video/*,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx" :disabled="uploading" @change="handleFileChange" />
                    </label>
                    <button type="button" class="rounded-lg border border-neutral-200 bg-white px-3 py-2 text-xs font-semibold text-neutral-700 transition hover:bg-neutral-50 disabled:cursor-not-allowed disabled:opacity-50" :disabled="uploading" @click="clearUploadFiles">
                      清空
                    </button>
                  </div>
                </div>
              </div>
            </section>
          </div>

          <div class="flex flex-col gap-3 border-t border-neutral-100 bg-neutral-50 px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="min-w-0 text-sm text-neutral-500">
              <span class="font-medium text-neutral-800">{{ selectedUploadTarget.label }}</span>
              <span class="mx-1 text-neutral-300">/</span>
              <span>{{ selectedUploadCategory?.name || '未选择分类' }}</span>
            </div>
            <div class="flex justify-end gap-2">
            <button class="rounded-lg border border-neutral-200 px-4 py-2 text-sm font-medium text-neutral-700 transition hover:bg-neutral-50" @click="closeUpload">取消</button>
            <button
              class="rounded-lg bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-neutral-800 disabled:cursor-not-allowed disabled:bg-neutral-300"
              :disabled="uploading"
              @click="createAsset"
            >
              {{ uploadSubmitText }}
            </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="assetPendingDelete" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm" @click.self="assetPendingDelete = null">
        <div class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl">
          <h2 class="text-base font-semibold text-neutral-900">删除资产</h2>
          <p class="mt-2 text-sm leading-6 text-neutral-500">删除后会从当前资产库移除。这个操作只影响当前前端演示数据。</p>
          <div class="mt-5 flex justify-end gap-2">
            <button class="rounded-lg border border-neutral-200 px-4 py-2 text-sm font-medium text-neutral-700 transition hover:bg-neutral-50" @click="assetPendingDelete = null">取消</button>
            <button class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-red-700" @click="deleteAsset">确认删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import request, { getDisplayImageUrl, getDisplayThumbnailUrl } from '@/utils/request'
import {
  ArrowDownTrayIcon,
  BuildingOffice2Icon,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  CloudArrowUpIcon,
  Cog6ToothIcon,
  FolderIcon,
  FolderOpenIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  RectangleStackIcon,
  ShieldCheckIcon,
  StarIcon,
  UserCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

const toast = useToast()
const authStore = useAuthStore()

const search = ref('')
const activeScope = ref('all')
const activeMajorCategory = ref('')
const activeCategory = ref('')
const activeSubCategory = ref('')
const typeFilter = ref('all')
const sortBy = ref('updated')
const selectedAsset = ref(null)
const detailEditing = ref(false)
const uploadOpen = ref(false)
const uploadFiles = ref([])
const uploadPreviewItems = ref([])
const uploading = ref(false)
const uploadingIndex = ref(0)
const categoryDropdownOpen = ref(false)
const categoryOpen = ref(false)
const creatingCategory = ref(false)
const categoryManageOpen = ref(false)
const categoryManageTarget = ref(null)
const categoryManageSaving = ref(false)
const assetPendingDelete = ref(null)
const loading = ref(false)
const assetCategories = ref([])
const sceneStyleCategories = ref([])
const sceneStyleTags = ref([])
const selectedAssetIds = ref([])
const batchMoveTargetId = ref('')
const repairingCategories = ref(false)

const formTypes = ['产品图', '场景图', '背景', '风格参考图', '色块', '参考方案图', '模型文件', '模板', '图标', '视频', '文档', '规范', '组件']

const majorCategoryDefs = [
  { label: '产品素材', value: 'product', shortLabel: '产品', types: ['产品图', '图片', '产品附件'] },
  { label: '模型文件', value: 'model', shortLabel: '模型', types: ['模型文件', 'STP模型'] },
  { label: '风格参考图', value: 'style_reference', shortLabel: '风格', types: ['风格参考图'] },
  { label: '色块图', value: 'color_swatch', shortLabel: '色块', types: ['色块'] },
  { label: '背景场景', value: 'scene', shortLabel: '场景', types: ['背景', '场景图'] },
  { label: '参考方案', value: 'reference', shortLabel: '方案', types: ['参考方案图'] },
  { label: '模板组件', value: 'template', shortLabel: '模板', types: ['模板', '组件', '图标'] },
  { label: '文档规范', value: 'document', shortLabel: '文档', types: ['文档', '规范', '视频'] },
  { label: '其他资产', value: 'other', shortLabel: '其他', types: [] },
]

const scopes = [
  { label: '全部', value: 'all', icon: RectangleStackIcon },
  { label: '系统资产', value: 'system', icon: ShieldCheckIcon },
  { label: '部门资产', value: 'department', icon: BuildingOffice2Icon },
  { label: '我的资产', value: 'personal', icon: UserCircleIcon },
  { label: '已分享', value: 'shared', icon: FolderIcon },
  { label: '收藏', value: 'favorite', icon: StarIcon },
]

const uploadTargets = [
  {
    label: '我的资产',
    value: 'personal',
    icon: UserCircleIcon,
    scopeText: '仅自己可用',
  },
  {
    label: '部门资产',
    value: 'department',
    icon: BuildingOffice2Icon,
    scopeText: '同部门共用',
  },
  {
    label: '系统资产',
    value: 'system',
    icon: ShieldCheckIcon,
    scopeText: '全员可用',
  },
]

const uploadForm = reactive({
  ownerType: 'personal',
  type: '产品图',
  category: '产品图',
  tagsText: '',
})

const categoryForm = reactive({
  ownerType: 'personal',
  majorCategory: 'product',
  name: '',
})
const categoryManageForm = reactive({
  name: '',
  targetCategoryId: '',
})

const editForm = reactive({
  id: null,
  name: '',
  type: '',
  category: '',
  tagsText: '',
})

const assets = ref([])

const scopedCategoryValues = ['system', 'department', 'personal', 'shared']

const isAdmin = computed(() => {
  const roles = authStore.user?.roles || []
  return roles.some(role => role?.name === 'admin' || role === 'admin')
})
const availableUploadTargets = computed(() => uploadTargets.filter(target => target.value !== 'system' || isAdmin.value))
const canCreateCategoryInCurrentView = computed(() => activeScope.value !== 'system' || isAdmin.value)
const canCreateUploadCategory = computed(() => canCreateCategoryForOwner(uploadForm.ownerType))
const canRepairCurrentCategories = computed(() => {
  return ['system', 'department', 'personal'].includes(activeScope.value)
    && !!activeMajorCategory.value
    && !isStyleReferenceMajor.value
    && canCreateCategoryForOwner(activeScope.value)
})
const isScopedLibrary = computed(() => scopedCategoryValues.includes(activeScope.value))
const scopeAssets = computed(() => assets.value.filter(scopeMatchesAsset))
const majorFilteredScopeAssets = computed(() => scopeAssets.value.filter(asset => majorCategoryMatchesAsset(asset)))
const isStyleReferenceMajor = computed(() => activeMajorCategory.value === 'style_reference')
const scopedCategoryAssets = computed(() => {
  if (!isScopedLibrary.value) return []
  return majorFilteredScopeAssets.value.filter(asset => asset.ownerType === activeScope.value)
})
const categories = computed(() => {
	if (!activeMajorCategory.value) return []
	return categoryRowsFor(activeScope.value, activeMajorCategory.value).map(item => item.name)
})
const assetTypes = computed(() => [...new Set([...formTypes, ...assets.value.map(asset => asset.type)])])
const majorCategoryCards = computed(() => {
  return majorCategoryDefs
    .map(def => createMajorCategoryCard({
      ...def,
      assets: scopeAssets.value.filter(asset => majorCategoryOf(asset) === def.value),
    }))
    .filter(card => card.count > 0)
})
const categoryCards = computed(() => {
	return categories.value.map(category => {
		const groupAssets = categoryAssetsFor(activeScope.value, activeMajorCategory.value, category)
		const explicit = categoryRowsFor(activeScope.value, activeMajorCategory.value).find(item => item.name === category)
		return createCategoryCard(category, groupAssets, isStyleReferenceMajor.value ? 'space' : 'category', explicit)
	})
})
const styleCategories = computed(() => {
  if (!isStyleReferenceMajor.value || !activeCategory.value) return []
  const baseAssets = isScopedLibrary.value ? scopedCategoryAssets.value : majorFilteredScopeAssets.value
  const assetsInScene = baseAssets.filter(asset => categoryGroupKey(asset) === activeCategory.value)
  return [...new Set(assetsInScene.map(styleGroupKey).filter(Boolean))]
})
const styleCategoryCards = computed(() => {
  const baseAssets = isScopedLibrary.value ? scopedCategoryAssets.value : majorFilteredScopeAssets.value
  return styleCategories.value.map(style => {
    const groupAssets = baseAssets.filter(asset => categoryGroupKey(asset) === activeCategory.value && styleGroupKey(asset) === style)
    return createCategoryCard(style, groupAssets, 'style')
  })
})
const activeMajorCategoryLabel = computed(() => {
  if (!activeMajorCategory.value) return '资产库'
  return majorCategoryDefs.find(item => item.value === activeMajorCategory.value)?.label || '资产'
})
const categoryLevelHint = computed(() => {
  if (isStyleReferenceMajor.value) return '先选择厨房/浴室等空间，再选择空间下的风格。'
  return '选择一个小分类后查看具体资产。'
})
const assetListTitle = computed(() => {
  if (isStyleReferenceMajor.value) return `${activeMajorCategoryLabel.value} / ${activeCategory.value} / ${activeSubCategory.value}`
  return `${activeMajorCategoryLabel.value} / ${activeCategory.value}`
})
const assetListBackText = computed(() => isStyleReferenceMajor.value ? '返回风格' : '返回小分类')
const uploadCategoryOptions = computed(() => {
  return assetCategories.value
    .filter(category => category.ownerType === uploadForm.ownerType)
    .map(category => ({
      ...category,
      key: `${category.type}|||${category.name}`,
      majorCategory: majorCategoryFromType(category.type),
      label: `${majorCategoryLabelFromType(category.type)} / ${category.name}`,
    }))
})
const uploadCategoryKey = computed({
  get() {
    if (!uploadForm.type || !uploadForm.category) return ''
    return `${uploadForm.type}|||${uploadForm.category}`
  },
  set(value) {
    const [type, category] = String(value || '').split('|||')
    uploadForm.type = type || ''
    uploadForm.category = category || ''
  },
})
const selectedUploadCategory = computed(() => {
  return uploadCategoryOptions.value.find(item => item.key === uploadCategoryKey.value) || null
})
const selectedUploadTarget = computed(() => uploadTargets.find(target => target.value === uploadForm.ownerType) || uploadTargets[0])
const categoryCreateOwner = computed(() => uploadTargets.find(target => target.value === categoryForm.ownerType) || uploadTargets[0])
const categoryCreateMajor = computed(() => majorCategoryDefs.find(item => item.value === categoryForm.majorCategory) || majorCategoryDefs[0])
const uploadCategoryPickerTitle = computed(() => {
  if (uploadForm.ownerType === 'department') return '部门共用分类'
  if (uploadForm.ownerType === 'system') return '系统分类'
  return '我的分类'
})
const uploadCategoryRuleText = computed(() => {
  if (uploadForm.ownerType === 'department') return '同部门成员创建的分类会显示在这里，所有同部门成员都可以把资产放入这些分类。'
  if (uploadForm.ownerType === 'system') return '系统分类由管理员维护，上传后全员可在系统资产库中使用。'
  return '个人分类只在你的资产库中可见，适合存放自己的常用素材。'
})
const uploadLibraryHint = computed(() => {
  if (uploadForm.ownerType === 'department') return '当前资产会进入部门资产库，并放入部门共用分类。'
  if (uploadForm.ownerType === 'system') return '当前资产会进入系统资产库，仅管理员可以维护。'
  return '当前资产会进入你的个人资产库，并放入个人分类。'
})
const categoryCreateHint = computed(() => {
  return `将在 ${categoryCreateOwner.value.label} / ${categoryCreateMajor.value.label} 下创建分类。`
})
const categoryNameChanged = computed(() => {
  return !!categoryManageTarget.value && categoryManageForm.name.trim() && categoryManageForm.name.trim() !== categoryManageTarget.value.name
})
const categoryMoveTargets = computed(() => {
  const target = categoryManageTarget.value
  if (!target) return []
  if (target.kind === 'style') {
    return styleCategoryCards.value
      .filter(category => Number(category.id) > 0 && Number(category.id) !== Number(target.id))
      .map(category => ({
        id: category.id,
        name: category.name,
        assetCount: category.count,
      }))
  }
  return assetCategories.value.filter(category => {
    return Number.isFinite(Number(category.id))
      && Number(category.id) > 0
      && Number(category.id) !== Number(target.id)
      && category.ownerType === target.ownerType
      && category.type === target.type
      && category.canManage
  })
})
const selectedAssets = computed(() => {
  const ids = new Set(selectedAssetIds.value)
  return filteredAssets.value.filter(asset => ids.has(asset.id))
})
const batchMoveTargets = computed(() => {
  if (!selectedAssets.value.length) return []
  if (isStyleReferenceMajor.value) {
    return styleCategoryCards.value
      .filter(category => Number(category.id) > 0 && category.value !== activeSubCategory.value)
      .map(category => ({
        id: category.id,
        name: category.name,
        assetCount: category.count,
      }))
  }
  const first = selectedAssets.value[0]
  if (!first) return []
  return assetCategories.value.filter(category => {
    return Number.isFinite(Number(category.id))
      && Number(category.id) > 0
      && category.ownerType === first.ownerType
      && category.type === first.type
      && category.name !== activeCategory.value
      && category.canManage
  })
})
const canMoveSelectedAssets = computed(() => selectedAssets.value.length > 0 && batchMoveTargets.value.length > 0)
const selectedUploadCategoryMeta = computed(() => {
  if (!selectedUploadCategory.value) return uploadCategoryOptions.value.length ? '请选择当前资产库中的分类' : '先创建分类，再上传资产'
  return categoryOptionMeta(selectedUploadCategory.value)
})
const uploadFileSummary = computed(() => {
  const count = uploadFiles.value.length
  if (!count) return '选择文件'
  if (count === 1) return uploadFiles.value[0].name
  return `已选择 ${count} 个文件`
})
const uploadProgressText = computed(() => {
  const count = uploadFiles.value.length
  if (!count) return '未选择文件'
  if (uploading.value) return `正在上传 ${uploadingIndex.value}/${count}`
  return `已加入 ${count} 个文件，确认后会批量上传到同一个分类`
})
const uploadSubmitText = computed(() => {
  if (uploading.value) return uploadProgressText.value
  const count = uploadFiles.value.length
  if (count > 1) return `确认上传 ${count} 个资产`
  return '确认上传'
})
const libraryStats = computed(() => [
  { label: '资产', value: assets.value.length },
  { label: '分类', value: new Set(assets.value.map(asset => `${asset.type}:${asset.category}`).filter(Boolean)).size },
  { label: '收藏', value: assets.value.filter(asset => asset.favorite).length },
])
const editTargetCategories = computed(() => {
  const asset = assets.value.find(item => item.id === editForm.id)
  return categoryOptionsForOwner(asset?.ownerType)
})

const hasActiveFilters = computed(() => {
  return search.value || activeScope.value !== 'all' || activeMajorCategory.value || activeCategory.value || activeSubCategory.value
})

const filteredAssets = computed(() => {
  if (!activeMajorCategory.value || !activeCategory.value) return []
  if (isStyleReferenceMajor.value && !activeSubCategory.value) return []
  const keyword = search.value.trim().toLowerCase()
  const list = assets.value.filter((asset) => {
    const scopeMatched = scopeMatchesAsset(asset)
    const majorMatched = majorCategoryMatchesAsset(asset)
    const categoryMatched = categoryGroupKey(asset) === activeCategory.value
    const subCategoryMatched = !isStyleReferenceMajor.value || styleGroupKey(asset) === activeSubCategory.value
    return scopeMatched && majorMatched && categoryMatched && subCategoryMatched && (!keyword || assetMatchesKeyword(asset, keyword))
  })

  return [...list].sort((a, b) => {
    if (sortBy.value === 'used') return b.usage - a.usage
    if (sortBy.value === 'name') return a.name.localeCompare(b.name, 'zh-CN')
    return b.id - a.id
  })
})

const detailRows = computed(() => {
  if (!selectedAsset.value) return []
  return [
    { label: '归属', value: selectedAsset.value.ownerName },
    { label: '权限', value: selectedAsset.value.permission },
    { label: '分类', value: selectedAsset.value.category },
    { label: '使用次数', value: `${selectedAsset.value.usage} 次` },
    { label: '更新时间', value: selectedAsset.value.updatedAt },
    { label: '管理权限', value: selectedAsset.value.canManage ? '可管理' : '仅可使用' },
  ]
})

onMounted(() => {
  Promise.all([fetchAssets(), fetchAssetCategories(), fetchSceneStyleMeta()])
})

watch(categories, (nextCategories) => {
  if (!nextCategories.includes(activeCategory.value)) {
    activeCategory.value = ''
    activeSubCategory.value = ''
  }
})
watch(styleCategories, (nextCategories) => {
  if (!nextCategories.includes(activeSubCategory.value)) {
    activeSubCategory.value = ''
  }
})

watch(majorCategoryCards, (nextCards) => {
  if (!nextCards.some(card => card.value === activeMajorCategory.value)) {
    activeMajorCategory.value = ''
    activeCategory.value = ''
    activeSubCategory.value = ''
  }
})

watch(() => uploadForm.ownerType, () => {
  categoryDropdownOpen.value = false
  if (!uploadCategoryOptions.value.some(item => item.key === uploadCategoryKey.value)) {
    const first = uploadCategoryOptions.value[0]
    uploadCategoryKey.value = first?.key || ''
  }
})

async function fetchAssets() {
  loading.value = true
  try {
    const res = await request.get('/assets')
    assets.value = (res.data?.items || []).map(mapAssetFromApi)
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产库加载失败')
  } finally {
    loading.value = false
  }
}

async function fetchAssetCategories() {
  try {
    const res = await request.get('/asset-categories')
    assetCategories.value = (res.data?.items || []).map(mapAssetCategoryFromApi)
  } catch (error) {
    assetCategories.value = []
    toast.error(error?.response?.data?.error || '资产分类加载失败')
  }
}

async function fetchSceneStyleMeta() {
  try {
    const [categoryRes, tagRes] = await Promise.all([
      request.get('/scene-style-categories'),
      request.get('/scene-style-tags'),
    ])
    sceneStyleCategories.value = Array.isArray(categoryRes.data) ? categoryRes.data : []
    sceneStyleTags.value = Array.isArray(tagRes.data) ? tagRes.data : []
  } catch (error) {
    sceneStyleCategories.value = []
    sceneStyleTags.value = []
  }
}

function mapAssetFromApi(item) {
  const sourceType = item.source_type || 'asset'
  const sourceId = item.source_id || ''
  return {
    id: item.id || `${sourceType}:${sourceId}`,
    sourceType,
    sourceId,
    name: item.name || '未命名资产',
    type: item.type || '文件',
    category: item.category || item.type || '未分类',
    ownerType: item.owner_type || 'personal',
    ownerName: item.owner_name || '',
    permission: item.permission || '',
    visibility: item.visibility || 'private',
    usage: item.usage || 0,
    updatedAt: item.updated_at || '',
    canManage: !!item.can_manage,
    favorite: !!item.favorite,
    tags: Array.isArray(item.tags) ? item.tags : [],
    ...assetPreviewFromApi(item),
    fileUrl: item.file_url || '',
    fileName: item.file_name || '',
    fileType: item.file_type || '',
    spu: item.spu || '',
    sku: item.sku || '',
    productId: item.product_id || 0,
    productPath: item.product_path || '',
    referenceSceneId: item.reference_scene_id || 0,
    referenceSceneName: item.reference_scene_name || '',
    referenceTagId: item.reference_tag_id || 0,
    referenceTagName: item.reference_tag_name || '',
    referenceGalleryId: item.reference_gallery_id || 0,
  }
}

function mapAssetCategoryFromApi(item) {
  return {
    id: item.id || `${item.owner_type}:${item.type}:${item.name}`,
    name: item.name || '未分类',
    type: item.type || '文件',
    ownerType: item.owner_type || 'personal',
    ownerName: item.owner_name || '',
    canManage: !!item.can_manage,
    assetCount: item.asset_count || 0,
    sort: item.sort || 0,
  }
}

function countByCategory(category) {
  const categoryAssets = isScopedLibrary.value ? scopedCategoryAssets.value : majorFilteredScopeAssets.value
  return categoryAssets.filter(asset => asset.category === category).length
}

function categoryRowsFor(scope, majorCategory) {
  const baseRows = assetCategories.value.filter(category => {
    const scopeMatched = scope === 'all' || category.ownerType === scope
    return scopeMatched && majorCategoryFromType(category.type) === majorCategory
  })
  const baseAssets = scope === 'all'
    ? majorFilteredScopeAssets.value
    : majorFilteredScopeAssets.value.filter(asset => asset.ownerType === scope)
  const assetRows = baseAssets
    .filter(asset => majorCategoryOf(asset) === majorCategory)
    .map(asset => ({
      id: `asset-derived:${asset.ownerType}:${asset.type}:${categoryGroupKey(asset)}`,
      name: categoryGroupKey(asset),
      type: asset.type,
      ownerType: asset.ownerType,
      ownerName: asset.ownerName,
      canManage: asset.canManage,
      assetCount: 0,
    }))
  const seen = new Set()
  return [...baseRows, ...assetRows].filter(item => {
    const key = `${item.ownerType}:${item.type}:${item.name}`
    if (seen.has(key)) return false
    seen.add(key)
    return item.name
  })
}

function categoryAssetsFor(scope, majorCategory, category) {
  const baseAssets = scope === 'all'
    ? majorFilteredScopeAssets.value
    : majorFilteredScopeAssets.value.filter(asset => asset.ownerType === scope)
  return baseAssets.filter(asset => majorCategoryOf(asset) === majorCategory && categoryGroupKey(asset) === category)
}

function scopeMatchesAsset(asset) {
  return activeScope.value === 'all'
    || asset.ownerType === activeScope.value
    || (activeScope.value === 'favorite' && asset.favorite)
}

function assetMatchesKeyword(asset, keyword) {
  const searchText = [asset.name, asset.type, asset.category, asset.referenceSceneName, asset.referenceTagName, asset.ownerName, ...asset.tags].join(' ').toLowerCase()
  return searchText.includes(keyword)
}

function majorCategoryMatchesAsset(asset) {
  return !activeMajorCategory.value || majorCategoryOf(asset) === activeMajorCategory.value
}

function majorCategoryOf(asset) {
  const matched = majorCategoryDefs.find(def => def.types.includes(asset.type))
  return matched?.value || 'other'
}

function createMajorCategoryCard({ label, value, shortLabel, assets: groupAssets }) {
  const previews = groupAssets.slice(0, 4).map(asset => assetPreviewItem(asset)).filter(Boolean)
  return {
    label,
    value,
    shortLabel,
    count: groupAssets.length,
    subCategoryCount: new Set(groupAssets.map(asset => value === 'style_reference' ? categoryGroupKey(asset) : asset.category).filter(Boolean)).size,
    previews,
  }
}

function createCategoryCard(category, groupAssets, level = 'category', explicitCategory = null) {
  const previews = groupAssets.slice(0, 4).map(asset => assetPreviewItem(asset)).filter(Boolean)
  const styleCount = new Set(groupAssets.map(styleGroupKey).filter(Boolean)).size
  const typeCount = new Set(groupAssets.map(asset => asset.type).filter(Boolean)).size
  const count = groupAssets.length || explicitCategory?.assetCount || 0
  const styleMeta = styleCategoryMeta(level, groupAssets)
  return {
    id: styleMeta.id || explicitCategory?.id || '',
    kind: styleMeta.kind || 'asset',
    label: category,
    name: category,
    value: category,
    type: styleMeta.type || explicitCategory?.type || groupAssets[0]?.type || defaultTypeForMajorCategory(activeMajorCategory.value),
    ownerType: styleMeta.ownerType || explicitCategory?.ownerType || groupAssets[0]?.ownerType || activeScope.value,
    ownerName: styleMeta.ownerName || explicitCategory?.ownerName || groupAssets[0]?.ownerName || ownerMeta(activeScope.value).label,
    canManage: styleMeta.canManage || !!explicitCategory?.canManage,
    sceneId: styleMeta.sceneId || 0,
    sort: explicitCategory?.sort || styleMeta.sort || 0,
    shortLabel: `${count}`,
    count,
    typeCount,
    description: categoryCardDescription(level, count, styleCount, typeCount),
    previews,
  }
}

function canManageCategoryCard(category) {
  if (category?.kind === 'style-space' || category?.kind === 'style') {
    return isAdmin.value && Number(category?.id) > 0
  }
  return ['system', 'department', 'personal'].includes(activeScope.value)
    && Number.isFinite(Number(category?.id))
    && Number(category.id) > 0
    && category?.canManage
}

function styleCategoryMeta(level, groupAssets) {
  if (!isStyleReferenceMajor.value || !groupAssets.length) return {}
  if (level === 'space') {
    const sceneID = groupAssets.find(asset => asset.referenceSceneId)?.referenceSceneId || 0
    const scene = sceneStyleCategories.value.find(item => Number(item.id) === Number(sceneID))
    return {
      id: sceneID,
      kind: 'style-space',
      type: 'scene_style_space',
      ownerType: 'system',
      ownerName: '风格库',
      canManage: isAdmin.value && sceneID > 0,
      sceneId: sceneID,
      sort: scene?.sort || 0,
    }
  }
  if (level === 'style') {
    const tagID = groupAssets.find(asset => asset.referenceTagId)?.referenceTagId || 0
    const tag = sceneStyleTags.value.find(item => Number(item.id) === Number(tagID))
    return {
      id: tagID,
      kind: 'style',
      type: 'scene_style_tag',
      ownerType: 'system',
      ownerName: '风格库',
      canManage: isAdmin.value && tagID > 0,
      sceneId: groupAssets.find(asset => asset.referenceSceneId)?.referenceSceneId || 0,
      sort: tag?.sort || 0,
    }
  }
  return {}
}

function categoryCardDescription(level, count, styleCount, typeCount) {
  if (level === 'space') return `${count} 张参考图 · ${styleCount} 个风格`
  if (level === 'style') return `${count} 张参考图`
  return `${count} 个资产 · ${typeCount} 个类型`
}

function categoryGroupKey(asset) {
  if (majorCategoryOf(asset) === 'style_reference') {
    return asset.referenceSceneName || (asset.referenceSceneId ? `场景 ${asset.referenceSceneId}` : '') || '未分组空间'
  }
  return asset.category || '未分类'
}

function styleGroupKey(asset) {
  return asset.referenceTagName || asset.category || '未分组风格'
}

function defaultTypeForMajorCategory(majorCategory) {
  return majorCategoryDefs.find(item => item.value === majorCategory)?.types?.[0] || '文件'
}

function majorCategoryFromType(type) {
  return majorCategoryDefs.find(def => def.types.includes(type))?.value || 'other'
}

function majorCategoryLabelFromType(type) {
  const major = majorCategoryDefs.find(def => def.types.includes(type))
  return major?.label || type || '资产'
}

function assetPreviewItem(asset) {
  if (!asset?.cover) return null
  return {
    src: asset.cover,
    alt: asset.name || asset.type || '资产预览',
    generated: !!asset.coverGenerated,
  }
}

function fallbackPreview(type, name = '') {
  return {
    src: fallbackCover(type, name),
    alt: name || type || '资产预览',
    generated: true,
  }
}

function categoryOptionsForOwner(ownerType) {
  if (!ownerType) return []
  return [...new Set([
    ...assetCategories.value
      .filter(category => category.ownerType === ownerType)
      .map(category => category.name),
    ...assets.value
    .filter(asset => asset.ownerType === ownerType)
      .map(asset => asset.category),
  ].filter(Boolean))]
}

function ownerMeta(ownerType) {
  const map = {
    system: {
      label: '系统',
      icon: ShieldCheckIcon,
      badgeClass: 'bg-blue-50 text-blue-700 ring-1 ring-blue-100',
    },
    department: {
      label: '部门',
      icon: BuildingOffice2Icon,
      badgeClass: 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-100',
    },
    personal: {
      label: '我的',
      icon: UserCircleIcon,
      badgeClass: 'bg-amber-50 text-amber-700 ring-1 ring-amber-100',
    },
    shared: {
      label: '分享',
      icon: FolderIcon,
      badgeClass: 'bg-violet-50 text-violet-700 ring-1 ring-violet-100',
    },
  }
  return map[ownerType] || map.personal
}

function resetFilters() {
  clearAssetSelection()
  search.value = ''
  activeScope.value = 'all'
  activeMajorCategory.value = ''
  activeCategory.value = ''
  activeSubCategory.value = ''
  typeFilter.value = 'all'
}

function setScope(scope) {
  clearAssetSelection()
  activeScope.value = scope
  activeMajorCategory.value = ''
  activeCategory.value = ''
  activeSubCategory.value = ''
}

function setMajorCategory(category) {
  clearAssetSelection()
  activeMajorCategory.value = category
  activeCategory.value = ''
  activeSubCategory.value = ''
  typeFilter.value = 'all'
}

function setCategory(category) {
  clearAssetSelection()
  activeCategory.value = category
  activeSubCategory.value = ''
}

function setSubCategory(category) {
  clearAssetSelection()
  activeSubCategory.value = category
}

function goBackFromAssetList() {
  clearAssetSelection()
  if (isStyleReferenceMajor.value) {
    activeSubCategory.value = ''
    return
  }
  activeCategory.value = ''
}

function goMajorHome() {
  clearAssetSelection()
  activeMajorCategory.value = ''
  activeCategory.value = ''
  activeSubCategory.value = ''
}

function openUpload() {
  resetUploadForm()
  const ownerType = uploadOwnerFromCurrentScope()
  uploadForm.ownerType = ownerType
  if (activeMajorCategory.value && activeCategory.value) {
    uploadForm.type = defaultTypeForMajorCategory(activeMajorCategory.value)
    uploadForm.category = activeCategory.value
  } else {
    const first = uploadCategoryOptions.value[0]
    uploadCategoryKey.value = first?.key || ''
  }
  uploadOpen.value = true
}

function closeUpload() {
  clearUploadPreviews()
  categoryDropdownOpen.value = false
  uploadOpen.value = false
}

function resetUploadForm() {
  Object.assign(uploadForm, {
    ownerType: 'personal',
    type: '',
    category: '',
    tagsText: '',
  })
  uploadFiles.value = []
  uploadingIndex.value = 0
  clearUploadPreviews()
}

function openCategoryCreate() {
  categoryDropdownOpen.value = false
  if (uploadOpen.value) {
    const uploadMajor = uploadForm.type ? majorCategoryFromType(uploadForm.type) : ''
    categoryForm.ownerType = uploadForm.ownerType || uploadOwnerFromCurrentScope()
    categoryForm.majorCategory = uploadMajor && uploadMajor !== 'other' ? uploadMajor : activeMajorCategory.value || 'product'
  } else {
    categoryForm.ownerType = uploadOwnerFromCurrentScope()
    categoryForm.majorCategory = activeMajorCategory.value || 'product'
  }
  if (!canCreateCategoryForOwner(categoryForm.ownerType)) {
    toast.warning('只有管理员可以在系统资产库中新建分类')
    return
  }
  categoryForm.name = ''
  categoryOpen.value = true
}

function closeCategoryCreate() {
  if (creatingCategory.value) return
  categoryOpen.value = false
}

function openCategoryManage(category) {
  if (!canManageCategoryCard(category)) return
  categoryManageTarget.value = { ...category, id: Number(category.id), name: category.name || category.label }
  categoryManageForm.name = categoryManageTarget.value.name
  categoryManageForm.targetCategoryId = ''
  categoryManageOpen.value = true
}

function closeCategoryManage() {
  if (categoryManageSaving.value) return
  categoryManageOpen.value = false
  categoryManageTarget.value = null
  categoryManageForm.name = ''
  categoryManageForm.targetCategoryId = ''
}

async function renameCategory() {
  const target = categoryManageTarget.value
  if (!target) return
  const name = categoryManageForm.name.trim()
  if (!name) {
    toast.warning('请输入分类名称')
    return
  }
  categoryManageSaving.value = true
  try {
    if (target.kind === 'style-space') {
      const scene = sceneStyleCategories.value.find(item => Number(item.id) === Number(target.id)) || {}
      await request.put(`/admin/scene-style-categories/${target.id}`, {
        name,
        icon: scene.icon || '',
        sort: scene.sort || 0,
      })
      await Promise.all([fetchAssets(), fetchSceneStyleMeta()])
      activeCategory.value = name
    } else if (target.kind === 'style') {
      const tag = sceneStyleTags.value.find(item => Number(item.id) === Number(target.id)) || {}
      await request.put(`/admin/scene-style-tags/${target.id}`, {
        name,
        description: tag.description || '',
        sort: tag.sort || 0,
      })
      await Promise.all([fetchAssets(), fetchSceneStyleMeta()])
      activeSubCategory.value = name
    } else {
      await request.put(`/asset-categories/${target.id}`, { name })
      await Promise.all([fetchAssets(), fetchAssetCategories()])
      activeCategory.value = name
    }
    categoryManageTarget.value = { ...target, name, label: name }
    categoryManageForm.name = name
    toast.success('分类名称已更新')
  } catch (error) {
    toast.error(error?.response?.data?.error || '分类更新失败')
  } finally {
    categoryManageSaving.value = false
  }
}

async function moveCategoryAssets() {
  const target = categoryManageTarget.value
  const toCategoryId = Number(categoryManageForm.targetCategoryId)
  if (!target || !toCategoryId) return
  const toCategory = categoryMoveTargets.value.find(category => Number(category.id) === toCategoryId)
  if (!toCategory) {
    toast.warning('请选择目标分类')
    return
  }
  categoryManageSaving.value = true
  try {
    let res
    if (target.kind === 'style') {
      res = await request.post('/assets/move-style-references', {
        from_scene_id: target.sceneId || 0,
        from_tag_id: target.id,
        to_tag_id: toCategoryId,
      })
      await Promise.all([fetchAssets(), fetchSceneStyleMeta()])
      activeSubCategory.value = toCategory.name
    } else {
      res = await request.post('/assets/move-category', {
        from_category_id: target.id,
        to_category_id: toCategoryId,
      })
      await Promise.all([fetchAssets(), fetchAssetCategories()])
      activeCategory.value = toCategory.name
    }
    categoryManageOpen.value = false
    categoryManageTarget.value = null
    toast.success(`已移动 ${res.data?.moved || 0} 个资产`)
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产移动失败')
  } finally {
    categoryManageSaving.value = false
  }
}

async function deleteCategory() {
  const target = categoryManageTarget.value
  if (!target) return
  categoryManageSaving.value = true
  try {
    await request.delete(`/asset-categories/${target.id}`)
    await fetchAssetCategories()
    if (activeCategory.value === target.name) {
      activeCategory.value = ''
      activeSubCategory.value = ''
    }
    categoryManageOpen.value = false
    categoryManageTarget.value = null
    toast.success('分类已删除')
  } catch (error) {
    toast.error(error?.response?.data?.error || '分类删除失败')
  } finally {
    categoryManageSaving.value = false
  }
}

async function moveCategorySort(category, direction) {
  const cards = category.kind === 'style' ? styleCategoryCards.value : categoryCards.value
  const currentIndex = cards.findIndex(item => item.value === category.value)
  const nextIndex = currentIndex + direction
  if (currentIndex < 0 || nextIndex < 0 || nextIndex >= cards.length) return
  const nextCards = [...cards]
  const temp = nextCards[currentIndex]
  nextCards[currentIndex] = nextCards[nextIndex]
  nextCards[nextIndex] = temp

  try {
    if (category.kind === 'style-space') {
      await saveSceneCategorySort(nextCards)
      await Promise.all([fetchSceneStyleMeta(), fetchAssets()])
    } else if (category.kind === 'style') {
      await saveSceneTagSort(nextCards)
      await Promise.all([fetchSceneStyleMeta(), fetchAssets()])
    } else {
      await request.post('/asset-categories/reorder', {
        category_ids: nextCards.map(item => Number(item.id)).filter(Boolean),
      })
      await fetchAssetCategories()
    }
    toast.success('分类排序已更新')
  } catch (error) {
    toast.error(error?.response?.data?.error || '分类排序保存失败')
  }
}

async function saveSceneCategorySort(cards) {
  await Promise.all(cards.map((card, index) => {
    const scene = sceneStyleCategories.value.find(item => Number(item.id) === Number(card.id)) || {}
    return request.put(`/admin/scene-style-categories/${card.id}`, {
      name: scene.name || card.name,
      icon: scene.icon || '',
      sort: (index + 1) * 10,
    })
  }))
}

async function saveSceneTagSort(cards) {
  await Promise.all(cards.map((card, index) => {
    const tag = sceneStyleTags.value.find(item => Number(item.id) === Number(card.id)) || {}
    return request.put(`/admin/scene-style-tags/${card.id}`, {
      name: tag.name || card.name,
      description: tag.description || '',
      sort: (index + 1) * 10,
    })
  }))
}

async function repairCurrentCategories() {
  if (!canRepairCurrentCategories.value) return
  repairingCategories.value = true
  try {
    const major = majorCategoryDefs.find(item => item.value === activeMajorCategory.value)
    const types = major?.types?.length ? major.types : [defaultTypeForMajorCategory(activeMajorCategory.value)]
    const results = await Promise.all(types.map(type => request.post('/asset-categories/repair', {
      owner_type: activeScope.value,
      type,
    })))
    const created = results.reduce((sum, res) => sum + Number(res.data?.created || 0), 0)
    await fetchAssetCategories()
    toast.success(`已补建 ${created} 个历史分类`)
  } catch (error) {
    toast.error(error?.response?.data?.error || '历史分类修复失败')
  } finally {
    repairingCategories.value = false
  }
}

function canMoveAsset(asset) {
  if (isStyleReferenceMajor.value) return isAdmin.value && asset.sourceType === 'scene_style_reference'
  return asset.sourceType === 'asset' && asset.canManage
}

function toggleAssetSelection(asset) {
  if (!canMoveAsset(asset)) return
  const id = asset.id
  if (selectedAssetIds.value.includes(id)) {
    selectedAssetIds.value = selectedAssetIds.value.filter(item => item !== id)
  } else {
    selectedAssetIds.value = [...selectedAssetIds.value, id]
  }
  batchMoveTargetId.value = ''
}

function clearAssetSelection() {
  selectedAssetIds.value = []
  batchMoveTargetId.value = ''
}

async function moveSelectedAssets() {
  const targetID = Number(batchMoveTargetId.value)
  if (!selectedAssets.value.length || !targetID) return
  const target = batchMoveTargets.value.find(item => Number(item.id) === targetID)
  try {
    let res
    if (isStyleReferenceMajor.value) {
      const fromTagID = Number(styleCategoryCards.value.find(item => item.value === activeSubCategory.value)?.id || 0)
      const fromSceneID = Number(selectedAssets.value[0]?.referenceSceneId || 0)
      res = await request.post('/assets/move-style-references', {
        from_scene_id: fromSceneID,
        from_tag_id: fromTagID,
        to_tag_id: targetID,
        reference_ids: selectedAssets.value.map(asset => Number(asset.sourceId)).filter(Boolean),
      })
      await Promise.all([fetchAssets(), fetchSceneStyleMeta()])
      if (target) activeSubCategory.value = target.name
    } else {
      const fromCategoryID = Number(categoryCards.value.find(item => item.value === activeCategory.value)?.id || 0)
      res = await request.post('/assets/move-category', {
        from_category_id: fromCategoryID,
        to_category_id: targetID,
        asset_ids: selectedAssets.value.map(asset => Number(asset.sourceId)).filter(Boolean),
      })
      await Promise.all([fetchAssets(), fetchAssetCategories()])
      if (target) activeCategory.value = target.name
    }
    clearAssetSelection()
    toast.success(`已移动 ${res.data?.moved || 0} 个资产`)
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产移动失败')
  }
}

function selectUploadCategory(category) {
  uploadCategoryKey.value = category?.key || ''
  categoryDropdownOpen.value = false
}

function categoryOptionMeta(category) {
  const count = Number(category?.assetCount || 0)
  const owner = category?.ownerName || selectedUploadTarget.value.label
  return `${majorCategoryLabelFromType(category?.type)} · ${owner} · ${count} 个资产`
}

async function createAssetCategory() {
  const name = categoryForm.name.trim()
  if (!name) {
    toast.warning('请输入分类名称')
    return
  }
  creatingCategory.value = true
  try {
    const res = await request.post('/asset-categories', {
      owner_type: categoryForm.ownerType,
      type: defaultTypeForMajorCategory(categoryForm.majorCategory),
      name,
    })
    const category = mapAssetCategoryFromApi(res.data)
    assetCategories.value = [
      category,
      ...assetCategories.value.filter(item => !(item.ownerType === category.ownerType && item.type === category.type && item.name === category.name)),
    ]
    categoryOpen.value = false
    activeScope.value = category.ownerType
    activeMajorCategory.value = majorCategoryFromType(category.type)
    activeCategory.value = category.name
    activeSubCategory.value = ''
    if (uploadOpen.value) {
      uploadForm.ownerType = category.ownerType
      uploadForm.type = category.type
      uploadForm.category = category.name
    }
    toast.success('分类已创建')
  } catch (error) {
    toast.error(error?.response?.data?.error || '分类创建失败')
  } finally {
    creatingCategory.value = false
  }
}

function uploadOwnerFromCurrentScope() {
  if (activeScope.value === 'system' && isAdmin.value) return 'system'
  if (activeScope.value === 'department') return 'department'
  return 'personal'
}

function canCreateCategoryForOwner(ownerType) {
  return ownerType !== 'system' || isAdmin.value
}

function handleFileChange(event) {
  const files = Array.from(event.target.files || [])
  appendUploadFiles(files)
  event.target.value = ''
}

function appendUploadFiles(files) {
  files.forEach((file) => {
    const duplicate = uploadFiles.value.some(item => item.name === file.name && item.size === file.size && item.lastModified === file.lastModified)
    if (duplicate) return
    uploadFiles.value.push(file)
    uploadPreviewItems.value.push({
      id: `${file.name}-${file.size}-${file.lastModified}`,
      name: file.name,
      preview: file.type?.startsWith('image/') ? URL.createObjectURL(file) : '',
    })
  })
}

function removeUploadFile(index) {
  const preview = uploadPreviewItems.value[index]
  if (preview?.preview) URL.revokeObjectURL(preview.preview)
  uploadFiles.value.splice(index, 1)
  uploadPreviewItems.value.splice(index, 1)
}

function clearUploadFiles() {
  uploadFiles.value = []
  clearUploadPreviews()
}

async function createAsset() {
  const files = [...uploadFiles.value]
  if (!files.length) {
    toast.warning('请选择要上传的文件')
    return
  }
  if (!uploadForm.type || !uploadForm.category.trim()) {
    toast.warning('请选择分类')
    return
  }

  uploading.value = true
  uploadingIndex.value = 0
  try {
    const created = []
    for (const [index, file] of files.entries()) {
      uploadingIndex.value = index + 1
      const form = new FormData()
      form.append('owner_type', uploadForm.ownerType)
      form.append('name', assetNameFromFile(file))
      form.append('type', uploadForm.type)
      form.append('category', uploadForm.category.trim())
      form.append('tags', uploadForm.tagsText)
      form.append('file', file)
      const res = await request.post('/assets', form)
      created.push(mapAssetFromApi(res.data))
    }
    assets.value.unshift(...created)
    const firstAsset = created[0]
    activeScope.value = firstAsset.ownerType
    activeMajorCategory.value = majorCategoryOf(firstAsset)
    activeCategory.value = categoryGroupKey(firstAsset)
    activeSubCategory.value = majorCategoryOf(firstAsset) === 'style_reference' ? styleGroupKey(firstAsset) : ''
    typeFilter.value = 'all'
    uploadOpen.value = false
    selectedAsset.value = firstAsset
    clearUploadPreviews()
    toast.success(created.length === 1 ? '资产已上传' : `已上传 ${created.length} 个资产`)
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产上传失败')
  } finally {
    uploading.value = false
    uploadingIndex.value = 0
  }
}

function clearUploadPreviews() {
  uploadPreviewItems.value.forEach(item => {
    if (item.preview) URL.revokeObjectURL(item.preview)
  })
  uploadPreviewItems.value = []
}

function assetNameFromFile(file) {
  return String(file?.name || '').replace(/\.[^.]+$/, '').trim() || '未命名资产'
}

function normalizeTags(text) {
  return text
    .split(/[,，]/)
    .map(tag => tag.trim())
    .filter(Boolean)
    .slice(0, 6)
}

function assetPreviewFromApi(item) {
  const fileType = item.file_type || ''
  const coverUrl = displayableImageUrl(item.cover || item.cover_url || '', fileType)
  const fileImageUrl = displayableImageUrl(item.file_url || '', fileType)
  const imageUrl = coverUrl || fileImageUrl
  if (imageUrl) {
    return {
      cover: imageUrl,
      coverGenerated: false,
    }
  }
  return {
    cover: fallbackCover(item.type || fileType || '文件', item.name || item.file_name || ''),
    coverGenerated: true,
  }
}

function displayableImageUrl(url, fileType = '') {
  if (!url) return ''
  if (fileType === 'image' || isImageUrl(url)) return getDisplayThumbnailUrl(url)
  return ''
}

function isImageUrl(url) {
  if (!url) return false
  if (url.startsWith('data:image/')) return true
  const path = url.split('?')[0].split('#')[0].toLowerCase()
  return /\.(png|jpe?g|webp|gif|bmp|avif|svg)$/.test(path)
}

function fallbackCover(type, name = '') {
  const meta = previewMeta(type)
  const title = meta.title
  const subtitle = name || type || '资产'
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="480" height="360" viewBox="0 0 480 360">
      <defs>
        <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stop-color="${meta.bg1}"/>
          <stop offset="1" stop-color="${meta.bg2}"/>
        </linearGradient>
      </defs>
      <rect width="480" height="360" rx="28" fill="url(#bg)"/>
      <path d="M64 266h352M92 240h296M120 214h240" stroke="${meta.line}" stroke-width="2" opacity=".46"/>
      <g transform="translate(150 70)">
        <rect x="0" y="0" width="180" height="132" rx="18" fill="${meta.card}" opacity=".95"/>
        <path d="${meta.path}" fill="none" stroke="${meta.accent}" stroke-width="10" stroke-linecap="round" stroke-linejoin="round"/>
      </g>
      <text x="240" y="254" text-anchor="middle" font-family="Arial, 'Microsoft YaHei', sans-serif" font-size="31" font-weight="800" fill="${meta.text}">${escapeSvgText(title)}</text>
      <text x="240" y="286" text-anchor="middle" font-family="Arial, 'Microsoft YaHei', sans-serif" font-size="18" font-weight="600" fill="${meta.muted}">${escapeSvgText(truncatePreviewText(subtitle))}</text>
    </svg>
  `
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}

function previewMeta(type = '') {
  if (['模型文件', 'STP模型'].includes(type) || String(type).toLowerCase() === 'model') {
    return {
      title: '模型文件',
      bg1: '#e8f1ff',
      bg2: '#f6f7fb',
      card: '#ffffff',
      accent: '#2563eb',
      line: '#94a3b8',
      text: '#172033',
      muted: '#64748b',
      path: 'M48 92 90 68l42 24-42 24-42-24Zm42-24V34l42 24M90 68 48 44m84 48V58m-84 34v-48',
    }
  }
  if (type === '色块') {
    return {
      title: '色块',
      bg1: '#fff7ed',
      bg2: '#fef2f2',
      card: '#ffffff',
      accent: '#ea580c',
      line: '#fdba74',
      text: '#1f2937',
      muted: '#9a3412',
      path: 'M48 38h40v40H48V38Zm58 0h40v40h-40V38ZM48 96h40v40H48V96Zm58 0h40v40h-40V96Z',
    }
  }
  if (['文档', '规范', '视频'].includes(type)) {
    return {
      title: type || '文档',
      bg1: '#ecfeff',
      bg2: '#f8fafc',
      card: '#ffffff',
      accent: '#0891b2',
      line: '#67e8f9',
      text: '#164e63',
      muted: '#64748b',
      path: 'M60 34h54l30 30v72H60V34Zm54 0v30h30M78 84h48M78 108h48',
    }
  }
  return {
    title: type || '资产',
    bg1: '#f5f5f4',
    bg2: '#eef2f7',
    card: '#ffffff',
    accent: '#111827',
    line: '#cbd5e1',
    text: '#111827',
    muted: '#64748b',
    path: 'M46 102 78 70l26 24 20-18 32 34M50 38h108v92H50V38Zm30 28a12 12 0 1 0 0-24 12 12 0 0 0 0 24Z',
  }
}

function escapeSvgText(value) {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function truncatePreviewText(value) {
  const text = String(value || '')
  return text.length > 14 ? `${text.slice(0, 14)}...` : text
}

function openDetail(asset, editing = false) {
  selectedAsset.value = asset
  detailEditing.value = false
  if (editing && asset.canManage) startEdit(asset)
}

function replaceAsset(nextAsset) {
  const index = assets.value.findIndex(item => item.id === nextAsset.id)
  if (index !== -1) {
    assets.value.splice(index, 1, nextAsset)
  }
}

function closeDetail() {
  selectedAsset.value = null
  detailEditing.value = false
}

function startEdit(asset) {
  Object.assign(editForm, {
    id: asset.id,
    name: asset.name,
    type: asset.type,
    category: asset.category,
    tagsText: asset.tags.join('，'),
  })
  detailEditing.value = true
}

async function saveAssetEdit() {
  const asset = assets.value.find(item => item.id === editForm.id)
  if (!asset) return
  const name = editForm.name.trim()
  if (!name) {
    toast.warning('请填写资产名称')
    return
  }

  if (asset.sourceType !== 'asset') {
    toast.warning('产品库资产只能在产品库中维护')
    return
  }

  try {
    const res = await request.put(`/assets/${asset.sourceId}`, {
      name,
      type: editForm.type,
      category: editForm.category.trim() || editForm.type,
      tags: normalizeTags(editForm.tagsText),
    })
    const updated = mapAssetFromApi(res.data)
    replaceAsset(updated)
    selectedAsset.value = updated
    detailEditing.value = false
    toast.success('资产信息已保存')
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产保存失败')
  }
}

async function toggleFavorite(asset) {
  const nextFavorite = !asset.favorite
  asset.favorite = nextFavorite
  try {
    await request.post('/assets/favorite', {
      source_type: asset.sourceType,
      source_id: asset.sourceId,
      favorite: nextFavorite,
    })
    toast.success(nextFavorite ? '已收藏' : '已取消收藏')
  } catch (error) {
    asset.favorite = !nextFavorite
    toast.error(error?.response?.data?.error || '收藏操作失败')
  }
}

function downloadAsset(asset) {
  if (!asset.fileUrl) {
    toast.warning('该资产没有可下载文件')
    return
  }
  window.open(asset.fileUrl, '_blank')
  toast.info(`开始下载「${asset.name}」`)
}

async function deleteAsset() {
  if (!assetPendingDelete.value) return
  const pending = assetPendingDelete.value
  if (pending.sourceType !== 'asset') {
    toast.warning('产品库资产只能在产品库中维护')
    assetPendingDelete.value = null
    return
  }
  try {
    await request.delete(`/assets/${pending.sourceId}`)
    assets.value = assets.value.filter(asset => asset.id !== pending.id)
    if (selectedAsset.value?.id === pending.id) {
      closeDetail()
    }
    assetPendingDelete.value = null
    toast.success('资产已删除')
  } catch (error) {
    toast.error(error?.response?.data?.error || '资产删除失败')
  }
}

</script>
