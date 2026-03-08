import { useState } from 'react'

import { toast } from 'sonner'

import { useModuleNamingSettings, useUpdateModuleNamingSettings } from '@/hooks'
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

function buildFormFromSettings(movie: ModuleNamingSettings, tv: ModuleNamingSettings): Partial<NamingFormats> {
  return {
    standardEpisodeFormat: tv.patterns['episode-file.standard'] ?? '',
    dailyEpisodeFormat: tv.patterns['episode-file.daily'] ?? '',
    animeEpisodeFormat: tv.patterns['episode-file.anime'] ?? '',
    movieFileFormat: movie.patterns['movie-file'] ?? '',
  }
}

function buildMovieUpdate(movie: ModuleNamingSettings, form: Partial<NamingFormats>) {
  return {
    renameEnabled: movie.renameEnabled,
    colonReplacement: movie.colonReplacement,
    customColonReplacement: movie.customColonReplacement,
    patterns: { ...movie.patterns, 'movie-file': form.movieFileFormat ?? movie.patterns['movie-file'] },
  }
}

function buildTvUpdate(tv: ModuleNamingSettings, form: Partial<NamingFormats>) {
  return {
    renameEnabled: tv.renameEnabled,
    colonReplacement: tv.colonReplacement,
    customColonReplacement: tv.customColonReplacement,
    patterns: {
      ...tv.patterns,
      'episode-file.standard': form.standardEpisodeFormat ?? tv.patterns['episode-file.standard'],
      'episode-file.daily': form.dailyEpisodeFormat ?? tv.patterns['episode-file.daily'],
      'episode-file.anime': form.animeEpisodeFormat ?? tv.patterns['episode-file.anime'],
    },
  }
}

export function useResolveNamingModal({
  open,
  onOpenChange,
  missingMovieTokens,
  missingEpisodeTokens,
  onResolved,
}: UseResolveNamingModalParams) {
  const { data: movieNaming } = useModuleNamingSettings('movie')
  const { data: tvNaming } = useModuleNamingSettings('tv')
  const updateMovie = useUpdateModuleNamingSettings('movie')
  const updateTv = useUpdateModuleNamingSettings('tv')

  const [form, setForm] = useState<Partial<NamingFormats>>({})
  const [saving, setSaving] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevMovie, setPrevMovie] = useState(movieNaming)
  const [prevTv, setPrevTv] = useState(tvNaming)

  if (open !== prevOpen || movieNaming !== prevMovie || tvNaming !== prevTv) {
    setPrevOpen(open)
    setPrevMovie(movieNaming)
    setPrevTv(tvNaming)
    if (open && movieNaming && tvNaming) {
      setForm(buildFormFromSettings(movieNaming, tvNaming))
    }
  }

  const handleSave = async () => {
    if (!movieNaming || !tvNaming) { return }
    setSaving(true)
    try {
      await Promise.all([
        updateMovie.mutateAsync(buildMovieUpdate(movieNaming, form)),
        updateTv.mutateAsync(buildTvUpdate(tvNaming, form)),
      ])
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
