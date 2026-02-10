import { apiFetch } from './client'
import type { RssSyncSettings, RssSyncStatus } from '@/types'

export const rssSyncApi = {
  getSettings: () => apiFetch<RssSyncSettings>('/settings/rsssync'),

  updateSettings: (settings: RssSyncSettings) =>
    apiFetch<RssSyncSettings>('/settings/rsssync', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),

  getStatus: () => apiFetch<RssSyncStatus>('/rsssync/status'),

  trigger: () => apiFetch<{ message: string }>('/rsssync/trigger', { method: 'POST' }),
}
