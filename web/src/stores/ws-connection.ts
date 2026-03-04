import { getAdminAuthToken } from '@/api/client'

import type { WebSocketState, WSMessage } from './ws-types'
import {
  INITIAL_RECONNECT_DELAY,
  MAX_RECONNECT_ATTEMPTS,
  MAX_RECONNECT_DELAY,
  WS_URL,
} from './ws-types'

type GetState = () => WebSocketState
type SetState = (
  partial:
    | Partial<WebSocketState>
    | ((state: WebSocketState) => Partial<WebSocketState>),
) => void

function closeSocketSilently(socket: WebSocket): void {
  try {
    socket.close()
  } catch {
    // Ignore close errors
  }
}

function shouldForceReconnect(
  force: boolean,
  lastMessageTime: number,
): boolean {
  const isStale = lastMessageTime > 0 && Date.now() - lastMessageTime > 60_000
  return force || isStale
}

function handleExistingSocket(
  set: SetState,
  socket: WebSocket,
  options: { force: boolean; lastMessageTime: number },
): boolean {
  if (shouldForceReconnect(options.force, options.lastMessageTime)) {
    closeSocketSilently(socket)
    set({ socket: null, connected: false })
    return false
  }

  if (socket.readyState === WebSocket.OPEN) {
    return true
  }

  closeSocketSilently(socket)
  return false
}

function attachListeners(
  ws: WebSocket,
  get: GetState,
  set: SetState,
): void {
  ws.addEventListener('open', () => {
    if (get().socket === ws) {
      set({ connected: true, reconnecting: false, reconnectAttempts: 0, lastMessageTime: Date.now() })
    }
  })

  ws.addEventListener('close', () => {
    if (get().socket !== ws) {
      return
    }
    set({ socket: null, connected: false })
    scheduleReconnect(get, set)
  })

  ws.addEventListener('message', (event) => {
    if (get().socket !== ws) {return}
    try {
      const eventData: string =
        typeof event.data === 'string' ? event.data : String(event.data)
      const message: WSMessage = JSON.parse(eventData) as WSMessage
      set({ lastMessage: message, lastMessageTime: Date.now() })
    } catch {
      // Ignore malformed messages
    }
  })
}

function computeReconnectDelay(attempts: number): number {
  const base = INITIAL_RECONNECT_DELAY * Math.pow(2, attempts)
  const withJitter = Math.min(base, MAX_RECONNECT_DELAY) * (1 + Math.random() * 0.3)
  return Math.min(withJitter, MAX_RECONNECT_DELAY)
}

function scheduleReconnect(get: GetState, set: SetState): void {
  const { reconnectAttempts } = get()
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    set({ reconnecting: false })
    return
  }
  const delay = computeReconnectDelay(reconnectAttempts)
  set({ reconnecting: true, reconnectAttempts: reconnectAttempts + 1 })
  setTimeout(() => {
    set({ reconnecting: false })
    get().connect()
  }, delay)
}

export function createConnect(
  get: GetState,
  set: SetState,
): (force?: boolean) => void {
  return (force = false) => {
    const { socket, reconnecting, lastMessageTime } = get()
    if (reconnecting) {
      if (!force) {return}
      set({ reconnecting: false, reconnectAttempts: 0 })
    }

    // Don't connect if there's no admin token — user is not logged in.
    const token = getAdminAuthToken()
    if (!token) {return}

    if (socket) {
      const alreadyConnected = handleExistingSocket(set, socket, {
        force,
        lastMessageTime,
      })
      if (alreadyConnected) {return}
    }

    // Pass the token via the Sec-WebSocket-Protocol header (the only way
    // the browser WebSocket API supports custom headers).
    const ws = new WebSocket(WS_URL, [token])
    attachListeners(ws, get, set)
    set({ socket: ws, reconnecting: false })
  }
}

export function createDisconnect(
  get: GetState,
  set: SetState,
): () => void {
  return () => {
    const { socket } = get()
    if (socket) {
      socket.close()
      set({ socket: null, connected: false })
    }
  }
}

export function createSend(get: GetState): (message: unknown) => void {
  return (message) => {
    const { socket, connected } = get()
    if (socket && connected) {
      socket.send(JSON.stringify(message))
    }
  }
}
