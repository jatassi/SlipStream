import { useState } from 'react'

import { toast } from 'sonner'

import { useModuleNamingSettings, useUpdateModuleNamingSettings } from '@/hooks'
import { getEnabledModules } from '@/modules'
import type { MissingTokenInfo, ModuleNamingSettings } from '@/types'

type NamingFormats = {
  standardEpisodeFormat: string
  dailyEpisodeFormat: string
  animeEpisodeFormat: string
  movieFileFormat: string
}

type UseResolveNamingModalParams = {
  open: boolean
  onOpenChange: (open: boolean) => void
  missingMovieTokens?: MissingTokenInfo[]
  missingEpisodeTokens?: MissingTokenInfo[]
  onResolved: () => void
}

function computeMissingTokens(format: string, requiredTokens: string[]) {
  return requiredTokens.filter((token) => !format.includes(token))
}

function computeAllMissing(
  form: Partial<NamingFormats>,
  episodeTokens: string[],
  movieTokens: string[],
) {
  const missingInStandard = computeMissingTokens(form.standardEpisodeFormat ?? '', episodeTokens)
  const missingInDaily = computeMissingTokens(form.dailyEpisodeFormat ?? '', episodeTokens)
  const missingInAnime = computeMissingTokens(form.animeEpisodeFormat ?? '', episodeTokens)
  const missingInMovie = computeMissingTokens(form.movieFileFormat ?? '', movieTokens)

  const stillMissingTokens = new Set<string>([
    ...missingInStandard,
    ...missingInDaily,
    ...missingInAnime,
    ...missingInMovie,
  ])

  return {
    missingInStandard,
    missingInDaily,
    missingInAnime,
    missingInMovie,
    stillMissingTokens,
    allResolved:
      missingInStandard.length === 0 &&
      missingInDaily.length === 0 &&
      missingInAnime.length === 0 &&
      missingInMovie.length === 0,
  }
}

function buildFormFromSettings(settingsMap: Map<string, ModuleNamingSettings>): Partial<NamingFormats> {
  const movie = settingsMap.get('movie')
  const tv = settingsMap.get('tv')
  return {
    standardEpisodeFormat: tv?.patterns['episode-file.standard'] ?? '',
    dailyEpisodeFormat: tv?.patterns['episode-file.daily'] ?? '',
    animeEpisodeFormat: tv?.patterns['episode-file.anime'] ?? '',
    movieFileFormat: movie?.patterns['movie-file'] ?? '',
  }
}

function buildModuleUpdate(settings: ModuleNamingSettings, form: Partial<NamingFormats>, moduleId: string) {
  const base = {
    renameEnabled: settings.renameEnabled,
    colonReplacement: settings.colonReplacement,
    customColonReplacement: settings.customColonReplacement,
  }

  if (moduleId === 'movie') {
    return {
      ...base,
      patterns: { ...settings.patterns, 'movie-file': form.movieFileFormat ?? settings.patterns['movie-file'] },
    }
  }

  if (moduleId === 'tv') {
    return {
      ...base,
      patterns: {
        ...settings.patterns,
        'episode-file.standard': form.standardEpisodeFormat ?? settings.patterns['episode-file.standard'],
        'episode-file.daily': form.dailyEpisodeFormat ?? settings.patterns['episode-file.daily'],
        'episode-file.anime': form.animeEpisodeFormat ?? settings.patterns['episode-file.anime'],
      },
    }
  }

  return { ...base, patterns: { ...settings.patterns } }
}

function useModuleNamingQueries() {
  const modules = getEnabledModules()
  const queries = modules.map((mod) => ({
    moduleId: mod.id,
    // eslint-disable-next-line react-hooks/rules-of-hooks
    query: useModuleNamingSettings(mod.id),
    // eslint-disable-next-line react-hooks/rules-of-hooks
    mutation: useUpdateModuleNamingSettings(mod.id),
  }))

  const allLoaded = queries.every((q) => q.query.data !== undefined)
  const settingsMap = new Map<string, ModuleNamingSettings>()
  for (const q of queries) {
    if (q.query.data) {
      settingsMap.set(q.moduleId, q.query.data)
    }
  }

  return { queries, allLoaded, settingsMap }
}

export function useResolveNamingModal({
  open,
  onOpenChange,
  missingMovieTokens,
  missingEpisodeTokens,
  onResolved,
}: UseResolveNamingModalParams) {
  const { queries, allLoaded, settingsMap } = useModuleNamingQueries()

  const [form, setForm] = useState<Partial<NamingFormats>>({})
  const [saving, setSaving] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevSettingsMap, setPrevSettingsMap] = useState(settingsMap)

  const settingsChanged = prevSettingsMap.size !== settingsMap.size ||
    [...settingsMap.entries()].some(([k, v]) => prevSettingsMap.get(k) !== v)

  if (open !== prevOpen || settingsChanged) {
    setPrevOpen(open)
    setPrevSettingsMap(settingsMap)
    if (open && allLoaded) {
      setForm(buildFormFromSettings(settingsMap))
    }
  }

  const handleSave = async () => {
    if (!allLoaded) { return }
    setSaving(true)
    try {
      await Promise.all(
        queries
          .filter((q): q is typeof q & { query: { data: ModuleNamingSettings } } => q.query.data !== undefined)
          .map((q) => q.mutation.mutateAsync(buildModuleUpdate(q.query.data, form, q.moduleId))),
      )
      toast.success('Naming formats updated')
      onOpenChange(false)
      onResolved()
    } catch {
      toast.error('Failed to update naming formats')
    } finally {
      setSaving(false)
    }
  }

  const updateField = (field: keyof NamingFormats, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  const episodeTokens = (missingEpisodeTokens ?? []).map((t) => t.suggestedToken)
  const movieTokens = (missingMovieTokens ?? []).map((t) => t.suggestedToken)

  return { form, saving, handleSave, updateField, ...computeAllMissing(form, episodeTokens, movieTokens) }
}
