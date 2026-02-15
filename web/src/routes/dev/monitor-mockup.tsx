import { Eye, EyeOff } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'
import { themeColor } from './controls-utils'

type MonitorMockupProps = {
  theme: MediaTheme
  size: ControlSize
  monitored: boolean
}

export function MonitorMockup({ theme, size, monitored }: MonitorMockupProps) {
  const activeColor = themeColor(theme)

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled>
        <span className="inline-grid [&>*]:col-start-1 [&>*]:row-start-1">
          <span className={cn('flex items-center', !monitored && 'invisible')}>
            <Eye className={cn('mr-2 size-4', activeColor)} />
            Monitored
          </span>
          <span className={cn('flex items-center', monitored && 'invisible')}>
            <EyeOff className="mr-2 size-4" />
            Unmonitored
          </span>
        </span>
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Button variant="outline" size="icon-sm" disabled>
        {monitored ? <Eye className={cn('size-4', activeColor)} /> : <EyeOff className="size-4" />}
      </Button>
    )
  }

  return (
    <Button variant="ghost" size="icon-sm" disabled>
      {monitored ? (
        <Eye className={cn('size-3.5', activeColor)} />
      ) : (
        <EyeOff className="size-3.5" />
      )}
    </Button>
  )
}
