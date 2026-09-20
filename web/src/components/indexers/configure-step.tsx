import { Controller, useFormContext } from 'react-hook-form'

import { ArrowLeft, Loader2 } from 'lucide-react'

import { ControlStack, InputRow, SwitchRow } from '@/components/settings/control-row'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Privacy, Protocol } from '@/types'

import { DynamicSettingsForm } from './dynamic-settings-form'
import { privacyColors, privacyIconsMd, protocolColors } from './prowlarr-indexer-constants'
import type { FormData, useIndexerDialog  } from './use-indexer-dialog'

type HookValues = ReturnType<typeof useIndexerDialog>

export function ConfigureStep({ hook, onBack }: { hook: HookValues; onBack?: () => void }) {
  if (!hook.selectedDefinition) {
    return null
  }

  return (
    <div className="space-y-4 py-2">
      {onBack === undefined ? null : (
        <Button variant="ghost" className="min-h-tap -ml-2" onClick={onBack}>
          <ArrowLeft className="size-4" />
          Indexer list
        </Button>
      )}
      <DefinitionBanner definition={hook.selectedDefinition} />
      <ControlStack>
        <NameInput />
      </ControlStack>
      <SchemaSettings hook={hook} />
      <ControlStack>
        <MediaTypeToggles />
        <PriorityInput />
        <EnabledToggle />
        <AutoSearchToggle definition={hook.selectedDefinition} />
        <RssToggle />
      </ControlStack>
    </div>
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
        {Boolean(definition.description) && <p className="text-muted-foreground text-sm">{definition.description}</p>}
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
  return <InputRow stacked label="Name" aria-label="Name" placeholder="My Indexer" {...register('name')} />
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
    <>
      <Controller
        control={control}
        name="supportsMovies"
        render={({ field }) => (
          <SwitchRow label="Movies" checked={field.value} onCheckedChange={field.onChange} />
        )}
      />
      <Controller
        control={control}
        name="supportsTv"
        render={({ field }) => (
          <SwitchRow label="TV Shows" checked={field.value} onCheckedChange={field.onChange} />
        )}
      />
    </>
  )
}

function PriorityInput() {
  const { register } = useFormContext<FormData>()
  return (
    <InputRow
      label="Priority"
      aria-label="Priority"
      type="number"
      min={1}
      max={100}
      {...register('priority', { valueAsNumber: true })}
    />
  )
}

function EnabledToggle() {
  const { control } = useFormContext<FormData>()
  return (
    <Controller
      control={control}
      name="enabled"
      render={({ field }) => (
        <SwitchRow label="Enabled" checked={field.value} onCheckedChange={field.onChange} />
      )}
    />
  )
}

function AutoSearchToggle({ definition }: { definition: { id: string } }) {
  const { control } = useFormContext<FormData>()
  const isGenericRss = definition.id === 'generic-rss'
  return (
    <Controller
      control={control}
      name="autoSearchEnabled"
      render={({ field }) => (
        <SwitchRow
          label="Enable for Automatic Search"
          description={
            isGenericRss
              ? 'Generic RSS feeds do not support search'
              : 'Use this indexer when automatically searching for releases'
          }
          checked={isGenericRss ? false : field.value}
          onCheckedChange={field.onChange}
          disabled={isGenericRss}
        />
      )}
    />
  )
}

function RssToggle() {
  const { control } = useFormContext<FormData>()
  return (
    <Controller
      control={control}
      name="rssEnabled"
      render={({ field }) => (
        <SwitchRow
          label="Enable for RSS Sync"
          description="Include this indexer when fetching RSS feeds for new releases"
          checked={field.value}
          onCheckedChange={field.onChange}
        />
      )}
    />
  )
}
