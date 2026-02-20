import { useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { formatDistanceToNow } from 'date-fns'
import { CheckCircle, Clock, Download, Loader2, User, XCircle } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { usePortalDownloads, useRequests } from '@/hooks'
import type { PortalDownload, Request, RequestStatus } from '@/types'

import { computeDownloadProgress, findMatchingDownloads } from './request-download-utils'
import { SearchSection } from './request-list-search'

const STATUS_CONFIG: Record<
  RequestStatus,
  { label: string; icon: React.ReactNode; color: string }
> = {
  pending: { label: 'Pending', icon: <Clock className="size-3 md:size-4" />, color: 'bg-yellow-500' },
  approved: { label: 'Approved', icon: <CheckCircle className="size-3 md:size-4" />, color: 'bg-blue-500' },
  denied: { label: 'Denied', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-red-500' },
  downloading: { label: 'Downloading', icon: <Download className="size-3 md:size-4" />, color: 'bg-purple-500' },
  failed: { label: 'Failed', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-red-700' },
  available: { label: 'Available', icon: <CheckCircle className="size-3 md:size-4" />, color: 'bg-green-500' },
  cancelled: { label: 'Cancelled', icon: <XCircle className="size-3 md:size-4" />, color: 'bg-gray-500' },
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
      void navigate({ to: '/requests/search', search: { q: searchQuery.trim() } })
    }
  }

  const goToRequestDetail = (id: number) => {
    void navigate({ to: '/requests/$id', params: { id: String(id) } })
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <SearchSection
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        searchFocused={searchFocused}
        setSearchFocused={setSearchFocused}
        onSearch={handleSearch}
      />
      <RequestsSection
        filter={filter}
        setFilter={setFilter}
        sortedRequests={sortedRequests}
        isLoading={isLoading}
        onRequestClick={goToRequestDetail}
      />
    </div>
  )
}

type RequestsSectionProps = {
  filter: 'mine' | 'all'
  setFilter: (f: 'mine' | 'all') => void
  sortedRequests: Request[]
  isLoading: boolean
  onRequestClick: (id: number) => void
}

function RequestsSection({
  filter,
  setFilter,
  sortedRequests,
  isLoading,
  onRequestClick,
}: RequestsSectionProps) {
  return (
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
          <RequestsList
            requests={sortedRequests}
            isLoading={isLoading}
            showUser={filter === 'all'}
            onRequestClick={onRequestClick}
          />
        </div>
      </div>
    </section>
  )
}

function RequestsList(props: {
  requests: Request[]
  isLoading: boolean
  showUser: boolean
  onRequestClick: (id: number) => void
}) {
  const { requests, isLoading, showUser, onRequestClick } = props

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  if (requests.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <Clock className="text-muted-foreground/50 mb-4 size-12" />
        <p className="text-muted-foreground">No requests yet</p>
        <p className="text-muted-foreground/70 text-sm">Search above to request content</p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {requests.map((request) => (
        <RequestCard
          key={request.id}
          request={request}
          showUser={showUser}
          onClick={() => onRequestClick(request.id)}
        />
      ))}
    </div>
  )
}

function RequestCard(props: { request: Request; showUser?: boolean; onClick: () => void }) {
  const { request, showUser, onClick } = props
  const { data: downloads } = usePortalDownloads()
  const statusConfig = STATUS_CONFIG[request.status]
  const matchingDownloads = findMatchingDownloads(downloads ?? [], request)
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
          type={isMovie ? 'movie' : 'series'}
          className="h-full w-full"
        />
      </div>
      <RequestCardBody
        request={request}
        showUser={showUser}
        statusConfig={statusConfig}
        matchingDownloads={matchingDownloads}
        isMovie={isMovie}
      />
      <Badge
        className={`${statusConfig.color} hidden shrink-0 px-1.5 py-0.5 text-[10px] text-white sm:flex md:px-2 md:text-xs`}
      >
        {statusConfig.icon}
        <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
      </Badge>
    </button>
  )
}

function RequestCardBody(props: {
  request: Request
  showUser?: boolean
  statusConfig: { label: string; icon: React.ReactNode; color: string }
  matchingDownloads: PortalDownload[]
  isMovie: boolean
}) {
  const { request, showUser, statusConfig, matchingDownloads, isMovie } = props
  const downloadProgress = computeDownloadProgress(matchingDownloads)

  return (
    <div className="min-w-0 flex-1">
      {showUser && request.user ? (
        <div className="text-muted-foreground mb-0.5 flex items-center gap-1 text-xs">
          <User className="size-3" />
          <span>{request.user.displayName ?? request.user.username}</span>
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
      <RequestMetadata request={request} isMovie={isMovie} />
      {downloadProgress ? (
        <div className="mt-2 max-w-48">
          <Progress
            value={downloadProgress.progress}
            variant={isMovie ? 'movie' : 'tv'}
            className="h-1.5"
          />
          <div className="text-muted-foreground mt-0.5 flex justify-between text-xs">
            <span>{Math.round(downloadProgress.progress)}%</span>
            <span>{downloadProgress.statusLabel}</span>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function RequestMetadata({ request, isMovie }: { request: Request; isMovie: boolean }) {
  return (
    <div className="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-2 text-sm">
      <Badge
        variant="outline"
        className={`text-xs capitalize ${
          isMovie ? 'border-movie-500/50 text-movie-400' : 'border-tv-500/50 text-tv-400'
        }`}
      >
        {request.mediaType}
      </Badge>
      <SeriesMetadata request={request} />
      {request.seasonNumber && request.mediaType !== 'series' ? (
        <span>Season {request.seasonNumber}</span>
      ) : null}
      {request.episodeNumber ? <span>Episode {request.episodeNumber}</span> : null}
      <span>•</span>
      <span>{formatDistanceToNow(new Date(request.createdAt), { addSuffix: true })}</span>
    </div>
  )
}

function SeriesMetadata({ request }: { request: Request }) {
  if (request.mediaType !== 'series') {
    return null
  }

  return (
    <>
      {request.requestedSeasons.length > 0 ? (
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
  )
}
