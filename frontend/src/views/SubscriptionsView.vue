<template>
  <AppLayout>
    <n-space vertical :size="16">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">{{ t('subscriptions.title') }}</n-text>
        <n-space :size="8">
          <n-space align="center" :size="4">
            <n-text depth="3" style="font-size: 12px">{{ t('subscriptions.exportWithSubs') }}</n-text>
            <n-switch v-model:value="exportIncludeSubs" size="small" @update:value="handleExportSettingChange" />
          </n-space>
          <n-button size="small" @click="handleExportAll">{{ t('subscriptions.export') }}</n-button>
          <n-upload :show-file-list="false" accept=".zip" @change="handleImportFile">
            <n-button size="small">{{ t('subscriptions.import') }}</n-button>
          </n-upload>
          <n-button type="primary" size="small" @click="showCreateModal = true">{{ t('subscriptions.addSubscription') }}</n-button>
        </n-space>
      </n-space>

      <!-- Subscription Cards -->
      <n-spin :show="loading">
        <n-empty v-if="subscriptions.length === 0 && !loading" :description="t('subscriptions.noSubscriptions')" />
        <n-grid v-else :x-gap="12" :y-gap="12" :cols="3" responsive="screen" item-responsive>
          <n-gi v-for="sub in subscriptions" :key="sub.name" span="3 m:1">
            <n-card
              size="small"
              hoverable
              style="cursor: pointer; height: 100%"
              :style="sub.name === activeSub ? 'border-left: 3px solid #18a058' : ''"
              @click="handleActivate(sub.name)"
            >
              <template #header>
                <n-space align="center" :size="8">
                  <n-tag v-if="sub.name === activeSub" type="success" size="tiny">{{ t('subscriptions.current') }}</n-tag>
                  <n-tag size="tiny" :type="sub.url ? 'info' : 'default'">
                    {{ sub.url ? 'URL' : t('subscriptions.local') }}
                  </n-tag>
                  <n-ellipsis :line-clamp="1" style="max-width: 160px">
                    <n-text strong>{{ sub.displayName || sub.name }}</n-text>
                  </n-ellipsis>
                </n-space>
              </template>
              <template #header-extra>
                <n-space :size="4" @click.stop>
                  <n-button v-if="sub.url" size="tiny" quaternary @click="handleSync(sub.name)" :loading="syncing === sub.name">{{ t('subscriptions.sync') }}</n-button>
                  <n-dropdown :options="getMoreOptions(sub)" @select="(k: string) => handleMore(k, sub)">
                    <n-button size="tiny" quaternary>⋯</n-button>
                  </n-dropdown>
                </n-space>
              </template>
              <n-descriptions :column="1" label-placement="left" size="small">
                <n-descriptions-item v-if="sub.url" label="URL">
                  <n-ellipsis :line-clamp="1" style="max-width: 200px">{{ sub.url }}</n-ellipsis>
                </n-descriptions-item>
                <n-descriptions-item v-if="sub.updatedAt" :label="t('subscriptions.updatedAt')">
                  {{ formatTime(sub.updatedAt) }}
                </n-descriptions-item>
              </n-descriptions>
              <template #action>
                <n-space :size="6">
                  <n-button size="tiny" :loading="activating === sub.name" @click.stop="handleActivate(sub.name)">
                    {{ sub.name === activeSub ? t('subscriptions.reactivate') : t('subscriptions.activate') }}
                  </n-button>
                  <n-button size="tiny" @click.stop="handlePreview(sub.name)">{{ t('subscriptions.preview') }}</n-button>
                </n-space>
              </template>
            </n-card>
          </n-gi>
        </n-grid>
      </n-spin>

      <!-- Global Rules Section -->
      <n-card :title="t('subscriptions.globalRules')" size="small">
        <template #header-extra>
          <n-button size="tiny" type="primary" :loading="savingRules" @click="handleSaveRules">{{ t('common.save') }}</n-button>
        </template>
        <n-alert type="info" style="margin-bottom: 8px" :show-icon="false">
          {{ t('subscriptions.globalRulesHint') }}
        </n-alert>
        <n-input v-model:value="globalRules" type="textarea" :rows="6" placeholder="# 全局规则&#10;DOMAIN-KEYWORD,google,Proxy&#10;GEOIP,CN,DIRECT" style="font-family: monospace; font-size: 12px" />
      </n-card>

      <!-- Global Override Section -->
      <n-card :title="t('subscriptions.globalOverride')" size="small">
        <template #header-extra>
          <n-button size="tiny" type="primary" :loading="savingOverride" @click="handleSaveOverride">{{ t('common.save') }}</n-button>
        </template>
        <n-alert type="warning" style="margin-bottom: 8px" :show-icon="false">
          {{ t('subscriptions.overrideHint1') }}
          <strong>{{ t('subscriptions.overrideHint2') }}</strong>
          {{ t('subscriptions.overrideHint3') }}
        </n-alert>
        <n-input v-model:value="globalOverride" type="textarea" :rows="8" placeholder="# 全局配置覆盖（YAML）&#10;dns:&#10;  enable: true&#10;  enhanced-mode: fake-ip&#10;sniffer:&#10;  enable: true" style="font-family: monospace; font-size: 12px" />
      </n-card>
    </n-space>

    <!-- Create/Edit Subscription Modal -->
    <n-modal v-model:show="showCreateModal" preset="dialog" :title="editingSub ? t('subscriptions.editSub') : t('subscriptions.addSub')" style="width: 560px">
      <n-form :model="form" label-placement="left" label-width="80">
        <n-form-item :label="t('subscriptions.name')">
          <n-input v-model:value="form.name" :placeholder="t('subscriptions.subNamePlaceholder')" :disabled="!!editingSub" />
        </n-form-item>
        <n-form-item :label="t('subscriptions.displayName')">
          <n-input v-model:value="form.displayName" :placeholder="t('subscriptions.displayNamePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('subscriptions.type')">
          <n-radio-group v-model:value="form.source" :disabled="!!editingSub">
            <n-radio-button value="url">{{ t('subscriptions.urlSubscription') }}</n-radio-button>
            <n-radio-button value="local">{{ t('subscriptions.localConfig') }}</n-radio-button>
          </n-radio-group>
        </n-form-item>
        <n-form-item v-if="form.source === 'url'" :label="t('subscriptions.subscriptionUrl')">
          <n-input v-model:value="form.url" type="textarea" placeholder="https://..." :rows="3" />
        </n-form-item>
        <n-form-item v-if="form.source === 'local'" :label="t('subscriptions.configContent')">
          <n-space vertical style="width: 100%">
            <n-upload :show-file-list="false" accept=".yaml,.yml,.txt,.conf" @change="handleFileUpload">
              <n-button size="small">{{ t('subscriptions.uploadFile') }}</n-button>
            </n-upload>
            <n-input v-model:value="form.content" type="textarea" :placeholder="t('subscriptions.contentPlaceholder')" :rows="10" style="font-family: monospace; font-size: 12px" />
          </n-space>
        </n-form-item>
        <n-form-item v-if="form.source === 'url'" label="User-Agent">
          <n-input v-model:value="form.ua" :placeholder="t('subscriptions.uaPlaceholder')" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showCreateModal = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">{{ t('common.save') }}</n-button>
      </template>
    </n-modal>

    <!-- Preview Modal -->
    <n-modal v-model:show="showPreviewModal" preset="card" :title="t('subscriptions.mergedPreview')" style="width: 700px; max-width: 90vw">
      <n-spin :show="previewLoading">
        <n-code :code="previewContent" language="yaml" word-wrap />
      </n-spin>
    </n-modal>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage, useDialog, type UploadFileInfo } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import AppLayout from '../components/layout/AppLayout.vue'
