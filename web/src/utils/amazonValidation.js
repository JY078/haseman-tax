// Parse an Amazon quarter validation message produced by the backend:
// "2026-01-01 至 2026-03-31 税费校验不一致：明细 123.45，PDF 122.00，差额 1.45 USD"
export function parseAmazonValidationMessage(message) {
  const match = String(message || '').match(/^(?:(.+?) 至 (.+?) )?(税费|仓储费|配送费)校验不一致：明细 ([^，]+)，PDF ([^，]+)，差额 (\S+)\s*([A-Za-z]{3})?$/)
  if (!match) return null
  const [, start, end, metric, detail, pdf, difference, currency] = match
  return {
    range: start && end ? `${start} ~ ${end}` : '',
    metric,
    detail,
    pdf,
    difference,
    currency: currency || '',
  }
}

export function parseAmazonValidationMessages(messages) {
  return (messages || [])
    .map(message => ({ message, issue: parseAmazonValidationMessage(message) }))
}
