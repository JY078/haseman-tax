<template>
  <Teleport to="body">
    <div v-if="visible" class="asset-picker-backdrop" @click.self="close">
      <section class="asset-picker-modal" role="dialog" aria-modal="true">
        <header class="asset-picker-head">
          <div>
            <h2>{{ title }}</h2>
            <p>{{ selectedCountText }}</p>
          </div>
          <button class="asset-icon-button" type="button" title="关闭" aria-label="关闭" @click="close">
            <XMarkIcon />
          </button>
        </header>

        <div class="asset-picker-toolbar">
          <div class="asset-search">
            <MagnifyingGlassIcon class="asset-search-icon" />
            <input v-model.trim="keyword" type="search" placeholder="搜索名称、分类、标签" />
          </div>
          <div class="asset-toolbar-actions">
            <select v-model="scope" aria-label="资产库范围">
              <option value="all">全部资产库</option>
              <option value="system">系统资产</option>
              <option value="department">部门资产</option>
              <option value="personal">我的资产</option>
              <option value="favorite">收藏</option>
            </select>
            <input
              v-if="allowStyleReferenceUpload"
              ref="styleUploadInput"
              class="asset-upload-input"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              @change="uploadStyleReference"
            />
            <button
              v-if="allowStyleReferenceUpload"
              class="asset-upload-button"
              type="button"
              :disabled="uploadingStyleReference"
              @click="styleUploadInput?.click()"
            >
              <ArrowPathIcon v-if="uploadingStyleReference" class="asset-upload-spinner" />
              <CloudArrowUpIcon v-else />
              {{ uploadingStyleReference ? '上传中' : '上传新图' }}
            </button>
          </div>
        </div>

        <div class="asset-picker-body">
          <aside class="asset-picker-categories">
            <div class="asset-nav-title">
              <span>资产类型</span>
              <strong>{{ majorCategoryCards.length }}</strong>
            </div>
            <button
              v-for="item in majorCategoryCards"
              :key="item.value"
              type="button"
              class="asset-major-nav"
              :class="{ active: activeMajorCategory === item.value }"
              @click="setMajorCategory(item.value)"
            >
              <span>
                <strong>{{ item.label }}</strong>
                <small>{{ item.subCategoryCount || 1 }} 个小分类</small>
              </span>
              <em>{{ item.count }}</em>
            </button>
          </aside>

          <main class="asset-result-area">
            <div v-if="loading" class="asset-empty">
              <PhotoIcon class="asset-empty-icon" />
              <strong>资产加载中</strong>
            </div>
            <template v-else-if="isSearching">
              <div class="asset-result-heading">
                <div>
                  <strong>搜索结果</strong>
                  <span>{{ searchResultAssets.length }} 个可用资产</span>
                </div>
              </div>
              <div v-if="searchResultAssets.length" class="asset-card-grid">
                <article
                  v-for="asset in searchResultAssets"
                  :key="asset.id"
                  class="asset-pick-card"
                  :class="{ active: isSelected(asset) }"
                >
                  <button type="button" class="asset-thumb" @click="toggleAsset(asset)">
                    <img :src="asset.cover" :alt="asset.name" :class="{ generated: asset.coverGenerated }" loading="lazy" />
                    <span class="asset-owner">{{ ownerLabel(asset.ownerType) }}</span>
                    <span v-if="isSelected(asset)" class="asset-check">
                      <CheckIcon />
                    </span>
                  </button>
                  <button type="button" class="asset-card-meta" @click="toggleAsset(asset)">
                    <strong>{{ asset.name }}</strong>
                    <span>{{ asset.ownerName }} · {{ asset.type }} · {{ asset.category || '未分类' }}</span>
                  </button>
                </article>
              </div>
              <div v-else class="asset-empty">
                <PhotoIcon class="asset-empty-icon" />
                <strong>没有匹配的资产</strong>
                <span>换一个关键词或资产库范围再试</span>
              </div>
            </template>
            <template v-else-if="activeMajorCategory && !activeCategory">
              <div class="asset-result-heading">
                <div>
                  <strong>{{ activeMajorCategoryLabel }}</strong>
                  <span>{{ categoryLevelHint }}</span>
                </div>
              </div>
              <div v-if="categoryCards.length" class="asset-category-grid">
                <button
                  v-for="item in categoryCards"
                  :key="item.value"
                  type="button"
                  class="asset-category-card asset-category-card--tile"
                  @click="setCategory(item.value)"
                >
                  <span class="asset-category-meta">
                    <strong>{{ item.label }}</strong>
                    <small>{{ item.description }}</small>
                  </span>
                  <span class="asset-preview-strip">
                    <img
                      v-for="(preview, index) in item.previews"
                      :key="`${item.value}-${index}`"
                      :src="preview.src"
                      :alt="preview.alt"
                      :class="{ generated: preview.generated }"
                      loading="lazy"
                    />
                  </span>
                </button>
              </div>
              <div v-else class="asset-empty">
                <PhotoIcon class="asset-empty-icon" />
                <strong>没有可用小分类</strong>
                <span>换一个资产大类或资产库范围再试</span>
              </div>
            </template>
            <template v-else-if="isStyleReferenceMajor && activeCategory && !activeSubCategory">
              <div class="asset-result-heading">
                <button type="button" @click="activeCategory = ''">返回空间</button>
                <div>
                  <strong>{{ activeCategory }}</strong>
                  <span>{{ styleCategoryCards.length }} 个风格，选择后查看参考图</span>
                </div>
              </div>
              <div v-if="styleCategoryCards.length" class="asset-category-grid">
                <button
                  v-for="item in styleCategoryCards"
                  :key="item.value"
                  type="button"
                  class="asset-category-card asset-category-card--tile"
                  @click="setSubCategory(item.value)"
                >
                  <span class="asset-category-meta">
                    <strong>{{ item.label }}</strong>
                    <small>{{ item.description }}</small>
                  </span>
                  <span class="asset-preview-strip">
                    <img
                      v-for="(preview, index) in item.previews"
                      :key="`${item.value}-${index}`"
                      :src="preview.src"
                      :alt="preview.alt"
                      :class="{ generated: preview.generated }"
                      loading="lazy"
                    />
                  </span>
                </button>
              </div>
              <div v-else class="asset-empty">
                <PhotoIcon class="asset-empty-icon" />
                <strong>没有可用风格</strong>
                <span>换一个空间或资产库范围再试</span>
              </div>
            </template>
            <template v-else-if="activeCategory && (!isStyleReferenceMajor || activeSubCategory)">
              <div class="asset-result-heading">
                <button type="button" @click="goBackFromAssetList">{{ assetListBackText }}</button>
                <div>
                  <strong>{{ assetListTitle }}</strong>
                  <span>{{ filteredAssets.length }} 个可用资产</span>
                </div>
              </div>
              <div v-if="filteredAssets.length" class="asset-card-grid">
              <article
                v-for="asset in filteredAssets"
                :key="asset.id"
                class="asset-pick-card"
                :class="{ active: isSelected(asset) }"
              >
                <button type="button" class="asset-thumb" @click="toggleAsset(asset)">
                  <img :src="asset.cover" :alt="asset.name" :class="{ generated: asset.coverGenerated }" loading="lazy" />
                  <span class="asset-owner">{{ ownerLabel(asset.ownerType) }}</span>
                  <span v-if="isSelected(asset)" class="asset-check">
                    <CheckIcon />
                  </span>
                </button>
                <button type="button" class="asset-card-meta" @click="toggleAsset(asset)">
                  <strong>{{ asset.name }}</strong>
                  <span>{{ asset.ownerName }} · {{ asset.type }}</span>
                </button>
              </article>
              </div>
              <div v-else class="asset-empty">
                <PhotoIcon class="asset-empty-icon" />
                <strong>没有可用资产</strong>
                <span>换一个小分类或关键词再试</span>
              </div>
            </template>
            <div v-else class="asset-empty">
              <PhotoIcon class="asset-empty-icon" />
              <strong>{{ pickerEmptyTitle }}</strong>
              <span>{{ pickerEmptyHint }}</span>
            </div>
          </main>
        </div>

        <footer class="asset-picker-foot">
          <span>{{ selectedSummaryText }}</span>
          <div>
            <button class="asset-secondary-button" type="button" @click="close">取消</button>
            <button class="asset-primary-button" type="button" :disabled="!selectedAssets.length" @click="confirm">
              使用选中资产
            </button>
          </div>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ArrowPathIcon, CheckIcon, CloudArrowUpIcon, MagnifyingGlassIcon, PhotoIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import request, { getDisplayImageUrl, getDisplayThumbnailUrl } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const props = defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '选择资产' },
  multiple: { type: Boolean, default: false },
  max: { type: Number, default: 1 },
  typeFilter: { type: Array, default: () => [] },
  sourceTypes: { type: Array, default: () => [] },
  allowStyleReferenceUpload: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'select'])
