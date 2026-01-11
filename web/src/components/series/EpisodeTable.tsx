import { Search, Check, X, MoreHorizontal, Download, Zap, Loader2 } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Switch } from '@/components/ui/switch'
import { QualityBadge } from '@/components/media/QualityBadge'
import { EpisodeAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { useDownloadingStore } from '@/stores'
import { formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
import type { Episode } from '@/types'

interface EpisodeTableProps {
  seriesId: number
  episodes: Episode[]
  onAutoSearch?: (episode: Episode) => void
  onManualSearch?: (episode: Episode) => void
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  searchingEpisodeId?: number | null
}

export function EpisodeTable({ seriesId, episodes, onAutoSearch, onManualSearch, onMonitoredChange, searchingEpisodeId }: EpisodeTableProps) {
  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isEpisodeDownloading = (episodeId: number, sId?: number, seasonNum?: number) => {
    return queueItems.some((item) => {
      if (item.status !== 'downloading' && item.status !== 'queued') return false
      // Direct episode match
      if (item.episodeId === episodeId) return true
      // Season pack or complete series covering this episode
      if (sId && item.seriesId === sId) {
        if (item.isCompleteSeries) return true
        if (seasonNum && item.seasonNumber === seasonNum && item.isSeasonPack) return true
      }
      return false
    })
  }

  // Sort by episode number
  const sortedEpisodes = [...episodes].sort(
    (a, b) => a.episodeNumber - b.episodeNumber
  )

  const handleAutoSearch = (episode: Episode) => {
    if (onAutoSearch) {
      onAutoSearch(episode)
    } else {
      toast.info('Automatic search not yet implemented')
    }
  }

  const handleManualSearch = (episode: Episode) => {
    if (onManualSearch) {
      onManualSearch(episode)
    } else {
      toast.info('Manual search not yet implemented')
    }
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-16">#</TableHead>
          <TableHead>Title</TableHead>
          <TableHead>Air Date</TableHead>
          <TableHead>Aired</TableHead>
          <TableHead className="max-w-xs">Description</TableHead>
          <TableHead className="w-24">Monitored</TableHead>
          <TableHead className="w-20">Status</TableHead>
          <TableHead>Quality</TableHead>
          <TableHead className="w-16">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedEpisodes.map((episode) => (
          <TableRow key={episode.id}>
            <TableCell className="font-mono">{episode.episodeNumber}</TableCell>
            <TableCell className="font-medium">{episode.title}</TableCell>
            <TableCell>
              {episode.airDate ? formatDate(episode.airDate) : '-'}
            </TableCell>
            <TableCell>
              <EpisodeAvailabilityBadge released={episode.released} />
            </TableCell>
            <TableCell className="max-w-xs">
              {episode.overview ? (
                <p className="text-xs text-muted-foreground line-clamp-2">
                  {episode.overview}
                </p>
              ) : (
                '-'
              )}
            </TableCell>
            <TableCell>
              <Switch
                checked={episode.monitored}
                onCheckedChange={(checked) => onMonitoredChange?.(episode, checked)}
                disabled={!onMonitoredChange}
              />
            </TableCell>
            <TableCell>
              {isEpisodeDownloading(episode.id, seriesId, episode.seasonNumber) ? (
                <Download className="size-4 text-green-500" />
              ) : episode.hasFile ? (
                <Check className="size-4 text-green-500" />
              ) : (
                <X className="size-4 text-red-500" />
              )}
            </TableCell>
            <TableCell>
              {episode.episodeFile ? (
                <QualityBadge quality={episode.episodeFile.quality} />
              ) : (
                '-'
              )}
            </TableCell>
            <TableCell>
              {isEpisodeDownloading(episode.id, seriesId, episode.seasonNumber) ? (
                <div className="flex items-center justify-center size-8">
                  <Download className="size-4 text-green-500" />
                </div>
              ) : searchingEpisodeId === episode.id ? (
                <div className="flex items-center justify-center size-8">
                  <Loader2 className="size-4 animate-spin" />
                </div>
              ) : (
                <DropdownMenu>
                  <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
                    <MoreHorizontal className="size-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => handleAutoSearch(episode)}>
                      <Zap className="size-4 mr-2" />
                      Automatic Search
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleManualSearch(episode)}>
                      <Search className="size-4 mr-2" />
                      Manual Search
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
