<template>
  <MainLayout>
    <main class="review-page">
      <section class="filters" aria-label="评论筛选">
        <Listbox v-model="filters.brand"><div class="filter-select"><ListboxButton class="select-button" aria-label="品牌"><span>{{ filters.brand || '所有品牌' }}</span><ChevronDownIcon class="select-chevron" /></ListboxButton><ListboxOptions class="select-options"><ListboxOption value="" class="select-option">所有品牌</ListboxOption><ListboxOption v-for="value in brands" :key="value" :value="value" class="select-option">{{ value }}</ListboxOption></ListboxOptions></div></Listbox>
        <Listbox v-model="filters.country"><div class="filter-select"><ListboxButton class="select-button" aria-label="国家"><span>{{ filters.country || '所有国家' }}</span><ChevronDownIcon class="select-chevron" /></ListboxButton><ListboxOptions class="select-options"><ListboxOption value="" class="select-option">所有国家</ListboxOption><ListboxOption v-for="value in countries" :key="value" :value="value" class="select-option">{{ value }}</ListboxOption></ListboxOptions></div></Listbox>
        <Listbox v-model="filters.rating"><div class="filter-select rating-filter"><ListboxButton class="select-button" aria-label="评分"><span>{{ filters.rating ? filters.rating + ' 星' : '所有评分' }}</span><ChevronDownIcon class="select-chevron" /></ListboxButton><ListboxOptions class="select-options"><ListboxOption value="" class="select-option">所有评分</ListboxOption><ListboxOption v-for="score in [5,4,3,2,1]" :key="score" :value="String(score)" class="select-option">{{ score }} 星</ListboxOption></ListboxOptions></div></Listbox>
        <CustomsDateRangePicker v-model:from="filters.from" v-model:to="filters.to" label="评论时间" />
        <label class="search"><input v-model="filters.keyword" aria-label="搜索评论或 ASIN" placeholder="搜索评论或 ASIN"><MagnifyingGlassIcon class="h-4 w-4" /></label>
        <button class="reset-button" type="button" @click="resetFilters">重置</button>
      </section>
      <p v-if="error" class="error-banner" role="alert">{{ error }}</p>
      <section class="workspace" aria-label="评论明细">
        <div class="table-toolbar">
          <div class="toolbar-actions">
            <label class="tool-button import-button" :class="{ disabled: importing }"><ArrowUpTrayIcon class="h-4 w-4" />{{ importing ? '正在读取…' : '导入评论表' }}<input type="file" accept=".xlsx,.csv" :disabled="importing" @change="importFile"></label>
            <button class="tool-button" type="button" :disabled="!filteredReviews.length" @click="exportCsv"><ArrowDownTrayIcon class="h-4 w-4" />导出</button>
          </div>
          <button class="refresh-button" type="button" aria-label="刷新评论" title="刷新评论" :disabled="loading || importing" @click="loadReviews"><ArrowPathIcon class="h-4 w-4" :class="{ spinning: loading }" /></button>
        </div>
        <div class="table-scroll" tabindex="0" role="region" aria-label="评论明细表，可横向滚动" :aria-busy="loading">
          <table class="review-table">
            <caption class="sr-only">评论明细，仅展示已有的评论字段</caption>
            <colgroup><col class="image-col"><col class="asin-col"><col class="rating-col"><col class="title-col"><col class="content-col"><col class="asin-col"><col class="brand-col"><col class="name-col"><col class="country-col"><col class="date-col"><col class="link-col"><col class="review-images-col"><col class="video-col"><col class="action-col"></colgroup>
            <thead><tr>
              <th scope="col" class="pinned-image">产品图</th><th scope="col" class="pinned-asin">子ASIN</th><th scope="col">评分</th><th scope="col">评论标题</th><th scope="col">评论内容</th><th scope="col">父ASIN</th><th scope="col">品牌</th><th scope="col">姓名</th><th scope="col">国家</th><th scope="col">评论时间</th><th scope="col">评论链接</th><th scope="col">评论图片</th><th scope="col">评论视频</th><th scope="col" class="pinned-action">操作</th>
            </tr></thead>
            <tbody>
              <tr v-if="loading && !reviews.length"><td colspan="14" class="empty" role="status">正在加载评论…</td></tr>
              <tr v-else-if="!reviews.length"><td colspan="14" class="empty"><span>暂无评论数据</span><label class="tool-button empty-cta">{{ importing ? '正在读取…' : '导入评论表' }}<input type="file" accept=".xlsx,.csv" :disabled="importing" @change="importFile"></label></td></tr>
              <tr v-else-if="!filteredReviews.length"><td colspan="14" class="empty">无匹配评论<button class="text-button" type="button" @click="resetFilters">重置筛选</button></td></tr>
              <tr v-for="review in pagedReviews" :key="review.id" class="review-row">
                <td class="pinned-image"><ProductThumbnail :src="review.product_image_url || catalogImages.get(productKey(review)) || ''" :alt="(review.child_asin || '产品') + ' 产品图'" /></td>
                <td class="pinned-asin"><span class="asin-text">{{ review.child_asin || '—' }}</span></td>
                <td><div v-if="review.rating" class="rating" :aria-label="review.rating + ' 星'"><span class="stars" aria-hidden="true"><span v-for="star in 5" :key="star" :class="{ filled: star <= review.rating }">★</span></span><span>{{ review.rating }}</span></div><span v-else class="muted">—</span></td>
                <td><p class="cell-clamp" :title="review.title">{{ review.title || '（无标题）' }}</p></td>
                <td><p class="review-content" :class="{ expanded: expanded.has(review.id) }" :title="review.content">{{ review.content || '（无评论内容）' }}</p><button v-if="(review.content || '').length > 100" class="text-button" type="button" :aria-expanded="expanded.has(review.id)" @click="toggleExpanded(review.id)">{{ expanded.has(review.id) ? '收起' : '展开' }}</button></td>
                <td>{{ review.parent_asin || '—' }}</td>
                <td><p class="cell-clamp" :title="review.brand">{{ review.brand || '—' }}</p></td>
                <td><p class="cell-clamp" :title="review.name">{{ review.name || '匿名' }}</p></td>
                <td>{{ review.country || '—' }}</td>
                <td><time :title="review.date">{{ review.date || '—' }}</time></td>
                <td><a v-if="safeUrl(review.link)" class="review-link" :href="safeUrl(review.link)" target="_blank" rel="noopener noreferrer">原评论<ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" /></a><span v-else class="muted">—</span></td>
                <td><div v-if="imageLinks(review.review_images ?? review.image).length" class="image-strip"><button v-for="(url, imageIndex) in imageLinks(review.review_images ?? review.image)" :key="url" type="button" @click="openMedia('image', imageLinks(review.review_images ?? review.image), imageIndex)" :aria-label="'查看评论图片 ' + (imageIndex + 1)"><img :src="url" loading="lazy" :alt="'评论图片 ' + (imageIndex + 1)"></button></div><span v-else class="muted">—</span></td>
                <td><div class="review-video">
                  <div v-if="reviewVideoUrls(review).length" class="image-strip video-strip"><button v-for="(url, index) in reviewVideoUrls(review)" :key="url" type="button" @click="openMedia('video', reviewVideoUrls(review), index)" :aria-label="'播放评论视频 ' + (index + 1)"><video :src="videoPosterUrl(url)" preload="metadata" muted playsinline aria-hidden="true" tabindex="-1"></video><span class="play-badge" aria-hidden="true"><PlayIcon class="h-3.5 w-3.5" /></span></button></div>
                  <p v-if="review.video_upload && !videoLinks(review.video_upload).length" class="cell-clamp" :title="review.video_upload">{{ review.video_upload }}</p>
                  <span v-if="review.video && !review.video_link && !review.video_upload" class="video-tag">含视频</span>
                  <span v-else-if="!review.video && !review.video_link && !review.video_upload" class="muted">—</span>
                  <p v-if="review.video_link && !safeUrl(review.video_link)" class="cell-clamp" :title="review.video_link">{{ review.video_link }}</p>
                </div></td>
                <td class="pinned-action"><button class="action-button" type="button" @click="openActions(review)">操作<ChevronDownIcon class="select-chevron" /></button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <footer class="pagination"><span>共 {{ filteredReviews.length.toLocaleString() }} 条<span class="page-size">每页 {{ pageSize }} 条</span></span><div><button type="button" :disabled="page <= 1" @click="page--">上一页</button><span>{{ page }} / {{ totalPages }}</span><button type="button" :disabled="page >= totalPages" @click="page++">下一页</button></div></footer>
      </section>
      <Dialog :open="!!activeReview" class="review-action-dialog" :style="{ position: 'fixed', inset: 0, zIndex: 100 }" @close="closeActions">
        <div class="action-overlay">
        <div class="dialog-backdrop" aria-hidden="true" />
        <div class="dialog-position"><DialogPanel class="action-panel">
          <header class="action-heading"><div class="action-heading-icon"><VideoCameraIcon /></div><div><DialogTitle>上传评论视频</DialogTitle><p>为这条评论添加视频附件</p></div><button type="button" class="action-close" aria-label="关闭评论操作" :disabled="savingVideo" @click="closeActions"><XMarkIcon /></button></header>
          <div class="action-body">
            <section class="action-summary"><span class="summary-asin">{{ activeReview?.child_asin || '未填写 ASIN' }}</span><p>{{ activeReview?.title || '（无标题）' }}</p><a v-if="safeUrl(activeReview?.link)" class="review-link" :href="safeUrl(activeReview?.link)" target="_blank" rel="noopener noreferrer">查看原评论<ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" /></a></section>
            <form id="review-video-form" @submit.prevent="uploadReviewVideo">
              <div class="upload-section-heading"><label for="review-video-file">评论视频</label><span>单个文件 · 最大 100 MB</span></div>
              <label class="video-dropzone" :class="{ 'is-dragging': draggingVideo, 'is-disabled': savingVideo }" @dragover.prevent="draggingVideo = !savingVideo" @dragleave.prevent="draggingVideo = false" @drop.prevent="dropVideo">
                <input id="review-video-file" type="file" accept=".mp4,.mov,.webm,video/mp4,video/quicktime,video/webm" :disabled="savingVideo" @change="selectVideo">
                <ArrowUpTrayIcon /><strong>{{ videoFile ? '点击更换视频' : '点击选择或拖入视频' }}</strong><span>支持 MP4、MOV、WebM 格式</span>
              </label>
              <div v-if="videoFile" class="selected-video"><VideoCameraIcon /><div><strong>{{ videoFile.name }}</strong><span>{{ (videoFile.size / 1024 / 1024).toFixed(1) }} MB · 待上传</span></div><button type="button" aria-label="移除已选视频" :disabled="savingVideo" @click="clearVideoFile"><XMarkIcon /></button></div>
              <div v-if="videoPreview || videoLinks(activeReview?.video_upload)[0]" class="video-preview"><video :key="videoPreview || activeReview?.video_upload" :src="videoPreview || videoLinks(activeReview?.video_upload)[0]" controls preload="metadata" playsinline /><span>{{ videoPreview ? '本地预览，上传后保存到此评论' : '当前已上传视频' }}</span></div>
              <p v-else-if="activeReview?.video_upload" class="existing-video">当前附件：{{ activeReview.video_upload }}</p>
              <p v-if="videoFile && activeReview?.video_upload" class="upload-hint">上传成功后将替换当前视频附件。</p>
              <div v-if="savingVideo" class="upload-progress" role="status"><div><span>{{ videoProgress >= 100 ? '正在保存视频…' : '正在上传视频…' }}</span><strong>{{ videoProgress }}%</strong></div><progress :value="videoProgress" max="100" aria-label="视频上传进度" /></div>
              <p v-if="videoError" class="video-error" role="alert">{{ videoError }}</p>
            </form>
          </div>
          <footer class="action-footer"><span>视频将关联至当前评论</span><button class="tool-button" type="button" :disabled="savingVideo" @click="closeActions">取消</button><button class="tool-button save-video" type="submit" form="review-video-form" :disabled="savingVideo || !videoFile"><ArrowUpTrayIcon class="h-4 w-4" />{{ savingVideo ? '上传中…' : '上传并保存' }}</button></footer>
        </DialogPanel></div>
        </div>
      </Dialog>
      <ReviewMediaPreview :media="previewMedia" @close="previewMedia = null" />
      <p v-if="importSummary" class="toast-note" role="status">{{ importSummary }}</p>
    </main>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { Listbox, ListboxButton, ListboxOptions, ListboxOption, Dialog, DialogPanel, DialogTitle } from '@headlessui/vue'
