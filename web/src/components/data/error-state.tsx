import { AlertCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'

type ErrorStateProps = {
  title?: string
  message?: string
  onRetry?: () => void
}

export function ErrorState({
  title = 'Something went wrong',
  message = 'An error occurred while loading data. Please try again.',
  onRetry,
}: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center px-4 py-12 text-center">
      <div className="text-destructive mb-4">
        <AlertCircle className="size-12" />
      </div>
      <h3 className="mb-1 text-lg font-semibold">{title}</h3>
      <p className="text-muted-foreground mb-4 max-w-md text-sm">{message}</p>
      {onRetry ? (
        <Button variant="outline" onClick={onRetry}>
          Try Again
        </Button>
      ) : null}
    </div>
  )
}
