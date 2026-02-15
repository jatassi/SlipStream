import { AlertCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { MediaTheme, ResolvedSize } from './media-search-monitor-types'

type ErrorStateProps = {
  size: ResolvedSize
  theme: MediaTheme
  message: string
  onClick: () => void
}

export function ErrorState({ size, theme, message, onClick }: ErrorStateProps) {
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button variant="outline" className="w-full cursor-pointer" onClick={onClick}>
        <AlertCircle className={cn('mr-2 size-4', colorClass)} />
        {message}
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
              className="w-full cursor-pointer"
              onClick={onClick}
            />
          }
        >
          <AlertCircle className={cn('size-4', colorClass)} />
        </TooltipTrigger>
        <TooltipContent>{message} — click to dismiss</TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger>
        <button onClick={onClick} className="flex w-full items-center justify-center p-1">
          <AlertCircle className={cn('size-3.5', colorClass)} />
        </button>
      </TooltipTrigger>
      <TooltipContent>{message}</TooltipContent>
    </Tooltip>
  )
}
