import { ChevronDown, ChevronUp } from 'lucide-react'

import { ControlStack, InputRow, SelectRow, SwitchRow } from '@/components/settings/control-row'
import { Button } from '@/components/ui/button'
import { useNotificationEventCatalog } from '@/hooks'
import type { CreateNotificationInput, NotificationEventGroup, NotifierType } from '@/types'

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
    <div className="space-y-4 py-2">
      <ControlStack>
        <TypeSelector formType={s.formData.type} schemas={s.schemas} isEditing={s.isEditing} onTypeChange={s.handleTypeChange} />
        <NameInput value={s.formData.name} onChange={(v) => s.handleFormDataChange('name', v)} />
      </ControlStack>
      <ProviderFields fields={basicFields} {...shared} />
      <PlexSections
        isPlex={s.isPlex} hasPlexToken={s.hasPlexToken} serverId={s.formData.settings.serverId}
        sectionIds={(s.formData.settings.sectionIds ?? []) as number[]}
        isLoadingSections={s.isLoadingSections} plexSections={s.plexSections} onSettingChange={s.handleSettingChange}
      />
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {s.hasAdvancedFields && <AdvancedToggle showAdvanced={s.showAdvanced} onToggle={s.toggleAdvanced} />}
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {s.showAdvanced && <ProviderFields fields={advancedFields} {...shared} />}
      <EventTriggersSection eventGroups={s.eventGroups} formData={s.formData} setFormData={s.setFormData} />
      <EnabledToggle enabled={s.formData.enabled ?? true} onChange={(c) => s.handleFormDataChange('enabled', c)} />
    </div>
  )
}

function EventTriggersSection({ eventGroups: overrideGroups, formData, setFormData }: {
  eventGroups?: NotificationEventGroup[]
  formData: CreateNotificationInput
  setFormData: React.Dispatch<React.SetStateAction<CreateNotificationInput>>
}) {
  const { data: fetchedCatalog } = useNotificationEventCatalog()
  const groups = overrideGroups ?? fetchedCatalog ?? []

  return (
    <EventTriggers
      groups={groups}
      toggles={formData.eventToggles ?? {}}
      onToggleChange={(eventId, enabled) => {
        setFormData((prev) => ({
          ...prev,
          eventToggles: { ...prev.eventToggles, [eventId]: enabled },
        }))
      }}
    />
  )
}

function TypeSelector({ formType, schemas, isEditing, onTypeChange }: {
  formType: string
  schemas: { type: string; name: string }[] | undefined
  isEditing: boolean
  onTypeChange: (type: NotifierType) => void
}) {
  const options = (schemas ?? []).map((schema) => ({ value: schema.type, label: schema.name }))
  return (
    <SelectRow
      label="Type"
      value={formType}
      onChange={(value) => onTypeChange(value as NotifierType)}
      options={options}
      disabled={isEditing}
    />
  )
}

function NameInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <InputRow
      stacked
      label="Name"
      aria-label="Name"
      placeholder="My Notification"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  )
}

function EnabledToggle({ enabled, onChange }: { enabled: boolean; onChange: (checked: boolean) => void }) {
  return (
    <ControlStack>
      <SwitchRow label="Enabled" checked={enabled} onCheckedChange={onChange} />
    </ControlStack>
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
