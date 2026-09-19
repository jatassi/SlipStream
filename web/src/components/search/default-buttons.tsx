import { Eye, EyeOff, UserSearch, Zap } from 'lucide-react'

import { PillAction } from '@/components/media/pill-action'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { ControlVariant, MediaTheme } from './media-search-monitor-types'

type DefaultButtonsProps = {
  variant: ControlVariant
  theme: MediaTheme
  monitored: boolean
  monitorDisabled?: boolean
  onManualSearch: () => void
  onAutoSearch: () => void
  onMonitoredChange: (monitored: boolean) => void
}

export function DefaultButtons(props: DefaultButtonsProps) {
  if (props.variant === 'pill') {
    return <PillButtons {...props} />
  }
  return <RowButtons {...props} />
}

function MonitorIcon({ theme, monitored, className }: { theme: MediaTheme; monitored: boolean; className: string }) {
  if (monitored) {
    return <Eye className={cn(className, theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
  }
  return <EyeOff className={className} />
}

function PillButtons({
  theme,
  monitored,
  monitorDisabled,
  onManualSearch,
  onAutoSearch,
  onMonitoredChange,
}: DefaultButtonsProps) {
  return (
    <>
      <PillAction icon={UserSearch} label="Search" tint={theme} onClick={onManualSearch} />
      <PillAction icon={Zap} label="Auto Search" tint={theme} onClick={onAutoSearch} />
      <PillAction
        icon={monitored ? Eye : EyeOff}
        label={monitored ? 'Monitored' : 'Unmonitored'}
        tint={theme}
        active={monitored}
        pressed={monitored}
        disabled={monitorDisabled}
        onClick={() => onMonitoredChange(!monitored)}
      />
    </>
  )
}

function RowButtons({
  theme,
  monitored,
  monitorDisabled,
  onManualSearch,
  onAutoSearch,
  onMonitoredChange,
}: DefaultButtonsProps) {
  return (
    <>
      <Tooltip>
        <TooltipTrigger render={<Button variant="outline" size="icon-sm" aria-label="Manual Search" onClick={onManualSearch} />}>
          <UserSearch className="size-4" />
        </TooltipTrigger>
        <TooltipContent>Manual Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger render={<Button variant="outline" size="icon-sm" aria-label="Auto Search" onClick={onAutoSearch} />}>
          <Zap className="size-4" />
        </TooltipTrigger>
        <TooltipContent>Auto Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="outline"
              size="icon-sm"
              aria-label={monitored ? 'Monitored' : 'Unmonitored'}
              aria-pressed={monitored}
              onClick={() => onMonitoredChange(!monitored)}
              disabled={monitorDisabled}
            />
          }
        >
          <MonitorIcon theme={theme} monitored={monitored} className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{monitored ? 'Monitored' : 'Unmonitored'}</TooltipContent>
      </Tooltip>
    </>
  )
}
