import { Search } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { LoadingButton } from '@/components/ui/loading-button'

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
      <LoadingButton loading={isLoading} icon={Search} onClick={onSearch} />
    </div>
  )
}
