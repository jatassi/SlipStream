import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Info } from 'lucide-react'
import type { DefinitionSetting } from '@/types'

interface DynamicSettingsFormProps {
  settings: DefinitionSetting[]
  values: Record<string, string>
  onChange: (values: Record<string, string>) => void
  disabled?: boolean
}

export function DynamicSettingsForm({
  settings,
  values,
  onChange,
  disabled,
}: DynamicSettingsFormProps) {
  const handleChange = (name: string, value: string) => {
    onChange({ ...values, [name]: value })
  }

  if (settings.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        No configuration required for this indexer.
      </p>
    )
  }

  return (
    <div className="space-y-4">
      {settings.map((setting) => (
        <SettingField
          key={setting.name}
          setting={setting}
          value={values[setting.name] ?? setting.default ?? ''}
          onChange={(value) => handleChange(setting.name, value)}
          disabled={disabled}
        />
      ))}
    </div>
  )
}

interface SettingFieldProps {
  setting: DefinitionSetting
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}

function SettingField({ setting, value, onChange, disabled }: SettingFieldProps) {
  switch (setting.type) {
    case 'text':
      return (
        <div className="space-y-2">
          <Label htmlFor={setting.name}>{setting.label}</Label>
          <Input
            id={setting.name}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
          />
        </div>
      )

    case 'password':
      return (
        <div className="space-y-2">
          <Label htmlFor={setting.name}>{setting.label}</Label>
          <Input
            id={setting.name}
            type="password"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            autoComplete="off"
          />
        </div>
      )

    case 'checkbox':
      return (
        <div className="flex items-center space-x-2">
          <Checkbox
            id={setting.name}
            checked={value === 'true'}
            onCheckedChange={(checked) => onChange(checked ? 'true' : 'false')}
            disabled={disabled}
          />
          <Label htmlFor={setting.name} className="font-normal cursor-pointer">
            {setting.label}
          </Label>
        </div>
      )

    case 'select':
      return (
        <div className="space-y-2">
          <Label htmlFor={setting.name}>{setting.label}</Label>
          <Select value={value} onValueChange={(v) => v && onChange(v)} disabled={disabled}>
            <SelectTrigger id={setting.name}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {setting.options &&
                Object.entries(setting.options).map(([optValue, optLabel]) => (
                  <SelectItem key={optValue} value={optValue}>
                    {optLabel}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>
      )

    case 'info':
      return (
        <Alert>
          <Info className="size-4" />
          <AlertDescription>{setting.label}</AlertDescription>
        </Alert>
      )

    default:
      return (
        <div className="space-y-2">
          <Label htmlFor={setting.name}>{setting.label}</Label>
          <Input
            id={setting.name}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
          />
        </div>
      )
  }
}
