import { useEffect, useRef } from 'react'

import { useQueryClient } from '@tanstack/react-query'
import { create } from 'zustand'

import { createConnect, createDisconnect, createSend } from './ws-connection'
import { type DispatchContext, dispatchWSMessage } from './ws-message-handlers'
import type { WebSocketState } from './ws-types'

export type { WSMessage } from './ws-types'

const REQUEST_INVALIDATE_DEBOUNCE_MS = 500

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  socket: null,
  connected: false,
  reconnecting: false,
  lastMessage: null,
  lastMessageTime: 0,
  connect: createConnect(get, set),
  disconnect: createDisconnect(get, set),
  send: createSend(get),
}))

export function useWebSocketHandler() {
  const queryClient = useQueryClient()
  const lastMessage = useWebSocketStore((state) => state.lastMessage)
  const processedRef = useRef<string | null>(null)
  const requestInvalidateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (!lastMessage) {return}

    const messageKey = `${lastMessage.type}-${lastMessage.timestamp}`
    if (processedRef.current === messageKey) {return}
    processedRef.current = messageKey

    const ctx: DispatchContext = {
      queryClient,
      requestTimeoutRef: requestInvalidateTimeoutRef,
      requestDebounceMs: REQUEST_INVALIDATE_DEBOUNCE_MS,
    }
    dispatchWSMessage(lastMessage, ctx)
  }, [lastMessage, queryClient])

  useEffect(() => {
    const timeoutRef = requestInvalidateTimeoutRef.current
    return () => {
      if (timeoutRef) {
        clearTimeout(timeoutRef)
      }
    }
  }, [])

  return lastMessage
}
