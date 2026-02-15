import { Loader2, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function SearchInputBar({
  query,
  isLoading,
  onQueryChange,
  onSearch,
}: {
  query: string
  isLoading: boolean
  onQueryChange: (value: string) => void
  onSearch: () => void
}) {
  return (
    <div className="flex gap-2">
      <Input
        placeholder="Search query (optional, overrides automatic search)"
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            onSearch()
          }
        }}
      />
      <Button onClick={onSearch} disabled={isLoading}>
        {isLoading ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Search className="size-4" />
        )}
      </Button>
    </div>
  )
}
