import { Check } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlVariant, MediaTheme } from './media-search-monitor-types'

type CompletedStateProps = {
  variant: ControlVariant
  theme: MediaTheme
  onClick: () => void
}

export function CompletedState({ variant, theme, onClick }: CompletedStateProps) {
  const flashClass =
    theme === 'movie'
      ? 'animate-[download-complete-flash-movie_800ms_ease-out]'
      : 'animate-[download-complete-flash-tv_800ms_ease-out]'
  const checkColor = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (variant === 'pill') {
    return (
      <Button
        variant="outline"
        aria-label="Downloaded — dismiss"
        className={cn(flashClass, 'text-footnote h-tap w-full cursor-pointer rounded-full')}
        onClick={onClick}
      >
        <Check className={cn('mr-2 size-4', checkColor)} />
        Downloaded
      </Button>
    )
  }

  return (
    <Button
      variant="outline"
      size="icon-sm"
      aria-label="Downloaded — dismiss"
      className={cn(flashClass, 'w-full cursor-pointer')}
      onClick={onClick}
    >
      <Check className={cn('size-4', checkColor)} />
    </Button>
  )
}
