import type { RefObject } from 'react'

import { Search } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { LoadingState } from '@/components/data/loading-state'
import { PosterImage } from '@/components/media/poster-image'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import type { SeriesSearchResult } from '@/types'

type AddSeriesSearchProps = {
  searchQuery: string
  onSearchChange: (query: string) => void
  searchInputRef: RefObject<HTMLInputElement | null>
  searching: boolean
  searchResults: SeriesSearchResult[] | undefined
  onSelect: (series: SeriesSearchResult) => void
}

export function AddSeriesSearch({
  searchQuery,
  onSearchChange,
  searchInputRef,
  searching,
  searchResults,
  onSelect,
}: AddSeriesSearchProps) {
  return (
    <div className="space-y-6">
      <div className="max-w-xl">
        <div className="relative">
          <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <Input
            ref={searchInputRef}
            placeholder="Search for a series..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      <SearchResults
        searching={searching}
        searchQuery={searchQuery}
        searchResults={searchResults}
        onSelect={onSelect}
      />
    </div>
  )
}

type SearchResultsProps = {
  searching: boolean
  searchQuery: string
  searchResults: SeriesSearchResult[] | undefined
  onSelect: (series: SeriesSearchResult) => void
}

function SearchResults({ searching, searchQuery, searchResults, onSelect }: SearchResultsProps) {
  if (searching) {
    return <LoadingState count={4} />
  }

  if (searchQuery.length < 2) {
    return (
      <EmptyState
        icon={<Search className="size-8" />}
        title="Search for a series"
        description="Enter at least 2 characters to search"
      />
    )
  }

  if (!searchResults?.length) {
    return (
      <EmptyState
        icon={<Search className="size-8" />}
        title="No results found"
        description="Try a different search term"
      />
    )
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      {searchResults.map((series) => (
        <Card
          key={series.tmdbId || series.id}
          className="hover:border-primary cursor-pointer transition-colors"
          onClick={() => onSelect(series)}
        >
          <div className="relative aspect-[2/3]">
            <PosterImage
              url={series.posterUrl}
              alt={series.title}
              type="series"
              className="absolute inset-0 rounded-t-lg"
            />
          </div>
          <CardContent className="p-3">
            <h3 className="truncate font-semibold">{series.title}</h3>
            <p className="text-muted-foreground text-sm">
              {series.year ?? 'Unknown year'}
              {series.network ? ` - ${series.network}` : null}
            </p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
