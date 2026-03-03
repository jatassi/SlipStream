export type WSMessage = {
  type: string
  payload: unknown
  timestamp: string
}

export type WebSocketState = {
  socket: WebSocket | null
  connected: boolean
  reconnecting: boolean
  reconnectAttempts: number
  lastMessage: WSMessage | null
  lastMessageTime: number
  connect: (force?: boolean) => void
  disconnect: () => void
  send: (message: unknown) => void
}

export const WS_URL = `${globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${globalThis.location.host}/ws`
export const INITIAL_RECONNECT_DELAY = 3000
export const MAX_RECONNECT_DELAY = 60_000
export const MAX_RECONNECT_ATTEMPTS = 20
