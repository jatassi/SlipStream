import type {
  ApproveRequestInput,
  DenyRequestInput,
  Request,
  RequestListFilters,
} from '@/types'

import { apiFetch, buildQueryString } from '../client'

const BASE_PATH = '/admin/requests'

export async function listRequests(filters?: RequestListFilters): Promise<Request[]> {
  const query = filters ? buildQueryString(filters) : ''
  return apiFetch<Request[]>(`${BASE_PATH}${query}`)
}

export async function approveRequest(id: number, input: ApproveRequestInput): Promise<Request> {
  return apiFetch<Request>(`${BASE_PATH}/${id}/approve`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function denyRequest(id: number, input?: DenyRequestInput): Promise<Request> {
  return apiFetch<Request>(`${BASE_PATH}/${id}/deny`, {
    method: 'POST',
    body: JSON.stringify(input ?? {}),
  })
}

export async function deleteRequest(id: number): Promise<undefined> {
  return apiFetch<undefined>(`${BASE_PATH}/${id}`, {
    method: 'DELETE',
  })
}
