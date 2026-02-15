import { Download, ExternalLink, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { TableCell, TableRow } from '@/components/ui/table'
import { formatBytes, formatRelativeTime } from '@/lib/formatters'
import type { TorrentInfo } from '@/types'

import { ReleaseSlotCell } from './release-slot-cell'
import type { ReleaseGrabHandler } from './search-modal-types'

function TitleCell({ release }: { release: TorrentInfo }) {
  return (
    <TableCell>
      <div className="flex flex-col gap-1">
        <span className="font-medium">{release.title}</span>
        <div className="flex gap-1">
          <Badge variant="outline" className="text-xs">
            {release.protocol}
          </Badge>
          {release.downloadVolumeFactor === 0 && (
            <Badge variant="secondary" className="text-xs">
              Freeleech
            </Badge>
          )}
        </div>
      </div>
    </TableCell>
  )
}

function ActionsCell({ release, isGrabbing, onGrab }: { release: TorrentInfo, isGrabbing: boolean, onGrab: ReleaseGrabHandler }) {
  return (
    <TableCell className="text-right">
      <div className="flex justify-end gap-1">
        {release.infoUrl ? (
          <a
            href={release.infoUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:bg-accent hover:text-accent-foreground inline-flex h-9 w-9 items-center justify-center rounded-md text-sm font-medium transition-colors"
          >
            <ExternalLink className="size-4" />
          </a>
        ) : null}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => onGrab(release)}
          disabled={isGrabbing}
        >
          {isGrabbing ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Download className="size-4" />
          )}
        </Button>
      </div>
    </TableCell>
  )
}

function PeersCell({ seeders, leechers }: { seeders?: number, leechers?: number }) {
  return (
    <TableCell>
      <span className="text-sm">
        <span className="text-green-500">{seeders}</span>
        {' / '}
        <span className="text-red-500">{leechers}</span>
      </span>
    </TableCell>
  )
}

export function SearchReleaseRow({
  release,
  grabbingGuid,
  hasTorrents,
  hasSlotInfo,
  onGrab,
}: {
  release: TorrentInfo
  grabbingGuid: string | null
  hasTorrents: boolean
  hasSlotInfo: boolean
  onGrab: ReleaseGrabHandler
}) {
  const isGrabbing = grabbingGuid === release.guid

  return (
    <TableRow>
      <TitleCell release={release} />
      <TableCell>
        <span className="font-medium">{release.normalizedScore ?? '-'}</span>
      </TableCell>
      <TableCell>
        {release.quality ? (
          <Badge variant="secondary">{release.quality}</Badge>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
      {hasSlotInfo ? (
        <TableCell>
          <ReleaseSlotCell release={release} />
        </TableCell>
      ) : null}
      <TableCell>
        <Badge variant="outline">{release.indexer}</Badge>
      </TableCell>
      <TableCell>{formatBytes(release.size)}</TableCell>
      <TableCell>
        {release.publishDate ? formatRelativeTime(release.publishDate) : '-'}
      </TableCell>
      {hasTorrents ? <PeersCell seeders={release.seeders} leechers={release.leechers} /> : null}
      <ActionsCell release={release} isGrabbing={isGrabbing} onGrab={onGrab} />
    </TableRow>
  )
}
