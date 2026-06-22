<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">{{ t('connections.title') }}</n-text>
        <n-space>
          <n-input v-model:value="searchText" :placeholder="t('connections.searchPlaceholder')" clearable style="width: 200px" size="small" />
          <n-switch v-model:value="paused" size="small">
            <template #checked>{{ t('connections.paused') }}</template>
            <template #unchecked>{{ t('connections.live') }}</template>
          </n-switch>
          <n-button size="small" type="error" @click="handleCloseAll" :loading="closingAll">
            {{ t('connections.closeAll') }}
          </n-button>
        </n-space>
      </n-space>

      <n-space :size="8">
        <n-text depth="3">{{ t('connections.connectionCount') }}: {{ connections.length }}</n-text>
        <n-text depth="3">|</n-text>
        <n-text depth="3">{{ t('connections.totalUpload') }}: {{ formatBytes(snapshot.uploadTotal) }}</n-text>
        <n-text depth="3">|</n-text>
        <n-text depth="3">{{ t('connections.totalDownload') }}: {{ formatBytes(snapshot.downloadTotal) }}</n-text>
      </n-space>

      <n-data-table
        :columns="columns"
        :data="filteredConnections"
        :bordered="false"
        :single-line="false"
        size="small"
        :pagination="{ pageSize: 30 }"
        max-height="calc(100vh - 240px)"
      />
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { NButton, NTag, NText, NSpace, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { DataTableColumns } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { useWebSocket } from '../composables/useWebSocket'
import { kernelApi, type Connection, type ConnectionsSnapshot } from '../api/kernel'
import { formatBytes } from '../utils/format'

const { t } = useI18n()
const message = useMessage()

const paused = ref(false)
const searchText = ref('')
const closingAll = ref(false)
const connections = ref<Connection[]>([])
const snapshot = ref({ uploadTotal: 0, downloadTotal: 0 })

useWebSocket({
  url: '/api/ws/connections',
  onMessage: (data: ConnectionsSnapshot) => {
    if (paused.value) return
    connections.value = data.connections || []
    snapshot.value = {
      uploadTotal: data.uploadTotal,
      downloadTotal: data.downloadTotal,
    }
  },
})

const filteredConnections = computed(() => {
  if (!searchText.value) return connections.value
  const q = searchText.value.toLowerCase()
  return connections.value.filter(
    (c) =>
      c.metadata.host?.toLowerCase().includes(q) ||
      c.metadata.destinationIP?.includes(q) ||
      c.chains?.some((ch) => ch.toLowerCase().includes(q)) ||
      c.rule?.toLowerCase().includes(q)
  )
})

const columns = computed<DataTableColumns<Connection>>(() => [
  {
    title: 'Host',
    key: 'host',
    width: 220,
    ellipsis: { tooltip: true },
    render(row) {
      return h(NText, {}, { default: () => row.metadata.host || row.metadata.destinationIP })
    },
  },
  {
    title: t('connections.type'),
    key: 'type',
    width: 80,
    render(row) {
      return h(NTag, { size: 'tiny' }, { default: () => row.metadata.network + '/' + row.metadata.type })
    },
  },
  {
    title: t('connections.rule'),
    key: 'rule',
    width: 180,
    render(row) {
      return h(NText, { depth: 3, style: 'font-size: 12px' }, {
        default: () => `${row.rule}${row.rulePayload ? ': ' + row.rulePayload : ''}`
      })
    },
  },
  {
    title: t('connections.proxyChain'),
    key: 'chains',
    width: 200,
    ellipsis: { tooltip: true },
    render(row) {
      return h(NText, { style: 'font-size: 12px' }, { default: () => row.chains?.join(' → ') || '' })
    },
  },
  {
    title: t('connections.upload'),
    key: 'upload',
    width: 90,
    render(row) { return h(NText, {}, { default: () => formatBytes(row.upload) }) },
    sorter: (a, b) => a.upload - b.upload,
  },
  {
    title: t('connections.download'),
    key: 'download',
    width: 110,
    render(row) { return h(NText, {}, { default: () => formatBytes(row.download) }) },
    sorter: (a, b) => a.download - b.download,
  },
  {
    title: t('connections.startTime'),
    key: 'start',
    width: 160,
    render(row) {
      return h(NText, { depth: 3, style: 'font-size: 12px' }, {
        default: () => new Date(row.start).toLocaleTimeString(),
      })
    },
    sorter: (a, b) => new Date(a.start).getTime() - new Date(b.start).getTime(),
    defaultSortOrder: 'descend',
  },
  {
    title: '',
    key: 'actions',
    width: 60,
    render(row) {
      return h(NButton, {
        size: 'tiny',
        type: 'error',
        quaternary: true,
        onClick: () => handleCloseOne(row.id),
      }, { default: () => '✕' })
    },
  },
])

async function handleCloseOne(id: string) {
  try {
    await kernelApi.closeConnection(id)
  } catch {
    // ignore
  }
}

async function handleCloseAll() {
  closingAll.value = true
  try {
    await kernelApi.closeAllConnections()
    message.success(t('connections.closeAll'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' + (err.message || err))
  } finally {
    closingAll.value = false
  }
}

</script>
