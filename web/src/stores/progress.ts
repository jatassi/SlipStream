import { create } from 'zustand'

import type { Activity, ProgressEventType } from '@/types/progress'

const AUTO_DISMISS_DELAY = 3000 // 3 seconds after completion
const MAX_VISIBLE_ACTIVITIES = 5

type ProgressState = {
  activities: Activity[]
  // Derived state stored directly to avoid selector issues
  visibleActivities: Activity[]
  activeCount: number

  // Actions
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

export const useProgressStore = create<ProgressState>((set, get) => ({
  activities: [],
  visibleActivities: [],
  activeCount: 0,

  handleProgressEvent: (eventType, activity) => {
    set((state) => {
      // Remove existing activity with same ID, then add updated one
      const filtered = state.activities.filter((a) => a.id !== activity.id)
      const activities = [...filtered, activity]
        .toSorted((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
        .slice(0, MAX_VISIBLE_ACTIVITIES * 2) // Keep some buffer

      // Schedule auto-dismiss for completed activities
      if (
        eventType === 'progress:completed' ||
        eventType === 'progress:error' ||
        eventType === 'progress:cancelled'
      ) {
        setTimeout(() => {
          get().dismissActivity(activity.id)
        }, AUTO_DISMISS_DELAY)
      }

      return { activities, ...computeDerivedState(activities) }
    })
  },

  dismissActivity: (id) => {
    set((state) => {
      const activities = state.activities.filter((a) => a.id !== id)
      return { activities, ...computeDerivedState(activities) }
    })
  },
}))
