import { cn } from '@/lib/utils'

import type { MediaStatus } from './media-status'
import { MEDIA_STATUS_LABEL, mediaStatusColor } from './media-status'
import { StatusDot } from './status-dot'

type StatusPillProps = {
  status: MediaStatus
  label?: string
  className?: string
}

export function StatusPill({ status, label, className }: StatusPillProps) {
  const color = mediaStatusColor(status)
  return (
    <span
      className={cn(
        'text-caption inline-flex h-6 items-center gap-1.5 rounded-full px-2 font-semibold',
        className,
      )}
      style={{
        color,
        background: `color-mix(in oklch, ${color} 14%, transparent)`,
      }}
    >
      <StatusDot status={status} decorative />
      {label ?? MEDIA_STATUS_LABEL[status]}
    </span>
  )
}