const toast = useToast()

const loading = ref(false)
const assets = ref([])
const selectedAssets = ref([])
const keyword = ref('')
const scope = ref('all')
const activeMajorCategory = ref('')
const activeCategory = ref('')
const activeSubCategory = ref('')
const styleUploadInput = ref(null)
const uploadingStyleReference = ref(false)

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

const allowedTypes = computed(() => props.typeFilter.map(item => String(item)))
const allowedSourceTypes = computed(() => props.sourceTypes.map(item => String(item)))
const selectableAssets = computed(() => {
  return assets.value.filter(asset => {
    const sourceMatched = !allowedSourceTypes.value.length || allowedSourceTypes.value.includes(asset.sourceType)
    const typeMatched = !allowedTypes.value.length || allowedTypes.value.includes(asset.type)
    const customStyleMatched = !props.allowStyleReferenceUpload || asset.sourceType !== 'asset' || asset.ownerType === 'personal'
    return sourceMatched && typeMatched && customStyleMatched && asset.fileUrl
  })
})
const scopedAssets = computed(() => {
  return selectableAssets.value.filter(scopeMatchesAsset)
})
const majorFilteredAssets = computed(() => {
  return scopedAssets.value.filter(majorCategoryMatchesAsset)
})
const isStyleReferenceMajor = computed(() => activeMajorCategory.value === 'style_reference')
const categories = computed(() => {
  if (!activeMajorCategory.value) return []
  return [...new Set(majorFilteredAssets.value.map(categoryGroupKey).filter(Boolean))]
})
const majorCategoryCards = computed(() => {
  return majorCategoryDefs
    .map(def => createMajorCategoryCard({
      ...def,
      assets: scopedAssets.value.filter(asset => majorCategoryOf(asset) === def.value),
    }))
    .filter(item => item.count > 0)
})
const categoryCards = computed(() => {
  return categories.value.map(category => {
    const groupAssets = majorFilteredAssets.value.filter(asset => categoryGroupKey(asset) === category)
    return createCategoryCard(category, groupAssets, isStyleReferenceMajor.value ? 'space' : 'category')
  })
})
const styleCategories = computed(() => {
  if (!isStyleReferenceMajor.value || !activeCategory.value) return []
  const assetsInScene = majorFilteredAssets.value.filter(asset => categoryGroupKey(asset) === activeCategory.value)
  return [...new Set(assetsInScene.map(styleGroupKey).filter(Boolean))]
})
const styleCategoryCards = computed(() => {
  return styleCategories.value.map(style => {
    const groupAssets = majorFilteredAssets.value.filter(asset => categoryGroupKey(asset) === activeCategory.value && styleGroupKey(asset) === style)
    return createCategoryCard(style, groupAssets, 'style')
  })
})
const activeMajorCategoryLabel = computed(() => {
  if (!activeMajorCategory.value) return '资产库'
  return majorCategoryDefs.find(item => item.value === activeMajorCategory.value)?.label || '资产'
})
const categoryLevelHint = computed(() => {
  if (isStyleReferenceMajor.value) return `${categoryCards.value.length} 个空间，先选择厨房/浴室`
  return `${categoryCards.value.length} 个小分类，选择后查看资产`
})
const assetListTitle = computed(() => {
  if (isStyleReferenceMajor.value) return `${activeCategory.value} / ${activeSubCategory.value}`
  return activeCategory.value
})
const assetListBackText = computed(() => isStyleReferenceMajor.value ? '返回风格' : '返回小分类')
const filteredAssets = computed(() => {
  if (!activeMajorCategory.value || !activeCategory.value) return []
  if (isStyleReferenceMajor.value && !activeSubCategory.value) return []
  const text = keyword.value.toLowerCase()
  return majorFilteredAssets.value.filter(asset => {
    const categoryMatched = categoryGroupKey(asset) === activeCategory.value
    const subCategoryMatched = !isStyleReferenceMajor.value || styleGroupKey(asset) === activeSubCategory.value
    const searchText = assetSearchText(asset)
    return categoryMatched && subCategoryMatched && (!text || searchText.includes(text))
  })
})
const isSearching = computed(() => keyword.value.trim().length > 0)
const searchResultAssets = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return []
  const baseAssets = activeMajorCategory.value ? majorFilteredAssets.value : scopedAssets.value
  return baseAssets.filter(asset => {
    const searchText = assetSearchText(asset)
    return searchText.includes(text)
  })
})
const selectedCountText = computed(() => {
  if (props.multiple) return `已选 ${selectedAssets.value.length}/${Math.max(1, props.max)}`
  return selectedAssets.value.length ? '已选择 1 个资产' : '请选择 1 个资产'
})
const selectedSummaryText = computed(() => {
  if (!selectedAssets.value.length) return selectedCountText.value
  const names = selectedAssets.value.slice(0, 2).map(asset => asset.name).join('、')
  const more = selectedAssets.value.length > 2 ? ` 等 ${selectedAssets.value.length} 个资产` : ''
  return `已选：${names}${more}`
})
const pickerEmptyTitle = computed(() => {
  if (!activeMajorCategory.value) return '请选择资产大类'
  if (!activeCategory.value) return isStyleReferenceMajor.value ? '请选择空间' : '请选择小分类'
  if (isStyleReferenceMajor.value && !activeSubCategory.value) return '请选择风格'
  return '没有可用资产'
})
const pickerEmptyHint = computed(() => {
  if (!activeMajorCategory.value) return '先从左侧选择模型文件、风格参考图、色块等大类'
  if (!activeCategory.value) return isStyleReferenceMajor.value ? '先选择厨房、浴室等空间' : '再选择该大类下的一个小分类'
  if (isStyleReferenceMajor.value && !activeSubCategory.value) return '再选择该空间下的风格'
  return '换一个分类或关键词再试'
})

