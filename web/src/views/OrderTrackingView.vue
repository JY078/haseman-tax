<template>
  <MainLayout>
    <div class="mx-auto w-full max-w-[1920px] px-3 py-6 sm:px-5 lg:px-6 2xl:px-8">
      <header class="flex flex-col gap-4 border-b border-neutral-200 pb-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-neutral-900">订单汇总</h1>
          <p class="mt-1 text-sm text-neutral-500">平台订单与物流追踪</p>
        </div>
        <div class="flex gap-2">
          <Button variant="outline" :disabled="loadingExportable" @click="openExportDialog">
            <ArrowDownTrayIcon class="h-4 w-4" />
            导出表格
          </Button>
          <router-link to="/order-to-mcf" class="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-neutral-300 bg-white px-4 text-sm font-medium text-neutral-700 hover:bg-neutral-50">
            <ArrowsRightLeftIcon class="h-4 w-4" />
            订单转 MCF
          </router-link>
        </div>
      </header>

      <div v-if="platformOptions.length > 1" class="flex gap-1 border-b border-neutral-200 py-4" aria-label="平台筛选">
        <button
          v-for="option in platformOptions"
          :key="option.value"
          type="button"
          class="h-9 rounded px-4 text-sm font-medium transition-colors"
          :class="selectedPlatform === option.value ? 'bg-neutral-900 text-white' : 'text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900'"
          @click="setPlatform(option.value)"
        >
          {{ option.label }}
        </button>
      </div>

      <section class="grid border-b border-neutral-200 bg-white sm:grid-cols-3" aria-label="订单统计">
        <div v-for="metric in metrics" :key="metric.label" class="border-b border-neutral-100 px-1 py-5 last:border-b-0 sm:border-b-0 sm:border-r sm:px-5 sm:first:pl-1 sm:last:border-r-0">
          <div class="text-xs font-medium text-neutral-500">{{ metric.label }}</div>
          <div class="mt-1 text-2xl font-semibold text-neutral-900">{{ metric.value }}</div>
        </div>
      </section>

      <section class="pt-5">
        <div class="flex flex-col gap-3 border-y border-neutral-200 bg-neutral-50/70 px-3 py-3 lg:flex-row lg:items-end lg:justify-between">
          <div class="flex flex-wrap items-end gap-3">
            <label class="block min-w-40">
              <span class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-neutral-500">
                <FunnelIcon class="h-3.5 w-3.5" />
                订单状态
              </span>
              <select v-model="orderStatus" class="h-10 w-full rounded-md border border-neutral-300 bg-white px-3 text-sm font-medium text-neutral-700 outline-none transition-colors hover:border-neutral-400 focus:border-neutral-700 focus:ring-2 focus:ring-neutral-200" @change="applyFilters">
                <option v-for="option in orderStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
            <div class="block min-w-56">
              <span class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-neutral-500">
                <CalendarDaysIcon class="h-3.5 w-3.5" />
                添加日期
              </span>
              <DatePicker v-model="createdDate" v-model:mode="dateFilterType" placeholder="选择日期或月份" aria-label="选择添加日期" @change="applyFilters" />
            </div>
            <button v-if="hasActiveFilters" type="button" class="mb-0.5 inline-flex h-9 w-9 items-center justify-center rounded-md text-neutral-500 transition-colors hover:bg-neutral-200 hover:text-neutral-900" title="清空筛选" aria-label="清空筛选" @click="clearFilters">
              <XMarkIcon class="h-4 w-4" />
            </button>
          </div>
          <div class="flex w-full items-end gap-2 lg:w-auto">
            <label class="min-w-0 flex-1 lg:w-80">
              <span class="mb-1.5 block text-xs font-medium text-neutral-500">搜索订单</span>
              <span class="relative block">
                <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-3 h-4 w-4 text-neutral-400" />
              <input
                v-model="query"
                type="search"
                class="h-10 w-full rounded-md border border-neutral-300 bg-white pl-9 pr-3 text-sm outline-none transition-colors hover:border-neutral-400 focus:border-neutral-700 focus:ring-2 focus:ring-neutral-200"
                placeholder="搜索订单号、SKU、收件人、追踪号或备注"
                @keydown.enter="search"
              >
              </span>
            </label>
            <Button class="h-10" variant="outline" @click="search">查询</Button>
          </div>
        </div>

        <div v-if="loadError" class="border-b border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {{ loadError }}
        </div>

        <div class="h-[calc(100vh-430px)] min-h-80 max-h-[680px] overflow-x-hidden overflow-y-auto overscroll-contain border-b border-neutral-200 bg-white">
          <table class="w-full table-fixed text-sm">
            <thead class="sticky top-0 z-10 bg-neutral-50 text-left text-xs font-medium text-neutral-500 shadow-[0_1px_0_0_rgb(229_229_229)]">
              <tr>
                <th
                  v-for="column in tableColumns"
                  :key="column.key"
                  class="group relative px-2 py-3 xl:px-3 2xl:px-4"
                  :class="column.align === 'center' ? 'text-center' : ''"
                  :style="{ width: columnWidthPercent(column) }"
                >
                  {{ column.label }}
                  <span
                    role="separator"
                    tabindex="0"
                    aria-orientation="vertical"
                    :aria-label="`调整${column.label}列宽`"
                    :aria-valuemin="column.minWidth"
                    :aria-valuemax="column.maxWidth"
                    :aria-valuenow="column.width"
                    class="absolute inset-y-0 right-0 z-20 flex w-2 cursor-col-resize touch-none select-none items-center justify-center outline-none focus:bg-neutral-200/70"
                    title="拖动调整列宽，双击恢复默认宽度"
                    @pointerdown.prevent="startColumnResize($event, column)"
                    @dblclick.prevent="resetColumnWidth(column)"
                    @keydown.left.prevent="resizeColumnBy(column, -10)"
                    @keydown.right.prevent="resizeColumnBy(column, 10)"
                  >
                    <span class="h-5 w-px bg-neutral-300 transition-colors group-hover:bg-neutral-500" />
                  </span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-100">
              <template v-for="item in items" :key="item.id">
              <tr
                class="transition-colors hover:bg-neutral-50/60"
                :class="isExpandableOrder(item) ? 'cursor-pointer' : ''"
                :aria-expanded="isExpandableOrder(item) ? expandedMergedOrders.has(item.id) : undefined"
                @click="toggleMergedOrderFromRow($event, item)"
              >
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <button
                    type="button"
                    class="block max-w-full cursor-copy break-all text-left font-mono text-xs font-semibold text-neutral-900 transition-colors hover:text-blue-600 focus-visible:rounded focus-visible:text-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
                    title="点击复制订单号"
                    @click="copyOrderID(item.order_id)"
                  >
                    {{ item.order_id }}
                  </button>
                  <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
                    <span class="inline-flex rounded px-1.5 py-0.5 text-[11px] font-medium" :class="platformBadgeClass(item.platform)">{{ platformLabel(item.platform) }}</span>
                    <span v-if="item.is_merged_order" class="inline-flex items-center gap-1 rounded bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700 ring-1 ring-inset ring-amber-200">
                      合并单 · {{ (item.original_order_ids || []).length || 1 }} 个原单
                      <ChevronUpIcon v-if="expandedMergedOrders.has(item.id)" class="h-3 w-3" />
                      <ChevronDownIcon v-else class="h-3 w-3" />
                    </span>
                    <span v-else-if="isExpandableOrder(item)" class="inline-flex items-center gap-1 rounded bg-neutral-100 px-1.5 py-0.5 text-[11px] font-medium text-neutral-600 ring-1 ring-inset ring-neutral-200">
                      {{ item.details.length }} 项商品
                      <ChevronUpIcon v-if="expandedMergedOrders.has(item.id)" class="h-3 w-3" />
                      <ChevronDownIcon v-else class="h-3 w-3" />
                    </span>
                  </div>
                </td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <button v-if="item.amazon_order_id" type="button" class="block max-w-full break-all text-left font-mono text-xs font-semibold text-neutral-800 hover:text-blue-600" title="点击复制亚马逊订单编号" @click="copyOrderID(item.amazon_order_id)">{{ item.amazon_order_id }}</button>
                  <span v-else class="text-xs text-neutral-400">未填写</span>
                </td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <span class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium" :class="orderStatusClass(item.status)">
                    <span class="h-1.5 w-1.5 rounded-full bg-current opacity-70" />
                    {{ orderStatusLabel(item.status) }}
                  </span>
                  <button
                    v-if="item.can_request_cancel"
                    type="button"
                    class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-neutral-500 transition-colors hover:text-red-700"
                    @click="openStatusConfirm(item, 'request_cancel')"
                  >
                    <NoSymbolIcon class="h-3.5 w-3.5" />
                    申请取消
                  </button>
                  <button
                    v-else-if="item.can_cancel"
                    type="button"
                    class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-red-700 transition-colors hover:text-red-900"
                    @click="openStatusConfirm(item, 'cancel')"
                  >
                    <XCircleIcon class="h-3.5 w-3.5" />
                    确认取消
                  </button>
                  <button
                    v-if="item.can_revoke_cancel && item.platform !== 'LOWES'"
                    type="button"
                    class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-neutral-500 transition-colors hover:text-neutral-900"
                    @click="openStatusConfirm(item, 'revoke_cancel')"
                  >
                    <ArrowUturnLeftIcon class="h-3.5 w-3.5" />
                    撤销申请
                  </button>
                </td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <div v-if="item.details?.length" class="min-w-0">
                    <div class="flex min-w-0 items-center gap-2 overflow-hidden">
                      <span class="min-w-0 truncate font-mono text-xs font-semibold" :class="item.details[0].amazon_sku ? 'text-neutral-800' : 'text-amber-700'" :title="item.details[0].amazon_sku || '未映射'">{{ item.details[0].amazon_sku || '未映射' }}</span>
                      <span class="shrink-0 text-xs font-medium text-neutral-500">× {{ detailStats(item).quantity }}</span>
                    </div>
                    <div v-if="item.details.length > 1" class="mt-1 text-[11px] text-neutral-400">{{ item.details.length }} 项商品明细</div>
                  </div>
                  <span v-else class="text-xs text-neutral-400">-</span>
                </td>
                <td class="break-words px-2 py-4 align-top text-neutral-700 xl:px-3 2xl:px-4">{{ item.recipient || '-' }}</td>
                <td class="break-all px-2 py-4 align-top text-neutral-700 xl:px-3 2xl:px-4">{{ item.phone || '-' }}</td>
                <td class="break-words px-2 py-4 align-top text-xs leading-5 text-neutral-600 xl:px-3 2xl:px-4">{{ item.address || '-' }}</td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <span v-if="item.carrier" class="inline-flex max-w-full whitespace-normal break-words rounded bg-neutral-100 px-2 py-1 text-xs font-medium text-neutral-700">{{ item.carrier }}</span>
                  <span v-else class="text-xs text-neutral-400">{{ item.tracking_number ? '未识别' : '-' }}</span>
                </td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <div class="flex items-start gap-1.5">
                    <span v-if="item.status === 'cancelled'" class="text-xs text-neutral-400">-</span>
                    <span v-else-if="trackingNumbers(item).length" class="min-w-0 break-all font-mono text-xs font-semibold text-neutral-800" :title="trackingNumbers(item).join('、')">{{ trackingNumbers(item).join('、') }}</span>
                    <span v-else class="text-xs text-amber-700">待回填</span>
                    <button
                      v-if="item.can_edit_tracking && (item.packages?.length || 0) <= 1"
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800"
                      title="编辑追踪号和备注"
                      @click="openTrackingEditor(item)"
                    >
                      <PencilSquareIcon class="h-4 w-4" />
                    </button>
                  </div>
                  <div v-if="item.packages?.length > 1" class="mt-1 text-[11px] text-neutral-400">{{ item.packages.length }} 个包裹</div>
                  <div v-if="item.status !== 'cancelled' && item.tracking_number && (item.tracking_updated_by || item.tracking_updated_at)" class="mt-1.5 text-[11px] leading-4 text-neutral-400">
                    填写：{{ trackingUpdaterLabel(item) }} · {{ formatTimestamp(item.tracking_updated_at) }}
                  </div>
                </td>
                <td class="px-2 py-4 align-top text-xs leading-5 text-neutral-600 xl:px-3 2xl:px-4">
                  <div class="whitespace-pre-wrap break-words">{{ item.status === 'cancelled' ? '-' : (item.remark || '-') }}</div>
                </td>
                <td class="px-2 py-4 align-top xl:px-3 2xl:px-4">
                  <div class="break-words text-xs font-medium text-neutral-700" :title="importerLabel(item)">{{ importerLabel(item) }}</div>
                  <div class="mt-0.5 break-words text-[11px] text-neutral-400">{{ formatTimestamp(item.created_at) }}</div>
                </td>
              </tr>
              <tr v-if="isExpandableOrder(item) && expandedMergedOrders.has(item.id)" class="bg-neutral-50/80">
                <td :colspan="tableColumns.length" class="px-4 py-0 xl:px-6">
                  <div class="border-l-2 border-amber-300 py-4 pl-4 pr-2">
                    <div class="mb-3 flex items-center justify-between gap-4">
                      <div>
                        <div class="text-xs font-semibold text-neutral-800">{{ item.is_merged_order ? '合并单商品明细' : '商品明细' }}</div>
                        <div class="mt-0.5 text-[11px] text-neutral-400">{{ item.is_merged_order ? `${detailStats(item).orderCount} 个原单 · ` : '' }}{{ detailStats(item).itemCount }} 项商品 · 共 {{ detailStats(item).quantity }} 件</div>
                      </div>
                      <button type="button" class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded text-neutral-400 transition-colors hover:bg-neutral-200 hover:text-neutral-700" title="收起明细" @click="toggleMergedOrder(item.id)">
                        <ChevronUpIcon class="h-4 w-4" />
                      </button>
                    </div>
                    <div v-if="item.packages?.length > 1" class="mb-3 overflow-hidden rounded border border-neutral-200 bg-white">
                      <div class="grid grid-cols-[80px_minmax(0,1fr)_minmax(0,1fr)_64px_minmax(0,1.2fr)_110px_54px] gap-3 border-b border-neutral-200 bg-neutral-50 px-4 py-2 text-[11px] font-medium text-neutral-500">
                        <span>包裹</span>
                        <span>原订单号</span>
                        <span>Amazon SKU</span>
                        <span>数量</span>
                        <span>追踪号</span>
                        <span>状态</span>
                        <span />
                      </div>
                      <div v-for="(packageItem, packageIndex) in item.packages" :key="packageItem.id || packageItem.package_key" class="border-b border-neutral-100 px-4 py-2.5 text-xs last:border-b-0">
                        <div v-for="(detail, detailIndex) in (packageItem.details?.length ? packageItem.details : [{ original_order_id: item.order_id, amazon_sku: '', quantity: packageItem.quantity }])" :key="`${packageItem.package_key}-${detail.original_order_id}-${detail.amazon_sku}-${detailIndex}`" class="grid grid-cols-[80px_minmax(0,1fr)_minmax(0,1fr)_64px_minmax(0,1.2fr)_110px_54px] items-center gap-3 not-first:mt-2">
                        <span class="font-medium text-neutral-700">包裹 {{ packageIndex + 1 }}</span>
                        <button type="button" class="min-w-0 truncate text-left font-mono font-medium text-neutral-700 hover:text-blue-600" title="点击复制原订单号" @click="copyOrderID(detail.original_order_id || item.order_id)">{{ detail.original_order_id || item.order_id }}</button>
                        <span class="min-w-0 break-all font-mono" :class="detail.amazon_sku ? 'text-neutral-700' : 'text-amber-700'">{{ detail.amazon_sku || '未映射' }}</span>
                        <span class="text-neutral-600">× {{ detail.quantity || 0 }}</span>
                        <span v-if="packageItem.tracking_number" class="break-all font-mono text-neutral-800">{{ packageItem.tracking_number }}</span>
                        <span v-else class="text-amber-700">待回填</span>
                        <span class="text-neutral-500">{{ orderStatusLabel(packageItem.status) }}</span>
                        <button v-if="detailIndex === 0 && packageItem.can_edit_tracking" type="button" class="inline-flex h-7 w-7 items-center justify-center rounded text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800" title="编辑此包裹追踪号" @click="openPackageTrackingEditor(item, packageItem)">
                          <PencilSquareIcon class="h-3.5 w-3.5" />
                        </button>
                        <span v-else />
                        </div>
                      </div>
                    </div>
                    <div v-if="item.packages?.length <= 1" class="max-h-72 overflow-auto rounded border border-neutral-200 bg-white">
                      <div class="sticky top-0 z-[1] grid min-w-[420px] grid-cols-[minmax(0,1fr)_minmax(0,1fr)_88px] gap-4 border-b border-neutral-200 bg-neutral-50 px-4 py-2 text-[11px] font-medium text-neutral-500">
                        <span>原订单号</span>
                        <span>Amazon SKU</span>
                        <span class="text-center">数量</span>
                      </div>
                      <div v-for="(detail, detailIndex) in item.details || []" :key="`${detail.original_order_id}-${detail.amazon_sku}-${detailIndex}`" class="grid min-w-[420px] grid-cols-[minmax(0,1fr)_minmax(0,1fr)_88px] items-center gap-4 border-b border-neutral-100 px-4 py-2.5 text-xs last:border-b-0">
                        <button type="button" class="min-w-0 truncate text-left font-mono font-medium text-neutral-700 transition-colors hover:text-blue-600" title="点击复制原订单号" @click="copyOrderID(detail.original_order_id || item.order_id)">{{ detail.original_order_id || item.order_id }}</button>
                        <span class="min-w-0 break-all font-mono" :class="detail.amazon_sku ? 'text-neutral-700' : 'text-amber-700'">{{ detail.amazon_sku || '未映射' }}</span>
                        <span class="text-center font-semibold text-neutral-700">× {{ Number(detail.quantity) || 0 }}</span>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
              </template>
              <tr v-if="!loading && !items.length">
                <td :colspan="tableColumns.length" class="px-4 py-20 text-center">
                  <ClipboardDocumentListIcon class="mx-auto h-9 w-9 text-neutral-300" />
                  <div class="mt-3 text-sm text-neutral-500">暂无订单数据</div>
                </td>
              </tr>
              <tr v-if="loading">
                <td :colspan="tableColumns.length" class="px-4 py-20 text-center text-sm text-neutral-500">正在加载...</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="total > 0" class="flex flex-col gap-3 py-4 text-sm text-neutral-500 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-4">
            <span>共 {{ total }} 条</span>
            <label class="flex items-center gap-2 text-xs">
              <span>每页</span>
              <select
                v-model.number="pageSize"
                class="h-8 rounded border border-neutral-300 bg-white px-2 text-xs text-neutral-700 outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100"
                :disabled="loading"
                @change="changePageSize"
              >
                <option v-for="option in pageSizeOptions" :key="option" :value="option">{{ option }} 条</option>
              </select>
            </label>
          </div>
          <div class="flex items-center gap-1.5">
            <button type="button" class="flex h-9 w-9 items-center justify-center rounded border border-neutral-300 bg-white hover:bg-neutral-50 disabled:cursor-not-allowed disabled:opacity-40" :disabled="loading || page <= 1" title="上一页" @click="setPage(page - 1)">
              <ChevronLeftIcon class="h-4 w-4" />
            </button>
            <template v-for="item in paginationItems" :key="item">
              <span v-if="typeof item === 'string'" class="flex h-9 w-7 items-center justify-center text-neutral-400">...</span>
              <button
                v-else
                type="button"
                class="flex h-9 min-w-9 items-center justify-center rounded border px-2 text-xs font-medium transition-colors"
                :class="page === item ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-300 bg-white text-neutral-600 hover:bg-neutral-50'"
                :disabled="loading || page === item"
                :aria-current="page === item ? 'page' : undefined"
                @click="setPage(item)"
              >
                {{ item }}
              </button>
            </template>
            <button type="button" class="flex h-9 w-9 items-center justify-center rounded border border-neutral-300 bg-white hover:bg-neutral-50 disabled:cursor-not-allowed disabled:opacity-40" :disabled="loading || page >= totalPages" title="下一页" @click="setPage(page + 1)">
              <ChevronRightIcon class="h-4 w-4" />
            </button>
          </div>
        </div>
      </section>
    </div>

    <Modal v-model:open="exportDialogOpen" title="导出订单表格" size="xl">
      <div class="flex items-center justify-between gap-4 border-b border-neutral-200 pb-4">
        <div class="flex rounded-md bg-neutral-100 p-1">
          <button
            v-for="option in exportGroupOptions"
            :key="option.value"
            type="button"
            class="rounded px-3 py-1.5 text-xs font-medium transition-colors"
            :class="exportGroup === option.value ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500 hover:text-neutral-800'"
            @click="exportGroup = option.value"
          >
            {{ option.label }} {{ option.count }}
          </button>
        </div>
        <span class="text-xs text-neutral-500">已选 {{ selectedExportIDs.length }} 个订单</span>
      </div>

      <div v-if="currentExportItems.length" class="mt-4 overflow-hidden rounded-md border border-neutral-200">
        <div class="grid grid-cols-[36px_80px_minmax(150px,1fr)_minmax(170px,1fr)_120px_86px] gap-3 bg-neutral-50 px-3 py-2.5 text-xs font-medium text-neutral-500">
          <label class="flex items-center" title="全选当前分组">
            <input type="checkbox" class="h-4 w-4 rounded border-neutral-300" :checked="allCurrentExportSelected" @change="toggleCurrentExportItems">
          </label>
          <span>平台</span>
          <span>订单号</span>
          <span>追踪号</span>
          <span>承运人</span>
          <span>导出状态</span>
        </div>
        <div class="max-h-96 overflow-y-auto divide-y divide-neutral-100">
          <label v-for="item in currentExportItems" :key="item.id" class="grid cursor-pointer grid-cols-[36px_80px_minmax(150px,1fr)_minmax(170px,1fr)_120px_86px] gap-3 px-3 py-3 text-sm transition-colors hover:bg-neutral-50">
            <input v-model="selectedExportIDs" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-neutral-300" :value="item.id">
            <span class="text-xs font-medium text-neutral-600">{{ platformLabel(item.platform) }}</span>
            <span class="break-all font-mono text-xs font-semibold text-neutral-800">{{ item.order_id }}</span>
            <span class="break-all font-mono text-xs text-neutral-700">{{ item.tracking_number }}</span>
            <span class="text-xs text-neutral-600">{{ item.carrier || '未识别' }}</span>
            <span class="text-xs font-medium" :class="item.exported ? 'text-neutral-500' : 'text-emerald-700'">{{ item.exported ? '已导出' : '未导出' }}</span>
          </label>
        </div>
      </div>
      <div v-else class="py-14 text-center">
        <ClipboardDocumentListIcon class="mx-auto h-8 w-8 text-neutral-300" />
        <div class="mt-3 text-sm text-neutral-500">{{ exportGroup === 'new' ? '暂无未导出的订单' : '暂无已导出的订单' }}</div>
      </div>
      <template #footer>
        <Button variant="outline" @click="exportDialogOpen = false">取消</Button>
        <Button :disabled="exporting || !selectedExportIDs.length" @click="exportSelectedOrders">
          {{ exporting ? '正在导出...' : `导出已选（${selectedExportIDs.length}）` }}
        </Button>
      </template>
    </Modal>

    <Modal v-model:open="trackingEditorOpen" title="编辑追踪信息" size="md">
      <div class="grid grid-cols-[96px_minmax(0,1fr)] gap-x-4 gap-y-3 text-sm">
        <span class="text-neutral-500">平台</span>
        <span class="font-medium text-neutral-800">{{ platformLabel(editingItem?.platform) }}</span>
        <span class="text-neutral-500">订单号</span>
        <span class="break-all font-mono text-xs font-semibold text-neutral-800">{{ editingItem?.order_id }}</span>
      </div>
      <label class="mt-5 block">
        <span class="mb-1.5 block text-xs font-medium text-neutral-600">追踪号</span>
        <input ref="trackingInput" v-model="trackingDraft" type="text" maxlength="160" class="h-11 w-full rounded-md border border-neutral-300 px-3 font-mono text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100" placeholder="输入物流追踪号" @keydown.enter="saveTracking">
      </label>
      <label class="mt-4 block">
        <span class="mb-1.5 block text-xs font-medium text-neutral-600">备注</span>
        <textarea v-model="remarkDraft" rows="4" maxlength="1000" class="w-full resize-y rounded-md border border-neutral-300 px-3 py-2 text-sm leading-6 outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100" placeholder="填写追踪号相关情况" />
      </label>
      <label class="mt-4 block">
        <span class="mb-1.5 block text-xs font-medium text-neutral-600">亚马逊订单编号</span>
        <input v-model="amazonOrderDraft" type="text" maxlength="160" class="h-11 w-full rounded-md border border-neutral-300 px-3 font-mono text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100" placeholder="输入亚马逊订单编号">
      </label>
      <template #footer>
        <Button variant="outline" @click="trackingEditorOpen = false">取消</Button>
        <Button :disabled="savingTracking" @click="saveTracking">保存</Button>
      </template>
    </Modal>

    <Modal v-model:open="statusConfirmOpen" :title="statusAction === 'cancel' ? '确认取消订单' : statusAction === 'revoke_cancel' ? '撤销取消申请' : '申请取消订单'" size="sm">
      <div class="rounded-md border border-neutral-200 bg-neutral-50 px-4 py-3">
        <div class="text-xs font-medium text-neutral-500">订单号</div>
        <div class="mt-1 break-all font-mono text-sm font-semibold text-neutral-900">{{ statusEditingItem?.order_id }}</div>
      </div>
      <p class="mt-4 text-sm leading-6 text-neutral-600">
        {{ statusAction === 'cancel' ? '确认后订单将变为已取消状态，且不能再回填追踪号。' : statusAction === 'revoke_cancel' ? '撤销后订单将恢复为待发货状态，可以继续处理订单。' : '提交后订单将变为申请取消状态，等待管理员或数据部处理。' }}
      </p>
      <template #footer>
        <Button variant="outline" @click="statusConfirmOpen = false">返回</Button>
        <Button :disabled="savingStatus" :class="statusAction === 'cancel' ? 'bg-red-700 hover:bg-red-800' : ''" @click="saveOrderStatus">
          {{ savingStatus ? '正在提交...' : statusAction === 'cancel' ? '确认取消' : statusAction === 'revoke_cancel' ? '撤销申请' : '提交申请' }}
        </Button>
      </template>
    </Modal>
  </MainLayout>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ArrowDownTrayIcon,
  ArrowsRightLeftIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  ClipboardDocumentListIcon,
  CalendarDaysIcon,
  FunnelIcon,
  MagnifyingGlassIcon,
  NoSymbolIcon,
  ArrowUturnLeftIcon,
  PencilSquareIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import ExcelJS from 'exceljs'
