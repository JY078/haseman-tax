<template>
  <a-layout class="app-layout">
    <a-layout-sider
      v-model:collapsed="collapsed"
      breakpoint="xl"
      :width="240"
      :collapsed-width="64"
      class="app-sider"
    >
      <div class="logo">
        <div class="logo-icon">H</div>
        <span v-if="!collapsed" class="logo-text">{{ siteName || 'AD Image' }}</span>
      </div>

      <a-menu
        v-model:selected-keys="selectedKeys"
        v-model:open-keys="openKeys"
        :collapsed="collapsed"
        :style="{ width: '100%' }"
        class="app-menu"
        @menu-item-click="handleMenuItemClick"
      >
        <template v-for="menu in menus" :key="menu.id">
          <!-- 有子菜单 → a-sub-menu -->
          <a-sub-menu
            v-if="menu.children && menu.children.length > 0"
            :key="String(menu.id)"
          >
            <template #icon>
              <component :is="getIconComponent(menu.icon)" />
            </template>
            <template #title>{{ menu.title }}</template>
            <a-menu-item
              v-for="child in menu.children"
              :key="child.path"
            >
              <template #icon>
                <component :is="getIconComponent(child.icon)" />
              </template>
              {{ child.title }}
            </a-menu-item>
          </a-sub-menu>

          <!-- 无子菜单 → 普通菜单项 -->
          <a-menu-item
            v-else
            :key="menu.path || String(menu.id)"
          >
            <template #icon>
              <component :is="getIconComponent(menu.icon)" />
            </template>
            {{ menu.title }}
          </a-menu-item>
        </template>
      </a-menu>

      <div class="sider-footer">
        <a-dropdown trigger="click" position="tr">
          <div class="user-section">
            <a-avatar :size="32" class="user-avatar">
              {{ userStore.userInfo?.username?.charAt(0)?.toUpperCase() }}
            </a-avatar>
            <div v-if="!collapsed" class="user-info">
              <span class="user-name">{{ userStore.userInfo?.username }}</span>
              <span class="user-role">管理员</span>
            </div>
          </div>
          <template #content>
            <a-doption @click="openChangePassword">
              <template #icon><icon-lock /></template>
              修改密码
            </a-doption>
            <a-doption @click="handleLogout">
              <template #icon><icon-export /></template>
              退出登录
            </a-doption>
          </template>
        </a-dropdown>

        <div class="collapse-btn" @click="toggleCollapse">
          <icon-left v-if="!collapsed" />
          <icon-right v-else />
        </div>
      </div>
    </a-layout-sider>

    <!-- 修改密码弹窗 -->
    <a-modal
      v-model:visible="pwdModalVisible"
      title="修改密码"
      :mask-closable="false"
      @ok="submitChangePassword"
      @cancel="resetPwdForm"
      ok-text="确认修改"
      cancel-text="取消"
      :ok-loading="pwdLoading"
    >
      <a-form :model="pwdForm" layout="vertical" ref="pwdFormRef">
        <a-form-item
          field="oldPassword"
          label="当前密码"
          :rules="[{ required: true, message: '请输入当前密码' }]"
        >
          <a-input-password
            v-model="pwdForm.oldPassword"
            placeholder="请输入当前密码"
            allow-clear
          />
        </a-form-item>
        <a-form-item
          field="newPassword"
          label="新密码"
          :rules="[
            { required: true, message: '请输入新密码' },
            { minLength: 6, message: '密码不能少于 6 位' },
          ]"
        >
          <a-input-password
            v-model="pwdForm.newPassword"
            placeholder="请输入新密码（至少 6 位）"
            allow-clear
          />
        </a-form-item>
        <a-form-item
          field="confirmPassword"
          label="确认新密码"
          :rules="[
            { required: true, message: '请再次输入新密码' },
            { validator: validateConfirmPwd, message: '两次密码输入不一致' },
          ]"
        >
          <a-input-password
            v-model="pwdForm.confirmPassword"
            placeholder="请再次输入新密码"
            allow-clear
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-layout class="app-main">
      <a-layout-header class="app-header">
        <div class="header-left">
          <a-breadcrumb>
            <a-breadcrumb-item>{{ getCurrentMenuTitle() }}</a-breadcrumb-item>
          </a-breadcrumb>
        </div>
        <div class="header-right"></div>
      </a-layout-header>

      <a-layout-content class="app-content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useUserStore } from '../stores/user';