watch(() => props.visible, (visible) => {
  if (!visible) return
  selectedAssets.value = []
  keyword.value = ''
  scope.value = 'all'
  activeMajorCategory.value = ''
  activeCategory.value = ''
  activeSubCategory.value = ''
  fetchAssets()
})

watch(categories, (next) => {
  if (!next.includes(activeCategory.value)) activeCategory.value = ''
})
watch(styleCategories, (next) => {
  if (!next.includes(activeSubCategory.value)) activeSubCategory.value = ''
})

watch(majorCategoryCards, (next) => {
  if (!next.some(item => item.value === activeMajorCategory.value)) {
    activeMajorCategory.value = ''
    activeCategory.value = ''
    activeSubCategory.value = ''
  }
  if (!activeMajorCategory.value && next.length === 1) {
    activeMajorCategory.value = next[0].value
  }
})

watch(categoryCards, (next) => {
  if (!next.some(item => item.value === activeCategory.value)) {
    activeCategory.value = ''
    activeSubCategory.value = ''
  }
  if (activeMajorCategory.value && !activeCategory.value && next.length === 1) {
    activeCategory.value = next[0].value
  }
})
watch(styleCategoryCards, (next) => {
  if (!next.some(item => item.value === activeSubCategory.value)) {
    activeSubCategory.value = ''
  }
  if (isStyleReferenceMajor.value && activeCategory.value && !activeSubCategory.value && next.length === 1) {
    activeSubCategory.value = next[0].value
  }
})

