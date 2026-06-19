<template>
  <AppLayout>
    <n-space vertical :size="16">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">{{ t('proxies.title') }}</n-text>
        <n-space :size="8">
          <n-input v-model:value="searchText" :placeholder="t('proxies.searchPlaceholder')" clearable style="width: 200px" size="small" />
          <n-button size="small" @click="handleTestAll" :loading="testingAll" type="primary">
            {{ t('proxies.testAll') }}
          </n-button>
          <n-button size="small" @click="fetchAll" :loading="loading">{{ t('common.refresh') }}</n-button>
        </n-space>
      </n-space>

      <!-- Stats -->
      <n-space v-if="groups.length > 0" :size="16">
        <n-text depth="3">{{ groups.length }} {{ t('proxies.groups') }}</n-text>
        <n-text depth="3">{{ totalNodes }} {{ t('proxies.nodes') }}</n-text>
      </n-space>

      <n-spin :show="loading">
        <n-empty v-if="groups.length === 0 && !loading" :description="t('proxies.noGroups')" />

        <n-space v-else vertical :size="16">
          <n-card
            v-for="group in filteredGroups"
            :key="group.name"
            size="small"
          >
            <template #header>
              <n-space align="center" :size="8">
                <n-tag size="tiny" :type="groupTypeColor(group.type)">{{ group.type }}</n-tag>
                <n-text strong>{{ group.name }}</n-text>
                <n-text v-if="group.now" depth="3" style="font-size: 12px">
                  → {{ group.now }}
                </n-text>
              </n-space>
            </template>
            <template #header-extra>
              <n-space :size="4">
                <n-text depth="3" style="font-size: 11px">{{ (group.all || []).length }} {{ t('proxies.nodes') }}</n-text>
                <n-button
                  size="tiny"
                  @click="handleTestGroup(group.name)"
                  :loading="testingGroup === group.name"
                >
                  {{ t('proxies.testSpeed') }}
                </n-button>
              </n-space>
            </template>

            <n-grid :x-gap="6" :y-gap="6" :cols="8" responsive="screen" item-responsive>
              <n-gi
                v-for="nodeName in filteredNodes(group)"
                :key="nodeName"
                span="8 s:4 m:2 l:1"
              >
                <div
                  class="node-card"
                  :class="{
                    'node-selected': group.now === nodeName,
                    'node-timeout': isTimeout(group, nodeName),
                  }"
                  :style="nodeCardStyle(group, nodeName)"
                  @click="handleSelectNode(group.name, nodeName, group.type)"
                >
                  <div class="node-name">
                    <n-ellipsis :line-clamp="1">{{ nodeName }}</n-ellipsis>
                  </div>
                  <div class="node-info">
                    <span class="node-type">{{ getNodeType(nodeName) }}</span>
                    <span class="node-delay" :class="delayClass(group, nodeName)">
                      {{ getDelayText(group, nodeName) }}
                    </span>
                  </div>
                </div>
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
import { useI18n } from 'vue-i18n'
import AppLayout from '../components/layout/AppLayout.vue'
import { kernelApi, type ProxyNode } from '../api/kernel'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const searchText = ref('')
const testingGroup = ref<string | null>(null)
const testingAll = ref(false)

const groups = ref<ProxyNode[]>([])
const proxies = ref<Record<string, ProxyNode>>({})
// delayMap: { groupName: { nodeName: delay } } where delay=-1 means timeout
const delayMap = ref<Record<string, Record<string, number>>>({})

onMounted(() => {
  fetchAll()
})

const totalNodes = computed(() => {
  const nodeSet = new Set<string>()
  for (const g of groups.value) {
    for (const n of (g.all || [])) nodeSet.add(n)
  }
  return nodeSet.size
})

async function fetchAll() {
  loading.value = true
  try {
    const [groupsRes, proxiesRes] = await Promise.all([
      kernelApi.getGroups(),
      kernelApi.getProxies(),
    ])
    groups.value = Object.values(groupsRes.proxies)
      .filter((p) => !['Compatible', 'Unknown', 'Direct', 'Reject', 'RejectDrop', 'Pass', 'PassRule'].includes(p.type))
      .sort((a, b) => a.name.localeCompare(b.name))
    proxies.value = proxiesRes.proxies

    // Extract historical delay data from proxies
    for (const [name, proxy] of Object.entries(proxiesRes.proxies)) {
      if (proxy.history && proxy.history.length > 0) {
        const last = proxy.history[proxy.history.length - 1]
        if (last && last.delay !== undefined) {
          // Store in all groups that contain this node
          for (const g of groups.value) {
            if (g.all?.includes(name)) {
              if (!delayMap.value[g.name]) delayMap.value[g.name] = {}
              delayMap.value[g.name][name] = last.delay > 0 ? last.delay : -1
            }
          }
        }
      }
    }
  } catch (err: any) {
    if (groups.value.length === 0) {
      groups.value = []
      proxies.value = {}
    } else {
      message.error(t('proxies.fetchFailed') + ': ' + (err.message || err))
    }
  } finally {
    loading.value = false
  }
}

const filteredGroups = computed(() => {
  if (!searchText.value) return groups.value
  const q = searchText.value.toLowerCase()
  return groups.value.filter(
    (g) => g.name.toLowerCase().includes(q) || g.all?.some((n) => n.toLowerCase().includes(q))
  )
})

function filteredNodes(group: ProxyNode): string[] {
  if (!group.all) return []
  if (!searchText.value) return group.all
  const q = searchText.value.toLowerCase()
  return group.all.filter((n) => n.toLowerCase().includes(q))
}

