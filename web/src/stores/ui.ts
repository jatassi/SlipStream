import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { getDefaultVisibleColumns, MOVIE_COLUMNS, SERIES_COLUMNS } from '@/lib/table-columns'

export interface Notification {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
}

interface UIState {
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
  setMoviesView: (view: 'grid' | 'table') => void
  setSeriesView: (view: 'grid' | 'table') => void
  setPosterSize: (size: number) => void
  setMovieTableColumns: (cols: string[]) => void
  setSeriesTableColumns: (cols: string[]) => void

  // Global loading override (dev tool)
  globalLoading: boolean
  setGlobalLoading: (loading: boolean) => void

  // Notifications
  notifications: Notification[]
  addNotification: (notification: Omit<Notification, 'id'>) => void
  dismissNotification: (id: string) => void
  clearNotifications: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // Sidebar
      sidebarCollapsed: false,
      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),

      // Sidebar menu expansion state
      expandedMenus: { settings: true, activity: false },
      toggleMenu: (menuId) =>
        set((state) => ({
          expandedMenus: {
            ...state.expandedMenus,
            [menuId]: !state.expandedMenus[menuId],
          },
        })),
      setMenuExpanded: (menuId, expanded) =>
        set((state) => ({
          expandedMenus: {
            ...state.expandedMenus,
            [menuId]: expanded,
          },
        })),

      // Theme
      theme: 'dark',
      setTheme: (theme) => set({ theme }),

      // View preferences
      moviesView: 'grid',
      seriesView: 'grid',
      posterSize: 150,
      movieTableColumns: getDefaultVisibleColumns(MOVIE_COLUMNS),
      seriesTableColumns: getDefaultVisibleColumns(SERIES_COLUMNS),
      setMoviesView: (view) => set({ moviesView: view }),
      setSeriesView: (view) => set({ seriesView: view }),
      setPosterSize: (size) => set({ posterSize: size }),
      setMovieTableColumns: (cols) => set({ movieTableColumns: cols }),
      setSeriesTableColumns: (cols) => set({ seriesTableColumns: cols }),

      // Global loading override
      globalLoading: false,
      setGlobalLoading: (loading) => set({ globalLoading: loading }),

      // Notifications
      notifications: [],
      addNotification: (notification) =>
        set((state) => ({
          notifications: [
            ...state.notifications,
            { ...notification, id: crypto.randomUUID() },
          ],
        })),
      dismissNotification: (id) =>
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        })),
      clearNotifications: () => set({ notifications: [] }),
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
      }),
    }
  )
)
