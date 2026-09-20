import { useCallback, useEffect, useRef, useState } from 'react'

import { Save } from 'lucide-react'
import { toast } from 'sonner'

import { Group } from '@/components/grouped-list'
import { InputRow, SelectRow, SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import { useModuleNamingSettings, useUpdateModuleNamingSettings } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { ModuleNamingSettings, TokenContext as BackendTokenContext, UpdateModuleNamingRequest } from '@/types'

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

function ColonReplacementGroup({
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
  const option = COLON_REPLACEMENT_OPTIONS.find((o) => o.value === value)
  return (
    <Group header="Colon Replacement" footer={`Example: ${option?.example ?? ''}`}>
      <SelectRow
        label="Colon Replacement"
        value={value}
        onChange={onChangeReplacement}
        options={COLON_REPLACEMENT_OPTIONS}
      />
      {value === 'custom' && (
        <InputRow
          label="Custom Replacement"
          stacked
          value={customValue}
          onChange={(e) => onChangeCustom(e.target.value)}
          placeholder="Enter custom replacement character"
        />
      )}
    </Group>
  )
}

function MultiEpisodeStyleGroup({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const style = MULTI_EPISODE_STYLES.find((s) => s.value === value)
  return (
    <Group
      header="Multi-Episode Style"
      footer={<span className="font-mono">Example: {style?.example ?? ''}</span>}
    >
      <SelectRow
        label="Multi-Episode Style"
        value={value}
        onChange={onChange}
        options={MULTI_EPISODE_STYLES}
      />
    </Group>
  )
}

function EpisodeRenamingGroups({
  form,
  updateField,
}: {
  form: UpdateModuleNamingRequest
  updateField: <K extends keyof UpdateModuleNamingRequest>(field: K, value: UpdateModuleNamingRequest[K]) => void
}) {
  return (
    <>
      <Group
        header="Episode Renaming"
        footer="Rename episode files according to the format patterns below during import."
      >
        <SwitchRow
          label="Rename Episodes"
          checked={form.renameEnabled ?? false}
          onCheckedChange={(v) => updateField('renameEnabled', v)}
        />
      </Group>
      <ColonReplacementGroup
        value={form.colonReplacement ?? 'delete'}
        customValue={form.customColonReplacement ?? ''}
        onChangeReplacement={(v) => updateField('colonReplacement', v)}
        onChangeCustom={(v) => updateField('customColonReplacement', v)}
      />
      <MultiEpisodeStyleGroup
        value={form.multiEpisodeStyle ?? 'extend'}
        onChange={(v) => updateField('multiEpisodeStyle', v)}
      />
    </>
  )
}

type FormatGroupProps = {
  form: UpdateModuleNamingRequest
  updatePattern: (key: string, value: string) => void
  dynamicTokenContexts?: BackendTokenContext[]
}

function EpisodeFormatGroups({ form, updatePattern, dynamicTokenContexts }: FormatGroupProps) {
  const shared = { mediaType: 'episode' as const, tokenContext: 'episode', moduleId: MODULE_ID, dynamicTokenContexts }
  return (
    <>
      <PatternEditor label="Standard Episode Format" value={form.patterns?.['episode-file.standard'] ?? ''} onChange={(v) => updatePattern('episode-file.standard', v)} description="For regular TV series." contextName="episode-file.standard" {...shared} />
      <PatternEditor label="Daily Episode Format" value={form.patterns?.['episode-file.daily'] ?? ''} onChange={(v) => updatePattern('episode-file.daily', v)} description="For daily, date-based shows." contextName="episode-file.daily" {...shared} />
      <PatternEditor label="Anime Episode Format" value={form.patterns?.['episode-file.anime'] ?? ''} onChange={(v) => updatePattern('episode-file.anime', v)} description="For anime series." contextName="episode-file.anime" {...shared} />
    </>
  )
}

function FolderFormatGroups({ form, updatePattern, dynamicTokenContexts }: FormatGroupProps) {
  const shared = { mediaType: 'folder' as const, moduleId: MODULE_ID, dynamicTokenContexts }
  return (
    <>
      <PatternEditor label="Series Folder Format" value={form.patterns?.['series-folder'] ?? ''} onChange={(v) => updatePattern('series-folder', v)} description="Root folder for each series." tokenContext="series-folder" contextName="series-folder" {...shared} />
      <PatternEditor label="Season Folder Format" value={form.patterns?.['season-folder'] ?? ''} onChange={(v) => updatePattern('season-folder', v)} description="Subfolder for each season." tokenContext="season-folder" contextName="season-folder" {...shared} />
      <PatternEditor label="Specials Folder Format" value={form.patterns?.['specials-folder'] ?? ''} onChange={(v) => updatePattern('specials-folder', v)} description="Folder for specials (Season 0)." tokenContext="series-folder" contextName="specials-folder" {...shared} />
    </>
  )
}

function TvNamingContent({ settings }: { settings: ModuleNamingSettings }) {
  const { form, updateField, updatePattern, isSaving } = useTvNamingForm(settings)
  const dynamicTokenContexts = settings.tokenContexts

  return (
    <>
      <div className="px-screen mb-3 flex justify-end">
        <span className="text-footnote flex items-center gap-2 text-muted-foreground">
          <Save className={`size-4 ${isSaving ? 'animate-pulse' : ''}`} />
          {isSaving ? 'Saving...' : 'Auto-save'}
        </span>
      </div>
      <FilenameTester mediaType="tv" />
      <EpisodeRenamingGroups form={form} updateField={updateField} />
      <EpisodeFormatGroups form={form} updatePattern={updatePattern} dynamicTokenContexts={dynamicTokenContexts} />
      <FolderFormatGroups form={form} updatePattern={updatePattern} dynamicTokenContexts={dynamicTokenContexts} />
    </>
  )
}

export function TvNamingTab() {
  const { data: settings, isLoading, isError, refetch } = useModuleNamingSettings(MODULE_ID)

  if (isLoading) {
    return <SectionLoading count={3} />
  }
  if (isError || !settings) {
    return <SectionError onRetry={refetch} />
  }

  return <TvNamingContent settings={settings} />
}
