import { Skeleton } from '@/components/ui/skeleton'

const SKELETON_WIDTHS = [55, 40, 65, 48, 60, 42, 70, 50]

export function FileBrowserSkeleton() {
  return (
    <div className="space-y-1">
      {SKELETON_WIDTHS.map((w) => (
        <div key={w} className="flex items-center gap-2 p-2">
          <Skeleton className="size-4 shrink-0 rounded" />
          <Skeleton className="h-4" style={{ width: `${w}%` }} />
          {w % 3 === 0 && <Skeleton className="ml-auto size-4 shrink-0" />}
        </div>
      ))}
    </div>
  )
}
