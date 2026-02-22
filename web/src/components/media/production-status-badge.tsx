import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type ProductionStatus = 'continuing' | 'ended' | 'upcoming'

type ProductionStatusBadgeProps = {
  status: ProductionStatus
  className?: string
}

const statusConfig: Record<
  ProductionStatus,
  {
    label: string
    variant: 'default' | 'secondary' | 'outline'
  }
> = {
  continuing: { label: 'Continuing', variant: 'default' },
  ended: { label: 'Ended', variant: 'secondary' },
  upcoming: { label: 'Upcoming', variant: 'outline' },
}

export function ProductionStatusBadge({ status, className }: ProductionStatusBadgeProps) {
  const config = statusConfig[status]
  return (
    <Badge variant={config.variant} className={cn(className)}>
      {config.label}
    </Badge>
  )
}
