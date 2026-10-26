<template>
  <MainLayout>
    <div class="catalog-page">
      <header class="page-header">
        <div class="page-heading">
          <h1>产品库</h1>
          <p>集中管理商品主数据与平台关联</p>
        </div>
        <div class="header-actions">
          <a class="template-link" href="/产品库导入模板.xlsx" download><DocumentArrowDownIcon class="h-4 w-4" />下载导入模板</a>
          <Button variant="outline" :disabled="importing" @click="importInput?.click()"><ArrowUpTrayIcon class="h-4 w-4" />{{ importing ? '导入中' : '导入产品库' }}</Button>
          <input ref="importInput" class="sr-only" type="file" accept=".xlsx" @change="importCatalog" />
          <Button @click="openSpuEditor()"><PlusIcon class="h-4 w-4" />新建 SPU</Button>
        </div>
      </header>

      <section class="catalog-surface">
        <div class="catalog-controls">
          <label class="relative block toolbar-search">
            <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
            <input v-model.trim="keyword" class="field toolbar-search-input" placeholder="搜索 SPU、SKU、ASIN、品名、品牌或型号" @keyup.enter="search" />
          </label>
          <div class="catalog-control-actions">
            <button v-if="isAdmin" class="batch-delete-button" type="button" :disabled="!selectedCount" @click="openBatchDeleteDialog">
              <TrashIcon class="h-4 w-4" />批量删除{{ selectedCount ? ` (${selectedCount})` : '' }}
            </button>
            <button class="control-icon-button" type="button" title="刷新产品库" aria-label="刷新产品库" @click="search">
              <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>

        <div class="summary-strip">
          <div v-for="item in summary" :key="item.label" class="summary-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>

        <div class="catalog-list">
          <div class="list-head">
            <label v-if="isAdmin" class="selection-cell" title="全选当前页"><input type="checkbox" :checked="allCurrentPageSelected" :indeterminate="someCurrentPageSelected" :disabled="!items.length" aria-label="全选当前页 SPU" @change="toggleCurrentPage($event.target.checked)" /></label>
            <div v-else></div>
            <div></div><div>产品</div><div>概览</div><div>平台关联</div><div>最后更新</div><div>操作</div>
          </div>

          <div v-if="loading" class="loading-row">
            <ArrowPathIcon class="mr-2 h-4 w-4 animate-spin" />正在加载产品库
          </div>
          <div v-else-if="!items.length" class="empty-row">
            <CubeIcon class="mb-3 h-8 w-8 text-neutral-300" />暂无产品
          </div>
          <template v-else>
            <article v-for="spu in items" :key="spu.id" class="spu-group">
              <div class="spu-list-row" :class="{ expanded: expandedId === spu.id }">
                <label v-if="isAdmin" class="selection-cell"><input type="checkbox" :checked="selectedSpuIds.has(spu.id)" :aria-label="`选择 SPU ${spu.spu}`" @change="toggleSpuSelection(spu.id, $event.target.checked)" /></label>
                <div v-else></div>
                <button class="expand-button" :class="{ open: expandedId === spu.id }" :aria-label="expandedId === spu.id ? '收起 SKU' : '展开 SKU'" @click="toggleSpu(spu.id)">
                  <ChevronRightIcon class="h-4 w-4 transition-transform" :class="{ 'rotate-90': expandedId === spu.id }" />
                </button>
                <button class="spu-code" @click="toggleSpu(spu.id)">
                  <strong>{{ spu.spu }}</strong>
                  <span>{{ spu.name || '未填写品名' }}</span>
                </button>
                <div class="cell-stack spu-overview"><strong>{{ spu.brand || '未填写品牌' }}</strong><span>{{ spu.model || '未填写型号' }} · {{ spu.sku_count || spu.skus?.length || 0 }} 个 SKU</span></div>
                <div class="platform-summary-cell"><span v-for="entry in spuPlatformSummary(spu)" :key="entry.platform" class="platform-summary-item"><b>{{ entry.platform }}</b>{{ entry.count }} 个</span><span v-if="!spuPlatformSummary(spu).length" class="muted-copy">未关联平台 SKU</span></div>
                <div class="audit-cell" :title="auditTitle(spu)">
                  <span class="audit-copy"><strong>{{ updaterLabel(spu) }}</strong><small>{{ updatedAtLabel(spu) }}</small></span>
                </div>
                <div class="row-actions">
                  <button class="row-button" title="查看产品详情" @click="openSpuEditor(spu)"><PencilSquareIcon class="h-3.5 w-3.5" />详情</button>
                  <button class="row-button row-button-primary" @click="openSkuEditor(spu)"><PlusIcon class="h-3.5 w-3.5" />新增 SKU</button>
                  <button v-if="isAdmin" class="icon-button danger" aria-label="删除 SPU" title="删除 SPU" @click="removeSpu(spu)"><TrashIcon class="h-4 w-4" /></button>
                </div>
              </div>

              <template v-if="expandedId === spu.id">
                <div v-if="!filteredSkus(spu).length" class="sku-empty-row">暂无 SKU</div>
                <div v-for="sku in filteredSkus(spu)" :key="sku.id" class="sku-list-row">
                  <div></div>
                  <div class="sku-marker">SKU</div>
                  <div class="product-identity"><ProductThumbnail :src="sku.image_url" :alt="sku.sku" /><div class="cell-stack"><strong>{{ sku.sku || '-' }}</strong><span>ASIN {{ sku.asin || '-' }}{{ sku.name ? ` · ${sku.name}` : '' }}</span></div></div>
                  <div class="cell-stack sku-overview"><strong>{{ sku.name || sku.style_name || '未填写名称' }}</strong><span>{{ [sku.color, sku.size].filter(Boolean).join(' / ') || '未填写颜色和尺寸' }}</span></div>
                  <div class="platform-sku-cell">
                    <span v-for="entry in platformSkuSummary(sku)" :key="entry.platform + entry.sku" class="platform-sku-tag"><b>{{ entry.platform }}</b>{{ entry.sku }}</span>
                    <span v-if="!platformSkuSummary(sku).length" class="sku-count sku-count-muted">未关联</span>
                  </div>
                  <div class="audit-cell" :title="auditTitle(sku)">
                    <span class="audit-copy"><strong>{{ updaterLabel(sku) }}</strong><small>{{ updatedAtLabel(sku) }}</small></span>
                  </div>
                  <div class="row-actions">
                    <button class="row-button" title="查看 SKU 详情" @click="openSkuDetail(spu, sku)"><DocumentTextIcon class="h-3.5 w-3.5" />详情</button>
                    <button class="icon-button" aria-label="编辑 SKU" title="编辑 SKU" @click="openSkuEditor(spu, sku)"><PencilSquareIcon class="h-4 w-4" /></button>
                    <button v-if="isAdmin" class="icon-button danger" aria-label="删除 SKU" title="删除 SKU" @click="removeSku(sku)"><TrashIcon class="h-4 w-4" /></button>
                  </div>
                </div>
              </template>
            </article>
          </template>
        </div>

        <footer class="pager">
          <div class="page-size-control">
            <span>共 {{ total }} 条</span>
            <label>每页
              <select v-model.number="pageSize" class="page-size-select" @change="changePageSize">
                <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }} 条</option>
              </select>
            </label>
          </div>
          <div class="flex items-center gap-2">
            <Button size="sm" variant="outline" :disabled="page === 1" @click="changePage(page - 1)">上一页</Button>
            <span class="px-2 tabular-nums">{{ page }} / {{ pageCount }}</span>
            <Button size="sm" variant="outline" :disabled="page === pageCount" @click="changePage(page + 1)">下一页</Button>
          </div>
        </footer>
      </section>
    </div>

    <Modal
      v-model:open="spuEditorOpen"
      :title="spuForm.id ? '产品详情' : '新建 SPU'"
      size="xl"
      :dismissible="!attachmentEditorOpen && !(deleteDialog.open && deleteDialog.type === 'attachment')"
    >
      <form class="editor-form" @submit.prevent="saveSpu">
        <section class="editor-section">
          <div class="editor-section-head"><h2>基础信息</h2></div>
          <div class="form-grid">
            <label class="form-label"><span class="form-label-text">SPU<span class="required">*</span></span><input v-model.trim="spuForm.spu" class="field" :class="{ 'bg-neutral-100 text-neutral-500': spuForm.id }" :disabled="Boolean(spuForm.id)" required /></label>
            <label class="form-label"><span class="form-label-text">品名</span><input v-model.trim="spuForm.name" class="field" /></label>
            <div class="form-label">
              <span id="spu-brand-label" class="form-label-text">品牌</span>
              <Listbox v-model="spuForm.brand">
                <div class="custom-select">
                  <ListboxButton class="custom-select-button" aria-labelledby="spu-brand-label">
                    <span :class="spuForm.brand ? 'text-neutral-900' : 'text-neutral-400'">{{ spuForm.brand || '请选择品牌' }}</span>
                    <ChevronUpDownIcon class="h-5 w-5 shrink-0 text-neutral-400" />
                  </ListboxButton>
                  <transition enter-active-class="transition duration-100 ease-out" enter-from-class="scale-95 opacity-0" enter-to-class="scale-100 opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="scale-100 opacity-100" leave-to-class="scale-95 opacity-0">
                    <ListboxOptions class="custom-select-options">
                      <ListboxOption v-slot="{ active, selected }" as="template" value="">
                        <li class="custom-select-option" :class="{ active }">
                          <span :class="selected ? 'font-semibold text-neutral-900' : 'text-neutral-500'">请选择品牌</span>
                          <CheckIcon v-if="selected" class="h-4 w-4 text-neutral-900" />
                        </li>
                      </ListboxOption>
                      <ListboxOption v-for="brand in spuBrandOptions" :key="brand" v-slot="{ active, selected }" as="template" :value="brand">
                        <li class="custom-select-option" :class="{ active }">
                          <span :class="selected ? 'font-semibold text-neutral-900' : 'text-neutral-700'">{{ brand }}</span>
                          <CheckIcon v-if="selected" class="h-4 w-4 text-neutral-900" />
                        </li>
                      </ListboxOption>
                    </ListboxOptions>
                  </transition>
                </div>
              </Listbox>
            </div>
            <label class="form-label"><span class="form-label-text">型号</span><input v-model.trim="spuForm.model" class="field" /></label>
            <label class="form-label"><span class="form-label-text">产品材质</span><input v-model.trim="spuForm.material" class="field" /></label>
            <label class="form-label"><span class="form-label-text">一级分类</span><input v-model.trim="spuForm.category1" class="field" /></label>
            <label class="form-label"><span class="form-label-text">二级分类</span><input v-model.trim="spuForm.category2" class="field" /></label>
          </div>
        </section>

        <template v-if="spuForm.id">
          <section class="editor-section">
            <div class="editor-section-head">
              <h2>产品模型</h2>
              <Button type="button" size="sm" variant="outline" :disabled="uploadingModel" @click="modelInput?.click()">
                <ArrowUpTrayIcon class="h-4 w-4" />{{ spuForm.model_url ? '更换模型' : '上传模型' }}
              </Button>
              <input ref="modelInput" class="hidden" type="file" accept=".stp,.step,.obj,.fbx" @change="uploadModel" />
            </div>
            <button v-if="spuForm.model_url" type="button" class="mt-3 text-sm text-blue-600 hover:underline" @click="openFile(spuForm.model_url)">{{ fileName(spuForm.model_url) }}</button>
            <p v-else class="mt-3 text-sm text-neutral-400">支持 STP、STEP、OBJ 和 FBX 文件</p>
          </section>

          <section class="editor-section">
            <div class="editor-section-head">
              <h2>认证资料</h2>
              <Button type="button" size="sm" variant="outline" @click="openAttachmentCreate"><PlusIcon class="h-4 w-4" />新增认证</Button>
            </div>
            <div v-if="!spuForm.attachments.length" class="inline-empty-state">暂无认证资料</div>
            <div v-else class="cert-table">
              <div class="cert-table-head hidden md:grid">
                <span>认证</span><span>认证日期</span><span>认证信息</span><span>测试报告</span><span>新增人</span><span></span>
              </div>
              <div v-for="attachment in spuForm.attachments" :key="attachment.id" class="cert-table-row grid md:grid-cols-[110px_110px_minmax(0,1fr)_minmax(0,1fr)_110px_36px] md:items-center">
                <div><span class="tag" :class="attachment.certification_type ? 'tag-ok' : ''">{{ attachment.certification_type || attachment.label || '历史资料' }}</span></div>
                <div class="text-sm tabular-nums text-neutral-700">{{ certificationDateLabel(attachment) }}</div>
                <button type="button" class="file-link" :disabled="!attachment.file_url" @click="openFile(attachment.file_url)">
                  <DocumentTextIcon class="h-4 w-4 shrink-0" />
                  <span class="truncate">{{ attachment.file_name || fileName(attachment.file_url) }}</span>
                </button>
                <button type="button" class="file-link" :disabled="!attachment.test_report_url" @click="openFile(attachment.test_report_url)">
                  <ClipboardDocumentCheckIcon class="h-4 w-4 shrink-0" />
                  <span class="truncate">{{ attachment.test_report_name || '无测试报告' }}</span>
                </button>
                <span class="text-xs text-neutral-500">{{ updaterLabel(attachment) }}</span>
                <button v-if="isAdmin" type="button" class="icon-button danger" aria-label="删除认证记录" title="删除认证记录" @click="removeAttachment(attachment)"><TrashIcon class="h-4 w-4" /></button>
              </div>
            </div>
          </section>
        </template>

        <div class="editor-footer">
          <Button type="button" variant="outline" @click="spuEditorOpen = false">取消</Button>
          <Button type="submit" :disabled="savingSpu">{{ savingSpu ? '保存中' : '保存' }}</Button>
        </div>
      </form>
    </Modal>

    <Modal v-model:open="skuDetailOpen" title="SKU 详情" size="lg">
      <div v-if="skuDetail" class="sku-detail">
        <div class="sku-detail-hero"><ProductThumbnail :src="skuDetail.image_url" :alt="skuDetail.sku" /><div><span class="detail-kicker">{{ skuDetail.spu || '产品 SKU' }}</span><h2>{{ skuDetail.sku || '-' }}</h2><p>{{ skuDetail.name || skuDetail.style_name || '未填写名称' }}</p></div><span class="tag" :class="platformSkuSummary(skuDetail).length ? 'tag-ok' : ''">{{ platformSkuSummary(skuDetail).length ? `${platformSkuSummary(skuDetail).length} 个平台编码` : '未关联平台' }}</span></div>
        <section class="sku-detail-section"><h3>识别信息</h3><dl class="sku-detail-grid"><div><dt>ASIN</dt><dd>{{ skuDetail.asin || '-' }}</dd></div><div><dt>款式</dt><dd>{{ skuDetail.style_name || '-' }}</dd></div><div><dt>颜色</dt><dd>{{ skuDetail.color || '-' }}</dd></div><div><dt>尺寸</dt><dd>{{ skuDetail.size || '-' }}</dd></div></dl></section>
        <section class="sku-detail-section"><h3>平台编码</h3><div class="sku-detail-platforms"><span v-for="entry in platformSkuSummary(skuDetail)" :key="entry.platform + entry.sku" class="platform-sku-tag"><b>{{ entry.platform }}</b>{{ entry.sku }}</span><span v-if="!platformSkuSummary(skuDetail).length" class="muted-copy">暂无平台编码</span></div></section>
        <section class="sku-detail-section"><h3>商品属性</h3><dl class="sku-detail-grid"><div><dt>功能</dt><dd>{{ skuDetail.feature || '-' }}</dd></div><div><dt>材质</dt><dd>{{ skuDetail.material || '-' }}</dd></div><div><dt>分类</dt><dd>{{ [skuDetail.category1, skuDetail.category2, skuDetail.category3].filter(Boolean).join(' / ') || '-' }}</dd></div><div><dt>包装数量</dt><dd>{{ skuDetail.package_qty || '-' }}</dd></div></dl></section>
      </div>
      <template #footer><Button variant="outline" @click="skuDetailOpen = false">关闭</Button><Button @click="editSkuFromDetail"><PencilSquareIcon class="h-4 w-4" />编辑 SKU</Button></template>
    </Modal>

    <Modal v-model:open="skuEditorOpen" :title="skuForm.id ? '编辑 SKU' : '新增 SKU'" size="xl">
      <form class="editor-form" @submit.prevent="saveSku">
        <section class="product-image-editor">
          <ProductThumbnail :src="skuForm.image_url" :alt="skuForm.sku || '产品图片'" />
          <div class="image-editor-fields">
            <label class="form-label"><span class="form-label-text">产品图片</span><input v-model.trim="skuForm.image_url" class="field" placeholder="粘贴图片链接或上传图片" :disabled="uploadingImage" /></label>
            <div class="image-editor-actions"><label class="row-button"><input type="file" class="sr-only" accept="image/png,image/jpeg,image/webp,image/gif" :disabled="uploadingImage" @change="selectProductImage" />{{ uploadingImage ? '上传中…' : '上传图片' }}</label><button v-if="skuForm.image_url" type="button" class="row-button" :disabled="uploadingImage" @click="skuForm.image_url = ''">移除</button><span>JPG / PNG / WebP / GIF · 最大 30 MB</span></div>
          </div>
        </section>
        <section class="editor-section">
          <div class="editor-section-head"><h2>基础信息</h2></div>
          <div class="form-grid">
            <label class="form-label"><span class="form-label-text">SKU<span class="required">*</span></span><input v-model.trim="skuForm.sku" class="field" required /></label>
            <label class="form-label"><span class="form-label-text">ASIN<span class="required">*</span></span><input v-model.trim="skuForm.asin" class="field" required /></label>
            <label class="form-label"><span class="form-label-text">SPU</span><input v-model="skuForm.spu" class="field bg-neutral-50" disabled /></label>
            <label class="form-label"><span class="form-label-text">名称</span><input v-model.trim="skuForm.name" class="field" /></label>
            <label class="form-label"><span class="form-label-text">款式名</span><input v-model.trim="skuForm.style_name" class="field" /></label>
          </div>
        </section>
        <section class="editor-section">
          <div class="editor-section-head"><h2>商品属性</h2></div>
          <div class="form-grid">
            <label class="form-label"><span class="form-label-text">颜色</span><input v-model.trim="skuForm.color" class="field" /></label>
            <label class="form-label"><span class="form-label-text">尺寸</span><input v-model.trim="skuForm.size" class="field" /></label>
            <label class="form-label"><span class="form-label-text">功能</span><input v-model.trim="skuForm.feature" class="field" /></label>
            <label class="form-label"><span class="form-label-text">材质</span><input v-model.trim="skuForm.material" class="field" /></label>
          </div>
        </section>
        <section class="editor-section">
          <div class="editor-section-head">
            <h2>平台 SKU</h2>
            <span class="section-hint">保存后订单转 MCF 会自动使用这些映射</span>
          </div>
          <div class="platform-sku-editor-grid">
            <div v-for="platform in platformOptions" :key="platform.code" class="platform-sku-editor">
              <span class="form-label-text">{{ platform.label }}</span>
              <div v-for="(value, index) in skuForm.platform_skus[platform.code]" :key="`${platform.code}-${index}`" class="platform-sku-input-row">
                <input v-model.trim="skuForm.platform_skus[platform.code][index]" class="field" :disabled="!platformEditable(platform.code)" :placeholder="`${platform.label} 商品编码`" />
                <button v-if="platformEditable(platform.code) && skuForm.platform_skus[platform.code].length > 1" type="button" class="icon-button danger" :aria-label="`删除 ${platform.label} SKU`" @click="removePlatformSku(platform.code, index)"><TrashIcon class="h-4 w-4" /></button>
              </div>
              <button v-if="platformEditable(platform.code)" type="button" class="add-platform-sku" @click="addPlatformSku(platform.code)"><PlusIcon class="h-3.5 w-3.5" />添加编码</button>
              <small v-else class="platform-sku-readonly">仅可查看</small>
            </div>
          </div>
        </section>
        <section class="editor-section">
          <div class="editor-section-head"><h2>分类与上架</h2></div>
          <div class="form-grid form-grid-4">
            <label class="form-label"><span class="form-label-text">一级分类</span><input v-model.trim="skuForm.category1" class="field" /></label>
            <label class="form-label"><span class="form-label-text">二级分类</span><input v-model.trim="skuForm.category2" class="field" /></label>
            <label class="form-label"><span class="form-label-text">三级分类</span><input v-model.trim="skuForm.category3" class="field" /></label>
            <label class="form-label"><span class="form-label-text">上架日期</span><input v-model="skuForm.listed_at" class="field" type="date" /></label>
          </div>
        </section>
        <section class="editor-section">
          <div class="editor-section-head"><h2>包装与价格</h2></div>
          <div class="form-grid form-grid-3">
            <label class="form-label"><span class="form-label-text">包装数量</span><input v-model.trim="skuForm.package_qty" class="field" /></label>
            <label class="form-label"><span class="form-label-text">采购价</span><input v-model.trim="skuForm.purchase_price" class="field" inputmode="decimal" /></label>
            <label class="form-label"><span class="form-label-text">包装重量</span><input v-model.trim="skuForm.package_weight" class="field" /></label>
            <label class="form-label"><span class="form-label-text">包装长</span><input v-model.trim="skuForm.package_length" class="field" /></label>
            <label class="form-label"><span class="form-label-text">包装宽</span><input v-model.trim="skuForm.package_width" class="field" /></label>
            <label class="form-label"><span class="form-label-text">包装高</span><input v-model.trim="skuForm.package_height" class="field" /></label>
          </div>
        </section>
        <div class="editor-footer">
          <Button type="button" variant="outline" @click="skuEditorOpen = false">取消</Button>
          <Button type="submit" :disabled="savingSku">{{ savingSku ? '保存中' : '保存' }}</Button>
        </div>
      </form>
    </Modal>

    <Modal v-model:open="attachmentEditorOpen" title="新增认证记录" size="lg">
      <form class="space-y-4" @submit.prevent="saveAttachment">
        <div class="form-grid">
          <label class="form-label"><span class="form-label-text">认证名称<span class="required">*</span></span>
            <select v-model="attachmentForm.certificationType" class="field" required>
              <option v-for="type in certificationTypes" :key="type" :value="type">{{ type }}</option>
            </select>
          </label>
          <label class="form-label"><span class="form-label-text">认证日期<span class="required">*</span></span><input v-model="attachmentForm.certificationDate" class="field" type="date" required /></label>
        </div>
        <div class="cert-files-grid">
          <label class="form-label">
            <span class="form-label-text">认证信息<span class="required">*</span></span>
            <span class="file-picker">
              <DocumentTextIcon class="h-5 w-5 shrink-0 text-neutral-500" />
              <span class="truncate">{{ attachmentForm.certificateFile?.name || '选择认证信息文件' }}</span>
              <input class="file-picker-input" type="file" accept="image/*,.pdf,.doc,.docx" required @change="attachmentForm.certificateFile = $event.target.files?.[0] || null" />
            </span>
          </label>
          <label class="form-label">
            <span class="form-label-text">测试报告<span class="required">*</span></span>
            <span class="file-picker">
              <ClipboardDocumentCheckIcon class="h-5 w-5 shrink-0 text-neutral-500" />
              <span class="truncate">{{ attachmentForm.testReportFile?.name || '选择测试报告文件' }}</span>
              <input class="file-picker-input" type="file" accept="image/*,.pdf,.doc,.docx" required @change="attachmentForm.testReportFile = $event.target.files?.[0] || null" />
            </span>
          </label>
        </div>
        <div class="flex justify-end gap-2 border-t border-neutral-200 pt-4">
          <Button type="button" variant="outline" @click="attachmentEditorOpen = false">取消</Button>
          <Button type="submit" :disabled="savingAttachment">{{ savingAttachment ? '新增中' : '新增记录' }}</Button>
        </div>
      </form>
    </Modal>

    <Modal v-model:open="deleteDialog.open" :title="deleteDialog.title" size="sm">
      <p class="text-sm leading-6 text-neutral-600">{{ deleteDialog.message }}</p>
      <template #footer>
        <Button type="button" variant="outline" :disabled="deleting" @click="deleteDialog.open = false">取消</Button>
        <button type="button" class="delete-confirm-button" :disabled="deleting" @click="confirmDelete">{{ deleting ? '删除中' : '确认删除' }}</button>
      </template>
    </Modal>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue'
