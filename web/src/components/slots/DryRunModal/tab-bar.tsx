import { Ban, CheckSquare, Film, Layers, RotateCcw, Square, Tv, XCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'

import type { MigrationPreview } from './types'

type TabBarProps = {
  activeTab: 'movies' | 'tv'
  onTabChange: (tab: 'movies' | 'tv') => void
  preview: MigrationPreview
  visibleFileIds: number[]
  selectedFileIds: Set<number>
  manualEditsCount: number
  onToggleSelectAll: () => void
  onIgnore: () => void
  onOpenAssign: () => void
  onUnassign: () => void
  onReset: () => void
}

export function TabBar(props: TabBarProps) {
  const allSelected =
    props.visibleFileIds.length > 0 && props.visibleFileIds.every((id) => props.selectedFileIds.has(id))

  return (
    <div className="mb-2 flex shrink-0 items-center justify-between border-b pb-2">
      <TabButtons activeTab={props.activeTab} onTabChange={props.onTabChange} preview={props.preview} />
      <ActionButtons
        allSelected={allSelected}
        selectedCount={props.selectedFileIds.size}
        visibleCount={props.visibleFileIds.length}
        manualEditsCount={props.manualEditsCount}
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
  activeTab: 'movies' | 'tv'
  onTabChange: (tab: 'movies' | 'tv') => void
  preview: MigrationPreview
}

function TabButtons({ activeTab, onTabChange, preview }: TabButtonsProps) {
  return (
    <div className="flex gap-2">
      <Button
        variant={activeTab === 'movies' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('movies')}
        disabled={preview.movies.length === 0}
      >
        <Film className="mr-2 size-4" />
        Movies ({preview.movies.length})
      </Button>
      <Button
        variant={activeTab === 'tv' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('tv')}
        disabled={preview.tvShows.length === 0}
      >
        <Tv className="mr-2 size-4" />
        Series ({preview.tvShows.length})
      </Button>
    </div>
  )
}

type ActionButtonsProps = {
  allSelected: boolean
  selectedCount: number
  visibleCount: number
  manualEditsCount: number
  onToggleSelectAll: () => void
  onIgnore: () => void
  onOpenAssign: () => void
  onUnassign: () => void
  onReset: () => void
}

function ActionButtons({
  allSelected,
  selectedCount,
  visibleCount,
  manualEditsCount,
  onToggleSelectAll,
  onIgnore,
  onOpenAssign,
  onUnassign,
  onReset,
}: ActionButtonsProps) {
  const SelectIcon = allSelected ? CheckSquare : Square
  const hasSelection = selectedCount > 0
  const ignoreLabel = hasSelection ? `Ignore (${selectedCount})` : 'Ignore'

  return (
    <div className="flex gap-1.5">
      <Button variant="outline" size="sm" onClick={onToggleSelectAll} disabled={visibleCount === 0}>
        <SelectIcon className="mr-1.5 size-4" />
        {allSelected ? 'Deselect All' : 'Select All'}
      </Button>
      <Button variant="outline" size="sm" onClick={onIgnore} disabled={!hasSelection}>
        <Ban className="mr-1.5 size-4" />
        {ignoreLabel}
      </Button>
      <Button variant="outline" size="sm" onClick={onOpenAssign} disabled={!hasSelection}>
        <Layers className="mr-1.5 size-4" />
        Assign...
      </Button>
      <Button variant="outline" size="sm" onClick={onUnassign} disabled={!hasSelection}>
        <XCircle className="mr-1.5 size-4" />
        Unassign
      </Button>
      <Button variant="outline" size="sm" onClick={onReset} disabled={manualEditsCount === 0}>
        <RotateCcw className="mr-1.5 size-4" />
        Reset
      </Button>
    </div>
  )
}
