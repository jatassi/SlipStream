import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { TorrentInfo } from '@/types'

import type { ReleaseGrabHandler, SortColumn, SortDirection } from './search-modal-types'
import { SearchReleaseRow } from './search-release-row'
import { SortIcon } from './sort-icon'

type SortableHeaderProps = {
  label: string
  column: SortColumn
  sortColumn: SortColumn
  sortDirection: SortDirection
  onSort: (column: SortColumn) => void
  className?: string
}

function SortableHeader({ label, column, sortColumn, sortDirection, onSort, className }: SortableHeaderProps) {
  return (
    <TableHead className={className}>
      <button
        className="hover:text-foreground flex items-center transition-colors"
        onClick={() => onSort(column)}
      >
        {label}
        <SortIcon column={column} sortColumn={sortColumn} sortDirection={sortDirection} />
      </button>
    </TableHead>
  )
}

type HeaderRowProps = {
  sortColumn: SortColumn
  sortDirection: SortDirection
  onSort: (column: SortColumn) => void
  hasTorrents: boolean
  hasSlotInfo: boolean
}

function HeaderRow({ sortColumn, sortDirection, onSort, hasTorrents, hasSlotInfo }: HeaderRowProps) {
  const sortProps = { sortColumn, sortDirection, onSort }

  return (
    <TableRow>
      <SortableHeader label="Title" column="title" {...sortProps} />
      <SortableHeader label="Score" column="score" className="w-[70px]" {...sortProps} />
      <SortableHeader label="Quality" column="quality" className="w-[100px]" {...sortProps} />
      {hasSlotInfo ? (
        <SortableHeader label="Slot" column="slot" className="w-[120px]" {...sortProps} />
      ) : null}
      <SortableHeader label="Indexer" column="indexer" className="w-[100px]" {...sortProps} />
      <SortableHeader label="Size" column="size" className="w-[80px]" {...sortProps} />
      <SortableHeader label="Age" column="age" className="w-[100px]" {...sortProps} />
      {hasTorrents ? (
        <SortableHeader label="Peers" column="peers" className="w-[100px]" {...sortProps} />
      ) : null}
      <TableHead className="w-[80px] text-right">Actions</TableHead>
    </TableRow>
  )
}

export function SearchResultsTable({
  releases,
  sortColumn,
  sortDirection,
  grabbingGuid,
  hasTorrents,
  hasSlotInfo,
  onSort,
  onGrab,
}: {
  releases: TorrentInfo[]
  sortColumn: SortColumn
  sortDirection: SortDirection
  grabbingGuid: string | null
  hasTorrents: boolean
  hasSlotInfo: boolean
  onSort: (column: SortColumn) => void
  onGrab: ReleaseGrabHandler
}) {
  return (
    <Table>
      <TableHeader>
        <HeaderRow
          sortColumn={sortColumn}
          sortDirection={sortDirection}
          onSort={onSort}
          hasTorrents={hasTorrents}
          hasSlotInfo={hasSlotInfo}
        />
      </TableHeader>
      <TableBody>
        {releases.map((release) => (
          <SearchReleaseRow
            key={release.guid}
            release={release}
            grabbingGuid={grabbingGuid}
            hasTorrents={hasTorrents}
            hasSlotInfo={hasSlotInfo}
            onGrab={onGrab}
          />
        ))}
      </TableBody>
    </Table>
  )
}
