import { create } from 'zustand'

import type { Activity, ProgressEventType } from '@/types/progress'

const AUTO_DISMISS_DELAY = 3000 // 3 seconds after completion
const MAX_VISIBLE_ACTIVITIES = 5
const ACTIVITY_BUFFER = MAX_VISIBLE_ACTIVITIES * 2

type ProgressState = {
  activities: Activity[]
  visibleActivities: Activity[]
  activeCount: number

  handleProgressEvent: (eventType: ProgressEventType, activity: Activity) => void
  dismissActivity: (id: string) => void
}

function computeDerivedState(activities: Activity[]) {
  const visibleActivities = activities.slice(0, MAX_VISIBLE_ACTIVITIES)
  const activeCount = activities.filter(
    (a) => a.status === 'in_progress' || a.status === 'pending',
  ).length
  return { visibleActivities, activeCount }
}

function buildUpdatedActivities(current: Activity[], incoming: Activity[]): Activity[] {
  const incomingIds = new Set(incoming.map((a) => a.id))
  const filtered = current.filter((a) => !incomingIds.has(a.id))
  return [...filtered, ...incoming]
    .toSorted((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
    .slice(0, ACTIVITY_BUFFER)
}

const TERMINAL_EVENTS = new Set<ProgressEventType>([
  'progress:completed',
  'progress:error',
  'progress:cancelled',
])

function scheduleDismissIfTerminal(
  eventType: ProgressEventType,
  activityId: string,
  dismiss: (id: string) => void,
) {
  if (TERMINAL_EVENTS.has(eventType)) {
    setTimeout(() => dismiss(activityId), AUTO_DISMISS_DELAY)
  }
}

export const useProgressStore = create<ProgressState>((set, get) => ({
  activities: [],
  visibleActivities: [],
  activeCount: 0,

  handleProgressEvent: (eventType, activity) => {
    const activities = buildUpdatedActivities(get().activities, [activity])
    set({ activities, ...computeDerivedState(activities) })
    scheduleDismissIfTerminal(eventType, activity.id, get().dismissActivity)
  },

  dismissActivity: (id) => {
    set((state) => {
      const activities = state.activities.filter((a) => a.id !== id)
      return { activities, ...computeDerivedState(activities) }
    })
  },
}))
