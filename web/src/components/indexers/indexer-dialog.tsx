import { Controller, FormProvider, useFormContext } from 'react-hook-form'

import { ArrowLeft, Globe, Loader2, Lock, TestTube, Unlock } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogBody,
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
import type { FormData } from './use-indexer-dialog'
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
    <FormProvider {...hook.form}>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className={hook.step === 'select' ? 'h-[600px] sm:max-w-3xl' : 'h-[80vh] sm:max-w-2xl'}
        >
          <DialogHeader>
            <HeaderContent hook={hook} />
          </DialogHeader>

          {hook.step === 'select' && (
            <DialogBody className="overflow-hidden">
              <DefinitionSearchTable
                definitions={hook.definitions}
                isLoading={hook.isLoadingDefinitions}
                onSelect={hook.handleDefinitionSelect}
              />
            </DialogBody>
          )}

          {hook.step === 'configure' && hook.selectedDefinition ? (
            <ConfigureStep hook={hook} />
          ) : null}

          {hook.step === 'configure' && (
            <FooterActions hook={hook} onOpenChange={onOpenChange} />
          )}
        </DialogContent>
      </Dialog>
    </FormProvider>
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

function ConfigureStep({ hook }: { hook: HookValues }) {
  if (!hook.selectedDefinition) {
    return null
  }

  return (
    <DialogBody className="overflow-hidden">
      <ScrollArea className="size-full">
        <div className="space-y-4 py-4 pr-4">
          <DefinitionBanner
            definition={hook.selectedDefinition}
            privacyIcons={privacyIcons}
            privacyColors={privacyColors}
            protocolColors={protocolColors}
          />
          <NameInput />
          <SchemaSettings hook={hook} />
          <MediaTypeToggles />
          <PriorityInput />
          <EnabledToggle />
          <AutoSearchToggle definition={hook.selectedDefinition} />
          <RssToggle />
        </div>
      </ScrollArea>
    </DialogBody>
  )
}

function DefinitionBanner({
  definition,
  privacyIcons: icons,
  privacyColors: pColors,
  protocolColors: prColors,
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
        <Badge variant="secondary" className={prColors[definition.protocol]}>
          {definition.protocol}
        </Badge>
        <Badge variant="secondary" className={pColors[definition.privacy]}>
          <span className="mr-1">{icons[definition.privacy]}</span>
          {definition.privacy}
        </Badge>
      </div>
    </div>
  )
}

function NameInput() {
  const { register } = useFormContext<FormData>()
  return (
    <div className="space-y-2">
      <Label htmlFor="name">Name</Label>
      <Input id="name" placeholder="My Indexer" {...register('name')} />
    </div>
  )
}

function SchemaSettings({ hook }: { hook: HookValues }) {
  const { control } = useFormContext<FormData>()

  if (hook.isLoadingSchema) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="mr-2 size-4 animate-spin" />
        Loading settings...
      </div>
    )
  }

  return (
    <Controller
      control={control}
      name="settings"
      render={({ field }) => (
        <DynamicSettingsForm
          settings={hook.schema}
          values={field.value}
          onChange={field.onChange}
        />
      )}
    />
  )
}

function MediaTypeToggles() {
  const { control } = useFormContext<FormData>()
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="flex items-center justify-between">
        <Label htmlFor="supportsMovies">Movies</Label>
        <Controller
          control={control}
          name="supportsMovies"
          render={({ field }) => (
            <Switch id="supportsMovies" checked={field.value} onCheckedChange={field.onChange} />
          )}
        />
      </div>
      <div className="flex items-center justify-between">
        <Label htmlFor="supportsTv">TV Shows</Label>
        <Controller
          control={control}
          name="supportsTv"
          render={({ field }) => (
            <Switch id="supportsTv" checked={field.value} onCheckedChange={field.onChange} />
          )}
        />
      </div>
    </div>
  )
}

function PriorityInput() {
  const { register } = useFormContext<FormData>()
  return (
    <div className="space-y-2">
      <Label htmlFor="priority">Priority</Label>
      <Input
        id="priority"
        type="number"
        min={1}
        max={100}
        {...register('priority', { valueAsNumber: true })}
      />
      <p className="text-muted-foreground text-xs">Lower values have higher priority (1-100)</p>
    </div>
  )
}

function EnabledToggle() {
  const { control } = useFormContext<FormData>()
  return (
    <div className="flex items-center justify-between">
      <Label htmlFor="enabled">Enabled</Label>
      <Controller
        control={control}
        name="enabled"
        render={({ field }) => (
          <Switch id="enabled" checked={field.value} onCheckedChange={field.onChange} />
        )}
      />
    </div>
  )
}

function AutoSearchToggle({ definition }: { definition: { id: string } }) {
  const { control } = useFormContext<FormData>()
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
      <Controller
        control={control}
        name="autoSearchEnabled"
        render={({ field }) => (
          <Switch
            id="autoSearchEnabled"
            checked={isGenericRss ? false : field.value}
            onCheckedChange={field.onChange}
            disabled={isGenericRss}
          />
        )}
      />
    </div>
  )
}

function RssToggle() {
  const { control } = useFormContext<FormData>()
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5">
        <Label htmlFor="rssEnabled">Enable for RSS Sync</Label>
        <p className="text-muted-foreground text-xs">
          Include this indexer when fetching RSS feeds for new releases
        </p>
      </div>
      <Controller
        control={control}
        name="rssEnabled"
        render={({ field }) => (
          <Switch id="rssEnabled" checked={field.value} onCheckedChange={field.onChange} />
        )}
      />
    </div>
  )
}
