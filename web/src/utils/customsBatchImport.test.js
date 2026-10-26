import test from 'node:test'
import assert from 'node:assert/strict'
import { importCustomsBatch } from './customsBatchImport.js'

test('批量逐份处理，重复跳过、单份失败后继续，保留识别提示', async () => {
  const files = ['first.pdf', 'duplicate.pdf', 'broken.pdf', 'last.PDF'].map(name => ({ name, size: 100 }))
  const calls = [], updates = []
  let active = 0, maxActive = 0
  const results = await importCustomsBatch(files, async file => {
    calls.push(file.name); active++; maxActive = Math.max(maxActive, active)
    await new Promise(resolve => setTimeout(resolve, 1))
    active--
    if (file.name === 'broken.pdf') throw { response: { data: { error: '未识别到海关编号' } } }
    if (file.name === 'duplicate.pdf') return { duplicate: true, message: '海关编号已导入' }
    return { message: '已保存', warnings: file.name === 'last.PDF' ? ['请核对 OCR 金额'] : [] }
  }, (index, result) => updates.push({ index, ...result }))
  assert.equal(maxActive, 1)
  assert.deepEqual(calls, files.map(file => file.name))
  assert.deepEqual(results.map(row => row.status), ['success', 'duplicate', 'failed', 'success'])
  assert.equal(results[2].message, '未识别到海关编号')
  assert.match(results[3].message, /OCR/)
  assert.equal(updates.filter(row => row.status === 'uploading').length, 4)
})

test('无效文件独立失败，不上传，边界大小和网络错误分别处理', async () => {
  const files = [{ name: 'not-pdf.txt', size: 10 }, { name: 'large.pdf', size: 20 * 1024 * 1024 + 1 }, { name: 'empty.pdf', size: 0 }, { name: 'valid.pdf', size: 20 * 1024 * 1024 }, { name: 'timeout.pdf', size: 10 }]
  const calls = []
  const results = await importCustomsBatch(files, async file => {
    calls.push(file.name)
    if (file.name === 'timeout.pdf') throw new Error('timeout')
    return {}
  }, () => {})
  assert.deepEqual(calls, ['valid.pdf', 'timeout.pdf'])
  assert.deepEqual(results.map(row => row.status), ['failed', 'failed', 'failed', 'success', 'failed'])
  assert.match(results[4].message, /核对是否已保存/)
  assert.deepEqual(await importCustomsBatch([], () => assert.fail(), () => assert.fail()), [])
})
