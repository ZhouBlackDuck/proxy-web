<template>
  <AppLayout>
    <n-grid :x-gap="16" :y-gap="16" :cols="4" responsive="screen" item-responsive style="align-items: stretch">
      <!-- 内核状态 -->
      <n-gi span="4 m:1">
        <n-card :title="t('dashboard.kernelStatus')" size="small" style="height: 100%">
          <n-descriptions :column="1" label-placement="left" size="small">
            <n-descriptions-item :label="t('dashboard.kernelStatus')">
              <n-tag :type="kernelStore.alive ? 'success' : 'error'" size="small">
                {{ kernelStore.alive ? t('dashboard.running') : t('dashboard.stopped') }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item :label="t('dashboard.version')">
              {{ kernelStore.version?.version || '-' }}
            </n-descriptions-item>
            <n-descriptions-item :label="t('dashboard.mode')">
              {{ kernelStore.config?.mode || '-' }}
            </n-descriptions-item>
            <n-descriptions-item :label="t('dashboard.memory')">
              {{ formatBytes(kernelStore.memory.inuse) }}
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-gi>

      <!-- 流量统计 + 曲线图 -->
      <n-gi span="4 m:2">
        <n-card :title="t('dashboard.trafficStats')" size="small" style="height: 100%">
          <div style="display: flex; gap: 16px; height: 100%">
            <!-- Left: speed numbers -->
            <div style="min-width: 100px">
              <div style="margin-bottom: 12px">
                <div style="font-size: 11px; color: #888; margin-bottom: 2px">{{ t("dashboard.uploadSpeed") }}</div>
                <div style="font-size: 18px; font-weight: 600; color: #f2c97d">{{ formatSpeed(kernelStore.traffic.up) }}/s</div>
              </div>
              <div style="margin-bottom: 12px">
                <div style="font-size: 11px; color: #888; margin-bottom: 2px">{{ t("dashboard.downloadSpeed") }}</div>
                <div style="font-size: 18px; font-weight: 600; color: #63e2b7">{{ formatSpeed(kernelStore.traffic.down) }}/s</div>
              </div>
              <div style="font-size: 11px; color: #666">
                <div>{{ t("dashboard.totalUpload") }} ↑ {{ formatBytes(kernelStore.totalUp) }}</div>
                <div>{{ t("dashboard.totalDownload") }} ↓ {{ formatBytes(kernelStore.totalDown) }}</div>
              </div>
            </div>
            <!-- Right: traffic graph -->
            <div style="flex: 1; position: relative; height: 120px; overflow: hidden">
              <canvas ref="trafficCanvas" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%"></canvas>
            </div>
          </div>
        </n-card>
      </n-gi>

      <!-- 代理模式 -->
      <n-gi span="4 m:1">
        <n-card :title="t('dashboard.proxyMode')" size="small" style="height: 100%">
          <n-radio-group
            :value="kernelStore.config?.mode || 'rule'"
            @update:value="handleModeChange"
          >
            <n-space>
              <n-radio-button value="rule">{{ t('dashboard.rule') }}</n-radio-button>
              <n-radio-button value="global">{{ t('dashboard.global') }}</n-radio-button>
              <n-radio-button value="direct">{{ t('dashboard.direct') }}</n-radio-button>
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
              <n-text>{{ t('dashboard.lanAccess') }}</n-text>
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
    </n-grid>

    <!-- Second row: quick actions + connectivity -->
    <n-grid :x-gap="16" :y-gap="16" :cols="4" responsive="screen" item-responsive style="align-items: stretch; margin-top: 16px">
      <!-- 快速操作 -->
      <n-gi span="4 m:1">
        <n-card :title="t('dashboard.quickActions')" size="small" style="height: 100%">
          <n-space vertical :size="8">
            <n-button size="small" :loading="closingConnections" @click="handleCloseAll" block>{{ t('dashboard.closeAllConnections') }}</n-button>
            <n-button size="small" :loading="updatingGeo" @click="handleUpdateGeo" block>{{ t('dashboard.updateGeo') }}</n-button>
            <n-button size="small" :loading="restarting" @click="handleRestart" block type="warning">{{ t('dashboard.restartKernel') }}</n-button>
          </n-space>
        </n-card>
      </n-gi>

      <!-- Connectivity Test -->
      <n-gi span="4 m:3">
        <n-card :title="t('dashboard.connectivity')" size="small" style="height: 100%">
          <template #header-extra>
            <n-space :size="4">
              <n-button size="tiny" :type="manageMode ? 'warning' : 'default'" :disabled="testingConnectivity || testingSingle !== null" @click="manageMode = !manageMode">
                {{ manageMode ? t('dashboard.done') : t('dashboard.manage') }}
              </n-button>
              <n-button size="tiny" :loading="testingConnectivity" :disabled="manageMode || testingSingle !== null" @click="handleTestAll">{{ t('dashboard.test') }}</n-button>
            </n-space>
          </template>
          <n-spin :show="testingConnectivity && testingSingle === null" style="min-height: 60px">
            <n-grid :x-gap="10" :y-gap="10" :cols="8" responsive="screen" item-responsive>
              <n-gi v-for="(site, i) in testSites" :key="site.name" span="4 m:2 l:1">
                <div
                  class="site-card"
                  :class="{
                    'site-ok': resultMap[site.name]?.ok,
                    'site-fail': resultMap[site.name] && !resultMap[site.name].ok && resultMap[site.name].latency < 0,
                    'site-manage': manageMode,
                    'site-loading': testingSingle === site.name,
                  }"
                  @click="manageMode ? openEditSite(i) : handleTestSingle(site)"
                >
                  <div v-if="manageMode" class="site-delete" @click.stop="removeSite(i)">✕</div>
                  <n-spin :show="testingSingle === site.name" size="small">
                    <div class="site-icon" v-html="getSiteIcon(site.icon)"></div>
                    <div class="site-name">{{ site.name }}</div>
                    <div v-if="resultMap[site.name]" class="site-latency" :class="{ timeout: resultMap[site.name].latency < 0 }">
                      {{ resultMap[site.name].latency >= 0 ? resultMap[site.name].latency + 'ms' : t('dashboard.timeout') }}
                    </div>
                    <div v-else class="site-latency idle">{{ t('dashboard.idle') }}</div>
                  </n-spin>
                </div>
              </n-gi>
              <n-gi v-if="manageMode" span="4 m:2 l:1">
                <div class="site-card site-add site-card-fixed" @click="addSite">
                  <div class="site-icon" style="font-size: 20px; color: #63e2b7">+</div>
                  <div class="site-name" style="color: #63e2b7">{{ t('dashboard.addSite') }}</div>
                </div>
              </n-gi>
            </n-grid>
            <n-empty v-if="testSites.length === 0 && !testingConnectivity" :description="t('dashboard.addManageHint')" size="small" />
          </n-spin>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Edit Site Modal -->
    <n-modal v-model:show="showEditModal" preset="dialog" :title="t('dashboard.editSite')" style="width: 400px" @update:show="handleEditModalClose">
      <n-form v-if="editingSite" label-placement="left" label-width="60">
        <n-form-item :label="t('dashboard.siteIcon')">
          <div
            class="icon-picker-trigger"
            @click="showIconPicker = true"
            v-html="getSiteIcon(editingSite.icon)"
          ></div>
        </n-form-item>
        <n-form-item :label="t('dashboard.siteName')" :rule="{ required: true, message: t('dashboard.nameOrUrlRequired') }">
          <n-input v-model:value="editingSite.name" :placeholder="t('dashboard.namePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('dashboard.siteUrl')">
          <n-input v-model:value="editingSite.url" placeholder="https://..." />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="cancelEditSite">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" @click="confirmEditSite">{{ t('common.save') }}</n-button>
      </template>
    </n-modal>
  </AppLayout>

    <!-- Icon Picker Modal -->
    <n-modal v-model:show="showIconPicker" preset="card" :title="t('dashboard.selectIcon')" style="width: 360px">
      <div class="icon-grid">
        <div
          v-for="emoji in presetEmojis"
          :key="emoji"
          class="icon-grid-item"
          @click="selectIcon(emoji)"
        >{{ emoji }}</div>
        <div
          v-for="(svg, i) in uploadedSvgs"
          :key="'svg-' + i"
          class="icon-grid-item svg-item"
          @click="selectIcon(svg)"
        >
          <img v-if="svg.startsWith('/api/icons/')" :src="svg" style="width: 24px; height: 24px" />
          <div v-else v-html="svg" style="width: 24px; height: 24px"></div>
          <div class="svg-delete" @click.stop="removeSvg(i)">✕</div>
        </div>
        <n-upload
          :show-file-list="false"
          accept=".svg"
          @change="handleSvgUpload"
        >
          <div class="icon-grid-item icon-add">+</div>
        </n-upload>
      </div>
    </n-modal>

