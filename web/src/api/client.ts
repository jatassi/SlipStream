import { ApiError } from '@/types'

import { getPortalAuthToken } from './portal/client'

const API_BASE = '/api/v1'

function extractState(parsed: unknown): unknown {
  if (!parsed || typeof parsed !== 'object' || !('state' in parsed)) {
    return null
  }
  return parsed.state
}

function extractUserAndToken(
  state: unknown,
): { user: unknown; token: string } | null {
  if (!state || typeof state !== 'object') {return null}
  if (!('user' in state) || !('token' in state)) {return null}
  if (typeof state.token !== 'string') {return null}
  return { user: state.user, token: state.token }
}

function isAdminUser(user: unknown): boolean {
  if (!user || typeof user !== 'object') {return false}
  if (!('isAdmin' in user)) {return false}
  return !!user.isAdmin
}

function getInitialAdminToken(): string | null {
  try {
    const stored = localStorage.getItem('slipstream-portal-auth')
    if (!stored) {return null}
    const parsed: unknown = JSON.parse(stored)
    const state = extractState(parsed)
    const userAndToken = extractUserAndToken(state)
    if (!userAndToken) {return null}
    return isAdminUser(userAndToken.user) ? userAndToken.token : null
  } catch {
    return null
  }
}

let adminAuthToken: string | null = getInitialAdminToken()

export function setAdminAuthToken(token: string | null): void {
  adminAuthToken = token
}

export function getAdminAuthToken(): string | null {
  return adminAuthToken
}

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Use admin token if available, otherwise fall back to portal token
  const token = adminAuthToken ?? getPortalAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...(options?.headers as Record<string, string> | undefined),
    },
  })

  if (!res.ok) {
    // Handle 401 - clear auth and redirect to login
    if (res.status === 401) {
      const hadToken = !!(adminAuthToken ?? getPortalAuthToken())
      adminAuthToken = null
      if (hadToken) {
        globalThis.dispatchEvent(new CustomEvent('auth:unauthorized'))
      }
    }
    let errorData: unknown = null
    try {
      errorData = (await res.json()) as unknown
    } catch {
      // Response might not be JSON
    }
    throw new ApiError(
      res.status,
      errorData as { message?: string; error?: string } | null,
    )
  }

  // Handle 204 No Content
  if (res.status === 204) {
    return undefined as T
  }

  return (await res.json()) as T
}

export function buildQueryString(params: object): string {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value))
    }
  }
  const queryString = searchParams.toString()
  return queryString ? `?${queryString}` : ''
}
