import { HeaderDevModePopover } from '@/components/layout/header-dev-mode-popover'
import { HeaderDevModeTrigger } from '@/components/layout/header-dev-mode-trigger'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

type HeaderDevModeProps = {
  devModeEnabled: boolean
  devModeSwitching: boolean
  onToggle: (pressed: boolean) => void
  globalLoading: boolean
  onGlobalLoadingChange: (checked: boolean) => void
}

export function HeaderDevMode({
  devModeEnabled,
  devModeSwitching,
  onToggle,
  globalLoading,
  onGlobalLoadingChange,
}: HeaderDevModeProps) {
  return (
    <div className="flex items-center gap-1.5">
      {devModeEnabled ? (
        <HeaderDevModePopover
          devModeSwitching={devModeSwitching}
          globalLoading={globalLoading}
          onGlobalLoadingChange={onGlobalLoadingChange}
        />
      ) : (
        <HeaderDevModeTrigger />
      )}
      <Switch
        checked={devModeEnabled}
        onCheckedChange={onToggle}
        disabled={devModeSwitching}
        size="sm"
        className={cn(devModeEnabled && 'data-checked:bg-amber-500')}
      />
    </div>
  )
}
