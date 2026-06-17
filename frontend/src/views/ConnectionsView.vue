<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">连接监控</n-text>
        <n-space>
          <n-input v-model:value="searchText" placeholder="搜索 host/chain..." clearable style="width: 200px" size="small" />
          <n-switch v-model:value="paused" size="small">
            <template #checked>暂停</template>
            <template #unchecked>实时</template>
          </n-switch>
          <n-button size="small" type="error" @click="handleCloseAll" :loading="closingAll">
            关闭全部
          </n-button>
        </n-space>
      </n-space>

      <n-space :size="8">
        <n-text depth="3">连接数: {{ connections.length }}</n-text>
        <n-text depth="3">|</n-text>
        <n-text depth="3">总上传: {{ formatBytes(snapshot.uploadTotal) }}</n-text>
        <n-text depth="3">|</n-text>
        <n-text depth="3">总下载: {{ formatBytes(snapshot.downloadTotal) }}</n-text>
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
import type { DataTableColumns } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { useWebSocket } from '../composables/useWebSocket'
import { kernelApi, type Connection, type ConnectionsSnapshot } from '../api/kernel'

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

const columns: DataTableColumns<Connection> = [
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
    title: '类型',
    key: 'type',
    width: 80,
    render(row) {
      return h(NTag, { size: 'tiny' }, { default: () => row.metadata.network + '/' + row.metadata.type })
    },
  },
  {
    title: '规则',
    key: 'rule',
    width: 180,
    render(row) {
      return h(NText, { depth: 3, style: 'font-size: 12px' }, {
        default: () => `${row.rule}${row.rulePayload ? ': ' + row.rulePayload : ''}`
      })
    },
  },
  {
    title: '代理链',
    key: 'chains',
    width: 200,
    ellipsis: { tooltip: true },
    render(row) {
      return h(NText, { style: 'font-size: 12px' }, { default: () => row.chains?.join(' → ') || '' })
    },
  },
  {
    title: '上传',
    key: 'upload',
    width: 90,
    render(row) { return h(NText, {}, { default: () => formatBytes(row.upload) }) },
    sorter: (a, b) => a.upload - b.upload,
  },
  {
    title: '下载',
    key: 'download',
    width: 90,
    render(row) { return h(NText, {}, { default: () => formatBytes(row.download) }) },
    sorter: (a, b) => a.download - b.download,
  },
  {
    title: '开始时间',
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
]

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
    message.success('已关闭全部连接')
  } catch (err: any) {
    message.error('关闭失败: ' + (err.message || err))
  } finally {
    closingAll.value = false
  }
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>
