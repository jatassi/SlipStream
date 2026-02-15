import { Skeleton } from '@/components/ui/skeleton'

function SkeletonRows({ count }: { count: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="border-border bg-card flex items-center gap-4 rounded-lg border px-4 py-3"
        >
          <Skeleton className="hidden h-[60px] w-10 shrink-0 rounded-md sm:block" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <div className="flex items-baseline gap-2">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-3 w-10" />
            </div>
            <Skeleton className="h-4 w-20 rounded-full" />
          </div>
          <div className="ml-auto flex shrink-0 items-center gap-1.5">
            <Skeleton className="size-8 rounded-md" />
            <Skeleton className="size-8 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  )
}

export function LoadingSkeleton() {
  return (
    <div className="mt-4 space-y-6">
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Skeleton className="size-4 rounded-full" />
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-8" />
        </div>
        <SkeletonRows count={5} />
      </div>
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Skeleton className="size-4 rounded-full" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-8" />
        </div>
        <SkeletonRows count={4} />
      </div>
    </div>
  )
}
