import { Film, Tv } from 'lucide-react'

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

import type { MediaFilter, ViewMode } from './use-missing-page'
import { ViewToggle } from './view-toggle'

type MediaTabsProps = {
  filter: MediaFilter
  onFilterChange: (filter: MediaFilter) => void
  isLoading: boolean
  isMissingView: boolean
  totalCount: number
  movieCount: number
  episodeCount: number
  onViewChange: (view: ViewMode) => void
  children: React.ReactNode
}

function CountBadge({ count, className }: { count: number; className?: string }) {
  if (count <= 0) {
    return null
  }
  return <span className={cn('ml-2 text-xs', className)}>({count})</span>
}

export function MediaTabs({
  filter,
  onFilterChange,
  isLoading,
  isMissingView,
  totalCount,
  movieCount,
  episodeCount,
  onViewChange,
  children,
}: MediaTabsProps) {
  return (
    <Tabs value={filter} onValueChange={(v) => onFilterChange(v as MediaFilter)} className="space-y-4">
      <div className={cn('flex flex-wrap items-center justify-between gap-3', isLoading && 'pointer-events-none opacity-50')}>
        <TabsList>
          <TabsTrigger value="all" className="data-active:glow-media-sm px-4 data-active:bg-white data-active:text-black">
            All
            {!isLoading && <CountBadge count={totalCount} className="data-active:text-black/60" />}
          </TabsTrigger>
          <TabsTrigger value="movies" className="data-active:glow-movie data-active:bg-white data-active:text-black">
            <Film className="mr-1.5 size-4" />
            Movies
            {!isLoading && <CountBadge count={movieCount} className="text-muted-foreground" />}
          </TabsTrigger>
          <TabsTrigger value="series" className="data-active:glow-tv data-active:bg-white data-active:text-black">
            <Tv className="mr-1.5 size-4" />
            Series
            {!isLoading && <CountBadge count={episodeCount} className="text-muted-foreground" />}
          </TabsTrigger>
        </TabsList>
        <ViewToggle isMissingView={isMissingView} onViewChange={onViewChange} />
      </div>
      {children}
    </Tabs>
  )
}
