import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import { getEnabledModules } from '@/modules'

import type { MediaFilter, ViewMode } from './use-missing-page'
import { ViewToggle } from './view-toggle'

const THEME_GLOW_CLASSES: Record<string, string> = {
  movie: 'data-active:glow-movie',
  tv: 'data-active:glow-tv',
}

// TODO: Module system — derive per-module counts from backend per-module missing API
// Tab values ("movies"/"series") are currently used as keys in search mutations
const MODULE_TAB_MAP: { moduleId: string; tabValue: MediaFilter; countKey: 'movieCount' | 'episodeCount' }[] = [
  { moduleId: 'movie', tabValue: 'movies', countKey: 'movieCount' },
  { moduleId: 'tv', tabValue: 'series', countKey: 'episodeCount' },
]

type MediaTabsProps = {
  filter: MediaFilter
  onFilterChange: (filter: MediaFilter) => void
  isLoading: boolean
  isMissingView: boolean
  totalCount: number
  movieCount: number
  episodeCount: number
  upgradableTotalCount: number
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
  upgradableTotalCount,
  onViewChange,
  children,
}: MediaTabsProps) {
  const modules = getEnabledModules()
  const counts: Record<string, number> = { movieCount, episodeCount }

  return (
    <Tabs value={filter} onValueChange={(v) => onFilterChange(v as MediaFilter)} className="space-y-4">
      <div className={cn('flex flex-wrap items-center justify-between gap-3', isLoading && 'pointer-events-none opacity-50')}>
        <TabsList>
          <TabsTrigger value="all" className="data-active:glow-media-sm px-4 data-active:bg-white data-active:text-black">
            All
            {isLoading ? null : <CountBadge count={totalCount} className="data-active:text-black/60" />}
          </TabsTrigger>
          {MODULE_TAB_MAP.flatMap((entry) => {
            const mod = modules.find((m) => m.id === entry.moduleId)
            if (!mod) {
              return []
            }
            return (
              <TabsTrigger
                key={entry.tabValue}
                value={entry.tabValue}
                className={cn(THEME_GLOW_CLASSES[mod.themeColor], 'data-active:bg-white data-active:text-black')}
              >
                <mod.icon className="mr-1.5 size-4" />
                {mod.name}
                {isLoading ? null : <CountBadge count={counts[entry.countKey]} className="text-muted-foreground" />}
              </TabsTrigger>
            )
          })}
        </TabsList>
        <ViewToggle isMissingView={isMissingView} upgradableTotalCount={upgradableTotalCount} isLoading={isLoading} onViewChange={onViewChange} />
      </div>
      {children}
    </Tabs>
  )
}
