import type { HealthStatus } from '@/types/health'

export function getStatusColor(status: HealthStatus): string {
  switch (status) {
    case 'ok': {
      return 'text-green-500'
    }
    case 'warning': {
      return 'text-yellow-500'
    }
    case 'error': {
      return 'text-red-500'
    }
    default: {
      return 'text-muted-foreground'
    }
  }
}

export function getStatusBgColor(status: HealthStatus): string {
  switch (status) {
    case 'ok': {
      return 'bg-green-500/10'
    }
    case 'warning': {
      return 'bg-yellow-500/10'
    }
    case 'error': {
      return 'bg-red-500/10'
    }
    default: {
      return 'bg-muted'
    }
  }
}
