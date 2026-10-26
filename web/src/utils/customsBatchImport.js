export async function importCustomsBatch(files, uploadFile, onUpdate) {
  const results = []
  for (const [index, file] of files.entries()) {
    let result
    if (!/\.pdf$/i.test(file.name) || !file.size || file.size > 20 * 1024 * 1024) {
      result = { status: 'failed', message: '请选择非空且不超过 20 MB 的 PDF 文件' }
    } else {
      onUpdate(index, { status: 'uploading', message: '正在识别并保存…' })
      try {
        const data = await uploadFile(file)
        const warnings = Array.isArray(data.warnings) ? data.warnings : []
        result = { status: data.duplicate ? 'duplicate' : 'success', message: [data.message || (data.duplicate ? '重复单据，已跳过' : '导入成功'), ...warnings].join('；') }
      } catch (error) {
        result = { status: 'failed', message: error.response?.data?.error || '上传或识别失败；如请求超时，请刷新核对是否已保存后再重试' }
      }
    }
    results.push(result)
    onUpdate(index, result)
  }
  return results
}
