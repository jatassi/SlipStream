import { HardDrive } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatBytes } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { StorageInfo } from '@/types/storage'

type StorageProgressBarProps = {
  value: number
  max?: number
  showPercentage?: boolean
  className?: string
}

function StorageProgressBar({
  value,
  max = 100,
  showPercentage = false,
  className,
}: StorageProgressBarProps) {
  const percentage = Math.min((value / max) * 100, 100)

  const colorClasses = getColorClasses(percentage)

  return (
    <div className={cn('flex items-center gap-2', className)}>
      <div className="bg-muted relative h-6 flex-1 overflow-hidden rounded-full">
        <div
          className={cn('h-full transition-all', colorClasses)}
          style={{ width: `${percentage}%` }}
        />
        {showPercentage ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="text-[16px] font-medium text-white drop-shadow-sm">
              {percentage.toFixed(0)}%
            </span>
          </div>
        ) : null}
      </div>
    </div>
  )
}

// Get color classes based on usage percentage
function getColorClasses(percent: number): string {
  if (percent < 30) {
    return 'bg-emerald-700'
  } // Dark green
  if (percent < 50) {
    return 'bg-emerald-500'
  } // Light green
  if (percent < 70) {
    return 'bg-yellow-500'
  } // Yellow
  if (percent < 85) {
    return 'bg-amber-500'
  } // Amber
  return 'bg-red-500' // Red
}

type StorageCardProps = {
  storage?: StorageInfo[]
  loading?: boolean
}

export function StorageCard({ storage, loading }: StorageCardProps) {
  if (loading) {
    return <StorageCardSkeleton />
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-muted-foreground text-sm font-medium">Storage</CardTitle>
        <HardDrive className="text-muted-foreground size-4" />
      </CardHeader>
      <CardContent>
        {/* Overall storage summary */}
        <div className="mb-4 text-2xl font-bold">
          {storage?.length === 1 ? storage[0].label : 'Storage'}
        </div>

        {/* Root folders as badges */}
        {storage?.length === 1 && storage[0].rootFolders && storage[0].rootFolders.length > 0 ? (
          <div className="mb-4 flex flex-wrap gap-1">
            {storage[0].rootFolders.map((folder) => (
              <Badge
                key={folder.id}
                variant="default"
                className="bg-blue-100 text-xs text-blue-800 dark:bg-blue-900 dark:text-blue-200"
              >
                {folder.name}
              </Badge>
            ))}
          </div>
        ) : null}

        <VolumeList storage={storage} />
      </CardContent>
    </Card>
  )
}

function VolumeList({ storage }: { storage?: StorageInfo[] }) {
  return (
    <div className="space-y-3">
      {storage
        ?.filter((volume) => volume.totalSpace > 1_000_000_000)
        .toSorted((a, b) => b.totalSpace - a.totalSpace)
        .slice(0, 3)
        .map((volume) => (
          <div key={volume.path} className="space-y-1">
            {storage.length > 1 && (
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">{volume.label}</span>
                {volume.rootFolders && volume.rootFolders.length > 0 ? (
                  <span className="text-muted-foreground text-xs">
                    {volume.rootFolders.length} folder
                    {volume.rootFolders.length === 1 ? '' : 's'}
                  </span>
                ) : null}
              </div>
            )}
            <StorageProgressBar value={volume.usedPercent} showPercentage />
            <div className="text-muted-foreground flex justify-between text-xs">
              <span>{formatBytes(volume.totalSpace)} Total</span>
              <span>{formatBytes(volume.freeSpace)} Free</span>
            </div>
          </div>
        ))}
    </div>
  )
}

function StorageCardSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-muted-foreground text-sm font-medium">Storage</CardTitle>
        <HardDrive className="text-muted-foreground size-4" />
      </CardHeader>
      <CardContent>
        <Skeleton className="mb-4 h-8 w-20" />
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-2 w-full" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
