import type { UpdateReleaseInfo, UpdateSettings, UpdateStatus } from '@/types/update'

import { apiFetch } from './client'

export const updateApi = {
  getStatus: () => apiFetch<UpdateStatus>('/update'),

  checkForUpdate: () =>
    apiFetch<{
      status: UpdateStatus
      updateAvailable: boolean
      release?: UpdateReleaseInfo
    }>('/update/check', { method: 'POST' }),

  install: () => apiFetch<{ message: string }>('/update/install', { method: 'POST' }),

  cancel: () => apiFetch<{ message: string }>('/update/cancel', { method: 'POST' }),

  getSettings: () => apiFetch<UpdateSettings>('/update/settings'),

  updateSettings: (settings: UpdateSettings) =>
    apiFetch<UpdateSettings>('/update/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
}
