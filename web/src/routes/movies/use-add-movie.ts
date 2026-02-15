import { useEffect, useMemo, useRef, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useAddFlowPreferences,
  useAddMovie,
  useDebounce,
  useDefault,
  useMovieMetadata,
  useMovieSearch,
  useQualityProfiles,
  useRootFoldersByType,
} from '@/hooks'
import type { AddMovieInput, MovieSearchResult } from '@/types'

type Step = 'search' | 'configure'

export function useAddMoviePage() {
  const nav = useNavigation()
  const search = useSearchStep(nav.tmdbId)
  const queries = useQueries(nav.tmdbId, search.debouncedSearchQuery)
  const form = useFormState()

  useSyncMetadata(nav.tmdbId, queries.movieMetadata, search)
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

export type AddMovieState = ReturnType<typeof useAddMoviePage>

function useNavigation() {
  const navigate = useNavigate()
  const searchParams: { tmdbId?: number } = useSearch({ strict: false })
  const tmdbId = useMemo(() => searchParams.tmdbId, [searchParams.tmdbId])

  return { navigate, tmdbId }
}

type Navigation = ReturnType<typeof useNavigation>

function useSearchStep(tmdbId: number | undefined) {
  const [step, setStep] = useState<Step>(tmdbId ? 'configure' : 'search')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMovie, setSelectedMovie] = useState<MovieSearchResult | null>(null)
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
    selectedMovie, setSelectedMovie,
    debouncedSearchQuery,
    searchInputRef,
  }
}

type SearchStep = ReturnType<typeof useSearchStep>

function useQueries(tmdbId: number | undefined, debouncedSearchQuery: string) {
  const { data: movieMetadata, isLoading: loadingMetadata } = useMovieMetadata(tmdbId ?? 0)
  const { data: searchResults, isLoading: searching } = useMovieSearch(debouncedSearchQuery)
  const { data: rootFolders } = useRootFoldersByType('movie')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'movie')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddMovie()

  return {
    movieMetadata, loadingMetadata,
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
  const [monitored, setMonitored] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState<boolean | undefined>(undefined)

  return {
    rootFolderId, setRootFolderId,
    qualityProfileId, setQualityProfileId,
    monitored, setMonitored,
    searchOnAdd, setSearchOnAdd,
  }
}

type FormState = ReturnType<typeof useFormState>

function useSyncMetadata(
  tmdbId: number | undefined,
  movieMetadata: Queries['movieMetadata'],
  search: SearchStep,
) {
  const [prev, setPrev] = useState(movieMetadata)
  if (tmdbId && movieMetadata && movieMetadata !== prev && !search.selectedMovie) {
    setPrev(movieMetadata)
    search.setSelectedMovie({
      id: movieMetadata.id,
      tmdbId: movieMetadata.tmdbId,
      imdbId: movieMetadata.imdbId,
      title: movieMetadata.title,
      originalTitle: movieMetadata.originalTitle,
      year: movieMetadata.year,
      overview: movieMetadata.overview,
      posterUrl: movieMetadata.posterUrl,
      backdropUrl: movieMetadata.backdropUrl,
      runtime: movieMetadata.runtime,
      genres: movieMetadata.genres,
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
    if (form.searchOnAdd === undefined) {
      form.setSearchOnAdd(addFlowPreferences.movieSearchOnAdd)
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

function useHandlers({ nav, search, form, queries }: {
  nav: Navigation
  search: SearchStep
  form: FormState
  queries: Queries
}) {
  const handleSelectMovie = (movie: MovieSearchResult) => {
    search.setSelectedMovie(movie)
    search.setStep('configure')
  }

  const handleBack = () => {
    if (search.step === 'configure') {
      search.setStep('search')
      search.setSelectedMovie(null)
    } else {
      void nav.navigate({ to: '/movies' })
    }
  }

  const handleAdd = async () => {
    if (!search.selectedMovie || !form.rootFolderId || !form.qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input: AddMovieInput = {
      title: search.selectedMovie.title,
      year: search.selectedMovie.year,
      tmdbId: search.selectedMovie.tmdbId,
      imdbId: search.selectedMovie.imdbId,
      overview: search.selectedMovie.overview,
      runtime: search.selectedMovie.runtime,
      rootFolderId: Number.parseInt(form.rootFolderId),
      qualityProfileId: Number.parseInt(form.qualityProfileId),
      monitored: form.monitored,
      posterUrl: search.selectedMovie.posterUrl,
      backdropUrl: search.selectedMovie.backdropUrl,
      searchOnAdd: form.searchOnAdd ?? false,
    }

    try {
      const movie = await queries.addMutation.mutateAsync(input)
      toast.success(`Added "${movie.title}"`)
      void nav.navigate({ to: '/movies/$id', params: { id: String(movie.id) } })
    } catch {
      toast.error('Failed to add movie')
    }
  }

  return { handleSelectMovie, handleBack, handleAdd }
}
