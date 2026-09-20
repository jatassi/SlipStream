import { useCallback, useEffect, useRef, useState } from 'react'

import { Save } from 'lucide-react'
import { toast } from 'sonner'

import { Group } from '@/components/grouped-list'
import { SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import { useModuleNamingSettings, useUpdateModuleNamingSettings } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { ModuleNamingSettings, TokenContext as BackendTokenContext, UpdateModuleNamingRequest } from '@/types'

import { FilenameTester } from './filename-tester'
import { PatternEditor } from './naming-pattern-editor'

const MODULE_ID = 'movie'

function toFormData(s: ModuleNamingSettings): UpdateModuleNamingRequest {
  return {
    renameEnabled: s.renameEnabled,
    colonReplacement: s.colonReplacement,
    customColonReplacement: s.customColonReplacement,
    patterns: { ...s.patterns },
  }
}

function useMovieNamingForm(settings: ModuleNamingSettings) {
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

function SaveIndicator({ isSaving }: { isSaving: boolean }) {
  return (
    <div className="px-screen mb-3 flex justify-end">
      <span className="text-footnote flex items-center gap-2 text-muted-foreground">
        <Save className={`size-4 ${isSaving ? 'animate-pulse' : ''}`} />
        {isSaving ? 'Saving...' : 'Auto-save'}
      </span>
    </div>
  )
}

type MovieFormProps = {
  form: UpdateModuleNamingRequest
  updateField: <K extends keyof UpdateModuleNamingRequest>(field: K, value: UpdateModuleNamingRequest[K]) => void
  updatePattern: (key: string, value: string) => void
  isSaving: boolean
  dynamicTokenContexts?: BackendTokenContext[]
}

function MovieRenamingGroups({ form, updateField, updatePattern, isSaving, dynamicTokenContexts }: MovieFormProps) {
  return (
    <>
      <SaveIndicator isSaving={isSaving} />
      <Group
        header="Movie Renaming"
        footer="Rename movie files according to the format patterns below during import."
      >
        <SwitchRow
          label="Rename Movies"
          checked={form.renameEnabled ?? false}
          onCheckedChange={(v) => updateField('renameEnabled', v)}
        />
      </Group>
      <PatternEditor
        label="Movie Folder Format"
        value={form.patterns?.['movie-folder'] ?? ''}
        onChange={(v) => updatePattern('movie-folder', v)}
        description="Folder name for each movie."
        mediaType="folder"
        tokenContext="movie-folder"
        moduleId={MODULE_ID}
        contextName="movie-folder"
        dynamicTokenContexts={dynamicTokenContexts}
      />
      <PatternEditor
        label="Movie File Format"
        value={form.patterns?.['movie-file'] ?? ''}
        onChange={(v) => updatePattern('movie-file', v)}
        description="Filename pattern for movie files."
        mediaType="movie"
        tokenContext="movie"
        moduleId={MODULE_ID}
        contextName="movie-file"
        dynamicTokenContexts={dynamicTokenContexts}
      />
    </>
  )
}

function MovieNamingContent({ settings }: { settings: ModuleNamingSettings }) {
  const { form, updateField, updatePattern, isSaving } = useMovieNamingForm(settings)
  const dynamicTokenContexts = settings.tokenContexts

  return (
    <>
      <FilenameTester mediaType="movie" />
      <MovieRenamingGroups form={form} updateField={updateField} updatePattern={updatePattern} isSaving={isSaving} dynamicTokenContexts={dynamicTokenContexts} />
    </>
  )
}

export function MovieNamingTab() {
  const { data: settings, isLoading, isError, refetch } = useModuleNamingSettings(MODULE_ID)

  if (isLoading) {
    return <SectionLoading count={2} />
  }
  if (isError || !settings) {
    return <SectionError onRetry={refetch} />
  }

  return <MovieNamingContent settings={settings} />
}
