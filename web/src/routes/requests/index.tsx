import { useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { formatDistanceToNow } from 'date-fns'
import {
  ArrowRight,
  CheckCircle,
  Clock,
  Download,
  Loader2,
  Search,
  User,
  XCircle,
} from 'lucide-react'

import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { usePortalDownloads, useRequests } from '@/hooks'
import { formatEta } from '@/lib/formatters'
import type { Request, RequestStatus } from '@/types'

const STATUS_CONFIG: Record<
  RequestStatus,
  { label: string; icon: React.ReactNode; color: string }
> = {
  pending: {
    label: 'Pending',
    icon: <Clock className="size-3 md:size-4" />,
    color: 'bg-yellow-500',
  },
  approved: {
    label: 'Approved',
    icon: <CheckCircle className="size-3 md:size-4" />,
    color: 'bg-blue-500',
  },
  denied: { label: 'Denied', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-red-500' },
  downloading: {
    label: 'Downloading',
    icon: <Download className="size-3 md:size-4" />,
    color: 'bg-purple-500',
  },
  failed: { label: 'Failed', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-red-700' },
  available: {
    label: 'Available',
    icon: <CheckCircle className="size-3 md:size-4" />,
    color: 'bg-green-500',
  },
  cancelled: {
    label: 'Cancelled',
    icon: <XCircle className="size-3 md:size-4" />,
    color: 'bg-gray-500',
  },
}

export function RequestsListPage() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [searchFocused, setSearchFocused] = useState(false)
  const [filter, setFilter] = useState<'mine' | 'all'>('mine')
  const { data: requests = [], isLoading } = useRequests({ scope: filter })

  const sortedRequests = requests.toSorted(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
  )

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate({ to: '/requests/search', search: { q: searchQuery.trim() } })
    }
  }

  const goToRequestDetail = (id: number) => {
    navigate({ to: '/requests/$id', params: { id: String(id) } })
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Search Section */}
      <section className="border-border from-movie-500/5 to-tv-500/5 flex flex-col items-center justify-center border-b bg-gradient-to-b via-transparent py-8">
        <div className="w-full max-w-2xl space-y-4 px-6">
          <div className="space-y-1 text-center">
            <Search className="text-media-gradient mx-auto size-10" />
            <h2 className="text-xl font-semibold">Search for Content</h2>
            <p className="text-muted-foreground text-sm">Find movies and TV series to request</p>
          </div>
          <form onSubmit={handleSearch} className="flex gap-2">
            <div
              className={`relative flex-1 rounded-md transition-shadow duration-300 ${searchFocused ? 'glow-media-sm' : ''}`}
            >
              <Search className="text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2" />
              <Input
                type="text"
                placeholder="Search movies and series..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                className="h-11 pl-10"
              />
            </div>
            <Button type="submit" size="icon" className="h-11 w-11">
              <ArrowRight className="size-5" />
            </Button>
          </form>
        </div>
      </section>

      {/* Requests Section */}
      <section className="flex flex-1 flex-col overflow-hidden">
        <div className="border-border bg-card/50 border-b px-6 py-4">
          <div className="mx-auto flex max-w-6xl items-center justify-between">
            <div className="flex items-center gap-4">
              <h2 className="text-xl font-semibold">Requests</h2>
              <Tabs value={filter} onValueChange={(v) => setFilter(v as 'mine' | 'all')}>
                <TabsList>
                  <TabsTrigger value="mine">Mine</TabsTrigger>
                  <TabsTrigger value="all">All</TabsTrigger>
                </TabsList>
              </Tabs>
            </div>
            <Badge variant="secondary">{sortedRequests.length} total</Badge>
          </div>
        </div>

        <div className="flex-1 overflow-auto px-6 py-4">
          <div className="mx-auto max-w-6xl">
            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="text-muted-foreground size-8 animate-spin" />
              </div>
            ) : sortedRequests.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <Clock className="text-muted-foreground/50 mb-4 size-12" />
                <p className="text-muted-foreground">No requests yet</p>
                <p className="text-muted-foreground/70 text-sm">Search above to request content</p>
              </div>
            ) : (
              <div className="space-y-2">
                {sortedRequests.map((request) => (
                  <RequestCard
                    key={request.id}
                    request={request}
                    showUser={filter === 'all'}
                    onClick={() => goToRequestDetail(request.id)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  )
}

type RequestCardProps = {
  request: Request
  showUser?: boolean
  onClick: () => void
}

function RequestCard({ request, showUser, onClick }: RequestCardProps) {
  const { data: downloads } = usePortalDownloads()
  const statusConfig = STATUS_CONFIG[request.status]

  // Find ALL active downloads for this request (there may be multiple season downloads)
  const matchingDownloads =
    downloads?.filter((d) => {
      if (request.mediaType === 'movie') {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        return d.movieId !== null && request.mediaId !== null && d.movieId === request.mediaId
      }
      if (request.mediaType === 'series') {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        return d.seriesId !== null && request.mediaId !== null && d.seriesId === request.mediaId
      }
      return false
    }) ?? []
  const hasActiveDownload = matchingDownloads.length > 0

  // Aggregate stats across all downloads
  const totalSize = matchingDownloads.reduce((sum, d) => sum + (d.size || 0), 0)
  const totalDownloaded = matchingDownloads.reduce((sum, d) => sum + (d.downloadedSize || 0), 0)
  const totalSpeed = matchingDownloads.reduce((sum, d) => sum + (d.downloadSpeed || 0), 0)
  const progress = totalSize > 0 ? (totalDownloaded / totalSize) * 100 : 0
  const remainingBytes = totalSize - totalDownloaded
  const eta = totalSpeed > 0 ? Math.ceil(remainingBytes / totalSpeed) : 0
  const isActive = matchingDownloads.some((d) => d.status === 'downloading')
  const isPaused =
    matchingDownloads.length > 0 && matchingDownloads.every((d) => d.status === 'paused')
  const isComplete = Math.round(progress) >= 100

  const isMovie = request.mediaType === 'movie'

  return (
    <button
      onClick={onClick}
      className={`bg-card flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-all sm:gap-4 sm:p-4 ${
        isMovie
          ? 'border-movie-500/20 hover:border-movie-500/50 hover:glow-movie-sm'
          : 'border-tv-500/20 hover:border-tv-500/50 hover:glow-tv-sm'
      }`}
    >
      <div className="h-18 w-12 flex-shrink-0 overflow-hidden rounded">
        <PosterImage
          url={request.posterUrl}
          alt={request.title}
          type={request.mediaType === 'movie' ? 'movie' : 'series'}
          className="h-full w-full"
        />
      </div>

      <div className="min-w-0 flex-1">
        {showUser && request.user ? (
          <div className="text-muted-foreground mb-0.5 flex items-center gap-1 text-xs">
            <User className="size-3" />
            <span>{request.user.displayName || request.user.username}</span>
          </div>
        ) : null}
        <div className="leading-snug">
          <span className="font-medium">{request.title}</span>
          {request.year ? (
            <span className="text-muted-foreground ml-1 text-sm">({request.year})</span>
          ) : null}
          <Badge
            className={`${statusConfig.color} ml-2 px-1.5 py-0.5 align-middle text-[10px] text-white sm:hidden md:px-2 md:text-xs`}
          >
            {statusConfig.icon}
            <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
          </Badge>
        </div>
        <div className="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-2 text-sm">
          <Badge
            variant="outline"
            className={`text-xs capitalize ${
              isMovie ? 'border-movie-500/50 text-movie-400' : 'border-tv-500/50 text-tv-400'
            }`}
          >
            {request.mediaType}
          </Badge>
          {request.mediaType === 'series' && (
            <>
              {request.requestedSeasons && request.requestedSeasons.length > 0 ? (
                <span>
                  {request.requestedSeasons.length <= 3
                    ? `S${request.requestedSeasons.join(', S')}`
                    : `${request.requestedSeasons.length} seasons`}
                </span>
              ) : (
                <span className="text-muted-foreground/70">No seasons</span>
              )}
              {request.monitorFuture ? (
                <Badge variant="secondary" className="text-xs">
                  Future
                </Badge>
              ) : null}
            </>
          )}
          {request.seasonNumber && request.mediaType !== 'series' ? (
            <span>Season {request.seasonNumber}</span>
          ) : null}
          {request.episodeNumber ? <span>Episode {request.episodeNumber}</span> : null}
          <span>•</span>
          <span>{formatDistanceToNow(new Date(request.createdAt), { addSuffix: true })}</span>
        </div>
        {hasActiveDownload ? (
          <div className="mt-2 max-w-48">
            <Progress value={progress} variant={isMovie ? 'movie' : 'tv'} className="h-1.5" />
            <div className="text-muted-foreground mt-0.5 flex justify-between text-xs">
              <span>{Math.round(progress)}%</span>
              <span>
                {isComplete
                  ? 'Importing'
                  : isPaused
                    ? 'Paused'
                    : isActive
                      ? formatEta(eta)
                      : 'Queued'}
              </span>
            </div>
          </div>
        ) : null}
      </div>

      <Badge
        className={`${statusConfig.color} hidden shrink-0 px-1.5 py-0.5 text-[10px] text-white sm:flex md:px-2 md:text-xs`}
      >
        {statusConfig.icon}
        <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
      </Badge>
    </button>
  )
}
