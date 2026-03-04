import type { QueryClient } from '@tanstack/react-query'

import { adminRequestKeys } from '@/hooks/admin/use-admin-requests'
import { inboxKeys } from '@/hooks/portal/use-inbox'
import { requestKeys } from '@/hooks/portal/use-requests'
import { systemHealthKeys } from '@/hooks/use-health'
import { historyKeys } from '@/hooks/use-history'
import { missingKeys } from '@/hooks/use-missing'
import { movieKeys } from '@/hooks/use-movies'
import { queueKeys } from '@/hooks/use-queue'
import { schedulerKeys } from '@/hooks/use-scheduler'
import { seriesKeys } from '@/hooks/use-series'
import type { ProgressEventType } from '@/types/progress'

import { useArtworkStore } from './artwork'
import { useAutoSearchStore } from './autosearch'
import { useDevModeStore } from './devmode'
import { useLogsStore } from './logs'
import { usePortalDownloadsStore } from './portal-downloads'
import { useProgressStore } from './progress'
import { useUIStore } from './ui'
import type { WSMessage, WSMessageType } from './ws-types'

export type DispatchContext = {
  queryClient: QueryClient
  requestTimeoutRef: React.RefObject<ReturnType<typeof setTimeout> | null>
  requestDebounceMs: number
}

function handleLibraryEvent(
  queryClient: QueryClient,
  type: string,
): void {
  const isMovie = type.startsWith('movie:')
  const keys = isMovie ? movieKeys.all : seriesKeys.all
  void queryClient.invalidateQueries({ queryKey: keys })
  void queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
}

function handleQueueEvent(
  queryClient: QueryClient,
  message: WSMessage,
): void {
  if (message.type === 'queue:state') {
    // Narrows to QueueStateMessage — payload is QueueResponse
    usePortalDownloadsStore.getState().setQueue(message.payload.items)
    queryClient.setQueryData(queueKeys.list(), message.payload)
  } else {
    void queryClient.refetchQueries({ queryKey: queueKeys.all })
  }
}

type ProgressMessage = Extract<WSMessage, { type: `progress:${string}` }>

function handleProgressEvent(message: WSMessage): void {
  // Only called for progress:* message types; cast to the narrowed variant
  const progressMsg = message as ProgressMessage
  useProgressStore
    .getState()
    .handleProgressEvent(
      progressMsg.type as ProgressEventType,
      progressMsg.payload,
    )
}

function handleAutoSearchEvent(message: WSMessage): void {
  const store = useAutoSearchStore.getState()
  switch (message.type) {
    case 'autosearch:task:started': {
      // Narrows to AutoSearchStartedMessage — payload is { totalItems: number }
      store.handleTaskStarted(message.payload)
      break
    }
    case 'autosearch:task:progress': {
      // Narrows to AutoSearchProgressMessage — payload is { currentItem, totalItems, currentTitle }
      store.handleTaskProgress(message.payload)
      break
    }
    case 'autosearch:task:completed': {
      // Narrows to AutoSearchCompletedMessage — payload is AutoSearchTaskResult
      store.handleTaskCompleted(message.payload)
      break
    }
  }
}

function handleDevModeEvent(
  queryClient: QueryClient,
  message: WSMessage,
): void {
  if (message.type === 'devmode:changed') {
    // Narrows to DevModeMessage — payload is { enabled: boolean }
    useDevModeStore.getState().setEnabled(message.payload.enabled)
    useDevModeStore.getState().setSwitching(false)
    if (!message.payload.enabled) {
      useUIStore.getState().setGlobalLoading(false)
    }
    void queryClient.invalidateQueries()
  } else {
    // devmode:error — also a DevModeMessage with { enabled: boolean }
    useDevModeStore.getState().setSwitching(false)
    useDevModeStore
      .getState()
      .setEnabled((message.payload as { enabled: boolean }).enabled)
  }
}

function handleRequestEvent(ctx: DispatchContext): void {
  if (ctx.requestTimeoutRef.current) {
    clearTimeout(ctx.requestTimeoutRef.current)
  }
  ctx.requestTimeoutRef.current = setTimeout(() => {
    void ctx.queryClient.invalidateQueries({ queryKey: requestKeys.all })
    void ctx.queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    ctx.requestTimeoutRef.current = null
  }, ctx.requestDebounceMs)
}

type MessageHandler = (message: WSMessage, ctx: DispatchContext) => void

const libraryHandler: MessageHandler = (message, ctx) =>
  handleLibraryEvent(ctx.queryClient, message.type)

const queueHandler: MessageHandler = (message, ctx) =>
  handleQueueEvent(ctx.queryClient, message)

const downloadCompletedHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: queueKeys.all })
}

const historyHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: historyKeys.all })
}

const importHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
}

const progressHandler: MessageHandler = (message) =>
  handleProgressEvent(message)

const artworkHandler: MessageHandler = (message) => {
  if (message.type === 'artwork:ready') {
    // Narrows to ArtworkMessage — payload is ArtworkReadyPayload
    useArtworkStore.getState().notifyReady(message.payload)
  }
}

const autoSearchHandler: MessageHandler = (message, ctx) => {
  handleAutoSearchEvent(message)
  void ctx.queryClient.invalidateQueries({ queryKey: schedulerKeys.tasks() })
}

const healthHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: systemHealthKeys.all })
}

const devModeHandler: MessageHandler = (message, ctx) =>
  handleDevModeEvent(ctx.queryClient, message)

const requestHandler: MessageHandler = (_message, ctx) =>
  handleRequestEvent(ctx)

const portalInboxHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: inboxKeys.all })
}

const schedulerTaskHandler: MessageHandler = (_message, ctx) => {
  void ctx.queryClient.invalidateQueries({ queryKey: schedulerKeys.tasks() })
}

const logsHandler: MessageHandler = (message) => {
  if (message.type === 'logs:entry') {
    // Narrows to LogsMessage — payload is LogEntry
    useLogsStore.getState().addEntry(message.payload)
  }
}

const handlerMap: Partial<Record<WSMessageType, MessageHandler>> = {
  'movie:added': libraryHandler,
  'movie:updated': libraryHandler,
  'movie:deleted': libraryHandler,
  'series:added': libraryHandler,
  'series:updated': libraryHandler,
  'series:deleted': libraryHandler,
  'queue:updated': queueHandler,
  'queue:state': queueHandler,
  'download:completed': downloadCompletedHandler,
  'history:added': historyHandler,
  'import:completed': importHandler,
  'progress:started': progressHandler,
  'progress:update': progressHandler,
  'progress:completed': progressHandler,
  'progress:error': progressHandler,
  'progress:cancelled': progressHandler,
  'artwork:ready': artworkHandler,
  'autosearch:task:started': autoSearchHandler,
  'autosearch:task:progress': autoSearchHandler,
  'autosearch:task:completed': autoSearchHandler,
  'rss-sync:started': schedulerTaskHandler,
  'rss-sync:completed': schedulerTaskHandler,
  'rss-sync:failed': schedulerTaskHandler,
  'scheduler:task:started': schedulerTaskHandler,
  'scheduler:task:completed': schedulerTaskHandler,
  'health:updated': healthHandler,
  'devmode:changed': devModeHandler,
  'devmode:error': devModeHandler,
  'request:created': requestHandler,
  'request:updated': requestHandler,
  'request:deleted': requestHandler,
  'portal:inbox:created': portalInboxHandler,
  'logs:entry': logsHandler,
}

export function dispatchWSMessage(
  message: WSMessage,
  ctx: DispatchContext,
): void {
  const handler = handlerMap[message.type]
  handler?.(message, ctx)
}
