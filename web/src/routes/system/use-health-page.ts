import { useGlobalLoading } from '@/hooks'
import { useSystemHealth } from '@/hooks/use-health'
import { useIndexerMode } from '@/hooks/use-prowlarr'
import type { HealthCategory, HealthItem, HealthResponse } from '@/types/health'

const EMPTY: HealthItem[] = []

function buildCategories(health: HealthResponse | undefined) {
  const categories: { category: HealthCategory; items: HealthItem[] }[] = [
    { category: 'rootFolders', items: health?.rootFolders ?? EMPTY },
    { category: 'metadata', items: health?.metadata ?? EMPTY },
    { category: 'storage', items: health?.storage ?? EMPTY },
  ]
  return categories
}

export function useHealthPage() {
  const globalLoading = useGlobalLoading()
  const { data: health, isLoading: queryLoading, error } = useSystemHealth()
  const { data: modeData } = useIndexerMode()

  return {
    isLoading: queryLoading || globalLoading,
    error,
    isProwlarrMode: modeData?.effectiveMode === 'prowlarr',
    downloadClients: health?.downloadClients ?? EMPTY,
    prowlarrItem: health?.prowlarr[0],
    indexerItems: health?.indexers ?? EMPTY,
    regularCategories: buildCategories(health),
  }
}
