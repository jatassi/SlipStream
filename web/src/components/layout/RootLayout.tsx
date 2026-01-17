import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { Toaster } from '@/components/ui/sonner'
import { useWebSocketStore, useWebSocketHandler, useUIStore, useDevModeStore } from '@/stores'
import { useQueue, useStatus } from '@/hooks'

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 1,
    },
  },
})

interface RootLayoutProps {
  children: ReactNode
}

function LayoutContent({ children }: RootLayoutProps) {
  const { connect, disconnect } = useWebSocketStore()
  const { theme } = useUIStore()
  const { setEnabled: setDevModeEnabled, setSwitching: setDevModeSwitching } = useDevModeStore()
  const { data: status } = useStatus()

  // Process WebSocket messages (handles progress events, query invalidation, etc.)
  useWebSocketHandler()

  // Keep downloading store synced globally (polls queue and syncs to store)
  useQueue()

  // Sync developer mode state from backend on initial load and after switches
  // Also clears switching state in case WebSocket message was lost during reconnection
  useEffect(() => {
    if (status?.developerMode !== undefined) {
      setDevModeEnabled(status.developerMode)
      setDevModeSwitching(false)
    }
  }, [status?.developerMode, setDevModeEnabled, setDevModeSwitching])

  // Apply theme
  useEffect(() => {
    const root = document.documentElement
    if (theme === 'dark') {
      root.classList.add('dark')
    } else if (theme === 'light') {
      root.classList.remove('dark')
    } else {
      // System theme
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      if (mediaQuery.matches) {
        root.classList.add('dark')
      } else {
        root.classList.remove('dark')
      }
    }
  }, [theme])

  // Connect WebSocket on mount
  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return (
    <div className="flex h-screen bg-background">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
      <Toaster />
    </div>
  )
}

export function RootLayout({ children }: RootLayoutProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <LayoutContent>{children}</LayoutContent>
    </QueryClientProvider>
  )
}
