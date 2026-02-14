import { useState } from 'react'

import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  Globe,
  Loader2,
  Lock,
  RotateCcw,
  Rss,
  Settings,
  TrendingDown,
  TrendingUp,
  Unlock,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/EmptyState'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import {
  useProwlarrIndexersWithSettings,
  useProwlarrStatus,
  useResetProwlarrIndexerStats,
  useUpdateProwlarrIndexerSettings,
} from '@/hooks'
import type {
  ContentType,
  Privacy,
  Protocol,
  ProwlarrIndexerSettingsInput,
  ProwlarrIndexerStatus,
  ProwlarrIndexerWithSettings,
} from '@/types'
import { ContentTypeLabels, ProwlarrIndexerStatusLabels } from '@/types'

const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-3" />,
  'semi-private': <Unlock className="size-3" />,
  private: <Lock className="size-3" />,
}

const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500',
  'semi-private': 'bg-yellow-500/10 text-yellow-500',
  private: 'bg-red-500/10 text-red-500',
}

const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500',
  usenet: 'bg-purple-500/10 text-purple-500',
}

const statusIcons: Record<ProwlarrIndexerStatus, React.ReactNode> = {
  0: <CheckCircle2 className="size-4 text-green-500" />,
  1: <AlertTriangle className="size-4 text-yellow-500" />,
  2: <Ban className="text-muted-foreground size-4" />,
  3: <XCircle className="size-4 text-red-500" />,
}

const statusColors: Record<ProwlarrIndexerStatus, string> = {
  0: 'text-green-500',
  1: 'text-yellow-500',
  2: 'text-muted-foreground',
  3: 'text-red-500',
}

const contentTypeColors: Record<ContentType, string> = {
  movies: 'bg-amber-500/10 text-amber-500',
  series: 'bg-cyan-500/10 text-cyan-500',
  both: 'bg-gray-500/10 text-gray-400',
}

type ProwlarrIndexerListProps = {
  showOnlyEnabled?: boolean
}

