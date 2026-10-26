import axios from 'axios';
import { Message } from '@arco-design/web-vue';
import router from '../router';
import { useUserStore } from '../stores/user';

const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 600000, // 10 minutes，兼容图片生成等耗时操作
});

/**
 * 将后端返回的相对图片路径转换为可访问的绝对 URL。
 * 生产环境后端返回 /uploads/... 形式的相对路径，需拼接后端 origin。
 */
export function resolveImageUrl(url) {
  if (!url || url.startsWith('data:') || url.startsWith('http://') || url.startsWith('https://')) {
    return url;
  }
  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';
  if (apiBase.startsWith('http')) {
    return new URL(apiBase).origin + url;
  }
  return url;
}

// Request interceptor
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      // Add token to headers
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor
service.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response) {
      const { status } = error.response;
      const { config } = error;
      
      if (status === 401) {
        // Check if it's a login request failure
        if (config && config.url && config.url.includes('/login')) {
             Message.error('用户名或密码错误');
        } else if (config && config.url && config.url.includes('/logout')) {
             // /logout 返回 401 说明 token 已失效，静默忽略，避免递归触发 logout
        } else {
             // Token expired or invalid for other requests
             Message.error('会话已过期，请重新登录');
             const userStore = useUserStore();
             userStore.logout();
             router.push({ name: 'login' });
        }
      } else {
        Message.error(error.response.data?.error || '请求失败');
      }
    } else {
      Message.error('网络错误，请检查您的连接');
      // If there's a token but network error, likely backend is down - redirect to login
      const userStore = useUserStore();
      const isLoginRequest = error.config?.url?.includes('/login');
      if (userStore.token && !isLoginRequest) {
        userStore.logout();
        router.push({ name: 'login' });
      }
    }
    return Promise.reject(error);
  }
);

export default service;