import MainLayout from '@/components/layout/MainLayout.vue'
import Button from '@/components/ui/Button.vue'
import DatePicker from '@/components/ui/DatePicker.vue'
import Modal from '@/components/ui/Modal.vue'
import request from '@/utils/request'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

const authStore = useAuthStore()
const toast = useToast()
const allOrderPlatformOptions = [
  { value: 'TEMU', label: 'TEMU' },
  { value: 'TIKTOK', label: 'TikTok' },
  { value: 'LOWES', label: "Lowe's" },
]
const orderStatusOptions = [
  { value: '', label: '全部订单状态' },
  { value: 'pending_shipment', label: '待发货' },
  { value: 'shipped', label: '已发货' },
  { value: 'cancel_requested', label: '申请取消' },
  { value: 'cancelled', label: '已取消' },
]
const isAdmin = computed(() => (authStore.user?.roles || []).some((role) => role.name === 'admin'))
const isDataUser = computed(() => {
  const platform = String(authStore.user?.platform || '').trim().toLowerCase()
  const department = String(authStore.user?.department || '').trim().toLowerCase()
  return platform === 'data' || ['data', '数据', '数据部'].includes(department)
})
const canViewAll = ref(isAdmin.value || isDataUser.value)
const assignedPlatforms = computed(() => (authStore.user?.order_platforms || []).map(normalizePlatform).filter(Boolean))
const platformOptions = computed(() => {
  const options = canViewAll.value
    ? allOrderPlatformOptions
    : allOrderPlatformOptions.filter((option) => assignedPlatforms.value.includes(option.value))
  const fallback = options.length ? options : allOrderPlatformOptions.filter((option) => option.value === normalizePlatform(authStore.user?.platform))
  if (canViewAll.value || fallback.length > 1) {
    return [{ value: '', label: canViewAll.value ? '全部平台' : '全部负责平台' }, ...fallback]
  }
  return fallback
})
const selectedPlatform = ref((canViewAll.value || assignedPlatforms.value.length > 1) ? '' : (assignedPlatforms.value[0] || normalizePlatform(authStore.user?.platform)))
const orderStatus = ref('')
const dateFilterType = ref('day')
const createdDate = ref('')
const query = ref('')
const items = ref([])
const stats = ref({ total: 0, pending: 0, filled: 0 })
const loading = ref(false)
const loadError = ref('')
const page = ref(1)
const pageSize = ref(50)
const pageSizeOptions = [20, 50, 100]
const total = ref(0)
const expandedMergedOrders = ref(new Set())
const columnWidthStorageKey = 'order-tracking-column-widths-v1'
const tableColumns = ref([
  { key: 'order', label: '订单号', width: 208, defaultWidth: 208, minWidth: 140, maxWidth: 480 },
  { key: 'amazonOrder', label: '亚马逊订单编号', width: 190, defaultWidth: 190, minWidth: 140, maxWidth: 360 },
  { key: 'status', label: '状态', width: 152, defaultWidth: 152, minWidth: 120, maxWidth: 220 },
  { key: 'amazonSku', label: '商品明细', width: 280, defaultWidth: 280, minWidth: 200, maxWidth: 560 },
  { key: 'recipient', label: '收件人', width: 160, defaultWidth: 160, minWidth: 110, maxWidth: 360 },
  { key: 'phone', label: '手机号码', width: 150, defaultWidth: 150, minWidth: 110, maxWidth: 260 },
  { key: 'address', label: '地址', width: 360, defaultWidth: 360, minWidth: 180, maxWidth: 720 },
  { key: 'carrier', label: '承运人', width: 176, defaultWidth: 176, minWidth: 120, maxWidth: 360 },
  { key: 'tracking', label: '追踪号', width: 288, defaultWidth: 288, minWidth: 180, maxWidth: 560 },
  { key: 'remark', label: '备注', width: 256, defaultWidth: 256, minWidth: 160, maxWidth: 640 },
  { key: 'audit', label: '添加信息', width: 192, defaultWidth: 192, minWidth: 140, maxWidth: 360 },
])
const tableWidth = computed(() => tableColumns.value.reduce((sum, column) => sum + column.width, 0))
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const paginationItems = computed(() => {
  if (totalPages.value <= 7) return Array.from({ length: totalPages.value }, (_, index) => index + 1)
  const pages = [...new Set([1, page.value - 1, page.value, page.value + 1, totalPages.value])]
    .filter((value) => value >= 1 && value <= totalPages.value)
    .sort((a, b) => a - b)
  const result = []
  for (let index = 0; index < pages.length; index += 1) {
    if (index > 0 && pages[index] - pages[index - 1] > 1) result.push(`ellipsis-${pages[index - 1]}`)
    result.push(pages[index])
  }
  return result
})
const metrics = computed(() => [
  { label: '订单明细', value: stats.value.total },
  { label: '待回填追踪号', value: stats.value.pending },
  { label: '已回填追踪号', value: stats.value.filled },
])
const trackingEditorOpen = ref(false)
const editingItem = ref(null)
const trackingDraft = ref('')
const remarkDraft = ref('')
const amazonOrderDraft = ref('')
const trackingInput = ref(null)
const savingTracking = ref(false)
const statusConfirmOpen = ref(false)
const statusEditingItem = ref(null)
const statusAction = ref('')
const savingStatus = ref(false)
const exportDialogOpen = ref(false)
const exportableItems = ref([])
const selectedExportIDs = ref([])
const exportGroup = ref('new')
const loadingExportable = ref(false)
const exporting = ref(false)
const newExportItems = computed(() => exportableItems.value.filter((item) => !item.exported))
const previousExportItems = computed(() => exportableItems.value.filter((item) => item.exported))
const currentExportItems = computed(() => exportGroup.value === 'new' ? newExportItems.value : previousExportItems.value)
const exportGroupOptions = computed(() => [
  { value: 'new', label: '未导出', count: newExportItems.value.length },
  { value: 'previous', label: '已导出', count: previousExportItems.value.length },
])
const allCurrentExportSelected = computed(() => currentExportItems.value.length > 0 && currentExportItems.value.every((item) => selectedExportIDs.value.includes(item.id)))
const hasActiveFilters = computed(() => Boolean(orderStatus.value || createdDate.value))

