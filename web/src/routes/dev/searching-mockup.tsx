import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'

type SearchingMockupProps = {
  theme: MediaTheme
  size: ControlSize
  mode: 'manual' | 'auto'
  fullWidth?: boolean
}

export function SearchingMockup({ theme, size, mode, fullWidth }: SearchingMockupProps) {
  if (mode === 'manual') {return <ManualSearching size={size} fullWidth={fullWidth} />}
  if (size === 'xs') {return <AutoSearchingXs theme={theme} fullWidth={fullWidth} />}
  return <AutoSearchingWithLights theme={theme} size={size} fullWidth={fullWidth} />
}

const SIZE_CLASS: Record<ControlSize, string> = {
  xs: 'h-6 text-xs',
  sm: 'h-8 text-xs',
  lg: '',
}

function ManualSearching({ size, fullWidth }: Pick<SearchingMockupProps, 'size' | 'fullWidth'>) {
  return (
    <Button
      variant="outline"
      disabled
      className={cn(fullWidth && 'w-full', SIZE_CLASS[size])}
    >
      Searching...
    </Button>
  )
}

function AutoSearchingXs({ theme, fullWidth }: Pick<SearchingMockupProps, 'theme' | 'fullWidth'>) {
  const shimmerClass = theme === 'movie' ? 'shimmer-text-movie' : 'shimmer-text-tv'
  return (
    <Button
      variant="ghost"
      disabled
      className={cn('h-6 text-xs disabled:opacity-100', fullWidth && 'w-full')}
    >
      <span className={shimmerClass} data-text="Searching...">
        Searching...
      </span>
    </Button>
  )
}

function AutoSearchingWithLights({
  theme,
  size,
  fullWidth,
}: Pick<SearchingMockupProps, 'theme' | 'size' | 'fullWidth'>) {
  const chasingClass = theme === 'movie' ? 'chasing-lights-movie' : 'chasing-lights-tv'
  const btnSize = size === 'sm' ? 'h-8 text-xs' : ''

  return (
    <div className={cn(chasingClass, fullWidth && 'w-full')}>
      <div className="bg-card absolute inset-0 z-[1] rounded-md" />
      <Button
        variant="outline"
        disabled
        className={cn('relative z-[2]', btnSize, fullWidth && 'w-full')}
      >
        Searching...
      </Button>
    </div>
  )
}
