import type { CreateInvitationRequest, Invitation } from '@/types'

import { apiFetch } from '../client'

const BASE_PATH = '/admin/requests/invitations'

export async function listInvitations(): Promise<Invitation[]> {
  return apiFetch<Invitation[]>(BASE_PATH)
}

export async function createInvitation(input: CreateInvitationRequest): Promise<Invitation> {
  return apiFetch<Invitation>(BASE_PATH, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function deleteInvitation(id: number): Promise<undefined> {
  return apiFetch<undefined>(`${BASE_PATH}/${id}`, {
    method: 'DELETE',
  })
}

export async function resendInvitation(id: number): Promise<Invitation> {
  return apiFetch<Invitation>(`${BASE_PATH}/${id}/resend`, {
    method: 'POST',
  })
}

export function getInvitationLink(token: string): string {
  const baseUrl = globalThis.location.origin
  return `${baseUrl}/requests/auth/signup?token=${token}`
}