watch(scope, () => {
  activeMajorCategory.value = ''
  activeCategory.value = ''
  activeSubCategory.value = ''
})

async function fetchAssets() {
  loading.value = true
  try {
    const res = await request.get('/assets')
    assets.value = (res.data?.items || []).map(mapAsset)
  } catch (error) {
    assets.value = []
    toast.error(error?.response?.data?.error || '资产库加载失败')
  } finally {
    loading.value = false
  }
}

async function uploadStyleReference(event) {
  const input = event.target
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 50 * 1024 * 1024) {
    toast.warning('请上传 50MB 以内的图片')
    return
  }

  uploadingStyleReference.value = true
  try {
    const categoryName = '我的风格参考图'
    await request.post('/asset-categories', {
      owner_type: 'personal',
      type: '风格参考图',
      name: categoryName,
    })

    const form = new FormData()
    form.append('owner_type', 'personal')
    form.append('name', assetNameFromFile(file))
    form.append('type', '风格参考图')
    form.append('category', categoryName)
    form.append('tags', '风格参考图')
    form.append('file', file)
    const res = await request.post('/assets', form)
    const uploadedAsset = mapAsset(res.data)

    assets.value = [uploadedAsset, ...assets.value.filter(asset => asset.id !== uploadedAsset.id)]
    selectedAssets.value = [uploadedAsset]
    keyword.value = ''
    scope.value = 'personal'
    await nextTick()
    activeMajorCategory.value = 'style_reference'
    activeCategory.value = categoryGroupKey(uploadedAsset)
    activeSubCategory.value = styleGroupKey(uploadedAsset)
    toast.success('上传成功，已为你选中')
  } catch (error) {
    toast.error(error?.response?.data?.error || '风格参考图上传失败')
  } finally {
    uploadingStyleReference.value = false
  }
}

