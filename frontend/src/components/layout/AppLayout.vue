<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      v-model:collapsed="collapsed"
      bordered
      :width="220"
      :collapsed-width="64"
      collapse-mode="width"
      show-trigger
      :native-scrollbar="false"
    >
      <div style="padding: 16px; text-align: center; display: flex; align-items: center; justify-content: center; gap: 8px; overflow: hidden; white-space: nowrap">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color: var(--n-color-target, #63e2b7); flex-shrink: 0">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          <path d="M9 12l2 2 4-4"/>
        </svg>
        <n-text v-if="!collapsed" strong style="font-size: 16px">Proxy</n-text>
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
import { computed, h, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../../stores/settings'
import { useAuthStore } from '../../stores/auth'
import { NIcon } from 'naive-ui'

const router = useRouter()
const route = useRoute()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { t, locale } = useI18n()
const collapsed = ref(false)

function renderIcon(path: string) {
  return () => h(NIcon, null, {
    default: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [
      ...path.split('|').map(d => h('path', { d }))
    ])
  })
}

const menuOptions = computed(() => [
  { label: t('nav.dashboard'), key: 'dashboard', icon: renderIcon('M3 3h7v7H3z|M14 3h7v7h-7z|M14 14h7v7h-7z|M3 14h7v7H3z') },
  { label: t('nav.subscriptions'), key: 'subscriptions', icon: renderIcon('M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71|M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71') },
  { label: t('nav.proxies'), key: 'proxies', icon: renderIcon('M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z|M2 12h20') },
  { label: t('nav.rules'), key: 'rules', icon: renderIcon('M3 6h18|M3 12h18|M3 18h18') },
  { label: t('nav.connections'), key: 'connections', icon: renderIcon('M15 7h3a5 5 0 0 1 0 10h-3|M9 17H6A5 5 0 0 1 6 7h3|M8 12h8') },
  { label: t('nav.logs'), key: 'logs', icon: renderIcon('M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z|M14 2v6h6|M16 13H8|M16 17H8|M10 9H8') },
  { label: t('nav.settings'), key: 'settings', icon: renderIcon('M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z|M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z') },
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

<style scoped>
:deep(.n-layout-sider--collapsed) .n-menu-item-content {
  padding-left: 0 !important;
  padding-right: 0 !important;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 64px !important;
}
:deep(.n-layout-sider--collapsed) .n-menu-item-content__icon {
  margin-right: 0 !important;
  margin-left: 0 !important;
}
:deep(.n-layout-sider--collapsed) .n-menu-item-content-header {
  display: none !important;
}
</style>
