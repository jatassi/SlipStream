import { useCallback, useEffect, useRef, useState } from 'react'

import { Save } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useModuleNamingSettings, useUpdateModuleNamingSettings } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { ModuleNamingSettings, UpdateModuleNamingRequest } from '@/types'

import {
  COLON_REPLACEMENT_OPTIONS,
  MULTI_EPISODE_STYLES,
} from './file-naming-constants'
import { FilenameTester } from './filename-tester'
import { PatternEditor } from './naming-pattern-editor'

const MODULE_ID = 'tv'

function toFormData(s: ModuleNamingSettings): UpdateModuleNamingRequest {
  return {
    renameEnabled: s.renameEnabled,
    colonReplacement: s.colonReplacement,
    customColonReplacement: s.customColonReplacement,
    multiEpisodeStyle: (s as ModuleNamingSettings & { multiEpisodeStyle?: string }).multiEpisodeStyle,
    patterns: { ...s.patterns },
  }
}

function useTvNamingForm(settings: ModuleNamingSettings) {
  const updateMutation = useUpdateModuleNamingSettings(MODULE_ID)
  const [form, setForm] = useState(() => toFormData(settings))
  const [prevSettings, setPrevSettings] = useState(settings)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    setForm(toFormData(settings))
  }

  const updateField = useCallback(<K extends keyof UpdateModuleNamingRequest>(field: K, value: UpdateModuleNamingRequest[K]) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }, [])

  const updatePattern = useCallback((key: string, value: string) => {
    setForm((prev) => ({ ...prev, patterns: { ...prev.patterns, [key]: value } }))
  }, [])

  const debouncedForm = useDebounce(form, 1000)
  const lastSavedRef = useRef<string | null>(null)

  useEffect(() => {
    const formJson = JSON.stringify(debouncedForm)
    const settingsJson = JSON.stringify(toFormData(settings))
    if (formJson !== settingsJson && formJson !== lastSavedRef.current) {
      lastSavedRef.current = formJson
      updateMutation.mutate(debouncedForm, {
        onError: () => {
          toast.error('Failed to auto-save naming settings')
          lastSavedRef.current = null
        },
      })
    }
  }, [debouncedForm, settings, updateMutation])

  return { form, updateField, updatePattern, isSaving: updateMutation.isPending }
}

