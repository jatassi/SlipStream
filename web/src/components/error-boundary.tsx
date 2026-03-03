import type { ErrorInfo, ReactNode } from 'react'
import { Component } from 'react'

import { AlertCircle, RefreshCw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type Props = {
  children: ReactNode
  fallback?: ReactNode
  className?: string
}

type State = {
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  render() {
    const { error } = this.state
    const { children, fallback, className } = this.props

    if (!error) {
      return children
    }

    if (fallback) {
      return fallback
    }

    return (
      <div
        className={cn(
          'bg-background text-foreground flex min-h-[200px] flex-col items-center justify-center gap-4 rounded-xl p-8',
          className,
        )}
      >
        <AlertCircle className="text-destructive size-10" />
        <div className="flex flex-col items-center gap-1 text-center">
          <p className="text-base font-medium">Something went wrong</p>
          <p className="text-muted-foreground max-w-sm text-sm">{error.message}</p>
        </div>
        <Button variant="outline" onClick={() => globalThis.location.reload()}>
          <RefreshCw className="size-4" />
          Reload
        </Button>
      </div>
    )
  }
}
