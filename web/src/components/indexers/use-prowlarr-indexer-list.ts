import { useProwlarrIndexersWithSettings, useProwlarrStatus } from '@/hooks'

export function useProwlarrIndexerList(showOnlyEnabled: boolean) {
  const { data: indexers, isLoading: indexersLoading } = useProwlarrIndexersWithSettings()
  const { data: status } = useProwlarrStatus()

  const connected = status?.connected ?? false

  const filteredIndexers = showOnlyEnabled ? indexers?.filter((i) => i.enable) : indexers
  const displayedIndexers = filteredIndexers?.toSorted((a, b) => {
    if (a.enable !== b.enable) {
      return a.enable ? -1 : 1
    }
    const priorityA = a.settings?.priority ?? 25
    const priorityB = b.settings?.priority ?? 25
    return priorityA - priorityB
  })

  const enabledCount = indexers?.filter((i) => i.enable).length ?? 0
  const totalCount = indexers?.length ?? 0

  return {
    connected,
    indexersLoading,
    displayedIndexers,
    enabledCount,
    totalCount,
  }
}
