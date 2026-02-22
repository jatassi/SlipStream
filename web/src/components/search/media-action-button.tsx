import { Check, Clock } from 'lucide-react'

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
}

export function MediaActionButton({
  hasActiveDownload,
  activeDownload,
  isInLibrary,
  isAvailable,
  isApproved,
  isPending,
  onAction,
  actionLabel,
  actionIcon,
  handleAdd,
}: MediaActionButtonProps) {
  if (hasActiveDownload && activeDownload) {
    return <DownloadStatusCard download={activeDownload} />
  }

  if ((isInLibrary || isAvailable) && !onAction) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Check className="mr-2 size-4" />
        In Library
      </Button>
    )
  }

  if (isApproved) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Check className="mr-2 size-4" />
        Approved
      </Button>
    )
  }

  if (isPending) {
    return (
      <Button variant="secondary" size="sm" disabled>
        <Clock className="mr-2 size-4" />
        Requested
      </Button>
    )
  }

  if (onAction) {
    return (
      <Button variant="default" size="sm" onClick={handleAdd}>
        {actionIcon}
        {actionLabel}
      </Button>
    )
  }

  return null
}
