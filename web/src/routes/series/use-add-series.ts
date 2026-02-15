import { useMemo, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useAddFlowPreferences,
  useAddSeries,
  useDefault,
  useQualityProfiles,
  useRootFoldersByType,
  useSeriesMetadata,
} from '@/hooks'
import type {
  AddSeriesInput,
  SeriesMonitorOnAdd,
  SeriesSearchOnAdd,
  SeriesSearchResult,
} from '@/types'

export function useAddSeriesPage() {
  const nav = useNavigation()
  const selected = useSelectedSeries(nav.tmdbId)
  const queries = useQueries()
  const form = useFormState()

  useSyncPreferences(queries.addFlowPreferences, form)
  useSyncDefaultRootFolder(queries.defaultRootFolder, form)

  const handlers = useHandlers({ nav, selected, form, queries })

  return {
    ...nav,
    ...selected,
    ...queries,
    ...form,
    ...handlers,
  }
}

export type AddSeriesState = ReturnType<typeof useAddSeriesPage>

function useNavigation() {
  const navigate = useNavigate()
  const searchParams: Record<string, unknown> = useSearch({ strict: false })
  const rawTmdbId = searchParams.tmdbId
  const tmdbId = useMemo(() => (rawTmdbId ? Number(rawTmdbId) : undefined), [rawTmdbId])

  return { navigate, tmdbId }
}

type Navigation = ReturnType<typeof useNavigation>

function useSelectedSeries(tmdbId: number | undefined) {
  const [selectedSeries, setSelectedSeries] = useState<SeriesSearchResult | null>(null)
  const { data: seriesMetadata, isLoading: loadingMetadata } = useSeriesMetadata(tmdbId ?? 0)

  const [prev, setPrev] = useState(seriesMetadata)
  if (tmdbId && seriesMetadata && seriesMetadata !== prev && !selectedSeries) {
    setPrev(seriesMetadata)
    setSelectedSeries({
      id: seriesMetadata.id,
      tmdbId: seriesMetadata.tmdbId,
      tvdbId: seriesMetadata.tvdbId,
      imdbId: seriesMetadata.imdbId,
      title: seriesMetadata.title,
      originalTitle: seriesMetadata.originalTitle,
      year: seriesMetadata.year,
      overview: seriesMetadata.overview,
      posterUrl: seriesMetadata.posterUrl,
      backdropUrl: seriesMetadata.backdropUrl,
      runtime: seriesMetadata.runtime,
      genres: seriesMetadata.genres,
      status: seriesMetadata.status,
      network: seriesMetadata.network,
      networkLogoUrl: seriesMetadata.networkLogoUrl,
    })
  }

  return { selectedSeries, loadingMetadata }
}

type SelectedSeries = ReturnType<typeof useSelectedSeries>

function useQueries() {
  const { data: rootFolders } = useRootFoldersByType('tv')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'tv')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddSeries()

  return {
    rootFolders, qualityProfiles,
    defaultRootFolder, addFlowPreferences,
    addMutation,
    isPending: addMutation.isPending,
  }
}

type Queries = ReturnType<typeof useQueries>

function useFormState() {
  const [rootFolderId, setRootFolderId] = useState<string>('')
  const [qualityProfileId, setQualityProfileId] = useState<string>('')
  const [seasonFolder, setSeasonFolder] = useState(true)
  const [monitorOnAdd, setMonitorOnAdd] = useState<SeriesMonitorOnAdd | undefined>(undefined)
  const [searchOnAdd, setSearchOnAdd] = useState<SeriesSearchOnAdd | undefined>(undefined)
  const [includeSpecials, setIncludeSpecials] = useState<boolean | undefined>(undefined)

  return {
    rootFolderId, setRootFolderId,
    qualityProfileId, setQualityProfileId,
    seasonFolder, setSeasonFolder,
    monitorOnAdd, setMonitorOnAdd,
    searchOnAdd, setSearchOnAdd,
    includeSpecials, setIncludeSpecials,
  }
}

type FormState = ReturnType<typeof useFormState>

function useSyncPreferences(
  addFlowPreferences: Queries['addFlowPreferences'],
  form: FormState,
) {
  const [prev, setPrev] = useState(addFlowPreferences)
  if (addFlowPreferences && addFlowPreferences !== prev) {
    setPrev(addFlowPreferences)
    if (form.monitorOnAdd === undefined) {
      form.setMonitorOnAdd(addFlowPreferences.seriesMonitorOnAdd)
    }
    if (form.searchOnAdd === undefined) {
      form.setSearchOnAdd(addFlowPreferences.seriesSearchOnAdd)
    }
    if (form.includeSpecials === undefined) {
      form.setIncludeSpecials(addFlowPreferences.seriesIncludeSpecials)
    }
  }
}

function useSyncDefaultRootFolder(
  defaultRootFolder: Queries['defaultRootFolder'],
  form: FormState,
) {
  const [prev, setPrev] = useState(defaultRootFolder)
  if (defaultRootFolder !== prev) {
    setPrev(defaultRootFolder)
    if (defaultRootFolder?.exists && defaultRootFolder.defaultEntry?.entityId && !form.rootFolderId) {
      form.setRootFolderId(String(defaultRootFolder.defaultEntry.entityId))
    }
  }
}

type HandlerDeps = { nav: Navigation; selected: SelectedSeries; form: FormState; queries: Queries }

function useHandlers({ nav, selected, form, queries }: HandlerDeps) {
  const handleBack = () => {
    void nav.navigate({ to: '/series' })
  }

  const handleAdd = async () => {
    if (!selected.selectedSeries || !form.rootFolderId || !form.qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input = buildAddInput(selected.selectedSeries, form)

    try {
      const series = await queries.addMutation.mutateAsync(input)
      toast.success(`Added "${series.title}"`)
      void nav.navigate({ to: '/series/$id', params: { id: String(series.id) } })
    } catch {
      toast.error('Failed to add series')
    }
  }

  return { handleBack, handleAdd }
}

function buildAddInput(series: SeriesSearchResult, form: FormState): AddSeriesInput {
  return {
    title: series.title,
    year: series.year,
    tvdbId: series.tvdbId,
    tmdbId: series.tmdbId,
    imdbId: series.imdbId,
    overview: series.overview,
    runtime: series.runtime,
    network: series.network,
    networkLogoUrl: series.networkLogoUrl,
    rootFolderId: Number.parseInt(form.rootFolderId),
    qualityProfileId: Number.parseInt(form.qualityProfileId),
    monitored: form.monitorOnAdd !== 'none',
    seasonFolder: form.seasonFolder,
    posterUrl: series.posterUrl,
    backdropUrl: series.backdropUrl,
    searchOnAdd: form.searchOnAdd ?? 'no',
    monitorOnAdd: form.monitorOnAdd ?? 'future',
    includeSpecials: form.includeSpecials ?? false,
  }
}
