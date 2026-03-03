import { CheckCircle, Clock, Download, Loader2, XCircle } from 'lucide-react'

import type { RequestStatus } from '@/types'

export type StatusConfigEntry = {
  label: string
  icon: React.ReactNode
  color: string
}

type IconSize = 'xs' | 'sm' | 'md'

const iconClass = (size: IconSize) => {
  if (size === 'md') {return 'size-4 md:size-5'}
  if (size === 'xs') {return 'size-3 md:size-4'}
  return 'size-4'
}

export function getStatusConfig(size: IconSize = 'sm'): Record<RequestStatus, StatusConfigEntry> {
  return {
    pending: {
      label: 'Pending',
      icon: <Clock className={iconClass(size)} />,
      color: 'bg-yellow-500',
    },
    approved: {
      label: 'Approved',
      icon: <CheckCircle className={iconClass(size)} />,
      color: 'bg-blue-500',
    },
    searching: {
      label: 'Searching',
      icon: <Loader2 className={`${iconClass(size)} animate-spin`} />,
      color: 'bg-blue-500',
    },
    denied: {
      label: 'Denied',
      icon: <XCircle className={iconClass(size)} />,
      color: 'bg-red-500',
    },
    downloading: {
      label: 'Downloading',
      icon: <Download className={iconClass(size)} />,
      color: 'bg-purple-500',
    },
    failed: {
      label: 'Failed',
      icon: <XCircle className={iconClass(size)} />,
      color: 'bg-red-700',
    },
    available: {
      label: 'Available',
      icon: <CheckCircle className={iconClass(size)} />,
      color: 'bg-green-500',
    },
    cancelled: {
      label: 'Cancelled',
      icon: <XCircle className={iconClass(size)} />,
      color: 'bg-gray-500',
    },
  }
}