function assetNameFromFile(file) {
  return String(file?.name || '').replace(/\.[^.]+$/, '').trim() || '风格参考图'
}

function mapAsset(item) {
  const sourceType = item.source_type || 'asset'
  const sourceId = item.source_id || ''
  const preview = assetPreviewFromApi(item)
  return {
    id: item.id || `${sourceType}:${sourceId}`,
    sourceType,
    sourceId,
    name: item.name || '未命名资产',
    fileName: item.file_name || item.fileName || '',
    type: item.type || '文件',
    category: item.category || '',
    ownerType: item.owner_type || 'personal',
    ownerName: item.owner_name || '',
    favorite: !!item.favorite,
    tags: Array.isArray(item.tags) ? item.tags : [],
    ...preview,
    thumbnailUrl: preview.cover,
    fileUrl: item.file_url || '',
    fileType: item.file_type || '',
    referenceSceneId: item.reference_scene_id || 0,
    referenceSceneName: item.reference_scene_name || '',
    referenceTagId: item.reference_tag_id || 0,
    referenceTagName: item.reference_tag_name || '',
    referenceGalleryId: item.reference_gallery_id || 0,
  }
}

function scopeMatchesAsset(asset) {
  return scope.value === 'all'
    || asset.ownerType === scope.value
    || (scope.value === 'favorite' && asset.favorite)
}

function majorCategoryMatchesAsset(asset) {
  return !activeMajorCategory.value || majorCategoryOf(asset) === activeMajorCategory.value
}

function majorCategoryOf(asset) {
  const matched = majorCategoryDefs.find(def => def.types.includes(asset.type))
  return matched?.value || 'other'
}

function setMajorCategory(category) {
  activeMajorCategory.value = category
  activeCategory.value = ''
  activeSubCategory.value = ''
}

