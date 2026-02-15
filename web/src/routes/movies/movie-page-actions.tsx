import { Pencil, Plus, RefreshCw, X } from 'lucide-react'

import { Button } from '@/components/ui/button'

type MoviePageActionsProps = {
  isLoading: boolean
  editMode: boolean
  isRefreshing: boolean
  onRefreshAll: () => void
  onEnterEdit: () => void
  onExitEdit: () => void
}

export function MoviePageActions({
  isLoading,
  editMode,
  isRefreshing,
  onRefreshAll,
  onEnterEdit,
  onExitEdit,
}: MoviePageActionsProps) {
  return (
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        onClick={onRefreshAll}
        disabled={isLoading || isRefreshing || editMode}
      >
        <RefreshCw className={`mr-1 size-4 ${isRefreshing ? 'animate-spin' : ''}`} />
        {isRefreshing ? 'Refreshing...' : 'Refresh'}
      </Button>
      {editMode ? (
        <Button variant="outline" onClick={onExitEdit}>
          <X className="mr-1 size-4" />
          Cancel
        </Button>
      ) : (
        <Button variant="outline" onClick={onEnterEdit} disabled={isLoading}>
          <Pencil className="mr-1 size-4" />
          Edit
        </Button>
      )}
      <Button
        disabled={isLoading || editMode}
        className="bg-movie-500 hover:bg-movie-400 border-movie-500"
        onClick={() => document.getElementById('global-search')?.focus()}
      >
        <Plus className="mr-1 size-4" />
        Add Movie
      </Button>
    </div>
  )
}
