import { AlertCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'
import { themeColor } from './controls-utils'

type ErrorMockupProps = {
  theme: MediaTheme
  size: ControlSize
  message: string
  fullWidth?: boolean
}

export function ErrorMockup({ theme, size, message, fullWidth }: ErrorMockupProps) {
  const colorClass = themeColor(theme)

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled className={cn(fullWidth && 'w-full')}>
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
              disabled
              className={cn(fullWidth && 'w-full')}
            />
          }
        >
          <AlertCircle className={cn('size-4', colorClass)} />
        </TooltipTrigger>
        <TooltipContent>{message}</TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger>
        <div className={cn('p-1', fullWidth && 'flex w-full items-center justify-center')}>
          <AlertCircle className={cn('size-3.5', colorClass)} />
        </div>
      </TooltipTrigger>
      <TooltipContent>{message}</TooltipContent>
    </Tooltip>
  )
}
