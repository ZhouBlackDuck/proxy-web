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
            @update:value="handleLevelChange"
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
          <n-button :type="autoScroll ? 'primary' : 'default'" size="small" @click="autoScroll = !autoScroll">
            {{ autoScroll ? t('logs.autoScroll') : t('logs.manualScroll') }}
          </n-button>
          <n-button size="small" @click="logStore.clearLogs()">{{ t('logs.clear') }}</n-button>
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
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '../components/layout/AppLayout.vue'
import { useWebSocket } from '../composables/useWebSocket'
import { useLogStore, type LogEntry } from '../stores/logs'

const { t } = useI18n()
const logStore = useLogStore()

const logLevel = ref('info')
const searchText = ref('')
const autoScroll = ref(true)
const logContainer = ref<HTMLElement | null>(null)

const levelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warning' },
  { label: 'Error', value: 'error' },
]

let wsDisconnect: (() => void) | null = null

function connectLogs() {
  if (wsDisconnect) wsDisconnect()

  const { disconnect } = useWebSocket({
    url: `/api/ws/logs?level=${logLevel.value}`,
    onMessage: (data: any) => {
      if (typeof data === 'string') {
        const entry = parseLogLine(data)
        if (entry) logStore.addLog(entry)
      } else if (data && data.type) {
        logStore.addLog({
          time: data.time || formatNow(),
          type: normalizeLevel(data.type),
          payload: data.payload || '',
        })
      }
    },
  })
  wsDisconnect = disconnect
}

// Auto-scroll when new logs arrive
watch(() => logStore.logs.length, () => {
  if (autoScroll.value) {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight
      }
    })
  }
})

function parseLogLine(line: string): LogEntry | null {
  const match = line.match(/^time="([^"]*)" level=(\w+) msg="(.*)"$/)
  if (match) {
    return { time: formatTime(match[1]), type: normalizeLevel(match[2]), payload: match[3] }
  }
  if (line.trim()) {
    return { time: formatNow(), type: guessLevel(line), payload: line }
  }
  return null
}

function formatTime(iso: string): string {
  const idx = iso.indexOf('T')
  if (idx === -1) return iso
  let timePart = iso.substring(idx + 1)
  const dotIdx = timePart.indexOf('.')
  if (dotIdx !== -1) timePart = timePart.substring(0, dotIdx)
  return timePart
}

function formatNow(): string {
  const d = new Date()
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function normalizeLevel(level: string): string {
  switch (level.toLowerCase()) {
    case 'error': case 'fatal': case 'panic': return 'error'
    case 'warning': case 'warn': return 'warning'
    case 'debug': return 'debug'
    default: return 'info'
  }
}

function guessLevel(line: string): string {
  const lower = line.toLowerCase()
  if (lower.includes('error') || lower.includes('fatal')) return 'error'
  if (lower.includes('warn')) return 'warning'
  if (lower.includes('debug')) return 'debug'
  return 'info'
}

const LOG_LEVEL_ORDER: Record<string, number> = { debug: 0, info: 1, warning: 2, error: 3 }

const filteredLogs = computed(() => {
  let result = logStore.logs
  // Filter by level (show selected level and above)
  const minLevel = LOG_LEVEL_ORDER[logLevel.value] ?? 0
  result = result.filter(l => (LOG_LEVEL_ORDER[l.type] ?? 1) >= minLevel)
  // Filter by search text
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(
      l => `${l.time} ${l.type} ${l.payload}`.toLowerCase().includes(q)
    )
  }
  return result
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

function handleLevelChange() {
  connectLogs()
}

onMounted(async () => {
  // Load historical logs from file first
  const token = localStorage.getItem('token') || ''
  await logStore.loadHistory(token)
  // Then connect WS for real-time
  connectLogs()
})

onUnmounted(() => {
  if (wsDisconnect) wsDisconnect()
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
.log-entry:hover { background: rgba(128,128,128,0.04); }
.log-time { color: #666; font-size: 11px; flex-shrink: 0; }
.log-type {
  font-size: 10px; font-weight: 600; text-transform: uppercase;
  border-radius: 3px; padding: 1px 5px; flex-shrink: 0; letter-spacing: 0.5px;
}
.log-type-info { color: #63e2b7; background: rgba(99,226,183,0.1); }
.log-type-warning { color: #f2c97d; background: rgba(242,201,125,0.1); }
.log-type-error { color: #e88080; background: rgba(232,128,128,0.1); }
.log-type-debug { color: #888; background: rgba(128,128,128,0.08); }
.log-payload { color: #ddd; word-break: break-all; overflow-wrap: anywhere; }
</style>
