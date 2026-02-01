import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  Clock,
  CheckCircle,
  XCircle,
  Download,
  Loader2,
  Search,
  ArrowRight,
  User,
} from 'lucide-react'
import { PosterImage } from '@/components/media/PosterImage'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useRequests, usePortalDownloads } from '@/hooks'
import { Progress } from '@/components/ui/progress'
import { formatEta } from '@/lib/formatters'
import type { Request, RequestStatus } from '@/types'
import { formatDistanceToNow } from 'date-fns'

const STATUS_CONFIG: Record<RequestStatus, { label: string; icon: React.ReactNode; color: string }> = {
  pending: { label: 'Pending', icon: <Clock className="size-3 md:size-4" />, color: 'bg-yellow-500' },
  approved: { label: 'Approved', icon: <CheckCircle className="size-3 md:size-4" />, color: 'bg-blue-500' },
  denied: { label: 'Denied', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-red-500' },
  downloading: { label: 'Downloading', icon: <Download className="size-3 md:size-4" />, color: 'bg-purple-500' },
  available: { label: 'Available', icon: <CheckCircle className="size-3 md:size-4" />, color: 'bg-green-500' },
  cancelled: { label: 'Cancelled', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-gray-500' },
}

export function RequestsListPage() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [searchFocused, setSearchFocused] = useState(false)
  const [filter, setFilter] = useState<'mine' | 'all'>('mine')
  const { data: requests = [], isLoading } = useRequests({ scope: filter })

  const sortedRequests = [...requests].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
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
    <div className="flex flex-col flex-1 min-h-0">
      {/* Search Section */}
      <section className="flex flex-col items-center justify-center py-8 border-b border-border bg-gradient-to-b from-movie-500/5 via-transparent to-tv-500/5">
        <div className="w-full max-w-2xl px-6 space-y-4">
          <div className="text-center space-y-1">
            <Search className="size-10 mx-auto text-media-gradient" />
            <h2 className="text-xl font-semibold">Search for Content</h2>
            <p className="text-sm text-muted-foreground">Find movies and TV series to request</p>
          </div>
          <form onSubmit={handleSearch} className="flex gap-2">
            <div className={`relative flex-1 rounded-md transition-shadow duration-300 ${searchFocused ? 'glow-media-sm' : ''}`}>
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground z-10" />
              <Input
                type="text"
                placeholder="Search movies and series..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                className="pl-10 h-11"
              />
            </div>
            <Button type="submit" size="icon" className="h-11 w-11">
              <ArrowRight className="size-5" />
            </Button>
          </form>
        </div>
      </section>

      {/* Requests Section */}
      <section className="flex-1 flex flex-col overflow-hidden">
        <div className="px-6 py-4 border-b border-border bg-card/50">
          <div className="flex items-center justify-between max-w-6xl mx-auto">
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
          <div className="max-w-6xl mx-auto">
            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
              </div>
            ) : sortedRequests.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <Clock className="size-12 text-muted-foreground/50 mb-4" />
                <p className="text-muted-foreground">No requests yet</p>
                <p className="text-sm text-muted-foreground/70">Search above to request content</p>
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

interface RequestCardProps {
  request: Request
  showUser?: boolean
  onClick: () => void
}

function RequestCard({ request, showUser, onClick }: RequestCardProps) {
  const { data: downloads } = usePortalDownloads()
  const statusConfig = STATUS_CONFIG[request.status]

  // Find ALL active downloads for this request (there may be multiple season downloads)
  const matchingDownloads = downloads?.filter(d => {
    if (request.mediaType === 'movie') return d.movieId != null && request.mediaId != null && d.movieId === request.mediaId
    if (request.mediaType === 'series') return d.seriesId != null && request.mediaId != null && d.seriesId === request.mediaId
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
  const isActive = matchingDownloads.some(d => d.status === 'downloading')
  const isPaused = matchingDownloads.length > 0 && matchingDownloads.every(d => d.status === 'paused')
  const isComplete = Math.round(progress) >= 100

  const isMovie = request.mediaType === 'movie'

  return (
    <button
      onClick={onClick}
      className={`flex items-start gap-3 sm:gap-4 p-3 sm:p-4 rounded-lg border bg-card transition-all w-full text-left ${
        isMovie
          ? 'border-movie-500/20 hover:border-movie-500/50 hover:glow-movie-sm'
          : 'border-tv-500/20 hover:border-tv-500/50 hover:glow-tv-sm'
      }`}
    >
      <div className="flex-shrink-0 w-12 h-18 rounded overflow-hidden">
        <PosterImage
          url={request.posterUrl}
          alt={request.title}
          type={request.mediaType === 'movie' ? 'movie' : 'series'}
          className="w-full h-full"
        />
      </div>

      <div className="flex-1 min-w-0">
        {showUser && request.user && (
          <div className="flex items-center gap-1 text-xs text-muted-foreground mb-0.5">
            <User className="size-3" />
            <span>{request.user.displayName || request.user.username}</span>
          </div>
        )}
        <div className="leading-snug">
          <span className="font-medium">{request.title}</span>
          {request.year && (
            <span className="text-sm text-muted-foreground ml-1">({request.year})</span>
          )}
          <Badge className={`${statusConfig.color} text-white sm:hidden ml-2 align-middle text-[10px] md:text-xs px-1.5 md:px-2 py-0.5`}>
            {statusConfig.icon}
            <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
          </Badge>
        </div>
        <div className="flex items-center gap-2 text-sm text-muted-foreground flex-wrap mt-0.5">
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
              {request.monitorFuture && (
                <Badge variant="secondary" className="text-xs">Future</Badge>
              )}
            </>
          )}
          {request.seasonNumber && request.mediaType !== 'series' && (
            <span>Season {request.seasonNumber}</span>
          )}
          {request.episodeNumber && (
            <span>Episode {request.episodeNumber}</span>
          )}
          <span>•</span>
          <span>{formatDistanceToNow(new Date(request.createdAt), { addSuffix: true })}</span>
        </div>
        {hasActiveDownload && (
          <div className="mt-2 max-w-48">
            <Progress value={progress} variant={isMovie ? 'movie' : 'tv'} className="h-1.5" />
            <div className="flex justify-between text-xs text-muted-foreground mt-0.5">
              <span>{Math.round(progress)}%</span>
              <span>{isComplete ? 'Importing' : isPaused ? 'Paused' : isActive ? formatEta(eta) : 'Queued'}</span>
            </div>
          </div>
        )}
      </div>

      <Badge className={`${statusConfig.color} text-white hidden sm:flex shrink-0 text-[10px] md:text-xs px-1.5 md:px-2 py-0.5`}>
        {statusConfig.icon}
        <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
      </Badge>
    </button>
  )
}
