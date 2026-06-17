<template>
  <AppLayout>
    <n-grid :x-gap="16" :y-gap="16" :cols="4" responsive="screen" item-responsive style="align-items: stretch">
      <!-- 内核状态 -->
      <n-gi span="4 m:1">
        <n-card title="内核状态" size="small" style="height: 100%">
          <n-descriptions :column="1" label-placement="left" size="small">
            <n-descriptions-item label="状态">
              <n-tag :type="kernelStore.alive ? 'success' : 'error'" size="small">
                {{ kernelStore.alive ? '运行中' : '已停止' }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="版本">
              {{ kernelStore.version?.version || '-' }}
            </n-descriptions-item>
            <n-descriptions-item label="模式">
              {{ kernelStore.config?.mode || '-' }}
            </n-descriptions-item>
            <n-descriptions-item label="内存">
              {{ formatBytes(kernelStore.memory.inuse) }}
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-gi>

      <!-- 流量统计 -->
      <n-gi span="4 m:1">
        <n-card title="流量统计" size="small" style="height: 100%">
          <n-statistic label="上传速率" :value="kernelStore.traffic.up">
            <template #suffix>
              <n-text depth="3">{{ formatSpeed(kernelStore.traffic.up) }}/s</n-text>
            </template>
          </n-statistic>
          <n-statistic label="下载速率" :value="kernelStore.traffic.down">
            <template #suffix>
              <n-text depth="3">{{ formatSpeed(kernelStore.traffic.down) }}/s</n-text>
            </template>
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- 代理模式 -->
      <n-gi span="4 m:1">
        <n-card title="代理模式" size="small" style="height: 100%">
          <n-radio-group
            :value="kernelStore.config?.mode || 'rule'"
            @update:value="handleModeChange"
          >
            <n-space>
              <n-radio-button value="rule">规则</n-radio-button>
              <n-radio-button value="global">全局</n-radio-button>
              <n-radio-button value="direct">直连</n-radio-button>
            </n-space>
          </n-radio-group>
          <n-divider style="margin: 12px 0" />
          <n-space vertical :size="8">
            <n-space justify="space-between" align="center">
              <n-text>IPv6</n-text>
              <n-switch
                :value="kernelStore.config?.ipv6 ?? false"
                @update:value="handleIPv6Change"
                size="small"
              />
            </n-space>
            <n-space justify="space-between" align="center">
              <n-text>局域网</n-text>
              <n-switch
                :value="kernelStore.config?.['allow-lan'] ?? false"
                @update:value="handleAllowLanChange"
                size="small"
              />
            </n-space>
            <n-space justify="space-between" align="center">
              <n-text>TUN</n-text>
              <n-switch
                :value="kernelStore.config?.tun?.enable ?? false"
                @update:value="handleTunChange"
                size="small"
              />
            </n-space>
          </n-space>
        </n-card>
      </n-gi>

      <!-- 快速操作 -->
      <n-gi span="4 m:1">
        <n-card title="快速操作" size="small" style="height: 100%">
          <n-space vertical :size="8">
            <n-button
              size="small"
              :loading="closingConnections"
              @click="handleCloseAll"
              block
            >
              关闭全部连接
            </n-button>
            <n-button
              size="small"
              :loading="updatingGeo"
              @click="handleUpdateGeo"
              block
            >
              更新 GeoIP/GeoSite
            </n-button>
            <n-button
              size="small"
              :loading="restarting"
              @click="handleRestart"
              block
              type="warning"
            >
              重启内核
            </n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Connectivity Test -->
    <n-card title="连通性测试" size="small" style="margin-top: 16px">
      <template #header-extra>
        <n-button size="tiny" :loading="testingConnectivity" @click="handleTestAll">
          {{ t('dashboard.testing') }}
        </n-button>
      </template>
      <n-spin :show="testingConnectivity">
        <n-grid :x-gap="8" :y-gap="8" :cols="6" responsive="screen" item-responsive>
          <n-gi v-for="result in testResults" :key="result.name" span="6 s:3 m:2 l:1">
            <n-card
              size="tiny"
              :style="{ borderLeft: `3px solid ${result.ok ? '#18a058' : '#d03050'}` }"
            >
              <n-space vertical :size="2">
                <n-text style="font-size: 13px">
                  {{ result.icon }} {{ result.name }}
                </n-text>
                <n-text v-if="result.latency >= 0" depth="3" style="font-size: 11px">
                  {{ result.latency }}ms
                </n-text>
                <n-text v-else-if="result.error" type="error" style="font-size: 11px">
                  {{ t('dashboard.unreachable') }}
                </n-text>
                <n-text v-else depth="3" style="font-size: 11px">
                  —
                </n-text>
              </n-space>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-if="testResults.length === 0 && !testingConnectivity" description="点击测试按钮检测连通性" size="small" />
      </n-spin>
    </n-card>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { useKernelStore } from '../stores/kernel'
import { kernelApi } from '../api/kernel'
import { testApi, type TestResult } from '../api/test'
import { useWebSocket } from '../composables/useWebSocket'

const { t } = useI18n()
const message = useMessage()
const kernelStore = useKernelStore()

const closingConnections = ref(false)
const updatingGeo = ref(false)
const restarting = ref(false)
const testingConnectivity = ref(false)
const testResults = ref<TestResult[]>([])

// Traffic WebSocket
useWebSocket({
  url: '/api/ws/traffic',
  onMessage: (data: { up: number; down: number }) => {
    kernelStore.updateTraffic(data)
  },
})

// Memory WebSocket
useWebSocket({
  url: '/api/ws/memory',
  onMessage: (data: { inuse: number }) => {
    kernelStore.updateMemory(data)
  },
})

onMounted(async () => {
  await kernelStore.initialize()
})

async function handleModeChange(mode: string) {
  try {
    await kernelStore.switchMode(mode)
    message.success(`模式已切换为: ${mode}`)
  } catch (err: any) {
    message.error('模式切换失败: ' + (err.message || err))
  }
}

async function handleIPv6Change(val: boolean) {
  try {
    await kernelStore.toggleIPv6(val)
    message.success(`IPv6 已${val ? '开启' : '关闭'}`)
  } catch (err: any) {
    message.error('操作失败: ' + (err.message || err))
  }
}

async function handleAllowLanChange(val: boolean) {
  try {
    await kernelStore.toggleAllowLan(val)
    message.success(`局域网已${val ? '开启' : '关闭'}`)
  } catch (err: any) {
    message.error('操作失败: ' + (err.message || err))
  }
}

async function handleTunChange(val: boolean) {
  try {
    await kernelStore.toggleTun(val)
    message.success(`TUN 模式已${val ? '开启' : '关闭'}`)
  } catch (err: any) {
    message.error('操作失败: ' + (err.message || err))
  }
}

async function handleCloseAll() {
  closingConnections.value = true
  try {
    await kernelApi.closeAllConnections()
    message.success('已关闭全部连接')
  } catch (err: any) {
    message.error('关闭失败: ' + (err.message || err))
  } finally {
    closingConnections.value = false
  }
}

async function handleUpdateGeo() {
  updatingGeo.value = true
  try {
    await Promise.all([
      kernelApi.updateGeo(),
      new Promise(resolve => setTimeout(resolve, 1500)), // minimum 1.5s loading
    ])
    message.success('GeoIP/GeoSite 更新已触发')
  } catch (err: any) {
    message.error('更新失败: ' + (err.message || err))
  } finally {
    updatingGeo.value = false
  }
}

async function handleRestart() {
  restarting.value = true
  try {
    await kernelApi.restart()
    message.success(t('dashboard.kernelRestarting'))
    // Wait for kernel to come back
    setTimeout(async () => {
      await kernelStore.initialize()
      restarting.value = false
    }, 5000)
  } catch (err: any) {
    message.error('重启失败: ' + (err.message || err))
    restarting.value = false
  }
}

async function handleTestAll() {
  testingConnectivity.value = true
  testResults.value = []
  try {
    testResults.value = await testApi.testAll()
  } catch (err: any) {
    message.error('测试失败: ' + (err.message || err))
  } finally {
    testingConnectivity.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatSpeed(bytesPerSec: number): string {
  return formatBytes(bytesPerSec)
}
</script>
