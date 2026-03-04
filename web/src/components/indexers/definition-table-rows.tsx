import { Badge } from '@/components/ui/badge'
import { TableCell, TableRow } from '@/components/ui/table'
import type { DefinitionMetadata } from '@/types'

import {
  privacyColorsInteractive,
  privacyIconsMd,
  protocolColorsInteractive,
} from './prowlarr-indexer-constants'

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
        <Badge variant="secondary" className={protocolColorsInteractive[definition.protocol]}>
          {definition.protocol}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant="secondary" className={privacyColorsInteractive[definition.privacy]}>
          <span className="mr-1">{privacyIconsMd[definition.privacy]}</span>
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
