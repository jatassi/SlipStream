import type {
  AttributeOptions,
  CheckExclusivityResponse,
  CreateQualityProfileInput,
  QualityProfile,
  UpdateQualityProfileInput,
} from '@/types'

import { apiFetch } from './client'

export const qualityProfilesApi = {
  list: () => apiFetch<QualityProfile[]>('/qualityprofiles'),

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

  delete: (id: number) => apiFetch<void>(`/qualityprofiles/${id}`, { method: 'DELETE' }),

  getAttributes: () => apiFetch<AttributeOptions>('/qualityprofiles/attributes'),

  checkExclusivity: (profileIds: number[]) =>
    apiFetch<CheckExclusivityResponse>('/qualityprofiles/check-exclusivity', {
      method: 'POST',
      body: JSON.stringify({ profileIds }),
    }),
}
