<template>
  <MainLayout>
    <div class="suite-page">
      <main class="conversation-shell">
        <div class="workspace-layout" :class="{ 'inspector-collapsed': inspectorCollapsed }">
          <aside class="suite-sidebar" aria-label="对话记录">
            <div class="sidebar-brand">
              <span class="sidebar-brand-mark"><SparklesIcon /></span>
              <div>
                <strong>套图助手</strong>
                <small>Image studio</small>
              </div>
            </div>
            <button type="button" class="sidebar-new-chat" @click="startNewConversation">
              <PlusIcon />
              <span>新建对话</span>
            </button>

            <div class="sidebar-section sidebar-history">
              <span class="sidebar-label">我的操作记录</span>
              <div v-if="sidebarConversationRecords.length" class="sidebar-history-list">
                <div v-for="record in sidebarConversationRecords" :key="record.session_id" class="sidebar-history-item" :class="{ active: record.session_id === conversation.session_id }">
                  <button type="button" class="sidebar-history-open" @click="openConversation(record)">
                    <span class="sidebar-history-copy">
                      <strong>{{ record.title }}</strong>
                      <small>{{ record.preview }}</small>
                    </span>
                  </button>
                  <button type="button" class="sidebar-history-delete" title="删除对话" aria-label="删除对话" :disabled="deletingConversation === record.session_id" @click="deleteConversation(record)">
                    <TrashIcon v-if="deletingConversation !== record.session_id" />
                    <ArrowPathIcon v-else class="spinning" />
                  </button>
                </div>
              </div>
              <p v-else class="sidebar-history-empty">暂无对话记录</p>
            </div>
            <RouterLink v-if="isAdmin" class="admin-history-trigger" to="/smart-suite/team"><DocumentTextIcon /><span>团队操作记录</span><ChevronRightIcon /></RouterLink>

          </aside>

          <section class="workspace-center">
            <section ref="chatMessagesEl" class="conversation" aria-live="polite">
          <section v-if="showEntryActions" class="entry-choice-panel" aria-label="选择任务类型">
            <div class="entry-choice-copy">
              <span class="toolbar-kicker">开始一个新任务</span>
              <h1>你今天想完成什么？</h1>
              <p>选择方向后直接填写对应表单。</p>
            </div>
            <div class="entry-choice-grid">
              <button v-for="option in entryModeOptions" :key="option.id" type="button" class="entry-choice-button" @click="chooseEntryMode(option)">
                <span class="entry-choice-icon"><component :is="option.icon" /></span>
                <span><strong>{{ option.label }}</strong><small>{{ option.description }}</small></span>
                <ChevronRightIcon />
              </button>
            </div>
          </section>
          <Teleport to="body">
          <div v-if="showRequirementForm" class="requirement-modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="requirement-form-title" @click.self="closeRequirementForm">
          <form class="requirement-form requirement-modal-panel" :class="{ 'listing-only-form': initialForm.requestType === 'listing' }" @submit.prevent="submitInitialRequirements">
            <div class="requirement-form-header">
              <div>
                <span class="toolbar-kicker">{{ initialForm.requestType === 'listing_evaluation' ? '评估现有 Listing' : initialForm.requestType === 'listing' ? '新建 Listing 任务' : initialForm.requestType === 'suite_and_listing' ? '新建套图 + Listing 任务' : '新建套图任务' }}</span>
                <h1 id="requirement-form-title">{{ initialForm.requestType === 'listing_evaluation' ? '按平台标准检查现有文案' : initialForm.requestType === 'listing' ? '把商品信息整理成可发布文案' : initialForm.requestType === 'suite_and_listing' ? '一次完成套图和商品文案' : '一次填写，开始制作套图' }}</h1>
                <p>{{ initialForm.requestType === 'listing_evaluation' ? '填写现有 Listing 与产品事实，平台规则和关键词库由系统自动加载。' : initialForm.requestType === 'listing' ? '填写产品事实、平台和站点，按平台字段生成商品文案。' : initialForm.requestType === 'suite_and_listing' ? '同一份产品事实会同步用于套图策划和 Listing，避免两边说法不一致。' : '只确认图片所需资料，后续可以继续在对话里调整。' }}</p>
              </div>
              <div class="requirement-form-header-actions">
                <SparklesIcon aria-hidden="true" />
                <button type="button" class="requirement-form-close" title="关闭需求表单" aria-label="关闭需求表单" @click="closeRequirementForm"><XMarkIcon /></button>
              </div>
            </div>

            <p v-if="initialFormError" class="requirement-form-error" role="alert">{{ initialFormError }}</p>

            <section v-if="initialForm.requestType === 'listing'" class="requirement-form-section listing-platform-first">
              <div class="requirement-section-heading">
                <div><strong>先选择 Listing 平台</strong><span>平台决定最终输出字段、文案顺序和适用规则</span></div>
                <span class="requirement-step">01</span>
              </div>
              <div class="preset-grid preset-grid-platform">
                <button v-for="preset in visiblePlatformPresets" :key="'listing-' + preset.id" type="button" class="preset-button" :class="{ selected: initialForm.platformPreset === preset.id }" @click="selectPlatformPreset(preset)">
                  <strong>{{ preset.label }}</strong><small>{{ preset.description }}</small>
                </button>
              </div>
              <div v-if="['custom', 'amazon_custom'].includes(initialForm.platformPreset)" class="requirement-inline-fields requirement-custom-market">
                <input v-if="initialForm.platformPreset === 'custom'" v-model.trim="initialForm.platformName" type="text" placeholder="平台名称">
                <input v-model.trim="initialForm.countryName" type="text" placeholder="国家 / 站点">
                <input v-model.trim="initialForm.countryCode" type="text" maxlength="2" placeholder="国家代码">
              </div>
              <span v-if="initialFormErrors.platform" class="field-error">{{ initialFormErrors.platform }}</span>
            </section>

            <section v-if="initialForm.requestType === 'listing_evaluation'" class="requirement-form-section evaluation-platform-section">
              <div class="requirement-section-heading">
                <div><strong>先选择评估平台</strong><span>选择后只展示该平台对应的 Listing 字段</span></div>
                <span class="requirement-step">01</span>
              </div>
              <div class="preset-grid preset-grid-platform">
                <button v-for="preset in evaluationPlatformPresets" :key="'evaluation-' + preset.id" type="button" class="preset-button" :class="{ selected: initialForm.platformPreset === preset.id }" @click="selectPlatformPreset(preset)">
                  <strong>{{ preset.label }}</strong><small>{{ preset.description }}</small>
                </button>
              </div>
              <span v-if="initialFormErrors.platform" class="field-error">{{ initialFormErrors.platform }}</span>
              <div v-if="['custom', 'amazon_custom'].includes(initialForm.platformPreset)" class="requirement-inline-fields requirement-custom-market">
                <input v-if="initialForm.platformPreset === 'custom'" v-model.trim="initialForm.platformName" type="text" placeholder="平台名称">
                <input v-model.trim="initialForm.countryName" type="text" placeholder="国家 / 站点">
                <input v-model.trim="initialForm.countryCode" type="text" maxlength="2" placeholder="国家代码">
              </div>
            </section>

            <section class="requirement-form-section">
              <div class="requirement-section-heading">
                <div><strong>{{ initialForm.requestType === 'listing' ? 'Listing 产品资料' : initialForm.requestType === 'listing_evaluation' ? 'Listing 产品事实' : '产品资料' }}</strong><span>{{ initialForm.requestType === 'listing' ? '填写用于标题、卖点、描述和搜索词的产品事实' : initialForm.requestType === 'listing_evaluation' ? '用于核对文案中的产品名称、规格、材质和功能事实' : '产品图和事实是整套视觉一致性的基础' }}</span></div>
                <span class="requirement-step">{{ initialForm.requestType === 'listing_evaluation' ? '02' : initialForm.requestType === 'listing' ? '02' : '01' }}</span>
              </div>
              <div class="requirement-form-grid requirement-form-grid-product" :class="{ 'listing-product-grid': initialForm.requestType !== 'suite' }">
                <div v-if="['listing', 'suite', 'suite_and_listing'].includes(initialForm.requestType) || (initialForm.requestType === 'listing_evaluation' && initialForm.platform === 'lowes')" class="requirement-field requirement-image-field">
                  <span class="requirement-label">产品图片 <em>必填</em><small>{{ ['listing', 'listing_evaluation'].includes(initialForm.requestType) ? '仅支持 1 张，用于识别产品差异和适用场景' : '最多 6 张，用于识别产品差异' }}</small></span>
                  <input ref="onboardingFileInput" type="file" accept="image/png,image/jpeg,image/webp" :multiple="!['listing', 'listing_evaluation'].includes(initialForm.requestType)" hidden @change="handleInitialImageChange">
                  <div class="product-image-list">
                    <article v-for="(image, index) in initialProductImages" :key="image.id" class="product-image-item" :class="{ primary: index === 0 }">
                      <button type="button" class="product-image-preview" :title="index === 0 ? '产品主图' : '设为产品主图'" @click="setInitialPrimaryImage(index)">
                        <img :src="image.previewUrl" :alt="index === 0 ? '产品主图预览' : `产品补充图 ${index}`">
                        <span>{{ index === 0 ? '主图' : `细节 ${index}` }}</span>
                      </button>
                      <button type="button" class="product-image-remove" title="移除图片" aria-label="移除图片" @click="removeInitialProductImage(index)"><XMarkIcon /></button>
                    </article>
                    <button v-if="initialProductImages.length < maxInitialProductImages" type="button" class="requirement-upload" @click="onboardingFileInput?.click()">
                      <PhotoIcon />
                      <strong>{{ initialProductImages.length ? '添加细节图' : '上传产品图' }}</strong>
                      <small>PNG / JPG / WEBP</small>
                    </button>
                  </div>
                  <small class="product-image-help">首张图锁定产品外观；其他角度和细节图用于补充结构、材质与部件信息，避免相似产品生成相同文案。</small>
                  <span v-if="initialFormErrors.image" class="field-error">{{ initialFormErrors.image }}</span>
                </div>
                <div class="requirement-form-stack">
                  <label class="requirement-field">
                    <span class="requirement-label">产品名称 <em>必填</em></span>
                    <input v-model.trim="initialForm.productName" type="text" placeholder="例如：3-Hole Widespread Bathroom Faucet">
                    <span v-if="initialFormErrors.productName" class="field-error">{{ initialFormErrors.productName }}</span>
                  </label>
                  <div class="requirement-field">
                    <span class="requirement-label">品类 / 品牌 <small>可选</small></span>
                    <div class="requirement-inline-fields">
                      <input v-model.trim="initialForm.productCategory" type="text" placeholder="产品品类">
                      <Listbox v-model="initialForm.productBrand">
                        <div class="requirement-select">
                          <ListboxButton class="requirement-select-button" aria-label="选择产品品牌">
                            <span :class="{ placeholder: !initialForm.productBrand }">{{ initialForm.productBrand || '选择品牌' }}</span>
                            <ChevronUpDownIcon />
                          </ListboxButton>
                          <transition enter-active-class="select-transition" enter-from-class="select-transition-from" leave-active-class="select-transition" leave-to-class="select-transition-from">
                            <ListboxOptions class="requirement-select-options">
                              <ListboxOption v-slot="{ active, selected }" as="template" value="">
                                <li class="requirement-select-option" :class="{ active }"><span>不指定品牌</span><CheckIcon v-if="selected" /></li>
                              </ListboxOption>
                              <ListboxOption v-for="brand in brandOptions" :key="brand" v-slot="{ active, selected }" as="template" :value="brand">
                                <li class="requirement-select-option" :class="{ active, selected }"><span>{{ brand }}</span><CheckIcon v-if="selected" /></li>
                              </ListboxOption>
                            </ListboxOptions>
                          </transition>
                        </div>
                      </Listbox>
                    </div>
                  </div>
                </div>
              </div>
              <label class="requirement-field requirement-field-wide">
                <span class="requirement-label">产品信息 <em>必填</em><small>尺寸、材质、结构、功能、适配、包装、认证和保修等已确认事实都可以粘贴</small></span>
                <textarea v-model.trim="initialForm.productFeatures" rows="7" placeholder="粘贴标题、卖点、尺寸、材质、功能、包装清单、测试结果、认证和保修信息。AI 只会使用这些已提供事实，不会把推断写成产品事实。"></textarea>
                <span v-if="initialFormErrors.productFeatures" class="field-error">{{ initialFormErrors.productFeatures }}</span>
              </label>
              <label v-if="initialForm.requestType === 'listing'" class="requirement-field requirement-field-wide">
                <span class="requirement-label">运营指定关键词 <small>可选</small><small>仅在有必须覆盖的搜索词时填写</small></span>
                <textarea v-model.trim="initialForm.listingKeywords" rows="3" placeholder="多个关键词用逗号或换行分隔；不填写时自动使用系统关键词库"></textarea>
              </label>
            </section>

            <section v-if="initialForm.requestType === 'listing_evaluation'" class="requirement-form-section platform-listing-fields">
              <div class="requirement-section-heading">
                <div><strong>{{ (initialForm.platformName || '平台') + ' Listing 字段' }}</strong><span>粘贴现有 Listing，系统会按平台字段、产品事实和本土化规则逐项检查</span></div>
                <span class="requirement-step">{{ initialForm.platform === 'home_depot' ? '02' : '03' }}</span>
              </div>
              <template v-if="initialForm.platform === 'home_depot'">
                <label class="requirement-field requirement-field-wide">
                  <span class="requirement-label">商品标题 <em>必填</em></span>
                  <input v-model.trim="initialForm.listingTitle" type="text" placeholder="粘贴 Home Depot 商品标题">
                </label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Salient Bullets <em>必填</em><small>每条单独一行，粘贴当前 Listing 的 3 条首屏卖点</small></span><textarea v-model.trim="initialForm.listingSalientBullets" rows="4" placeholder="每行一条 Salient Bullet"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Bullet Points <em>必填</em><small>每条单独一行，粘贴当前 Listing 的 6 条卖点</small></span><textarea v-model.trim="initialForm.listingBullets" rows="7" placeholder="每行一条 Bullet Point"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">About This Product <em>必填</em></span><textarea v-model.trim="initialForm.listingDescription" rows="7" placeholder="粘贴当前 Listing 的 About This Product"></textarea></label>
              </template>
              <template v-else-if="initialForm.platform === 'amazon'">
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">商品标题 <em>必填</em></span><input v-model.trim="initialForm.listingTitle" type="text" placeholder="粘贴 Amazon 当前标题"></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">五点卖点 <em>必填</em><small>每条单独一行</small></span><textarea v-model.trim="initialForm.listingBullets" rows="7" placeholder="每行一条卖点"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">商品描述 <em>必填</em></span><textarea v-model.trim="initialForm.listingDescription" rows="6" placeholder="粘贴 Amazon 商品描述"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">搜索词 <small>可选</small></span><textarea v-model.trim="initialForm.listingSearchTerms" rows="3" placeholder="粘贴后台搜索词"></textarea></label>
              </template>
              <template v-else-if="initialForm.platform === 'tiktok'">
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">商品标题 <em>必填</em></span><input v-model.trim="initialForm.listingTitle" type="text" placeholder="粘贴 TikTok Shop 商品标题"></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">商品描述 <em>必填</em></span><textarea v-model.trim="initialForm.listingDescription" rows="6" placeholder="粘贴短视频电商商品描述"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">产品属性 <small>可选</small></span><textarea v-model.trim="initialForm.tiktokAttributes" rows="4" placeholder="每行一个属性，例如 Color: Black"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">变体信息 <small>可选</small></span><textarea v-model.trim="initialForm.tiktokVariants" rows="4" placeholder="粘贴规格、颜色或套装变体"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">包装内容 <small>可选</small></span><textarea v-model.trim="initialForm.tiktokPackageContents" rows="3" placeholder="粘贴 TikTok Shop 包装清单"></textarea></label>
              </template>
              <template v-else-if="initialForm.platform === 'lowes'">
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Title <em>必填</em></span><input v-model.trim="initialForm.listingTitle" type="text" placeholder="粘贴 Lowe’s Title"></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Description <em>必填</em></span><textarea v-model.trim="initialForm.listingDescription" rows="6" placeholder="粘贴 Lowe’s Description"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Bullets <em>必填</em><small>每条单独一行</small></span><textarea v-model.trim="initialForm.listingBullets" rows="6" placeholder="每行一条 Bullet"></textarea></label>
              </template>
              <template v-else-if="initialForm.platform === 'wayfair'">
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">About This Product <em>必填</em></span><textarea v-model.trim="initialForm.wayfairDescription" rows="6" placeholder="粘贴 Wayfair About This Product"></textarea></label>
                <label class="requirement-field requirement-field-wide"><span class="requirement-label">Feature Bullets <em>必填</em><small>每条单独一行</small></span><textarea v-model.trim="initialForm.wayfairFeatures" rows="6" placeholder="每行一条 Feature Bullet"></textarea></label>
              </template>
              <span v-if="initialFormErrors.existingListing" class="field-error">{{ initialFormErrors.existingListing }}</span>
            </section>

            <section v-if="initialForm.requestType !== 'listing_evaluation' && initialForm.requestType !== 'listing'" class="requirement-form-section">
              <div class="requirement-section-heading">
                <div><strong>{{ initialForm.requestType !== 'suite' ? '平台与产出' : '市场与产出' }}</strong><span>{{ initialForm.requestType !== 'suite' ? 'TikTok Shop、Home Depot、Lowe’s、Wayfair 和 Amazon 使用对应字段结构' : '先选常用预设，也可以切换为自定义' }}</span></div>
                <span class="requirement-step">{{ initialForm.requestType === 'listing' ? '03' : '02' }}</span>
              </div>
              <div class="requirement-field">
                <span class="requirement-label">平台站点 <em>必填</em></span>
                <div class="preset-grid preset-grid-platform">
                  <button v-for="preset in visiblePlatformPresets" :key="preset.id" type="button" class="preset-button" :class="{ selected: initialForm.platformPreset === preset.id }" @click="selectPlatformPreset(preset)">
                    <strong>{{ preset.label }}</strong><small>{{ preset.description }}</small>
                  </button>
                </div>
                <div v-if="['custom', 'amazon_custom'].includes(initialForm.platformPreset)" class="requirement-inline-fields requirement-custom-market" :class="{ 'amazon-custom-market': initialForm.platformPreset === 'amazon_custom' }">
                  <input v-if="initialForm.platformPreset === 'custom'" v-model.trim="initialForm.platformName" type="text" placeholder="平台名称">
                  <input v-model.trim="initialForm.countryName" type="text" placeholder="国家 / 站点">
                  <input v-model.trim="initialForm.countryCode" type="text" maxlength="2" placeholder="国家代码">
                </div>
                <span v-if="initialFormErrors.platform" class="field-error">{{ initialFormErrors.platform }}</span>
              </div>

              <div v-if="['suite', 'suite_and_listing'].includes(initialForm.requestType)" class="requirement-field">
                <span class="requirement-label">套图数量 <em>至少选择一种</em></span>
                <div class="output-option-list">
                  <div v-for="output in outputPresets" :key="output.key" class="output-option" :class="{ selected: initialForm[output.enabledKey] }">
                    <input v-model="initialForm[output.enabledKey]" type="checkbox" :aria-label="`生成${output.label}`">
                    <span class="output-option-copy"><strong>{{ output.label }}</strong><small>{{ output.description }}</small></span>
                    <input v-model.number="initialForm[output.countKey]" class="count-input" type="number" min="1" :max="output.max" :disabled="!initialForm[output.enabledKey]" :aria-label="`${output.label}数量`">
                  </div>
                </div>
                <span v-if="initialFormErrors.output" class="field-error">{{ initialFormErrors.output }}</span>
              </div>
            </section>

            <section v-if="['suite', 'suite_and_listing'].includes(initialForm.requestType)" class="requirement-form-section">
              <div class="requirement-section-heading">
                <div><strong>视觉与模型</strong><span>默认交给 AI 设计高级统一风格，只有你有明确偏好时再指定</span></div>
                <span class="requirement-step">{{ initialForm.requestType === 'suite_and_listing' ? '04' : '03' }}</span>
              </div>
              <div class="requirement-field">
                <span class="requirement-label">视觉要求 <small>可选</small></span>
                <div class="preset-grid preset-grid-style">
                  <button v-for="preset in visualPresets" :key="preset.id" type="button" class="preset-button" :class="{ selected: initialForm.visualPreset === preset.id }" @click="selectVisualPreset(preset)">
                    <strong>{{ preset.label }}</strong><small>{{ preset.description }}</small>
                  </button>
                </div>
                <textarea v-if="initialForm.visualPreset === 'custom'" v-model.trim="initialForm.styleRequirements" class="requirement-custom-textarea" rows="3" placeholder="补充光线、场景、色调、材质表现或需要避开的视觉元素"></textarea>
                <span v-if="initialFormErrors.visual" class="field-error">{{ initialFormErrors.visual }}</span>
              </div>
              <div class="requirement-field">
                <span class="requirement-label">生图模型 <em>必选</em></span>
                <div class="model-preset-list">
                  <button v-for="model in drawingModelOptions" :key="model.name" type="button" class="model-preset" :class="{ selected: initialForm.drawingModel === model.name }" @click="initialForm.drawingModel = model.name">
                    <span><strong>{{ model.name }}</strong><small>{{ model.description }}</small></span>
                    <em v-if="model.is_default">默认</em>
                  </button>
                </div>
                <span v-if="!drawingModelOptions.length" class="model-empty">当前没有可用的生图模型，请先在模型设置中启用。</span>
                <span v-if="initialFormErrors.model" class="field-error">{{ initialFormErrors.model }}</span>
              </div>
            </section>

            <div class="requirement-form-footer">
              <span>{{ initialForm.requestType === 'listing_evaluation' ? '提交后在当前页面查看评分、中文扣分说明和修改后的完整 Listing。' : initialForm.requestType === 'listing' ? '提交后会直接生成 Listing，之后仍可在对话里修改产品事实。' : initialForm.requestType === 'suite_and_listing' ? '提交后同步生成套图和 Listing，已有生成记录会保留。' : '提交后仍可在对话里修改需求，已有生成记录会保留。' }}</span>
              <button type="submit" class="requirement-submit" :disabled="submittingRequirements || (['suite', 'suite_and_listing'].includes(initialForm.requestType) && !drawingModelOptions.length)">
                <span>{{ submittingRequirements ? '正在提交' : initialForm.requestType === 'listing_evaluation' ? '提交并开始评估' : initialForm.requestType === 'listing' ? '提交并开始生成' : '保存需求并进入对话' }}</span>
              </button>
            </div>
          </form>
          </div>
          </Teleport>

          <div v-if="!showRequirementForm" class="conversation-body">
          <div v-if="submittingRequirements && initialForm.requestType === 'listing_evaluation'" class="message-row assistant processing-row" aria-live="polite">
            <span class="thinking-state shimmer">{{ initialForm.platform === 'home_depot' ? '正在获取 Home Depot Listing 并评估' : '正在评估 Listing' }}</span>
          </div>
          <div v-if="listingEvaluationError && !submittingRequirements" class="message-row assistant listing-evaluation-error-row" role="alert">
            <div class="message-bubble">
              <p>{{ listingEvaluationError }}</p>
              <button type="button" @click="requirementFormVisible = true">重新填写</button>
            </div>
          </div>
          <section v-if="listingEvaluationResult" class="listing-evaluation-result" aria-label="Listing 评分结果">
            <section v-if="listingEvaluationResult.listing_source" class="listing-source-preview" aria-label="抓取到的 Home Depot Listing">
              <div class="listing-source-preview-head">
                <div><span>Firecrawl 抓取结果</span><strong>Home Depot Listing</strong></div>
                <a :href="listingEvaluationResult.listing_source.canonical_url" target="_blank" rel="noopener noreferrer">查看商品页</a>
              </div>
              <dl class="listing-source-meta">
                <div><dt>Internet #</dt><dd>{{ listingEvaluationResult.listing_source.internet_number }}</dd></div>
                <div v-if="listingEvaluationResult.listing_source.brand"><dt>Brand</dt><dd>{{ listingEvaluationResult.listing_source.brand }}</dd></div>
                <div v-if="listingEvaluationResult.listing_source.model_number"><dt>Model #</dt><dd>{{ listingEvaluationResult.listing_source.model_number }}</dd></div>
                <div v-if="listingEvaluationResult.listing_source.product_category"><dt>Category</dt><dd>{{ listingEvaluationResult.listing_source.product_category }}</dd></div>
              </dl>
              <div class="listing-source-field">
                <strong>Title</strong>
                <p>{{ listingEvaluationResult.listing_source.title || listingEvaluationResult.listing_source.product_title }}</p>
              </div>
              <div v-if="listingEvaluationResult.listing_source.salient_bullets?.length" class="listing-source-field">
                <strong>Salient Bullets</strong>
                <ul><li v-for="(highlight, index) in listingEvaluationResult.listing_source.salient_bullets" :key="index">{{ highlight }}</li></ul>
              </div>
              <div v-if="listingEvaluationResult.listing_source.bullet_points?.length" class="listing-source-field">
                <strong>Bullet Points</strong>
                <ul><li v-for="(bullet, index) in listingEvaluationResult.listing_source.bullet_points" :key="`bullet-${index}`">{{ bullet }}</li></ul>
              </div>
              <div v-if="listingEvaluationResult.listing_source.about_this_product" class="listing-source-field">
                <strong>About This Product</strong>
                <p>{{ listingEvaluationResult.listing_source.about_this_product }}</p>
              </div>
            </section>
            <div class="listing-evaluation-head">
              <div><span>Listing 评估</span><strong>{{ listingEvaluationResult.total_score }} / 100</strong></div>
              <em :class="{ fail: listingEvaluationResult.needs_optimization }">{{ listingEvaluationResult.needs_optimization ? '需要优化' : '达到通过线' }}</em>
            </div>
            <p class="listing-evaluation-summary">{{ listingEvaluationResult.evaluation?.summary }}</p>
            <div class="listing-evaluation-table" :class="{ 'is-hd-rubric': listingEvaluationUsesRawPoints }">
              <div v-if="listingEvaluationUsesRawPoints" class="listing-evaluation-row listing-evaluation-header"><span>评分项</span><span>满分</span><span>得分</span></div>
              <div v-else class="listing-evaluation-row listing-evaluation-header"><span>评估维度</span><span>占比</span><span>得分</span><span>加权分</span></div>
              <div v-for="dimension in listingEvaluationResult.evaluation?.dimensions || []" :key="dimension.key" class="listing-evaluation-row"><span><strong>{{ dimension.label }}</strong><small>{{ dimension.comment }}</small></span><template v-if="listingEvaluationUsesRawPoints"><span>{{ dimension.weight }}</span><span>{{ dimension.weighted_score }}</span></template><template v-else><span>{{ dimension.weight }}%</span><span>{{ dimension.score }}</span><span>{{ dimension.weighted_score }}</span></template></div>
            </div>
            <div v-if="listingEvaluationResult.evaluation?.deductions?.length" class="listing-evaluation-deductions">
              <strong>扣分点</strong>
              <div v-for="(deduction, index) in listingEvaluationResult.evaluation.deductions" :key="index"><span>{{ deduction.issue }}</span><em>-{{ deduction.points }} 分</em><small class="deduction-plan"><strong>优化方案</strong>{{ deduction.optimization_plan || deduction.suggestion }}</small></div>
            </div>
            <div v-if="listingEvaluationResult.evaluation?.recommendations?.length" class="listing-evaluation-recommendations"><strong>优化执行顺序</strong><p v-for="(item, index) in listingEvaluationResult.evaluation.recommendations" :key="index">{{ item }}</p></div>
            <section v-if="optimizedListingSections.length" class="listing-optimized-result" aria-label="优化后的 Listing">
              <div class="listing-optimized-head"><span>优化结果</span><strong>修改后的 {{ listingEvaluationPlatformName }} Listing</strong></div>
              <div v-for="section in optimizedListingSections" :key="section.key" class="listing-optimized-field">
                <strong>{{ section.label }}</strong>
                <ul v-if="section.items?.length"><li v-for="(item, index) in section.items" :key="`${section.key}-${index}`">{{ item }}</li></ul>
                <p v-else>{{ section.text }}</p>
              </div>
            </section>
          </section>
          <div v-if="!showEntryActions && !listingEvaluationResult && !timelineItems.length && !thinking" class="conversation-empty">
            <div class="conversation-empty-icon"><SparklesIcon /></div>
            <h2>开始一段套图对话</h2>
            <p>告诉我产品、平台和需要的图片类型，或者直接上传产品图。</p>
          </div>

          <div v-for="item in timelineItems" :key="item.key" class="conversation-timeline">
            <div v-if="item.type === 'message'" class="message-row" :class="item.data.role" :data-message-id="item.data.id || item.key">
              <div class="message-bubble">
                <img v-if="item.data.image_url" :src="messageImageUrl(item.data.image_url)" alt="产品参考图" class="uploaded-image">
                <p v-if="item.data.text">{{ item.data.text }}</p>
                <div
                  v-if="item.data.role === 'assistant' && item.data.id === latestAssistantMessageId && isCollectingModel && drawingModelOptions.length"
                  class="model-suggestions"
                >
                  <div class="model-choice-heading">
                    <strong>选择生图模型</strong>
                    <span>整套生成和后续对话修改都会使用这个模型。</span>
                  </div>
                  <button
                    v-for="model in drawingModelOptions"
                    :key="model.name"
                    type="button"
                    class="model-choice"
                    :disabled="thinking || uploading || startingTask || candidateActionLoading"
                    @click="chooseDrawingModel(model)"
                  >
                    <span class="model-choice-copy">
                      <strong>{{ model.name }}</strong>
                      <small>{{ model.description }}</small>
                    </span>
                    <em v-if="model.is_default">默认</em>
                  </button>
                </div>
                <div
                  v-if="item.data.role === 'assistant' && item.data.id === latestAssistantMessageId && suggestionOptions.length"
                  class="suggestion-row"
                  :class="{ 'style-suggestions': isCollectingStyle }"
                >
                  <div v-if="isCollectingStyle" class="style-choice-heading">
                    <strong>选择整套图的视觉方向</strong>
                    <span>选一个接近的方向，后续可以继续用文字细调。</span>
                  </div>
                  <button
                    v-for="suggestion in suggestionOptions"
                    :key="suggestion"
                    type="button"
                    class="suggestion-button"
                    :class="{ 'style-suggestion': isCollectingStyle }"
                    :disabled="thinking || uploading || startingTask || candidateActionLoading"
                    @click="chooseSuggestion(suggestion)"
                  >
                    <component :is="suggestion === '上传风格参考图' ? PaperClipIcon : PhotoIcon" v-if="isCollectingStyle" />
                    <span v-if="isCollectingStyle" class="suggestion-copy">
                      <strong>{{ suggestion }}</strong>
                      <small>{{ styleSuggestionDescription(suggestion) }}</small>
                    </span>
                    <template v-else>{{ suggestion }}</template>
                  </button>
                </div>
              </div>
            </div>

            <div v-else-if="item.type === 'task'" class="message-row assistant task-row">
              <div class="task-message">
                <div class="task-todo todo">
                  <div class="task-todo-header">
                    <button
                      type="button"
                      class="task-todo-toggle todoHead"
                      :aria-expanded="!taskTodoCollapsed"
                      aria-label="展开或收起套图生成任务"
                      @click="taskTodoCollapsed = !taskTodoCollapsed"
                    >
                      <span class="task-todo-head-icon todoHeadIcon" aria-hidden="true">
                        <svg v-if="taskTodoAllDone" class="task-todo-head-check todoHeadCheck" viewBox="0 0 24 24">
                          <circle cx="12" cy="12" r="10" />
                          <path d="m7.8 12.2 2.7 2.7 5.8-6" />
                        </svg>
                        <span v-else-if="isTaskActive" class="task-todo-progress todoHeadPie" :style="{ '--task-progress': taskProgressPercent, '--todo-pie': `${taskProgressPercent}%` }">
                          <svg class="todoHeadPieRing" viewBox="0 0 24 24">
                            <circle class="progress-base" cx="12" cy="12" r="9" pathLength="100" />
                            <circle class="progress-value" cx="12" cy="12" r="9" pathLength="100" />
                          </svg>
                        </span>
                        <XCircleIcon v-else-if="taskHasError" class="task-todo-failed-icon" aria-hidden="true" />
                        <svg v-else class="task-todo-list-icon todoListIcon" viewBox="0 0 24 24">
                          <path d="M9 6.5h11M9 12h11M9 17.5h11M4 6.5h.01M4 12h.01M4 17.5h.01" />
                        </svg>
                        <ChevronUpIcon class="task-todo-chevron todoChevron" :class="{ collapsed: taskTodoCollapsed }" />
                      </span>
                      <span class="task-todo-title todoTitle">
                        <strong>{{ taskStateTitle }}</strong>
                        <small>{{ taskProgressText }}</small>
                        <span class="task-status-badge" :class="taskStatusToneClass(currentTask?.status)">
                          <i class="task-status-dot" :class="{ pulse: isTaskActive }"></i>
                          {{ taskStatusLabel(currentTask?.status) }}
                        </span>
                      </span>
                      <span :key="taskTodoCompletedCount" class="task-todo-count todoCount">{{ taskTodoCompletedCount }}/{{ taskTodoTotalCount }}</span>
                    </button>
                  </div>

                  <div class="task-todo-collapsible todoCollapsible" :class="{ collapsed: taskTodoCollapsed, isCollapsed: taskTodoCollapsed }">
                    <div class="task-todo-inner todoInner">
                      <ol v-if="taskTodoItems.length" class="task-todo-list todoList">
                        <li
                          v-for="(todo, index) in taskTodoItems"
                          :key="todo.id"
                          class="task-todo-item todoItem"
                          :style="{ '--i': index }"
                          :class="taskTodoItemClass(todo)"
                        >
                          <span class="task-todo-item-icon todoIconWrap" aria-hidden="true">
                            <svg class="todo-icon todoIcon todo-pending" :class="{ on: todo.status === 'pending' }" viewBox="0 0 24 24">
                              <circle cx="12" cy="12" r="9" />
                            </svg>
                            <svg class="todo-icon todoIcon todo-current" :class="{ on: todo.status === 'active', strong: todo.status === 'active' }" viewBox="0 0 24 24">
                              <circle cx="12" cy="12" r="9" />
                              <path d="m11 8.5 3.5 3.5-3.5 3.5M7.5 12h7" />
                            </svg>
                            <svg class="todo-icon todoIcon todo-done" :class="{ on: todo.status === 'done' }" viewBox="0 0 24 24">
                              <circle cx="12" cy="12" r="9" />
                              <path d="m8.5 12.2 2.2 2.2 4.8-5" />
                            </svg>
                            <XCircleIcon class="todo-icon todo-failed" :class="{ on: todo.status === 'failed' }" />
                            <MinusCircleIcon class="todo-icon todo-skipped" :class="{ on: todo.status === 'skipped' }" />
                          </span>
                          <span class="task-todo-item-copy">
                            <span class="todoLabel" :data-label="todo.label">{{ todo.label }}</span>
                            <small v-if="todo.detail">{{ todo.detail }}</small>
                            <div v-if="todo.keywords?.length" class="todo-keywords" aria-label="SEO 候选词">
                              <span
                                v-for="keyword in todo.keywords"
                                :key="keyword.term"
                                class="todo-keyword"
                                :title="keyword.reason || (keyword.tags || []).join(' / ')"
                              >{{ keyword.term }}</span>
                            </div>
                          </span>
                        </li>
                      </ol>
                    </div>
                  </div>
                </div>

                <section v-if="keywordCandidatesForTask(item.data).length" class="listing-keyword-candidates" aria-label="SEO 候选词">
                  <div class="listing-keyword-candidates-head">
                    <strong>SEO 候选词</strong>
                    <span>{{ keywordCandidatesForTask(item.data).length }} 个</span>
                  </div>
                  <div class="listing-keyword-candidate-list">
                    <article
                      v-for="keyword in keywordCandidatesForTask(item.data)"
                      :key="keyword.term"
                      class="listing-keyword-candidate"
                    >
                      <strong>{{ keyword.term }}</strong>
                      <div class="listing-keyword-meta">
                        <span>来源：{{ keywordSourceLabel(keyword.source) }}</span>
                        <span v-if="keyword.tags?.length">类型：{{ keyword.tags.join('、') }}</span>
                      </div>
                      <small v-if="keyword.reason" class="listing-keyword-reason">理由：{{ keyword.reason }}</small>
                      <div v-if="keyword.evidence_urls?.length" class="listing-keyword-evidence">
                        <a v-for="(url, urlIndex) in keyword.evidence_urls" :key="url" :href="url" target="_blank" rel="noreferrer">依据{{ urlIndex + 1 }}</a>
                      </div>
                    </article>
                  </div>
                </section>

				<section v-if="listingForTask(item.data) && !listingIncompleteForTask(item.data)" class="listing-result" aria-label="Listing 文案">
					<div class="listing-result-head">
						<div><span>Listing 文案</span><strong>{{ listingPlatformLabel(listingForTask(item.data)) }} · {{ listingLocaleLabel(listingForTask(item.data)) }}</strong><small>{{ item.data.workflow_type === 'smart_listing' ? '按平台字段生成' : '已与套图事实同步' }}</small></div>
					<!-- <button type="button" :title="copiedListingTaskId === item.data.task_id ? '已复制' : '复制全部 Listing'" @click="copyListing(item.data)">
					  <CheckIcon v-if="copiedListingTaskId === item.data.task_id" />
					  <ClipboardDocumentIcon v-else />
					  {{ copiedListingTaskId === item.data.task_id ? '已复制' : '复制全部' }}
					</button> -->
				  </div>
				  <div v-for="section in listingSections(listingForTask(item.data))" :key="section.key" class="listing-field" :class="{ 'listing-title-field': section.key === 'title', 'listing-terms': section.type === 'terms' }">
                    <span>{{ section.label }}</span>
                    <p v-if="section.type === 'text'">{{ section.value }}</p>
                    <ol v-else-if="section.type === 'list'"><li v-for="(value, index) in section.value" :key="section.key + '-' + index">{{ value }}</li></ol>
                    <div v-else-if="section.type === 'terms'"><em v-for="value in section.value" :key="value">{{ value }}</em></div>
                    <div v-if="section.reason || section.platformBasis || section.writingDecision || section.expectedImpact || section.evidence.length" class="listing-reason">
                      <strong>为什么这样写</strong>
                      <span v-if="section.platformBasis">平台决策依据：{{ section.platformBasis }}</span>
                      <span v-if="section.evidence.length">产品事实依据：{{ section.evidence.join('；') }}</span>
                      <span v-if="section.writingDecision">具体写法：{{ section.writingDecision }}</span>
                      <span v-if="section.expectedImpact">预期作用：{{ section.expectedImpact }}</span>
                      <span v-if="!section.platformBasis && section.reason">{{ section.reason }}</span>
                    </div>
                  </div>
				</section>
				<p v-if="listingForTask(item.data) && listingIncompleteForTask(item.data)" class="listing-result-error">Listing 内容不完整，任务未成功完成：{{ listingErrorForTask(item.data) || '缺少平台要求的字段' }}</p>
				<p v-else-if="listingErrorForTask(item.data)" class="listing-result-error">Listing 暂未生成：{{ listingErrorForTask(item.data) }}</p>
                <div v-if="isTaskActive && currentTask?.current_action" class="agent-live-status" aria-live="polite">
                  <ArrowPathIcon class="spinning" />
                  <span>{{ currentTask.current_action }}</span>
                </div>

              </div>
            </div>

            <div v-else-if="item.type === 'image_group'" class="message-row assistant task-row image-group-row">
              <div class="task-message">
                <div class="image-group">
                  <div class="group-heading">
                    <div><strong>{{ item.data.label }}</strong><span>{{ item.data.ratio }}</span></div>
                    <span class="group-count">{{ item.data.completed }}/{{ item.data.images.length }} 张</span>
                  </div>

                  <div class="image-grid" :class="`grid-${item.data.kind}`">
                    <article
                      v-for="image in item.data.images"
                      :key="image.id"
                      class="image-item"
                      :class="{ selected: selectedImage?.id === image.id, pending: !image.image_url, corrected: image.review_status === 'auto_corrected', 'reviewing-card': image.review_status === 'reviewing', 'correcting-card': image.review_status === 'correcting' }"
                    >
                      <div v-if="isImageGenerationPending(image)" class="image-generation-card" :class="`generation-${item.data.kind}`" role="img" aria-label="Generating image">
                        <div class="ig-canvas" aria-hidden="true">
                          <span class="ig-dots"></span>
                          <span class="ig-glow"></span>
                        </div>
                        <div class="ig-meta">
                          <span class="ig-label">Generating image</span>
                          <span class="ig-prompt">&ldquo;{{ imageGenerationPrompt(image) }}&rdquo;</span>
                        </div>
                      </div>
                      <template v-else>
                        <button type="button" class="image-select" :disabled="!image.image_url" @click="openImagePreview(image, item.data)">
                          <span class="image-frame">
                            <img v-if="image.image_url" :src="resolveImageUrl(currentImageUrl(image))" :alt="image.title || `${item.data.label} ${image.image_index}`">
                            <span v-else class="image-skeleton"><PhotoIcon /></span>
                            <span class="image-number">{{ String(image.image_index).padStart(2, '0') }}</span>
                            <span v-if="image.status !== 'success'" class="image-status">{{ imageStatusText(image) }}</span>
                            <span v-if="image.review_status" class="review-badge" :class="image.review_status === 'correcting' ? 'correcting' : image.review_status === 'auto_corrected' ? 'corrected' : image.review_status">
                              {{ imageReviewStatusText(image.review_status) }}
                            </span>
                          </span>
                          <span class="image-copy">
                            <strong>{{ image.title || `${item.data.label} ${image.image_index}` }}</strong>
                            <span>{{ image.role || pendingImageCopy(image) }}</span>
                          </span>
                        </button>

                        <div v-if="image.image_url" class="image-actions">
                          <button v-if="image.review_source_image_url && !(image.review_status === 'auto_corrected' && image.candidate_image_url)" type="button" @click="openImagePreview(image, item.data, image.review_source_image_url, '审核前版本')"><PhotoIcon />审核前版本</button>
                          <button type="button" @click="openImagePreview(image, item.data)"><ArrowsPointingOutIcon />预览大图</button>
                          <button v-if="image.review_status === 'auto_corrected' && image.review_source_image_url && image.candidate_image_url" type="button" @click="openAuditComparison(image, item.data)"><ArrowsRightLeftIcon />查看对比审核前版本</button>
                          <button type="button" :disabled="isTaskActive" :title="isTaskActive ? '整套任务完成后即可修改' : '通过对话修改图片'" @click="startEditingImage(image, item.data)"><PencilSquareIcon />{{ isTaskActive ? '任务完成后可修改' : '对话修改' }}</button>
                        </div>
                      </template>

                      <div v-if="image.candidate_status && image.candidate_status !== 'auto_corrected'" class="candidate-inline">
                        <button v-if="image.candidate_status === 'ready' && image.candidate_image_url" type="button" class="audit-compare-button" @click="openImagePreview(image, item.data, image.candidate_image_url, '修改版本')">
                          <PhotoIcon />
                          <span><strong>查看修改版本</strong></span>
                        </button>
                        <div v-else class="candidate-state">
                          <ArrowPathIcon :class="{ spinning: image.candidate_status === 'generating' }" />
                          <span>{{ candidateStatusText(image) }}</span>
                        </div>
                        <p v-if="image.candidate_status === 'ready' && image.candidate_prompt">{{ image.candidate_prompt }}</p>
                        <div v-if="image.candidate_status === 'ready'" class="candidate-actions">
                          <button type="button" :disabled="candidateActionLoading" @click="sendImageCommand(image, item.data, '保留原图')">保留原图</button>
                          <button type="button" :disabled="candidateActionLoading" class="primary" @click="sendImageCommand(image, item.data, '采用这个版本')">
                            <CheckIcon />采用修改版本
                          </button>
                        </div>
                      </div>
                    </article>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="message-row assistant task-row task-history-row">
              <div class="task-message task-history-message">
                <div class="task-history-heading">
                  <div>
                    <strong>历史生图记录</strong>
                    <span>{{ formatTaskCreatedAt(item.data.created_at) }}</span>
                  </div>
                  <span class="task-status-badge" :class="taskStatusToneClass(item.data.status)">
                    <i class="task-status-dot"></i>{{ taskStatusLabel(item.data.status) }}
                  </span>
                </div>
                <div v-for="group in taskResultGroups(item.data)" :key="`${item.data.task_id}-${group.type}`" class="image-group">
                  <div class="group-heading">
                    <div><strong>{{ group.label }}</strong><span>{{ group.ratio }}</span></div>
                    <span>{{ group.completed }}/{{ group.images.length }} 张</span>
                  </div>
                  <div class="image-grid" :class="`grid-${group.kind}`">
                    <article v-for="image in group.images" :key="image.id" class="image-item history-image-item">
                      <button type="button" class="image-select" :disabled="!image.image_url" @click="openImagePreview(image, group)">
                        <span class="image-frame">
                          <img v-if="image.image_url" :src="resolveImageUrl(currentImageUrl(image))" :alt="image.title || `${group.label} ${image.image_index}`">
                          <span v-else class="image-skeleton"><PhotoIcon /></span>
                          <span class="image-number">{{ String(image.image_index).padStart(2, '0') }}</span>
                        </span>
                        <span class="image-copy">
                          <strong>{{ image.title || `${group.label} ${image.image_index}` }}</strong>
                          <span>{{ image.role || pendingImageCopy(image) }}</span>
                        </span>
                      </button>
                    </article>
                  </div>
                </div>
				<section v-if="listingForTask(item.data) && !listingIncompleteForTask(item.data)" class="listing-result listing-result-history" aria-label="历史 Listing 文案">
					<div class="listing-result-head">
						<div><span>Listing 文案</span><strong>{{ listingPlatformLabel(listingForTask(item.data)) }} · {{ listingLocaleLabel(listingForTask(item.data)) }}</strong><small>{{ item.data.workflow_type === 'smart_listing' ? '按平台字段生成' : '随该次套图保留' }}</small></div>
					<!-- <button type="button" title="复制全部 Listing" @click="copyListing(item.data)"><ClipboardDocumentIcon />复制全部</button> -->
				  </div>
				  <div v-for="section in listingSections(listingForTask(item.data))" :key="section.key" class="listing-field" :class="{ 'listing-title-field': section.key === 'title', 'listing-terms': section.type === 'terms' }">
                    <span>{{ section.label }}</span>
                    <p v-if="section.type === 'text'">{{ section.value }}</p>
                    <ol v-else-if="section.type === 'list'"><li v-for="(value, index) in section.value" :key="section.key + '-' + index">{{ value }}</li></ol>
                    <div v-else-if="section.type === 'terms'"><em v-for="value in section.value" :key="value">{{ value }}</em></div>
                    <div v-if="section.reason || section.platformBasis || section.writingDecision || section.expectedImpact || section.evidence.length" class="listing-reason">
                      <strong>为什么这样写</strong>
                      <span v-if="section.platformBasis">平台决策依据：{{ section.platformBasis }}</span>
                      <span v-if="section.evidence.length">产品事实依据：{{ section.evidence.join('；') }}</span>
                      <span v-if="section.writingDecision">具体写法：{{ section.writingDecision }}</span>
                      <span v-if="section.expectedImpact">预期作用：{{ section.expectedImpact }}</span>
                      <span v-if="!section.platformBasis && section.reason">{{ section.reason }}</span>
                    </div>
                  </div>
				</section>
				<p v-if="listingForTask(item.data) && listingIncompleteForTask(item.data)" class="listing-result-error">Listing 内容不完整，任务未成功完成：{{ listingErrorForTask(item.data) || '缺少平台要求的字段' }}</p>
			  </div>
            </div>
          </div>

          <div v-if="thinking" class="message-row assistant processing-row">
            <span class="thinking-state shimmer" aria-live="polite">Thinking</span>
          </div>
          </div>
        </section>

        <footer v-if="!showRequirementForm" class="composer-wrap">
          <div v-if="selectedImage" class="image-context">
            <div>
              <PencilSquareIcon />
              <span>正在修改</span>
              <strong>{{ selectedImage.label }}</strong>
            </div>
            <button type="button" title="退出图片修改" @click="clearSelectedImage"><XMarkIcon /></button>
          </div>
          <div class="composer-box">
            <div v-if="pendingAttachment" class="pending-attachment">
              <div class="attachment-tile">
                <img :src="pendingAttachment.previewUrl" :alt="pendingAttachment.name">
                <button type="button" title="移除图片" aria-label="移除图片" @click="removePendingAttachment"><XMarkIcon /></button>
              </div>
              <div class="attachment-info">
                <strong>{{ pendingAttachment.name }}</strong>
                <span>{{ pendingAttachment.purpose === 'product_image' ? '产品图片' : '风格参考图' }}</span>
              </div>
            </div>
            <div class="composer-editor">
              <textarea
                ref="composerTextarea"
                v-model="chatInput"
                rows="1"
                :placeholder="composerPlaceholder"
                :disabled="uploading || thinking || startingTask || candidateActionLoading"
                @input="resizeComposer"
                @keydown="handleComposerKeydown"
              ></textarea>
            </div>
            <div class="composer-toolbar">
              <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp" hidden @change="handleFileChange">
              <button type="button" class="icon-button" :title="conversation.context?.white_bg_image_url ? '添加风格参考图' : '添加白底产品图'" :disabled="uploading || thinking || startingTask || candidateActionLoading" @click="fileInput?.click()"><PlusIcon /></button>
              <button
                type="button"
                class="send-button"
                :class="{ 'send-button-stop': isTaskActive || stopping }"
                :title="isTaskActive || stopping ? '停止生成' : '发送'"
                :aria-label="isTaskActive || stopping ? '停止生成' : '发送'"
                :disabled="isTaskActive || stopping ? stopping : !canSubmitMessage"
                @click="isTaskActive || stopping ? requestStop() : submitMessage()"
              >
                <StopIcon v-if="isTaskActive || stopping" />
                <ArrowPathIcon v-else-if="uploading" class="spinning" />
                <ArrowUpIcon v-else />
              </button>
            </div>
          </div>
          <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>
        </footer>
          </section>

          <div class="inspector-dock" :class="{ collapsed: inspectorCollapsed }">
            <button type="button" class="inspector-toggle" :title="inspectorCollapsed ? '展开工作区详情' : '收起工作区详情'" :aria-label="inspectorCollapsed ? '展开工作区详情' : '收起工作区详情'" @click="inspectorCollapsed = !inspectorCollapsed">
              <ChevronRightIcon v-if="inspectorCollapsed" />
              <ChevronLeftIcon v-else />
            </button>
          <aside class="suite-inspector" aria-label="任务详情">
            <div class="inspector-header">
              <div><span class="toolbar-kicker">图片工作区</span><strong>{{ selectedImage ? '当前图片' : '该对话生成的图片' }}</strong></div>
            </div>

            <div v-if="selectedImage" class="inspector-selection">
              <span class="inspector-eyebrow">正在修改</span>
              <strong>{{ selectedImage.label }}</strong>
              <small>描述修改要求后发送，AI 会生成候选版本。</small>
            </div>

            <div v-if="selectedImage" class="inspector-section inspector-image-meta">
              <span class="inspector-label">修改方式</span>
              <div class="inspector-tip"><PencilSquareIcon /><span>直接在底部输入框描述需要调整的内容。</span></div>
              <button type="button" class="inspector-clear" @click="clearSelectedImage">退出图片修改</button>
            </div>

            <div class="inspector-section inspector-images-section">
              <span class="inspector-label">生成图片</span>
              <div v-if="resultGroups.length" class="inspector-image-groups">
                <div v-for="group in resultGroups" :key="`inspector-${group.type}`" class="inspector-image-group">
                  <div class="inspector-image-group-heading"><strong>{{ group.label }}</strong><span>{{ group.completed }}/{{ group.images.length }}</span></div>
                  <div class="inspector-image-grid" :class="`grid-${group.kind}`">
                    <article v-for="image in group.images" :key="image.id" class="inspector-image-entry">
                      <button type="button" class="inspector-image-card" :class="{ selected: selectedImage?.id === image.id }" :disabled="!image.image_url" @click="openImagePreview(image, group)">
                        <span class="inspector-image-frame">
                          <img v-if="image.image_url" :src="resolveImageUrl(currentImageUrl(image))" :alt="image.title || `${group.label} ${image.image_index}`">
                          <PhotoIcon v-else />
                          <span class="inspector-image-number">{{ String(image.image_index).padStart(2, '0') }}</span>
                          <span v-if="image.status !== 'success'" class="inspector-image-status">{{ imageStatusText(image) }}</span>
                          <span v-else-if="image.review_status" class="inspector-image-status" :class="`review-status-${image.review_status}`">{{ imageReviewStatusText(image.review_status) }}</span>
                          <span v-else-if="image.candidate_status === 'ready'" class="inspector-image-status candidate-ready">有修改版本</span>
                          <span v-else-if="image.candidate_status === 'auto_corrected'" class="inspector-image-status candidate-ready">审核重做版本</span>
                        </span>
                        <span class="inspector-image-title">{{ image.title || `${group.label} ${image.image_index}` }}</span>
                      </button>
                      <button v-if="image.review_source_image_url && !(image.review_status === 'auto_corrected' && image.candidate_image_url)" type="button" class="inspector-source-button" @click="openImagePreview(image, group, image.review_source_image_url, '审核前版本')"><PhotoIcon />查看审核前版本</button>
                      <button v-if="image.candidate_status === 'ready' && image.candidate_image_url" type="button" class="inspector-source-button" @click="openImagePreview(image, group, image.candidate_image_url, '修改版本')"><PhotoIcon />查看修改版本</button>
                      <div v-if="image.candidate_status === 'ready'" class="inspector-candidate-actions">
                        <button type="button" :disabled="candidateActionLoading" @click="sendImageCommand(image, group, '保留原图')">保留</button>
                        <button type="button" :disabled="candidateActionLoading" class="primary" @click="sendImageCommand(image, group, '采用这个版本')">采用</button>
                      </div>
                      <div v-else-if="image.candidate_status && image.candidate_status !== 'auto_corrected'" class="inspector-candidate-state">
                        <ArrowPathIcon :class="{ spinning: image.candidate_status === 'generating' }" />
                        <span>{{ candidateStatusText(image) }}</span>
                      </div>
                    </article>
                  </div>
                </div>
              </div>
              <p v-else class="inspector-images-empty">生成完成的图片会显示在这里。</p>
            </div>
          </aside>
          </div>
        </div>
      </main>

      <ImagePreview
        :visible="!!previewImage"
        :imageUrl="previewImage?.url || ''"
        :alt="previewImage?.title || '套图预览'"
        :maxWidth="1240"
        @close="closeImagePreview"
      >
        <template #footer>
          <div class="preview-footer-bar">
            <div>
              <strong>{{ previewImage?.label }}</strong>
              <span>{{ previewImage?.title }}</span>
            </div>
            <button type="button" :disabled="isTaskActive" :title="isTaskActive ? '整套任务完成后即可修改' : '通过对话修改图片'" @click="editPreviewImage"><PencilSquareIcon />{{ isTaskActive ? '任务完成后可修改' : '对话修改' }}</button>
          </div>
        </template>
      </ImagePreview>

      <div v-if="auditComparison" class="audit-compare-backdrop" role="dialog" aria-modal="true" aria-labelledby="audit-compare-title" @keydown.esc.window="closeAuditComparison" @click.self="closeAuditComparison">
        <section class="audit-compare-modal">
          <header class="audit-compare-header">
            <div>
              <span>审核版本对比</span>
              <strong id="audit-compare-title">{{ auditComparison.label }}</strong>
            </div>
            <button type="button" class="audit-compare-close" title="关闭对比" aria-label="关闭对比" @click="closeAuditComparison"><XMarkIcon /></button>
          </header>
          <div class="audit-compare-grid">
            <figure>
              <figcaption><strong>审核前版本</strong><span>重做前</span></figcaption>
              <div class="audit-compare-image"><img :src="auditComparison.beforeUrl" alt="审核重做前的图片"></div>
            </figure>
            <figure>
              <figcaption><strong>审核重做版本</strong><span>重做后</span></figcaption>
              <div class="audit-compare-image"><img :src="auditComparison.afterUrl" alt="审核重做后的图片"></div>
            </figure>
          </div>
        </section>
      </div>
    </div>
  </MainLayout>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue'
