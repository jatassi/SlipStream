import { Group, RowSkeleton } from '@/components/grouped-list'
import { Skeleton } from '@/components/ui/skeleton'

const PILLS = ['search', 'auto', 'monitor']
const ROWS = ['a', 'b', 'c']

export function MediaDetailSkeleton() {
  return (
    <div className="mx-auto w-full max-w-3xl">
      <Skeleton className="h-56 w-full rounded-none md:h-72" />
      <div className="relative -mt-24 flex items-end gap-4 px-screen">
        <Skeleton className="rounded-card aspect-[2/3] w-28 shrink-0 bg-white/10" />
        <div className="flex-1 space-y-2 pb-1">
          <Skeleton className="h-3 w-14 bg-white/10" />
          <Skeleton className="h-6 w-48 bg-white/10" />
          <Skeleton className="h-4 w-40 bg-white/10" />
          <Skeleton className="h-6 w-24 rounded-full bg-white/10" />
        </div>
      </div>
      <div className="flex gap-2 px-screen pt-5 pb-7">
        {PILLS.map((key) => (
          <Skeleton key={key} className="h-tap flex-1 rounded-full" />
        ))}
      </div>
      <Group header="Overview">
        <div className="space-y-2 px-4 py-3">
          <Skeleton className="h-3.5 w-full" />
          <Skeleton className="h-3.5 w-11/12" />
          <Skeleton className="h-3.5 w-3/5" />
        </div>
      </Group>
      <Group header="File">
        {ROWS.map((key) => (
          <RowSkeleton key={key} />
        ))}
      </Group>
    </div>
  )
}