function dateFilterRange() {
  if (dateFilterType.value === 'month' && /^\d{4}-\d{2}$/.test(createdDate.value)) {
    const [year, month] = createdDate.value.split('-').map(Number)
    const lastDay = new Date(year, month, 0).getDate()
    return { from: `${createdDate.value}-01`, to: `${createdDate.value}-${String(lastDay).padStart(2, '0')}` }
  }
  return { from: createdDate.value || '', to: createdDate.value || '' }
}

onMounted(() => {
  loadColumnWidths()
  loadItems()
})
onBeforeUnmount(() => finishColumnResize(false))

async function loadItems() {
  loading.value = true
  loadError.value = ''
  try {
    const dateRange = dateFilterRange()
    const response = await request.get('/order-tracking', {
      params: {
        platform: selectedPlatform.value || undefined,
        status: orderStatus.value || undefined,
        created_from: dateRange.from || undefined,
        created_to: dateRange.to || undefined,
        q: query.value.trim() || undefined,
        page: page.value,
        page_size: pageSize.value,
      },
    })
    items.value = response.data.items || []
    total.value = Number(response.data.total) || 0
    stats.value = response.data.stats || { total: 0, pending: 0, filled: 0 }
    canViewAll.value = Boolean(response.data.can_view_all)
    const lastPage = Math.max(1, Math.ceil(total.value / pageSize.value))
    if (page.value > lastPage) {
      page.value = lastPage
      await loadItems()
    }
  } catch (error) {
    items.value = []
    total.value = 0
    stats.value = { total: 0, pending: 0, filled: 0 }
    loadError.value = error.response?.data?.error || '订单汇总加载失败'
  } finally {
    loading.value = false
  }
}

