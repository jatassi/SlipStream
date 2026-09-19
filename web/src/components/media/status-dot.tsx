import { cn } from '@/lib/utils'

import type { MediaStatus } from './media-status'
import { MEDIA_STATUS_LABEL, mediaStatusColor } from './media-status'

type StatusDotProps = {
  status: MediaStatus
  className?: string
  decorative?: boolean
}

export function StatusDot({ status, className, decorative }: StatusDotProps) {
  const color = mediaStatusColor(status)
  if (decorative) {
    return (
      <span
        className={cn('inline-block size-1.5 shrink-0 rounded-full', className)}
        style={{ background: color }}
        aria-hidden
      />
    )
  }
  return (
    <span
      className={cn('inline-block size-1.5 shrink-0 rounded-full', className)}
      style={{ background: color }}
      role="img"
      aria-label={MEDIA_STATUS_LABEL[status]}
    />
  )
}