</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { useKernelStore } from '../stores/kernel'
import { kernelApi } from '../api/kernel'
import { testApi, type TestResult, type TestSite } from '../api/test'
import { useWebSocket } from '../composables/useWebSocket'

const { t } = useI18n()
const message = useMessage()
const kernelStore = useKernelStore()

const closingConnections = ref(false)
const updatingGeo = ref(false)
const restarting = ref(false)
const testingConnectivity = ref(false)
const testingSingle = ref<string | null>(null)

// Test sites management
const DEFAULT_SVG = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="24" height="24"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>'
const DEFAULT_SITES: TestSite[] = [
  { icon: '', name: 'Google', url: 'https://www.google.com/generate_204' },
  { icon: '', name: 'GitHub', url: 'https://api.github.com' },
]
const testSites = ref<TestSite[]>(loadSites())
const manageMode = ref(false)
const showEditModal = ref(false)
const editingSite = ref<TestSite | null>(null)
const editingIndex = ref(-1)

// Icon picker
const showIconPicker = ref(false)
const presetEmojis = ['🔍', '🐙', '🌐', '📺', '🎵', '🎬', '🎮', '📱', '💻', '☁️', '🔒', '📡', '⚡', '🚀', '🌍', '🇺🇸', '🇯🇵', '🇬🇧', '🇩🇪', '🇫🇷', '🇰🇷', '🇨🇳', '🇭🇰', '🇸🇬']
const uploadedSvgs = ref<string[]>(JSON.parse(localStorage.getItem('uploadedSvgs') || '[]'))

