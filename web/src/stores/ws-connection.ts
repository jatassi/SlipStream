import type { WebSocketState, WSMessage } from './ws-types'
import { RECONNECT_DELAY, WS_URL } from './ws-types'

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
      set({ connected: true, reconnecting: false, lastMessageTime: Date.now() })
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

function scheduleReconnect(get: GetState, set: SetState): void {
  if (document.visibilityState === 'hidden') {return}
  set({ reconnecting: true })
  setTimeout(() => {
    set({ reconnecting: false })
    get().connect()
  }, RECONNECT_DELAY)
}

export function createConnect(
  get: GetState,
  set: SetState,
): (force?: boolean) => void {
  return (force = false) => {
    const { socket, reconnecting, lastMessageTime } = get()
    if (reconnecting) {return}

    if (socket) {
      const alreadyConnected = handleExistingSocket(set, socket, {
        force,
        lastMessageTime,
      })
      if (alreadyConnected) {return}
    }

    const ws = new WebSocket(WS_URL)
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
