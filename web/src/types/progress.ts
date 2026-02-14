// Progress activity types matching the backend definitions

export type ActivityType = 'scan' | 'download' | 'import' | 'metadata-refresh' | 'file-operation'

export type ActivityStatus = 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled'

export type Activity = {
  id: string
  type: ActivityType
  title: string
  subtitle: string
  progress: number // 0-100, -1 for indeterminate
  status: ActivityStatus
  startedAt: string
  completedAt: string | null
  metadata: Record<string, unknown>
}

export type ProgressEventType =
  | 'progress:started'
  | 'progress:update'
  | 'progress:completed'
  | 'progress:error'
  | 'progress:cancelled'
