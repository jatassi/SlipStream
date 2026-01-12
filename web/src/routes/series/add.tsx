import { useState, useEffect, useMemo } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { ArrowLeft, Search, Check, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { PosterImage } from '@/components/media/PosterImage'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { useSeriesSearch, useSeriesMetadata, useQualityProfiles, useRootFoldersByType, useAddSeries, useDefault, useDebounce, useAddFlowPreferences } from '@/hooks'
import { toast } from 'sonner'
import type { SeriesSearchResult, AddSeriesInput, SeriesSearchOnAdd, SeriesMonitorOnAdd } from '@/types'

type Step = 'search' | 'configure'

export function AddSeriesPage() {
  const navigate = useNavigate()
  // Get tmdbId from URL search params
  const searchParams = useSearch({ strict: false }) as { tmdbId?: string }
  const tmdbId = useMemo(() => {
    const id = searchParams.tmdbId
    return id ? Number(id) : undefined
  }, [searchParams.tmdbId])

  const [step, setStep] = useState<Step>(tmdbId ? 'configure' : 'search')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSeries, setSelectedSeries] = useState<SeriesSearchResult | null>(null)
  const debouncedSearchQuery = useDebounce(searchQuery, 900)

  // Fetch series metadata if tmdbId is provided
  const { data: seriesMetadata, isLoading: loadingMetadata } = useSeriesMetadata(tmdbId || 0)

  // Auto-select series when metadata is loaded
  useEffect(() => {
    if (tmdbId && seriesMetadata && !selectedSeries) {
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
      })
      setStep('configure')
    }
  }, [tmdbId, seriesMetadata, selectedSeries])

  // Form state
  const [rootFolderId, setRootFolderId] = useState<string>('')
  const [qualityProfileId, setQualityProfileId] = useState<string>('')
  const [seasonFolder, setSeasonFolder] = useState(true)
  const [monitorOnAdd, setMonitorOnAdd] = useState<SeriesMonitorOnAdd | undefined>(undefined)
  const [searchOnAdd, setSearchOnAdd] = useState<SeriesSearchOnAdd | undefined>(undefined)
  const [includeSpecials, setIncludeSpecials] = useState<boolean | undefined>(undefined)

  const { data: searchResults, isLoading: searching } = useSeriesSearch(debouncedSearchQuery)
  const { data: rootFolders } = useRootFoldersByType('tv')
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: defaultRootFolder } = useDefault('root_folder', 'tv')
  const { data: addFlowPreferences } = useAddFlowPreferences()
  const addMutation = useAddSeries()

  // Initialize from preferences
  useEffect(() => {
    if (addFlowPreferences) {
      if (monitorOnAdd === undefined) {
        setMonitorOnAdd(addFlowPreferences.seriesMonitorOnAdd)
      }
      if (searchOnAdd === undefined) {
        setSearchOnAdd(addFlowPreferences.seriesSearchOnAdd)
      }
      if (includeSpecials === undefined) {
        setIncludeSpecials(addFlowPreferences.seriesIncludeSpecials)
      }
    }
  }, [addFlowPreferences, monitorOnAdd, searchOnAdd, includeSpecials])

  // Pre-populate root folder with default
  useEffect(() => {
    if (defaultRootFolder?.exists && defaultRootFolder.defaultEntry?.entityId && !rootFolderId) {
      setRootFolderId(String(defaultRootFolder.defaultEntry.entityId))
    }
  }, [defaultRootFolder, rootFolderId])

  const handleSelectSeries = (series: SeriesSearchResult) => {
    setSelectedSeries(series)
    setStep('configure')
  }

  const handleBack = () => {
    if (step === 'configure') {
      setStep('search')
      setSelectedSeries(null)
    } else {
      navigate({ to: '/series' })
    }
  }

  const handleAdd = async () => {
    if (!selectedSeries || !rootFolderId || !qualityProfileId) {
      toast.error('Please fill in all required fields')
      return
    }

    const input: AddSeriesInput = {
      title: selectedSeries.title,
      year: selectedSeries.year,
      tvdbId: selectedSeries.tvdbId,
      tmdbId: selectedSeries.tmdbId,
      imdbId: selectedSeries.imdbId,
      overview: selectedSeries.overview,
      runtime: selectedSeries.runtime,
      rootFolderId: parseInt(rootFolderId),
      qualityProfileId: parseInt(qualityProfileId),
      monitored: monitorOnAdd !== 'none',
      seasonFolder,
      posterUrl: selectedSeries.posterUrl,
      backdropUrl: selectedSeries.backdropUrl,
      searchOnAdd: searchOnAdd ?? 'no',
      monitorOnAdd: monitorOnAdd ?? 'future',
      includeSpecials: includeSpecials ?? false,
    }

    try {
      const series = await addMutation.mutateAsync(input)
      toast.success(`Added "${series.title}"`)
      navigate({ to: '/series/$id', params: { id: String(series.id) } })
    } catch {
      toast.error('Failed to add series')
    }
  }

  return (
    <div>
      <PageHeader
        title="Add Series"
        breadcrumbs={[
          { label: 'Series', href: '/series' },
          { label: 'Add' },
        ]}
        actions={
          <Button variant="ghost" onClick={handleBack}>
            <ArrowLeft className="size-4 mr-2" />
            Back
          </Button>
        }
      />

      {/* Loading state when fetching by tmdbId */}
      {tmdbId && loadingMetadata && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {step === 'search' && !tmdbId && (
        <div className="space-y-6">
          {/* Search input */}
          <div className="max-w-xl">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search for a series..."
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
              title="Search for a series"
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
              {searchResults.map((series) => (
                <Card
                  key={series.tmdbId || series.id}
                  className="cursor-pointer hover:border-primary transition-colors"
                  onClick={() => handleSelectSeries(series)}
                >
                  <div className="aspect-[2/3] relative">
                    <PosterImage
                      url={series.posterUrl}
                      alt={series.title}
                      type="series"
                      className="absolute inset-0 rounded-t-lg"
                    />
                  </div>
                  <CardContent className="p-3">
                    <h3 className="font-semibold truncate">{series.title}</h3>
                    <p className="text-sm text-muted-foreground">
                      {series.year || 'Unknown year'}
                      {series.network && ` - ${series.network}`}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {step === 'configure' && selectedSeries && (
        <div className="max-w-2xl space-y-6">
          {/* Selected series preview */}
          <Card>
            <CardContent className="p-4 flex gap-4">
              <PosterImage
                url={selectedSeries.posterUrl}
                alt={selectedSeries.title}
                type="series"
                className="w-24 h-36 rounded shrink-0"
              />
              <div>
                <h2 className="text-xl font-semibold">{selectedSeries.title}</h2>
                <p className="text-muted-foreground">
                  {selectedSeries.year || 'Unknown year'}
                  {selectedSeries.network && ` - ${selectedSeries.network}`}
                </p>
                {selectedSeries.overview && (
                  <p className="text-sm text-muted-foreground mt-2 line-clamp-3">
                    {selectedSeries.overview}
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
                    {rootFolderId && rootFolders?.find(f => f.id === parseInt(rootFolderId))?.name || "Select a root folder"}
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
                    {qualityProfileId && qualityProfiles?.find(p => p.id === parseInt(qualityProfileId))?.name || "Select a quality profile"}
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

              <div className="space-y-2">
                <Label>Monitor</Label>
                <Select value={monitorOnAdd ?? 'future'} onValueChange={(v) => setMonitorOnAdd(v as SeriesMonitorOnAdd)}>
                  <SelectTrigger>
                    {{
                      all: 'All Episodes',
                      future: 'Future Episodes Only',
                      first_season: 'First Season Only',
                      latest_season: 'Latest Season Only',
                      none: 'None',
                    }[monitorOnAdd ?? 'future']}
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Episodes</SelectItem>
                    <SelectItem value="future">Future Episodes Only</SelectItem>
                    <SelectItem value="first_season">First Season Only</SelectItem>
                    <SelectItem value="latest_season">Latest Season Only</SelectItem>
                    <SelectItem value="none">None</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-sm text-muted-foreground">
                  Which episodes should be monitored for automatic downloads
                </p>
              </div>

              <div className="space-y-2">
                <Label>Search on Add</Label>
                <Select value={searchOnAdd ?? 'no'} onValueChange={(v) => setSearchOnAdd(v as SeriesSearchOnAdd)}>
                  <SelectTrigger>
                    {{
                      no: "Don't Search",
                      first_episode: 'First Episode Only',
                      first_season: 'First Season Only',
                      latest_season: 'Latest Season Only',
                      all: 'All Monitored Episodes',
                    }[searchOnAdd ?? 'no']}
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="no">Don't Search</SelectItem>
                    <SelectItem value="first_episode">First Episode Only</SelectItem>
                    <SelectItem value="first_season">First Season Only</SelectItem>
                    <SelectItem value="latest_season">Latest Season Only</SelectItem>
                    <SelectItem value="all">All Monitored Episodes</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-sm text-muted-foreground">
                  Start searching for releases immediately after adding
                </p>
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Season Folder</Label>
                  <p className="text-sm text-muted-foreground">
                    Organize episodes into season folders
                  </p>
                </div>
                <Switch checked={seasonFolder} onCheckedChange={setSeasonFolder} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Include Specials</Label>
                  <p className="text-sm text-muted-foreground">
                    Monitor and search for special episodes (Season 0)
                  </p>
                </div>
                <Switch checked={includeSpecials ?? false} onCheckedChange={setIncludeSpecials} />
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
              Add Series
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
