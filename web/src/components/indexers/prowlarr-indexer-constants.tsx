import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  Globe,
  Lock,
  Unlock,
  XCircle,
} from 'lucide-react'

import type { ContentType, Privacy, Protocol, ProwlarrIndexerStatus } from '@/types'

export const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-3" />,
  'semi-private': <Unlock className="size-3" />,
  private: <Lock className="size-3" />,
}

export const privacyIconsMd: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-4" />,
  'semi-private': <Unlock className="size-4" />,
  private: <Lock className="size-4" />,
}

export const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500',
  'semi-private': 'bg-yellow-500/10 text-yellow-500',
  private: 'bg-red-500/10 text-red-500',
}

export const privacyColorsInteractive: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500 hover:bg-green-500/20',
  'semi-private': 'bg-yellow-500/10 text-yellow-500 hover:bg-yellow-500/20',
  private: 'bg-red-500/10 text-red-500 hover:bg-red-500/20',
}

export const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500',
  usenet: 'bg-purple-500/10 text-purple-500',
}

export const protocolColorsInteractive: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500 hover:bg-blue-500/20',
  usenet: 'bg-purple-500/10 text-purple-500 hover:bg-purple-500/20',
}

export const statusIcons: Record<ProwlarrIndexerStatus, React.ReactNode> = {
  0: <CheckCircle2 className="size-4 text-green-500" />,
  1: <AlertTriangle className="size-4 text-yellow-500" />,
  2: <Ban className="text-muted-foreground size-4" />,
  3: <XCircle className="size-4 text-red-500" />,
}

export const statusColors: Record<ProwlarrIndexerStatus, string> = {
  0: 'text-green-500',
  1: 'text-yellow-500',
  2: 'text-muted-foreground',
  3: 'text-red-500',
}

export const contentTypeColors: Record<ContentType, string> = {
  movies: 'bg-amber-500/10 text-amber-500',
  series: 'bg-cyan-500/10 text-cyan-500',
  both: 'bg-gray-500/10 text-gray-400',
}
