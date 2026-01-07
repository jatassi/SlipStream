import { HardDrive } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatBytes } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { StorageInfo } from '@/types/storage'

interface StorageProgressBarProps {
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
      <div className="flex-1 h-6 rounded-full bg-muted relative overflow-hidden">
        <div
          className={cn('h-full transition-all', colorClasses)}
          style={{ width: `${percentage}%` }}
        />
        {showPercentage && (
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="text-[16px] font-medium text-white drop-shadow-sm">
              {percentage.toFixed(0)}%
            </span>
          </div>
        )}
      </div>
    </div>
  )
}

// Get color classes based on usage percentage
function getColorClasses(percent: number): string {
  if (percent < 30) return 'bg-emerald-700' // Dark green
  if (percent < 50) return 'bg-emerald-500' // Light green
  if (percent < 70) return 'bg-yellow-500'  // Yellow
  if (percent < 85) return 'bg-amber-500'   // Amber
  return 'bg-red-500'                      // Red
}

interface StorageCardProps {
  storage?: StorageInfo[]
  loading?: boolean
}

export function StorageCard({ storage, loading }: StorageCardProps) {
  if (loading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Storage
          </CardTitle>
          <HardDrive className="size-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-8 w-20 mb-4" />
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



  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          Storage
        </CardTitle>
        <HardDrive className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {/* Overall storage summary */}
        <div className="text-2xl font-bold mb-4">
          {storage?.length === 1 ? storage[0].label : 'Storage'}
        </div>

        {/* Root folders as badges */}
        {storage?.length === 1 && storage[0].rootFolders && storage[0].rootFolders.length > 0 && (
          <div className="flex flex-wrap gap-1 mb-4">
            {storage[0].rootFolders.map((folder) => (
              <Badge 
                key={folder.id}
                variant="default"
                className="text-xs bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
              >
                {folder.name}
              </Badge>
            ))}
          </div>
        )}

        {/* Storage volumes */}
        <div className="space-y-3">
          {storage
            ?.filter(volume => volume && volume.totalSpace > 1000000000) // Show volumes > 1GB
            ?.sort((a, b) => b.totalSpace - a.totalSpace) // Sort by size
            ?.slice(0, 3)
            .map((volume) => (
            <div key={volume.path} className="space-y-1">
              {/* Only show volume label if multiple volumes */}
              {storage?.length > 1 && (
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">
                    {volume.label}
                  </span>
                  {volume.rootFolders && volume.rootFolders.length > 0 && (
                    <span className="text-xs text-muted-foreground">
                      {volume.rootFolders.length} folder{volume.rootFolders.length !== 1 ? 's' : ''}
                    </span>
                  )}
                </div>
              )}

              {/* Storage usage progress bar */}
              <StorageProgressBar 
                value={volume.usedPercent} 
                showPercentage={true}
              />
              
              {/* Storage details */}
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{formatBytes(volume.totalSpace)} Total</span>
                <span>{formatBytes(volume.freeSpace)} Free</span>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

