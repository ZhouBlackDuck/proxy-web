import { ref, onMounted, onUnmounted } from 'vue'

interface UseWebSocketOptions {
  url: string
  onMessage: (data: any) => void
  reconnectInterval?: number
}

export function useWebSocket(options: UseWebSocketOptions) {
  const connected = ref(false)
  const error = ref<string | null>(null)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let shouldReconnect = true

  function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}${options.url}`

    try {
      ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        connected.value = true
        error.value = null
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          options.onMessage(data)
        } catch {
          options.onMessage(event.data)
        }
      }

      ws.onclose = () => {
        connected.value = false
        if (shouldReconnect) {
          reconnectTimer = setTimeout(connect, options.reconnectInterval || 3000)
        }
      }

      ws.onerror = () => {
        error.value = 'WebSocket connection error'
        connected.value = false
      }
    } catch (e) {
      error.value = String(e)
    }
  }

  function disconnect() {
    shouldReconnect = false
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    if (ws) {
      ws.close()
      ws = null
    }
  }

  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    disconnect()
  })

  return { connected, error, connect, disconnect }
}