function setPlatform(value) {
  selectedPlatform.value = value
  page.value = 1
  loadItems()
}

function applyFilters() {
  page.value = 1
  loadItems()
}

function clearFilters() {
  orderStatus.value = ''
  createdDate.value = ''
  dateFilterType.value = 'day'
  applyFilters()
}

function search() {
  page.value = 1
  loadItems()
}

function setPage(value) {
  const nextPage = Math.min(Math.max(1, value), totalPages.value)
  if (nextPage === page.value) return
  page.value = nextPage
  loadItems()
}

function changePageSize() {
  page.value = 1
  loadItems()
}

let columnResizeState = null
let previousBodyCursor = ''
let previousBodyUserSelect = ''

function clampColumnWidth(column, width) {
  return Math.min(column.maxWidth, Math.max(column.minWidth, Math.round(width)))
}

function columnWidthPercent(column) {
  return `${(column.width / tableWidth.value) * 100}%`
}

function loadColumnWidths() {
  try {
    const saved = JSON.parse(window.localStorage.getItem(columnWidthStorageKey) || '{}')
    for (const column of tableColumns.value) {
      if (Number.isFinite(saved[column.key])) column.width = clampColumnWidth(column, saved[column.key])
    }
  } catch {
    // Ignore unavailable storage or stale values and keep the defaults.
  }
}

