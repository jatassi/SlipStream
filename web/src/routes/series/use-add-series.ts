import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'
import { z } from 'zod'

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

const addSeriesSchema = z.object({
  rootFolderId: z.string().min(1),
  qualityProfileId: z.string().min(1),
  monitorOnAdd: z.string(),
  searchOnAdd: z.string(),
  seasonFolder: z.boolean(),
  includeSpecials: z.boolean(),
})

type AddSeriesFormData = z.infer<typeof addSeriesSchema>

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
    form,
    ...handlers,
  }
}

export type AddSeriesState = ReturnType<typeof useAddSeriesPage>

function useNavigation() {
  const navigate = useNavigate()
  const searchParams: Record<string, unknown> = useSearch({ strict: false })
  const rawTmdbId = searchParams.tmdbId
  const tmdbId = rawTmdbId ? Number(rawTmdbId) : undefined

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
  const { data: qualityProfiles } = useQualityProfiles('tv')
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
  return useForm<AddSeriesFormData>({
    resolver: zodResolver(addSeriesSchema),
    defaultValues: {
      rootFolderId: '',
      qualityProfileId: '',
      seasonFolder: true,
      monitorOnAdd: 'future',
      searchOnAdd: 'no',
      includeSpecials: false,
    },
  })
}

type FormState = ReturnType<typeof useFormState>

function useSyncPreferences(
  addFlowPreferences: Queries['addFlowPreferences'],
  form: FormState,
) {
  const [prefsSynced, setPrefsSynced] = useState(false)
  const [prev, setPrev] = useState(addFlowPreferences)
  if (addFlowPreferences && addFlowPreferences !== prev) {
    setPrev(addFlowPreferences)
    if (!prefsSynced) {
      setPrefsSynced(true)
      form.setValue('monitorOnAdd', addFlowPreferences.seriesMonitorOnAdd)
      form.setValue('searchOnAdd', addFlowPreferences.seriesSearchOnAdd)
      form.setValue('includeSpecials', addFlowPreferences.seriesIncludeSpecials)
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
    if (defaultRootFolder?.exists && defaultRootFolder.defaultEntry?.entityId && !form.getValues('rootFolderId')) {
      form.setValue('rootFolderId', String(defaultRootFolder.defaultEntry.entityId))
    }
  }
}

type HandlerDeps = { nav: Navigation; selected: SelectedSeries; form: FormState; queries: Queries }

function useHandlers({ nav, selected, form, queries }: HandlerDeps) {
  const handleBack = () => {
    void nav.navigate({ to: '/series' })
  }

  const handleAdd = async () => {
    const values = form.getValues()
    if (!selected.selectedSeries || !values.rootFolderId || !values.qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input = buildAddInput(selected.selectedSeries, values)

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

function buildAddInput(series: SeriesSearchResult, data: AddSeriesFormData): AddSeriesInput {
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
    rootFolderId: Number.parseInt(data.rootFolderId),
    qualityProfileId: Number.parseInt(data.qualityProfileId),
    monitored: data.monitorOnAdd !== 'none',
    seasonFolder: data.seasonFolder,
    posterUrl: series.posterUrl,
    backdropUrl: series.backdropUrl,
    searchOnAdd: data.searchOnAdd as SeriesSearchOnAdd,
    monitorOnAdd: data.monitorOnAdd as SeriesMonitorOnAdd,
    includeSpecials: data.includeSpecials,
  }
}