function setCategory(category) {
  activeCategory.value = category
  activeSubCategory.value = ''
}

function setSubCategory(category) {
  activeSubCategory.value = category
}

function goBackFromAssetList() {
  if (isStyleReferenceMajor.value) {
    activeSubCategory.value = ''
    return
  }
  activeCategory.value = ''
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

function createCategoryCard(category, groupAssets, level = 'category') {
  const previews = groupAssets.slice(0, 4).map(asset => assetPreviewItem(asset)).filter(Boolean)
  const styleCount = new Set(groupAssets.map(styleGroupKey).filter(Boolean)).size
  const typeCount = new Set(groupAssets.map(asset => asset.type).filter(Boolean)).size
  return {
    label: category,
    value: category,
    shortLabel: `${groupAssets.length}`,
    count: groupAssets.length,
    typeCount,
    description: categoryCardDescription(level, groupAssets.length, styleCount, typeCount),
    previews,
  }
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

function assetSearchText(asset) {
  return [asset.name, asset.type, asset.category, asset.referenceSceneName, asset.referenceTagName, asset.ownerName, ...asset.tags].join(' ').toLowerCase()
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

function ownerLabel(ownerType) {
  if (ownerType === 'system') return '系统'
  if (ownerType === 'department') return '部门'
  return '我的'
}

function isSelected(asset) {
  return selectedAssets.value.some(item => item.id === asset.id)
}

function toggleAsset(asset) {
  if (isSelected(asset)) {
    selectedAssets.value = selectedAssets.value.filter(item => item.id !== asset.id)
    return
  }
  if (!props.multiple) {
    selectedAssets.value = [asset]
    return
  }
  const limit = Math.max(1, props.max)
  if (selectedAssets.value.length >= limit) {
    toast.warning(`最多选择 ${limit} 个资产`)
    return
  }
  selectedAssets.value = [...selectedAssets.value, asset]
}

async function confirm() {
  const selected = [...selectedAssets.value]
  selected.forEach(asset => {
    request.post('/assets/usage', {
      source_type: asset.sourceType,
      source_id: asset.sourceId,
    }).catch(() => {})
  })
  emit('select', props.multiple ? selected : selected[0])
}

function close() {
  emit('close')
}
</script>

<style scoped>
.asset-picker-backdrop {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgba(15, 23, 42, .46);
  backdrop-filter: blur(3px);
}

.asset-picker-modal {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  width: min(1440px, calc(100vw - 32px));
  height: min(900px, calc(100vh - 32px));
  overflow: hidden;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 24px 70px rgba(15, 23, 42, .22);
}

.asset-picker-head,
.asset-picker-toolbar,
.asset-picker-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 22px;
  border-bottom: 1px solid #edf0f5;
}

.asset-picker-head h2,
.asset-picker-head p {
  margin: 0;
}

.asset-picker-head h2 {
  color: #111827;
  font-size: 18px;
  font-weight: 800;
}

.asset-picker-head p,
.asset-picker-foot span {
  margin-top: 3px;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.asset-icon-button {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fff;
  color: #475467;
  cursor: pointer;
}

.asset-icon-button svg {
  width: 18px;
  height: 18px;
}

.asset-picker-toolbar {
  background: #f8fafc;
}

.asset-search {
  position: relative;
  flex: 1;
  max-width: 420px;
}

.asset-search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  width: 16px;
  height: 16px;
  color: #98a2b3;
  transform: translateY(-50%);
}

.asset-search input,
.asset-picker-toolbar select {
  height: 40px;
  border: 1px solid #d7dde8;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  font-size: 14px;
  outline: none;
}

.asset-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.asset-search input {
  width: 100%;
  padding: 0 12px 0 36px;
}

.asset-picker-toolbar select {
  min-width: 150px;
  padding: 0 12px;
}

.asset-upload-input {
  display: none;
}

.asset-upload-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  height: 40px;
  padding: 0 14px;
  border: 1px solid #111827;
  border-radius: 8px;
  background: #111827;
  color: #fff;
  font-size: 13px;
  font-weight: 800;
  white-space: nowrap;
  cursor: pointer;
  transition: background .16s, border-color .16s, transform .16s;
}

.asset-upload-button:hover:not(:disabled) {
  background: #293241;
  border-color: #293241;
  transform: translateY(-1px);
}

.asset-upload-button:disabled {
  border-color: #98a2b3;
  background: #98a2b3;
  cursor: wait;
}

.asset-upload-button svg {
  width: 17px;
  height: 17px;
}

.asset-upload-spinner {
  animation: asset-upload-spin .8s linear infinite;
}

@keyframes asset-upload-spin {
  to { transform: rotate(360deg); }
}

.asset-picker-body {
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr);
  min-height: 0;
  background: #fff;
}

