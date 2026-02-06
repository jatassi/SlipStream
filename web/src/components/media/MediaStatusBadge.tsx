import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Clock, AlertCircle, ArrowDownCircle, XCircle, ArrowUpCircle, CheckCircle } from 'lucide-react'

export type MediaStatus = 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'

interface MediaStatusBadgeProps {
  status: MediaStatus
  className?: string
}

const statusConfig: Record<MediaStatus, {
  label: string
  variant: 'default' | 'secondary' | 'destructive' | 'outline'
  icon: React.ReactNode
  className?: string
}> = {
  unreleased: { label: 'Unreleased', variant: 'outline', icon: <Clock className="size-3" />, className: 'border-amber-500 text-amber-500' },
  missing: { label: 'Missing', variant: 'destructive', icon: <AlertCircle className="size-3" /> },
  downloading: { label: 'Downloading', variant: 'default', icon: <ArrowDownCircle className="size-3" />, className: 'bg-blue-600 hover:bg-blue-600 text-white' },
  failed: { label: 'Failed', variant: 'destructive', icon: <XCircle className="size-3" />, className: 'bg-red-900/50 border border-red-500 text-red-400' },
  upgradable: { label: 'Upgradable', variant: 'outline', icon: <ArrowUpCircle className="size-3" />, className: 'border-yellow-500 text-yellow-500' },
  available: { label: 'Available', variant: 'default', icon: <CheckCircle className="size-3" />, className: 'bg-green-600 hover:bg-green-600 text-white' },
}

export function MediaStatusBadge({ status, className }: MediaStatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.missing
  return (
    <Badge variant={config.variant} className={cn('gap-1', config.className, className)}>
      {config.icon}
      {config.label}
    </Badge>
  )
}
