<template>
  <n-config-provider :theme="theme">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { darkTheme } from 'naive-ui'
import { useSettingsStore } from './stores/settings'

const settingsStore = useSettingsStore()
const theme = computed(() => settingsStore.theme === 'dark' ? darkTheme : null)
</script>
