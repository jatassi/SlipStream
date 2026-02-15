import { useMemo, useState } from 'react'

import { useDebounce } from '@/hooks'
import type { DefinitionMetadata, Privacy, Protocol } from '@/types'

type DefinitionStats = {
  total: number
  torrent: number
  usenet: number
  public: number
  private: number
}

function computeStats(definitions: DefinitionMetadata[]): DefinitionStats {
  return {
    total: definitions.length,
    torrent: definitions.filter((d) => d.protocol === 'torrent').length,
    usenet: definitions.filter((d) => d.protocol === 'usenet').length,
    public: definitions.filter((d) => d.privacy === 'public').length,
    private: definitions.filter((d) => d.privacy === 'private').length,
  }
}

function matchesSearch(def: DefinitionMetadata, query: string): boolean {
  const lower = query.toLowerCase()
  return (
    def.name.toLowerCase().includes(lower) ||
    def.id.toLowerCase().includes(lower) ||
    (def.description?.toLowerCase().includes(lower) ?? false)
  )
}

export function useDefinitionSearchTable(definitions: DefinitionMetadata[]) {
  const [searchQuery, setSearchQuery] = useState('')
  const [protocolFilter, setProtocolFilter] = useState<Protocol | 'all'>('all')
  const [privacyFilter, setPrivacyFilter] = useState<Privacy | 'all'>('all')
  const [showFilters, setShowFilters] = useState(false)

  const debouncedQuery = useDebounce(searchQuery, 300)

  const filteredDefinitions = useMemo(() => {
    return definitions.filter((def) => {
      if (debouncedQuery && !matchesSearch(def, debouncedQuery)) {
        return false
      }
      if (protocolFilter !== 'all' && def.protocol !== protocolFilter) {
        return false
      }
      if (privacyFilter !== 'all' && def.privacy !== privacyFilter) {
        return false
      }
      return true
    })
  }, [definitions, debouncedQuery, protocolFilter, privacyFilter])

  const stats = useMemo(() => computeStats(definitions), [definitions])

  const hasActiveFilters =
    searchQuery !== '' || protocolFilter !== 'all' || privacyFilter !== 'all'

  return {
    searchQuery,
    setSearchQuery,
    protocolFilter,
    setProtocolFilter,
    privacyFilter,
    setPrivacyFilter,
    showFilters,
    setShowFilters,
    filteredDefinitions,
    stats,
    hasActiveFilters,
  }
}

export type { DefinitionStats }
