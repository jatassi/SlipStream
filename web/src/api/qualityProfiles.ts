import { apiFetch } from './client'
import type {
  QualityProfile,
  CreateQualityProfileInput,
  UpdateQualityProfileInput,
} from '@/types'

export const qualityProfilesApi = {
  list: () =>
    apiFetch<QualityProfile[]>('/qualityprofiles'),

  get: (id: number) =>
    apiFetch<QualityProfile>(`/qualityprofiles/${id}`),

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

  delete: (id: number) =>
    apiFetch<void>(`/qualityprofiles/${id}`, { method: 'DELETE' }),
}
