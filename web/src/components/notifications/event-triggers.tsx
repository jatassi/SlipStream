import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import type { EventTrigger } from './notification-dialog-types'

type EventTriggersProps = {
  triggers: EventTrigger[]
  formValues: Record<string, unknown>
  onTriggerChange: (key: string, value: unknown) => void
}

export function EventTriggers({ triggers, formValues, onTriggerChange }: EventTriggersProps) {
  if (triggers.length === 0) {return null}

  return (
    <div className="space-y-3">
      <Label>Event Triggers</Label>
      <div className="space-y-2 rounded-lg border p-3">
        {triggers.map(({ key, label }) => (
          <div key={key} className="flex items-center justify-between">
            <Label htmlFor={key} className="cursor-pointer font-normal">
              {label}
            </Label>
            <Switch
              id={key}
              checked={Boolean(formValues[key])}
              onCheckedChange={(checked) => onTriggerChange(key, checked)}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