// Result map keyed by site name
const resultMap = ref<Record<string, TestResult>>({})

function loadSites(): TestSite[] {
  try {
    const stored = localStorage.getItem('testSites')
    if (stored) return JSON.parse(stored)
  } catch { /* ignore */ }
  return [...DEFAULT_SITES]
}

function saveSites() {
  localStorage.setItem('testSites', JSON.stringify(testSites.value))
}

function getSiteIcon(icon: string): string {
  if (!icon) return DEFAULT_SVG
  if (icon.startsWith('/api/icons/')) {
    return `<img src="${icon}" style="width: 24px; height: 24px" />`
  }
  if (icon.startsWith('<')) return icon
  return `<span style="font-size:24px">${icon}</span>`
}

function addSite() {
  testSites.value.push({ icon: '', name: '', url: '' })
  openEditSite(testSites.value.length - 1)
}

function removeSite(index: number) {
  testSites.value.splice(index, 1)
  saveSites()
}

function openEditSite(index: number) {
  editingIndex.value = index
  editingSite.value = { ...testSites.value[index] }
  showEditModal.value = true
}

function confirmEditSite() {
  if (!editingSite.value) return
  if (!editingSite.value.name && !editingSite.value.url) {
    message.error(t('dashboard.nameOrUrlRequired'))
    return
  }
  if (editingIndex.value >= 0) {
    testSites.value[editingIndex.value] = { ...editingSite.value }
    saveSites()
  }
  showEditModal.value = false
  editingSite.value = null
}
const trafficCanvas = ref<HTMLCanvasElement | null>(null)

