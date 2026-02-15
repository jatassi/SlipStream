import { Globe, Lock, Unlock } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { TableCell, TableRow } from '@/components/ui/table'
import type { DefinitionMetadata, Privacy, Protocol } from '@/types'

const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-4" />,
  'semi-private': <Unlock className="size-4" />,
  private: <Lock className="size-4" />,
}

const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500 hover:bg-green-500/20',
  'semi-private': 'bg-yellow-500/10 text-yellow-500 hover:bg-yellow-500/20',
  private: 'bg-red-500/10 text-red-500 hover:bg-red-500/20',
}

const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500 hover:bg-blue-500/20',
  usenet: 'bg-purple-500/10 text-purple-500 hover:bg-purple-500/20',
}

type DefinitionTableRowsProps = {
  definitions: DefinitionMetadata[]
  isLoading?: boolean
  hasActiveFilters: boolean
  onSelect: (definition: DefinitionMetadata) => void
}

function LoadingRow() {
  return (
    <TableRow>
      <TableCell colSpan={4} className="text-muted-foreground py-8 text-center">
        Loading definitions...
      </TableCell>
    </TableRow>
  )
}

function EmptyRow({ hasActiveFilters }: { hasActiveFilters: boolean }) {
  return (
    <TableRow>
      <TableCell colSpan={4} className="text-muted-foreground py-8 text-center">
        {hasActiveFilters ? 'No definitions match your filters' : 'No definitions available'}
      </TableCell>
    </TableRow>
  )
}

function DefinitionRow({
  definition,
  onSelect,
}: {
  definition: DefinitionMetadata
  onSelect: (definition: DefinitionMetadata) => void
}) {
  return (
    <TableRow
      className="hover:bg-muted/50 cursor-pointer"
      onClick={() => onSelect(definition)}
    >
      <TableCell className="font-medium">{definition.name}</TableCell>
      <TableCell>
        <Badge variant="secondary" className={protocolColors[definition.protocol]}>
          {definition.protocol}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant="secondary" className={privacyColors[definition.privacy]}>
          <span className="mr-1">{privacyIcons[definition.privacy]}</span>
          {definition.privacy}
        </Badge>
      </TableCell>
      <TableCell className="text-muted-foreground max-w-[300px] truncate">
        {definition.description ?? '-'}
      </TableCell>
    </TableRow>
  )
}

export function DefinitionTableRows({
  definitions,
  isLoading,
  hasActiveFilters,
  onSelect,
}: DefinitionTableRowsProps) {
  if (isLoading) {
    return <LoadingRow />
  }
  if (definitions.length === 0) {
    return <EmptyRow hasActiveFilters={hasActiveFilters} />
  }

  return definitions.map((def) => (
    <DefinitionRow key={def.id} definition={def} onSelect={onSelect} />
  ))
}
