import { useMemo, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useAddFlowPreferences,
  useAddMovie,
  useDefault,
  useMovieMetadata,
  useQualityProfiles,
  useRootFoldersByType,
} from '@/hooks'
import type { AddMovieInput, MovieSearchResult } from '@/types'

export function useAddMoviePage() {
  const nav = useNavigation()
  const selected = useSelectedMovie(nav.tmdbId)
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

function useNavigation() {
  const navigate = useNavigate()
  const searchParams: { tmdbId?: number } = useSearch({ strict: false })
  const tmdbId = useMemo(() => searchParams.tmdbId, [searchParams.tmdbId])

  return { navigate, tmdbId }
}

type Navigation = ReturnType<typeof useNavigation>

function useSelectedMovie(tmdbId: number | undefined) {
  const [selectedMovie, setSelectedMovie] = useState<MovieSearchResult | null>(null)
  const { data: movieMetadata, isLoading: loadingMetadata } = useMovieMetadata(tmdbId ?? 0)

  const [prev, setPrev] = useState(movieMetadata)
  if (tmdbId && movieMetadata && movieMetadata !== prev && !selectedMovie) {
    setPrev(movieMetadata)
    setSelectedMovie({
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
  }

  return { selectedMovie, loadingMetadata }
}

type SelectedMovie = ReturnType<typeof useSelectedMovie>

function useQueries() {
  const { data: rootFolders } = useRootFoldersByType('movie')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'movie')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddMovie()

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

function useHandlers({ nav, selected, form, queries }: {
  nav: Navigation
  selected: SelectedMovie
  form: FormState
  queries: Queries
}) {
  const handleBack = () => {
    void nav.navigate({ to: '/movies' })
  }

  const handleAdd = async () => {
    if (!selected.selectedMovie || !form.rootFolderId || !form.qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input: AddMovieInput = {
      title: selected.selectedMovie.title,
      year: selected.selectedMovie.year,
      tmdbId: selected.selectedMovie.tmdbId,
      imdbId: selected.selectedMovie.imdbId,
      overview: selected.selectedMovie.overview,
      runtime: selected.selectedMovie.runtime,
      rootFolderId: Number.parseInt(form.rootFolderId),
      qualityProfileId: Number.parseInt(form.qualityProfileId),
      monitored: form.monitored,
      posterUrl: selected.selectedMovie.posterUrl,
      backdropUrl: selected.selectedMovie.backdropUrl,
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

  return { handleBack, handleAdd }
}