function ColonReplacementSelect({
  value,
  customValue,
  onChangeReplacement,
  onChangeCustom,
}: {
  value: string
  customValue: string
  onChangeReplacement: (v: string) => void
  onChangeCustom: (v: string) => void
}) {
  return (
    <div className="space-y-3">
      <Label>Colon Replacement</Label>
      <Select value={value} onValueChange={(v) => v && onChangeReplacement(v)}>
        <SelectTrigger>
          {COLON_REPLACEMENT_OPTIONS.find((o) => o.value === value)?.label}
        </SelectTrigger>
        <SelectContent>
          {COLON_REPLACEMENT_OPTIONS.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-xs">
        Example:{' '}
        {COLON_REPLACEMENT_OPTIONS.find((o) => o.value === value)?.example}
      </p>
      {value === 'custom' && (
        <Input
          value={customValue}
          onChange={(e) => onChangeCustom(e.target.value)}
          placeholder="Enter custom replacement character"
        />
      )}
    </div>
  )
}

function MultiEpisodeStyleSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="space-y-3">
      <Label>Multi-Episode Style</Label>
      <Select value={value} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger>
          {MULTI_EPISODE_STYLES.find((s) => s.value === value)?.label}
        </SelectTrigger>
        <SelectContent>
          {MULTI_EPISODE_STYLES.map((style) => (
            <SelectItem key={style.value} value={style.value}>
              {style.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground font-mono text-xs">
        Example:{' '}
        {MULTI_EPISODE_STYLES.find((s) => s.value === value)?.example}
      </p>
    </div>
  )
}

function EpisodeRenamingCard({
  form,
  updateField,
}: {
  form: UpdateModuleNamingRequest
  updateField: <K extends keyof UpdateModuleNamingRequest>(field: K, value: UpdateModuleNamingRequest[K]) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Episode Renaming</CardTitle>
        <CardDescription>Configure how TV episodes are renamed during import</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Rename Episodes</Label>
            <p className="text-muted-foreground text-sm">
              Rename files according to format patterns
            </p>
          </div>
          <Switch
            checked={form.renameEnabled ?? false}
            onCheckedChange={(v) => updateField('renameEnabled', v)}
          />
        </div>

        <ColonReplacementSelect
          value={form.colonReplacement ?? 'delete'}
          customValue={form.customColonReplacement ?? ''}
          onChangeReplacement={(v) => updateField('colonReplacement', v)}
          onChangeCustom={(v) => updateField('customColonReplacement', v)}
        />
        <MultiEpisodeStyleSelect
          value={form.multiEpisodeStyle ?? 'extend'}
          onChange={(v) => updateField('multiEpisodeStyle', v)}
        />
      </CardContent>
    </Card>
  )
}

function EpisodeFormatCard({
  form,
  updatePattern,
}: {
  form: UpdateModuleNamingRequest
  updatePattern: (key: string, value: string) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Episode Format Patterns</CardTitle>
        <CardDescription>Define naming patterns for different episode types</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <PatternEditor
          label="Standard Episode Format"
          value={form.patterns?.['episode-file.standard'] ?? ''}
          onChange={(v) => updatePattern('episode-file.standard', v)}
          description="For regular TV series"
          mediaType="episode"
          tokenContext="episode"
          moduleId={MODULE_ID}
          contextName="episode-file.standard"
        />
        <PatternEditor
          label="Daily Episode Format"
          value={form.patterns?.['episode-file.daily'] ?? ''}
          onChange={(v) => updatePattern('episode-file.daily', v)}
          description="For daily/date-based shows"
          mediaType="episode"
          tokenContext="episode"
          moduleId={MODULE_ID}
          contextName="episode-file.daily"
        />
        <PatternEditor
          label="Anime Episode Format"
          value={form.patterns?.['episode-file.anime'] ?? ''}
          onChange={(v) => updatePattern('episode-file.anime', v)}
          description="For anime series"
          mediaType="episode"
          tokenContext="episode"
          moduleId={MODULE_ID}
          contextName="episode-file.anime"
        />
      </CardContent>
    </Card>
  )
}

function FolderFormatCard({
  form,
  updatePattern,
}: {
  form: UpdateModuleNamingRequest
  updatePattern: (key: string, value: string) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Folder Format Patterns</CardTitle>
        <CardDescription>
          Define folder naming patterns for series organization
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <PatternEditor
          label="Series Folder Format"
          value={form.patterns?.['series-folder'] ?? ''}
          onChange={(v) => updatePattern('series-folder', v)}
          description="Root folder for each series"
          mediaType="folder"
          tokenContext="series-folder"
          moduleId={MODULE_ID}
          contextName="series-folder"
        />
        <PatternEditor
          label="Season Folder Format"
          value={form.patterns?.['season-folder'] ?? ''}
          onChange={(v) => updatePattern('season-folder', v)}
          description="Subfolder for each season"
          mediaType="folder"
          tokenContext="season-folder"
          moduleId={MODULE_ID}
          contextName="season-folder"
        />
        <PatternEditor
          label="Specials Folder Format"
          value={form.patterns?.['specials-folder'] ?? ''}
          onChange={(v) => updatePattern('specials-folder', v)}
          description="Folder for specials (Season 0)"
          mediaType="folder"
          tokenContext="series-folder"
          moduleId={MODULE_ID}
          contextName="specials-folder"
        />
      </CardContent>
    </Card>
  )
}

function TvNamingContent({ settings }: { settings: ModuleNamingSettings }) {
  const { form, updateField, updatePattern, isSaving } = useTvNamingForm(settings)

  return (
    <>
      <div className="flex justify-end">
        <span className="text-muted-foreground flex items-center gap-2 text-sm">
          <Save className={`size-4 ${isSaving ? 'animate-pulse' : ''}`} />
          {isSaving ? 'Saving...' : 'Auto-save'}
        </span>
      </div>
      <FilenameTester mediaType="tv" />
      <EpisodeRenamingCard form={form} updateField={updateField} />
      <EpisodeFormatCard form={form} updatePattern={updatePattern} />
      <FolderFormatCard form={form} updatePattern={updatePattern} />
    </>
  )
}

export function TvNamingTab() {
  const { data: settings, isLoading, isError, refetch } = useModuleNamingSettings(MODULE_ID)

  if (isLoading) {return <LoadingState variant="list" count={3} />}
  if (isError || !settings) {return <ErrorState onRetry={refetch} />}

  return <TvNamingContent settings={settings} />
}
