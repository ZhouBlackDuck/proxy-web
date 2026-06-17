<template>
  <AppLayout>
    <n-space vertical :size="16">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">订阅管理</n-text>
        <n-button type="primary" @click="showCreateModal = true">
          添加订阅
        </n-button>
      </n-space>

      <!-- Subscription List -->
      <n-spin :show="loading">
        <n-empty v-if="subscriptions.length === 0 && !loading" description="暂无订阅" />
        <n-grid v-else :x-gap="12" :y-gap="12" :cols="3" responsive="screen" item-responsive>
          <n-gi v-for="sub in subscriptions" :key="sub.name" span="3 m:1">
            <n-card :title="sub.displayName || sub.name" size="small">
              <template #header-extra>
                <n-space :size="4">
                  <n-button size="tiny" quaternary @click="handleSync(sub.name)" :loading="syncing === sub.name">
                    同步
                  </n-button>
                  <n-dropdown :options="moreOptions" @select="(k: string) => handleMore(k, sub)">
                    <n-button size="tiny" quaternary>⋯</n-button>
                  </n-dropdown>
                </n-space>
              </template>
              <n-descriptions :column="1" label-placement="left" size="small">
                <n-descriptions-item label="来源">
                  <n-tag size="tiny" :type="sub.url ? 'info' : 'default'">
                    {{ sub.url ? 'URL' : '本地' }}
                  </n-tag>
                </n-descriptions-item>
                <n-descriptions-item v-if="sub.url" label="URL">
                  <n-ellipsis :line-clamp="1" style="max-width: 200px">{{ sub.url }}</n-ellipsis>
                </n-descriptions-item>
                <n-descriptions-item v-if="sub.updatedAt" label="更新时间">
                  {{ formatTime(sub.updatedAt) }}
                </n-descriptions-item>
              </n-descriptions>
              <template #action>
                <n-space :size="8">
                  <n-button size="tiny" @click="handlePreview(sub.name)">预览配置</n-button>
                  <n-button size="tiny" @click="handleEdit(sub)">编辑</n-button>
                </n-space>
              </template>
            </n-card>
          </n-gi>
        </n-grid>
      </n-spin>
    </n-space>

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showCreateModal" preset="dialog" :title="editingSub ? '编辑订阅' : '添加订阅'">
      <n-form :model="form" label-placement="left" label-width="80">
        <n-form-item label="名称">
          <n-input v-model:value="form.name" placeholder="订阅名称（英文）" :disabled="!!editingSub" />
        </n-form-item>
        <n-form-item label="显示名称">
          <n-input v-model:value="form.displayName" placeholder="显示名称（可选）" />
        </n-form-item>
        <n-form-item label="类型">
          <n-radio-group v-model:value="form.source">
            <n-radio-button value="url">URL 订阅</n-radio-button>
            <n-radio-button value="local">本地配置</n-radio-button>
          </n-radio-group>
        </n-form-item>
        <n-form-item v-if="form.source === 'url'" label="订阅 URL">
          <n-input v-model:value="form.url" type="textarea" placeholder="https://..." :rows="3" />
        </n-form-item>
        <n-form-item v-if="form.source === 'local'" label="配置内容">
          <n-input v-model:value="form.content" type="textarea" placeholder="粘贴 Clash YAML 配置..." :rows="8" />
        </n-form-item>
        <n-form-item v-if="form.source === 'url'" label="User-Agent">
          <n-input v-model:value="form.ua" placeholder="可选，自定义 UA" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showCreateModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </template>
    </n-modal>

    <!-- Preview Modal -->
    <n-modal v-model:show="showPreviewModal" preset="card" title="配置预览" style="width: 700px; max-width: 90vw">
      <n-spin :show="previewLoading">
        <n-code :code="previewContent" language="yaml" word-wrap />
      </n-spin>
    </n-modal>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { subscriptionApi, type Subscription } from '../api/subscriptions'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const saving = ref(false)
const syncing = ref<string | null>(null)
const subscriptions = ref<Subscription[]>([])
const showCreateModal = ref(false)
const showPreviewModal = ref(false)
const editingSub = ref<Subscription | null>(null)
const previewContent = ref('')
const previewLoading = ref(false)

const form = ref({
  name: '',
  displayName: '',
  source: 'url' as 'url' | 'local',
  url: '',
  content: '',
  ua: '',
})

const moreOptions = [
  { label: '删除', key: 'delete' },
]

onMounted(() => {
  fetchSubscriptions()
})

async function fetchSubscriptions() {
  loading.value = true
  try {
    subscriptions.value = await subscriptionApi.list()
  } catch (err: any) {
    message.error('获取订阅列表失败: ' + (err.message || err))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { name: '', displayName: '', source: 'url', url: '', content: '', ua: '' }
  editingSub.value = null
}

function handleEdit(sub: Subscription) {
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
  if (!form.value.name) {
    message.error('请填写名称')
    return
  }

  saving.value = true
  try {
    if (editingSub.value) {
      // Update
      await subscriptionApi.update(editingSub.value.name, {
        displayName: form.value.displayName,
        url: form.value.source === 'url' ? form.value.url : undefined,
        content: form.value.source === 'local' ? form.value.content : undefined,
        ua: form.value.ua || undefined,
      })
      message.success('订阅已更新')
    } else {
      // Create
      await subscriptionApi.create({
        name: form.value.name,
        displayName: form.value.displayName,
        url: form.value.source === 'url' ? form.value.url : undefined,
        content: form.value.source === 'local' ? form.value.content : undefined,
        ua: form.value.ua || undefined,
      })
      message.success('订阅已创建')
    }
    showCreateModal.value = false
    resetForm()
    await fetchSubscriptions()
  } catch (err: any) {
    message.error('保存失败: ' + (err.message || err))
  } finally {
    saving.value = false
  }
}

function handleMore(key: string, sub: Subscription) {
  if (key === 'delete') {
    dialog.warning({
      title: '确认删除',
      content: `确定删除订阅「${sub.displayName || sub.name}」吗？`,
      positiveText: '删除',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await subscriptionApi.delete(sub.name)
          message.success('已删除')
          await fetchSubscriptions()
        } catch (err: any) {
          message.error('删除失败: ' + (err.message || err))
        }
      },
    })
  }
}

async function handleSync(name: string) {
  syncing.value = name
  try {
    await subscriptionApi.sync(name)
    message.success('同步已触发')
    setTimeout(() => fetchSubscriptions(), 3000)
  } catch (err: any) {
    message.error('同步失败: ' + (err.message || err))
  } finally {
    syncing.value = null
  }
}

async function handlePreview(name: string) {
  showPreviewModal.value = true
  previewLoading.value = true
  previewContent.value = ''
  try {
    previewContent.value = await subscriptionApi.download(name)
  } catch (err: any) {
    previewContent.value = '# 预览失败\n# ' + (err.message || err)
  } finally {
    previewLoading.value = false
  }
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}
</script>
