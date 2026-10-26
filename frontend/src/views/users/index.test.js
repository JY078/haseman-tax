import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import { parse } from '@vue/compiler-sfc';

test('用户列表自动刷新退出后的状态，并在离开页面时停止刷新', async () => {
  const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8');
  const script = parse(source).descriptor.scriptSetup.content.replace(/^import .*;$/gm, '');
  let mount, unmount, tick, visibilityListener, cleared = false;
  let online = true;
  const context = vm.createContext({
    ref: value => ({ value }),
    computed: fn => ({ get value() { return fn(); } }),
    onMounted: fn => { mount = fn; },
    onUnmounted: fn => { unmount = fn; },
    useUserStore: () => ({ userInfo: { id: 9 } }),
    request: { get: async url => ({ data: url === '/users' ? { data: [{ id: 1, online }], total: 1 } : [] }) },
    setInterval: (fn, ms) => { assert.equal(ms, 10000); tick = fn; return 1; },
    clearInterval: id => { assert.equal(id, 1); cleared = true; },
    document: {
      visibilityState: 'visible',
      addEventListener: (_, fn) => { visibilityListener = fn; },
      removeEventListener: (_, fn) => { assert.equal(fn, visibilityListener); visibilityListener = null; },
    },
  });
  vm.runInContext(script, context);
  mount();
  await new Promise(setImmediate);
  assert.equal(vm.runInContext('users.value[0].online', context), true);
  online = false;
  tick();
  await new Promise(setImmediate);
  assert.equal(vm.runInContext('users.value[0].online', context), false);
  online = true;
  visibilityListener();
  await new Promise(setImmediate);
  assert.equal(vm.runInContext('users.value[0].online', context), true);
  unmount();
  assert.equal(cleared, true);
  assert.equal(visibilityListener, null);
});
