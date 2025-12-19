import { Search, Check, X } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { QualityBadge } from '@/components/media/QualityBadge'
import { formatDate } from '@/lib/formatters'
import type { Episode } from '@/types'

interface EpisodeTableProps {
  episodes: Episode[]
  onSearch?: (episode: Episode) => void
}

export function EpisodeTable({ episodes, onSearch }: EpisodeTableProps) {
  // Sort by episode number
  const sortedEpisodes = [...episodes].sort(
    (a, b) => a.episodeNumber - b.episodeNumber
  )

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-16">#</TableHead>
          <TableHead>Title</TableHead>
          <TableHead>Air Date</TableHead>
          <TableHead>Quality</TableHead>
          <TableHead className="w-20">Status</TableHead>
          <TableHead className="w-16"></TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedEpisodes.map((episode) => (
          <TableRow key={episode.id}>
            <TableCell className="font-mono">{episode.episodeNumber}</TableCell>
            <TableCell>
              <div>
                <span className="font-medium">{episode.title}</span>
                {episode.overview && (
                  <p className="text-xs text-muted-foreground line-clamp-1 mt-0.5">
                    {episode.overview}
                  </p>
                )}
              </div>
            </TableCell>
            <TableCell>
              {episode.airDate ? formatDate(episode.airDate) : '-'}
            </TableCell>
            <TableCell>
              {episode.episodeFile ? (
                <QualityBadge quality={episode.episodeFile.quality} />
              ) : (
                '-'
              )}
            </TableCell>
            <TableCell>
              {episode.hasFile ? (
                <Check className="size-4 text-green-500" />
              ) : (
                <X className="size-4 text-red-500" />
              )}
            </TableCell>
            <TableCell>
              {onSearch && !episode.hasFile && episode.monitored && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  onClick={() => onSearch(episode)}
                >
                  <Search className="size-4" />
                </Button>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
