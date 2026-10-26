import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveReferenceAspectRatio } from './referenceAspectRatio.js'

const supportedRatios = ['1:1', '4:3', '3:4', '16:9', '9:16']

function mockImage(t, dimensions) {
  const loaded = []
  const previousImage = globalThis.Image
  t.after(() => { globalThis.Image = previousImage })
  globalThis.Image = class {
    set src(source) {
      loaded.push(source)
      const size = dimensions[source]
      if (!size) return this.onerror()
      this.naturalWidth = size[0]
      this.naturalHeight = size[1]
      this.onload()
    }
  }
  return loaded
}

test('多个参考图只读取第一张，移除首图后使用新的第一张', async t => {
  const loaded = mockImage(t, { landscape: [1600, 900], portrait: [900, 1600] })
  const refs = [{ url: 'landscape' }, { url: 'portrait' }]
  assert.equal(await resolveReferenceAspectRatio(refs, supportedRatios), '16:9')
  assert.deepEqual(loaded, ['landscape'])
  refs.shift()
  assert.equal(await resolveReferenceAspectRatio(refs, supportedRatios), '9:16')
})

test('非标准尺寸按当前模型支持的比例匹配，横竖规则对称', async t => {
  mockImage(t, { landscape: [1500, 1000], portrait: [1000, 1500] })
  assert.equal(await resolveReferenceAspectRatio([{ url: 'landscape' }], supportedRatios), '4:3')
  assert.equal(await resolveReferenceAspectRatio([{ url: 'portrait' }], supportedRatios), '3:4')
  assert.equal(await resolveReferenceAspectRatio([{ url: 'landscape' }], ['1:1', '3:2']), '3:2')
})

test('优先使用当前编辑后的图片数据，不使用旧的远程地址', async t => {
  const loaded = mockImage(t, { 'data:image/png;base64,edited': [1200, 1200] })
  assert.equal(await resolveReferenceAspectRatio([
    { data: 'edited', mimeType: 'image/png', url: 'old', remoteUrl: 'old-remote' },
  ], supportedRatios), '1:1')
  assert.deepEqual(loaded, ['data:image/png;base64,edited'])
})

test('无参考图或首图读取失败时报告错误，不改用第二张', async t => {
  const loaded = mockImage(t, { second: [1000, 1000], empty: [0, 0] })
  await assert.rejects(resolveReferenceAspectRatio([], supportedRatios), /请先添加参考图/)
  await assert.rejects(resolveReferenceAspectRatio([{ url: 'broken' }, { url: 'second' }], supportedRatios), /无法读取第一张/)
  assert.deepEqual(loaded, ['broken'])
  await assert.rejects(resolveReferenceAspectRatio([{ url: 'empty' }], supportedRatios), /无法读取第一张/)
})
