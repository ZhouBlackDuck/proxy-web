<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">配置管理</n-text>
        <n-space>
          <n-upload
            :show-file-list="false"
            accept=".zip"
            @change="handleImportFile"
          >
            <n-button>导入 Profile</n-button>
          </n-upload>
          <n-button type="primary" @click="showCreateModal = true">新建 Profile</n-button>
        </n-space>
      </n-space>

      <n-spin :show="loading">
        <n-empty v-if="profiles.length === 0 && !loading" description="暂无配置" />
        <n-grid v-else :x-gap="12" :y-gap="12" :cols="2" responsive="screen" item-responsive>
          <n-gi v-for="profile in profiles" :key="profile.id" span="2 m:1">
            <n-card
              :title="profile.name"
              size="small"
              :style="profile.id === activeId ? 'border-left: 3px solid #18a058' : ''"
            >
              <template #header-extra>
                <n-space :size="4">
                  <n-tag v-if="profile.id === activeId" type="success" size="tiny">当前</n-tag>
                  <n-button
                    v-else
                    size="tiny"
                    type="primary"
                    @click="handleActivate(profile.id)"
                  >
                    激活
                  </n-button>
                </n-space>
              </template>

              <n-descriptions :column="1" label-placement="left" size="small">
                <n-descriptions-item v-if="profile.description" label="描述">
                  {{ profile.description }}
                </n-descriptions-item>
                <n-descriptions-item v-if="profile.subscriptionName" label="订阅">
                  {{ profile.subscriptionName }}
                </n-descriptions-item>
                <n-descriptions-item label="更新时间">
                  {{ new Date(profile.updatedAt).toLocaleString() }}
                </n-descriptions-item>
                <n-descriptions-item label="导出订阅">
                  <n-switch
                    :value="profile.exportSettings?.includeSubscriptions ?? false"
                    @update:value="(v: boolean) => handleToggleExport(profile.id, v)"
                    size="small"
                  />
                </n-descriptions-item>
              </n-descriptions>

              <template #action>
                <n-space :size="8">
                  <n-button size="tiny" @click="handleEditRules(profile)">编辑规则</n-button>
                  <n-button size="tiny" @click="handleEditOverride(profile)">编辑覆盖</n-button>
                  <n-button size="tiny" @click="handlePreview(profile.id)">预览</n-button>
                  <n-button size="tiny" @click="handleExport(profile.id)">导出</n-button>
                  <n-button size="tiny" @click="handleEdit(profile)">编辑</n-button>
                  <n-button size="tiny" type="error" @click="handleDelete(profile)">删除</n-button>
                </n-space>
              </template>
            </n-card>
          </n-gi>
        </n-grid>
      </n-spin>
    </n-space>

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showCreateModal" preset="dialog" :title="editing ? '编辑 Profile' : '新建 Profile'">
      <n-form :model="form" label-placement="left" label-width="80">
        <n-form-item label="名称">
          <n-input v-model:value="form.name" placeholder="Profile 名称" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="form.description" placeholder="描述（可选）" />
        </n-form-item>
        <n-form-item label="关联订阅">
          <n-select
            v-model:value="form.subscriptionName"
            :options="subscriptionOptions"
            placeholder="选择关联的订阅（可选）"
            clearable
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showCreateModal = false; editing = null">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </template>
    </n-modal>

    <!-- Rules Editor Modal -->
    <n-modal v-model:show="showRulesModal" preset="card" title="全局规则编辑" style="width: 700px; max-width: 90vw">
      <n-alert type="info" style="margin-bottom: 12px">
        每行一条规则，格式: TYPE,PAYLOAD,PROXY（如: DOMAIN-SUFFIX,google.com,Proxy）
      </n-alert>
      <n-input
        v-model:value="rulesContent"
        type="textarea"
        :rows="20"
        placeholder="# 每行一条规则&#10;DOMAIN-SUFFIX,google.com,Proxy&#10;GEOIP,CN,DIRECT&#10;MATCH,Proxy"
        style="font-family: monospace"
      />
      <template #action>
        <n-button @click="showRulesModal = false">取消</n-button>
        <n-button type="primary" :loading="savingRules" @click="handleSaveRules">保存规则</n-button>
      </template>
    </n-modal>

    <!-- Override Editor Modal -->
    <n-modal v-model:show="showOverrideModal" preset="card" title="全局配置覆盖" style="width: 700px; max-width: 90vw">
      <n-alert type="info" style="margin-bottom: 12px">
        YAML 格式，此处配置将浅合并到订阅配置上（此处值优先）
      </n-alert>
      <n-input
        v-model:value="overrideContent"
        type="textarea"
        :rows="20"
        placeholder="# YAML 配置覆盖&#10;mixed-port: 7890&#10;allow-lan: true&#10;log-level: info"
        style="font-family: monospace"
      />
      <template #action>
        <n-button @click="showOverrideModal = false">取消</n-button>
        <n-button type="primary" :loading="savingOverride" @click="handleSaveOverride">保存覆盖</n-button>
      </template>
    </n-modal>

    <!-- Preview Modal -->
    <n-modal v-model:show="showPreviewModal" preset="card" title="合并配置预览" style="width: 700px; max-width: 90vw">
      <n-spin :show="previewLoading">
        <n-code :code="previewContent" language="yaml" word-wrap />
      </n-spin>
    </n-modal>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage, useDialog, type UploadFileInfo } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { profileApi, type Profile } from '../api/profiles'
