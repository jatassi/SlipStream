import { Label } from '@/components/ui/label'
import type { SettingsField } from '@/types'

import type { PlexServer } from './notification-dialog-types'
import { SettingsFieldRenderer } from './settings-field-renderer'

type ProviderFieldsProps = {
  fields: SettingsField[]
  settings: Record<string, unknown>
  isPlex: boolean
  hasPlexToken: boolean
  isPlexConnecting: boolean
  isLoadingServers: boolean
  plexServers: PlexServer[]
  onSettingChange: (name: string, value: unknown) => void
  onPlexConnect: () => void
  onPlexDisconnect: () => void
}

export function ProviderFields({
  fields,
  settings,
  isPlex,
  hasPlexToken,
  isPlexConnecting,
  isLoadingServers,
  plexServers,
  onSettingChange,
  onPlexConnect,
  onPlexDisconnect,
}: ProviderFieldsProps) {
  return (
    <>
      {fields.map((field) => (
        <div
          key={field.name}
          className={field.type === 'bool' ? 'flex items-center justify-between' : 'space-y-2'}
        >
          <div>
            <Label htmlFor={field.name}>
              {field.label}
              {!field.required && field.type !== 'bool' && field.type !== 'action' && (
                <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
              )}
            </Label>
            {field.helpText && field.type !== 'bool' ? (
              <p className="text-muted-foreground text-xs">{field.helpText}</p>
            ) : null}
          </div>
          <SettingsFieldRenderer
            field={field}
            value={settings[field.name]}
            isPlex={isPlex}
            hasPlexToken={hasPlexToken}
            isPlexConnecting={isPlexConnecting}
            isLoadingServers={isLoadingServers}
            plexServers={plexServers}
            onSettingChange={onSettingChange}
            onPlexConnect={onPlexConnect}
            onPlexDisconnect={onPlexDisconnect}
          />
        </div>
      ))}
    </>
  )
}
