import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface LogEntry {
  time: string
  type: string
  payload: string
}

const MAX_LOGS = 500

export const useLogStore = defineStore('logs', () => {
  const logs = ref<LogEntry[]>([])
  const connected = ref(false)
  const clearedAt = ref<number>(0) // timestamp of last clear

  function addLog(entry: LogEntry) {
    // Skip logs that were generated before the last clear
    if (clearedAt.value > 0) {
      const logTime = parseLogTime(entry.time)
      if (logTime > 0 && logTime < clearedAt.value) return
    }
    logs.value.push(entry)
    if (logs.value.length > MAX_LOGS) {
      logs.value = logs.value.slice(-MAX_LOGS)
    }
  }

  function clearLogs() {
    logs.value = []
    clearedAt.value = Date.now()
  }

  function parseLogTime(timeStr: string): number {
    if (!timeStr) return 0
    // Try HH:MM:SS format - assume today
    const match = timeStr.match(/(\d{2}):(\d{2}):(\d{2})/)
    if (match) {
      const now = new Date()
      now.setHours(parseInt(match[1]), parseInt(match[2]), parseInt(match[3]), 0)
      return now.getTime()
    }
    return 0
  }

  // Load historical logs from file
  async function loadHistory(token: string) {
    // Skip if logs were cleared
    if (clearedAt.value > 0) return
    try {
      const resp = await fetch('/api/logs?limit=200', {
        headers: { Authorization: `Bearer ${token}` },
      })
      const data = await resp.json()
      if (data.logs && data.logs.length > 0) {
        if (logs.value.length < data.logs.length) {
          logs.value = data.logs.map((l: any) => ({
            time: l.time || '',
            type: l.type || 'info',
            payload: l.payload || '',
          }))
        }
      }
    } catch {
      // ignore
    }
  }

  return { logs, connected, addLog, clearLogs, loadHistory }
})
