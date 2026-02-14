import type {
  HealthCategory,
  HealthItem,
  HealthResponse,
  HealthSummary,
  TestCategoryResult,
  TestItemResult,
} from '@/types/health'

import { apiFetch } from './client'

export const healthApi = {
  // Get all health items grouped by category
  getAll: () => apiFetch<HealthResponse>('/system/health'),

  // Get summary counts for dashboard
  getSummary: () => apiFetch<HealthSummary>('/system/health/summary'),

  // Get items for a specific category
  getCategory: (category: HealthCategory) => apiFetch<HealthItem[]>(`/system/health/${category}`),

  // Test all items in a category
  testCategory: (category: HealthCategory) =>
    apiFetch<TestCategoryResult>(`/system/health/${category}/test`, {
      method: 'POST',
    }),

  // Test a specific item
  testItem: (category: HealthCategory, id: string) =>
    apiFetch<TestItemResult>(`/system/health/${category}/${id}/test`, {
      method: 'POST',
    }),
}
