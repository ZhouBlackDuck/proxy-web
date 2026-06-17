import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '../api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const needsSetup = ref(false)
  const configured = ref(false)

  const isAuthenticated = computed(() => !!token.value)

  async function login(password: string) {
    const res = await authApi.login(password)
    token.value = res.token
    localStorage.setItem('token', res.token)
  }

  async function setup(password: string) {
    const res = await authApi.setup(password)
    token.value = res.token
    localStorage.setItem('token', res.token)
    needsSetup.value = false
    configured.value = true
  }

  async function checkAuth() {
    try {
      // First check if password is configured
      const status = await authApi.status()
      configured.value = status.configured

      if (!status.configured) {
        // No password set yet, need setup
        needsSetup.value = true
        token.value = null
        localStorage.removeItem('token')
        return
      }

      // Password is configured, check if we have a valid token
      if (token.value) {
        try {
          await authApi.check()
        } catch {
          // Token invalid/expired
          token.value = null
          localStorage.removeItem('token')
        }
      }
    } catch {
      // API unreachable
      needsSetup.value = true
    }
  }

  function logout() {
    token.value = null
    localStorage.removeItem('token')
  }

  return { token, needsSetup, configured, isAuthenticated, login, setup, checkAuth, logout }
})
