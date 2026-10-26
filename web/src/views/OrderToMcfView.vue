<template>
  <MainLayout>
    <div class="mx-auto w-full max-w-[1600px] px-4 py-6 sm:px-6 lg:px-8">
      <header class="flex flex-col gap-4 border-b border-neutral-200 pb-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-neutral-900">订单转 MCF</h1>
          <p class="mt-1 text-sm text-neutral-500">订单转换与平台映射管理</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <router-link to="/order-tracking" class="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-neutral-300 bg-white px-4 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50 hover:text-neutral-900">
            <ClipboardDocumentListIcon class="h-4 w-4" />
            订单汇总
          </router-link>
          <div class="inline-flex h-10 w-fit rounded-md bg-neutral-100 p-1" aria-label="来源平台">
            <button
              v-for="option in availablePlatformOptions"
              :key="option.value"
              type="button"
              class="min-w-24 rounded px-4 text-sm font-medium transition-colors"
              :class="platform === option.value ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500 hover:text-neutral-900'"
              @click="setPlatform(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </header>

      <nav class="flex gap-6 border-b border-neutral-200" aria-label="订单工具工作区">
        <button
          v-for="item in workspaceOptions"
          :key="item.value"
          type="button"
          class="flex h-12 items-center gap-2 border-b-2 px-1 text-sm font-medium transition-colors"
          :class="activeWorkspace === item.value ? 'border-neutral-900 text-neutral-900' : 'border-transparent text-neutral-500 hover:text-neutral-800'"
          @click="setWorkspace(item.value)"
        >
          <component :is="item.icon" class="h-4 w-4" />
          {{ item.label }}
          <span v-if="item.value === 'mapping' && unmappedSkuRows.length" class="rounded bg-red-50 px-1.5 py-0.5 text-xs text-red-700">
            {{ unmappedSkuRows.length }}
          </span>
        </button>
      </nav>

      <section v-if="activeWorkspace === 'convert'" class="pb-10">
        <div class="mt-6 grid gap-0 border-y border-neutral-200 bg-white lg:grid-cols-[minmax(0,1fr)_240px]">
          <div class="py-6 lg:pr-8">
            <div class="mb-3 flex items-center justify-between gap-3">
              <h2 class="text-base font-semibold text-neutral-900">订单文件</h2>
              <button v-if="fileName" type="button" class="inline-flex items-center gap-1.5 text-xs text-neutral-600 hover:text-neutral-900" @click="clearFile">
                <ArrowPathIcon class="h-4 w-4" />
                清除
              </button>
            </div>
            <input
              ref="fileInput"
              class="hidden"
              type="file"
              @change="onFileChange"
            >
            <button
              type="button"
              class="flex min-h-[104px] w-full items-center gap-4 rounded-md border border-dashed px-5 text-left transition-colors"
              :class="dragging ? 'border-neutral-900 bg-neutral-50' : fileName ? 'border-emerald-300 bg-emerald-50/40 hover:border-emerald-400' : 'border-neutral-300 hover:border-neutral-500 hover:bg-neutral-50/60'"
              @click="fileInput?.click()"
              @dragover.prevent="dragging = true"
              @dragleave.prevent="dragging = false"
              @drop.prevent="onDrop"
            >
              <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-md" :class="fileName ? 'bg-emerald-100 text-emerald-700' : 'bg-neutral-100 text-neutral-600'">
                <DocumentTextIcon v-if="fileName" class="h-5 w-5" />
                <CloudArrowUpIcon v-else class="h-5 w-5" />
              </span>
              <span class="min-w-0">
                <span class="block truncate text-sm font-medium text-neutral-900">
                  {{ fileName || '选择订单文件' }}
                </span>
                <span class="mt-1 block text-xs text-neutral-500">
                  {{ parsing ? '正在解析...' : fileName ? `${formatFileSize(fileSize)} · 本地解析` : '最大 10 MB' }}
                </span>
              </span>
            </button>
          </div>
          <div class="flex items-center justify-center border-t border-neutral-200 py-6 lg:border-l lg:border-t-0 lg:px-8">
            <Button size="lg" class="w-full whitespace-nowrap" :disabled="!canExport" :title="exportDisabledReason" @click="downloadMCF">
              <ArrowDownTrayIcon class="h-4 w-4" />
              生成 MCF 文件
            </Button>
          </div>
        </div>

        <div v-if="fieldIssue" class="border-b border-red-200 bg-red-50 px-4 py-4">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex min-w-0 items-start gap-3">
              <ExclamationTriangleIcon class="mt-0.5 h-5 w-5 shrink-0 text-red-700" />
              <div>
                <div class="text-sm font-semibold text-red-900">字段匹配失败，已阻止生成 MCF</div>
                <div class="mt-1 text-sm text-red-800">
                  未匹配：{{ fieldIssueMissingLabels.join('、') }}
                </div>
              </div>
            </div>
            <Button variant="outline" @click="openFieldRepair">
              <AdjustmentsHorizontalIcon class="h-4 w-4" />
              修复字段规则
            </Button>
          </div>
        </div>

        <div v-if="records.length" class="grid border-b border-neutral-200 sm:grid-cols-3 lg:grid-cols-6">
          <div v-for="metric in metrics" :key="metric.label" class="border-b border-neutral-100 px-4 py-4 sm:border-b-0 sm:border-r last:border-r-0">
            <div class="text-xs text-neutral-500">{{ metric.label }}</div>
            <div class="mt-1 text-lg font-semibold text-neutral-900">{{ metric.value }}</div>
          </div>
        </div>

        <div v-if="unmappedSkuRows.length" class="flex flex-col gap-3 border-b border-red-200 bg-red-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-start gap-2.5 text-sm text-red-800">
            <ExclamationTriangleIcon class="mt-0.5 h-4 w-4 shrink-0" />
            <span><strong>{{ unmappedSkuRows.length }}</strong> 个平台 SKU 尚未映射，订单暂时不能导出。</span>
          </div>
          <Button variant="outline" size="sm" @click="setWorkspace('mapping')">
            <LinkIcon class="h-4 w-4" />
            管理 SKU 映射
          </Button>
        </div>

        <div v-if="records.length" class="mt-7">
          <div class="mb-3 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <h2 class="text-base font-semibold text-neutral-900">订单校验</h2>
            <div class="flex w-fit flex-wrap gap-1 rounded-md bg-neutral-100 p-1">
              <button
                v-for="option in filterOptions"
                :key="option.value"
                type="button"
                class="rounded px-3 py-1.5 text-xs font-medium transition-colors"
                :class="statusFilter === option.value ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500 hover:text-neutral-900'"
                @click="setFilter(option.value)"
              >
                {{ option.label }} {{ option.count }}
              </button>
            </div>
          </div>

          <div class="overflow-hidden border-y border-neutral-200 bg-white">
            <div class="overflow-x-auto">
              <table class="w-full min-w-[1180px] text-sm">
                <thead class="bg-neutral-50 text-left text-xs font-medium text-neutral-500">
                  <tr>
                    <th class="w-24 px-4 py-3">状态</th>
                    <th class="w-52 px-4 py-3">订单号</th>
                    <th class="w-44 px-4 py-3">SKU</th>
                    <th class="w-20 px-4 py-3 text-right">数量</th>
                    <th class="w-40 px-4 py-3">收件人</th>
                    <th class="px-4 py-3">地址</th>
                    <th class="w-64 px-4 py-3">问题</th>
                    <th class="w-16 px-4 py-3"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-neutral-100">
                  <tr v-for="row in pagedDisplayRows" :key="row.id" class="align-top">
                    <td class="px-4 py-3"><StatusBadge :status="row.status" /></td>
                    <td class="px-4 py-3 font-mono text-xs text-neutral-700">{{ row.orderId }}</td>
                    <td class="px-4 py-3">
                      <div class="font-mono text-xs text-neutral-500">{{ row.sourceSkus.join(', ') }}</div>
                      <div class="mt-1 font-mono text-xs font-semibold text-neutral-900">{{ row.amazonSku || '未映射' }}</div>
                    </td>
                    <td class="px-4 py-3 text-right tabular-nums">{{ row.quantity }}</td>
                    <td class="px-4 py-3 text-neutral-700">{{ row.recipient || '-' }}</td>
                    <td class="px-4 py-3 text-xs leading-5 text-neutral-600">{{ formatAddress(row) }}</td>
                    <td class="px-4 py-3 text-xs leading-5">
                      <div v-for="error in row.errors" :key="error" class="text-red-700">{{ error }}</div>
                      <div v-for="warning in row.warnings" :key="warning" class="text-amber-700">{{ warning }}</div>
                    </td>
                    <td class="px-4 py-3 text-right">
                      <button
                        v-if="row.status !== 'excluded'"
                        type="button"
                        class="flex h-8 w-8 items-center justify-center rounded text-neutral-600 hover:bg-neutral-100"
                        title="编辑收件信息"
                        @click="openAddressEditor(row)"
                      >
                        <PencilSquareIcon class="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                  <tr v-if="!pagedDisplayRows.length">
                    <td colspan="8" class="px-4 py-12 text-center text-sm text-neutral-500">当前筛选没有订单</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <PaginationBar
              v-if="filteredDisplayRows.length > pageSize"
              :page="currentPage"
              :total-pages="totalPages"
              @previous="currentPage -= 1"
              @next="currentPage += 1"
            />
          </div>
        </div>

      </section>

      <section v-else-if="activeWorkspace === 'mapping'" class="pt-6">
        <div class="flex flex-col gap-4 border-b border-neutral-200 pb-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-semibold text-neutral-900">SKU 映射</h2>
              <span class="text-sm text-neutral-500">{{ mappingTotal }} 条</span>
            </div>
            <div class="mt-2 flex items-center gap-1.5 text-xs" :class="mappingListCanEdit ? 'text-emerald-700' : 'text-neutral-500'">
              <CheckCircleIcon v-if="mappingListCanEdit" class="h-4 w-4" />
              <LockClosedIcon v-else class="h-4 w-4" />
              {{ mappingListCanEdit ? `可维护 ${platformLabel} 映射` : `${platformLabel} 映射只读` }}
            </div>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <button type="button" class="h-10 whitespace-nowrap rounded-md border px-3 text-sm font-medium transition-colors" :class="mappingDuplicatesOnly ? 'border-amber-300 bg-amber-50 text-amber-800' : 'border-neutral-300 bg-white text-neutral-600 hover:bg-neutral-50'" @click="toggleDuplicateMappings">
              {{ mappingDuplicatesOnly ? '显示全部' : '仅看重复' }}
            </button>
            <div class="relative min-w-72">
              <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-neutral-400" />
              <input
                v-model="mappingQuery"
                type="search"
                class="h-10 w-full rounded-md border border-neutral-300 pl-9 pr-3 text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100"
                placeholder="搜索 SKU 或备注"
                @keydown.enter="searchMappings"
              >
            </div>
            <Button variant="outline" @click="searchMappings">查询</Button>
            <Button variant="outline" :disabled="!mappingListCanEdit" @click="openMappingImport">
              <ArrowUpTrayIcon class="h-4 w-4" />
              导入表格
            </Button>
            <Button :disabled="!mappingListCanEdit" @click="openMappingEditor()">
              <PlusIcon class="h-4 w-4" />
              新增映射
            </Button>
          </div>
        </div>

        <div class="mt-4 grid gap-3 border-b border-neutral-200 pb-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
          <div class="flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
            <ExclamationTriangleIcon class="mt-0.5 h-4 w-4 shrink-0 text-amber-700" />
            <div class="leading-5">
              <div class="font-semibold">重复 Amazon SKU 需要指定当前平台 SKU</div>
              <div class="mt-0.5 text-xs text-amber-800">订单校验只允许当前项生成 MCF；切换后会长期保存，不会影响其他平台 SKU 的映射记录。</div>
            </div>
          </div>
          <div class="flex items-center gap-2 text-xs text-neutral-500"><span class="h-2 w-2 rounded-full bg-amber-400"></span>{{ duplicateMappingCount }} 个重复映射组</div>
        </div>

        <div v-if="unmappedSkuRows.length" class="border-b border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
          当前文件待补充：{{ unmappedSkuRows.map((item) => item.sourceSku).slice(0, 6).join('、') }}<span v-if="unmappedSkuRows.length > 6"> 等 {{ unmappedSkuRows.length }} 个</span>
        </div>

        <div class="overflow-hidden border-b border-neutral-200 bg-white">
          <div class="overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead class="bg-neutral-50 text-left text-xs font-medium text-neutral-500">
                <tr>
                  <th class="w-[20%] px-4 py-3">平台 SKU</th>
                  <th class="w-[17%] px-4 py-3">Amazon SKU</th>
                  <th class="w-[17%] px-4 py-3">非亚 SKU</th>
                  <th class="min-w-72 px-4 py-3">备注</th>
                  <th class="w-52 px-4 py-3">更新信息</th>
                  <th class="w-24 px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-neutral-100">
                <tr v-for="item in mappingList" :key="item.id">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <span class="font-mono text-xs font-semibold text-neutral-900">{{ item.platform_sku }}</span>
                      <span v-if="item.is_current && item.related_amazon_skus?.length > 1" class="whitespace-nowrap rounded bg-emerald-50 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700 ring-1 ring-inset ring-emerald-200">当前</span>
                    </div>
                  </td>
                  <td class="px-4 py-3 font-mono text-xs text-neutral-700">{{ item.amazon_sku }}</td>
                  <td class="px-4 py-3 font-mono text-xs text-neutral-700">{{ item.non_amazon_sku || '-' }}</td>
                  <td class="px-4 py-3">
                    <div v-if="item.related_amazon_skus?.length > 1" class="text-xs leading-5 text-amber-700">
                      同一平台 SKU 维护 {{ item.related_amazon_skus.length }} 个 Amazon SKU；当前：<span class="font-mono font-semibold">{{ item.current_amazon_sku }}</span>
                    </div>
                    <div v-if="item.remark" class="mt-1 whitespace-pre-wrap text-xs leading-5 text-neutral-600">{{ item.remark }}</div>
                    <span v-if="item.related_amazon_skus?.length <= 1 && !item.remark" class="text-xs text-neutral-400">-</span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="min-w-40">
                      <div class="truncate text-xs font-medium text-neutral-700" :title="updaterLabel(item)">{{ updaterLabel(item) }}</div>
                      <div class="mt-0.5 whitespace-nowrap text-[11px] text-neutral-400">{{ formatTimestamp(item.updated_at) }}</div>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex justify-end gap-1">
                      <button v-if="item.related_amazon_skus?.length > 1" type="button" class="h-8 whitespace-nowrap rounded px-2 text-xs font-medium text-amber-700 hover:bg-amber-50 disabled:opacity-40" :disabled="!mappingListCanEdit" title="切换当前 Amazon SKU" @click="openMappingSwitch(item)">切换 Amazon SKU</button>
                      <button type="button" class="flex h-8 w-8 items-center justify-center rounded text-neutral-600 hover:bg-neutral-100 disabled:opacity-40" :disabled="!mappingListCanEdit" title="编辑映射" @click="openMappingEditor(item)">
                        <PencilSquareIcon class="h-4 w-4" />
                      </button>
                      <button type="button" class="flex h-8 w-8 items-center justify-center rounded text-red-600 hover:bg-red-50 disabled:opacity-40" :disabled="!mappingListCanEdit" title="停用映射" @click="deactivateMapping(item)">
                        <TrashIcon class="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!mappingLoading && !mappingList.length">
                  <td colspan="6" class="px-4 py-16 text-center text-sm text-neutral-500">暂无 SKU 映射</td>
                </tr>
                <tr v-if="mappingLoading">
                  <td colspan="6" class="px-4 py-16 text-center text-sm text-neutral-500">正在加载...</td>
                </tr>
              </tbody>
            </table>
          </div>
          <PaginationBar
            v-if="mappingTotalPages > 1"
            :page="mappingPage"
            :total-pages="mappingTotalPages"
            @previous="changeMappingPage(mappingPage - 1)"
            @next="changeMappingPage(mappingPage + 1)"
          />
        </div>
      </section>

      <section v-else-if="activeWorkspace === 'history'" class="pt-6">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div><h2 class="text-sm font-semibold text-neutral-900">转换历史</h2><p class="mt-1 text-xs text-neutral-500">保存每次转换的原始订单文件和 MCF 文件，可随时重新下载。</p></div>
          <Button variant="outline" :disabled="historyLoading" @click="loadHistory">刷新</Button>
        </div>
        <p v-if="historyError" role="alert" class="py-4 text-sm text-red-700">{{ historyError }}</p>
        <div v-else-if="historyLoading" class="py-16 text-center text-sm text-neutral-500">正在加载历史记录...</div>
        <div v-else-if="!historyItems.length" class="py-16 text-center text-sm text-neutral-500">暂无转换历史，生成 MCF 后会自动保存到这里。</div>
        <div v-else class="divide-y divide-neutral-200 border-y border-neutral-200">
          <article v-for="record in historyItems" :key="record.id" class="grid gap-4 py-4 sm:grid-cols-[160px_minmax(0,1fr)_minmax(0,1fr)]">
            <div class="text-xs text-neutral-500"><div class="mb-1 font-medium text-neutral-800">{{ record.platform }} · #{{ record.id }}</div>{{ formatTimestamp(record.created_at) }}</div>
            <div class="min-w-0"><div class="mb-1 text-xs text-neutral-500">原始订单文件</div><button type="button" class="break-all text-left text-sm text-neutral-800 underline decoration-neutral-300 underline-offset-4 hover:text-blue-600 disabled:opacity-50" :disabled="Boolean(historyDownloading)" @click="downloadHistoryFile(record, 'source')">{{ record.source_name }}</button></div>
            <div class="min-w-0"><div class="mb-1 text-xs text-neutral-500">MCF 文件</div><button type="button" class="break-all text-left text-sm text-neutral-800 underline decoration-neutral-300 underline-offset-4 hover:text-blue-600 disabled:opacity-50" :disabled="Boolean(historyDownloading)" @click="downloadHistoryFile(record, 'mcf')">{{ record.mcf_name }}</button></div>
          </article>
        </div>
        <p v-if="historyDownloading" role="status" class="mt-3 text-xs text-neutral-500">正在下载文件...</p>
        <PaginationBar v-if="historyTotal > 20" :page="historyPage" :total-pages="Math.ceil(historyTotal / 20)" @previous="changeHistoryPage(historyPage - 1)" @next="changeHistoryPage(historyPage + 1)" />
      </section>

      <section v-else class="pt-6">
        <div class="flex flex-col gap-4 border-b border-neutral-200 pb-5 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-semibold text-neutral-900">字段规则</h2>
              <span class="rounded bg-neutral-100 px-2 py-0.5 text-xs text-neutral-600">{{ platformLabel }}</span>
            </div>
            <p class="mt-2 text-xs text-neutral-500">全局配置 · 保存后作用于后续导入</p>
          </div>
          <div class="flex gap-2">
            <Button v-if="editingFields" variant="outline" @click="cancelFieldEditing">取消</Button>
            <Button v-if="editingFields" :disabled="savingFields" @click="prepareFieldSave">
              <CheckIcon class="h-4 w-4" />
              保存整套规则
            </Button>
            <Button v-else :disabled="!canEditFields || fieldLoading" @click="startFieldEditing">
              <PencilSquareIcon class="h-4 w-4" />
              编辑规则
            </Button>
          </div>
        </div>

        <div v-if="fieldIssue" class="border-b border-red-200 bg-red-50 px-4 py-4">
          <div class="flex items-start gap-3">
            <ExclamationTriangleIcon class="mt-0.5 h-5 w-5 shrink-0 text-red-700" />
            <div class="min-w-0">
              <div class="text-sm font-semibold text-red-900">本次导入有 {{ fieldIssue.missingFields.length }} 个必需字段未匹配</div>
              <div class="mt-2 flex flex-wrap gap-2">
                <span v-for="label in fieldIssueMissingLabels" :key="label" class="rounded border border-red-200 bg-white px-2 py-1 text-xs text-red-800">{{ label }}</span>
              </div>
              <details v-if="fieldIssue.detectedHeaders.length" class="mt-3 text-xs text-neutral-600">
                <summary class="cursor-pointer font-medium text-neutral-700">查看本次文件表头</summary>
                <div class="mt-2 flex flex-wrap gap-1.5">
                  <span v-for="header in fieldIssue.detectedHeaders" :key="header" class="rounded bg-white px-2 py-1 font-mono text-neutral-600 ring-1 ring-inset ring-neutral-200">{{ header }}</span>
                </div>
              </details>
            </div>
          </div>
        </div>

        <div class="overflow-hidden border-b border-neutral-200 bg-white">
          <div class="overflow-x-auto">
            <table class="min-w-full table-fixed text-sm">
              <thead class="bg-neutral-50 text-left text-xs font-medium text-neutral-500">
                <tr>
                  <th class="w-56 px-4 py-3">标准字段</th>
                  <th class="w-28 px-4 py-3">类型</th>
                  <th class="px-4 py-3">源文件表头别名</th>
                  <th class="w-44 px-4 py-3">更新人</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-neutral-100">
                <tr v-for="item in fieldRules" :key="item.field_key" class="align-top" :class="isMissingField(item.field_key) ? 'bg-red-50/60' : ''">
                  <td class="px-4 py-4">
                    <div class="font-medium text-neutral-900">{{ fieldMeta(item.field_key).label }}</div>
                    <div class="mt-1 font-mono text-xs text-neutral-500">{{ fieldMeta(item.field_key).code }}</div>
                  </td>
                  <td class="px-4 py-4">
                    <span v-if="item.required" class="rounded bg-red-50 px-2 py-1 text-xs font-medium text-red-700">必填</span>
                    <span v-else class="rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-600">可选</span>
                  </td>
                  <td class="px-4 py-3">
                    <div v-if="editingFields">
                      <textarea
                        v-model="fieldDrafts[item.field_key]"
                        rows="2"
                        class="min-h-16 w-full resize-y rounded-md border border-neutral-300 px-3 py-2 text-sm leading-5 outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100"
                        :placeholder="item.required ? '至少填写一个表头，一行一个' : '一行一个表头别名'"
                      />
                      <select
                        v-if="isMissingField(item.field_key) && fieldIssue.detectedHeaders.length"
                        class="mt-2 h-9 max-w-md rounded-md border border-neutral-300 bg-white px-3 text-xs text-neutral-700 outline-none focus:border-neutral-700"
                        @change="appendDetectedHeader(item.field_key, $event)"
                      >
                        <option value="">从本次文件选择正确表头</option>
                        <option v-for="header in fieldIssue.detectedHeaders" :key="header" :value="header">{{ header }}</option>
                      </select>
                    </div>
                    <div v-else class="flex flex-wrap gap-2">
                      <span v-for="header in item.source_headers" :key="header" class="rounded border border-neutral-200 bg-neutral-50 px-2.5 py-1 font-mono text-xs text-neutral-700">{{ header }}</span>
                      <span v-if="!item.source_headers.length" class="text-xs text-neutral-400">未配置</span>
                    </div>
                  </td>
                  <td class="px-4 py-4 text-xs text-neutral-500">
                    <div class="font-medium text-neutral-700">{{ updaterLabel(item) }}</div>
                    <div v-if="item.updated_at" class="mt-1">{{ formatTimestamp(item.updated_at) }}</div>
                  </td>
                </tr>
                <tr v-if="fieldLoading">
                  <td colspan="4" class="px-4 py-16 text-center text-sm text-neutral-500">正在加载...</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>

    <Modal v-model:open="mappingEditorOpen" :title="mappingForm.id ? '编辑 SKU 映射' : '新增 SKU 映射'" size="lg">
      <div class="space-y-4">
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium text-neutral-600">{{ platformLabel }} SKU</span>
          <input v-model="mappingForm.platformSku" type="text" class="h-10 w-full rounded-md border border-neutral-300 px-3 font-mono text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100 disabled:bg-neutral-50 disabled:text-neutral-500" :disabled="Boolean(mappingForm.id)" placeholder="平台导出的 Seller SKU">
        </label>
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium text-neutral-600">非亚 SKU</span>
          <input v-model="mappingForm.nonAmazonSku" type="text" class="h-10 w-full rounded-md border border-neutral-300 px-3 font-mono text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100" placeholder="输入非亚 SKU">
        </label>
        <div>
          <span class="mb-1.5 block text-xs font-medium text-neutral-600">Amazon SKU</span>
          <input
            v-model="mappingForm.amazonSku"
            type="text"
            class="h-10 w-full rounded-md border border-neutral-300 px-3 font-mono text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100"
            placeholder="输入 SKU、商品名或 SPU"
            autocomplete="off"
            @input="scheduleSkuSearch"
          >
          <div class="mt-2 max-h-64 overflow-y-auto rounded-md border border-neutral-200 bg-white">
            <button
              v-for="item in skuSearchResults"
              :key="item.id"
              type="button"
              class="flex w-full items-start justify-between gap-4 border-b border-neutral-100 px-3 py-3 text-left last:border-0 hover:bg-neutral-50"
              :class="mappingForm.amazonSku === item.sku ? 'bg-neutral-50' : ''"
              @click="selectProductSku(item)"
            >
              <span class="min-w-0">
                <span class="block font-mono text-sm font-semibold text-neutral-900">{{ item.sku }}</span>
                <span class="mt-1 block truncate text-xs text-neutral-500">{{ [item.name, item.spu, item.style_name, item.color].filter(Boolean).join(' / ') }}</span>
              </span>
              <CheckIcon v-if="mappingForm.amazonSku === item.sku" class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
            </button>
            <div v-if="skuSearching" class="px-3 py-6 text-center text-xs text-neutral-500">正在搜索...</div>
            <div v-else-if="!mappingForm.amazonSku.trim()" class="px-3 py-6 text-center text-xs text-neutral-500">输入关键词搜索 Amazon SKU</div>
            <div v-else-if="!skuSearchResults.length" class="px-3 py-6 text-center text-xs text-neutral-500">没有匹配的 Amazon SKU</div>
          </div>
        </div>
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium text-neutral-600">人工备注</span>
          <textarea v-model="mappingForm.remark" rows="3" maxlength="1000" class="w-full resize-y rounded-md border border-neutral-300 px-3 py-2 text-sm leading-5 outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100" placeholder="可填写选用原因、款式差异等说明"></textarea>
          <span class="mt-1 block text-right text-[11px] text-neutral-400">{{ mappingForm.remark.length }}/1000</span>
        </label>
        <div v-if="mappingForm.relatedAmazonSkus.length > 1" class="rounded-md border border-amber-200 bg-amber-50 px-3 py-3">
          <div class="text-xs font-semibold text-amber-900">该平台 SKU 已维护多个 Amazon SKU</div>
          <div class="mt-1 text-xs leading-5 text-amber-800">{{ mappingForm.relatedAmazonSkus.join('、') }}</div>
          <label class="mt-3 flex cursor-pointer items-start gap-2 text-sm text-neutral-800">
            <input v-model="mappingForm.isCurrent" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-neutral-300">
            <span>将 <span class="font-mono font-semibold">{{ mappingForm.amazonSku }}</span> 设为当前 Amazon SKU</span>
          </label>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="mappingEditorOpen = false">取消</Button>
        <Button :disabled="savingMapping || !mappingForm.platformSku.trim() || !mappingForm.amazonSku.trim()" @click="saveMapping">
          保存映射
        </Button>
      </template>
    </Modal>

    <Modal v-model:open="mappingSwitchOpen" title="切换当前 Amazon SKU" size="md">
      <div v-if="mappingSwitchTarget" class="space-y-4">
        <div class="rounded-md border border-neutral-200 bg-neutral-50 px-3 py-3">
          <div class="text-xs text-neutral-500">平台 SKU</div>
          <div class="mt-1 font-mono text-sm font-semibold text-neutral-900">{{ mappingSwitchTarget.platform_sku }}</div>
        </div>
        <div>
          <div class="mb-2 text-xs font-medium text-neutral-600">选择当前使用的 Amazon SKU</div>
          <div class="divide-y divide-neutral-100 overflow-hidden rounded-md border border-neutral-200">
            <button v-for="candidate in mappingSwitchCandidates" :key="candidate.id" type="button" class="flex w-full items-center justify-between gap-3 px-3 py-3 text-left hover:bg-neutral-50" :class="mappingSwitchSelectedID === candidate.id ? 'bg-emerald-50/70' : 'bg-white'" @click="mappingSwitchSelectedID = candidate.id">
              <span>
                <span class="block font-mono text-sm font-semibold text-neutral-900">{{ candidate.amazon_sku }}</span>
                <span v-if="candidate.remark" class="mt-1 block text-xs text-neutral-500">{{ candidate.remark }}</span>
              </span>
              <span class="flex h-5 w-5 items-center justify-center rounded-full border" :class="mappingSwitchSelectedID === candidate.id ? 'border-emerald-600 bg-emerald-600 text-white' : 'border-neutral-300'">
                <CheckIcon v-if="mappingSwitchSelectedID === candidate.id" class="h-3.5 w-3.5" />
              </span>
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="mappingSwitchOpen = false">取消</Button>
        <Button :disabled="mappingSwitchSaving || !mappingSwitchSelectedID" @click="confirmMappingSwitch">确认切换</Button>
      </template>
    </Modal>

    <Modal v-model:open="mappingImportOpen" title="导入 SKU 映射" size="xl">
      <input ref="mappingImportInput" type="file" class="hidden" @change="onMappingImportFile">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div class="inline-flex rounded-md border border-neutral-300 bg-neutral-50 p-1" role="group" aria-label="导入方式">
          <button
            v-for="option in mappingImportModes"
            :key="option.value"
            type="button"
            class="h-8 rounded px-4 text-sm font-medium transition-colors"
            :class="mappingImportMode === option.value ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500 hover:text-neutral-800'"
            @click="mappingImportMode = option.value"
          >
            {{ option.label }}
          </button>
        </div>
        <Button variant="outline" size="sm" :disabled="mappingTemplateDownloading" @click="downloadMappingTemplate">
          <ArrowDownTrayIcon class="h-4 w-4" />
          下载模板
        </Button>
      </div>
      <button
        type="button"
        class="flex min-h-20 w-full items-center gap-4 rounded-md border border-dashed border-neutral-300 px-4 text-left hover:border-neutral-500"
        @click="mappingImportInput?.click()"
      >
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-neutral-100 text-neutral-600">
          <ArrowUpTrayIcon class="h-5 w-5" />
        </span>
        <span class="min-w-0">
          <span class="block truncate text-sm font-medium text-neutral-900">{{ mappingImportFileName || '选择映射表格' }}</span>
          <span class="mt-1 block text-xs text-neutral-500">需要前三个 SKU 列，“备注”为可选列</span>
        </span>
      </button>

      <div v-if="mappingImportError" class="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
        {{ mappingImportError }}
      </div>

      <div v-if="mappingImportRows.length" class="mt-5">
        <div class="mb-2 flex items-center justify-between">
          <span class="text-sm font-semibold text-neutral-900">导入预览</span>
          <span class="text-xs text-neutral-500">{{ mappingImportRows.length }} 条映射</span>
        </div>
        <div class="max-h-80 overflow-y-auto border-y border-neutral-200">
          <table class="min-w-full text-sm">
            <thead class="sticky top-0 bg-neutral-50 text-left text-xs font-medium text-neutral-500">
              <tr>
                <th class="w-20 px-4 py-3">行号</th>
                <th class="px-4 py-3">{{ platformLabel }} SKU</th>
                <th class="px-4 py-3">Amazon SKU</th>
                <th class="px-4 py-3">非亚 SKU</th>
                <th class="px-4 py-3">备注</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-100">
              <tr v-for="item in mappingImportRows.slice(0, 200)" :key="`${item.row_number}-${item.platform_sku}`">
                <td class="px-4 py-3 text-xs text-neutral-500">{{ item.row_number }}</td>
                <td class="px-4 py-3 font-mono text-xs font-semibold text-neutral-900">{{ item.platform_sku }}</td>
                <td class="px-4 py-3 font-mono text-xs text-neutral-700">{{ item.amazon_sku }}</td>
                <td class="px-4 py-3 font-mono text-xs text-neutral-700">{{ item.non_amazon_sku }}</td>
                <td class="px-4 py-3 text-xs text-neutral-600">{{ item.remark || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="mappingImportRows.length > 200" class="mt-2 text-xs text-neutral-500">预览前 200 条，导入时会处理全部数据。</p>
      </div>

      <template #footer>
        <Button variant="outline" @click="mappingImportOpen = false">取消</Button>
        <Button :disabled="mappingImportSaving || !mappingImportRows.length" @click="importMappings">
          {{ mappingImportMode === 'create' ? '新增' : '更新' }} {{ mappingImportRows.length || '' }} 条映射
        </Button>
      </template>
    </Modal>

    <Modal v-model:open="addressEditorOpen" title="编辑收件信息" size="lg">
      <div class="grid gap-4 sm:grid-cols-2">
        <label v-for="field in addressFields" :key="field.key" class="block" :class="field.wide ? 'sm:col-span-2' : ''">
          <span class="mb-1.5 block text-xs font-medium text-neutral-600">{{ field.label }}</span>
          <input v-model="addressDraft[field.key]" type="text" class="h-10 w-full rounded-md border border-neutral-300 px-3 text-sm outline-none focus:border-neutral-700 focus:ring-2 focus:ring-neutral-100">
        </label>
      </div>
      <template #footer>
        <Button variant="outline" @click="addressEditorOpen = false">取消</Button>
        <Button @click="saveAddressChanges">保存</Button>
      </template>
    </Modal>

    <Modal v-model:open="fieldSaveConfirmOpen" title="确认更新字段规则" size="md">
      <div class="flex gap-3 text-sm leading-6 text-neutral-700">
        <ExclamationTriangleIcon class="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
        <p>这会替换 {{ platformLabel }} 的整套字段匹配规则，并作用于之后所有订单导入。</p>
      </div>
      <template #footer>
        <Button variant="outline" @click="fieldSaveConfirmOpen = false">取消</Button>
        <Button :disabled="savingFields" @click="saveFieldRules">确认保存</Button>
      </template>
    </Modal>
  </MainLayout>
</template>

<script setup>
import { computed, defineComponent, h, onMounted, reactive, ref, watch } from 'vue'
import {
  AdjustmentsHorizontalIcon,
  ArrowDownTrayIcon,
  ArrowPathIcon,
  ArrowUpTrayIcon,
  ArrowsRightLeftIcon,
  CheckCircleIcon,
  CheckIcon,
  ClipboardDocumentListIcon,
  CloudArrowUpIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  LinkIcon,
  LockClosedIcon,
  MagnifyingGlassIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import request from '@/utils/request'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import {
  DEFAULT_FIELD_RULES,
  buildMCFPreview,
  buildOrderTrackingItems,
  createMCFBlob,
  exportableMCFRows,
  unmappedMCFNotifications,
  excludeGeneratedMCFRecords,
  normalizeFieldRules,
  readOrderFile,
  readSkuMappingFile,
  summarizeRecords,
  updateRecordAddress,
} from '@/utils/orderMcf'

const StatusBadge = defineComponent({
  props: { status: { type: String, required: true } },
  setup(props) {
    const labels = { ready: '可导出', warning: '警告', blocked: '阻断', excluded: '排除' }
    const classes = {
      ready: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
      warning: 'bg-amber-50 text-amber-700 ring-amber-200',
      blocked: 'bg-red-50 text-red-700 ring-red-200',
      excluded: 'bg-neutral-100 text-neutral-600 ring-neutral-200',
    }
    return () => h('span', { class: `inline-flex rounded px-2 py-1 text-xs font-medium ring-1 ring-inset ${classes[props.status] || classes.excluded}` }, labels[props.status] || props.status)
  },
})

const PaginationBar = defineComponent({
  props: { page: { type: Number, required: true }, totalPages: { type: Number, required: true } },
  emits: ['previous', 'next'],
  setup(props, { emit }) {
    return () => h('div', { class: 'flex items-center justify-between border-t border-neutral-200 px-4 py-3' }, [
      h('span', { class: 'text-xs text-neutral-500' }, `第 ${props.page} / ${props.totalPages} 页`),
      h('div', { class: 'flex gap-2' }, [
        h(Button, { variant: 'outline', size: 'sm', disabled: props.page <= 1, onClick: () => emit('previous') }, () => '上一页'),
        h(Button, { variant: 'outline', size: 'sm', disabled: props.page >= props.totalPages, onClick: () => emit('next') }, () => '下一页'),
      ]),
    ])
  },
})

const toast = useToast()
const platformOptions = [
  { value: 'TEMU', label: 'TEMU' },
  { value: 'TIKTOK', label: 'TikTok' },
  { value: 'LOWES', label: "Lowe's" },
]
const workspaceOptions = [
  { value: 'convert', label: '订单转换', icon: ArrowsRightLeftIcon },
  { value: 'mapping', label: 'SKU 映射', icon: LinkIcon },
  { value: 'fields', label: '字段规则', icon: AdjustmentsHorizontalIcon },
  { value: 'history', label: '历史记录', icon: ClipboardDocumentListIcon },
]
const fieldMetadata = {
  orderId: ['订单号', 'ORDER_ID'], status: ['订单子状态', 'STATUS'], quantity: ['履约数量', 'QUANTITY'],
  returnQuantity: ['退货数量', 'RETURN_QUANTITY'], fulfillmentType: ['配送类型', 'FULFILLMENT_TYPE'],
  productName: ['商品名称', 'PRODUCT_NAME'], sku: ['平台 SKU', 'SKU'], recipient: ['收件人', 'RECIPIENT'],
  recipientFirstName: ['收件人名字', 'RECIPIENT_FIRST_NAME'], recipientLastName: ['收件人姓氏', 'RECIPIENT_LAST_NAME'],
  phone: ['联系电话', 'PHONE'], email: ['通知邮箱', 'EMAIL'], address1: ['地址 1', 'ADDRESS_1'],
  address2: ['地址 2', 'ADDRESS_2'], address3: ['地址 3', 'ADDRESS_3'], district: ['区县', 'DISTRICT'],
  city: ['城市', 'CITY'], state: ['州 / 省', 'STATE'], postalCode: ['邮编', 'POSTAL_CODE'],
  country: ['国家', 'COUNTRY'], createdTime: ['订单创建时间', 'CREATED_TIME'],
}
const addressFields = [
  { key: 'recipient', label: '收件人' }, { key: 'phone', label: '电话' },
  { key: 'address1', label: '地址 1', wide: true }, { key: 'address2', label: '地址 2', wide: true },
  { key: 'address3', label: '地址 3', wide: true }, { key: 'city', label: '城市' },
  { key: 'state', label: '州代码' }, { key: 'postalCode', label: '邮编' },
  { key: 'country', label: '国家代码' }, { key: 'email', label: '通知邮箱', wide: true },
]

const authStore = useAuthStore()
const isOrderAdmin = (authStore.user?.roles || []).some((role) => role.name === 'admin')
const isOrderDataUser = ['data', '数据', '数据部'].includes(String(authStore.user?.platform || '').trim().toLowerCase())
  || ['data', '数据', '数据部'].includes(String(authStore.user?.department || '').trim().toLowerCase())
const assignedOrderPlatforms = (authStore.user?.order_platforms || []).map(normalizeBoundPlatform).filter(Boolean)
const availablePlatformOptions = computed(() => {
  if (isOrderAdmin || isOrderDataUser) return platformOptions
  const allowed = assignedOrderPlatforms.length ? assignedOrderPlatforms : [normalizeBoundPlatform(authStore.user?.platform)].filter(Boolean)
  return platformOptions.filter((option) => allowed.includes(option.value))
})
const platform = ref(availablePlatformOptions.value[0]?.value || 'TEMU')
const activeWorkspace = ref('convert')
const historyItems = ref([])
const historyPage = ref(1)
const historyTotal = ref(0)
const historyLoading = ref(false)
const historyError = ref('')
const historyDownloading = ref('')

async function loadHistory() {
  historyLoading.value = true
  historyError.value = ''
  const selectedPlatform = platform.value
  try {
    const response = await request.get('/order-conversions', { params: { platform: selectedPlatform, page: historyPage.value } })
    if (selectedPlatform !== platform.value) return
    historyItems.value = response.data.items || []
    historyTotal.value = response.data.total || 0
  } catch (error) {
    historyError.value = error.response?.data?.error || '历史记录加载失败'
  } finally {
    historyLoading.value = false
  }
}

function changeHistoryPage(value) {
  historyPage.value = value
  loadHistory()
}

async function downloadHistoryFile(record, type) {
  if (historyDownloading.value) return
  historyDownloading.value = `${record.id}-${type}`
  try {
    const response = await request.get(`/order-conversions/${record.id}/${type}`, { responseType: 'blob' })
    const url = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = url
    link.download = type === 'source' ? record.source_name : record.mcf_name
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch {
    toast.error('文件下载失败，请重试')
  } finally {
    historyDownloading.value = ''
  }
}
const fileInput = ref(null)
const fileName = ref('')
const fileSize = ref(0)
const pendingFile = ref(null)
const fieldIssue = ref(null)
const records = ref([])
const mappings = reactive({})
const parsing = ref(false)
const exporting = ref(false)
const dragging = ref(false)
const statusFilter = ref('all')
const currentPage = ref(1)
const pageSize = 50

const mappingList = ref([])
const mappingTotal = ref(0)
const mappingPage = ref(1)
const mappingPageSize = 50
const mappingQuery = ref('')
const mappingLoading = ref(false)
const mappingDuplicatesOnly = ref(false)
const duplicateMappingCount = ref(0)
const mappingListCanEdit = ref(false)
const mappingEditorOpen = ref(false)
const savingMapping = ref(false)
const mappingSwitchOpen = ref(false)
const mappingSwitchTarget = ref(null)
const mappingSwitchCandidates = ref([])
const mappingSwitchSelectedID = ref(0)
const mappingSwitchSaving = ref(false)
const mappingForm = reactive({ id: 0, platformSku: '', amazonSku: '', nonAmazonSku: '', remark: '', isCurrent: false, relatedAmazonSkus: [] })
const mappingImportOpen = ref(false)
const mappingImportInput = ref(null)
const mappingImportFileName = ref('')
const mappingImportRows = ref([])
const mappingImportError = ref('')
const mappingImportSaving = ref(false)
const mappingTemplateDownloading = ref(false)
const mappingImportMode = ref('create')
const mappingImportModes = [
  { value: 'create', label: '新增' },
  { value: 'update', label: '更新' },
]

const skuSearchResults = ref([])
const skuSearching = ref(false)
let skuSearchTimer = null

const fieldRules = ref(normalizeFieldRules(DEFAULT_FIELD_RULES.TEMU))
const fieldDrafts = reactive({})
const fieldLoading = ref(false)
const canEditFields = ref(false)
const editingFields = ref(false)
const savingFields = ref(false)
const fieldSaveConfirmOpen = ref(false)
const pendingFieldItems = ref([])

const addressEditorOpen = ref(false)
const editingRowNumbers = ref([])
const addressDraft = reactive({})

const platformLabel = computed(() => ({ TEMU: 'TEMU', TIKTOK: 'TikTok', LOWES: "Lowe's" }[platform.value] || platform.value))
const fieldIssueMissingLabels = computed(() => (fieldIssue.value?.missingFields || []).map((key) => fieldMeta(key).label))
const previewRows = computed(() => buildMCFPreview(records.value, mappings))
const summary = computed(() => summarizeRecords(records.value, previewRows.value))
const skuRows = computed(() => {
  const grouped = new Map()
  for (const record of records.value) {
    if (record.excludedReason || !record.sourceSku) continue
    if (!grouped.has(record.sourceSku)) grouped.set(record.sourceSku, { sourceSku: record.sourceSku, productName: record.productName, rows: 0, quantity: 0 })
    const item = grouped.get(record.sourceSku)
    item.rows += 1
    if (Number.isInteger(record.quantity)) item.quantity += record.quantity
  }
  return [...grouped.values()].sort((a, b) => a.sourceSku.localeCompare(b.sourceSku))
})
const unmappedSkuRows = computed(() => skuRows.value.filter((item) => !mappings[item.sourceSku]))
const mappedSkuCount = computed(() => skuRows.value.length - unmappedSkuRows.value.length)
const mappingTotalPages = computed(() => Math.max(1, Math.ceil(mappingTotal.value / mappingPageSize)))

const excludedRows = computed(() => records.value.filter((record) => record.excludedReason).map((record) => ({
  id: `excluded-${record.id}`, status: 'excluded', orderId: record.orderId || record.sourceOrderId || '-',
  sourceSkus: [record.sourceSku || '-'], amazonSku: mappings[record.sourceSku]?.amazon_sku || '',
  quantity: Number.isInteger(record.quantity) ? record.quantity : '-', recipient: record.recipient,
  address1: record.address1, address2: record.address2, address3: record.address3, city: record.city,
  state: record.state, postalCode: record.postalCode, country: record.country, sourceRows: [record.rowNumber],
  errors: [record.excludedReason], warnings: [],
})))
const allDisplayRows = computed(() => [...previewRows.value, ...excludedRows.value])
const filteredDisplayRows = computed(() => statusFilter.value === 'all' ? allDisplayRows.value : allDisplayRows.value.filter((row) => row.status === statusFilter.value))
const totalPages = computed(() => Math.max(1, Math.ceil(filteredDisplayRows.value.length / pageSize)))
const pagedDisplayRows = computed(() => filteredDisplayRows.value.slice((currentPage.value - 1) * pageSize, currentPage.value * pageSize))
const filterOptions = computed(() => [
  { value: 'all', label: '全部', count: allDisplayRows.value.length },
  { value: 'blocked', label: '阻断', count: previewRows.value.filter((row) => row.status === 'blocked').length },
  { value: 'warning', label: '警告', count: previewRows.value.filter((row) => row.status === 'warning').length },
  { value: 'ready', label: '可导出', count: previewRows.value.filter((row) => row.status === 'ready').length },
  { value: 'excluded', label: '排除', count: excludedRows.value.length },
])
const metrics = computed(() => [
  { label: '源明细', value: summary.value.sourceRows }, { label: '参与转换', value: summary.value.activeRows },
  { label: 'SKU 映射', value: `${mappedSkuCount.value}/${skuRows.value.length}` }, { label: '输出明细', value: summary.value.outputRows },
  { label: '商品件数', value: summary.value.quantity }, { label: '阻断 / 排除', value: `${summary.value.blockedRows} / ${summary.value.excludedRows}` },
])
const exportRows = computed(() => exportableMCFRows(previewRows.value))
const canExport = computed(() => exportRows.value.length > 0 && !parsing.value && !exporting.value)
const exportDisabledReason = computed(() => {
  if (exporting.value) return 'MCF 文件生成中，请稍候'
  if (!records.value.length) return '请先导入订单文件'
  if (!exportRows.value.length) return '没有可导出的订单，请先完成 SKU 映射或处理阻断问题'
  return ''
})

onMounted(() => loadFieldRules())

async function setPlatform(value) {
  if (exporting.value) return
  if (platform.value === value) return
  platform.value = value
  clearFile()
  cancelFieldEditing()
  fieldRules.value = normalizeFieldRules(DEFAULT_FIELD_RULES[value])
  await loadFieldRules()
  if (activeWorkspace.value === 'mapping') await loadMappingList()
  historyPage.value = 1
  if (activeWorkspace.value === 'history') await loadHistory()
}

async function setWorkspace(value) {
  activeWorkspace.value = value
  if (value === 'mapping') await loadMappingList()
  if (value === 'history') await loadHistory()
}

function setFilter(value) {
  statusFilter.value = value
  currentPage.value = 1
}

async function onFileChange(event) {
  const file = event.target.files?.[0]
  if (file) await processFile(file)
  event.target.value = ''
}

async function onDrop(event) {
  dragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) await processFile(file)
}

async function processFile(file) {
  if (exporting.value) return
  if (file.size > 20 * 1024 * 1024) {
    toast.error('原始订单文件不能超过 20 MB')
    return
  }
  parsing.value = true
  pendingFile.value = file
  try {
    const parsed = await readOrderFile(file, platform.value, fieldRules.value)
    records.value = await excludePreviouslyGeneratedRecords(parsed)
    await notifyExcludedOrders(records.value)
    fieldIssue.value = null
    fileName.value = file.name
    fileSize.value = file.size
    statusFilter.value = 'all'
    currentPage.value = 1
    await resolveMappings()
    toast.success(`已读取 ${parsed.length} 条订单明细`)
  } catch (error) {
    if (error?.code === 'FIELD_MAPPING_MISMATCH') {
      records.value = []
      for (const key of Object.keys(mappings)) delete mappings[key]
      fileName.value = file.name
      fileSize.value = file.size
      fieldIssue.value = {
        missingFields: error.missingFields || [],
        detectedHeaders: error.detectedHeaders || [],
      }
      toast.error('字段匹配失败，已阻止生成 MCF')
    } else {
      clearFile()
      toast.error(error?.message || '订单文件解析失败')
    }
  } finally {
    parsing.value = false
  }
}

async function notifyExcludedOrders(orderRecords) {
  const items = orderRecords
    .filter((record) => record.excludedReason && (record.orderId || record.sourceOrderId))
    .map((record) => ({ order_id: record.orderId || record.sourceOrderId, reason: record.excludedReason }))
  if (!items.length) return
  try {
    await request.post('/order-tracking/mcf-excluded/notify', { platform: platform.value, items })
  } catch {
    toast.warning('异常订单通知发送失败')
  }
}

async function resolveMappings() {
  for (const key of Object.keys(mappings)) delete mappings[key]
  // 排除的历史订单仍需要显示其映射，避免“已生成”提示和“未映射”同时出现。
  const skus = [...new Set(records.value.map((record) => record.sourceSku).filter(Boolean))]
  const response = await request.post('/order-sku-mappings/resolve', { platform: platform.value, skus })
  for (const item of response.data.items || []) mappings[item.platform_sku] = item
}

async function excludePreviouslyGeneratedRecords(parsed) {
  const items = parsed
	.filter((record) => !record.excludedReason && record.sourceOrderId && record.sourceSku)
	.map((record) => ({ order_id: record.sourceOrderId, sku: record.sourceSku }))
  if (!items.length) return parsed
  const response = await request.post('/order-tracking/generated/check', { platform: platform.value, items })
  return excludeGeneratedMCFRecords(parsed, response.data.items || [])
}

async function archiveExportedOrders(exportedRows, blob, outputName) {
  const items = buildOrderTrackingItems(exportedRows)
  if (!items.length || !pendingFile.value) throw new Error('请先上传订单文件')
  try {
    const form = new FormData()
    form.append('payload', JSON.stringify({ platform: platform.value, items }))
    form.append('source_file', pendingFile.value, pendingFile.value.name)
    form.append('mcf_file', blob, outputName)
    await request.post('/order-tracking/import', form)
  } catch (error) {
    const backendMessage = error.response?.data?.error
    const message = error.response?.status === 403
      ? `当前账号无权维护 ${platformLabel.value} 数据，本次订单未进入汇总`
      : backendMessage
        ? `写入订单汇总失败：${backendMessage}`
        : '写入订单汇总失败，请确认后端已启动后重试'
    throw new Error(message)
  }
}

async function loadMappingList() {
  mappingLoading.value = true
  try {
    const response = await request.get('/order-sku-mappings', { params: { platform: platform.value, q: mappingQuery.value.trim(), duplicates_only: mappingDuplicatesOnly.value || undefined, page: mappingPage.value, page_size: mappingPageSize } })
    mappingList.value = response.data.items || []
    mappingTotal.value = Number(response.data.total) || 0
    mappingListCanEdit.value = Boolean(response.data.can_edit)
    duplicateMappingCount.value = Number(response.data.duplicate_groups) || 0
  } catch (error) {
    mappingList.value = []
    mappingTotal.value = 0
    toast.error(error.response?.data?.error || 'SKU 映射加载失败')
  } finally {
    mappingLoading.value = false
  }
}

function searchMappings() {
  mappingPage.value = 1
  loadMappingList()
}

function toggleDuplicateMappings() {
  mappingDuplicatesOnly.value = !mappingDuplicatesOnly.value
  mappingPage.value = 1
  loadMappingList()
}

function changeMappingPage(page) {
  mappingPage.value = page
  loadMappingList()
}

function openMappingEditor(item = null) {
  window.clearTimeout(skuSearchTimer)
  mappingForm.id = item?.id || 0
  mappingForm.platformSku = item?.platform_sku || unmappedSkuRows.value[0]?.sourceSku || ''
  mappingForm.amazonSku = item?.amazon_sku || ''
  mappingForm.nonAmazonSku = item?.non_amazon_sku || ''
  mappingForm.remark = item?.remark || ''
  mappingForm.isCurrent = Boolean(item?.is_current)
  mappingForm.relatedAmazonSkus = [...(item?.related_amazon_skus || [])]
  skuSearchResults.value = []
  mappingEditorOpen.value = true
  if (mappingForm.amazonSku) searchProductSkus()
  if (mappingForm.amazonSku) loadMappingGroup(mappingForm.amazonSku)
}

function openMappingImport() {
  mappingImportFileName.value = ''
  mappingImportRows.value = []
  mappingImportError.value = ''
  mappingImportMode.value = 'create'
  mappingImportOpen.value = true
}

async function downloadMappingTemplate() {
  mappingTemplateDownloading.value = true
  try {
    const ExcelJSModule = await import('exceljs')
    const Workbook = ExcelJSModule.Workbook || ExcelJSModule.default?.Workbook
    if (!Workbook) throw new Error('Excel 模板组件加载失败')

    const workbook = new Workbook()
    const worksheet = workbook.addWorksheet('SKU 映射')
    worksheet.columns = [
      { header: '平台 SKU', key: 'platform_sku', width: 32 },
      { header: 'Amazon SKU', key: 'amazon_sku', width: 32 },
      { header: '非亚 SKU', key: 'non_amazon_sku', width: 32 },
      { header: '备注', key: 'remark', width: 40 },
    ]
    worksheet.views = [{ state: 'frozen', ySplit: 1 }]
    worksheet.getRow(1).eachCell((cell) => {
      cell.font = { bold: true, color: { argb: 'FFFFFFFF' } }
      cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF262626' } }
      cell.alignment = { vertical: 'middle' }
    })
    worksheet.getRow(1).height = 24

    const buffer = await workbook.xlsx.writeBuffer()
    const blob = new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${platformLabel.value}-SKU映射导入模板.xlsx`
    document.body.appendChild(link)
    link.click()
    window.setTimeout(() => {
      URL.revokeObjectURL(url)
      link.remove()
    }, 0)
  } catch (error) {
    toast.error(error?.message || '模板下载失败')
  } finally {
    mappingTemplateDownloading.value = false
  }
}

async function onMappingImportFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  mappingImportFileName.value = file.name
  mappingImportRows.value = []
  mappingImportError.value = ''
  try {
    mappingImportRows.value = await readSkuMappingFile(file)
  } catch (error) {
    mappingImportError.value = error?.message || '映射表格解析失败'
  }
}

async function importMappings() {
  mappingImportSaving.value = true
  mappingImportError.value = ''
  try {
    const response = await request.post('/order-sku-mappings/import', {
      platform: platform.value,
      mode: mappingImportMode.value,
      items: mappingImportRows.value.map((item) => ({
        platform_sku: item.platform_sku,
        amazon_sku: item.amazon_sku,
        non_amazon_sku: item.non_amazon_sku,
        remark: item.remark,
      })),
    })
    mappingImportOpen.value = false
    await loadMappingList()
    if (records.value.length) await resolveMappings()
    const action = mappingImportMode.value === 'create' ? '新增' : '更新'
    const missing = Array.isArray(response.data.missing_skus) ? response.data.missing_skus : []
    const existingPlatform = Array.isArray(response.data.existing_platform_skus) ? response.data.existing_platform_skus : []
    const missingPlatform = Array.isArray(response.data.missing_platform_skus) ? response.data.missing_platform_skus : []
    const skipped = Number(response.data.skipped) || 0
    if (skipped > 0) {
      const reasons = []
      if (missing.length) reasons.push(`产品库不存在的 Amazon SKU：${formatImportSkuList(missing)}`)
      if (existingPlatform.length) reasons.push(`已存在的平台 SKU：${formatImportSkuList(existingPlatform)}`)
      if (missingPlatform.length) reasons.push(`尚不存在的平台 SKU：${formatImportSkuList(missingPlatform)}`)
      toast.warning(`已${action} ${response.data.imported} 条，跳过 ${skipped} 条；${reasons.join('；')}`)
    } else {
      toast.success(`已${action} ${response.data.imported} 条 SKU 映射`)
    }
  } catch (error) {
    const missing = error.response?.data?.missing_skus
    const existingPlatform = error.response?.data?.existing_platform_skus
    const missingPlatform = error.response?.data?.missing_platform_skus
    if (Array.isArray(missing) && missing.length) {
      mappingImportError.value = `以下 Amazon SKU 不存在：${formatImportSkuList(missing)}`
    } else if (Array.isArray(existingPlatform) && existingPlatform.length) {
      mappingImportError.value = `以下平台 SKU 已存在，不能新增：${formatImportSkuList(existingPlatform)}`
    } else if (Array.isArray(missingPlatform) && missingPlatform.length) {
      mappingImportError.value = `以下平台 SKU 尚不存在，不能更新：${formatImportSkuList(missingPlatform)}`
    } else {
      mappingImportError.value = error.response?.data?.error || 'SKU 映射导入失败'
    }
  } finally {
    mappingImportSaving.value = false
  }
}

function formatImportSkuList(items) {
  const visible = items.slice(0, 20)
  return `${visible.join('、')}${items.length > visible.length ? ` 等 ${items.length} 个` : ''}`
}

async function saveMapping() {
  savingMapping.value = true
  try {
    const response = await request.put('/order-sku-mappings', {
      platform: platform.value,
      platform_sku: mappingForm.platformSku.trim(),
      amazon_sku: mappingForm.amazonSku.trim(),
      non_amazon_sku: mappingForm.nonAmazonSku.trim(),
      remark: mappingForm.remark.trim(),
      is_current: mappingForm.isCurrent,
    })
    mappings[response.data.platform_sku] = response.data
    mappingEditorOpen.value = false
    await loadMappingList()
    toast.success('SKU 映射已保存')
  } catch (error) {
    toast.error(error.response?.data?.error || 'SKU 映射保存失败')
  } finally {
    savingMapping.value = false
  }
}

async function setCurrentMapping(item) {
  if (!mappingListCanEdit.value) return
  savingMapping.value = true
  try {
    await request.put('/order-sku-mappings', {
      platform: platform.value,
      platform_sku: item.platform_sku,
      amazon_sku: item.amazon_sku,
      non_amazon_sku: item.non_amazon_sku,
      remark: item.remark || '',
      is_current: true,
    })
    await loadMappingList()
    if (records.value.length) await resolveMappings()
    toast.success(`${item.platform_sku} 已设为当前平台 SKU`)
  } catch (error) {
    toast.error(error.response?.data?.error || '当前平台 SKU 设置失败')
  } finally {
    savingMapping.value = false
  }
}

async function openMappingSwitch(item) {
  mappingSwitchTarget.value = item
  mappingSwitchCandidates.value = []
  mappingSwitchSelectedID.value = item.id
  mappingSwitchOpen.value = true
  try {
    const response = await request.get('/order-sku-mappings/group', { params: { platform: platform.value, platform_sku: item.platform_sku } })
    mappingSwitchCandidates.value = response.data.items || []
    mappingSwitchSelectedID.value = mappingSwitchCandidates.value.find((candidate) => candidate.is_current)?.id || item.id
  } catch (error) {
    mappingSwitchOpen.value = false
    toast.error(error.response?.data?.error || '平台 SKU 候选加载失败')
  }
}

async function confirmMappingSwitch() {
  const candidate = mappingSwitchCandidates.value.find((item) => item.id === mappingSwitchSelectedID.value)
  if (!candidate) return
  mappingSwitchSaving.value = true
  try {
    await setCurrentMapping(candidate)
    mappingSwitchOpen.value = false
  } finally {
    mappingSwitchSaving.value = false
  }
}

async function deactivateMapping(item) {
  if (!window.confirm(`停用 ${item.platform_sku} 的 SKU 映射？`)) return
  try {
    await request.delete(`/order-sku-mappings/${item.id}`)
    delete mappings[item.platform_sku]
    await loadMappingList()
    toast.success('SKU 映射已停用')
  } catch (error) {
    toast.error(error.response?.data?.error || 'SKU 映射停用失败')
  }
}

function scheduleSkuSearch() {
  window.clearTimeout(skuSearchTimer)
  skuSearchResults.value = []
  if (!mappingForm.amazonSku.trim()) {
    skuSearching.value = false
    mappingForm.relatedAmazonSkus = []
    mappingForm.isCurrent = false
    return
  }
  skuSearching.value = true
  skuSearchTimer = window.setTimeout(searchProductSkus, 200)
  window.setTimeout(() => loadMappingGroup(mappingForm.amazonSku.trim()), 250)
}

async function searchProductSkus() {
  const query = mappingForm.amazonSku.trim()
  if (!query) {
    skuSearching.value = false
    return
  }
  skuSearching.value = true
  try {
    const response = await request.get('/product-skus', { params: { q: query } })
    if (mappingForm.amazonSku.trim() === query) {
      skuSearchResults.value = Array.isArray(response.data) ? response.data : []
    }
  } catch {
    if (mappingForm.amazonSku.trim() === query) {
      skuSearchResults.value = []
      toast.error('产品 SKU 搜索失败')
    }
  } finally {
    if (mappingForm.amazonSku.trim() === query) skuSearching.value = false
  }
}

async function selectProductSku(item) {
  mappingForm.amazonSku = item.sku
  skuSearchResults.value = [item]
  await loadMappingGroup(item.sku)
}

async function loadMappingGroup(amazonSku) {
  try {
    const response = await request.get('/order-sku-mappings/group', { params: { platform: platform.value, platform_sku: mappingForm.platformSku.trim() } })
    const related = response.data.items || []
    const amazonSkus = [...new Set([...related.map((item) => item.amazon_sku), mappingForm.amazonSku.trim()].filter(Boolean))].sort()
    mappingForm.relatedAmazonSkus = amazonSkus
    const current = related.find((item) => item.is_current)
    mappingForm.isCurrent = current ? current.amazon_sku === mappingForm.amazonSku.trim() : amazonSkus.length <= 1
  } catch {
    mappingForm.relatedAmazonSkus = []
  }
}

watch(() => mappingForm.amazonSku, (value) => {
  if (!mappingEditorOpen.value || !value.trim()) return
  window.setTimeout(() => loadMappingGroup(value.trim()), 250)
})

async function loadFieldRules() {
  fieldLoading.value = true
  try {
    const response = await request.get(`/order-field-mappings/${platform.value}`)
    fieldRules.value = normalizeFieldRules(response.data.items)
    canEditFields.value = Boolean(response.data.can_edit)
  } catch (error) {
    canEditFields.value = false
    toast.error(error.response?.data?.error || '字段规则加载失败')
  } finally {
    fieldLoading.value = false
  }
}

function startFieldEditing() {
  for (const item of fieldRules.value) fieldDrafts[item.field_key] = item.source_headers.join('\n')
  editingFields.value = true
}

function openFieldRepair() {
  activeWorkspace.value = 'fields'
  if (canEditFields.value && !editingFields.value) startFieldEditing()
}

function cancelFieldEditing() {
  editingFields.value = false
  for (const key of Object.keys(fieldDrafts)) delete fieldDrafts[key]
}

function prepareFieldSave() {
  const items = fieldRules.value.map((item) => ({
    field_key: item.field_key,
    source_headers: splitHeaders(fieldDrafts[item.field_key]),
  }))
  const missing = items.find((item) => item.required && !item.source_headers.length)
  if (missing) {
    toast.error(`${fieldMeta(missing.field_key).label} 至少需要一个源表头`)
    return
  }
  pendingFieldItems.value = items
  fieldSaveConfirmOpen.value = true
}

async function saveFieldRules() {
  savingFields.value = true
  try {
    const response = await request.put(`/order-field-mappings/${platform.value}`, { items: pendingFieldItems.value })
    fieldRules.value = normalizeFieldRules(response.data.items)
    fieldSaveConfirmOpen.value = false
    cancelFieldEditing()
    const fileToRetry = pendingFile.value
    if (fileToRetry) {
      activeWorkspace.value = 'convert'
      await processFile(fileToRetry)
      if (!fieldIssue.value) toast.success('字段规则已更新，当前文件已重新校验')
    } else {
      toast.success('字段规则已更新')
    }
  } catch (error) {
    toast.error(error.response?.data?.error || '字段规则保存失败')
  } finally {
    savingFields.value = false
  }
}

function fieldMeta(key) {
  const [label = key, code = key] = fieldMetadata[key] || []
  return { label, code }
}

function isMissingField(key) {
  return Boolean(fieldIssue.value?.missingFields.includes(key))
}

function appendDetectedHeader(key, event) {
  const header = event.target.value
  if (!header) return
  const headers = splitHeaders(fieldDrafts[key])
  if (!headers.includes(header)) headers.push(header)
  fieldDrafts[key] = headers.join('\n')
  event.target.value = ''
}

function splitHeaders(value) {
  return [...new Set(String(value || '').split(/\r?\n/).map((item) => item.trim()).filter(Boolean))]
}

function openAddressEditor(row) {
  const source = records.value.find((record) => row.sourceRows.includes(record.rowNumber))
  if (!source) return
  editingRowNumbers.value = [...row.sourceRows]
  for (const field of addressFields) addressDraft[field.key] = source[field.key] || ''
  addressEditorOpen.value = true
}

function saveAddressChanges() {
  records.value = records.value.map((record) => editingRowNumbers.value.includes(record.rowNumber) ? updateRecordAddress(record, addressDraft) : record)
  addressEditorOpen.value = false
  toast.success('收件信息已更新')
}

async function downloadMCF() {
  if (!canExport.value) return
  exporting.value = true
  try {
    const rows = exportRows.value
    const unmappedOrders = unmappedMCFNotifications(previewRows.value)
    const blob = createMCFBlob(rows)
    const link = document.createElement('a')
    const now = new Date()
    const stamp = `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}_${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
    const platformCode = { TEMU: 'TEMU', TIKTOK: 'TK', LOWES: 'LOWES' }[platform.value] || platform.value
    link.download = `MCF_${platformCode}_${stamp}.txt`
    // 原文件、MCF 和订单汇总保存成功后，再交付文件。
    await archiveExportedOrders(rows, blob, link.download)
    await notifyExcludedOrders(unmappedOrders)
    if (platform.value === 'LOWES') {
      const form = new FormData()
      form.append('platform', platform.value)
      form.append('file', blob, link.download)
      // 让浏览器/Axios 自动补 multipart boundary，手动设置 Header 会导致后端无法解析文件。
      await request.post('/order-tracking/mcf-email', form)
    } else {
      const url = URL.createObjectURL(blob)
      link.href = url
      document.body.appendChild(link)
      link.click()
      link.remove()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    }
    clearFile()
    toast.success(platform.value === 'LOWES' ? 'MCF 文件已生成并发送邮件' : 'MCF 文件已生成')
  } catch (error) {
    toast.error(error.response?.data?.error || error.message || 'MCF 文件生成失败')
  } finally {
    exporting.value = false
  }
}

function clearFile() {
  records.value = []
  fileName.value = ''
  fileSize.value = 0
  pendingFile.value = null
  if (fileInput.value) fileInput.value.value = ''
  fieldIssue.value = null
  statusFilter.value = 'all'
  currentPage.value = 1
  for (const key of Object.keys(mappings)) delete mappings[key]
}

function formatAddress(row) {
  return [row.address1, row.address2, row.address3, row.city, row.state, row.postalCode, row.country].filter(Boolean).join(', ') || '-'
}

function formatFileSize(bytes) {
  return bytes < 1024 * 1024 ? `${Math.max(1, Math.round(bytes / 1024))} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatTimestamp(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function updaterLabel(item) {
  if (item?.updated_by_name) return item.updated_by_name
  if (item?.updated_by) return `用户 #${item.updated_by}`
  return '-'
}

function normalizeBoundPlatform(value) {
  const normalized = String(value || '').trim().toUpperCase()
  if (normalized === 'TEMU') return 'TEMU'
  if (['TIKTOK', 'TIKTOK SHOP', 'TK'].includes(normalized)) return 'TIKTOK'
  if (['LOWES', "LOWE'S", 'LOWE’S', 'LOWE'].includes(normalized)) return 'LOWES'
  return ''
}

function pad2(value) {
  return String(value).padStart(2, '0')
}
</script>