import request from '../utils/request';
import { Message } from '@arco-design/web-vue';
import {
  IconDashboard,
  IconUser,
  IconSettings,
  IconHistory,
  IconSafe,
  IconRobot,
  IconImage,
  IconFile,
  IconBook,
  IconPalette,
  IconMenuUnfold,
  IconExport,
  IconLeft,
  IconRight,
  IconLock,
  IconBarChart,
} from '@arco-design/web-vue/es/icon';

const router = useRouter();
const route = useRoute();
const userStore = useUserStore();

const menus = ref([]);
const siteName = ref('');
const selectedKeys = ref([route.path]);
const openKeys = ref([]);
const collapsed = ref(false);

// 修改密码弹窗状态
const pwdModalVisible = ref(false);
const pwdLoading = ref(false);
const pwdFormRef = ref(null);
const pwdForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
});

const toggleCollapse = () => {
  collapsed.value = !collapsed.value;
};

const iconMap = {
  'icon-dashboard': IconDashboard,
  'icon-user': IconUser,
  'icon-settings': IconSettings,
  'icon-history': IconHistory,
  'icon-safe': IconSafe,
  'icon-robot': IconRobot,
  'icon-image': IconImage,
  'icon-file': IconFile,
  'icon-book': IconBook,
  'icon-menu': IconMenuUnfold,
  'icon-palette': IconPalette,
  'icon-bar-chart': IconBarChart,
};

const getIconComponent = (iconName) => {
  return iconMap[iconName] || IconDashboard;
};

/** 计算当前路由所在父菜单的 key（String 类型 ID），用于自动展开 */
const computeOpenKeys = (path) => {
  const keys = [];
  for (const menu of menus.value) {
    if (menu.children && menu.children.some(c => c.path === path)) {
      keys.push(String(menu.id));
    }
  }
  return keys;
};

/** 遍历两级树查找当前路由对应的菜单标题 */
const getCurrentMenuTitle = () => {
  for (const menu of menus.value) {
    if (menu.path === route.path) return menu.title;
    if (menu.children) {
      const child = menu.children.find(c => c.path === route.path);
      if (child) return child.title;
    }
  }
  return '首页';
};

const fetchMenus = async () => {
  try {
    // 税务副本只展示用户管理，原系统的其他后台菜单不加载。
    menus.value = [{ id: 'users', title: '用户管理', path: '/users', icon: 'icon-user' }];
    // 加载完成后立即计算需要展开的父菜单
    openKeys.value = computeOpenKeys(route.path);
  } catch (error) {
    console.error('Failed to fetch menus', error);
  }
};

// 原系统后台设置页属于非税务功能，税务副本不加载全局后台设置。

const handleMenuItemClick = (key) => {
  // 路径为空时为一级分类菜单，不执行跳转
  if (!key) return;
  router.push(key);
};

const handleLogout = () => {
  userStore.logout();
  router.push({ name: 'login' });
};

// ── 心跳轮询 ──
// 每 30 秒主动检查登录有效性并上报在线心跳。
// AuthMiddleware 会同时验证 JWT 签名和 DB session 是否存在：
//   - session 失效（token 过期 / 被踢下线）→ 返回 401
//   - 响应拦截器捕获 401 → 自动调用 logout + 跳转登录页
const HEARTBEAT_INTERVAL = 30 * 1000;
let heartbeatTimer = null;

const sendHeartbeat = async () => {
  if (!userStore.token) return;
  try {
    await request.post('/heartbeat');
  } catch {
    // 401 已由响应拦截器统一处理（logout + redirect）
  }
};

const checkOnVisible = () => {
  if (document.visibilityState === 'visible') sendHeartbeat();
};

