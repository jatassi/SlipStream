import type { LogEntry } from '@/types/logs'
import type { Activity } from '@/types/progress'
import type { QueueResponse } from '@/types/queue'

import type { ArtworkReadyPayload } from './artwork'
import type { AutoSearchTaskResult } from './autosearch'

// Entity events carry structured module/entity fields alongside the legacy type
type EntityEventFields = {
  module: string
  entityType: string
  entityId: number
  action: string
}

// Library events just trigger invalidation, payload is unused
type LibraryMessage = {
  type: 'movie:added' | 'movie:updated' | 'movie:deleted' | 'series:added' | 'series:updated' | 'series:deleted'
  payload: unknown
  timestamp: string
} & EntityEventFields

type QueueUpdatedMessage = {
  type: 'queue:updated'
  payload: unknown
  timestamp: string
}

type QueueStateMessage = {
  type: 'queue:state'
  payload: QueueResponse
  timestamp: string
}

type DownloadCompletedMessage = {
  type: 'download:completed'
  payload: unknown
  timestamp: string
}

type HistoryMessage = {
  type: 'history:added'
  payload: unknown
  timestamp: string
}

type ImportMessage = {
  type: 'import:completed'
  payload: unknown
  timestamp: string
}

type ProgressMessage = {
  type: `progress:${'started' | 'update' | 'completed' | 'error' | 'cancelled'}`
  payload: Activity
  timestamp: string
}

type ArtworkMessage = {
  type: 'artwork:ready'
  payload: ArtworkReadyPayload
  timestamp: string
}

type AutoSearchStartedMessage = {
  type: 'autosearch:task:started'
  payload: { totalItems: number }
  timestamp: string
}

type AutoSearchProgressMessage = {
  type: 'autosearch:task:progress'
  payload: { currentItem: number; totalItems: number; currentTitle: string }
  timestamp: string
}

type AutoSearchCompletedMessage = {
  type: 'autosearch:task:completed'
  payload: AutoSearchTaskResult
  timestamp: string
}

type SchedulerMessage = {
  type: 'rss-sync:started' | 'rss-sync:completed' | 'rss-sync:failed' | 'scheduler:task:started' | 'scheduler:task:completed'
  payload: unknown
  timestamp: string
}

type HealthMessage = {
  type: 'health:updated'
  payload: unknown
  timestamp: string
}

type DevModeMessage = {
  type: 'devmode:changed' | 'devmode:error'
  payload: { enabled: boolean }
  timestamp: string
}

type RequestMessage = {
  type: 'request:created' | 'request:updated' | 'request:deleted'
  payload: unknown
  timestamp: string
}

type PortalInboxMessage = {
  type: 'portal:inbox:created'
  payload: unknown
  timestamp: string
}

type LogsMessage = {
  type: 'logs:entry'
  payload: LogEntry
  timestamp: string
}

export type WSMessage =
  | LibraryMessage
  | QueueUpdatedMessage
  | QueueStateMessage
  | DownloadCompletedMessage
  | HistoryMessage
  | ImportMessage
  | ProgressMessage
  | ArtworkMessage
  | AutoSearchStartedMessage
  | AutoSearchProgressMessage
  | AutoSearchCompletedMessage
  | SchedulerMessage
  | HealthMessage
  | DevModeMessage
  | RequestMessage
  | PortalInboxMessage
  | LogsMessage

export type WSMessageType = WSMessage['type']

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
