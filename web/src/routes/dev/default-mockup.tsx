import { Eye, EyeOff, UserSearch, Zap } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'
import { gapForSize, themeColor } from './controls-utils'

type DefaultMockupProps = {
  theme: MediaTheme
  size: ControlSize
  monitored: boolean
  onManualSearch?: () => void
  onAutoSearch?: () => void
  onMonitoredChange?: (v: boolean) => void
}

export function DefaultMockup(props: DefaultMockupProps) {
  const { size } = props
  if (size === 'lg') {return <DefaultLg {...props} />}
  if (size === 'sm') {return <DefaultSm {...props} />}
  return <DefaultXs {...props} />
}

function DefaultLg({ theme, monitored, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultMockupProps) {
  const interactive = !!onManualSearch
  return (
    <div className={cn('flex items-center', gapForSize('lg'))}>
      <Button variant="outline" size="default" disabled={!interactive} onClick={onManualSearch}>
        <UserSearch className="mr-2 size-4" />
        Search
      </Button>
      <Button variant="outline" size="default" disabled={!interactive} onClick={onAutoSearch}>
        <Zap className="mr-2 size-4" />
        Auto Search
      </Button>
      <Button
        variant="outline"
        size="default"
        disabled={!interactive}
        onClick={() => onMonitoredChange?.(!monitored)}
      >
        <span className="inline-grid [&>*]:col-start-1 [&>*]:row-start-1">
          <span className={cn('flex items-center', !monitored && 'invisible')}>
            <Eye className={cn('mr-2 size-4', themeColor(theme))} />
            Monitored
          </span>
          <span className={cn('flex items-center', monitored && 'invisible')}>
            <EyeOff className="mr-2 size-4" />
            Unmonitored
          </span>
        </span>
      </Button>
    </div>
  )
}

function DefaultSm({ theme, monitored, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultMockupProps) {
  const interactive = !!onManualSearch
  return (
    <div className={cn('flex items-center', gapForSize('sm'))}>
      <Button variant="outline" size="icon-sm" disabled={!interactive} onClick={onManualSearch}>
        <UserSearch className="size-4" />
      </Button>
      <Button variant="outline" size="icon-sm" disabled={!interactive} onClick={onAutoSearch}>
        <Zap className="size-4" />
      </Button>
      <Button
        variant="outline"
        size="icon-sm"
        disabled={!interactive}
        onClick={() => onMonitoredChange?.(!monitored)}
      >
        {monitored ? (
          <Eye className={cn('size-4', themeColor(theme))} />
        ) : (
          <EyeOff className="size-4" />
        )}
      </Button>
    </div>
  )
}

function DefaultXs({ theme, monitored, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultMockupProps) {
  const interactive = !!onManualSearch
  return (
    <div className={cn('flex items-center', gapForSize('xs'))}>
      <Button variant="ghost" size="icon-sm" disabled={!interactive} onClick={onManualSearch}>
        <UserSearch className="size-3.5" />
      </Button>
      <Button variant="ghost" size="icon-sm" disabled={!interactive} onClick={onAutoSearch}>
        <Zap className="size-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        disabled={!interactive}
        onClick={() => onMonitoredChange?.(!monitored)}
      >
        {monitored ? (
          <Eye className={cn('size-3.5', themeColor(theme))} />
        ) : (
          <EyeOff className="size-3.5" />
        )}
      </Button>
    </div>
  )
}
