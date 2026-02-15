import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { QualityItem, UpgradeStrategy } from '@/types'

import { UPGRADE_STRATEGY_OPTIONS } from './constants'

type UpgradeSettingsProps = {
  upgradesEnabled: boolean
  upgradeStrategy: UpgradeStrategy
  cutoffOverridesStrategy: boolean
  allowAutoApprove: boolean
  cutoff: number
  cutoffOptions: QualityItem[]
  onFieldChange: (key: string, value: unknown) => void
}

export function UpgradeSettings({
  upgradesEnabled,
  upgradeStrategy,
  cutoffOverridesStrategy,
  allowAutoApprove,
  cutoff,
  cutoffOptions,
  onFieldChange,
}: UpgradeSettingsProps) {
  const cutoffOverrideDisabled = !upgradesEnabled || upgradeStrategy === 'aggressive'

  return (
    <>
      <UpgradesToggle
        enabled={upgradesEnabled}
        onChange={(checked) => onFieldChange('upgradesEnabled', checked)}
      />

      <StrategySelect
        value={upgradeStrategy}
        disabled={!upgradesEnabled}
        onChange={(v) => onFieldChange('upgradeStrategy', v)}
      />

      <CutoffOverrideToggle
        checked={cutoffOverridesStrategy}
        disabled={cutoffOverrideDisabled}
        onChange={(checked) => onFieldChange('cutoffOverridesStrategy', checked)}
      />

      <AutoApproveToggle
        checked={allowAutoApprove}
        onChange={(checked) => onFieldChange('allowAutoApprove', checked)}
      />

      <CutoffSelect
        value={cutoff}
        options={cutoffOptions}
        disabled={!upgradesEnabled}
        onChange={(v) => onFieldChange('cutoff', Number.parseInt(v))}
      />
    </>
  )
}

function UpgradesToggle({
  enabled,
  onChange,
}: {
  enabled: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border p-3">
      <div className="space-y-0.5">
        <Label>Upgrades</Label>
        <p className="text-muted-foreground text-xs">Search for better quality when file exists</p>
      </div>
      <Switch checked={enabled} onCheckedChange={onChange} />
    </div>
  )
}

function StrategySelect({
  value,
  disabled,
  onChange,
}: {
  value: UpgradeStrategy
  disabled: boolean
  onChange: (v: UpgradeStrategy) => void
}) {
  return (
    <div className="space-y-2">
      <Label className={disabled ? 'text-muted-foreground' : ''}>Upgrade Strategy</Label>
      <Select
        value={value}
        onValueChange={(v) => v && onChange(v as UpgradeStrategy)}
        disabled={disabled}
      >
        <SelectTrigger className={`w-full ${disabled ? 'opacity-50' : ''}`}>
          {UPGRADE_STRATEGY_OPTIONS.find((o) => o.value === value)?.label ?? 'Select strategy'}
        </SelectTrigger>
        <SelectContent>
          {UPGRADE_STRATEGY_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              <div>
                <div>{option.label}</div>
                <div className="text-muted-foreground text-xs">{option.description}</div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function CutoffOverrideToggle({
  checked,
  disabled,
  onChange,
}: {
  checked: boolean
  disabled: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div
      className={`flex items-center justify-between rounded-lg border p-3 ${disabled ? 'opacity-50' : ''}`}
    >
      <div className="space-y-0.5">
        <Label className={disabled ? 'text-muted-foreground' : ''}>Cutoff Overrides Strategy</Label>
        <p className="text-muted-foreground text-xs">
          Always grab cutoff quality even if strategy would block it
        </p>
      </div>
      <Switch checked={checked} disabled={disabled} onCheckedChange={onChange} />
    </div>
  )
}

function AutoApproveToggle({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border p-3">
      <div className="space-y-0.5">
        <Label>Allow Auto-Approve</Label>
        <p className="text-muted-foreground text-xs">
          Requests using this profile can be auto-approved
        </p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

function CutoffSelect({
  value,
  options,
  disabled,
  onChange,
}: {
  value: number
  options: QualityItem[]
  disabled: boolean
  onChange: (v: string) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="cutoff" className={disabled ? 'text-muted-foreground' : ''}>
        Cutoff
      </Label>
      <Select
        value={value.toString()}
        onValueChange={(v) => v && onChange(v)}
        disabled={disabled}
      >
        <SelectTrigger className={`w-full ${disabled ? 'opacity-50' : ''}`}>
          {options.find((i) => i.quality.id === value)?.quality.name ?? 'Select cutoff'}
        </SelectTrigger>
        <SelectContent>
          {options.map((item) => (
            <SelectItem key={item.quality.id} value={item.quality.id.toString()}>
              {item.quality.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p
        className={`text-xs ${disabled ? 'text-muted-foreground/50' : 'text-muted-foreground'}`}
      >
        Stop upgrading once this quality is reached
      </p>
    </div>
  )
}
