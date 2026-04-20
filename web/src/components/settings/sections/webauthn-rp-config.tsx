import { useState } from 'react'

import { Save, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import { Textarea } from '@/components/ui/textarea'
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
  if (!baseline) {return false}
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

function SectionHeader() {
  return (
    <div>
      <Label className="text-base">Passkey Relying Party</Label>
      <p className="text-muted-foreground text-sm">
        Configure the WebAuthn relying party for this server. Required when SlipStream is reached
        via a domain other than localhost.
      </p>
    </div>
  )
}

function RestartNotice() {
  return (
    <Alert>
      <TriangleAlert className="size-4 text-amber-500" />
      <AlertTitle>Restart required</AlertTitle>
      <AlertDescription>
        Saved changes only apply after the server is restarted. Existing passkeys registered under
        a different Relying Party ID will stop working.
      </AlertDescription>
    </Alert>
  )
}

function FormFields({ draft, onChange }: { draft: Draft; onChange: (next: Draft) => void }) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="webauthnRpId">Relying Party ID</Label>
        <Input
          id="webauthnRpId"
          value={draft.rpId}
          onChange={(e) => onChange({ ...draft, rpId: e.target.value })}
          placeholder="example.com"
        />
        <p className="text-muted-foreground text-xs">
          Bare hostname only — no scheme, port, or path. Must match (or be a registrable suffix of)
          the host shown in the browser address bar.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="webauthnRpOrigins">Allowed Origins</Label>
        <Textarea
          id="webauthnRpOrigins"
          value={draft.rpOrigins}
          onChange={(e) => onChange({ ...draft, rpOrigins: e.target.value })}
          placeholder="https://example.com"
          rows={3}
          className="font-mono text-sm"
        />
        <p className="text-muted-foreground text-xs">
          One full origin per line, including scheme (e.g. <code>https://example.com</code>).
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="webauthnRpDisplayName">Display Name</Label>
        <Input
          id="webauthnRpDisplayName"
          value={draft.rpDisplayName}
          onChange={(e) => onChange({ ...draft, rpDisplayName: e.target.value })}
          placeholder="SlipStream"
        />
        <p className="text-muted-foreground text-xs">
          Shown to users by their authenticator during passkey prompts.
        </p>
      </div>
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
    <div className="space-y-4">
      <SectionHeader />
      <FormFields draft={draft} onChange={setDraft} />
      {dirty ? <RestartNotice /> : null}
      <div className="flex justify-end">
        <LoadingButton loading={isSaving} icon={Save} onClick={() => void handleSave()} disabled={!dirty}>
          Save
        </LoadingButton>
      </div>
    </div>
  )
}