const startHeartbeat = () => {
  sendHeartbeat();
  heartbeatTimer = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL);
  document.addEventListener('visibilitychange', checkOnVisible);
};

const stopHeartbeat = () => {
  document.removeEventListener('visibilitychange', checkOnVisible);
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
};

const openChangePassword = () => {
  resetPwdForm();
  pwdModalVisible.value = true;
};

const resetPwdForm = () => {
  pwdForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' };
  pwdFormRef.value?.clearValidate();
};

const validateConfirmPwd = (value, cb) => {
  if (value !== pwdForm.value.newPassword) {
    cb('两次密码输入不一致');
  } else {
    cb();
  }
};

const submitChangePassword = async () => {
  try {
    await pwdFormRef.value?.validate();
  } catch {
    return;
  }

  pwdLoading.value = true;
  try {
    await request.put('/me/password', {
      old_password: pwdForm.value.oldPassword,
      new_password: pwdForm.value.newPassword,
    });
    Message.success('密码修改成功，请重新登录');
    pwdModalVisible.value = false;
    setTimeout(() => {
      userStore.logout();
      router.push({ name: 'login' });
    }, 1200);
  } catch (error) {
    const msg = error?.response?.data?.error || '密码修改失败，请检查当前密码是否正确';
    Message.error(msg);
  } finally {
    pwdLoading.value = false;
  }
};

watch(
  () => route.path,
  (newPath) => {
    selectedKeys.value = [newPath];
    // 路由变化时同步展开包含该路由的父菜单
    const parentKeys = computeOpenKeys(newPath);
    if (parentKeys.length > 0) {
      openKeys.value = [...new Set([...openKeys.value, ...parentKeys])];
    }
  }
);

onMounted(() => {
  fetchMenus();
  startHeartbeat();
});

onUnmounted(() => {
  stopHeartbeat();
});
</script>

<style scoped>
.app-layout {
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: var(--bg-primary);
}

.app-sider {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary) !important;
  border-right: 1px solid var(--border-color) !important;
}

.app-sider :deep(.arco-layout-sider-children) {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 16px;
  margin-bottom: 8px;
  transition: padding 0.2s;
}

.app-sider.arco-layout-sider-collapsed .logo {
  justify-content: center;
  padding: 20px 0;
}

.logo-icon {
  width: 36px;
  height: 36px;
  background: var(--accent-color);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
  font-size: 18px;
  flex-shrink: 0;
}

.logo-text {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  letter-spacing: -0.3px;
}

.app-menu {
  flex: 1;
  overflow-y: auto;
  border: none !important;
  background: transparent !important;
  padding: 0 12px;
}

.app-menu :deep(.arco-menu-item) {
  height: 40px !important;
  line-height: 40px !important;
  margin: 2px 0 !important;
  padding: 0 !important;
  border-radius: 0 !important;
}

.app-menu :deep(.arco-menu-item-inner) {
  border-radius: var(--radius-sm) !important;
  padding: 0 12px !important;
  width: 100% !important;
  height: 100% !important;
  display: flex !important;
  align-items: center !important;
  color: var(--text-secondary) !important;
  font-size: 14px !important;
  font-weight: 500 !important;
  transition: all 0.15s ease !important;
}

.app-menu :deep(.arco-menu-item:hover .arco-menu-item-inner) {
  background: var(--bg-tertiary) !important;
  color: var(--text-primary) !important;
}

.app-menu :deep(.arco-menu-item.arco-menu-selected .arco-menu-item-inner) {
  background: var(--accent-light) !important;
  color: var(--accent-color) !important;
  border-left: 3px solid var(--accent-color) !important;
  padding-left: 9px !important;
}

.app-menu :deep(.arco-menu-item.arco-menu-selected .arco-menu-icon) {
  color: var(--accent-color) !important;
}

.app-menu :deep(.arco-menu-item.arco-menu-selected:hover .arco-menu-item-inner) {
  background: var(--accent-light) !important;
}

.app-menu :deep(.arco-sub-menu) {
  width: 100% !important;
}

