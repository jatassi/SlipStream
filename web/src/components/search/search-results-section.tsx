import type { ReactNode } from 'react'

import { Loader2 } from 'lucide-react'

function LoadingGrid() {
  return (
    <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
      {Array.from({ length: 8 }, (_, i) => i).map((i) => (
        <div key={i} className="bg-muted aspect-[2/3] animate-pulse rounded-lg" />
      ))}
    </div>
  )
}

function EmptyMessage({ message }: { message: string }) {
  return (
    <div className="border-border bg-card text-muted-foreground rounded-lg border p-6 text-center">
      {message}
    </div>
  )
}

function SectionHeader({ title, icon, isLoading, hasResults }: {
  title: string
  icon?: ReactNode
  isLoading?: boolean
  hasResults: boolean
}) {
  const noResults = !isLoading && !hasResults
  return (
    <div className="flex items-center gap-2">
      <h2 className={`flex items-center gap-2 text-lg font-semibold ${noResults ? 'text-muted-foreground' : ''}`}>
        {icon}
        {title}
        {noResults ? ' (0 results)' : null}
      </h2>
      {Boolean(isLoading) && <Loader2 className="text-muted-foreground size-4 animate-spin" />}
    </div>
  )
}

function SectionContent({ isLoading, hasResults, emptyMessage, children }: {
  isLoading?: boolean
  hasResults: boolean
  emptyMessage?: string
  children: ReactNode
}): ReactNode {
  if (isLoading) {
    return <LoadingGrid />
  }
  if (hasResults) {
    return children
  }
  if (emptyMessage) {
    return <EmptyMessage message={emptyMessage} />
  }
  return null
}

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
  title, icon, isLoading, hasResults, emptyMessage, children, className,
}: SearchResultsSectionProps) {
  return (
    <section className={`space-y-4 ${className ?? ''}`}>
      <SectionHeader title={title} icon={icon} isLoading={isLoading} hasResults={hasResults} />
      <SectionContent isLoading={isLoading} hasResults={hasResults} emptyMessage={emptyMessage}>{children}</SectionContent>
    </section>
  )
}
