import { Controller, useFormContext } from 'react-hook-form'

import { Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { DialogBody } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import type { Privacy, Protocol } from '@/types'

import { DynamicSettingsForm } from './dynamic-settings-form'
import { privacyColors, privacyIconsMd, protocolColors } from './prowlarr-indexer-constants'
import type { FormData, useIndexerDialog  } from './use-indexer-dialog'

type HookValues = ReturnType<typeof useIndexerDialog>

export function ConfigureStep({ hook }: { hook: HookValues }) {
  if (!hook.selectedDefinition) {
    return null
  }

  return (
    <DialogBody className="overflow-hidden">
      <ScrollArea className="size-full">
        <div className="space-y-4 py-4 pr-4">
          <DefinitionBanner definition={hook.selectedDefinition} />
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
}: {
  definition: { name: string; description?: string; protocol: Protocol; privacy: Privacy }
}) {
  return (
    <div className="bg-muted/50 flex items-center gap-2 rounded-lg p-3">
      <div className="flex-1">
        <p className="font-medium">{definition.name}</p>
        {definition.description ? <p className="text-muted-foreground text-sm">{definition.description}</p> : null}
      </div>
      <div className="flex gap-2">
        <Badge variant="secondary" className={protocolColors[definition.protocol]}>
          {definition.protocol}
        </Badge>
        <Badge variant="secondary" className={privacyColors[definition.privacy]}>
          <span className="mr-1">{privacyIconsMd[definition.privacy]}</span>
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
