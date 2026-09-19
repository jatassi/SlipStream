import { Skeleton } from '@/components/ui/skeleton'

const ROW_PAD = 'flex min-h-tap w-full items-center gap-3 px-4 py-2.5'

type RowSkeletonProps = {
  leading?: 'tile' | 'poster'
  subtitle?: boolean
  trailing?: boolean
  chevron?: boolean
  progress?: boolean
}

function LeadingSkeleton({ kind }: { kind: NonNullable<RowSkeletonProps['leading']> }) {
  if (kind === 'poster') {
    return <Skeleton className="h-12 w-8 rounded-[4px]" />
  }
  return <Skeleton className="size-7 rounded-[7px]" />
}

function SubtitleSkeleton({ progress }: { progress: boolean }) {
  if (progress) {
    return (
      <div className="mt-0.5 flex items-center gap-2">
        <Skeleton className="h-1 w-24 rounded-full" />
        <Skeleton className="h-3.5 w-28" />
      </div>
    )
  }
  return <Skeleton className="mt-0.5 h-[18px] w-1/2" />
}

export function RowSkeleton({
  leading = 'tile',
  subtitle = true,
  trailing = false,
  chevron = false,
  progress = false,
}: RowSkeletonProps) {
  return (
    <div role="status" aria-label="Loading" className={ROW_PAD}>
      <LeadingSkeleton kind={leading} />
      <div className="min-w-0 flex-1">
        <Skeleton className="h-5 w-2/3" />
        {subtitle ? <SubtitleSkeleton progress={progress} /> : null}
      </div>
      {trailing ? <Skeleton className="h-[18px] w-12" /> : null}
      {chevron ? <Skeleton className="size-4 rounded" /> : null}
    </div>
  )
}