.app-menu :deep(.arco-sub-menu-title) {
  height: 40px !important;
  line-height: 40px !important;
  margin: 2px 0 !important;
  padding: 0 12px !important;
  border-radius: var(--radius-sm) !important;
  display: flex !important;
  align-items: center !important;
  color: var(--text-secondary) !important;
  font-size: 14px !important;
  font-weight: 500 !important;
  transition: all 0.15s ease !important;
}

.app-menu :deep(.arco-sub-menu-title:hover) {
  background: var(--bg-tertiary) !important;
  color: var(--text-primary) !important;
}

.app-menu :deep(.arco-sub-menu-title .arco-menu-icon) {
  color: var(--text-secondary) !important;
}

.app-menu :deep(.arco-sub-menu.arco-menu-opened .arco-sub-menu-title) {
  background: var(--accent-light) !important;
  color: var(--accent-color) !important;
}

.app-menu :deep(.arco-sub-menu.arco-menu-opened .arco-sub-menu-title .arco-menu-icon) {
  color: var(--accent-color) !important;
}

/* 折叠状态下菜单样式 */
.app-sider.arco-layout-sider-collapsed .app-menu {
  padding: 0 8px;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-menu-item-inner) {
  padding: 0 !important;
  justify-content: center !important;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-menu-icon) {
  margin-right: 0 !important;
  font-size: 18px !important;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-sub-menu) {
  width: 100% !important;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-sub-menu-title) {
  padding: 0 !important;
  justify-content: center !important;
  width: 100% !important;
  margin-left: 0 !important;
  margin-right: 0 !important;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-sub-menu-title .arco-menu-icon) {
  margin-right: 0 !important;
}

.app-sider.arco-layout-sider-collapsed .app-menu :deep(.arco-menu-item) {
  width: 100% !important;
  margin-left: 0 !important;
  margin-right: 0 !important;
}

.sider-footer {
  padding: 8px 12px;
  border-top: 1px solid var(--border-color);
  margin-top: auto;
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 4px;
}

.app-sider.arco-layout-sider-collapsed .sider-footer {
  padding: 8px;
  justify-content: center;
  flex-direction: column;
}

.app-sider.arco-layout-sider-collapsed .user-section {
  justify-content: center;
  padding: 8px;
}

.user-section {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.15s;
  flex: 1;
  min-width: 0;
}

.user-section:hover {
  background: var(--bg-tertiary);
}

.user-avatar {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-weight: 600;
  flex-shrink: 0;
}

.user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.user-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}

.user-role {
  font-size: 11px;
  color: var(--text-tertiary);
  line-height: 1.3;
}

.user-more {
  color: var(--text-tertiary);
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.user-section:hover .user-more {
  opacity: 1;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-tertiary);
  transition: all 0.15s;
}

.collapse-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.app-main {
  flex: 1;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-header {
  height: 56px;
  line-height: 56px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-secondary) !important;
  border-bottom: 1px solid var(--border-color) !important;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.task-indicator {
  display: flex;
  align-items: center;
  padding: 5px 12px;
  border-radius: 20px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-1);
  color: var(--color-text-2);
  transition: all 0.2s;
  white-space: nowrap;
}

.task-indicator:hover {
  background: var(--color-fill-2);
  border-color: rgb(var(--primary-5));
  color: rgb(var(--primary-6));
}

.task-indicator.is-running {
  border-color: rgb(var(--primary-5));
  background: var(--color-primary-light-1);
  color: rgb(var(--primary-6));
  animation: pulse-border 2s ease-in-out infinite;
}

@keyframes pulse-border {
  0%, 100% { box-shadow: 0 0 0 0 rgba(var(--primary-6), 0.15); }
  50%       { box-shadow: 0 0 0 4px rgba(var(--primary-6), 0); }
}

.task-text {
  font-weight: 500;
}

.header-left :deep(.arco-breadcrumb) {
  font-size: 14px;
  color: var(--text-secondary);
}

.header-left :deep(.arco-breadcrumb-item:last-child) {
  color: var(--text-primary);
  font-weight: 500;
}

.app-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.app-content :deep(.arco-layout-content) {
  background: transparent;
}
</style>
