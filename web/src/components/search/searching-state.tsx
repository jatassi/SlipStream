import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { MediaTheme, ResolvedSize } from './media-search-monitor-types'

const HEIGHT_CLASS: Record<ResolvedSize, string> = {
  xs: 'h-6 text-xs',
  sm: 'h-8 text-xs',
  lg: '',
}

type SearchingStateProps = {
  size: ResolvedSize
  theme: MediaTheme
  mode: 'manual' | 'auto'
}

export function SearchingState({ size, theme, mode }: SearchingStateProps) {
  if (mode === 'manual') {
    return <ManualSearchingLabel size={size} />
  }
  if (size === 'xs') {
    return <XsAutoSearching theme={theme} />
  }
  return <ChasingLightsSearching size={size} theme={theme} />
}

function ManualSearchingLabel({ size }: { size: ResolvedSize }) {
  return (
    <Button variant="outline" disabled className={cn('w-full', HEIGHT_CLASS[size])}>
      Searching...
    </Button>
  )
}

function XsAutoSearching({ theme }: { theme: MediaTheme }) {
  const shimmerClass = theme === 'movie' ? 'shimmer-text-movie' : 'shimmer-text-tv'
  return (
    <Button variant="ghost" disabled className="h-6 w-full text-xs disabled:opacity-100">
      <span className={shimmerClass} data-text="Searching...">
        Searching...
      </span>
    </Button>
  )
}

function ChasingLightsSearching({ size, theme }: { size: ResolvedSize; theme: MediaTheme }) {
  const chasingClass = theme === 'movie' ? 'chasing-lights-movie' : 'chasing-lights-tv'
  const btnClass = size === 'sm' ? 'relative z-[2] h-8 w-full text-xs' : 'relative z-[2] w-full'
  return (
    <div className={cn(chasingClass, 'w-full')}>
      <div className="bg-card absolute inset-0 z-[1] rounded-md" />
      <Button variant="outline" disabled className={btnClass}>
        Searching...
      </Button>
    </div>
  )
}
