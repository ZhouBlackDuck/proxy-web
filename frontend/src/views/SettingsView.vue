<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-text strong style="font-size: 18px">{{ t('settings.title') }}</n-text>

      <n-grid :x-gap="16" :y-gap="16" :cols="2" responsive="screen" item-responsive style="align-items: stretch">
        <!-- General Settings -->
        <n-gi span="2 m:1">
          <n-card :title="t('settings.title')" size="small" style="height: 100%">
            <n-form label-placement="left" label-width="100">
              <n-form-item :label="t('settings.theme')">
                <n-switch
                  :value="settingsStore.theme === 'dark'"
                  @update:value="(v: boolean) => settingsStore.setTheme(v ? 'dark' : 'light')"
                >
                  <template #checked>{{ t('settings.dark') }}</template>
                  <template #unchecked>{{ t('settings.light') }}</template>
                </n-switch>
              </n-form-item>
              <n-form-item :label="t('settings.language')">
                <n-select
                  :value="settingsStore.language"
                  :options="languageOptions"
                  @update:value="handleLanguageChange"
                  style="width: 200px"
                />
              </n-form-item>
            </n-form>
          </n-card>
        </n-gi>

        <!-- Password Change -->
        <n-gi span="2 m:1">
          <n-card :title="t('settings.changePassword')" size="small" style="height: 100%">
            <n-form label-placement="left" label-width="100">
              <n-form-item :label="t('auth.oldPassword')">
                <n-input v-model:value="oldPassword" type="password" show-password-on="click" :placeholder="t('auth.oldPasswordPlaceholder')" />
              </n-form-item>
              <n-form-item :label="t('auth.newPassword')">
                <n-input v-model:value="newPassword" type="password" show-password-on="click" :placeholder="t('auth.newPasswordPlaceholder')" />
              </n-form-item>
              <n-form-item>
                <n-button type="primary" :loading="changingPassword" @click="handleChangePassword">
                  {{ t('settings.changePassword') }}
                </n-button>
              </n-form-item>
            </n-form>
          </n-card>
        </n-gi>

        <!-- Process Management -->
        <n-gi span="2 m:1">
          <n-card :title="t('settings.processManagement')" size="small" style="height: 100%">
            <n-spin :show="processLoading">
              <n-space vertical :size="12">
                <n-space v-for="proc in processes" :key="proc.name" justify="space-between" align="center">
                  <n-space align="center">
                    <n-tag :type="proc.running ? 'success' : 'error'" size="small">
                      {{ proc.running ? t('dashboard.running') : t('dashboard.stopped') }}
                    </n-tag>
                    <n-text>{{ proc.name === 'mihomo' ? t('settings.mihomo') : t('settings.subconverter') }}</n-text>
                    <n-text v-if="proc.running" depth="3" style="font-size: 12px">
                      PID: {{ proc.pid }} | {{ formatUptime(proc.uptime) }}
                    </n-text>
                  </n-space>
                  <n-space :size="4">
                    <n-button v-if="!proc.running" size="tiny" type="primary" :loading="processLoading === proc.name + ':start'" @click="startProcess(proc.name)">
                      {{ t('settings.start') }}
                    </n-button>
                    <n-button v-if="proc.running" size="tiny" type="warning" :loading="processLoading === proc.name + ':restart'" @click="restartProcess(proc.name)">
                      {{ t('settings.restart') }}
                    </n-button>
                    <n-button v-if="proc.running" size="tiny" type="error" :loading="processLoading === proc.name + ':stop'" @click="stopProcess(proc.name)">
                      {{ t('settings.stop') }}
                    </n-button>
                  </n-space>
                </n-space>
                <n-button size="small" @click="fetchProcesses">
                  {{ t('common.refresh') }}
                </n-button>
              </n-space>
            </n-spin>
          </n-card>
        </n-gi>

        <!-- GeoIP Management -->
        <n-gi span="2 m:1">
          <n-card :title="t('settings.geoManagement')" size="small" style="height: 100%">
            <n-space vertical :size="12">
              <n-descriptions :column="1" label-placement="left" size="small" v-if="geoStatus.length > 0">
                <n-descriptions-item v-for="f in geoStatus" :key="f.name" :label="f.name">
                  <template v-if="f.exists">
                    <n-text>{{ formatFileSize(f.size) }}</n-text>
                    <n-text depth="3" style="margin-left: 8px; font-size: 11px">{{ formatDateTime(f.updatedAt) }}</n-text>
                  </template>
                  <n-text v-else type="error">{{ t('settings.missing') }}</n-text>
                </n-descriptions-item>
              </n-descriptions>
              <n-button
                type="primary"
                :loading="updatingGeo"
                @click="handleUpdateGeo"
                size="small"
              >
                {{ updatingGeo ? t('settings.geoUpdating') : t('settings.updateGeo') }}
              </n-button>
            </n-space>
          </n-card>
        </n-gi>

        <!-- Log Level -->
        <n-gi span="2 m:1">
          <n-card :title="t('settings.logLevel')" size="small" style="height: 100%">
            <n-space vertical :size="12">
              <n-text depth="3">
                {{ t('settings.logLevelHint') }}
              </n-text>
              <n-space align="center" :size="8">
                <n-select
                  v-model:value="kernelLogLevel"
                  :options="logLevelOptions"
                  style="width: 140px"
                  size="small"
                />
                <n-button size="small" type="primary" :loading="savingLogLevel" @click="handleSaveLogLevel">
                  {{ t('common.save') }}
                </n-button>
              </n-space>
            </n-space>
          </n-card>
        </n-gi>

        <!-- Port Management -->
        <n-gi span="2">
          <n-card :title="t('settings.portManagement')" size="small" style="height: 100%">
            <n-spin :show="portLoading">
              <n-space vertical :size="12">
                <n-space v-for="p in portList" :key="p.key" justify="space-between" align="center">
                  <n-space align="center" :size="8">
                    <n-switch
                      v-model:value="p.enabled"
                      size="small"
                    />
                    <n-text>{{ p.label }}</n-text>
                  </n-space>
                  <n-input-number
                    v-model:value="p.value"
                    :min="1" :max="65535"
                    :disabled="!p.enabled"
                    style="width: 120px"
                    size="small"
                  />
                </n-space>
              </n-space>
              <n-button type="primary" size="small" style="margin-top: 12px" :loading="savingPorts" @click="handleSavePorts">
                {{ t('common.save') }}
              </n-button>
            </n-spin>
          </n-card>
        </n-gi>
      </n-grid>
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { useSettingsStore } from '../stores/settings'
import { authApi } from '../api/auth'
import { kernelApi } from '../api/kernel'
import { profileApi } from '../api/profiles'
import client from '../api/client'

