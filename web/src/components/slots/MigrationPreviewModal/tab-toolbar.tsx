import { Ban, CheckSquare, Film, Layers, RotateCcw, Square, Tv, XCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'

type TabType = 'movies' | 'tv'

type TabToolbarProps = {
  activeTab: TabType
  onTabChange: (tab: TabType) => void
  movieCount: number
  tvShowCount: number
  allSelected: boolean
  visibleCount: number
  selectedCount: number
  hasEdits: boolean
  onToggleSelectAll: () => void
  onIgnore: () => void
  onOpenAssign: () => void
  onUnassign: () => void
  onReset: () => void
}

export function TabToolbar(props: TabToolbarProps) {
  return (
    <div className="mb-2 flex shrink-0 items-center justify-between border-b pb-2">
      <TabButtons
        activeTab={props.activeTab}
        onTabChange={props.onTabChange}
        movieCount={props.movieCount}
        tvShowCount={props.tvShowCount}
      />
      <ActionButtons
        allSelected={props.allSelected}
        visibleCount={props.visibleCount}
        selectedCount={props.selectedCount}
        hasEdits={props.hasEdits}
        onToggleSelectAll={props.onToggleSelectAll}
        onIgnore={props.onIgnore}
        onOpenAssign={props.onOpenAssign}
        onUnassign={props.onUnassign}
        onReset={props.onReset}
      />
    </div>
  )
}

type TabButtonsProps = {
  activeTab: TabType
  onTabChange: (tab: TabType) => void
  movieCount: number
  tvShowCount: number
}

function TabButtons({ activeTab, onTabChange, movieCount, tvShowCount }: TabButtonsProps) {
  return (
    <div className="flex gap-2">
      <Button
        variant={activeTab === 'movies' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('movies')}
        disabled={movieCount === 0}
      >
        <Film className="mr-2 size-4" />
        Movies ({movieCount})
      </Button>
      <Button
        variant={activeTab === 'tv' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('tv')}
        disabled={tvShowCount === 0}
      >
        <Tv className="mr-2 size-4" />
        Series ({tvShowCount})
      </Button>
    </div>
  )
}

type ActionButtonsProps = {
  allSelected: boolean
  visibleCount: number
  selectedCount: number
  hasEdits: boolean
  onToggleSelectAll: () => void
  onIgnore: () => void
  onOpenAssign: () => void
  onUnassign: () => void
  onReset: () => void
}

function ActionButtons({
  allSelected,
  visibleCount,
  selectedCount,
  hasEdits,
  onToggleSelectAll,
  onIgnore,
  onOpenAssign,
  onUnassign,
  onReset,
}: ActionButtonsProps) {
  const SelectIcon = allSelected ? CheckSquare : Square

  return (
    <div className="flex gap-1.5">
      <Button variant="outline" size="sm" onClick={onToggleSelectAll} disabled={visibleCount === 0}>
        <SelectIcon className="mr-1.5 size-4" />
        {allSelected ? 'Deselect All' : 'Select All'}
      </Button>
      <Button variant="outline" size="sm" onClick={onIgnore} disabled={selectedCount === 0}>
        <Ban className="mr-1.5 size-4" />
        Ignore{selectedCount > 0 ? ` (${selectedCount})` : ''}
      </Button>
      <Button variant="outline" size="sm" onClick={onOpenAssign} disabled={selectedCount === 0}>
        <Layers className="mr-1.5 size-4" />
        Assign...
      </Button>
      <Button variant="outline" size="sm" onClick={onUnassign} disabled={selectedCount === 0}>
        <XCircle className="mr-1.5 size-4" />
        Unassign
      </Button>
      <Button variant="outline" size="sm" onClick={onReset} disabled={!hasEdits}>
        <RotateCcw className="mr-1.5 size-4" />
        Reset
      </Button>
    </div>
  )
}