function saveColumnWidths() {
  try {
    const widths = Object.fromEntries(tableColumns.value.map((column) => [column.key, column.width]))
    window.localStorage.setItem(columnWidthStorageKey, JSON.stringify(widths))
  } catch {
    // Column resizing still works when browser storage is unavailable.
  }
}

function startColumnResize(event, column) {
  if (event.button !== 0) return
  finishColumnResize(false)
  columnResizeState = { column, startX: event.clientX, startWidth: column.width }
  previousBodyCursor = document.body.style.cursor
  previousBodyUserSelect = document.body.style.userSelect
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  event.currentTarget?.focus({ preventScroll: true })
  window.addEventListener('pointermove', moveColumnResize)
  window.addEventListener('pointerup', finishColumnResize)
  window.addEventListener('pointercancel', finishColumnResize)
}

function moveColumnResize(event) {
  if (!columnResizeState) return
  const { column, startX, startWidth } = columnResizeState
  column.width = clampColumnWidth(column, startWidth + event.clientX - startX)
}

function finishColumnResize(persist = true) {
  if (!columnResizeState) return
  columnResizeState = null
  window.removeEventListener('pointermove', moveColumnResize)
  window.removeEventListener('pointerup', finishColumnResize)
  window.removeEventListener('pointercancel', finishColumnResize)
  document.body.style.cursor = previousBodyCursor
  document.body.style.userSelect = previousBodyUserSelect
  if (persist !== false) saveColumnWidths()
}

