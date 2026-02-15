import { Loader2, Zap } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type SearchButtonProps = {
  isLoading: boolean
  isSearching: boolean
  searchCount: number
  searchButtonStyle: string
  onSearch: () => void
}

export function SearchButton({
  isLoading,
  isSearching,
  searchCount,
  searchButtonStyle,
  onSearch,
}: SearchButtonProps) {
  if (isLoading) {
    return (
      <Button disabled>
        <Zap className="mr-2 size-4" />
        Search All
      </Button>
    )
  }

  if (searchCount <= 0) {
    return null
  }

  return (
    <Button disabled={isSearching} onClick={onSearch} className={cn(searchButtonStyle)}>
      {isSearching ? (
        <Loader2 className="mr-2 size-4 animate-spin" />
      ) : (
        <Zap className="mr-2 size-4" />
      )}
      Search All ({searchCount})
    </Button>
  )
}
