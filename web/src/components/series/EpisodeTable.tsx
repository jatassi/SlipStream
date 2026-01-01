import { Search, Check, X, Eye, EyeOff, MoreHorizontal } from 'lucide-react'
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
import { QualityBadge } from '@/components/media/QualityBadge'
import { formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
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

  const handleAutoSearch = (episode: Episode) => {
    if (onSearch) {
      onSearch(episode)
    } else {
      toast.info('Automatic search not yet implemented')
    }
  }

  const handleManualSearch = (_episode: Episode) => {
    toast.info('Manual search not yet implemented')
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-16">#</TableHead>
          <TableHead>Title</TableHead>
          <TableHead>Air Date</TableHead>
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
              {episode.monitored ? (
                <Eye className="size-4 text-green-500" />
              ) : (
                <EyeOff className="size-4 text-muted-foreground" />
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
              {episode.episodeFile ? (
                <QualityBadge quality={episode.episodeFile.quality} />
              ) : (
                '-'
              )}
            </TableCell>
            <TableCell>
              <DropdownMenu>
                <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
                  <MoreHorizontal className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => handleAutoSearch(episode)}>
                    <Search className="size-4 mr-2" />
                    Automatic Search
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => handleManualSearch(episode)}>
                    <Search className="size-4 mr-2" />
                    Manual Search
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
