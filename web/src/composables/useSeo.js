/** 网站根域名，从环境变量读取，生产环境需配置 VITE_SITE_URL */
const SITE_URL = import.meta.env.VITE_SITE_URL || 'https://home.radium.net.cn'
const SITE_NAME = ''
const DEFAULT_OG_IMAGE = `${SITE_URL}/og-image.png`

/**
 * 更新或创建指定 name 的 <meta> 标签
 * @param {string} name - meta name 属性
 * @param {string} content - meta content 值
 */
function setMetaName(name, content) {
  if (!content) return
  let el = document.querySelector(`meta[name="${name}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', name)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

/**
 * 更新或创建指定 property 的 <meta> Open Graph 标签
 * @param {string} property - meta property 属性
 * @param {string} content - meta content 值
 */
function setMetaProperty(property, content) {
  if (!content) return
  let el = document.querySelector(`meta[property="${property}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('property', property)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

/**
 * 更新或创建 canonical 链接标签
 * @param {string} url - 规范 URL
 */
function setCanonical(url) {
  let el = document.querySelector('link[rel="canonical"]')
  if (!el) {
    el = document.createElement('link')
    el.setAttribute('rel', 'canonical')
    document.head.appendChild(el)
  }
  el.setAttribute('href', url)
}

/**
 * SEO composable，在每个页面/路由中调用以更新页面 meta 信息
 *
 * @param {object} options
 * @param {string} options.title          页面标题（不含站点名后缀）
 * @param {string} options.description    页面描述（建议 80-150 字）
 * @param {string} [options.keywords]     关键词（逗号分隔，可选）
 * @param {string} [options.path]         页面路径，如 '/features'，用于生成 canonical
 * @param {string} [options.ogImage]      OG 分享图，默认使用全站默认图
 * @param {string} [options.ogType]       OG 类型，默认 'website'
 */
export function useSeo({ title, description, keywords, path = '', ogImage, ogType = 'website' } = {}) {
  const fullTitle = title
    ? (SITE_NAME ? `${title} - ${SITE_NAME}` : title)
    : `${SITE_NAME} - AI智能图像生成平台 | 让创意变为现实`
  const canonicalUrl = `${SITE_URL}${path}`
  const image = ogImage || DEFAULT_OG_IMAGE

  /** 更新 document.title */
  document.title = fullTitle

  /** 更新基础 meta */
  setMetaName('description', description)
  if (keywords) {
    setMetaName('keywords', keywords)
  }

  /** 更新 canonical */
  setCanonical(canonicalUrl)

  /** 更新 Open Graph */
  setMetaProperty('og:title', fullTitle)
  setMetaProperty('og:description', description)
  setMetaProperty('og:url', canonicalUrl)
  setMetaProperty('og:type', ogType)
  setMetaProperty('og:image', image)

  /** 更新 Twitter Card */
  setMetaName('twitter:title', fullTitle)
  setMetaName('twitter:description', description)
  setMetaName('twitter:image', image)
}