import { ArrowDownTrayIcon, ArrowPathIcon, ArrowTopRightOnSquareIcon, ArrowUpTrayIcon, ChevronDownIcon, MagnifyingGlassIcon, PlayIcon, VideoCameraIcon, XMarkIcon } from '@heroicons/vue/24/solid'
import CustomsDateRangePicker from '@/components/CustomsDateRangePicker.vue'
import { createReviewProductImageLoader, reviewDateKey } from '@/utils/reviewAnalysis'
import ReviewMediaPreview from '@/components/ReviewMediaPreview.vue'
import ProductThumbnail from '@/components/ui/ProductThumbnail.vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { resolveImageUrl } from '@/utils/request'

const headers = ['品牌', '国家', '父ASIN', '子ASIN', '评分', '姓名', '评论时间', '评论标题', '评论内容', '评论链接', '图片链接', '视频附件']
const exportHeaders = headers
const reviews = ref([])
const importing = ref(false)
const loading = ref(false)
const error = ref('')
const importSummary = ref('')
const page = ref(1)
const pageSize = 30
const expanded = reactive(new Set())
const activeReview = ref(null)
const previewMedia = ref(null)
function openMedia(type, urls, index = 0) { previewMedia.value = { type, urls, index } }
const videoFile = ref(null)
const videoPreview = ref('')
const videoProgress = ref(0)
const draggingVideo = ref(false)
const videoError = ref('')
const savingVideo = ref(false)
const catalogImages = reactive(new Map())
let loadProductImage = createReviewProductImageLoader(request)
let imageLoadVersion = 0
const productKey = review => String(review.child_asin || '').trim().toUpperCase()
const filters = reactive({ keyword: '', brand: '', country: '', rating: '', from: '', to: '' })
onMounted(loadReviews)
const brands = computed(() => [...new Set(reviews.value.map(row => row.brand).filter(Boolean))].sort())
const countries = computed(() => [...new Set(reviews.value.map(row => row.country).filter(Boolean))].sort())
const filteredReviews = computed(() => reviews.value.filter(row => {
  const query = filters.keyword.trim().toLocaleLowerCase()
  const date = reviewDateKey(row.date)
  return (!query || [row.brand, row.country, row.parent_asin, row.child_asin, row.name, row.title, row.content].some(value => String(value || '').toLocaleLowerCase().includes(query))) && (!filters.brand || row.brand === filters.brand) && (!filters.country || row.country === filters.country) && (!filters.rating || row.rating === Number(filters.rating)) && (!filters.from || (date && date >= filters.from)) && (!filters.to || (date && date <= filters.to))
}))
const totalPages = computed(() => Math.max(1, Math.ceil(filteredReviews.value.length / pageSize)))
const pagedReviews = computed(() => filteredReviews.value.slice((page.value - 1) * pageSize, page.value * pageSize))
watch(filters, () => { page.value = 1 })
watch(totalPages, value => { if (page.value > value) page.value = value })
watch(pagedReviews, loadVisibleProductImages)
onBeforeUnmount(() => { imageLoadVersion++; clearVideoFile() })

