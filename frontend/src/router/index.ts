import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginPage.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/identity',
    name: 'IdentityPicker',
    component: () => import('@/views/IdentityPicker.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/list',
    name: 'ChatList',
    component: () => import('@/views/ChatListView.vue'),
    meta: { requiresAuth: true, requiresIdentity: true }
  },
  {
    path: '/chat/:userId?',
    name: 'ChatRoom',
    component: () => import('@/views/ChatRoomView.vue'),
    meta: { requiresAuth: true, requiresIdentity: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  const userStore = useUserStore()

  // 需要认证的路由
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    const isValid = await authStore.checkToken()
    if (!isValid) {
      next('/login')
      return
    }
  }

  // 需要身份的路由
  if (to.meta.requiresIdentity && !userStore.currentUser) {
    next('/identity')
    return
  }

  next()
})

export default router
