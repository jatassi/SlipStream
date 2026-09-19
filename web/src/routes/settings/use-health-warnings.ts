import { useSystemHealth } from '@/hooks'
import type { HealthCategory, HealthItem } from '@/types/health'

function countIssues(items: HealthItem[] | undefined): number {
  return (items ?? []).filter((item) => item.status !== 'ok').length
}

export function useHealthWarnings(): (categories: HealthCategory[]) => number {
  const { data } = useSystemHealth()
  return (categories: HealthCategory[]) =>
    categories.reduce((total, category) => total + countIssues(data?.[category]), 0)
}
