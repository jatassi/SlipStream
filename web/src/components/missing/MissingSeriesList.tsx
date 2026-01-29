import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Search, Zap, Tv, ChevronDown, ChevronRight, Loader2, Download, BookmarkX } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { SearchModal } from '@/components/search/SearchModal'
import { EmptyState } from '@/components/data/EmptyState'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import { useAutoSearchSeries, useAutoSearchSeason, useAutoSearchEpisode, useUpdateSeries } from '@/hooks'
import { useDownloadingStore } from '@/stores'
import { toast } from 'sonner'
import type { MissingSeries, MissingSeason, MissingEpisode } from '@/types/missing'
import type { AutoSearchResult, BatchAutoSearchResult } from '@/types'

interface MissingSeriesListProps {
  series: MissingSeries[]
  isSearchingAll?: boolean
}

interface SearchContext {
  qualityProfileId?: number
  seriesId?: number
  seriesTitle?: string
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  season?: number
  episode?: number
}

export function MissingSeriesList({ series, isSearchingAll = false }: MissingSeriesListProps) {
  const [expandedSeries, setExpandedSeries] = useState<Set<number>>(new Set())
  const [expandedSeasons, setExpandedSeasons] = useState<Set<string>>(new Set())
  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [searchContext, setSearchContext] = useState<SearchContext>({})
  const [searchingSeriesId, setSearchingSeriesId] = useState<number | null>(null)
  const [searchingSeasonKey, setSearchingSeasonKey] = useState<string | null>(null)
  const [searchingEpisodeId, setSearchingEpisodeId] = useState<number | null>(null)
  const [unmonitoringSeriesId, setUnmonitoringSeriesId] = useState<number | null>(null)

  const seriesAutoSearchMutation = useAutoSearchSeries()
  const seasonAutoSearchMutation = useAutoSearchSeason()
  const episodeAutoSearchMutation = useAutoSearchEpisode()
  const updateSeriesMutation = useUpdateSeries()

  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isSeriesDownloading = (seriesId: number) => {
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        item.isCompleteSeries &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isSeasonDownloading = (seriesId: number, seasonNumber: number) => {
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        ((item.seasonNumber === seasonNumber && item.isSeasonPack) ||
          item.isCompleteSeries) &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isEpisodeDownloading = (episodeId: number, seriesId?: number, seasonNumber?: number) => {
    return queueItems.some((item) => {
      if (item.status !== 'downloading' && item.status !== 'queued') return false
      if (item.episodeId === episodeId) return true
      if (seriesId && item.seriesId === seriesId) {
        if (item.isCompleteSeries) return true
        if (seasonNumber && item.seasonNumber === seasonNumber && item.isSeasonPack) return true
      }
      return false
    })
  }

  const toggleSeries = (seriesId: number) => {
    const newExpanded = new Set(expandedSeries)
    if (newExpanded.has(seriesId)) {
      newExpanded.delete(seriesId)
    } else {
      newExpanded.add(seriesId)
    }
    setExpandedSeries(newExpanded)
  }

  const toggleSeason = (seriesId: number, seasonNumber: number) => {
    const key = `${seriesId}-${seasonNumber}`
    const newExpanded = new Set(expandedSeasons)
    if (newExpanded.has(key)) {
      newExpanded.delete(key)
    } else {
      newExpanded.add(key)
    }
    setExpandedSeasons(newExpanded)
  }

  const handleSeriesSearch = (s: MissingSeries) => {
    setSearchContext({
      qualityProfileId: s.qualityProfileId,
      seriesId: s.id,
      seriesTitle: s.title,
      tvdbId: s.tvdbId,
      tmdbId: s.tmdbId,
      imdbId: s.imdbId,
    })
    setSearchModalOpen(true)
  }

  const handleSeasonSearch = (s: MissingSeries, season: MissingSeason) => {
    setSearchContext({
      qualityProfileId: s.qualityProfileId,
      seriesId: s.id,
      seriesTitle: s.title,
      tvdbId: s.tvdbId,
      tmdbId: s.tmdbId,
      imdbId: s.imdbId,
      season: season.seasonNumber,
    })
    setSearchModalOpen(true)
  }

  const handleEpisodeSearch = (s: MissingSeries, episode: MissingEpisode) => {
    setSearchContext({
      qualityProfileId: s.qualityProfileId,
      seriesId: s.id,
      seriesTitle: s.title,
      tvdbId: s.tvdbId,
      tmdbId: s.tmdbId,
      imdbId: s.imdbId,
      season: episode.seasonNumber,
      episode: episode.episodeNumber,
    })
    setSearchModalOpen(true)
  }

  const formatBatchResult = (result: BatchAutoSearchResult, label: string) => {
    if (result.downloaded > 0) {
      toast.success(`Found ${result.downloaded} releases for ${label}`, {
        description: `Searched ${result.totalSearched} items`,
      })
    } else if (result.found > 0) {
      toast.info(`Found ${result.found} releases but none downloaded for ${label}`)
    } else if (result.failed > 0) {
      toast.error(`Search failed for ${result.failed} items in ${label}`)
    } else {
      toast.warning(`No releases found for ${label}`)
    }
  }

  const formatSingleResult = (result: AutoSearchResult, title: string) => {
    if (result.error) {
      toast.error(`Search failed for "${title}"`, { description: result.error })
      return
    }
    if (!result.found) {
      toast.warning(`No releases found for "${title}"`)
      return
    }
    if (result.downloaded) {
      const message = result.upgraded ? 'Quality upgrade found' : 'Found and downloading'
      toast.success(`${message}: ${result.release?.title || title}`, {
        description: result.clientName ? `Sent to ${result.clientName}` : undefined,
      })
    } else {
      toast.info(`Release found but not downloaded: ${result.release?.title || title}`)
    }
  }

  const handleSeriesAutoSearch = async (s: MissingSeries) => {
    setSearchingSeriesId(s.id)
    try {
      const result = await seriesAutoSearchMutation.mutateAsync(s.id)
      formatBatchResult(result, s.title)
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`"${s.title}" is already in the download queue`)
      } else {
        toast.error(`Search failed for "${s.title}"`)
      }
    } finally {
      setSearchingSeriesId(null)
    }
  }

  const handleSeasonAutoSearch = async (s: MissingSeries, season: MissingSeason) => {
    const key = `${s.id}-${season.seasonNumber}`
    setSearchingSeasonKey(key)
    try {
      const result = await seasonAutoSearchMutation.mutateAsync({
        seriesId: s.id,
        seasonNumber: season.seasonNumber,
      })
      formatBatchResult(result, `${s.title} Season ${season.seasonNumber}`)
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`Season ${season.seasonNumber} is already in the download queue`)
      } else {
        toast.error(`Search failed for Season ${season.seasonNumber}`)
      }
    } finally {
      setSearchingSeasonKey(null)
    }
  }

  const handleEpisodeAutoSearch = async (_s: MissingSeries, episode: MissingEpisode) => {
    setSearchingEpisodeId(episode.id)
    try {
      const result = await episodeAutoSearchMutation.mutateAsync(episode.id)
      formatSingleResult(result, `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`)
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`Episode is already in the download queue`)
      } else {
        toast.error(`Search failed for episode`)
      }
    } finally {
      setSearchingEpisodeId(null)
    }
  }

  const handleUnmonitorSeries = async (s: MissingSeries) => {
    setUnmonitoringSeriesId(s.id)
    try {
      await updateSeriesMutation.mutateAsync({
        id: s.id,
        data: { monitored: false },
      })
      toast.success(`"${s.title}" unmonitored`)
    } catch {
      toast.error(`Failed to unmonitor "${s.title}"`)
    } finally {
      setUnmonitoringSeriesId(null)
    }
  }

  const getSeasonSummary = (seasons: MissingSeason[]): string => {
    if (seasons.length === 1) {
      return `Season ${seasons[0].seasonNumber}`
    }
    const seasonNumbers = seasons.map((s) => s.seasonNumber).sort((a, b) => a - b)
    return `Seasons ${seasonNumbers[0]}-${seasonNumbers[seasonNumbers.length - 1]}`
  }

  if (series.length === 0) {
    return (
      <EmptyState
        icon={<Tv className="size-8" />}
        title="No missing episodes"
        description="All monitored episodes that have aired have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <>
      <div className="space-y-2">
        {series.map((s) => {
          const isSeriesExpanded = expandedSeries.has(s.id)

          return (
            <div key={s.id} className="border rounded-lg">
              {/* Series Header */}
              <div className="flex items-center justify-between p-4">
                <Collapsible open={isSeriesExpanded} onOpenChange={() => toggleSeries(s.id)}>
                  <CollapsibleTrigger className="flex items-center gap-3 hover:text-primary transition-colors">
                    {isSeriesExpanded ? (
                      <ChevronDown className="size-4 shrink-0" />
                    ) : (
                      <ChevronRight className="size-4 shrink-0" />
                    )}
                    <div className="flex items-center gap-2">
                      <Tv className="size-4 text-muted-foreground" />
                      <Link
                        to="/series/$id"
                        params={{ id: s.id.toString() }}
                        className="font-medium hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {s.title}
                      </Link>
                      {s.year && (
                        <span className="text-muted-foreground">({s.year})</span>
                      )}
                    </div>
                  </CollapsibleTrigger>
                </Collapsible>

                <div className="flex items-center gap-4">
                  <span className="text-sm text-muted-foreground">
                    {s.missingCount} episode{s.missingCount !== 1 ? 's' : ''} missing
                    {s.missingSeasons.length > 0 && (
                      <> across {getSeasonSummary(s.missingSeasons)}</>
                    )}
                  </span>
                  <div className="flex gap-1">
                    {isSeriesDownloading(s.id) ? (
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled
                        title="Downloading"
                      >
                        <Download className="size-4 text-green-500" />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleSeriesAutoSearch(s)}
                        disabled={isSearchingAll || searchingSeriesId === s.id}
                        title="Automatic Search (All Missing)"
                      >
                        {isSearchingAll || searchingSeriesId === s.id ? (
                          <Loader2 className="size-4 animate-spin" />
                        ) : (
                          <Zap className="size-4" />
                        )}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleSeriesSearch(s)}
                      title="Manual Search (All Missing)"
                    >
                      <Search className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleUnmonitorSeries(s)}
                      disabled={unmonitoringSeriesId === s.id}
                      title="Unmonitor Series"
                    >
                      {unmonitoringSeriesId === s.id ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : (
                        <BookmarkX className="size-4" />
                      )}
                    </Button>
                  </div>
                </div>
              </div>

              {/* Seasons and Episodes */}
              <Collapsible open={isSeriesExpanded}>
                <CollapsibleContent>
                  <div className="border-t px-4 pb-4 space-y-2">
                    {s.missingSeasons.map((season) => {
                      const seasonKey = `${s.id}-${season.seasonNumber}`
                      const isSeasonExpanded = expandedSeasons.has(seasonKey)

                      return (
                        <div key={seasonKey} className="ml-6 mt-2">
                          {/* Season Header */}
                          <div className="flex items-center justify-between py-2">
                            <Collapsible
                              open={isSeasonExpanded}
                              onOpenChange={() => toggleSeason(s.id, season.seasonNumber)}
                            >
                              <CollapsibleTrigger className="flex items-center gap-2 hover:text-primary transition-colors">
                                {isSeasonExpanded ? (
                                  <ChevronDown className="size-3 shrink-0" />
                                ) : (
                                  <ChevronRight className="size-3 shrink-0" />
                                )}
                                <span className="font-medium">
                                  Season {season.seasonNumber}
                                </span>
                                <span className="text-sm text-muted-foreground">
                                  ({season.missingEpisodes.length} missing)
                                </span>
                              </CollapsibleTrigger>
                            </Collapsible>

                            <div className="flex gap-1">
                              {isSeasonDownloading(s.id, season.seasonNumber) ? (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  disabled
                                  title="Downloading"
                                  className="h-7 w-7 p-0"
                                >
                                  <Download className="size-3 text-green-500" />
                                </Button>
                              ) : (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleSeasonAutoSearch(s, season)}
                                  disabled={isSearchingAll || searchingSeasonKey === seasonKey}
                                  title="Automatic Search (Season)"
                                  className="h-7 w-7 p-0"
                                >
                                  {isSearchingAll || searchingSeasonKey === seasonKey ? (
                                    <Loader2 className="size-3 animate-spin" />
                                  ) : (
                                    <Zap className="size-3" />
                                  )}
                                </Button>
                              )}
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleSeasonSearch(s, season)}
                                title="Manual Search (Season)"
                                className="h-7 w-7 p-0"
                              >
                                <Search className="size-3" />
                              </Button>
                            </div>
                          </div>

                          {/* Episodes */}
                          <Collapsible open={isSeasonExpanded}>
                            <CollapsibleContent>
                              <div className="ml-5 space-y-1 border-l pl-4">
                                {[...season.missingEpisodes]
                                  .sort((a, b) => a.episodeNumber - b.episodeNumber)
                                  .map((episode) => (
                                  <div
                                    key={episode.id}
                                    className="flex items-center justify-between py-1.5 text-sm"
                                  >
                                    <div className="flex items-center gap-2">
                                      <span className="text-muted-foreground w-8">
                                        {episode.episodeNumber.toString().padStart(2, '0')}
                                      </span>
                                      <span
                                        className={cn(
                                          'truncate max-w-[300px]',
                                          !episode.title && 'text-muted-foreground italic'
                                        )}
                                        title={episode.title || 'TBA'}
                                      >
                                        {episode.title || 'TBA'}
                                      </span>
                                      {episode.airDate && (
                                        <span className="text-muted-foreground text-xs">
                                          ({formatDate(episode.airDate)})
                                        </span>
                                      )}
                                    </div>

                                    <div className="flex gap-1">
                                      {isEpisodeDownloading(episode.id, s.id, episode.seasonNumber) ? (
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          disabled
                                          title="Downloading"
                                          className="h-6 w-6 p-0"
                                        >
                                          <Download className="size-3 text-green-500" />
                                        </Button>
                                      ) : (
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          onClick={() => handleEpisodeAutoSearch(s, episode)}
                                          disabled={isSearchingAll || searchingEpisodeId === episode.id}
                                          title="Automatic Search"
                                          className="h-6 w-6 p-0"
                                        >
                                          {isSearchingAll || searchingEpisodeId === episode.id ? (
                                            <Loader2 className="size-3 animate-spin" />
                                          ) : (
                                            <Zap className="size-3" />
                                          )}
                                        </Button>
                                      )}
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleEpisodeSearch(s, episode)}
                                        title="Manual Search"
                                        className="h-6 w-6 p-0"
                                      >
                                        <Search className="size-3" />
                                      </Button>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </CollapsibleContent>
                          </Collapsible>
                        </div>
                      )
                    })}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            </div>
          )
        })}
      </div>

      <SearchModal
        open={searchModalOpen}
        onOpenChange={setSearchModalOpen}
        qualityProfileId={searchContext.qualityProfileId ?? 0}
        seriesId={searchContext.seriesId}
        seriesTitle={searchContext.seriesTitle}
        tvdbId={searchContext.tvdbId}
        season={searchContext.season}
        episode={searchContext.episode}
      />
    </>
  )
}