function resizeColumnBy(column, delta) {
  column.width = clampColumnWidth(column, column.width + delta)
  saveColumnWidths()
}

function resetColumnWidth(column) {
  column.width = column.defaultWidth
  saveColumnWidths()
}

function openTrackingEditor(item) {
  editingItem.value = item
  trackingDraft.value = item.tracking_number || ''
  remarkDraft.value = item.remark || ''
  amazonOrderDraft.value = item.amazon_order_id || ''
  trackingEditorOpen.value = true
  nextTick(() => trackingInput.value?.focus())
}

function openPackageTrackingEditor(order, packageItem) {
  openTrackingEditor({
    ...order,
    id: packageItem.id,
    package_key: packageItem.package_key,
    tracking_number: packageItem.tracking_number || '',
    amazon_order_id: packageItem.amazon_order_id || order.amazon_order_id || '',
    remark: packageItem.remark || '',
  })
}

function trackingNumbers(item) {
  const packageNumbers = (item?.packages || [])
    .map((packageItem) => String(packageItem.tracking_number || '').trim())
    .filter(Boolean)
  if (packageNumbers.length) return [...new Set(packageNumbers)]
  const fallback = String(item?.tracking_number || '').trim()
  return fallback ? [fallback] : []
}

function detailStats(item) {
  const details = item?.details || []
  const orderIDs = new Set(details.map((detail) => detail.original_order_id).filter(Boolean))
  return {
    orderCount: orderIDs.size || (details.length ? 1 : 0),
    itemCount: details.length,
    quantity: details.reduce((sum, detail) => sum + (Number(detail.quantity) || 0), 0),
  }
}

