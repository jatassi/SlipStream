import type { ReactNode } from 'react'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'

import { Toaster } from '@/components/ui/sonner'

import { Header } from './header'
import { Sidebar } from './sidebar'
import { useLayoutEffects } from './use-layout-effects'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,
      retry: 1,
    },
  },
})

type RootLayoutProps = {
  children: ReactNode
}

function LoadingScreen() {
  return (
    <div className="bg-background flex min-h-screen items-center justify-center">
      <Loader2 className="text-muted-foreground size-8 animate-spin" />
    </div>
  )
}

function LayoutContent({ children }: RootLayoutProps) {
  const layout = useLayoutEffects()

  if (layout.isPublicRoute) {
    return (
      <div className="bg-background min-h-screen">
        {children}
        <Toaster />
      </div>
    )
  }

  if (layout.isLoadingAuth && !(layout.isAuthenticated && layout.isAdmin)) {
    return <LoadingScreen />
  }

  if (!layout.isAuthenticated || !layout.isAdmin) {
    return <LoadingScreen />
  }

  return (
    <div className="bg-background flex h-screen">
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
