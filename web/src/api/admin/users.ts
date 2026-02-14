import type { AdminUpdateUserInput, PortalUserWithQuota, QuotaLimits } from '@/types'

import { apiFetch } from '../client'

const BASE_PATH = '/admin/requests/users'

export async function listUsers(): Promise<PortalUserWithQuota[]> {
  return apiFetch<PortalUserWithQuota[]>(BASE_PATH)
}

export async function getUser(id: number): Promise<PortalUserWithQuota> {
  return apiFetch<PortalUserWithQuota>(`${BASE_PATH}/${id}`)
}

export async function updateUser(
  id: number,
  input: AdminUpdateUserInput,
): Promise<PortalUserWithQuota> {
  return apiFetch<PortalUserWithQuota>(`${BASE_PATH}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function enableUser(id: number): Promise<PortalUserWithQuota> {
  return apiFetch<PortalUserWithQuota>(`${BASE_PATH}/${id}/enable`, {
    method: 'POST',
  })
}

export async function disableUser(id: number): Promise<PortalUserWithQuota> {
  return apiFetch<PortalUserWithQuota>(`${BASE_PATH}/${id}/disable`, {
    method: 'POST',
  })
}

export async function deleteUser(id: number): Promise<undefined> {
  return apiFetch<undefined>(`${BASE_PATH}/${id}`, {
    method: 'DELETE',
  })
}

export async function setUserQuota(id: number, quota: QuotaLimits): Promise<PortalUserWithQuota> {
  return apiFetch<PortalUserWithQuota>(`${BASE_PATH}/${id}/quota`, {
    method: 'PUT',
    body: JSON.stringify(quota),
  })
}
