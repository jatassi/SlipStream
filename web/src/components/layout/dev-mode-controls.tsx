import { LayoutTemplate } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

type DevModeControlsProps = {
  devModeEnabled: boolean
  devModeSwitching: boolean
  onToggle: (pressed: boolean) => void
  globalLoading: boolean
  onGlobalLoadingChange: (checked: boolean) => void
}

export function DevModeControls({
  devModeEnabled,
  devModeSwitching,
  onToggle,
  globalLoading,
  onGlobalLoadingChange,
}: DevModeControlsProps) {
  return (
    <div className="flex flex-col gap-1">
      <Label className="flex min-h-tap items-center gap-3 px-1">
        <span className="flex-1 text-body font-medium">Developer mode</span>
        <Switch
          checked={devModeEnabled}
          onCheckedChange={onToggle}
          disabled={devModeSwitching}
          size="sm"
          aria-label="Developer mode"
          className={cn(devModeEnabled && 'data-checked:bg-amber-500')}
        />
      </Label>
      {devModeEnabled ? (
        <Label className="flex min-h-tap items-center gap-3 px-1">
          <LayoutTemplate className="text-muted-foreground size-4 shrink-0" />
          <span className="flex-1 text-body font-medium">Force Loading</span>
          <Switch
            id="force-loading-toggle"
            checked={globalLoading}
            onCheckedChange={onGlobalLoadingChange}
            size="sm"
            aria-label="Force Loading"
          />
        </Label>
      ) : null}
    </div>
  )
}
