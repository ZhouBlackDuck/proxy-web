import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface LogEntry {
  time: string
  type: string
  payload: string
  timestamp: number
}

export const useLogStore = defineStore('logs', () => {
  const logs = ref<LogEntry[]>([])
  let fetching = false

  function clearLogs() {
    logs.value = []
  }

  function getFilteredLogs(levelFilter: string, searchText: string): LogEntry[] {
    const levelOrder: Record<string, number> = { debug: 0, info: 1, warning: 2, error: 3 }
    let filtered = logs.value

    // Filter by level: show logs >= selected level
    if (levelFilter && levelFilter !== 'all') {
      const minLevel = levelOrder[levelFilter] ?? 0
      filtered = filtered.filter(log => (levelOrder[log.type] ?? 1) >= minLevel)
    }

    // Filter by search text
    if (searchText) {
      const search = searchText.toLowerCase()
      filtered = filtered.filter(log =>
        log.payload.toLowerCase().includes(search) ||
        log.type.toLowerCase().includes(search)
      )
    }

    return filtered
  }

  // Fetch logs from server (skips if a fetch is already in flight)
  async function fetchLogs(token: string) {
    if (fetching) return
    fetching = true
    try {
      const resp = await fetch('/api/logs', {
        headers: { Authorization: `Bearer ${token}` },
      })
      const data = await resp.json()
      if (data.logs) {
        logs.value = data.logs
      }
    } catch {
      // ignore
    } finally {
      fetching = false
    }
  }

  return { logs, clearLogs, getFilteredLogs, fetchLogs }
})
