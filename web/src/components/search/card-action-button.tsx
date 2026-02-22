import { Check, CheckCircle, Clock } from 'lucide-react'

import { Button } from '@/components/ui/button'

import { DownloadProgressBar } from './download-progress-bar'

const BTN_CLASS = 'w-full text-xs md:text-sm'
const ICON_CLASS = 'mr-1 size-3 md:mr-2 md:size-4'

type CardActionButtonProps = {
  mediaType: 'movie' | 'series'
  hasActiveDownload: boolean
  activeDownloadMediaId?: number
  isInLibrary: boolean
  canRequest: boolean
  hasExistingRequest: boolean
  isAvailable: boolean
  isApproved: boolean
  isOwnRequest: boolean
  viewRequestId?: number
  onAction?: () => void
  onViewRequest?: (id: number) => void
  actionLabel: string
  actionIcon: React.ReactNode
  requestedLabel: string
}

export function CardActionButton(props: CardActionButtonProps) {
  if (props.hasActiveDownload) {
    return <DownloadProgressBar mediaId={props.activeDownloadMediaId} mediaType={props.mediaType} />
  }
  if (props.isInLibrary && !props.canRequest) {
    return <StatusButton icon={<Check className={ICON_CLASS} />} label="In Library" />
  }
  if (props.isInLibrary && props.canRequest) {
    return (
      <Button
        variant="default"
        size="sm"
        className={BTN_CLASS}
        onClick={(e: React.MouseEvent) => {
          e.stopPropagation()
          props.onAction?.()
        }}
      >
        {props.actionIcon}
        {props.actionLabel}
      </Button>
    )
  }
  if (props.hasExistingRequest) {
    return <ExistingRequestButton {...props} />
  }
  return (
    <Button
      variant="default"
      size="sm"
      className={BTN_CLASS}
      onClick={(e: React.MouseEvent) => {
        e.stopPropagation()
        props.onAction?.()
      }}
    >
      {props.actionIcon}
      {props.actionLabel}
    </Button>
  )
}

function StatusButton({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <Button variant="secondary" size="sm" className={BTN_CLASS} disabled>
      {icon}
      {label}
    </Button>
  )
}

function ExistingRequestButton({
  isAvailable,
  isApproved,
  isOwnRequest,
  viewRequestId,
  onViewRequest,
  requestedLabel,
}: CardActionButtonProps) {
  if (isAvailable) {
    return <StatusButton icon={<CheckCircle className={ICON_CLASS} />} label="Available" />
  }
  if (isApproved) {
    return <StatusButton icon={<Check className={ICON_CLASS} />} label="Approved" />
  }
  if (isOwnRequest) {
    return <StatusButton icon={<Clock className={ICON_CLASS} />} label={requestedLabel} />
  }
  if (viewRequestId && onViewRequest) {
    return (
      <Button
        variant="secondary"
        size="sm"
        className={BTN_CLASS}
        onClick={(e: React.MouseEvent) => {
          e.stopPropagation()
          onViewRequest(viewRequestId)
        }}
      >
        View Request
      </Button>
    )
  }
  return <StatusButton icon={<Clock className={ICON_CLASS} />} label={requestedLabel} />
}