import {
  ArrowPathIcon,
  ArrowUpIcon,
  ArrowsRightLeftIcon,
  ArrowsPointingOutIcon,
  CheckIcon,
  ClipboardDocumentIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpDownIcon,
  ChevronUpIcon,
  DocumentTextIcon,
  MinusCircleIcon,
  PaperClipIcon,
  PencilSquareIcon,
  PhotoIcon,
  PlusIcon,
  SparklesIcon,
  StopIcon,
  TrashIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { downloadBrandOptions, resolveImageUrl } from '@/utils/request'
import { useAuthStore } from '@/stores/auth'

const emptyContext = () => ({
  white_bg_image_url: '',
  product_reference_urls: [],
  product_name: '',
  product_category: '',
  product_brand: '',
  product_features: '',
  product_proof: '',
  product_length: '',
  product_width: '',
  product_height: '',
  product_dimension_unit: '',
  product_weight: '',
  product_weight_unit: '',
  package_length: '',
  package_width: '',
  package_height: '',
  package_dimension_unit: '',
  package_weight: '',
  package_weight_unit: '',
  package_contents: '',
  listing_platform_rules: '',
  listing_quality_rules: '',
  listing_keywords: '',
  platform: '',
  platform_name: '',
  country_code: '',
  country_name: '',
  language: '',
  language_name: '',
  output_target: '',
  request_type: '',
  include_listing: false,
  requirement_form_open: false,
  secondary_count: 0,
  desktop_a_plus_count: 0,
  mobile_a_plus_count: 0,
  visual_style: '',
  style_requirements: '',
  style_reference_urls: [],
  drawing_model: '',
})

const conversation = ref({ session_id: '', task_id: '', task_ids: [], task_history: [], context: emptyContext(), messages: [], ready: false, missing_fields: [] })
const conversationHistory = ref([])
const authStore = useAuthStore()
const isAdmin = computed(() => (authStore.user?.roles || []).some(role => role?.name === 'admin' || role === 'admin'))
const currentTask = ref(null)
const chatInput = ref('')
const selectedImage = ref(null)
const previewImage = ref(null)
const auditComparison = ref(null)
const pendingAttachment = ref(null)
const thinking = ref(false)
const uploading = ref(false)
const startingTask = ref(false)
const stopping = ref(false)
const candidateActionLoading = ref(false)
const errorMessage = ref('')
const chatMessagesEl = ref(null)
const fileInput = ref(null)
const onboardingFileInput = ref(null)
const composerTextarea = ref(null)
const pollingTimer = ref(null)
const taskTodoCollapsed = ref(false)
const inspectorCollapsed = ref(false)
const deletingConversation = ref('')
const submittingRequirements = ref(false)
const initialProductImages = ref([])
const maxInitialProductImages = computed(() => ['listing', 'listing_evaluation'].includes(initialForm.value.requestType) ? 1 : 6)
const brandOptions = ref([...downloadBrandOptions])
const initialFormError = ref('')
const initialFormErrors = ref({})
const copiedListingTaskId = ref('')
const listingEvaluationResult = ref(null)
const listingEvaluationError = ref('')
const requirementFormVisible = ref(false)
let activePollTaskId = ''

const listingEvaluationPlatformName = computed(() => {
  const platform = listingEvaluationResult.value?.platform || listingEvaluationResult.value?.evaluation?.optimized_listing?.platform
  return ({ home_depot: 'Home Depot', tiktok: 'TikTok Shop', lowes: 'Lowe’s', wayfair: 'Wayfair', amazon: 'Amazon' })[platform] || '平台'
})

const listingEvaluationUsesRawPoints = computed(() => {
  const platform = listingEvaluationResult.value?.platform || listingEvaluationResult.value?.evaluation?.optimized_listing?.platform
  return ['home_depot', 'lowes', 'wayfair'].includes(platform)
})

const optimizedListingSections = computed(() => {
  const result = listingEvaluationResult.value
  const listing = result?.evaluation?.optimized_listing
  if (!listing || typeof listing !== 'object') return []
  const platform = result?.platform || listing.platform
  const text = (key, label, value) => String(value || '').trim() ? { key, label, text: String(value).trim() } : null
  const items = (key, label, values) => Array.isArray(values) && values.length ? { key, label, items: values.filter(Boolean) } : null
  const pairs = (key, label, values) => items(key, label, Array.isArray(values) ? values.map(item => `${item?.name || ''}: ${item?.value || ''}`.replace(/^:\s*|:\s*$/g, '')).filter(Boolean) : [])
  let sections = []
  if (platform === 'home_depot') {
    sections = [text('title', 'Title', listing.title), items('salient_bullets', 'Salient Bullets', listing.salient_bullets), items('bullet_points', 'Bullet Points', listing.bullet_points), text('about_this_product', 'About This Product', listing.about_this_product)]
  } else if (platform === 'tiktok') {
    sections = [text('title', 'Title', listing.title), text('description', 'Description', listing.description), pairs('attributes', 'Attributes', listing.attributes), items('variants', 'Variants', listing.variants), items('whats_included', 'Package Contents', listing.whats_included)]
  } else if (platform === 'lowes') {
    sections = [text('title', 'Title', listing.title), text('description', 'Description', listing.description), items('bullets', 'Bullets', listing.bullets)]
  } else if (platform === 'wayfair') {
    sections = [text('about_this_product', 'About This Product', listing.about_this_product), items('features', 'Feature Bullets', listing.features)]
  } else {
    sections = [text('title', 'Title', listing.title), items('bullets', 'Bullet Points', listing.bullets), text('description', 'Product Description', listing.description), items('search_terms', 'Search Terms', listing.search_terms)]
  }
  return sections.filter(Boolean)
})

const smartSuiteAutoVisualStyle = 'AI 根据产品、品类、平台和站点自主制定高端风格'
const platformPresets = [
  { id: 'amazon_us', label: 'Amazon 美国站', description: '英文 · US', platform: 'amazon', platformName: 'Amazon', countryCode: 'US', countryName: '美国站' },
  { id: 'amazon_de', label: 'Amazon 德国站', description: '德文 · DE', platform: 'amazon', platformName: 'Amazon', countryCode: 'DE', countryName: '德国站' },
  { id: 'walmart_us', label: 'Walmart 美国站', description: '英文 · US', platform: 'walmart', platformName: 'Walmart', countryCode: 'US', countryName: '美国站' },
  { id: 'home_depot_us', label: 'Home Depot 美国站', description: '英文 · US', platform: 'home_depot', platformName: 'Home Depot', countryCode: 'US', countryName: '美国站' },
  { id: 'wayfair_us', label: 'Wayfair 美国站', description: '英文 · US', platform: 'wayfair', platformName: 'Wayfair', countryCode: 'US', countryName: '美国站' },
  { id: 'custom', label: '自定义平台', description: '自行填写站点', platform: '', platformName: '', countryCode: '', countryName: '' },
]
const listingPlatformPresets = [
  { id: 'tiktok_us', label: 'TikTok Shop 美国站', description: '英文 · US', platform: 'tiktok', platformName: 'TikTok Shop', countryCode: 'US', countryName: '美国站' },
  platformPresets[3],
  { id: 'amazon_custom', label: 'Amazon 其他站点', description: '自行填写国家 / 站点', platform: 'amazon', platformName: 'Amazon', countryCode: '', countryName: '' },
  { id: 'lowes_us', label: 'Lowe’s 美国站', description: '英文 · US', platform: 'lowes', platformName: 'Lowe’s', countryCode: 'US', countryName: '美国站' },
  platformPresets[4],
  platformPresets[0],
  { id: 'custom', label: '自定义平台', description: '自行填写站点', platform: '', platformName: '', countryCode: '', countryName: '' },
]
const outputPresets = [
  { key: 'secondary', label: '商品副图', description: '产品事实驱动，覆盖功能、场景、材质与适配', enabledKey: 'secondaryEnabled', countKey: 'secondaryCount', max: 9 },
  { key: 'desktop', label: '桌面 A+', description: '按买家顾虑组织连续横版说服路径', enabledKey: 'desktopEnabled', countKey: 'desktopCount', max: 8 },
  { key: 'mobile', label: '移动 A+', description: '延续 A+ 说服顺序，独立移动端构图', enabledKey: 'mobileEnabled', countKey: 'mobileCount', max: 8 },
]
const visualPresets = [
  { id: 'auto', label: 'AI 自动设计', description: '默认高端风格，按产品和平台制定', value: smartSuiteAutoVisualStyle },
  { id: 'brand', label: '高端品牌场景', description: '克制布光，突出材质和品质', value: '高端品牌场景' },
  { id: 'lifestyle', label: '真实生活场景', description: '自然使用状态，强调代入感', value: '真实生活使用场景' },
  { id: 'modern_home', label: '现代家居场景', description: '真实空间，兼顾产品和环境', value: '现代家居场景' },
  { id: 'outdoor', label: '户外自然光', description: '自然环境，画面更有呼吸感', value: '户外自然光场景' },
  { id: 'custom', label: '自定义要求', description: '补充你明确的视觉偏好', value: '' },
]
const entryModeOptions = [
  { id: 'suite', requestType: 'suite', label: '生成套图', description: '副图、桌面 A+、移动 A+，按需选择', icon: SparklesIcon },
  { id: 'listing', requestType: 'listing', label: '只写 Listing', description: '按平台字段生成商品文案', icon: DocumentTextIcon },
  { id: 'listing_evaluation', requestType: 'listing_evaluation', label: '评估现有 Listing', description: '评分、扣分并给出修改后的完整文案', icon: ClipboardDocumentIcon },
  { id: 'suite_and_listing', requestType: 'suite_and_listing', label: '套图 + Listing', description: '一次填写，同时产出图片和商品文案', icon: PhotoIcon },
]

function createInitialForm() {
  return {
    productName: '', productCategory: '', productBrand: '', productFeatures: '', productProof: '',
    productLength: '', productWidth: '', productHeight: '', productDimensionUnit: '',
    productWeight: '', productWeightUnit: '', packageLength: '', packageWidth: '', packageHeight: '',
    packageDimensionUnit: '', packageWeight: '', packageWeightUnit: '', packageContents: '',
    internetNumber: '',
    listingTitle: '', listingSalientBullets: '', listingBullets: '', listingDescription: '', listingSearchTerms: '', listingKeywords: '',
    tiktokAttributes: '', tiktokVariants: '', tiktokPackageContents: '',
    lowesProductDescription: '', lowesSupplierBullets: '', lowesMarketingCopy: '', lowesAttributes: '',
    wayfairDescription: '', wayfairFeatures: '', wayfairIncluded: '', wayfairDetails: '',
    requestType: 'suite',
    platformPreset: 'amazon_us', platform: 'amazon', platformName: 'Amazon', countryCode: 'US', countryName: '美国站',
    secondaryEnabled: true, secondaryCount: 6,
    desktopEnabled: true, desktopCount: 7,
    mobileEnabled: true, mobileCount: 7,
    visualPreset: 'auto', visualStyle: smartSuiteAutoVisualStyle, styleRequirements: '', drawingModel: '',
  }
}
const initialForm = ref(createInitialForm())
const visiblePlatformPresets = computed(() => initialForm.value.requestType === 'suite' ? platformPresets : listingPlatformPresets)
const evaluationPlatformPresets = computed(() => listingPlatformPresets.filter(preset => ['tiktok', 'home_depot', 'lowes', 'wayfair', 'amazon'].includes(preset.platform)))

function syncImagePresentation() {
  const images = currentTask.value?.images || []
  const selectedId = Number(selectedImage.value?.id || 0)
  if (selectedId) {
    const image = images.find(item => Number(item.id) === selectedId)
    if (!image || !image.image_url) {
      selectedImage.value = null
    } else {
      const group = resultGroups.value.find(item => item.images.some(candidate => candidate.id === image.id))
      const label = group ? `${group.label} ${String(image.image_index).padStart(2, '0')}` : selectedImage.value.label
      selectedImage.value = {
        ...selectedImage.value,
        id: image.id,
        label: `${label}「${image.title || image.role || '未命名画面'}」`,
        hasCandidate: image.candidate_status === 'ready',
      }
    }
  }
  const previewId = Number(previewImage.value?.id || 0)
  if (previewId) {
    const image = images.find(item => Number(item.id) === previewId)
    if (!image || !image.image_url) {
      previewImage.value = null
    } else {
      const group = resultGroups.value.find(item => item.images.some(candidate => candidate.id === image.id))
      const previewVersion = previewImage.value.version || (previewImage.value.candidate ? 'candidate' : 'original')
      const previewURL = previewVersion === 'review_source'
        ? image.review_source_image_url
        : previewVersion === 'candidate' && ['ready', 'auto_corrected'].includes(image.candidate_status)
          ? image.candidate_image_url
          : image.image_url
      if (!previewURL) {
        previewImage.value = null
        return
      }
      const showingCandidate = previewVersion === 'candidate'
      previewImage.value = {
        ...previewImage.value,
        url: resolveImageUrl(previewURL),
        title: previewVersion === 'review_source' ? '审核前版本' : showingCandidate ? (image.candidate_status === 'auto_corrected' ? '审核重做版本' : '修改版本') : image.title || image.role || '未命名画面',
        image,
        group: group || previewImage.value.group,
        candidate: showingCandidate,
        version: previewVersion,
      }
    }
  }
}

const activeStatuses = ['pending', 'planning', 'generating', 'reviewing', 'correcting']
const isTaskActive = computed(() => activeStatuses.includes(currentTask.value?.status))
const completedCount = computed(() => currentTask.value?.images?.filter(image => image.status === 'success').length || 0)
const totalCount = computed(() => currentTask.value?.total_image_count || currentTask.value?.images?.length || 0)
const smartStage = computed(() => currentTask.value?.smart_stage || '')
const stageDetails = {
	planning: { title: '主 Agent 正在规划智能套图流程', activity: '主 Agent 正在分析任务依赖并安排下一步 Agent', progress: 2 },
  researching_listing_keywords: { title: '正在搜索高 SEO 关键词', activity: 'Listing Keyword Agent 正在联网搜索平台类目词并结合运营关键词', progress: 6 },
  analyzing_buyers: { title: '正在规划 A+ 购买路径', activity: '买家洞察 Agent 正在基于产品事实整理 A+ 需要消除的购买顾虑', progress: 3 },
  confirming_suite_prompts: { title: '正在确认整套生图指令', activity: '整套提示词 Agent 正在一次性锁定每张图的产品、卖点、场景、构图、文案位置和负面约束', progress: 7 },
	writing_listing: { title: '正在同步撰写 Listing', activity: 'Listing Writer Agent 正在按平台字段组织标题、卖点、描述、属性和搜索词', progress: 9 },
	reviewing_listing: { title: '正在审核并自动修复 Listing', activity: 'Listing Review Agent 正在检查语法、事实边界、本土化和平台字段，发现问题会自动修复', progress: 12 },
  generating_secondary: { title: '正在生成商品副图', activity: '按确认的产品事实生成独立功能、场景与细节任务', progress: 10 },
  generating_desktop: { title: '正在生成桌面 A+', activity: '正在按买家顾虑和购买路径生成 21:9 横版模块', progress: 40 },
  generating_mobile: { title: '正在生成移动 A+', activity: '正在生成 7 张独立 4:3 构图，不会裁切桌面图', progress: 73 },
}
const currentStageDetail = computed(() => isTaskActive.value ? stageDetails[smartStage.value] || null : null)
const taskTodoItems = computed(() => {
  const items = Array.isArray(currentTask.value?.todo_items) ? currentTask.value.todo_items : []
  if (!listingIncompleteForTask(currentTask.value)) return items
  const platform = listingPlatformLabel(listingForTask(currentTask.value))
  return items.map(item => item.id === 'product-listing'
    ? { ...item, status: 'failed', detail: `缺少 ${platform} 平台要求的字段` }
    : item)
})
const taskTodoAllDone = computed(() => currentTask.value?.status === 'success' && !listingIncompleteForTask(currentTask.value))
const taskHasError = computed(() => ['failed', 'partial_failed'].includes(String(currentTask.value?.status || '')))
const taskTodoCompletedCount = computed(() => listingIncompleteForTask(currentTask.value)
  ? taskTodoItems.value.filter(todo => todo.status === 'done').length
  : Number(currentTask.value?.todo_completed_count || 0))
const taskTodoTotalCount = computed(() => Number(currentTask.value?.todo_total_count ?? taskTodoItems.value.length))
const currentTodoItem = computed(() => taskTodoItems.value.find(todo => todo.status === 'active' || todo.status === 'failed') || null)

function taskTodoItemClass(todo) {
  return {
    done: todo.status === 'done',
    active: todo.status === 'active',
    interrupted: todo.status === 'failed',
    skipped: todo.status === 'skipped',
  }
}

const smartSuiteStageTypes = ['smart_secondary', 'smart_desktop_a_plus', 'smart_mobile_a_plus']
const smartSuiteStageNames = [
	['analyzing_buyers', 'confirming_suite_prompts', 'writing_listing', 'generating_secondary'],
  ['generating_desktop'],
  ['generating_mobile'],
]
const visibleSmartSuiteStageIndex = computed(() => {
  const task = currentTask.value
  if (!task?.task_id) return -1
  if (['success', 'partial_failed', 'cancelled'].includes(task.status)) return smartSuiteStageTypes.length - 1
  const stage = String(task.smart_stage || '').trim()
  if (!stage) return -1
  const index = smartSuiteStageNames.findIndex(names => names.includes(stage))
  return index >= 0 ? Math.min(index, smartSuiteStageTypes.length - 1) : -1
})
const taskProgressPercent = computed(() => {
  if (taskTodoTotalCount.value > 0) {
    const progress = Math.round((taskTodoCompletedCount.value / taskTodoTotalCount.value) * 100)
    if (currentTask.value?.status === 'success') return 100
    return isTaskActive.value ? Math.max(3, Math.min(progress, 99)) : Math.min(progress, 99)
  }
  if (currentStageDetail.value?.progress) {
    const imageProgress = totalCount.value ? Math.round((completedCount.value / totalCount.value) * 82) : 0
    return Math.max(currentStageDetail.value.progress, imageProgress)
  }
  if (!totalCount.value) return currentTask.value?.status === 'planning' ? 8 : 3
  const base = Math.round((completedCount.value / totalCount.value) * 82)
  if (currentTask.value?.status === 'reviewing') return 90
  if (currentTask.value?.status === 'correcting') return 95
  if (!isTaskActive.value) return currentTask.value?.status === 'success' ? 100 : Math.min(base, 99)
  return Math.max(currentTask.value?.status === 'planning' ? 8 : 12, base)
})

const composerPlaceholder = computed(() => {
  if (selectedImage.value) return `告诉 AI 如何修改${selectedImage.value.label}`
  if (isTaskActive.value) return '任务生成中，可点击右侧停止按钮中止任务'
  if (conversation.value.ready) return '输入“开始生成”，或继续调整需求'
  if (pendingAttachment.value) return '可以继续补充文字，再一起发送'
  if (isCollectingModel.value) return '选择上方模型，或继续补充你的生成要求'
  return '描述产品、平台和需要生成的图片'
})

const suggestionOptions = computed(() => {
  const ctx = conversation.value.context || emptyContext()
  if (startingTask.value) return []
  if (currentTask.value?.workflow_type === 'smart_listing' && isTaskActive.value) return []
  if (Array.isArray(conversation.value.suggestions) && conversation.value.suggestions.length) {
    if (conversation.value.ready && !selectedImage.value && !isTaskActive.value) {
      return [...new Set(['开始生成', ...conversation.value.suggestions])]
    }
    return conversation.value.suggestions
  }
  const missing = Array.isArray(conversation.value.missing_fields) ? conversation.value.missing_fields : []
  if (missing.includes('white_bg_image')) return []
  if (!ctx.product_name || !ctx.product_features) return []
  if (!ctx.platform || !ctx.country_code) return ['Amazon 美国站', 'Amazon 欧洲站', 'Walmart 美国站', 'Home Depot 美国站', 'Wayfair 美国站', "Lowe's 美国站", 'Temu 美国站']
  if (!(Number(ctx.secondary_count) + Number(ctx.desktop_a_plus_count) + Number(ctx.mobile_a_plus_count))) return ['6 张副图', '7 张桌面 A+', '7 张移动 A+', '副图、桌面 A+、移动 A+ 都要']
  if (!ctx.drawing_model) return []
  if (conversation.value.ready) return ['开始生成', '我还要调整需求']
  return []
})
const isCollectingStyle = computed(() => {
  return conversation.value.suggestion_field === 'visual_style'
})
const drawingModelOptions = computed(() => Array.isArray(conversation.value.model_options) ? conversation.value.model_options : [])
const isCollectingModel = computed(() => conversation.value.suggestion_field === 'drawing_model' && drawingModelOptions.value.length > 0)
const showRequirementForm = computed(() => {
	return requirementFormVisible.value || Boolean(conversation.value.requirement_form_open)
})
function closeRequirementForm() {
  requirementFormVisible.value = false
  conversation.value.requirement_form_open = false
}
const showEntryActions = computed(() => {
  const messages = Array.isArray(conversation.value.messages) ? conversation.value.messages : []
  const hasUserMessage = messages.some(message => message?.role === 'user' && (String(message.text || '').trim() || message.image_url))
  return !showRequirementForm.value && !submittingRequirements.value && !listingEvaluationResult.value && !currentTask.value && !hasUserMessage
})
const latestAssistantMessageId = computed(() => {
  const messages = conversation.value.messages || []
  return [...messages].reverse().find(message => message.role === 'assistant')?.id || ''
})
const sidebarConversationRecords = computed(() => {
  return Array.isArray(conversationHistory.value) ? conversationHistory.value : []
})
const conversationTasks = computed(() => {
  const byId = new Map()
  for (const task of Array.isArray(conversation.value.task_history) ? conversation.value.task_history : []) {
    if (task?.task_id) byId.set(task.task_id, task)
  }
  if (currentTask.value?.task_id) byId.set(currentTask.value.task_id, currentTask.value)
  return [...byId.values()].sort((left, right) => Date.parse(left.created_at || '') - Date.parse(right.created_at || ''))
})
const canSubmitMessage = computed(() => Boolean(chatInput.value.trim() || pendingAttachment.value) && !uploading.value && !thinking.value && !startingTask.value && !candidateActionLoading.value)

function isStartConversationMessage(message) {
  if (message?.role !== 'user') return false
  return isStartCommandText(message.text)
}

function isStartCommandText(value) {
  const text = String(value || '').trim().toLowerCase()
  return /^(开始生成|确认生成|开始|开始吧|生成吧|重新生成|再生成|start|go)[。.!！\s]*$/i.test(text)
}

function timelineTaskTime(task) {
  const createdTime = Date.parse(task?.created_at || '')
  if (!Number.isFinite(createdTime)) return Date.now()
  // Anchor the task card right after the message that triggered it; later
  // conversation turns must not push the card around the timeline.
  const triggerTimes = (conversation.value.messages || [])
    .map(message => Date.parse(message?.at || ''))
    .filter(value => Number.isFinite(value) && value <= createdTime + 2000)
  return triggerTimes.length ? Math.max(createdTime, Math.max(...triggerTimes) + 1) : createdTime
}

const groupTimelineOffsets = { smart_secondary: 1, smart_desktop_a_plus: 2, smart_mobile_a_plus: 3 }
const groupActivity = ref({})
function trackGroupActivity(task) {
  if (!task?.task_id) return
  for (const group of taskResultGroups(task)) {
    const key = `${task.task_id}:${group.type}`
    const fingerprint = group.images.map(image => [image.id, image.status, image.image_url, image.candidate_image_url, image.candidate_status].join('|')).join('::')
    const record = groupActivity.value[key]
    if (!record) {
      groupActivity.value[key] = { fingerprint, seenAt: 0 }
    } else if (record.fingerprint !== fingerprint) {
      groupActivity.value[key] = { fingerprint, seenAt: Date.now() }
    }
  }
}

function groupTimelineTime(task, group) {
  const seenAt = groupActivity.value[`${task?.task_id || ''}:${group.type}`]?.seenAt || 0
  if (seenAt) return seenAt
  return timelineTaskTime(task) + (groupTimelineOffsets[group.type] || 0) * 1000
}

watch(drawingModelOptions, options => {
  if (!initialForm.value.drawingModel && initialForm.value.requestType !== 'listing' && options.length) {
    initialForm.value.drawingModel = options.find(option => option.is_default)?.name || options[0].name
  }
}, { immediate: true })

const timelineItems = computed(() => {
  const items = (conversation.value.messages || [])
    .filter(message => String(message.text || '').trim() !== '信息完整，开始生成。')
    .map((message, index) => ({
    key: `message-${message.id || index}`,
    type: 'message',
    data: message,
    time: Date.parse(message.at || '') || index,
    order: index,
    }))
  for (const task of conversationTasks.value) {
    const isCurrent = task.task_id === currentTask.value?.task_id
    items.push({
      key: `task-${task.task_id}`,
      type: isCurrent ? 'task' : 'task_history',
      data: task,
      time: timelineTaskTime(task),
      order: items.length,
    })
    if (!isCurrent) continue
    for (const group of resultGroups.value) {
      items.push({
        key: `group-${task.task_id}-${group.type}`,
        type: 'image_group',
        data: group,
        time: groupTimelineTime(task, group),
        order: items.length,
      })
    }
  }
  return items.sort((a, b) => a.time - b.time || a.order - b.order)
})

function taskResultGroups(task) {
  const images = task?.images || []
  const status = String(task?.status || '').trim()
  const isCurrent = task?.task_id === currentTask.value?.task_id
  const visibleStage = isCurrent ? visibleSmartSuiteStageIndex.value : smartSuiteStageTypes.length - 1
  return [
    { type: 'smart_secondary', kind: 'secondary', label: '副图', ratio: '1:1' },
    { type: 'smart_desktop_a_plus', kind: 'desktop', label: '桌面 A+', ratio: '21:9' },
    { type: 'smart_mobile_a_plus', kind: 'mobile', label: '移动 A+', ratio: '4:3' },
  ].map(group => {
    const groupImages = images.filter(image => image.image_type === group.type && (status !== 'cancelled' || image.image_url || image.candidate_image_url))
    return { ...group, taskId: task?.task_id || '', images: groupImages, completed: groupImages.filter(image => image.status === 'success').length }
  }).filter(group => {
    const hasRenderedImage = group.images.some(image => image.image_url || image.candidate_image_url || image.candidate_status)
    if (hasRenderedImage) return true
    if (!['generating', 'reviewing', 'correcting'].includes(status)) return false
    const stageIndex = smartSuiteStageTypes.indexOf(group.type)
    if (visibleStage >= 0 && stageIndex >= 0) return stageIndex <= visibleStage
    return false
  }).filter(group => group.images.length)
}

const resultGroups = computed(() => taskResultGroups(currentTask.value))

function listingForTask(task) {
	const draft = task?.listing || task?.plan_snapshot?.listing || task?.listing_result || null
	if (!draft) return null
	const rawPlatform = String(task?.platform || draft.platform || '').trim().toLowerCase()
	const platform = ({ homedepot: 'home_depot', 'home depot': 'home_depot', home_depot: 'home_depot', 'home-depot': 'home_depot', lowe: 'lowes', lowes: 'lowes', lowe_s: 'lowes', "lowe's": 'lowes', 'lowe’s': 'lowes', tiktok_shop: 'tiktok', tiktokshop: 'tiktok', amazon_com: 'amazon' })[rawPlatform] || rawPlatform
	const listing = draft
	if (!listing) return null
	return { ...listing, platform }
}

function keywordCandidatesForTask(task) {
  const todoCandidates = (Array.isArray(task?.todo_items) ? task.todo_items : [])
    .find(item => item?.id === 'listing-keyword-research')?.keywords
  const candidates = Array.isArray(task?.listing_keywords)
    ? task.listing_keywords
    : Array.isArray(task?.plan_snapshot?.listing_keywords)
      ? task.plan_snapshot.listing_keywords
      : Array.isArray(todoCandidates) ? todoCandidates : []
  const seen = new Set()
  return candidates.filter(keyword => {
    const term = String(keyword?.term || '').trim()
    const key = term.toLowerCase()
    if (!term || seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function keywordSourceLabel(source) {
  const value = String(source || '').trim().toLowerCase()
  return ({
    operator: '运营输入',
    marketplace: '平台搜索',
    autocomplete: '平台自动补全',
    product_title: '平台商品标题',
    keyword_research_agent: '关键词 Agent',
  })[value] || (value || '平台搜索')
}

function listingPlatformLabel(listing) {
  const platform = String(listing?.platform || '').toLowerCase()
  return ({ home_depot: 'Home Depot', lowes: 'Lowe’s', wayfair: 'Wayfair' })[platform] || 'Listing'
}

function listingLocaleLabel(listing) {
  const platform = String(listing?.platform || '').toLowerCase()
  return ['home_depot', 'lowes', 'wayfair'].includes(platform) ? '美国英语 · 本土化' : '目标站点语言'
}

function listingIncompleteForTask(task) {
	const listing = listingForTask(task)
	if (!listing) return false
	if (listing.platform === 'home_depot') {
		return !String(listing.title || '').trim() || listing.salient_bullets?.length !== 3 || listing.bullet_points?.length !== 6 || !String(listing.about_this_product || '').trim()
	}
	if (listing.platform === 'lowes') {
		return !String(listing.title || '').trim() || !String(listing.description || '').trim() || !Array.isArray(listing.bullets) || listing.bullets.length === 0
	}
	if (listing.platform === 'wayfair') {
		return !String(listing.about_this_product || '').trim() || !Array.isArray(listing.features) || listing.features.length === 0
	}
	return false
}

function listingReviewFailed(task) {
  return Boolean(task?.listing_review_error || task?.plan_snapshot?.listing_review_error)
}

function listingSections(listing) {
  if (!listing) return []
  const sections = []
  const platform = String(listing.platform || '').toLowerCase()
  const normalizeSectionKey = value => {
    const normalized = String(value || '').trim().toLowerCase().replace(/[\s\-/]+/g, '_')
    return ({ product_title: 'title', product_description: 'description', product_overview: 'description', key_features: 'bullets', product_highlights: 'highlights', variant_options: 'variants', options: 'variants', specifications: 'attributes', specs: 'attributes', dimensions: 'details', wayfair_details: 'details', certifications: 'documents', certification: 'documents', support_documents: 'documents', search_keywords: 'search_terms', backend_search_terms: 'search_terms' })[normalized] || normalized
  }
  const evidenceFor = key => {
    const seen = new Set()
    return (Array.isArray(listing.evidence) ? listing.evidence : [])
      .filter(item => normalizeSectionKey(item?.field) === key)
      .flatMap(item => String(item?.quote || '').split(/[；;\n]/))
      .map(item => item.trim())
      .filter(item => item && !seen.has(item) && seen.add(item))
  }
  const reasonFor = key => {
    const item = (Array.isArray(listing.optimization_reasons) ? listing.optimization_reasons : [])
      .find(reason => normalizeSectionKey(reason?.field) === key)
    return item || null
  }
  const addSection = (key, label, type, value) => {
    const rationale = reasonFor(key)
    sections.push({ key, label, type, value, evidence: evidenceFor(key), reason: String(rationale?.reason || '').trim(), platformBasis: String(rationale?.platform_basis || '').trim(), writingDecision: String(rationale?.writing_decision || '').trim(), expectedImpact: String(rationale?.expected_impact || '').trim() })
  }
  const addText = (key, label, value) => { if (String(value || '').trim()) addSection(key, label, 'text', value) }
  const addList = (key, label, value) => { if (Array.isArray(value) && value.length) addSection(key, label, 'list', value) }
  if (platform === 'home_depot') {
    addText('title', 'Title', listing.title)
    addList('salient_bullets', 'Salient Bullets', listing.salient_bullets)
    addList('bullet_points', 'Bullet Points', listing.bullet_points)
    addText('about_this_product', 'About This Product', listing.about_this_product)
    return sections
  }
  if (platform === 'lowes') {
    addText('title', 'Title', listing.title)
    addText('description', 'Description', listing.description)
    addList('bullets', 'Bullets', listing.bullets)
    return sections
  }
  if (platform === 'wayfair') {
    addText('about_this_product', 'About This Product', listing.about_this_product)
    addList('features', 'Feature Bullets', listing.features)
    return sections
  }
  addText('title', platform === 'home_depot' || platform === 'amazon' || platform === 'tiktok' || platform === 'lowes' ? 'Title' : 'Product Title', listing.title)
  addList('salient_bullets', platform === 'home_depot' ? 'Salient Bullets' : 'Highlights', listing.salient_bullets)
  addList('bullets', platform === 'amazon' || platform === 'home_depot' ? 'Bullet Points' : platform === 'lowes' ? 'Bullets' : 'Supplier Bullets', listing.bullets)
  if (platform !== 'home_depot') addList('highlights', 'Highlights', listing.highlights)
  addList('supplier_bullets', 'Supplier Bullets', listing.supplier_bullets)
  if (platform === 'wayfair') addText('about_this_product', 'About This Product', listing.about_this_product)
  else addText('description', platform === 'home_depot' ? 'About this product' : platform === 'lowes' ? 'Product Description' : platform === 'amazon' ? 'Product Description' : 'Product Description', listing.description)
  addText('lowes_product_description', 'Product Description', listing.lowes_product_description)
  addText('marketing_copy', 'Marketing Copy', listing.marketing_copy)
  addList('features', platform === 'wayfair' ? 'Feature Bullets' : '产品 Features', listing.features)
  addList('variants', 'Variants', listing.variants)
  if (platform !== 'wayfair') addList('whats_included', platform === 'tiktok' ? 'Package Contents' : 'Included', listing.whats_included)
  addList('not_included', 'Not Included', listing.not_included)
  if (platform !== 'home_depot') addList('documents', 'Documents / Certifications', listing.documents)
  if (platform !== 'wayfair' && platform !== 'home_depot' && Array.isArray(listing.attributes) && listing.attributes.length) addSection('attributes', platform === 'tiktok' ? 'Attributes' : 'Category Attributes', 'list', listing.attributes.map(item => String(item.name || '') + ': ' + String(item.value || '')))
  if (platform !== 'wayfair' && Array.isArray(listing.details) && listing.details.length) addSection('details', 'Dimensions / Details', 'list', listing.details.map(item => String(item.name || '') + ': ' + String(item.value || '')))
  if (Array.isArray(listing.search_terms) && listing.search_terms.length) addSection('search_terms', platform === 'home_depot' ? 'Search Keywords' : platform === 'amazon' ? 'Search Terms' : '搜索词', 'terms', listing.search_terms)
  return sections
}

function listingErrorForTask(task) {
	return String(task?.listing_error || task?.plan_snapshot?.listing_error || '').trim()
}

function listingPlainText(listing) {
	return listingSections(listing).map(section => {
    const value = section.type === 'list' ? section.value.map((item, index) => String(index + 1) + '. ' + item).join('\n') : section.type === 'terms' ? section.value.join(', ') : section.value
    return section.label + '\n' + value
  }).join('\n\n')
}

async function copyListing(task) {
	const listing = listingForTask(task)
	if (!listing) return
	try {
		await window.navigator.clipboard.writeText(listingPlainText(listing))
		copiedListingTaskId.value = task?.task_id || 'current'
		window.setTimeout(() => {
			if (copiedListingTaskId.value === (task?.task_id || 'current')) copiedListingTaskId.value = ''
		}, 1800)
	} catch {
		errorMessage.value = '复制 Listing 失败，请检查浏览器剪贴板权限。'
	}
}

function formatTaskCreatedAt(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function taskStatusLabel(status) {
  return ({ pending: '排队中', planning: '规划中', generating: '生成中', reviewing: '审核中', correcting: '修正中', success: '已完成', partial_failed: '部分完成', failed: '未完成', cancelled: '已停止' })[status] || '处理中'
}

function taskStatusToneClass(status) {
  const tone = ({ success: 'success', partial_failed: 'partial', failed: 'failed', cancelled: 'stopped', pending: 'pending', planning: 'active', generating: 'active', reviewing: 'active', correcting: 'active' })[String(status || '')] || 'pending'
  return `tone-${tone}`
}

const taskStateTitle = computed(() => {
	if (currentTask.value?.workflow_type === 'smart_listing') {
		if (listingReviewFailed(currentTask.value)) return 'Listing 已生成，但审核未通过'
		if (currentTask.value?.smart_stage === 'reviewing_listing') return '正在审核 Listing'
		if (listingIncompleteForTask(currentTask.value)) return 'Listing 生成未完成'
    const listingLabels = {
      pending: 'Listing 任务已创建',
      planning: '正在撰写 Listing',
      success: 'Listing 已完成',
      failed: 'Listing 生成未完成',
      cancelled: 'Listing 已停止',
    }
    return listingLabels[currentTask.value?.status] || '正在准备 Listing'
  }
  const labels = {
    pending: '任务已创建',
    planning: '正在理解产品并规划整套画面',
    generating: '正在生成套图',
    reviewing: '正在检查生成结果',
    correcting: '正在修正少数图片',
    success: '套图已完成',
    partial_failed: '套图部分完成',
    failed: '这次生成没有完成',
    cancelled: '生成已停止',
  }
  if (currentTask.value?.status === 'partial_failed' && completedCount.value === 0) return labels.failed
  if (taskHasError.value) return labels[currentTask.value?.status] || labels.failed
  return currentTodoItem.value?.label || currentStageDetail.value?.title || labels[currentTask.value?.status] || '正在准备套图'
})

const taskProgressText = computed(() => {
	if (currentTask.value?.workflow_type === 'smart_listing') {
		if (listingReviewFailed(currentTask.value)) return 'Listing 已生成，但独立审核未通过，请根据审核意见修复后重试'
		if (currentTask.value?.smart_stage === 'reviewing_listing') return 'Listing 已生成，正在进行平台合规和事实审核'
		if (listingIncompleteForTask(currentTask.value)) return '缺少平台要求的字段，请调整产品事实后重试'
    if (currentTask.value?.status === 'success') return '该平台所需字段和编写理由已生成'
    if (currentTask.value?.status === 'failed') return '可以在对话中调整产品事实后重试'
    if (currentTask.value?.status === 'cancelled') return '已保留当前对话记录'
    if (taskTodoTotalCount.value > 0) return `${taskTodoCompletedCount.value}/${taskTodoTotalCount.value} 项已完成`
    return '正在准备 Listing'
  }
  if (currentTask.value?.status === 'failed') return '可以在对话中调整需求后重试'
  if (currentTask.value?.status === 'cancelled') return `已保留 ${completedCount.value} 张图片`
  if (currentTask.value?.status === 'partial_failed') return completedCount.value > 0 ? `${completedCount.value}/${totalCount.value} 张已完成` : '没有生成出可用图片'
  if (taskTodoTotalCount.value > 0) return `${taskTodoCompletedCount.value}/${taskTodoTotalCount.value} 项已完成`
  if (currentTask.value?.status === 'success') return `${completedCount.value} 张图片已完成`
  return totalCount.value ? `${completedCount.value}/${totalCount.value} 张已完成` : '正在准备任务'
})

function taskErrorText(error) {
  const messages = {
    smart_suite_buyer_insight_failed: '买家洞察 Agent 未能可靠拆解 6 个购买问题，本次任务已停止，没有继续使用通用问题生成。请直接回复“开始生成”重试。',
    smart_suite_copywriter_failed: '文案 Agent 没有可靠生成当前阶段的标题、副文案和标签，本次任务已停止，没有回退到旧文案。请直接回复“开始生成”重试。',
    smart_suite_suite_prompt_confirmation_failed: '整套提示词 Agent 没有完整确认每张图的产品锁、画面、文案位置和负面约束，本次任务已停止，没有提交生图。请调整资料后重试。',
    smart_suite_stage_planning_failed: '套图策划 Agent 没有完整规划当前阶段的每张图片，本次任务已停止，没有使用通用模板继续生图。请直接回复“开始生成”重试。',
    smart_suite_image_plan_incomplete: '当前图片缺少完整的单图策划字段，已停止调用生图模型，没有使用通用模板继续生成。请直接回复“开始生成”重试。',
  }
  return messages[String(error || '').trim()] || String(error || '').trim()
}

function messageImageUrl(url) {
  return String(url || '').startsWith('data:') ? url : resolveImageUrl(url)
}

function styleSuggestionDescription(suggestion) {
  const descriptions = {
    '交给 AI 自主设计': '根据产品、品类、平台和站点制定整套风格',
    '真实生活场景': '自然使用状态，重视真实感和代入感',
    '高端品牌场景': '克制布光与真实环境，突出品质感',
    '户外自然光': '自然环境与明快光线，画面更有呼吸感',
    '现代家居场景': '现代空间与真实陈设，兼顾产品和使用环境',
    '上传风格参考图': '参考它的场景、构图、光线与色调',
  }
  return descriptions[suggestion] || ''
}

function imageStatusText(image) {
  if (image.status === 'failed') return '生成失败'
  if (image.status === 'cancelled') return '已停止'
  if (image.status === 'reviewing' || image.review_status === 'reviewing') return '审核中'
  if (image.review_status === 'correcting') return '重做中'
  return '生成中'
}

function imageReviewStatusText(status) {
  const labels = { reviewing: '审核中', correcting: '正在重做', auto_corrected: '重做版本已生成', unresolved: '审核未通过', correction_failed: '重做失败', degraded: '审核异常' }
  return labels[String(status || '')] || '审核中'
}

function pendingImageCopy(image) {
  if (image.status === 'failed') return image.error_message || '生成失败'
  if (image.status === 'cancelled') return '任务已停止'
  return '画面完成后会显示在这里'
}

function isImageGenerationPending(image) {
  return ['generating', 'reviewing', 'correcting'].includes(currentTask.value?.status) && image.status !== 'success' && !['failed', 'cancelled'].includes(image.status) && !image.image_url
}

function imageGenerationPrompt(image) {
  return image.title || currentStageDetail.value?.activity || 'creating the next product scene'
}

function candidateStatusText(image) {
  if (image.candidate_status === 'generating') return '正在生成修改版本'
  if (image.candidate_status === 'cancelled') return '图片修改已停止'
  return image.candidate_error_message || '修改版本生成失败'
}

function currentImageUrl(image) {
  if (image?.candidate_status === 'auto_corrected' && image?.candidate_image_url) return image.candidate_image_url
  return image?.image_url || ''
}

function openImagePreview(image, group, imageUrl = currentImageUrl(image), versionLabel = '') {
  if (!imageUrl) return
  const version = imageUrl === image.review_source_image_url
    ? 'review_source'
    : imageUrl === image.candidate_image_url
      ? 'candidate'
      : 'original'
  previewImage.value = {
    id: image.id,
    url: resolveImageUrl(imageUrl),
    label: `${group.label} ${String(image.image_index).padStart(2, '0')}`,
    title: versionLabel || image.title || image.role || '未命名画面',
    image,
    group,
    candidate: imageUrl === image.candidate_image_url,
    version,
  }
}

function closeImagePreview() {
  previewImage.value = null
}

function openAuditComparison(image, group) {
  if (!image?.review_source_image_url || !image?.candidate_image_url) return
  auditComparison.value = {
    label: `${group.label} ${String(image.image_index).padStart(2, '0')}`,
    beforeUrl: resolveImageUrl(image.review_source_image_url),
    afterUrl: resolveImageUrl(image.candidate_image_url),
  }
  closeImagePreview()
}

function closeAuditComparison() {
  auditComparison.value = null
}

function startEditingImage(image, group) {
  if (!image.image_url || isTaskActive.value) return
  selectedImage.value = {
    id: image.id,
    label: `${group.label} ${String(image.image_index).padStart(2, '0')}「${image.title || image.role || '未命名画面'}」`,
    hasCandidate: image.candidate_status === 'ready',
  }
  closeImagePreview()
  nextTick(() => composerTextarea.value?.focus())
}

function editPreviewImage() {
  if (!isTaskActive.value && previewImage.value?.image && previewImage.value?.group) startEditingImage(previewImage.value.image, previewImage.value.group)
}

function clearSelectedImage() {
  selectedImage.value = null
}

function handleComposerKeydown(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    if (!isTaskActive.value) submitMessage()
  }
}

function resizeComposer() {
  const textarea = composerTextarea.value
  if (!textarea) return
  textarea.style.height = 'auto'
  textarea.style.height = `${Math.min(textarea.scrollHeight, 132)}px`
}

function scrollToBottom() {
  nextTick(() => {
    if (chatMessagesEl.value) chatMessagesEl.value.scrollTop = chatMessagesEl.value.scrollHeight
  })
}

function appendLocalMessage(role, text = '', imageUrl = '') {
  const id = `local-${Date.now()}-${Math.random()}`
  conversation.value.messages = [...(conversation.value.messages || []), {
    id,
    role,
    text,
    image_url: imageUrl,
    at: new Date().toISOString(),
  }]
  scrollToBottom()
  return id
}

function evaluationRequestSummary(form) {
  const heading = `评估现有 ${form.platformName || '平台'} Listing`
  const details = [`平台：${form.platformName || form.platform}`, `站点：${form.countryName || form.countryCode}`]
  {
    if (form.productName) details.push(`产品名称：${form.productName}`)
    if (form.productCategory) details.push(`产品品类：${form.productCategory}`)
    if (form.productFeatures) details.push(`产品信息：${form.productFeatures}`)
    if (form.productProof) details.push(`证明 / 合规信息：${form.productProof}`)
    const listing = evaluationListingPayload(form)
    if (listing) details.push(listing)
  }
  return [heading, ...details].filter(Boolean).join('\n')
}

async function loadConversationHistory() {
  try {
    const response = await request.get('/smart-suite/conversations')
    conversationHistory.value = Array.isArray(response.data?.items) ? response.data.items : []
  } catch {
    conversationHistory.value = []
  }
}

async function restoreConversation(sessionId = '') {
  thinking.value = true
  errorMessage.value = ''
  try {
    const response = await request.get('/smart-suite/conversation', { params: sessionId ? { session_id: sessionId } : {} })
    conversation.value = response.data
    listingEvaluationResult.value = response.data?.context?.listing_evaluation || null
    requirementFormVisible.value = Boolean(response.data?.requirement_form_open)
    if (response.data?.task_id) {
      await restoreTask(response.data.task_id)
    } else {
      currentTask.value = null
    }
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '对话加载失败，请刷新重试'
  } finally {
    await loadConversationHistory()
    thinking.value = false
    scrollToBottom()
  }
}

async function openConversation(record) {
  const sessionId = String(record?.session_id || '').trim()
  if (!sessionId || sessionId === conversation.value.session_id) return
  stopPolling()
  selectedImage.value = null
  previewImage.value = null
  pendingAttachment.value = null
  await restoreConversation(sessionId)
}

async function deleteConversation(record) {
  const sessionId = String(record?.session_id || '').trim()
  if (!sessionId || deletingConversation.value) return

  deletingConversation.value = sessionId
  errorMessage.value = ''
  try {
    await request.delete(`/smart-suite/conversations/${encodeURIComponent(sessionId)}`)
    conversationHistory.value = conversationHistory.value.filter(item => item.session_id !== sessionId)
    if (sessionId === conversation.value.session_id) {
      stopPolling()
      selectedImage.value = null
      previewImage.value = null
      await restoreConversation()
    }
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '删除对话失败，请重试'
  } finally {
    deletingConversation.value = ''
  }
}

async function restoreTask(taskId) {
  try {
    const response = await request.get(`/ecom-suite/tasks/${taskId}`)
    currentTask.value = response.data
    upsertConversationTask(response.data)
    if (activeStatuses.includes(response.data?.status) || response.data?.images?.some(image => image.candidate_status === 'generating')) {
      pollTask(taskId)
    }
  } catch {
    currentTask.value = null
  }
}

function upsertConversationTask(task) {
  if (!task?.task_id) return
  trackGroupActivity(task)
  const history = Array.isArray(conversation.value.task_history) ? [...conversation.value.task_history] : []
  const index = history.findIndex(item => item?.task_id === task.task_id)
  if (index >= 0) history[index] = task
  else history.push(task)
  conversation.value.task_history = history
  conversation.value.task_ids = history.map(item => item.task_id).filter(Boolean)
}

function resetInitialRequirementsForm() {
  const defaultModel = drawingModelOptions.value.find(option => option.is_default)?.name || drawingModelOptions.value[0]?.name || ''
  const requestType = normalizeClientRequestType(conversation.value.request_type || 'suite')
  initialForm.value = { ...createInitialForm(), requestType, drawingModel: ['listing', 'listing_evaluation'].includes(requestType) ? '' : defaultModel }
  initialProductImages.value = []
  initialFormError.value = ''
  initialFormErrors.value = {}
	requirementFormVisible.value = false
  if (onboardingFileInput.value) onboardingFileInput.value.value = ''
}

function normalizeClientRequestType(value) {
  const normalized = String(value || '').trim().toLowerCase()
  return ['listing', 'listing_evaluation', 'suite_and_listing'].includes(normalized) ? normalized : 'suite'
}

function chooseEntryMode(option) {
  if (!option || thinking.value || uploading.value || startingTask.value) return
  initialForm.value.requestType = option.requestType
  if (option.requestType === 'listing_evaluation') {
    initialForm.value.platformPreset = ''
    initialForm.value.platform = ''
    initialForm.value.platformName = ''
    initialForm.value.countryCode = ''
    initialForm.value.countryName = ''
  }
  if (option.requestType !== 'suite' && option.requestType !== 'listing_evaluation' && initialForm.value.platform !== 'amazon') {
    selectPlatformPreset(listingPlatformPresets[0])
  }
  if (option.requestType === 'listing' || option.requestType === 'listing_evaluation') {
    initialForm.value.drawingModel = ''
  } else if (!initialForm.value.drawingModel) {
    initialForm.value.drawingModel = drawingModelOptions.value.find(model => model.is_default)?.name || drawingModelOptions.value[0]?.name || ''
  }
  initialFormError.value = ''
  initialFormErrors.value = {}
  requirementFormVisible.value = true
}

function selectPlatformPreset(preset) {
  initialForm.value.platformPreset = preset.id
  initialForm.value.platform = preset.platform
  initialForm.value.platformName = preset.platformName
  initialForm.value.countryCode = preset.countryCode
  initialForm.value.countryName = preset.countryName
}

function selectVisualPreset(preset) {
  initialForm.value.visualPreset = preset.id
  initialForm.value.visualStyle = preset.value
  if (preset.id !== 'custom') initialForm.value.styleRequirements = ''
}

function normalizeCustomPlatform(value) {
  return String(value || '').trim().toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

async function handleInitialImageChange(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length) return
  if (files.some(file => !file.type.startsWith('image/'))) {
    initialFormErrors.value = { ...initialFormErrors.value, image: '请上传 PNG、JPG 或 WEBP 图片' }
    return
  }
  const remaining = maxInitialProductImages.value - initialProductImages.value.length
  if (remaining <= 0) return
  try {
    const additions = await Promise.all(files.slice(0, remaining).map(async file => ({
      id: `${file.name}-${file.size}-${file.lastModified}-${crypto.randomUUID()}`,
      name: file.name,
      previewUrl: await readFileAsDataURL(file),
      file,
      purpose: 'product_image',
    })))
    initialProductImages.value = [...initialProductImages.value, ...additions]
    const { image, ...rest } = initialFormErrors.value
    initialFormErrors.value = rest
    if (files.length > remaining) initialFormError.value = `当前模式最多上传 ${maxInitialProductImages.value} 张图片，超出的图片未添加。`
  } catch (error) {
    initialFormErrors.value = { ...initialFormErrors.value, image: error?.message || '产品图读取失败，请重试' }
  }
}

function setInitialPrimaryImage(index) {
  if (index <= 0 || index >= initialProductImages.value.length) return
  const images = [...initialProductImages.value]
  const [primary] = images.splice(index, 1)
  images.unshift(primary)
  initialProductImages.value = images
}

function removeInitialProductImage(index) {
  initialProductImages.value = initialProductImages.value.filter((_, imageIndex) => imageIndex !== index)
  if (onboardingFileInput.value) onboardingFileInput.value.value = ''
}

function validateInitialRequirements() {
  const form = initialForm.value
  const errors = {}
  if (form.requestType === 'listing_evaluation' && !form.platform) errors.platform = '请先选择评估平台'
  if (form.requestType === 'listing_evaluation' && form.platform && !evaluationListingComplete(form)) errors.existingListing = '请补全当前平台要求的 Listing 字段'
  const imagesRequired = ['listing', 'suite', 'suite_and_listing'].includes(form.requestType) || (form.requestType === 'listing_evaluation' && form.platform === 'lowes')
  if (imagesRequired && !initialProductImages.value.length) errors.image = '请至少上传一张产品图'
  if (!form.productName.trim()) errors.productName = '请填写产品名称'
  if (!form.productFeatures.trim()) errors.productFeatures = '请填写尺寸、材质、功能或已确认卖点'
  const platformValue = form.platformPreset === 'custom' ? normalizeCustomPlatform(form.platformName) : form.platform.trim()
  const customMarket = ['custom', 'amazon_custom'].includes(form.platformPreset)
  if (form.requestType !== 'listing_evaluation' && (!platformValue || !form.countryCode.trim() || (customMarket && !form.countryName.trim()))) {
    errors.platform = '请完整填写平台、国家 / 站点和两位国家代码'
  }
  const selectedOutputs = outputPresets.filter(output => form[output.enabledKey])
  if (!['listing', 'listing_evaluation'].includes(form.requestType) && (!selectedOutputs.length || selectedOutputs.some(output => !Number.isInteger(Number(form[output.countKey])) || Number(form[output.countKey]) < 1 || Number(form[output.countKey]) > output.max))) {
    errors.output = '请至少选择一种套图，并填写有效数量'
  }
  if (!['listing', 'listing_evaluation'].includes(form.requestType) && form.visualPreset === 'custom' && !form.styleRequirements.trim()) errors.visual = '请填写自定义视觉要求'
  if (!['listing', 'listing_evaluation'].includes(form.requestType) && !form.drawingModel) errors.model = '请选择生图模型'
  initialFormErrors.value = errors
  return Object.keys(errors).length === 0
}

function evaluationListingPayload(form) {
  const lines = value => String(value || '').split('\n').map(item => item.trim()).filter(Boolean)
  const parts = []
  const add = (label, value) => { if (String(value || '').trim()) parts.push(label + ':\n' + String(value).trim()) }
  const addLines = (label, value) => { const values = lines(value); if (values.length) parts.push(label + ':\n' + values.map((item, index) => (index + 1) + '. ' + item).join('\n')) }
  if (form.platform === 'home_depot') { add('Title', form.listingTitle); addLines('Salient Bullets', form.listingSalientBullets); addLines('Bullet Points', form.listingBullets); add('About This Product', form.listingDescription) }
  else if (form.platform === 'amazon') { add('Title', form.listingTitle); addLines('Bullet Points', form.listingBullets); add('Description', form.listingDescription); add('Search Terms', form.listingSearchTerms) }
  else if (form.platform === 'tiktok') { add('Title', form.listingTitle); add('Description', form.listingDescription); add('Attributes', form.tiktokAttributes); add('Variants', form.tiktokVariants); add('Package Contents', form.tiktokPackageContents) }
  else if (form.platform === 'lowes') { add('Title', form.listingTitle); add('Description', form.listingDescription); addLines('Bullets', form.listingBullets) }
  else if (form.platform === 'wayfair') { add('About This Product', form.wayfairDescription); addLines('Feature Bullets', form.wayfairFeatures) }
  return parts.join('\n\n')
}

function evaluationListingComplete(form) {
  if (form.platform === 'amazon') return form.listingTitle.trim() && form.listingBullets.trim() && form.listingDescription.trim()
  if (form.platform === 'tiktok') return form.listingTitle.trim() && form.listingDescription.trim()
  if (form.platform === 'lowes') return form.listingTitle.trim() && form.listingDescription.trim() && form.listingBullets.trim()
  if (form.platform === 'wayfair') return form.wayfairDescription.trim() && form.wayfairFeatures.trim()
  if (form.platform === 'home_depot') return form.listingTitle.trim() && form.listingSalientBullets.trim() && form.listingBullets.trim() && form.listingDescription.trim()
  return false
}

async function submitInitialRequirements() {
  if (submittingRequirements.value || !validateInitialRequirements()) return
  const form = initialForm.value
  const listingEvaluation = form.requestType === 'listing_evaluation'
  submittingRequirements.value = true
  uploading.value = true
  initialFormError.value = ''
  listingEvaluationError.value = ''
  try {
    const listingOnly = form.requestType === 'listing'
    if (listingEvaluation) {
	  const [image] = await Promise.all(initialProductImages.value.slice(0, 1).map(uploadPendingAttachment))
      const requestSummary = evaluationRequestSummary(form)
      conversation.value.messages = (conversation.value.messages || []).filter(message => String(message?.text || '').trim() !== '你今天想完成什么？先选一个方向，也可以直接描述你的目标。')
      if (!(conversation.value.messages || []).some(message => message?.role === 'user' && String(message?.text || '').trim() === requestSummary)) {
        appendLocalMessage('user', requestSummary)
      }
      closeRequirementForm()
      const response = await request.post('/smart-suite/listing/evaluate', {
        session_id: conversation.value.session_id,
        request_summary: requestSummary,
        platform: form.platform, platform_name: form.platformName, country_code: form.countryCode, country_name: form.countryName,
        internet_number: form.internetNumber,
        product_name: form.productName, product_category: form.productCategory, product_features: form.productFeatures, product_proof: form.productProof,
        product_length: form.productLength, product_width: form.productWidth, product_height: form.productHeight, product_dimension_unit: form.productDimensionUnit,
        product_weight: form.productWeight, product_weight_unit: form.productWeightUnit, package_contents: form.packageContents,
        image_url: image || '',
        existing_listing: evaluationListingPayload(form),
      })
      if (response.data?.conversation) conversation.value = response.data.conversation
      listingEvaluationResult.value = response.data
      requirementFormVisible.value = false
      initialProductImages.value = []
      return
    }
    const selectedProductImages = form.requestType === 'listing'
      ? initialProductImages.value.slice(0, 1)
      : initialProductImages.value
    const imageUrls = ['listing', 'suite', 'suite_and_listing'].includes(form.requestType)
      ? await Promise.all(selectedProductImages.map(uploadPendingAttachment))
      : []
    const [imageUrl = '', ...productReferenceUrls] = imageUrls
    const visualStyle = form.visualPreset === 'custom' ? form.styleRequirements.trim() : form.visualStyle
    const response = await request.post('/smart-suite/conversation/turn', {
      session_id: conversation.value.session_id,
      image_url: imageUrl,
      product_reference_urls: productReferenceUrls,
      listing_keywords: form.listingKeywords,
      image_purpose: 'product_image',
      drawing_model: listingOnly ? '' : form.drawingModel,
      request_type: form.requestType,
      initial_requirements: {
        request_type: form.requestType,
        product_name: form.productName,
        product_category: form.productCategory,
        product_brand: form.productBrand,
        product_features: form.productFeatures,
        product_proof: form.productProof,
        product_length: form.productLength,
        product_width: form.productWidth,
        product_height: form.productHeight,
        product_dimension_unit: form.productDimensionUnit,
        product_weight: form.productWeight,
        product_weight_unit: form.productWeightUnit,
        listing_keywords: form.listingKeywords,
        package_length: form.packageLength,
        package_width: form.packageWidth,
        package_height: form.packageHeight,
        package_dimension_unit: form.packageDimensionUnit,
        package_weight: form.packageWeight,
        package_weight_unit: form.packageWeightUnit,
        package_contents: form.packageContents,
        product_reference_urls: productReferenceUrls,
        platform: form.platformPreset === 'custom' ? normalizeCustomPlatform(form.platformName) : form.platform,
        platform_name: form.platformName,
        country_code: form.countryCode,
        country_name: form.countryName,
        secondary_count: form.secondaryEnabled ? Number(form.secondaryCount) : 0,
        desktop_a_plus_count: form.desktopEnabled ? Number(form.desktopCount) : 0,
        mobile_a_plus_count: form.mobileEnabled ? Number(form.mobileCount) : 0,
        visual_style: listingOnly ? '' : visualStyle,
        style_requirements: listingOnly ? '' : form.styleRequirements,
      },
    })
    conversation.value = response.data
    requirementFormVisible.value = false
    currentTask.value = null
    initialProductImages.value = []
    if (listingOnly && response.data?.ready) {
      await startPlanning(response.data.context)
    }
    await loadConversationHistory()
    scrollToBottom()
  } catch (error) {
    const message = error?.response?.data?.message || error?.response?.data?.error || error?.message || '需求保存失败，请重试'
    initialFormError.value = message
    if (listingEvaluation) {
      requirementFormVisible.value = false
      listingEvaluationError.value = `评估失败：${message}`
      scrollToBottom()
    }
  } finally {
    uploading.value = false
    submittingRequirements.value = false
  }
}

function startNewConversation() {
  if (isTaskActive.value) {
    errorMessage.value = '当前任务仍在生成，请先停止后再新建对话'
    return
  }
  stopPolling()
  const modelOptions = drawingModelOptions.value
  conversation.value = {
    session_id: `new-${crypto.randomUUID()}`,
    task_id: '',
    task_ids: [],
    task_history: [],
    context: emptyContext(),
    ready: false,
    missing_fields: [],
    request_type: '',
    requirement_form_open: false,
    model_options: modelOptions,
    messages: [{
      id: `greeting-${Date.now()}`,
      role: 'assistant',
      text: '你今天想完成什么？先选一个方向，也可以直接描述你的目标。',
      at: new Date().toISOString(),
    }],
  }
  currentTask.value = null
  taskTodoCollapsed.value = false
  selectedImage.value = null
  previewImage.value = null
  pendingAttachment.value = null
  resetInitialRequirementsForm()
  listingEvaluationResult.value = null
  listingEvaluationError.value = ''
  errorMessage.value = ''
  scrollToBottom()
}

async function submitMessage() {
  const text = chatInput.value.trim()
  if ((!text && !pendingAttachment.value) || thinking.value || uploading.value || startingTask.value || candidateActionLoading.value) return
  const attachment = pendingAttachment.value
  let imageUrl = ''
  chatInput.value = ''
  nextTick(resizeComposer)
  uploading.value = Boolean(attachment)
  errorMessage.value = ''
  try {
    if (attachment) imageUrl = await uploadPendingAttachment(attachment)
    const sent = await sendConversationTurn({
      text,
      answerField: currentConversationAnswerField(text, attachment?.purpose || ''),
      imageUrl,
      previewUrl: attachment?.previewUrl || '',
      imagePurpose: attachment?.purpose || '',
    })
    if (sent) {
      pendingAttachment.value = null
    } else if (text) {
      chatInput.value = text
      nextTick(resizeComposer)
    }
  } catch (error) {
    if (text) {
      chatInput.value = text
      nextTick(resizeComposer)
    }
    errorMessage.value = error?.response?.data?.error || error?.message || '图片上传失败，请重试'
  } finally {
    uploading.value = false
  }
}

function chooseSuggestion(suggestion) {
  if (!suggestion || thinking.value || uploading.value || startingTask.value || candidateActionLoading.value) return
  if (suggestion === '上传风格参考图') {
    fileInput.value?.click()
    return
  }
  sendConversationTurn({ text: suggestion, answerField: conversation.value.suggestion_field || currentSuggestionAnswerField() })
}

function chooseDrawingModel(model) {
  const name = String(model?.name || '').trim()
  if (!name || thinking.value || uploading.value || startingTask.value || candidateActionLoading.value) return
  sendConversationTurn({ text: `使用模型：${name}`, drawingModel: name })
}

function currentSuggestionAnswerField() {
  const missing = Array.isArray(conversation.value.missing_fields) ? conversation.value.missing_fields : []
  return ['platform_site', 'output_counts'].find(field => missing.includes(field)) || ''
}

function currentConversationAnswerField(text = '', imagePurpose = '') {
  if (selectedImage.value || isTaskActive.value) return ''
  if (imagePurpose === 'product_image' && String(text || '').trim()) return 'product_info'
  if (['product_info', 'platform_site', 'output_counts'].includes(conversation.value.suggestion_field)) return conversation.value.suggestion_field
  const missing = Array.isArray(conversation.value.missing_fields) ? conversation.value.missing_fields : []
  if (missing.includes('product_name') || missing.includes('product_features')) return 'product_info'
  return currentSuggestionAnswerField()
}

async function sendConversationTurn({ text = '', answerField = '', imageUrl = '', previewUrl = '', imagePurpose = '', drawingModel = '', requestType = '', targetImageId, hasCandidate } = {}) {
  if (thinking.value || startingTask.value) return
  const target = targetImageId ?? selectedImage.value?.id ?? 0
  const candidate = hasCandidate ?? selectedImage.value?.hasCandidate ?? false
  const previousMessages = conversation.value.messages || []
  appendLocalMessage('user', text, previewUrl || imageUrl)
  thinking.value = true
  errorMessage.value = ''
  await nextTick()
  try {
    const response = await request.post('/smart-suite/conversation/turn', {
      session_id: conversation.value.session_id,
      text,
      answer_field: answerField,
      image_url: imageUrl,
      image_purpose: imagePurpose,
      drawing_model: drawingModel,
      request_type: requestType,
      target_image_id: target,
      has_candidate: candidate,
    })
    conversation.value = response.data
	requirementFormVisible.value = Boolean(response.data?.requirement_form_open)
    if (response.data?.requirement_form_open) {
      initialForm.value.requestType = normalizeClientRequestType(response.data?.request_type)
    }
    try {
      await executeConversationIntent(response.data, text, target)
    } catch (actionError) {
      errorMessage.value = actionError?.response?.data?.error || actionError?.message || '操作没有完成，请重试'
    }
    await loadConversationHistory()
    return true
  } catch (error) {
    conversation.value.messages = previousMessages
    errorMessage.value = error?.response?.data?.error || '这条消息没有发送成功，请重试'
    return false
  } finally {
    thinking.value = false
    scrollToBottom()
  }
}

async function executeConversationIntent(result, originalText, requestedTargetImageId = 0) {
  const targetImageId = Number(result?.target_image_id || 0)
  switch (result?.intent) {
    case 'start':
      if (requestedTargetImageId) {
        await startImageEdit(Number(requestedTargetImageId), originalText)
      } else if (!result.ready) {
        appendLocalMessage('assistant', '还缺少必要信息，暂时不能开始生成。请继续补充上面提到的内容。')
      } else if (!isTaskActive.value) {
        await startPlanning(result.context)
      }
      break
    case 'stop':
      await stopCurrentTask()
      break
    case 'edit_image':
      await startImageEdit(targetImageId, originalText)
      break
    case 'accept_candidate':
      await handleCandidateAction(targetImageId, 'accept')
      break
    case 'reject_candidate':
      await handleCandidateAction(targetImageId, 'reject')
      break
  }
}

async function startPlanning(context) {
  if (!context?.product_name || !context?.product_features || startingTask.value) return
  startingTask.value = true
  try {
    const response = await request.post('/smart-suite/tasks', {
      session_id: conversation.value.session_id,
      white_bg_image_url: context.white_bg_image_url,
      product_reference_urls: context.product_reference_urls || [],
      listing_keywords: context.listing_keywords || '',
      product_name: context.product_name,
      product_category: context.product_category,
      product_brand: context.product_brand,
      product_features: context.product_features,
      product_proof: context.product_proof,
      product_length: context.product_length,
      product_width: context.product_width,
      product_height: context.product_height,
      product_dimension_unit: context.product_dimension_unit,
      product_weight: context.product_weight,
      product_weight_unit: context.product_weight_unit,
      package_length: context.package_length,
      package_width: context.package_width,
      package_height: context.package_height,
      package_dimension_unit: context.package_dimension_unit,
      package_weight: context.package_weight,
      package_weight_unit: context.package_weight_unit,
      package_contents: context.package_contents,
      platform: context.platform,
      platform_name: context.platform_name,
      country_code: context.country_code,
      country_name: context.country_name,
      language: context.language,
      output_target: context.output_target,
      include_listing: Boolean(context.include_listing),
      secondary_count: context.secondary_count,
      desktop_a_plus_count: context.desktop_a_plus_count,
      mobile_a_plus_count: context.mobile_a_plus_count,
      visual_style: context.visual_style,
      style_requirements: context.style_requirements,
      style_reference_urls: context.style_reference_urls || [],
      drawing_model: context.drawing_model,
    })
    const taskId = response.data?.task_id
    if (!taskId) throw new Error('任务创建接口没有返回任务 ID')
    if (currentTask.value?.task_id) upsertConversationTask(currentTask.value)
    conversation.value.task_id = taskId
    currentTask.value = {
      task_id: taskId,
      status: response.data?.status || 'pending',
      workflow_type: context.output_target === 'listing' ? 'smart_listing' : 'smart_suite',
      output_target: context.output_target,
      total_image_count: response.data?.total_image_count || 0,
      include_listing: Boolean(context.include_listing),
      images: [],
      created_at: new Date().toISOString(),
    }
    upsertConversationTask(currentTask.value)
    selectedImage.value = null
    pollTask(taskId)
  } catch (error) {
	const reason = error?.response?.data?.message || error?.response?.data?.error || error?.message || '未知错误'
    appendLocalMessage('assistant', `任务没有创建成功：${reason}。请修正后重新提交。`)
    throw error
  } finally {
    startingTask.value = false
  }
}

async function startImageEdit(imageId, prompt) {
  const taskId = currentTask.value?.task_id
  const image = currentTask.value?.images?.find(item => item.id === imageId)
  if (!taskId || !imageId || !image?.image_url) {
    appendLocalMessage('assistant', '我还不能确定你要修改哪张图。请先点选一张已完成的图片，再告诉我如何修改。')
    return
  }
  await request.post(`/smart-suite/tasks/${taskId}/images/${imageId}/edit`, { prompt })
  selectedImage.value = { ...selectedImage.value, id: imageId, hasCandidate: false }
  await refreshCurrentTask()
  pollTask(taskId)
}

async function sendImageCommand(image, group, text) {
  if (thinking.value || candidateActionLoading.value) return
  startEditingImage(image, group)
  await sendConversationTurn({ text, targetImageId: image.id, hasCandidate: true })
}

async function handleCandidateAction(imageId, action) {
  const taskId = currentTask.value?.task_id
  const image = currentTask.value?.images?.find(item => item.id === imageId)
  if (!taskId || !imageId || image?.candidate_status !== 'ready' || !image?.candidate_image_url) {
    appendLocalMessage('assistant', '我还不能确定你指的是哪张修改版本，请先点选图片。')
    return
  }
  candidateActionLoading.value = true
  try {
    const response = await request.post(`/smart-suite/tasks/${taskId}/images/${imageId}/${action}`, { candidate_image_url: image.candidate_image_url })
    currentTask.value = response.data?.task || currentTask.value
    upsertConversationTask(currentTask.value)
    syncImagePresentation()
    selectedImage.value = null
  } finally {
    candidateActionLoading.value = false
  }
}

async function handleFileChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    errorMessage.value = '请上传 PNG、JPG 或 WEBP 图片'
    return
  }
  errorMessage.value = ''
  try {
    const dataUrl = await readFileAsDataURL(file)
    const hasProductImage = Boolean(conversation.value.context?.white_bg_image_url)
    pendingAttachment.value = { name: file.name, previewUrl: dataUrl, file, purpose: hasProductImage ? 'style_reference' : 'product_image' }
  } catch (error) {
    errorMessage.value = error?.message || '图片读取失败，请重试'
  }
}

async function uploadPendingAttachment(attachment) {
  if (!attachment?.previewUrl) throw new Error('图片内容为空，请重新选择')
  const [meta, data] = String(attachment.previewUrl).split(',')
  const mimeType = String(meta || '').split(':')[1]?.split(';')[0] || attachment.file?.type || 'image/png'
  const response = await request.post('/upload/image', { data, mime_type: mimeType })
  const url = response.data?.url || ''
  if (!url) throw new Error('图片上传接口没有返回地址')
  return url
}

function removePendingAttachment() {
  pendingAttachment.value = null
  if (fileInput.value) fileInput.value.value = ''
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('read_image_failed'))
    reader.readAsDataURL(file)
  })
}

