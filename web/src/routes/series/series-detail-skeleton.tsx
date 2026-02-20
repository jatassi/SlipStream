import { Skeleton } from '@/components/ui/skeleton'

const EPISODE_PLACEHOLDERS = ['e1', 'e2', 'e3', 'e4', 'e5']

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
              <Skeleton className="h-5 w-28 rounded-full bg-white/10" />
              <Skeleton className="h-5 w-24 rounded-full bg-white/10" />
            </div>
            <Skeleton className="h-9 w-64 bg-white/10" />
            <div className="flex items-center gap-3">
              <Skeleton className="h-4 w-16 bg-white/10" />
              <Skeleton className="h-4 w-12 bg-white/10" />
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

function SeasonsSkeleton() {
  return (
    <div className="space-y-6 p-6">
      <div className="border-border bg-card rounded-lg border">
        <div className="px-6 py-4">
          <Skeleton className="h-5 w-40" />
        </div>
        <div className="border-t">
          <div>
            <div className="flex items-center gap-3 px-6 py-3">
              <Skeleton className="size-4" />
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-10" />
            </div>
            <div className="space-y-0.5 px-6 pb-3">
              {EPISODE_PLACEHOLDERS.map((key) => (
                <div key={key} className="flex items-center gap-4 rounded-md px-3 py-2.5">
                  <Skeleton className="h-4 w-6 shrink-0" />
                  <Skeleton className="h-4 w-48" />
                  <div className="ml-auto flex items-center gap-2">
                    <Skeleton className="h-5 w-16 rounded-full" />
                    <Skeleton className="size-7 rounded-md" />
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-3 px-6 py-3">
            <Skeleton className="size-4" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-4 w-10" />
          </div>
        </div>
      </div>
    </div>
  )
}

export function SeriesDetailSkeleton() {
  return (
    <div className="-m-6">
      <HeroSkeleton />
      <ActionBarSkeleton />
      <SeasonsSkeleton />
    </div>
  )
}
