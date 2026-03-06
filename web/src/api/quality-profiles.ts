import type {
  AttributeOptions,
  CheckExclusivityResponse,
  CreateQualityProfileInput,
  QualityProfile,
  UpdateQualityProfileInput,
} from '@/types'

import { apiFetch } from './client'

export const qualityProfilesApi = {
  list: (moduleType?: string) => {
    const params = moduleType ? `?moduleType=${moduleType}` : ''
    return apiFetch<QualityProfile[]>(`/qualityprofiles${params}`)
  },

  get: (id: number) => apiFetch<QualityProfile>(`/qualityprofiles/${id}`),

  create: (data: CreateQualityProfileInput) =>
    apiFetch<QualityProfile>('/qualityprofiles', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateQualityProfileInput) =>
    apiFetch<QualityProfile>(`/qualityprofiles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) => apiFetch<undefined>(`/qualityprofiles/${id}`, { method: 'DELETE' }),

  getAttributes: () => apiFetch<AttributeOptions>('/qualityprofiles/attributes'),

  checkExclusivity: (profileIds: number[]) =>
    apiFetch<CheckExclusivityResponse>('/qualityprofiles/check-exclusivity', {
      method: 'POST',
      body: JSON.stringify({ profileIds }),
    }),
}
