<template>
  <AppLayout>
    <n-space vertical :size="16">
      <n-space justify="space-between" align="center">
        <n-text strong style="font-size: 18px">规则查看</n-text>
        <n-space>
          <n-select
            v-model:value="filterType"
            :options="typeOptions"
            placeholder="规则类型"
            clearable
            style="width: 160px"
            size="small"
          />
          <n-input
            v-model:value="searchText"
            placeholder="搜索规则..."
            clearable
            style="width: 200px"
            size="small"
          />
          <n-button size="small" @click="fetchRules" :loading="loading">刷新</n-button>
        </n-space>
      </n-space>

      <n-spin :show="loading">
        <n-empty v-if="rules.length === 0 && !loading" description="暂无规则" />
        <n-data-table
          v-else
          :columns="columns"
          :data="filteredRules"
          :bordered="false"
          :single-line="false"
          size="small"
          :pagination="{ pageSize: 50 }"
          virtual-scroll
          max-height="calc(100vh - 200px)"
        />
      </n-spin>
    </n-space>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { NTag, NText, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import AppLayout from '../components/layout/AppLayout.vue'
import { kernelApi, type Rule } from '../api/kernel'

const message = useMessage()
const loading = ref(false)
const rules = ref<Rule[]>([])
const searchText = ref('')
const filterType = ref<string | null>(null)

const typeOptions = computed(() => {
  const types = [...new Set(rules.value.map((r) => r.type))]
  return types.map((t) => ({ label: t, value: t }))
})

const filteredRules = computed(() => {
  let result = rules.value
  if (filterType.value) {
    result = result.filter((r) => r.type === filterType.value)
  }
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(
      (r) =>
        r.payload.toLowerCase().includes(q) ||
        r.proxy.toLowerCase().includes(q) ||
        r.type.toLowerCase().includes(q)
    )
  }
  return result
})

const columns: DataTableColumns<Rule> = [
  { title: '#', key: 'index', width: 50 },
  { title: '类型', key: 'type', width: 140,
    render(row) {
      return h(NTag, { size: 'tiny', type: 'info' }, { default: () => row.type })
    }
  },
  { title: '匹配条件', key: 'payload', ellipsis: { tooltip: true } },
  { title: '策略', key: 'proxy', width: 140,
    render(row) {
      const color = row.proxy === 'DIRECT' ? 'success' : row.proxy === 'REJECT' ? 'error' : 'warning'
      return h(NTag, { size: 'tiny', type: color }, { default: () => row.proxy })
    }
  },
  { title: '命中', key: 'hitCount', width: 80,
    render(row) {
      const count = row.extra?.hitCount ?? 0
      return h(NText, { depth: count > 0 ? undefined : 3 }, { default: () => String(count) })
    },
    sorter: (a, b) => (a.extra?.hitCount ?? 0) - (b.extra?.hitCount ?? 0),
  },
]

onMounted(() => {
  fetchRules()
})

async function fetchRules() {
  loading.value = true
  try {
    const res = await kernelApi.getRules()
    rules.value = res.rules
  } catch (err: any) {
    message.error('获取规则失败: ' + (err.message || err))
  } finally {
    loading.value = false
  }
}
</script>