const { t, locale } = useI18n()
const message = useMessage()
const settingsStore = useSettingsStore()

const oldPassword = ref('')
const newPassword = ref('')
const changingPassword = ref(false)

const processLoading = ref<string | null>(null)
const processes = ref<any[]>([])

const updatingGeo = ref(false)
const geoStatus = ref<{ name: string; exists: boolean; size: number; updatedAt: string }[]>([])

const kernelLogLevel = ref('info')
const savingLogLevel = ref(false)
const logLevelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warning' },
  { label: 'Error', value: 'error' },
  { label: 'Silent', value: 'silent' },
]

const portLoading = ref(false)
const savingPorts = ref(false)
interface PortItem {
  key: string
  label: string
  configKey: string
  value: number
  enabled: boolean
  defaultPort: number
}

const portList = reactive<PortItem[]>([
  { key: 'mixed', label: 'Mixed (HTTP+SOCKS)', configKey: 'mixed-port', value: 7890, enabled: true, defaultPort: 7890 },
  { key: 'http', label: 'HTTP', configKey: 'port', value: 0, enabled: false, defaultPort: 7891 },
  { key: 'socks', label: 'SOCKS', configKey: 'socks-port', value: 0, enabled: false, defaultPort: 7892 },
  { key: 'redir', label: 'Redir', configKey: 'redir-port', value: 0, enabled: false, defaultPort: 7893 },
  { key: 'tproxy', label: 'TProxy', configKey: 'tproxy-port', value: 0, enabled: false, defaultPort: 7894 },
])

