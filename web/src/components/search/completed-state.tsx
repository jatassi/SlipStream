import { Check } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { MediaTheme, ResolvedSize } from './media-search-monitor-types'

type CompletedStateProps = {
  size: ResolvedSize
  theme: MediaTheme
  onClick: () => void
}

export function CompletedState({ size, theme, onClick }: CompletedStateProps) {
  const flashClass =
    theme === 'movie'
      ? 'animate-[download-complete-flash-movie_800ms_ease-out]'
      : 'animate-[download-complete-flash-tv_800ms_ease-out]'

  const checkColor = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button
        variant="outline"
        className={cn(flashClass, 'w-full cursor-pointer')}
        onClick={onClick}
      >
        <Check className={cn('mr-2 size-4', checkColor)} />
        Downloaded
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="outline"
              size="icon-sm"
              className={cn(flashClass, 'w-full cursor-pointer')}
              onClick={onClick}
            />
          }
        >
          <Check className={cn('size-4', checkColor)} />
        </TooltipTrigger>
        <TooltipContent>Downloaded — click to dismiss</TooltipContent>
      </Tooltip>
    )
  }

  return (
    <button onClick={onClick} className="flex w-full items-center justify-center p-1">
      <Check className={cn('size-3.5', checkColor)} />
    </button>
  )
}
