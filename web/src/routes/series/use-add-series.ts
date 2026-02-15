import { useEffect, useMemo, useRef, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useAddFlowPreferences,
  useAddSeries,
  useDebounce,
  useDefault,
  useQualityProfiles,
  useRootFoldersByType,
  useSeriesMetadata,
  useSeriesSearch,
} from '@/hooks'
import type {
  AddSeriesInput,
  SeriesMonitorOnAdd,
  SeriesSearchOnAdd,
  SeriesSearchResult,
} from '@/types'

type Step = 'search' | 'configure'

export function useAddSeriesPage() {
  const nav = useNavigation()
  const search = useSearchStep(nav.tmdbId)
  const queries = useQueries(nav.tmdbId, search.debouncedSearchQuery)
  const form = useFormState()

  useSyncMetadata(nav.tmdbId, queries.seriesMetadata, search)
  useSyncPreferences(queries.addFlowPreferences, form)
  useSyncDefaultRootFolder(queries.defaultRootFolder, form)

  const handlers = useHandlers({ nav, search, form, queries })

  return {
    ...nav,
    ...search,
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

function useSearchStep(tmdbId: number | undefined) {
  const [step, setStep] = useState<Step>(tmdbId ? 'configure' : 'search')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSeries, setSelectedSeries] = useState<SeriesSearchResult | null>(null)
  const debouncedSearchQuery = useDebounce(searchQuery, 900)
  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (step === 'search') {
      searchInputRef.current?.focus()
    }
  }, [step])

  return {
    step, setStep,
    searchQuery, setSearchQuery,
    selectedSeries, setSelectedSeries,
    debouncedSearchQuery,
    searchInputRef,
  }
}

type SearchStep = ReturnType<typeof useSearchStep>

function useQueries(tmdbId: number | undefined, debouncedSearchQuery: string) {
  const { data: seriesMetadata, isLoading: loadingMetadata } = useSeriesMetadata(tmdbId ?? 0)
  const { data: searchResults, isLoading: searching } = useSeriesSearch(debouncedSearchQuery)
  const { data: rootFolders } = useRootFoldersByType('tv')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'tv')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddSeries()

  return {
    seriesMetadata, loadingMetadata,
    searchResults, searching,
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

function useSyncMetadata(
  tmdbId: number | undefined,
  seriesMetadata: Queries['seriesMetadata'],
  search: SearchStep,
) {
  const [prev, setPrev] = useState(seriesMetadata)
  if (tmdbId && seriesMetadata && seriesMetadata !== prev && !search.selectedSeries) {
    setPrev(seriesMetadata)
    search.setSelectedSeries({
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
    search.setStep('configure')
  }
}

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

type HandlerDeps = { nav: Navigation; search: SearchStep; form: FormState; queries: Queries }

function useHandlers({ nav, search, form, queries }: HandlerDeps) {
  const handleSelectSeries = (series: SeriesSearchResult) => {
    search.setSelectedSeries(series)
    search.setStep('configure')
  }

  const handleBack = () => {
    if (search.step === 'configure') {
      search.setStep('search')
      search.setSelectedSeries(null)
    } else {
      void nav.navigate({ to: '/series' })
    }
  }

  const handleAdd = async () => {
    if (!search.selectedSeries || !form.rootFolderId || !form.qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input = buildAddInput(search.selectedSeries, form)

    try {
      const series = await queries.addMutation.mutateAsync(input)
      toast.success(`Added "${series.title}"`)
      void nav.navigate({ to: '/series/$id', params: { id: String(series.id) } })
    } catch {
      toast.error('Failed to add series')
    }
  }

  return { handleSelectSeries, handleBack, handleAdd }
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
