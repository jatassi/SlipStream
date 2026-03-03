import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  useAddFlowPreferences,
  useAddMovie,
  useDefault,
  useMovieMetadata,
  useQualityProfiles,
  useRootFoldersByType,
} from '@/hooks'
import type { AddMovieInput, MovieSearchResult } from '@/types'

const addMovieSchema = z.object({
  rootFolderId: z.string().min(1),
  qualityProfileId: z.string().min(1),
  monitored: z.boolean(),
  searchOnAdd: z.boolean(),
})

type AddMovieFormData = z.infer<typeof addMovieSchema>

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
    form,
    ...handlers,
  }
}

function useNavigation() {
  const navigate = useNavigate()
  const searchParams: { tmdbId?: number } = useSearch({ strict: false })
  const tmdbId = searchParams.tmdbId

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
  return useForm<AddMovieFormData>({
    resolver: zodResolver(addMovieSchema),
    defaultValues: {
      rootFolderId: '',
      qualityProfileId: '',
      monitored: true,
      searchOnAdd: false,
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
      form.setValue('searchOnAdd', addFlowPreferences.movieSearchOnAdd)
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
    const values = form.getValues()
    if (!selected.selectedMovie || !values.rootFolderId || !values.qualityProfileId) {
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
      rootFolderId: Number.parseInt(values.rootFolderId),
      qualityProfileId: Number.parseInt(values.qualityProfileId),
      monitored: values.monitored,
      posterUrl: selected.selectedMovie.posterUrl,
      backdropUrl: selected.selectedMovie.backdropUrl,
      searchOnAdd: values.searchOnAdd,
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