function isExpandableOrder(item) {
  return Boolean(item?.is_merged_order || (item?.details?.length || 0) > 1 || (item?.packages?.length || 0) > 1)
}

function groupedDetails(item) {
  const groups = new Map()
  for (const detail of item?.details || []) {
    const orderID = detail.original_order_id || item?.order_id || '-'
    if (!groups.has(orderID)) groups.set(orderID, [])
    groups.get(orderID).push(detail)
  }
  return [...groups].map(([orderID, groupItems]) => ({ orderID, items: groupItems }))
}

async function saveTracking() {
  if (!editingItem.value || savingTracking.value) return
  savingTracking.value = true
  try {
    const response = await request.put(`/order-tracking/${editingItem.value.id}`, {
      tracking_number: trackingDraft.value.trim(),
      amazon_order_id: amazonOrderDraft.value.trim(),
      remark: remarkDraft.value.trim(),
    })
    trackingEditorOpen.value = false
    await loadItems()
    toast.success(`已更新 ${response.data.updated || 1} 条订单明细`)
  } catch (error) {
    const data = error.response?.data
    if (data?.duplicate_tracking) {
      const conflicts = Array.isArray(data.conflicts) ? data.conflicts : []
      const orders = conflicts.map((item) => `${platformLabel(item.platform)} ${item.order_id}`).join('、')
      toast.warning(`该追踪号已被其他订单使用${orders ? `：${orders}` : ''}`)
    } else {
      toast.error(data?.error || '追踪号保存失败')
    }
  } finally {
    savingTracking.value = false
  }
}

async function copyOrderID(orderID) {
  const text = String(orderID || '').trim()
  if (!text) return
  try {
    if (navigator.clipboard?.writeText && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.setAttribute('readonly', '')
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      try {
        if (!document.execCommand('copy')) throw new Error('copy failed')
      } finally {
        document.body.removeChild(textarea)
      }
    }
    toast.success('订单号已复制')
  } catch {
    toast.error('订单号复制失败')
  }
}

function toggleMergedOrder(itemID) {
  const next = new Set(expandedMergedOrders.value)
  if (next.has(itemID)) next.delete(itemID)
  else next.add(itemID)
  expandedMergedOrders.value = next
}

function toggleMergedOrderFromRow(event, item) {
  if (!isExpandableOrder(item) || event.target.closest('button, a, input, select, textarea, label')) return
  toggleMergedOrder(item.id)
}

function openStatusConfirm(item, action) {
  statusEditingItem.value = item
  statusAction.value = action
  statusConfirmOpen.value = true
}

