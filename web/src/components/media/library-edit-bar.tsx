import type { ReactNode, RefObject } from 'react'
import { useRef, useState } from 'react'

import { Eye, EyeOff, SlidersHorizontal, Trash2 } from 'lucide-react'

import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'
import type { QualityProfile } from '@/types'

export type LibraryEditBarProps = {
  selectedCount: number
  totalCount: number
  pluralMediaLabel: string
  qualityProfiles: QualityProfile[] | undefined
  isBulkUpdating: boolean
  onSelectAll: () => void
  onMonitor: (monitored: boolean) => void
  onChangeQualityProfile: (id: number) => void
  onDelete: () => void
}

const ACTION_CLASSES =
  'press-dim focus-visible:ring-ring text-primary flex size-11 items-center justify-center rounded-full outline-none focus-visible:ring-[3px] disabled:opacity-40'

function BarAction({
  label,
  disabled,
  onClick,
  children,
  buttonRef,
  destructive,
}: {
  label: string
  disabled: boolean
  onClick: () => void
  children: ReactNode
  buttonRef?: RefObject<HTMLButtonElement | null>
  destructive?: boolean
}) {
  return (
    <button
      ref={buttonRef}
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={destructive === true ? `${ACTION_CLASSES} text-destructive` : ACTION_CLASSES}
    >
      {children}
    </button>
  )
}

function profileActions(
  profiles: QualityProfile[] | undefined,
  onChange: (id: number) => void,
): ActionItem[] {
  return (profiles ?? []).map((profile) => ({
    label: profile.name,
    onClick: () => {
      onChange(profile.id)
    },
  }))
}

function SelectAllButton({ allSelected, onClick }: { allSelected: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="press-dim focus-visible:ring-ring text-footnote text-primary min-h-tap rounded-full px-1 font-semibold outline-none focus-visible:ring-[3px]"
    >
      {allSelected ? 'Deselect All' : 'Select All'}
    </button>
  )
}

type BulkActionsProps = {
  plural: string
  selectedCount: number
  isBulkUpdating: boolean
  onMonitor: (monitored: boolean) => void
  onDelete: () => void
  profileAnchor: RefObject<HTMLButtonElement | null>
  onOpenProfiles: () => void
}

function BulkActions({
  plural,
  selectedCount,
  isBulkUpdating,
  onMonitor,
  onDelete,
  profileAnchor,
  onOpenProfiles,
}: BulkActionsProps) {
  const disabled = selectedCount === 0 || isBulkUpdating

  return (
    <div className="ml-auto flex items-center">
      <BarAction
        label={`Monitor selected ${plural}`}
        disabled={disabled}
        onClick={() => {
          onMonitor(true)
        }}
      >
        <Eye className="size-5" />
      </BarAction>
      <BarAction
        label={`Unmonitor selected ${plural}`}
        disabled={disabled}
        onClick={() => {
          onMonitor(false)
        }}
      >
        <EyeOff className="size-5" />
      </BarAction>
      <BarAction
        label="Set quality profile"
        disabled={disabled}
        buttonRef={profileAnchor}
        onClick={onOpenProfiles}
      >
        <SlidersHorizontal className="size-5" />
      </BarAction>
      <BarAction
        label={`Delete selected ${plural}`}
        disabled={selectedCount === 0}
        destructive
        onClick={onDelete}
      >
        <Trash2 className="size-5" />
      </BarAction>
    </div>
  )
}

export function LibraryEditBar(props: LibraryEditBarProps) {
  const profileAnchor = useRef<HTMLButtonElement>(null)
  const [profilesOpen, setProfilesOpen] = useState(false)
  const allSelected = props.selectedCount === props.totalCount && props.totalCount > 0

  return (
    <div role="toolbar" aria-label="Selection actions" className="px-screen flex items-center gap-2 py-2">
      <SelectAllButton allSelected={allSelected} onClick={props.onSelectAll} />
      <span className="text-footnote text-muted-foreground nums truncate">
        {props.selectedCount} selected
      </span>
      <BulkActions
        plural={props.pluralMediaLabel.toLowerCase()}
        selectedCount={props.selectedCount}
        isBulkUpdating={props.isBulkUpdating}
        onMonitor={props.onMonitor}
        onDelete={props.onDelete}
        profileAnchor={profileAnchor}
        onOpenProfiles={() => {
          setProfilesOpen(true)
        }}
      />
      <ActionPresenter
        open={profilesOpen}
        onOpenChange={setProfilesOpen}
        title="Quality profile"
        actions={profileActions(props.qualityProfiles, props.onChangeQualityProfile)}
        anchor={profileAnchor}
      />
    </div>
  )
}
