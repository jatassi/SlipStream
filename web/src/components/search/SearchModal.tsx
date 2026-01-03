import { useState, useEffect, useMemo } from 'react'
import { Search, Download, Loader2, ExternalLink, AlertCircle } from 'lucide-react'
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
import { useIndexerMovieSearch, useIndexerTVSearch, useGrab } from '@/hooks'
import { formatBytes, formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
import type { ReleaseInfo, TorrentInfo, SearchCriteria } from '@/types'

interface SearchModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
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

type Release = ReleaseInfo | TorrentInfo

function isTorrent(release: Release): release is TorrentInfo {
  return 'seeders' in release
}

export function SearchModal({
  open,
  onOpenChange,
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

  const isMovie = !!movieId || !!tmdbId
  const mediaTitle = movieTitle || seriesTitle || ''
  const mediaId = movieId || seriesId

  // Build search criteria based on what we have
  const criteria: SearchCriteria = useMemo(() => ({
    query: query || mediaTitle,
    tmdbId: tmdbId,
    imdbId: imdbId,
    tvdbId: tvdbId,
    season: season,
    episode: episode,
    year: year,
    limit: 100,
  }), [query, mediaTitle, tmdbId, imdbId, tvdbId, season, episode, year])

  // Use appropriate search hook
  const movieSearch = useIndexerMovieSearch(criteria, { enabled: searchEnabled && isMovie })
  const tvSearch = useIndexerTVSearch(criteria, { enabled: searchEnabled && !isMovie })

  const searchResult = isMovie ? movieSearch : tvSearch
  const { data, isLoading, isError, error, refetch } = searchResult

  const grabMutation = useGrab()

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setQuery('')
      setSearchEnabled(true)
    } else {
      setSearchEnabled(false)
    }
  }, [open])

  const handleSearch = () => {
    setSearchEnabled(true)
    refetch()
  }

  const handleGrab = async (release: Release) => {
    try {
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
        mediaType: isMovie ? 'movie' : 'episode',
        mediaId: mediaId,
      })

      if (result.success) {
        toast.success(`Grabbed "${release.title}"`)
      } else {
        toast.error(result.error || 'Failed to grab release')
      }
    } catch {
      toast.error('Failed to grab release')
    }
  }

  const releases = data?.releases || []
  const errors = data?.errors || []
  const hasTorrents = releases.some(isTorrent)

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
      <DialogContent className="max-w-5xl max-h-[85vh]">
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
        <ScrollArea className="h-[500px]">
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
                  <TableHead className="w-[300px]">Title</TableHead>
                  <TableHead>Indexer</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Age</TableHead>
                  {hasTorrents && <TableHead>Peers</TableHead>}
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {releases.map((release) => (
                  <TableRow key={release.guid}>
                    <TableCell className="max-w-[300px]">
                      <div className="flex flex-col gap-1">
                        <span className="font-medium truncate" title={release.title}>
                          {release.title}
                        </span>
                        <div className="flex gap-1">
                          <Badge variant="outline" className="text-xs">
                            {release.protocol}
                          </Badge>
                          {isTorrent(release) && release.downloadVolumeFactor === 0 && (
                            <Badge variant="secondary" className="text-xs">
                              Freeleech
                            </Badge>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">{release.indexer}</Badge>
                    </TableCell>
                    <TableCell>{formatBytes(release.size)}</TableCell>
                    <TableCell>
                      {release.publishDate ? formatDate(release.publishDate) : '-'}
                    </TableCell>
                    {hasTorrents && (
                      <TableCell>
                        {isTorrent(release) ? (
                          <span className="text-sm">
                            <span className="text-green-500">{release.seeders}</span>
                            {' / '}
                            <span className="text-red-500">{release.leechers}</span>
                          </span>
                        ) : (
                          '-'
                        )}
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
                          disabled={grabMutation.isPending}
                        >
                          {grabMutation.isPending ? (
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