// Traffic WebSocket
useWebSocket({
  url: '/api/ws/traffic',
  onMessage: (data: { up: number; down: number }) => {
    kernelStore.updateTraffic(data)
    drawGraph()
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
  nextTick(() => drawGraph())
})

// --- Traffic Graph ---
function drawGraph() {
  const canvas = trafficCanvas.value
  if (!canvas) return

  const container = canvas.parentElement
  if (!container) return

  // Use container's fixed dimensions, not canvas dimensions
  const w = container.clientWidth
  const h = container.clientHeight
  if (w === 0 || h === 0) return

  const dpr = window.devicePixelRatio || 1
  canvas.width = w * dpr
  canvas.height = h * dpr

  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.scale(dpr, dpr)

  const pad = { top: 8, bottom: 4, left: 0, right: 0 }
  const plotW = w - pad.left - pad.right
  const plotH = h - pad.top - pad.bottom

  ctx!.clearRect(0, 0, w, h)

  const allVals = [...kernelStore.upHistory, ...kernelStore.downHistory]
  const maxVal = Math.max(...allVals, 1024) // min scale 1KB

  // Draw grid lines
  ctx!.strokeStyle = 'rgba(128,128,128,0.1)'
  ctx!.lineWidth = 0.5
  for (let i = 0; i <= 3; i++) {
    const y = pad.top + (plotH / 3) * i
    ctx!.beginPath()
    ctx!.moveTo(pad.left, y)
    ctx!.lineTo(pad.left + plotW, y)
    ctx!.stroke()
  }

  // Draw lines
  function drawLine(data: number[], color: string) {
    if (data.length < 2 || !ctx) return
    ctx.beginPath()
    ctx.strokeStyle = color
    ctx.lineWidth = 1.5
    ctx.lineJoin = 'round'

    for (let i = 0; i < data.length; i++) {
      const x = pad.left + (plotW / 59) * (60 - data.length + i)
      const y = pad.top + plotH - (data[i] / maxVal) * plotH
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.stroke()

    // Fill area
    const lastX = pad.left + plotW
    const firstX = pad.left + (plotW / 59) * (60 - data.length)
    ctx.lineTo(lastX, pad.top + plotH)
    ctx.lineTo(firstX, pad.top + plotH)
    ctx.closePath()
    ctx.fillStyle = color.replace('1)', '0.08)')
    ctx.fill()
  }

  drawLine(kernelStore.downHistory, 'rgba(99, 226, 183, 1)')  // green for download
  drawLine(kernelStore.upHistory, 'rgba(242, 201, 125, 1)')    // yellow for upload
}

// --- Actions ---
async function handleModeChange(mode: string) {
  try {
    await kernelStore.switchMode(mode)
    message.success(t('dashboard.modeSwitched', { mode }))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  }
}

async function handleIPv6Change(val: boolean) {
  try {
    await kernelStore.toggleIPv6(val)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  }
}

async function handleAllowLanChange(val: boolean) {
  try {
    await kernelStore.toggleAllowLan(val)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  }
}

async function handleTunChange(val: boolean) {
  try {
    await kernelStore.toggleTun(val)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  }
}

async function handleCloseAll() {
  closingConnections.value = true
  try {
    await kernelApi.closeAllConnections()
    message.success(t('dashboard.allConnectionsClosed'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  } finally {
    closingConnections.value = false
  }
}

async function handleUpdateGeo() {
  updatingGeo.value = true
  try {
    await Promise.all([
      kernelApi.updateGeo(),
      new Promise(resolve => setTimeout(resolve, 1500)),
    ])
    message.success(t('dashboard.geoUpdated'))
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  } finally {
    updatingGeo.value = false
  }
}

async function handleRestart() {
  restarting.value = true
  try {
    await kernelApi.restart()
    message.success(t('dashboard.kernelRestarting'))
    setTimeout(async () => {
      await kernelStore.initialize()
      restarting.value = false
    }, 5000)
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
    restarting.value = false
  }
}

async function handleTestAll() {
  testingConnectivity.value = true
  resultMap.value = {}
  try {
    const results = await testApi.testAll(testSites.value)
    const map: Record<string, TestResult> = {}
    for (const r of results) {
      map[r.name] = r
    }
    resultMap.value = map
  } catch (err: any) {
    message.error(t('common.failed') + ': ' +  + (err.message || err))
  } finally {
    testingConnectivity.value = false
  }
}

async function handleTestSingle(site: TestSite) {
  if (testingConnectivity.value || testingSingle.value) return
  testingSingle.value = site.name
  try {
    const results = await testApi.testAll([site])
    if (results.length > 0) {
      resultMap.value = { ...resultMap.value, [results[0].name]: results[0] }
    }
  } catch (err: any) {
    message.error(site.name + ': ' + t('dashboard.testFailed'))
  } finally {
    testingSingle.value = null
  }
}

async function handleSvgUpload({ file }: any) {
  if (!file?.file) return

  const formData = new FormData()
  formData.append('file', file.file)

  try {
    const token = localStorage.getItem('token') || ''
    const resp = await fetch('/api/icons/upload', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` },
      body: formData,
    })

    if (!resp.ok) {
      throw new Error('Upload failed')
    }

    const data = await resp.json()
    const iconUrl = data.url

    // Store the URL reference in localStorage
    uploadedSvgs.value.push(iconUrl)
    localStorage.setItem('uploadedSvgs', JSON.stringify(uploadedSvgs.value))

    // Select it
    if (editingSite.value) {
      editingSite.value.icon = iconUrl
    }
    showIconPicker.value = false
  } catch (err) {
    message.error('上传失败')
  }
}

function selectIcon(icon: string) {
  if (editingSite.value) {
    editingSite.value.icon = icon
  }
  showIconPicker.value = false
}

async function removeSvg(index: number) {
  const iconUrl = uploadedSvgs.value[index]
  
  // Delete from backend if it's a server-stored icon
  if (iconUrl && iconUrl.startsWith('/api/icons/')) {
    const filename = iconUrl.split('/').pop()
    try {
      const token = localStorage.getItem('token') || ''
      await fetch(`/api/icons?file=${filename}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      })
    } catch (err) {
      console.error('Failed to delete icon from server:', err)
    }
  }
  
  // Remove from localStorage
  uploadedSvgs.value.splice(index, 1)
  localStorage.setItem('uploadedSvgs', JSON.stringify(uploadedSvgs.value))
}

function cancelEditSite() {
  // If new site with empty name AND url, remove it
  if (editingSite.value && !editingSite.value.name && !editingSite.value.url && editingIndex.value >= 0) {
    testSites.value.splice(editingIndex.value, 1)
  }
  showEditModal.value = false
  editingSite.value = null
}

function handleEditModalClose(val: boolean) {
  if (!val) cancelEditSite()
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k))
  const val = parseFloat((bytes / Math.pow(k, i)).toFixed(1))
  return val + ' ' + sizes[i]
}

