import { useState, useMemo } from 'react'
import { Search, Download, Loader2, ExternalLink, AlertCircle, ArrowUp, ArrowDown, ArrowUpDown, Layers } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useIndexerMovieSearch, useIndexerTVSearch, useGrab } from '@/hooks'
import { formatBytes, formatRelativeTime } from '@/lib/formatters'
import { toast } from 'sonner'
import type { TorrentInfo, ScoredSearchCriteria } from '@/types'

interface SearchModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  // Required for scoring
  qualityProfileId: number
  // Movie search props
  movieId?: number
  movieTitle?: string
  tmdbId?: number
  imdbId?: string
  year?: number
  // TV search props
  seriesId?: number
  seriesTitle?: string
  tvdbId?: number
  season?: number
  episode?: number
}

type SortColumn = 'score' | 'title' | 'quality' | 'slot' | 'indexer' | 'size' | 'age' | 'peers'
type SortDirection = 'asc' | 'desc'

// Resolution order for quality sorting (higher = better)
const RESOLUTION_ORDER: Record<string, number> = {
  '2160p': 4,
  '1080p': 3,
  '720p': 2,
  '480p': 1,
  'SD': 0,
}

function SortIcon({ column, sortColumn, sortDirection }: { column: SortColumn; sortColumn: SortColumn; sortDirection: SortDirection }) {
  if (sortColumn !== column) {
    return <ArrowUpDown className="ml-1 size-3 text-muted-foreground" />
  }
  return sortDirection === 'asc'
    ? <ArrowUp className="ml-1 size-3" />
    : <ArrowDown className="ml-1 size-3" />
}

