import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type QualityBadgeProps = {
  quality?: string
  resolution?: string
  className?: string
}

export function QualityBadge({ quality, resolution, className }: QualityBadgeProps) {
  const label = quality ?? resolution ?? 'Unknown'

  // Determine variant based on resolution
  let variant: 'default' | 'secondary' | 'outline' = 'secondary'
  if (quality?.includes('2160') || resolution === '4K') {
    variant = 'default'
  } else if (quality?.includes('1080') || resolution === 'Full HD') {
    variant = 'secondary'
  }

  return (
    <Badge variant={variant} className={cn('font-mono text-xs', className)}>
      {label}
    </Badge>
  )
}
