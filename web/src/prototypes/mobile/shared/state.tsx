import type { ReactNode } from 'react'
import { useEffect, useMemo, useReducer } from 'react'

import type { MobileStore } from './mobile-state-context'
import { MobileStateContext } from './mobile-state-context'
import { LIBRARY, QUEUE } from './mock-data'
import type { MediaItem, QueueItem } from './types'

type State = {
  queue: QueueItem[]
  library: MediaItem[]
}

type Action =
  | { type: 'togglePause'; id: string }
  | { type: 'remove'; id: string }
  | { type: 'toggleMonitored'; id: number }
  | { type: 'tick' }

const TICK_MS = 1000
const TICK_PROGRESS = 0.35

function tickQueue(queue: QueueItem[]): QueueItem[] {
  return queue.map((q) => {
    if (q.state !== 'downloading' || q.progress >= 100) {
      return q
    }
    const progress = Math.min(100, q.progress + TICK_PROGRESS)
    return { ...q, progress, state: progress >= 100 ? 'importing' : 'downloading', speedMbps: q.speedMbps + (Math.random() - 0.5) * 1.6 }
  })
}

function togglePause(q: QueueItem): QueueItem {
  if (q.state === 'paused') {
    return { ...q, state: 'downloading', speedMbps: 31.4, etaMin: 9 }
  }
  return { ...q, state: 'paused', speedMbps: 0, etaMin: 0 }
}

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'togglePause': {
      return { ...state, queue: state.queue.map((q) => (q.id === action.id ? togglePause(q) : q)) }
    }
    case 'remove': {
      return { ...state, queue: state.queue.filter((q) => q.id !== action.id) }
    }
    case 'toggleMonitored': {
      return { ...state, library: state.library.map((m) => (m.id === action.id ? { ...m, monitored: !m.monitored } : m)) }
    }
    case 'tick': {
      return { ...state, queue: tickQueue(state.queue) }
    }
  }
}

export function MobileStateProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, { queue: QUEUE, library: LIBRARY })

  useEffect(() => {
    const timer = setInterval(() => dispatch({ type: 'tick' }), TICK_MS)
    return () => clearInterval(timer)
  }, [])

  const store = useMemo<MobileStore>(
    () => ({
      ...state,
      togglePause: (id) => dispatch({ type: 'togglePause', id }),
      removeFromQueue: (id) => dispatch({ type: 'remove', id }),
      toggleMonitored: (id) => dispatch({ type: 'toggleMonitored', id }),
      media: (id) => {
        const item = state.library.find((m) => m.id === id)
        if (!item) {
          throw new Error(`Unknown media id ${id}`)
        }
        return item
      },
    }),
    [state],
  )

  return <MobileStateContext.Provider value={store}>{children}</MobileStateContext.Provider>
}
