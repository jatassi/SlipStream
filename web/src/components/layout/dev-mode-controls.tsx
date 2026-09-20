import type { ReactNode } from 'react'

import { Hammer, LayoutTemplate } from 'lucide-react'

import { Group, IconTile } from '@/components/grouped-list'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

type DevModeControlsProps = {
  devModeEnabled: boolean
  devModeSwitching: boolean
  onToggle: (pressed: boolean) => void
  globalLoading: boolean
  onGlobalLoadingChange: (checked: boolean) => void
}

function SwitchRow({
  leading,
  title,
  children,
}: {
  leading: ReactNode
  title: string
  children: ReactNode
}) {
  return (
    <label className="min-h-tap flex w-full cursor-pointer items-center gap-3 px-4 py-2.5">
      <span className="flex shrink-0 items-center">{leading}</span>
      <span className="text-body min-w-0 flex-1 truncate font-medium">{title}</span>
      {children}
    </label>
  )
}

export function DevModeControls({
  devModeEnabled,
  devModeSwitching,
  onToggle,
  globalLoading,
  onGlobalLoadingChange,
}: DevModeControlsProps) {
  return (
    <Group header="Developer Tools">
      <SwitchRow
        leading={
          <IconTile className="bg-amber-500">
            <Hammer />
          </IconTile>
        }
        title="Developer mode"
      >
        <Switch
          checked={devModeEnabled}
          onCheckedChange={onToggle}
          disabled={devModeSwitching}
          size="sm"
          aria-label="Developer mode"
          className={cn(devModeEnabled && 'data-checked:bg-amber-500')}
        />
      </SwitchRow>
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {devModeEnabled && <SwitchRow
          leading={
            <IconTile className="bg-zinc-600">
              <LayoutTemplate />
            </IconTile>
          }
          title="Force Loading"
        >
          <Switch
            checked={globalLoading}
            onCheckedChange={onGlobalLoadingChange}
            size="sm"
            aria-label="Force Loading"
          />
        </SwitchRow>}
    </Group>
  )
}
