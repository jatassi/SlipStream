import { Check } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'
import { themeColor } from './controls-utils'

type CompletedMockupProps = {
  theme: MediaTheme
  size: ControlSize
  fullWidth?: boolean
}

export function CompletedMockup({ theme, size, fullWidth }: CompletedMockupProps) {
  const colorClass = themeColor(theme)

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled className={cn(fullWidth && 'w-full')}>
        <Check className={cn('mr-2 size-4', colorClass)} />
        Downloaded
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Button variant="outline" size="icon-sm" disabled className={cn(fullWidth && 'w-full')}>
        <Check className={cn('size-4', colorClass)} />
      </Button>
    )
  }

  return (
    <div className={cn('p-1', fullWidth && 'flex w-full items-center justify-center')}>
      <Check className={cn('size-3.5', colorClass)} />
    </div>
  )
}
