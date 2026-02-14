import { useMemo, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { ArrowLeft, Check, Loader2, Search } from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/EmptyState'
import { LoadingState } from '@/components/data/LoadingState'
import { PageHeader } from '@/components/layout/PageHeader'
import { PosterImage } from '@/components/media/PosterImage'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
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

export function AddMoviePage() {
  const navigate = useNavigate()
  // Get tmdbId from URL search params
  const searchParams = useSearch({ strict: false })
  const tmdbId = useMemo(() => {
    const id = searchParams.tmdbId
    return id ? Number(id) : undefined
  }, [searchParams.tmdbId])

  const [step, setStep] = useState<Step>(tmdbId ? 'configure' : 'search')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMovie, setSelectedMovie] = useState<MovieSearchResult | null>(null)
  const debouncedSearchQuery = useDebounce(searchQuery, 900)

  // Fetch movie metadata if tmdbId is provided
  const { data: movieMetadata, isLoading: loadingMetadata } = useMovieMetadata(tmdbId || 0)

  // Track previous values for render-time state sync
  const [prevMovieMetadata, setPrevMovieMetadata] = useState(movieMetadata)

  // Auto-select movie when metadata is loaded (React-recommended pattern)
  if (tmdbId && movieMetadata && movieMetadata !== prevMovieMetadata && !selectedMovie) {
    setPrevMovieMetadata(movieMetadata)
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
    setStep('configure')
  }

  // Form state
  const [rootFolderId, setRootFolderId] = useState<string>('')
  const [qualityProfileId, setQualityProfileId] = useState<string>('')
  const [monitored, setMonitored] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState<boolean | undefined>(undefined)

  const { data: searchResults, isLoading: searching } = useMovieSearch(debouncedSearchQuery)
  const { data: rootFolders } = useRootFoldersByType('movie')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'movie')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddMovie()

  // Track previous values for render-time state sync
  const [prevAddFlowPreferences, setPrevAddFlowPreferences] = useState(addFlowPreferences)
  const [prevDefaultRootFolder, setPrevDefaultRootFolder] = useState(defaultRootFolder)

  // Initialize searchOnAdd from preferences (React-recommended pattern)
  if (
    addFlowPreferences &&
    addFlowPreferences !== prevAddFlowPreferences &&
    searchOnAdd === undefined
  ) {
    setPrevAddFlowPreferences(addFlowPreferences)
    setSearchOnAdd(addFlowPreferences.movieSearchOnAdd)
  }

  // Pre-populate root folder with default (React-recommended pattern)
  if (defaultRootFolder !== prevDefaultRootFolder) {
    setPrevDefaultRootFolder(defaultRootFolder)
    if (defaultRootFolder?.exists && defaultRootFolder.defaultEntry?.entityId && !rootFolderId) {
      setRootFolderId(String(defaultRootFolder.defaultEntry.entityId))
    }
  }

  const handleSelectMovie = (movie: MovieSearchResult) => {
    setSelectedMovie(movie)
    setStep('configure')
  }

  const handleBack = () => {
    if (step === 'configure') {
      setStep('search')
      setSelectedMovie(null)
    } else {
      navigate({ to: '/movies' })
    }
  }

  const handleAdd = async () => {
    if (!selectedMovie || !rootFolderId || !qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input: AddMovieInput = {
      title: selectedMovie.title,
      year: selectedMovie.year,
      tmdbId: selectedMovie.tmdbId,
      imdbId: selectedMovie.imdbId,
      overview: selectedMovie.overview,
      runtime: selectedMovie.runtime,
      rootFolderId: Number.parseInt(rootFolderId),
      qualityProfileId: Number.parseInt(qualityProfileId),
      monitored,
      posterUrl: selectedMovie.posterUrl,
      backdropUrl: selectedMovie.backdropUrl,
      searchOnAdd: searchOnAdd ?? false,
    }

    try {
      const movie = await addMutation.mutateAsync(input)
      toast.success(`Added "${movie.title}"`)
      navigate({ to: '/movies/$id', params: { id: String(movie.id) } })
    } catch {
      toast.error('Failed to add movie')
    }
  }

  return (
    <div>
      <PageHeader
        title="Add Movie"
        breadcrumbs={[{ label: 'Movies', href: '/movies' }, { label: 'Add' }]}
        actions={
          <Button variant="ghost" onClick={handleBack}>
            <ArrowLeft className="mr-2 size-4" />
            Back
          </Button>
        }
      />

      {/* Loading state when fetching by tmdbId */}
      {tmdbId && loadingMetadata ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="text-muted-foreground size-8 animate-spin" />
        </div>
      ) : null}

      {step === 'search' && !tmdbId && (
        <div className="space-y-6">
          {/* Search input */}
          <div className="max-w-xl">
            <div className="relative">
              <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
              <Input
                placeholder="Search for a movie..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
                autoFocus
              />
            </div>
          </div>

          {/* Results */}
          {searching ? (
            <LoadingState count={4} />
          ) : searchQuery.length < 2 ? (
            <EmptyState
              icon={<Search className="size-8" />}
              title="Search for a movie"
              description="Enter at least 2 characters to search"
            />
          ) : searchResults?.length ? (
            <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {searchResults.map((movie) => (
                <Card
                  key={movie.tmdbId || movie.id}
                  className="hover:border-primary cursor-pointer transition-colors"
                  onClick={() => handleSelectMovie(movie)}
                >
                  <div className="relative aspect-[2/3]">
                    <PosterImage
                      url={movie.posterUrl}
                      alt={movie.title}
                      type="movie"
                      className="absolute inset-0 rounded-t-lg"
                    />
                  </div>
                  <CardContent className="p-3">
                    <h3 className="truncate font-semibold">{movie.title}</h3>
                    <p className="text-muted-foreground text-sm">{movie.year || 'Unknown year'}</p>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<Search className="size-8" />}
              title="No results found"
              description="Try a different search term"
            />
          )}
        </div>
      )}

      {step === 'configure' && selectedMovie ? (
        <div className="max-w-2xl space-y-6">
          {/* Selected movie preview */}
          <Card>
            <CardContent className="flex gap-4 p-4">
              <PosterImage
                url={selectedMovie.posterUrl}
                alt={selectedMovie.title}
                type="movie"
                className="h-36 w-24 shrink-0 rounded"
              />
              <div>
                <h2 className="text-xl font-semibold">{selectedMovie.title}</h2>
                <p className="text-muted-foreground">{selectedMovie.year || 'Unknown year'}</p>
                {selectedMovie.overview ? (
                  <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
                    {selectedMovie.overview}
                  </p>
                ) : null}
              </div>
            </CardContent>
          </Card>

          {/* Configuration form */}
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="rootFolder">Root Folder *</Label>
                <Select value={rootFolderId} onValueChange={(v) => v && setRootFolderId(v)}>
                  <SelectTrigger>
                    <SelectValue>
                      {(rootFolderId &&
                        rootFolders?.find((f) => f.id === Number.parseInt(rootFolderId))?.name) ||
                        'Select a root folder'}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {rootFolders?.map((folder) => (
                      <SelectItem key={folder.id} value={String(folder.id)}>
                        {folder.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="qualityProfile">Quality Profile *</Label>
                <Select value={qualityProfileId} onValueChange={(v) => v && setQualityProfileId(v)}>
                  <SelectTrigger>
                    <SelectValue>
                      {(qualityProfileId &&
                        qualityProfiles?.find((p) => p.id === Number.parseInt(qualityProfileId))
                          ?.name) ||
                        'Select a quality profile'}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {qualityProfiles?.map((profile) => (
                      <SelectItem key={profile.id} value={String(profile.id)}>
                        {profile.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Monitored</Label>
                  <p className="text-muted-foreground text-sm">
                    Automatically search for and download releases
                  </p>
                </div>
                <Switch checked={monitored} onCheckedChange={setMonitored} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Search on Add</Label>
                  <p className="text-muted-foreground text-sm">
                    Start searching for releases immediately
                  </p>
                </div>
                <Switch checked={searchOnAdd ?? false} onCheckedChange={setSearchOnAdd} />
              </div>
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={handleBack}>
              Back
            </Button>
            <Button
              onClick={handleAdd}
              disabled={!rootFolderId || !qualityProfileId || addMutation.isPending}
            >
              <Check className="mr-2 size-4" />
              Add Movie
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
