import { AlertCircle, Loader2, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'

export function SearchLoadingState() {
  return (
    <div className="flex h-40 items-center justify-center">
      <Loader2 className="text-muted-foreground size-8 animate-spin" />
    </div>
  )
}

export function SearchErrorState({
  error,
  onRetry,
}: {
  error: unknown
  onRetry: () => void
}) {
  return (
    <div className="flex h-40 flex-col items-center justify-center gap-2">
      <AlertCircle className="text-destructive size-8" />
      <p className="text-muted-foreground">
        {error instanceof Error ? error.message : 'Failed to search'}
      </p>
      <Button variant="outline" onClick={onRetry}>
        Retry
      </Button>
    </div>
  )
}

export function SearchEmptyState() {
  return (
    <div className="flex h-40 flex-col items-center justify-center gap-2">
      <Search className="text-muted-foreground size-8" />
      <p className="text-muted-foreground">No releases found</p>
    </div>
  )
}
