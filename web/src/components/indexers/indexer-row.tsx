import { Rss, TrendingDown, TrendingUp } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import type { ProwlarrIndexerWithSettings } from '@/types'
import { ContentTypeLabels, ProwlarrIndexerStatusLabels } from '@/types'

import { IndexerSettingsDialog } from './indexer-settings-dialog'
import {
  contentTypeColors,
  privacyColors,
  privacyIcons,
  protocolColors,
  statusColors,
  statusIcons,
} from './prowlarr-indexer-constants'

export function IndexerRow({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const contentType = indexer.settings?.contentType ?? 'both'

  return (
    <div className="hover:bg-muted/50 flex items-center justify-between rounded-lg border p-3 transition-colors">
      <div className="flex items-center gap-3">
        <div className="bg-muted flex size-8 items-center justify-center rounded-lg">
          <Rss className="size-4" />
        </div>
        <div>
          <IndexerBadges indexer={indexer} contentType={contentType} />
          <IndexerMeta indexer={indexer} />
        </div>
      </div>
      <IndexerStatus indexer={indexer} />
    </div>
  )
}

function IndexerBadges({
  indexer,
  contentType,
}: {
  indexer: ProwlarrIndexerWithSettings
  contentType: 'movies' | 'series' | 'both'
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium">{indexer.name}</span>
      <Badge variant="secondary" className={`text-xs ${protocolColors[indexer.protocol]}`}>
        {indexer.protocol}
      </Badge>
      {indexer.privacy ? (
        <Badge variant="secondary" className={`text-xs ${privacyColors[indexer.privacy]}`}>
          <span className="mr-1">{privacyIcons[indexer.privacy]}</span>
          {indexer.privacy}
        </Badge>
      ) : null}
      <Badge variant="secondary" className={`text-xs ${contentTypeColors[contentType]}`}>
        {ContentTypeLabels[contentType]}
      </Badge>
    </div>
  )
}

function IndexerMeta({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const priority = indexer.settings?.priority ?? 25
  const settings = indexer.settings
  const caps = indexer.capabilities
  const hasStats = settings && (settings.successCount > 0 || settings.failureCount > 0)

  return (
    <div className="text-muted-foreground mt-0.5 flex items-center gap-2 text-xs">
      <span>Priority: {priority}</span>
      <CapabilitiesInfo capabilities={caps} />
      {hasStats ? <StatsInfo successCount={settings.successCount} failureCount={settings.failureCount} /> : null}
    </div>
  )
}

function CapabilitiesInfo({ capabilities }: { capabilities?: ProwlarrIndexerWithSettings['capabilities'] }) {
  if (!capabilities) {
    return null
  }
  const { supportsMovieSearch, supportsTvSearch } = capabilities

  return (
    <>
      <span className="text-muted-foreground/50">|</span>
      {supportsMovieSearch ? <span>Movies</span> : null}
      {supportsMovieSearch && supportsTvSearch ? (
        <span className="text-muted-foreground/50">/</span>
      ) : null}
      {supportsTvSearch ? <span>TV</span> : null}
    </>
  )
}

function StatsInfo({ successCount, failureCount }: { successCount: number; failureCount: number }) {
  return (
    <>
      <span className="text-muted-foreground/50">|</span>
      <span className="flex items-center gap-1">
        <TrendingUp className="size-3 text-green-500" />
        {successCount}
      </span>
      <span className="flex items-center gap-1">
        <TrendingDown className="size-3 text-red-500" />
        {failureCount}
      </span>
    </>
  )
}

function IndexerStatus({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const statusLabel = ProwlarrIndexerStatusLabels[indexer.status]

  return (
    <div className="flex items-center gap-2">
      <div className={`flex items-center gap-1.5 text-xs ${statusColors[indexer.status]}`}>
        {statusIcons[indexer.status]}
        <span>{statusLabel}</span>
      </div>
      {!indexer.enable && (
        <Badge variant="outline" className="text-xs">
          Disabled
        </Badge>
      )}
      <IndexerSettingsDialog indexer={indexer} />
    </div>
  )
}
