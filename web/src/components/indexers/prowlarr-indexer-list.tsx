import { Loader2, Rss } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import type { ProwlarrIndexerWithSettings } from '@/types'

import { IndexerRow } from './indexer-row'
import { useProwlarrIndexerList } from './use-prowlarr-indexer-list'

type ProwlarrIndexerListProps = {
  showOnlyEnabled?: boolean
}

export function ProwlarrIndexerList({ showOnlyEnabled = false }: ProwlarrIndexerListProps) {
  const { connected, indexersLoading, displayedIndexers, enabledCount, totalCount } =
    useProwlarrIndexerList(showOnlyEnabled)

  if (!connected) {
    return <IndexerPlaceholder title="Prowlarr not connected" description="Configure and test your Prowlarr connection to view indexers" />
  }

  if (indexersLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </CardContent>
      </Card>
    )
  }

  if (!displayedIndexers?.length) {
    return (
      <IndexerPlaceholder
        title={showOnlyEnabled ? 'No enabled indexers' : 'No indexers found'}
        description={showOnlyEnabled ? 'Enable indexers in Prowlarr to use them with SlipStream' : 'Add indexers in Prowlarr to search for releases'}
      />
    )
  }

  return <IndexerListContent indexers={displayedIndexers} enabledCount={enabledCount} totalCount={totalCount} />
}

function IndexerPlaceholder({ title, description }: { title: string; description: string }) {
  return (
    <Card>
      <CardContent className="py-8">
        <EmptyState icon={<Rss className="size-8" />} title={title} description={description} />
      </CardContent>
    </Card>
  )
}

function IndexerListContent({
  indexers,
  enabledCount,
  totalCount,
}: {
  indexers: ProwlarrIndexerWithSettings[]
  enabledCount: number
  totalCount: number
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base">Prowlarr Indexers</CardTitle>
            <CardDescription>
              Configure per-indexer settings for priority and content filtering
            </CardDescription>
          </div>
          <Badge variant="secondary">
            {enabledCount} / {totalCount} enabled
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {indexers.map((indexer) => (
            <IndexerRow key={indexer.id} indexer={indexer} />
          ))}
        </div>
        <p className="text-muted-foreground mt-4 text-xs">
          Priority: Lower numbers are preferred during deduplication. Content type filters which
          searches use this indexer.
        </p>
      </CardContent>
    </Card>
  )
}
