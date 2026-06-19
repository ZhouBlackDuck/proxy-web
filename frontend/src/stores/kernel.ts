import { defineStore } from 'pinia'
import { ref } from 'vue'
import { kernelApi, type KernelVersion, type KernelConfig } from '../api/kernel'

const MAX_HISTORY = 60

export const useKernelStore = defineStore('kernel', () => {
  const version = ref<KernelVersion | null>(null)
  const config = ref<KernelConfig | null>(null)
  const alive = ref(false)
  const traffic = ref({ up: 0, down: 0 })
  const memory = ref({ inuse: 0 })
  const totalUp = ref(0)
  const totalDown = ref(0)

  // Traffic history for graph (persists across page navigations)
  const upHistory = ref<number[]>([])
  const downHistory = ref<number[]>([])

  async function fetchVersion() {
    try {
      version.value = await kernelApi.getVersion()
      alive.value = true
    } catch {
      alive.value = false
    }
  }

  async function fetchConfig() {
    try {
      config.value = await kernelApi.getConfigs()
    } catch {
      // ignore
    }
  }

  function updateTraffic(data: { up: number; down: number }) {
    traffic.value.up = data.up
    traffic.value.down = data.down
    totalUp.value += data.up
    totalDown.value += data.down

    upHistory.value.push(data.up)
    downHistory.value.push(data.down)
    if (upHistory.value.length > MAX_HISTORY) upHistory.value.shift()
    if (downHistory.value.length > MAX_HISTORY) downHistory.value.shift()
  }

  function updateMemory(data: { inuse: number }) {
    memory.value.inuse = data.inuse
  }

  async function switchMode(mode: string) {
    await kernelApi.patchConfig({ mode })
    if (config.value) {
      config.value.mode = mode
    }
  }

  async function toggleIPv6(ipv6: boolean) {
    await kernelApi.patchConfig({ ipv6 })
    if (config.value) {
      config.value.ipv6 = ipv6
    }
  }

  async function toggleAllowLan(allowLan: boolean) {
    await kernelApi.patchConfig({ 'allow-lan': allowLan })
    if (config.value) {
      config.value['allow-lan'] = allowLan
    }
  }

  async function toggleTun(enable: boolean) {
    await kernelApi.patchConfig({ tun: { enable } })
    if (config.value) {
      config.value.tun.enable = enable
    }
  }

  async function initialize() {
    await Promise.all([fetchVersion(), fetchConfig()])
  }

  return {
    version,
    config,
    alive,
    traffic,
    memory,
    totalUp,
    totalDown,
    upHistory,
    downHistory,
    fetchVersion,
    fetchConfig,
    updateTraffic,
    updateMemory,
    switchMode,
    toggleIPv6,
    toggleAllowLan,
    toggleTun,
    initialize,
  }
})
