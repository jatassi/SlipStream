import { ChevronDown, ChevronUp } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { NotifierType } from '@/types'

import { EventTriggers } from './event-triggers'
import { PlexSections } from './plex-sections'
import { ProviderFields } from './provider-fields'
import type { NotificationDialogState } from './use-notification-dialog'

export function NotificationFormBody({ state: s }: { state: NotificationDialogState }) {
  const basicFields = s.currentSchema?.fields.filter((f) => !f.advanced) ?? []
  const advancedFields = s.currentSchema?.fields.filter((f) => f.advanced) ?? []
  const shared = {
    settings: s.formData.settings, isPlex: s.isPlex, hasPlexToken: s.hasPlexToken,
    isPlexConnecting: s.isPlexConnecting, isLoadingServers: s.isLoadingServers,
    plexServers: s.plexServers, onSettingChange: s.handleSettingChange,
    onPlexConnect: s.handlePlexOAuth, onPlexDisconnect: s.handlePlexDisconnect,
  }

  return (
    <div className="space-y-6 py-4">
      <TypeSelector formType={s.formData.type} schemas={s.schemas} isEditing={s.isEditing} onTypeChange={s.handleTypeChange} />
      <NameInput value={s.formData.name} onChange={(v) => s.handleFormDataChange('name', v)} />
      <ProviderFields fields={basicFields} {...shared} />
      <PlexSections
        isPlex={s.isPlex} hasPlexToken={s.hasPlexToken} serverId={s.formData.settings.serverId}
        sectionIds={(s.formData.settings.sectionIds ?? []) as number[]}
        isLoadingSections={s.isLoadingSections} plexSections={s.plexSections} onSettingChange={s.handleSettingChange}
      />
      {s.hasAdvancedFields ? <AdvancedToggle showAdvanced={s.showAdvanced} onToggle={s.toggleAdvanced} /> : null}
      {s.showAdvanced ? <ProviderFields fields={advancedFields} {...shared} /> : null}
      <EventTriggers triggers={s.triggers} formValues={s.formData as unknown as Record<string, unknown>} onTriggerChange={s.handleFormDataChange} />
      <EnabledToggle enabled={s.formData.enabled ?? true} onChange={(c) => s.handleFormDataChange('enabled', c)} />
    </div>
  )
}

function TypeSelector({ formType, schemas, isEditing, onTypeChange }: {
  formType: string
  schemas: { type: string; name: string }[] | undefined
  isEditing: boolean
  onTypeChange: (type: NotifierType) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="type">Type</Label>
      <Select value={formType} onValueChange={(v) => v && onTypeChange(v as NotifierType)} disabled={isEditing}>
        <SelectTrigger>{schemas?.find((s) => s.type === formType)?.name ?? formType}</SelectTrigger>
        <SelectContent>
          {schemas?.map((schema) => (
            <SelectItem key={schema.type} value={schema.type}>{schema.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function NameInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="name">Name</Label>
      <Input id="name" placeholder="My Notification" value={value} onChange={(e) => onChange(e.target.value)} />
    </div>
  )
}

function EnabledToggle({ enabled, onChange }: { enabled: boolean; onChange: (checked: boolean) => void }) {
  return (
    <div className="flex items-center justify-between">
      <Label htmlFor="enabled">Enabled</Label>
      <Switch id="enabled" checked={enabled} onCheckedChange={onChange} />
    </div>
  )
}

function AdvancedToggle({ showAdvanced, onToggle }: { showAdvanced: boolean; onToggle: () => void }) {
  const Icon = showAdvanced ? ChevronUp : ChevronDown
  const label = showAdvanced ? 'Hide Advanced Settings' : 'Show Advanced Settings'
  return (
    <Button type="button" variant="ghost" size="sm" className="w-full" onClick={onToggle}>
      <Icon className="mr-2 size-4" />
      {label}
    </Button>
  )
}
