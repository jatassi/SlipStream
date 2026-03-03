import type { ReactNode } from 'react'
import { Suspense } from 'react'

import { QueryClientProvider } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'

import { ErrorBoundary } from '@/components/error-boundary'
import { Toaster } from '@/components/ui/sonner'
import { useDocumentTitle } from '@/hooks/use-document-title'
import { queryClient } from '@/lib/query-client'

import { Header } from './header'
import { Sidebar } from './sidebar'
import { useLayoutEffects } from './use-layout-effects'

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
  useDocumentTitle()
  const layout = useLayoutEffects()

  if (layout.isPublicRoute) {
    return (
      <div className="bg-background min-h-screen">
        <ErrorBoundary>
          <Suspense fallback={<LoadingScreen />}>{children}</Suspense>
        </ErrorBoundary>
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
        <main className="flex-1 overflow-auto p-6">
          <ErrorBoundary>
            <Suspense fallback={<LoadingScreen />}>{children}</Suspense>
          </ErrorBoundary>
        </main>
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
