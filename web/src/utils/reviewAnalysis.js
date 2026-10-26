export function reviewDateKey(value) {
  const match = /^(\d{4})[-/年](\d{1,2})[-/月](\d{1,2})(?:日)?(?:$|[T\s])/.exec(String(value || '').trim())
  if (!match) return ''
  const [, year, month, day] = match.map(Number)
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return ''
  return [year, String(month).padStart(2, '0'), String(day).padStart(2, '0')].join('-')
}

export function matchReviewProductImage(key, products) {
  const normalize = value => String(value || '').trim().toUpperCase()
  key = normalize(key)
  if (!key) return ''
  const matches = products.filter(product => normalize(product.asin) === key)
  const images = [...new Set(matches.map(product => String(product.image_url || '').trim()).filter(Boolean))]
  return images.length === 1 ? images[0] : ''
}

export function createReviewProductImageLoader(request) {
  const cache = new Map()
  return function load(key) {
    key = String(key || '').trim().toUpperCase()
    if (!key) return Promise.resolve('')
    if (!cache.has(key)) {
      const pending = (async () => {
        const products = []
        for (let page = 1; ; page++) {
          const response = await request.get('/product-catalog/spus', { params: { keyword: key, page, page_size: 100 } })
          const { items = [], total = 0 } = response.data || {}
          products.push(...items.flatMap(item => item.skus || []))
          if (!items.length || page * 100 >= total) break
        }
        return matchReviewProductImage(key, products)
      })().catch(error => { cache.delete(key); throw error })
      cache.set(key, pending)
    }
    return cache.get(key)
  }
}
