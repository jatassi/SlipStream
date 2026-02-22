import { create } from 'zustand'

type AutoSearchTaskState = {
  isRunning: boolean
  totalItems: number
  currentItem: number
  currentTitle: string
  result: AutoSearchTaskResult | null
}

export type AutoSearchTaskResult = {
  totalSearched: number
  found: number
  downloaded: number
  failed: number
  elapsedMs: number
}

type AutoSearchStore = {
  task: AutoSearchTaskState
  handleTaskStarted: (payload: { totalItems: number }) => void
  handleTaskProgress: (payload: {
    currentItem: number
    totalItems: number
    currentTitle: string
  }) => void
  handleTaskCompleted: (payload: AutoSearchTaskResult) => void
  clearResult: () => void
}

const initialTaskState: AutoSearchTaskState = {
  isRunning: false,
  totalItems: 0,
  currentItem: 0,
  currentTitle: '',
  result: null,
}

export const useAutoSearchStore = create<AutoSearchStore>((set) => ({
  task: initialTaskState,

  handleTaskStarted: (payload) => {
    set({
      task: {
        isRunning: true,
        totalItems: payload.totalItems,
        currentItem: 0,
        currentTitle: '',
        result: null,
      },
    })
  },

  handleTaskProgress: (payload) => {
    set((state) => ({
      task: {
        ...state.task,
        currentItem: payload.currentItem,
        totalItems: payload.totalItems,
        currentTitle: payload.currentTitle,
      },
    }))
  },

  handleTaskCompleted: (payload) => {
    set({
      task: {
        isRunning: false,
        totalItems: 0,
        currentItem: 0,
        currentTitle: '',
        result: payload,
      },
    })
  },

  clearResult: () => {
    set((state) => ({
      task: {
        ...state.task,
        result: null,
      },
    }))
  },
}))