import { subscriptionApi, type Subscription } from '../api/subscriptions'
import client from '../api/client'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const saving = ref(false)
const syncing = ref<string | null>(null)
const activating = ref<string | null>(null)
const subscriptions = ref<Subscription[]>([])
const activeSub = ref('')
const exportIncludeSubs = ref(false)

const showCreateModal = ref(false)
const showPreviewModal = ref(false)
const editingSub = ref<Subscription | null>(null)
const previewContent = ref('')
const previewLoading = ref(false)
const savingRules = ref(false)
const savingOverride = ref(false)

const globalRules = ref('')
const globalOverride = ref('')

const form = ref({
  name: '',
  displayName: '',
  source: 'url' as 'url' | 'local',
  url: '',
  content: '',
  ua: '',
})

onMounted(() => {
  fetchAll()
})

async function fetchAll() {
  loading.value = true
  try {
    const [subs, activeResp] = await Promise.all([
      subscriptionApi.list(),
      client.get('/subscriptions/active'),
    ])
    subscriptions.value = subs
    activeSub.value = activeResp.data.activeSubscription || ''

    // Fetch global rules and override (stored under __global__ key)
    try {
      const [rulesResp, overrideResp] = await Promise.all([
        client.get('/subscriptions/__global__/rules'),
        client.get('/subscriptions/__global__/override'),
      ])
      globalRules.value = typeof rulesResp.data === 'string' ? rulesResp.data : ''
      globalOverride.value = typeof overrideResp.data === 'string' ? overrideResp.data : ''
    } catch { /* first time */ }

    try {
      const settingResp = await client.get('/config/export-setting')
      exportIncludeSubs.value = settingResp.data.includeSubscriptions || false
    } catch { /* ignore */ }
  } catch (err: any) {
    message.error(t('subscriptions.dataLoadFailed') + ': ' + (err.message || err))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { name: '', displayName: '', source: 'url', url: '', content: '', ua: '' }
  editingSub.value = null
}

function handleFileUpload({ file }: { file: UploadFileInfo }) {
  if (!file?.file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    form.value.content = e.target?.result as string || ''
    message.success(t('subscriptions.fileLoaded') + ': ' + file.name)
  }
  reader.onerror = () => message.error(t('subscriptions.fileReadFailed'))
  reader.readAsText(file.file)
}

function getMoreOptions(_sub: Subscription) {
  return [
    { label: t('subscriptions.editSubLabel'), key: 'edit' },
    { label: t('subscriptions.deleteLabel'), key: 'delete' },
  ]
}

function handleEditSub(sub: Subscription) {
  editingSub.value = sub
  form.value = {
    name: sub.name,
    displayName: sub.displayName || '',
    source: sub.url ? 'url' : 'local',
    url: sub.url || '',
    content: sub.content || '',
    ua: sub.ua || '',
  }
  showCreateModal.value = true
}

async function handleSave() {
  if (!form.value.name) { message.error(t('subscriptions.nameRequired')); return }
  saving.value = true
  try {
    if (editingSub.value) {
      const payload: Record<string, any> = { displayName: form.value.displayName, ua: form.value.ua || undefined }
      if (form.value.source === 'local') {
        payload.source = 'local'
        payload.content = form.value.content
      } else {
        payload.url = form.value.url
      }
      await subscriptionApi.update(editingSub.value.name, payload)
      message.success(t('subscriptions.subUpdated'))
    } else {
      const payload: Record<string, any> = { name: form.value.name, displayName: form.value.displayName, ua: form.value.ua || undefined }
      if (form.value.source === 'local') {
        payload.source = 'local'
        payload.content = form.value.content
      } else {
        payload.url = form.value.url
      }
      await subscriptionApi.create(payload)
      message.success(t('subscriptions.subCreated'))
    }
    showCreateModal.value = false
    resetForm()
    await fetchAll()
  } catch (err: any) {
    message.error(t('subscriptions.saveFailed') + ': ' + (err.response?.data?.error || err.message || err))
  } finally {
    saving.value = false
  }
}

async function handleActivate(name: string) {
  activating.value = name
  try {
    await client.post(`/subscriptions/${encodeURIComponent(name)}/activate`)
    message.success(t('subscriptions.activated') + ': ' + name)
    activeSub.value = name
  } catch (err: any) {
    message.error(t('subscriptions.activateFailed') + ': ' + (err.response?.data?.error || err.message || err))
  } finally {
    activating.value = null
  }
}

function handleMore(key: string, sub: Subscription) {
  if (key === 'edit') {
    handleEditSub(sub)
  } else if (key === 'delete') {
    dialog.warning({
      title: t('common.confirm'),
      content: t('subscriptions.confirmDelete', { name: sub.displayName || sub.name }),
      positiveText: t('common.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        try {
          const wasActive = sub.name === activeSub.value
          await subscriptionApi.delete(sub.name)
          message.success(t('subscriptions.deleted'))
          await fetchAll()
          // If deleted sub was active, activate another one
          if (wasActive) {
            const remaining = subscriptions.value.filter(s => s.name !== sub.name)
            if (remaining.length > 0) {
              await handleActivate(remaining[0].name)
            } else {
              // No subs left, clear kernel state
              try {
                await client.post('/subscriptions/__empty__/activate')
                activeSub.value = ''
              } catch { /* ignore */ }
            }
          }
        } catch (err: any) {
          message.error(t('common.failed') + ': ' + (err.message || err))
        }
      },
    })
  }
}

