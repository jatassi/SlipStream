import type { HealthStatus } from '@/types/health'

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
