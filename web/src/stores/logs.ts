import { create } from 'zustand'
import type { LogEntry, LogLevel } from '@/types/logs'

const MAX_ENTRIES = 2000
export const ALL_LOG_LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error']

interface LogsState {
  entries: LogEntry[]
  filterLevels: LogLevel[]
  searchText: string
  isPaused: boolean
  autoScroll: boolean

  addEntry: (entry: LogEntry) => void
  setEntries: (entries: LogEntry[]) => void
  toggleFilterLevel: (level: LogLevel) => void
  resetFilterLevels: () => void
  setSearchText: (text: string) => void
  togglePaused: () => void
  setAutoScroll: (value: boolean) => void
  toggleAutoScroll: () => void
  clear: () => void

  getFilteredEntries: () => LogEntry[]
}

export const useLogsStore = create<LogsState>((set, get) => ({
  entries: [],
  filterLevels: [...ALL_LOG_LEVELS],
  searchText: '',
  isPaused: false,
  autoScroll: true,

  addEntry: (entry) => {
    if (get().isPaused) return

    set((state) => {
      const newEntries = [...state.entries, entry]
      if (newEntries.length > MAX_ENTRIES) {
        newEntries.shift()
      }
      return { entries: newEntries }
    })
  },

  setEntries: (entries) => {
    set({ entries: entries.slice(-MAX_ENTRIES) })
  },

  toggleFilterLevel: (level) =>
    set((state) => {
      const has = state.filterLevels.includes(level)
      return {
        filterLevels: has
          ? state.filterLevels.filter((l) => l !== level)
          : [...state.filterLevels, level],
      }
    }),

  resetFilterLevels: () => set({ filterLevels: [...ALL_LOG_LEVELS] }),

  setSearchText: (text) => set({ searchText: text }),

  togglePaused: () => set((state) => ({ isPaused: !state.isPaused })),

  setAutoScroll: (value) => set({ autoScroll: value }),

  toggleAutoScroll: () => set((state) => ({ autoScroll: !state.autoScroll })),

  clear: () => set({ entries: [] }),

  getFilteredEntries: () => {
    const { entries, filterLevels, searchText } = get()

    return entries.filter((entry) => {
      if (filterLevels.length < ALL_LOG_LEVELS.length && !filterLevels.includes(entry.level as LogLevel)) {
        return false
      }

      if (searchText) {
        const lowerSearch = searchText.toLowerCase()
        const matchesMessage = entry.message.toLowerCase().includes(lowerSearch)
        const matchesComponent = entry.component?.toLowerCase().includes(lowerSearch)
        const matchesFields = Object.values(entry.fields || {}).some(
          (v) => String(v).toLowerCase().includes(lowerSearch)
        )
        if (!matchesMessage && !matchesComponent && !matchesFields) {
          return false
        }
      }

      return true
    })
  },
}))