async function handleSync(name: string) {
  syncing.value = name
  try {
    await subscriptionApi.sync(name)
    message.success(t('subscriptions.syncTriggered'))
    setTimeout(() => fetchAll(), 3000)
  } catch (err: any) {
    message.error(t('subscriptions.syncFailed') + ': ' + (err.message || err))
  } finally {
    syncing.value = null
  }
}

async function handleSaveRules() {
  savingRules.value = true
  try {
    await client.put('/subscriptions/__global__/rules', { content: globalRules.value })
    message.success(t('subscriptions.rulesSaved'))
    // Re-activate current subscription to apply changes
    if (activeSub.value) {
      await client.post(`/subscriptions/${encodeURIComponent(activeSub.value)}/activate`)
    }
  } catch (err: any) {
    message.error(t('subscriptions.saveFailed') + ': ' + (err.message || err))
  } finally {
    savingRules.value = false
  }
}

async function handleSaveOverride() {
  savingOverride.value = true
  try {
    await client.put('/subscriptions/__global__/override', { content: globalOverride.value })
    message.success(t('subscriptions.overrideSaved'))
    // Re-activate current subscription to apply changes
    if (activeSub.value) {
      await client.post(`/subscriptions/${encodeURIComponent(activeSub.value)}/activate`)
    }
  } catch (err: any) {
    message.error(t('subscriptions.saveFailed') + ': ' + (err.message || err))
  } finally {
    savingOverride.value = false
  }
}

