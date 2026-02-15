import { Zap } from 'lucide-react'

import { LoadingButton } from '@/components/ui/loading-button'
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
      <LoadingButton loading={false} icon={Zap} disabled>
        Search All
      </LoadingButton>
    )
  }

  if (searchCount <= 0) {
    return null
  }

  return (
    <LoadingButton loading={isSearching} icon={Zap} onClick={onSearch} className={cn(searchButtonStyle)}>
      Search All ({searchCount})
    </LoadingButton>
  )
}
