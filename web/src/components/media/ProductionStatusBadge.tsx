import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

export type ProductionStatus = 'continuing' | 'ended' | 'upcoming'

interface ProductionStatusBadgeProps {
  status: ProductionStatus
  className?: string
}

const statusConfig: Record<ProductionStatus, {
  label: string
  variant: 'default' | 'secondary' | 'outline'
}> = {
  continuing: { label: 'Continuing', variant: 'default' },
  ended: { label: 'Ended', variant: 'secondary' },
  upcoming: { label: 'Upcoming', variant: 'outline' },
}

export function ProductionStatusBadge({ status, className }: ProductionStatusBadgeProps) {
  const config = statusConfig[status] || { label: status, variant: 'secondary' as const }
  return (
    <Badge variant={config.variant} className={cn(className)}>
      {config.label}
    </Badge>
  )
}