async function requestStop() {
  errorMessage.value = ''
  try {
    await stopCurrentTask()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || '停止任务失败，请重试'
  }
}

async function stopCurrentTask() {
  const taskId = currentTask.value?.task_id || activePollTaskId
  if (!taskId || stopping.value) return
  stopping.value = true
  try {
    const response = await request.post(`/smart-suite/tasks/${taskId}/stop`)
    currentTask.value = response.data?.task || currentTask.value
    stopPolling()
  } finally {
    stopping.value = false
  }
}

async function refreshCurrentTask() {
  const taskId = currentTask.value?.task_id
  if (!taskId) return
  const response = await request.get(`/ecom-suite/tasks/${taskId}`)
  currentTask.value = response.data
  upsertConversationTask(response.data)
  syncImagePresentation()
}

function pollTask(taskId) {
  stopPolling()
  activePollTaskId = taskId
  const fetchTask = async () => {
    try {
      const response = await request.get(`/ecom-suite/tasks/${taskId}`)
      if (activePollTaskId !== taskId) return
      currentTask.value = response.data
      upsertConversationTask(response.data)
      syncImagePresentation()
      const candidateGenerating = response.data?.images?.some(image => image.candidate_status === 'generating')
      if (!activeStatuses.includes(response.data?.status) && !candidateGenerating) stopPolling()
    } catch {
      // Transient polling failures are retried by the next interval.
    }
  }
  fetchTask()
  pollingTimer.value = window.setInterval(fetchTask, 3000)
}

