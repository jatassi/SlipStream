import { useEffect, useRef } from 'react'

import { useQueryClient } from '@tanstack/react-query'
import { create } from 'zustand'

import { inboxKeys } from '@/hooks/portal/useInbox'
import { requestKeys } from '@/hooks/portal/useRequests'
import { systemHealthKeys } from '@/hooks/useHealth'
import { historyKeys } from '@/hooks/useHistory'
import { missingKeys } from '@/hooks/useMissing'
import { movieKeys } from '@/hooks/useMovies'
import { queueKeys } from '@/hooks/useQueue'
import { seriesKeys } from '@/hooks/useSeries'
import type { LogEntry } from '@/types/logs'
import type { Activity, ProgressEventType } from '@/types/progress'
import type { QueueResponse } from '@/types/queue'

import { type ArtworkReadyPayload, useArtworkStore } from './artwork'
import { type AutoSearchTaskResult, useAutoSearchStore } from './autosearch'
import { useDevModeStore } from './devmode'
import { useLogsStore } from './logs'
import { usePortalDownloadsStore } from './portalDownloads'
import { useProgressStore } from './progress'
import { useUIStore } from './ui'

export type WSMessage = {
  type: string
  payload: unknown
  timestamp: string
}

type WebSocketState = {
  socket: WebSocket | null
  connected: boolean
  reconnecting: boolean
  lastMessage: WSMessage | null
  lastMessageTime: number
  connect: (force?: boolean) => void
  disconnect: () => void
  send: (message: unknown) => void
}

