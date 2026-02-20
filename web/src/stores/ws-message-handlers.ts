import type { QueryClient } from '@tanstack/react-query'

import { adminRequestKeys } from '@/hooks/admin/use-admin-requests'
import { inboxKeys } from '@/hooks/portal/use-inbox'
import { requestKeys } from '@/hooks/portal/use-requests'
import { systemHealthKeys } from '@/hooks/use-health'
import { historyKeys } from '@/hooks/use-history'
import { missingKeys } from '@/hooks/use-missing'
import { movieKeys } from '@/hooks/use-movies'
import { queueKeys } from '@/hooks/use-queue'
import { seriesKeys } from '@/hooks/use-series'
import type { LogEntry } from '@/types/logs'
import type { Activity, ProgressEventType } from '@/types/progress'
import type { QueueResponse } from '@/types/queue'

import { type ArtworkReadyPayload, useArtworkStore } from './artwork'
import { type AutoSearchTaskResult, useAutoSearchStore } from './autosearch'
import { useDevModeStore } from './devmode'
import { useDownloadingStore } from './downloading'
import { useLogsStore } from './logs'
import { usePortalDownloadsStore } from './portal-downloads'
import { useProgressStore } from './progress'
import { useUIStore } from './ui'
import type { WSMessage } from './ws-types'

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
    const queueResp = message.payload as QueueResponse
    useDownloadingStore.getState().setQueueItems(queueResp.items)
    usePortalDownloadsStore.getState().setQueue(queueResp.items)
    queryClient.setQueryData(queueKeys.list(), queueResp)
  } else {
    void queryClient.refetchQueries({ queryKey: queueKeys.all })
  }
}

function handleProgressEvent(message: WSMessage): void {
  useProgressStore
    .getState()
    .handleProgressEvent(
      message.type as ProgressEventType,
      message.payload as Activity,
    )
}

function handleAutoSearchEvent(message: WSMessage): void {
  const store = useAutoSearchStore.getState()
  switch (message.type) {
    case 'autosearch:task:started': {
      store.handleTaskStarted(message.payload as { totalItems: number })
      break
    }
    case 'autosearch:task:progress': {
      store.handleTaskProgress(
        message.payload as {
          currentItem: number
          totalItems: number
          currentTitle: string
        },
      )
      break
    }
    case 'autosearch:task:completed': {
      store.handleTaskCompleted(message.payload as AutoSearchTaskResult)
      break
    }
  }
}

function handleDevModeEvent(
  queryClient: QueryClient,
  message: WSMessage,
): void {
  if (message.type === 'devmode:changed') {
    const { enabled } = message.payload as { enabled: boolean }
    useDevModeStore.getState().setEnabled(enabled)
    useDevModeStore.getState().setSwitching(false)
    if (!enabled) {
      useUIStore.getState().setGlobalLoading(false)
    }
    void queryClient.invalidateQueries()
  } else {
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
  useArtworkStore
    .getState()
    .notifyReady(message.payload as ArtworkReadyPayload)
}

const autoSearchHandler: MessageHandler = (message) =>
  handleAutoSearchEvent(message)

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

const logsHandler: MessageHandler = (message) => {
  useLogsStore.getState().addEntry(message.payload as LogEntry)
}

const handlerMap: Partial<Record<string, MessageHandler>> = {
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
  console.log('[WS Event]', message.type, message.payload)
  const handler = handlerMap[message.type]
  handler?.(message, ctx)
}