export function SearchModal({
  open,
  onOpenChange,
  qualityProfileId,
  movieId,
  movieTitle,
  tmdbId,
  imdbId,
  year,
  seriesId,
  seriesTitle,
  tvdbId,
  season,
  episode,
}: SearchModalProps) {
  const [query, setQuery] = useState('')
  const [searchEnabled, setSearchEnabled] = useState(false)
  const [sortColumn, setSortColumn] = useState<SortColumn>('score')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

  const isMovie = !!movieId || !!tmdbId
  const mediaTitle = movieTitle || seriesTitle || ''
  const mediaId = movieId || seriesId

  // Build search criteria based on what we have (requires qualityProfileId for scoring)
  const criteria: ScoredSearchCriteria = useMemo(() => ({
    query: query || mediaTitle,
    qualityProfileId: qualityProfileId,
    tmdbId: tmdbId,
    imdbId: imdbId,
    tvdbId: tvdbId,
    season: season,
    episode: episode,
    year: year,
    limit: 100,
  }), [query, mediaTitle, qualityProfileId, tmdbId, imdbId, tvdbId, season, episode, year])

  // Use appropriate search hook
  const movieSearchEnabled = searchEnabled && isMovie
  const tvSearchEnabled = searchEnabled && !isMovie

  console.log('[SearchModal] searchEnabled:', searchEnabled, 'isMovie:', isMovie,
    'movieSearchEnabled:', movieSearchEnabled, 'tvSearchEnabled:', tvSearchEnabled,
    'criteria:', criteria)

  const movieSearch = useIndexerMovieSearch(criteria, { enabled: movieSearchEnabled })
  const tvSearch = useIndexerTVSearch(criteria, { enabled: tvSearchEnabled })

  const searchResult = isMovie ? movieSearch : tvSearch
  const { data, isLoading, isError, error, refetch } = searchResult

  console.log('[SearchModal] isLoading:', isLoading, 'isError:', isError, 'data:', data, 'error:', error)

  const grabMutation = useGrab()
  const [grabbingGuid, setGrabbingGuid] = useState<string | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)

  // Reset state when modal opens (React-recommended pattern)
  if (open !== prevOpen) {
    setPrevOpen(open)
    if (open) {
      setQuery('')
      setSearchEnabled(true)
      setSortColumn('score')
      setSortDirection('desc')
      setGrabbingGuid(null)
    } else {
      setSearchEnabled(false)
    }
  }

  const handleSearch = () => {
    setSearchEnabled(true)
    refetch()
  }

  const handleGrab = async (release: TorrentInfo) => {
    setGrabbingGuid(release.guid)
    try {
      // Determine media type and flags based on search context
      let mediaType: 'movie' | 'episode' | 'season' = 'episode'
      let isSeasonPack = false
      let isCompleteSeries = false

      if (isMovie) {
        mediaType = 'movie'
      } else if (seriesId) {
        if (season !== undefined && episode === undefined) {
          // Season search without specific episode = season pack
          mediaType = 'season'
          isSeasonPack = true
        } else if (season === undefined && episode === undefined) {
          // Series search without season or episode = complete series
          mediaType = 'season'
          isCompleteSeries = true
        }
        // Otherwise it's a specific episode search, mediaType stays 'episode'
      }

      const result = await grabMutation.mutateAsync({
        release: {
          guid: release.guid,
          title: release.title,
          downloadUrl: release.downloadUrl,
          indexerId: release.indexerId,
          indexer: release.indexer,
          protocol: release.protocol,
          size: release.size,
          tmdbId: release.tmdbId,
          tvdbId: release.tvdbId,
          imdbId: release.imdbId,
        },
        mediaType,
        mediaId: mediaId,
        seriesId: seriesId,
        seasonNumber: season,
        isSeasonPack,
        isCompleteSeries,
        targetSlotId: release.targetSlotId,
      })

      if (result.success) {
        toast.success(`Grabbed "${release.title}"`)
      } else {
        toast.error(result.error || 'Failed to grab release')
      }
    } catch {
      toast.error('Failed to grab release')
    } finally {
      setGrabbingGuid(null)
    }
  }

  const rawReleases = data?.releases || []
  const errors = data?.errors || []
  // All results now include torrent info (seeders/leechers)
  const hasTorrents = rawReleases.length > 0
  // Check if any releases have slot info (multi-version mode)
  const hasSlotInfo = rawReleases.some(r => r.targetSlotId !== undefined)

  // Toggle sort or change column
  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column)
      // Default direction based on column type
      setSortDirection(column === 'title' || column === 'indexer' ? 'asc' : 'desc')
    }
  }

  // Sort releases
  const releases = useMemo(() => {
    const sorted = [...rawReleases]
    sorted.sort((a, b) => {
      let comparison = 0
      switch (sortColumn) {
        case 'score':
          comparison = (a.score ?? 0) - (b.score ?? 0)
          break
        case 'title':
          comparison = a.title.localeCompare(b.title)
          break
        case 'quality': {
          const aRes = RESOLUTION_ORDER[a.quality || ''] ?? -1
          const bRes = RESOLUTION_ORDER[b.quality || ''] ?? -1
          comparison = aRes - bRes
          break
        }
        case 'slot':
          comparison = (a.targetSlotNumber ?? 99) - (b.targetSlotNumber ?? 99)
          break
        case 'indexer':
          comparison = a.indexer.localeCompare(b.indexer)
          break
        case 'size':
          comparison = a.size - b.size
          break
        case 'age': {
          const aDate = a.publishDate ? new Date(a.publishDate).getTime() : 0
          const bDate = b.publishDate ? new Date(b.publishDate).getTime() : 0
          comparison = aDate - bDate
          break
        }
        case 'peers':
          comparison = (a.seeders ?? 0) - (b.seeders ?? 0)
          break
      }
      return sortDirection === 'asc' ? comparison : -comparison
    })
    return sorted
  }, [rawReleases, sortColumn, sortDirection])

  // Build title
  let title = 'Search Releases'
  if (seriesTitle && season !== undefined && episode !== undefined) {
    title = `Search: ${seriesTitle} S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
  } else if (seriesTitle && season !== undefined) {
    title = `Search: ${seriesTitle} Season ${season}`
  } else if (mediaTitle) {
    title = `Search: ${mediaTitle}`
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-6xl h-[85vh] flex flex-col overflow-hidden">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            Search indexers for releases and send to download client.
          </DialogDescription>
        </DialogHeader>

        {/* Search input */}
        <div className="flex gap-2">
          <Input
            placeholder="Search query (optional, overrides automatic search)"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <Button onClick={handleSearch} disabled={isLoading}>
            {isLoading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Search className="size-4" />
            )}
          </Button>
        </div>

        {/* Errors from indexers */}
        {errors.length > 0 && (
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertDescription>
              {errors.length} indexer(s) returned errors. Some results may be missing.
            </AlertDescription>
          </Alert>
        )}

        {/* Results */}
        <ScrollArea className="flex-1 min-h-0">
          {isLoading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
          ) : isError ? (
            <div className="flex flex-col items-center justify-center h-40 gap-2">
              <AlertCircle className="size-8 text-destructive" />
              <p className="text-muted-foreground">
                {error instanceof Error ? error.message : 'Failed to search'}
              </p>
              <Button variant="outline" onClick={() => refetch()}>
                Retry
              </Button>
            </div>
          ) : releases.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 gap-2">
              <Search className="size-8 text-muted-foreground" />
              <p className="text-muted-foreground">No releases found</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('title')}
                    >
                      Title
                      <SortIcon column="title" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  <TableHead className="w-[70px]">
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('score')}
                    >
                      Score
                      <SortIcon column="score" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  <TableHead className="w-[100px]">
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('quality')}
                    >
                      Quality
                      <SortIcon column="quality" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  {hasSlotInfo && (
                    <TableHead className="w-[120px]">
                      <button
                        className="flex items-center hover:text-foreground transition-colors"
                        onClick={() => handleSort('slot')}
                      >
                        Slot
                        <SortIcon column="slot" sortColumn={sortColumn} sortDirection={sortDirection} />
                      </button>
                    </TableHead>
                  )}
                  <TableHead className="w-[100px]">
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('indexer')}
                    >
                      Indexer
                      <SortIcon column="indexer" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  <TableHead className="w-[80px]">
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('size')}
                    >
                      Size
                      <SortIcon column="size" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  <TableHead className="w-[100px]">
                    <button
                      className="flex items-center hover:text-foreground transition-colors"
                      onClick={() => handleSort('age')}
                    >
                      Age
                      <SortIcon column="age" sortColumn={sortColumn} sortDirection={sortDirection} />
                    </button>
                  </TableHead>
                  {hasTorrents && (
                    <TableHead className="w-[100px]">
                      <button
                        className="flex items-center hover:text-foreground transition-colors"
                        onClick={() => handleSort('peers')}
                      >
                        Peers
                        <SortIcon column="peers" sortColumn={sortColumn} sortDirection={sortDirection} />
                      </button>
                    </TableHead>
                  )}
                  <TableHead className="w-[80px] text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {releases.map((release) => (
                  <TableRow key={release.guid}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-medium">
                          {release.title}
                        </span>
                        <div className="flex gap-1">
                          <Badge variant="outline" className="text-xs">
                            {release.protocol}
                          </Badge>
                          {release.downloadVolumeFactor === 0 && (
                            <Badge variant="secondary" className="text-xs">
                              Freeleech
                            </Badge>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="font-medium">{release.normalizedScore ?? '-'}</span>
                    </TableCell>
                    <TableCell>
                      {release.quality ? (
                        <Badge variant="secondary">{release.quality}</Badge>
                      ) : (
                        <span className="text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    {hasSlotInfo && (
                      <TableCell>
                        {release.targetSlotName ? (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger>
                                <div className="flex items-center gap-1">
                                  <Layers className="size-3" />
                                  <span className="text-sm">{release.targetSlotName}</span>
                                  {release.isSlotUpgrade && (
                                    <Badge variant="secondary" className="text-xs px-1">
                                      <ArrowUp className="size-3" />
                                    </Badge>
                                  )}
                                  {release.isSlotNewFill && (
                                    <Badge variant="outline" className="text-xs px-1 text-green-500 border-green-500">
                                      New
                                    </Badge>
                                  )}
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                {release.isSlotUpgrade && 'Will upgrade existing file in this slot'}
                                {release.isSlotNewFill && 'Will fill empty slot'}
                                {!release.isSlotUpgrade && !release.isSlotNewFill && `Target: ${release.targetSlotName}`}
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                    )}
                    <TableCell>
                      <Badge variant="outline">{release.indexer}</Badge>
                    </TableCell>
                    <TableCell>{formatBytes(release.size)}</TableCell>
                    <TableCell>
                      {release.publishDate ? formatRelativeTime(release.publishDate) : '-'}
                    </TableCell>
                    {hasTorrents && (
                      <TableCell>
                        <span className="text-sm">
                          <span className="text-green-500">{release.seeders ?? 0}</span>
                          {' / '}
                          <span className="text-red-500">{release.leechers ?? 0}</span>
                        </span>
                      </TableCell>
                    )}
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        {release.infoUrl && (
                          <a
                            href={release.infoUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground h-9 w-9"
                          >
                            <ExternalLink className="size-4" />
                          </a>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleGrab(release)}
                          disabled={grabbingGuid === release.guid}
                        >
                          {grabbingGuid === release.guid ? (
                            <Loader2 className="size-4 animate-spin" />
                          ) : (
                            <Download className="size-4" />
                          )}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </ScrollArea>

        {/* Footer with stats */}
        {data && (
          <div className="flex items-center justify-between text-sm text-muted-foreground border-t pt-4">
            <span>
              {data.total} release{data.total !== 1 ? 's' : ''} from {data.indexersSearched} indexer{data.indexersSearched !== 1 ? 's' : ''}
            </span>
            {errors.length > 0 && (
              <span className="text-destructive">
                {errors.length} error{errors.length !== 1 ? 's' : ''}
              </span>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
