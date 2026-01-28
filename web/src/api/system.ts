import { apiFetch } from './client'
import type { SystemStatus, HealthCheck, Settings, UpdateSettingsInput, FirewallStatus } from '@/types'

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

  updateTMDBSearchOrdering: (disableSearchOrdering: boolean) =>
    apiFetch<{ disableSearchOrdering: boolean }>('/metadata/tmdb/search-ordering', {
      method: 'POST',
      body: JSON.stringify({ disableSearchOrdering }),
    }),

  restart: () =>
    apiFetch<{ message: string }>('/system/restart', { method: 'POST' }),

  checkFirewall: () =>
    apiFetch<FirewallStatus>('/system/firewall'),
}
