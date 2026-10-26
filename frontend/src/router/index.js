import { createRouter, createWebHistory } from 'vue-router';
import { useUserStore } from '../stores/user';
import DefaultLayout from '../layout/default-layout.vue';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/login/index.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: DefaultLayout,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: 'users',
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('../views/users/index.vue'),
        },
        // 税务副本仅保留用户管理。原系统后台的其他页面已停用：
        // departments / settings / models / usage / image-stats / roles /
        // prompts / product-prompts / scene-styles / product-colors /
        // reference-galleries / task-queue / announcements / menus / feedbacks。
      ],
    },
    // 旧后台页面在税务副本中停用，统一回到用户管理。
    {
      path: '/:pathMatch(.*)*',
      redirect: '/users',
    },
  ],
});

router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore();
  
  // If the route explicitly requires NO auth (like login), allow access
  if (to.meta.requiresAuth === false) {
    // 已登录用户访问登录页时，直接进入用户管理。
    if (to.name === 'login' && userStore.token) {
          next({ name: 'users' });
      return;
    }
    next();
    return;
  }

  // For all other routes (default behavior), require authentication
  if (!userStore.token) {
    next({ name: 'login' });
  } else {
    // If we have a token but no user info (e.g. after page refresh), fetch it to validate session
    if (!userStore.userInfo) {
      try {
        const success = await userStore.fetchUserInfo();
        if (success) {
          next();
        } else {
          // fetchUserInfo failed (likely 401), store/interceptor handles logout
          next({ name: 'login' });
        }
      } catch (error) {
        next({ name: 'login' });
      }
    } else {
      // userInfo already in memory; session validity is maintained by heartbeat polling
      next();
    }
  }
});

export default router;
