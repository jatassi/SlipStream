import { CheckCircle2, AlertTriangle, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HealthStatus } from '@/types/health'

interface StatusIndicatorProps {
  status: HealthStatus
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeClasses = {
  sm: 'size-3',
  md: 'size-4',
  lg: 'size-5',
}

export function StatusIndicator({ status, size = 'md', className }: StatusIndicatorProps) {
  const sizeClass = sizeClasses[size]

  switch (status) {
    case 'ok':
      return (
        <CheckCircle2
          className={cn(sizeClass, 'text-green-500', className)}
          aria-label="Healthy"
        />
      )
    case 'warning':
      return (
        <AlertTriangle
          className={cn(sizeClass, 'text-yellow-500', className)}
          aria-label="Warning"
        />
      )
    case 'error':
      return (
        <XCircle
          className={cn(sizeClass, 'text-red-500', className)}
          aria-label="Error"
        />
      )
    default:
      return null
  }
}

export function getStatusColor(status: HealthStatus): string {
  switch (status) {
    case 'ok':
      return 'text-green-500'
    case 'warning':
      return 'text-yellow-500'
    case 'error':
      return 'text-red-500'
    default:
      return 'text-muted-foreground'
  }
}

export function getStatusBgColor(status: HealthStatus): string {
  switch (status) {
    case 'ok':
      return 'bg-green-500/10'
    case 'warning':
      return 'bg-yellow-500/10'
    case 'error':
      return 'bg-red-500/10'
    default:
      return 'bg-muted'
  }
}
