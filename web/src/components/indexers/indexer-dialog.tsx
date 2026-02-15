import { ArrowLeft, Globe, Loader2, Lock, TestTube, Unlock } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import type { Indexer, Privacy, Protocol } from '@/types'

import { DefinitionSearchTable } from './definition-search-table'
import { DynamicSettingsForm } from './dynamic-settings-form'
import { useIndexerDialog } from './use-indexer-dialog'

type IndexerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  indexer?: Indexer | null
}

const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-4" />,
  'semi-private': <Unlock className="size-4" />,
  private: <Lock className="size-4" />,
}

const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500',
  'semi-private': 'bg-yellow-500/10 text-yellow-500',
  private: 'bg-red-500/10 text-red-500',
}

const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500',
  usenet: 'bg-purple-500/10 text-purple-500',
}

export function IndexerDialog({ open, onOpenChange, indexer }: IndexerDialogProps) {
  const hook = useIndexerDialog(open, indexer, onOpenChange)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={
          hook.step === 'select'
            ? 'flex h-[600px] flex-col overflow-hidden sm:max-w-3xl'
            : 'flex h-[80vh] flex-col overflow-hidden sm:max-w-2xl'
        }
      >
        <DialogHeader>
          <HeaderContent hook={hook} />
        </DialogHeader>

        {hook.step === 'select' && (
          <div className="min-h-0 flex-1 overflow-hidden">
            <DefinitionSearchTable
              definitions={hook.definitions}
              isLoading={hook.isLoadingDefinitions}
              onSelect={hook.handleDefinitionSelect}
            />
          </div>
        )}

        {hook.step === 'configure' && hook.selectedDefinition ? <ConfigureStep hook={hook} privacyIcons={privacyIcons} privacyColors={privacyColors} protocolColors={protocolColors} /> : null}

        {hook.step === 'configure' && (
          <FooterActions hook={hook} onOpenChange={onOpenChange} />
        )}
      </DialogContent>
    </Dialog>
  )
}

type HookValues = ReturnType<typeof useIndexerDialog>

function HeaderContent({ hook }: { hook: HookValues }) {
  let titleText: string
  let descriptionText: string

  if (hook.step === 'select') {
    titleText = 'Add Indexer'
    descriptionText = 'Select an indexer from the list below.'
  } else if (hook.isEditing) {
    titleText = 'Edit Indexer'
    descriptionText = 'Configure the indexer settings.'
  } else {
    titleText = 'Configure Indexer'
    descriptionText = 'Configure the indexer settings.'
  }

  return (
    <>
      <DialogTitle className="flex items-center gap-2">
        {hook.step === 'configure' && !hook.isEditing && (
          <Button variant="ghost" size="icon" className="size-6" onClick={hook.handleBack}>
            <ArrowLeft className="size-4" />
          </Button>
        )}
        {titleText}
      </DialogTitle>
      <DialogDescription>{descriptionText}</DialogDescription>
    </>
  )
}

function FooterActions({
  hook,
  onOpenChange,
}: {
  hook: HookValues
  onOpenChange: (open: boolean) => void
}) {
  return (
    <DialogFooter className="flex-col gap-2 sm:flex-row">
      <LoadingButton loading={hook.isTesting} icon={TestTube} variant="outline" onClick={hook.handleTest}>
        Test
      </LoadingButton>
      <div className="flex gap-2 sm:ml-auto">
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          Cancel
        </Button>
        <LoadingButton loading={hook.isPending} onClick={hook.handleSubmit}>
          {hook.isEditing ? 'Save' : 'Add'}
        </LoadingButton>
      </div>
    </DialogFooter>
  )
}

function ConfigureStep({
  hook,
  privacyIcons,
  privacyColors,
  protocolColors,
}: {
  hook: HookValues
  privacyIcons: Record<Privacy, React.ReactNode>
  privacyColors: Record<Privacy, string>
  protocolColors: Record<Protocol, string>
}) {
  if (!hook.selectedDefinition) {
    return null
  }

  return (
    <ScrollArea className="min-h-0 flex-1">
      <div className="space-y-4 py-4 pr-4">
        <DefinitionBanner
          definition={hook.selectedDefinition}
          privacyIcons={privacyIcons}
          privacyColors={privacyColors}
          protocolColors={protocolColors}
        />

        <NameInput hook={hook} />
        <SchemaSettings hook={hook} />
        <MediaTypeToggles hook={hook} />
        <PriorityInput hook={hook} />
        <EnabledToggle hook={hook} />
        <AutoSearchToggle hook={hook} definition={hook.selectedDefinition} />
        <RssToggle hook={hook} />
      </div>
    </ScrollArea>
  )
}