import {
  ArrowPathIcon,
  ArrowUpTrayIcon,
  CheckIcon,
  ChevronRightIcon,
  ChevronUpDownIcon,
  ClipboardDocumentCheckIcon,
  CubeIcon,
  DocumentArrowDownIcon,
  DocumentTextIcon,
  MagnifyingGlassIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import ProductThumbnail from '@/components/ui/ProductThumbnail.vue'
import { uploadProductImage } from '@/utils/productImage'
import MainLayout from '@/components/layout/MainLayout.vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import request, { downloadBrandOptions, resolveImageUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'
import { readProductCatalogFile } from '@/utils/productCatalogImport'
import { useAuthStore } from '@/stores/auth'

const toast = useToast()
const authStore = useAuthStore()
const items = ref([])
const loading = ref(false)
const keyword = ref('')
const expandedId = ref(null)
const page = ref(1)
const pageSize = ref(12)
const pageSizeOptions = [12, 24, 48, 96]
const total = ref(0)
const selectedSpuIds = ref(new Set())
const brandOptions = ref([...downloadBrandOptions])

const spuEditorOpen = ref(false)
const skuEditorOpen = ref(false)
const skuDetailOpen = ref(false)
const skuDetail = ref(null)
const skuDetailSpu = ref(null)
const attachmentEditorOpen = ref(false)
const savingSpu = ref(false)
const savingSku = ref(false)
const savingAttachment = ref(false)
const deleting = ref(false)
const uploadingModel = ref(false)
const importing = ref(false)
const uploadingImage = ref(false)
const importInput = ref(null)
const modelInput = ref(null)
const certificationTypes = ['CEC', 'DOE', 'cUPC', 'NSF 61', 'NSF 372', 'WARS', 'ACS']
const platformOptions = [
  { code: 'TEMU', label: 'TEMU' },
  { code: 'TIKTOK', label: 'TikTok Shop' },
  { code: 'LOWES', label: "Lowe's" },
]
const deleteDialog = reactive({ open: false, type: '', record: null, title: '', message: '' })

const spuForm = reactive(defaultSpu())
const skuForm = reactive(defaultSku())
const attachmentForm = reactive({ certificationType: 'CEC', certificationDate: todayInputValue(), certificateFile: null, testReportFile: null })

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const isAdmin = computed(() => (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin'))
const editablePlatformCodes = computed(() => {
  if (isAdmin.value) return platformOptions.map(platform => platform.code)
  const assigned = Array.isArray(authStore.user?.order_platforms) && authStore.user.order_platforms.length
    ? authStore.user.order_platforms
    : [authStore.user?.platform]
  return platformOptions.filter(platform => assigned.some(value => normalizePlatform(value) === platform.code)).map(platform => platform.code)
})
const selectedCount = computed(() => selectedSpuIds.value.size)
const allCurrentPageSelected = computed(() => items.value.length > 0 && items.value.every(item => selectedSpuIds.value.has(item.id)))
const someCurrentPageSelected = computed(() => !allCurrentPageSelected.value && items.value.some(item => selectedSpuIds.value.has(item.id)))
const spuBrandOptions = computed(() => {
  if (!spuForm.brand || brandOptions.value.includes(spuForm.brand)) return brandOptions.value
  return [spuForm.brand, ...brandOptions.value]
})
const summary = computed(() => [
  { label: 'SPU', value: total.value },
  { label: '当前页 SKU', value: items.value.reduce((sum, item) => sum + (item.sku_count || item.skus?.length || 0), 0) },
  { label: '平台关联', value: items.value.reduce((sum, item) => sum + spuPlatformSummary(item).reduce((inner, entry) => inner + entry.count, 0), 0) },
  { label: '资料记录', value: items.value.reduce((sum, item) => sum + (item.model_url ? 1 : 0) + (item.attachments?.length || 0), 0) },
])

function defaultSpu() {
  return { id: null, spu: '', name: '', brand: '', model: '', material: '', category1: '', category2: '', model_url: '', attachments: [] }
}

function defaultSku() {
  return {
    id: null, image_url: '', spu_id: null, sku: '', asin: '', spu: '', name: '', style_name: '', color: '', size: '', feature: '', package_qty: '', material: '',
    category1: '', category2: '', category3: '', listed_at: '', package_length: '', package_width: '', package_height: '', package_weight: '', purchase_price: '',
    platform_skus: { TEMU: [''], TIKTOK: [''], LOWES: [''] },
  }
}

function errorMessage(error, fallback) {
  return error?.response?.data?.error || fallback
}

function updaterLabel(item) {
  if (item?.updated_by_name) return item.updated_by_name
  if (item?.updated_by) return `用户 #${item.updated_by}`
  return '-'
}

function updatedAtLabel(item) {
  if (!item?.updated_at) return '暂无时间'
  const date = new Date(item.updated_at)
  if (Number.isNaN(date.getTime())) return String(item.updated_at).slice(0, 16).replace('T', ' ')
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}

function auditTitle(item) {
  return `${updaterLabel(item)} · ${updatedAtLabel(item)}`
}

async function fetchSpus() {
  loading.value = true
  try {
    const response = await request.get('/product-catalog/spus', { params: { keyword: keyword.value, page: page.value, page_size: pageSize.value } })
    items.value = response.data?.items || []
    total.value = response.data?.total || 0
    if (expandedId.value && !items.value.some(item => item.id === expandedId.value)) expandedId.value = null
    syncOpenSpu()
  } catch (error) {
    toast.error(errorMessage(error, '产品库加载失败'))
  } finally {
    loading.value = false
  }
}

async function fetchBrandOptions() {
  try {
    const response = await request.get('/download-options')
    if (Array.isArray(response.data?.brands) && response.data.brands.length) {
      brandOptions.value = response.data.brands
    }
  } catch {
    // 与下载弹窗一致：接口不可用时保留本地兜底品牌。
  }
}

function clearSelection() { selectedSpuIds.value = new Set() }
function search() { page.value = 1; clearSelection(); fetchSpus() }
function changePage(value) { page.value = value; clearSelection(); fetchSpus() }
function changePageSize() { page.value = 1; clearSelection(); fetchSpus() }
function toggleSpu(id) { expandedId.value = expandedId.value === id ? null : id }

function toggleSpuSelection(id, checked) {
  const next = new Set(selectedSpuIds.value)
  if (checked) next.add(id)
  else next.delete(id)
  selectedSpuIds.value = next
}

function toggleCurrentPage(checked) {
  selectedSpuIds.value = checked ? new Set(items.value.map(item => item.id)) : new Set()
}

function openSpuEditor(record = {}) {
  Object.assign(spuForm, defaultSpu(), record, { attachments: Array.isArray(record.attachments) ? record.attachments : [] })
  spuEditorOpen.value = true
}

function syncOpenSpu() {
  if (!spuEditorOpen.value || !spuForm.id) return
  const current = items.value.find(item => item.id === spuForm.id)
  if (current) Object.assign(spuForm, current, { attachments: current.attachments || [] })
}

async function saveSpu() {
  if (!spuForm.spu) return
  savingSpu.value = true
  try {
    const payload = Object.fromEntries(['spu', 'name', 'brand', 'model', 'material', 'category1', 'category2'].map(key => [key, spuForm[key]]))
    const response = spuForm.id
      ? await request.put(`/product-catalog/spus/${spuForm.id}`, payload)
      : await request.post('/product-catalog/spus', payload)
    spuEditorOpen.value = false
    await fetchSpus()
    if (response.data?.id) expandedId.value = response.data.id
    toast.success(spuForm.id ? 'SPU 已更新' : 'SPU 已创建')
  } catch (error) {
    toast.error(errorMessage(error, 'SPU 保存失败'))
  } finally {
    savingSpu.value = false
  }
}

function removeSpu(spu) {
  openDeleteDialog('spu', spu, `删除 SPU「${spu.spu}」`, '该 SPU 下的 SKU 和认证资料也会一并删除。')
}

function openBatchDeleteDialog() {
  if (!selectedCount.value) return
  openDeleteDialog('spu-batch', null, `批量删除 ${selectedCount.value} 个 SPU`, '所选 SPU 下的全部 SKU 和认证资料也会一并删除。此操作无法撤销。')
}

function openSkuEditor(spu, record = {}) {
  const platformSKUs = Object.fromEntries(platformOptions.map(({ code }) => {
    const values = Array.isArray(record.platform_skus?.[code]) ? record.platform_skus[code].filter(Boolean) : []
    return [code, values.length ? [...values] : ['']]
  }))
  Object.assign(skuForm, defaultSku(), record, { platform_skus: platformSKUs, spu_id: spu.id, spu: spu.spu })
  skuEditorOpen.value = true
}

function openSkuDetail(spu, record) {
  skuDetailSpu.value = spu
  skuDetail.value = record
  skuDetailOpen.value = true
}

function editSkuFromDetail() {
  if (!skuDetail.value || !skuDetailSpu.value) return
  skuDetailOpen.value = false
  openSkuEditor(skuDetailSpu.value, skuDetail.value)
}

function addPlatformSku(platform) {
  skuForm.platform_skus[platform].push('')
}

function removePlatformSku(platform, index) {
  skuForm.platform_skus[platform].splice(index, 1)
  if (!skuForm.platform_skus[platform].length) skuForm.platform_skus[platform].push('')
}

function platformSkuSummary(sku) {
  return platformOptions.flatMap(({ code, label }) => (sku.platform_skus?.[code] || []).map(value => ({ platform: code === 'TIKTOK' ? 'TK' : label, sku: value })).filter(item => item.sku))
}

function spuPlatformSummary(spu) {
  const counts = new Map()
  for (const sku of filteredSkus(spu)) {
    for (const entry of platformSkuSummary(sku)) {
      counts.set(entry.platform, (counts.get(entry.platform) || 0) + 1)
    }
  }
  return [...counts.entries()].map(([platform, count]) => ({ platform, count }))
}

function normalizePlatform(value) {
  const normalized = String(value || '').trim().toUpperCase()
  if (normalized === 'TK' || normalized === 'TIKTOK SHOP') return 'TIKTOK'
  if (normalized === 'LOWE' || normalized === "LOWE'S" || normalized === 'LOWES' || normalized === 'LOWE’S') return 'LOWES'
  return normalized
}

function platformEditable(platform) {
  return editablePlatformCodes.value.includes(platform)
}

async function selectProductImage(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  uploadingImage.value = true
  try { skuForm.image_url = await uploadProductImage(file) }
  catch (error) { toast.error(errorMessage(error, error.message || '图片上传失败')) }
  finally { uploadingImage.value = false }
}

async function saveSku() {
  if (uploadingImage.value) return
  if (!skuForm.sku || !skuForm.asin) return
  savingSku.value = true
  try {
    const wasExisting = Boolean(skuForm.id)
    const payload = { ...skuForm }
    const response = wasExisting
      ? await request.put(`/product-catalog/skus/${skuForm.id}`, payload)
      : await request.post('/product-catalog/skus', payload)
    const savedId = response.data?.id || skuForm.id
    const platform_skus = Object.fromEntries(editablePlatformCodes.value.map(code => [code, skuForm.platform_skus[code] || []]))
    await request.put(`/product-catalog/skus/${savedId}/platform-skus`, { platform_skus })
    skuEditorOpen.value = false
    await fetchSpus()
    toast.success(wasExisting ? 'SKU 已更新' : 'SKU 已创建')
  } catch (error) { toast.error(errorMessage(error, 'SKU 保存失败')) }
  finally { savingSku.value = false }
}

async function importCatalog(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  importing.value = true
  try {
    const rows = await readProductCatalogFile(file, { uploadImage: uploadProductImage })
    const response = await request.post('/product-catalog/import', { rows })
    page.value = 1
    await fetchSpus()
    const result = response.data || {}
    toast.success(`导入完成：新增 ${result.created_skus || 0} 个 SKU，更新 ${result.updated_skus || 0} 个 SKU，新建 ${result.created_spus || 0} 个 SPU`)
  } catch (error) {
    toast.error(errorMessage(error, error?.message || '产品库导入失败'))
  } finally {
    importing.value = false
  }
}

function removeSku(sku) {
  openDeleteDialog('sku', sku, `删除 SKU「${sku.sku}」`, '删除后无法恢复。')
}

function filteredSkus(spu) {
  return Array.isArray(spu.skus) ? spu.skus : []
}

async function uploadModel(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file || !spuForm.id) return
  uploadingModel.value = true
  try {
    const data = new FormData()
    data.append('file', file)
    await request.post(`/product-catalog/spus/${spuForm.id}/model`, data)
    await fetchSpus()
    toast.success('产品模型已更新')
  } catch (error) { toast.error(errorMessage(error, '模型上传失败')) }
  finally { uploadingModel.value = false }
}

function openAttachmentCreate() {
  Object.assign(attachmentForm, { certificationType: 'CEC', certificationDate: todayInputValue(), certificateFile: null, testReportFile: null })
  attachmentEditorOpen.value = true
}

async function saveAttachment() {
  if (!spuForm.id || !attachmentForm.certificationType || !attachmentForm.certificationDate || !attachmentForm.certificateFile || !attachmentForm.testReportFile) return
  savingAttachment.value = true
  try {
    const data = new FormData()
    data.append('certification_type', attachmentForm.certificationType)
    data.append('certification_date', attachmentForm.certificationDate)
    data.append('certificate_file', attachmentForm.certificateFile)
    data.append('test_report_file', attachmentForm.testReportFile)
    await request.post(`/product-catalog/spus/${spuForm.id}/attachments`, data)
    attachmentEditorOpen.value = false
    await fetchSpus()
    toast.success('认证记录已新增')
  } catch (error) { toast.error(errorMessage(error, '认证资料保存失败')) }
  finally { savingAttachment.value = false }
}

function removeAttachment(record) {
  const recordName = `${record.certification_type || record.label || '历史资料'} ${certificationDateLabel(record)}`
  openDeleteDialog('attachment', record, `删除认证记录「${recordName}」`, '认证信息和测试报告将同时删除。')
}

function openDeleteDialog(type, record, title, message) {
  Object.assign(deleteDialog, { open: true, type, record, title, message })
}

async function confirmDelete() {
  const record = deleteDialog.record
  if (deleting.value || (deleteDialog.type !== 'spu-batch' && !record?.id)) return
  deleting.value = true
  try {
    if (deleteDialog.type === 'spu-batch') {
      const deletedCurrentPage = items.value.every(item => selectedSpuIds.value.has(item.id))
      const response = await request.post('/product-catalog/spus/batch-delete', { ids: [...selectedSpuIds.value] })
      clearSelection()
      if (deletedCurrentPage && page.value > 1) page.value -= 1
      toast.success(`已删除 ${response.data?.deleted || 0} 个 SPU`)
    } else if (deleteDialog.type === 'spu') {
      await request.delete(`/product-catalog/spus/${record.id}`)
      if (selectedSpuIds.value.has(record.id)) toggleSpuSelection(record.id, false)
    } else if (deleteDialog.type === 'sku') {
      await request.delete(`/product-catalog/skus/${record.id}`)
    } else if (deleteDialog.type === 'attachment') {
      await request.delete(`/product-catalog/spus/${spuForm.id}/attachments/${record.id}`)
    }
    deleteDialog.open = false
    await fetchSpus()
    if (deleteDialog.type !== 'spu-batch') toast.success(deleteDialog.type === 'spu' ? 'SPU 已删除' : deleteDialog.type === 'sku' ? 'SKU 已删除' : '认证资料已删除')
  } catch (error) {
    const fallback = deleteDialog.type === 'spu-batch' ? '批量删除失败' : deleteDialog.type === 'spu' ? 'SPU 删除失败' : deleteDialog.type === 'sku' ? 'SKU 删除失败' : '认证资料删除失败'
    toast.error(errorMessage(error, fallback))
  } finally {
    deleting.value = false
  }
}

function openFile(url) { if (url) window.open(resolveImageUrl(url), '_blank', 'noopener,noreferrer') }
function fileName(url) { return String(url || '').split('?')[0].split('/').pop() || '文件' }
function todayInputValue() {
  const today = new Date()
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
function certificationDateLabel(record) {
  const value = record?.certification_date || record?.created_at
  if (!value) return '-'
  const date = String(value).slice(0, 10)
  const [year, month, day] = date.split('-')
  return year && month && day ? `${year}-${month}-${day}` : date
}

onMounted(() => {
  fetchSpus()
  fetchBrandOptions()
})
</script>

<style scoped>
.product-identity { display: flex; align-items: center; gap: 12px; min-width: 0; padding: 8px 0; }
.product-identity .cell-stack { min-width: 0; }
.product-image-editor { display: flex; align-items: center; gap: 20px; padding: 20px 0; border-bottom: 1px solid #e5e7eb; }
.product-image-editor :deep(.product-thumbnail) { width: 88px; height: 88px; flex-basis: 88px; }
.image-editor-fields { flex: 1; min-width: 0; }
.image-editor-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.image-editor-actions label { cursor: pointer; }
.image-editor-actions span { color: #98a2b3; font-size: 11px; }

.catalog-page { width: min(100%, 1540px); margin: 0 auto; padding: 24px; color: #1f2937; }
.page-header { min-height: 64px; margin-bottom: 18px; display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.page-heading h1 { color: #111827; font-size: 24px; line-height: 32px; font-weight: 700; letter-spacing: 0; }
.page-heading p { margin-top: 4px; color: #6b7280; font-size: 13px; line-height: 20px; }
.header-actions { display: flex; align-items: center; gap: 8px; }
.template-link { height: 40px; display: inline-flex; align-items: center; gap: 7px; border: 1px solid #d7dce2; border-radius: 8px; padding: 0 13px; color: #374151; font-size: 14px; font-weight: 600; }
.template-link:hover { border-color: #9ca3af; background: #f9fafb; color: #111827; }
.catalog-surface { overflow: hidden; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; box-shadow: 0 1px 2px rgba(16,24,40,.04); }
.catalog-controls { min-height: 66px; padding: 12px 16px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border-bottom: 1px solid #e8ebef; background: #fff; }
.catalog-control-actions { display: flex; align-items: center; gap: 8px; }
.batch-delete-button { height: 40px; display: inline-flex; align-items: center; gap: 7px; border: 1px solid #efb6b6; border-radius: 8px; padding: 0 13px; background: #fff; color: #b42318; font-size: 13px; font-weight: 650; }
.batch-delete-button:hover:not(:disabled) { border-color: #dc8989; background: #fff5f5; color: #991b1b; }
.batch-delete-button:disabled { cursor: not-allowed; border-color: #e5e7eb; background: #f9fafb; color: #a3a3a3; }
.toolbar-search { width: min(440px, 52vw); }
.field.toolbar-search-input { padding-left: 38px; }
.control-icon-button { width: 40px; height: 40px; display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; border: 1px solid #d7dce2; border-radius: 8px; background: #fff; color: #4b5563; transition: border-color .15s, background-color .15s, color .15s; }
.control-icon-button:hover { border-color: #9ca3af; background: #f9fafb; color: #111827; }
.summary-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); border-bottom: 1px solid #e8ebef; background: #fafbfc; }
.summary-item { min-height: 66px; padding: 12px 18px; display: flex; flex-direction: column; justify-content: center; border-right: 1px solid #e8ebef; }
.summary-item:last-child { border-right: 0; }
.summary-item span { color: #6b7280; font-size: 12px; line-height: 18px; }
.summary-item strong { margin-top: 2px; color: #111827; font-size: 20px; line-height: 26px; font-weight: 700; }
.catalog-list { min-height: 360px; overflow: auto; background: #fff; scrollbar-gutter: stable; }
.list-head, .spu-list-row, .sku-list-row, .sku-empty-row { min-width: 1220px; display: grid; grid-template-columns: 22px 32px minmax(240px,1.35fr) minmax(190px,.95fr) minmax(200px,1fr) 150px 212px; align-items: center; column-gap: 14px; }
.list-head { position: sticky; top: 0; z-index: 2; height: 46px; padding: 0 18px; border-bottom: 1px solid #dfe3e8; background: #f4f6f8; color: #667085; font-size: 11px; line-height: 16px; font-weight: 700; text-transform: uppercase; white-space: nowrap; }
.spu-group { min-width: 1220px; border-bottom: 1px solid #e8ebef; background: #fff; }
.selection-cell { width: 22px; height: 32px; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; }
.selection-cell input { width: 15px; height: 15px; accent-color: #252a31; cursor: pointer; }
.selection-cell input:disabled { cursor: not-allowed; }
.spu-group:last-child { border-bottom: 0; }
.spu-list-row { min-height: 70px; padding: 0 18px; background: #fff; transition: background-color .15s, box-shadow .15s; }
.spu-list-row:hover { background: #fafbfc; }
.spu-list-row.expanded { background: #f8fafc; box-shadow: inset 3px 0 0 #374151; }
.expand-button { width: 26px; height: 26px; display: inline-flex; align-items: center; justify-content: center; border-radius: 6px; color: #667085; transition: background-color .15s, color .15s; }
.expand-button:hover, .expand-button.open { background: #e9edf2; color: #111827; }
.spu-code { min-width: 0; display: flex; flex-direction: column; gap: 3px; overflow: hidden; text-align: left; }
.spu-code strong, .spu-code span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.spu-code strong { color: #111827; font-size: 14px; line-height: 20px; font-weight: 650; }
.spu-code span { color: #6b7280; font-size: 12px; line-height: 18px; }
.spu-code:hover strong { color: #000; text-decoration: underline; text-underline-offset: 3px; }
.cell-stack { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.cell-stack strong { overflow: hidden; color: #344054; font-size: 13px; line-height: 19px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.cell-stack span { overflow: hidden; color: #7a8493; font-size: 12px; line-height: 18px; text-overflow: ellipsis; white-space: nowrap; }
.row-actions, .audit-cell { display: flex; align-items: center; }
.row-actions { min-width: 0; justify-content: flex-end; gap: 6px; flex-wrap: nowrap; }
.platform-summary-cell { min-width: 0; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.platform-summary-item { display: inline-flex; align-items: center; gap: 5px; color: #475467; font-size: 12px; line-height: 18px; }
.platform-summary-item b { color: #111827; font-size: 10px; font-weight: 700; letter-spacing: .02em; }
.muted-copy { color: #98a2b3; font-size: 12px; line-height: 18px; }
.sku-overview span { color: #98a2b3; }
.platform-sku-cell { min-width: 0; display: flex; align-items: center; gap: 4px; flex-wrap: wrap; }
.platform-sku-tag { max-width: 145px; display: inline-flex; align-items: center; gap: 4px; overflow: hidden; border: 1px solid #d8dde5; border-radius: 5px; background: #fff; padding: 2px 6px; color: #475467; font-size: 10px; line-height: 16px; text-overflow: ellipsis; white-space: nowrap; }
.platform-sku-tag b { color: #667085; font-size: 9px; font-weight: 700; }
.sku-list-row, .sku-empty-row { min-height: 58px; padding: 0 18px; border-top: 1px solid #e8ebef; background: #fbfcfd; box-shadow: inset 3px 0 0 #d1d5db; }
.sku-list-row:hover { background: #f7f9fb; }
.sku-empty-row { color: #7a8493; font-size: 12px; }
.sku-marker { width: 28px; height: 20px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid #d8dde5; border-radius: 4px; background: #fff; color: #667085; font-size: 9px; font-weight: 700; }
.sku-count { color: #344054; font-size: 13px; line-height: 20px; font-weight: 650; text-align: center; }
.sku-count-muted { color: #98a2b3; font-weight: 500; }
.audit-cell { min-width: 0; }
.audit-copy { min-width: 0; display: flex; flex-direction: column; }
.audit-copy strong, .audit-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.audit-copy strong { color: #344054; font-size: 12px; line-height: 17px; font-weight: 600; }
.audit-copy small { color: #98a2b3; font-size: 10px; line-height: 15px; }
.loading-row, .empty-row { min-height: 250px; display: flex; align-items: center; justify-content: center; color: #667085; font-size: 13px; }
.empty-row { flex-direction: column; }
.pager { min-height: 54px; padding: 9px 16px; display: flex; align-items: center; justify-content: space-between; gap: 16px; border-top: 1px solid #dfe3e8; background: #fafbfc; color: #667085; font-size: 12px; }
.page-size-control, .page-size-control label { display: flex; align-items: center; gap: 8px; }
.page-size-select { height: 32px; border: 1px solid #d5dae1; border-radius: 6px; background: #fff; padding: 0 28px 0 9px; color: #344054; font-size: 12px; outline: none; }
.page-size-select:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1); }
.field { width: 100%; height: 42px; border: 1px solid #d5dae1; border-radius: 8px; background: #fff; padding: 0 13px; font-size: 14px; color: #111827; outline: none; box-shadow: 0 1px 2px rgba(16,24,40,.03); transition: border-color .15s, box-shadow .15s, background-color .15s; }
.field::placeholder { color: #a1a8b3; }
.field:hover:not(:disabled) { border-color: #aeb6c2; }
.field:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1), 0 1px 2px rgba(16,24,40,.04); }
.field:disabled { cursor: not-allowed; border-color: #e2e5e9; background: #f4f5f7; color: #7a8493; box-shadow: none; }
.icon-button { width: 32px; height: 32px; display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; border: 1px solid #d7dce2; border-radius: 6px; background: #fff; color: #475467; transition: border-color .15s, background-color .15s, color .15s; }
.icon-button:hover { border-color: #9ca3af; background: #f4f5f7; color: #111827; }
.icon-button.danger { border-color: #f0d0d0; color: #b42318; }
.icon-button.danger:hover { border-color: #e7a9a9; background: #fff5f5; color: #991b1b; }
.row-button { display: inline-flex; height: 32px; align-items: center; justify-content: center; gap: 5px; border: 1px solid #d7dce2; border-radius: 6px; padding: 0 10px; background: #fff; color: #475467; font-size: 12px; font-weight: 550; transition: border-color .15s, background-color .15s, color .15s; }
.row-button:hover { border-color: #9ca3af; background: #f9fafb; color: #111827; }
.row-button-primary { border-color: #252a31; background: #252a31; color: #fff; }
.row-button-primary:hover { border-color: #111827; background: #111827; color: #fff; }
.tag { display: inline-flex; align-items: center; height: 23px; border: 1px solid #dde1e6; border-radius: 5px; background: #f8f9fa; padding: 0 7px; color: #667085; font-size: 10px; line-height: 16px; font-weight: 600; }
.tag-ok { border-color: #b9dec9; background: #f0faf4; color: #16794a; }
.file-link { display: inline-flex; min-width: 0; align-items: center; gap: 6px; color: #2563eb; font-size: 13px; text-align: left; }
.file-link:hover:not(:disabled) { text-decoration: underline; }
.file-link:disabled { cursor: default; color: #a3a3a3; }
.inline-empty-state { min-height: 74px; display: flex; align-items: center; justify-content: center; border: 1px dashed #cfd5dd; border-radius: 8px; background: #fafbfc; color: #98a2b3; font-size: 13px; }
.cert-table { overflow: hidden; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; }
.cert-table-head, .cert-table-row { grid-template-columns: 110px 110px minmax(0,1fr) minmax(0,1fr) 110px 36px; gap: 12px; padding: 0 14px; }
.cert-table-head { min-height: 38px; align-items: center; border-bottom: 1px solid #e5e7eb; background: #f6f7f9; color: #667085; font-size: 11px; font-weight: 700; }
.cert-table-row { min-height: 54px; border-bottom: 1px solid #e8ebef; }
.cert-table-row:last-child { border-bottom: 0; }
.editor-form { display: flex; flex-direction: column; }
.editor-section { padding: 22px 0; border-top: 1px solid #e5e7eb; }
.editor-section:first-child { border-top: 0; padding-top: 0; }
.editor-section-head { min-height: 32px; margin-bottom: 16px; display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.editor-section-head h2 { display: flex; align-items: center; gap: 8px; color: #1f2937; font-size: 14px; line-height: 20px; font-weight: 700; }
.editor-section-head h2::before { width: 3px; height: 16px; border-radius: 2px; background: #374151; content: ''; }
.section-hint { color: #98a2b3; font-size: 12px; }
.platform-sku-editor-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px 20px; }
.platform-sku-editor { min-width: 0; display: flex; flex-direction: column; gap: 8px; }
.platform-sku-input-row { display: flex; align-items: center; gap: 6px; }
.platform-sku-input-row .field { min-width: 0; }
.platform-sku-readonly { color: #98a2b3; font-size: 12px; }
.add-platform-sku { min-height: 28px; display: inline-flex; align-items: center; align-self: flex-start; gap: 4px; border: 0; padding: 0; color: #475467; font-size: 12px; }
.add-platform-sku:hover { color: #111827; text-decoration: underline; text-underline-offset: 3px; }
.sku-detail { display: flex; flex-direction: column; gap: 22px; }
.sku-detail-hero { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; padding-bottom: 18px; border-bottom: 1px solid #e5e7eb; }
.detail-kicker { color: #667085; font-size: 12px; line-height: 18px; }
.sku-detail-hero h2 { margin-top: 4px; color: #111827; font-size: 22px; line-height: 28px; font-weight: 700; }
.sku-detail-hero p { margin-top: 3px; color: #667085; font-size: 13px; line-height: 20px; }
.sku-detail-section { display: flex; flex-direction: column; gap: 12px; }
.sku-detail-section h3 { color: #344054; font-size: 13px; line-height: 20px; font-weight: 700; }
.sku-detail-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px 20px; }
.sku-detail-grid div { min-width: 0; padding: 10px 12px; border-radius: 6px; background: #f6f7f9; }
.sku-detail-grid dt { color: #98a2b3; font-size: 11px; line-height: 16px; }
.sku-detail-grid dd { margin-top: 3px; overflow: hidden; color: #344054; font-size: 13px; line-height: 19px; text-overflow: ellipsis; white-space: nowrap; }
.sku-detail-platforms { display: flex; gap: 6px; flex-wrap: wrap; }
.editor-footer { position: sticky; bottom: -24px; z-index: 2; margin: 0 -24px -24px; padding: 14px 24px; display: flex; justify-content: flex-end; gap: 8px; border-top: 1px solid #e5e7eb; background: rgba(255,255,255,.96); backdrop-filter: blur(8px); }
.form-grid, .cert-files-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: start; gap: 18px 20px; }
.form-grid-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.form-grid-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.form-label { min-width: 0; display: flex; flex-direction: column; gap: 8px; color: #475467; }
.form-label-text { display: block; min-height: 18px; font-size: 12px; line-height: 18px; font-weight: 650; }
.required { margin-left: 3px; color: #dc2626; }
.custom-select { position: relative; }
.custom-select-button { width: 100%; height: 42px; padding: 0 12px 0 13px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #d5dae1; border-radius: 8px; background: #fff; box-shadow: 0 1px 2px rgba(16,24,40,.03); font-size: 14px; text-align: left; outline: none; transition: border-color .15s, box-shadow .15s; }
.custom-select-button:hover { border-color: #aeb6c2; }
.custom-select-button:focus { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1), 0 1px 2px rgba(16,24,40,.04); }
.custom-select-options { position: absolute; z-index: 30; top: calc(100% + 6px); width: 100%; max-height: 240px; overflow-y: auto; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; padding: 5px; box-shadow: 0 12px 28px rgba(16,24,40,.14), 0 2px 8px rgba(16,24,40,.06); outline: none; transform-origin: top; }
.custom-select-option { min-height: 36px; padding: 0 9px; display: flex; align-items: center; justify-content: space-between; gap: 10px; border-radius: 5px; font-size: 13px; cursor: pointer; }
.custom-select-option.active { background: #f1f3f5; }
.file-picker { position: relative; min-height: 96px; display: flex; align-items: center; justify-content: center; gap: 10px; overflow: hidden; border: 1px dashed #bdc5d0; border-radius: 8px; background: #fafbfc; padding: 18px; color: #475467; font-size: 13px; cursor: pointer; transition: border-color .15s, background-color .15s, box-shadow .15s; }
.file-picker:hover { border-color: #667085; background: #f6f7f9; box-shadow: inset 0 0 0 1px rgba(102,112,133,.06); }
.file-picker-input { position: absolute; inset: 0; width: 100%; height: 100%; cursor: pointer; opacity: 0; }
.delete-confirm-button { min-height: 36px; display: inline-flex; align-items: center; justify-content: center; border-radius: 6px; background: #dc2626; padding: 0 16px; color: #fff; font-size: 14px; font-weight: 600; transition: background-color .15s; }
.delete-confirm-button:hover:not(:disabled) { background: #b91c1c; }
.delete-confirm-button:disabled { cursor: not-allowed; opacity: .5; }
@media (max-width: 768px) {
  .catalog-page { padding: 16px 12px; }
  .page-header { min-height: 0; margin-bottom: 14px; align-items: flex-start; flex-direction: column; }
  .header-actions { width: 100%; flex-wrap: wrap; }
  .page-heading h1 { font-size: 21px; line-height: 28px; }
  .catalog-controls { align-items: stretch; }
  .catalog-control-actions { flex: 0 0 auto; }
  .toolbar-search { width: 100%; }
  .summary-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .summary-item:nth-child(2) { border-right: 0; }
  .summary-item:nth-child(-n+2) { border-bottom: 1px solid #e8edf4; }
  .pager { align-items: flex-start; flex-direction: column; }
  .form-grid, .form-grid-3, .form-grid-4, .cert-files-grid, .platform-sku-editor-grid { grid-template-columns: 1fr; }
  .sku-detail-grid { grid-template-columns: 1fr; }
  .cert-table-row { gap: 10px; padding: 14px; }
}
</style>
