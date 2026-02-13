import { getAccessToken } from '@/composables/useAuth'

export class WebSocketManager {
  constructor(url) {
    this.baseUrl = url || this._getWsUrl()
    this.url = this.baseUrl
    this.ws = null
    this.listeners = new Map()
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 10
    this.reconnectDelay = 1000
    this.maxDelay = 30000
    this.shouldReconnect = true
    this.connected = false
    this.reconnecting = false
    this._reconnectTimeout = null
  }

  _getWsUrl() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/ws`
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return

    // Reset reconnect state when explicitly connecting
    this.shouldReconnect = true

    try {
      this.ws = new WebSocket(this.baseUrl)
    } catch (err) {
      console.error('WebSocket connection failed:', err)
      this._scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      // Authenticate via first message (not URL query param)
      const token = getAccessToken()
      if (token) {
        this.ws.send(JSON.stringify({ type: 'auth', token }))
      }
      this.connected = true
      this.reconnecting = false
      this.reconnectAttempts = 0
      this._emit('connected', {})
    }

    this.ws.onmessage = (event) => {
      let message
      try {
        message = JSON.parse(event.data)
      } catch (err) {
        console.warn('WebSocket: failed to parse message:', err.message)
        return
      }
      this._emit(message.type, message.data)
      this._emit('message', message)
    }

    this.ws.onclose = () => {
      this.connected = false
      this._emit('disconnected', {})
      this._scheduleReconnect()
    }

    this.ws.onerror = (event) => {
      console.error('WebSocket error:', event)
      this._emit('error', { message: 'Connection error' })
    }
  }

  _scheduleReconnect() {
    if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnecting = true
      this._emit('reconnecting', { attempt: this.reconnectAttempts + 1 })
      const delay = Math.min(
        this.reconnectDelay * Math.pow(2, this.reconnectAttempts),
        this.maxDelay,
      )
      this.reconnectAttempts++
      this._reconnectTimeout = setTimeout(() => this.connect(), delay)
    } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.reconnecting = false
      this._emit('connection_lost', {})
    }
  }

  disconnect() {
    this.shouldReconnect = false
    this.connected = false
    if (this._reconnectTimeout != null) {
      clearTimeout(this._reconnectTimeout)
      this._reconnectTimeout = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  isConnected() {
    return this.connected && this.ws?.readyState === WebSocket.OPEN
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event).add(callback)
    return () => this.off(event, callback)
  }

  off(event, callback) {
    this.listeners.get(event)?.delete(callback)
  }

  _emit(event, data) {
    this.listeners.get(event)?.forEach((cb) => {
      try {
        cb(data)
      } catch (err) {
        console.error(`WebSocket listener error [${event}]:`, err)
      }
    })
  }

  send(data) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    }
  }
}

let instance = null

export function getWebSocket() {
  if (!instance) {
    instance = new WebSocketManager()
  }
  return instance
}
