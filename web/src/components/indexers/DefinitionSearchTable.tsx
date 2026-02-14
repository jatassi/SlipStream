import { useMemo, useState } from 'react'

import { Filter, Globe, Lock, Search, Unlock } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useDebounce } from '@/hooks'
import type { DefinitionMetadata, Privacy, Protocol } from '@/types'

type DefinitionSearchTableProps = {
  definitions: DefinitionMetadata[]
  isLoading?: boolean
  onSelect: (definition: DefinitionMetadata) => void
}

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

export function DefinitionSearchTable({
  definitions,
  isLoading,
  onSelect,
}: DefinitionSearchTableProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<Protocol | 'all'>('all')
  const [privacyFilter, setPrivacyFilter] = useState<Privacy | 'all'>('all')
  const [showFilters, setShowFilters] = useState(false)

  const debouncedQuery = useDebounce(searchQuery, 300)

  const filteredDefinitions = useMemo(() => {
    return (definitions || []).filter((def) => {
      // Search filter
      if (debouncedQuery) {
        const query = debouncedQuery.toLowerCase()
        const matchesName = def.name.toLowerCase().includes(query)
        const matchesId = def.id.toLowerCase().includes(query)
        const matchesDescription = def.description?.toLowerCase().includes(query)
        if (!matchesName && !matchesId && !matchesDescription) {
          return false
        }
      }

      // Protocol filter
      if (protocolFilter !== 'all' && def.protocol !== protocolFilter) {
        return false
      }

      // Privacy filter
      if (privacyFilter !== 'all' && def.privacy !== privacyFilter) {
        return false
      }

      return true
    })
  }, [definitions, debouncedQuery, protocolFilter, privacyFilter])

  const stats = useMemo(() => {
    const defs = definitions || []
    const total = defs.length
    const torrent = defs.filter((d) => d.protocol === 'torrent').length
    const usenet = defs.filter((d) => d.protocol === 'usenet').length
    const publicCount = defs.filter((d) => d.privacy === 'public').length
    const privateCount = defs.filter((d) => d.privacy === 'private').length
    return { total, torrent, usenet, public: publicCount, private: privateCount }
  }, [definitions])

  return (
    <div className="flex h-full flex-col">
      {/* Search and filters */}
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
          <div className="flex gap-2">
            <Select
              value={protocolFilter}
              onValueChange={(v) => v && setProtocolFilter(v as Protocol | 'all')}
            >
              <SelectTrigger className="w-32">
                <SelectValue>
                  {protocolFilter === 'all'
                    ? `All (${stats.total})`
                    : protocolFilter === 'torrent'
                      ? `Torrent (${stats.torrent})`
                      : protocolFilter === 'usenet'
                        ? `Usenet (${stats.usenet})`
                        : protocolFilter}
                </SelectValue>
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
                <SelectValue>
                  {privacyFilter === 'all'
                    ? 'All'
                    : privacyFilter === 'public'
                      ? `Public (${stats.public})`
                      : privacyFilter === 'semi-private'
                        ? 'Semi-Private'
                        : privacyFilter === 'private'
                          ? `Private (${stats.private})`
                          : privacyFilter}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="public">Public ({stats.public})</SelectItem>
                <SelectItem value="semi-private">Semi-Private</SelectItem>
                <SelectItem value="private">Private ({stats.private})</SelectItem>
              </SelectContent>
            </Select>
          </div>
        ) : null}

        <p className="text-muted-foreground text-sm">
          {filteredDefinitions.length} of {definitions.length} definitions
        </p>
      </div>

      {/* Table */}
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
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground py-8 text-center">
                  Loading definitions...
                </TableCell>
              </TableRow>
            ) : filteredDefinitions.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground py-8 text-center">
                  {searchQuery || protocolFilter !== 'all' || privacyFilter !== 'all'
                    ? 'No definitions match your filters'
                    : 'No definitions available'}
                </TableCell>
              </TableRow>
            ) : (
              filteredDefinitions.map((def) => (
                <TableRow
                  key={def.id}
                  className="hover:bg-muted/50 cursor-pointer"
                  onClick={() => onSelect(def)}
                >
                  <TableCell className="font-medium">{def.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className={protocolColors[def.protocol]}>
                      {def.protocol}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className={privacyColors[def.privacy]}>
                      <span className="mr-1">{privacyIcons[def.privacy]}</span>
                      {def.privacy}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground max-w-[300px] truncate">
                    {def.description || '-'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </ScrollArea>
    </div>
  )
}
