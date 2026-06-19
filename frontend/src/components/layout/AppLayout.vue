<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      :width="220"
      :collapsed-width="64"
      collapse-mode="width"
      show-trigger
      :native-scrollbar="false"
    >
      <div style="padding: 16px; text-align: center; display: flex; align-items: center; justify-content: center; gap: 8px">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color: var(--n-color-target, #63e2b7)">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          <path d="M9 12l2 2 4-4"/>
        </svg>
        <n-text strong style="font-size: 16px">Proxy</n-text>
      </div>
      <n-menu
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="padding: 12px 24px; display: flex; align-items: center; justify-content: space-between">
        <n-breadcrumb>
          <n-breadcrumb-item>{{ currentTitle }}</n-breadcrumb-item>
        </n-breadcrumb>
        <n-space align="center">
          <n-switch
            :value="settingsStore.theme === 'dark'"
            @update:value="toggleTheme"
            size="small"
          >
            <template #checked>🌙</template>
            <template #unchecked>☀️</template>
          </n-switch>
          <n-button quaternary size="small" @click="toggleLocale">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </n-button>
          <n-button quaternary size="small" @click="handleLogout">
            {{ t('auth.logout') }}
          </n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content
        content-style="padding: 24px;"
        :native-scrollbar="false"
      >
        <slot />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../../stores/settings'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const route = useRoute()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { t, locale } = useI18n()

const menuOptions = computed(() => [
  { label: t('nav.dashboard'), key: 'dashboard' },
  { label: t('nav.subscriptions'), key: 'subscriptions' },
  { label: t('nav.proxies'), key: 'proxies' },
  { label: t('nav.rules'), key: 'rules' },
  { label: t('nav.connections'), key: 'connections' },
  { label: t('nav.logs'), key: 'logs' },
  { label: t('nav.settings'), key: 'settings' },
])

const activeKey = computed(() => {
  const name = route.name as string
  return name || 'dashboard'
})

const currentTitle = computed(() => {
  const item = menuOptions.value.find(m => m.key === activeKey.value)
  return item?.label || 'Proxy WebUI'
})

function handleMenuSelect(key: string) {
  router.push({ name: key })
}

function toggleTheme(val: boolean) {
  settingsStore.setTheme(val ? 'dark' : 'light')
}

function toggleLocale() {
  const newLocale = locale.value === 'zh' ? 'en' : 'zh'
  locale.value = newLocale
  settingsStore.setLanguage(newLocale)
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>
