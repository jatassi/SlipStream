import { useState } from 'react'

import { Save, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'

import { Group } from '@/components/grouped-list'
import { InputRow, TextareaRow } from '@/components/settings/control-row'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { LoadingButton } from '@/components/ui/loading-button'
import { useSettings, useUpdateSettings } from '@/hooks'

type Draft = {
  rpId: string
  rpOrigins: string
  rpDisplayName: string
}

type SettingsSnapshot = {
  webauthnRpId: string
  webauthnRpOrigins: string[]
  webauthnRpDisplayName: string
}

function toDraft(settings: SettingsSnapshot): Draft {
  return {
    rpId: settings.webauthnRpId,
    rpOrigins: settings.webauthnRpOrigins.join('\n'),
    rpDisplayName: settings.webauthnRpDisplayName,
  }
}

function isDirty(draft: Draft, baseline: Draft | null): boolean {
  if (!baseline) {
    return false
  }
  return (
    draft.rpId !== baseline.rpId ||
    draft.rpOrigins !== baseline.rpOrigins ||
    draft.rpDisplayName !== baseline.rpDisplayName
  )
}

function parseOrigins(raw: string): string[] {
  return raw
    .split('\n')
    .map((o) => o.trim())
    .filter((o) => o.length > 0)
}

function RestartNotice() {
  return (
    <div className="px-screen mb-7">
      <Alert>
        <TriangleAlert className="size-4 text-amber-500" />
        <AlertTitle>Restart required</AlertTitle>
        <AlertDescription>
          Saved changes only apply after the server is restarted. Passkeys are bound to the
          hostname they were registered on, so you may need to register a new passkey for any
          newly added origin.
        </AlertDescription>
      </Alert>
    </div>
  )
}

function FormGroups({ draft, onChange }: { draft: Draft; onChange: (next: Draft) => void }) {
  return (
    <>
      <Group
        header="Passkey Relying Party"
        footer="Bare hostname only — no scheme, port, or path. Required for startup validation; the RP ID actually used at sign-in is derived from the matching allowed origin."
      >
        <InputRow
          label="Default Relying Party ID"
          stacked
          value={draft.rpId}
          onChange={(e) => onChange({ ...draft, rpId: e.target.value })}
          placeholder="example.com"
        />
      </Group>
      <Group
        header="Allowed Origins"
        footer="One full origin per line, including scheme (e.g. https://example.com). SlipStream picks the matching RP ID per request and rejects sign-in attempts from hosts not on the list."
      >
        <TextareaRow
          label="Allowed Origins"
          value={draft.rpOrigins}
          onChange={(e) => onChange({ ...draft, rpOrigins: e.target.value })}
          placeholder={'http://localhost:3000\nhttps://example.com'}
          rows={3}
          className="font-mono"
        />
      </Group>
      <Group
        header="Display Name"
        footer="Shown to users by their authenticator during passkey prompts."
      >
        <InputRow
          label="Display Name"
          stacked
          value={draft.rpDisplayName}
          onChange={(e) => onChange({ ...draft, rpDisplayName: e.target.value })}
          placeholder="SlipStream"
        />
      </Group>
    </>
  )
}

function useWebAuthnDraft() {
  const { data: settings } = useSettings()
  const updateMutation = useUpdateSettings()

  const [draft, setDraft] = useState<Draft | null>(null)
  const [baseline, setBaseline] = useState<Draft | null>(null)

  if (settings && baseline === null) {
    const initial = toDraft(settings)
    setBaseline(initial)
    setDraft(initial)
  }

  const handleSave = async () => {
    if (!draft) {return}
    const origins = parseOrigins(draft.rpOrigins)
    try {
      await updateMutation.mutateAsync({
        webauthnRpId: draft.rpId.trim(),
        webauthnRpOrigins: origins,
        webauthnRpDisplayName: draft.rpDisplayName.trim(),
      })
      const next: Draft = {
        rpId: draft.rpId.trim(),
        rpOrigins: origins.join('\n'),
        rpDisplayName: draft.rpDisplayName.trim(),
      }
      setBaseline(next)
      setDraft(next)
      toast.success('Passkey relying party settings saved. Restart required to take effect.')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to save passkey settings')
    }
  }

  return {
    draft,
    setDraft,
    dirty: isDirty(draft ?? { rpId: '', rpOrigins: '', rpDisplayName: '' }, baseline),
    isSaving: updateMutation.isPending,
    isReady: settings !== undefined && draft !== null,
    handleSave,
  }
}

export function WebAuthnRPConfig() {
  const { draft, setDraft, dirty, isSaving, isReady, handleSave } = useWebAuthnDraft()

  if (!isReady || !draft) {
    return null
  }

  return (
    <>
      <FormGroups draft={draft} onChange={setDraft} />
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {dirty && <RestartNotice />}
      <div className="px-screen mb-7 flex justify-end">
        <LoadingButton
          className="h-11"
          loading={isSaving}
          icon={Save}
          onClick={() => void handleSave()}
          disabled={!dirty}
        >
          Save
        </LoadingButton>
      </div>
    </>
  )
}