function getNodeType(nodeName: string): string {
  const p = proxies.value[nodeName]
  if (!p) return ''
  const typeMap: Record<string, string> = {
    Shadowsocks: 'SS',
    Vmess: 'VMess',
    Vless: 'VLESS',
    Trojan: 'Trojan',
    Hysteria: 'HY',
    Hysteria2: 'HY2',
    Tuic: 'TUIC',
    Selector: t('proxies.typeSelector'),
    URLTest: t('proxies.typeUrlTest'),
    Fallback: t('proxies.typeFallback'),
    LoadBalance: t('proxies.typeLoadBalance'),
    Direct: t('proxies.typeDirect'),
    Reject: t('proxies.typeReject'),
  }
  return typeMap[p.type] || p.type
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

function isTimeout(group: ProxyNode, nodeName: string): boolean {
  return delayMap.value[group.name]?.[nodeName] === -1
}

function nodeCardStyle(group: ProxyNode, nodeName: string) {
  const isSelected = group.now === nodeName
  const delay = delayMap.value[group.name]?.[nodeName]
  let bgColor = ''
  if (delay !== undefined) {
    if (delay <= 0) bgColor = 'rgba(128,128,128,0.06)'       // timeout/untested
    else if (delay < 150) bgColor = 'rgba(24,160,88,0.1)'    // green: excellent
    else if (delay < 300) bgColor = 'rgba(24,160,88,0.06)'   // light green: good
    else if (delay < 600) bgColor = 'rgba(240,160,32,0.08)'  // yellow: ok
    else bgColor = 'rgba(208,48,80,0.08)'                     // red: slow
  }
  return {
    borderLeft: isSelected ? '3px solid #18a058' : '3px solid transparent',
    backgroundColor: bgColor,
  }
}

function delayClass(group: ProxyNode, nodeName: string): string {
  const delay = delayMap.value[group.name]?.[nodeName]
  if (delay === undefined) return ''
  if (delay <= 0) return 'delay-timeout'
  if (delay < 150) return 'delay-excellent'
  if (delay < 300) return 'delay-good'
  if (delay < 600) return 'delay-ok'
  return 'delay-slow'
}

function getDelayText(group: ProxyNode, nodeName: string): string {
  const delay = delayMap.value[group.name]?.[nodeName]
  if (delay === undefined) return ''
  if (delay <= 0) return t('proxies.timeout')
  return `${delay}ms`
}

async function handleSelectNode(groupName: string, nodeName: string, groupType: string) {
  if (groupType !== 'Selector') {
    message.info(t('proxies.switchNotSupported', { type: groupType }))
    return
  }
  try {
    await kernelApi.switchProxy(groupName, nodeName)
    message.success(t('proxies.switched', { node: nodeName }))
    await fetchAll()
  } catch (err: any) {
    message.error(t('common.failed') + ': ' + (err.message || err))
  }
}

async function handleTestGroup(groupName: string) {
  testingGroup.value = groupName
  try {
    const result = await kernelApi.testGroupDelay(groupName, 'http://www.gstatic.com/generate_204', 10000)
    if (!delayMap.value[groupName]) delayMap.value[groupName] = {}
    for (const [node, delay] of Object.entries(result)) {
      delayMap.value[groupName][node] = delay > 0 ? delay : -1
    }
    message.success(groupName + ': ' + t('proxies.speedTestDone'))
  } catch (err: any) {
    // "all proxies timeout" is not really an error
    if (err.message?.includes('timeout') || err.message?.includes('Timeout')) {
      if (!delayMap.value[groupName]) delayMap.value[groupName] = {}
      const group = groups.value.find(g => g.name === groupName)
      if (group?.all) {
        for (const node of group.all) {
          delayMap.value[groupName][node] = -1
        }
      }
      message.warning(groupName + ': ' + t('proxies.allTimeout'))
    } else {
      message.error(t('proxies.testFailed') + ': ' + (err.message || err))
    }
  } finally {
    testingGroup.value = null
  }
}

async function handleTestAll() {
  testingAll.value = true
  for (const group of groups.value) {
    testingGroup.value = group.name
    try {
      const result = await kernelApi.testGroupDelay(group.name, 'http://www.gstatic.com/generate_204', 10000)
      if (!delayMap.value[group.name]) delayMap.value[group.name] = {}
      for (const [node, delay] of Object.entries(result)) {
        delayMap.value[group.name][node] = delay > 0 ? delay : -1
      }
    } catch {
      if (!delayMap.value[group.name]) delayMap.value[group.name] = {}
      if (group.all) {
        for (const node of group.all) {
          delayMap.value[group.name][node] = -1
        }
      }
    }
  }
  testingGroup.value = null
  testingAll.value = false
  message.success(t('proxies.allTestDone'))
}
</script>

<style scoped>
.node-card {
  padding: 6px 8px;
  border-radius: 4px;
  border: 1px solid rgba(128,128,128,0.12);
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.node-card:hover {
  border-color: rgba(128,128,128,0.3);
}
.node-selected {
  border-color: #18a058 !important;
  border-left-width: 3px;
}
.node-name {
  font-size: 12px;
  font-weight: 500;
  line-height: 1.3;
  margin-bottom: 3px;
}
.node-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 10px;
}
.node-type {
  color: #888;
}
.node-delay {
  font-weight: 600;
  font-size: 11px;
}
.delay-excellent { color: #18a058; }
.delay-good { color: #63e2b7; }
.delay-ok { color: #f0a020; }
.delay-slow { color: #d03050; }
.delay-timeout { color: #999; }
.node-timeout .node-name { opacity: 0.5; }
</style>