async function loadVisibleProductImages(rows) {
  const version = ++imageLoadVersion
  const load = loadProductImage
  const keys = [...new Set(rows.filter(row => !row.product_image_url).map(productKey).filter(Boolean))]
  let cursor = 0
  await Promise.all(Array.from({ length: Math.min(4, keys.length) }, async () => {
    while (cursor < keys.length && version === imageLoadVersion) {
      const key = keys[cursor++]
      try {
        const image = await load(key)
        if (version === imageLoadVersion) catalogImages.set(key, image)
      } catch {
        if (version === imageLoadVersion) error.value = '产品库图片加载失败，请点击刷新重试'
      }
    }
  }))
}

function resetFilters() { Object.assign(filters, { keyword: '', brand: '', country: '', rating: '', from: '', to: '' }) }
function toggleExpanded(key) { expanded.has(key) ? expanded.delete(key) : expanded.add(key) }
function safeUrl(value) { try { const url = new URL(String(value || '').trim()); return ['http:', 'https:'].includes(url.protocol) ? url.href : '' } catch { return '' } }
function imageLinks(value) { return String(value || '').split(/[\r\n,，;；]+/).map(safeUrl).filter(Boolean) }
async function loadReviews() {
  loading.value = true; error.value = ''
  imageLoadVersion++; catalogImages.clear(); loadProductImage = createReviewProductImageLoader(request)
  try { const response = await request.get('/buyer-reviews'); reviews.value = response.data?.data || [] }
  catch (cause) { error.value = cause?.response?.data?.error || '评论加载失败' }
  finally { loading.value = false }
}
function cellText(value) { if (value == null) return ''; if (value instanceof Date) return value.toISOString().slice(0, 10); return String(value).trim() }
function normalizeHeader(value) { return cellText(value).replace(/[\s\u3000]/g, '').toLowerCase() }
function mapRows(headersRow, rows) {
  const positions = new Map(headersRow.map((header, index) => [normalizeHeader(header), index]))
  const missing = headers.filter(header => !positions.has(normalizeHeader(header)))
  if (missing.length) throw new Error(`缺少模板列：${missing.join('、')}`)
  const get = (row, header) => cellText(row[positions.get(normalizeHeader(header))])
  return rows.filter(row => row.some(value => cellText(value))).map((row, index) => ({
    key: `${index}-${get(row, '子ASIN')}-${get(row, '评论时间')}`,
    brand: get(row, '品牌'), country: get(row, '国家'), parentAsin: get(row, '父ASIN'), childAsin: get(row, '子ASIN'),
    rating: Number.parseFloat(get(row, '评分')) || 0, name: get(row, '姓名'), date: get(row, '评论时间'), title: get(row, '评论标题'),
    content: get(row, '评论内容'), link: get(row, '评论链接'), image: '', reviewImages: get(row, '图片链接'), video: !!get(row, '视频附件'),
    videoUpload: get(row, '视频附件'), videoLink: '',
  }))
}
async function importFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  importing.value = true; error.value = ''; importSummary.value = ''
  try {
    let next
    if (/\.csv$/i.test(file.name)) {
      const text = new TextDecoder('utf-8').decode(await file.arrayBuffer())
      const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/).filter(Boolean)
      const parse = line => { const cells = []; let current = ''; let quoted = false; for (let i = 0; i < line.length; i++) { const ch = line[i]; if (ch === '"' && line[i + 1] === '"' && quoted) { current += '"'; i++ } else if (ch === '"') quoted = !quoted; else if (ch === ',' && !quoted) { cells.push(current); current = '' } else current += ch } cells.push(current); return cells }
      next = mapRows(parse(lines[0]), lines.slice(1).map(parse))
    } else {
      const data = new FormData(); data.append('file', file)
      const response = await request.post('/buyer-reviews/import-workbook', data)
      await loadReviews(); page.value = 1
      importSummary.value = `已导入 ${response.data.imported} 条评论，${response.data.videos} 个视频附件`
      return
    }
    const payload = next.map(({ brand, country, parentAsin, childAsin, rating, name, date, title, content, link, image, reviewImages, video, videoUpload }) => ({ brand, country, parent_asin: parentAsin, child_asin: childAsin, rating, name, date, title, content, link, image, review_images: reviewImages, video, video_upload: videoUpload }))
    await request.post('/buyer-reviews/import', payload)
    await loadReviews(); page.value = 1; importSummary.value = `已导入 ${next.length.toLocaleString()} 条`
  } catch (cause) { error.value = cause?.response?.data?.error || cause?.message || '评论表读取失败，请检查文件格式和表头。' }
  finally { importing.value = false }
}
function videoLinks(value) { return String(value || '').split(/[\r\n,，;；]+/).map(value => mediaUrl(value.trim())).filter(Boolean) }
// reviewVideoUrls lists every playable video of one review (upload links
// first, then the manually maintained video link) for the thumbnail strip.
function reviewVideoUrls(review) { return [...videoLinks(review.video_upload), ...videoLinks(review.video_link)] }
// videoPosterUrl points a muted preview <video> at the first frame.
function videoPosterUrl(url) { return url }
function mediaUrl(value) { return String(value || '').startsWith('/uploads/') ? resolveImageUrl(value) : safeUrl(value) }
function clearVideoFile() { if (videoPreview.value) URL.revokeObjectURL(videoPreview.value); videoPreview.value = ''; videoFile.value = null }
function openActions(review) { clearVideoFile(); activeReview.value = review; videoError.value = ''; videoProgress.value = 0; draggingVideo.value = false }
function closeActions() { if (!savingVideo.value) { activeReview.value = null; clearVideoFile() } }
function chooseVideo(files) {
  if (savingVideo.value || !files?.length) return
  videoError.value = ''
  if (files.length !== 1) { videoError.value = '一次只能上传一个视频'; return }
  const file = files[0]
  if (!/\.(mp4|mov|webm)$/i.test(file.name)) { videoError.value = '请选择 MP4、MOV 或 WebM 视频'; return }
  if (!file.size || file.size > 100 * 1024 * 1024) { videoError.value = '请选择非空且不超过 100 MB 的视频'; return }
  clearVideoFile(); videoFile.value = file; videoPreview.value = URL.createObjectURL(file); videoProgress.value = 0
}
function selectVideo(event) { chooseVideo(event.target.files); event.target.value = '' }
function dropVideo(event) { draggingVideo.value = false; chooseVideo(event.dataTransfer.files) }
async function uploadReviewVideo() {
  const review = activeReview.value
  if (!review || !videoFile.value || savingVideo.value) return
  const data = new FormData(); data.append('video', videoFile.value)
  savingVideo.value = true; videoError.value = ''; videoProgress.value = 0
  try {
    const response = await request.post(`/buyer-reviews/${review.id}/video`, data, { onUploadProgress: event => { if (event.total) videoProgress.value = Math.min(100, Math.round(event.loaded / event.total * 100)) } })
    if (!response.data?.video_upload) throw new Error('服务器未返回视频文件，请重试')
    review.video_upload = response.data.video_upload; review.video = true
    activeReview.value = null; clearVideoFile(); importSummary.value = '评论视频上传成功'
  } catch (cause) { videoError.value = cause?.response?.data?.error || cause?.message || '视频上传失败，请重试' }
  finally { savingVideo.value = false }
}
function exportCsv() {
  const rows = [exportHeaders, ...filteredReviews.value.map(row => [row.brand, row.country, row.parent_asin, row.child_asin, row.rating, row.name, row.date, row.title, row.content, row.link, row.review_images ?? row.image, row.video_upload || row.video_link || ''])]
  const csv = '\uFEFF' + rows.map(row => row.map(value => `"${String(value ?? '').replaceAll('"', '""')}"`).join(',')).join('\r\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' })); const anchor = document.createElement('a'); anchor.href = url; anchor.download = '评论分析-筛选结果.csv'; anchor.click(); URL.revokeObjectURL(url)
}
</script>

