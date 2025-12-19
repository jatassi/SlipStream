import { apiFetch } from './client'
import type { SystemStatus, HealthCheck, Settings, UpdateSettingsInput } from '@/types'

export const systemApi = {
  status: () =>
    apiFetch<SystemStatus>('/status'),

  health: () =>
    fetch('/health').then(res => res.json() as Promise<HealthCheck>),

  getSettings: () =>
    apiFetch<Settings>('/settings'),

  updateSettings: (data: UpdateSettingsInput) =>
    apiFetch<Settings>('/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  regenerateApiKey: () =>
    apiFetch<{ apiKey: string }>('/settings/apikey', { method: 'POST' }),
}
