import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { healthApi } from '@/api/health'
import type { HealthCategory } from '@/types/health'

export const systemHealthKeys = {
  all: ['systemHealth'] as const,
  list: () => [...systemHealthKeys.all, 'list'] as const,
  summary: () => [...systemHealthKeys.all, 'summary'] as const,
  category: (category: HealthCategory) => [...systemHealthKeys.all, 'category', category] as const,
}

// Fetch all health items
export function useSystemHealth() {
  return useQuery({
    queryKey: systemHealthKeys.list(),
    queryFn: () => healthApi.getAll(),
  })
}

// Fetch health summary for dashboard
export function useSystemHealthSummary() {
  return useQuery({
    queryKey: systemHealthKeys.summary(),
    queryFn: () => healthApi.getSummary(),
  })
}

// Fetch health items for a specific category
export function useSystemHealthCategory(category: HealthCategory) {
  return useQuery({
    queryKey: systemHealthKeys.category(category),
    queryFn: () => healthApi.getCategory(category),
    enabled: !!category,
  })
}

// Test all items in a category
export function useTestHealthCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (category: HealthCategory) => healthApi.testCategory(category),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemHealthKeys.all })
    },
  })
}

// Test a specific health item
export function useTestHealthItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ category, id }: { category: HealthCategory; id: string }) =>
      healthApi.testItem(category, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemHealthKeys.all })
    },
  })
}