<style scoped>
.review-page{--ink:#303640;--muted:#8b929c;--line:#e6e9ee;--accent:#387cc1;--hover:#f0f7ff;min-width:0;padding:0 16px 24px;color:var(--ink);font-size:13px;font-variant-numeric:tabular-nums}
.filters{display:flex;align-items:center;flex-wrap:wrap;gap:8px;padding:12px 16px;margin:0 -16px;background:#fff;border-bottom:1px solid var(--line)}
.filter-select{position:relative;min-width:126px;max-width:240px}.rating-filter{min-width:110px}
.select-button,.search,.reset-button,.tool-button{height:34px;border:1px solid #d6dbe2;border-radius:4px;background:#fff;font-size:12px}
.select-button{display:flex;width:100%;align-items:center;justify-content:space-between;gap:16px;padding:0 11px;cursor:pointer;color:#5d6672}
.select-button span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.select-chevron{width:13px;height:13px;flex:none;color:#969eaa}
.select-options{position:absolute;z-index:30;top:calc(100% + 4px);left:0;min-width:100%;max-height:240px;overflow:auto;padding:4px;background:white;border:1px solid var(--line);border-radius:4px;box-shadow:0 6px 18px #26395014}
.select-option{padding:8px 10px;color:#46515b;font-size:12px;cursor:pointer;white-space:nowrap}
.select-option[data-headlessui-state~="active"]{background:var(--hover);color:var(--accent)}.select-option[data-headlessui-state~="selected"]{font-weight:600;color:var(--accent)}
.search{display:flex;align-items:center;gap:10px;width:300px;max-width:100%;padding:0 10px;color:#89929f}
.search input{width:100%;min-width:0;border:0;outline:0;background:transparent;font:inherit;color:var(--ink)}.search input::placeholder{color:#a0a7b1}
.reset-button{padding:0 15px;background:#eaecf0;border-color:#eaecf0;color:#545d68;cursor:pointer}
.workspace{min-width:0}.table-toolbar{display:flex;align-items:center;gap:16px;flex-wrap:wrap;min-height:54px;padding:10px 0;background:#f1f3f6}
.toolbar-actions{display:flex;align-items:center;gap:8px}.tool-button{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:0 11px;color:#485362;cursor:pointer;white-space:nowrap}
.import-button input,.empty-cta input{position:absolute;width:1px;height:1px;opacity:0;overflow:hidden}.tool-button:disabled,.tool-button.disabled,.refresh-button:disabled{opacity:.5;cursor:default}
.refresh-button{display:grid;place-items:center;margin-left:auto;padding:7px;border:0;border-radius:4px;background:transparent;color:#75808f;cursor:pointer}
.table-scroll{max-height:calc(100dvh - 250px);min-height:240px;overflow:auto;background:#fff;border:1px solid var(--line);border-radius:4px 4px 0 0;scrollbar-color:#cfd5df #f8f9fb;scrollbar-width:thin}
.review-table{width:100%;min-width:2050px;border-collapse:separate;border-spacing:0;table-layout:fixed;text-align:left;font-size:12px}
.image-col{width:84px}.asin-col{width:136px}.rating-col{width:132px}.title-col{width:200px}.content-col{width:320px}.brand-col{width:130px}.name-col{width:160px}.country-col{width:90px}.date-col{width:125px}.link-col{width:105px}.review-images-col{width:160px}.video-col{width:200px}.action-col{width:86px}
.review-table th{position:sticky;top:0;z-index:3;height:42px;background:#fff;padding:10px 12px;border-bottom:1px solid var(--line);font-weight:600;color:#303640;white-space:nowrap}
.review-table th:not(:last-child)::after{position:absolute;right:0;top:14px;width:1px;height:14px;background:#e0e4e9;content:''}
.review-table td{height:76px;padding:12px;vertical-align:top;border-bottom:1px solid var(--line);overflow-wrap:anywhere;background:#fff;transition:background .15s}
.review-table .pinned-image{position:sticky;left:0;z-index:2}.review-table .pinned-asin{position:sticky;left:84px;z-index:2;border-right:1px solid #edf0f3;box-shadow:3px 0 5px -4px #aab5c4}.review-table th.pinned-image,.review-table th.pinned-asin{z-index:4}
.review-row:hover td,.review-row:focus-within td{background:var(--hover)}.asin-text{color:var(--accent);font-weight:500;white-space:nowrap}.muted{color:var(--muted)}
.rating{display:flex;align-items:center;gap:7px;white-space:nowrap}.stars{color:#ccd2da;font-size:14px;letter-spacing:1px;line-height:18px}.stars .filled{color:#efb83d}
.cell-clamp,.review-content{display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:3;overflow:hidden;line-height:1.6;margin:0;white-space:pre-wrap}.review-content.expanded{display:block}
.text-button{padding:3px 0 0;border:0;background:transparent;color:var(--accent);font-size:12px;cursor:pointer}
.image-strip{display:flex;flex-wrap:wrap;gap:4px}.image-strip button{padding:0;cursor:pointer;display:block;width:40px;height:40px;border:1px solid #edf0f3;border-radius:2px;overflow:hidden;background:#fff}.image-strip img{width:100%;height:100%;object-fit:contain}
.video-strip button{position:relative;background:#18212d}.video-strip video{width:100%;height:100%;object-fit:contain;pointer-events:none}.play-badge{position:absolute;inset:0;display:grid;place-items:center;color:#fff;background:rgba(16,22,30,.38)}.play-badge svg{filter:drop-shadow(0 1px 2px #00000073)}.video-strip button:hover .play-badge{background:rgba(22,86,148,.55)}
.review-link{display:inline-flex;align-items:center;gap:4px;color:var(--accent);text-decoration:none;white-space:nowrap}.review-link:hover,.text-button:hover{text-decoration:underline}
.video-tag{display:inline-block;padding:1px 6px;border:1px solid #c6daef;border-radius:2px;background:#eff6ff;color:var(--accent);font-size:11px;white-space:nowrap}
.review-table td.empty{height:240px;text-align:left;vertical-align:middle;padding-left:24px;color:var(--muted)}.empty-cta,.empty .text-button{margin-left:16px}
.pagination{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;background:#fff;border:1px solid var(--line);border-top:0;border-radius:0 0 4px 4px;color:#78818e;font-size:12px}.page-size{margin-left:18px;color:#9199a4}.pagination div{display:flex;align-items:center;gap:12px}
.pagination button{border:1px solid #d6dbe2;border-radius:3px;background:#fff;padding:5px 10px;color:#606b78;font:inherit;cursor:pointer}.pagination button:disabled{opacity:.4;cursor:default}
.select-button:hover,.tool-button:hover:not(:disabled),.pagination button:hover:not(:disabled){border-color:#9cbddd;color:var(--accent)}.refresh-button:hover:not(:disabled),.reset-button:hover{background:#e2e8ef}
.import-button:focus-within,.empty-cta:focus-within,.review-page :is(button,a,input,[tabindex]):focus-visible{outline:2px solid #90b9e5;outline-offset:2px}.review-page .search input:focus-visible{outline:none;box-shadow:none}.search:focus-within{border-color:var(--accent);box-shadow:0 0 0 2px #387cc11a}.search input{height:100%;padding:0;box-shadow:none}.search svg{flex-shrink:0}
.error-banner{margin:10px 0;padding:10px 13px;border:1px solid #f0ceca;border-radius:4px;background:#fff4f3;color:#a1453b;font-size:12px}
.toast-note{position:fixed;right:24px;bottom:22px;border-radius:4px;background:#303c4b;color:#fff;padding:10px 14px;font-size:12px;box-shadow:0 4px 18px #26395020;z-index:60}
.spinning{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:900px){.review-page{padding:0 12px 20px}.filters{margin-left:-12px;margin-right:-12px;padding-left:12px;padding-right:12px}.table-scroll{max-height:65dvh}.search{flex:1;min-width:200px}}
@media(max-width:560px){.filters{gap:6px}.filter-select{min-width:0;max-width:none;flex:1 1 calc(33.333% - 4px)}.select-button{gap:6px;padding:0 8px}.search{min-width:0;flex:1 1 calc(100% - 64px)}.table-toolbar{gap:10px}.pagination{padding:10px 8px;font-size:11px}.pagination div{gap:8px}.page-size{display:none}}
@media(prefers-reduced-motion:reduce){*,*::before,*::after{transition:none!important;animation:none!important}}
.review-table .pinned-action{position:sticky;right:0;z-index:2;border-left:1px solid var(--line);box-shadow:-3px 0 5px -4px #aab5c4}.review-table th.pinned-action{z-index:4}
.action-button{display:inline-flex;align-items:center;gap:5px;border:0;background:transparent;padding:0;color:var(--accent);font:inherit;cursor:pointer}.action-button:hover{text-decoration:underline}.action-button .select-chevron{color:inherit;width:12px}
.review-video{display:flex;flex-direction:column;align-items:flex-start;gap:6px}
.action-overlay{position:fixed;inset:0;z-index:100;isolation:isolate}.dialog-backdrop{position:absolute;inset:0;z-index:0;background:rgba(23,35,51,.4)}.dialog-position{position:absolute;inset:0;z-index:1;display:flex;align-items:center;justify-content:center;padding:20px;overflow-y:auto;overscroll-behavior:contain} .action-panel{position:relative;display:flex;flex-direction:column;width:560px;max-width:100%;max-height:calc(100dvh - 40px);overflow:hidden;flex-shrink:0;background:#fff;border:1px solid #e0e6ed;border-radius:12px;color:#303640;font-size:13px;box-shadow:0 20px 60px #17233333}
.action-heading{display:flex;align-items:center;gap:12px;padding:20px 24px;border-bottom:1px solid #e8edf3;flex-shrink:0}.action-heading-icon{display:grid;place-items:center;width:40px;height:40px;border-radius:10px;background:#edf5ff;color:#387cc1}.action-heading-icon svg{width:22px;height:22px}.action-heading h2{font-size:17px;font-weight:600;margin:0}.action-heading p{margin:4px 0 0;color:#8993a1;font-size:12px}.action-close{margin-left:auto;display:grid;place-items:center;width:30px;height:30px;border:0;border-radius:5px;background:transparent;color:#8793a2;cursor:pointer}.action-close svg,.selected-video button svg{width:18px;height:18px}.action-close:hover{background:#f0f3f7}
.action-body{padding:20px 24px;overflow-y:auto;min-height:0}.action-summary{padding:14px 16px;background:#f6f8fb;border:1px solid #e9edf3;border-radius:7px}.summary-asin{font-size:12px;font-weight:600;color:#387cc1}.action-summary p{margin:6px 0 8px;line-height:1.6;overflow-wrap:anywhere;max-height:80px;overflow:auto}.action-summary .review-link{font-size:12px;color:#387cc1}.upload-section-heading{display:flex;justify-content:space-between;gap:8px;align-items:center;margin:20px 0 10px}.upload-section-heading label{font-weight:600}.upload-section-heading span{font-size:11px;color:#8993a1}
.video-dropzone{position:relative;display:flex;flex-direction:column;align-items:center;gap:9px;padding:24px 12px;border:1px dashed #b9cce2;border-radius:8px;background:#f8fbff;cursor:pointer;transition:background .15s,border-color .15s}.video-dropzone>svg{width:26px;height:26px;color:#387cc1}.video-dropzone strong{font-size:13px;font-weight:500;color:#387cc1}.video-dropzone span{font-size:12px;color:#8993a1}.video-dropzone input{position:absolute;width:1px;height:1px;opacity:0}.video-dropzone:hover,.video-dropzone.is-dragging{border-color:#387cc1;background:#edf5ff}.video-dropzone:focus-within{outline:2px solid #90b9e5;outline-offset:2px}.video-dropzone.is-disabled{opacity:.6;cursor:wait}
.selected-video{display:flex;align-items:center;gap:10px;margin-top:12px;padding:10px 12px;border:1px solid #e3e9f0;border-radius:6px}.selected-video>svg{width:22px;height:22px;flex-shrink:0;color:#687c93}.selected-video>div{min-width:0;flex:1}.selected-video strong{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:12px;font-weight:500}.selected-video span{display:block;margin-top:4px;color:#8993a1;font-size:11px}.selected-video button{border:0;background:transparent;color:#8993a1;cursor:pointer}.video-preview{margin-top:14px}.video-preview video{display:block;width:100%;max-height:220px;border-radius:6px;background:#18212d}.video-preview>span,.upload-hint,.existing-video{display:block;margin:8px 0 0;color:#8993a1;font-size:12px;overflow-wrap:anywhere}.upload-progress{margin-top:14px}.upload-progress>div{display:flex;justify-content:space-between;color:#387cc1;font-size:12px}.upload-progress progress{display:block;width:100%;height:6px;margin-top:8px;accent-color:#387cc1}.video-error{padding:10px 12px;margin:12px 0 0;border:1px solid #f0d5d1;border-radius:6px;background:#fff6f5;color:#a1453b;font-size:12px}
.action-footer{display:flex;align-items:center;gap:10px;padding:16px 24px;border-top:1px solid #e8edf3;background:#fafbfd;flex-shrink:0}.action-footer>span{margin-right:auto;color:#8993a1;font-size:11px}.action-footer .tool-button{height:36px;padding:0 16px}.action-footer .save-video{background:#387cc1;border-color:#387cc1;color:#fff}.action-footer .save-video:hover:not(:disabled){background:#2d6cac}.action-panel button:disabled{opacity:.5;cursor:default}.action-panel :is(button,a):focus-visible{outline:2px solid #90b9e5;outline-offset:2px}
@media(max-width:560px){.action-heading,.action-body{padding:16px}.action-footer{padding:14px 16px}.action-footer>span{display:none}.action-footer{justify-content:flex-end}.upload-section-heading{align-items:flex-start;flex-wrap:wrap}}
</style>
