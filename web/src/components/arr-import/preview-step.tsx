import { useState } from 'react'

import { CheckCircle2, CircleHelp, Eye, EyeOff, Film, LibraryBig, Plus, Tv } from 'lucide-react'

import { metadataApi } from '@/api/metadata'
import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { LoadingButton } from '@/components/ui/loading-button'
import { useQualityProfiles } from '@/hooks/use-quality-profiles'
import { cn } from '@/lib/utils'
import type { ImportMappings, ImportPreview, MoviePreview, SeriesPreview, SourceType } from '@/types/arr-import'

import { usePreviewState } from './use-preview-state'

type PreviewFilter = 'all' | 'new' | 'duplicate' | 'skip'

type PreviewStepProps = {
  preview: ImportPreview
  sourceType: SourceType
  mappings: ImportMappings
  onStartImport: (selectedIds: number[]) => void
  isImporting: boolean
}

export function PreviewStep(props: PreviewStepProps) {
  const { preview, sourceType, mappings } = props
  const isMovie = sourceType === 'radarr'
  const { data: targetQualityProfiles } = useQualityProfiles()
  const profileNameMap = buildProfileNameMap(mappings, targetQualityProfiles)
  const state = usePreviewState(preview, isMovie)

  return (
    <div className="space-y-6">
      <SummaryCards preview={preview} isMovie={isMovie} activeFilter={state.filter} onFilterChange={state.setFilter} />
      <SelectionToolbar selectedCount={state.selectedIds.size} totalNew={state.newCount} onToggleAll={state.toggleSelectAll} isMovie={isMovie} isImporting={props.isImporting} onStartImport={() => props.onStartImport([...state.selectedIds])} />
      <PreviewGrid items={state.filtered} isMovie={isMovie} selectedIds={state.selectedIds} onToggleSelect={state.toggleSelect} profileNameMap={profileNameMap} />
    </div>
  )
}

function buildProfileNameMap(
  mappings: ImportMappings,
  profiles: { id: number; name: string }[] | undefined,
): Record<number, string> {
  const map: Record<number, string> = {}
  if (!profiles) {
    return map
  }
  const profileById = new Map(profiles.map((p) => [p.id, p.name]))
  for (const [sourceId, targetId] of Object.entries(mappings.qualityProfileMapping)) {
    const name = profileById.get(targetId)
    if (name) {
      map[Number(sourceId)] = name
    }
  }
  return map
}

function SummaryCards({
  preview,
  isMovie,
  activeFilter,
  onFilterChange,
}: {
  preview: ImportPreview
  isMovie: boolean
  activeFilter: PreviewFilter
  onFilterChange: (filter: PreviewFilter) => void
}) {
  const s = preview.summary
  const total = isMovie ? s.totalMovies : s.totalSeries
  const newCount = isMovie ? s.newMovies : s.newSeries
  const dupCount = isMovie ? s.duplicateMovies : s.duplicateSeries
  const skipCount = isMovie ? s.skippedMovies : s.skippedSeries

  const cards: { filter: PreviewFilter; label: string; value: number; icon: React.ElementType }[] = [
    { filter: 'all', label: isMovie ? 'All Movies' : 'All TV Shows', value: total, icon: isMovie ? Film : Tv },
    { filter: 'new', label: 'Ready to Migrate', value: newCount, icon: CheckCircle2 },
    { filter: 'duplicate', label: 'Already in Library', value: dupCount, icon: LibraryBig },
    { filter: 'skip', label: 'Unknown', value: skipCount, icon: CircleHelp },
  ]

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
      {cards.map((card) => (
        <FilterCard key={card.filter} card={card} active={activeFilter === card.filter} onClick={() => onFilterChange(card.filter)} />
      ))}
    </div>
  )
}

