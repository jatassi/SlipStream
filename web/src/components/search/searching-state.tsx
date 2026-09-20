import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlVariant, MediaTheme } from './media-search-monitor-types'

type SearchingStateProps = {
  variant: ControlVariant
  theme: MediaTheme
  mode: 'manual' | 'auto'
}

const SHELL: Record<ControlVariant, string> = {
  pill: 'h-tap w-full rounded-full text-footnote',
  row: 'h-8 w-full text-xs',
}

export function SearchingState({ variant, theme, mode }: SearchingStateProps) {
  if (mode === 'manual') {
    return (
      <Button variant="outline" disabled className={SHELL[variant]}>
        Searching...
      </Button>
    )
  }
  const chasingClass = theme === 'movie' ? 'chasing-lights-movie' : 'chasing-lights-tv'
  return (
    <div className={cn(chasingClass, 'w-full', variant === 'pill' && 'rounded-full')}>
      <div className={cn('bg-card absolute inset-0 z-[1]', variant === 'pill' ? 'rounded-full' : 'rounded-md')} />
      <Button variant="outline" disabled className={cn('relative z-[2]', SHELL[variant])}>
        Searching...
      </Button>
    </div>
  )
}
