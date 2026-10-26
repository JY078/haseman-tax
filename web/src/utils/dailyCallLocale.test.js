import test from 'node:test'
import assert from 'node:assert/strict'
import { dailyCallLocale, dailyCallText, dailyCallError } from './dailyCallLocale.js'

test('daily call language follows department, independently of admin permissions', () => {
  assert.equal(dailyCallLocale({ department: '海外客服部' }), 'en')
  assert.equal(dailyCallLocale({ department: '客服', sub_department: '海外客服' }), 'en')
  assert.equal(dailyCallLocale({ department: '海外客服', sub_department: '客服' }), 'zh')
  assert.equal(dailyCallLocale({ department: '海外客服', roles: [{ name: 'admin' }] }), 'en')
  assert.equal(dailyCallLocale({ roles: [{ name: 'admin' }] }), 'zh')
  assert.equal(dailyCallLocale({ department: '客服运营' }), 'zh')
  assert.equal(dailyCallLocale(null), 'zh')
})

test('both languages localize the page name and quality fields', () => {
  assert.equal(dailyCallText('zh', 'Daily Call Registration'), '每日来电登记')
  assert.equal(dailyCallText('en', '每日来电登记'), 'Daily Call Registration')
  assert.equal(dailyCallText('en', '质检结果'), 'Quality result')
  assert.equal(dailyCallText('en', '理由'), 'Reason')
})

test('API errors cannot leak untranslated Chinese into the English page', () => {
  const error = message => ({ response: { data: { error: message } } })
  assert.equal(dailyCallError('en', error('当前部门无权修改这部分信息'), '保存失败，请重试'), 'Your department cannot edit this section.')
  assert.equal(dailyCallError('en', error('新的未知错误'), '保存失败，请重试'), 'Unable to save. Please try again.')
  assert.equal(dailyCallError('zh', error('Network error'), '保存失败，请重试'), '保存失败，请重试')
})