function FilterCard({ card, active, onClick }: { card: { label: string; value: number; icon: React.ElementType }; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex cursor-pointer items-center gap-3 rounded-lg border p-4 transition-colors',
        active ? 'border-primary bg-primary/5' : 'border-border bg-muted/50 hover:border-primary/50',
      )}
    >
      <card.icon className="text-muted-foreground size-5" />
      <div className="text-left">
        <div className="text-2xl font-semibold">{card.value}</div>
        <div className="text-muted-foreground text-sm">{card.label}</div>
      </div>
    </button>
  )
}

function SelectionToolbar({ selectedCount, totalNew, onToggleAll, isMovie, isImporting, onStartImport }: { selectedCount: number; totalNew: number; onToggleAll: () => void; isMovie: boolean; isImporting: boolean; onStartImport: () => void }) {
  const allSelected = selectedCount === totalNew && totalNew > 0
  const itemType = isMovie ? 'Movie' : 'Series'
  const plural = selectedCount === 1 || !isMovie ? '' : 's'
  const label = `Import ${selectedCount} ${itemType}${plural}`

  return (
    <div className={cn('flex items-center gap-3 rounded-lg border p-3', isMovie ? 'border-movie-500/20 bg-movie-500/10' : 'border-tv-500/20 bg-tv-500/10')}>
      <Checkbox checked={allSelected} onCheckedChange={onToggleAll} />
      <span className="text-sm">
        {selectedCount} of {totalNew} new {isMovie ? 'movies' : 'series'} selected
      </span>
      <div className="ml-auto">
        <LoadingButton loading={isImporting} icon={Plus} onClick={onStartImport} disabled={selectedCount === 0} className={cn(isMovie ? 'bg-movie-500 hover:bg-movie-600 glow-movie-sm' : 'bg-tv-500 hover:bg-tv-600 glow-tv-sm')}>
          {label}
        </LoadingButton>
      </div>
    </div>
  )
}

function PreviewGrid({ items, isMovie, selectedIds, onToggleSelect, profileNameMap }: { items: (MoviePreview | SeriesPreview)[]; isMovie: boolean; selectedIds: Set<number>; onToggleSelect: (id: number) => void; profileNameMap: Record<number, string> }) {
  if (items.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground text-sm">No items match this filter</p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-3 gap-4 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
      {items.map((item) => {
        const id = getItemId(item, isMovie)
        return <PreviewCard key={id} item={item} isMovie={isMovie} selected={selectedIds.has(id)} onToggleSelect={() => onToggleSelect(id)} profileName={profileNameMap[item.qualityProfileId]} />
      })}
    </div>
  )
}

function getItemId(item: MoviePreview | SeriesPreview, isMovie: boolean): number {
  return isMovie ? (item as MoviePreview).tmdbId : (item as SeriesPreview).tvdbId
}

function getQualityLabel(item: MoviePreview | SeriesPreview): string {
  if ('hasFile' in item) {
    if (!item.hasFile) {
      return 'No File'
    }
    return item.quality || 'Unknown'
  }
  if (item.fileCount === 0) {
    return 'No Files'
  }
  return `${item.fileCount} file${item.fileCount === 1 ? '' : 's'}`
}

function getMonitorColor(monitored: boolean, theme: 'movie' | 'tv'): string {
  if (!monitored) {
    return 'text-gray-500'
  }
  return theme === 'movie' ? 'text-movie-400' : 'text-tv-400'
}

function getCardClassName(isSelectable: boolean, selected: boolean, isMovie: boolean): string {
  if (!isSelectable) {
    return 'border-border cursor-default'
  }
  if (!selected) {
    return isMovie ? 'border-border cursor-pointer hover:border-movie-500/50' : 'border-border cursor-pointer hover:border-tv-500/50'
  }
  return isMovie ? 'border-movie-500 cursor-pointer' : 'border-tv-500 cursor-pointer'
}

