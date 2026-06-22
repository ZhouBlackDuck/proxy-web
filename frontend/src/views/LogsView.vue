<template>
  <AppLayout>
    <n-space vertical :size="12" style="height: 100%">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">{{ t('logs.title') }}</n-text>
        <n-space :size="8">
          <n-select
            v-model:value="logLevel"
            :options="levelOptions"
            style="width: 120px"
            size="small"
          />
          <n-input
            v-model:value="searchText"
            :placeholder="t('logs.searchPlaceholder')"
            clearable
            style="width: 240px"
            size="small"
          >
            <template #suffix>
              <n-text depth="3" style="font-size: 11px">{{ filteredLogs.length }}</n-text>
            </template>
          </n-input>
          <n-button
            :type="paused ? 'default' : 'primary'"
            size="small"
            @click="togglePause"
          >
            {{ paused ? t('logs.paused') : t('logs.live') }}
          </n-button>
          <n-button size="small" @click="handleClearLogs">{{ t('logs.clear') }}</n-button>
        </n-space>
      </n-space>

      <!-- Log list -->
      <n-card
        style="flex: 1; overflow: hidden"
        content-style="padding: 0; height: 100%; overflow-y: auto; font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace; font-size: 12px; line-height: 1.5"
        :bordered="true"
      >
        <div ref="logContainer" style="padding: 8px 0">
          <div v-if="filteredLogs.length === 0" style="color: #666; text-align: center; padding: 60px 0">
            {{ t('logs.waiting') }}
          </div>
          <div
            v-for="(log, i) in filteredLogs"
            :key="i"
            class="log-entry"
          >
            <span class="log-time">{{ log.time }}</span>
            <span class="log-type" :class="'log-type-' + log.type">{{ log.type.toUpperCase() }}</span>
            <span class="log-payload" v-html="highlightPayload(log.payload)"></span>
          </div>
        </div>
      </n-card>
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '../components/layout/AppLayout.vue'
import { useLogStore } from '../stores/logs'

const { t } = useI18n()
const logStore = useLogStore()

const logLevel = ref('all')
const searchText = ref('')
const paused = ref(false)

function togglePause() {
  paused.value = !paused.value
  if (paused.value) {
    // Pausing: record current time, hide logs arriving after this
    logStore.pauseTime = Math.floor(Date.now() / 1000)
  } else {
    // Resuming: clear pause filter, show all current logs
    logStore.pauseTime = null
  }
}

const levelOptions = [
  { label: 'All', value: 'all' },
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warning' },
  { label: 'Error', value: 'error' },
]

const filteredLogs = computed(() => {
  return logStore.getFilteredLogs(logLevel.value, searchText.value)
})

function highlightPayload(text: string): string {
  if (!searchText.value.trim()) return escapeHtml(text)
  const escaped = escapeHtml(searchText.value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return escapeHtml(text).replace(
    new RegExp(`(${escaped})`, 'gi'),
    '<mark style="background:rgba(255,235,59,0.3);border-radius:2px;padding:0 2px">$1</mark>'
  )
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

async function fetchLogs() {
  const token = localStorage.getItem('token') || ''
  await logStore.fetchLogs(token)
}

async function handleClearLogs() {
  const token = localStorage.getItem('token') || ''
  try {
    await fetch('/api/logs', {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    })
    logStore.clearLogs()
    logStore.pauseTime = null
  } catch {
    // ignore
  }
}

onMounted(() => {
  fetchLogs()
  // Auto-refresh every 2 seconds
  setInterval(() => {
    if (!paused.value) {
      fetchLogs()
    }
  }, 2000)
})
</script>

<style scoped>
.log-entry {
  padding: 3px 12px;
  border-bottom: 1px solid rgba(128,128,128,0.06);
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.log-entry:hover {
  background: rgba(128,128,128,0.04);
}
.log-time {
  color: #666;
  font-size: 11px;
  flex-shrink: 0;
}
.log-type {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 3px;
  padding: 1px 5px;
  flex-shrink: 0;
  letter-spacing: 0.5px;
}
.log-type-info { color: #63e2b7; background: rgba(99,226,183,0.1); }
.log-type-warning { color: #f2c97d; background: rgba(242,201,125,0.1); }
.log-type-error { color: #e88080; background: rgba(232,128,128,0.1); }
.log-type-debug { color: #888; background: rgba(128,128,128,0.08); }
.log-payload {
  color: #ddd;
  word-break: break-all;
  overflow-wrap: anywhere;
}
</style>