const languageOptions = [
  { label: '中文', value: 'zh' },
  { label: 'English', value: 'en' },
]

onMounted(() => {
  fetchProcesses()
  fetchPorts()
  fetchLogLevel()
  fetchGeoStatus()
})

function handleLanguageChange(lang: string) {
  settingsStore.setLanguage(lang)
  locale.value = lang
}

async function handleChangePassword() {
  if (!oldPassword.value || !newPassword.value) {
    message.error(t('auth.passwordMinLength'))
    return
  }
  changingPassword.value = true
  try {
    await authApi.changePassword(oldPassword.value, newPassword.value)
    message.success(t('auth.passwordChanged'))
    oldPassword.value = ''
    newPassword.value = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || t('common.failed'))
  } finally {
    changingPassword.value = false
  }
}

async function fetchProcesses() {
  try {
    const { data } = await client.get('/status')
    processes.value = data.processes || []
  } catch {
    // ignore
  }
}

async function startProcess(name: string) {
  processLoading.value = `${name}:start`
  try {
    await client.post(`/process/${name}/start`)
    message.success(t('settings.started'))
    await fetchProcesses()
  } catch (err: any) {
    message.error(err.response?.data?.error || t('common.failed'))
  } finally {
    processLoading.value = null
  }
}

async function stopProcess(name: string) {
  processLoading.value = `${name}:stop`
  try {
    await client.post(`/process/${name}/stop`)
    message.success(t('settings.stopped'))
    await fetchProcesses()
  } catch (err: any) {
    message.error(err.response?.data?.error || t('common.failed'))
  } finally {
    processLoading.value = null
  }
}

async function restartProcess(name: string) {
  processLoading.value = `${name}:restart`
  try {
    await client.post(`/process/${name}/restart`)
    message.success(t('settings.restarting'))
    setTimeout(() => fetchProcesses(), 5000)
  } catch (err: any) {
    message.error(err.response?.data?.error || t('common.failed'))
  } finally {
    processLoading.value = null
  }
}

async function handleUpdateGeo() {
  updatingGeo.value = true
  try {
    await kernelApi.updateGeo()
    message.success(t('dashboard.geoUpdated'))
    await fetchGeoStatus()
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    updatingGeo.value = false
  }
}

async function fetchGeoStatus() {
  try {
    const { data } = await client.get('/geo/status')
    geoStatus.value = data.files || []
  } catch {
    // ignore
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDateTime(iso: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

async function fetchLogLevel() {
  try {
    const config = await kernelApi.getConfigs()
    kernelLogLevel.value = config['log-level'] || 'info'
  } catch {
    // ignore
  }
}

async function handleSaveLogLevel() {
  savingLogLevel.value = true
  try {
    await kernelApi.patchConfig({ 'log-level': kernelLogLevel.value })
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    savingLogLevel.value = false
  }
}

async function fetchPorts() {
  portLoading.value = true
  try {
    const data = await profileApi.getPorts()
    const mapping: Record<string, { enabled: boolean; port: number }> = {
      mixed: data.mixedPort,
      http: data.httpPort,
      socks: data.socksPort,
      redir: data.redirPort,
      tproxy: data.tproxyPort,
    }
    for (const p of portList) {
      const entry = mapping[p.key]
      if (entry) {
        p.enabled = entry.enabled
        p.value = entry.port || p.defaultPort
      }
    }
  } catch {
    // ignore
  } finally {
    portLoading.value = false
  }
}

async function handleSavePorts() {
  savingPorts.value = true
  try {
    const payload = {
      mixedPort: { enabled: portList[0].enabled, port: portList[0].value },
      httpPort: { enabled: portList[1].enabled, port: portList[1].value },
      socksPort: { enabled: portList[2].enabled, port: portList[2].value },
      redirPort: { enabled: portList[3].enabled, port: portList[3].value },
      tproxyPort: { enabled: portList[4].enabled, port: portList[4].value },
    }
    await profileApi.updatePorts(payload)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    savingPorts.value = false
  }
}

function formatUptime(seconds: number): string {
  if (!seconds) return ''
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}
</script>