function PreviewCard({ item, isMovie, selected, onToggleSelect, profileName }: { item: MoviePreview | SeriesPreview; isMovie: boolean; selected: boolean; onToggleSelect: () => void; profileName: string | undefined }) {
  const isSelectable = item.status === 'new'
  const theme = isMovie ? 'movie' : 'tv'

  return (
    <button type="button" disabled={!isSelectable} onClick={onToggleSelect} className={cn('group bg-card block overflow-hidden rounded-lg border-2 transition-all text-left w-full [content-visibility:auto] [contain-intrinsic-size:auto_200px]', getCardClassName(isSelectable, selected, isMovie))}>
      <div className="relative aspect-[2/3]">
        <PreviewPoster item={item} isMovie={isMovie} />
        {isSelectable ? <SelectionCheckbox selected={selected} theme={theme} /> : null}
        <StatusBadge status={item.status} />
        <PreviewOverlay item={item} isMovie={isMovie} profileName={profileName} />
        {isSelectable ? null : <div className="pointer-events-none absolute inset-0 z-[5] bg-black/40" />}
      </div>
    </button>
  )
}

function PreviewPoster({ item, isMovie }: { item: MoviePreview | SeriesPreview; isMovie: boolean }) {
  const [metadataUrl, setMetadataUrl] = useState<string | null>(null)
  const tmdbId = isMovie ? (item as MoviePreview).tmdbId : (item as SeriesPreview).tmdbId
  const type = isMovie ? 'movie' : 'series'

  const handleAllFailed = () => {
    if (metadataUrl || !tmdbId) {
      return
    }
    const fetchFn = isMovie ? metadataApi.getMovie(tmdbId) : metadataApi.getSeries(tmdbId)
    void fetchFn.then((result) => {
      if (result.posterUrl) {
        setMetadataUrl(result.posterUrl)
      }
    })
  }

  return <PosterImage url={metadataUrl ?? item.posterUrl} alt={item.title} type={type} className="absolute inset-0" onAllFailed={handleAllFailed} />
}

function SelectionCheckbox({ selected, theme }: { selected: boolean; theme: 'movie' | 'tv' }) {
  const checkedClass = theme === 'movie' ? 'border-movie-500 data-[checked]:bg-movie-500' : 'border-tv-500 data-[checked]:bg-tv-500'

  return (
    <div className="pointer-events-none absolute top-2 left-2 z-10">
      <Checkbox checked={selected} className={cn('bg-background/80 size-5 border-2', selected && checkedClass)} />
    </div>
  )
}

function PreviewOverlay({ item, isMovie, profileName }: { item: MoviePreview | SeriesPreview; isMovie: boolean; profileName: string | undefined }) {
  const MonitorIcon = item.monitored ? Eye : EyeOff
  const theme = isMovie ? 'movie' : 'tv'
  const monitorColor = getMonitorColor(item.monitored, theme)

  return (
    <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-2 pt-8">
      <div className="mb-1 flex flex-wrap gap-1">
        <Badge variant="secondary" className="text-[10px] leading-tight">{getQualityLabel(item)}</Badge>
        {profileName ? <Badge variant="outline" className="border-primary/30 bg-primary/10 text-primary text-[10px] leading-tight">{profileName}</Badge> : null}
      </div>
      <div className="flex items-end gap-1">
        <h3 className="line-clamp-2 text-xs font-semibold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">{item.title}</h3>
        <MonitorIcon className={cn('mb-0.5 size-3 shrink-0', monitorColor)} />
      </div>
      <p className="text-[10px] text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">{item.year}</p>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const config: Partial<Record<string, { label: string; className: string }>> = {
    new: { label: 'Ready', className: 'bg-green-600 text-white' },
    duplicate: { label: 'Duplicate', className: 'bg-muted text-muted-foreground' },
    skip: { label: 'Unknown', className: 'bg-destructive text-white' },
  }
  const c = config[status]
  if (!c) {
    return null
  }
  return <Badge className={cn('absolute top-2 right-2 z-10 text-[10px]', c.className)}>{c.label}</Badge>
}