export function ProwlarrIndexerList({ showOnlyEnabled = false }: ProwlarrIndexerListProps) {
  const { data: indexers, isLoading: indexersLoading } = useProwlarrIndexersWithSettings()
  const { data: status } = useProwlarrStatus()

  if (!status?.connected) {
    return (
      <Card>
        <CardContent className="py-8">
          <EmptyState
            icon={<Rss className="size-8" />}
            title="Prowlarr not connected"
            description="Configure and test your Prowlarr connection to view indexers"
          />
        </CardContent>
      </Card>
    )
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

  const filteredIndexers = showOnlyEnabled ? indexers?.filter((i) => i.enable) : indexers
  const displayedIndexers = filteredIndexers?.slice().sort((a, b) => {
    // Sort by enabled state first (enabled at top)
    if (a.enable !== b.enable) {
      return a.enable ? -1 : 1
    }
    // Then by priority (lower numbers first)
    const priorityA = a.settings?.priority ?? 25
    const priorityB = b.settings?.priority ?? 25
    return priorityA - priorityB
  })

  if (!displayedIndexers?.length) {
    return (
      <Card>
        <CardContent className="py-8">
          <EmptyState
            icon={<Rss className="size-8" />}
            title={showOnlyEnabled ? 'No enabled indexers' : 'No indexers found'}
            description={
              showOnlyEnabled
                ? 'Enable indexers in Prowlarr to use them with SlipStream'
                : 'Add indexers in Prowlarr to search for releases'
            }
          />
        </CardContent>
      </Card>
    )
  }

  const enabledCount = indexers?.filter((i) => i.enable).length ?? 0
  const totalCount = indexers?.length ?? 0

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
          {displayedIndexers.map((indexer) => (
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

function IndexerRow({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const statusLabel = ProwlarrIndexerStatusLabels[indexer.status] ?? 'Unknown'
  const settings = indexer.settings
  const priority = settings?.priority ?? 25
  const contentType = settings?.contentType ?? 'both'

  return (
    <div className="hover:bg-muted/50 flex items-center justify-between rounded-lg border p-3 transition-colors">
      <div className="flex items-center gap-3">
        <div className="bg-muted flex size-8 items-center justify-center rounded-lg">
          <Rss className="size-4" />
        </div>
        <div>
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
          <div className="text-muted-foreground mt-0.5 flex items-center gap-2 text-xs">
            <span>Priority: {priority}</span>
            {indexer.capabilities ? (
              <>
                <span className="text-muted-foreground/50">|</span>
                {indexer.capabilities.supportsMovieSearch ? <span>Movies</span> : null}
                {indexer.capabilities.supportsMovieSearch &&
                indexer.capabilities.supportsTvSearch ? (
                  <span className="text-muted-foreground/50">/</span>
                ) : null}
                {indexer.capabilities.supportsTvSearch ? <span>TV</span> : null}
              </>
            ) : null}
            {settings && (settings.successCount > 0 || settings.failureCount > 0) ? (
              <>
                <span className="text-muted-foreground/50">|</span>
                <span className="flex items-center gap-1">
                  <TrendingUp className="size-3 text-green-500" />
                  {settings.successCount}
                </span>
                <span className="flex items-center gap-1">
                  <TrendingDown className="size-3 text-red-500" />
                  {settings.failureCount}
                </span>
              </>
            ) : null}
          </div>
        </div>
      </div>
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
    </div>
  )
}

function IndexerSettingsDialog({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const updateSettings = useUpdateProwlarrIndexerSettings()
  const resetStats = useResetProwlarrIndexerStats()

  const [priority, setPriority] = useState(indexer.settings?.priority ?? 25)
  const [contentType, setContentType] = useState<ContentType>(
    indexer.settings?.contentType ?? 'both',
  )
  const [open, setOpen] = useState(false)

  const handleSave = async () => {
    const data: ProwlarrIndexerSettingsInput = {
      priority,
      contentType,
    }

    try {
      await updateSettings.mutateAsync({ indexerId: indexer.id, data })
      toast.success(`Settings updated for ${indexer.name}`)
      setOpen(false)
    } catch {
      toast.error('Failed to update settings')
    }
  }

  const handleResetStats = async () => {
    try {
      await resetStats.mutateAsync(indexer.id)
      toast.success('Stats reset')
    } catch {
      toast.error('Failed to reset stats')
    }
  }

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen)
    if (newOpen) {
      setPriority(indexer.settings?.priority ?? 25)
      setContentType(indexer.settings?.contentType ?? 'both')
    }
  }

  const settings = indexer.settings

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
        <Settings className="size-4" />
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Settings for {indexer.name}</DialogTitle>
          <DialogDescription>
            Configure priority and content type filtering for this indexer
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="priority">Priority (1-50)</Label>
            <Input
              id="priority"
              type="number"
              min={1}
              max={50}
              value={priority}
              onChange={(e) =>
                setPriority(Math.min(50, Math.max(1, Number.parseInt(e.target.value) || 1)))
              }
            />
            <p className="text-muted-foreground text-xs">
              Lower priority indexers are preferred during deduplication
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="contentType">Content Type</Label>
            <Select value={contentType} onValueChange={(v) => setContentType(v!)}>
              <SelectTrigger id="contentType">{ContentTypeLabels[contentType]}</SelectTrigger>
              <SelectContent>
                <SelectItem value="both">Both</SelectItem>
                <SelectItem value="movies">Movies Only</SelectItem>
                <SelectItem value="series">Series Only</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-muted-foreground text-xs">
              Filter this indexer to only be used for specific content types
            </p>
          </div>

          {settings && (settings.successCount > 0 || settings.failureCount > 0) ? (
            <div className="space-y-2 rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Statistics</span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleResetStats}
                  disabled={resetStats.isPending}
                >
                  <RotateCcw className="mr-1 size-3" />
                  Reset
                </Button>
              </div>
              <div className="flex gap-4 text-sm">
                <div className="flex items-center gap-1">
                  <TrendingUp className="size-4 text-green-500" />
                  <span>{settings.successCount} successful</span>
                </div>
                <div className="flex items-center gap-1">
                  <TrendingDown className="size-4 text-red-500" />
                  <span>{settings.failureCount} failed</span>
                </div>
              </div>
              {settings.lastFailureReason ? (
                <p className="text-muted-foreground text-xs">
                  Last failure: {settings.lastFailureReason}
                </p>
              ) : null}
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button onClick={handleSave} disabled={updateSettings.isPending}>
            {updateSettings.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
