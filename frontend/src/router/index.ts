import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/subscriptions',
      name: 'subscriptions',
      component: () => import('../views/SubscriptionsView.vue'),
    },
    {
      path: '/profiles',
      name: 'profiles',
      component: () => import('../views/ProfilesView.vue'),
    },
    {
      path: '/proxies',
      name: 'proxies',
      component: () => import('../views/ProxiesView.vue'),
    },
    {
      path: '/rules',
      name: 'rules',
      component: () => import('../views/RulesView.vue'),
    },
    {
      path: '/connections',
      name: 'connections',
      component: () => import('../views/ConnectionsView.vue'),
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import('../views/LogsView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
    },
  ],
})

let authChecked = false

router.beforeEach(async (to, _from, next) => {
  if (to.meta.public) {
    next()
    return
  }

  const authStore = useAuthStore()

  // On first navigation, verify auth status with server
  if (!authChecked) {
    authChecked = true
    await authStore.checkAuth()
  }

  if (!authStore.isAuthenticated) {
    next({ name: 'login' })
  } else {
    next()
  }
})

export default router
