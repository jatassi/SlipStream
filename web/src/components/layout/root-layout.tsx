import type { ReactNode } from 'react'
import { Suspense } from 'react'

import { QueryClientProvider } from '@tanstack/react-query'

import { ErrorBoundary } from '@/components/error-boundary'
import { Toaster } from '@/components/ui/sonner'
import { useDocumentTitle } from '@/hooks/use-document-title'
import { useViewport } from '@/hooks/use-viewport'
import { queryClient } from '@/lib/query-client'

import { LoadingScreen } from './loading-screen'
import { ShellLayout } from './shell-layout'
import { useLayoutEffects } from './use-layout-effects'

type RootLayoutProps = {
  children: ReactNode
}

function LayoutContent({ children }: RootLayoutProps) {
  useDocumentTitle()
  const layout = useLayoutEffects()
  const shell = useViewport()

  if (layout.isPublicRoute) {
    return (
      <div className="bg-background min-h-dvh" data-shell={shell}>
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
    <ShellLayout>
      <ErrorBoundary>
        <Suspense fallback={<LoadingScreen />}>{children}</Suspense>
      </ErrorBoundary>
    </ShellLayout>
  )
}

export function RootLayout({ children }: RootLayoutProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <LayoutContent>{children}</LayoutContent>
    </QueryClientProvider>
  )
}
