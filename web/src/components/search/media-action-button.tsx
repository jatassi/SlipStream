import { Check, Clock, Play } from 'lucide-react'

import { Button } from '@/components/ui/button'
import type { PortalDownload } from '@/types'

import { DownloadStatusCard } from './download-status-card'

type MediaActionButtonProps = {
  hasActiveDownload: boolean
  activeDownload: PortalDownload | undefined
  isInLibrary: boolean
  isAvailable: boolean
  isApproved: boolean
  isPending: boolean
  onAction?: () => void
  actionLabel: string
  actionIcon: React.ReactNode
  handleAdd: () => void
  trailerUrl?: string
}

export function MediaActionButton(props: MediaActionButtonProps) {
  if (props.hasActiveDownload && props.activeDownload) {
    return <DownloadStatusCard download={props.activeDownload} />
  }

  const actionButton = resolveActionButton(props)

  if (!actionButton && !props.trailerUrl) {
    return null
  }

  return (
    <div className="flex gap-2">
      {actionButton}
      <TrailerButton url={props.trailerUrl} />
    </div>
  )
}

function resolveActionButton(props: MediaActionButtonProps) {
  if ((props.isInLibrary || props.isAvailable) && !props.onAction) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Check className="mr-2 size-4" />
        In Library
      </Button>
    )
  }
  if (props.isApproved) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Check className="mr-2 size-4" />
        Approved
      </Button>
    )
  }
  if (props.isPending) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Clock className="mr-2 size-4" />
        Requested
      </Button>
    )
  }
  if (props.onAction) {
    return (
      <Button variant="default" size="sm" onClick={props.handleAdd}>
        {props.actionIcon}
        {props.actionLabel}
      </Button>
    )
  }
  return null
}

function TrailerButton({ url }: { url?: string }) {
  if (!url) {
    return null
  }
  return (
    <Button
      variant="secondary"
      size="sm"
      onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
    >
      <Play className="mr-2 size-4" />
      Trailer
    </Button>
  )
}