function stopPolling() {
  activePollTaskId = ''
  if (pollingTimer.value) {
    window.clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

async function fetchBrandOptions() {
  try {
    const response = await request.get('/download-options')
    if (Array.isArray(response.data?.brands) && response.data.brands.length) {
      brandOptions.value = response.data.brands
    }
  } catch {
    // 与下载弹窗一致：接口不可用时保留公共兜底品牌。
  }
}

onMounted(() => {
  restoreConversation()
  fetchBrandOptions()
})
onBeforeUnmount(() => {
  stopPolling()
})
</script>

<style scoped>
:global(body) { background: #f6f7f8; }
.suite-page { min-height: calc(100vh - 64px); padding: 20px clamp(14px, 3vw, 38px) 28px; color: #182033; }
.conversation-shell { width: min(1020px, 100%); height: calc(100vh - 150px); min-height: 570px; margin: 0 auto; display: grid; grid-template-rows: minmax(0, 1fr) auto; overflow: hidden; border: 1px solid #dfe4e9; border-radius: 8px; background: #fff; box-shadow: 0 10px 30px rgba(29, 41, 57, .055); }
.listing-evaluation-result { margin: 0 0 24px; border: 1px solid #dfe4e9; border-radius: 12px; padding: 18px; background: #fff; }
.conversation-body { display: flex; flex-direction: column; }
.conversation-timeline { order: 1; }
.conversation-body > .processing-row { order: 2; }
.conversation-body > .listing-evaluation-error-row { order: 3; }
.conversation-body > .listing-evaluation-result { order: 3; }
.conversation-body > .conversation-empty { order: 1; }
.listing-evaluation-error-row .message-bubble { color: #914d46; }
.listing-evaluation-error-row button { margin-top: 8px; padding: 0; border: 0; color: #6e3934; background: transparent; cursor: pointer; font: inherit; font-size: 12px; font-weight: 650; text-decoration: underline; text-underline-offset: 3px; }
.listing-source-preview { margin: -18px -18px 18px; padding: 18px; border-bottom: 1px solid #dfe4e9; background: #fafbfc; }
.listing-source-preview-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 14px; }
.listing-source-preview-head div { display: flex; flex-direction: column; gap: 2px; }
.listing-source-preview-head span { color: #f96302; font-size: 11px; font-weight: 600; }
.listing-source-preview-head strong { color: #182033; font-size: 16px; }
.listing-source-preview-head a { color: #475467; font-size: 12px; text-decoration: underline; text-underline-offset: 2px; }
.listing-source-meta { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); margin: 0 0 14px; border: 1px solid #e4e7ec; border-radius: 8px; overflow: hidden; background: #fff; }
.listing-source-meta > div { min-width: 0; padding: 9px 10px; border-left: 1px solid #e4e7ec; }
.listing-source-meta > div:first-child { border-left: 0; }
.listing-source-meta dt, .listing-source-specifications dt { color: #667085; font-size: 10px; }
.listing-source-meta dd { margin: 3px 0 0; overflow: hidden; color: #344054; font-size: 12px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.listing-source-field { padding: 13px 0; border-top: 1px solid #e4e7ec; }
.listing-source-field > strong { color: #344054; font-size: 12px; }
.listing-source-field p { margin: 7px 0 0; color: #475467; font-size: 13px; line-height: 1.65; white-space: pre-wrap; }
.listing-source-field ul { margin: 7px 0 0; padding-left: 18px; color: #475467; font-size: 13px; line-height: 1.6; }
.listing-source-field li + li { margin-top: 4px; }
.listing-source-specifications { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); margin: 8px 0 0; border: 1px solid #e4e7ec; border-radius: 8px; overflow: hidden; background: #fff; }
.listing-source-specifications > div { min-width: 0; padding: 8px 10px; border-top: 1px solid #e4e7ec; }
.listing-source-specifications > div:nth-child(-n + 2) { border-top: 0; }
.listing-source-specifications > div:nth-child(even) { border-left: 1px solid #e4e7ec; }
.listing-source-specifications dd { margin: 3px 0 0; color: #344054; font-size: 12px; overflow-wrap: anywhere; }
.listing-evaluation-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.listing-evaluation-head div { display: flex; align-items: baseline; gap: 12px; }
.listing-evaluation-head span { color: #667085; font-size: 13px; }
.listing-evaluation-head strong { font-size: 26px; letter-spacing: 0; }
.listing-evaluation-head em { color: #147d4b; background: #e8f7ef; border-radius: 999px; padding: 4px 10px; font-size: 12px; font-style: normal; }
.listing-evaluation-head em.fail { color: #b54708; background: #fff4e5; }
.listing-evaluation-summary { margin: 12px 0 16px; color: #475467; }
.listing-evaluation-table { border: 1px solid #eaecf0; border-radius: 8px; overflow: hidden; }
.listing-evaluation-row { display: grid; grid-template-columns: minmax(180px, 1.8fr) .6fr .6fr .7fr; gap: 12px; padding: 11px 12px; border-top: 1px solid #eaecf0; align-items: center; font-size: 13px; }
.listing-evaluation-table.is-hd-rubric .listing-evaluation-row { grid-template-columns: minmax(180px, 1.8fr) .6fr .6fr; }
.listing-evaluation-row:first-child { border-top: 0; }
.listing-evaluation-header { color: #667085; background: #f8fafc; }
.listing-evaluation-row strong { display: block; }
.listing-evaluation-row small { display: block; margin-top: 3px; color: #667085; line-height: 1.45; }
.listing-evaluation-deductions, .listing-evaluation-recommendations { margin-top: 16px; }
.listing-evaluation-deductions > strong, .listing-evaluation-recommendations > strong { display: block; margin-bottom: 8px; }
.listing-evaluation-deductions > div { display: grid; grid-template-columns: 1.2fr auto 2fr; gap: 10px; padding: 10px 0; border-top: 1px solid #f0f2f5; font-size: 13px; }
.listing-evaluation-deductions em { color: #c4320a; font-style: normal; }
.listing-evaluation-deductions small { color: #667085; }
.listing-evaluation-deductions .deduction-plan { line-height: 1.55; }
.listing-evaluation-deductions .deduction-plan strong { display: block; margin-bottom: 2px; color: #344054; font-size: 11px; }
.listing-evaluation-recommendations p { margin: 6px 0 0; color: #475467; font-size: 13px; }
.listing-optimized-result { margin-top: 20px; border-top: 1px solid #dfe3e8; }
.listing-optimized-head { display: flex; flex-direction: column; gap: 3px; padding: 16px 0 12px; }
.listing-optimized-head span { color: #168463; font-size: 12px; font-weight: 600; }
.listing-optimized-head strong { color: #101828; font-size: 16px; }
.listing-optimized-field { display: grid; grid-template-columns: minmax(140px, .32fr) minmax(0, 1fr); gap: 20px; padding: 14px 0; border-top: 1px solid #eaecf0; }
.listing-optimized-field > strong { color: #667085; font-size: 12px; font-weight: 600; }
.listing-optimized-field p, .listing-optimized-field ul { margin: 0; color: #1d2939; font-size: 14px; line-height: 1.65; }
.listing-optimized-field ul { padding-left: 20px; }
.listing-optimized-field li + li { margin-top: 6px; }
@media (max-width: 640px) {
  .listing-evaluation-result { padding: 14px; }
  .listing-source-preview { margin: -14px -14px 16px; padding: 14px; }
  .listing-source-meta { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .listing-source-meta > div:nth-child(odd) { border-left: 0; }
  .listing-source-meta > div:nth-child(n + 3) { border-top: 1px solid #e4e7ec; }
  .listing-source-specifications { grid-template-columns: 1fr; }
  .listing-source-specifications > div:nth-child(n + 2) { border-top: 1px solid #e4e7ec; }
  .listing-source-specifications > div:nth-child(even) { border-left: 0; }
  .listing-evaluation-row { grid-template-columns: minmax(130px, 1.6fr) .55fr .55fr .65fr; gap: 7px; padding: 9px 8px; font-size: 12px; }
  .listing-evaluation-table.is-hd-rubric .listing-evaluation-row { grid-template-columns: minmax(130px, 1.6fr) .55fr .55fr; }
  .listing-evaluation-deductions > div { grid-template-columns: 1fr auto; }
  .listing-evaluation-deductions small { grid-column: 1 / -1; }
  .listing-optimized-field { grid-template-columns: 1fr; gap: 6px; }
}
.conversation { min-height: 0; overflow-y: auto; overflow-anchor: none; padding: 30px clamp(18px, 5vw, 64px) 36px; background: linear-gradient(#fff 0, #fff 84%, #fbfcfd 100%); scrollbar-width: thin; scrollbar-color: #d5dbe2 transparent; }
.message-row { display: flex; align-items: flex-start; gap: 10px; margin-top: 20px; }
.conversation > div:first-child > .message-row { margin-top: 0; }
.message-row.user { flex-direction: row-reverse; }
.avatar { flex: 0 0 30px; width: 30px; height: 30px; display: grid; place-items: center; border: 1px solid #e2e6ea; border-radius: 7px; color: #55657b; background: #f5f7f9; }
.message-row.user .avatar { border-color: #253248; color: #fff; background: #253248; }
.avatar svg { width: 15px; }
.message-bubble { max-width: min(720px, calc(100% - 48px)); padding: 13px 15px; border: 1px solid #e2e6eb; border-radius: 8px 8px 8px 3px; color: #344157; background: #fff; box-shadow: 0 4px 16px rgba(37, 50, 72, .045); font-size: 13px; line-height: 1.7; }
.message-row.user .message-bubble { border-color: #253248; border-radius: 8px 8px 3px 8px; color: #fff; background: #253248; box-shadow: 0 5px 16px rgba(37, 50, 72, .14); }
.message-bubble p { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.suggestion-row { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 13px; }
.style-choice-heading { flex: 0 0 100%; display: flex; flex-direction: column; gap: 2px; margin-bottom: 2px; }
.style-choice-heading strong { color: #2f4056; font-size: 11px; font-weight: 700; }
.style-choice-heading span { color: #8a95a3; font-size: 9px; line-height: 1.4; }
.style-suggestions { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 7px; }
.style-suggestion { min-height: 54px; display: flex; align-items: center; justify-content: flex-start; gap: 8px; padding: 8px 10px; border-radius: 7px; text-align: left; }
.style-suggestion svg { flex: 0 0 16px; width: 16px; height: 16px; color: #647990; }
.style-suggestion:last-child { grid-column: 1 / -1; }
.suggestion-copy { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.suggestion-copy strong { color: #34465d; font-size: 10px; font-weight: 700; }
.suggestion-copy small { color: #8793a1; font-size: 9px; line-height: 1.35; }
.suggestion-button { min-height: 34px; padding: 7px 12px; border: 1px solid #cbd5df; border-radius: 17px; color: #34465d; background: #f7f9fa; cursor: pointer; font: inherit; font-size: 11px; line-height: 1.2; transition: transform .15s ease, border-color .15s ease, background .15s ease, color .15s ease; }
.suggestion-button:hover:not(:disabled) { transform: translateY(-1px); border-color: #72869b; color: #1f2e43; background: #edf2f5; }
.suggestion-button:disabled { cursor: default; opacity: .45; }
.model-suggestions { width: min(520px, 100%); display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 7px; margin-top: 13px; }
.model-choice-heading { grid-column: 1 / -1; display: flex; flex-direction: column; gap: 2px; margin-bottom: 2px; }
.model-choice-heading strong { color: #2f4056; font-size: 11px; font-weight: 700; }
.model-choice-heading span { color: #8a95a3; font-size: 9px; line-height: 1.4; }
.model-choice { min-width: 0; min-height: 58px; display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 9px 10px; border: 1px solid #d4dce4; border-radius: 7px; color: #34465d; background: #f8fafb; cursor: pointer; text-align: left; transition: border-color .15s ease, background .15s ease, transform .15s ease; }
.model-choice:hover:not(:disabled) { transform: translateY(-1px); border-color: #71859b; background: #f2f6f8; }
.model-choice:disabled { cursor: default; opacity: .45; }
.model-choice-copy { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.model-choice-copy strong { color: #2c3d53; font-size: 10px; font-weight: 700; line-height: 1.35; overflow-wrap: anywhere; }
.model-choice-copy small { color: #7c8998; font-size: 9px; line-height: 1.35; }
.model-choice em { flex: 0 0 auto; padding: 2px 5px; border: 1px solid #cbd9d3; border-radius: 4px; color: #38705f; background: #edf6f2; font-size: 8px; font-style: normal; line-height: 1.2; }
.uploaded-image { display: block; width: min(280px, 100%); max-height: 300px; object-fit: contain; border-radius: 6px; background: #fff; }
.uploaded-image + p { margin-top: 9px; }
.task-row { margin-top: 19px; }
.agent-live-status { display: flex; align-items: center; gap: 8px; margin: 10px 2px 0; color: #748078; font-size: 11px; line-height: 1.5; animation: agent-live-in .28s ease; }
.agent-live-status svg { flex: 0 0 14px; width: 14px; height: 14px; color: #5f8f7f; }
@keyframes agent-live-in { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
.task-message { width: min(800px, calc(100% - 38px)); min-width: 0; padding: 2px 0 4px; }
.task-todo { overflow: hidden; border: 1px solid #e2e6eb; border-radius: 8px; background: #fff; box-shadow: 0 4px 16px rgba(37, 50, 72, .045); }
.task-todo-header { display: flex; align-items: center; gap: 8px; padding: 4px 6px 4px 4px; }
.task-todo-toggle { min-width: 0; min-height: 50px; display: grid; flex: 1; grid-template-columns: 34px minmax(0, 1fr) auto; align-items: center; gap: 8px; padding: 5px 7px; border: 0; border-radius: 6px; color: inherit; background: transparent; cursor: pointer; text-align: left; }
.task-todo-toggle:hover { background: #f7f9fa; }
.task-todo-toggle:focus-visible { outline: 2px solid #9eb7ad; outline-offset: -2px; }
.task-todo-head-icon { position: relative; width: 34px; height: 28px; display: flex; align-items: center; }
.task-todo-head-check, .task-todo-list-icon, .task-todo-progress { flex: 0 0 18px; width: 18px; height: 18px; }
.task-todo-head-check circle { fill: #168a67; stroke: none; }
.task-todo-head-check path { fill: none; stroke: #fff; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
.task-todo-list-icon { fill: none; stroke: #758192; stroke-width: 1.7; stroke-linecap: round; }
.task-todo-progress { position: relative; display: block; color: #168a67; }
.task-todo-progress svg { width: 100%; height: 100%; overflow: visible; transform: rotate(-90deg); }
.task-todo-progress circle { fill: none; stroke-width: 2; stroke-linecap: round; }
.task-todo-progress .progress-base { stroke: #dce3e7; stroke-dasharray: 2.3 4; }
.task-todo-progress .progress-value { stroke: currentColor; stroke-dasharray: calc(var(--task-progress) * 1) 100; transition: stroke-dasharray .35s ease; }
.task-todo-chevron { position: absolute; right: 0; width: 13px; height: 13px; color: #9aa3ae; transition: transform .2s ease; }
.task-todo-chevron.collapsed { transform: rotate(180deg); }
.task-todo-title { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.task-todo-title strong { overflow: hidden; color: #263248; font-size: 12px; font-weight: 650; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; }
.task-todo-title small { overflow: hidden; color: #8a94a4; font-size: 9px; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; }
.task-todo-count { min-width: 28px; color: #7b8695; font-size: 10px; font-variant-numeric: tabular-nums; text-align: right; animation: task-count-in .28s ease both; }
@keyframes task-count-in { from { opacity: .2; transform: translateY(5px); } to { opacity: 1; transform: translateY(0); } }
.stop-action { height: 30px; display: inline-flex; align-items: center; gap: 5px; flex: 0 0 auto; padding: 0 9px; border: 1px solid #e1d4d3; border-radius: 5px; color: #9a4e49; background: #fff; cursor: pointer; font-size: 10px; }
.stop-action:disabled { cursor: default; opacity: .5; }
.stop-action svg { width: 13px; }
.task-todo-collapsible { display: grid; grid-template-rows: 1fr; opacity: 1; transition: grid-template-rows .22s ease, opacity .17s ease; }
.task-todo-collapsible.collapsed { grid-template-rows: 0fr; opacity: 0; }
.task-todo-inner { min-height: 0; overflow: hidden; }
.task-todo-list { display: flex; flex-direction: column; gap: 1px; margin: 0; padding: 3px 12px 13px 13px; list-style: none; }
.task-todo-item { min-width: 0; display: grid; grid-template-columns: 22px minmax(0, 1fr); align-items: flex-start; gap: 7px; padding: 5px 4px; color: #8d97a4; }
.task-todo-item-icon { position: relative; width: 18px; height: 18px; display: block; margin-top: 1px; }
.todo-icon { position: absolute; inset: 0; width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; opacity: 0; transform: scale(.78); transition: opacity .18s ease, transform .18s ease; }
.todo-pending circle { stroke-dasharray: 1.6 3.2; }
.task-todo-item:not(.done):not(.active) .todo-pending, .task-todo-item.active .todo-current, .task-todo-item.done .todo-done { opacity: 1; transform: scale(1); }
.task-todo-item.active { color: #263b36; }
.task-todo-item.done { color: #a0a8b2; }
.task-todo-item.done .todo-done { color: #5f8f7f; }
.task-todo-item.interrupted { color: #8d6c69; }
.task-todo-item-copy { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.task-todo-item-copy > span { font-size: 11px; line-height: 1.45; }
.task-todo-item.active .task-todo-item-copy > span { color: #31453f; font-weight: 650; }
.task-todo-item.done .task-todo-item-copy > span { color: #8f98a3; }
.task-todo-item-copy small { color: #748078; font-size: 9px; line-height: 1.5; overflow-wrap: anywhere; }
.todo-keywords { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 2px; }
.todo-keyword { color: #466257; background: #edf4ef; border: 1px solid #d8e6dc; border-radius: 4px; font-size: 9px; line-height: 1.35; padding: 1px 4px; overflow-wrap: anywhere; }
.task-todo-item.done .todo-keyword { color: #68766e; background: #f1f4f2; border-color: #e1e8e3; }
.listing-keyword-candidates { margin: 10px 0 14px; padding: 10px 12px; border: 1px solid #dce7df; border-radius: 6px; background: #f8fbf9; }
.listing-keyword-candidates-head { display: flex; justify-content: space-between; align-items: center; color: #3d5549; font-size: 11px; margin-bottom: 7px; }
.listing-keyword-candidates-head span { color: #829087; font-size: 10px; }
.listing-keyword-candidate-list { display: flex; flex-wrap: wrap; gap: 5px; }
.listing-keyword-candidate { color: #355b4a; background: #edf5ef; border: 1px solid #cfe2d4; border-radius: 4px; font-size: 11px; line-height: 1.4; padding: 3px 6px; overflow-wrap: anywhere; }
.listing-keyword-candidate { flex: 1 1 250px; min-width: 220px; }
.listing-keyword-candidate > strong { display: block; color: #294b3c; font-size: 12px; margin-bottom: 4px; }
.listing-keyword-meta { display: flex; flex-wrap: wrap; gap: 3px 10px; color: #697970; font-size: 9px; line-height: 1.45; }
.listing-keyword-reason { display: block; color: #748078; font-size: 9px; line-height: 1.45; margin-top: 3px; }
.listing-keyword-evidence { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 3px; }
.listing-keyword-evidence a { color: #39705a; font-size: 9px; text-decoration: underline; }
.task-todo-item.interrupted .task-todo-item-copy small { color: #94635f; }
.todo { font-family: "Inter", system-ui, sans-serif; font-size: 13px; color: #1a1a1a; background: #fff; border-radius: 8px; padding: 6px 12px 12px; box-shadow: 0 0 0 1px #e6e8ec; }
.todoHead { display: flex; width: 100%; align-items: center; gap: 8px; padding: 0; border: 0; background: transparent; cursor: pointer; color: #1a1a1a; font-size: 13px; min-height: 22px; }
.todoHeadIcon { position: relative; width: 16px; height: 16px; flex: none; color: #a1a1a1; }
.todoListIcon, .todoChevron, .todoHeadCheck { position: absolute; inset: 0; margin: auto; transition: opacity 140ms ease; }
.todoListIcon, .todoChevron { width: 13px; height: 13px; }
.todoHeadCheck { width: 16px; height: 16px; color: #15a06a; }
.todoChevron { opacity: 0; transition: opacity 140ms ease, transform 220ms ease; }
.todoHead[aria-expanded="false"] .todoChevron { transform: rotate(-90deg); }
.todoHead:hover .todoListIcon, .todoHead:hover .todoHeadPie, .todoHead:hover .todoHeadCheck { opacity: 0; }
.todoHead:hover .todoChevron { opacity: 1; }
.todoTitle { font-weight: 500; }
.todoCount { margin-left: auto; color: #a1a1a1; font-variant-numeric: tabular-nums; }
.todoCollapsible { display: grid; grid-template-rows: 1fr; opacity: 1; transition: grid-template-rows 280ms ease, opacity 200ms ease; }
.todoCollapsible.isCollapsed { grid-template-rows: 0fr; opacity: 0; pointer-events: none; }
.todoInner { min-height: 0; overflow: hidden; }
.todoList { list-style: none; display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 10px 0 0; }
.todoItem { display: flex; align-items: flex-start; gap: 9px; line-height: 18px; color: #a1a1a1; animation: todo-item-in 360ms ease backwards; animation-delay: calc(var(--i, 0) * 50ms); }
@keyframes todo-item-in { from { opacity: 0; transform: translateY(-7px); } to { opacity: 1; transform: translateY(0); } }
.todoIconWrap { position: relative; width: 16px; height: 16px; flex: none; margin-top: 1px; }
.todoIcon { position: absolute; inset: 0; width: 16px; height: 16px; color: #a1a1a1; opacity: 0; transform: scale(1); transition: opacity 320ms ease, transform 320ms ease; }
.todoIcon.on { opacity: 1; }
.todoIcon.strong { color: #1a1a1a; }
.todoLabel { position: relative; font-weight: 400; color: #a1a1a1; transition: color 360ms ease; }
.todoLabel::before { content: attr(data-label); position: absolute; inset: 0; background: linear-gradient(90deg, #1a1a1a 0%, #1a1a1a 30%, rgba(26, 26, 26, .45) 45%, rgba(26, 26, 26, .45) 55%, #1a1a1a 70%, #1a1a1a 100%); background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; color: transparent; -webkit-text-fill-color: transparent; opacity: 0; transition: opacity 360ms ease; pointer-events: none; }
.todoItem.active .todoLabel { color: transparent; }
.todoItem.active .todoLabel::before { opacity: 1; animation: todo-shine 2.25s cubic-bezier(.25, .1, .25, 1) infinite; }
.todoItem.done .todoLabel { color: #a1a1a1; text-decoration: line-through; }
@keyframes todo-shine { 0%, 18% { background-position: 100% 0; } 82%, 100% { background-position: 0% 0; } }
.todoHeadPie { position: absolute; inset: 0; margin: auto; width: 13px; height: 13px; border-radius: 50%; color: #1a1a1a; transition: opacity 140ms ease; }
.todoHeadPie::after { content: ""; position: absolute; inset: 2.6px; border-radius: 50%; background: conic-gradient(currentColor var(--todo-pie, 0%), transparent 0); }
.todoHeadPieRing { position: absolute; inset: 0; width: 100% !important; height: 100% !important; overflow: visible; color: #a1a1a1; }
@media (prefers-reduced-motion: reduce) {
  .todoItem { animation: none; }
  .todoIcon, .todoLabel, .todoLabel::before { transition: none; }
  .todoItem.active .todoLabel::before { animation: none; }
}
@media (prefers-color-scheme: dark) {
  .todo { color: #f5f5f5; background: #1a1a1a; box-shadow: 0 0 0 1px #303030; }
  .todoHead { color: #f5f5f5; }
  .todoHeadCheck { color: #34d399; }
  .todoHeadIcon, .todoCount, .todoItem, .todoIcon, .todoLabel { color: #737373; }
  .todoHeadPieRing { color: #737373; }
  .todoHeadPie, .todoIcon.strong { color: #f5f5f5; }
  .todoLabel::before { background: linear-gradient(90deg, #f5f5f5 0%, #f5f5f5 30%, rgba(245, 245, 245, .45) 45%, rgba(245, 245, 245, .45) 55%, #f5f5f5 70%, #f5f5f5 100%); background-size: 300% 100%; }
}
.audit-notice { display: flex; align-items: flex-start; gap: 9px; margin-top: 10px; padding: 11px 12px; border: 1px solid #dce5e9; border-left: 3px solid #168a67; border-radius: 6px; color: #526376; background: #f3f8f6; font-size: 10px; line-height: 1.45; }
.audit-notice svg { flex: 0 0 16px; width: 16px; height: 16px; color: #168a67; }
.audit-notice > div { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.audit-notice strong { color: #2e5146; font-size: 11px; }
.audit-notice span { color: #647c74; }
.audit-correcting { border-color: #eadfca; border-left-color: #c27a16; background: #fff9ee; }
.audit-correcting svg { color: #c27a16; }
.audit-correcting strong { color: #754e18; }
.audit-correcting span { color: #8b6b3a; }
.audit-success { border-color: #dce7dc; background: #f5faf5; }
.audit-success svg { color: #4e8a55; }
.audit-success strong { color: #35633c; }
.audit-success span { color: #66816b; }
.review-badge.unresolved { color: #8b5f24; background: rgba(255, 247, 230, .94); }
.image-group { margin-top: 22px; }
.image-group-row .image-group { margin-top: 0; }
.group-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 8px; }
.group-heading > div { display: flex; align-items: baseline; gap: 7px; }
.group-heading strong { color: #354158; font-size: 11px; }
.group-heading span { color: #919baa; font-size: 9px; }
.task-history-message { padding: 14px 0 8px; border-top: 1px solid #e5e9ed; }
.task-history-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.task-history-heading > div { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.task-history-heading strong { color: #344157; font-size: 12px; }
.task-history-heading span { color: #8a95a3; font-size: 9px; }
.task-history-heading > span { flex: 0 0 auto; padding: 3px 6px; border: 1px solid #d9e0e6; border-radius: 4px; background: #f7f9fa; }
.history-image-item .image-select { cursor: zoom-in; }
.image-grid { display: grid; gap: 9px; }
.grid-secondary { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.grid-desktop { grid-template-columns: 1fr; }
.grid-mobile { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.image-item { min-width: 0; overflow: hidden; border: 1px solid #e2e7ec; border-radius: 6px; background: #fff; transition: border-color .15s ease, box-shadow .15s ease; }
.image-item.selected { border-color: #5d708a; box-shadow: 0 0 0 2px #e6ebf0; }
.image-item.pending { background: #fafbfc; }
.image-generation-card { display: flex; flex-direction: column; align-items: center; gap: 14px; padding: 14px 0; }
.ig-canvas { position: relative; width: 100%; max-width: 208px; aspect-ratio: 1 / 1; border-radius: 12px; background: #fafafa; overflow: hidden; }
.generation-desktop .ig-canvas { max-width: none; aspect-ratio: 21 / 9; border-radius: 8px; }
.generation-mobile .ig-canvas { max-width: none; aspect-ratio: 4 / 3; border-radius: 8px; }
.generation-desktop .ig-dots, .generation-mobile .ig-dots { inset: 0; background-repeat: repeat; }
.generation-desktop .ig-glow, .generation-mobile .ig-glow { inset: 0; background-repeat: repeat; -webkit-mask-size: 88% 80%, 64% 72%; mask-size: 88% 80%, 64% 72%; -webkit-mask-position: -10% 4%, 24% 28%; mask-position: -10% 4%, 24% 28%; animation: ig-morph-wide 4.2s cubic-bezier(.35, 1.55, .65, 1) infinite, ig-breathe 1.9s cubic-bezier(.66, 0, .34, 1) infinite; }
.ig-dots { position: absolute; inset: 2px; background-image: radial-gradient(circle, #a1a1a1 .7px, transparent 1.3px); background-size: 11px 11px; background-repeat: space; opacity: .22; }
.ig-glow { position: absolute; inset: 2px; background-image: radial-gradient(circle, #0b0d12 1.1px, transparent 1.6px); background-size: 11px 11px; background-repeat: space; -webkit-mask-image: radial-gradient(ellipse at center, #000 0%, transparent 60%), radial-gradient(ellipse at center, #000 0%, transparent 62%); mask-image: radial-gradient(ellipse at center, #000 0%, transparent 60%), radial-gradient(ellipse at center, #000 0%, transparent 62%); -webkit-mask-repeat: no-repeat, no-repeat; mask-repeat: no-repeat, no-repeat; -webkit-mask-size: 72% 62%, 52% 52%; mask-size: 72% 62%, 52% 52%; -webkit-mask-position: 16% 20%, 30% 32%; mask-position: 16% 20%, 30% 32%; animation: ig-morph 4.2s cubic-bezier(.35, 1.55, .65, 1) infinite, ig-breathe 1.9s cubic-bezier(.66, 0, .34, 1) infinite; }
@keyframes ig-morph {
  0% { -webkit-mask-size: 52% 46%, 40% 40%; mask-size: 52% 46%, 40% 40%; -webkit-mask-position: 16% 20%, 30% 32%; mask-position: 16% 20%, 30% 32%; }
  25% { -webkit-mask-size: 46% 58%, 44% 38%; mask-size: 46% 58%, 44% 38%; -webkit-mask-position: 84% 16%, 66% 30%; mask-position: 84% 16%, 66% 30%; }
  50% { -webkit-mask-size: 60% 44%, 38% 46%; mask-size: 60% 44%, 38% 46%; -webkit-mask-position: 82% 84%, 62% 68%; mask-position: 82% 84%, 62% 68%; }
  75% { -webkit-mask-size: 48% 54%, 46% 40%; mask-size: 48% 54%, 46% 40%; -webkit-mask-position: 14% 82%, 34% 66%; mask-position: 14% 82%, 34% 66%; }
  100% { -webkit-mask-size: 52% 46%, 40% 40%; mask-size: 52% 46%, 40% 40%; -webkit-mask-position: 16% 20%, 30% 32%; mask-position: 16% 20%, 30% 32%; }
}
@keyframes ig-morph-wide {
  0% { -webkit-mask-size: 88% 80%, 64% 72%; mask-size: 88% 80%, 64% 72%; -webkit-mask-position: -10% 4%, 24% 28%; mask-position: -10% 4%, 24% 28%; }
  25% { -webkit-mask-size: 82% 92%, 70% 68%; mask-size: 82% 92%, 70% 68%; -webkit-mask-position: 110% -4%, 74% 22%; mask-position: 110% -4%, 74% 22%; }
  50% { -webkit-mask-size: 96% 78%, 64% 82%; mask-size: 96% 78%, 64% 82%; -webkit-mask-position: 108% 108%, 68% 78%; mask-position: 108% 108%, 68% 78%; }
  75% { -webkit-mask-size: 84% 90%, 72% 70%; mask-size: 84% 90%, 72% 70%; -webkit-mask-position: -8% 106%, 24% 76%; mask-position: -8% 106%, 24% 76%; }
  100% { -webkit-mask-size: 88% 80%, 64% 72%; mask-size: 88% 80%, 64% 72%; -webkit-mask-position: -10% 4%, 24% 28%; mask-position: -10% 4%, 24% 28%; }
}
@keyframes ig-breathe { 0%, 100% { opacity: .55; } 50% { opacity: 1; } }
.ig-meta { display: flex; flex-direction: column; gap: 2px; align-items: flex-start; text-align: left; width: 100%; max-width: 208px; }
.generation-desktop .ig-meta, .generation-mobile .ig-meta { max-width: none; }
.ig-label { color: transparent; background: linear-gradient(90deg, #1a1a1a 0%, #1a1a1a 30%, rgba(26, 26, 26, .45) 45%, rgba(26, 26, 26, .45) 55%, #1a1a1a 70%, #1a1a1a 100%); background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent; animation: ig-shine 2.25s cubic-bezier(.25, .1, .25, 1) infinite; font-size: 14px; font-weight: 550; }
@keyframes ig-shine { 0%, 18% { background-position: 100% 0; } 82%, 100% { background-position: 0% 0; } }
.ig-prompt { color: #a1a1a1; font-size: 13px; }
@media (prefers-color-scheme: dark) {
  .ig-canvas { background: #1f1f1f; }
  .ig-glow { background-image: radial-gradient(circle, #f5f5f5 1.1px, transparent 1.6px); }
  .ig-label { background: linear-gradient(90deg, #f5f5f5 0%, #f5f5f5 30%, rgba(245, 245, 245, .45) 45%, rgba(245, 245, 245, .45) 55%, #f5f5f5 70%, #f5f5f5 100%); background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; }
}
.image-select { width: 100%; display: block; padding: 0; border: 0; color: inherit; background: transparent; text-align: left; cursor: pointer; }
.image-select:disabled { cursor: default; }
.image-frame { position: relative; display: block; aspect-ratio: 1 / 1; overflow: hidden; background: #f0f2f4; }
.grid-desktop .image-frame { aspect-ratio: 21 / 9; }
.grid-mobile .image-frame { aspect-ratio: 4 / 3; }
.image-frame img { width: 100%; height: 100%; display: block; object-fit: cover; }
.image-skeleton { width: 100%; height: 100%; display: grid; place-items: center; color: #b0b8c2; }
.image-skeleton svg { width: 24px; }
.image-number, .image-status { position: absolute; top: 7px; padding: 3px 5px; border-radius: 4px; font-size: 8px; line-height: 1; backdrop-filter: blur(4px); }
.image-number { right: 7px; color: #566276; background: rgba(255, 255, 255, .84); font-weight: 700; }
.image-status { left: 7px; color: #657185; background: rgba(255, 255, 255, .88); }
.review-badge { position: absolute; left: 7px; bottom: 7px; padding: 4px 6px; border-radius: 4px; color: #fff; background: rgba(28, 105, 82, .9); font-size: 8px; line-height: 1; }
.review-badge.correcting { background: rgba(166, 102, 16, .92); }
.review-badge.corrected { background: rgba(28, 105, 82, .92); }
.image-item.corrected { border-color: #b7d7c5; }
.image-copy { display: block; min-height: 53px; padding: 8px 9px; }
.image-copy strong { display: block; color: #465267; font-size: 10px; line-height: 1.4; }
.image-copy > span { display: -webkit-box; margin-top: 3px; overflow: hidden; color: #929ba9; font-size: 9px; line-height: 1.4; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.image-item.reviewing-card { border-color: #99b9c8; box-shadow: 0 0 0 2px rgba(64, 117, 139, .08); }
.image-item.correcting-card { border-color: #d4a66a; box-shadow: 0 0 0 2px rgba(194, 122, 22, .1); }
.image-actions { display: flex; flex-wrap: wrap; gap: 6px; padding: 0 9px 9px; }
.image-actions button { min-width: 0; height: 28px; display: inline-flex; align-items: center; justify-content: center; gap: 5px; padding: 0 8px; border: 1px solid #dfe4e9; border-radius: 5px; color: #56647a; background: #fff; cursor: pointer; font-size: 9px; white-space: nowrap; }
.image-actions button:hover { border-color: #aab5c1; background: #f8f9fa; }
.image-actions button:disabled { cursor: default; opacity: .48; }
.image-actions svg { width: 12px; }
.candidate-inline { grid-column: 1 / -1; padding: 10px; border-top: 1px solid #e5e9ed; background: #f8f9fa; }
.audit-compare-button { min-width: 0; display: inline-flex; align-items: center; gap: 8px; padding: 0; border: 0; color: #536075; background: transparent; cursor: pointer; text-align: left; }
.audit-compare-button:hover { color: #263247; }
.audit-compare-button svg { width: 16px; flex: 0 0 16px; }
.audit-compare-button span { display: flex; flex-direction: column; gap: 2px; }
.audit-compare-button strong { color: inherit; font-size: 10px; font-weight: 650; }
.candidate-inline p { margin: 8px 0 0; color: #667286; font-size: 9px; line-height: 1.5; }
.candidate-state { min-height: 46px; display: flex; align-items: center; justify-content: center; gap: 7px; color: #6c7788; font-size: 10px; }
.candidate-state svg { width: 15px; }
.spinning { animation: spin .9s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.candidate-actions { display: flex; justify-content: flex-end; gap: 7px; margin-top: 10px; }
.candidate-actions button { height: 29px; display: inline-flex; align-items: center; gap: 5px; padding: 0 9px; border: 1px solid #d9dfe5; border-radius: 5px; color: #536075; background: #fff; cursor: pointer; font-size: 9px; }
.candidate-actions button.primary { border-color: #263247; color: #fff; background: #263247; }
.candidate-actions button:disabled { cursor: default; opacity: .45; }
.candidate-actions svg { width: 12px; }
.thinking-state { display: inline-block; margin-top: 1px; font-size: 13px; line-height: 18px; }
.shimmer { color: transparent; -webkit-text-fill-color: transparent; background: linear-gradient(90deg, #a1a1a1 0%, #a1a1a1 30%, rgba(161, 161, 161, .45) 45%, rgba(161, 161, 161, .45) 55%, #a1a1a1 70%, #a1a1a1 100%); background-size: 300% 100%; -webkit-background-clip: text; background-clip: text; animation: thinking-shine 2.25s cubic-bezier(.25, .1, .25, 1) infinite; }
@keyframes thinking-shine { 0%, 18% { background-position: 100% 0; } 82%, 100% { background-position: 0% 0; } }
@media (prefers-reduced-motion: reduce) {
  .shimmer { animation: none; }
  .ig-glow, .ig-label { animation: none; }
  .ig-label { color: #1a1a1a; -webkit-text-fill-color: #1a1a1a; }
  .task-todo-collapsible, .task-todo-chevron, .todo-icon { transition: none; }
  .task-todo-count { animation: none; }
}
.composer-wrap { padding: 12px clamp(18px, 5vw, 64px) 17px; border-top: 1px solid #e5e8ec; background: #fbfcfd; }
.image-context { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 8px; padding: 8px 10px; border: 1px solid #dfe5eb; border-left: 3px solid #253248; border-radius: 6px; color: #6c7788; background: #f5f7f8; font-size: 10px; }
.image-context > div { min-width: 0; display: flex; align-items: center; gap: 6px; }
.image-context svg { flex: 0 0 auto; width: 13px; }
.image-context strong { overflow: hidden; color: #39475d; text-overflow: ellipsis; white-space: nowrap; }
.image-context button { flex: 0 0 24px; width: 24px; height: 24px; display: grid; place-items: center; border: 0; color: #778295; background: transparent; cursor: pointer; }
.composer-box { overflow: visible; border: 1px solid #d2d8df; border-radius: 12px; background: #fff; box-shadow: 0 4px 14px rgba(29, 41, 57, .055), 0 1px 2px rgba(29, 41, 57, .04); transition: border-color .16s ease, box-shadow .16s ease; }
.composer-box:focus-within { border-color: #8a96a5; box-shadow: 0 0 0 3px rgba(94, 107, 124, .1), 0 6px 18px rgba(29, 41, 57, .07); }
.pending-attachment { width: 100%; min-height: 78px; display: flex; align-items: center; gap: 10px; margin: 10px 0 0; padding: 0 14px; }
.attachment-tile { position: relative; width: 64px; height: 64px; flex: 0 0 64px; overflow: visible; border: 1px solid #dfe1e6; border-radius: 9px; background: #f7f7f8; }
.attachment-tile img { width: 100%; height: 100%; display: block; overflow: hidden; border-radius: 8px; object-fit: cover; }
.attachment-tile button { position: absolute; top: -8px; right: -8px; width: 22px; height: 22px; display: grid; place-items: center; border: 1px solid #d9d9e3; border-radius: 50%; color: #565663; background: #fff; box-shadow: 0 1px 4px rgba(0, 0, 0, .12); cursor: pointer; transition: color .15s ease, background .15s ease, transform .15s ease; }
.attachment-tile button:hover { color: #343541; background: #f1f1f3; transform: scale(1.04); }
.attachment-tile svg { width: 12px; height: 12px; }
.attachment-info { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.attachment-info strong { min-width: 0; overflow: hidden; color: #343541; font-size: 11px; font-weight: 600; line-height: 1.35; text-overflow: ellipsis; white-space: nowrap; }
.attachment-info span { color: #8e8e9b; font-size: 10px; line-height: 1.35; }
.composer-editor { padding: 5px 14px 0; }
.composer-editor textarea { width: 100%; height: 54px; min-height: 54px; max-height: 132px; display: block; resize: none; padding: 10px 0 5px; border: 0; outline: none; color: #273247; background: transparent; font: inherit; font-size: 13px; line-height: 1.55; }
.composer-editor textarea::placeholder { color: #969fac; }
.composer-editor textarea:disabled { cursor: default; color: #7f8995; }
.composer-toolbar { min-height: 48px; display: flex; align-items: center; justify-content: space-between; padding: 4px 8px 8px; }
.icon-button, .send-button { width: 36px; height: 36px; display: grid; flex: 0 0 36px; place-items: center; border-radius: 50%; cursor: pointer; transition: color .15s ease, background .15s ease, border-color .15s ease, transform .15s ease; }
.icon-button { border: 1px solid transparent; color: #556174; background: #f0f2f4; }
.send-button { border: 1px solid #202b3c; color: #fff; background: #202b3c; }
.icon-button:hover:not(:disabled) { border-color: #d7dce1; color: #2f3b4d; background: #e8ebee; }
.send-button:hover:not(:disabled) { transform: translateY(-1px); border-color: #131c2a; background: #131c2a; }
.icon-button:disabled, .send-button:disabled { cursor: default; opacity: .34; transform: none; }
.icon-button svg, .send-button svg { width: 17px; height: 17px; stroke-width: 1.8; }
.error-message { margin: 7px 43px 0; color: #a24e48; text-align: center; font-size: 10px; line-height: 1.4; }
.preview-footer-bar { display: flex; align-items: center; justify-content: space-between; gap: 16px; width: min(760px, calc(100% - 24px)); margin: 0 auto; padding: 10px 12px; border: 1px solid rgba(255, 255, 255, .14); border-radius: 7px; color: #fff; background: rgba(24, 32, 45, .92); box-shadow: 0 8px 24px rgba(0, 0, 0, .2); }
.preview-footer-bar > div { min-width: 0; }
.preview-footer-bar strong, .preview-footer-bar span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.preview-footer-bar strong { font-size: 11px; }
.preview-footer-bar span { margin-top: 2px; color: #b8c1cc; font-size: 9px; }
.preview-footer-bar button { flex: 0 0 auto; height: 32px; display: inline-flex; align-items: center; gap: 6px; padding: 0 10px; border: 1px solid #fff; border-radius: 6px; color: #263247; background: #fff; cursor: pointer; font-size: 10px; }
.preview-footer-bar svg { width: 14px; }
.audit-compare-backdrop { position: fixed; inset: 0; z-index: 80; display: grid; place-items: center; padding: 24px; background: rgba(22, 25, 31, .58); backdrop-filter: blur(5px); }
.audit-compare-modal { width: min(1080px, 100%); max-height: min(820px, calc(100vh - 48px)); overflow: auto; border: 1px solid #dcdde5; border-radius: 12px; background: #fff; box-shadow: 0 24px 70px rgba(25, 30, 40, .24); }
.audit-compare-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 18px; border-bottom: 1px solid #e7e7ec; }
.audit-compare-header span, .audit-compare-header strong { display: block; }
.audit-compare-header span { color: #8b8b98; font-size: 10px; }
.audit-compare-header strong { margin-top: 3px; color: #343541; font-size: 14px; font-weight: 650; }
.audit-compare-close { width: 30px; height: 30px; display: grid; flex: 0 0 30px; place-items: center; border: 1px solid #dedee7; border-radius: 7px; color: #686875; background: #fff; cursor: pointer; }
.audit-compare-close:hover { color: #343541; background: #f4f4f6; }
.audit-compare-close svg { width: 16px; }
.audit-compare-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; padding: 18px; }
.audit-compare-grid figure { min-width: 0; margin: 0; }
.audit-compare-grid figcaption { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; margin-bottom: 8px; }
.audit-compare-grid figcaption strong { color: #454553; font-size: 11px; font-weight: 650; }
.audit-compare-grid figcaption span { color: #9a9aa7; font-size: 9px; }
.audit-compare-image { display: grid; min-height: 240px; aspect-ratio: 1 / 1; place-items: center; overflow: hidden; border: 1px solid #e1e2e8; border-radius: 8px; background: #f5f5f7; }
.audit-compare-image img { width: 100%; height: 100%; display: block; object-fit: contain; }
@media (max-width: 720px) {
  .suite-page { padding: 15px 10px 20px; }
  .conversation-shell { height: calc(100vh - 116px); min-height: 480px; }
  .conversation { padding: 20px 12px 24px; }
  .composer-wrap { padding: 9px 12px 11px; }
  .grid-secondary, .grid-mobile { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .message-bubble { max-width: calc(100% - 42px); }
  .task-message { width: calc(100% - 37px); }
  .suggestion-button { min-height: 33px; }
  .style-suggestions { grid-template-columns: 1fr; }
  .style-suggestion:last-child { grid-column: auto; }
  .model-suggestions { grid-template-columns: 1fr; }
  .preview-footer-bar { width: calc(100% - 12px); }
}
@media (max-width: 430px) {
  .grid-secondary, .grid-mobile { grid-template-columns: 1fr; }
  .audit-compare-backdrop { align-items: end; padding: 10px; }
  .audit-compare-modal { max-height: calc(100vh - 20px); border-radius: 10px 10px 0 0; }
  .audit-compare-grid { grid-template-columns: 1fr; gap: 18px; padding: 14px; }
  .audit-compare-image { min-height: 0; }
  .task-todo-header { align-items: center; }
  .task-todo-title small { white-space: normal; }
  .avatar { width: 27px; height: 27px; flex-basis: 27px; }
  .message-row { gap: 7px; }
  .message-bubble { max-width: calc(100% - 34px); padding: 11px 12px; font-size: 12px; }
  .composer-editor { padding-inline: 12px; }
  .composer-editor textarea { font-size: 12px; }
  .composer-toolbar { padding: 3px 7px 7px; }
  .pending-attachment { min-height: 72px; padding-inline: 12px; }
  .preview-footer-bar { align-items: flex-start; }
}


<style scoped>
:global(body) {
  background: #f7f7f8;
  color: #2f2f2f;
}

.suite-page {
  height: calc(100vh - 64px);
  min-height: 0;
  overflow: hidden;
  padding: 0;
  color: #2f2f2f;
  background: #f7f7f8;
}

.toolbar-kicker {
  color: #8e8ea0;
  font-size: 10px;
  letter-spacing: .02em;
}

.conversation-shell {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: block;
  margin: 0;
  overflow: hidden;
  grid-template-rows: none;
  border: 0;
  border-radius: 0;
  background: #fff;
  box-shadow: none;
}

.workspace-layout {
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr) 272px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.suite-sidebar,
.suite-inspector {
  min-width: 0;
  background: #f7f7f8;
}

.suite-sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px 12px 14px;
  border-right: 1px solid #e5e5e5;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 0 7px 3px;
}

.sidebar-brand-mark {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 1px solid #dedee8;
  border-radius: 8px;
  color: #343541;
  background: #fff;
}

.sidebar-brand-mark svg {
  width: 15px;
}

.sidebar-brand strong,
.sidebar-brand small {
  display: block;
}

.sidebar-brand strong {
  color: #343541;
  font-size: 12px;
  font-weight: 650;
}

.sidebar-brand small {
  margin-top: 1px;
  color: #9a9aa7;
  font-size: 9px;
}

.sidebar-new-chat {
  width: 100%;
  min-height: 38px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid #d9d9e3;
  border-radius: 8px;
  color: #343541;
  background: #fff;
  cursor: pointer;
  font-size: 11px;
  text-align: left;
  transition: background .15s ease, border-color .15s ease;
}

.sidebar-new-chat:hover {
  border-color: #bcbcc8;
  background: #f0f0f1;
}

.sidebar-new-chat svg {
  width: 15px;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.sidebar-label,
.inspector-label {
  display: block;
  padding: 0 8px;
  color: #9a9aa7;
  font-size: 9px;
  font-weight: 650;
  letter-spacing: .05em;
  text-transform: uppercase;
}

.stage-nav ol {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.stage-nav li {
  min-width: 0;
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border-radius: 7px;
  color: #a0a0ab;
}

.stage-nav li.active {
  color: #343541;
  background: #ececef;
}

.stage-nav li.done {
  color: #6b8b7d;
}

.stage-nav li.error {
  color: #a25e55;
}

.stage-nav-index {
  width: 19px;
  height: 19px;
  display: grid;
  place-items: center;
  border: 1px solid #dcdce4;
  border-radius: 50%;
  font-size: 9px;
  font-variant-numeric: tabular-nums;
}

.stage-nav-index svg {
  width: 11px;
}

.stage-nav li.active .stage-nav-index {
  border-color: #10a37f;
  color: #10a37f;
}

.stage-nav li.done .stage-nav-index {
  border-color: #b7ded1;
  color: #10a37f;
  background: #e7f7f1;
}

.stage-nav-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stage-nav-copy strong,
.stage-nav-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stage-nav-copy strong {
  color: currentColor;
  font-size: 10px;
  font-weight: 600;
}

.stage-nav-copy small {
  color: #a0a0ab;
  font-size: 9px;
}

.sidebar-stop {
  min-height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid #ead8d5;
  border-radius: 7px;
  color: #9a534b;
  background: #fff;
  cursor: pointer;
  font-size: 10px;
}

.sidebar-stop svg {
  width: 13px;
}

.sidebar-stop:disabled {
  cursor: default;
  opacity: .5;
}

.workspace-center {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  overflow: hidden;
  background: #fff;
}
.conversation {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 26px clamp(18px, 5vw, 64px) 34px;
  background: #fff;
  scrollbar-color: #d3d3da transparent;
}

.conversation > div {
  width: min(820px, 100%);
  margin-inline: auto;
}

.conversation-empty {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px 18px 64px;
  color: #8e8e9b;
  text-align: center;
}

.conversation-empty-icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border: 1px solid #e5e5e5;
  border-radius: 12px;
  color: #6e6e7b;
  background: #f7f7f8;
}

.conversation-empty-icon svg {
  width: 20px;
}

.conversation-empty h2 {
  margin: 4px 0 0;
  color: #343541;
  font-size: 18px;
  font-weight: 650;
}

.conversation-empty p {
  max-width: 360px;
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
}

.message-row {
  gap: 11px;
  margin-top: 26px;
}

.avatar {
  width: 28px;
  height: 28px;
  flex-basis: 28px;
  border: 0;
  border-radius: 7px;
  color: #6e6e7b;
  background: #f1f1f3;
}

.message-row.user .avatar {
  border: 0;
  color: #fff;
  background: #343541;
}

.message-bubble {
  max-width: min(700px, calc(100% - 42px));
  padding: 1px 0;
  border: 0;
  border-radius: 0;
  color: #343541;
  background: transparent;
  box-shadow: none;
  font-size: 13px;
  line-height: 1.7;
}

.message-row.user .message-bubble {
  max-width: min(640px, calc(100% - 42px));
  padding: 10px 14px;
  border: 0;
  border-radius: 14px;
  color: #343541;
  background: #f1f1f3;
  box-shadow: none;
}

.message-bubble p {
  line-height: 1.7;
}

.uploaded-image {
  width: min(300px, 100%);
  max-height: 280px;
  border: 1px solid #e5e5e5;
  border-radius: 10px;
}

.suggestion-row,
.model-suggestions {
  margin-top: 14px;
}

.suggestion-button {
  min-height: 32px;
  border-color: #d9d9e3;
  border-radius: 8px;
  color: #565663;
  background: #fff;
  font-size: 10px;
}

.suggestion-button:hover:not(:disabled),
.model-choice:hover:not(:disabled) {
  transform: none;
  border-color: #a9a9b5;
  color: #343541;
  background: #f7f7f8;
}

.style-suggestions,
.model-suggestions {
  gap: 8px;
}

.style-suggestion,
.model-choice {
  border-color: #d9d9e3;
  border-radius: 9px;
  background: #fff;
}

.style-choice-heading strong,
.model-choice-heading strong {
  color: #343541;
}

.style-choice-heading span,
.model-choice-heading span {
  color: #8e8e9b;
}

.task-row {
  margin-top: 28px;
}

.task-message {
  width: min(820px, calc(100% - 40px));
}

.task-todo {
  padding: 0;
  border: 1px solid #e5e5e5;
  border-radius: 12px;
  color: #343541;
  background: #fff;
  box-shadow: none;
}

.task-todo-header {
  padding: 7px;
}

.task-todo-toggle {
  width: auto;
  min-height: 50px;
  display: grid;
  flex: 1;
  grid-template-columns: 34px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 5px 7px;
}

.task-todo-toggle:hover {
  background: #f7f7f8;
}

.task-todo-title strong {
  color: #343541;
}

.task-todo-title small,
.task-todo-count {
  color: #8e8e9b;
}

.stop-action {
  border-color: #ead8d5;
  border-radius: 7px;
  color: #9a534b;
  background: #fff;
}

.audit-notice {
  border: 1px solid #dcefe9;
  border-left: 3px solid #10a37f;
  border-radius: 9px;
  background: #f4fbf8;
  color: #5e706a;
}

.audit-correcting {
  border-color: #f0e3ca;
  border-left-color: #c0841a;
  background: #fffaf0;
}

.image-group {
  margin-top: 28px;
}

.group-heading strong {
  color: #343541;
  font-size: 12px;
}

.group-heading span {
  color: #9a9aa7;
}

.image-grid {
  gap: 10px;
}

.image-item {
  border-color: #e5e5e5;
  border-radius: 10px;
  background: #fff;
  box-shadow: none;
}

.image-item:hover,
.image-item.selected {
  border-color: #b8b8c4;
  box-shadow: 0 0 0 2px #f0f0f2;
}

.image-frame {
  background: #f1f1f3;
}

.image-copy {
  min-height: 62px;
  padding: 9px 10px 10px;
}

.image-copy strong {
  color: #343541;
  font-size: 10px;
}

.image-copy > span {
  color: #8e8e9b;
}

.image-actions button,
.candidate-actions button {
  border-color: #d9d9e3;
  border-radius: 7px;
  color: #565663;
  background: #fff;
}

.image-actions button:hover,
.candidate-actions button:hover:not(:disabled) {
  border-color: #adadb8;
  background: #f7f7f8;
}

.candidate-inline {
  border-top-color: #e5e5e5;
  background: #f7f7f8;
}

.candidate-actions button.primary {
  border-color: #343541;
  background: #343541;
}

.thinking-state {
  color: #8e8e9b;
}

.composer-wrap {
  position: relative;
  z-index: 3;
  flex: 0 0 auto;
  padding: 12px 24px 15px;
  border: 0;
  background: transparent;
  box-shadow: none;
}

.image-context,
.composer-box {
  width: min(820px, 100%);
  margin-inline: auto;
}

.image-context {
  margin-bottom: 8px;
  border: 1px solid #d9d9e3;
  border-left: 3px solid #343541;
  border-radius: 9px;
  color: #6e6e7b;
  background: #f7f7f8;
}

.image-context strong {
  color: #343541;
}

.composer-box {
  border: 1px solid #cfcfd8;
  border-radius: 15px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, .04);
}

.composer-box:focus-within {
  border-color: #9d9da9;
  box-shadow: 0 0 0 2px rgba(52, 53, 65, .08), 0 2px 7px rgba(0, 0, 0, .06);
}

.composer-editor textarea {
  color: #343541;
}

.composer-toolbar {
  min-height: 44px;
  padding: 3px 7px 7px;
}

.icon-button {
  color: #6e6e7b;
  background: transparent;
}

.icon-button:hover:not(:disabled) {
  border-color: #e5e5e5;
  color: #343541;
  background: #f1f1f3;
}

.send-button {
  border-color: #343541;
  background: #343541;
}

.send-button:hover:not(:disabled) {
  border-color: #202123;
  background: #202123;
}

.error-message {
  width: min(820px, 100%);
  margin-inline: auto;
  color: #a25e55;
}

.suite-inspector {
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow-y: auto;
  padding: 20px 16px;
  border-left: 1px solid #e5e5e5;
}

.inspector-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.inspector-header > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.inspector-header strong {
  color: #343541;
  font-size: 13px;
  font-weight: 650;
}

.inspector-section {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding-bottom: 18px;
  border-bottom: 1px solid #e5e5e5;
}

.inspector-section:last-child {
  border-bottom: 0;
}

.inspector-progress-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.inspector-progress-row strong {
  color: #343541;
  font-size: 20px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.inspector-progress-row span {
  max-width: 150px;
  overflow: hidden;
  color: #8e8e9b;
  font-size: 9px;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inspector-progress {
  height: 5px;
  overflow: hidden;
  border-radius: 5px;
  background: #e8e8ed;
}

.inspector-progress i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #10a37f;
  transition: width .35s ease;
}

.inspector-section p,
.inspector-style small,
.inspector-selection small {
  margin: 0;
  color: #8e8e9b;
  font-size: 10px;
  line-height: 1.55;
}

.inspector-scope-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.inspector-scope-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #565663;
  font-size: 10px;
}

.inspector-scope-row span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.inspector-scope-row i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #c4c4cc;
}

.inspector-scope-row i.active {
  background: #10a37f;
}

.inspector-scope-row i.done {
  background: #74b9a5;
}

.inspector-scope-row i.error {
  background: #c57970;
}

.inspector-scope-row strong {
  color: #343541;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.inspector-style,
.inspector-selection {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 10px;
  border: 1px solid #e1e1e6;
  border-radius: 9px;
  background: #fff;
}

.inspector-style strong,
.inspector-selection strong {
  color: #565663;
  font-size: 10px;
  line-height: 1.45;
}

.inspector-selection {
  border-color: #cde9df;
  background: #f4fbf8;
}

.inspector-eyebrow {
  color: #10a37f;
  font-size: 9px;
  font-weight: 650;
}

.inspector-tip {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  color: #6e6e7b;
  font-size: 10px;
  line-height: 1.5;
}

.inspector-tip svg {
  width: 14px;
  flex: 0 0 14px;
  color: #10a37f;
}

.inspector-clear {
  min-height: 29px;
  border: 1px solid #d9d9e3;
  border-radius: 7px;
  color: #565663;
  background: #fff;
  cursor: pointer;
  font-size: 10px;
}

.inspector-clear:hover {
  background: #f1f1f3;
}

.workspace-layout.inspector-collapsed {
  grid-template-columns: 232px minmax(0, 1fr) 0;
}

.sidebar-history {
  min-height: 0;
  flex: 1;
}

.sidebar-history-list {
  min-height: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #d3d3da transparent;
}

.sidebar-history-item {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 7px 8px;
  border: 0;
  border-radius: 7px;
  color: #686875;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.sidebar-history-item:hover {
  color: #343541;
  background: #ececef;
}

.sidebar-history-item.active {
  background: #ececef;
}

.sidebar-history-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar-history-copy strong,
.sidebar-history-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-history-copy strong {
  color: #4f4f5b;
  font-size: 10px;
  font-weight: 600;
}

.sidebar-history-copy small {
  color: #9a9aa7;
  font-size: 8px;
}

.sidebar-history-empty {
  margin: 4px 8px;
  color: #a0a0ab;
  font-size: 10px;
}

.sidebar-history-item {
  display: flex;
  align-items: stretch;
  gap: 2px;
  padding: 0;
}

.sidebar-history-open {
  min-width: 0;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: 7px;
  padding: 7px 8px;
  border: 0;
  border-radius: 7px;
  color: inherit;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.sidebar-history-open:hover {
  background: transparent;
}

.sidebar-history-delete {
  width: 29px;
  display: grid;
  flex: 0 0 29px;
  place-items: center;
  border: 0;
  border-radius: 7px;
  color: #a0a0ab;
  background: transparent;
  cursor: pointer;
}

.sidebar-history-delete:hover:not(:disabled) {
  color: #b4534b;
  background: #f8ecea;
}

.sidebar-history-delete:disabled {
  cursor: default;
  opacity: .55;
}

.sidebar-history-delete svg {
  width: 14px;
}

.message-row {
  gap: 0;
}

.message-bubble,
.message-row.user .message-bubble {
  max-width: min(760px, 100%);
}

.task-message {
  width: min(820px, 100%);
}

.send-button-stop {
  border-color: #343541;
  background: #343541;
}

.send-button-stop:hover:not(:disabled) {
  border-color: #202123;
  background: #202123;
}

.inspector-dock {
  position: relative;
  min-width: 0;
  width: 100%;
  height: 100%;
  background: #f7f7f8;
}

.inspector-dock.collapsed {
  width: 0;
}

.inspector-dock.collapsed .suite-inspector {
  display: none;
}

.inspector-toggle {
  position: absolute;
  z-index: 5;
  top: 14px;
  right: 14px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 1px solid #d9d9e3;
  border-radius: 7px;
  color: #6e6e7b;
  background: #fff;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, .05);
}

.inspector-toggle:hover {
  color: #343541;
  background: #f1f1f3;
}

.inspector-toggle svg {
  width: 15px;
}

.inspector-dock.collapsed .inspector-toggle {
  right: auto;
  left: -38px;
}

.suite-inspector {
  width: 100%;
  height: 100%;
}

.inspector-header {
  padding-right: 38px;
}

.inspector-images-section {
  min-height: 0;
  flex: 1;
}

.inspector-image-groups {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.inspector-image-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.inspector-image-group-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #8e8e9b;
  font-size: 9px;
}

.inspector-image-group-heading strong {
  color: #565663;
  font-size: 10px;
  font-weight: 650;
}

.inspector-image-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.inspector-image-entry {
  min-width: 0;
}

.inspector-image-card {
  width: 100%;
  min-width: 0;
  padding: 0;
  overflow: hidden;
  border: 1px solid #e1e1e6;
  border-radius: 7px;
  color: #565663;
  background: #fff;
  cursor: pointer;
  text-align: left;
}

.inspector-image-card:hover:not(:disabled),
.inspector-image-card.selected {
  border-color: #a9a9b5;
  box-shadow: 0 0 0 2px #ececef;
}

.inspector-image-card:disabled {
  cursor: default;
}

.inspector-image-frame {
  position: relative;
  display: grid;
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  color: #a0a0ab;
  background: #ececef;
}

.inspector-image-grid.grid-desktop .inspector-image-frame {
  aspect-ratio: 21 / 9;
}

.inspector-image-grid.grid-mobile .inspector-image-frame {
  aspect-ratio: 4 / 3;
}

.inspector-image-frame img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.inspector-image-frame > svg {
  width: 18px;
}

.inspector-image-number,
.inspector-image-status {
  position: absolute;
  left: 5px;
  padding: 2px 4px;
  border-radius: 4px;
  color: #fff;
  background: rgba(32, 33, 35, .72);
  font-size: 8px;
}

.inspector-image-status.candidate-ready {
  background: rgba(16, 163, 127, .82);
}

.inspector-image-status.review-status-reviewing {
  background: rgba(46, 105, 132, .9);
}

.inspector-image-status.review-status-correcting {
  background: rgba(166, 102, 16, .92);
}

.inspector-image-status.review-status-auto_corrected {
  background: rgba(28, 105, 82, .92);
}

.inspector-image-number {
  top: 5px;
}

.inspector-image-status {
  right: 5px;
  bottom: 5px;
  left: auto;
}

.inspector-image-title {
  display: block;
  overflow: hidden;
  padding: 6px 7px;
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inspector-source-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 5px;
  padding: 0;
  border: 0;
  color: #687588;
  background: transparent;
  cursor: pointer;
  font-size: 8px;
}

.inspector-source-button:hover { color: #344157; }
.inspector-source-button svg { width: 11px; height: 11px; }

.inspector-candidate-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px;
  margin-top: 5px;
}

.inspector-candidate-actions button {
  min-height: 25px;
  border: 1px solid #d9d9e3;
  border-radius: 6px;
  color: #565663;
  background: #fff;
  cursor: pointer;
  font-size: 9px;
}

.inspector-candidate-actions button.primary {
  border-color: #343541;
  color: #fff;
  background: #343541;
}

.inspector-candidate-actions button:disabled {
  cursor: default;
  opacity: .5;
}

.inspector-candidate-state {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 5px;
  color: #8e8e9b;
  font-size: 8px;
  line-height: 1.4;
}

.inspector-candidate-state svg {
  width: 12px;
  flex: 0 0 12px;
}

.inspector-images-empty {
  margin: 0;
  padding: 16px 10px;
  border: 1px dashed #d9d9e3;
  border-radius: 7px;
  color: #9a9aa7;
  font-size: 10px;
  line-height: 1.5;
  text-align: center;
}

.suite-page {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - 64px);
}

.conversation-shell {
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
}

.workspace-layout {
  height: 100%;
  min-height: 0;
}

.suite-sidebar {
  min-height: 0;
  overflow: hidden;
}

.sidebar-history {
  min-height: 0;
  overflow: hidden;
}

.workspace-center {
  height: 100%;
  min-height: 0;
  grid-template-rows: minmax(0, 1fr) auto;
}

.conversation {
  height: 100%;
  min-height: 0;
  max-height: 100%;
  overflow-y: auto;
}

.composer-wrap {
  min-height: 0;
  overflow: visible;
}

.inspector-dock,
.suite-inspector {
  min-height: 0;
  box-sizing: border-box;
}

.suite-inspector {
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: #d3d3da transparent;
}

.inspector-toggle {
  top: 16px;
  right: auto;
  left: -16px;
  width: 32px;
  height: 32px;
  border-color: #d1d1da;
  border-radius: 9px;
  color: #565663;
  background: #fff;
  box-shadow: 0 3px 10px rgba(0, 0, 0, .1);
  transition: color .15s ease, background .15s ease, box-shadow .15s ease, transform .15s ease;
}

.inspector-toggle:hover {
  color: #202123;
  background: #f7f7f8;
  box-shadow: 0 4px 12px rgba(0, 0, 0, .14);
  transform: translateX(-1px);
}

.inspector-dock.collapsed .inspector-toggle {
  left: -40px;
}

.requirement-modal-backdrop {
  position: fixed;
  z-index: 80;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(29, 35, 40, .38);
  backdrop-filter: blur(3px);
}

.requirement-modal-panel {
  width: min(900px, 100%);
  max-height: min(900px, calc(100vh - 48px));
  overflow: auto;
  margin: 0;
  padding: 2px 0 34px;
  border: 1px solid #dfe5e8;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 24px 70px rgba(30, 39, 48, .18);
}

.requirement-modal-panel.listing-only-form {
  width: min(700px, 100%);
  background: #fbfcfc;
}

.listing-only-form .requirement-form-header,
.listing-only-form .requirement-form-section,
.listing-only-form .requirement-form-footer {
  padding-left: 4px;
  padding-right: 4px;
}

.listing-only-form .requirement-form-header {
  padding-top: 22px;
  padding-bottom: 18px;
}

.listing-only-form .requirement-form-section {
  padding-top: 22px;
  padding-bottom: 22px;
}

.listing-only-form .requirement-form-grid-product > .requirement-form-stack {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, .8fr);
  gap: 14px;
}

.listing-only-form .requirement-form-grid-product > .requirement-form-stack > .requirement-field:first-child {
  grid-column: 1 / -1;
}

.listing-only-form .requirement-section-heading strong {
  font-size: 15px;
}

.listing-only-form .requirement-field input[type='text'],
.listing-only-form .requirement-field textarea {
  background: #fff;
}

.listing-only-form .requirement-form-footer {
  margin-left: -22px;
  margin-right: -22px;
  padding-top: 15px;
  padding-bottom: 17px;
  background: rgba(255, 255, 255, .96);
}

.requirement-form-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 8px 0 24px;
  border-bottom: 1px solid #e5e5e8;
}

.requirement-form-header h1 {
  margin: 5px 0 6px;
  color: #202123;
  font-size: 24px;
  font-weight: 680;
  letter-spacing: 0;
  line-height: 1.25;
}

.requirement-form-header p {
  margin: 0;
  color: #73737f;
  font-size: 12px;
  line-height: 1.6;
}

.requirement-form-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.requirement-form-header-actions > svg {
  width: 34px;
  height: 34px;
  padding: 7px;
  border: 1px solid #dadde1;
  border-radius: 8px;
  color: #4f655e;
  background: #f7f9f8;
}

.requirement-form-close {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: 1px solid #dadde1;
  border-radius: 8px;
  color: #747b84;
  background: #fff;
  cursor: pointer;
}

.requirement-form-close:hover {
  color: #26323b;
  background: #f5f7f7;
}

.requirement-form-close svg {
  width: 16px;
  height: 16px;
}

.requirement-form-error {
  margin: 16px 0 0;
  padding: 10px 12px;
  border: 1px solid #efd5d2;
  border-radius: 7px;
  color: #914d46;
  background: #fff8f7;
  font-size: 11px;
  line-height: 1.5;
}

.requirement-form-section {
  padding: 26px 0 28px;
  border-bottom: 1px solid #ececef;
}

.requirement-section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 18px;
}

.requirement-section-heading > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.requirement-section-heading strong {
  color: #27272b;
  font-size: 14px;
  font-weight: 650;
}

.requirement-section-heading span:not(.requirement-step) {
  color: #8a8a94;
  font-size: 10px;
  line-height: 1.5;
}

.requirement-step {
  color: #a1a1aa;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.requirement-form-grid {
  display: grid;
  gap: 18px;
}

.requirement-form-grid-product {
  grid-template-columns: 228px minmax(0, 1fr);
  align-items: start;
}

.listing-product-grid {
  grid-template-columns: minmax(0, 1fr);
}

.requirement-form-stack {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.requirement-field {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 7px;
  margin-top: 17px;
}

.requirement-form-grid .requirement-field,
.requirement-form-stack .requirement-field {
  margin-top: 0;
}

.requirement-label {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px;
  color: #3b3b40;
  font-size: 11px;
  font-weight: 620;
}

.requirement-label em {
  color: #a05a52;
  font-size: 9px;
  font-style: normal;
  font-weight: 500;
}

.requirement-label small {
  color: #9696a0;
  font-size: 9px;
  font-weight: 400;
}

.requirement-field input[type='text'],
.requirement-field input[type='number'],
.requirement-field textarea {
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #d9d9df;
  border-radius: 7px;
  outline: none;
  color: #29292d;
  background: #fff;
  font: inherit;
  font-size: 12px;
  line-height: 1.55;
  transition: border-color .15s ease, box-shadow .15s ease;
}

.requirement-field input[type='text'],
.requirement-field input[type='number'] {
  height: 40px;
  padding: 0 11px;
}

.requirement-field textarea {
  min-height: 88px;
  resize: vertical;
  padding: 10px 11px;
}

.requirement-field input:focus,
.requirement-field textarea:focus {
  border-color: #7b948b;
  box-shadow: 0 0 0 3px rgba(83, 116, 104, .1);
}

.requirement-field input::placeholder,
.requirement-field textarea::placeholder {
  color: #aaaab2;
}

.requirement-inline-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
}

.requirement-image-field {
  grid-row: span 2;
}

.product-image-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
}

.product-image-item {
  position: relative;
  min-width: 0;
  aspect-ratio: 1;
  overflow: hidden;
  border: 1px solid #d9d9df;
  border-radius: 7px;
  background: #fff;
}

.product-image-item.primary {
  border-color: #66887b;
  box-shadow: 0 0 0 2px rgba(83, 116, 104, .1);
}

.product-image-preview {
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: #fff;
  cursor: pointer;
}

.product-image-preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.product-image-preview span {
  position: absolute;
  left: 5px;
  bottom: 5px;
  padding: 2px 5px;
  border-radius: 4px;
  color: #fff;
  background: rgba(38, 48, 45, .82);
  font-size: 8px;
  line-height: 1.4;
}

.product-image-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid rgba(214, 217, 220, .9);
  border-radius: 5px;
  color: #62666b;
  background: rgba(255, 255, 255, .92);
  cursor: pointer;
}

.product-image-remove:hover {
  color: #944e47;
  border-color: #dfc1bd;
}

.product-image-remove svg {
  width: 12px;
}

.product-image-help {
  color: #8d9097;
  font-size: 9px;
  line-height: 1.5;
}

.requirement-upload {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  overflow: hidden;
  border: 1px dashed #cfd2d6;
  border-radius: 8px;
  color: #6d7278;
  background: #fafafa;
  cursor: pointer;
  transition: border-color .15s ease, background .15s ease;
}

.product-image-list .requirement-upload {
  min-width: 0;
  aspect-ratio: 1;
  padding: 6px;
}

.product-image-list .requirement-upload svg {
  width: 18px;
}

.product-image-list .requirement-upload strong {
  font-size: 9px;
}

.product-image-list .requirement-upload small {
  font-size: 7px;
}

.requirement-select {
  position: relative;
  min-width: 0;
}

.requirement-select-button {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 10px 0 11px;
  border: 1px solid #d9d9df;
  border-radius: 7px;
  color: #29292d;
  background: #fff;
  cursor: pointer;
  font-size: 12px;
  text-align: left;
}

.requirement-select-button:hover {
  border-color: #b6bcc0;
}

.requirement-select-button:focus-visible {
  outline: none;
  border-color: #7b948b;
  box-shadow: 0 0 0 3px rgba(83, 116, 104, .1);
}

.requirement-select-button .placeholder {
  color: #aaaab2;
}

.requirement-select-button svg {
  width: 16px;
  flex: 0 0 16px;
  color: #888d93;
}

.requirement-select-options {
  position: absolute;
  z-index: 20;
  top: calc(100% + 5px);
  right: 0;
  left: 0;
  max-height: 218px;
  overflow-y: auto;
  margin: 0;
  padding: 4px;
  border: 1px solid #dfe2e4;
  border-radius: 7px;
  outline: none;
  background: #fff;
  box-shadow: 0 12px 28px rgba(30, 39, 48, .13);
  list-style: none;
}

.requirement-select-option {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px;
  border-radius: 5px;
  color: #4a4d52;
  cursor: pointer;
  font-size: 11px;
}

.requirement-select-option.active {
  background: #f2f5f4;
}

.requirement-select-option.selected {
  color: #2f5c4e;
  font-weight: 620;
}

.requirement-select-option svg {
  width: 14px;
  color: #477565;
}

.select-transition {
  transition: opacity .1s ease, transform .1s ease;
  transform-origin: top;
}

.select-transition-from {
  opacity: 0;
  transform: translateY(-3px) scale(.985);
}

.requirement-upload:hover {
  border-color: #8ca198;
  background: #f6f9f7;
}

.requirement-upload svg {
  width: 22px;
}

.requirement-upload strong {
  font-size: 11px;
  font-weight: 620;
}

.requirement-upload small {
  color: #9b9ba3;
  font-size: 9px;
}

.requirement-upload.filled {
  border-style: solid;
  background: #fff;
}

.requirement-upload img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #fff;
}

.field-error {
  color: #a14f47;
  font-size: 9px;
  line-height: 1.4;
}

.home-depot-fetch-note {
  margin: 9px 0 0;
  color: #73737c;
  font-size: 10px;
  line-height: 1.55;
}

.preset-grid {
  display: grid;
  gap: 8px;
}

.preset-grid-platform,
.preset-grid-style {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.preset-button,
.model-preset {
  min-width: 0;
  min-height: 58px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 3px;
  padding: 9px 11px;
  border: 1px solid #dedee3;
  border-radius: 7px;
  color: #4c4c54;
  background: #fff;
  cursor: pointer;
  text-align: left;
  transition: border-color .15s ease, background .15s ease;
}

.preset-button:hover,
.model-preset:hover {
  border-color: #a7b5af;
  background: #fafcfb;
}

.preset-button.selected,
.model-preset.selected {
  border-color: #668479;
  color: #29473d;
  background: #f3f8f6;
  box-shadow: inset 0 0 0 1px rgba(76, 112, 99, .08);
}

.preset-button strong,
.model-preset strong {
  font-size: 10px;
  font-weight: 640;
}

.preset-button small,
.model-preset small {
  color: #94949e;
  font-size: 8px;
  line-height: 1.45;
}

.requirement-custom-market {
  grid-template-columns: 1.2fr 1fr 92px;
  margin-top: 2px;
}

.requirement-custom-market.amazon-custom-market {
  grid-template-columns: minmax(0, 1fr) 92px;
}

.output-option-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.output-option {
  min-width: 0;
  min-height: 64px;
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr) 48px;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid #dedee3;
  border-radius: 7px;
  background: #fff;
  cursor: default;
}

.output-option.selected {
  border-color: #8ba199;
  background: #f8faf9;
}

.output-option > input[type='checkbox'] {
  width: 14px;
  height: 14px;
  accent-color: #527466;
  cursor: pointer;
}

.output-option-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.output-option-copy strong {
  color: #3b3b40;
  font-size: 10px;
  font-weight: 640;
}

.output-option-copy small {
  color: #9696a0;
  font-size: 8px;
  line-height: 1.35;
}

.requirement-field .count-input {
  height: 32px;
  padding: 0 4px;
  text-align: center;
}

.model-preset-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.model-preset {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.model-preset > span {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.model-preset em {
  flex: 0 0 auto;
  padding: 2px 5px;
  border-radius: 4px;
  color: #547166;
  background: #e9f1ee;
  font-size: 8px;
  font-style: normal;
}

.model-empty {
  color: #9a5b54;
  font-size: 9px;
}

.requirement-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding-top: 22px;
}

.requirement-form-footer > span {
  color: #92929b;
  font-size: 9px;
  line-height: 1.5;
}

.requirement-submit {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 15px;
  border: 1px solid #303532;
  border-radius: 7px;
  color: #fff;
  background: #303532;
  cursor: pointer;
  font-size: 11px;
  font-weight: 620;
}

.requirement-submit:hover:not(:disabled) {
  background: #1f2321;
}

.requirement-submit:disabled {
  cursor: default;
  opacity: .55;
}

.task-todo-failed-icon {
  flex: 0 0 18px;
  width: 18px;
  height: 18px;
  color: #b45c53;
}

.task-todo-item.interrupted .todo-failed,
.task-todo-item.skipped .todo-skipped {
  opacity: 1;
  transform: scale(1);
}

.task-todo-item.interrupted .todo-pending,
.task-todo-item.skipped .todo-pending {
  opacity: 0;
  transform: scale(.78);
}

.task-todo-item.interrupted .todo-failed {
  color: #b45c53;
}

.task-todo-item.skipped {
  color: #a2a2aa;
}

.task-todo-item.skipped .todo-skipped {
  color: #aaaab2;
}

.task-todo-item.skipped .task-todo-item-copy small {
  color: #a0a0a8;
}

@media (max-width: 1120px) {
  .workspace-layout {
    grid-template-columns: 210px minmax(0, 1fr) 240px;
  }

  .workspace-layout.inspector-collapsed {
    grid-template-columns: 210px minmax(0, 1fr) 0;
  }

  .suite-sidebar {
    padding-inline: 9px;
  }

  .suite-inspector {
    padding-inline: 13px;
  }
}

@media (max-width: 900px) {
  .workspace-layout {
    grid-template-columns: 190px minmax(0, 1fr);
  }

  .inspector-dock {
    display: none;
  }
}

@media (max-width: 680px) {
  .requirement-modal-backdrop {
    align-items: end;
    padding: 10px;
  }

  .requirement-modal-panel {
    max-height: calc(100vh - 20px);
    border-radius: 10px 10px 0 0;
  }

  .conversation-shell {
    height: 100%;
    min-height: 0;
  }

  .workspace-layout {
    display: block;
  }

  .suite-sidebar {
    display: none;
  }

  .workspace-center {
    height: 100%;
    min-height: 0;
  }

  .conversation {
    padding: 20px 12px 25px;
  }

  .conversation > div {
    width: 100%;
  }

  .requirement-form-header h1 {
    font-size: 20px;
  }

  .requirement-form-grid-product,
  .preset-grid-platform,
  .preset-grid-style,
  .output-option-list,
  .model-preset-list {
    grid-template-columns: 1fr;
  }

  .requirement-image-field {
    width: min(320px, 100%);
    grid-row: auto;
  }

  .requirement-custom-market,
  .requirement-inline-fields {
    grid-template-columns: 1fr;
  }

  .requirement-form-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .requirement-submit {
    width: 100%;
  }

  .message-row {
    gap: 8px;
    margin-top: 22px;
  }

  .message-bubble {
    max-width: 100%;
    font-size: 12px;
  }

  .message-row.user .message-bubble {
    max-width: 100%;
  }

  .task-message {
    width: 100%;
  }

  .composer-wrap {
    padding: 9px 12px 11px;
  }

  .composer-editor textarea {
    font-size: 12px;
  }

  .grid-secondary,
  .grid-mobile {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 430px) {
  .grid-secondary,
  .grid-mobile {
    grid-template-columns: 1fr;
  }
}

.listing-result {
  margin-top: 22px;
  overflow: hidden;
  border: 1px solid #dfe4e8;
  border-radius: 8px;
  background: #fff;
}

.listing-result-head {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 11px 13px;
  border-bottom: 1px solid #e8ebee;
  background: #f8f9fa;
}

.listing-result-head > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.listing-result-head span {
  color: #7c8794;
  font-size: 12px;
  line-height: 1.3;
}

.listing-result-head strong {
  color: #2f3b4c;
  font-size: 14px;
  line-height: 1.35;
}

.listing-result-head small {
  color: #6f7b88;
  font-size: 11px;
  line-height: 1.3;
}

.listing-result-head button {
  height: 30px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  padding: 0 9px;
  border: 1px solid #cfd6dc;
  border-radius: 5px;
  color: #495768;
  background: #fff;
  cursor: pointer;
  font-size: 12px;
}

.listing-result-head button:hover {
  border-color: #8794a0;
  background: #f4f6f7;
}

.listing-result-head svg {
  width: 13px;
  height: 13px;
}

.listing-field {
  display: grid;
  grid-template-columns: 130px minmax(0, 1fr);
  gap: 14px;
  padding: 13px;
  border-bottom: 1px solid #eceff1;
}

.listing-field:last-child {
  border-bottom: 0;
}

.listing-field > span {
  color: #87919d;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.6;
}

.listing-field p,
.listing-field ol {
  margin: 0;
  color: #3d4858;
  font-size: 14px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.listing-reason {
  grid-column: 2;
  display: grid;
  gap: 3px;
  margin-top: 4px;
  color: #4e6f62;
  font-size: 12px;
  line-height: 1.5;
}

.listing-reason strong {
  color: #36564b;
  font-weight: 650;
}

.listing-title-field p {
  color: #293647;
  font-weight: 650;
}

.listing-field ol {
  padding-left: 17px;
}

.listing-field li + li {
  margin-top: 7px;
}

.listing-terms > div {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.listing-terms em {
  padding: 3px 6px;
  border: 1px solid #dfe4e8;
  border-radius: 4px;
  color: #66717f;
  background: #f7f8f9;
  font-size: 11px;
  font-style: normal;
  line-height: 1.35;
}

.listing-result-error {
  margin: 16px 0 0;
  padding: 9px 11px;
  border-left: 3px solid #b36d5e;
  color: #84564e;
  background: #fbf5f3;
  font-size: 9px;
  line-height: 1.5;
}

.listing-result-history {
  margin-top: 16px;
}

@media (max-width: 680px) {
  .listing-field {
    grid-template-columns: 1fr;
    gap: 5px;
  }

  .listing-reason {
    grid-column: 1;
  }
}

.entry-choice-panel {
  width: min(760px, 100%);
  margin: 22px auto 0;
  padding: 22px;
  border: 1px solid #dfe5e8;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(36, 52, 67, .05);
}

.entry-choice-copy { max-width: 540px; }
.entry-choice-copy h1 { margin: 7px 0 6px; color: #233143; font-size: clamp(20px, 3vw, 28px); letter-spacing: 0; line-height: 1.18; }
.entry-choice-copy p { margin: 0; color: #778391; font-size: 12px; line-height: 1.6; }
.entry-choice-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 9px; margin-top: 22px; }
.entry-choice-button { min-width: 0; min-height: 112px; display: grid; grid-template-columns: 28px minmax(0, 1fr) 15px; align-items: center; gap: 9px; padding: 13px; border: 1px solid #d8e0e3; border-radius: 8px; color: #2f3d4d; background: #fbfcfc; cursor: pointer; text-align: left; transition: border-color .16s ease, background .16s ease, transform .16s ease; }
.entry-choice-button:hover { border-color: #6e9586; background: #f4f8f6; transform: translateY(-1px); }
.entry-choice-button:active { transform: translateY(0) scale(.99); }
.entry-choice-button > span:nth-child(2) { min-width: 0; display: flex; flex-direction: column; gap: 5px; }
.entry-choice-button strong { color: #2d3c4d; font-size: 11px; line-height: 1.35; }
.entry-choice-button small { color: #89949d; font-size: 9px; line-height: 1.45; }
.entry-choice-icon { width: 28px; height: 28px; display: grid; place-items: center; border: 1px solid #c9ddd4; border-radius: 7px; color: #437666; background: #eef6f2; }
.entry-choice-icon svg, .entry-choice-button > svg { width: 15px; height: 15px; }
.entry-choice-button > svg { color: #98a3ad; }

@media (max-width: 680px) {
  .entry-choice-panel { margin-top: 12px; padding: 17px; }
  .entry-choice-grid { grid-template-columns: 1fr; gap: 7px; margin-top: 17px; }
  .entry-choice-button { min-height: 70px; }
}

/* Keep the requirement dialog focused on the fields that need an answer. */
.requirement-modal-backdrop {
  padding: 16px;
  background: rgba(29, 35, 40, .24);
  backdrop-filter: blur(2px);
}

.requirement-modal-panel {
  box-sizing: border-box;
  isolation: isolate;
  width: min(760px, 100%);
  max-height: min(820px, calc(100vh - 32px));
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 22px;
  border-color: #e3e6e8;
  border-radius: 9px;
  background: #fff;
  box-shadow: 0 18px 48px rgba(30, 39, 48, .14);
}

.requirement-modal-panel .requirement-form-header {
  padding: 18px 0 14px;
}

.requirement-modal-panel .requirement-form-header h1 {
  margin: 3px 0 0;
  font-size: 20px;
}

.requirement-modal-panel .requirement-form-header p,
.requirement-modal-panel .requirement-section-heading span:not(.requirement-step),
.requirement-modal-panel .requirement-form-footer > span {
  display: none;
}

.requirement-modal-panel .requirement-form-header-actions > svg {
  display: none;
}

.requirement-modal-panel .requirement-form-section {
  padding: 18px 0 20px;
}

.requirement-modal-panel .requirement-section-heading {
  margin-bottom: 12px;
}

.requirement-modal-panel .requirement-form-grid {
  gap: 14px;
}

.requirement-modal-panel .requirement-form-stack {
  gap: 12px;
}

.requirement-modal-panel .requirement-field {
  gap: 6px;
  margin-top: 12px;
}

.requirement-modal-panel .requirement-field textarea {
  min-height: 76px;
}

.requirement-modal-panel .requirement-form-footer {
  position: sticky;
  z-index: 2;
  bottom: 0;
  justify-content: flex-end;
  margin: 0 -22px;
  padding: 14px 22px 16px;
  border-top: 1px solid #eceeef;
  background: #fff;
}

@media (max-width: 680px) {
  .requirement-modal-backdrop {
    align-items: end;
    padding: 8px;
  }

  .requirement-modal-panel {
    max-height: calc(100vh - 16px);
    padding: 0 16px;
    border-radius: 9px 9px 0 0;
  }

  .requirement-modal-panel .requirement-form-footer {
    margin: 0 -16px;
    padding: 14px 16px;
  }
}

/* Keep every task surface on one agent-style status language. */
.task-status-badge {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  width: fit-content;
  gap: 5px;
  margin-top: 3px;
  padding: 3px 7px;
  border: 1px solid #e2e7ea;
  border-radius: 4px;
  color: #66727c;
  background: #f7f9f9;
  font-size: 9px;
  font-weight: 650;
  line-height: 1;
}
.task-status-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #98a4ad;
}
.task-status-dot.pulse { animation: task-status-pulse 1.4s ease-in-out infinite; }
.task-status-badge.tone-active { color: #236e58; border-color: #cfe4da; background: #f0f8f4; }
.task-status-badge.tone-active .task-status-dot { background: #2f9b78; }
.task-status-badge.tone-success { color: #2d7658; border-color: #cfe6d9; background: #f2f9f5; }
.task-status-badge.tone-success .task-status-dot { background: #3a9a73; }
.task-status-badge.tone-partial { color: #876b35; border-color: #eadfca; background: #fcf8ef; }
.task-status-badge.tone-partial .task-status-dot { background: #c89134; }
.task-status-badge.tone-failed { color: #a05855; border-color: #efd8d5; background: #fff6f5; }
.task-status-badge.tone-failed .task-status-dot { background: #c66b65; }
.task-status-badge.tone-stopped { color: #69737d; border-color: #dfe4e7; background: #f6f7f8; }
.task-status-badge.tone-pending .task-status-dot { background: #9aa5ae; }
.task-history-heading > .task-status-badge { margin-top: 0; }
.group-count { font-variant-numeric: tabular-nums; }
@keyframes task-status-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(47, 155, 120, .18); }
  50% { box-shadow: 0 0 0 4px rgba(47, 155, 120, 0); }
}



.admin-history-trigger { display:flex; align-items:center; gap:6px; align-self:flex-start; margin:8px 7px 0; padding:5px 0; color:#64706b; font-size:12px; text-decoration:none; }
.admin-history-trigger:hover { color:#23715b; }
.admin-history-trigger svg { width:15px; height:15px; flex-shrink:0; }

</style>
