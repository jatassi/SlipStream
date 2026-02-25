import { CheckCircle, Clock, Download, XCircle } from 'lucide-react'

import type { RequestStatus } from '@/types'

export type SearchModalState = {
  open: boolean
  mediaType: 'movie' | 'series'
  mediaId: number
  mediaTitle: string
  tmdbId?: number
  imdbId?: string
  tvdbId?: number
  qualityProfileId: number
  year?: number
  season?: number
  pendingSeasons?: number[]
}

export const STATUS_CONFIG: Record<
  RequestStatus,
  { label: string; icon: React.ReactNode; color: string }
> = {
  pending: { label: 'Pending', icon: <Clock className="size-4" />, color: 'bg-yellow-500' },
  approved: { label: 'Approved', icon: <CheckCircle className="size-4" />, color: 'bg-blue-500' },
  denied: { label: 'Denied', icon: <XCircle className="size-4" />, color: 'bg-red-500' },
  downloading: {
    label: 'Downloading',
    icon: <Download className="size-4" />,
    color: 'bg-purple-500',
  },
  failed: { label: 'Failed', icon: <XCircle className="size-4" />, color: 'bg-red-700' },
  available: {
    label: 'Available',
    icon: <CheckCircle className="size-4" />,
    color: 'bg-green-500',
  },
  cancelled: { label: 'Cancelled', icon: <XCircle className="size-4" />, color: 'bg-gray-500' },
}
