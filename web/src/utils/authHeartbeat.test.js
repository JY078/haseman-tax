import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import vm from 'node:vm'

function setup() {
  const saved = new Map(), intervals = new Map(), timeouts = new Map()
  const listeners = new Map(), activity = new Map(), calls = []
  let nextID = 0, failure = null
  let now = 1000000
  const context = vm.createContext({
    Date: { now: () => now },
    defineStore: (_, factory) => factory,
    ref: value => ({ value }),
    computed: fn => ({ get value() { return fn() } }),
    localStorage: { getItem: key => saved.get(key), setItem: (key, value) => saved.set(key, value), removeItem: key => saved.delete(key) },
    request: { post: async (path, data, config) => { calls.push({ path, config }); if (failure) throw failure } },
    setInterval: (fn, delay) => { assert.equal(delay, 30000); intervals.set(++nextID, fn); return nextID },
    clearInterval: id => intervals.delete(id),
    setTimeout: (fn, delay) => { assert.equal(delay, 1800000); timeouts.set(++nextID, fn); return nextID },
    clearTimeout: id => timeouts.delete(id),
    window: { location: {}, addEventListener: (event, fn) => activity.set(event, fn), removeEventListener: event => activity.delete(event) },
    document: { visibilityState: 'visible', addEventListener: (event, fn) => listeners.set(event, fn), removeEventListener: event => listeners.delete(event) },
  })
  const source = readFileSync(new URL('../stores/auth.js', import.meta.url), 'utf8').replace(/^import .*$/gm, '').replace('export const useAuthStore', 'const useAuthStore')
  const store = vm.runInContext(source + String.fromCharCode(10) + 'useAuthStore()', context)
  return { store, saved, intervals, timeouts, listeners, activity, calls, fail: error => { failure = error }, advance: ms => { now += ms } }
}

test('401清除长期登录并停止心跳', async () => {
  const s = setup()
  s.store.login({ id: 1, long_login: true }, 'token')
  await new Promise(setImmediate)
  assert.deepEqual(s.calls.map(c => c.path), ['/heartbeat'])
  assert.equal(s.timeouts.size, 0)
  s.fail({ response: { status: 401 } })
  await [...s.intervals.values()][0]()
  assert.equal(s.store.isLoggedIn.value, false)
  assert.equal(s.saved.size, 0)
  assert.equal(s.intervals.size, 0)
  assert.equal(s.listeners.size, 0)
  assert.equal(s.activity.size, 0)
})

test('网络失败保留登录，切回页面重新检查', async () => {
  const s = setup()
  s.store.login({ id: 1 }, 'token')
  await new Promise(setImmediate)
  s.fail(new Error('network'))
  await [...s.intervals.values()][0]()
  assert.equal(s.store.isLoggedIn.value, true)
  s.fail({ response: { status: 401 } })
  s.listeners.get('visibilitychange')()
  await new Promise(setImmediate)
  assert.equal(s.store.isLoggedIn.value, false)
})

test('心跳不延长30分钟计时，真实操作重置计时，超时注销会话', async () => {
  const s = setup()
  s.store.login({ id: 1 }, 'token')
  await new Promise(setImmediate)
  const initialIdle = [...s.timeouts.keys()][0]
  await [...s.intervals.values()][0]()
  assert.equal([...s.timeouts.keys()][0], initialIdle)
  s.activity.get('keydown')()
  assert.notEqual([...s.timeouts.keys()][0], initialIdle)
  await [...s.timeouts.values()][0]()
  assert.equal(s.store.isLoggedIn.value, false)
  assert.equal(s.intervals.size, 0)
  assert.equal(s.calls.find(c => c.path === '/logout').config.headers.Authorization, 'Bearer token')
})

test('休眠超过30分钟，恢复可见或首次操作均不能延长登录', async () => {
  for (const event of ['visible', 'activity']) {
    const s = setup()
    s.store.login({ id: 1 }, 'token')
    await new Promise(setImmediate)
    s.advance(31 * 60 * 1000)
    if (event === 'visible') s.listeners.get('visibilitychange')()
    else s.activity.get('mousemove')()
    await new Promise(setImmediate)
    assert.equal(s.store.isLoggedIn.value, false)
    assert.deepEqual(s.calls.map(c => c.path), ['/heartbeat', '/logout'])
  }
})
