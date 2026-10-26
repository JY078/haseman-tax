import { defineStore } from 'pinia';
import { ref } from 'vue';
import request from '../utils/request';
import { Message } from '@arco-design/web-vue';

export const useUserStore = defineStore('user', () => {
  const token    = ref(localStorage.getItem('token') || '');
  const userInfo = ref(null);

  const login = async (username, password) => {
    try {
      const response = await request.post('/login', {
        username,
        password,
        client_type: 'admin', // admin bypasses device-limit check
      });
      token.value    = response.data.token;
      userInfo.value = response.data.user;
      localStorage.setItem('token', token.value);
      Message.success('登录成功');
      return true;
    } catch {
      return false;
    }
  };

  const fetchUserInfo = async () => {
    try {
      const response = await request.get('/me');
      userInfo.value = response.data;
      return true;
    } catch {
      return false;
    }
  };

  const logout = async () => {
    try {
      await request.post('/logout');
    } catch {
      // Ignore errors (token may already be invalid)
    }
    token.value    = '';
    userInfo.value = null;
    localStorage.removeItem('token');
  };

  const isAdmin = () => {
    return (userInfo.value?.roles || []).some(r => r.name === 'admin');
  };

  return { token, userInfo, isAdmin, login, logout, fetchUserInfo };
});