.asset-picker-categories {
  display: grid;
  align-content: start;
  gap: 6px;
  padding: 14px 10px;
  border-right: 1px solid #edf0f5;
  background: #fafbfc;
  overflow: auto;
}

.asset-nav-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 10px 10px;
  color: #98a2b3;
  font-size: 12px;
  font-weight: 800;
}

.asset-nav-title strong {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 22px;
  border-radius: 999px;
  background: #eef2f7;
  color: #667085;
  font-size: 11px;
}

.asset-major-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 52px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #475467;
  cursor: pointer;
  padding: 9px 10px;
  text-align: left;
  transition: background .16s, border-color .16s, color .16s;
}

.asset-major-nav:hover {
  background: #fff;
  border-color: #e4e8ef;
  color: #111827;
}

.asset-major-nav.active {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.asset-major-nav span {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.asset-major-nav strong,
.asset-major-nav small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-major-nav strong {
  font-size: 14px;
  font-weight: 800;
}

.asset-major-nav small {
  color: #98a2b3;
  font-size: 12px;
  font-weight: 700;
}

.asset-major-nav.active small {
  color: rgba(255, 255, 255, .68);
}

.asset-major-nav em {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 30px;
  height: 26px;
  border-radius: 999px;
  background: #eef2f7;
  color: #667085;
  font-size: 12px;
  font-style: normal;
  font-weight: 800;
}

.asset-major-nav.active em {
  background: rgba(255, 255, 255, .16);
  color: #fff;
}

.asset-category-card {
  display: grid;
  gap: 10px;
  width: 100%;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fff;
  color: #475467;
  cursor: pointer;
  padding: 10px;
  text-align: left;
  transition: border-color .2s, box-shadow .2s, transform .2s;
}

.asset-category-card:hover,
.asset-category-card.active {
  border-color: #111827;
  color: #111827;
  box-shadow: 0 10px 22px rgba(15, 23, 42, .08);
  transform: translateY(-1px);
}

.asset-category-card.compact {
  padding: 9px;
}

.asset-category-card--tile {
  min-height: 118px;
  grid-template-rows: auto 68px;
  align-content: start;
}

.asset-category-meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.asset-category-meta strong,
.asset-category-meta small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-category-meta strong {
  flex: 1;
  min-width: 0;
  color: #111827;
  font-size: 14px;
  font-weight: 800;
}

.asset-category-meta small {
  flex-shrink: 0;
  color: #98a2b3;
  font-size: 12px;
  font-weight: 700;
}

.asset-preview-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 4px;
  height: 68px;
  overflow: hidden;
  border-radius: 8px;
  background: #eef2f7;
}

.asset-preview-strip img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: #e4e8ef;
}

.asset-preview-strip img.generated {
  object-fit: contain;
  padding: 6px;
}

.asset-subcategory-title {
  margin-top: 4px;
  padding: 4px 2px 0;
  color: #667085;
  font-size: 12px;
  font-weight: 800;
}

.asset-result-area {
  min-height: 0;
  overflow: auto;
  padding: 18px;
}

.asset-result-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.asset-result-heading > div {
  display: grid;
  gap: 2px;
}

