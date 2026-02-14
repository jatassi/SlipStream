import type { ReactNode } from 'react'

import { Loader2, Plus, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'

type SearchResultsSectionProps = {
  title: string
  icon?: ReactNode
  isLoading?: boolean
  hasResults: boolean
  emptyMessage?: string
  children: ReactNode
  className?: string
}

export function SearchResultsSection({
  title,
  icon,
  isLoading,
  hasResults,
  emptyMessage,
  children,
  className,
}: SearchResultsSectionProps) {
  return (
    <section className={`space-y-4 ${className || ''}`}>
      <div className="flex items-center gap-2">
        <h2
          className={`flex items-center gap-2 text-lg font-semibold ${
            !isLoading && !hasResults ? 'text-muted-foreground' : ''
          }`}
        >
          {icon}
          {title}
          {!isLoading && !hasResults && ' (0 results)'}
        </h2>
        {isLoading ? <Loader2 className="text-muted-foreground size-4 animate-spin" /> : null}
      </div>

      {isLoading ? (
        <LoadingGrid />
      ) : !hasResults && emptyMessage ? (
        <div className="border-border bg-card text-muted-foreground rounded-lg border p-6 text-center">
          {emptyMessage}
        </div>
      ) : hasResults ? (
        children
      ) : null}
    </section>
  )
}

function LoadingGrid() {
  return (
    <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
      {Array.from({ length: 8 }, (_, i) => i).map((i) => (
        <div key={i} className="bg-muted aspect-[2/3] animate-pulse rounded-lg" />
      ))}
    </div>
  )
}

type ExternalSearchSectionProps = {
  query: string
  enabled: boolean
  onEnable: () => void
  isLoading?: boolean
  hasResults: boolean
  children: ReactNode
  title?: string
  emptyMessage?: string
}

export function ExternalSearchSection({
  query,
  enabled,
  onEnable,
  isLoading,
  hasResults,
  children,
  title = 'Add New',
  emptyMessage,
}: ExternalSearchSectionProps) {
  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2">
        <h2 className="flex items-center gap-2 text-lg font-semibold">
          <Plus className="size-5" />
          {title}
        </h2>
        {isLoading ? <Loader2 className="text-muted-foreground size-4 animate-spin" /> : null}
      </div>

      {enabled ? (
        isLoading ? (
          <LoadingGrid />
        ) : hasResults ? (
          children
        ) : (
          <div className="border-border bg-card text-muted-foreground rounded-lg border p-6 text-center">
            {emptyMessage || `No external results found for "${query}"`}
          </div>
        )
      ) : (
        <div className="border-border bg-card/50 rounded-lg border border-dashed p-6 text-center">
          <p className="text-muted-foreground mb-4">Want to add something new to your library?</p>
          <Button onClick={onEnable}>
            <Search className="mr-2 size-4" />
            Search TMDB for &quot;{query}&quot;
          </Button>
        </div>
      )}
    </section>
  )
}
