import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Search, Zap, Tv, ChevronDown, ChevronRight } from 'lucide-react'
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
import type { MissingSeries, MissingSeason, MissingEpisode } from '@/types/missing'

interface MissingSeriesListProps {
  series: MissingSeries[]
}

interface SearchContext {
  seriesId?: number
  seriesTitle?: string
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  season?: number
  episode?: number
}

export function MissingSeriesList({ series }: MissingSeriesListProps) {
  const [expandedSeries, setExpandedSeries] = useState<Set<number>>(new Set())
  const [expandedSeasons, setExpandedSeasons] = useState<Set<string>>(new Set())
  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [searchContext, setSearchContext] = useState<SearchContext>({})

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

  const handleAutoSearch = () => {
    // Placeholder for automatic search - will be wired up later
    console.log('Auto search triggered')
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
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleAutoSearch()}
                      title="Automatic Search (All Missing)"
                    >
                      <Zap className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleSeriesSearch(s)}
                      title="Manual Search (All Missing)"
                    >
                      <Search className="size-4" />
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
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleAutoSearch()}
                                title="Automatic Search (Season)"
                                className="h-7 w-7 p-0"
                              >
                                <Zap className="size-3" />
                              </Button>
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
                                {season.missingEpisodes.map((episode) => (
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
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleAutoSearch()}
                                        title="Automatic Search"
                                        className="h-6 w-6 p-0"
                                      >
                                        <Zap className="size-3" />
                                      </Button>
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
        seriesId={searchContext.seriesId}
        seriesTitle={searchContext.seriesTitle}
        tvdbId={searchContext.tvdbId}
        season={searchContext.season}
        episode={searchContext.episode}
      />
    </>
  )
}
