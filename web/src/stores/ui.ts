import type { StoreApi } from 'zustand'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { getDefaultVisibleColumns, MOVIE_COLUMNS, SERIES_COLUMNS } from '@/lib/table-columns'

type Notification = {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
}

type UIState = {
  // Sidebar
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  setSidebarCollapsed: (collapsed: boolean) => void

  // Sidebar menu expansion state
  expandedMenus: Record<string, boolean>
  toggleMenu: (menuId: string) => void
  setMenuExpanded: (menuId: string, expanded: boolean) => void

  // Theme
  theme: 'light' | 'dark' | 'system'
  setTheme: (theme: 'light' | 'dark' | 'system') => void

  // View preferences
  moviesView: 'grid' | 'table'
  seriesView: 'grid' | 'table'
  posterSize: number // 100-250, represents min-width in pixels
  movieTableColumns: string[]
  seriesTableColumns: string[]
  movieSortField: string
  movieSortDirection: 'asc' | 'desc'
  seriesSortField: string
  seriesSortDirection: 'asc' | 'desc'
  setMoviesView: (view: 'grid' | 'table') => void
  setSeriesView: (view: 'grid' | 'table') => void
  setPosterSize: (size: number) => void
  setMovieTableColumns: (cols: string[]) => void
  setSeriesTableColumns: (cols: string[]) => void
  setMovieSortField: (field: string) => void
  setMovieSortDirection: (direction: 'asc' | 'desc') => void
  setSeriesSortField: (field: string) => void
  setSeriesSortDirection: (direction: 'asc' | 'desc') => void

  // Module-keyed view preferences
  moduleViewPrefs: Partial<Record<string, 'grid' | 'table'>>
  moduleTableColumns: Partial<Record<string, string[]>>
  getModuleView: (moduleId: string) => 'grid' | 'table'
  setModuleView: (moduleId: string, view: 'grid' | 'table') => void
  getModuleTableColumns: (moduleId: string) => string[]
  setModuleTableColumns: (moduleId: string, cols: string[]) => void

  // Global loading override (dev tool)
  globalLoading: boolean
  setGlobalLoading: (loading: boolean) => void

  // Notifications
  notifications: Notification[]
  addNotification: (notification: Omit<Notification, 'id'>) => void
  dismissNotification: (id: string) => void
  clearNotifications: () => void
}

function resolveModuleView(state: UIState, moduleId: string): 'grid' | 'table' {
  const stored = state.moduleViewPrefs[moduleId]
  if (stored) { return stored }
  if (moduleId === 'movie') { return state.moviesView }
  if (moduleId === 'tv') { return state.seriesView }
  return 'grid'
}

function resolveModuleTableColumns(state: UIState, moduleId: string): string[] {
  const stored = state.moduleTableColumns[moduleId]
  if (stored) { return stored }
  if (moduleId === 'movie') { return state.movieTableColumns }
  if (moduleId === 'tv') { return state.seriesTableColumns }
  return []
}

type SetState = StoreApi<UIState>['setState']

function createNotificationSlice(set: SetState) {
  return {
    notifications: [] as Notification[],
    addNotification: (notification: Omit<Notification, 'id'>) =>
      set((state) => ({
        notifications: [...state.notifications, { ...notification, id: crypto.randomUUID() }],
      })),
    dismissNotification: (id: string) =>
      set((state) => ({
        notifications: state.notifications.filter((n) => n.id !== id),
      })),
    clearNotifications: () => set({ notifications: [] }),
  }
}

export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      sidebarCollapsed: false,
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      expandedMenus: { settings: true, activity: false },
      toggleMenu: (menuId) =>
        set((state) => ({
          expandedMenus: { ...state.expandedMenus, [menuId]: !state.expandedMenus[menuId] },
        })),
      setMenuExpanded: (menuId, expanded) =>
        set((state) => ({
          expandedMenus: { ...state.expandedMenus, [menuId]: expanded },
        })),
      theme: 'dark',
      setTheme: (theme) => set({ theme }),
      moviesView: 'grid',
      seriesView: 'grid',
      posterSize: 150,
      movieTableColumns: getDefaultVisibleColumns(MOVIE_COLUMNS),
      seriesTableColumns: getDefaultVisibleColumns(SERIES_COLUMNS),
      movieSortField: 'title',
      movieSortDirection: 'asc',
      seriesSortField: 'title',
      seriesSortDirection: 'asc',
      setMoviesView: (view) => set({ moviesView: view }),
      setSeriesView: (view) => set({ seriesView: view }),
      setPosterSize: (size) => set({ posterSize: size }),
      setMovieTableColumns: (cols) => set({ movieTableColumns: cols }),
      setSeriesTableColumns: (cols) => set({ seriesTableColumns: cols }),
      setMovieSortField: (field) => set({ movieSortField: field }),
      setMovieSortDirection: (direction) => set({ movieSortDirection: direction }),
      setSeriesSortField: (field) => set({ seriesSortField: field }),
      setSeriesSortDirection: (direction) => set({ seriesSortDirection: direction }),
      moduleViewPrefs: {},
      moduleTableColumns: {},
      getModuleView: (moduleId) => resolveModuleView(get(), moduleId),
      setModuleView: (moduleId, view) =>
        set((s) => ({ moduleViewPrefs: { ...s.moduleViewPrefs, [moduleId]: view } })),
      getModuleTableColumns: (moduleId) => resolveModuleTableColumns(get(), moduleId),
      setModuleTableColumns: (moduleId, cols) =>
        set((s) => ({ moduleTableColumns: { ...s.moduleTableColumns, [moduleId]: cols } })),
      globalLoading: false,
      setGlobalLoading: (loading) => set({ globalLoading: loading }),
      ...createNotificationSlice(set),
    }),
    {
      name: 'slipstream-ui',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        expandedMenus: state.expandedMenus,
        theme: state.theme,
        moviesView: state.moviesView,
        seriesView: state.seriesView,
        posterSize: state.posterSize,
        movieTableColumns: state.movieTableColumns,
        seriesTableColumns: state.seriesTableColumns,
        movieSortField: state.movieSortField,
        movieSortDirection: state.movieSortDirection,
        seriesSortField: state.seriesSortField,
        seriesSortDirection: state.seriesSortDirection,
        moduleViewPrefs: state.moduleViewPrefs,
        moduleTableColumns: state.moduleTableColumns,
      }),
    },
  ),
)
