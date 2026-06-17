<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">节点管理</n-text>
        <n-space>
          <n-input v-model:value="searchText" placeholder="搜索节点..." clearable style="width: 200px" size="small" />
          <n-button size="small" @click="fetchAll" :loading="loading">刷新</n-button>
        </n-space>
      </n-space>

      <n-spin :show="loading">
        <n-empty v-if="groups.length === 0 && !loading" description="暂无代理组" />

        <n-space v-else vertical :size="16">
          <n-card
            v-for="group in filteredGroups"
            :key="group.name"
            :title="group.name"
            size="small"
          >
            <template #header-extra>
              <n-space :size="8" align="center">
                <n-tag size="tiny" :type="groupTypeColor(group.type)">{{ group.type }}</n-tag>
                <n-text v-if="group.now" depth="3" style="font-size: 12px">
                  当前: {{ group.now }}
                </n-text>
                <n-button
                  size="tiny"
                  @click="handleTestGroup(group.name)"
                  :loading="testingGroup === group.name"
                >
                  测速
                </n-button>
              </n-space>
            </template>

            <n-grid :x-gap="8" :y-gap="8" :cols="6" responsive="screen" item-responsive>
              <n-gi
                v-for="nodeName in filteredNodes(group)"
                :key="nodeName"
                span="6 s:3 m:2 l:1"
              >
                <n-card
                  size="tiny"
                  :class="{ 'node-selected': group.now === nodeName }"
                  :style="nodeCardStyle(group, nodeName)"
                  hoverable
                  @click="handleSelectNode(group.name, nodeName, group.type)"
                  style="cursor: pointer; transition: all 0.2s"
                >
                  <n-ellipsis :line-clamp="1" style="font-size: 12px">
                    {{ nodeName }}
                  </n-ellipsis>
                  <n-text depth="3" style="font-size: 10px">
                    {{ getDelay(group, nodeName) }}
                  </n-text>
                </n-card>
              </n-gi>
            </n-grid>
          </n-card>
        </n-space>
      </n-spin>
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { kernelApi, type ProxyNode } from '../api/kernel'

const message = useMessage()

const loading = ref(false)
const searchText = ref('')
const testingGroup = ref<string | null>(null)

const groups = ref<ProxyNode[]>([])
const proxies = ref<Record<string, ProxyNode>>({})
const delayMap = ref<Record<string, Record<string, number>>>({})

onMounted(() => {
  fetchAll()
})

async function fetchAll() {
  loading.value = true
  try {
    const [groupsRes, proxiesRes] = await Promise.all([
      kernelApi.getGroups(),
      kernelApi.getProxies(),
    ])

    groups.value = Object.values(groupsRes.proxies)
      .filter((p) => p.type !== 'Compatible' && p.type !== 'Unknown')
      .sort((a, b) => a.name.localeCompare(b.name))

    proxies.value = proxiesRes.proxies
  } catch (err: any) {
    // Don't show error if kernel is simply not configured yet
    if (groups.value.length === 0) {
      groups.value = []
      proxies.value = {}
    } else {
      message.error('获取节点信息失败: ' + (err.message || err))
    }
  } finally {
    loading.value = false
  }
}

const filteredGroups = computed(() => {
  if (!searchText.value) return groups.value
  const q = searchText.value.toLowerCase()
  return groups.value.filter(
    (g) =>
      g.name.toLowerCase().includes(q) ||
      g.all?.some((n) => n.toLowerCase().includes(q))
  )
})

function filteredNodes(group: ProxyNode): string[] {
  if (!group.all) return []
  if (!searchText.value) return group.all
  const q = searchText.value.toLowerCase()
  return group.all.filter((n) => n.toLowerCase().includes(q))
}

function groupTypeColor(type: string): 'default' | 'info' | 'success' | 'warning' {
  switch (type) {
    case 'Selector': return 'info'
    case 'URLTest': return 'success'
    case 'Fallback': return 'warning'
    case 'LoadBalance': return 'warning'
    default: return 'default'
  }
}

function nodeCardStyle(group: ProxyNode, nodeName: string) {
  const isSelected = group.now === nodeName
  const delay = delayMap.value[group.name]?.[nodeName]

  let bgColor = ''
  if (delay !== undefined) {
    if (delay === 0) bgColor = 'rgba(255, 0, 0, 0.05)'
    else if (delay < 200) bgColor = 'rgba(24, 160, 88, 0.08)'
    else if (delay < 500) bgColor = 'rgba(240, 160, 32, 0.08)'
    else bgColor = 'rgba(208, 48, 80, 0.08)'
  }

  return {
    borderLeft: isSelected ? '3px solid #18a058' : '3px solid transparent',
    backgroundColor: bgColor,
  }
}

function getDelay(group: ProxyNode, nodeName: string): string {
  const delay = delayMap.value[group.name]?.[nodeName]
  if (delay === undefined) return ''
  if (delay === 0) return 'Timeout'
  return `${delay}ms`
}

async function handleSelectNode(groupName: string, nodeName: string, groupType: string) {
  if (groupType !== 'Selector') {
    message.info(`${groupType} 类型不支持手动切换节点`)
    return
  }
  try {
    await kernelApi.switchProxy(groupName, nodeName)
    message.success(`已切换到: ${nodeName}`)
    // Refresh
    await fetchAll()
  } catch (err: any) {
    message.error('切换失败: ' + (err.message || err))
  }
}

async function handleTestGroup(groupName: string) {
  testingGroup.value = groupName
  try {
    const result = await kernelApi.testGroupDelay(groupName)
    delayMap.value[groupName] = result
    message.success('测速完成')
  } catch (err: any) {
    message.error('测速失败: ' + (err.message || err))
  } finally {
    testingGroup.value = null
  }
}
</script>