async function handlePreview(name: string) {
  showPreviewModal.value = true
  previewLoading.value = true
  previewContent.value = ''
  try {
    const { data } = await client.get(`/subscriptions/${encodeURIComponent(name)}/preview`)
    previewContent.value = typeof data === 'string' ? data : JSON.stringify(data, null, 2)
  } catch (err: any) {
    previewContent.value = '# Preview failed\n# ' + (err.response?.data?.error || err.message || err)
  } finally {
    previewLoading.value = false
  }
}

async function handleExportAll() {
  try {
    // Get test sites from DashboardView localStorage
    const testSites = JSON.parse(localStorage.getItem('testSites') || '[]')
    const resp = await client.post('/config/export', { testSites }, { responseType: 'blob' })
    const blob = new Blob([resp.data])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `proxy-web-config-${new Date().toISOString().slice(0,10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(t('subscriptions.exportFailed') + ': ' + (err.message || err))
  }
}

async function handleExportSettingChange(val: boolean) {
  try {
    await client.put('/config/export-setting', { includeSubscriptions: val })
  } catch (err: any) {
    message.error(t('common.failed') + ': ' + (err.message || err))
  }
}

function handleImportFile({ file }: { file: UploadFileInfo }) {
  if (!file?.file) return
  dialog.info({
    title: t('subscriptions.importSubsOnly'),
    content: t('subscriptions.importSubsOrNot'),
    positiveText: t('subscriptions.importSubsOnly'),
    negativeText: t('subscriptions.importRulesOnly'),
    onPositiveClick: async () => { await doImport(file.file!, true) },
    onNegativeClick: async () => { await doImport(file.file!, false) },
  })
}

async function doImport(file: File, importSubs: boolean) {
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('importSubscriptions', String(importSubs))
    const { data } = await client.post('/subscriptions/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    
    // Restore test sites if included in import
    if (data.testSites && Array.isArray(data.testSites)) {
      localStorage.setItem('testSites', JSON.stringify(data.testSites))
    }
    
    message.success(t('subscriptions.importSuccess', { name: data.subscriptionName, count: data.subscriptionsImported || 0 }))
    await fetchAll()
  } catch (err: any) {
    message.error(t('subscriptions.importFailed') + ': ' + (err.message || err))
  }
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}
</script>
