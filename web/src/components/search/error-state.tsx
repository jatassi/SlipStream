import { AlertCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { ControlVariant, MediaTheme } from './media-search-monitor-types'

type ErrorStateProps = {
  variant: ControlVariant
  theme: MediaTheme
  message: string
  onClick: () => void
}

export function ErrorState({ variant, theme, message, onClick }: ErrorStateProps) {
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'
  const label = `${message} — dismiss`

  if (variant === 'pill') {
    return (
      <Button
        variant="outline"
        aria-label={label}
        className="text-footnote h-tap w-full cursor-pointer rounded-full"
        onClick={onClick}
      >
        <AlertCircle className={cn('mr-2 size-4', colorClass)} />
        {message}
      </Button>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={label}
            className="w-full cursor-pointer"
            onClick={onClick}
          />
        }
      >
        <AlertCircle className={cn('size-4', colorClass)} />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}
