import test from 'node:test'
import assert from 'node:assert/strict'
import { userLandingRoute } from './userLandingRoute.js'

test('finance departments land on tax data before other department defaults', () => {
  for (const user of [
    { department: '财务部' },
    { department: '运营', sub_department: '财务核算' },
    { department: '财务部', sub_department: '客服' },
    { department: '财务部', roles: [{ name: 'admin' }] },
  ]) assert.equal(userLandingRoute(user), '/tax-data')
})

test('customer service departments land on call registration', () => {
  for (const user of [
    { department: '海外客服' }, { department: '客服部' },
    { department: '运营', sub_department: '海外客服部' },
    { department: '客服', roles: [{ name: 'admin' }] },
  ]) assert.equal(userLandingRoute(user), '/daily-call-registration')
})

test('other users retain the AI image generation landing page', () => {
  for (const user of [null, {}, { department: '运营' }, { department: '客服运营' }, { roles: [{ name: 'admin' }] }]) {
    assert.equal(userLandingRoute(user), '/generate')
  }
})


test('tax data is restricted to administrators or finance departments', async () => {
  const { isTaxDataUser } = await import('./userLandingRoute.js')
  assert.equal(isTaxDataUser(null), false)
  assert.equal(isTaxDataUser({ department: '数据部' }), false)
  assert.equal(isTaxDataUser({ department: '运营', sub_department: '财务核算' }), true)
  assert.equal(isTaxDataUser({ department: '财务部' }), true)
  assert.equal(isTaxDataUser({ roles: [{ name: 'admin' }] }), true)
  assert.equal(isTaxDataUser({ roles: ['admin'] }), true)
  assert.equal(isTaxDataUser({ department_name: '财务部' }), false)
})

test('operations tools are hidden from design and finance departments', async () => {
  const { canViewOperationsTools } = await import('./userLandingRoute.js')
  assert.equal(canViewOperationsTools({ department: '设计部' }), false)
  assert.equal(canViewOperationsTools({ department: '财务部' }), false)
  assert.equal(canViewOperationsTools({ department: '运营', sub_department: '财务核算' }), false)
  assert.equal(canViewOperationsTools({ department: '运营' }), true)
  assert.equal(canViewOperationsTools(null), true)
})

test('finance departments are identified for restricted navigation', async () => {
  const { isFinanceUser } = await import('./userLandingRoute.js')
  assert.equal(isFinanceUser({ department: '财务部' }), true)
  assert.equal(isFinanceUser({ department: '运营', sub_department: '财务核算' }), true)
  assert.equal(isFinanceUser({ department: '运营' }), false)
  assert.equal(isFinanceUser(null), false)
})

test('design departments are identified for restricted navigation', async () => {
  const { isDesignUser } = await import('./userLandingRoute.js')
  assert.equal(isDesignUser({ department: '设计部' }), true)
  assert.equal(isDesignUser({ department: '运营', sub_department: '设计组' }), true)
  assert.equal(isDesignUser({ department: '运营' }), false)
  assert.equal(isDesignUser(null), false)
})