import { subscriptionApi, type Subscription } from '../api/subscriptions'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const saving = ref(false)
const savingRules = ref(false)
const savingOverride = ref(false)
const profiles = ref<Profile[]>([])
const activeId = ref('')
const subscriptions = ref<Subscription[]>([])

const showCreateModal = ref(false)
const showRulesModal = ref(false)
const showOverrideModal = ref(false)
const showPreviewModal = ref(false)
const editing = ref<Profile | null>(null)
const currentProfileId = ref('')
const previewContent = ref('')
const previewLoading = ref(false)

const form = ref({ name: '', description: '', subscriptionName: '' as string | null })
const rulesContent = ref('')
const overrideContent = ref('')

const subscriptionOptions = computed(() =>
  subscriptions.value.map((s) => ({ label: s.displayName || s.name, value: s.name }))
)

onMounted(() => {
  fetchAll()
})

async function fetchAll() {
  loading.value = true
  try {
    const [registry, subs] = await Promise.all([
      profileApi.list(),
      subscriptionApi.list(),
    ])
    profiles.value = registry.profiles
    activeId.value = registry.activeProfileId
    subscriptions.value = subs
  } catch (err: any) {
    message.error('获取数据失败: ' + (err.message || err))
  } finally {
    loading.value = false
  }
}

function handleEdit(profile: Profile) {
  editing.value = profile
  form.value = {
    name: profile.name,
    description: profile.description || '',
    subscriptionName: profile.subscriptionName || null,
  }
  showCreateModal.value = true
}

async function handleSave() {
  if (!form.value.name) {
    message.error('请填写名称')
    return
  }

  saving.value = true
  try {
    if (editing.value) {
      await profileApi.update(editing.value.id, {
        name: form.value.name,
        description: form.value.description,
        subscriptionName: form.value.subscriptionName || undefined,
      })
      message.success('已更新')
    } else {
      await profileApi.create({
        name: form.value.name,
        description: form.value.description,
        subscriptionName: form.value.subscriptionName || undefined,
      })
      message.success('已创建')
    }
    showCreateModal.value = false
    editing.value = null
    form.value = { name: '', description: '', subscriptionName: null }
    await fetchAll()
  } catch (err: any) {
    message.error('保存失败: ' + (err.message || err))
  } finally {
    saving.value = false
  }
}

async function handleActivate(id: string) {
  try {
    await profileApi.activate(id)
    message.success('已激活')
    await fetchAll()
  } catch (err: any) {
    message.error('激活失败: ' + (err.message || err))
  }
}

function handleDelete(profile: Profile) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除 Profile「${profile.name}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await profileApi.delete(profile.id)
        message.success('已删除')
        await fetchAll()
      } catch (err: any) {
        message.error('删除失败: ' + (err.message || err))
      }
    },
  })
}

async function handleToggleExport(id: string, val: boolean) {
  try {
    await profileApi.update(id, { exportSettings: { includeSubscriptions: val } })
    await fetchAll()
  } catch (err: any) {
    message.error('更新失败: ' + (err.message || err))
  }
}

async function handleEditRules(profile: Profile) {
  currentProfileId.value = profile.id
  try {
    rulesContent.value = await profileApi.getRules(profile.id)
  } catch {
    rulesContent.value = ''
  }
  showRulesModal.value = true
}

async function handleSaveRules() {
  savingRules.value = true
  try {
    await profileApi.updateRules(currentProfileId.value, rulesContent.value)
    message.success('规则已保存')
    showRulesModal.value = false
  } catch (err: any) {
    message.error('保存失败: ' + (err.message || err))
  } finally {
    savingRules.value = false
  }
}

async function handleEditOverride(profile: Profile) {
  currentProfileId.value = profile.id
  try {
    overrideContent.value = await profileApi.getOverride(profile.id)
  } catch {
    overrideContent.value = ''
  }
  showOverrideModal.value = true
}

async function handleSaveOverride() {
  savingOverride.value = true
  try {
    await profileApi.updateOverride(currentProfileId.value, overrideContent.value)
    message.success('覆盖配置已保存')
    showOverrideModal.value = false
  } catch (err: any) {
    message.error('保存失败: ' + (err.message || err))
  } finally {
    savingOverride.value = false
  }
}

async function handlePreview(id: string) {
  showPreviewModal.value = true
  previewLoading.value = true
  previewContent.value = ''
  try {
    previewContent.value = await profileApi.preview(id)
  } catch (err: any) {
    previewContent.value = '# 预览失败\n# ' + (err.message || err)
  } finally {
    previewLoading.value = false
  }
}

async function handleExport(id: string) {
  try {
    const blob = await profileApi.export(id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `profile-${id}.zip`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (err: any) {
    message.error('导出失败: ' + (err.message || err))
  }
}

async function handleImportFile({ file }: { file: UploadFileInfo }) {
  if (!file?.file) return

  dialog.info({
    title: '导入 Profile',
    content: '是否同时导入订阅配置？（如果备份中包含订阅数据）',
    positiveText: '导入订阅',
    negativeText: '仅导入平台配置',
    onPositiveClick: async () => {
      await doImport(file.file!, true)
    },
    onNegativeClick: async () => {
      await doImport(file.file!, false)
    },
  })
}

async function doImport(file: File, importSubs: boolean) {
  try {
    const result = await profileApi.import(file, importSubs)
    message.success(`导入成功: ${result.profileName}（订阅: ${result.subscriptionsImported} 个）`)
    await fetchAll()
  } catch (err: any) {
    message.error('导入失败: ' + (err.message || err))
  }
}
</script>
