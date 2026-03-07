import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import type { NotificationEventGroup } from '@/types/notification'

type EventTriggersProps = {
  groups: NotificationEventGroup[]
  toggles: Record<string, boolean>
  onToggleChange: (eventId: string, enabled: boolean) => void
}

export function EventTriggers({ groups, toggles, onToggleChange }: EventTriggersProps) {
  if (groups.length === 0) {return null}

  return (
    <div className="space-y-4">
      <Label>Event Triggers</Label>
      {groups.map((group) => (
        <div key={group.id} className="space-y-2 rounded-lg border p-3">
          <p className="text-sm font-medium text-muted-foreground">{group.label}</p>
          {group.events.map((event) => (
            <div key={event.id} className="flex items-center justify-between">
              <Label htmlFor={event.id} className="cursor-pointer font-normal">
                {event.label}
              </Label>
              <Switch
                id={event.id}
                checked={toggles[event.id] ?? false}
                onCheckedChange={(checked) => onToggleChange(event.id, checked)}
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
