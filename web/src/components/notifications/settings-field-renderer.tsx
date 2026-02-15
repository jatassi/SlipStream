import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { SettingsField } from '@/types'

import type { PlexServer } from './notification-dialog-types'
import { PlexOAuthButton } from './plex-oauth-button'

type SettingsFieldRendererProps = {
  field: SettingsField
  value: unknown
  isPlex: boolean
  hasPlexToken: boolean
  isPlexConnecting: boolean
  isLoadingServers: boolean
  plexServers: PlexServer[]
  onSettingChange: (name: string, value: unknown) => void
  onPlexConnect: () => void
  onPlexDisconnect: () => void
}

const fieldRenderers: Partial<
  Record<string, (props: SettingsFieldRendererProps) => React.ReactNode>
> = {
  text: TextOrUrlField,
  url: TextOrUrlField,
  password: PasswordField,
  number: NumberField,
  bool: BoolField,
  select: SelectField,
  action: ActionField,
}

export function SettingsFieldRenderer(props: SettingsFieldRendererProps) {
  const renderer = fieldRenderers[props.field.type]
  if (!renderer) {
    return null
  }
  return <>{renderer(props)}</>
}

function TextOrUrlField({ field, value, onSettingChange }: SettingsFieldRendererProps) {
  return (
    <Input
      id={field.name}
      type={field.type === 'url' ? 'url' : 'text'}
      placeholder={field.placeholder}
      value={typeof value === 'string' ? value : ''}
      onChange={(e) => onSettingChange(field.name, e.target.value)}
    />
  )
}

function PasswordField({ field, value, onSettingChange }: SettingsFieldRendererProps) {
  return (
    <Input
      id={field.name}
      type="password"
      placeholder={field.placeholder}
      value={typeof value === 'string' ? value : ''}
      onChange={(e) => onSettingChange(field.name, e.target.value)}
    />
  )
}

function NumberField({ field, value, onSettingChange }: SettingsFieldRendererProps) {
  const displayValue = value === undefined || value === null ? '' : `${value as number}`
  return (
    <Input
      id={field.name}
      type="number"
      placeholder={field.placeholder}
      value={displayValue}
      onChange={(e) =>
        onSettingChange(field.name, e.target.value ? Number(e.target.value) : undefined)
      }
    />
  )
}

function BoolField({ field, value, onSettingChange }: SettingsFieldRendererProps) {
  return (
    <Switch
      id={field.name}
      checked={Boolean(value)}
      onCheckedChange={(checked) => onSettingChange(field.name, checked)}
    />
  )
}

function SelectField({
  field,
  value,
  isPlex,
  hasPlexToken,
  isLoadingServers,
  plexServers,
  onSettingChange,
}: SettingsFieldRendererProps) {
  if (isPlex && field.name === 'serverId') {
    return (
      <PlexServerSelect
        value={value}
        hasPlexToken={hasPlexToken}
        isLoadingServers={isLoadingServers}
        plexServers={plexServers}
        onSettingChange={onSettingChange}
        fieldName={field.name}
      />
    )
  }

  const selectValue = typeof value === 'string' ? value : (field.default as string | undefined) ?? ''
  const selectedLabel =
    field.options?.find((o) => o.value === (value ?? field.default))?.label ?? 'Select...'

  return (
    <Select value={selectValue} onValueChange={(v) => onSettingChange(field.name, v)}>
      <SelectTrigger>{selectedLabel}</SelectTrigger>
      <SelectContent>
        {field.options?.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function ActionField({
  field,
  isPlex,
  hasPlexToken,
  isPlexConnecting,
  onPlexConnect,
  onPlexDisconnect,
}: SettingsFieldRendererProps) {
  if (field.actionType === 'oauth' && isPlex) {
    return (
      <PlexOAuthButton
        hasPlexToken={hasPlexToken}
        isPlexConnecting={isPlexConnecting}
        actionLabel={field.actionLabel}
        onConnect={onPlexConnect}
        onDisconnect={onPlexDisconnect}
      />
    )
  }
  return (
    <Button
      variant="outline"
      onClick={() => field.actionEndpoint && fetch(field.actionEndpoint)}
    >
      {field.actionLabel ?? 'Action'}
    </Button>
  )
}

type PlexServerSelectProps = {
  value: unknown
  hasPlexToken: boolean
  isLoadingServers: boolean
  plexServers: PlexServer[]
  onSettingChange: (name: string, value: unknown) => void
  fieldName: string
}

function PlexServerSelect({
  value,
  hasPlexToken,
  isLoadingServers,
  plexServers,
  onSettingChange,
  fieldName,
}: PlexServerSelectProps) {
  const triggerContent = isLoadingServers ? (
    <span className="flex items-center gap-2">
      <Loader2 className="size-4 animate-spin" />
      Loading servers...
    </span>
  ) : (
    (plexServers.find((s) => s.id === value)?.name ?? 'Select server...')
  )

  const selectValue = typeof value === 'string' ? value : ''

  return (
    <Select
      value={selectValue}
      onValueChange={(v) => onSettingChange(fieldName, v)}
      disabled={!hasPlexToken || isLoadingServers}
    >
      <SelectTrigger>{triggerContent}</SelectTrigger>
      <SelectContent>
        {plexServers.map((server) => (
          <SelectItem key={server.id} value={server.id}>
            {server.name} {server.owned ? '(owned)' : null}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
