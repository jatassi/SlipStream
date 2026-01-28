import { create } from 'zustand'
import type { LogEntry, LogLevel } from '@/types/logs'

const MAX_ENTRIES = 2000
const LOG_LEVEL_PRIORITY: Record<string, number> = {
  trace: 0,
  debug: 1,
  info: 2,
  warn: 3,
  error: 4,
  fatal: 5,
}

interface LogsState {
  entries: LogEntry[]
  filterLevel: LogLevel | 'all'
  searchText: string
  isPaused: boolean
  autoScroll: boolean

  addEntry: (entry: LogEntry) => void
  setEntries: (entries: LogEntry[]) => void
  setFilterLevel: (level: LogLevel | 'all') => void
  setSearchText: (text: string) => void
  togglePaused: () => void
  toggleAutoScroll: () => void
  clear: () => void

  getFilteredEntries: () => LogEntry[]
}

export const useLogsStore = create<LogsState>((set, get) => ({
  entries: [],
  filterLevel: 'all',
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

  setFilterLevel: (level) => set({ filterLevel: level }),

  setSearchText: (text) => set({ searchText: text }),

  togglePaused: () => set((state) => ({ isPaused: !state.isPaused })),

  toggleAutoScroll: () => set((state) => ({ autoScroll: !state.autoScroll })),

  clear: () => set({ entries: [] }),

  getFilteredEntries: () => {
    const { entries, filterLevel, searchText } = get()

    return entries.filter((entry) => {
      if (filterLevel !== 'all') {
        const entryPriority = LOG_LEVEL_PRIORITY[entry.level] ?? 0
        const filterPriority = LOG_LEVEL_PRIORITY[filterLevel] ?? 0
        if (entryPriority < filterPriority) {
          return false
        }
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
