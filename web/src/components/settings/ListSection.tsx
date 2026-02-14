import type { ReactNode } from 'react'

import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { AddPlaceholderCard } from '@/components/settings/AddPlaceholderCard'
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

export function ListSection<T>({
  data,
  isLoading,
  isError,
  refetch,
  emptyIcon,
  emptyTitle,
  emptyDescription,
  emptyAction,
  renderItem,
  gridCols = 1,
  keyExtractor,
  addPlaceholder,
}: ListSectionProps<T>) {
  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }

  if (isError) {
    return <ErrorState onRetry={refetch} />
  }

  if (!data?.length) {
    if (addPlaceholder) {
      return (
        <AddPlaceholderCard
          label={addPlaceholder.label}
          onClick={addPlaceholder.onClick}
          icon={addPlaceholder.icon}
        />
      )
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

  const gridClassName = cn(
    gridCols === 1 && 'space-y-4',
    gridCols === 2 && 'grid gap-4 md:grid-cols-2',
    gridCols === 3 && 'grid gap-4 md:grid-cols-3',
  )

  return (
    <div className={gridClassName}>
      {data.map((item) => (
        <div key={keyExtractor(item)}>{renderItem(item)}</div>
      ))}
      {addPlaceholder ? (
        <AddPlaceholderCard
          label={addPlaceholder.label}
          onClick={addPlaceholder.onClick}
          icon={addPlaceholder.icon}
        />
      ) : null}
    </div>
  )
}
