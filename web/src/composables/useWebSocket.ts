import { ref, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'

export interface WebSocketMessage {
  type: string
  data: any
}

export function useStatusWebSocket() {
  const messages = ref<WebSocketMessage[]>([])
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  
  const connect = () => {
    // 清除之前的重连定时器
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    
    const authStore = useAuthStore()
    if (!authStore.token) {
      console.warn('No token available for WebSocket connection')
      return
    }
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/status?token=${authStore.token}`
    
    try {
      ws = new WebSocket(wsUrl)
      
      ws.onopen = () => {
        console.log('WebSocket connected')
        connected.value = true
      }
      
      ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason)
        connected.value = false
        ws = null
        // 5秒后重连
        reconnectTimer = setTimeout(connect, 5000)
      }
      
      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
      
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WebSocketMessage
          messages.value.push(data)
          // 保留最近 100 条
          if (messages.value.length > 100) {
            messages.value = messages.value.slice(-100)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
      connected.value = false
      reconnectTimer = setTimeout(connect, 5000)
    }
  }
  
  const disconnect = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      // 移除 onclose 处理器以避免重连
      ws.onclose = null
      ws.close()
      ws = null
    }
    connected.value = false
  }
  
  const send = (data: any) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data))
    } else {
      console.warn('WebSocket is not connected')
    }
  }
  
  // 自动连接
  onMounted(() => {
    connect()
  })
  
  // 清理
  onUnmounted(() => {
    disconnect()
  })
  
  return { 
    messages, 
    connected, 
    connect, 
    disconnect,
    send
  }
}
