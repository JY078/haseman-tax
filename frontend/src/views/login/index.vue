<template>
  <div class="login-container">
    <div class="login-content">
      <div class="login-header">
        <h2 class="login-title">管理员登录</h2>
        <p class="login-subtitle">欢迎回来！请登录您的账号。</p>
      </div>
      <a-form :model="form" @submit="handleSubmit" layout="vertical">
        <a-form-item field="username" label="用户名" :rules="[{ required: true, message: '请输入用户名' }]">
          <a-input v-model="form.username" placeholder="请输入您的用户名" allow-clear>
            <template #prefix>
              <icon-user />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item field="password" label="密码" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model="form.password" placeholder="请输入您的密码">
            <template #prefix>
              <icon-lock />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" long :loading="loading">
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useUserStore } from '../../stores/user';
import { IconUser, IconLock } from '@arco-design/web-vue/es/icon';
import { Message } from '@arco-design/web-vue';

const router = useRouter();
const userStore = useUserStore();
const loading = ref(false);

const form = reactive({
  username: '',
  password: '',
});

const handleSubmit = async ({ errors, values }) => {
  if (errors) return;
  
  loading.value = true;
  try {
    const success = await userStore.login(values.username, values.password);
    if (success) {
      router.push({ name: 'dashboard' });
    }
  } catch (err) {
    // Error handling is done in store
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: var(--bg-primary);
}

.login-content {
  width: 100%;
  max-width: 380px;
  padding: 40px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-md);
}

.login-header {
  margin-bottom: 32px;
  text-align: center;
}

.login-title {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
}

.login-subtitle {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
