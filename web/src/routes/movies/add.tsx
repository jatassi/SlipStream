import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Search, Check } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { PosterImage } from '@/components/media/PosterImage'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { useMovieSearch, useQualityProfiles, useRootFoldersByType, useAddMovie, useDebounce } from '@/hooks'
import { toast } from 'sonner'
import type { MovieSearchResult, AddMovieInput } from '@/types'

type Step = 'search' | 'configure'

export function AddMoviePage() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>('search')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMovie, setSelectedMovie] = useState<MovieSearchResult | null>(null)
  const debouncedSearchQuery = useDebounce(searchQuery, 900)

  // Form state
  const [rootFolderId, setRootFolderId] = useState<string>('')
  const [qualityProfileId, setQualityProfileId] = useState<string>('')
  const [monitored, setMonitored] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState(true)

  const { data: searchResults, isLoading: searching } = useMovieSearch(debouncedSearchQuery)
  const { data: rootFolders } = useRootFoldersByType('movie')
  const { data: qualityProfiles } = useQualityProfiles()
  const addMutation = useAddMovie()

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
      rootFolderId: parseInt(rootFolderId),
      qualityProfileId: parseInt(qualityProfileId),
      monitored,
      posterUrl: selectedMovie.posterUrl,
      backdropUrl: selectedMovie.backdropUrl,
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
        breadcrumbs={[
          { label: 'Movies', href: '/movies' },
          { label: 'Add' },
        ]}
        actions={
          <Button variant="ghost" onClick={handleBack}>
            <ArrowLeft className="size-4 mr-2" />
            Back
          </Button>
        }
      />

      {step === 'search' && (
        <div className="space-y-6">
          {/* Search input */}
          <div className="max-w-xl">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
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
          ) : !searchResults?.length ? (
            <EmptyState
              icon={<Search className="size-8" />}
              title="No results found"
              description="Try a different search term"
            />
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {searchResults.map((movie) => (
                <Card
                  key={movie.tmdbId || movie.id}
                  className="cursor-pointer hover:border-primary transition-colors"
                  onClick={() => handleSelectMovie(movie)}
                >
                  <div className="aspect-[2/3] relative">
                    <PosterImage
                      url={movie.posterUrl}
                      alt={movie.title}
                      type="movie"
                      className="absolute inset-0 rounded-t-lg"
                    />
                  </div>
                  <CardContent className="p-3">
                    <h3 className="font-semibold truncate">{movie.title}</h3>
                    <p className="text-sm text-muted-foreground">
                      {movie.year || 'Unknown year'}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {step === 'configure' && selectedMovie && (
        <div className="max-w-2xl space-y-6">
          {/* Selected movie preview */}
          <Card>
            <CardContent className="p-4 flex gap-4">
              <PosterImage
                url={selectedMovie.posterUrl}
                alt={selectedMovie.title}
                type="movie"
                className="w-24 h-36 rounded shrink-0"
              />
              <div>
                <h2 className="text-xl font-semibold">{selectedMovie.title}</h2>
                <p className="text-muted-foreground">
                  {selectedMovie.year || 'Unknown year'}
                </p>
                {selectedMovie.overview && (
                  <p className="text-sm text-muted-foreground mt-2 line-clamp-3">
                    {selectedMovie.overview}
                  </p>
                )}
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
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {rootFolders?.map((folder) => (
                      <SelectItem key={folder.id} value={String(folder.id)}>
                        {folder.path}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="qualityProfile">Quality Profile *</Label>
                <Select value={qualityProfileId} onValueChange={(v) => v && setQualityProfileId(v)}>
                  <SelectTrigger>
                    <SelectValue />
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
                  <p className="text-sm text-muted-foreground">
                    Automatically search for and download releases
                  </p>
                </div>
                <Switch checked={monitored} onCheckedChange={setMonitored} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Search on Add</Label>
                  <p className="text-sm text-muted-foreground">
                    Start searching for releases immediately
                  </p>
                </div>
                <Switch checked={searchOnAdd} onCheckedChange={setSearchOnAdd} />
              </div>
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex gap-2 justify-end">
            <Button variant="outline" onClick={handleBack}>
              Back
            </Button>
            <Button
              onClick={handleAdd}
              disabled={!rootFolderId || !qualityProfileId || addMutation.isPending}
            >
              <Check className="size-4 mr-2" />
              Add Movie
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
