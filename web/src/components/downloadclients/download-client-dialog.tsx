import { Bug, TestTube } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { ControlStack, InputRow, SelectRow, SwitchRow } from '@/components/settings/control-row'
import { LoadingButton } from '@/components/ui/loading-button'
import type { DownloadClient, DownloadClientType } from '@/types'

import { clientTypeConfigs, useDownloadClientDialog } from './use-download-client-dialog'

type DownloadClientDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  client?: DownloadClient | null
}

export function DownloadClientDialog({ open, onOpenChange, client }: DownloadClientDialogProps) {
  const hook = useDownloadClientDialog(open, client, onOpenChange)

  return (
    <SheetPresenter
      open={open}
      onOpenChange={onOpenChange}
      title={hook.isEditing ? 'Edit Download Client' : 'Add Download Client'}
      description="Configure connection settings for your download client."
      footer={<ActionButtons hook={hook} onCancel={() => onOpenChange(false)} />}
    >
      <ClientForm hook={hook} />
    </SheetPresenter>
  )
}

type HookValues = ReturnType<typeof useDownloadClientDialog>

const TYPE_OPTIONS = (Object.entries(clientTypeConfigs) as [DownloadClientType, { label: string }][])
  .map(([value, config]) => ({ value, label: config.label }))

function ClientForm({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-4 py-2">
      <IdentityFields hook={hook} />
      <ConnectionFields hook={hook} />
      <CredentialFields hook={hook} />
      <BehaviourFields hook={hook} />
    </div>
  )
}

function IdentityFields({ hook }: { hook: HookValues }) {
  const { formData, setFormData } = hook

  return (
    <ControlStack>
        <SelectRow
          label="Client Type"
          value={formData.type}
          onChange={(value) => hook.handleTypeChange(value as DownloadClientType)}
          options={TYPE_OPTIONS}
        />
        <InputRow
          stacked
          label="Name"
          aria-label="Name"
          placeholder="My Download Client"
          value={formData.name}
          onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
        />
    </ControlStack>
  )
}

function ConnectionFields({ hook }: { hook: HookValues }) {
  const { formData, setFormData, config } = hook

  return (
    <ControlStack>
        <InputRow
          stacked
          label="Host"
          aria-label="Host"
          placeholder="localhost"
          value={formData.host}
          onChange={(e) => setFormData((prev) => ({ ...prev, host: e.target.value }))}
        />
        <InputRow
          label="Port"
          aria-label="Port"
          type="number"
          value={formData.port}
          onChange={(e) =>
            setFormData((prev) => ({ ...prev, port: Number.parseInt(e.target.value) || 0 }))
          }
        />
        <SwitchRow
          label="Use SSL"
          checked={formData.useSsl ?? false}
          onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, useSsl: checked }))}
        />
        {config.supportsUrlBase ? (
          <InputRow
            stacked
            label="URL Base (optional)"
            aria-label="URL Base"
            placeholder="/"
            value={formData.urlBase}
            onChange={(e) => setFormData((prev) => ({ ...prev, urlBase: e.target.value }))}
          />
      ) : null}
    </ControlStack>
  )
}

function BehaviourFields({ hook }: { hook: HookValues }) {
  const { formData, setFormData } = hook

  return (
    <ControlStack footer="Lower values have higher priority (1-100).">
        <InputRow
          label="Priority"
          aria-label="Priority"
          type="number"
          min={1}
          max={100}
          value={formData.priority}
          onChange={(e) =>
            setFormData((prev) => ({ ...prev, priority: Number.parseInt(e.target.value) || 50 }))
          }
        />
        <SwitchRow
          label="Enabled"
          checked={formData.enabled ?? true}
      onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked }))}
      />
    </ControlStack>
  )
}

function CredentialFields({ hook }: { hook: HookValues }) {
  const { formData, setFormData, config } = hook
  const hasAny =
    config.supportsUsername || config.supportsPassword || config.supportsApiKey || config.supportsCategory

  if (!hasAny) {
    return null
  }

  return (
    <ControlStack>
      <AuthFields hook={hook} />
      {config.supportsCategory ? (
        <InputRow
          stacked
          label="Category (optional)"
          aria-label="Category"
          placeholder="slipstream"
          value={formData.category}
          onChange={(e) => setFormData((prev) => ({ ...prev, category: e.target.value }))}
        />
      ) : null}
    </ControlStack>
  )
}

function AuthFields({ hook }: { hook: HookValues }) {
  const { formData, setFormData, config } = hook

  return (
    <>
      {config.supportsUsername ? (
        <InputRow
          stacked
          label={`${config.usernameLabel} (optional)`}
          aria-label={config.usernameLabel}
          value={formData.username}
          onChange={(e) => setFormData((prev) => ({ ...prev, username: e.target.value }))}
        />
      ) : null}
      {config.supportsPassword ? (
        <InputRow
          stacked
          label={config.passwordRequired ? config.passwordLabel : `${config.passwordLabel} (optional)`}
          aria-label={config.passwordLabel}
          type="password"
          value={formData.password}
          onChange={(e) => setFormData((prev) => ({ ...prev, password: e.target.value }))}
        />
      ) : null}
      {config.supportsApiKey ? (
        <InputRow
          stacked
          label={`${config.apiKeyLabel} (optional)`}
          aria-label={config.apiKeyLabel}
          type="password"
          value={formData.apiKey}
          onChange={(e) => setFormData((prev) => ({ ...prev, apiKey: e.target.value }))}
        />
      ) : null}
    </>
  )
}

function ActionButtons({ hook, onCancel }: { hook: HookValues; onCancel: () => void }) {
  const showDebug = hook.developerMode && hook.isEditing

  return (
    <FormActions
      leading={<SecondaryActions hook={hook} showDebug={showDebug} />}
      onCancel={onCancel}
      confirmLabel={hook.isEditing ? 'Save' : 'Add'}
      onConfirm={hook.handleSubmit}
      loading={hook.isPending}
    />
  )
}

function SecondaryActions({ hook, showDebug }: { hook: HookValues; showDebug: boolean }) {
  return (
    <div className="flex gap-2">
      <LoadingButton
        className="min-h-tap flex-1"
        loading={hook.isTesting}
        icon={TestTube}
        variant="outline"
        onClick={hook.handleTest}
      >
        Test
      </LoadingButton>
      {showDebug ? (
        <LoadingButton
          className="min-h-tap flex-1"
          loading={hook.isAddingDebugTorrent}
          icon={Bug}
          variant="outline"
          onClick={hook.handleDebugTorrent}
          title="Add mock download for testing"
        >
          Debug
        </LoadingButton>
      ) : null}
    </div>
  )
}