.asset-result-heading strong {
  color: #111827;
  font-size: 15px;
  font-weight: 800;
}

.asset-result-heading span {
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.asset-result-heading button {
  height: 32px;
  border: 1px solid #d7dde8;
  border-radius: 8px;
  background: #fff;
  color: #475467;
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
  padding: 0 10px;
}

.asset-result-heading button:hover {
  border-color: #111827;
  color: #111827;
}

.asset-category-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.asset-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 12px;
}

.asset-pick-card {
  overflow: hidden;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #fff;
  transition: border-color .2s, box-shadow .2s, transform .2s;
}

.asset-pick-card:hover,
.asset-pick-card.active {
  border-color: #111827;
  box-shadow: 0 12px 26px rgba(15, 23, 42, .1);
}

.asset-thumb {
  position: relative;
  display: block;
  width: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border: 0;
  background: #f3f6fa;
  cursor: pointer;
}

.asset-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-thumb img.generated {
  object-fit: contain;
  padding: 24px;
}

.asset-owner,
.asset-check {
  position: absolute;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.asset-owner {
  left: 8px;
  top: 8px;
  min-height: 24px;
  padding: 0 8px;
  background: rgba(255, 255, 255, .92);
  color: #111827;
  font-size: 11px;
  font-weight: 800;
}

.asset-check {
  right: 8px;
  top: 8px;
  width: 26px;
  height: 26px;
  background: #111827;
  color: #fff;
}

.asset-check svg {
  width: 15px;
  height: 15px;
}

.asset-card-meta {
  display: grid;
  gap: 3px;
  width: 100%;
  border: 0;
  background: #fff;
  padding: 10px;
  text-align: left;
  cursor: pointer;
}

.asset-card-meta strong,
.asset-card-meta span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-card-meta strong {
  color: #111827;
  font-size: 13px;
}

.asset-card-meta span {
  color: #667085;
  font-size: 12px;
}

.asset-empty {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  min-height: 420px;
  border: 1px dashed #d7dde8;
  border-radius: 8px;
  background: #f8fafc;
  color: #667085;
  text-align: center;
}

.asset-empty strong {
  color: #172033;
  font-size: 15px;
}

.asset-empty-icon {
  width: 34px;
  height: 34px;
  color: #98a2b3;
}

.asset-picker-foot {
  border-top: 1px solid #edf0f5;
  border-bottom: 0;
}

.asset-picker-foot > div {
  display: flex;
  gap: 10px;
}

.asset-secondary-button,
.asset-primary-button {
  height: 38px;
  border-radius: 8px;
  padding: 0 16px;
  font-size: 14px;
  font-weight: 800;
  cursor: pointer;
}

.asset-secondary-button {
  border: 1px solid #d7dde8;
  background: #fff;
  color: #172033;
}

.asset-primary-button {
  border: 1px solid #111827;
  background: #111827;
  color: #fff;
}

.asset-primary-button:disabled {
  border-color: #d1d5db;
  background: #d1d5db;
  cursor: not-allowed;
}

@media (max-width: 760px) {
  .asset-picker-backdrop {
    padding: 8px;
  }

  .asset-picker-modal {
    width: calc(100vw - 16px);
    height: calc(100vh - 16px);
  }

  .asset-picker-toolbar,
  .asset-picker-foot {
    align-items: stretch;
    flex-direction: column;
  }

  .asset-search {
    max-width: none;
  }

  .asset-toolbar-actions {
    width: 100%;
  }

  .asset-picker-toolbar select,
  .asset-upload-button {
    flex: 1;
    min-width: 0;
  }

  .asset-picker-body {
    grid-template-columns: 1fr;
  }

  .asset-picker-categories {
    display: flex;
    overflow-x: auto;
    border-right: 0;
    border-bottom: 1px solid #edf0f5;
  }

  .asset-major-nav {
    min-width: 156px;
  }

  .asset-nav-title {
    display: none;
  }

  .asset-card-grid,
  .asset-category-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