const WS_URL = `${globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${globalThis.location.host}/ws`
const RECONNECT_DELAY = 3000

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  socket: null,
  connected: false,
  reconnecting: false,
  lastMessage: null,
  lastMessageTime: 0,

  connect: (force = false) => {
    const { socket, reconnecting, lastMessageTime } = get()

    // If already reconnecting, don't start another attempt
    if (reconnecting) {
      return
    }

    // Close existing socket if it's in a bad state (Safari mobile can leave sockets in zombie state)
    if (socket) {
      // Force reconnect: always close and reconnect (used on visibility change for Safari mobile)
      // Also force if no message received in 60+ seconds (likely zombie connection)
      const isStale = lastMessageTime > 0 && Date.now() - lastMessageTime > 60_000
      if (force || isStale) {
        console.log(`WebSocket force reconnect (force=${force}, stale=${isStale})`)
        try {
          socket.close()
        } catch {
          // Ignore close errors
        }
        set({ socket: null, connected: false })
      } else if (socket.readyState === WebSocket.OPEN) {
        return // Already connected and not forcing
      } else {
        // Force close any non-open socket to clean up
        try {
          socket.close()
        } catch {
          // Ignore close errors
        }
      }
    }

    console.log('[WebSocket] Connecting to', WS_URL)
    const ws = new WebSocket(WS_URL)

    ws.addEventListener('open', () => {
      // Only update state if this is still the current socket
      if (get().socket === ws) {
        set({ connected: true, reconnecting: false, lastMessageTime: Date.now() })
        console.log('[WebSocket] Connected successfully')
      }
    })

    ws.addEventListener('close', (event) => {
      // Only update state if this is still the current socket (prevents race with force reconnect)
      if (get().socket === ws) {
        set({ socket: null, connected: false })
        console.log('[WebSocket] Disconnected', {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
        })

        // Auto-reconnect (but not if page is hidden - will reconnect on visibility change)
        if (document.visibilityState !== 'hidden') {
          set({ reconnecting: true }) // Mark that reconnection is scheduled
          setTimeout(() => {
            set({ reconnecting: false }) // Clear before connect so it doesn't bail out
            get().connect()
          }, RECONNECT_DELAY)
        }
      } else {
        console.log('[WebSocket] Old socket closed (ignored)')
      }
    })

    ws.onerror = (error) => {
      console.error('[WebSocket] Error:', error)
    }

    ws.onmessage = (event) => {
      // Only process messages from current socket
      if (get().socket !== ws) {
        return
      }
      try {
        const message: WSMessage = JSON.parse(event.data)
        set({ lastMessage: message, lastMessageTime: Date.now() })
      } catch (error) {
        console.error('[WebSocket] Failed to parse message:', error)
      }
    }

    set({ socket: ws, reconnecting: false })
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

// Debounce delay for request invalidations (ms)
const REQUEST_INVALIDATE_DEBOUNCE_MS = 500

// Hook to handle WebSocket messages and invalidate queries
export function useWebSocketHandler() {
  const queryClient = useQueryClient()
  const lastMessage = useWebSocketStore((state) => state.lastMessage)
  const processedRef = useRef<string | null>(null)
  const requestInvalidateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Handle message types and invalidate appropriate queries
  // IMPORTANT: This must be in a useEffect to avoid state updates during render
  useEffect(() => {
    if (!lastMessage) {
      return
    }

    // Avoid processing the same message twice
    const messageKey = `${lastMessage.type}-${lastMessage.timestamp}`
    if (processedRef.current === messageKey) {
      return
    }
    processedRef.current = messageKey

    // Debug: log all websocket messages
    console.log('[WS Event]', lastMessage.type, lastMessage.payload)

    switch (lastMessage.type) {
      case 'movie:added':
      case 'movie:updated':
      case 'movie:deleted': {
        queryClient.invalidateQueries({ queryKey: movieKeys.all })
        queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
        break
      }
      case 'series:added':
      case 'series:updated':
      case 'series:deleted': {
        queryClient.invalidateQueries({ queryKey: seriesKeys.all })
        queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
        break
      }
      case 'queue:updated': {
        // Force immediate refetch to get current queue state
        queryClient.refetchQueries({ queryKey: queueKeys.all })
        break
      }
      case 'queue:state': {
        // Real-time queue state pushed from backend - update stores directly (no API call)
        const queueResp = lastMessage.payload as QueueResponse
        usePortalDownloadsStore.getState().setQueue(queueResp.items)
        queryClient.setQueryData(queueKeys.list(), queueResp)
        break
      }
      case 'download:completed': {
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
        break
      }
      case 'history:added': {
        queryClient.invalidateQueries({ queryKey: historyKeys.all })
        break
      }
      case 'import:completed': {
        // Note: We intentionally don't invalidate portal search here.
        // Imports can happen frequently (especially with mock downloads),
        // and refetching search results on every import would cause rate limiting.
        // Users can re-search to get updated availability info.
        queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
        break
      }

      // Progress events
      case 'progress:started':
      case 'progress:update':
      case 'progress:completed':
      case 'progress:error':
      case 'progress:cancelled': {
        useProgressStore
          .getState()
          .handleProgressEvent(
            lastMessage.type as ProgressEventType,
            lastMessage.payload as Activity,
          )
        break
      }

      // Artwork events
      case 'artwork:ready': {
        useArtworkStore.getState().notifyReady(lastMessage.payload as ArtworkReadyPayload)
        break
      }

      // Autosearch task events
      case 'autosearch:task:started': {
        useAutoSearchStore
          .getState()
          .handleTaskStarted(lastMessage.payload as { totalItems: number })
        break
      }
      case 'autosearch:task:progress': {
        useAutoSearchStore
          .getState()
          .handleTaskProgress(
            lastMessage.payload as {
              currentItem: number
              totalItems: number
              currentTitle: string
            },
          )
        break
      }
      case 'autosearch:task:completed': {
        useAutoSearchStore
          .getState()
          .handleTaskCompleted(lastMessage.payload as AutoSearchTaskResult)
        break
      }

      // Health events
      case 'health:updated': {
        queryClient.invalidateQueries({ queryKey: systemHealthKeys.all })
        break
      }

      // Developer mode events
      case 'devmode:changed': {
        const { enabled } = lastMessage.payload as { enabled: boolean }
        useDevModeStore.getState().setEnabled(enabled)
        useDevModeStore.getState().setSwitching(false)
        if (!enabled) {
          useUIStore.getState().setGlobalLoading(false)
        }
        // Invalidate all queries to refresh with potentially different database
        queryClient.invalidateQueries()
        break
      }

      case 'devmode:error': {
        useDevModeStore.getState().setSwitching(false)
        // Revert to the opposite of what was requested
        useDevModeStore.getState().setEnabled((lastMessage.payload as { enabled: boolean }).enabled)
        break
      }

      // Request events (portal) - debounced to avoid rapid refetches during auto-approve flow
      case 'request:created':
      case 'request:updated':
      case 'request:deleted': {
        // Clear any pending invalidation and schedule a new one
        if (requestInvalidateTimeoutRef.current) {
          clearTimeout(requestInvalidateTimeoutRef.current)
        }
        requestInvalidateTimeoutRef.current = setTimeout(() => {
          queryClient.invalidateQueries({ queryKey: requestKeys.all })
          requestInvalidateTimeoutRef.current = null
        }, REQUEST_INVALIDATE_DEBOUNCE_MS)
        break
      }

      // Portal inbox notification events
      case 'portal:inbox:created': {
        queryClient.invalidateQueries({ queryKey: inboxKeys.all })
        break
      }

      // Log streaming events
      case 'logs:entry': {
        useLogsStore.getState().addEntry(lastMessage.payload as LogEntry)
        break
      }
    }
  }, [lastMessage, queryClient])

  // Cleanup pending timeout on unmount
  useEffect(() => {
    return () => {
      if (requestInvalidateTimeoutRef.current) {
        clearTimeout(requestInvalidateTimeoutRef.current)
      }
    }
  }, [])

  return lastMessage
}
