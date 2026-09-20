import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'

export function SectionLoading({ count }: { count: number }) {
  return (
    <div className="px-screen">
      <LoadingState variant="list" count={count} />
    </div>
  )
}

export function SectionError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="px-screen">
      <ErrorState onRetry={onRetry} />
    </div>
  )
}
