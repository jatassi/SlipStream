import type { ReactNode } from 'react'

import { EmptyState } from '@/components/data/empty-state'
import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { AddPlaceholderCard } from '@/components/settings/add-placeholder-card'
import { cn } from '@/lib/utils'

type AddPlaceholder = {
  label: string
  onClick: () => void
  icon?: ReactNode
}

type ListSectionProps<T> = {
  data: T[] | undefined
  isLoading: boolean
  isError: boolean
  refetch: () => void
  emptyIcon: ReactNode
  emptyTitle: string
  emptyDescription: string
  emptyAction: { label: string; onClick: () => void }
  renderItem: (item: T) => ReactNode
  gridCols?: 1 | 2 | 3
  keyExtractor: (item: T) => string | number
  addPlaceholder?: AddPlaceholder
}

const gridColsClassName: Record<number, string> = {
  1: 'space-y-4',
  2: 'grid gap-4 md:grid-cols-2',
  3: 'grid gap-4 md:grid-cols-3',
}

function PlaceholderCard({ placeholder }: { placeholder: AddPlaceholder }) {
  return (
    <AddPlaceholderCard
      label={placeholder.label}
      onClick={placeholder.onClick}
      icon={placeholder.icon}
    />
  )
}

function ListEmptyState<T>({
  emptyIcon,
  emptyTitle,
  emptyDescription,
  emptyAction,
  addPlaceholder,
}: Pick<ListSectionProps<T>, 'emptyIcon' | 'emptyTitle' | 'emptyDescription' | 'emptyAction' | 'addPlaceholder'>) {
  if (addPlaceholder) {
    return <PlaceholderCard placeholder={addPlaceholder} />
  }
  return (
    <EmptyState
      icon={emptyIcon}
      title={emptyTitle}
      description={emptyDescription}
      action={emptyAction}
    />
  )
}

export function ListSection<T>(props: ListSectionProps<T>) {
  const { data, isLoading, isError, refetch, renderItem, gridCols = 1, keyExtractor, addPlaceholder, emptyIcon, emptyTitle, emptyDescription, emptyAction } = props

  if (isLoading) {return <LoadingState variant="list" count={3} />}
  if (isError) {return <ErrorState onRetry={refetch} />}
  if (!data?.length) {return <ListEmptyState emptyIcon={emptyIcon} emptyTitle={emptyTitle} emptyDescription={emptyDescription} emptyAction={emptyAction} addPlaceholder={addPlaceholder} />}

  return (
    <div className={cn(gridColsClassName[gridCols])}>
      {data.map((item) => (
        <div key={keyExtractor(item)}>{renderItem(item)}</div>
      ))}
      {addPlaceholder ? <PlaceholderCard placeholder={addPlaceholder} /> : null}
    </div>
  )
}