function DefinitionBanner({
  definition,
  privacyIcons,
  privacyColors,
  protocolColors,
}: {
  definition: { name: string; description?: string; protocol: Protocol; privacy: Privacy }
  privacyIcons: Record<Privacy, React.ReactNode>
  privacyColors: Record<Privacy, string>
  protocolColors: Record<Protocol, string>
}) {
  return (
    <div className="bg-muted/50 flex items-center gap-2 rounded-lg p-3">
      <div className="flex-1">
        <p className="font-medium">{definition.name}</p>
        {definition.description ? (
          <p className="text-muted-foreground text-sm">{definition.description}</p>
        ) : null}
      </div>
      <div className="flex gap-2">
        <Badge variant="secondary" className={protocolColors[definition.protocol]}>
          {definition.protocol}
        </Badge>
        <Badge variant="secondary" className={privacyColors[definition.privacy]}>
          <span className="mr-1">{privacyIcons[definition.privacy]}</span>
          {definition.privacy}
        </Badge>
      </div>
    </div>
  )
}

function NameInput({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="name">Name</Label>
      <Input
        id="name"
        placeholder="My Indexer"
        value={hook.formData.name}
        onChange={(e) => hook.setFormData((prev) => ({ ...prev, name: e.target.value }))}
      />
    </div>
  )
}

function SchemaSettings({ hook }: { hook: HookValues }) {
  if (hook.isLoadingSchema) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="mr-2 size-4 animate-spin" />
        Loading settings...
      </div>
    )
  }
  return (
    <DynamicSettingsForm
      settings={hook.schema}
      values={hook.formData.settings}
      onChange={(settings) => hook.setFormData((prev) => ({ ...prev, settings }))}
    />
  )
}

function MediaTypeToggles({ hook }: { hook: HookValues }) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="flex items-center justify-between">
        <Label htmlFor="supportsMovies">Movies</Label>
        <Switch
          id="supportsMovies"
          checked={hook.formData.supportsMovies}
          onCheckedChange={(checked) =>
            hook.setFormData((prev) => ({ ...prev, supportsMovies: checked }))
          }
        />
      </div>
      <div className="flex items-center justify-between">
        <Label htmlFor="supportsTv">TV Shows</Label>
        <Switch
          id="supportsTv"
          checked={hook.formData.supportsTv}
          onCheckedChange={(checked) =>
            hook.setFormData((prev) => ({ ...prev, supportsTv: checked }))
          }
        />
      </div>
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
      <p className="text-muted-foreground text-xs">Lower values have higher priority (1-100)</p>
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

function AutoSearchToggle({
  hook,
  definition,
}: {
  hook: HookValues
  definition: { id: string }
}) {
  const isGenericRss = definition.id === 'generic-rss'
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5">
        <Label htmlFor="autoSearchEnabled">Enable for Automatic Search</Label>
        <p className="text-muted-foreground text-xs">
          {isGenericRss
            ? 'Generic RSS feeds do not support search'
            : 'Use this indexer when automatically searching for releases'}
        </p>
      </div>
      <Switch
        id="autoSearchEnabled"
        checked={isGenericRss ? false : hook.formData.autoSearchEnabled}
        onCheckedChange={(checked) =>
          hook.setFormData((prev) => ({ ...prev, autoSearchEnabled: checked }))
        }
        disabled={isGenericRss}
      />
    </div>
  )
}

function RssToggle({ hook }: { hook: HookValues }) {
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5">
        <Label htmlFor="rssEnabled">Enable for RSS Sync</Label>
        <p className="text-muted-foreground text-xs">
          Include this indexer when fetching RSS feeds for new releases
        </p>
      </div>
      <Switch
        id="rssEnabled"
        checked={hook.formData.rssEnabled}
        onCheckedChange={(checked) =>
          hook.setFormData((prev) => ({ ...prev, rssEnabled: checked }))
        }
      />
    </div>
  )
}
