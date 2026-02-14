import type { PortalUser } from '@/types'

import { apiFetch } from './client'

export type AuthStatus = {
  requiresSetup: boolean
  requiresAuth: boolean
}

export type AdminSetupRequest = {
  password: string
}

export type AdminSetupResponse = {
  token: string
  user: PortalUser
}

export async function getAuthStatus(): Promise<AuthStatus> {
  return apiFetch<AuthStatus>('/auth/status')
}

export async function adminSetup(password: string): Promise<AdminSetupResponse> {
  return apiFetch<AdminSetupResponse>('/auth/setup', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
}

export async function deleteAdmin(): Promise<void> {
  await apiFetch('/auth/admin', { method: 'DELETE' })
}
