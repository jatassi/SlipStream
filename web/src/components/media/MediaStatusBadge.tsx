import {
  ArrowDownCircle,
  ArrowUpCircle,
  Binoculars,
  CheckCircle,
  Clock,
  XCircle,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export type MediaStatus =
  | 'unreleased'
  | 'missing'
  | 'downloading'
  | 'failed'
  | 'upgradable'
  | 'available'

type MediaStatusBadgeProps = {
  status: MediaStatus
  iconOnly?: boolean
  className?: string
}

const statusConfig: Record<
  MediaStatus,
  {
    label: string
    variant: 'default' | 'secondary' | 'destructive' | 'outline'
    icon: React.ReactNode
    iconClassName: string
    className?: string
  }
> = {
  unreleased: {
    label: 'Unreleased',
    variant: 'default',
    icon: <Clock className="size-3" />,
    iconClassName: 'text-blue-500',
    className: 'bg-blue-600 hover:bg-blue-600 text-white',
  },
  missing: {
    label: 'Missing',
    variant: 'default',
    icon: <Binoculars className="size-3" />,
    iconClassName: 'text-amber-500',
    className: 'bg-amber-600 hover:bg-amber-600 text-white',
  },
  downloading: {
    label: 'Downloading',
    variant: 'default',
    icon: <ArrowDownCircle className="size-3" />,
    iconClassName: 'text-purple-500',
    className: 'bg-purple-600 hover:bg-purple-600 text-white',
  },
  failed: {
    label: 'Failed',
    variant: 'default',
    icon: <XCircle className="size-3" />,
    iconClassName: 'text-red-500',
    className: 'bg-red-600 hover:bg-red-600 text-white',
  },
  upgradable: {
    label: 'Upgradable',
    variant: 'default',
    icon: <ArrowUpCircle className="size-3" />,
    iconClassName: 'text-yellow-500',
    className: 'bg-yellow-500 hover:bg-yellow-500 text-black',
  },
  available: {
    label: 'Available',
    variant: 'default',
    icon: <CheckCircle className="size-3" />,
    iconClassName: 'text-green-500',
    className: 'bg-green-600 hover:bg-green-600 text-white',
  },
}

export function MediaStatusBadge({ status, iconOnly, className }: MediaStatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.missing
  if (iconOnly) {
    return <span className={cn(config.iconClassName, className)}>{config.icon}</span>
  }
  return (
    <Badge variant={config.variant} className={cn('gap-1', config.className, className)}>
      {config.icon}
      {config.label}
    </Badge>
  )
}
