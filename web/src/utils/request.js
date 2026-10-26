import axios from 'axios'
import router from '@/router'

export const downloadBrandOptions = [
  'DEAROCEAN',
  'FEECOOL',
  'FORIOUS',
  'QFN',
  'SUNOUTLY',
  '无品牌',
  'Androme',
  'ANZA',
  'FURUISI',
  'VEUUI',
]
const enableDownloadSkuPicker = true

const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 3000000,
})

/**
 * 将后端返回的相对图片路径转换为可访问的绝对 URL。
 * 生产环境中后端返回 /uploads/... 形式的相对路径，需要拼接后端 origin。
 */
export function resolveImageUrl(url) {
  if (!url || url.startsWith('data:') || url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  if (apiBase.startsWith('http')) {
    return new URL(apiBase).origin + url
  }
  return url
}

/**
 * 获取用于在页面上展示的图片 URL（自动处理 OSS CORS 问题）。
 *
 * - data: URL / 非 OSS URL：直接返回
 * - OSS 图片：返回后端代理 URL，通过 /api/proxy 避免跨域
 *
 * 注意：对于需要用 <img src="..."> 直接展示的图片，必须使用此函数。
 * resolveImageUrl 适合用于需要实际 URL 的场景（如背景图、CSS 等）。
 */
export function getDisplayImageUrl(url) {
  if (!url) return ''

  // data: URL 直接返回
  if (url.startsWith('data:')) return url

  // OSS 图片走代理
  if (url.includes('oss-cn-')) {
    const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
    return `${apiBase}/proxy?url=${encodeURIComponent(url)}`
  }

  // 其他 HTTP(S) URL 或相对路径直接返回
  return url
}

export function getDisplayThumbnailUrl(url, options = {}) {
  if (!url) return ''
  if (url.startsWith('data:') || url.startsWith('blob:')) return url

  const sourceUrl = getProxyTargetUrl(url) || url
  const width = Number(options.width || 360)
  const hasHeight = options.height !== null
  const height = hasHeight ? Number(options.height || width) : 0
  const quality = Number(options.quality || 80)
  const fit = options.fit || 'fill'
  const thumbnailUrl = withThumbnailProcess(sourceUrl, width, height, quality, fit)
  return getDisplayImageUrl(thumbnailUrl)
}

function getProxyTargetUrl(url) {
  try {
    const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost'
    const parsed = new URL(url, base)
    if (!parsed.pathname.endsWith('/proxy')) return ''
    return parsed.searchParams.get('url') || ''
  } catch {
    return ''
  }
}

function withThumbnailProcess(url, width, height, quality, fit) {
  if (!isOSSImageUrl(url)) return url
  const heightPart = height > 0 ? `,h_${height}` : ''
  const processValue = `image/resize,m_${fit},w_${width}${heightPart},limit_0/quality,q_${quality}/format,webp`
  try {
    const parsed = new URL(url)
    if (!parsed.searchParams.has('x-oss-process')) {
      parsed.searchParams.set('x-oss-process', processValue)
    }
    return parsed.toString()
  } catch {
    const separator = url.includes('?') ? '&' : '?'
    if (url.includes('x-oss-process=')) return url
    return `${url}${separator}x-oss-process=${encodeURIComponent(processValue)}`
  }
}

function isOSSImageUrl(url) {
  return typeof url === 'string' && url.includes('oss-cn-') && /\.(png|jpe?g|webp|gif|bmp|avif)(\?|#|$)/i.test(url)
}

export async function fetchImageAsDataUrl(url) {
  if (!url) throw new Error('图片地址为空')

  const fetchUrl = getImageFetchUrl(url)
  const token = localStorage.getItem('token')
  const res = await fetch(fetchUrl, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`图片加载失败：HTTP ${res.status}`)

  const blob = await res.blob()
  const mimeType = normalizeImageMimeType(blob.type, url)
  if (!mimeType.startsWith('image/')) throw new Error('请选择图片资产')

  const dataUrl = await blobToDataUrl(blob, mimeType)
  return {
    dataUrl,
    data: dataUrl.includes(',') ? dataUrl.split(',')[1] : dataUrl,
    mimeType,
    blob,
  }
}

function getImageFetchUrl(url) {
  if (!url || url.startsWith('data:') || url.startsWith('blob:')) return url

  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  const resolved = resolveImageUrl(url)
  const proxyPath = `${apiBase}/proxy`
  if (resolved.startsWith(proxyPath)) return resolved

  if (/^https?:\/\//i.test(resolved)) {
    const sameOrigin = typeof window !== 'undefined' && resolved.startsWith(window.location.origin)
    if (sameOrigin) return resolved
    return `${proxyPath}?url=${encodeURIComponent(resolved)}`
  }

  return resolved
}

function normalizeImageMimeType(type, url = '') {
  const cleanType = String(type || '').split(';')[0].trim().toLowerCase()
  if (cleanType.startsWith('image/')) return cleanType

  const cleanUrl = String(url || '').split('?')[0].split('#')[0].toLowerCase()
  if (cleanUrl.endsWith('.jpg') || cleanUrl.endsWith('.jpeg')) return 'image/jpeg'
  if (cleanUrl.endsWith('.webp')) return 'image/webp'
  if (cleanUrl.endsWith('.gif')) return 'image/gif'
  if (cleanUrl.endsWith('.bmp')) return 'image/bmp'
  if (cleanUrl.endsWith('.avif')) return 'image/avif'
  if (cleanUrl.endsWith('.svg')) return 'image/svg+xml'
  return 'image/png'
}

function blobToDataUrl(blob, mimeType) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = String(reader.result || '')
      const data = result.includes(',') ? result.split(',')[1] : result
      resolve(`data:${mimeType};base64,${data}`)
    }
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

/**
 * 强制触发浏览器下载图片。
 *
 * 对于后端 /uploads/ 路径下的图片，通过 /api/download 代理接口下载：
 *   - 走完整 Gin 链路，携带 CORS 响应头，避免 nginx 静态文件无 CORS 问题
 *   - 后端设置 Content-Disposition: attachment，浏览器必定触发下载而非预览
 *
 * data:、blob: 和外部图片在需要标注时，会先上传到内存处理接口。
 */
// transformBlob applies page-specific compositing after confirmation and before metadata.
export async function downloadImage(url, filename, metadata = {}, transformBlob) {
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  const token = localStorage.getItem('token')
  const finalFilename = filename || `image-${Date.now()}.png`
  const canAnnotateSyntheticPerformer = isSupportedMetadataImageDownload(url, finalFilename)

  /** 提取 URL 中的 /uploads/... 路径部分 */
  let uploadsPath = null
  try {
    const urlObj = new URL(url)
    if (urlObj.pathname.startsWith('/uploads/')) {
      uploadsPath = urlObj.pathname
    }
  } catch {
    if (typeof url === 'string' && url.startsWith('/uploads/')) {
      uploadsPath = url
    }
  }

  let downloadMeta = metadata
  const isInlineImage = typeof url === 'string' && (url.startsWith('data:image/') || url.startsWith('blob:'))
  if (enableDownloadSkuPicker && !metadata.skipSkuPicker && (uploadsPath || metadata.forceSkuPicker || isInlineImage || canAnnotateSyntheticPerformer)) {
    downloadMeta = await requestDownloadMetadata(apiBase, token, {
      ...metadata,
      filename: finalFilename,
      canAnnotateSyntheticPerformer,
    })
    if (downloadMeta === null) return null
  }
  let downloadFilename = downloadMeta.filename || finalFilename
  const shouldAnnotate = canAnnotateSyntheticPerformer && downloadMeta.annotateSyntheticPerformer === true

  if (url && (url.startsWith('data:') || url.startsWith('blob:'))) {
    try {
      const res = await fetch(url)
      let blob = await res.blob()
      if (transformBlob) blob = await transformBlob(blob)
      if (shouldAnnotate) {
        const annotated = await annotateSyntheticPerformerBlob(apiBase, token, blob, downloadFilename, downloadMeta)
        blob = annotated.blob
        downloadFilename = annotated.filename || downloadFilename
      }
      triggerBlobDownload(blob, downloadFilename)
    } catch (error) {
      if (transformBlob) throw error
      window.alert(shouldAnnotate ? '图片元数据标注失败，请重试' : '图片下载失败，请重试')
    }
    return downloadMeta
  }

  if (uploadsPath) {
    /** 走后端 /api/download 代理，携带 Authorization，后端设置 attachment 头 */
    const params = new URLSearchParams({
      path: uploadsPath,
      filename: downloadFilename,
    })
    if (typeof url === 'string' && /^https?:\/\//i.test(url)) params.set('source_url', url)
    if (downloadMeta.brand) params.set('brand', downloadMeta.brand)
    if (downloadMeta.sku) params.set('sku', downloadMeta.sku)
    if (shouldAnnotate && !transformBlob) params.set('annotate', '1')
    const downloadUrl = `${apiBase}/download?${params.toString()}`
    try {
      const res = await fetch(downloadUrl, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      let blob = await res.blob()
      downloadFilename = getDownloadFilename(res) || downloadFilename
      if (transformBlob) {
        blob = await transformBlob(blob)
        if (shouldAnnotate) {
          const annotated = await annotateSyntheticPerformerBlob(apiBase, token, blob, downloadFilename, downloadMeta)
          blob = annotated.blob
          downloadFilename = annotated.filename || downloadFilename
        }
      }
      triggerBlobDownload(blob, downloadFilename)
    } catch (error) {
      if (transformBlob) throw error
      if (shouldAnnotate) window.alert('图片元数据标注失败，请重试')
      else window.open(url, '_blank')
    }
    return downloadMeta
  }

  /** 其他 URL：直接 fetch → blob（同域或已允许 CORS 的外部链接） */
  try {
    const fetchUrl = shouldAnnotate ? getImageFetchUrl(url) : url
    const res = await fetch(fetchUrl, {
      headers: shouldAnnotate && token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    let blob = await res.blob()
    if (transformBlob) blob = await transformBlob(blob)
    if (shouldAnnotate) {
      const annotated = await annotateSyntheticPerformerBlob(apiBase, token, blob, downloadFilename, downloadMeta)
      blob = annotated.blob
      downloadFilename = annotated.filename || downloadFilename
    }
    triggerBlobDownload(blob, downloadFilename)
    return downloadMeta
  } catch (error) {
    if (transformBlob) throw error
    if (shouldAnnotate) window.alert('图片元数据标注失败，请重试')
    else window.open(url, '_blank')
  }
}

function isSupportedMetadataImageDownload(url, filename) {
  const name = String(filename || '').split('?')[0].split('#')[0]
  if (/\.(png|jpe?g|webp)$/i.test(name)) return true
  const source = String(url || '')
  if (/^data:image\/(png|jpe?g|webp)[;,]/i.test(source)) return true
  return /\.(png|jpe?g|webp)(\?|#|$)/i.test(source)
}

async function annotateSyntheticPerformerBlob(apiBase, token, blob, filename, metadata) {
  const form = new FormData()
  form.append('file', blob, filename)
  form.append('filename', filename)
  if (metadata.brand) form.append('brand', metadata.brand)
  if (metadata.sku) form.append('sku', metadata.sku)
  const response = await fetch(`${apiBase}/image-metadata/synthetic-performer`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  return {
    blob: await response.blob(),
    filename: getDownloadFilename(response),
  }
}

function triggerBlobDownload(blob, filename) {
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  // 调用方历史上经常把文件名写死为 .png，但接口返回的图片可能仍是 JPEG。
  // 以实际 Blob MIME 为准修正扩展名，避免下载后的文件名和文件内容不一致。
  a.download = normalizeImageDownloadFilename(blob, filename)
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(objectUrl)
}

function normalizeImageDownloadFilename(blob, filename) {
  const mimeType = String(blob?.type || '').split(';')[0].trim().toLowerCase()
  const extension = {
    'image/jpeg': 'jpg',
    'image/jpg': 'jpg',
    'image/png': 'png',
    'image/webp': 'webp',
    'image/gif': 'gif',
    'image/bmp': 'bmp',
    'image/avif': 'avif',
  }[mimeType]
  if (!extension) return filename

  const current = String(filename || `image-${Date.now()}`)
  if (/\.[^.]+$/.test(current)) return current.replace(/\.[^.]+$/, `.${extension}`)
  return `${current}.${extension}`
}

function getDownloadFilename(response) {
  const disposition = response.headers.get('content-disposition')
  if (!disposition) return ''
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) return decodeURIComponent(utf8Match[1])
  const match = disposition.match(/filename="?([^"]+)"?/i)
  return match?.[1] || ''
}

export async function requestDownloadMetadata(apiBase, token, initial = {}) {
  let options = { brands: downloadBrandOptions }
  try {
    const res = await fetch(`${apiBase}/download-options`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) {
      options = await res.json()
    }
  } catch {
    // Keep the dialog usable even if options cannot be loaded.
  }

  const brands = Array.isArray(options.brands) && options.brands.length ? options.brands : downloadBrandOptions
  const selectedBrand = brands.includes(initial.brand) ? initial.brand : ''

  return showDownloadSkuDialog({
    apiBase,
    token,
    brands,
    brand: selectedBrand,
    sku: initial.sku || '',
    filename: initial.filename || '',
    canAnnotateSyntheticPerformer: initial.canAnnotateSyntheticPerformer === true,
  })
}

function showDownloadSkuDialog({ apiBase, token, brands, brand, sku, filename, canAnnotateSyntheticPerformer }) {
  return new Promise((resolve) => {
    const overlay = document.createElement('div')
    overlay.className = 'download-sku-overlay'
    const brandOptions = (brands || downloadBrandOptions)
      .map((item) => `<option value="${escapeHtml(item)}"${item === brand ? ' selected' : ''}>${escapeHtml(item)}</option>`)
      .join('')
    overlay.innerHTML = `
      <div class="download-sku-dialog" role="dialog" aria-modal="true">
        <div class="download-sku-head">
          <strong>${/\.mp4$/i.test(filename) ? '下载视频' : '下载图片'}</strong>
          <button type="button" class="download-sku-close" aria-label="关闭">×</button>
        </div>
        <label class="download-sku-field">
          <span>品牌</span>
          <span class="download-brand-select-wrap">
            <select class="download-brand-select">
              <option value="">请选择品牌</option>
              ${brandOptions}
            </select>
          </span>
        </label>
        <label class="download-sku-field">
          <span>SKU</span>
          <input type="text" class="download-sku-input" placeholder="请输入 SKU" value="${escapeHtml(sku || '')}" autocomplete="off">
        </label>
        ${canAnnotateSyntheticPerformer ? `<fieldset class="download-synthetic-field">
          <legend>图中是否包含 AI 生成的人物或人物部位？<span>必选</span></legend>
          <p>人物部位包括手、脚、半张脸等。</p>
          <div class="download-synthetic-options">
            <label>
              <input type="radio" name="download-synthetic-performer" value="yes">
              <span class="download-synthetic-option">
                <strong>是，包含</strong>
                <small>下载时写入 AI 人物标注</small>
              </span>
            </label>
            <label>
              <input type="radio" name="download-synthetic-performer" value="no">
              <span class="download-synthetic-option">
                <strong>否，不包含</strong>
                <small>保持图片原始元数据</small>
              </span>
            </label>
          </div>
          <div class="download-synthetic-required">请选择“是”或“否”后再下载</div>
        </fieldset>` : ''}
        <div class="download-sku-results"></div>
        <div class="download-sku-actions">
          <button type="button" class="download-sku-cancel">取消</button>
          <button type="button" class="download-sku-confirm" disabled>下载</button>
        </div>
      </div>
    `
    ensureDownloadSkuStyles()
    document.body.appendChild(overlay)

    const brandSelect = overlay.querySelector('.download-brand-select')
    const input = overlay.querySelector('.download-sku-input')
    const results = overlay.querySelector('.download-sku-results')
    const confirm = overlay.querySelector('.download-sku-confirm')
    const syntheticInputs = [...overlay.querySelectorAll('input[name="download-synthetic-performer"]')]
    const syntheticRequired = overlay.querySelector('.download-synthetic-required')
    let selectedBrand = brandSelect.value
    let selected = sku ? { sku } : null
    let syntheticChoice = null
    let timer = null
    const updateConfirm = () => {
      const hasSyntheticChoice = !canAnnotateSyntheticPerformer || syntheticChoice !== null
      confirm.disabled = !(selectedBrand && input.value.trim() && hasSyntheticChoice)
      if (syntheticRequired) syntheticRequired.classList.toggle('is-complete', hasSyntheticChoice)
    }
    const close = (value) => {
      document.body.removeChild(overlay)
      resolve(value)
    }
    const setSelected = (item) => {
      selected = item
      input.value = item.sku || ''
      updateConfirm()
      renderResults([item])
    }
    const renderResults = (items) => {
      results.innerHTML = ''
      if (!items.length) {
        results.innerHTML = '<div class="download-sku-empty">没有匹配的 SKU，可直接使用输入内容下载</div>'
        return
      }
      for (const item of items) {
        const button = document.createElement('button')
        button.type = 'button'
        button.className = `download-sku-option${selected?.sku === item.sku ? ' is-active' : ''}`
        button.innerHTML = `
          <strong>${escapeHtml(item.sku || '')}</strong>
          <span>${escapeHtml([item.style_name, item.color, item.size, item.category2].filter(Boolean).join(' / '))}</span>
        `
        button.addEventListener('click', () => setSelected(item))
        results.appendChild(button)
      }
    }
    const search = async () => {
      const q = input.value.trim()
      selected = selected?.sku === q ? selected : null
      updateConfirm()
      if (!q) {
        results.innerHTML = '<div class="download-sku-empty">请输入关键词搜索 SKU</div>'
        return
      }
      try {
        const res = await fetch(`${apiBase}/product-skus?q=${encodeURIComponent(q)}`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        })
        renderResults(res.ok ? await res.json() : [])
      } catch {
        results.innerHTML = '<div class="download-sku-empty">搜索失败，请重试</div>'
      }
    }

    overlay.querySelector('.download-sku-close').addEventListener('click', () => close(null))
    overlay.querySelector('.download-sku-cancel').addEventListener('click', () => close(null))
    overlay.addEventListener('click', (event) => {
      if (event.target === overlay) close(null)
    })
    brandSelect.addEventListener('change', () => {
      selectedBrand = brandSelect.value
      updateConfirm()
    })
    syntheticInputs.forEach((radio) => {
      radio.addEventListener('change', () => {
        syntheticChoice = radio.value === 'yes'
        updateConfirm()
      })
    })
    const submitDownload = () => {
      const inputSku = input.value.trim()
      if (selectedBrand && inputSku && (!canAnnotateSyntheticPerformer || syntheticChoice !== null)) {
        close({ brand: selectedBrand, sku: inputSku, annotateSyntheticPerformer: syntheticChoice === true })
      }
    }
    confirm.addEventListener('click', () => {
      submitDownload()
    })
    input.addEventListener('input', () => {
      updateConfirm()
      window.clearTimeout(timer)
      timer = window.setTimeout(search, 200)
    })
    input.addEventListener('keydown', (event) => {
      if (event.key === 'Enter') submitDownload()
      if (event.key === 'Escape') close(null)
    })

    if (sku) {
      updateConfirm()
      search()
    } else {
      results.innerHTML = '<div class="download-sku-empty">请输入 SKU</div>'
    }
    input.focus()
  })
}

function escapeHtml(value) {
  return String(value || '').replace(/[&<>"']/g, (char) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  })[char])
}

function ensureDownloadSkuStyles() {
  if (document.getElementById('download-sku-styles')) return
  const style = document.createElement('style')
  style.id = 'download-sku-styles'
  style.textContent = `
    .download-sku-overlay{position:fixed;inset:0;z-index:9999;display:flex;align-items:center;justify-content:center;background:rgba(15,23,42,.46);padding:18px}
    .download-sku-dialog{width:min(500px,100%);max-height:92vh;display:flex;flex-direction:column;gap:14px;overflow-y:auto;border-radius:16px;background:#fff;padding:18px;box-shadow:0 24px 80px rgba(15,23,42,.24);font-family:inherit}
    .download-sku-head{display:flex;align-items:center;justify-content:space-between;gap:12px}
    .download-sku-head strong{font-size:16px;color:#111827}
    .download-sku-close{border:0;background:#f3f4f6;border-radius:999px;width:30px;height:30px;font-size:20px;line-height:1;cursor:pointer}
    .download-sku-field{display:flex;flex-direction:column;gap:7px;font-size:13px;font-weight:600;color:#374151}
    .download-sku-field input,.download-brand-select{height:42px;width:100%;border:1px solid #d1d5db;border-radius:11px;padding:0 13px;font-size:14px;line-height:42px;outline:none;background:#fff;color:#111827;box-shadow:0 1px 2px rgba(15,23,42,.04);transition:border-color .15s ease,box-shadow .15s ease,background-color .15s ease}
    .download-sku-field input:focus,.download-brand-select:focus{border-color:#111827;box-shadow:0 0 0 3px rgba(17,24,39,.08),0 1px 2px rgba(15,23,42,.05)}
    .download-brand-select-wrap{position:relative;display:block}
    .download-brand-select{appearance:none;-webkit-appearance:none;padding-right:42px;cursor:pointer}
    .download-brand-select-wrap::after{content:"";position:absolute;right:15px;top:50%;width:8px;height:8px;border-right:1.8px solid #6b7280;border-bottom:1.8px solid #6b7280;transform:translateY(-65%) rotate(45deg);pointer-events:none}
    .download-brand-select:hover,.download-sku-field input:hover{border-color:#9ca3af;background:#fff}
    .download-sku-field input[readonly]{background:#f9fafb;color:#4b5563}
    .download-synthetic-field{min-width:0;margin:0;border:0;padding:0}
    .download-synthetic-field legend{display:flex;align-items:center;gap:7px;padding:0;font-size:13px;font-weight:700;color:#374151}
    .download-synthetic-field legend span{border-radius:4px;background:#fef2f2;padding:2px 5px;font-size:10px;font-weight:700;line-height:1.2;color:#b91c1c}
    .download-synthetic-field>p{margin:5px 0 9px;font-size:12px;line-height:1.45;color:#6b7280}
    .download-synthetic-options{display:grid;grid-template-columns:1fr 1fr;gap:9px}
    .download-synthetic-options label{min-width:0;cursor:pointer}
    .download-synthetic-options input{position:absolute;width:1px;height:1px;overflow:hidden;opacity:0;pointer-events:none}
    .download-synthetic-option{display:flex;min-height:62px;flex-direction:column;justify-content:center;border:1px solid #d1d5db;border-radius:8px;background:#fff;padding:10px 12px;transition:border-color .15s ease,box-shadow .15s ease,background-color .15s ease}
    .download-synthetic-option strong{font-size:13px;line-height:1.35;color:#1f2937}
    .download-synthetic-option small{margin-top:3px;font-size:11px;line-height:1.4;color:#6b7280}
    .download-synthetic-options label:hover .download-synthetic-option{border-color:#9ca3af;background:#f9fafb}
    .download-synthetic-options input:checked+.download-synthetic-option{border-color:#111827;background:#f9fafb;box-shadow:0 0 0 2px rgba(17,24,39,.10)}
    .download-synthetic-options input:checked+.download-synthetic-option strong{color:#111827}
    .download-synthetic-options input:focus-visible+.download-synthetic-option{outline:2px solid #111827;outline-offset:2px}
    .download-synthetic-required{margin-top:6px;font-size:11px;line-height:1.4;color:#b91c1c}
    .download-synthetic-required.is-complete{visibility:hidden}
    .download-sku-results{min-height:120px;max-height:260px;overflow:auto;border:1px solid #e5e7eb;border-radius:12px;padding:6px;background:#f9fafb}
    .download-sku-option{width:100%;border:0;background:transparent;border-radius:9px;padding:9px 10px;text-align:left;cursor:pointer}
    .download-sku-option:hover,.download-sku-option.is-active{background:#fff;box-shadow:0 1px 4px rgba(15,23,42,.08)}
    .download-sku-option strong{display:block;font-size:13px;color:#111827}
    .download-sku-option span{display:block;margin-top:3px;font-size:12px;color:#6b7280;line-height:1.35}
    .download-sku-empty{padding:18px 10px;text-align:center;font-size:13px;color:#6b7280}
    .download-sku-actions{display:flex;justify-content:flex-end;gap:10px}
    .download-sku-actions button{height:38px;border-radius:10px;padding:0 16px;font-size:14px;font-weight:700;cursor:pointer}
    .download-sku-cancel{border:1px solid #d1d5db;background:#fff;color:#374151}
    .download-sku-confirm{border:1px solid #111827;background:#111827;color:#fff}
    .download-sku-confirm:disabled{border-color:#d1d5db;background:#d1d5db;cursor:not-allowed}
    @media (max-width:480px){.download-synthetic-options{grid-template-columns:1fr}.download-synthetic-option{min-height:56px}}
  `
  document.head.appendChild(style)
}

service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

service.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      const url = error.config?.url || ''
      // /logout 和 /heartbeat 的 401 静默处理，避免递归触发 logout
      const isSilent = url.includes('/logout') || url.includes('/heartbeat')
      if (!isSilent) {
        // 延迟导入避免循环依赖
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        // 更新响应式状态并停止心跳/空闲计时器
        await authStore.logout()
        // 如果当前页面需要鉴权，跳回首页
        if (router.currentRoute.value.meta?.requiresAuth) {
          router.push({ name: 'home' })
        }
      }
    }
    return Promise.reject(error)
  }
)

export default service
