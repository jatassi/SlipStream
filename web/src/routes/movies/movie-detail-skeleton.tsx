import { Skeleton } from '@/components/ui/skeleton'

const CREDIT_PLACEHOLDERS = ['c1', 'c2', 'c3', 'c4', 'c5', 'c6']

function HeroSkeleton() {
  return (
    <div className="relative h-64 md:h-80">
      <Skeleton className="absolute inset-0 rounded-none" />
      <Skeleton className="absolute top-4 right-4 z-10 h-6 w-20 rounded bg-white/5" />
      <div className="absolute inset-0 flex items-end p-6">
        <div className="flex max-w-4xl items-end gap-6">
          <div className="hidden shrink-0 md:block">
            <Skeleton className="h-60 w-40 rounded-lg bg-white/10" />
          </div>
          <div className="flex-1 space-y-2">
            <div className="flex items-center gap-2">
              <Skeleton className="h-5 w-16 rounded-full bg-white/10" />
              <Skeleton className="h-5 w-24 rounded-full bg-white/10" />
            </div>
            <Skeleton className="h-9 w-72 bg-white/10" />
            <div className="flex items-center gap-3">
              <Skeleton className="h-4 w-10 bg-white/10" />
              <Skeleton className="h-4 w-16 bg-white/10" />
              <Skeleton className="h-4 w-14 bg-white/10" />
              <Skeleton className="h-4 w-20 bg-white/10" />
            </div>
            <div className="flex items-center gap-3">
              <Skeleton className="h-5 w-12 rounded bg-white/10" />
              <Skeleton className="h-5 w-12 rounded bg-white/10" />
            </div>
            <div className="max-w-2xl space-y-1.5">
              <Skeleton className="h-3.5 w-full bg-white/10" />
              <Skeleton className="h-3.5 w-4/5 bg-white/10" />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function ActionBarSkeleton() {
  return (
    <div className="bg-card flex flex-wrap gap-2 border-b px-6 py-4">
      <div className="flex gap-2">
        <Skeleton className="h-9 w-9 rounded-md min-[820px]:hidden" />
        <Skeleton className="hidden h-9 w-24 rounded-md min-[820px]:block" />
        <Skeleton className="h-9 w-9 rounded-md min-[820px]:hidden" />
        <Skeleton className="hidden h-9 w-32 rounded-md min-[820px]:block" />
      </div>
      <div className="ml-auto flex gap-2">
        <Skeleton className="h-9 w-9 rounded-md" />
        <Skeleton className="h-9 w-9 rounded-md" />
        <Skeleton className="h-9 w-9 rounded-md" />
      </div>
    </div>
  )
}

function ContentSkeleton() {
  return (
    <div className="space-y-6 p-6">
      <div className="border-border bg-card rounded-lg border">
        <div className="flex items-center justify-between px-6 py-4">
          <Skeleton className="h-5 w-24" />
        </div>
        <div className="border-t px-6 py-4">
          <div className="flex items-center justify-center py-6">
            <Skeleton className="h-4 w-40" />
          </div>
        </div>
      </div>
      <div className="border-border bg-card rounded-lg border">
        <div className="px-6 py-4">
          <Skeleton className="h-5 w-16" />
        </div>
        <div className="border-t px-6 py-4">
          <div className="flex gap-4 overflow-hidden">
            {CREDIT_PLACEHOLDERS.map((key) => (
              <div key={key} className="w-28 shrink-0 space-y-2">
                <Skeleton className="aspect-[2/3] w-full rounded-md" />
                <Skeleton className="h-3.5 w-full" />
                <Skeleton className="h-3 w-3/4" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export function MovieDetailSkeleton() {
  return (
    <div className="-m-6">
      <HeroSkeleton />
      <ActionBarSkeleton />
      <ContentSkeleton />
    </div>
  )
}
