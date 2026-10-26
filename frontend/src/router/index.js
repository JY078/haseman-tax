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
          redirect: 'dashboard',
        },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('../views/dashboard/index.vue'),
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('../views/users/index.vue'),
        },
        {
          path: 'departments',
          name: 'departments',
          component: () => import('../views/departments/index.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('../views/settings/index.vue'),
        },
        {
          path: 'models',
          name: 'models',
          component: () => import('../views/models/index.vue'),
        },
        {
          path: 'usage',
          name: 'usage',
          component: () => import('../views/usage/index.vue'),
        },
        {
          path: 'image-stats',
          name: 'image-stats',
          component: () => import('../views/image-stats/index.vue'),
        },
        {
          path: 'roles',
          name: 'roles',
          component: () => import('../views/roles/index.vue'),
        },
        {
          path: 'prompts',
          name: 'prompts',
          component: () => import('../views/prompts/index.vue'),
        },
        {
          path: 'product-prompts',
          name: 'product-prompts',
          component: () => import('../views/product-prompts/index.vue'),
        },
        {
          path: 'scene-styles',
          name: 'scene-styles',
          component: () => import('../views/scene-styles/index.vue'),
        },
        {
          path: 'product-colors',
          name: 'product-colors',
          component: () => import('../views/product-colors/index.vue'),
        },
        {
          path: 'reference-galleries',
          name: 'reference-galleries',
          component: () => import('../views/reference-galleries/index.vue'),
        },
        {
          path: 'task-queue',
          name: 'task-queue',
          component: () => import('../views/task-queue/index.vue'),
        },
        {
          path: 'announcements',
          name: 'announcements',
          component: () => import('../views/announcements/index.vue'),
        },
        {
          path: 'menus',
          name: 'menus',
          component: () => import('../views/menus/index.vue'),
        },
        {
          path: 'feedbacks',
          name: 'feedbacks',
          component: () => import('../views/feedbacks/index.vue'),
        },
      ],
    },
  ],
});

router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore();
  
  // If the route explicitly requires NO auth (like login), allow access
  if (to.meta.requiresAuth === false) {
    // But if user is already logged in and tries to access login page, redirect to dashboard
    if (to.name === 'login' && userStore.token) {
      next({ name: 'dashboard' });
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
