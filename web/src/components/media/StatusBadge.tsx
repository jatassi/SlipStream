import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

type MovieStatus = 'missing' | 'downloading' | 'available'
type SeriesStatus = 'continuing' | 'ended' | 'upcoming'

interface StatusBadgeProps {
  status: MovieStatus | SeriesStatus
  className?: string
}

const statusConfig: Record<
  MovieStatus | SeriesStatus,
  { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }
> = {
  available: { label: 'Available', variant: 'default' },
  downloading: { label: 'Downloading', variant: 'secondary' },
  missing: { label: 'Missing', variant: 'destructive' },
  continuing: { label: 'Continuing', variant: 'default' },
  ended: { label: 'Ended', variant: 'secondary' },
  upcoming: { label: 'Upcoming', variant: 'outline' },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] || { label: status, variant: 'secondary' as const }

  return (
    <Badge variant={config.variant} className={cn(className)}>
      {config.label}
    </Badge>
  )
}
