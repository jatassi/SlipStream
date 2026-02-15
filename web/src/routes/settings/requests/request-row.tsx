import { formatDistanceToNow } from 'date-fns'

import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import type { Request } from '@/types'

import { RequestActions } from './request-actions'
import { STATUS_CONFIG } from './status-config'

export type RequestRowProps = {
  request: Request
  selected: boolean
  isProcessing: boolean
  onToggleSelect: () => void
  onApproveOnly: () => void
  onApproveAndManualSearch: () => void
  onApproveAndAutoSearch: () => void
  onDeny: () => void
  onDelete: () => void
}

export function RequestRow({
  request,
  selected,
  isProcessing,
  onToggleSelect,
  onApproveOnly,
  onApproveAndManualSearch,
  onApproveAndAutoSearch,
  onDeny,
  onDelete,
}: RequestRowProps) {
  const statusConfig = STATUS_CONFIG[request.status]
  const isPending = request.status === 'pending'

  return (
    <div className="hover:bg-muted/40 flex items-center gap-4 p-4">
      <Checkbox checked={selected} onCheckedChange={onToggleSelect} />

      <div className="h-15 w-10 flex-shrink-0 overflow-hidden rounded">
        <PosterImage
          url={request.posterUrl}
          alt={request.title}
          type={request.mediaType === 'movie' ? 'movie' : 'series'}
          className="h-full w-full"
        />
      </div>

      <div className="min-w-0 flex-1">
        <RequestTitle title={request.title} year={request.year} />
        <RequestMeta request={request} />
        {request.deniedReason ? (
          <p className="mt-1 text-sm text-red-500">Reason: {request.deniedReason}</p>
        ) : null}
      </div>

      <Badge className={`${statusConfig.color} text-white`}>
        {statusConfig.icon}
        <span className="ml-1">{statusConfig.label}</span>
      </Badge>

      <RequestActions
        isPending={isPending}
        isProcessing={isProcessing}
        onApproveOnly={onApproveOnly}
        onApproveAndManualSearch={onApproveAndManualSearch}
        onApproveAndAutoSearch={onApproveAndAutoSearch}
        onDeny={onDeny}
        onDelete={onDelete}
      />
    </div>
  )
}

function RequestTitle({ title, year }: { title: string; year: number | null }) {
  return (
    <div className="flex items-center gap-2">
      <span className="truncate font-medium">{title}</span>
      {year ? <span className="text-muted-foreground text-sm">({year})</span> : null}
    </div>
  )
}

function RequestMeta({ request }: { request: Request }) {
  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
      <Badge variant="outline" className="text-xs capitalize">
        {request.mediaType}
      </Badge>
      {request.mediaType === 'series' && <SeriesSeasonInfo request={request} />}
      {request.seasonNumber && request.mediaType !== 'series' ? (
        <span>Season {request.seasonNumber}</span>
      ) : null}
      {request.episodeNumber ? <span>Episode {request.episodeNumber}</span> : null}
      <span>•</span>
      <span>{formatDistanceToNow(new Date(request.createdAt), { addSuffix: true })}</span>
      {request.user ? (
        <>
          <span>•</span>
          <span>by {request.user.displayName ?? request.user.username}</span>
        </>
      ) : null}
    </div>
  )
}

function SeriesSeasonInfo({ request }: { request: Request }) {
  return (
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
  )
}
