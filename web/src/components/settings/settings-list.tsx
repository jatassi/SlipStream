import type { ReactNode } from 'react'

import { Plus } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { Group, Row, RowSkeleton } from '@/components/grouped-list'

export function AddAction({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="press-dim focus-visible:ring-ring text-primary flex size-11 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]"
    >
      <Plus className="size-6" />
    </button>
  )
}

type SettingsListProps = {
  state: {
    isLoading: boolean
    isError: boolean
    isEmpty: boolean
    refetch: () => unknown
  }
  empty: string
  header?: string
  children: ReactNode
}

export function SettingsList({ state, empty, header, children }: SettingsListProps) {
  if (state.isLoading) {
    return (
      <Group header={header}>
        <RowSkeleton />
        <RowSkeleton />
        <RowSkeleton />
      </Group>
    )
  }

  if (state.isError) {
    return (
      <div className="px-screen">
        <ErrorState onRetry={() => void state.refetch()} />
      </div>
    )
  }

  if (state.isEmpty) {
    return (
      <Group header={header}>
        <Row title={<span className="text-muted-foreground font-normal">{empty}</span>} />
      </Group>
    )
  }

  return <Group header={header}>{children}</Group>
}
