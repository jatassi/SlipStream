import type { ReactNode } from 'react'
import { Loader2, Plus, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface SearchResultsSectionProps {
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
          className={`text-lg font-semibold flex items-center gap-2 ${
            !isLoading && !hasResults ? 'text-muted-foreground' : ''
          }`}
        >
          {icon}
          {title}
          {!isLoading && !hasResults && ' (0 results)'}
        </h2>
        {isLoading && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
      </div>

      {isLoading ? (
        <LoadingGrid />
      ) : !hasResults && emptyMessage ? (
        <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
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
    <div className="grid gap-3 grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="aspect-[2/3] rounded-lg bg-muted animate-pulse" />
      ))}
    </div>
  )
}

interface ExternalSearchSectionProps {
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
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <Plus className="size-5" />
          {title}
        </h2>
        {isLoading && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
      </div>

      {!enabled ? (
        <div className="rounded-lg border border-dashed border-border bg-card/50 p-6 text-center">
          <p className="text-muted-foreground mb-4">
            Want to add something new to your library?
          </p>
          <Button onClick={onEnable}>
            <Search className="size-4 mr-2" />
            Search TMDB for "{query}"
          </Button>
        </div>
      ) : isLoading ? (
        <LoadingGrid />
      ) : !hasResults ? (
        <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
          {emptyMessage || `No external results found for "${query}"`}
        </div>
      ) : (
        children
      )}
    </section>
  )
}