async function saveOrderStatus() {
  if (!statusEditingItem.value || !statusAction.value || savingStatus.value) return
  savingStatus.value = true
  try {
    const response = await request.put(`/order-tracking/${statusEditingItem.value.id}/status`, { action: statusAction.value })
    const message = statusAction.value === 'cancel' ? '订单已取消' : statusAction.value === 'revoke_cancel' ? '取消申请已撤销' : '取消申请已提交'
    statusConfirmOpen.value = false
    await loadItems()
    toast.success(`${message}，已更新 ${response.data.updated || 1} 条订单明细`)
  } catch (error) {
    toast.error(error.response?.data?.error || '订单状态更新失败')
  } finally {
    savingStatus.value = false
  }
}

async function openExportDialog() {
  if (loadingExportable.value) return
  loadingExportable.value = true
  try {
    const dateRange = dateFilterRange()
    const response = await request.get('/order-tracking/exportable', {
      params: {
        platform: selectedPlatform.value || undefined,
        created_from: dateRange.from || undefined,
        created_to: dateRange.to || undefined,
      },
    })
    exportableItems.value = response.data.items || []
    exportGroup.value = 'new'
    selectedExportIDs.value = newExportItems.value.map((item) => item.id)
    exportDialogOpen.value = true
  } catch (error) {
    toast.error(error.response?.data?.error || '未导出订单加载失败')
  } finally {
    loadingExportable.value = false
  }
}

function toggleCurrentExportItems(event) {
  const currentIDs = currentExportItems.value.map((item) => item.id)
  if (event.target.checked) {
    selectedExportIDs.value = [...new Set([...selectedExportIDs.value, ...currentIDs])]
    return
  }
  const currentSet = new Set(currentIDs)
  selectedExportIDs.value = selectedExportIDs.value.filter((id) => !currentSet.has(id))
}

async function exportSelectedOrders() {
  if (exporting.value || !selectedExportIDs.value.length) return
  exporting.value = true
  try {
    const response = await request.post('/order-tracking/export', { ids: selectedExportIDs.value })
    const exportedItems = response.data.items || []
    const workbook = new ExcelJS.Workbook()
    const worksheet = workbook.addWorksheet('订单物流')
    worksheet.columns = [
      { header: '订单号', key: 'order_id', width: 26 },
      { header: '亚马逊订单编号', key: 'amazon_order_id', width: 28 },
      { header: '状态', key: 'status_label', width: 16 },
      { header: '商品明细（SKU）', key: 'detail_skus', width: 32 },
      { header: '商品明细（数量）', key: 'detail_quantities', width: 18 },
      { header: '收件人', key: 'recipient', width: 24 },
      { header: '手机号码', key: 'phone', width: 22 },
      { header: '地址', key: 'address', width: 55 },
      { header: '承运人', key: 'carrier', width: 22 },
      { header: '追踪号', key: 'tracking_number', width: 28 },
      { header: '备注', key: 'remark', width: 36 },
      { header: '添加人', key: 'importer_label', width: 20 },
      { header: '添加时间', key: 'created_time', width: 24 },
    ]
    const exportRows = exportedItems.flatMap((item) => {
      const details = item.details?.length ? item.details : [{}]
      return details.map((detail) => ({
        ...item,
        status_label: orderStatusLabel(item.status),
        detail_skus: detail.amazon_sku || '未映射',
        detail_quantities: detail.quantity ?? '',
        importer_label: importerLabel(item),
        created_time: formatTimestamp(item.created_at),
      }))
    })
    worksheet.addRows(exportRows)
    worksheet.columns.forEach((column) => { column.alignment = { vertical: 'top', wrapText: true } })
    for (const key of ['order_id', 'amazon_order_id', 'phone', 'tracking_number']) worksheet.getColumn(key).numFmt = '@'
    worksheet.views = [{ state: 'frozen', ySplit: 1 }]
    worksheet.getRow(1).font = { bold: true }
    const buffer = await workbook.xlsx.writeBuffer()
    const link = document.createElement('a')
    link.href = URL.createObjectURL(new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }))
    link.download = `订单物流_${new Date().toISOString().slice(0, 10)}.xlsx`
    link.click()
    URL.revokeObjectURL(link.href)
    exportDialogOpen.value = false
    await loadItems()
    toast.success(`已导出 ${exportRows.length} 条商品明细`)
  } catch (error) {
    toast.error(error.response?.data?.error || '订单表格导出失败')
  } finally {
    exporting.value = false
  }
}

function normalizePlatform(value) {
  const normalized = String(value || '').trim().toUpperCase()
  if (normalized === 'TEMU') return 'TEMU'
  if (['TIKTOK', 'TIKTOK SHOP', 'TK'].includes(normalized)) return 'TIKTOK'
  if (['LOWES', "LOWE'S", 'LOWE’S', 'LOWE'].includes(normalized)) return 'LOWES'
  return ''
}

function platformLabel(value) {
  return ({ TEMU: 'TEMU', TIKTOK: 'TikTok', LOWES: "Lowe's" }[value] || value || '-')
}

function platformBadgeClass(value) {
  return {
    TIKTOK: 'bg-sky-50 text-sky-700',
    LOWES: 'bg-emerald-50 text-emerald-700',
  }[value] || 'bg-orange-50 text-orange-700'
}

function orderStatusLabel(value) {
  return {
    pending_shipment: '待发货',
    shipped: '已发货',
    cancel_requested: '申请取消',
    cancelled: '已取消',
  }[value] || '待发货'
}

function orderStatusClass(value) {
  return {
    pending_shipment: 'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-200',
    shipped: 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200',
    cancel_requested: 'bg-orange-50 text-orange-700 ring-1 ring-inset ring-orange-200',
    cancelled: 'bg-neutral-100 text-neutral-500 ring-1 ring-inset ring-neutral-200',
  }[value] || 'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-200'
}

function importerLabel(item) {
  if (item?.imported_by_name) return item.imported_by_name
  if (item?.imported_by) return `用户 #${item.imported_by}`
  return '-'
}

function trackingUpdaterLabel(item) {
  if (item?.tracking_updated_by_name) return item.tracking_updated_by_name
  if (item?.tracking_updated_by) return `用户 #${item.tracking_updated_by}`
  return '-'
}

function formatTimestamp(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
</script>
