import { create } from 'zustand'
import { useQueryClient } from '@tanstack/react-query'
import { movieKeys } from '@/hooks/useMovies'
import { seriesKeys } from '@/hooks/useSeries'
import { queueKeys } from '@/hooks/useQueue'
import { historyKeys } from '@/hooks/useHistory'
import { systemHealthKeys } from '@/hooks/useHealth'
import { useProgressStore } from './progress'
import { useArtworkStore, type ArtworkReadyPayload } from './artwork'
import { useAutoSearchStore, type AutoSearchTaskResult } from './autosearch'
import type { Activity, ProgressEventType } from '@/types/progress'

export interface WSMessage {
  type: string
  payload: unknown
  timestamp: string
}

interface WebSocketState {
  socket: WebSocket | null
  connected: boolean
  reconnecting: boolean
  lastMessage: WSMessage | null
  connect: () => void
  disconnect: () => void
  send: (message: unknown) => void
}

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
const RECONNECT_DELAY = 3000

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  socket: null,
  connected: false,
  reconnecting: false,
  lastMessage: null,

  connect: () => {
    const { socket, reconnecting } = get()
    if (socket?.readyState === WebSocket.OPEN || reconnecting) {
      return
    }

    const ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      set({ socket: ws, connected: true, reconnecting: false })
      console.log('WebSocket connected')
    }

    ws.onclose = () => {
      set({ socket: null, connected: false })
      console.log('WebSocket disconnected')

      // Auto-reconnect
      setTimeout(() => {
        set({ reconnecting: true })
        get().connect()
      }, RECONNECT_DELAY)
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data)
        set({ lastMessage: message })

        // The message handler will be called via the useWebSocketHandler hook
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    set({ socket: ws })
  },

  disconnect: () => {
    const { socket } = get()
    if (socket) {
      socket.close()
      set({ socket: null, connected: false })
    }
  },

  send: (message) => {
    const { socket, connected } = get()
    if (socket && connected) {
      socket.send(JSON.stringify(message))
    }
  },
}))

// Hook to handle WebSocket messages and invalidate queries
export function useWebSocketHandler() {
  const queryClient = useQueryClient()
  const lastMessage = useWebSocketStore((state) => state.lastMessage)

  // Handle message types and invalidate appropriate queries
  if (lastMessage) {
    switch (lastMessage.type) {
      case 'movie:added':
      case 'movie:updated':
      case 'movie:deleted':
        queryClient.invalidateQueries({ queryKey: movieKeys.all })
        break
      case 'series:added':
      case 'series:updated':
      case 'series:deleted':
        queryClient.invalidateQueries({ queryKey: seriesKeys.all })
        break
      case 'queue:updated':
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
        break
      case 'history:added':
        queryClient.invalidateQueries({ queryKey: historyKeys.all })
        break
      case 'download:progress':
        queryClient.invalidateQueries({ queryKey: queueKeys.list() })
        break

      // Progress events
      case 'progress:started':
      case 'progress:update':
      case 'progress:completed':
      case 'progress:error':
      case 'progress:cancelled':
        useProgressStore.getState().handleProgressEvent(
          lastMessage.type as ProgressEventType,
          lastMessage.payload as Activity
        )
        break

      // Artwork events
      case 'artwork:ready':
        useArtworkStore.getState().notifyReady(
          lastMessage.payload as ArtworkReadyPayload
        )
        break

      // Autosearch task events
      case 'autosearch:task:started':
        useAutoSearchStore.getState().handleTaskStarted(
          lastMessage.payload as { totalItems: number }
        )
        break
      case 'autosearch:task:progress':
        useAutoSearchStore.getState().handleTaskProgress(
          lastMessage.payload as { currentItem: number; totalItems: number; currentTitle: string }
        )
        break
      case 'autosearch:task:completed':
        useAutoSearchStore.getState().handleTaskCompleted(
          lastMessage.payload as AutoSearchTaskResult
        )
        break

      // Health events
      case 'health:updated':
        queryClient.invalidateQueries({ queryKey: systemHealthKeys.all })
        break
    }
  }

  return lastMessage
}
