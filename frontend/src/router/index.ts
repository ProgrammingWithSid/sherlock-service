import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/signup',
    name: 'signup',
    component: () => import('@/views/Signup.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/auth/callback',
    name: 'auth-callback',
    component: () => import('@/views/AuthCallback.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/admin',
    name: 'admin-dashboard',
    component: () => import('@/views/AdminDashboard.vue'),
    meta: { requiresAuth: true, requiresSuperAdmin: true },
  },
  {
    path: '/',
    name: 'dashboard',
    component: () => import('@/views/Dashboard.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/repositories',
    name: 'repositories',
    component: () => import('@/views/Repositories.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/reviews',
    name: 'reviews',
    component: () => import('@/views/Reviews.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/reviews/:id',
    name: 'review-detail',
    component: () => import('@/views/ReviewDetail.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/Settings.vue'),
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard for authentication
router.beforeEach(async (to, from, next) => {
  const { useAuthStore } = await import('@/stores/auth')
  const authStore = useAuthStore()

  // Initialize auth if not already done
  if (!authStore.isAuthenticated) {
    await authStore.init()
  }

  // Check if route requires authentication
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  // Check if route requires super admin
  if (to.meta.requiresSuperAdmin && !authStore.isSuperAdmin) {
    next({ name: 'dashboard' })
    return
  }

  // Redirect authenticated users away from login/signup
  if ((to.name === 'login' || to.name === 'signup') && authStore.isAuthenticated) {
    if (authStore.isSuperAdmin) {
      next('/admin')
    } else {
      next('/')
    }
    return
  }

  next()
})

export default router
