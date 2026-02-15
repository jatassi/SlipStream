import { Filter, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { DefinitionMetadata, Privacy, Protocol } from '@/types'

import { DefinitionTableRows } from './definition-table-rows'
import type { DefinitionStats } from './use-definition-search-table'
import { useDefinitionSearchTable } from './use-definition-search-table'

type DefinitionSearchTableProps = {
  definitions: DefinitionMetadata[]
  isLoading?: boolean
  onSelect: (definition: DefinitionMetadata) => void
}

const PROTOCOL_LABELS: Record<Protocol | 'all', string> = {
  all: 'All',
  torrent: 'Torrent',
  usenet: 'Usenet',
}

function getProtocolLabel(filter: Protocol | 'all', stats: DefinitionStats): string {
  const counts: Record<Protocol | 'all', number> = {
    all: stats.total,
    torrent: stats.torrent,
    usenet: stats.usenet,
  }
  return `${PROTOCOL_LABELS[filter]} (${counts[filter]})`
}

function getPrivacyLabel(filter: Privacy | 'all', stats: DefinitionStats): string {
  if (filter === 'all') {
    return 'All'
  }
  if (filter === 'semi-private') {
    return 'Semi-Private'
  }
  const countMap: Record<string, number> = { public: stats.public, private: stats.private }
  const labels: Record<string, string> = { public: 'Public', private: 'Private' }
  return `${labels[filter]} (${countMap[filter]})`
}

type FilterDropdownsProps = {
  protocolFilter: Protocol | 'all'
  setProtocolFilter: (v: Protocol | 'all') => void
  privacyFilter: Privacy | 'all'
  setPrivacyFilter: (v: Privacy | 'all') => void
  stats: DefinitionStats
}

function FilterDropdowns({
  protocolFilter,
  setProtocolFilter,
  privacyFilter,
  setPrivacyFilter,
  stats,
}: FilterDropdownsProps) {
  return (
    <div className="flex gap-2">
      <Select
        value={protocolFilter}
        onValueChange={(v) => v && setProtocolFilter(v as Protocol | 'all')}
      >
        <SelectTrigger className="w-32">
          {getProtocolLabel(protocolFilter, stats)}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All ({stats.total})</SelectItem>
          <SelectItem value="torrent">Torrent ({stats.torrent})</SelectItem>
          <SelectItem value="usenet">Usenet ({stats.usenet})</SelectItem>
        </SelectContent>
      </Select>

      <Select
        value={privacyFilter}
        onValueChange={(v) => v && setPrivacyFilter(v as Privacy | 'all')}
      >
        <SelectTrigger className="w-36">
          {getPrivacyLabel(privacyFilter, stats)}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All</SelectItem>
          <SelectItem value="public">Public ({stats.public})</SelectItem>
          <SelectItem value="semi-private">Semi-Private</SelectItem>
          <SelectItem value="private">Private ({stats.private})</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )
}

type SearchToolbarProps = {
  searchQuery: string
  setSearchQuery: (v: string) => void
  showFilters: boolean
  setShowFilters: (v: boolean) => void
  protocolFilter: Protocol | 'all'
  setProtocolFilter: (v: Protocol | 'all') => void
  privacyFilter: Privacy | 'all'
  setPrivacyFilter: (v: Privacy | 'all') => void
  stats: DefinitionStats
  filteredCount: number
  totalCount: number
}

function SearchToolbar({
  searchQuery,
  setSearchQuery,
  showFilters,
  setShowFilters,
  protocolFilter,
  setProtocolFilter,
  privacyFilter,
  setPrivacyFilter,
  stats,
  filteredCount,
  totalCount,
}: SearchToolbarProps) {
  return (
    <div className="space-y-3 pb-4">
      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <Input
            placeholder="Search definitions..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button
          variant={showFilters ? 'secondary' : 'outline'}
          size="icon"
          onClick={() => setShowFilters(!showFilters)}
        >
          <Filter className="size-4" />
        </Button>
      </div>

      {showFilters ? (
        <FilterDropdowns
          protocolFilter={protocolFilter}
          setProtocolFilter={setProtocolFilter}
          privacyFilter={privacyFilter}
          setPrivacyFilter={setPrivacyFilter}
          stats={stats}
        />
      ) : null}

      <p className="text-muted-foreground text-sm">
        {filteredCount} of {totalCount} definitions
      </p>
    </div>
  )
}

export function DefinitionSearchTable({
  definitions,
  isLoading,
  onSelect,
}: DefinitionSearchTableProps) {
  const state = useDefinitionSearchTable(definitions)

  return (
    <div className="flex h-full flex-col">
      <SearchToolbar
        searchQuery={state.searchQuery}
        setSearchQuery={state.setSearchQuery}
        showFilters={state.showFilters}
        setShowFilters={state.setShowFilters}
        protocolFilter={state.protocolFilter}
        setProtocolFilter={state.setProtocolFilter}
        privacyFilter={state.privacyFilter}
        setPrivacyFilter={state.setPrivacyFilter}
        stats={state.stats}
        filteredCount={state.filteredDefinitions.length}
        totalCount={definitions.length}
      />

      <ScrollArea className="min-h-0 flex-1 rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[200px]">Name</TableHead>
              <TableHead className="w-[100px]">Protocol</TableHead>
              <TableHead className="w-[100px]">Privacy</TableHead>
              <TableHead>Description</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <DefinitionTableRows
              definitions={state.filteredDefinitions}
              isLoading={isLoading}
              hasActiveFilters={state.hasActiveFilters}
              onSelect={onSelect}
            />
          </TableBody>
        </Table>
      </ScrollArea>
    </div>
  )
}
