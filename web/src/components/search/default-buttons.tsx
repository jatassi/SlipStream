import { Eye, EyeOff, UserSearch, Zap } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { MediaTheme, ResolvedSize } from './media-search-monitor-types'

type DefaultButtonsProps = {
  size: ResolvedSize
  theme: MediaTheme
  monitored: boolean
  monitorDisabled?: boolean
  onManualSearch: () => void
  onAutoSearch: () => void
  onMonitoredChange: (monitored: boolean) => void
}

export function DefaultButtons(props: DefaultButtonsProps) {
  const { size } = props
  if (size === 'lg') {
    return <LargeButtons {...props} />
  }
  if (size === 'sm') {
    return <SmallButtons {...props} />
  }
  return <ExtraSmallButtons {...props} />
}

function MonitorIcon({ theme, monitored, className }: { theme: MediaTheme; monitored: boolean; className: string }) {
  if (monitored) {
    return <Eye className={cn(className, theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
  }
  return <EyeOff className={className} />
}

function LargeButtons({ theme, monitored, monitorDisabled, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultButtonsProps) {
  return (
    <>
      <Button variant="outline" onClick={onManualSearch}>
        <UserSearch className="mr-2 size-4" />
        Search
      </Button>
      <Button variant="outline" onClick={onAutoSearch}>
        <Zap className="mr-2 size-4" />
        Auto Search
      </Button>
      <Button variant="outline" onClick={() => onMonitoredChange(!monitored)} disabled={monitorDisabled}>
        <MonitorIcon theme={theme} monitored={monitored} className="mr-2 size-4" />
        {monitored ? 'Monitored' : 'Unmonitored'}
      </Button>
    </>
  )
}

function SmallButtons({ theme, monitored, monitorDisabled, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultButtonsProps) {
  return (
    <>
      <Tooltip>
        <TooltipTrigger render={<Button variant="outline" size="icon-sm" onClick={onManualSearch} />}>
          <UserSearch className="size-4" />
        </TooltipTrigger>
        <TooltipContent>Manual Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger render={<Button variant="outline" size="icon-sm" onClick={onAutoSearch} />}>
          <Zap className="size-4" />
        </TooltipTrigger>
        <TooltipContent>Auto Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button variant="outline" size="icon-sm" onClick={() => onMonitoredChange(!monitored)} disabled={monitorDisabled} />
          }
        >
          <MonitorIcon theme={theme} monitored={monitored} className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{monitored ? 'Monitored' : 'Unmonitored'}</TooltipContent>
      </Tooltip>
    </>
  )
}

function ExtraSmallButtons({ theme, monitored, monitorDisabled, onManualSearch, onAutoSearch, onMonitoredChange }: DefaultButtonsProps) {
  return (
    <>
      <Tooltip>
        <TooltipTrigger render={<Button variant="ghost" size="icon-sm" onClick={onManualSearch} />}>
          <UserSearch className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>Manual Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger render={<Button variant="ghost" size="icon-sm" onClick={onAutoSearch} />}>
          <Zap className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>Auto Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button variant="ghost" size="icon-sm" onClick={() => onMonitoredChange(!monitored)} disabled={monitorDisabled} />
          }
        >
          <MonitorIcon theme={theme} monitored={monitored} className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>{monitored ? 'Monitored' : 'Unmonitored'}</TooltipContent>
      </Tooltip>
    </>
  )
}
