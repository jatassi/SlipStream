import { useRef, useState } from 'react'

import { formatDistanceToNow } from 'date-fns'
import { Check, Ellipsis, Loader2, X } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { StatusPill } from '@/components/media/status-pill'
import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'
import type { Request } from '@/types'

import type { RequestAction } from './request-actions'
import { REQUEST_STATUS_LABEL, requestStatusTone } from './request-status'

const CONTROL =
  'press size-tap focus-visible:ring-ring flex shrink-0 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]'

type RequestRowProps = {
  request: Request
  requester: string | undefined
  isProcessing: boolean
  onAction: (action: RequestAction) => void
}

function mediaTypeLabel(request: Request): string {
  if (request.mediaType === 'movie') {
    return 'Movie'
  }
  if (request.mediaType === 'series') {
    return 'Series'
  }
  return request.mediaType === 'season' ? 'Season' : 'Episode'
}

function seasonLabel(request: Request): string | null {
  if (request.requestedSeasons.length > 0) {
    if (request.requestedSeasons.length <= 3) {
      return `S${request.requestedSeasons.join(', S')}`
    }
    return `${request.requestedSeasons.length} seasons`
  }
  if (request.seasonNumber !== null) {
    return `Season ${request.seasonNumber}`
  }
  return null
}

function metaLine(request: Request, requester: string | undefined): string {
  const parts = [mediaTypeLabel(request)]
  const seasons = seasonLabel(request)
  if (seasons !== null) {
    parts.push(seasons)
  }
  if (requester !== undefined) {
    parts.push(`by ${requester}`)
  }
  parts.push(formatDistanceToNow(new Date(request.createdAt), { addSuffix: true }))
  return parts.join(' · ')
}

function RequestThumbnail({ request }: { request: Request }) {
  return (
    <PosterImage
      url={request.posterUrl}
      tmdbId={request.tmdbId}
      tvdbId={request.tvdbId}
      alt={request.title}
      type={request.mediaType === 'movie' ? 'movie' : 'series'}
      size="w92"
      className="rounded-thumb h-15 w-10 shrink-0 object-cover"
    />
  )
}

function RequestBody({ request, requester }: { request: Request; requester: string | undefined }) {
  return (
    <div className="min-w-0 flex-1">
      <div className="flex items-baseline gap-2">
        <span className="text-body truncate font-medium">{request.title}</span>
        {request.year === null ? null : (
          <span className="nums text-footnote text-muted-foreground shrink-0">{request.year}</span>
        )}
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
        <StatusPill
          status={requestStatusTone(request.status)}
          label={REQUEST_STATUS_LABEL[request.status]}
        />
        <span className="text-footnote text-muted-foreground truncate">
          {metaLine(request, requester)}
        </span>
      </div>
      {request.deniedReason === null ? null : (
        <p className="text-footnote text-destructive mt-1 truncate">
          Reason: {request.deniedReason}
        </p>
      )}
    </div>
  )
}

function ApproveButton({
  title,
  isProcessing,
  onApprove,
}: {
  title: string
  isProcessing: boolean
  onApprove: () => void
}) {
  const Icon = isProcessing ? Loader2 : Check
  return (
    <button
      type="button"
      onClick={onApprove}
      disabled={isProcessing}
      aria-label={`Approve ${title}`}
      className={`${CONTROL} text-foreground`}
    >
      <Icon className={isProcessing ? 'size-5 animate-spin' : 'size-5'} />
    </button>
  )
}

function DenyButton({
  title,
  isProcessing,
  onOpen,
}: {
  title: string
  isProcessing: boolean
  onOpen: () => void
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      disabled={isProcessing}
      aria-label={`Deny ${title}`}
      className={`${CONTROL} text-destructive`}
    >
      <X className="size-5" />
    </button>
  )
}

function moreActions(request: Request, onAction: (action: RequestAction) => void): ActionItem[] {
  const deleteAction: ActionItem = {
    label: 'Delete request',
    destructive: true,
    confirm: {
      title: `Delete ${request.title}?`,
      description: 'The request is removed permanently. This cannot be undone.',
      actions: [
        {
          label: 'Delete request',
          destructive: true,
          onClick: () => {
            onAction('delete')
          },
        },
      ],
    },
  }

  if (request.status !== 'pending') {
    return [deleteAction]
  }

  return [
    {
      label: 'Approve & Manual Search',
      onClick: () => {
        onAction('approve-manual-search')
      },
    },
    {
      label: 'Approve & Auto Search',
      onClick: () => {
        onAction('approve-auto-search')
      },
    },
    deleteAction,
  ]
}

function PendingControls({
  request,
  isProcessing,
  onAction,
  onDeny,
}: {
  request: Request
  isProcessing: boolean
  onAction: (action: RequestAction) => void
  onDeny: () => void
}) {
  if (request.status !== 'pending') {
    return null
  }
  return (
    <>
      <ApproveButton
        title={request.title}
        isProcessing={isProcessing}
        onApprove={() => {
          onAction('approve')
        }}
      />
      <DenyButton title={request.title} isProcessing={isProcessing} onOpen={onDeny} />
    </>
  )
}

function MoreButton({ title, onOpen }: { title: string; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`More actions for ${title}`}
      className={`${CONTROL} text-muted-foreground`}
    >
      <Ellipsis className="size-5" />
    </button>
  )
}

function DenyConfirm({
  request,
  open,
  onOpenChange,
  onAction,
}: {
  request: Request
  open: boolean
  onOpenChange: (open: boolean) => void
  onAction: (action: RequestAction) => void
}) {
  return (
    <ActionPresenter
      open={open}
      onOpenChange={onOpenChange}
      wide="dialog"
      title={`Deny ${request.title}?`}
      description="The requester sees this request as denied. It stays in the Denied list."
      actions={[
        {
          label: 'Deny request',
          destructive: true,
          onClick: () => {
            onAction('deny')
          },
        },
      ]}
    />
  )
}

export function RequestRow({ request, requester, isProcessing, onAction }: RequestRowProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [denyOpen, setDenyOpen] = useState(false)
  const anchor = useRef<HTMLDivElement>(null)

  return (
    <div
      ref={anchor}
      role="group"
      aria-label={request.title}
      className="flex items-center gap-3 px-4 py-3"
    >
      <RequestThumbnail request={request} />
      <RequestBody request={request} requester={requester} />
      <PendingControls
        request={request}
        isProcessing={isProcessing}
        onAction={onAction}
        onDeny={() => {
          setDenyOpen(true)
        }}
      />
      <MoreButton
        title={request.title}
        onOpen={() => {
          setMenuOpen(true)
        }}
      />
      <ActionPresenter
        open={menuOpen}
        onOpenChange={setMenuOpen}
        title={request.title}
        actions={moreActions(request, onAction)}
        anchor={anchor}
      />
      <DenyConfirm
        request={request}
        open={denyOpen}
        onOpenChange={setDenyOpen}
        onAction={onAction}
      />
    </div>
  )
}