function formatSpeed(bytesPerSec: number): string {
  if (!bytesPerSec || bytesPerSec === 0) return '0 B'
  return formatBytes(bytesPerSec)
}
</script>

<style scoped>
.site-card {
  position: relative;
  text-align: center;
  padding: 10px 6px 8px;
  border-radius: 8px;
  border: 1px solid rgba(128,128,128,0.15);
  cursor: pointer;
  transition: all 0.2s;
  height: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.site-card:hover { border-color: rgba(128,128,128,0.3); }
.site-card.site-ok { border-color: rgba(24,160,88,0.5); background: rgba(24,160,88,0.06); }
.site-card.site-fail { border-color: rgba(208,48,80,0.4); background: rgba(208,48,80,0.06); }
.site-card.site-manage { border-style: dashed; cursor: pointer; }
.site-card.site-add { border-style: dashed; border-color: rgba(99,226,183,0.4); opacity: 0.7; }
.site-card.site-add:hover { opacity: 1; }
.site-delete {
  position: absolute;
  top: 2px;
  right: 4px;
  width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  font-size: 12px;
  color: #e88080;
  cursor: pointer;
  border-radius: 50%;
  z-index: 1;
}
.site-delete:hover { background: rgba(232,128,128,0.15); }
.site-icon { height: 28px; display: flex; align-items: center; justify-content: center; margin-bottom: 4px; color: #aaa; }
.site-icon :deep(svg) { width: 24px; height: 24px; }
.site-name { font-size: 11px; color: #ccc; margin-bottom: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.site-latency { font-size: 11px; font-weight: 600; color: #63e2b7; }
.site-latency.timeout { color: #999; }
.site-latency.idle { color: #555; }
.site-card.site-loading { opacity: 0.6; pointer-events: none; }

.icon-picker-trigger {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed rgba(128,128,128,0.3);
  border-radius: 6px;
  cursor: pointer;
  font-size: 24px;
  transition: all 0.2s;
}
.icon-picker-trigger:hover {
  border-color: rgba(99,226,183,0.5);
  background: rgba(99,226,183,0.05);
}
.icon-picker-trigger :deep(svg) {
  width: 24px;
  height: 24px;
}
.icon-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
}
.icon-grid-item {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(128,128,128,0.15);
  border-radius: 6px;
  cursor: pointer;
  font-size: 20px;
  transition: all 0.15s;
  position: relative;
}
.icon-grid-item:hover {
  border-color: rgba(99,226,183,0.5);
  background: rgba(99,226,183,0.08);
  transform: scale(1.1);
}
.icon-grid-item :deep(svg) {
  width: 20px;
  height: 20px;
}
.icon-add {
  font-size: 18px;
  color: #63e2b7;
  border-style: dashed;
}
.svg-item .svg-delete {
  display: none;
  position: absolute;
  top: -4px;
  right: -4px;
  width: 14px;
  height: 14px;
  line-height: 14px;
  text-align: center;
  font-size: 10px;
  color: #fff;
  background: #e88080;
  border-radius: 50%;
  cursor: pointer;
}
.svg-item:hover .svg-delete {
  display: block;
}
</style>
