<template>
  <AppLayout>
    <n-space vertical :size="12" style="height: 100%">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">{{ t('logs.title') }}</n-text>
        <n-space align="center" :size="8">
          <n-button
            :type="autoRefresh ? 'primary' : 'default'"
            size="small"
            @click="autoRefresh = !autoRefresh"
          >
            <template #icon>
              <span>{{ autoRefresh ? '⏸' : '▶' }}</span>
            </template>
            {{ autoRefresh ? t('logs.live') : t('logs.paused') }}
          </n-button>
          <n-button size="small" @click="isDescending = !isDescending">
            {{ isDescending ? '↓ 降序' : '↑ 升序' }}
          </n-button>
          <n-button size="small" @click="clearLogs">{{ t('logs.clear') }}</n-button>
        </n-space>
      </n-space>

      <!-- Filter bar -->
      <n-space align="center" :size="8">
        <n-input
          v-model:value="searchText"
          :placeholder="t('logs.searchPlaceholder')"
          clearable
          style="width: 300px"
          size="small"
        >
          <template #suffix>
            <n-text depth="3" style="font-size: 11px">{{ filteredLogs.length }}</n-text>
          </template>
        </n-input>
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
            v-for="(log, i) in displayLogs"
            :key="i"
            class="log-entry"
          >
            <div class="log-header">
              <span class="log-time">{{ log.time }}</span>
              <span class="log-type" :class="'log-type-' + log.type">{{ log.type.toUpperCase() }}</span>
            </div>
            <div class="log-payload" v-html="highlightPayload(log.payload)"></div>
          </div>
        </div>
      </n-card>
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '../components/layout/AppLayout.vue'
import client from '../api/client'

const { t } = useI18n()

interface LogEntry {
  time: string
  type: string
  payload: string
}

const searchText = ref('')
const autoRefresh = ref(true)
const isDescending = ref(false)
const logs = ref<LogEntry[]>([])
const logContainer = ref<HTMLElement | null>(null)

let refreshTimer: ReturnType<typeof setInterval> | null = null

// Client-side search filtering
const filteredLogs = computed(() => {
  let result = logs.value
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(
      l => `${l.time} ${l.type} ${l.payload}`.toLowerCase().includes(q)
    )
  }
  return result
})

// Apply sort order
const displayLogs = computed(() => {
  return isDescending.value ? [...filteredLogs.value].reverse() : filteredLogs.value
})

function highlightPayload(text: string): string {
  if (!searchText.value.trim()) {
    return escapeHtml(text)
  }
  const escapedSearch = escapeHtml(searchText.value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return escapeHtml(text).replace(
    new RegExp(`(${escapedSearch})`, 'gi'),
    '<mark style="background:rgba(255,235,59,0.3);border-radius:2px;padding:0 2px">$1</mark>'
  )
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

async function fetchLogs() {
  try {
    const { data } = await client.get('/logs', { params: { limit: 500 } })
    const newLogs: LogEntry[] = data.logs || []
    // Append new logs, deduplicate by keeping last 500
    if (logs.value.length === 0) {
      logs.value = newLogs
    } else {
      // Merge: keep existing + add new ones that aren't already there
      const existingSet = new Set(logs.value.map(l => `${l.time}|${l.payload}`))
      const fresh = newLogs.filter(l => !existingSet.has(`${l.time}|${l.payload}`))
      if (fresh.length > 0) {
        logs.value = [...logs.value, ...fresh].slice(-500)
      }
    }
    nextTick(() => {
      if (logContainer.value && !isDescending.value) {
        const el = logContainer.value
        const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40
        if (isNearBottom) {
          el.scrollTop = el.scrollHeight
        }
      }
    })
  } catch {
    // ignore
  }
}

function clearLogs() {
  client.delete('/logs').then(() => {
    logs.value = []
  }).catch(() => {
    // ignore
  })
}

function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(() => {
    if (autoRefresh.value) {
      fetchLogs()
    }
  }, 2000)
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  fetchLogs()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

watch(autoRefresh, (val) => {
  if (val) startAutoRefresh()
  else stopAutoRefresh()
})
</script>

<style scoped>
.log-entry {
  padding: 6px 12px;
  border-bottom: 1px solid rgba(128, 128, 128, 0.08);
  user-select: text;
}
.log-entry:hover {
  background: rgba(128, 128, 128, 0.04);
}
.log-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}
.log-time {
  color: #888;
  font-size: 11px;
}
.log-type {
  display: inline-block;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 3px;
  padding: 1px 6px;
  letter-spacing: 0.5px;
}
.log-type-info {
  color: #63e2b7;
  background: rgba(99, 226, 183, 0.12);
}
.log-type-warning {
  color: #f2c97d;
  background: rgba(242, 201, 125, 0.12);
}
.log-type-error {
  color: #e88080;
  background: rgba(232, 128, 128, 0.12);
}
.log-type-debug {
  color: #999;
  background: rgba(128, 128, 128, 0.1);
}
.log-payload {
  color: #ddd;
  word-break: break-all;
  overflow-wrap: anywhere;
}
</style>
