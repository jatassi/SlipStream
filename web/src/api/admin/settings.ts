import type { RequestSettings } from '@/types'

import { apiFetch } from '../client'

const BASE_PATH = '/admin/requests/settings'

export async function getRequestSettings(): Promise<RequestSettings> {
  return apiFetch<RequestSettings>(BASE_PATH)
}

export async function updateRequestSettings(
  settings: Partial<RequestSettings>,
): Promise<RequestSettings> {
  return apiFetch<RequestSettings>(BASE_PATH, {
    method: 'PUT',
    body: JSON.stringify(settings),
  })
}
