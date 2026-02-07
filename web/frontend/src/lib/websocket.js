export class WebSocketManager {
  constructor(url) {
    this.url = url || this._getWsUrl()
    this.ws = null
    this.listeners = new Map()
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 10
    this.reconnectDelay = 1000
    this.maxDelay = 30000
    this.shouldReconnect = true
    this.connected = false
  }

  _getWsUrl() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/ws`
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return

    try {
      this.ws = new WebSocket(this.url)
    } catch (err) {
      console.error('WebSocket connection failed:', err)
      this._scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.connected = true
      this.reconnectAttempts = 0
      this._emit('connected', {})
    }

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        this._emit(message.type, message.data)
        this._emit('message', message)
      } catch (err) {
        console.warn('WebSocket: failed to parse message:', err.message)
      }
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
      const delay = Math.min(
        this.reconnectDelay * Math.pow(2, this.reconnectAttempts),
        this.maxDelay,
      )
      this.reconnectAttempts++
      setTimeout(() => this.connect(), delay)
    }
  }

  disconnect() {
    this.shouldReconnect = false
    this.connected = false
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
    this.listeners.get(event)?.forEach((cb) => cb(data))
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
