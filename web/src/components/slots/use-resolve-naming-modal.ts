import { useState } from 'react'

import { toast } from 'sonner'

import { useImportSettings, useUpdateImportSettings } from '@/hooks'
import type { ImportSettings, MissingTokenInfo } from '@/types'

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
  form: Partial<ImportSettings>,
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

  const allResolved =
    missingInStandard.length === 0 &&
    missingInDaily.length === 0 &&
    missingInAnime.length === 0 &&
    missingInMovie.length === 0

  return {
    missingInStandard,
    missingInDaily,
    missingInAnime,
    missingInMovie,
    stillMissingTokens,
    allResolved,
  }
}

function pickFormats(s: ImportSettings): Partial<ImportSettings> {
  return {
    standardEpisodeFormat: s.standardEpisodeFormat,
    dailyEpisodeFormat: s.dailyEpisodeFormat,
    animeEpisodeFormat: s.animeEpisodeFormat,
    movieFileFormat: s.movieFileFormat,
  }
}

export function useResolveNamingModal({
  open,
  onOpenChange,
  missingMovieTokens,
  missingEpisodeTokens,
  onResolved,
}: UseResolveNamingModalParams) {
  const { data: settings } = useImportSettings()
  const updateMutation = useUpdateImportSettings()
  const [form, setForm] = useState<Partial<ImportSettings>>({})
  const [saving, setSaving] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)

  if (open !== prevOpen || settings !== prevSettings) {
    setPrevOpen(open)
    setPrevSettings(settings)
    if (open && settings) {
      setForm(pickFormats(settings))
    }
  }

  const handleSave = async () => {
    if (!settings) {
      return
    }
    setSaving(true)
    try {
      await updateMutation.mutateAsync({ ...settings, ...form })
      toast.success('Naming formats updated')
      onOpenChange(false)
      onResolved()
    } catch {
      toast.error('Failed to update naming formats')
    } finally {
      setSaving(false)
    }
  }

  const updateField = (field: keyof ImportSettings, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  const episodeTokens = (missingEpisodeTokens ?? []).map((t) => t.suggestedToken)
  const movieTokens = (missingMovieTokens ?? []).map((t) => t.suggestedToken)

  return { form, saving, handleSave, updateField, ...computeAllMissing(form, episodeTokens, movieTokens) }
}
