import { Bug, TestTube } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {hook.isEditing ? 'Edit Download Client' : 'Add Download Client'}
          </DialogTitle>
          <DialogDescription>
            Configure connection settings for your download client.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <ClientTypeSelect hook={hook} />
          <NameInput hook={hook} />
          <HostPortInputs hook={hook} />
          <SslToggle hook={hook} />
          {hook.config.supportsUrlBase ? <UrlBaseInput hook={hook} /> : null}
          {hook.config.supportsUsername ? <UsernameInput hook={hook} /> : null}
          {hook.config.supportsPassword ? <PasswordInput hook={hook} /> : null}
          {hook.config.supportsApiKey ? <ApiKeyInput hook={hook} /> : null}
          {hook.config.supportsCategory ? <CategoryInput hook={hook} /> : null}
          <PriorityInput hook={hook} />
          <EnabledToggle hook={hook} />
        </div>

        <DialogFooter className="flex-col gap-2 sm:flex-row">
          <ActionButtons hook={hook} onCancel={() => onOpenChange(false)} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type HookValues = ReturnType<typeof useDownloadClientDialog>

function ClientTypeSelect({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="type">Client Type</Label>
      <Select
        value={hook.formData.type}
        onValueChange={(v) => v && hook.handleTypeChange(v as DownloadClientType)}
      >
        <SelectTrigger>
          <SelectValue>{clientTypeConfigs[hook.formData.type].label}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          {(Object.entries(clientTypeConfigs) as [DownloadClientType, { label: string }][]).map(
            ([value, config]) => (
              <SelectItem key={value} value={value}>
                {config.label}
              </SelectItem>
            ),
          )}
        </SelectContent>
      </Select>
    </div>
  )
}

function NameInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="name">Name</Label>
      <Input
        id="name"
        placeholder="My Download Client"
        value={hook.formData.name}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, name: e.target.value }))}
      />
    </div>
  )
}

function HostPortInputs({ hook }: { hook: HookValues }) {
  return (
    <div className="grid grid-cols-3 gap-4">
      <div className="col-span-2 space-y-2">
        <Label htmlFor="host">Host</Label>
        <Input
          id="host"
          placeholder="localhost"
          value={hook.formData.host}
          onChange={(e) => hook.setFormData((prev) => ({ ...prev, host: e.target.value }))}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="port">Port</Label>
        <Input
          id="port"
          type="number"
          value={hook.formData.port}
          onChange={(e) =>
            hook.setFormData((prev) => ({ ...prev, port: Number.parseInt(e.target.value) || 0 }))
          }
        />
      </div>
    </div>
  )
}

function SslToggle({ hook }: { hook: HookValues }) {
  return (
    <div className="flex items-center justify-between">
      <Label htmlFor="useSsl">Use SSL</Label>
      <Switch
        id="useSsl"
        checked={hook.formData.useSsl}
        onCheckedChange={(checked) => hook.setFormData((prev) => ({ ...prev, useSsl: checked }))}
      />
    </div>
  )
}

function UrlBaseInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="urlBase">
        URL Base
        <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
      </Label>
      <Input
        id="urlBase"
        placeholder="/"
        value={hook.formData.urlBase}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, urlBase: e.target.value }))}
      />
    </div>
  )
}

function UsernameInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="username">
        {hook.config.usernameLabel}
        <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
      </Label>
      <Input
        id="username"
        value={hook.formData.username}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, username: e.target.value }))}
      />
    </div>
  )
}

function PasswordInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="password">
        {hook.config.passwordLabel}
        {!hook.config.passwordRequired && (
          <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
        )}
      </Label>
      <Input
        id="password"
        type="password"
        value={hook.formData.password}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, password: e.target.value }))}
      />
    </div>
  )
}

function ApiKeyInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="apiKey">
        {hook.config.apiKeyLabel}
        <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
      </Label>
      <Input
        id="apiKey"
        type="password"
        value={hook.formData.apiKey}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, apiKey: e.target.value }))}
      />
    </div>
  )
}

function CategoryInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="category">
        Category
        <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
      </Label>
      <Input
        id="category"
        placeholder="slipstream"
        value={hook.formData.category}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, category: e.target.value }))}
      />
    </div>
  )
}

function PriorityInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="priority">Priority</Label>
      <Input
        id="priority"
        type="number"
        min={1}
        max={100}
        value={hook.formData.priority}
        onChange={(e) =>
          hook.setFormData((prev) => ({
            ...prev,
            priority: Number.parseInt(e.target.value) || 50,
          }))
        }
      />
      <p className="text-muted-foreground text-xs">
        Lower values have higher priority (1-100)
      </p>
    </div>
  )
}

function EnabledToggle({ hook }: { hook: HookValues }) {
  return (
    <div className="flex items-center justify-between">
      <Label htmlFor="enabled">Enabled</Label>
      <Switch
        id="enabled"
        checked={hook.formData.enabled}
        onCheckedChange={(checked) => hook.setFormData((prev) => ({ ...prev, enabled: checked }))}
      />
    </div>
  )
}

function ActionButtons({ hook, onCancel }: { hook: HookValues; onCancel: () => void }) {
  return (
    <>
      <LeftButtons hook={hook} />
      <RightButtons onCancel={onCancel} hook={hook} />
    </>
  )
}

function LeftButtons({ hook }: { hook: HookValues }) {
  const showDebug = hook.developerMode && hook.isEditing
  return (
    <div className="flex gap-2">
      <LoadingButton loading={hook.isTesting} icon={TestTube} variant="outline" onClick={hook.handleTest}>
        Test
      </LoadingButton>
      {showDebug ? (
        <LoadingButton loading={hook.isAddingDebugTorrent} icon={Bug} variant="outline" onClick={hook.handleDebugTorrent} title="Add mock download for testing">
          Debug
        </LoadingButton>
      ) : null}
    </div>
  )
}

function RightButtons({ onCancel, hook }: { onCancel: () => void; hook: HookValues }) {
  return (
    <div className="flex gap-2 sm:ml-auto">
      <Button variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <LoadingButton loading={hook.isPending} onClick={hook.handleSubmit}>
        {hook.isEditing ? 'Save' : 'Add'}
      </LoadingButton>
    </div>
  )
}
